// 中文注释：本服务负责外部文件/文件夹导入，包含安全作用域访问、内容哈希去重、缩略图和元数据提取。
import CryptoKit
import Foundation
import ImageIO
import UniformTypeIdentifiers

nonisolated struct AssetImportBatch: Sendable {
    var newAssets: [AssetItem]
    var folderAssignmentsByContentHash: [String: [[String]]]
}

nonisolated enum AssetImportProgressPhase: Sendable, Equatable {
    case preparing
    case importing
    case finalizing
}

nonisolated struct AssetImportProgress: Sendable, Equatable {
    var phase: AssetImportProgressPhase
    var totalFileCount: Int?
    var processedFileCount: Int
    var importedFileCount: Int
    var skippedFileCount: Int
    var currentFileName: String?

    static func preparing() -> AssetImportProgress {
        AssetImportProgress(
            phase: .preparing,
            totalFileCount: nil,
            processedFileCount: 0,
            importedFileCount: 0,
            skippedFileCount: 0,
            currentFileName: nil
        )
    }
}

typealias AssetImportProgressHandler = @Sendable (AssetImportProgress) async -> Void

nonisolated private struct AssetImportCandidate: Sendable {
    var sourceURL: URL
    var rootURL: URL?
    var relativeFolderComponents: [String]
}

struct AssetImportService: Sendable {
    private let storage: LibraryStorage
    private let thumbnailService: AssetThumbnailService
    private let colorAnalysisService: AssetColorAnalysisService

    init(applicationSupportRoot: URL? = nil) {
        let storage = LibraryStorage(applicationSupportRoot: applicationSupportRoot)
        self.storage = storage
        self.thumbnailService = AssetThumbnailService(storage: storage)
        self.colorAnalysisService = AssetColorAnalysisService()
    }

    nonisolated func importItems(from urls: [URL], into library: AssetLibrary) async throws -> [AssetItem] {
        try await importItems(from: urls, into: library, excludingContentHashes: [])
    }

    nonisolated func importItems(
        from urls: [URL],
        into library: AssetLibrary,
        excludingContentHashes existingContentHashes: Set<String>,
        sourcePageURL: URL? = nil,
        progressHandler: AssetImportProgressHandler? = nil,
        libraryAccessValidator: (@Sendable () throws -> Void)? = nil
    ) async throws -> [AssetItem] {
        try await importBatch(
            from: urls,
            into: library,
            excludingContentHashes: existingContentHashes,
            sourcePageURL: sourcePageURL,
            progressHandler: progressHandler,
            libraryAccessValidator: libraryAccessValidator
        ).newAssets
    }

    nonisolated func importBatch(
        from urls: [URL],
        into library: AssetLibrary,
        excludingContentHashes existingContentHashes: Set<String>,
        sourcePageURL: URL? = nil,
        progressHandler: AssetImportProgressHandler? = nil,
        libraryAccessValidator: (@Sendable () throws -> Void)? = nil
    ) async throws -> AssetImportBatch {
        // 用户从 Finder 选择的文件/文件夹可能来自 sandbox 外部。访问权限必须覆盖
        // 后面的 detached import task，所以 scope 在 await 之前创建，并在整个导入
        // 完成后统一释放，而不是只包住收集 URL 的同步阶段。
        let scopes = urls.map(SourceAccessScope.init(url:))
        defer {
            scopes.forEach { $0.stop() }
        }

        return try await Task.detached(priority: .userInitiated) {
            var progressReporter = AssetImportProgressReporter(handler: progressHandler)
            await progressReporter.report(.preparing(), force: true)
            try libraryAccessValidator?()
            try storage.prepareLibraryDirectories(for: library)

            let candidates = try collectImportCandidates(from: urls)
            var imported: [AssetItem] = []
            var folderAssignmentsByContentHash: [String: [[String]]] = [:]
            var seenHashes = existingContentHashes
            var processedFileCount = 0
            var importedFileCount = 0
            var skippedFileCount = 0
            let totalFileCount = candidates.count

            await progressReporter.report(
                AssetImportProgress(
                    phase: .importing,
                    totalFileCount: totalFileCount,
                    processedFileCount: processedFileCount,
                    importedFileCount: importedFileCount,
                    skippedFileCount: skippedFileCount,
                    currentFileName: candidates.first?.sourceURL.lastPathComponent
                ),
                force: true
            )

            for candidate in candidates {
                let fileURL = candidate.sourceURL
                guard let kind = assetKind(for: fileURL) else {
                    processedFileCount += 1
                    skippedFileCount += 1
                    await progressReporter.report(
                        AssetImportProgress(
                            phase: .importing,
                            totalFileCount: totalFileCount,
                            processedFileCount: processedFileCount,
                            importedFileCount: importedFileCount,
                            skippedFileCount: skippedFileCount,
                            currentFileName: nextFileName(after: processedFileCount, in: candidates)
                        ),
                        force: processedFileCount == totalFileCount
                    )
                    continue
                }

                let hash = try contentHash(for: fileURL)
                if !candidate.relativeFolderComponents.isEmpty {
                    var assignedPaths = folderAssignmentsByContentHash[hash, default: []]
                    if !assignedPaths.contains(candidate.relativeFolderComponents) {
                        assignedPaths.append(candidate.relativeFolderComponents)
                        folderAssignmentsByContentHash[hash] = assignedPaths
                    }
                }

                // 导入阶段先用 hash 做一次批内和库内去重，避免重复复制物理文件；
                // Core Data 层仍有唯一约束，负责处理并发或历史数据带来的最终一致性。
                guard seenHashes.insert(hash).inserted else {
                    processedFileCount += 1
                    skippedFileCount += 1
                    await progressReporter.report(
                        AssetImportProgress(
                            phase: .importing,
                            totalFileCount: totalFileCount,
                            processedFileCount: processedFileCount,
                            importedFileCount: importedFileCount,
                            skippedFileCount: skippedFileCount,
                            currentFileName: nextFileName(after: processedFileCount, in: candidates)
                        ),
                        force: processedFileCount == totalFileCount
                    )
                    continue
                }

                let fileExtension = fileURL.pathExtension.lowercased()
                let destination = storage.assetStorageURL(
                    forContentHash: hash,
                    fileExtension: fileExtension,
                    in: library
                )

                try libraryAccessValidator?()
                if !FileManager.default.fileExists(atPath: destination.path) {
                    try libraryAccessValidator?()
                    try FileManager.default.createDirectory(
                        at: destination.deletingLastPathComponent(),
                        withIntermediateDirectories: true
                    )
                    try libraryAccessValidator?()
                    try FileManager.default.copyItem(at: fileURL, to: destination)
                }

                try libraryAccessValidator?()
                let thumbnailURL = try? thumbnailService.generateThumbnail(
                    for: destination,
                    contentHash: hash,
                    in: library
                )
                let paletteSourceURL = thumbnailURL ?? destination
                try libraryAccessValidator?()
                let paletteColors = colorAnalysisService.paletteColors(
                    for: paletteSourceURL,
                    libraryID: library.id,
                    assetID: hash,
                    maxColorCount: 8
                )

                let values = try fileURL.resourceValues(forKeys: [.fileSizeKey])
                let imageProperties = imageImportProperties(for: fileURL)
                let importedAt = Date()
                imported.append(
                    AssetItem(
                        id: hash,
                        libraryID: library.id,
                        displayName: fileURL.deletingPathExtension().lastPathComponent,
                        originalFileName: fileURL.lastPathComponent,
                        originalURL: fileURL,
                        sourcePageURL: sourcePageURL,
                        storageURL: destination,
                        kind: kind,
                        fileExtension: fileExtension,
                        utiIdentifier: UTType(filenameExtension: fileExtension)?.identifier,
                        byteSize: Int64(values.fileSize ?? 0),
                        contentHash: hash,
                        dimensions: imageProperties.dimensions,
                        exifMetadata: imageProperties.exifMetadata,
                        colorProfileName: imageProperties.exifMetadata?.profileName,
                        tags: [],
                        paletteColors: paletteColors,
                        thumbnailURL: thumbnailURL,
                        isFavorite: false,
                        importedAt: importedAt,
                        updatedAt: importedAt
                    )
                )
                processedFileCount += 1
                importedFileCount += 1
                await progressReporter.report(
                    AssetImportProgress(
                        phase: .importing,
                        totalFileCount: totalFileCount,
                        processedFileCount: processedFileCount,
                        importedFileCount: importedFileCount,
                        skippedFileCount: skippedFileCount,
                        currentFileName: nextFileName(after: processedFileCount, in: candidates)
                    ),
                    force: processedFileCount == totalFileCount
                )
            }

            await progressReporter.report(
                AssetImportProgress(
                    phase: .finalizing,
                    totalFileCount: totalFileCount,
                    processedFileCount: processedFileCount,
                    importedFileCount: importedFileCount,
                    skippedFileCount: skippedFileCount,
                    currentFileName: nil
                ),
                force: true
            )

            return AssetImportBatch(
                newAssets: imported,
                folderAssignmentsByContentHash: folderAssignmentsByContentHash
            )
        }.value
    }
}

nonisolated private struct AssetImportProgressReporter: Sendable {
    private let handler: AssetImportProgressHandler?
    private let minimumReportInterval: TimeInterval = 0.1
    private var lastReportedAt: Date?

    init(handler: AssetImportProgressHandler?) {
        self.handler = handler
    }

    mutating func report(_ progress: AssetImportProgress, force: Bool = false) async {
        guard let handler else {
            return
        }

        let now = Date()
        if force || shouldReportProgress(now: now) {
            lastReportedAt = now
            await handler(progress)
        }
    }

    private func shouldReportProgress(now: Date) -> Bool {
        guard let lastReportedAt else {
            return true
        }

        return now.timeIntervalSince(lastReportedAt) >= minimumReportInterval
    }
}

nonisolated private func nextFileName(
    after processedFileCount: Int,
    in candidates: [AssetImportCandidate]
) -> String? {
    guard processedFileCount < candidates.count else {
        return nil
    }

    return candidates[processedFileCount].sourceURL.lastPathComponent
}

nonisolated private final class SourceAccessScope: @unchecked Sendable {
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

nonisolated private func collectImportCandidates(from urls: [URL]) throws -> [AssetImportCandidate] {
    var candidates: [AssetImportCandidate] = []

    for url in urls {
        var isDirectory: ObjCBool = false
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

            for case let fileURL as URL in enumerator {
                guard isRegularFile(fileURL), isSupportedAsset(fileURL) else {
                    continue
                }
                candidates.append(
                    AssetImportCandidate(
                        sourceURL: fileURL,
                        rootURL: url,
                        relativeFolderComponents: relativeFolderComponents(for: fileURL, rootURL: url)
                    )
                )
            }
        } else if isSupportedAsset(url) {
            candidates.append(
                AssetImportCandidate(
                    sourceURL: url,
                    rootURL: nil,
                    relativeFolderComponents: []
                )
            )
        }
    }

    return candidates.sorted { $0.sourceURL.path < $1.sourceURL.path }
}

nonisolated private func isRegularFile(_ url: URL) -> Bool {
    (try? url.resourceValues(forKeys: [.isRegularFileKey]).isRegularFile) == true
}

nonisolated private func relativeFolderComponents(for fileURL: URL, rootURL: URL) -> [String] {
    let rootComponents = rootURL.standardizedFileURL.pathComponents
    let parentComponents = fileURL.deletingLastPathComponent().standardizedFileURL.pathComponents

    guard parentComponents.count >= rootComponents.count,
          Array(parentComponents.prefix(rootComponents.count)) == rootComponents else {
        return []
    }

    return parentComponents.dropFirst(rootComponents.count).filter { component in
        !component.isEmpty && component != "/"
    }
}

nonisolated private func isSupportedAsset(_ url: URL) -> Bool {
    assetKind(for: url) != nil
}

nonisolated private func assetKind(for url: URL) -> AssetKind? {
    let fileExtension = url.pathExtension.lowercased()

    if fileExtension == "gif" {
        return .gif
    }

    if fileExtension == "svg" || fileExtension == "svgz" {
        return nil
    }

    guard let type = UTType(filenameExtension: fileExtension) else {
        return nil
    }

    if type.conforms(to: .image) {
        return .image
    }

    return nil
}

nonisolated private func contentHash(for url: URL) throws -> String {
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

nonisolated private struct ImageImportProperties: Sendable {
    var dimensions: AssetDimensions?
    var exifMetadata: AssetExifMetadata?
}

nonisolated private func imageImportProperties(for url: URL) -> ImageImportProperties {
    let resourceValues = try? url.resourceValues(forKeys: [.creationDateKey, .contentModificationDateKey])
    guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
          let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any] else {
        let metadata = AssetExifMetadata(
            fileCreatedAt: resourceValues?.creationDate,
            fileModifiedAt: resourceValues?.contentModificationDate
        )
        return ImageImportProperties(
            dimensions: nil,
            exifMetadata: metadata.isEmpty ? nil : metadata
        )
    }

    return ImageImportProperties(
        dimensions: imageDimensions(from: properties),
        exifMetadata: imageExifMetadata(from: properties, resourceValues: resourceValues)
    )
}

nonisolated private func imageDimensions(from properties: [CFString: Any]) -> AssetDimensions? {
    guard let width = properties[kCGImagePropertyPixelWidth] as? Int,
          let height = properties[kCGImagePropertyPixelHeight] as? Int else {
        return nil
    }

    return AssetDimensions(width: width, height: height)
}

nonisolated private func imageExifMetadata(
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

nonisolated private func stringValue(_ value: Any?) -> String? {
    guard let string = value as? String else {
        return nil
    }

    let trimmed = string.trimmingCharacters(in: .whitespacesAndNewlines)
    return trimmed.isEmpty ? nil : trimmed
}

nonisolated private func intValue(_ value: Any?) -> Int? {
    if let value = value as? Int {
        return value
    }
    if let value = value as? NSNumber {
        return value.intValue
    }
    return nil
}

nonisolated private func doubleValue(_ value: Any?) -> Double? {
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

nonisolated private func intArrayValue(_ value: Any?) -> [Int] {
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

nonisolated private func dateTimeValue(_ value: Any?) -> Date? {
    guard let string = stringValue(value) else {
        return nil
    }

    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.dateFormat = "yyyy:MM:dd HH:mm:ss"
    return formatter.date(from: string)
}
