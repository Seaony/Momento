import SwiftUI

struct MomentoShellView<Content: View>: View {
    @Binding var sidebarSelection: MomentoSidebarItem.ID?
    @Binding var searchQuery: String
    @Binding var isCommandPalettePresented: Bool
    @Binding var isInspectorPresented: Bool

    var sidebarSections: [MomentoSidebarSection]
    var libraryName: String?
    var recentLibraries: [RecentLibraryReference]
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
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

    init(
        sidebarSelection: Binding<MomentoSidebarItem.ID?>,
        searchQuery: Binding<String>,
        isCommandPalettePresented: Binding<Bool>,
        isInspectorPresented: Binding<Bool> = .constant(true),
        sidebarSections: [MomentoSidebarSection] = .momentoDefaultSections,
        libraryName: String? = nil,
        recentLibraries: [RecentLibraryReference] = [],
        onCreateLibrary: @escaping () -> Void = {},
        onOpenLibrary: @escaping () -> Void = {},
        onSwitchLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
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
        self.recentLibraries = recentLibraries
        self.onCreateLibrary = onCreateLibrary
        self.onOpenLibrary = onOpenLibrary
        self.onSwitchLibrary = onSwitchLibrary
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
        HStack(spacing: 0) {
            floatingSidebar

            VStack(spacing: 0) {
                content()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
            .frame(minWidth: MomentoTheme.contentMinWidth, maxWidth: .infinity, maxHeight: .infinity)

            if isInspectorPresented {
                trailingInspector
            }
        }
        .animation(.smooth(duration: 0.18), value: isInspectorPresented)
        .momentoCommandPalette(
            isPresented: $isCommandPalettePresented,
            commands: commands,
            onSelect: onCommandSelected
        )
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
    }

    private var floatingSidebar: some View {
        MomentoSidebarView(
            sections: sidebarSections,
            selection: $sidebarSelection,
            libraryName: libraryName,
            recentLibraries: recentLibraries,
            onCreateLibrary: onCreateLibrary,
            onOpenLibrary: onOpenLibrary,
            onSwitchLibrary: onSwitchLibrary,
            onCloseLibrary: onCloseLibrary
        )
        .frame(width: sidebarWidth)
        .overlay(alignment: .trailing) {
            sidebarResizeHandle
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

    private var sidebarResizeHandle: some View {
        Rectangle()
            .fill(Color.clear)
            .frame(width: 14)
            .contentShape(Rectangle())
            .pointerStyle(.columnResize(directions: .all))
            .gesture(
                DragGesture(minimumDistance: 0, coordinateSpace: .global)
                    .onChanged { value in
                        let startWidth = sidebarResizeStartWidth ?? sidebarWidth
                        sidebarResizeStartWidth = startWidth
                        sidebarWidth = (startWidth + value.translation.width)
                            .clamped(to: MomentoTheme.sidebarMinWidth...MomentoTheme.sidebarMaxWidth)
                    }
                    .onEnded { _ in
                        sidebarResizeStartWidth = nil
                    }
            )
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
                Text("Grid bridge placeholder")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(MomentoTheme.secondaryText)
            }
        }
    }
}
