import CryptoKit
import Foundation
import ImageIO
import UniformTypeIdentifiers

nonisolated struct CloudImportResult: Hashable, Sendable {
    var assets: [AssetItem]
    var skippedCount: Int
}

nonisolated protocol CloudImporting: Sendable {
    func importFiles(
        from urls: [URL],
        into library: AssetLibrary,
        excludingContentHashes existingContentHashes: Set<String>,
        sourcePageURL: URL?
    ) async throws -> CloudImportResult
}

nonisolated struct CloudImportService: CloudImporting {
    private let cachePaths: CloudLibraryCachePaths
    private let thumbnailService: CloudImportThumbnailService
    private let colorAnalysisService: AssetColorAnalysisService

    init(cachePaths: CloudLibraryCachePaths = CloudLibraryCachePaths()) {
        self.cachePaths = cachePaths
        self.thumbnailService = CloudImportThumbnailService(cachePaths: cachePaths)
        self.colorAnalysisService = AssetColorAnalysisService()
    }

    func importFiles(
        from urls: [URL],
        into library: AssetLibrary,
        excludingContentHashes existingContentHashes: Set<String> = [],
        sourcePageURL: URL? = nil
    ) async throws -> CloudImportResult {
        let scopes = urls.map(CloudImportSourceAccessScope.init(url:))
        defer {
            scopes.forEach { $0.stop() }
        }

        return try await Task.detached(priority: .userInitiated) {
            try importFilesSync(
                from: urls,
                into: library,
                excludingContentHashes: existingContentHashes,
                sourcePageURL: sourcePageURL
            )
        }.value
    }

    private func importFilesSync(
        from urls: [URL],
        into library: AssetLibrary,
        excludingContentHashes existingContentHashes: Set<String>,
        sourcePageURL: URL?
    ) throws -> CloudImportResult {
        let cloudAccountID = try requiredCloudAccountID(in: library)
        try cachePaths.prepareLibraryDirectories(cloudAccountID: cloudAccountID, libraryID: library.id)

        var seenHashes = existingContentHashes
        var importedAssets: [AssetItem] = []
        var skippedCount = 0

        for url in try Self.collectImportCandidates(from: urls) {
            guard let kind = Self.assetKind(for: url) else {
                skippedCount += 1
                continue
            }

            let contentHash = try Self.contentHash(for: url)
            guard seenHashes.insert(contentHash).inserted else {
                skippedCount += 1
                continue
            }

            let fileExtension = url.pathExtension.lowercased()
            let destinationURL = try cachePaths.originalURL(
                cloudAccountID: cloudAccountID,
                libraryID: library.id,
                contentHash: contentHash,
                fileExtension: fileExtension
            )
            try FileManager.default.createDirectory(
                at: destinationURL.deletingLastPathComponent(),
                withIntermediateDirectories: true
            )
            if !FileManager.default.fileExists(atPath: destinationURL.path) {
                try FileManager.default.copyItem(at: url, to: destinationURL)
                try cachePaths.applyDurabilityAttributes(to: destinationURL, role: .uploadPendingOriginal)
            }

            let thumbnailURL = try thumbnailService.generateThumbnail(
                for: destinationURL,
                contentHash: contentHash,
                in: library
            )
            let paletteSourceURL = thumbnailURL ?? destinationURL
            let paletteColors = colorAnalysisService.paletteColors(
                for: paletteSourceURL,
                libraryID: library.id,
                assetID: contentHash,
                maxColorCount: 8
            )
            let resourceValues = try url.resourceValues(forKeys: [.fileSizeKey])
            let imageProperties = Self.imageImportProperties(for: url)
            let importedAt = Date()

            importedAssets.append(
                AssetItem(
                    id: contentHash,
                    libraryID: library.id,
                    displayName: url.deletingPathExtension().lastPathComponent,
                    originalFileName: url.lastPathComponent,
                    originalURL: url,
                    sourcePageURL: sourcePageURL,
                    storageURL: destinationURL,
                    kind: kind,
                    fileExtension: fileExtension,
                    utiIdentifier: UTType(filenameExtension: fileExtension)?.identifier,
                    byteSize: Int64(resourceValues.fileSize ?? 0),
                    contentHash: contentHash,
                    dimensions: imageProperties.dimensions,
                    exifMetadata: imageProperties.exifMetadata,
                    orientation: imageProperties.orientation,
                    colorProfileName: imageProperties.colorProfileName,
                    tags: [],
                    folderIDs: [],
                    paletteColors: paletteColors,
                    thumbnailURL: thumbnailURL,
                    isFavorite: false,
                    importedAt: importedAt,
                    updatedAt: importedAt,
                    availability: AssetFileAvailability(
                        original: .uploadPending,
                        thumbnail: thumbnailURL == nil ? .generationPending : .local,
                        lastError: nil
                    ),
                    syncState: .syncing
                )
            )
        }

        return CloudImportResult(assets: importedAssets, skippedCount: skippedCount)
    }

    private func requiredCloudAccountID(in library: AssetLibrary) throws -> String {
        guard let cloudAccountID = library.cloudAccountID?.trimmingCharacters(in: .whitespacesAndNewlines),
              !cloudAccountID.isEmpty else {
            throw CloudImportError.missingCloudAccountID(libraryID: library.id)
        }
        return cloudAccountID
    }

    private static func collectImportCandidates(from urls: [URL]) throws -> [URL] {
        var candidates: [URL] = []
        for url in urls {
            var isDirectory = ObjCBool(false)
            guard FileManager.default.fileExists(atPath: url.path, isDirectory: &isDirectory) else {
                continue
            }

            if isDirectory.boolValue {
                guard let enumerator = FileManager.default.enumerator(
                    at: url,
                    includingPropertiesForKeys: [.isRegularFileKey],
                    options: [.skipsHiddenFiles, .skipsPackageDescendants]
                ) else {
                    continue
                }
                for case let fileURL as URL in enumerator where isRegularFile(fileURL) && assetKind(for: fileURL) != nil {
                    candidates.append(fileURL)
                }
            } else if assetKind(for: url) != nil {
                candidates.append(url)
            }
        }
        return candidates.sorted { $0.path < $1.path }
    }

    private static func isRegularFile(_ url: URL) -> Bool {
        (try? url.resourceValues(forKeys: [.isRegularFileKey]).isRegularFile) == true
    }

    private static func assetKind(for url: URL) -> AssetKind? {
        let fileExtension = url.pathExtension.lowercased()
        if fileExtension == "gif" {
            return .gif
        }
        if fileExtension == "svg" || fileExtension == "svgz" {
            return .svg
        }

        guard let type = UTType(filenameExtension: fileExtension) else {
            return nil
        }
        if type.conforms(to: .image) {
            return .image
        }
        if type.conforms(to: .movie) || type.conforms(to: .video) {
            return .video
        }
        if type.conforms(to: .pdf) {
            return .pdf
        }
        return nil
    }

    private static func contentHash(for url: URL) throws -> String {
        let handle = try FileHandle(forReadingFrom: url)
        defer {
            try? handle.close()
        }

        var hasher = SHA256()
        while true {
            let data = try handle.read(upToCount: 1024 * 1024) ?? Data()
            if data.isEmpty {
                break
            }
            hasher.update(data: data)
        }
        return hasher.finalize().map { String(format: "%02x", $0) }.joined()
    }

    private static func imageImportProperties(for url: URL) -> CloudImageImportProperties {
        let resourceValues = try? url.resourceValues(forKeys: [.creationDateKey, .contentModificationDateKey])
        guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
              let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any] else {
            let metadata = AssetExifMetadata(
                fileCreatedAt: resourceValues?.creationDate,
                fileModifiedAt: resourceValues?.contentModificationDate
            )
            return CloudImageImportProperties(
                dimensions: nil,
                exifMetadata: metadata.isEmpty ? nil : metadata,
                orientation: nil,
                colorProfileName: nil
            )
        }

        let metadata = imageExifMetadata(from: properties, resourceValues: resourceValues)
        let colorProfileName = metadata?.profileName
            ?? stringValue(properties[kCGImagePropertyProfileName])
            ?? stringValue(properties[kCGImagePropertyNamedColorSpace])
        return CloudImageImportProperties(
            dimensions: imageDimensions(from: properties),
            exifMetadata: metadata,
            orientation: intValue(properties[kCGImagePropertyOrientation]),
            colorProfileName: colorProfileName
        )
    }

    private static func imageDimensions(from properties: [CFString: Any]) -> AssetDimensions? {
        guard let width = intValue(properties[kCGImagePropertyPixelWidth]),
              let height = intValue(properties[kCGImagePropertyPixelHeight]) else {
            return nil
        }
        return AssetDimensions(width: width, height: height)
    }

    private static func imageExifMetadata(
        from properties: [CFString: Any],
        resourceValues: URLResourceValues?
    ) -> AssetExifMetadata? {
        let exif = properties[kCGImagePropertyExifDictionary] as? [CFString: Any] ?? [:]
        let tiff = properties[kCGImagePropertyTIFFDictionary] as? [CFString: Any] ?? [:]
        let contentCreatedAt = dateTimeValue(exif[kCGImagePropertyExifDateTimeOriginal])
            ?? dateTimeValue(exif[kCGImagePropertyExifDateTimeDigitized])
            ?? dateTimeValue(tiff[kCGImagePropertyTIFFDateTime])
        let flashValue = intValue(exif[kCGImagePropertyExifFlash])
        let metadata = AssetExifMetadata(
            fileCreatedAt: resourceValues?.creationDate,
            fileModifiedAt: resourceValues?.contentModificationDate,
            contentCreatedAt: contentCreatedAt,
            pixelWidth: intValue(properties[kCGImagePropertyPixelWidth]),
            pixelHeight: intValue(properties[kCGImagePropertyPixelHeight]),
            dpiWidth: doubleValue(properties[kCGImagePropertyDPIWidth]),
            dpiHeight: doubleValue(properties[kCGImagePropertyDPIHeight]),
            colorModel: stringValue(properties[kCGImagePropertyColorModel]),
            profileName: stringValue(properties[kCGImagePropertyProfileName])
                ?? stringValue(properties[kCGImagePropertyNamedColorSpace]),
            cameraMake: stringValue(tiff[kCGImagePropertyTIFFMake]),
            cameraModel: stringValue(tiff[kCGImagePropertyTIFFModel]),
            lensModel: stringValue(exif[kCGImagePropertyExifLensModel]),
            exposureTime: doubleValue(exif[kCGImagePropertyExifExposureTime]),
            focalLength: doubleValue(exif[kCGImagePropertyExifFocalLength]),
            isoSpeedRatings: intArrayValue(exif[kCGImagePropertyExifISOSpeedRatings]),
            flashFired: flashValue.map { ($0 & 1) == 1 },
            fNumber: doubleValue(exif[kCGImagePropertyExifFNumber]),
            exposureProgram: intValue(exif[kCGImagePropertyExifExposureProgram]),
            meteringMode: intValue(exif[kCGImagePropertyExifMeteringMode]),
            whiteBalance: intValue(exif[kCGImagePropertyExifWhiteBalance]),
            creator: stringValue(tiff[kCGImagePropertyTIFFArtist])
        )
        return metadata.isEmpty ? nil : metadata
    }

    private static func stringValue(_ value: Any?) -> String? {
        guard let string = value as? String else {
            return nil
        }
        let trimmed = string.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func intValue(_ value: Any?) -> Int? {
        if let value = value as? Int {
            return value
        }
        if let value = value as? NSNumber {
            return value.intValue
        }
        return nil
    }

    private static func doubleValue(_ value: Any?) -> Double? {
        if let value = value as? Double {
            return value
        }
        if let value = value as? Float {
            return Double(value)
        }
        if let value = value as? Int {
            return Double(value)
        }
        if let value = value as? NSNumber {
            return value.doubleValue
        }
        return nil
    }

    private static func intArrayValue(_ value: Any?) -> [Int] {
        if let values = value as? [Int] {
            return values
        }
        if let values = value as? [NSNumber] {
            return values.map(\.intValue)
        }
        if let value = intValue(value) {
            return [value]
        }
        return []
    }

    private static func dateTimeValue(_ value: Any?) -> Date? {
        guard let string = stringValue(value) else {
            return nil
        }
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.dateFormat = "yyyy:MM:dd HH:mm:ss"
        return formatter.date(from: string)
    }
}

nonisolated private struct CloudImageImportProperties: Sendable {
    var dimensions: AssetDimensions?
    var exifMetadata: AssetExifMetadata?
    var orientation: Int?
    var colorProfileName: String?
}

nonisolated private final class CloudImportSourceAccessScope: @unchecked Sendable {
    private let url: URL
    private let didStartAccessing: Bool

    init(url: URL) {
        self.url = url
        self.didStartAccessing = url.startAccessingSecurityScopedResource()
    }

    func stop() {
        if didStartAccessing {
            url.stopAccessingSecurityScopedResource()
        }
    }
}

nonisolated private struct CloudImportThumbnailService: Sendable {
    private let cachePaths: CloudLibraryCachePaths

    init(cachePaths: CloudLibraryCachePaths) {
        self.cachePaths = cachePaths
    }

    func generateThumbnail(for sourceURL: URL, contentHash: String, in library: AssetLibrary) throws -> URL? {
        guard library.storageMode == .cloud,
              sourceURL.pathExtension.lowercased() != "svg",
              let cloudAccountID = library.cloudAccountID,
              let source = CGImageSourceCreateWithURL(sourceURL as CFURL, nil) else {
            return nil
        }

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: 512
        ]
        guard let thumbnail = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return nil
        }

        let destinationURL = try cachePaths.thumbnailURL(
            cloudAccountID: cloudAccountID,
            libraryID: library.id,
            contentHash: contentHash
        )
        try FileManager.default.createDirectory(
            at: destinationURL.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        guard let destination = CGImageDestinationCreateWithURL(
            destinationURL as CFURL,
            UTType.png.identifier as CFString,
            1,
            nil
        ) else {
            return nil
        }

        CGImageDestinationAddImage(destination, thumbnail, nil)
        guard CGImageDestinationFinalize(destination) else {
            return nil
        }
        try cachePaths.applyDurabilityAttributes(to: destinationURL, role: .thumbnail)
        return destinationURL
    }
}

enum CloudImportError: LocalizedError, Equatable {
    case missingCloudAccountID(libraryID: String)

    var errorDescription: String? {
        switch self {
        case .missingCloudAccountID(let libraryID):
            "Cloud library \(libraryID) is missing its iCloud account identity."
        }
    }
}
