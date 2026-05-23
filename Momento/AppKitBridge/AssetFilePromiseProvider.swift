// 中文注释：本类实现拖出到 Finder 的文件承诺写入，把库内素材复制到系统请求的位置。
import AppKit
import AudioToolbox
import Foundation

nonisolated final class AssetFilePromiseProvider: NSFilePromiseProvider, NSFilePromiseProviderDelegate {
    enum UserInfoKey {
        static let sourceURL = "sourceURL"
        static let fileName = "fileName"
    }

    private let payloadData: Data
    private let sourceURL: URL
    private let fileName: String
    private let exportBatch: AssetDragExportBatch

    init?(
        asset: AssetItem,
        libraryID: AssetLibrary.ID,
        assetIDs: [AssetItem.ID],
        primaryAssetID: AssetItem.ID,
        exportBatch: AssetDragExportBatch
    ) {
        guard let payloadData = AssetDragPasteboardWriter.encodedPayload(
            libraryID: libraryID,
            assetIDs: assetIDs,
            primaryAssetID: primaryAssetID
        ) else {
            return nil
        }

        self.payloadData = payloadData
        sourceURL = asset.storageURL
        fileName = Self.promisedFileName(for: asset)
        self.exportBatch = exportBatch
        super.init()
        fileType = asset.utiIdentifier
        delegate = self
        userInfo = [
            UserInfoKey.sourceURL: sourceURL,
            UserInfoKey.fileName: fileName
        ]
    }

    override func writableTypes(for pasteboard: NSPasteboard) -> [NSPasteboard.PasteboardType] {
        [AssetDragPasteboardWriter.assetIDsPasteboardType] + super.writableTypes(for: pasteboard)
    }

    override func pasteboardPropertyList(forType type: NSPasteboard.PasteboardType) -> Any? {
        if type == AssetDragPasteboardWriter.assetIDsPasteboardType {
            return payloadData
        }

        return super.pasteboardPropertyList(forType: type)
    }

    override func writingOptions(
        forType type: NSPasteboard.PasteboardType,
        pasteboard: NSPasteboard
    ) -> NSPasteboard.WritingOptions {
        if type == AssetDragPasteboardWriter.assetIDsPasteboardType {
            return []
        }

        return super.writingOptions(forType: type, pasteboard: pasteboard)
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
            notifyExportBatch(success: true)
        } catch {
            completionHandler(error)
            notifyExportBatch(success: false)
        }
    }

    private func notifyExportBatch(success: Bool) {
        if exportBatch.promiseDidFinish(success: success) {
            AssetDeletionSoundPlayer.playDeletionSound()
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

nonisolated final class AssetDragExportBatch: @unchecked Sendable {
    private let lock = NSLock()
    private let expectedFileCount: Int
    private var completedFileCount = 0
    private var hasFailure = false
    private var didNotifyCompletion = false

    init(expectedFileCount: Int) {
        self.expectedFileCount = max(expectedFileCount, 1)
    }

    func promiseDidFinish(success: Bool) -> Bool {
        lock.lock()
        defer {
            lock.unlock()
        }

        if !success {
            hasFailure = true
        }
        completedFileCount += 1

        guard !didNotifyCompletion, completedFileCount >= expectedFileCount else {
            return false
        }

        didNotifyCompletion = true
        return !hasFailure
    }
}

nonisolated enum AssetDeletionSoundPlayer {
    private static let bundledSoundName = "MomentoActionSuccess"
    private static let bundledSoundExtension = "wav"

    @MainActor
    private static var successSoundID: SystemSoundID?

    static func playDeletionSound() {
        Task { @MainActor in
            playDeletionSoundOnMainActor()
        }
    }

    @MainActor
    private static func playDeletionSoundOnMainActor() {
        if successSoundID == nil {
            successSoundID = createBundledSoundID()
        }

        if let successSoundID {
            AudioServicesPlaySystemSound(successSoundID)
        }
    }

    private static func createBundledSoundID() -> SystemSoundID? {
        guard let soundURL = Bundle.main.url(
            forResource: bundledSoundName,
            withExtension: bundledSoundExtension
        ) else {
            return nil
        }

        var soundID: SystemSoundID = 0
        let status = AudioServicesCreateSystemSoundID(soundURL as CFURL, &soundID)
        guard status == noErr else {
            return nil
        }

        return soundID
    }
}
