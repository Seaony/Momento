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
    static let hoverScale: CGFloat = 1.025
    static let selectionCornerRadius: CGFloat = 12
    static let imageCornerRadius: CGFloat = 10
    static let dimensionBadgeHeight: CGFloat = 16
    static let dimensionBadgeHorizontalPadding: CGFloat = 3
}

struct AssetCollectionGridView: NSViewRepresentable {
    var assets: [AssetItem]
    var selectedAssetIDs: Set<AssetItem.ID>
    var viewMode: AssetViewMode
    var onSelectionChange: (Set<AssetItem.ID>) -> Void
    var onDoubleClick: (AssetItem) -> Void
    var onSpacePreviewStart: (AssetItem, NSRect?) -> Void
    var onSpacePreviewEnd: () -> Void

    init(
        assets: [AssetItem],
        selectedAssetIDs: Set<AssetItem.ID> = [],
        viewMode: AssetViewMode = .grid,
        onSelectionChange: @escaping (Set<AssetItem.ID>) -> Void = { _ in },
        onDoubleClick: @escaping (AssetItem) -> Void = { _ in },
        onSpacePreviewStart: @escaping (AssetItem, NSRect?) -> Void = { _, _ in },
        onSpacePreviewEnd: @escaping () -> Void = {}
    ) {
        self.assets = assets
        self.selectedAssetIDs = selectedAssetIDs
        self.viewMode = viewMode
        self.onSelectionChange = onSelectionChange
        self.onDoubleClick = onDoubleClick
        self.onSpacePreviewStart = onSpacePreviewStart
        self.onSpacePreviewEnd = onSpacePreviewEnd
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
        collectionView.addGestureRecognizer(doubleClickRecognizer)

        let scrollView = NSScrollView()
        scrollView.drawsBackground = false
        scrollView.hasVerticalScroller = true
        scrollView.hasHorizontalScroller = false
        scrollView.autohidesScrollers = true
        scrollView.documentView = collectionView

        context.coordinator.collectionView = collectionView
        return scrollView
    }

    func updateNSView(_ scrollView: NSScrollView, context: Context) {
        context.coordinator.parent = self

        guard let collectionView = scrollView.documentView as? NSCollectionView else {
            return
        }

        if context.coordinator.currentViewMode != viewMode {
            collectionView.collectionViewLayout = makeLayout(for: viewMode, assets: assets)
            context.coordinator.currentViewMode = viewMode
            context.coordinator.currentAssets = assets
            collectionView.reloadData()
            context.coordinator.syncSelection()
            return
        }

        if context.coordinator.currentAssets != assets {
            context.coordinator.currentAssets = assets
            if let masonryLayout = collectionView.collectionViewLayout as? AssetMasonryCollectionViewLayout {
                masonryLayout.assets = assets
            }
            collectionView.reloadData()
        }

        context.coordinator.syncSelection()
    }

    private func makeLayout(for viewMode: AssetViewMode, assets: [AssetItem]) -> NSCollectionViewLayout {
        if viewMode == .masonry {
            return AssetMasonryCollectionViewLayout(assets: assets)
        }

        let layout = NSCollectionViewFlowLayout()
        layout.sectionInset = NSEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)

        switch viewMode {
        case .grid:
            layout.itemSize = NSSize(width: 148, height: 178)
            layout.minimumInteritemSpacing = 14
            layout.minimumLineSpacing = 18
        case .masonry:
            break
        case .list:
            layout.itemSize = NSSize(width: 320, height: 54)
            layout.minimumInteritemSpacing = 0
            layout.minimumLineSpacing = 1
        }

        return layout
    }
}

extension AssetCollectionGridView {
    final class Coordinator: NSObject, NSCollectionViewDataSource, NSCollectionViewDelegateFlowLayout {
        var parent: AssetCollectionGridView
        weak var collectionView: NSCollectionView?
        var currentViewMode: AssetViewMode
        var currentAssets: [AssetItem]
        private var isSyncingSelection = false

        init(_ parent: AssetCollectionGridView) {
            self.parent = parent
            self.currentViewMode = parent.viewMode
            self.currentAssets = parent.assets
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

            assetItem.configure(with: parent.assets[indexPath.item], viewMode: parent.viewMode)
            assetItem.onHoverFocus = { [weak self, weak collectionView] in
                guard let collectionView else {
                    return
                }

                self?.focusAsset(at: indexPath, in: collectionView)
            }
            return assetItem
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            layout collectionViewLayout: NSCollectionViewLayout,
            sizeForItemAt indexPath: IndexPath
        ) -> NSSize {
            switch parent.viewMode {
            case .grid:
                return NSSize(width: 148, height: 178)
            case .masonry:
                return .zero
            case .list:
                let width = max(collectionView.enclosingScrollView?.contentSize.width ?? 320, 240)
                return NSSize(width: width, height: 54)
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
                parent.assets.enumerated().compactMap { index, asset in
                    parent.selectedAssetIDs.contains(asset.id) ? IndexPath(item: index, section: 0) : nil
                }
            )
            isSyncingSelection = false
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
                  let indexPath = collectionView.selectionIndexPaths.sorted(by: { $0.item < $1.item }).first,
                  parent.assets.indices.contains(indexPath.item) else {
                return
            }

            let sourceFrame = sourceFrameForPreview(at: indexPath, in: collectionView)
            parent.onSpacePreviewStart(parent.assets[indexPath.item], sourceFrame)
        }

        func endSpacePreview() {
            parent.onSpacePreviewEnd()
        }

        private func focusAsset(at indexPath: IndexPath, in collectionView: NSCollectionView) {
            guard parent.assets.indices.contains(indexPath.item) else {
                return
            }

            collectionView.window?.makeFirstResponder(collectionView)
            let assetID = parent.assets[indexPath.item].id

            let targetSelection: Set<IndexPath> = [indexPath]
            guard collectionView.selectionIndexPaths != targetSelection else {
                return
            }

            isSyncingSelection = true
            collectionView.selectionIndexPaths = targetSelection
            isSyncingSelection = false
            parent.onSelectionChange([assetID])
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

private final class AssetMasonryCollectionViewLayout: NSCollectionViewLayout {
    var assets: [AssetItem] {
        didSet {
            invalidateLayout()
        }
    }

    private var cachedAttributes: [IndexPath: NSCollectionViewLayoutAttributes] = [:]
    private var contentSize: NSSize = .zero
    private var preparedBoundsSize: NSSize = .zero

    private let itemWidth = AssetCollectionMetrics.masonryItemWidth
    private let interitemSpacing: CGFloat = 4
    private let lineSpacing: CGFloat = 4
    private let sectionInset = NSEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)

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
            columnHeights[columnIndex] = frame.maxY + lineSpacing
        }

        cachedAttributes = attributesByIndexPath

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
        cachedAttributes.values.filter { $0.frame.intersects(rect) }
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
            return 214
        }

        let imageHorizontalInset = AssetCollectionMetrics.masonryImageInset * 2
        let imageVerticalInset = AssetCollectionMetrics.masonryImageInset * 2
        let imageWidth = max(width - imageHorizontalInset, 1)
        let ratio = CGFloat(dimensions.height) / CGFloat(dimensions.width)
        return (imageWidth * ratio + imageVerticalInset).clamped(to: 132...278)
    }

    private func shortestColumnIndex(in columnHeights: [CGFloat]) -> Int {
        columnHeights.enumerated().min { left, right in
            left.element < right.element
        }?.offset ?? 0
    }
}

private final class AssetCollectionViewItem: NSCollectionViewItem {
    static let reuseIdentifier = NSUserInterfaceItemIdentifier("AssetCollectionViewItem")

    var onHoverFocus: (() -> Void)?

    private let containerView = HoverSelectionView()
    private let previewImageView = NSImageView()
    private let fileNameLabel = NSTextField(labelWithString: "")
    private let subtitleLabel = NSTextField(labelWithString: "")
    private let dimensionBadgeView = DimensionBadgeView()
    private var gridConstraints: [NSLayoutConstraint] = []
    private var masonryConstraints: [NSLayoutConstraint] = []
    private var listConstraints: [NSLayoutConstraint] = []
    private var mode: AssetViewMode = .grid
    private var isHovering = false
    private let gridTitleHeight: CGFloat = 16
    private let gridSubtitleHeight: CGFloat = 14

    override var isSelected: Bool {
        didSet {
            containerView.isSelected = isSelected
            updateContainerScale(animated: true)
        }
    }

    override func loadView() {
        containerView.wantsLayer = true
        containerView.layer?.cornerRadius = AssetCollectionMetrics.selectionCornerRadius
        containerView.layer?.cornerCurve = .continuous
        containerView.layer?.masksToBounds = false
        containerView.layer?.anchorPoint = CGPoint(x: 0.5, y: 0.5)
        containerView.translatesAutoresizingMaskIntoConstraints = false
        containerView.hoverChanged = { [weak self] isHovered in
            self?.isHovering = isHovered
            self?.containerView.isHovered = isHovered
            self?.updateContainerScale(animated: true)
            if isHovered {
                self?.onHoverFocus?()
            }
        }

        previewImageView.imageScaling = .scaleProportionallyUpOrDown
        previewImageView.translatesAutoresizingMaskIntoConstraints = false
        previewImageView.wantsLayer = true
        previewImageView.layer?.cornerRadius = AssetCollectionMetrics.imageCornerRadius
        previewImageView.layer?.cornerCurve = .continuous
        previewImageView.layer?.masksToBounds = true

        fileNameLabel.lineBreakMode = .byTruncatingTail
        fileNameLabel.maximumNumberOfLines = 1
        fileNameLabel.alignment = .center
        fileNameLabel.font = .systemFont(ofSize: 12, weight: .regular)
        fileNameLabel.textColor = .tertiaryLabelColor
        fileNameLabel.translatesAutoresizingMaskIntoConstraints = false
        fileNameLabel.setContentCompressionResistancePriority(.required, for: .vertical)

        subtitleLabel.lineBreakMode = .byTruncatingTail
        subtitleLabel.maximumNumberOfLines = 1
        subtitleLabel.alignment = .center
        subtitleLabel.font = .systemFont(ofSize: 11)
        subtitleLabel.textColor = .quaternaryLabelColor
        subtitleLabel.translatesAutoresizingMaskIntoConstraints = false
        subtitleLabel.setContentCompressionResistancePriority(.required, for: .vertical)

        dimensionBadgeView.translatesAutoresizingMaskIntoConstraints = false
        dimensionBadgeView.isHidden = true

        containerView.addSubview(previewImageView)
        containerView.addSubview(fileNameLabel)
        containerView.addSubview(subtitleLabel)
        containerView.addSubview(dimensionBadgeView)
        view = containerView

        gridConstraints = [
            previewImageView.topAnchor.constraint(equalTo: containerView.topAnchor, constant: 10),
            previewImageView.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 10),
            previewImageView.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -10),
            previewImageView.bottomAnchor.constraint(equalTo: fileNameLabel.topAnchor, constant: -8),
            previewImageView.heightAnchor.constraint(greaterThanOrEqualToConstant: 96),

            fileNameLabel.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 8),
            fileNameLabel.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -8),
            fileNameLabel.heightAnchor.constraint(equalToConstant: gridTitleHeight),
            fileNameLabel.bottomAnchor.constraint(equalTo: subtitleLabel.topAnchor, constant: -2),

            subtitleLabel.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 8),
            subtitleLabel.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -8),
            subtitleLabel.heightAnchor.constraint(equalToConstant: gridSubtitleHeight),
            subtitleLabel.bottomAnchor.constraint(equalTo: containerView.bottomAnchor, constant: -8)
        ]

        masonryConstraints = [
            previewImageView.topAnchor.constraint(equalTo: containerView.topAnchor, constant: AssetCollectionMetrics.masonryImageInset),
            previewImageView.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: AssetCollectionMetrics.masonryImageInset),
            previewImageView.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -AssetCollectionMetrics.masonryImageInset),
            previewImageView.bottomAnchor.constraint(equalTo: containerView.bottomAnchor, constant: -AssetCollectionMetrics.masonryImageInset),
            dimensionBadgeView.topAnchor.constraint(equalTo: previewImageView.topAnchor, constant: 8),
            dimensionBadgeView.trailingAnchor.constraint(equalTo: previewImageView.trailingAnchor, constant: -8),
            dimensionBadgeView.heightAnchor.constraint(equalToConstant: AssetCollectionMetrics.dimensionBadgeHeight)
        ]

        listConstraints = [
            previewImageView.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 10),
            previewImageView.centerYAnchor.constraint(equalTo: containerView.centerYAnchor),
            previewImageView.widthAnchor.constraint(equalToConstant: 36),
            previewImageView.heightAnchor.constraint(equalToConstant: 36),

            fileNameLabel.leadingAnchor.constraint(equalTo: previewImageView.trailingAnchor, constant: 10),
            fileNameLabel.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -12),
            fileNameLabel.topAnchor.constraint(equalTo: containerView.topAnchor, constant: 9),

            subtitleLabel.leadingAnchor.constraint(equalTo: fileNameLabel.leadingAnchor),
            subtitleLabel.trailingAnchor.constraint(equalTo: fileNameLabel.trailingAnchor),
            subtitleLabel.topAnchor.constraint(equalTo: fileNameLabel.bottomAnchor, constant: 2),
            subtitleLabel.bottomAnchor.constraint(lessThanOrEqualTo: containerView.bottomAnchor, constant: -8)
        ]
    }

    override func prepareForReuse() {
        super.prepareForReuse()
        previewImageView.image = nil
        fileNameLabel.stringValue = ""
        subtitleLabel.stringValue = ""
        dimensionBadgeView.stringValue = ""
        dimensionBadgeView.isHidden = true
        onHoverFocus = nil
        isHovering = false
        containerView.isHovered = false
        containerView.isSelected = false
        updateContainerScale(animated: false)
        mode = .grid
    }

    func configure(with asset: AssetItem, viewMode: AssetViewMode) {
        mode = viewMode
        fileNameLabel.stringValue = asset.displayName
        subtitleLabel.stringValue = subtitle(for: asset)
        dimensionBadgeView.stringValue = subtitle(for: asset)
        previewImageView.image = previewImage(for: asset)
        applyModeLayout()
        updateContainerScale(animated: false)
    }

    private func applyModeLayout() {
        NSLayoutConstraint.deactivate(gridConstraints + masonryConstraints + listConstraints)

        switch mode {
        case .grid:
            fileNameLabel.isHidden = false
            subtitleLabel.isHidden = false
            dimensionBadgeView.isHidden = true
            fileNameLabel.alignment = .center
            fileNameLabel.maximumNumberOfLines = 1
            subtitleLabel.alignment = .center
            NSLayoutConstraint.activate(gridConstraints)
        case .masonry:
            fileNameLabel.isHidden = true
            subtitleLabel.isHidden = true
            dimensionBadgeView.isHidden = dimensionBadgeView.stringValue.isEmpty
            NSLayoutConstraint.activate(masonryConstraints)
        case .list:
            fileNameLabel.isHidden = false
            subtitleLabel.isHidden = false
            dimensionBadgeView.isHidden = true
            fileNameLabel.alignment = .left
            fileNameLabel.maximumNumberOfLines = 1
            subtitleLabel.alignment = .left
            NSLayoutConstraint.activate(listConstraints)
        }
    }

    private func subtitle(for asset: AssetItem) -> String {
        if let dimensions = asset.dimensions {
            return "\(dimensions.width) x \(dimensions.height)"
        }

        return ""
    }

    private func previewImage(for asset: AssetItem) -> NSImage {
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

    func previewSourceFrameInScreen() -> NSRect? {
        guard let window = containerView.window else {
            return nil
        }

        let rectInWindow = containerView.convert(containerView.bounds, to: nil)
        return window.convertToScreen(rectInWindow)
    }

    private func updateContainerScale(animated: Bool) {
        guard let layer = containerView.layer else {
            return
        }

        let scale = mode == .masonry && (isHovering || isSelected) ? AssetCollectionMetrics.hoverScale : 1
        let targetTransform = CATransform3DMakeScale(scale, scale, 1)
        let currentTransform = layer.presentation()?.transform ?? layer.transform
        layer.zPosition = scale > 1 ? 8 : 0

        guard animated else {
            layer.removeAnimation(forKey: "hoverScale")
            layer.transform = targetTransform
            return
        }

        layer.transform = targetTransform

        let animation = CABasicAnimation(keyPath: "transform")
        animation.fromValue = currentTransform
        animation.toValue = targetTransform
        animation.duration = 0.16
        animation.timingFunction = CAMediaTimingFunction(name: .easeOut)
        layer.add(animation, forKey: "hoverScale")
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
        layer?.cornerRadius = 7
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

private final class HoverSelectionView: NSView {
    var hoverChanged: ((Bool) -> Void)?
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

    private let glassBackgroundView = NSGlassEffectView()
    private var trackingArea: NSTrackingArea?

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

    override func updateTrackingAreas() {
        super.updateTrackingAreas()

        if let trackingArea {
            removeTrackingArea(trackingArea)
        }

        let options: NSTrackingArea.Options = [.activeInActiveApp, .mouseEnteredAndExited, .inVisibleRect]
        let newTrackingArea = NSTrackingArea(rect: bounds, options: options, owner: self)
        addTrackingArea(newTrackingArea)
        trackingArea = newTrackingArea
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }

    override func mouseEntered(with event: NSEvent) {
        hoverChanged?(true)
    }

    override func mouseExited(with event: NSEvent) {
        hoverChanged?(false)
    }

    private func updateAppearance() {
        wantsLayer = true
        glassBackgroundView.isHidden = true
        layer?.borderColor = NSColor.clear.cgColor
        layer?.borderWidth = 0

        if isSelected {
            layer?.backgroundColor = NSColor.white.withAlphaComponent(0.14).cgColor
        } else if isHovered {
            layer?.backgroundColor = NSColor.white.withAlphaComponent(0.08).cgColor
        } else {
            layer?.backgroundColor = NSColor.clear.cgColor
        }
    }
}

private extension CGFloat {
    func clamped(to range: ClosedRange<CGFloat>) -> CGFloat {
        Swift.min(Swift.max(self, range.lowerBound), range.upperBound)
    }
}
