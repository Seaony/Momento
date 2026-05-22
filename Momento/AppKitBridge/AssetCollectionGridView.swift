//
//  AssetCollectionGridView.swift
//  Momento
//

import AppKit
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
    static let dimensionBadgeHorizontalPadding: CGFloat = 3
    static let sectionHorizontalInset: CGFloat = 8
    static let sectionVerticalInset: CGFloat = 14
    static let listItemHeight: CGFloat = 96
    static let listThumbnailSize: CGFloat = 78
    static let listSeparatorHorizontalInset: CGFloat = 18
    static let listSeparatorAlpha: CGFloat = 0.055
    static let selectionBackgroundAnimationDuration: CFTimeInterval = 0.12
    static let titleTextColor = NSColor.labelColor.withAlphaComponent(0.5)
    static let subtitleTextColor = NSColor.labelColor.withAlphaComponent(0.3)
    static let listDateTextColor = NSColor.labelColor.withAlphaComponent(0.72)
    static let selectedTitleTextColor = NSColor.white.withAlphaComponent(0.95)
    static let selectedSubtitleTextColor = NSColor.white.withAlphaComponent(0.72)
    static let imageEntranceAnimationDuration: CFTimeInterval = 0.18
    static let imageEntranceScale: CGFloat = 0.985
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
}

enum AssetContextMenuAction: CaseIterable {
    case previewOriginal
    case refreshThumbnail
    case reanalyzeColors
    case revealInFinder
    case moveToTrash

    var titleKey: String {
        switch self {
        case .previewOriginal:
            "Preview Original"
        case .refreshThumbnail:
            "Refresh Thumbnail"
        case .reanalyzeColors:
            "Reanalyze Colors"
        case .revealInFinder:
            "Reveal in Finder"
        case .moveToTrash:
            "Move to Trash"
        }
    }

    var systemImageName: String {
        switch self {
        case .previewOriginal:
            "eye"
        case .refreshThumbnail:
            "arrow.clockwise"
        case .reanalyzeColors:
            "paintpalette"
        case .revealInFinder:
            "finder"
        case .moveToTrash:
            "trash"
        }
    }

    var isDestructive: Bool {
        self == .moveToTrash
    }

    var showsSeparatorAfter: Bool {
        self == .previewOriginal || self == .revealInFinder
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
    var onContextMenuAction: (AssetItem, AssetContextMenuAction) -> Void

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
        onContextMenuAction: @escaping (AssetItem, AssetContextMenuAction) -> Void = { _, _ in }
    ) {
        self.assets = assets
        self.selectedAssetIDs = selectedAssetIDs
        self.viewMode = viewMode
        self.localization = localization
        self.onSelectionChange = onSelectionChange
        self.onDoubleClick = onDoubleClick
        self.onSpacePreviewStart = onSpacePreviewStart
        self.onSpacePreviewEnd = onSpacePreviewEnd
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
        scrollView.drawsBackground = false
        scrollView.hasVerticalScroller = false
        scrollView.hasHorizontalScroller = false
        scrollView.verticalScroller = nil
        scrollView.horizontalScroller = nil
        scrollView.autohidesScrollers = true
        scrollView.scrollerStyle = .overlay
        scrollView.automaticallyAdjustsContentInsets = false
        scrollView.contentInsets = AssetCollectionMetrics.zeroEdgeInsets
        scrollView.scrollerInsets = AssetCollectionMetrics.zeroEdgeInsets
    }

    private func applyAssetChanges(to collectionView: NSCollectionView, coordinator: Coordinator) {
        let previousAssets = coordinator.currentAssets
        let deletedIndexPaths = coordinator.deletedIndexPaths(from: previousAssets, to: assets)

        coordinator.currentAssets = assets
        coordinator.rebuildAssetIndex(for: assets)
        prepareLayout(for: collectionView)

        guard let deletedIndexPaths else {
            collectionView.reloadData()
            return
        }

        collectionView.performBatchUpdates {
            collectionView.deleteItems(at: deletedIndexPaths)
        } completionHandler: { _ in
            coordinator.syncSelection()
            coordinator.syncHoveredPreviewAsset()
        }
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
    final class Coordinator: NSObject, NSCollectionViewDataSource, NSCollectionViewDelegateFlowLayout {
        var parent: AssetCollectionGridView
        weak var collectionView: NSCollectionView?
        var currentViewMode: AssetViewMode
        var currentAssets: [AssetItem]
        var currentLocalization: AppLocalization
        private var isSyncingSelection = false
        private var hoveredPreviewAssetID: AssetItem.ID?
        private var assetIndexByID: [AssetItem.ID: Int]

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
            assetItem.configure(with: asset, viewMode: parent.viewMode, localization: parent.localization)
            return assetItem
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

        func deletedIndexPaths(from oldAssets: [AssetItem], to newAssets: [AssetItem]) -> Set<IndexPath>? {
            guard oldAssets.count > newAssets.count else {
                return nil
            }

            let newIDs = Set(newAssets.map(\.id))
            let deletedIndexPaths = Set(oldAssets.enumerated().compactMap { index, asset in
                newIDs.contains(asset.id) ? nil : IndexPath(item: index, section: 0)
            })

            guard !deletedIndexPaths.isEmpty else {
                return nil
            }

            let remainingOldAssets = oldAssets.filter { newIDs.contains($0.id) }
            guard remainingOldAssets == newAssets else {
                return nil
            }

            return deletedIndexPaths
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

            parent.onContextMenuAction(parent.assets[index], action)
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

private final class AssetPreviewCollectionView: NSCollectionView {
    var onSpacePreviewStart: (() -> Void)?
    var onSpacePreviewEnd: (() -> Void)?

    override var acceptsFirstResponder: Bool {
        true
    }

    override func keyDown(with event: NSEvent) {
        if event.charactersIgnoringModifiers == " " {
            if !event.isARepeat {
                onSpacePreviewStart?()
            }
            return
        }

        super.keyDown(with: event)
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
    override func tile() {
        hideScrollIndicators()
        super.tile()
        hideScrollIndicators()
    }

    override func reflectScrolledClipView(_ clipView: NSClipView) {
        hideScrollIndicators()
        super.reflectScrolledClipView(clipView)
        hideScrollIndicators()
    }

    private func hideScrollIndicators() {
        hasVerticalScroller = false
        hasHorizontalScroller = false
        verticalScroller = nil
        horizontalScroller = nil
        contentInsets = AssetCollectionMetrics.zeroEdgeInsets
        scrollerInsets = AssetCollectionMetrics.zeroEdgeInsets
    }
}

private final class AssetGridCollectionViewLayout: NSCollectionViewLayout {
    private var cachedAttributes: [IndexPath: NSCollectionViewLayoutAttributes] = [:]
    private var cachedAttributesList: [NSCollectionViewLayoutAttributes] = []
    private var contentSize: NSSize = .zero
    private var preparedBoundsSize: NSSize = .zero
    private var preparedColumnCount = 1
    private var preparedItemCount = 0

    private let itemSize = NSSize(
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
            cachedAttributes = [:]
            cachedAttributesList = []
            contentSize = .zero
            preparedItemCount = 0
            return
        }

        let itemCount = collectionView.numberOfItems(inSection: 0)
        preparedItemCount = itemCount
        preparedBoundsSize = collectionView.bounds.size
        let contentWidth = max(collectionView.bounds.width, itemSize.width + sectionInset.left + sectionInset.right)
        let availableWidth = max(contentWidth - sectionInset.left - sectionInset.right, itemSize.width)
        let columnCount = max(Int((availableWidth + interitemSpacing) / (itemSize.width + interitemSpacing)), 1)
        preparedColumnCount = columnCount
        let columnsWidth = CGFloat(columnCount) * itemSize.width + CGFloat(columnCount - 1) * interitemSpacing
        let startX = sectionInset.left + max((availableWidth - columnsWidth) / 2, 0)
        var attributesByIndexPath: [IndexPath: NSCollectionViewLayoutAttributes] = [:]
        var attributesList: [NSCollectionViewLayoutAttributes] = []
        attributesList.reserveCapacity(itemCount)

        for item in 0..<itemCount {
            let indexPath = IndexPath(item: item, section: 0)
            let row = item / columnCount
            let column = item % columnCount
            let frame = NSRect(
                x: startX + CGFloat(column) * (itemSize.width + interitemSpacing),
                y: sectionInset.top + CGFloat(row) * (itemSize.height + lineSpacing),
                width: itemSize.width,
                height: itemSize.height
            )

            let attributes = NSCollectionViewLayoutAttributes(forItemWith: indexPath)
            attributes.frame = frame
            attributesByIndexPath[indexPath] = attributes
            attributesList.append(attributes)
        }

        cachedAttributes = attributesByIndexPath
        cachedAttributesList = attributesList

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
        guard preparedItemCount > 0, !cachedAttributesList.isEmpty else {
            return []
        }

        let rowStride = itemSize.height + lineSpacing
        let firstRow = max(Int(floor((rect.minY - sectionInset.top) / rowStride)), 0)
        let maximumRow = max((preparedItemCount - 1) / preparedColumnCount, 0)
        let lastRow = min(max(Int(floor((rect.maxY - sectionInset.top) / rowStride)), 0), maximumRow)
        let firstItem = min(firstRow * preparedColumnCount, preparedItemCount)
        let lastItem = min((lastRow + 1) * preparedColumnCount, preparedItemCount)

        guard firstItem < lastItem else {
            return []
        }

        return cachedAttributesList[firstItem..<lastItem].filter { $0.frame.intersects(rect) }
    }

    override func layoutAttributesForItem(at indexPath: IndexPath) -> NSCollectionViewLayoutAttributes? {
        cachedAttributes[indexPath]
    }

    override func shouldInvalidateLayout(forBoundsChange newBounds: NSRect) -> Bool {
        newBounds.size != preparedBoundsSize
    }
}

private final class AssetMasonryCollectionViewLayout: NSCollectionViewLayout {
    var assets: [AssetItem] {
        didSet {
            invalidateLayout()
        }
    }

    private var cachedAttributes: [IndexPath: NSCollectionViewLayoutAttributes] = [:]
    private var cachedAttributesList: [NSCollectionViewLayoutAttributes] = []
    private var contentSize: NSSize = .zero
    private var preparedBoundsSize: NSSize = .zero

    private let itemWidth = AssetCollectionMetrics.masonryItemWidth
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
            cachedAttributes = [:]
            cachedAttributesList = []
            contentSize = .zero
            return
        }

        let itemCount = collectionView.numberOfItems(inSection: 0)
        preparedBoundsSize = collectionView.bounds.size
        let contentWidth = max(collectionView.bounds.width, itemWidth + sectionInset.left + sectionInset.right)
        let availableWidth = max(contentWidth - sectionInset.left - sectionInset.right, itemWidth)
        let columnCount = max(Int((availableWidth + interitemSpacing) / (itemWidth + interitemSpacing)), 1)
        let columnsWidth = CGFloat(columnCount) * itemWidth + CGFloat(columnCount - 1) * interitemSpacing
        let startX = sectionInset.left + max((availableWidth - columnsWidth) / 2, 0)
        var columnHeights = Array(repeating: sectionInset.top, count: columnCount)
        var attributesByIndexPath: [IndexPath: NSCollectionViewLayoutAttributes] = [:]
        var attributesList: [NSCollectionViewLayoutAttributes] = []
        attributesList.reserveCapacity(itemCount)

        for item in 0..<itemCount {
            let indexPath = IndexPath(item: item, section: 0)
            let columnIndex = shortestColumnIndex(in: columnHeights)
            let itemHeight = masonryItemHeight(forItemAt: item, width: itemWidth)
            let frame = NSRect(
                x: startX + CGFloat(columnIndex) * (itemWidth + interitemSpacing),
                y: columnHeights[columnIndex],
                width: itemWidth,
                height: itemHeight
            )

            let attributes = NSCollectionViewLayoutAttributes(forItemWith: indexPath)
            attributes.frame = frame
            attributesByIndexPath[indexPath] = attributes
            attributesList.append(attributes)
            columnHeights[columnIndex] = frame.maxY + lineSpacing
        }

        cachedAttributes = attributesByIndexPath
        cachedAttributesList = attributesList

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
        guard !cachedAttributesList.isEmpty else {
            return []
        }

        let earliestPossibleMinY = rect.minY - AssetCollectionMetrics.masonryMaximumItemHeight
        var visibleAttributes: [NSCollectionViewLayoutAttributes] = []
        let startIndex = firstAttributeIndex(withMinYAtLeast: earliestPossibleMinY)

        for attributes in cachedAttributesList[startIndex...] {
            if attributes.frame.minY > rect.maxY {
                break
            }

            if attributes.frame.intersects(rect) {
                visibleAttributes.append(attributes)
            }
        }

        return visibleAttributes
    }

    override func layoutAttributesForItem(at indexPath: IndexPath) -> NSCollectionViewLayoutAttributes? {
        cachedAttributes[indexPath]
    }

    override func shouldInvalidateLayout(forBoundsChange newBounds: NSRect) -> Bool {
        newBounds.size != preparedBoundsSize
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
        var upperBound = cachedAttributesList.count

        while lowerBound < upperBound {
            let middle = (lowerBound + upperBound) / 2
            if cachedAttributesList[middle].frame.minY < minY {
                lowerBound = middle + 1
            } else {
                upperBound = middle
            }
        }

        return lowerBound
    }
}

private final class AssetPreviewImageProvider {
    static let shared = AssetPreviewImageProvider()

    private let cache = NSCache<NSString, NSImage>()

    private init() {
        cache.countLimit = 512
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

    func image(for asset: AssetItem) -> NSImage {
        let key = identity(for: asset) as NSString
        if let cachedImage = cache.object(forKey: key) {
            return cachedImage
        }

        let image = loadImage(for: asset)
        cache.setObject(image, forKey: key)
        return image
    }

    func invalidateImage(for asset: AssetItem) {
        cache.removeObject(forKey: identity(for: asset) as NSString)
    }

    private func loadImage(for asset: AssetItem) -> NSImage {
        if asset.kind == .image || asset.kind == .gif {
            if let thumbnailURL = asset.thumbnailURL,
               let image = NSImage(contentsOf: thumbnailURL) {
                return image
            }
        }

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
}

private final class AssetCollectionViewItem: NSCollectionViewItem {
    static let reuseIdentifier = NSUserInterfaceItemIdentifier("AssetCollectionViewItem")

    var onHoverPreviewChange: ((Bool) -> Void)?
    var onContextMenuOpen: (() -> Void)?
    var onContextMenuAction: ((AssetContextMenuAction) -> Void)?

    private let containerView = HoverTrackingView()
    private let contentView = HoverSelectionView()
    private let previewImageView = AssetPreviewImageView()
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
            separatorView.heightAnchor.constraint(equalToConstant: 1)
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
            dimensionBadgeView.topAnchor.constraint(equalTo: previewImageView.topAnchor, constant: 8),
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
        previewImageView.resetImage()
        fileNameLabel.stringValue = ""
        subtitleLabel.stringValue = ""
        dateLabel.stringValue = ""
        dimensionBadgeView.stringValue = ""
        dimensionBadgeView.isHidden = true
        separatorView.isHidden = true
        containerView.resetHoverState()
        onHoverPreviewChange = nil
        onContextMenuOpen = nil
        onContextMenuAction = nil
        asset = nil
        contentView.isHovered = false
        contentView.isSelected = false
        contentView.viewMode = .grid
        dateLabel.isHidden = true
        mode = .grid
        updateTextColors()
    }

    func configure(with asset: AssetItem, viewMode: AssetViewMode, localization: AppLocalization) {
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
        previewImageView.setImage(
            previewProvider.image(for: asset),
            identity: "\(viewMode.rawValue):\(previewIdentity)",
            animated: true
        )
        previewImageView.contentMode = imageContentMode(for: asset, viewMode: viewMode)
        applyModeLayout()
        containerView.synchronizeHoverStateWithPointer()
    }

    @objc private func handleRightClick(_ sender: NSClickGestureRecognizer) {
        showContextMenu(at: sender.location(in: containerView))
    }

    private func showContextMenu(at location: NSPoint) {
        guard asset != nil else {
            return
        }

        onContextMenuOpen?()
        let menu = NSMenu()
        menu.showsStateColumn = false
        let menuItem = NSMenuItem()
        let menuView = AssetContextMenuView(
            localization: localization,
            actions: AssetContextMenuAction.allCases,
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
        return .aspectFit
    }

    func previewSourceFrameInScreen() -> NSRect? {
        guard let window = previewImageView.window else {
            return nil
        }

        let rectInWindow = previewImageView.convert(previewImageView.visibleImageBounds, to: nil)
        return window.convertToScreen(rectInWindow)
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

private final class AssetContextMenuBackgroundView: NSView {
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
        layer?.backgroundColor = NSColor.windowBackgroundColor.withAlphaComponent(0.82).cgColor
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
