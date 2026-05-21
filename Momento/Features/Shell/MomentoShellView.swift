import SwiftUI

struct MomentoShellView<Content: View>: View {
    @Environment(\.appLocalization) private var localization

    @Binding var sidebarSelection: MomentoSidebarItem.ID?
    @Binding var searchQuery: String
    @Binding var isCommandPalettePresented: Bool
    @Binding var isInspectorPresented: Bool

    var sidebarSections: [MomentoSidebarSection]
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
    var title: String
    var subtitle: String?
    var inspectorAsset: MomentoInspectorAsset?
    @Binding var inspectorTags: [String]
    @Binding var inspectorNotes: String
    var commands: [MomentoCommand]
    var onCommandSelected: (MomentoCommand) -> Void
    var content: () -> Content

    @State private var sidebarWidth = MomentoTheme.sidebarWidth
    @State private var sidebarResizeStartWidth: CGFloat?
    @State private var isSidebarCollapsed = false

    init(
        sidebarSelection: Binding<MomentoSidebarItem.ID?>,
        searchQuery: Binding<String>,
        isCommandPalettePresented: Binding<Bool>,
        isInspectorPresented: Binding<Bool> = .constant(true),
        sidebarSections: [MomentoSidebarSection] = .momentoDefaultSections,
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
        title: String = "All Assets",
        subtitle: String? = "0 items",
        inspectorAsset: MomentoInspectorAsset? = nil,
        inspectorTags: Binding<[String]> = .constant([]),
        inspectorNotes: Binding<String> = .constant(""),
        commands: [MomentoCommand] = .momentoDefaultCommands,
        onCommandSelected: @escaping (MomentoCommand) -> Void = { _ in },
        @ViewBuilder content: @escaping () -> Content
    ) {
        self._sidebarSelection = sidebarSelection
        self._searchQuery = searchQuery
        self._isCommandPalettePresented = isCommandPalettePresented
        self._isInspectorPresented = isInspectorPresented
        self.sidebarSections = sidebarSections
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
        self.title = title
        self.subtitle = subtitle
        self.inspectorAsset = inspectorAsset
        self._inspectorTags = inspectorTags
        self._inspectorNotes = inspectorNotes
        self.commands = commands
        self.onCommandSelected = onCommandSelected
        self.content = content
    }

    var body: some View {
        GeometryReader { geometry in
            shellContent(availableWidth: geometry.size.width)
        }
    }

    private func shellContent(availableWidth: CGFloat) -> some View {
        HStack(spacing: 0) {
            if !isSidebarCollapsed {
                floatingSidebar(availableWidth: availableWidth)
                    .transition(.move(edge: .leading).combined(with: .opacity))
            }

            VStack(spacing: 0) {
                content()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
            .frame(minWidth: MomentoTheme.contentMinWidth, maxWidth: .infinity, maxHeight: .infinity)

            if isInspectorPresented {
                trailingInspector
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)
        .animation(.smooth(duration: 0.18), value: isInspectorPresented)
        .animation(.smooth(duration: 0.18), value: isSidebarCollapsed)
        .momentoCommandPalette(
            isPresented: $isCommandPalettePresented,
            commands: commands,
            onSelect: onCommandSelected
        )
        .background {
            SidebarTitlebarToggleConfigurator(
                isCollapsed: $isSidebarCollapsed,
                buttonMinX: sidebarToggleButtonMinX(availableWidth: availableWidth),
                label: sidebarToggleLabel
            )
            .frame(width: 0, height: 0)
        }
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
    }

    private func floatingSidebar(availableWidth: CGFloat) -> some View {
        MomentoSidebarView(
            sections: sidebarSections,
            selection: $sidebarSelection,
            libraryName: libraryName,
            currentLibraryID: currentLibraryID,
            recentLibraries: recentLibraries,
            onCreateLibrary: onCreateLibrary,
            onOpenLibrary: onOpenLibrary,
            onSwitchLibrary: onSwitchLibrary,
            onRenameLibrary: onRenameLibrary,
            onDeleteLibrary: onDeleteLibrary,
            onMoveLibrary: onMoveLibrary,
            onReloadLibrary: onReloadLibrary,
            onCloseLibrary: onCloseLibrary
        )
        .frame(width: effectiveSidebarWidth(availableWidth: availableWidth))
        .overlay(alignment: .trailing) {
            sidebarResizeHandle(availableWidth: availableWidth)
        }
        .padding(.leading, MomentoTheme.floatingSidebarInset)
        .padding(.trailing, MomentoTheme.floatingSidebarInset)
        .padding(.vertical, MomentoTheme.floatingSidebarInset)
        .ignoresSafeArea(.container, edges: .top)
    }

    private var trailingInspector: some View {
        MomentoInspectorView(
            asset: inspectorAsset,
            tags: $inspectorTags,
            notes: $inspectorNotes
        )
        .frame(width: MomentoTheme.inspectorWidth)
        .transition(.move(edge: .trailing).combined(with: .opacity))
    }

    private func sidebarResizeHandle(availableWidth: CGFloat) -> some View {
        Rectangle()
            .fill(Color.clear)
            .frame(width: 14)
            .contentShape(Rectangle())
            .pointerStyle(.columnResize(directions: .all))
            .gesture(
                DragGesture(minimumDistance: 0, coordinateSpace: .global)
                    .onChanged { value in
                        let startWidth = sidebarResizeStartWidth ?? effectiveSidebarWidth(availableWidth: availableWidth)
                        let widthRange = sidebarWidthRange(availableWidth: availableWidth)
                        let proposedWidth = startWidth + value.translation.width

                        sidebarResizeStartWidth = startWidth

                        if proposedWidth < widthRange.lowerBound - MomentoTheme.sidebarCollapseDragOvershoot {
                            sidebarWidth = widthRange.lowerBound
                            collapseSidebarFromResize()
                            return
                        }

                        sidebarWidth = proposedWidth.clamped(to: widthRange)
                    }
                    .onEnded { _ in
                        sidebarResizeStartWidth = nil
                    }
            )
    }

    private func effectiveSidebarWidth(availableWidth: CGFloat) -> CGFloat {
        sidebarWidth.clamped(to: sidebarWidthRange(availableWidth: availableWidth))
    }

    private func sidebarWidthRange(availableWidth: CGFloat) -> ClosedRange<CGFloat> {
        let inspectorWidth = isInspectorPresented ? MomentoTheme.inspectorWidth : 0
        let availableSidebarWidth = availableWidth - MomentoTheme.contentMinWidth - inspectorWidth - MomentoTheme.floatingSidebarInset * 2
        let maxWidth = min(MomentoTheme.sidebarMaxWidth, max(MomentoTheme.sidebarMinWidth, availableSidebarWidth))

        return MomentoTheme.sidebarMinWidth...maxWidth
    }

    private var sidebarToggleLabel: String {
        localization.string(isSidebarCollapsed ? "Expand Sidebar" : "Collapse Sidebar")
    }

    private func sidebarToggleButtonMinX(availableWidth: CGFloat) -> CGFloat {
        if isSidebarCollapsed {
            return MomentoTheme.collapsedSidebarToggleLeadingInset
        }

        return MomentoTheme.floatingSidebarInset + effectiveSidebarWidth(availableWidth: availableWidth) - MomentoTheme.sidebarTitlebarButtonTrailingInset - MomentoTheme.sidebarTitlebarButtonSize
    }

    private func collapseSidebarFromResize() {
        guard !isSidebarCollapsed else {
            return
        }

        sidebarResizeStartWidth = nil

        withAnimation(.smooth(duration: 0.18)) {
            isSidebarCollapsed = true
        }
    }
}

private extension CGFloat {
    func clamped(to range: ClosedRange<CGFloat>) -> CGFloat {
        Swift.min(Swift.max(self, range.lowerBound), range.upperBound)
    }
}

#Preview {
    MomentoShellPreview()
        .frame(width: 1180, height: 760)
}

private struct MomentoShellPreview: View {
    @State private var sidebarSelection: MomentoSidebarItem.ID? = "all-assets"
    @State private var searchQuery = ""
    @State private var isCommandPalettePresented = false
    @State private var isInspectorPresented = true
    @State private var tags = ["UI", "Reference"]
    @State private var notes = "A calm shell foundation for the first milestone."

    var body: some View {
        MomentoShellView(
            sidebarSelection: $sidebarSelection,
            searchQuery: $searchQuery,
            isCommandPalettePresented: $isCommandPalettePresented,
            isInspectorPresented: $isInspectorPresented,
            inspectorAsset: MomentoInspectorAsset(
                id: "preview",
                title: "Desktop Reference",
                fileName: "desktop-reference.png",
                dimensions: "3024 × 1964",
                colorHexes: ["#2F80ED", "#27AE60", "#F2994A"],
                filePath: "~/Pictures/Momento/desktop-reference.png",
                fileSize: "3.2 MB",
                addedDate: .now,
                kind: "PNG Image"
            ),
            inspectorTags: $tags,
            inspectorNotes: $notes
        ) {
            ZStack {
                MomentoGlassBackground(cornerRadius: 0)
                Text(verbatim: "Grid bridge placeholder")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(MomentoTheme.secondaryText)
            }
        }
    }
}
