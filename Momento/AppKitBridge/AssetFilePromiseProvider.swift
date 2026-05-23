import AppKit
import Foundation

final class AssetFilePromiseProvider: NSObject, NSFilePromiseProviderDelegate {
    enum UserInfoKey {
        static let sourceURL = "sourceURL"
        static let fileName = "fileName"
    }

    private(set) var provider: NSFilePromiseProvider!
    private let sourceURL: URL
    private let fileName: String

    init(asset: AssetItem) {
        sourceURL = asset.storageURL
        fileName = Self.promisedFileName(for: asset)
        super.init()
        provider = NSFilePromiseProvider(fileType: asset.utiIdentifier, delegate: self)
        provider.userInfo = [
            UserInfoKey.sourceURL: sourceURL,
            UserInfoKey.fileName: fileName
        ]
    }

    func filePromiseProvider(
        _ filePromiseProvider: NSFilePromiseProvider,
        fileNameForType fileType: String
    ) -> String {
        fileName
    }

    func filePromiseProvider(
        _ filePromiseProvider: NSFilePromiseProvider,
        writePromiseTo url: URL,
        completionHandler: @escaping (Error?) -> Void
    ) {
        do {
            try FileManager.default.copyItem(at: sourceURL, to: availableDestinationURL(for: url))
            completionHandler(nil)
        } catch {
            completionHandler(error)
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

    private static func promisedFileName(for asset: AssetItem) -> String {
        let displayName = sanitizedFileName(asset.displayName)
        let baseName = displayName.isEmpty ? sanitizedFileName(asset.originalFileName) : displayName
        let fallbackName = asset.storageURL.deletingPathExtension().lastPathComponent
        let resolvedBaseName = baseName.isEmpty ? fallbackName : baseName
        let fileExtension = asset.fileExtension.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !fileExtension.isEmpty else {
            return resolvedBaseName
        }
        return "\(resolvedBaseName).\(fileExtension)"
    }

    private static func sanitizedFileName(_ name: String) -> String {
        name
            .replacingOccurrences(of: "/", with: "-")
            .trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
