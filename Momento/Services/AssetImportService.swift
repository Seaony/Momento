import CryptoKit
import Foundation
import ImageIO
import UniformTypeIdentifiers

struct AssetImportService: Sendable {
    private let storage: LibraryStorage

    init(applicationSupportRoot: URL? = nil) {
        self.storage = LibraryStorage(applicationSupportRoot: applicationSupportRoot)
    }

    nonisolated func importItems(from urls: [URL], into library: AssetLibrary) async throws -> [AssetItem] {
        try await Task.detached(priority: .userInitiated) {
            try storage.prepareLibraryDirectories(for: library)

            let files = try collectSupportedFiles(from: urls)
            var imported: [AssetItem] = []
            var seenHashes = Set<String>()

            for fileURL in files {
                guard let kind = assetKind(for: fileURL) else {
                    continue
                }

                let hash = try contentHash(for: fileURL)
                guard seenHashes.insert(hash).inserted else {
                    continue
                }

                let fileExtension = fileURL.pathExtension.lowercased()
                let destination = storage.assetsURL(for: library)
                    .appendingPathComponent(hash)
                    .appendingPathExtension(fileExtension)

                if !FileManager.default.fileExists(atPath: destination.path) {
                    try FileManager.default.copyItem(at: fileURL, to: destination)
                }

                let values = try fileURL.resourceValues(forKeys: [.fileSizeKey])
                imported.append(
                    AssetItem(
                        id: hash,
                        libraryID: library.id,
                        displayName: fileURL.deletingPathExtension().lastPathComponent,
                        originalURL: fileURL,
                        storageURL: destination,
                        kind: kind,
                        fileExtension: fileExtension,
                        byteSize: Int64(values.fileSize ?? 0),
                        contentHash: hash,
                        dimensions: imageDimensions(for: fileURL),
                        tags: [],
                        isFavorite: false,
                        importedAt: Date()
                    )
                )
            }

            return imported
        }.value
    }
}

nonisolated private func collectSupportedFiles(from urls: [URL]) throws -> [URL] {
    var files: [URL] = []

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

            for case let fileURL as URL in enumerator where isSupportedAsset(fileURL) {
                files.append(fileURL)
            }
        } else if isSupportedAsset(url) {
            files.append(url)
        }
    }

    return files.sorted { $0.path < $1.path }
}

nonisolated private func isSupportedAsset(_ url: URL) -> Bool {
    assetKind(for: url) != nil
}

nonisolated private func assetKind(for url: URL) -> AssetKind? {
    let fileExtension = url.pathExtension.lowercased()

    if fileExtension == "gif" {
        return .gif
    }

    if fileExtension == "svg" {
        return .svg
    }

    if fileExtension == "pdf" {
        return .pdf
    }

    guard let type = UTType(filenameExtension: fileExtension) else {
        return nil
    }

    if type.conforms(to: .movie) {
        return .video
    }

    if type.conforms(to: .image) {
        return .image
    }

    if type.conforms(to: .pdf) {
        return .pdf
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

nonisolated private func imageDimensions(for url: URL) -> AssetDimensions? {
    guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
          let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any] else {
        return nil
    }

    guard let width = properties[kCGImagePropertyPixelWidth] as? Int,
          let height = properties[kCGImagePropertyPixelHeight] as? Int else {
        return nil
    }

    return AssetDimensions(width: width, height: height)
}
