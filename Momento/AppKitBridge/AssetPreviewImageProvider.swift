//
//  AssetPreviewImageProvider.swift
//  Momento
//

// 中文注释：本文件封装素材缩略图/预览图的解码、降采样与内存缓存，以及可见/预取解码的并发限流。
// 从 AssetCollectionGridView.swift 拆出，让网格视图文件聚焦渲染，图片缓存服务独立可测（见 AssetPreviewImageProviderTests）。
import AppKit
import ImageIO
import UniformTypeIdentifiers

private actor AssetPreviewDecodeLimiter {
    private struct VisibleWaiter {
        let id: UUID
        let continuation: CheckedContinuation<Bool, Never>
    }

    private let visibleLimit: Int
    private let prefetchLimit: Int
    private var activeVisibleCount = 0
    private var activePrefetchCount = 0
    private var visibleWaiters: [VisibleWaiter] = []

    init(visibleLimit: Int, prefetchLimit: Int) {
        self.visibleLimit = visibleLimit
        self.prefetchLimit = prefetchLimit
    }

    func acquireVisible() async -> Bool {
        if activeVisibleCount < visibleLimit {
            activeVisibleCount += 1
            return true
        }

        let waiterID = UUID()
        return await withTaskCancellationHandler {
            await withCheckedContinuation { continuation in
                if Task.isCancelled {
                    continuation.resume(returning: false)
                } else {
                    visibleWaiters.append(VisibleWaiter(id: waiterID, continuation: continuation))
                }
            }
        } onCancel: {
            Task {
                await self.cancelVisibleWaiter(id: waiterID)
            }
        }
    }

    func releaseVisible() {
        if visibleWaiters.isEmpty {
            activeVisibleCount = max(activeVisibleCount - 1, 0)
        } else {
            visibleWaiters.removeFirst().continuation.resume(returning: true)
        }
    }

    func tryAcquirePrefetch() -> Bool {
        guard visibleWaiters.isEmpty,
              activeVisibleCount < visibleLimit,
              activePrefetchCount < prefetchLimit else {
            return false
        }

        activePrefetchCount += 1
        return true
    }

    func releasePrefetch() {
        activePrefetchCount = max(activePrefetchCount - 1, 0)
    }

    private func cancelVisibleWaiter(id: UUID) {
        guard let waiterIndex = visibleWaiters.firstIndex(where: { $0.id == id }) else {
            return
        }

        let waiter = visibleWaiters.remove(at: waiterIndex)
        waiter.continuation.resume(returning: false)
    }
}

nonisolated final class AssetPreviewImageProvider: @unchecked Sendable {
    typealias ThumbnailDecoder = @Sendable (AssetItem) -> NSImage?
    typealias FallbackImageProvider = @Sendable (AssetItem) -> NSImage
    typealias SourceAccessValidator = @Sendable () throws -> Void

    static let shared = AssetPreviewImageProvider()

    private static let previewDecodeMaxPixelSize = 512
    private static let maxConcurrentVisiblePreviewDecodes = 3
    private static let maxConcurrentPrefetchPreviewDecodes = 1

    private let cache = NSCache<NSString, NSImage>()
    private let thumbnailDecoder: ThumbnailDecoder
    private let fallbackImageProvider: FallbackImageProvider
    private let taskLock = NSLock()
    private let decodeLimiter = AssetPreviewDecodeLimiter(
        visibleLimit: AssetPreviewImageProvider.maxConcurrentVisiblePreviewDecodes,
        prefetchLimit: AssetPreviewImageProvider.maxConcurrentPrefetchPreviewDecodes
    )
    private var prefetchDecodeTasks: [String: Task<Void, Never>] = [:]
    private var prefetchingIdentities: Set<String> = []

    init(
        thumbnailDecoder: @escaping ThumbnailDecoder = AssetPreviewImageProvider.decodedThumbnailImage(for:),
        fallbackImageProvider: @escaping FallbackImageProvider = AssetPreviewImageProvider.defaultFallbackIcon(for:)
    ) {
        self.thumbnailDecoder = thumbnailDecoder
        self.fallbackImageProvider = fallbackImageProvider
        cache.countLimit = 512
        cache.totalCostLimit = 96 * 1024 * 1024
    }

    func identity(for asset: AssetItem) -> String {
        let sourcePath = asset.thumbnailURL?.path ?? asset.storageURL.path
        return [
            asset.kind.rawValue,
            asset.contentHash,
            sourcePath,
            asset.fileExtension.lowercased()
        ].joined(separator: ":")
    }

    func cachedImage(for asset: AssetItem) -> NSImage? {
        cache.object(forKey: identity(for: asset) as NSString)
    }

    func cachedImage(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator?) -> NSImage? {
        guard canReadStoredSource(sourceAccessValidator) else {
            return nil
        }

        return cachedImage(for: asset)
    }

    func image(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator? = nil) -> NSImage {
        let key = identity(for: asset) as NSString
        guard canReadStoredSource(sourceAccessValidator) else {
            return typeIcon(for: asset)
        }

        if let cachedImage = cache.object(forKey: key) {
            return cachedImage
        }

        let image = loadImage(for: asset, sourceAccessValidator: sourceAccessValidator)
        cache.setObject(image, forKey: key, cost: cacheCost(for: image))
        return image
    }

    func imageAsync(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator? = nil) async -> NSImage {
        let identity = identity(for: asset)
        let key = identity as NSString
        guard canReadStoredSource(sourceAccessValidator) else {
            return typeIcon(for: asset)
        }

        if let cachedImage = cache.object(forKey: key) {
            return cachedImage
        }

        if shouldDecodeThumbnail(for: asset) {
            let image = await decodedThumbnailImageAsync(
                for: asset,
                priority: .userInitiated,
                sourceAccessValidator: sourceAccessValidator
            )
            if let image {
                return image
            }
        }

        if Task.isCancelled {
            return placeholderImage(for: asset, sourceAccessValidator: sourceAccessValidator)
        }

        let image = fallbackIcon(for: asset, sourceAccessValidator: sourceAccessValidator)
        cache.setObject(image, forKey: key, cost: cacheCost(for: image))
        return image
    }

    func shouldLoadImageAsync(for asset: AssetItem) -> Bool {
        shouldDecodeThumbnail(for: asset)
    }

    func placeholderImage(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator? = nil) -> NSImage {
        if shouldDecodeThumbnail(for: asset),
           let type = UTType(filenameExtension: asset.fileExtension) {
            return NSWorkspace.shared.icon(for: type)
        }

        return fallbackIcon(for: asset, sourceAccessValidator: sourceAccessValidator)
    }

    func prefetchImage(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator? = nil) {
        let identity = identity(for: asset)
        let key = identity as NSString
        guard cache.object(forKey: key) == nil,
              shouldDecodeThumbnail(for: asset),
              canReadStoredSource(sourceAccessValidator),
              beginPrefetching(identity) else {
            return
        }

        let task = Task.detached(priority: .utility) { [self] in
            guard !Task.isCancelled else {
                completePrefetch(identity: identity)
                return
            }

            let acquiredSlot = await decodeLimiter.tryAcquirePrefetch()
            guard acquiredSlot else {
                completePrefetch(identity: identity)
                return
            }

            guard !Task.isCancelled else {
                await decodeLimiter.releasePrefetch()
                completePrefetch(identity: identity)
                return
            }

            guard cache.object(forKey: key) == nil else {
                await decodeLimiter.releasePrefetch()
                completePrefetch(identity: identity)
                return
            }

            let image = autoreleasepool { () -> NSImage? in
                guard canReadStoredSource(sourceAccessValidator) else {
                    return nil
                }
                return thumbnailDecoder(asset)
            }
            await decodeLimiter.releasePrefetch()
            completePrefetch(identity: identity)

            if !Task.isCancelled, let image {
                cache.setObject(image, forKey: key, cost: cacheCost(for: image))
            }
        }

        storePrefetchTask(task, identity: identity)
    }

    func cancelPrefetch(for asset: AssetItem) {
        cancelPrefetch(identity: identity(for: asset))
    }

    func invalidateImage(for asset: AssetItem) {
        let identity = identity(for: asset)
        cancelPrefetch(identity: identity)
        cache.removeObject(forKey: identity as NSString)
    }

    private func loadImage(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator?) -> NSImage {
        if asset.kind == .image || asset.kind == .gif {
            guard canReadStoredSource(sourceAccessValidator) else {
                return typeIcon(for: asset)
            }
            if let image = thumbnailDecoder(asset) {
                return image
            }
        }

        return fallbackIcon(for: asset, sourceAccessValidator: sourceAccessValidator)
    }

    private func fallbackIcon(for asset: AssetItem, sourceAccessValidator: SourceAccessValidator?) -> NSImage {
        guard canReadStoredSource(sourceAccessValidator) else {
            return typeIcon(for: asset)
        }

        return fallbackImageProvider(asset)
    }

    private static func defaultFallbackIcon(for asset: AssetItem) -> NSImage {
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            return NSWorkspace.shared.icon(forFile: asset.storageURL.path)
        }

        if let originalURL = asset.originalURL {
            return NSWorkspace.shared.icon(forFile: originalURL.path)
        }

        if let type = UTType(filenameExtension: asset.fileExtension) {
            return NSWorkspace.shared.icon(for: type)
        }

        return NSWorkspace.shared.icon(for: .data)
    }

    private func canReadStoredSource(_ sourceAccessValidator: SourceAccessValidator?) -> Bool {
        guard let sourceAccessValidator else {
            return true
        }

        do {
            try sourceAccessValidator()
            return true
        } catch {
            return false
        }
    }

    private static func typeIcon(for asset: AssetItem) -> NSImage {
        if let type = UTType(filenameExtension: asset.fileExtension) {
            return NSWorkspace.shared.icon(for: type)
        }

        return NSWorkspace.shared.icon(for: .data)
    }

    private func typeIcon(for asset: AssetItem) -> NSImage {
        Self.typeIcon(for: asset)
    }

    private func shouldDecodeThumbnail(for asset: AssetItem) -> Bool {
        (asset.kind == .image || asset.kind == .gif) && asset.thumbnailURL != nil
    }

    private func decodedThumbnailImageAsync(
        for asset: AssetItem,
        priority: TaskPriority,
        sourceAccessValidator: SourceAccessValidator?
    ) async -> NSImage? {
        let identity = identity(for: asset)
        let key = identity as NSString
        if let cachedImage = cachedImage(for: asset, sourceAccessValidator: sourceAccessValidator) {
            return cachedImage
        }

        cancelPrefetch(identity: identity)
        let task = visibleDecodeTask(
            for: asset,
            priority: priority,
            sourceAccessValidator: sourceAccessValidator
        )
        let image = await withTaskCancellationHandler {
            await task.value
        } onCancel: {
            task.cancel()
        }

        guard !Task.isCancelled else {
            return nil
        }

        if let image {
            cache.setObject(image, forKey: key, cost: cacheCost(for: image))
        }
        return image
    }

    private func visibleDecodeTask(
        for asset: AssetItem,
        priority: TaskPriority,
        sourceAccessValidator: SourceAccessValidator?
    ) -> Task<NSImage?, Never> {
        let decodeLimiter = decodeLimiter
        let thumbnailDecoder = thumbnailDecoder
        return Task.detached(priority: priority) {
            let acquiredSlot = await decodeLimiter.acquireVisible()
            guard acquiredSlot else {
                return nil
            }

            guard !Task.isCancelled else {
                await decodeLimiter.releaseVisible()
                return nil
            }

            let image = autoreleasepool { () -> NSImage? in
                guard self.canReadStoredSource(sourceAccessValidator) else {
                    return nil
                }
                return thumbnailDecoder(asset)
            }
            await decodeLimiter.releaseVisible()
            return Task.isCancelled ? nil : image
        }
    }

    private func beginPrefetching(_ identity: String) -> Bool {
        taskLock.lock()
        defer { taskLock.unlock() }

        guard prefetchDecodeTasks[identity] == nil,
              !prefetchingIdentities.contains(identity) else {
            return false
        }

        prefetchingIdentities.insert(identity)
        return true
    }

    private func storePrefetchTask(_ task: Task<Void, Never>, identity: String) {
        taskLock.lock()
        if prefetchingIdentities.contains(identity), prefetchDecodeTasks[identity] == nil {
            prefetchDecodeTasks[identity] = task
        } else {
            task.cancel()
        }
        taskLock.unlock()
    }

    private func cancelPrefetch(identity: String) {
        taskLock.lock()
        let task = prefetchDecodeTasks.removeValue(forKey: identity)
        prefetchingIdentities.remove(identity)
        taskLock.unlock()
        task?.cancel()
    }

    private func completePrefetch(identity: String) {
        taskLock.lock()
        prefetchDecodeTasks[identity] = nil
        prefetchingIdentities.remove(identity)
        taskLock.unlock()
    }

    private static func decodedThumbnailImage(for asset: AssetItem) -> NSImage? {
        guard let thumbnailURL = asset.thumbnailURL,
              let source = CGImageSourceCreateWithURL(thumbnailURL as CFURL, [
                kCGImageSourceShouldCache: false
              ] as CFDictionary) else {
            return nil
        }

        let options = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceShouldCacheImmediately: true,
            kCGImageSourceThumbnailMaxPixelSize: Self.previewDecodeMaxPixelSize
        ] as CFDictionary

        guard let image = CGImageSourceCreateThumbnailAtIndex(source, 0, options) else {
            return nil
        }

        return NSImage(
            cgImage: image,
            size: NSSize(width: image.width, height: image.height)
        )
    }

    private func cacheCost(for image: NSImage) -> Int {
        let width = max(Int(image.size.width.rounded(.up)), 1)
        let height = max(Int(image.size.height.rounded(.up)), 1)
        return width * height * 4
    }
}
