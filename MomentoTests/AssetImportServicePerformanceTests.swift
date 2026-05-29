// 中文注释：本测试为图片导入提供稳定的性能观察夹具，同时锁定导入完整性。
import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers
import XCTest
@testable import Momento

final class AssetImportServicePerformanceTests: XCTestCase {
    func testFolderImportPerformanceFixturePreservesCompleteness() async throws {
        let importService = AssetImportService()
        let clock = ContinuousClock()
        let start = clock.now
        let snapshot = try await Self.importFixture(importService: importService)
        let elapsed = start.duration(to: clock.now)

        // 中文注释：这里记录耗时供本地对比观察，不作为 CI 性能门禁。
        await XCTContext.runActivity(named: "Folder import elapsed time") { activity in
            activity.add(XCTAttachment(string: "\(elapsed)"))
        }

        XCTAssertEqual(snapshot.assetCount, 57)
        XCTAssertEqual(snapshot.folderNames, Set(["A", "B", "C"]))
        XCTAssertEqual(snapshot.storedAssetFileCount, 57)
        XCTAssertTrue(snapshot.duplicateAssetFolderNames.isSuperset(of: ["A", "B"]))
        XCTAssertEqual(snapshot.jpegCameraMake, "Momento Tests")
    }

    private static func importFixture(importService: AssetImportService) async throws -> ImportPerformanceSnapshot {
        let environment = try ImportPerformanceEnvironment()
        defer {
            environment.cleanup()
        }

        let storage = LibraryStorage(applicationSupportRoot: environment.rootURL)
        let library = try storage.createLibraryPackage(at: environment.packageURL, name: "Performance")
        let metadataStore = try LibraryMetadataStore(library: library, storage: storage)
        let sourceRoot = try environment.createFixture()

        let batch = try await importService.importBatch(
            from: [sourceRoot],
            into: library,
            excludingContentHashes: metadataStore.existingContentHashes()
        )
        let assets = try metadataStore.saveImportedBatch(batch)
        let folders = try metadataStore.loadFolders()

        let foldersByID = Dictionary(uniqueKeysWithValues: folders.map { ($0.id, $0.name) })
        let duplicateAsset = try XCTUnwrap(assets.first { $0.displayName == "png-00" })
        let jpegAsset = try XCTUnwrap(assets.first { $0.originalFileName == "jpg-00.jpg" })

        return ImportPerformanceSnapshot(
            assetCount: assets.count,
            folderNames: Set(folders.map(\.name)),
            storedAssetFileCount: try environment.storedAssetFiles().count,
            duplicateAssetFolderNames: Set(duplicateAsset.folderIDs.compactMap { foldersByID[$0] }),
            jpegCameraMake: jpegAsset.exifMetadata?.cameraMake
        )
    }
}

private struct ImportPerformanceSnapshot: Sendable {
    var assetCount: Int
    var folderNames: Set<String>
    var storedAssetFileCount: Int
    var duplicateAssetFolderNames: Set<String>
    var jpegCameraMake: String?
}

private struct ImportPerformanceEnvironment {
    let rootURL: URL
    let inputURL: URL
    let packageURL: URL
    let defaults: UserDefaults
    let defaultsSuiteName: String

    init() throws {
        rootURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        inputURL = rootURL.appendingPathComponent("input", isDirectory: true)
        packageURL = rootURL.appendingPathComponent("Performance.momento", isDirectory: true)
        defaultsSuiteName = "MomentoPerformanceTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: defaultsSuiteName)!

        try FileManager.default.createDirectory(at: inputURL, withIntermediateDirectories: true)
    }

    func cleanup() {
        defaults.removePersistentDomain(forName: defaultsSuiteName)
        try? FileManager.default.removeItem(at: rootURL)
    }

    func createFixture() throws -> URL {
        let sourceRoot = inputURL.appendingPathComponent("Source", isDirectory: true)
        let folders = try ["A", "B", "C"].map { name in
            let url = sourceRoot.appendingPathComponent(name, isDirectory: true)
            try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
            return url
        }

        var pngURLs: [URL] = []
        for index in 0..<27 {
            let folder = folders[index % folders.count]
            let url = try writePNG(
                named: String(format: "png-%02d.png", index),
                in: folder,
                rgba: rgba(seed: index)
            )
            pngURLs.append(url)
        }

        for index in 0..<3 {
            let destinationFolder = folders[(index + 1) % folders.count]
            let destinationURL = destinationFolder.appendingPathComponent(
                String(format: "png-duplicate-%02d.png", index)
            )
            try FileManager.default.copyItem(at: pngURLs[index], to: destinationURL)
        }

        for index in 0..<30 {
            let folder = folders[(index + 1) % folders.count]
            try writeJPEG(
                named: String(format: "jpg-%02d.jpg", index),
                in: folder,
                rgba: rgba(seed: index + 40),
                index: index
            )
        }

        return sourceRoot
    }

    func storedAssetFiles() throws -> [URL] {
        let assetsURL = packageURL.appendingPathComponent("assets", isDirectory: true)
        guard let enumerator = FileManager.default.enumerator(
            at: assetsURL,
            includingPropertiesForKeys: [.isRegularFileKey]
        ) else {
            return []
        }

        return try enumerator.compactMap { item in
            guard let url = item as? URL else {
                return nil
            }
            let values = try url.resourceValues(forKeys: [.isRegularFileKey])
            return values.isRegularFile == true ? url : nil
        }
    }

    private func writePNG(named fileName: String, in directory: URL, rgba: [UInt8]) throws -> URL {
        try writeImage(named: fileName, in: directory, type: .png, rgba: rgba, properties: nil)
    }

    private func writeJPEG(named fileName: String, in directory: URL, rgba: [UInt8], index: Int) throws {
        let properties: [CFString: Any] = [
            kCGImagePropertyDPIWidth: 300,
            kCGImagePropertyDPIHeight: 300,
            kCGImagePropertyColorModel: kCGImagePropertyColorModelRGB,
            kCGImagePropertyProfileName: "sRGB IEC61966-2.1",
            kCGImagePropertyTIFFDictionary: [
                kCGImagePropertyTIFFMake: "Momento Tests",
                kCGImagePropertyTIFFModel: "Import Fixture \(index)"
            ],
            kCGImagePropertyExifDictionary: [
                kCGImagePropertyExifLensModel: "Fixture Lens",
                kCGImagePropertyExifExposureTime: 0.004,
                kCGImagePropertyExifFocalLength: 35.0,
                kCGImagePropertyExifISOSpeedRatings: [100 + index],
                kCGImagePropertyExifFNumber: 2.8
            ]
        ]
        _ = try writeImage(named: fileName, in: directory, type: .jpeg, rgba: rgba, properties: properties)
    }

    private func writeImage(
        named fileName: String,
        in directory: URL,
        type: UTType,
        rgba: [UInt8],
        properties: [CFString: Any]?
    ) throws -> URL {
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        let url = directory.appendingPathComponent(fileName)
        let colorSpace = CGColorSpaceCreateDeviceRGB()
        let pixel = Data(rgba)
        guard
            rgba.count == 4,
            let provider = CGDataProvider(data: pixel as CFData),
            let image = CGImage(
                width: 1,
                height: 1,
                bitsPerComponent: 8,
                bitsPerPixel: 32,
                bytesPerRow: 4,
                space: colorSpace,
                bitmapInfo: CGBitmapInfo(rawValue: CGImageAlphaInfo.premultipliedLast.rawValue),
                provider: provider,
                decode: nil,
                shouldInterpolate: false,
                intent: .defaultIntent
            ),
            let destination = CGImageDestinationCreateWithURL(url as CFURL, type.identifier as CFString, 1, nil)
        else {
            throw CocoaError(.fileWriteUnknown)
        }

        CGImageDestinationAddImage(destination, image, properties as CFDictionary?)
        guard CGImageDestinationFinalize(destination) else {
            throw CocoaError(.fileWriteUnknown)
        }
        return url
    }

    private func rgba(seed: Int) -> [UInt8] {
        [
            UInt8((seed * 53) % 255),
            UInt8((seed * 97 + 40) % 255),
            UInt8((seed * 193 + 80) % 255),
            255
        ]
    }
}
