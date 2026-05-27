// 中文注释：本测试覆盖素材右键导出的核心文件写入与转码行为。
import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers
import XCTest
@testable import Momento

final class AssetExportServiceTests: XCTestCase {
    private enum TestExportError: Error {
        case sourceUnavailable
    }

    func testExportOriginalCopiesStoredFile() throws {
        let directory = try temporaryDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }

        let sourceURL = directory.appendingPathComponent("source.png")
        try writeOnePixelPNG(to: sourceURL)
        let asset = try makeAsset(storageURL: sourceURL, displayName: "Reference")
        let destinationURL = directory.appendingPathComponent("Reference.png")

        let exportedURL = try AssetExportService().export(
            asset,
            configuration: AssetExportConfiguration(format: .original, jpegQuality: 0.9),
            to: destinationURL
        )

        XCTAssertEqual(exportedURL, destinationURL)
        XCTAssertEqual(try Data(contentsOf: exportedURL), try Data(contentsOf: sourceURL))
    }

    func testExportOriginalValidatesSourceBeforeReadingStoredFile() throws {
        let directory = try temporaryDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }

        let sourceURL = directory.appendingPathComponent("source.png")
        try writeOnePixelPNG(to: sourceURL)
        let asset = try makeAsset(storageURL: sourceURL, displayName: "Reference")
        let destinationURL = directory.appendingPathComponent("Reference.png")

        XCTAssertThrowsError(
            try AssetExportService().export(
                asset,
                configuration: AssetExportConfiguration(format: .original, jpegQuality: 0.9),
                to: destinationURL,
                sourceAccessValidator: {
                    throw TestExportError.sourceUnavailable
                }
            )
        ) { error in
            XCTAssertEqual(error as? TestExportError, .sourceUnavailable)
        }

        XCTAssertFalse(FileManager.default.fileExists(atPath: destinationURL.path))
    }

    func testExportJPEGWritesJPEGImage() throws {
        let directory = try temporaryDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }

        let sourceURL = directory.appendingPathComponent("source.png")
        try writeOnePixelPNG(to: sourceURL)
        let asset = try makeAsset(storageURL: sourceURL, displayName: "Reference")
        let destinationURL = directory.appendingPathComponent("Reference.jpg")

        let exportedURL = try AssetExportService().export(
            asset,
            configuration: AssetExportConfiguration(format: .jpeg, jpegQuality: 0.55),
            to: destinationURL
        )

        let source = try XCTUnwrap(CGImageSourceCreateWithURL(exportedURL as CFURL, nil))
        XCTAssertEqual(CGImageSourceGetType(source) as String?, UTType.jpeg.identifier)
    }

    func testExportMultipleAssetsUsesAvailableFileNames() throws {
        let directory = try temporaryDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }

        let firstSourceURL = directory.appendingPathComponent("first.png")
        let secondSourceURL = directory.appendingPathComponent("second.png")
        try writeOnePixelPNG(to: firstSourceURL)
        try writeOnePixelPNG(to: secondSourceURL)
        let exportDirectoryURL = directory.appendingPathComponent("exports", isDirectory: true)
        let firstAsset = try makeAsset(storageURL: firstSourceURL, displayName: "Reference")
        let secondAsset = try makeAsset(storageURL: secondSourceURL, displayName: "Reference")

        let exportedURLs = try AssetExportService().export(
            [firstAsset, secondAsset],
            configuration: AssetExportConfiguration(format: .original, jpegQuality: 0.9),
            toDirectory: exportDirectoryURL
        )

        XCTAssertEqual(exportedURLs.map(\.lastPathComponent), ["Reference.png", "Reference 2.png"])
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURLs[0].path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURLs[1].path))
    }

    func testExportMultipleAssetsValidatesBeforeEachStoredFileRead() throws {
        let directory = try temporaryDirectory()
        defer { try? FileManager.default.removeItem(at: directory) }

        let firstSourceURL = directory.appendingPathComponent("first.png")
        let secondSourceURL = directory.appendingPathComponent("second.png")
        try writeOnePixelPNG(to: firstSourceURL)
        try writeOnePixelPNG(to: secondSourceURL)
        let exportDirectoryURL = directory.appendingPathComponent("exports", isDirectory: true)
        let firstAsset = try makeAsset(storageURL: firstSourceURL, displayName: "First")
        let secondAsset = try makeAsset(storageURL: secondSourceURL, displayName: "Second")
        let markerURL = directory.appendingPathComponent("source-validated")

        XCTAssertThrowsError(
            try AssetExportService().export(
                [firstAsset, secondAsset],
                configuration: AssetExportConfiguration(format: .original, jpegQuality: 0.9),
                toDirectory: exportDirectoryURL,
                sourceAccessValidator: {
                    if FileManager.default.fileExists(atPath: markerURL.path) {
                        throw TestExportError.sourceUnavailable
                    }
                    try Data("validated".utf8).write(to: markerURL)
                }
            )
        ) { error in
            XCTAssertEqual(error as? TestExportError, .sourceUnavailable)
        }

        XCTAssertTrue(FileManager.default.fileExists(atPath: exportDirectoryURL.appendingPathComponent("First.png").path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: exportDirectoryURL.appendingPathComponent("Second.png").path))
    }

    private func temporaryDirectory() throws -> URL {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(
            "momento-export-tests-\(UUID().uuidString)",
            isDirectory: true
        )
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory
    }

    private func writeOnePixelPNG(to url: URL) throws {
        let colorSpace = CGColorSpaceCreateDeviceRGB()
        let provider = try XCTUnwrap(CGDataProvider(data: Data([255, 0, 0, 255]) as CFData))
        let bitmapInfo = CGBitmapInfo(rawValue: CGImageAlphaInfo.premultipliedLast.rawValue)
        let image = try XCTUnwrap(
            CGImage(
                width: 1,
                height: 1,
                bitsPerComponent: 8,
                bitsPerPixel: 32,
                bytesPerRow: 4,
                space: colorSpace,
                bitmapInfo: bitmapInfo,
                provider: provider,
                decode: nil,
                shouldInterpolate: false,
                intent: .defaultIntent
            )
        )
        let destination = try XCTUnwrap(
            CGImageDestinationCreateWithURL(url as CFURL, UTType.png.identifier as CFString, 1, nil)
        )
        CGImageDestinationAddImage(destination, image, nil)
        XCTAssertTrue(CGImageDestinationFinalize(destination))
    }

    private func makeAsset(storageURL: URL, displayName: String) throws -> AssetItem {
        let fileSize = try XCTUnwrap(
            FileManager.default.attributesOfItem(atPath: storageURL.path)[.size] as? NSNumber
        )
        return AssetItem(
            id: UUID().uuidString,
            libraryID: "library",
            displayName: displayName,
            originalURL: storageURL,
            storageURL: storageURL,
            kind: .image,
            fileExtension: "png",
            utiIdentifier: UTType.png.identifier,
            byteSize: fileSize.int64Value,
            contentHash: UUID().uuidString,
            dimensions: AssetDimensions(width: 1, height: 1),
            tags: [],
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
    }
}
