//
//  AssetCollectionGridView.swift
//  Momento
//

import AppKit
import SwiftUI

struct AssetCollectionGridView: NSViewRepresentable {
    var assets: [AssetItem]
    var selectedAssetIDs: Set<AssetItem.ID>
    var viewMode: AssetViewMode
    var onSelectionChange: (Set<AssetItem.ID>) -> Void
    var onDoubleClick: (AssetItem) -> Void

    init(
        assets: [AssetItem],
        selectedAssetIDs: Set<AssetItem.ID> = [],
        viewMode: AssetViewMode = .grid,
        onSelectionChange: @escaping (Set<AssetItem.ID>) -> Void = { _ in },
        onDoubleClick: @escaping (AssetItem) -> Void = { _ in }
    ) {
        self.assets = assets
        self.selectedAssetIDs = selectedAssetIDs
        self.viewMode = viewMode
        self.onSelectionChange = onSelectionChange
        self.onDoubleClick = onDoubleClick
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    func makeNSView(context: Context) -> NSScrollView {
        let collectionView = NSCollectionView()
        collectionView.collectionViewLayout = makeLayout(for: viewMode)
        collectionView.backgroundColors = [.clear]
        collectionView.isSelectable = true
        collectionView.allowsMultipleSelection = true
        collectionView.dataSource = context.coordinator
        collectionView.delegate = context.coordinator
        collectionView.register(
            AssetCollectionViewItem.self,
            forItemWithIdentifier: AssetCollectionViewItem.reuseIdentifier
        )

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
            collectionView.collectionViewLayout = makeLayout(for: viewMode)
            context.coordinator.currentViewMode = viewMode
        }

        collectionView.reloadData()
        context.coordinator.syncSelection()
    }

    private func makeLayout(for viewMode: AssetViewMode) -> NSCollectionViewFlowLayout {
        let layout = NSCollectionViewFlowLayout()
        layout.sectionInset = NSEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)

        switch viewMode {
        case .grid:
            layout.itemSize = NSSize(width: 148, height: 178)
            layout.minimumInteritemSpacing = 14
            layout.minimumLineSpacing = 18
        case .masonry:
            layout.itemSize = NSSize(width: 164, height: 220)
            layout.minimumInteritemSpacing = 14
            layout.minimumLineSpacing = 16
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
        private var isSyncingSelection = false

        init(_ parent: AssetCollectionGridView) {
            self.parent = parent
            self.currentViewMode = parent.viewMode
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
                let extraHeight = CGFloat(abs(parent.assets[indexPath.item].displayName.hashValue) % 48)
                return NSSize(width: 164, height: 196 + extraHeight)
            case .list:
                let width = max(collectionView.enclosingScrollView?.contentSize.width ?? 320, 240)
                return NSSize(width: width, height: 54)
            }
        }

        func collectionView(
            _ collectionView: NSCollectionView,
            didSelectItemsAt indexPaths: Set<IndexPath>
        ) {
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

        private func publishSelection(from collectionView: NSCollectionView) {
            guard !isSyncingSelection else {
                return
            }

            let selectedIDs = Set(collectionView.selectionIndexPaths.compactMap { indexPath in
                parent.assets.indices.contains(indexPath.item) ? parent.assets[indexPath.item].id : nil
            })

            parent.onSelectionChange(selectedIDs)
        }
    }
}

private final class AssetCollectionViewItem: NSCollectionViewItem {
    static let reuseIdentifier = NSUserInterfaceItemIdentifier("AssetCollectionViewItem")

    private let containerView = HoverSelectionView()
    private let previewImageView = NSImageView()
    private let fileNameLabel = NSTextField(labelWithString: "")
    private let subtitleLabel = NSTextField(labelWithString: "")
    private var gridConstraints: [NSLayoutConstraint] = []
    private var listConstraints: [NSLayoutConstraint] = []
    private var mode: AssetViewMode = .grid

    override var isSelected: Bool {
        didSet {
            containerView.isSelected = isSelected
        }
    }

    override func loadView() {
        containerView.wantsLayer = true
        containerView.layer?.cornerRadius = 8
        containerView.layer?.cornerCurve = .continuous
        containerView.translatesAutoresizingMaskIntoConstraints = false
        containerView.hoverChanged = { [weak self] isHovered in
            self?.containerView.isHovered = isHovered
        }

        previewImageView.imageScaling = .scaleProportionallyUpOrDown
        previewImageView.translatesAutoresizingMaskIntoConstraints = false
        previewImageView.wantsLayer = true
        previewImageView.layer?.cornerRadius = 6
        previewImageView.layer?.cornerCurve = .continuous
        previewImageView.layer?.masksToBounds = true

        fileNameLabel.lineBreakMode = .byTruncatingMiddle
        fileNameLabel.maximumNumberOfLines = 2
        fileNameLabel.alignment = .center
        fileNameLabel.font = .systemFont(ofSize: 12)
        fileNameLabel.textColor = .secondaryLabelColor
        fileNameLabel.translatesAutoresizingMaskIntoConstraints = false

        subtitleLabel.lineBreakMode = .byTruncatingTail
        subtitleLabel.maximumNumberOfLines = 1
        subtitleLabel.alignment = .center
        subtitleLabel.font = .systemFont(ofSize: 11)
        subtitleLabel.textColor = .tertiaryLabelColor
        subtitleLabel.translatesAutoresizingMaskIntoConstraints = false

        containerView.addSubview(previewImageView)
        containerView.addSubview(fileNameLabel)
        containerView.addSubview(subtitleLabel)
        view = containerView

        gridConstraints = [
            previewImageView.topAnchor.constraint(equalTo: containerView.topAnchor, constant: 10),
            previewImageView.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 10),
            previewImageView.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -10),
            previewImageView.bottomAnchor.constraint(equalTo: fileNameLabel.topAnchor, constant: -8),
            previewImageView.heightAnchor.constraint(greaterThanOrEqualToConstant: 96),

            fileNameLabel.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 8),
            fileNameLabel.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -8),
            fileNameLabel.bottomAnchor.constraint(equalTo: subtitleLabel.topAnchor, constant: -2),

            subtitleLabel.leadingAnchor.constraint(equalTo: containerView.leadingAnchor, constant: 8),
            subtitleLabel.trailingAnchor.constraint(equalTo: containerView.trailingAnchor, constant: -8),
            subtitleLabel.bottomAnchor.constraint(equalTo: containerView.bottomAnchor, constant: -8)
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
        containerView.isHovered = false
        mode = .grid
    }

    func configure(with asset: AssetItem, viewMode: AssetViewMode) {
        mode = viewMode
        fileNameLabel.stringValue = asset.displayName
        subtitleLabel.stringValue = subtitle(for: asset)
        previewImageView.image = previewImage(for: asset)
        applyModeLayout()
    }

    private func applyModeLayout() {
        NSLayoutConstraint.deactivate(gridConstraints + listConstraints)

        switch mode {
        case .grid, .masonry:
            fileNameLabel.alignment = .center
            fileNameLabel.maximumNumberOfLines = 2
            subtitleLabel.alignment = .center
            NSLayoutConstraint.activate(gridConstraints)
        case .list:
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
            if let image = NSImage(contentsOf: asset.storageURL) ?? NSImage(contentsOf: asset.originalURL) {
                return image
            }
        }

        let previewURL = FileManager.default.fileExists(atPath: asset.storageURL.path)
            ? asset.storageURL
            : asset.originalURL

        return NSWorkspace.shared.icon(forFile: previewURL.path)
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

    private var trackingArea: NSTrackingArea?

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

    override func mouseEntered(with event: NSEvent) {
        hoverChanged?(true)
    }

    override func mouseExited(with event: NSEvent) {
        hoverChanged?(false)
    }

    private func updateAppearance() {
        wantsLayer = true

        if isSelected {
            layer?.backgroundColor = NSColor.controlAccentColor.withAlphaComponent(0.22).cgColor
            layer?.borderColor = NSColor.controlAccentColor.withAlphaComponent(0.72).cgColor
            layer?.borderWidth = 1
        } else if isHovered {
            layer?.backgroundColor = NSColor.controlAccentColor.withAlphaComponent(0.08).cgColor
            layer?.borderColor = NSColor.separatorColor.withAlphaComponent(0.35).cgColor
            layer?.borderWidth = 1
        } else {
            layer?.backgroundColor = NSColor.clear.cgColor
            layer?.borderColor = NSColor.clear.cgColor
            layer?.borderWidth = 0
        }
    }
}
