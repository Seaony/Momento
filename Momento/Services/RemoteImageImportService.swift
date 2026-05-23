// 中文注释：本服务下载浏览器传来的远程图片到临时文件，再交给常规导入管线处理。
import Foundation
import UniformTypeIdentifiers

nonisolated enum RemoteImageImportError: LocalizedError, Sendable {
    case unsupportedURLScheme
    case downloadFailed
    case unsupportedImageContent

    var errorDescription: String? {
        switch self {
        case .unsupportedURLScheme:
            "Only HTTP or HTTPS image URLs can be imported from Chrome."
        case .downloadFailed:
            "Momento could not download the image from Chrome."
        case .unsupportedImageContent:
            "The selected Chrome image is not a supported image format."
        }
    }
}

nonisolated struct RemoteImageImportService: Sendable {
    func downloadImage(from sourceURL: URL) async throws -> URL {
        guard Self.isSupportedNetworkURL(sourceURL) else {
            throw RemoteImageImportError.unsupportedURLScheme
        }

        var request = URLRequest(url: sourceURL)
        request.setValue("image/avif,image/webp,image/apng,image/*,*/*;q=0.8", forHTTPHeaderField: "Accept")

        let temporaryURL: URL
        let response: URLResponse
        do {
            (temporaryURL, response) = try await URLSession.shared.download(for: request)
        } catch {
            throw RemoteImageImportError.downloadFailed
        }
        defer {
            try? FileManager.default.removeItem(at: temporaryURL)
        }

        if let httpResponse = response as? HTTPURLResponse,
           !(200..<300).contains(httpResponse.statusCode) {
            throw RemoteImageImportError.downloadFailed
        }

        guard let fileExtension = Self.fileExtension(for: sourceURL, response: response) else {
            throw RemoteImageImportError.unsupportedImageContent
        }

        do {
            let destination = try Self.temporaryDestination(for: sourceURL, fileExtension: fileExtension)
            try? FileManager.default.removeItem(at: destination)
            try FileManager.default.moveItem(at: temporaryURL, to: destination)
            return destination
        } catch {
            throw RemoteImageImportError.downloadFailed
        }
    }

    private static func isSupportedNetworkURL(_ url: URL) -> Bool {
        guard let scheme = url.scheme?.lowercased() else {
            return false
        }
        return scheme == "http" || scheme == "https"
    }

    private static func fileExtension(for sourceURL: URL, response: URLResponse) -> String? {
        if let sourceExtension = supportedImageFileExtension(sourceURL.pathExtension) {
            return sourceExtension
        }

        guard let mimeType = response.mimeType?.lowercased(),
              mimeType != "image/svg+xml",
              let type = UTType(mimeType: mimeType),
              let preferredExtension = type.preferredFilenameExtension else {
            return nil
        }

        return supportedImageFileExtension(preferredExtension)
    }

    private static func supportedImageFileExtension(_ fileExtension: String) -> String? {
        let normalized = fileExtension.lowercased()
        guard !normalized.isEmpty,
              normalized != "svg",
              normalized != "svgz",
              let type = UTType(filenameExtension: normalized),
              type.conforms(to: .image) else {
            return nil
        }
        return normalized
    }

    private static func temporaryDestination(for sourceURL: URL, fileExtension: String) throws -> URL {
        let directory = FileManager.default.temporaryDirectory
            .appendingPathComponent("MomentoChromeImports", isDirectory: true)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)

        let baseName = sanitizedBaseName(for: sourceURL)
        return directory
            .appendingPathComponent("\(baseName)-\(UUID().uuidString)")
            .appendingPathExtension(fileExtension)
    }

    private static func sanitizedBaseName(for sourceURL: URL) -> String {
        let rawName = sourceURL.deletingPathExtension().lastPathComponent
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let fallbackName = rawName.isEmpty ? "Chrome Image" : rawName
        let invalidCharacters = CharacterSet(charactersIn: "/\\:?%*|\"<>")
        let name = fallbackName
            .components(separatedBy: invalidCharacters)
            .filter { !$0.isEmpty }
            .joined(separator: "-")
        return String((name.isEmpty ? "Chrome Image" : name).prefix(80))
    }
}
