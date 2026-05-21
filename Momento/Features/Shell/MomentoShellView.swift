import SwiftUI

struct MomentoShellView<Content: View>: View {
    @Environment(\.appLocalization) private var localization

    @Binding var sidebarSelection: String?
    @Binding var searchQuery: String
    @Binding var isCommandPalettePresented: Bool
    @Binding var isInspectorPresented: Bool

    var libraryName: String?
    var currentLibraryID: RecentLibraryReference.ID?
    var recentLibraries: [RecentLibraryReference]
    var folders: [AssetFolder]
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onRenameLibrary: (RecentLibraryReference.ID) -> Void
    var onDeleteLibrary: (RecentLibraryReference.ID) -> Void
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void
    var onReloadLibrary: () -> Void
    var onCloseLibrary: () -> Void
    var onCreateFolder: (AssetFolder.ID?) -> Void
    var onDeleteFolder: (AssetFolder.ID) -> Void
    var title: String
    var subtitle: String?
    var showsChromeControls: Bool
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
        sidebarSelection: Binding<String?>,
        searchQuery: Binding<String>,
        isCommandPalettePresented: Binding<Bool>,
        isInspectorPresented: Binding<Bool> = .constant(true),
        libraryName: String? = nil,
        currentLibraryID: RecentLibraryReference.ID? = nil,
        recentLibraries: [RecentLibraryReference] = [],
        folders: [AssetFolder] = [],
        onCreateLibrary: @escaping () -> Void = {},
        onOpenLibrary: @escaping () -> Void = {},
        onSwitchLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onRenameLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onDeleteLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onMoveLibrary: @escaping (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void = { _, _, _ in },
        onReloadLibrary: @escaping () -> Void = {},
        onCloseLibrary: @escaping () -> Void = {},
        onCreateFolder: @escaping (AssetFolder.ID?) -> Void = { _ in },
        onDeleteFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        title: String = "All Assets",
        subtitle: String? = "0 items",
        showsChromeControls: Bool = true,
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
        self.libraryName = libraryName
        self.currentLibraryID = currentLibraryID
        self.recentLibraries = recentLibraries
        self.folders = folders
        self.onCreateLibrary = onCreateLibrary
        self.onOpenLibrary = onOpenLibrary
        self.onSwitchLibrary = onSwitchLibrary
        self.onRenameLibrary = onRenameLibrary
        self.onDeleteLibrary = onDeleteLibrary
        self.onMoveLibrary = onMoveLibrary
        self.onReloadLibrary = onReloadLibrary
        self.onCloseLibrary = onCloseLibrary
        self.onCreateFolder = onCreateFolder
        self.onDeleteFolder = onDeleteFolder
        self.title = title
        self.subtitle = subtitle
        self.showsChromeControls = showsChromeControls
        self.inspectorAsset = inspectorAsset
        self._inspectorTags = inspectorTags
        self._inspectorNotes = inspectorNotes
        self.commands = commands
        self.onCommandSelected = onCommandSelected
        self.content = content
    }

    var body: some View {
        shellContent
    }

    private var shellContent: some View {
        HStack(spacing: 0) {
            if !isSidebarCollapsed {
                floatingSidebar
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
        .animation(.spring(response: 0.28, dampingFraction: 0.88), value: isInspectorPresented)
        .animation(.smooth(duration: 0.18), value: isSidebarCollapsed)
        .momentoCommandPalette(
            isPresented: $isCommandPalettePresented,
            commands: commands,
            onSelect: onCommandSelected
        )
        .background {
            SidebarTitlebarToggleConfigurator(
                isCollapsed: $isSidebarCollapsed,
                isVisible: showsChromeControls,
                buttonMinX: sidebarToggleButtonMinX,
                label: sidebarToggleLabel
            )
            .frame(width: 0, height: 0)
        }
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
    }

    private var floatingSidebar: some View {
        MomentoSidebarView(
            selection: $sidebarSelection,
            libraryName: libraryName,
            currentLibraryID: currentLibraryID,
            recentLibraries: recentLibraries,
            folders: folders,
            onCreateLibrary: onCreateLibrary,
            onOpenLibrary: onOpenLibrary,
            onSwitchLibrary: onSwitchLibrary,
            onRenameLibrary: onRenameLibrary,
            onDeleteLibrary: onDeleteLibrary,
            onMoveLibrary: onMoveLibrary,
            onReloadLibrary: onReloadLibrary,
            onCloseLibrary: onCloseLibrary,
            onCreateFolder: onCreateFolder,
            onDeleteFolder: onDeleteFolder
        )
        .frame(width: effectiveSidebarWidth)
        .fixedSize(horizontal: true, vertical: false)
        .layoutPriority(2)
        .overlay(alignment: .trailing) {
            sidebarResizeHandle()
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
        .fixedSize(horizontal: true, vertical: false)
        .layoutPriority(1)
        .ignoresSafeArea(.container, edges: .top)
        .transition(
            .asymmetric(
                insertion: .move(edge: .trailing).combined(with: .opacity),
                removal: .move(edge: .trailing).combined(with: .opacity)
            )
        )
        .zIndex(1)
    }

    private func sidebarResizeHandle() -> some View {
        Rectangle()
            .fill(Color.clear)
            .frame(width: 14)
            .contentShape(Rectangle())
            .pointerStyle(.columnResize(directions: .all))
            .gesture(
                DragGesture(minimumDistance: 0, coordinateSpace: .global)
                    .onChanged { value in
                        let startWidth = sidebarResizeStartWidth ?? effectiveSidebarWidth
                        let widthRange = sidebarWidthRange
                        let proposedWidth = startWidth + value.translation.width

                        sidebarResizeStartWidth = startWidth

                        sidebarWidth = proposedWidth.clamped(to: widthRange)
                    }
                    .onEnded { _ in
                        sidebarResizeStartWidth = nil
                    }
            )
    }

    private var effectiveSidebarWidth: CGFloat {
        sidebarWidth.clamped(to: sidebarWidthRange)
    }

    private var sidebarWidthRange: ClosedRange<CGFloat> {
        MomentoTheme.sidebarMinWidth...MomentoTheme.sidebarMaxWidth
    }

    private var sidebarToggleLabel: String {
        localization.string(isSidebarCollapsed ? "Expand Sidebar" : "Collapse Sidebar")
    }

    private var sidebarToggleButtonMinX: CGFloat {
        if isSidebarCollapsed {
            return MomentoTheme.collapsedSidebarToggleLeadingInset
        }

        return MomentoTheme.floatingSidebarInset + effectiveSidebarWidth - MomentoTheme.sidebarTitlebarButtonTrailingInset - MomentoTheme.sidebarTitlebarButtonSize
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
    @State private var sidebarSelection: String? = "all-assets"
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
