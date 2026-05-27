// 中文注释：本服务负责把资源库内素材导出到用户选择的位置，并按需转码图片格式。
import Foundation
import ImageIO
import UniformTypeIdentifiers

nonisolated enum AssetExportFormat: String, CaseIterable, Identifiable, Sendable {
    case original
    case jpeg
    case png

    var id: String {
        rawValue
    }

    var titleKey: String {
        switch self {
        case .original:
            "Original File"
        case .jpeg:
            "JPEG Image"
        case .png:
            "PNG Image"
        }
    }

    var subtitleKey: String {
        switch self {
        case .original:
            "Copy the stored file without conversion."
        case .jpeg:
            "Export a compressed JPEG file."
        case .png:
            "Export a PNG file."
        }
    }

    var systemImageName: String {
        switch self {
        case .original:
            "doc"
        case .jpeg:
            "photo"
        case .png:
            "photo.on.rectangle"
        }
    }

    func contentType(for asset: AssetItem) -> UTType {
        switch self {
        case .original:
            UTType(asset.utiIdentifier)
                ?? UTType(filenameExtension: asset.fileExtension)
                ?? .data
        case .jpeg:
            .jpeg
        case .png:
            .png
        }
    }

    func fileExtension(for asset: AssetItem) -> String {
        switch self {
        case .original:
            asset.fileExtension.trimmingCharacters(in: .whitespacesAndNewlines)
        case .jpeg:
            "jpg"
        case .png:
            "png"
        }
    }
}

nonisolated struct AssetExportConfiguration: Sendable {
    var format: AssetExportFormat
    var jpegQuality: Double

    static let `default` = AssetExportConfiguration(format: .original, jpegQuality: 0.9)
}

nonisolated enum AssetExportError: Error {
    case emptySelection
    case unsupportedFormat
}

nonisolated struct AssetExportService: Sendable {
    func export(
        _ asset: AssetItem,
        configuration: AssetExportConfiguration,
        to destinationURL: URL,
        sourceAccessValidator: (@Sendable () throws -> Void)? = nil
    ) throws -> URL {
        try export(
            asset,
            configuration: configuration,
            to: destinationURL,
            replacingExistingFile: true,
            sourceAccessValidator: sourceAccessValidator
        )
    }

    func export(
        _ assets: [AssetItem],
        configuration: AssetExportConfiguration,
        toDirectory directoryURL: URL,
        sourceAccessValidator: (@Sendable () throws -> Void)? = nil
    ) throws -> [URL] {
        guard !assets.isEmpty else {
            throw AssetExportError.emptySelection
        }

        try FileManager.default.createDirectory(at: directoryURL, withIntermediateDirectories: true)

        return try assets.map { asset in
            let requestedDestination = directoryURL.appendingPathComponent(
                Self.fileName(for: asset, format: configuration.format)
            )
            return try export(
                asset,
                configuration: configuration,
                to: availableDestinationURL(for: requestedDestination),
                replacingExistingFile: false,
                sourceAccessValidator: sourceAccessValidator
            )
        }
    }

    static func fileName(for asset: AssetItem, format: AssetExportFormat) -> String {
        let displayName = sanitizedFileName(asset.displayName)
        let originalName = sanitizedFileName(asset.originalFileName)
        let fallbackName = asset.storageURL.deletingPathExtension().lastPathComponent
        let baseName = displayName.isEmpty ? (originalName.isEmpty ? fallbackName : originalName) : displayName
        let pathExtension = format.fileExtension(for: asset)

        guard !pathExtension.isEmpty else {
            return baseName
        }
        return "\(baseName).\(pathExtension)"
    }

    private func export(
        _ asset: AssetItem,
        configuration: AssetExportConfiguration,
        to destinationURL: URL,
        replacingExistingFile: Bool,
        sourceAccessValidator: (@Sendable () throws -> Void)?
    ) throws -> URL {
        let temporaryURL = destinationURL
            .deletingLastPathComponent()
            .appendingPathComponent(".\(destinationURL.lastPathComponent).tmp-\(UUID().uuidString)")

        do {
            try FileManager.default.createDirectory(
                at: destinationURL.deletingLastPathComponent(),
                withIntermediateDirectories: true
            )
            switch configuration.format {
            case .original:
                try sourceAccessValidator?()
                try FileManager.default.copyItem(at: asset.storageURL, to: temporaryURL)
            case .jpeg, .png:
                try sourceAccessValidator?()
                try writeConvertedImage(asset, configuration: configuration, to: temporaryURL)
            }

            if replacingExistingFile, FileManager.default.fileExists(atPath: destinationURL.path) {
                try FileManager.default.removeItem(at: destinationURL)
            }
            try FileManager.default.moveItem(at: temporaryURL, to: destinationURL)
            return destinationURL
        } catch {
            try? FileManager.default.removeItem(at: temporaryURL)
            throw error
        }
    }

    private func writeConvertedImage(
        _ asset: AssetItem,
        configuration: AssetExportConfiguration,
        to destinationURL: URL
    ) throws {
        guard let source = CGImageSourceCreateWithURL(asset.storageURL as CFURL, nil) else {
            throw AssetExportError.unsupportedFormat
        }

        let contentType = configuration.format.contentType(for: asset)
        guard let destination = CGImageDestinationCreateWithURL(
            destinationURL as CFURL,
            contentType.identifier as CFString,
            1,
            nil
        ) else {
            throw AssetExportError.unsupportedFormat
        }

        let options: [CFString: Any]
        if configuration.format == .jpeg {
            options = [
                kCGImageDestinationLossyCompressionQuality: max(0, min(configuration.jpegQuality, 1))
            ]
        } else {
            options = [:]
        }

        CGImageDestinationAddImageFromSource(destination, source, 0, options as CFDictionary)
        guard CGImageDestinationFinalize(destination) else {
            throw AssetExportError.unsupportedFormat
        }
    }

    private func availableDestinationURL(for url: URL) -> URL {
        guard FileManager.default.fileExists(atPath: url.path) else {
            return url
        }

        let directoryURL = url.deletingLastPathComponent()
        let baseName = url.deletingPathExtension().lastPathComponent
        let pathExtension = url.pathExtension
        var index = 2

        while true {
            let candidateName = pathExtension.isEmpty
                ? "\(baseName) \(index)"
                : "\(baseName) \(index).\(pathExtension)"
            let candidate = directoryURL.appendingPathComponent(candidateName)
            if !FileManager.default.fileExists(atPath: candidate.path) {
                return candidate
            }
            index += 1
        }
    }

    private static func sanitizedFileName(_ name: String) -> String {
        name
            .replacingOccurrences(of: "/", with: "-")
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
