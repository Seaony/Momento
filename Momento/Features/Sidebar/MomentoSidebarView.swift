// 中文注释：侧边栏展示资源库导航、文件夹树和底部操作入口，并承接素材拖拽归类。
import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct MomentoSidebarAssetCounts: Equatable {
    var all: Int
    var favorites: Int
    var uncategorized: Int
    var untagged: Int
    var trash: Int
    var folders: [AssetFolder.ID: Int]

    static let empty = MomentoSidebarAssetCounts(
        all: 0,
        favorites: 0,
        uncategorized: 0,
        untagged: 0,
        trash: 0,
        folders: [:]
    )
}

private enum MomentoSidebarMenuMetrics {
    static let separatorAlpha: Double = 0.1
    static let separatorHeight: CGFloat = 1
    static let separatorHorizontalInset: CGFloat = 10
    static let separatorVerticalPadding: CGFloat = 3
    static let folderDisclosureHitWidth: CGFloat = 24
    static let folderDisclosureHitHeight: CGFloat = 30
    static let libraryMoreMenuWidth: CGFloat = 176
}

struct MomentoSidebarView: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.openSettings) private var openSettings

    @Binding var selection: String?
    var libraryName: String?
    var currentLibraryID: RecentLibraryReference.ID?
    var recentLibraries: [RecentLibraryReference]
    var folders: [AssetFolder]
    var counts: MomentoSidebarAssetCounts
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onImportLibrary: () -> Void
    var onExportLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onRenameLibrary: (RecentLibraryReference.ID) -> Void
    var onDeleteLibrary: (RecentLibraryReference.ID) -> Void
    var onRevealLibrary: (RecentLibraryReference.ID) -> Void
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void
    var onReloadLibrary: () -> Void
    var onCloseLibrary: () -> Void
    var onCreateFolder: (AssetFolder.ID?) -> Void
    var onRenameFolder: (AssetFolder.ID) -> Void
    var onDeleteFolder: (AssetFolder.ID) -> Void
    var onAssignDroppedAssetsToFolder: (Set<AssetItem.ID>, AssetFolder.ID) -> Void

    @State private var hoveredFooterActionID: String?
    @State private var hoveredNavigationItemID: String?
    @State private var hoveredFolderControlID: String?
    @State private var hoveredFolderID: AssetFolder.ID?
    @State private var targetedAssetDropID: String?
    @State private var isFolderSectionHovered = false
    @State private var isFolderSectionExpanded = true
    @State private var expandedFolderIDs: Set<AssetFolder.ID> = []
    @State private var isLibraryMenuHovered = false
    @State private var isLibrarySwitcherPresented = false

    init(
        selection: Binding<String?>,
        libraryName: String? = nil,
        currentLibraryID: RecentLibraryReference.ID? = nil,
        recentLibraries: [RecentLibraryReference] = [],
        folders: [AssetFolder] = [],
        counts: MomentoSidebarAssetCounts = .empty,
        onCreateLibrary: @escaping () -> Void = {},
        onOpenLibrary: @escaping () -> Void = {},
        onImportLibrary: @escaping () -> Void = {},
        onExportLibrary: @escaping () -> Void = {},
        onSwitchLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onRenameLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onDeleteLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onRevealLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onMoveLibrary: @escaping (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void = { _, _, _ in },
        onReloadLibrary: @escaping () -> Void = {},
        onCloseLibrary: @escaping () -> Void = {},
        onCreateFolder: @escaping (AssetFolder.ID?) -> Void = { _ in },
        onRenameFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        onDeleteFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        onAssignDroppedAssetsToFolder: @escaping (Set<AssetItem.ID>, AssetFolder.ID) -> Void = { _, _ in }
    ) {
        self._selection = selection
        self.libraryName = libraryName
        self.currentLibraryID = currentLibraryID
        self.recentLibraries = recentLibraries
        self.folders = folders
        self.counts = counts
        self.onCreateLibrary = onCreateLibrary
        self.onOpenLibrary = onOpenLibrary
        self.onImportLibrary = onImportLibrary
        self.onExportLibrary = onExportLibrary
        self.onSwitchLibrary = onSwitchLibrary
        self.onRenameLibrary = onRenameLibrary
        self.onDeleteLibrary = onDeleteLibrary
        self.onRevealLibrary = onRevealLibrary
        self.onMoveLibrary = onMoveLibrary
        self.onReloadLibrary = onReloadLibrary
        self.onCloseLibrary = onCloseLibrary
        self.onCreateFolder = onCreateFolder
        self.onRenameFolder = onRenameFolder
        self.onDeleteFolder = onDeleteFolder
        self.onAssignDroppedAssetsToFolder = onAssignDroppedAssetsToFolder
    }

    var body: some View {
        sidebarPanel
        .frame(
            minWidth: MomentoTheme.sidebarMinWidth,
            idealWidth: MomentoTheme.sidebarWidth,
            maxWidth: MomentoTheme.sidebarMaxWidth
        )
        .frame(maxHeight: .infinity)
        .overlay(alignment: .topLeading) {
            librarySwitcherOverlay
        }
        .zIndex(isLibrarySwitcherPresented ? 20 : 0)
    }

    private var sidebarPanel: some View {
        VStack(spacing: 0) {
            libraryMenu
                .padding(.horizontal, 14)
                .padding(.top, MomentoTheme.floatingSidebarTitlebarContentInset)
                .padding(.bottom, 18)

            ScrollView {
                sidebarNavigation
                    .padding(.horizontal, 14)
                    .padding(.bottom, 18)
            }
            .scrollIndicators(.never)

            sidebarBottomSeparator
            bottomActionBar
        }
        .frame(maxHeight: .infinity)
        .background {
            MomentoGlassBackground(cornerRadius: MomentoTheme.floatingSidebarRadius)
        }
        .clipShape(sidebarShape)
        .overlay {
            sidebarShape.strokeBorder(MomentoTheme.subtleStroke.opacity(0.42), lineWidth: 1)
        }
    }

    @ViewBuilder
    private var librarySwitcherOverlay: some View {
        if isLibrarySwitcherPresented {
            MomentoLibrarySwitcherMenu(
                recentLibraries: recentLibraries,
                currentLibraryID: currentLibraryID,
                onCreateLibrary: performLibrarySwitcherAction(onCreateLibrary),
                onOpenLibrary: performLibrarySwitcherAction(onOpenLibrary),
                onImportLibrary: performLibrarySwitcherAction(onImportLibrary),
                onExportLibrary: performLibrarySwitcherAction(onExportLibrary),
                onSwitchLibrary: { id in
                    dismissLibrarySwitcher()
                    onSwitchLibrary(id)
                },
                onRenameLibrary: { id in
                    dismissLibrarySwitcher()
                    onRenameLibrary(id)
                },
                onDeleteLibrary: { id in
                    dismissLibrarySwitcher()
                    onDeleteLibrary(id)
                },
                onRevealLibrary: { id in
                    dismissLibrarySwitcher()
                    onRevealLibrary(id)
                },
                onMoveLibrary: onMoveLibrary,
                onReloadLibrary: performLibrarySwitcherAction(onReloadLibrary)
            )
            .background {
                LibrarySwitcherDismissMonitor(isPresented: $isLibrarySwitcherPresented)
            }
            .padding(.top, MomentoTheme.floatingSidebarTitlebarContentInset + MomentoTheme.librarySelectorHeight + MomentoTheme.librarySwitcherVerticalGap)
            .padding(.leading, 14)
            .transition(.opacity.combined(with: .scale(scale: 0.98, anchor: .topLeading)))
            .zIndex(30)
        }
    }

    private var sidebarShape: RoundedRectangle {
        RoundedRectangle(cornerRadius: MomentoTheme.floatingSidebarRadius, style: .continuous)
    }

    private var sidebarBottomSeparator: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(1))
            .frame(height: 0.5)
            .padding(.horizontal, 14)
    }

    private var sidebarNavigation: some View {
        VStack(alignment: .leading, spacing: 6) {
            VStack(spacing: 2) {
                sidebarNavigationItem(
                    id: "all-assets",
                    title: localization.string("All"),
                    systemImage: "square.3.layers.3d",
                    count: counts.all
                )

                sidebarNavigationDivider

                sidebarNavigationItem(
                    id: "favorites",
                    title: localization.string("Favorited"),
                    systemImage: "heart",
                    count: counts.favorites
                )

                sidebarNavigationItem(
                    id: "uncategorized",
                    title: localization.string("Uncategorized"),
                    systemImage: "folder.badge.questionmark",
                    count: counts.uncategorized
                )

                sidebarNavigationItem(
                    id: "untagged",
                    title: localization.string("Untagged"),
                    systemImage: "xmark.triangle.circle.square",
                    count: counts.untagged
                )

                sidebarNavigationItem(
                    id: "tag-management",
                    title: localization.string("Tag Management"),
                    systemImage: "number"
                )
            }

            sidebarNavigationDivider

            sidebarFolderSection
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var sidebarNavigationDivider: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(0.72))
            .frame(height: 0.5)
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
    }

    private func sidebarNavigationItem(
        id: String,
        title: String,
        systemImage: String,
        count: Int? = nil
    ) -> some View {
        let isSelected = selection == id
        let isHovered = hoveredNavigationItemID == id
        let foregroundStyle = sidebarNavigationForeground(isSelected: isSelected, isHovered: isHovered)

        return Button {
            withAnimation(.smooth(duration: 0.16)) {
                selection = id
            }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: systemImage)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(foregroundStyle)
                    .frame(width: 18)

                Text(title)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(foregroundStyle)
                    .lineLimit(1)
                    .truncationMode(.tail)

                Spacer(minLength: 0)

                countBadge(count, foregroundStyle: foregroundStyle)
            }
            .padding(.horizontal, 8)
            .frame(height: 30)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background {
                sidebarNavigationItemBackground(id: id, isSelected: isSelected)
            }
            .contentShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateNavigationHover(id: id, isHovering: hovering)
        }
        .help(title)
        .accessibilityLabel(title)
    }

    private func sidebarNavigationForeground(isSelected: Bool, isHovered: Bool) -> Color {
        if isSelected {
            return .white
        }

        if isHovered {
            return MomentoTheme.primaryText
        }

        return MomentoTheme.primaryText.opacity(0.72)
    }

    @ViewBuilder
    private func sidebarNavigationItemBackground(id: String, isSelected: Bool) -> some View {
        let shape = RoundedRectangle(cornerRadius: 9, style: .continuous)

        if isSelected || hoveredNavigationItemID == id {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func updateNavigationHover(id: String, isHovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            if isHovering {
                hoveredNavigationItemID = id
            } else if hoveredNavigationItemID == id {
                hoveredNavigationItemID = nil
            }
        }
    }

    private var sidebarFolderSection: some View {
        VStack(alignment: .leading, spacing: 5) {
            HStack(spacing: 8) {
                Text(localization.string("Folders"))
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)

                Spacer(minLength: 8)

                HStack(spacing: 0) {
                    folderSectionButton(
                        id: "new-folder",
                        systemImage: "plus",
                        label: localization.string("New Folder")
                    ) {
                        onCreateFolder(nil)
                    }

                    folderSectionButton(
                        id: "toggle-folders",
                        systemImage: isFolderSectionExpanded ? "chevron.down" : "chevron.right",
                        label: localization.string("Folders")
                    ) {
                        withAnimation(.smooth(duration: 0.18)) {
                            isFolderSectionExpanded.toggle()
                        }
                    }
                }
                .opacity(isFolderSectionHovered ? 1 : 0)
                .allowsHitTesting(isFolderSectionHovered)
            }
            .padding(.leading, 10)
            .padding(.trailing, 7)
            .frame(height: 32)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background {
                folderSectionHeaderBackground
            }
            .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            .onHover(perform: updateFolderSectionHover)

            if isFolderSectionExpanded {
                if visibleFolderRows.isEmpty {
                    emptyFolderPlaceholder
                        .transition(.opacity.combined(with: .move(edge: .top)))
                } else {
                    VStack(alignment: .leading, spacing: 2) {
                        ForEach(visibleFolderRows) { row in
                            folderRow(row)
                        }
                    }
                    .transition(.opacity.combined(with: .move(edge: .top)))
                }
            }
        }
    }

    private var visibleFolderRows: [MomentoSidebarFolderRow] {
        var rows: [MomentoSidebarFolderRow] = []
        appendVisibleFolders(parentID: nil, depth: 0, to: &rows)
        return rows
    }

    private func appendVisibleFolders(
        parentID: AssetFolder.ID?,
        depth: Int,
        to rows: inout [MomentoSidebarFolderRow]
    ) {
        for folder in folders.filter({ $0.parentID == parentID }).sorted(by: folderSort) {
            let hasChildren = folders.contains { $0.parentID == folder.id }
            rows.append(MomentoSidebarFolderRow(folder: folder, depth: depth, hasChildren: hasChildren))
            if hasChildren, expandedFolderIDs.contains(folder.id) {
                appendVisibleFolders(parentID: folder.id, depth: depth + 1, to: &rows)
            }
        }
    }

    private func folderSort(_ lhs: AssetFolder, _ rhs: AssetFolder) -> Bool {
        if lhs.sortIndex == rhs.sortIndex {
            return lhs.createdAt < rhs.createdAt
        }
        return lhs.sortIndex < rhs.sortIndex
    }

    private func folderRow(_ row: MomentoSidebarFolderRow) -> some View {
        let folder = row.folder
        let rowID = "folder-\(folder.id)"
        let isSelected = selection == "folder-\(folder.id)"
        let isHovered = hoveredFolderID == folder.id
        let isDropTargeted = targetedAssetDropID == rowID
        let foregroundStyle = sidebarNavigationForeground(
            isSelected: isSelected,
            isHovered: isHovered || isDropTargeted
        )

        return HStack(spacing: 0) {
            Button {
                withAnimation(.smooth(duration: 0.16)) {
                    if expandedFolderIDs.contains(folder.id) {
                        expandedFolderIDs.remove(folder.id)
                    } else {
                        expandedFolderIDs.insert(folder.id)
                    }
                }
            } label: {
                Image(systemName: disclosureIcon(for: folder.id))
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(foregroundStyle)
                    .frame(
                        width: MomentoSidebarMenuMetrics.folderDisclosureHitWidth,
                        height: MomentoSidebarMenuMetrics.folderDisclosureHitHeight
                    )
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            Text(folder.name)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(foregroundStyle)
                .lineLimit(1)
                .truncationMode(.tail)

            Spacer(minLength: 6)

            countBadge(counts.folders[folder.id], foregroundStyle: foregroundStyle)
        }
        .padding(.leading, CGFloat(row.depth) * 18 + 4)
        .padding(.trailing, 7)
        .frame(height: 30)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background {
            sidebarAssetDropRowBackground(
                isSelected: isSelected,
                isHovered: isHovered,
                isDropTargeted: isDropTargeted
            )
        }
        .contentShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
        .pointerStyle(.link)
        .onTapGesture {
            withAnimation(.smooth(duration: 0.16)) {
                selection = "folder-\(folder.id)"
            }
        }
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                if hovering {
                    hoveredFolderID = folder.id
                } else if hoveredFolderID == folder.id {
                    hoveredFolderID = nil
                }
            }
        }
        .contextMenu {
            folderContextMenuItems(for: folder)
        }
        .onDrop(of: [AssetDragPasteboardWriter.assetIDsUTType], delegate: MomentoSidebarAssetDropDelegate(
            currentLibraryID: currentLibraryID,
            targetID: rowID,
            targetedAssetDropID: $targetedAssetDropID
        ) { assetIDs in
            onAssignDroppedAssetsToFolder(assetIDs, folder.id)
        })
        .help(folder.name)
        .accessibilityLabel(folder.name)
    }

    @ViewBuilder
    private func sidebarAssetDropRowBackground(
        isSelected: Bool,
        isHovered: Bool,
        isDropTargeted: Bool
    ) -> some View {
        let shape = RoundedRectangle(cornerRadius: 9, style: .continuous)

        if isSelected || isHovered || isDropTargeted {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func disclosureIcon(for folderID: AssetFolder.ID) -> String {
        expandedFolderIDs.contains(folderID) ? "chevron.down" : "chevron.right"
    }

    @ViewBuilder
    private func folderContextMenuItems(for folder: AssetFolder) -> some View {
        Button {
            withAnimation(.smooth(duration: 0.16)) {
                _ = expandedFolderIDs.insert(folder.id)
            }
            onCreateFolder(folder.id)
        } label: {
            Label(localization.string("New Subfolder"), systemImage: "folder.badge.plus")
        }

        Button {
            onRenameFolder(folder.id)
        } label: {
            Label(localization.string("Edit Folder"), systemImage: "pencil")
        }

        Divider()

        Button(role: .destructive) {
            onDeleteFolder(folder.id)
        } label: {
            Label {
                Text(localization.string("Delete Folder"))
                    .foregroundStyle(.red)
            } icon: {
                Image(systemName: "trash")
                    .foregroundStyle(.red)
            }
        }
    }

    @ViewBuilder
    private func countBadge(_ count: Int?, foregroundStyle: Color) -> some View {
        if let count, count > 0 {
            Text("\(count)")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(foregroundStyle.opacity(0.58))
                .monospacedDigit()
                .lineLimit(1)
                .frame(minWidth: 16, alignment: .trailing)
        }
    }

    private func folderSectionButton(
        id: String,
        systemImage: String,
        label: String,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            Image(systemName: systemImage)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(hoveredFolderControlID == id ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
                .frame(width: 24, height: 24)
                .background {
                    folderSectionButtonBackground(id: id)
                }
                .contentShape(RoundedRectangle(cornerRadius: 7, style: .continuous))
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateFolderControlHover(id: id, isHovering: hovering)
        }
        .help(label)
        .accessibilityLabel(label)
    }

    @ViewBuilder
    private var folderSectionHeaderBackground: some View {
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        if isFolderSectionHovered {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    @ViewBuilder
    private func folderSectionButtonBackground(id: String) -> some View {
        let shape = RoundedRectangle(cornerRadius: 7, style: .continuous)

        if hoveredFolderControlID == id {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private var emptyFolderPlaceholder: some View {
        HStack {
            Text(localization.string("No folders"))
                .font(.system(size: 12, weight: .regular))
                .foregroundStyle(MomentoTheme.tertiaryText.opacity(0.72))
                .lineLimit(1)
        }
        .padding(.horizontal, 8)
        .frame(height: 30)
        .frame(maxWidth: .infinity, alignment: .center)
    }

    private func updateFolderControlHover(id: String, isHovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            if isHovering {
                hoveredFolderControlID = id
            } else if hoveredFolderControlID == id {
                hoveredFolderControlID = nil
            }
        }
    }

    private func updateFolderSectionHover(_ hovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            isFolderSectionHovered = hovering
            if !hovering {
                hoveredFolderControlID = nil
            }
        }
    }

    private var bottomActionBar: some View {
        HStack(spacing: 6) {
            sidebarFooterButton(
                id: "trash",
                systemImage: "trash",
                label: localization.string("Trash"),
                isSelected: selection == "trash",
                count: counts.trash
            ) {
                withAnimation(.smooth(duration: 0.16)) {
                    selection = "trash"
                }
            }

            sidebarFooterButton(
                id: "settings",
                systemImage: "gear",
                label: localization.string("Settings"),
                action: openSettings.callAsFunction
            )

            sidebarFooterIcon(
                id: "help",
                systemImage: "questionmark.circle",
                label: localization.string("Help Center")
            )
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func sidebarFooterButton(
        id: String,
        systemImage: String,
        label: String,
        isSelected: Bool = false,
        count: Int? = nil,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            sidebarFooterIconContent(
                id: id,
                systemImage: systemImage,
                isSelected: isSelected,
                count: count
            )
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateFooterHover(id: id, isHovering: hovering)
        }
        .help(label)
        .accessibilityLabel(label)
    }

    private func sidebarFooterIcon(id: String, systemImage: String, label: String) -> some View {
        sidebarFooterIconContent(
            id: id,
            systemImage: systemImage,
            isSelected: false,
            count: nil
        )
        .onHover { hovering in
            updateFooterHover(id: id, isHovering: hovering)
        }
        .help(label)
        .accessibilityLabel(label)
    }

    private func sidebarFooterIconContent(
        id: String,
        systemImage: String,
        isSelected: Bool,
        count: Int?
    ) -> some View {
        Image(systemName: systemImage)
            .font(.system(size: 14, weight: .medium))
            .foregroundStyle(isSelected || hoveredFooterActionID == id ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
            .frame(width: 28, height: 28)
            .background {
                sidebarFooterIconBackground(id: id, isSelected: isSelected)
            }
            .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            .overlay(alignment: .topTrailing) {
                if let count, count > 0 {
                    Text("\(count)")
                        .font(.system(size: 8, weight: .bold))
                        .foregroundStyle(MomentoTheme.primaryText)
                        .monospacedDigit()
                        .padding(.horizontal, 3)
                        .frame(minWidth: 11, minHeight: 11)
                        .background(Capsule().fill(MomentoTheme.sidebarIconHoverBackground))
                        .offset(x: 5, y: -4)
                }
            }
    }

    @ViewBuilder
    private func sidebarFooterIconBackground(id: String, isSelected: Bool) -> some View {
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        if isSelected || hoveredFooterActionID == id {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func updateFooterHover(id: String, isHovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            if isHovering {
                hoveredFooterActionID = id
            } else if hoveredFooterActionID == id {
                hoveredFooterActionID = nil
            }
        }
    }

    private var libraryMenu: some View {
        Button {
            withAnimation(.smooth(duration: 0.16)) {
                isLibrarySwitcherPresented.toggle()
            }
        } label: {
            HStack(spacing: 8) {
                Image(systemName: "archivebox.fill")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 20, height: 20)
                    .background(.blue.gradient, in: RoundedRectangle(cornerRadius: 5, style: .continuous))

                Text(libraryName ?? localization.string("No library selected"))
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .lineLimit(1)
                    .truncationMode(.tail)

                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 8, weight: .semibold))
                    .foregroundStyle(isLibraryMenuHovered || isLibrarySwitcherPresented ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
            }
            .padding(.leading, 5)
            .padding(.trailing, 7)
            .padding(.vertical, 3)
            .frame(height: MomentoTheme.librarySelectorHeight)
            .background { libraryMenuBackground }
            .contentShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateLibraryMenuHover(hovering)
        }
        .accessibilityLabel(localization.string("Library"))
        .help(localization.string("Library"))
    }

    @ViewBuilder
    private var libraryMenuBackground: some View {
        let shape = RoundedRectangle(cornerRadius: 9, style: .continuous)

        if isLibraryMenuHovered || isLibrarySwitcherPresented {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func updateLibraryMenuHover(_ hovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            isLibraryMenuHovered = hovering
        }
    }

    private func performLibrarySwitcherAction(_ action: @escaping () -> Void) -> () -> Void {
        {
            dismissLibrarySwitcher()
            action()
        }
    }

    private func dismissLibrarySwitcher() {
        withAnimation(.smooth(duration: 0.16)) {
            isLibrarySwitcherPresented = false
        }
    }
}

private struct MomentoSidebarFolderRow: Identifiable {
    var folder: AssetFolder
    var depth: Int
    var hasChildren: Bool

    var id: AssetFolder.ID { folder.id }
}

private struct MomentoSidebarAssetDropDelegate: DropDelegate {
    var currentLibraryID: AssetLibrary.ID?
    var targetID: String
    @Binding var targetedAssetDropID: String?
    var onDropAssetIDs: (Set<AssetItem.ID>) -> Void

    func validateDrop(info: DropInfo) -> Bool {
        currentLibraryID != nil &&
            info.hasItemsConforming(to: [AssetDragPasteboardWriter.assetIDsUTType])
    }

    func dropEntered(info: DropInfo) {
        guard validateDrop(info: info) else {
            return
        }

        withAnimation(.smooth(duration: 0.12)) {
            targetedAssetDropID = targetID
        }
    }

    func dropExited(info: DropInfo) {
        clearTargetIfNeeded()
    }

    func dropUpdated(info: DropInfo) -> DropProposal? {
        guard validateDrop(info: info) else {
            return nil
        }

        return DropProposal(operation: .copy)
    }

    func performDrop(info: DropInfo) -> Bool {
        clearTargetIfNeeded()

        guard validateDrop(info: info),
              let provider = info.itemProviders(for: [AssetDragPasteboardWriter.assetIDsUTType]).first else {
            return false
        }

        let expectedLibraryID = currentLibraryID
        provider.loadDataRepresentation(forTypeIdentifier: AssetDragPasteboardWriter.assetIDsTypeIdentifier) { data, _ in
            guard let data,
                  let payload = try? JSONDecoder.momento.decode(AssetDragPasteboardPayload.self, from: data),
                  payload.libraryID == expectedLibraryID else {
                return
            }

            Task { @MainActor in
                onDropAssetIDs(Set(payload.assetIDs))
            }
        }

        return true
    }

    private func clearTargetIfNeeded() {
        guard targetedAssetDropID == targetID else {
            return
        }

        withAnimation(.smooth(duration: 0.12)) {
            targetedAssetDropID = nil
        }
    }
}

private struct MomentoLibrarySwitcherMenu: View {
    @Environment(\.appLocalization) private var localization

    var recentLibraries: [RecentLibraryReference]
    var currentLibraryID: RecentLibraryReference.ID?
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onImportLibrary: () -> Void
    var onExportLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onRenameLibrary: (RecentLibraryReference.ID) -> Void
    var onDeleteLibrary: (RecentLibraryReference.ID) -> Void
    var onRevealLibrary: (RecentLibraryReference.ID) -> Void
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void
    var onReloadLibrary: () -> Void

    @State private var hoveredLibraryID: RecentLibraryReference.ID?
    @State private var hoveredActionID: String?
    @State private var activeMoreLibraryID: RecentLibraryReference.ID?
    @State private var hoveredMoreActionID: String?
    @State private var displayedLibraries: [RecentLibraryReference] = []
    @State private var draggingLibraryID: RecentLibraryReference.ID?

    var body: some View {
        ZStack(alignment: .topLeading) {
            menuPanel

            if let activeLibrary = activeMoreLibrary,
               let activeIndex = visibleLibraries.firstIndex(where: { $0.id == activeLibrary.id }) {
                libraryMoreMenu(activeLibrary)
                    .offset(libraryMoreMenuOffset(for: activeIndex))
                    .transition(.opacity.combined(with: .scale(scale: 0.98, anchor: .topLeading)))
                    .zIndex(60)
            }
        }
        .frame(width: MomentoTheme.librarySwitcherWidth + 178, alignment: .topLeading)
        .onAppear(perform: syncDisplayedLibraries)
        .onChange(of: recentLibraries) { _, libraries in
            guard draggingLibraryID == nil else {
                return
            }

            displayedLibraries = libraries
        }
    }

    private var menuPanel: some View {
        VStack(alignment: .leading, spacing: 5) {
            if !visibleLibraries.isEmpty {
                VStack(spacing: 2) {
                    ForEach(visibleLibraries) { library in
                        libraryRow(library)
                    }
                }

                librarySwitcherSeparator
            }

            VStack(spacing: 1) {
                actionRow(
                    id: "create-library",
                    title: localization.string("Create Library"),
                    systemImage: "archivebox",
                    action: onCreateLibrary
                )

                actionRow(
                    id: "open-library",
                    title: localization.string("Open Other Library"),
                    systemImage: "folder",
                    action: onOpenLibrary
                )

                actionRow(
                    id: "import-library",
                    title: localization.string("Import Library"),
                    systemImage: "square.and.arrow.down.on.square",
                    action: onImportLibrary
                )

                actionRow(
                    id: "export-library",
                    title: localization.string("Export Library"),
                    systemImage: "square.and.arrow.up.on.square",
                    action: onExportLibrary
                )

                actionRow(
                    id: "reload-library",
                    title: localization.string("Clear Cache and Reload"),
                    systemImage: "arrow.clockwise",
                    action: onReloadLibrary
                )
            }
        }
        .padding(8)
        .frame(width: MomentoTheme.librarySwitcherWidth, alignment: .topLeading)
        .background {
            MomentoGlassBackground(glass: .regular, cornerRadius: 14)
        }
        .overlay {
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .strokeBorder(MomentoTheme.subtleStroke.opacity(0.5), lineWidth: 0.6)
        }
        .shadow(color: .black.opacity(0.22), radius: 18, y: 10)
    }

    private var visibleLibraries: [RecentLibraryReference] {
        displayedLibraries.isEmpty ? recentLibraries : displayedLibraries
    }

    private var activeMoreLibrary: RecentLibraryReference? {
        guard let activeMoreLibraryID else {
            return nil
        }

        return visibleLibraries.first { $0.id == activeMoreLibraryID }
    }

    private func syncDisplayedLibraries() {
        displayedLibraries = recentLibraries
    }

    private func libraryRow(_ library: RecentLibraryReference) -> some View {
        let isSelected = library.id == currentLibraryID
        let isHovered = hoveredLibraryID == library.id

        return HStack(spacing: 8) {
            libraryDragHandle(isActive: isSelected || isHovered)
                .contentShape(Rectangle())
                .dragHandleCursor()

            Button {
                onSwitchLibrary(library.id)
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "archivebox.fill")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(.white)
                        .frame(width: 26, height: 26)
                        .background(.blue.gradient, in: RoundedRectangle(cornerRadius: 7, style: .continuous))

                    VStack(alignment: .leading, spacing: 1) {
                        Text(library.name)
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(isSelected ? .white : MomentoTheme.primaryText)
                            .lineLimit(1)

                        Text(libraryPath(for: library))
                            .font(.system(size: 11))
                            .foregroundStyle(isSelected ? MomentoTheme.primaryText.opacity(0.82) : MomentoTheme.secondaryText)
                            .lineLimit(1)
                    }

                    Spacer(minLength: 8)

                    if isSelected {
                        Image(systemName: "checkmark")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(MomentoTheme.primaryText)
                            .frame(width: 18)
                    }
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .pointerStyle(.link)

            Button {
                withAnimation(.smooth(duration: 0.14)) {
                    hoveredMoreActionID = nil
                    activeMoreLibraryID = activeMoreLibraryID == library.id ? nil : library.id
                }
            } label: {
                Image(systemName: "ellipsis")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(isSelected || isHovered ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
                    .frame(width: 24, height: 24)
                    .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
            }
            .buttonStyle(.plain)
            .pointerStyle(.link)
            .help(localization.string("More Actions"))
        }
        .padding(.horizontal, 7)
        .frame(height: 42)
        .background {
            menuRowBackground(isHovered: isHovered, isSelected: isSelected)
        }
        .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .contentShape(.dragPreview, RoundedRectangle(cornerRadius: 10, style: .continuous))
        .onDrag {
            draggingLibraryID = library.id
            activeMoreLibraryID = nil
            return NSItemProvider(object: library.id as NSString)
        } preview: {
            libraryDragPreview(library)
        }
        .onDrop(of: [.plainText], delegate: LibraryReorderDropDelegate(
            targetLibrary: library,
            displayedLibraries: $displayedLibraries,
            draggingLibraryID: $draggingLibraryID,
            onMoveLibrary: onMoveLibrary
        ))
        .zIndex(activeMoreLibraryID == library.id ? 50 : 0)
        .onHover { hovering in
            updateLibraryHover(library.id, hovering: hovering)
        }
    }

    private func libraryDragPreview(_ library: RecentLibraryReference) -> some View {
        HStack(spacing: 8) {
            libraryDragHandle(isActive: true)

            Image(systemName: "archivebox.fill")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(.white)
                .frame(width: 26, height: 26)
                .background(.blue.gradient, in: RoundedRectangle(cornerRadius: 7, style: .continuous))

            VStack(alignment: .leading, spacing: 1) {
                Text(library.name)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .lineLimit(1)

                Text(libraryPath(for: library))
                    .font(.system(size: 11))
                    .foregroundStyle(MomentoTheme.primaryText.opacity(0.82))
                    .lineLimit(1)
            }

            Spacer(minLength: 8)
        }
        .padding(.horizontal, 7)
        .frame(width: MomentoTheme.librarySwitcherWidth - 16, height: 42)
        .background {
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(MomentoTheme.sidebarIconHoverBackground)
        }
    }

    private func libraryDragHandle(isActive: Bool) -> some View {
        VStack(spacing: 2) {
            ForEach(0..<3, id: \.self) { _ in
                HStack(spacing: 2) {
                    Circle()
                        .fill(isActive ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
                        .frame(width: 2.4, height: 2.4)

                    Circle()
                        .fill(isActive ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
                        .frame(width: 2.4, height: 2.4)
                }
            }
        }
        .frame(width: 16)
    }

    private func libraryMoreMenuOffset(for index: Int) -> CGSize {
        CGSize(width: MomentoTheme.librarySwitcherWidth - 11, height: 15 + CGFloat(index) * 44)
    }

    private func libraryMoreMenu(_ library: RecentLibraryReference) -> some View {
        VStack(spacing: 2) {
            libraryMoreActionRow(
                id: "\(library.id)-rename",
                title: localization.string("Edit Library"),
                systemImage: "pencil",
                isDestructive: false
            ) {
                activeMoreLibraryID = nil
                onRenameLibrary(library.id)
            }

            libraryMoreActionRow(
                id: "\(library.id)-reveal",
                title: localization.string("Reveal in Finder"),
                systemImage: "finder",
                isDestructive: false
            ) {
                activeMoreLibraryID = nil
                onRevealLibrary(library.id)
            }

            librarySwitcherSeparator

            libraryMoreActionRow(
                id: "\(library.id)-delete",
                title: localization.string("Delete Library"),
                systemImage: "trash",
                isDestructive: true
            ) {
                activeMoreLibraryID = nil
                onDeleteLibrary(library.id)
            }
        }
        .padding(6)
        .frame(width: MomentoSidebarMenuMetrics.libraryMoreMenuWidth, alignment: .leading)
        .background {
            MomentoGlassBackground(glass: .regular, cornerRadius: 12)
        }
        .overlay {
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .strokeBorder(MomentoTheme.subtleStroke.opacity(0.45), lineWidth: 0.6)
        }
        .shadow(color: .black.opacity(0.2), radius: 14, y: 8)
    }

    private var librarySwitcherSeparator: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(MomentoSidebarMenuMetrics.separatorAlpha))
            .frame(height: MomentoSidebarMenuMetrics.separatorHeight)
            .padding(.horizontal, MomentoSidebarMenuMetrics.separatorHorizontalInset)
            .padding(.vertical, MomentoSidebarMenuMetrics.separatorVerticalPadding)
    }

    private func libraryMoreActionRow(
        id: String,
        title: String,
        systemImage: String,
        isDestructive: Bool,
        action: @escaping () -> Void
    ) -> some View {
        let isHovered = hoveredMoreActionID == id

        return Button(action: action) {
            HStack(spacing: 8) {
                Image(systemName: systemImage)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(isDestructive ? Color.red : MomentoTheme.secondaryText)
                    .frame(width: 16)

                Text(title)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(isDestructive ? Color.red : MomentoTheme.primaryText)

                Spacer()
            }
            .padding(.horizontal, 8)
            .frame(height: 28)
            .background {
                moreActionBackground(isHovered: isHovered)
            }
            .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.12)) {
                if hovering {
                    hoveredMoreActionID = id
                } else if hoveredMoreActionID == id {
                    hoveredMoreActionID = nil
                }
            }
        }
    }

    private func actionRow(
        id: String,
        title: String,
        systemImage: String,
        action: @escaping () -> Void
    ) -> some View {
        let isHovered = hoveredActionID == id

        return Button(action: action) {
            HStack(spacing: 10) {
                Image(systemName: systemImage)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(MomentoTheme.secondaryText)
                    .frame(width: 18)

                Text(title)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(MomentoTheme.primaryText)

                Spacer()
            }
            .padding(.horizontal, 9)
            .frame(height: 30)
            .background {
                menuRowBackground(isHovered: isHovered, isSelected: false)
            }
            .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateActionHover(id, hovering: hovering)
        }
    }

    @ViewBuilder
    private func menuRowBackground(isHovered: Bool, isSelected: Bool) -> some View {
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        if isSelected {
            Color.clear
                .glassEffect(.regular.tint(Color.accentColor), in: shape)
        } else if isHovered {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    @ViewBuilder
    private func moreActionBackground(isHovered: Bool) -> some View {
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        if isHovered {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func updateLibraryHover(_ id: RecentLibraryReference.ID, hovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            if hovering {
                hoveredLibraryID = id
            } else if hoveredLibraryID == id {
                hoveredLibraryID = nil
            }
        }
    }

    private func updateActionHover(_ id: String, hovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            if hovering {
                hoveredActionID = id
            } else if hoveredActionID == id {
                hoveredActionID = nil
            }
        }
    }

    private func libraryPath(for library: RecentLibraryReference) -> String {
        guard let resolved = try? RecentLibraryStore().resolve(library) else {
            return ""
        }

        return resolved.url.deletingLastPathComponent().path
    }
}

private struct LibraryReorderDropDelegate: DropDelegate {
    var targetLibrary: RecentLibraryReference
    @Binding var displayedLibraries: [RecentLibraryReference]
    @Binding var draggingLibraryID: RecentLibraryReference.ID?
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void

    func dropEntered(info: DropInfo) {
        reorderIfNeeded(info: info)
    }

    func dropUpdated(info: DropInfo) -> DropProposal? {
        reorderIfNeeded(info: info)
        return DropProposal(operation: .move)
    }

    func performDrop(info: DropInfo) -> Bool {
        draggingLibraryID = nil
        return true
    }

    private func reorderIfNeeded(info: DropInfo) {
        guard let draggedID = draggingLibraryID,
              draggedID != targetLibrary.id,
              let sourceIndex = displayedLibraries.firstIndex(where: { $0.id == draggedID }),
              let targetIndex = displayedLibraries.firstIndex(where: { $0.id == targetLibrary.id }) else {
            return
        }

        let insertAfterTarget = info.location.y > 21
        if !insertAfterTarget, sourceIndex + 1 == targetIndex {
            return
        }
        if insertAfterTarget, sourceIndex == targetIndex + 1 {
            return
        }

        withAnimation(.smooth(duration: 0.14)) {
            let movedLibrary = displayedLibraries.remove(at: sourceIndex)
            guard let updatedTargetIndex = displayedLibraries.firstIndex(where: { $0.id == targetLibrary.id }) else {
                displayedLibraries.insert(movedLibrary, at: sourceIndex)
                return
            }

            let insertionIndex = insertAfterTarget ? updatedTargetIndex + 1 : updatedTargetIndex
            displayedLibraries.insert(movedLibrary, at: min(insertionIndex, displayedLibraries.endIndex))
            onMoveLibrary(draggedID, targetLibrary.id, insertAfterTarget)
        }
    }
}

private struct LibrarySwitcherDismissMonitor: NSViewRepresentable {
    @Binding var isPresented: Bool

    func makeCoordinator() -> Coordinator {
        Coordinator(isPresented: $isPresented)
    }

    func makeNSView(context: Context) -> NSView {
        let view = NSView()
        context.coordinator.view = view
        context.coordinator.update(isPresented: isPresented)
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        context.coordinator.isPresented = $isPresented
        context.coordinator.view = nsView
        context.coordinator.update(isPresented: isPresented)
    }

    static func dismantleNSView(_ nsView: NSView, coordinator: Coordinator) {
        coordinator.remove()
    }

    final class Coordinator {
        var isPresented: Binding<Bool>
        weak var view: NSView?
        private var monitor: Any?

        init(isPresented: Binding<Bool>) {
            self.isPresented = isPresented
        }

        func update(isPresented: Bool) {
            if isPresented {
                install()
            } else {
                remove()
            }
        }

        func remove() {
            if let monitor {
                NSEvent.removeMonitor(monitor)
                self.monitor = nil
            }
        }

        private func install() {
            guard monitor == nil else {
                return
            }

            monitor = NSEvent.addLocalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown]) { [weak self] event in
                self?.handle(event) ?? event
            }
        }

        private func handle(_ event: NSEvent) -> NSEvent {
            guard isPresented.wrappedValue else {
                return event
            }

            if let view, event.window === view.window {
                let location = view.convert(event.locationInWindow, from: nil)
                if view.bounds.contains(location) {
                    return event
                }
            }

            DispatchQueue.main.async { [weak self] in
                self?.isPresented.wrappedValue = false
            }
            return event
        }
    }
}

private extension View {
    func dragHandleCursor() -> some View {
        background(DragHandleCursorView())
    }
}

private struct DragHandleCursorView: NSViewRepresentable {
    func makeNSView(context: Context) -> DragHandleCursorNSView {
        DragHandleCursorNSView()
    }

    func updateNSView(_ nsView: DragHandleCursorNSView, context: Context) {
        nsView.window?.invalidateCursorRects(for: nsView)
    }
}

private final class DragHandleCursorNSView: NSView {
    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .openHand)
    }

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()
        window?.invalidateCursorRects(for: self)
    }
}

#Preview {
    MomentoSidebarView(selection: .constant("all-assets"))
        .frame(width: 236, height: 620)
}
