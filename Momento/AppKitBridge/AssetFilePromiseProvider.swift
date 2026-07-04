// 中文注释：本类实现拖出到 Finder 的文件承诺写入，把库内素材复制到系统请求的位置。
import AppKit
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
    private let sourceAccessValidator: @Sendable () throws -> Void
    private let onSourceAccessError: (Error) -> Void

    init?(
        asset: AssetItem,
        libraryID: AssetLibrary.ID,
        assetIDs: [AssetItem.ID],
        primaryAssetID: AssetItem.ID,
        exportBatch: AssetDragExportBatch,
        sourceAccessValidator: @escaping @Sendable () throws -> Void = {},
        onSourceAccessError: @escaping (Error) -> Void = { _ in }
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
        self.sourceAccessValidator = sourceAccessValidator
        self.onSourceAccessError = onSourceAccessError
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

    // 中文注释：不实现此方法时 AppKit 会在主队列回调 writePromiseTo，导致大图/多选批量导出的 copyItem 在主线程同步执行、阻塞 UI。
    // 返回一个进程级共享的后台队列，让实际文件复制在后台线程完成，completionHandler 也由系统在该队列回调。
    func operationQueue(for filePromiseProvider: NSFilePromiseProvider) -> OperationQueue {
        Self.promiseCopyQueue
    }

    private static let promiseCopyQueue: OperationQueue = {
        let queue = OperationQueue()
        queue.name = "com.momento.file-promise-copy"
        queue.maxConcurrentOperationCount = 2
        queue.qualityOfService = .userInitiated
        return queue
    }()

    func filePromiseProvider(
        _ filePromiseProvider: NSFilePromiseProvider,
        writePromiseTo url: URL,
        completionHandler: @escaping (Error?) -> Void
    ) {
        do {
            try sourceAccessValidator()
        } catch {
            onSourceAccessError(error)
            completionHandler(error)
            notifyExportBatch(success: false)
            return
        }

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
    private static let moveToTrashSoundPath =
        "/System/Library/Components/CoreAudio.component/Contents/SharedSupport/SystemSounds/finder/move to trash.aif"
    private static let playbackDurationNanoseconds: UInt64 = 500_000_000

    @MainActor
    private static var successSound: NSSound?
    @MainActor
    private static var playbackToken = 0

    static func playDeletionSound() {
        Task { @MainActor in
            playDeletionSoundOnMainActor()
        }
    }

    @MainActor
    private static func playDeletionSoundOnMainActor() {
        if successSound == nil {
            successSound = NSSound(contentsOfFile: moveToTrashSoundPath, byReference: true)
        }

        playbackToken += 1
        let currentPlaybackToken = playbackToken
        successSound?.stop()
        successSound?.currentTime = 0
        successSound?.play()
        schedulePlaybackStop(token: currentPlaybackToken)
    }

    @MainActor
    private static func schedulePlaybackStop(token: Int) {
        Task { @MainActor in
            try? await Task.sleep(nanoseconds: playbackDurationNanoseconds)
            guard playbackToken == token else {
                return
            }

            successSound?.stop()
            successSound?.currentTime = 0
        }
    }
}
