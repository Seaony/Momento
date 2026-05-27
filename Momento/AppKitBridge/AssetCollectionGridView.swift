//
//  AssetCollectionGridView.swift
//  Momento
//

// 中文注释：本文件封装高性能素材列表的 AppKit NSCollectionView 渲染、选择、预览、拖拽和快捷键桥接。
import AppKit
import ImageIO
import QuartzCore
import SwiftUI
import UniformTypeIdentifiers

private enum AssetCollectionMetrics {
    static let masonryItemWidth: CGFloat = 164
    static let masonryImageInset: CGFloat = 3
    static let masonryFallbackItemHeight: CGFloat = 214
    static let masonryMinimumItemHeight: CGFloat = 132
    static let masonryMaximumItemHeight: CGFloat = 278
    static let gridItemWidth: CGFloat = 160
    static let gridItemHeight: CGFloat = 190
    static let gridInteritemSpacing: CGFloat = 8
    static let gridLineSpacing: CGFloat = 12
    static let selectionCornerRadius = MomentoTheme.assetImageCornerRadius
    static let hoverBackgroundAlpha: CGFloat = 0.08
    static let masonryImageCornerRadius = MomentoTheme.assetImageCornerRadius
    static let gridImageCornerRadius: CGFloat = 6
    static let gridImageInset: CGFloat = 8
    static let listImageCornerRadius: CGFloat = 6
    static let dimensionBadgeCornerRadius: CGFloat = 5
    static let dimensionBadgeHeight: CGFloat = 16
    static let dimensionBadgeTopInset: CGFloat = 8
    static let dimensionBadgeHorizontalPadding: CGFloat = 3
    static let sectionHorizontalInset: CGFloat = 8
    static let sectionVerticalInset: CGFloat = 14
    static let listItemHeight: CGFloat = 96
    static let listThumbnailSize: CGFloat = 78
    static let listSeparatorHorizontalInset: CGFloat = 18
    static let listSeparatorAlpha: CGFloat = 0.055
    static let favoriteButtonWidth: CGFloat = 22
    static let favoriteButtonHeight: CGFloat = 16
    static let favoriteButtonLeadingInset: CGFloat = 7
    static let favoriteButtonTopInset: CGFloat = dimensionBadgeTopInset + (dimensionBadgeHeight - favoriteButtonHeight) / 2
    static let favoriteButtonCornerRadius: CGFloat = favoriteButtonHeight / 2
    static let favoriteSymbolPointSize: CGFloat = 14
    static let favoriteButtonBackgroundAlpha: CGFloat = 0.3
    static let favoriteButtonAppearanceAnimationDuration: CFTimeInterval = 0.14
    static let favoriteButtonBackgroundAnimationDuration: CFTimeInterval = 0.12
    static let favoriteButtonEntranceScale: CGFloat = 0.9
    static let selectionBackgroundAnimationDuration: CFTimeInterval = 0.12
    static let titleTextColor = NSColor.labelColor.withAlphaComponent(0.5)
    static let subtitleTextColor = NSColor.labelColor.withAlphaComponent(0.3)
    static let listDateTextColor = NSColor.labelColor.withAlphaComponent(0.72)
    static let selectedTitleTextColor = NSColor.white.withAlphaComponent(0.95)
    static let selectedSubtitleTextColor = NSColor.white.withAlphaComponent(0.72)
    static let imageEntranceAnimationDuration: CFTimeInterval = 0.18
    static let imageEntranceScale: CGFloat = 0.985
    static let liveResizeWidthInvalidationThreshold: CGFloat = 8
    static let contextMenuWidth: CGFloat = 184
    static let contextMenuPadding: CGFloat = 6
    static let contextMenuRowHeight: CGFloat = 28
    static let contextMenuRowSpacing: CGFloat = 2
    static let contextMenuRowCornerRadius: CGFloat = 8
    static let contextMenuPanelCornerRadius: CGFloat = 12
    static let contextMenuPanelBleed: CGFloat = 6
    static let contextMenuSeparatorHeight: CGFloat = 1
    static let contextMenuSeparatorHorizontalInset: CGFloat = 10
    static let contextMenuSeparatorVerticalPadding: CGFloat = 3
    static let contextMenuSeparatorAlpha: CGFloat = 0.1
    static let zeroEdgeInsets = NSEdgeInsets(top: 0, left: 0, bottom: 0, right: 0)

    static func columnLayout(
        availableWidth: CGFloat,
        minimumItemWidth: CGFloat,
        interitemSpacing: CGFloat
    ) -> AssetColumnLayout {
        let columnCount = max(Int((availableWidth + interitemSpacing) / (minimumItemWidth + interitemSpacing)), 1)
        let spacingWidth = CGFloat(max(columnCount - 1, 0)) * interitemSpacing
        let itemWidth = max((availableWidth - spacingWidth) / CGFloat(columnCount), minimumItemWidth)

        return AssetColumnLayout(columnCount: columnCount, itemWidth: itemWidth)
    }

    static func gridItemHeight(for itemWidth: CGFloat) -> CGFloat {
        max(gridItemHeight, itemWidth * gridItemHeight / gridItemWidth)
    }
}

private struct AssetColumnLayout {
    let columnCount: Int
    let itemWidth: CGFloat
}

enum AssetContextMenuAction: CaseIterable {
    case previewOriginal
    case export
    case refreshThumbnail
    case reanalyzeColors
    case revealInFinder
    case moveToTrash
    case restore
    case deletePermanently

    static func actions(for asset: AssetItem) -> [AssetContextMenuAction] {
        if asset.isTrashed {
            return [.previewOriginal, .export, .revealInFinder, .restore, .deletePermanently]
        }

        return [.previewOriginal, .export, .refreshThumbnail, .reanalyzeColors, .revealInFinder, .moveToTrash]
    }

    var titleKey: String {
        switch self {
        case .previewOriginal:
            "Preview Original"
        case .export:
            "Export"
        case .refreshThumbnail:
            "Refresh Thumbnail"
        case .reanalyzeColors:
            "Reanalyze Colors"
        case .revealInFinder:
            "Reveal in Finder"
        case .moveToTrash:
            "Move to Trash"
        case .restore:
            "Restore"
        case .deletePermanently:
            "Delete Permanently"
        }
    }

    var systemImageName: String {
        switch self {
        case .previewOriginal:
            "eye"
        case .export:
            "square.and.arrow.up"
        case .refreshThumbnail:
            "arrow.clockwise"
        case .reanalyzeColors:
            "paintpalette"
        case .revealInFinder:
            "finder"
        case .moveToTrash:
            "trash"
        case .restore:
            "arrow.uturn.backward"
        case .deletePermanently:
            "trash.slash"
        }
    }

    var isDestructive: Bool {
        self == .moveToTrash || self == .deletePermanently
    }

    var showsSeparatorAfter: Bool {
        self == .export || self == .revealInFinder
    }
}

struct AssetCollectionGridView: NSViewRepresentable {
    var assets: [AssetItem]
    var selectedAssetIDs: Set<AssetItem.ID>
    var viewMode: AssetViewMode
    var localization: AppLocalization
    var onSelectionChange: (Set<AssetItem.ID>) -> Void
    var onDoubleClick: (AssetItem) -> Void
    var onSpacePreviewStart: (AssetItem, NSRect?) -> Void
    var onSpacePreviewEnd: () -> Void
    var onFavoriteToggle: (AssetItem) -> Void
    var onCommandDelete: (Set<AssetItem.ID>) -> Bool
    var onContextMenuAction: (AssetItem, [AssetItem], AssetContextMenuAction) -> Void

    static func invalidatePreviewCache(for asset: AssetItem) {
        AssetPreviewImageProvider.shared.invalidateImage(for: asset)
    }

    init(
        assets: [AssetItem],
        selectedAssetIDs: Set<AssetItem.ID> = [],
        viewMode: AssetViewMode = .grid,
        localization: AppLocalization = AppLocalization(language: .system),
        onSelectionChange: @escaping (Set<AssetItem.ID>) -> Void = { _ in },
        onDoubleClick: @escaping (AssetItem) -> Void = { _ in },
        onSpacePreviewStart: @escaping (AssetItem, NSRect?) -> Void = { _, _ in },
        onSpacePreviewEnd: @escaping () -> Void = {},
        onFavoriteToggle: @escaping (AssetItem) -> Void = { _ in },
        onCommandDelete: @escaping (Set<AssetItem.ID>) -> Bool = { _ in false },
        onContextMenuAction: @escaping (AssetItem, [AssetItem], AssetContextMenuAction) -> Void = { _, _, _ in }
    ) {
        self.assets = assets
        self.selectedAssetIDs = selectedAssetIDs
        self.viewMode = viewMode
        self.localization = localization
        self.onSelectionChange = onSelectionChange
        self.onDoubleClick = onDoubleClick
        self.onSpacePreviewStart = onSpacePreviewStart
        self.onSpacePreviewEnd = onSpacePreviewEnd
        self.onFavoriteToggle = onFavoriteToggle
        self.onCommandDelete = onCommandDelete
        self.onContextMenuAction = onContextMenuAction
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeNSView(context: Context) -> NSScrollView {
        let collectionView = AssetPreviewCollectionView()
        collectionView.collectionViewLayout = makeLayout(for: viewMode, assets: assets)
        collectionView.backgroundColors = [.clear]
        collectionView.isSelectable = true
        collectionView.allowsMultipleSelection = true
        collectionView.allowsEmptySelection = false
        collectionView.dataSource = context.coordinator
        collectionView.delegate = context.coordinator
        collectionView.prefetchDataSource = context.coordinator
        collectionView.setDraggingSourceOperationMask(.copy, forLocal: false)
        collectionView.setDraggingSourceOperationMask(.copy, forLocal: true)
        collectionView.register(
            AssetCollectionViewItem.self,
            forItemWithIdentifier: AssetCollectionViewItem.reuseIdentifier
        )
        collectionView.onSpacePreviewStart = { [weak coordinator = context.coordinator] in
            coordinator?.startSpacePreview()
        }
        collectionView.onSpacePreviewEnd = { [weak coordinator = context.coordinator] in
            coordinator?.endSpacePreview()
        }
        collectionView.onFavoriteShortcut = { [weak coordinator = context.coordinator] in
            coordinator?.toggleHoveredFavorite() ?? false
        }
        collectionView.onCommandDeleteShortcut = { [weak coordinator = context.coordinator] in
            coordinator?.commandDeleteSelectedAssets() ?? false
        }

        let doubleClickRecognizer = NSClickGestureRecognizer(
            target: context.coordinator,
            action: #selector(Coordinator.handleDoubleClick(_:))
        )
        doubleClickRecognizer.numberOfClicksRequired = 2
        doubleClickRecognizer.delaysPrimaryMouseButtonEvents = false
        collectionView.addGestureRecognizer(doubleClickRecognizer)

        let scrollView = AssetCollectionScrollView()
        configureScrollView(scrollView)
        scrollView.documentView = collectionView

        context.coordinator.collectionView = collectionView
        return scrollView
    }

    func updateNSView(_ scrollView: NSScrollView, context: Context) {
        context.coordinator.parent = self
        configureScrollView(scrollView)

        guard let collectionView = scrollView.documentView as? NSCollectionView else {
            return
        }

        if context.coordinator.currentViewMode != viewMode {
            collectionView.collectionViewLayout = makeLayout(for: viewMode, assets: assets)
            context.coordinator.currentViewMode = viewMode
            context.coordinator.currentAssets = assets
            context.coordinator.rebuildAssetIndex(for: assets)
            context.coordinator.currentLocalization = localization
            collectionView.reloadData()
            context.coordinator.syncSelection()
            context.coordinator.syncHoveredPreviewAsset()
            return
        }

        if context.coordinator.currentAssets != assets {
            applyAssetChanges(to: collectionView, coordinator: context.coordinator)
        }

        if context.coordinator.currentLocalization != localization {
            context.coordinator.currentLocalization = localization
            collectionView.reloadData()
        }

        context.coordinator.syncSelection()
        context.coordinator.syncHoveredPreviewAsset()
    }

    private func makeLayout(for viewMode: AssetViewMode, assets: [AssetItem]) -> NSCollectionViewLayout {
        if viewMode == .masonry {
            return AssetMasonryCollectionViewLayout(assets: assets)
        }

        if viewMode == .grid {
            return AssetGridCollectionViewLayout()
        }

        let layout = NSCollectionViewFlowLayout()
        layout.sectionInset = NSEdgeInsets(
            top: AssetCollectionMetrics.sectionVerticalInset,
            left: AssetCollectionMetrics.sectionHorizontalInset,
            bottom: AssetCollectionMetrics.sectionVerticalInset,
            right: AssetCollectionMetrics.sectionHorizontalInset
        )

        layout.itemSize = NSSize(width: 320, height: AssetCollectionMetrics.listItemHeight)
        layout.minimumInteritemSpacing = 0
        layout.minimumLineSpacing = 1

        return layout
    }

    private func configureScrollView(_ scrollView: NSScrollView) {
        if scrollView.drawsBackground {
            scrollView.drawsBackground = false
        }

        if scrollView.hasVerticalScroller {
            scrollView.hasVerticalScroller = false
        }

        if scrollView.hasHorizontalScroller {
            scrollView.hasHorizontalScroller = false
        }

        if scrollView.verticalScroller != nil {
            scrollView.verticalScroller = nil
        }

        if scrollView.horizontalScroller != nil {
            scrollView.horizontalScroller = nil
        }

        if !scrollView.autohidesScrollers {
            scrollView.autohidesScrollers = true
        }

        if scrollView.scrollerStyle != .overlay {
            scrollView.scrollerStyle = .overlay
        }

        if scrollView.automaticallyAdjustsContentInsets {
            scrollView.automaticallyAdjustsContentInsets = false
        }

        if !scrollView.contentInsets.areZero {
            scrollView.contentInsets = AssetCollectionMetrics.zeroEdgeInsets
        }

        if !scrollView.scrollerInsets.areZero {
            scrollView.scrollerInsets = AssetCollectionMetrics.zeroEdgeInsets
        }
    }

    private func applyAssetChanges(to collectionView: NSCollectionView, coordinator: Coordinator) {
        // 中文注释：收藏、标题、标签、文件夹等轻量字段变化可以原地刷新 cell，
        // 避免 reloadData 造成整个瀑布流闪烁和滚动位置抖动。
        if let itemUpdateIndexPaths = itemUpdateIndexPaths(from: coordinator.currentAssets, to: assets) {
            coordinator.currentAssets = assets
            coordinator.rebuildAssetIndex(for: assets)

            for indexPath in itemUpdateIndexPaths {
                guard let item = collectionView.item(at: indexPath) as? AssetCollectionViewItem,
                      assets.indices.contains(indexPath.item) else {
                    continue
                }

                item.updateVisibleState(
                    with: assets[indexPath.item],
                    viewMode: viewMode,
                    localization: localization
                )
            }
            return
        }

        coordinator.currentAssets = assets
        coordinator.rebuildAssetIndex(for: assets)
        prepareLayout(for: collectionView)
        collectionView.reloadData()
    }

    private func itemUpdateIndexPaths(from oldAssets: [AssetItem], to newAssets: [AssetItem]) -> Set<IndexPath>? {
        guard oldAssets.count == newAssets.count else {
            return nil
        }

        var changedIndexPaths: Set<IndexPath> = []

        for index in oldAssets.indices {
            let oldAsset = oldAssets[index]
            let newAsset = newAssets[index]
            guard oldAsset.id == newAsset.id else {
                return nil
            }

            if oldAsset == newAsset {
                continue
            }

            guard canUpdateItemInPlace(from: oldAsset, to: newAsset) else {
                return nil
            }

            changedIndexPaths.insert(IndexPath(item: index, section: 0))
        }

        return changedIndexPaths
    }

    private func canUpdateItemInPlace(from oldAsset: AssetItem, to newAsset: AssetItem) -> Bool {
        // 中文注释：这里显式列出允许原地刷新的字段。只要布局、文件路径或缩略图相关字段变化，
        // 比较结果就会失败并回退到完整 reload，保证布局缓存不会拿到过期数据。
        var comparableOldAsset = oldAsset
        comparableOldAsset.displayName = newAsset.displayName
        comparableOldAsset.byteSize = newAsset.byteSize
        comparableOldAsset.exifMetadata = newAsset.exifMetadata
        comparableOldAsset.tags = newAsset.tags
        comparableOldAsset.folderIDs = newAsset.folderIDs
        comparableOldAsset.paletteColors = newAsset.paletteColors
        comparableOldAsset.note = newAsset.note
        comparableOldAsset.sourcePageURL = newAsset.sourcePageURL
        comparableOldAsset.isFavorite = newAsset.isFavorite
        comparableOldAsset.importedAt = newAsset.importedAt
        comparableOldAsset.updatedAt = newAsset.updatedAt
        return comparableOldAsset == newAsset
    }

    private func prepareLayout(for collectionView: NSCollectionView) {
        if let masonryLayout = collectionView.collectionViewLayout as? AssetMasonryCollectionViewLayout {
            masonryLayout.assets = assets
        } else if collectionView.collectionViewLayout is AssetGridCollectionViewLayout {
            collectionView.collectionViewLayout?.invalidateLayout()
        }
    }
}

extension AssetCollectionGridView {
    final class Coordinator: NSObject, NSCollectionViewDataSource, NSCollectionViewDelegateFlowLayout, NSCollectionViewPrefetching {
        var parent: AssetCollectionGridView
        weak var collectionView: NSCollectionView?
        var currentViewMode: AssetViewMode
        var currentAssets: [AssetItem]
        var currentLocalization: AppLocalization
        private var isSyncingSelection = false
        private var hoveredPreviewAssetID: AssetItem.ID?
        private var assetIndexByID: [AssetItem.ID: Int]
        private var activeDragPrimaryAssetID: AssetItem.ID?
        private var activeDragExportBatch: AssetDragExportBatch?
        private var activeDragSourcePlaceholders: [NSView] = []

        init(_ parent: AssetCollectionGridView) {
            self.parent = parent
            self.currentViewMode = parent.viewMode
            self.currentAssets = parent.assets
            self.currentLocalization = parent.localization
            self.assetIndexByID = Self.assetIndexByID(for: parent.assets)
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            numberOfItemsInSection section: Int
        ) -> Int {
            parent.assets.count
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            itemForRepresentedObjectAt indexPath: IndexPath
        ) -> NSCollectionViewItem {
            let item = collectionView.makeItem(
                withIdentifier: AssetCollectionViewItem.reuseIdentifier,
                for: indexPath
            )

            guard let assetItem = item as? AssetCollectionViewItem else {
                return item
            }

            let asset = parent.assets[indexPath.item]
            assetItem.onHoverPreviewChange = { [weak self, assetID = asset.id] isHovered in
                self?.updateHoveredPreviewAsset(assetID: assetID, isHovered: isHovered)
            }
            assetItem.onContextMenuOpen = { [weak self, assetID = asset.id] in
                self?.selectAssetForContextMenu(assetID: assetID)
            }
            assetItem.onContextMenuAction = { [weak self, assetID = asset.id] action in
                self?.performContextMenuAction(assetID: assetID, action: action)
            }
            assetItem.onFavoriteToggle = { [weak self, assetID = asset.id] in
                self?.toggleFavorite(assetID: assetID)
            }
            assetItem.configure(with: asset, viewMode: parent.viewMode, localization: parent.localization)
            return assetItem
        }

        func collectionView(_ collectionView: NSCollectionView, prefetchItemsAt indexPaths: [IndexPath]) {
            for indexPath in indexPaths where parent.assets.indices.contains(indexPath.item) {
                AssetPreviewImageProvider.shared.prefetchImage(for: parent.assets[indexPath.item])
            }
        }

        func collectionView(_ collectionView: NSCollectionView, cancelPrefetchingForItemsAt indexPaths: [IndexPath]) {
            for indexPath in indexPaths where parent.assets.indices.contains(indexPath.item) {
                AssetPreviewImageProvider.shared.cancelPrefetch(for: parent.assets[indexPath.item])
            }
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            layout collectionViewLayout: NSCollectionViewLayout,
            sizeForItemAt indexPath: IndexPath
        ) -> NSSize {
            switch parent.viewMode {
            case .grid:
                return NSSize(
                    width: AssetCollectionMetrics.gridItemWidth,
                    height: AssetCollectionMetrics.gridItemHeight
                )
            case .masonry:
                return .zero
            case .list:
                let width = max(collectionView.enclosingScrollView?.contentSize.width ?? 320, 240)
                return NSSize(width: width, height: AssetCollectionMetrics.listItemHeight)
            }
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            didSelectItemsAt indexPaths: Set<IndexPath>
        ) {
            collectionView.window?.makeFirstResponder(collectionView)
            publishSelection(from: collectionView)
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            didDeselectItemsAt indexPaths: Set<IndexPath>
        ) {
            publishSelection(from: collectionView)
        }

        func collectionView(_ collectionView: NSCollectionView, canDragItemsAt indexPaths: Set<IndexPath>, with event: NSEvent) -> Bool {
            let location = collectionView.convert(event.locationInWindow, from: nil)
            guard let indexPath = collectionView.indexPathForItem(at: location),
                  parent.assets.indices.contains(indexPath.item) else {
                return false
            }

            let asset = parent.assets[indexPath.item]
            guard !asset.isTrashed else {
                return false
            }

            activeDragPrimaryAssetID = asset.id
            if !collectionView.selectionIndexPaths.contains(indexPath) {
                collectionView.selectionIndexPaths = [indexPath]
                publishSelection(from: collectionView)
            }

            return true
        }

        func collectionView(_ collectionView: NSCollectionView, pasteboardWriterForItemAt indexPath: IndexPath) -> NSPasteboardWriting? {
            guard parent.assets.indices.contains(indexPath.item) else {
                return nil
            }

            let asset = parent.assets[indexPath.item]
            guard !asset.isTrashed else {
                return nil
            }

            let primaryAssetID = activeDragPrimaryAssetID ?? asset.id
            let selectedAssetIDs = orderedDragAssetIDs(primaryAssetID: primaryAssetID, in: collectionView)
            let exportBatch: AssetDragExportBatch
            if let activeDragExportBatch {
                exportBatch = activeDragExportBatch
            } else {
                let newExportBatch = AssetDragExportBatch(expectedFileCount: selectedAssetIDs.count)
                activeDragExportBatch = newExportBatch
                exportBatch = newExportBatch
            }

            return AssetFilePromiseProvider(
                asset: asset,
                libraryID: asset.libraryID,
                assetIDs: selectedAssetIDs,
                primaryAssetID: primaryAssetID,
                exportBatch: exportBatch
            )
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            draggingSession session: NSDraggingSession,
            willBeginAt screenPoint: NSPoint,
            forItemsAt indexPaths: Set<IndexPath>
        ) {
            showDragSourcePlaceholders(for: indexPaths, in: collectionView)
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            draggingSession session: NSDraggingSession,
            endedAt screenPoint: NSPoint,
            dragOperation operation: NSDragOperation
        ) {
            activeDragPrimaryAssetID = nil
            activeDragExportBatch = nil
            hideDragSourcePlaceholders()
        }

        func syncSelection() {
            guard let collectionView else {
                return
            }

            isSyncingSelection = true
            collectionView.selectionIndexPaths = Set(
                parent.selectedAssetIDs.compactMap { assetID in
                    assetIndexByID[assetID].map { IndexPath(item: $0, section: 0) }
                }
            )
            isSyncingSelection = false
        }

        func syncHoveredPreviewAsset() {
            guard let hoveredPreviewAssetID,
                  assetIndexByID[hoveredPreviewAssetID] != nil else {
                self.hoveredPreviewAssetID = nil
                return
            }
        }

        func rebuildAssetIndex(for assets: [AssetItem]) {
            assetIndexByID = Self.assetIndexByID(for: assets)
        }

        @objc func handleDoubleClick(_ sender: NSClickGestureRecognizer) {
            guard let collectionView = sender.view as? NSCollectionView else {
                return
            }

            guard
                let indexPath = collectionView.indexPathForItem(at: sender.location(in: collectionView)),
                parent.assets.indices.contains(indexPath.item)
            else {
                return
            }

            parent.onDoubleClick(parent.assets[indexPath.item])
        }

        func startSpacePreview() {
            guard let collectionView,
                  let indexPath = previewIndexPath(in: collectionView),
                  parent.assets.indices.contains(indexPath.item) else {
                return
            }

            let sourceFrame = sourceFrameForPreview(at: indexPath, in: collectionView)
            parent.onSpacePreviewStart(parent.assets[indexPath.item], sourceFrame)
        }

        func endSpacePreview() {
            parent.onSpacePreviewEnd()
        }

        private func updateHoveredPreviewAsset(assetID: AssetItem.ID, isHovered: Bool) {
            if isHovered {
                hoveredPreviewAssetID = assetID
                collectionView?.window?.makeFirstResponder(collectionView)
            } else if hoveredPreviewAssetID == assetID {
                hoveredPreviewAssetID = nil
            }
        }

        private func selectAssetForContextMenu(assetID: AssetItem.ID) {
            guard let collectionView,
                  let index = assetIndexByID[assetID] else {
                return
            }

            let indexPath = IndexPath(item: index, section: 0)
            if !collectionView.selectionIndexPaths.contains(indexPath) {
                collectionView.selectionIndexPaths = [indexPath]
                publishSelection(from: collectionView)
            }
        }

        private func performContextMenuAction(assetID: AssetItem.ID, action: AssetContextMenuAction) {
            guard let index = assetIndexByID[assetID],
                  parent.assets.indices.contains(index) else {
                return
            }

            let primaryAsset = parent.assets[index]
            parent.onContextMenuAction(primaryAsset, contextAssets(primaryAssetID: assetID), action)
        }

        private func contextAssets(primaryAssetID: AssetItem.ID) -> [AssetItem] {
            guard let collectionView,
                  let primaryIndex = assetIndexByID[primaryAssetID],
                  parent.assets.indices.contains(primaryIndex) else {
                return []
            }

            let primaryIndexPath = IndexPath(item: primaryIndex, section: 0)
            guard collectionView.selectionIndexPaths.contains(primaryIndexPath) else {
                return [parent.assets[primaryIndex]]
            }

            let selectedAssets = collectionView.selectionIndexPaths
                .sorted { $0.item < $1.item }
                .compactMap { indexPath -> AssetItem? in
                    guard parent.assets.indices.contains(indexPath.item) else {
                        return nil
                    }
                    return parent.assets[indexPath.item]
                }

            return selectedAssets.isEmpty ? [parent.assets[primaryIndex]] : selectedAssets
        }

        private func toggleFavorite(assetID: AssetItem.ID) {
            guard let index = assetIndexByID[assetID],
                  parent.assets.indices.contains(index) else {
                return
            }

            parent.onFavoriteToggle(parent.assets[index])
        }

        func toggleHoveredFavorite() -> Bool {
            guard let hoveredPreviewAssetID,
                  let index = assetIndexByID[hoveredPreviewAssetID],
                  parent.assets.indices.contains(index) else {
                return false
            }

            parent.onFavoriteToggle(parent.assets[index])
            return true
        }

        func commandDeleteSelectedAssets() -> Bool {
            guard let collectionView else {
                return false
            }

            let selectedIDs = Set<AssetItem.ID>(collectionView.selectionIndexPaths.compactMap { indexPath in
                guard parent.assets.indices.contains(indexPath.item) else {
                    return nil
                }

                return parent.assets[indexPath.item].id
            })

            guard !selectedIDs.isEmpty else {
                return false
            }

            return parent.onCommandDelete(selectedIDs)
        }

        private func previewIndexPath(in collectionView: NSCollectionView) -> IndexPath? {
            if let hoveredPreviewAssetID,
               let index = assetIndexByID[hoveredPreviewAssetID] {
                return IndexPath(item: index, section: 0)
            }

            return collectionView.selectionIndexPaths.sorted(by: { $0.item < $1.item }).first
        }

        private func publishSelection(from collectionView: NSCollectionView) {
            guard !isSyncingSelection else {
                return
            }

            let selectedIDs = Set(collectionView.selectionIndexPaths.compactMap { indexPath in
                parent.assets.indices.contains(indexPath.item) ? parent.assets[indexPath.item].id : nil
            })

            parent.onSelectionChange(selectedIDs)
        }

        private func orderedDragAssetIDs(
            primaryAssetID: AssetItem.ID,
            in collectionView: NSCollectionView
        ) -> [AssetItem.ID] {
            let selectedIndexPaths = collectionView.selectionIndexPaths
            let orderedIDs = parent.assets.enumerated().compactMap { index, asset in
                selectedIndexPaths.contains(IndexPath(item: index, section: 0)) && !asset.isTrashed ? asset.id : nil
            }

            return orderedIDs.isEmpty ? [primaryAssetID] : orderedIDs
        }

        private func showDragSourcePlaceholders(
            for indexPaths: Set<IndexPath>,
            in collectionView: NSCollectionView
        ) {
            hideDragSourcePlaceholders()

            activeDragSourcePlaceholders = indexPaths.compactMap { indexPath in
                guard let item = collectionView.item(at: indexPath) as? AssetCollectionViewItem,
                      let snapshot = item.dragSourcePlaceholderImage() else {
                    return nil
                }

                let placeholder = AssetDragSourcePlaceholderView(frame: item.view.frame)
                placeholder.image = snapshot
                placeholder.imageScaling = .scaleAxesIndependently
                collectionView.addSubview(placeholder, positioned: .above, relativeTo: nil)
                return placeholder
            }
        }

        private func hideDragSourcePlaceholders() {
            activeDragSourcePlaceholders.forEach { $0.removeFromSuperview() }
            activeDragSourcePlaceholders = []
        }

        private func sourceFrameForPreview(at indexPath: IndexPath, in collectionView: NSCollectionView) -> NSRect? {
            guard let item = collectionView.item(at: indexPath) as? AssetCollectionViewItem else {
                return nil
            }

            return item.previewSourceFrameInScreen()
        }

        private static func assetIndexByID(for assets: [AssetItem]) -> [AssetItem.ID: Int] {
            Dictionary(uniqueKeysWithValues: assets.enumerated().map { ($0.element.id, $0.offset) })
        }
    }
}

private final class AssetDragSourcePlaceholderView: NSImageView {
    override func hitTest(_ point: NSPoint) -> NSView? {
        nil
    }
}

private final class AssetPreviewCollectionView: NSCollectionView {
    private static let deleteKeyCode: UInt16 = 51

    var onSpacePreviewStart: (() -> Void)?
    var onSpacePreviewEnd: (() -> Void)?
    var onFavoriteShortcut: (() -> Bool)?
    var onCommandDeleteShortcut: (() -> Bool)?

    override var acceptsFirstResponder: Bool {
        true
    }

    override func viewDidEndLiveResize() {
        super.viewDidEndLiveResize()
        collectionViewLayout?.invalidateLayout()
    }

    override func keyDown(with event: NSEvent) {
        if event.charactersIgnoringModifiers == " " {
            if !event.isARepeat {
                onSpacePreviewStart?()
            }
            return
        }

        if event.charactersIgnoringModifiers?.lowercased() == "f",
           event.modifierFlags.intersection([.command, .control, .option]).isEmpty,
           !event.isARepeat,
           onFavoriteShortcut?() == true {
            return
        }

        if isCommandDelete(event),
           !event.isARepeat,
           onCommandDeleteShortcut?() == true {
            return
        }

        super.keyDown(with: event)
    }

    private func isCommandDelete(_ event: NSEvent) -> Bool {
        let modifiers = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
        return modifiers == .command
            && (event.keyCode == Self.deleteKeyCode || event.charactersIgnoringModifiers == "\u{7F}")
    }

    override func keyUp(with event: NSEvent) {
        if event.charactersIgnoringModifiers == " " {
            onSpacePreviewEnd?()
            return
        }

        super.keyUp(with: event)
    }
}

private final class AssetCollectionScrollView: NSScrollView {
    override func viewDidEndLiveResize() {
        super.viewDidEndLiveResize()
        (documentView as? NSCollectionView)?.collectionViewLayout?.invalidateLayout()
    }

    override func tile() {
        hideScrollIndicators()
        super.tile()
        hideScrollIndicators()
    }

    override func reflectScrolledClipView(_ clipView: NSClipView) {
        super.reflectScrolledClipView(clipView)
        hideScrollIndicators()
    }

    private func hideScrollIndicators() {
        guard needsHiddenScrollIndicatorReset else {
            return
        }

        hasVerticalScroller = false
        hasHorizontalScroller = false
        verticalScroller = nil
        horizontalScroller = nil
        contentInsets = AssetCollectionMetrics.zeroEdgeInsets
        scrollerInsets = AssetCollectionMetrics.zeroEdgeInsets
    }

    private var needsHiddenScrollIndicatorReset: Bool {
        hasVerticalScroller
            || hasHorizontalScroller
            || verticalScroller != nil
            || horizontalScroller != nil
            || !contentInsets.areZero
            || !scrollerInsets.areZero
    }
}

private final class AssetGridCollectionViewLayout: NSCollectionViewLayout {
    private var contentSize: NSSize = .zero
    private var preparedBoundsSize: NSSize = .zero
    private var preparedColumnCount = 1
    private var preparedItemCount = 0

    private let minimumItemSize = NSSize(
        width: AssetCollectionMetrics.gridItemWidth,
        height: AssetCollectionMetrics.gridItemHeight
    )
    private var preparedItemSize = NSSize(
        width: AssetCollectionMetrics.gridItemWidth,
        height: AssetCollectionMetrics.gridItemHeight
    )
    private let interitemSpacing = AssetCollectionMetrics.gridInteritemSpacing
    private let lineSpacing = AssetCollectionMetrics.gridLineSpacing
    private let sectionInset = NSEdgeInsets(
        top: AssetCollectionMetrics.sectionVerticalInset,
        left: AssetCollectionMetrics.sectionHorizontalInset,
        bottom: AssetCollectionMetrics.sectionVerticalInset,
        right: AssetCollectionMetrics.sectionHorizontalInset
    )

    override func prepare() {
        super.prepare()

        guard let collectionView else {
            contentSize = .zero
            preparedItemCount = 0
            return
        }

        let itemCount = collectionView.numberOfItems(inSection: 0)
        preparedItemCount = itemCount
        preparedBoundsSize = collectionView.bounds.size
        let contentWidth = contentWidth(forBoundsWidth: collectionView.bounds.width)
        let columnLayout = columnLayout(forContentWidth: contentWidth)
        let columnCount = columnLayout.columnCount
        let itemSize = NSSize(
            width: columnLayout.itemWidth,
            height: AssetCollectionMetrics.gridItemHeight(for: columnLayout.itemWidth)
        )
        preparedItemSize = itemSize
        preparedColumnCount = columnCount

        let rowCount = itemCount == 0 ? 0 : Int(ceil(CGFloat(itemCount) / CGFloat(columnCount)))
        let itemsHeight = CGFloat(rowCount) * itemSize.height + CGFloat(max(rowCount - 1, 0)) * lineSpacing
        let contentHeight = max(
            sectionInset.top + itemsHeight + sectionInset.bottom,
            collectionView.enclosingScrollView?.contentSize.height ?? collectionView.bounds.height
        )
        contentSize = NSSize(width: contentWidth, height: contentHeight)
    }

    override var collectionViewContentSize: NSSize {
        contentSize
    }

    override func layoutAttributesForElements(in rect: NSRect) -> [NSCollectionViewLayoutAttributes] {
        guard preparedItemCount > 0 else {
            return []
        }

        let rowStride = preparedItemSize.height + lineSpacing
        let firstRow = max(Int(floor((rect.minY - sectionInset.top) / rowStride)), 0)
        let maximumRow = max((preparedItemCount - 1) / preparedColumnCount, 0)
        let lastRow = min(max(Int(floor((rect.maxY - sectionInset.top) / rowStride)), 0), maximumRow)
        let firstItem = min(firstRow * preparedColumnCount, preparedItemCount)
        let lastItem = min((lastRow + 1) * preparedColumnCount, preparedItemCount)

        guard firstItem < lastItem else {
            return []
        }

        return (firstItem..<lastItem).compactMap { item in
            guard let attributes = layoutAttributes(forItem: item),
                  attributes.frame.intersects(rect) else {
                return nil
            }

            return attributes
        }
    }

    override func layoutAttributesForItem(at indexPath: IndexPath) -> NSCollectionViewLayoutAttributes? {
        layoutAttributes(forItem: indexPath.item)
    }

    override func shouldInvalidateLayout(forBoundsChange newBounds: NSRect) -> Bool {
        guard let collectionView, collectionView.inLiveResize else {
            return newBounds.size != preparedBoundsSize
        }

        let newContentWidth = contentWidth(forBoundsWidth: newBounds.width)
        let newColumnLayout = columnLayout(forContentWidth: newContentWidth)
        if newColumnLayout.columnCount != preparedColumnCount {
            return true
        }

        return abs(newColumnLayout.itemWidth - preparedItemSize.width) >= AssetCollectionMetrics.liveResizeWidthInvalidationThreshold
    }

    private func contentWidth(forBoundsWidth boundsWidth: CGFloat) -> CGFloat {
        max(boundsWidth, minimumItemSize.width + sectionInset.left + sectionInset.right)
    }

    private func columnLayout(forContentWidth contentWidth: CGFloat) -> AssetColumnLayout {
        let availableWidth = max(contentWidth - sectionInset.left - sectionInset.right, minimumItemSize.width)
        return AssetCollectionMetrics.columnLayout(
            availableWidth: availableWidth,
            minimumItemWidth: minimumItemSize.width,
            interitemSpacing: interitemSpacing
        )
    }

    private func layoutAttributes(forItem item: Int) -> NSCollectionViewLayoutAttributes? {
        guard item >= 0, item < preparedItemCount else {
            return nil
        }

        let row = item / preparedColumnCount
        let column = item % preparedColumnCount
        let frame = NSRect(
            x: sectionInset.left + CGFloat(column) * (preparedItemSize.width + interitemSpacing),
            y: sectionInset.top + CGFloat(row) * (preparedItemSize.height + lineSpacing),
            width: preparedItemSize.width,
            height: preparedItemSize.height
        )
        let indexPath = IndexPath(item: item, section: 0)
        let attributes = NSCollectionViewLayoutAttributes(forItemWith: indexPath)
        attributes.frame = frame
        return attributes
    }
}

private final class AssetMasonryCollectionViewLayout: NSCollectionViewLayout {
    var assets: [AssetItem] {
        didSet {
            invalidateLayout()
        }
    }

    private var cachedFrames: [NSRect] = []
    private var contentSize: NSSize = .zero
    private var preparedBoundsSize: NSSize = .zero
    private var preparedColumnCount = 1
    private var preparedItemWidth = AssetCollectionMetrics.masonryItemWidth

    private let minimumItemWidth = AssetCollectionMetrics.masonryItemWidth
    private let interitemSpacing: CGFloat = 4
    private let lineSpacing: CGFloat = 4
    private let sectionInset = NSEdgeInsets(
        top: AssetCollectionMetrics.sectionVerticalInset,
        left: AssetCollectionMetrics.sectionHorizontalInset,
        bottom: AssetCollectionMetrics.sectionVerticalInset,
        right: AssetCollectionMetrics.sectionHorizontalInset
    )

    init(assets: [AssetItem]) {
        self.assets = assets
        super.init()
    }

    required init?(coder: NSCoder) {
        self.assets = []
        super.init(coder: coder)
    }

    override func prepare() {
        super.prepare()

        guard let collectionView else {
            cachedFrames = []
            contentSize = .zero
            return
        }

        let itemCount = collectionView.numberOfItems(inSection: 0)
        preparedBoundsSize = collectionView.bounds.size
        let contentWidth = contentWidth(forBoundsWidth: collectionView.bounds.width)
        let columnLayout = columnLayout(forContentWidth: contentWidth)
        let columnCount = columnLayout.columnCount
        let itemWidth = columnLayout.itemWidth
        preparedColumnCount = columnCount
        preparedItemWidth = itemWidth
        let startX = sectionInset.left
        var columnHeights = Array(repeating: sectionInset.top, count: columnCount)
        var frames: [NSRect] = []
        frames.reserveCapacity(itemCount)

        for item in 0..<itemCount {
            let columnIndex = shortestColumnIndex(in: columnHeights)
            let itemHeight = masonryItemHeight(forItemAt: item, width: itemWidth)
            let frame = NSRect(
                x: startX + CGFloat(columnIndex) * (itemWidth + interitemSpacing),
                y: columnHeights[columnIndex],
                width: itemWidth,
                height: itemHeight
            )

            frames.append(frame)
            columnHeights[columnIndex] = frame.maxY + lineSpacing
        }

        cachedFrames = frames

        let tallestColumn = columnHeights.max() ?? sectionInset.top
        let contentHeight = max(
            tallestColumn - lineSpacing + sectionInset.bottom,
            collectionView.enclosingScrollView?.contentSize.height ?? collectionView.bounds.height
        )
        contentSize = NSSize(width: contentWidth, height: contentHeight)
    }

    override var collectionViewContentSize: NSSize {
        contentSize
    }

    override func layoutAttributesForElements(in rect: NSRect) -> [NSCollectionViewLayoutAttributes] {
        guard !cachedFrames.isEmpty else {
            return []
        }

        let earliestPossibleMinY = rect.minY - AssetCollectionMetrics.masonryMaximumItemHeight
        var visibleAttributes: [NSCollectionViewLayoutAttributes] = []
        let startIndex = firstAttributeIndex(withMinYAtLeast: earliestPossibleMinY)

        for item in startIndex..<cachedFrames.count {
            let frame = cachedFrames[item]
            if frame.minY > rect.maxY {
                break
            }

            if frame.intersects(rect),
               let attributes = layoutAttributes(forItem: item, frame: frame) {
                visibleAttributes.append(attributes)
            }
        }

        return visibleAttributes
    }

    override func layoutAttributesForItem(at indexPath: IndexPath) -> NSCollectionViewLayoutAttributes? {
        guard cachedFrames.indices.contains(indexPath.item) else {
            return nil
        }

        return layoutAttributes(forItem: indexPath.item, frame: cachedFrames[indexPath.item])
    }

    override func shouldInvalidateLayout(forBoundsChange newBounds: NSRect) -> Bool {
        guard let collectionView, collectionView.inLiveResize else {
            return newBounds.size != preparedBoundsSize
        }

        let newContentWidth = contentWidth(forBoundsWidth: newBounds.width)
        let newColumnLayout = columnLayout(forContentWidth: newContentWidth)
        if newColumnLayout.columnCount != preparedColumnCount {
            return true
        }

        return abs(newColumnLayout.itemWidth - preparedItemWidth) >= AssetCollectionMetrics.liveResizeWidthInvalidationThreshold
    }

    private func contentWidth(forBoundsWidth boundsWidth: CGFloat) -> CGFloat {
        max(boundsWidth, minimumItemWidth + sectionInset.left + sectionInset.right)
    }

    private func columnLayout(forContentWidth contentWidth: CGFloat) -> AssetColumnLayout {
        let availableWidth = max(contentWidth - sectionInset.left - sectionInset.right, minimumItemWidth)
        return AssetCollectionMetrics.columnLayout(
            availableWidth: availableWidth,
            minimumItemWidth: minimumItemWidth,
            interitemSpacing: interitemSpacing
        )
    }

    private func masonryItemHeight(forItemAt item: Int, width: CGFloat) -> CGFloat {
        guard assets.indices.contains(item),
              let dimensions = assets[item].dimensions,
              dimensions.width > 0 else {
            return AssetCollectionMetrics.masonryFallbackItemHeight
        }

        let imageHorizontalInset = AssetCollectionMetrics.masonryImageInset * 2
        let imageVerticalInset = AssetCollectionMetrics.masonryImageInset * 2
        let imageWidth = max(width - imageHorizontalInset, 1)
        let ratio = CGFloat(dimensions.height) / CGFloat(dimensions.width)
        return (imageWidth * ratio + imageVerticalInset).clamped(
            to: AssetCollectionMetrics.masonryMinimumItemHeight...AssetCollectionMetrics.masonryMaximumItemHeight
        )
    }

    private func shortestColumnIndex(in columnHeights: [CGFloat]) -> Int {
        columnHeights.enumerated().min { left, right in
            left.element < right.element
        }?.offset ?? 0
    }

    private func firstAttributeIndex(withMinYAtLeast minY: CGFloat) -> Int {
        var lowerBound = 0
        var upperBound = cachedFrames.count

        while lowerBound < upperBound {
            let middle = (lowerBound + upperBound) / 2
            if cachedFrames[middle].minY < minY {
                lowerBound = middle + 1
            } else {
                upperBound = middle
            }
        }

        return lowerBound
    }

    private func layoutAttributes(forItem item: Int, frame: NSRect) -> NSCollectionViewLayoutAttributes? {
        guard item >= 0 else {
            return nil
        }

        let indexPath = IndexPath(item: item, section: 0)
        let attributes = NSCollectionViewLayoutAttributes(forItemWith: indexPath)
        attributes.frame = frame
        return attributes
    }
}

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

    func image(for asset: AssetItem) -> NSImage {
        let key = identity(for: asset) as NSString
        if let cachedImage = cache.object(forKey: key) {
            return cachedImage
        }

        let image = loadImage(for: asset)
        cache.setObject(image, forKey: key, cost: cacheCost(for: image))
        return image
    }

    func imageAsync(for asset: AssetItem) async -> NSImage {
        let identity = identity(for: asset)
        let key = identity as NSString
        if let cachedImage = cache.object(forKey: key) {
            return cachedImage
        }

        if shouldDecodeThumbnail(for: asset) {
            let image = await decodedThumbnailImageAsync(for: asset, priority: .userInitiated)
            if let image {
                return image
            }
        }

        if Task.isCancelled {
            return placeholderImage(for: asset)
        }

        let image = fallbackIcon(for: asset)
        cache.setObject(image, forKey: key, cost: cacheCost(for: image))
        return image
    }

    func shouldLoadImageAsync(for asset: AssetItem) -> Bool {
        shouldDecodeThumbnail(for: asset)
    }

    func placeholderImage(for asset: AssetItem) -> NSImage {
        if shouldDecodeThumbnail(for: asset),
           let type = UTType(filenameExtension: asset.fileExtension) {
            return NSWorkspace.shared.icon(for: type)
        }

        return fallbackIcon(for: asset)
    }

    func prefetchImage(for asset: AssetItem) {
        let identity = identity(for: asset)
        let key = identity as NSString
        guard cache.object(forKey: key) == nil,
              shouldDecodeThumbnail(for: asset),
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

            let image = autoreleasepool {
                thumbnailDecoder(asset)
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

    private func loadImage(for asset: AssetItem) -> NSImage {
        if asset.kind == .image || asset.kind == .gif {
            if let image = thumbnailDecoder(asset) {
                return image
            }
        }

        return fallbackIcon(for: asset)
    }

    private func fallbackIcon(for asset: AssetItem) -> NSImage {
        fallbackImageProvider(asset)
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

    private func shouldDecodeThumbnail(for asset: AssetItem) -> Bool {
        (asset.kind == .image || asset.kind == .gif) && asset.thumbnailURL != nil
    }

    private func decodedThumbnailImageAsync(for asset: AssetItem, priority: TaskPriority) async -> NSImage? {
        let identity = identity(for: asset)
        let key = identity as NSString
        if let cachedImage = cache.object(forKey: key) {
            return cachedImage
        }

        cancelPrefetch(identity: identity)
        let task = visibleDecodeTask(for: asset, priority: priority)
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
        priority: TaskPriority
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

            let image = autoreleasepool {
                thumbnailDecoder(asset)
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

private final class AssetCollectionViewItem: NSCollectionViewItem {
    static let reuseIdentifier = NSUserInterfaceItemIdentifier("AssetCollectionViewItem")

    var onHoverPreviewChange: ((Bool) -> Void)?
    var onContextMenuOpen: (() -> Void)?
    var onContextMenuAction: ((AssetContextMenuAction) -> Void)?
    var onFavoriteToggle: (() -> Void)?

    private let containerView = HoverTrackingView()
    private let contentView = HoverSelectionView()
    private let previewImageView = AssetPreviewImageView()
    private let favoriteButton = FavoriteButton()
    private let fileNameLabel = NSTextField(labelWithString: "")
    private let subtitleLabel = NSTextField(labelWithString: "")
    private let dateLabel = NSTextField(labelWithString: "")
    private let dimensionBadgeView = DimensionBadgeView()
    private let separatorView = NSView()
    private var gridConstraints: [NSLayoutConstraint] = []
    private var masonryConstraints: [NSLayoutConstraint] = []
    private var listConstraints: [NSLayoutConstraint] = []
    private var mode: AssetViewMode = .grid
    private var asset: AssetItem?
    private var localization = AppLocalization(language: .system)
    private var previewImageTask: Task<Void, Never>?
    private let gridTitleHeight: CGFloat = 16
    private let gridSubtitleHeight: CGFloat = 14

    override var isSelected: Bool {
        didSet {
            contentView.isSelected = isSelected
            updateTextColors()
        }
    }

    override func loadView() {
        containerView.translatesAutoresizingMaskIntoConstraints = false
        containerView.hoverChanged = { [weak self] isHovered in
            self?.contentView.isHovered = isHovered
            self?.updateFavoriteButton(animated: true)
            self?.favoriteButton.synchronizeHoverStateWithPointer()
            if isHovered {
                self?.onHoverPreviewChange?(true)
            } else {
                self?.onHoverPreviewChange?(false)
            }
        }
        let rightClickRecognizer = NSClickGestureRecognizer(
            target: self,
            action: #selector(handleRightClick(_:))
        )
        rightClickRecognizer.buttonMask = 0x2
        containerView.addGestureRecognizer(rightClickRecognizer)

        contentView.translatesAutoresizingMaskIntoConstraints = false
        contentView.layer?.masksToBounds = false

        previewImageView.translatesAutoresizingMaskIntoConstraints = false
        previewImageView.cornerRadius = AssetCollectionMetrics.gridImageCornerRadius

        favoriteButton.translatesAutoresizingMaskIntoConstraints = false
        favoriteButton.isBordered = false
        favoriteButton.imagePosition = .imageOnly
        favoriteButton.bezelStyle = .regularSquare
        favoriteButton.target = self
        favoriteButton.action = #selector(handleFavoriteClick(_:))
        favoriteButton.wantsLayer = true
        favoriteButton.layer?.cornerRadius = AssetCollectionMetrics.favoriteButtonCornerRadius
        favoriteButton.layer?.cornerCurve = .continuous
        favoriteButton.hoverChanged = { [weak self] _ in
            self?.updateFavoriteButton(animated: true)
        }
        favoriteButton.toolTip = localization.string("Favorites")
        favoriteButton.isHidden = true

        fileNameLabel.lineBreakMode = .byTruncatingTail
        fileNameLabel.maximumNumberOfLines = 1
        fileNameLabel.cell?.truncatesLastVisibleLine = true
        fileNameLabel.alignment = .center
        fileNameLabel.font = .systemFont(ofSize: 12, weight: .regular)
        fileNameLabel.textColor = AssetCollectionMetrics.titleTextColor
        fileNameLabel.translatesAutoresizingMaskIntoConstraints = false
        fileNameLabel.setContentCompressionResistancePriority(.required, for: .vertical)
        fileNameLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        fileNameLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

        subtitleLabel.lineBreakMode = .byTruncatingTail
        subtitleLabel.maximumNumberOfLines = 1
        subtitleLabel.cell?.truncatesLastVisibleLine = true
        subtitleLabel.alignment = .center
        subtitleLabel.font = .systemFont(ofSize: 11)
        subtitleLabel.textColor = AssetCollectionMetrics.subtitleTextColor
        subtitleLabel.translatesAutoresizingMaskIntoConstraints = false
        subtitleLabel.setContentCompressionResistancePriority(.required, for: .vertical)
        subtitleLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        subtitleLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

        dateLabel.lineBreakMode = .byClipping
        dateLabel.maximumNumberOfLines = 1
        dateLabel.cell?.truncatesLastVisibleLine = false
        dateLabel.alignment = .center
        dateLabel.font = .systemFont(ofSize: 12, weight: .medium)
        dateLabel.textColor = AssetCollectionMetrics.listDateTextColor
        dateLabel.translatesAutoresizingMaskIntoConstraints = false
        dateLabel.setContentCompressionResistancePriority(.required, for: .horizontal)
        dateLabel.setContentHuggingPriority(.required, for: .horizontal)

        dimensionBadgeView.translatesAutoresizingMaskIntoConstraints = false
        dimensionBadgeView.isHidden = true

        separatorView.translatesAutoresizingMaskIntoConstraints = false
        separatorView.wantsLayer = true
        separatorView.layer?.backgroundColor = NSColor.white.withAlphaComponent(AssetCollectionMetrics.listSeparatorAlpha).cgColor
        separatorView.isHidden = true

        contentView.addSubview(previewImageView)
        contentView.addSubview(fileNameLabel)
        contentView.addSubview(subtitleLabel)
        contentView.addSubview(dateLabel)
        contentView.addSubview(dimensionBadgeView)
        contentView.addSubview(separatorView)
        contentView.addSubview(favoriteButton)
        containerView.addSubview(contentView)
        view = containerView

        NSLayoutConstraint.activate([
            contentView.leadingAnchor.constraint(equalTo: containerView.leadingAnchor),
            contentView.trailingAnchor.constraint(equalTo: containerView.trailingAnchor),
            contentView.topAnchor.constraint(equalTo: containerView.topAnchor),
            contentView.bottomAnchor.constraint(equalTo: containerView.bottomAnchor),

            separatorView.leadingAnchor.constraint(
                equalTo: contentView.leadingAnchor,
                constant: AssetCollectionMetrics.listSeparatorHorizontalInset
            ),
            separatorView.trailingAnchor.constraint(
                equalTo: contentView.trailingAnchor,
                constant: -AssetCollectionMetrics.listSeparatorHorizontalInset
            ),
            separatorView.bottomAnchor.constraint(equalTo: contentView.bottomAnchor),
            separatorView.heightAnchor.constraint(equalToConstant: 1),

            favoriteButton.leadingAnchor.constraint(
                equalTo: previewImageView.leadingAnchor,
                constant: AssetCollectionMetrics.favoriteButtonLeadingInset
            ),
            favoriteButton.topAnchor.constraint(
                equalTo: previewImageView.topAnchor,
                constant: AssetCollectionMetrics.favoriteButtonTopInset
            ),
            favoriteButton.widthAnchor.constraint(equalToConstant: AssetCollectionMetrics.favoriteButtonWidth),
            favoriteButton.heightAnchor.constraint(equalToConstant: AssetCollectionMetrics.favoriteButtonHeight)
        ])

        gridConstraints = [
            previewImageView.topAnchor.constraint(equalTo: contentView.topAnchor, constant: AssetCollectionMetrics.gridImageInset),
            previewImageView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: AssetCollectionMetrics.gridImageInset),
            previewImageView.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -AssetCollectionMetrics.gridImageInset),
            previewImageView.bottomAnchor.constraint(equalTo: fileNameLabel.topAnchor, constant: -AssetCollectionMetrics.gridImageInset),
            previewImageView.heightAnchor.constraint(greaterThanOrEqualToConstant: 96),

            fileNameLabel.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 6),
            fileNameLabel.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -6),
            fileNameLabel.heightAnchor.constraint(equalToConstant: gridTitleHeight),
            fileNameLabel.bottomAnchor.constraint(equalTo: subtitleLabel.topAnchor, constant: -2),

            subtitleLabel.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 6),
            subtitleLabel.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -6),
            subtitleLabel.heightAnchor.constraint(equalToConstant: gridSubtitleHeight),
            subtitleLabel.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -8)
        ]

        masonryConstraints = [
            previewImageView.topAnchor.constraint(equalTo: contentView.topAnchor, constant: AssetCollectionMetrics.masonryImageInset),
            previewImageView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: AssetCollectionMetrics.masonryImageInset),
            previewImageView.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -AssetCollectionMetrics.masonryImageInset),
            previewImageView.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -AssetCollectionMetrics.masonryImageInset),
            dimensionBadgeView.topAnchor.constraint(
                equalTo: previewImageView.topAnchor,
                constant: AssetCollectionMetrics.dimensionBadgeTopInset
            ),
            dimensionBadgeView.trailingAnchor.constraint(equalTo: previewImageView.trailingAnchor, constant: -8),
            dimensionBadgeView.heightAnchor.constraint(equalToConstant: AssetCollectionMetrics.dimensionBadgeHeight)
        ]

        listConstraints = [
            previewImageView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 8),
            previewImageView.centerYAnchor.constraint(equalTo: contentView.centerYAnchor),
            previewImageView.widthAnchor.constraint(equalToConstant: AssetCollectionMetrics.listThumbnailSize),
            previewImageView.heightAnchor.constraint(equalToConstant: AssetCollectionMetrics.listThumbnailSize),

            fileNameLabel.leadingAnchor.constraint(equalTo: previewImageView.trailingAnchor, constant: 10),
            fileNameLabel.trailingAnchor.constraint(equalTo: dateLabel.leadingAnchor, constant: -12),
            fileNameLabel.bottomAnchor.constraint(equalTo: contentView.centerYAnchor, constant: -1),

            subtitleLabel.leadingAnchor.constraint(equalTo: fileNameLabel.leadingAnchor),
            subtitleLabel.trailingAnchor.constraint(equalTo: fileNameLabel.trailingAnchor),
            subtitleLabel.topAnchor.constraint(equalTo: fileNameLabel.bottomAnchor, constant: 4),
            subtitleLabel.bottomAnchor.constraint(lessThanOrEqualTo: contentView.bottomAnchor, constant: -8),

            dateLabel.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -14),
            dateLabel.centerYAnchor.constraint(equalTo: contentView.centerYAnchor)
        ]
    }

    override func prepareForReuse() {
        super.prepareForReuse()
        previewImageTask?.cancel()
        previewImageTask = nil
        previewImageView.resetImage()
        fileNameLabel.stringValue = ""
        subtitleLabel.stringValue = ""
        dateLabel.stringValue = ""
        dimensionBadgeView.stringValue = ""
        dimensionBadgeView.isHidden = true
        separatorView.isHidden = true
        containerView.resetHoverState()
        favoriteButton.resetAppearance()
        onHoverPreviewChange = nil
        onContextMenuOpen = nil
        onContextMenuAction = nil
        onFavoriteToggle = nil
        asset = nil
        updateFavoriteButton(animated: false)
        contentView.isHovered = false
        contentView.isSelected = false
        contentView.viewMode = .grid
        dateLabel.isHidden = true
        mode = .grid
        updateTextColors()
    }

    func configure(with asset: AssetItem, viewMode: AssetViewMode, localization: AppLocalization) {
        previewImageTask?.cancel()
        previewImageTask = nil
        self.asset = asset
        self.localization = localization
        mode = viewMode
        contentView.viewMode = viewMode
        fileNameLabel.stringValue = asset.displayName
        subtitleLabel.stringValue = subtitle(for: asset, viewMode: viewMode, localization: localization)
        dateLabel.stringValue = viewMode == .list ? localization.relativeOrDateTime(asset.importedAt) : ""
        dimensionBadgeView.stringValue = dimensionsSubtitle(for: asset)
        let previewProvider = AssetPreviewImageProvider.shared
        let previewIdentity = previewProvider.identity(for: asset)
        let viewImageIdentity = "\(viewMode.rawValue):\(previewIdentity)"
        let cachedImage = previewProvider.cachedImage(for: asset)
        let canLoadImageAsync = previewProvider.shouldLoadImageAsync(for: asset)
        previewImageView.setImage(
            cachedImage ?? (canLoadImageAsync ? previewProvider.placeholderImage(for: asset) : previewProvider.image(for: asset)),
            identity: cachedImage == nil && canLoadImageAsync ? "\(viewImageIdentity):placeholder" : viewImageIdentity,
            animated: false
        )
        if cachedImage == nil && canLoadImageAsync {
            previewImageTask = Task { @MainActor [weak self, asset, viewMode, viewImageIdentity] in
                let image = await AssetPreviewImageProvider.shared.imageAsync(for: asset)
                guard !Task.isCancelled,
                      let self,
                      self.asset?.id == asset.id,
                      self.mode == viewMode else {
                    return
                }

                self.previewImageView.setImage(
                    image,
                    identity: viewImageIdentity,
                    animated: false
                )
                self.previewImageTask = nil
            }
        }
        previewImageView.contentMode = imageContentMode(for: asset, viewMode: viewMode)
        applyModeLayout()
        updateFavoriteButton(animated: false)
        containerView.synchronizeHoverStateWithPointer()
        favoriteButton.synchronizeHoverStateWithPointer()
    }

    func updateVisibleState(with asset: AssetItem, viewMode: AssetViewMode, localization: AppLocalization) {
        guard self.asset?.id == asset.id else {
            return
        }

        self.asset = asset
        self.localization = localization
        fileNameLabel.stringValue = asset.displayName
        subtitleLabel.stringValue = subtitle(for: asset, viewMode: viewMode, localization: localization)
        dateLabel.stringValue = viewMode == .list ? localization.relativeOrDateTime(asset.importedAt) : ""
        dimensionBadgeView.stringValue = dimensionsSubtitle(for: asset)
        dimensionBadgeView.isHidden = viewMode != .masonry || dimensionBadgeView.stringValue.isEmpty
        updateFavoriteButton(animated: true)
        updateTextColors()
    }

    @objc private func handleFavoriteClick(_ sender: NSButton) {
        onFavoriteToggle?()
    }

    @objc private func handleRightClick(_ sender: NSClickGestureRecognizer) {
        showContextMenu(at: sender.location(in: containerView))
    }

    private func showContextMenu(at location: NSPoint) {
        guard let asset else {
            return
        }

        onContextMenuOpen?()
        let menu = NSMenu()
        menu.showsStateColumn = false
        let menuItem = NSMenuItem()
        let menuView = AssetContextMenuView(
            localization: localization,
            actions: AssetContextMenuAction.actions(for: asset),
            onSelect: { [weak self] action in
                self?.onContextMenuAction?(action)
            }
        )
        menuView.frame = NSRect(origin: .zero, size: menuView.intrinsicContentSize)
        menuView.autoresizingMask = [.width]
        menuItem.view = menuView
        menu.addItem(menuItem)
        menu.popUp(positioning: nil, at: location, in: containerView)
    }

    private func applyModeLayout() {
        NSLayoutConstraint.deactivate(gridConstraints + masonryConstraints + listConstraints)

        switch mode {
        case .grid:
            fileNameLabel.isHidden = false
            subtitleLabel.isHidden = false
            dateLabel.isHidden = true
            dimensionBadgeView.isHidden = true
            fileNameLabel.alignment = .center
            fileNameLabel.maximumNumberOfLines = 1
            fileNameLabel.lineBreakMode = .byTruncatingTail
            fileNameLabel.font = .systemFont(ofSize: 12, weight: .regular)
            subtitleLabel.alignment = .center
            subtitleLabel.maximumNumberOfLines = 1
            subtitleLabel.lineBreakMode = .byTruncatingTail
            subtitleLabel.font = .systemFont(ofSize: 11, weight: .regular)
            separatorView.isHidden = true
            previewImageView.cornerRadius = AssetCollectionMetrics.gridImageCornerRadius
            NSLayoutConstraint.activate(gridConstraints)
        case .masonry:
            fileNameLabel.isHidden = true
            subtitleLabel.isHidden = true
            dateLabel.isHidden = true
            dimensionBadgeView.isHidden = dimensionBadgeView.stringValue.isEmpty
            separatorView.isHidden = true
            previewImageView.cornerRadius = AssetCollectionMetrics.masonryImageCornerRadius
            NSLayoutConstraint.activate(masonryConstraints)
        case .list:
            fileNameLabel.isHidden = false
            subtitleLabel.isHidden = false
            dateLabel.isHidden = false
            dimensionBadgeView.isHidden = true
            fileNameLabel.alignment = .left
            fileNameLabel.maximumNumberOfLines = 1
            fileNameLabel.lineBreakMode = .byTruncatingTail
            fileNameLabel.font = .systemFont(ofSize: 13, weight: .regular)
            subtitleLabel.alignment = .left
            subtitleLabel.maximumNumberOfLines = 1
            subtitleLabel.lineBreakMode = .byTruncatingTail
            subtitleLabel.font = .systemFont(ofSize: 11, weight: .regular)
            separatorView.isHidden = false
            previewImageView.cornerRadius = AssetCollectionMetrics.listImageCornerRadius
            NSLayoutConstraint.activate(listConstraints)
        }

        updateTextColors()
    }

    private func updateTextColors() {
        let usesSelectedText = isSelected && (mode == .grid || mode == .list)
        fileNameLabel.textColor = usesSelectedText
            ? AssetCollectionMetrics.selectedTitleTextColor
            : AssetCollectionMetrics.titleTextColor
        subtitleLabel.textColor = usesSelectedText
            ? AssetCollectionMetrics.selectedSubtitleTextColor
            : AssetCollectionMetrics.subtitleTextColor
        dateLabel.textColor = usesSelectedText
            ? AssetCollectionMetrics.selectedTitleTextColor
            : AssetCollectionMetrics.listDateTextColor
    }

    private func updateFavoriteButton(animated: Bool = true) {
        guard let asset else {
            favoriteButton.image = nil
            favoriteButton.resetAppearance()
            return
        }

        let isVisible = asset.isFavorite || contentView.isHovered
        if !isVisible {
            favoriteButton.resetHoverState()
        }
        favoriteButton.image = favoriteImage(isFavorite: asset.isFavorite)
        favoriteButton.contentTintColor = asset.isFavorite
            ? .systemRed
            : NSColor.white.withAlphaComponent(0.92)
        favoriteButton.setHoverBackgroundVisible(isVisible && favoriteButton.isPointerInside, animated: animated)
        favoriteButton.setVisible(isVisible, animated: animated)
        favoriteButton.toolTip = localization.string(asset.isFavorite ? "Favorited" : "Favorites")
    }

    private func favoriteImage(isFavorite: Bool) -> NSImage? {
        NSImage(
            systemSymbolName: isFavorite ? "heart.fill" : "heart",
            accessibilityDescription: localization.string("Favorites")
        )?.withSymbolConfiguration(.init(pointSize: AssetCollectionMetrics.favoriteSymbolPointSize, weight: .semibold))
    }

    private func subtitle(for asset: AssetItem, viewMode: AssetViewMode, localization: AppLocalization) -> String {
        var parts: [String] = []
        if let dimensions = asset.dimensions {
            parts.append("\(dimensions.width) x \(dimensions.height)")
        }

        if viewMode == .list {
            parts.append(localization.fileSize(asset.byteSize))
        }

        return parts.joined(separator: " • ")
    }

    private func dimensionsSubtitle(for asset: AssetItem) -> String {
        guard let dimensions = asset.dimensions else {
            return ""
        }

        return "\(dimensions.width) x \(dimensions.height)"
    }

    private func imageContentMode(for asset: AssetItem, viewMode: AssetViewMode) -> AssetPreviewImageView.ContentMode {
        switch viewMode {
        case .grid, .masonry, .list:
            return .aspectFill
        }
    }

    func previewSourceFrameInScreen() -> NSRect? {
        guard let window = previewImageView.window else {
            return nil
        }

        let rectInWindow = previewImageView.convert(previewImageView.visibleImageBounds, to: nil)
        return window.convertToScreen(rectInWindow)
    }

    func dragSourcePlaceholderImage() -> NSImage? {
        view.layoutSubtreeIfNeeded()

        let bounds = view.bounds
        guard bounds.width > 0,
              bounds.height > 0,
              let representation = view.bitmapImageRepForCachingDisplay(in: bounds) else {
            return nil
        }

        representation.size = bounds.size
        view.cacheDisplay(in: bounds, to: representation)

        let image = NSImage(size: bounds.size)
        image.addRepresentation(representation)
        return image
    }
}

private final class AssetPreviewImageView: NSView {
    private static let imageEntranceAnimationKey = "assetImageEntrance"

    enum ContentMode {
        case aspectFit
        case aspectFill
    }

    private var image: NSImage? {
        didSet {
            needsDisplay = true
        }
    }
    private var imageIdentity: String?

    var contentMode: ContentMode = .aspectFit {
        didSet {
            needsDisplay = true
        }
    }

    var cornerRadius: CGFloat = 0 {
        didSet {
            needsDisplay = true
        }
    }

    override var isFlipped: Bool {
        true
    }

    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        setUpLayer()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setUpLayer()
    }

    func setImage(_ newImage: NSImage, identity: String, animated: Bool) {
        let shouldAnimate = animated && imageIdentity != identity
        imageIdentity = identity
        image = newImage

        if shouldAnimate {
            animateImageEntrance()
        }
    }

    func resetImage() {
        imageIdentity = nil
        image = nil
        layer?.removeAnimation(forKey: Self.imageEntranceAnimationKey)
    }

    override func draw(_ dirtyRect: NSRect) {
        super.draw(dirtyRect)

        guard let image, image.size.width > 0, image.size.height > 0, bounds.width > 0, bounds.height > 0 else {
            return
        }

        let drawRect = imageDrawRect(for: image.size)
        let clipPath = NSBezierPath(roundedRect: visibleImageBounds, xRadius: cornerRadius, yRadius: cornerRadius)
        NSGraphicsContext.saveGraphicsState()
        clipPath.addClip()
        NSGraphicsContext.current?.imageInterpolation = .high
        image.draw(in: drawRect)
        NSGraphicsContext.restoreGraphicsState()
    }

    var visibleImageBounds: NSRect {
        guard let image, image.size.width > 0, image.size.height > 0, bounds.width > 0, bounds.height > 0 else {
            return bounds
        }

        switch contentMode {
        case .aspectFit:
            return imageDrawRect(for: image.size).intersection(bounds)
        case .aspectFill:
            return bounds
        }
    }

    private func imageDrawRect(for imageSize: NSSize) -> NSRect {
        let widthScale = bounds.width / imageSize.width
        let heightScale = bounds.height / imageSize.height
        let scale: CGFloat

        switch contentMode {
        case .aspectFit:
            scale = min(widthScale, heightScale)
        case .aspectFill:
            scale = max(widthScale, heightScale)
        }

        let drawSize = NSSize(width: imageSize.width * scale, height: imageSize.height * scale)
        return NSRect(
            x: bounds.midX - drawSize.width / 2,
            y: bounds.midY - drawSize.height / 2,
            width: drawSize.width,
            height: drawSize.height
        )
    }

    private func setUpLayer() {
        wantsLayer = true
        layer?.opacity = 1
    }

    private func animateImageEntrance() {
        guard !NSWorkspace.shared.accessibilityDisplayShouldReduceMotion,
              let layer else {
            return
        }

        layer.removeAnimation(forKey: Self.imageEntranceAnimationKey)

        let opacity = CABasicAnimation(keyPath: "opacity")
        opacity.fromValue = 0
        opacity.toValue = 1

        let scale = CABasicAnimation(keyPath: "transform.scale")
        scale.fromValue = AssetCollectionMetrics.imageEntranceScale
        scale.toValue = 1

        let group = CAAnimationGroup()
        group.animations = [opacity, scale]
        group.duration = AssetCollectionMetrics.imageEntranceAnimationDuration
        group.timingFunction = CAMediaTimingFunction(name: .easeOut)

        layer.add(group, forKey: Self.imageEntranceAnimationKey)
    }
}

private final class DimensionBadgeView: NSView {
    var stringValue: String {
        get {
            label.stringValue
        }
        set {
            label.stringValue = newValue
        }
    }

    private let label = NSTextField(labelWithString: "")

    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        setUpView()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setUpView()
    }

    private func setUpView() {
        wantsLayer = true
        layer?.backgroundColor = NSColor.black.withAlphaComponent(0.34).cgColor
        layer?.cornerRadius = AssetCollectionMetrics.dimensionBadgeCornerRadius
        layer?.cornerCurve = .continuous

        label.lineBreakMode = .byTruncatingTail
        label.maximumNumberOfLines = 1
        label.alignment = .center
        label.font = .systemFont(ofSize: 10, weight: .medium)
        label.textColor = .white.withAlphaComponent(0.82)
        label.translatesAutoresizingMaskIntoConstraints = false

        addSubview(label)

        NSLayoutConstraint.activate([
            label.leadingAnchor.constraint(equalTo: leadingAnchor, constant: AssetCollectionMetrics.dimensionBadgeHorizontalPadding),
            label.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -AssetCollectionMetrics.dimensionBadgeHorizontalPadding),
            label.centerYAnchor.constraint(equalTo: centerYAnchor)
        ])
    }
}

private final class AssetContextMenuView: NSView {
    private let backgroundView = AssetContextMenuBackgroundView()
    private let rowViews: [AssetContextMenuRowView]
    private let separatorViewsByRowIndex: [Int: AssetContextMenuSeparatorView]

    override var isFlipped: Bool {
        true
    }

    init(
        localization: AppLocalization,
        actions: [AssetContextMenuAction],
        onSelect: @escaping (AssetContextMenuAction) -> Void
    ) {
        rowViews = actions.map { action in
            AssetContextMenuRowView(
                title: localization.string(action.titleKey),
                systemImageName: action.systemImageName,
                isDestructive: action.isDestructive,
                onSelect: { onSelect(action) }
            )
        }
        separatorViewsByRowIndex = Dictionary(
            uniqueKeysWithValues: actions.enumerated().compactMap { index, action in
                action.showsSeparatorAfter ? (index, AssetContextMenuSeparatorView()) : nil
            }
        )
        super.init(frame: .zero)
        setUpView()
    }

    required init?(coder: NSCoder) {
        rowViews = []
        separatorViewsByRowIndex = [:]
        super.init(coder: coder)
        setUpView()
    }

    override var intrinsicContentSize: NSSize {
        let rowCount = CGFloat(rowViews.count)
        return NSSize(
            width: AssetCollectionMetrics.contextMenuWidth,
            height: AssetCollectionMetrics.contextMenuPadding * 2
                + rowCount * AssetCollectionMetrics.contextMenuRowHeight
                + max(rowCount - 1, 0) * AssetCollectionMetrics.contextMenuRowSpacing
                + CGFloat(separatorViewsByRowIndex.count) * (
                    AssetCollectionMetrics.contextMenuSeparatorHeight
                        + AssetCollectionMetrics.contextMenuSeparatorVerticalPadding * 2
                )
        )
    }

    override func layout() {
        super.layout()

        backgroundView.frame = bounds.insetBy(
            dx: -AssetCollectionMetrics.contextMenuPanelBleed,
            dy: -AssetCollectionMetrics.contextMenuPanelBleed
        )

        let rowWidth = max(bounds.width - AssetCollectionMetrics.contextMenuPadding * 2, 0)
        var y = AssetCollectionMetrics.contextMenuPadding
        for (index, rowView) in rowViews.enumerated() {
            rowView.frame = NSRect(
                x: AssetCollectionMetrics.contextMenuPadding,
                y: y,
                width: rowWidth,
                height: AssetCollectionMetrics.contextMenuRowHeight
            )
            y += AssetCollectionMetrics.contextMenuRowHeight

            if let separatorView = separatorViewsByRowIndex[index] {
                y += AssetCollectionMetrics.contextMenuSeparatorVerticalPadding
                separatorView.frame = NSRect(
                    x: AssetCollectionMetrics.contextMenuPadding
                        + AssetCollectionMetrics.contextMenuSeparatorHorizontalInset,
                    y: y,
                    width: max(
                        rowWidth - AssetCollectionMetrics.contextMenuSeparatorHorizontalInset * 2,
                        0
                    ),
                    height: AssetCollectionMetrics.contextMenuSeparatorHeight
                )
                y += AssetCollectionMetrics.contextMenuSeparatorHeight
                    + AssetCollectionMetrics.contextMenuSeparatorVerticalPadding
            }

            if index < rowViews.count - 1 {
                y += AssetCollectionMetrics.contextMenuRowSpacing
            }
        }
    }

    private func setUpView() {
        addSubview(backgroundView)
        for rowView in rowViews {
            addSubview(rowView)
        }
        for separatorView in separatorViewsByRowIndex.values {
            addSubview(separatorView)
        }
    }
}

private final class AssetContextMenuBackgroundView: NSGlassEffectView {
    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        setUpView()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setUpView()
    }

    private func setUpView() {
        style = .regular
        tintColor = NSColor.black.withAlphaComponent(0.16)
        cornerRadius = AssetCollectionMetrics.contextMenuPanelCornerRadius
        wantsLayer = true
        layer?.backgroundColor = NSColor.clear.cgColor
        layer?.cornerRadius = AssetCollectionMetrics.contextMenuPanelCornerRadius
        layer?.cornerCurve = .continuous
        layer?.borderColor = NSColor.white.withAlphaComponent(0.12).cgColor
        layer?.borderWidth = 0.6
    }
}

private final class AssetContextMenuSeparatorView: NSView {
    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        setUpView()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setUpView()
    }

    private func setUpView() {
        wantsLayer = true
        layer?.backgroundColor = NSColor.separatorColor
            .withAlphaComponent(AssetCollectionMetrics.contextMenuSeparatorAlpha)
            .cgColor
    }
}

private final class AssetContextMenuRowView: NSView {
    private let title: String
    private let systemImageName: String
    private let isDestructive: Bool
    private let onSelect: () -> Void
    private let imageView = NSImageView()
    private let label = NSTextField(labelWithString: "")
    private var trackingArea: NSTrackingArea?
    private var didPushCursor = false
    private var isHovered = false {
        didSet {
            updateAppearance()
        }
    }

    init(
        title: String,
        systemImageName: String,
        isDestructive: Bool,
        onSelect: @escaping () -> Void
    ) {
        self.title = title
        self.systemImageName = systemImageName
        self.isDestructive = isDestructive
        self.onSelect = onSelect
        super.init(frame: .zero)
        setUpView()
    }

    required init?(coder: NSCoder) {
        title = ""
        systemImageName = ""
        isDestructive = false
        onSelect = {}
        super.init(coder: coder)
        setUpView()
    }

    override func acceptsFirstMouse(for event: NSEvent?) -> Bool {
        true
    }

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()

        if window == nil {
            popPointingHandCursorIfNeeded()
        }
    }

    override func updateTrackingAreas() {
        super.updateTrackingAreas()

        if let trackingArea {
            removeTrackingArea(trackingArea)
        }

        let newTrackingArea = NSTrackingArea(
            rect: bounds,
            options: [.activeInActiveApp, .mouseEnteredAndExited, .inVisibleRect],
            owner: self
        )
        addTrackingArea(newTrackingArea)
        trackingArea = newTrackingArea
    }

    override func mouseEntered(with event: NSEvent) {
        isHovered = true
        pushPointingHandCursorIfNeeded()
    }

    override func mouseExited(with event: NSEvent) {
        isHovered = false
        popPointingHandCursorIfNeeded()
    }

    override func mouseUp(with event: NSEvent) {
        guard bounds.contains(convert(event.locationInWindow, from: nil)) else {
            return
        }

        popPointingHandCursorIfNeeded()
        enclosingMenuItem?.menu?.cancelTracking()
        onSelect()
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }

    private func setUpView() {
        wantsLayer = true
        layer?.cornerRadius = AssetCollectionMetrics.contextMenuRowCornerRadius
        layer?.cornerCurve = .continuous

        imageView.translatesAutoresizingMaskIntoConstraints = false
        imageView.image = NSImage(systemSymbolName: systemImageName, accessibilityDescription: title)
        imageView.symbolConfiguration = .init(pointSize: 13, weight: .medium)
        imageView.contentTintColor = isDestructive ? .systemRed : .secondaryLabelColor

        label.stringValue = title
        label.font = .systemFont(ofSize: 12, weight: .medium)
        label.textColor = isDestructive ? .systemRed : .labelColor
        label.lineBreakMode = .byTruncatingTail
        label.translatesAutoresizingMaskIntoConstraints = false

        addSubview(imageView)
        addSubview(label)

        NSLayoutConstraint.activate([
            imageView.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 8),
            imageView.centerYAnchor.constraint(equalTo: centerYAnchor),
            imageView.widthAnchor.constraint(equalToConstant: 16),
            imageView.heightAnchor.constraint(equalToConstant: 16),

            label.leadingAnchor.constraint(equalTo: imageView.trailingAnchor, constant: 8),
            label.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -8),
            label.centerYAnchor.constraint(equalTo: centerYAnchor)
        ])

        updateAppearance()
    }

    private func updateAppearance() {
        layer?.backgroundColor = isHovered
            ? NSColor.white.withAlphaComponent(AssetCollectionMetrics.hoverBackgroundAlpha).cgColor
            : NSColor.clear.cgColor
    }

    private func pushPointingHandCursorIfNeeded() {
        guard !didPushCursor else {
            return
        }

        NSCursor.pointingHand.push()
        didPushCursor = true
    }

    private func popPointingHandCursorIfNeeded() {
        guard didPushCursor else {
            return
        }

        NSCursor.pop()
        didPushCursor = false
    }
}

private final class FavoriteButton: NSButton {
    private static let backgroundAnimationKey = "favoriteButtonBackground"
    private static let scaleAnimationKey = "favoriteButtonScale"

    var hoverChanged: ((Bool) -> Void)?
    private var trackingArea: NSTrackingArea?
    private var visibilityAnimationGeneration = 0
    private var showsHoverBackground = false
    private(set) var isPointerInside = false

    override func updateTrackingAreas() {
        super.updateTrackingAreas()

        if let trackingArea {
            removeTrackingArea(trackingArea)
        }

        let options: NSTrackingArea.Options = [.activeInActiveApp, .mouseEnteredAndExited, .inVisibleRect]
        let newTrackingArea = NSTrackingArea(rect: bounds, options: options, owner: self)
        addTrackingArea(newTrackingArea)
        trackingArea = newTrackingArea
        synchronizeHoverStateWithPointer()
    }

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()

        if window == nil {
            resetHoverState()
        } else {
            synchronizeHoverStateWithPointer()
        }
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }

    override func mouseEntered(with event: NSEvent) {
        setHovered(true)
    }

    override func mouseExited(with event: NSEvent) {
        setHovered(false)
    }

    func resetHoverState() {
        setHovered(false, notify: false)
    }

    func resetAppearance() {
        visibilityAnimationGeneration += 1
        isEnabled = false
        isHidden = true
        alphaValue = 1
        resetHoverState()
        resetLayerAnimations()
        setHoverBackgroundVisible(false, animated: false)
        layer?.backgroundColor = NSColor.clear.cgColor
    }

    func synchronizeHoverStateWithPointer() {
        guard let window,
              !isHidden,
              !bounds.isEmpty else {
            setHovered(false)
            return
        }

        let pointerLocation = convert(window.mouseLocationOutsideOfEventStream, from: nil)
        setHovered(bounds.contains(pointerLocation) && visibleRect.contains(pointerLocation))
    }

    func setVisible(_ isVisible: Bool, animated: Bool) {
        visibilityAnimationGeneration += 1
        let generation = visibilityAnimationGeneration

        guard animated && !NSWorkspace.shared.accessibilityDisplayShouldReduceMotion else {
            resetLayerAnimations()
            alphaValue = 1
            isEnabled = isVisible
            isHidden = !isVisible
            return
        }

        if isVisible {
            let shouldAnimateEntrance = isHidden
            isHidden = false
            isEnabled = true

            if shouldAnimateEntrance {
                alphaValue = 0
                animateEntrance()
            } else {
                alphaValue = 1
            }
        } else {
            guard !isHidden else {
                isEnabled = false
                alphaValue = 1
                return
            }

            isEnabled = false
            resetLayerAnimations()
            NSAnimationContext.runAnimationGroup { context in
                context.duration = AssetCollectionMetrics.favoriteButtonAppearanceAnimationDuration * 0.75
                context.timingFunction = Self.favoriteTimingFunction()
                animator().alphaValue = 0
            } completionHandler: { [weak self] in
                Task { @MainActor [weak self] in
                    guard let self,
                          self.visibilityAnimationGeneration == generation else {
                        return
                    }

                    self.isHidden = true
                    self.alphaValue = 1
                }
            }
        }
    }

    func setHoverBackgroundVisible(_ isVisible: Bool, animated: Bool) {
        guard showsHoverBackground != isVisible else {
            return
        }

        showsHoverBackground = isVisible
        guard let layer else {
            return
        }

        let targetColor = isVisible
            ? NSColor.black.withAlphaComponent(AssetCollectionMetrics.favoriteButtonBackgroundAlpha).cgColor
            : NSColor.clear.cgColor
        let startColor = layer.presentation()?.backgroundColor ?? layer.backgroundColor ?? NSColor.clear.cgColor

        CATransaction.begin()
        CATransaction.setDisableActions(true)
        layer.backgroundColor = targetColor
        CATransaction.commit()

        guard animated && !NSWorkspace.shared.accessibilityDisplayShouldReduceMotion else {
            layer.removeAnimation(forKey: Self.backgroundAnimationKey)
            return
        }

        let animation = CABasicAnimation(keyPath: "backgroundColor")
        animation.fromValue = startColor
        animation.toValue = targetColor
        animation.duration = AssetCollectionMetrics.favoriteButtonBackgroundAnimationDuration
        animation.timingFunction = Self.favoriteTimingFunction()
        layer.add(animation, forKey: Self.backgroundAnimationKey)
    }

    private func setHovered(_ isHovered: Bool, notify: Bool = true) {
        guard isPointerInside != isHovered else {
            return
        }

        isPointerInside = isHovered
        if notify {
            hoverChanged?(isHovered)
        }
    }

    private func animateEntrance() {
        guard let layer else {
            alphaValue = 1
            return
        }

        resetLayerAnimations()
        NSAnimationContext.runAnimationGroup { context in
            context.duration = AssetCollectionMetrics.favoriteButtonAppearanceAnimationDuration
            context.timingFunction = Self.favoriteTimingFunction()
            animator().alphaValue = 1
        }

        let scaleAnimation = CABasicAnimation(keyPath: "transform.scale")
        scaleAnimation.fromValue = AssetCollectionMetrics.favoriteButtonEntranceScale
        scaleAnimation.toValue = 1
        scaleAnimation.duration = AssetCollectionMetrics.favoriteButtonAppearanceAnimationDuration
        scaleAnimation.timingFunction = Self.favoriteTimingFunction()
        layer.add(scaleAnimation, forKey: Self.scaleAnimationKey)
    }

    private func resetLayerAnimations() {
        layer?.removeAnimation(forKey: Self.scaleAnimationKey)
        layer?.removeAnimation(forKey: Self.backgroundAnimationKey)
    }

    private static func favoriteTimingFunction() -> CAMediaTimingFunction {
        CAMediaTimingFunction(controlPoints: 0.22, 1, 0.36, 1)
    }
}

private final class HoverTrackingView: NSView {
    var hoverChanged: ((Bool) -> Void)?
    private var trackingArea: NSTrackingArea?
    private var isPointerInside = false

    override func updateTrackingAreas() {
        super.updateTrackingAreas()

        if let trackingArea {
            removeTrackingArea(trackingArea)
        }

        let options: NSTrackingArea.Options = [.activeInActiveApp, .mouseEnteredAndExited, .inVisibleRect]
        let newTrackingArea = NSTrackingArea(rect: bounds, options: options, owner: self)
        addTrackingArea(newTrackingArea)
        trackingArea = newTrackingArea
        synchronizeHoverStateWithPointer()
    }

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()

        if window == nil {
            setHovered(false)
        } else {
            synchronizeHoverStateWithPointer()
        }
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }

    override func mouseEntered(with event: NSEvent) {
        setHovered(true)
    }

    override func mouseExited(with event: NSEvent) {
        setHovered(false)
    }

    func resetHoverState() {
        setHovered(false)
    }

    func synchronizeHoverStateWithPointer() {
        guard let window,
              !isHidden,
              !bounds.isEmpty else {
            setHovered(false)
            return
        }

        let pointerLocation = convert(window.mouseLocationOutsideOfEventStream, from: nil)
        setHovered(bounds.contains(pointerLocation) && visibleRect.contains(pointerLocation))
    }

    private func setHovered(_ isHovered: Bool) {
        guard isPointerInside != isHovered else {
            return
        }

        isPointerInside = isHovered
        hoverChanged?(isHovered)
    }
}

private final class HoverSelectionView: NSView {
    private static let backgroundColorAnimationKey = "selectionBackgroundColor"

    var isHovered = false {
        didSet {
            updateAppearance()
        }
    }
    var isSelected = false {
        didSet {
            updateAppearance()
        }
    }
    var viewMode: AssetViewMode = .grid {
        didSet {
            updateAppearance()
        }
    }

    private let glassBackgroundView = NSGlassEffectView()

    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        setUpGlassBackground()
    }

    required init?(coder: NSCoder) {
        super.init(coder: coder)
        setUpGlassBackground()
    }

    private func setUpGlassBackground() {
        wantsLayer = true
        layer?.cornerRadius = AssetCollectionMetrics.selectionCornerRadius
        layer?.cornerCurve = .continuous

        glassBackgroundView.translatesAutoresizingMaskIntoConstraints = false
        glassBackgroundView.cornerRadius = AssetCollectionMetrics.selectionCornerRadius
        glassBackgroundView.style = .regular
        glassBackgroundView.isHidden = true
        addSubview(glassBackgroundView)

        NSLayoutConstraint.activate([
            glassBackgroundView.leadingAnchor.constraint(equalTo: leadingAnchor),
            glassBackgroundView.trailingAnchor.constraint(equalTo: trailingAnchor),
            glassBackgroundView.topAnchor.constraint(equalTo: topAnchor),
            glassBackgroundView.bottomAnchor.constraint(equalTo: bottomAnchor)
        ])
    }

    override func layout() {
        super.layout()
        updateShadowPath()
    }

    func updateShadowPath() {
        layer?.shadowPath = CGPath(
            roundedRect: bounds,
            cornerWidth: AssetCollectionMetrics.selectionCornerRadius,
            cornerHeight: AssetCollectionMetrics.selectionCornerRadius,
            transform: nil
        )
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }

    private func updateAppearance() {
        wantsLayer = true
        glassBackgroundView.isHidden = true
        layer?.borderColor = NSColor.clear.cgColor
        layer?.borderWidth = 0

        let targetColor = backgroundColor
        updateBackgroundColor(targetColor)
    }

    private var backgroundColor: CGColor {
        if isSelected {
            return NSColor.controlAccentColor.cgColor
        }

        if isHovered {
            return NSColor.white.withAlphaComponent(AssetCollectionMetrics.hoverBackgroundAlpha).cgColor
        }

        return NSColor.clear.cgColor
    }

    private func updateBackgroundColor(_ targetColor: CGColor) {
        guard let layer else {
            return
        }

        let currentColor = layer.presentation()?.backgroundColor ?? layer.backgroundColor ?? NSColor.clear.cgColor
        if currentColor == targetColor {
            return
        }

        layer.backgroundColor = targetColor

        guard !NSWorkspace.shared.accessibilityDisplayShouldReduceMotion else {
            layer.removeAnimation(forKey: Self.backgroundColorAnimationKey)
            return
        }

        let animation = CABasicAnimation(keyPath: "backgroundColor")
        animation.fromValue = currentColor
        animation.toValue = targetColor
        animation.duration = AssetCollectionMetrics.selectionBackgroundAnimationDuration
        animation.timingFunction = CAMediaTimingFunction(name: .easeOut)
        layer.add(animation, forKey: Self.backgroundColorAnimationKey)
    }
}

private extension CGFloat {
    func clamped(to range: ClosedRange<CGFloat>) -> CGFloat {
        Swift.min(Swift.max(self, range.lowerBound), range.upperBound)
    }
}

private extension NSEdgeInsets {
    var areZero: Bool {
        top == 0 && left == 0 && bottom == 0 && right == 0
    }
}
