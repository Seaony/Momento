import AppKit
import SwiftUI

struct MomentoSidebarItem: Identifiable, Hashable {
    var id: String
    var title: String
    var systemImage: String
    var count: Int?
    var tint: Color?

    init(
        id: String,
        title: String,
        systemImage: String,
        count: Int? = nil,
        tint: Color? = nil
    ) {
        self.id = id
        self.title = title
        self.systemImage = systemImage
        self.count = count
        self.tint = tint
    }
}

struct MomentoSidebarSection: Identifiable, Hashable {
    var id: String
    var title: String
    var items: [MomentoSidebarItem]
    var isCollapsible: Bool

    init(
        id: String,
        title: String,
        items: [MomentoSidebarItem],
        isCollapsible: Bool = true
    ) {
        self.id = id
        self.title = title
        self.items = items
        self.isCollapsible = isCollapsible
    }
}

struct MomentoSidebarView: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.openSettings) private var openSettings

    var sections: [MomentoSidebarSection]
    @Binding var selection: MomentoSidebarItem.ID?
    var libraryName: String?
    var currentLibraryID: RecentLibraryReference.ID?
    var recentLibraries: [RecentLibraryReference]
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onRenameLibrary: (RecentLibraryReference.ID) -> Void
    var onDeleteLibrary: (RecentLibraryReference.ID) -> Void
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void
    var onReloadLibrary: () -> Void
    var onCloseLibrary: () -> Void
    var onItemContextMenu: ((MomentoSidebarItem) -> AnyView)?

    @State private var collapsedSectionIDs: Set<MomentoSidebarSection.ID> = []
    @State private var hoveredFooterActionID: String?
    @State private var isLibrarySwitcherPresented = false

    init(
        sections: [MomentoSidebarSection] = .momentoDefaultSections,
        selection: Binding<MomentoSidebarItem.ID?>,
        libraryName: String? = nil,
        currentLibraryID: RecentLibraryReference.ID? = nil,
        recentLibraries: [RecentLibraryReference] = [],
        onCreateLibrary: @escaping () -> Void = {},
        onOpenLibrary: @escaping () -> Void = {},
        onSwitchLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onRenameLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onDeleteLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onMoveLibrary: @escaping (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void = { _, _, _ in },
        onReloadLibrary: @escaping () -> Void = {},
        onCloseLibrary: @escaping () -> Void = {},
        onItemContextMenu: ((MomentoSidebarItem) -> AnyView)? = nil
    ) {
        self.sections = sections
        self._selection = selection
        self.libraryName = libraryName
        self.currentLibraryID = currentLibraryID
        self.recentLibraries = recentLibraries
        self.onCreateLibrary = onCreateLibrary
        self.onOpenLibrary = onOpenLibrary
        self.onSwitchLibrary = onSwitchLibrary
        self.onRenameLibrary = onRenameLibrary
        self.onDeleteLibrary = onDeleteLibrary
        self.onMoveLibrary = onMoveLibrary
        self.onReloadLibrary = onReloadLibrary
        self.onCloseLibrary = onCloseLibrary
        self.onItemContextMenu = onItemContextMenu
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
                .padding(.bottom, 10)

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 14) {
                    ForEach(sections) { section in
                        sectionView(section)
                    }
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
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
                onMoveLibrary: onMoveLibrary,
                onReloadLibrary: performLibrarySwitcherAction(onReloadLibrary)
            )
            .background {
                LibrarySwitcherDismissMonitor(isPresented: $isLibrarySwitcherPresented)
            }
            .padding(.top, MomentoTheme.floatingSidebarTitlebarContentInset + 42)
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

    private var bottomActionBar: some View {
        HStack(spacing: 6) {
            sidebarFooterButton(
                id: "trash",
                systemImage: "trash",
                label: localization.string("Trash"),
                isSelected: selection == "trash"
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
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            sidebarFooterIconContent(
                id: id,
                systemImage: systemImage,
                isSelected: isSelected
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
            isSelected: false
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
        isSelected: Bool
    ) -> some View {
        Image(systemName: systemImage)
            .font(.system(size: 14, weight: .medium))
            .foregroundStyle(isSelected || hoveredFooterActionID == id ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
            .frame(width: 28, height: 28)
            .background {
                sidebarFooterIconBackground(id: id, isSelected: isSelected)
            }
            .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
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

    private func sectionView(_ section: MomentoSidebarSection) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            Button {
                guard section.isCollapsible else { return }
                withAnimation(.smooth(duration: 0.18)) {
                    if collapsedSectionIDs.contains(section.id) {
                        collapsedSectionIDs.remove(section.id)
                    } else {
                        collapsedSectionIDs.insert(section.id)
                    }
                }
            } label: {
                HStack(spacing: 5) {
                    Image(systemName: "chevron.down")
                        .font(.system(size: 9, weight: .bold))
                        .rotationEffect(.degrees(collapsedSectionIDs.contains(section.id) ? -90 : 0))
                        .opacity(section.isCollapsible ? 0.75 : 0)

                    Text(section.title.uppercased())
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(MomentoTheme.tertiaryText)
                    Spacer()
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 6)

            if !collapsedSectionIDs.contains(section.id) {
                VStack(spacing: 2) {
                    ForEach(section.items) { item in
                        MomentoSidebarRow(item: item, isSelected: selection == item.id) {
                            withAnimation(.smooth(duration: 0.16)) {
                                selection = item.id
                            }
                        }
                        .contextMenu {
                            if let menu = onItemContextMenu?(item) {
                                menu
                            }
                        }
                    }
                }
                .transition(.opacity.combined(with: .move(edge: .top)))
            }
        }
    }

    private var libraryMenu: some View {
        Button {
            withAnimation(.smooth(duration: 0.16)) {
                isLibrarySwitcherPresented.toggle()
            }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "archivebox.fill")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 28, height: 28)
                    .background(.blue.gradient, in: RoundedRectangle(cornerRadius: 7, style: .continuous))

                VStack(alignment: .leading, spacing: 1) {
                    Text(verbatim: "Momento")
                        .font(.system(size: 14, weight: .semibold))
                    Text(libraryName ?? localization.string("No library selected"))
                        .font(.system(size: 11))
                        .foregroundStyle(MomentoTheme.secondaryText)
                        .lineLimit(1)
                }

                Spacer()

                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 9, weight: .semibold))
                    .foregroundStyle(MomentoTheme.tertiaryText)
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .accessibilityLabel(localization.string("Library"))
        .help(localization.string("Library"))
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

private struct MomentoLibrarySwitcherMenu: View {
    @Environment(\.appLocalization) private var localization

    var recentLibraries: [RecentLibraryReference]
    var currentLibraryID: RecentLibraryReference.ID?
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onRenameLibrary: (RecentLibraryReference.ID) -> Void
    var onDeleteLibrary: (RecentLibraryReference.ID) -> Void
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void
    var onReloadLibrary: () -> Void

    @State private var hoveredLibraryID: RecentLibraryReference.ID?
    @State private var hoveredActionID: String?
    @State private var activeMoreLibraryID: RecentLibraryReference.ID?
    @State private var hoveredMoreActionID: String?

    var body: some View {
        ZStack(alignment: .topLeading) {
            menuPanel

            if let activeLibrary = activeMoreLibrary,
               let activeIndex = recentLibraries.firstIndex(where: { $0.id == activeLibrary.id }) {
                libraryMoreMenu(activeLibrary)
                    .offset(libraryMoreMenuOffset(for: activeIndex))
                    .transition(.opacity.combined(with: .scale(scale: 0.98, anchor: .topLeading)))
                    .zIndex(60)
            }
        }
        .frame(width: MomentoTheme.librarySwitcherWidth + 178, alignment: .topLeading)
    }

    private var menuPanel: some View {
        VStack(alignment: .leading, spacing: 5) {
            if !recentLibraries.isEmpty {
                VStack(spacing: 2) {
                    ForEach(recentLibraries) { library in
                        libraryRow(library)
                    }
                }

                Divider()
                    .overlay(MomentoTheme.subtleStroke.opacity(0.65))
                    .padding(.vertical, 1)
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

    private var activeMoreLibrary: RecentLibraryReference? {
        guard let activeMoreLibraryID else {
            return nil
        }

        return recentLibraries.first { $0.id == activeMoreLibraryID }
    }

    private func libraryRow(_ library: RecentLibraryReference) -> some View {
        let isSelected = library.id == currentLibraryID
        let isHovered = hoveredLibraryID == library.id

        return HStack(spacing: 8) {
            libraryDragHandle(isActive: isSelected || isHovered)
                .contentShape(Rectangle())
                .dragHandleCursor()
                .draggable(library.id) {
                    libraryDragPreview(library)
                }

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
                            .foregroundStyle(MomentoTheme.primaryText)
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
        .dropDestination(for: RecentLibraryReference.ID.self) { draggedIDs, location in
            guard let draggedID = draggedIDs.first,
                  draggedID != library.id else {
                return false
            }

            let insertAfterTarget = location.y > 21
            onMoveLibrary(draggedID, library.id, insertAfterTarget)
            return true
        }
        .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
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
        .frame(width: 150, alignment: .leading)
        .background {
            MomentoGlassBackground(glass: .regular.tint(Color.white.opacity(0.08)), cornerRadius: 12)
        }
        .overlay {
            RoundedRectangle(cornerRadius: 12, style: .continuous)
                .strokeBorder(MomentoTheme.subtleStroke.opacity(0.45), lineWidth: 0.6)
        }
        .shadow(color: .black.opacity(0.2), radius: 14, y: 8)
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

private struct MomentoSidebarRow: View {
    @Environment(\.appLocalization) private var localization

    var item: MomentoSidebarItem
    var isSelected: Bool
    var action: () -> Void

    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Image(systemName: item.systemImage)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(item.tint ?? (isSelected ? Color.accentColor : MomentoTheme.secondaryText))
                    .frame(width: 18)

                Text(item.title)
                    .font(.system(size: 13, weight: isSelected ? .medium : .regular))
                    .lineLimit(1)

                Spacer(minLength: 8)

                if let count = item.count {
                    Text(count.formatted(.number.locale(localization.locale)))
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(MomentoTheme.tertiaryText)
                }
            }
            .padding(.horizontal, 8)
            .frame(height: 28)
            .background {
                rowBackground
            }
            .contentShape(RoundedRectangle(cornerRadius: MomentoTheme.rowRadius, style: .continuous))
        }
        .buttonStyle(.plain)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                isHovered = hovering
            }
        }
    }

    @ViewBuilder
    private var rowBackground: some View {
        let shape = RoundedRectangle(cornerRadius: MomentoTheme.rowRadius, style: .continuous)

        if isSelected {
            Color.clear
                .glassEffect(.regular.tint(Color.accentColor), in: shape)
        } else if isHovered {
            Color.clear
                .glassEffect(.regular, in: shape)
        } else {
            Color.clear
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

extension Array where Element == MomentoSidebarSection {
    static func momentoDefaultSections(localization: AppLocalization) -> [MomentoSidebarSection] {
        [
            MomentoSidebarSection(
                id: "favorites",
                title: localization.string("Favorites"),
                items: [
                    MomentoSidebarItem(id: "favorites", title: localization.string("Starred Assets"), systemImage: "star", count: 0, tint: .yellow)
                ],
                isCollapsible: false
            ),
            MomentoSidebarSection(
                id: "folders",
                title: localization.string("Folders"),
                items: [
                    MomentoSidebarItem(id: "folder-inspiration", title: localization.string("Inspiration"), systemImage: "folder", count: 0),
                    MomentoSidebarItem(id: "folder-screenshots", title: localization.string("Screenshots"), systemImage: "folder", count: 0)
                ]
            ),
            MomentoSidebarSection(
                id: "tags",
                title: localization.string("Tags"),
                items: [
                    MomentoSidebarItem(id: "tag-ui", title: "UI", systemImage: "tag", count: 0, tint: .blue),
                    MomentoSidebarItem(id: "tag-brand", title: "Brand", systemImage: "tag", count: 0, tint: .purple)
                ]
            )
        ]
    }

    static var momentoDefaultSections: [MomentoSidebarSection] {
        momentoDefaultSections(localization: AppLocalization(language: .system))
    }
}

#Preview {
    MomentoSidebarView(selection: .constant("all-assets"))
        .frame(width: 236, height: 620)
}
