// 中文注释：本服务负责把可重建缩略图写入资源库缓存目录，原始素材仍由 assets 目录保存。
import Foundation
import ImageIO
import UniformTypeIdentifiers

struct AssetThumbnailService: Sendable {
    private let storage: LibraryStorage
    private let maxPixelSize: Int

    init(storage: LibraryStorage = LibraryStorage(), maxPixelSize: Int = 512) {
        self.storage = storage
        self.maxPixelSize = maxPixelSize
    }

    nonisolated func generateThumbnail(
        for sourceURL: URL,
        contentHash: String,
        in library: AssetLibrary
    ) throws -> URL? {
        guard let source = CGImageSourceCreateWithURL(sourceURL as CFURL, nil) else {
            return nil
        }

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: maxPixelSize
        ]

        guard let thumbnail = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return nil
        }

        let destinationURL = storage.thumbnailURL(forContentHash: contentHash, in: library)
        try FileManager.default.createDirectory(
            at: destinationURL.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )

        let temporaryURL = destinationURL
            .deletingLastPathComponent()
            .appendingPathComponent(".\(destinationURL.lastPathComponent).tmp-\(UUID().uuidString)")

        guard let destination = CGImageDestinationCreateWithURL(
            temporaryURL as CFURL,
            UTType.png.identifier as CFString,
            1,
            nil
        ) else {
            return nil
        }

        CGImageDestinationAddImage(destination, thumbnail, nil)
        guard CGImageDestinationFinalize(destination) else {
            try? FileManager.default.removeItem(at: temporaryURL)
            return nil
        }

        if FileManager.default.fileExists(atPath: destinationURL.path) {
            try FileManager.default.removeItem(at: destinationURL)
        }
        try FileManager.default.moveItem(at: temporaryURL, to: destinationURL)
        return destinationURL
    }
}
