import SwiftUI

struct MomentoToastRequest: Equatable {
    let id: UUID
    var message: String

    init(message: String) {
        id = UUID()
        self.message = message
    }
}

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
    var onRenameFolder: (AssetFolder.ID) -> Void
    var onDeleteFolder: (AssetFolder.ID) -> Void
    var title: String
    var subtitle: String?
    var showsChromeControls: Bool
    var inspectorAsset: MomentoInspectorAsset?
    @Binding var inspectorTags: [String]
    @Binding var inspectorNotes: String
    @Binding var toastRequest: MomentoToastRequest?
    var commands: [MomentoCommand]
    var onCommandSelected: (MomentoCommand) -> Void
    var content: () -> Content

    @State private var sidebarWidth = MomentoTheme.sidebarWidth
    @State private var sidebarResizeStartWidth: CGFloat?
    @State private var isSidebarCollapsed = false
    @State private var inspectorWidth = MomentoTheme.inspectorWidth
    @State private var inspectorResizeStartWidth: CGFloat?
    @State private var isInspectorHovered = false
    @State private var isInspectorResizeHandleHovered = false
    @State private var activeToastMessage: String?
    @State private var isToastVisible = false
    @State private var toastToken = UUID()

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
        onRenameFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        onDeleteFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        title: String = "All Assets",
        subtitle: String? = "0 items",
        showsChromeControls: Bool = true,
        inspectorAsset: MomentoInspectorAsset? = nil,
        inspectorTags: Binding<[String]> = .constant([]),
        inspectorNotes: Binding<String> = .constant(""),
        toastRequest: Binding<MomentoToastRequest?> = .constant(nil),
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
        self.onRenameFolder = onRenameFolder
        self.onDeleteFolder = onDeleteFolder
        self.title = title
        self.subtitle = subtitle
        self.showsChromeControls = showsChromeControls
        self.inspectorAsset = inspectorAsset
        self._inspectorTags = inspectorTags
        self._inspectorNotes = inspectorNotes
        self._toastRequest = toastRequest
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
            .padding(.horizontal, MomentoTheme.contentSidebarGap)
            .frame(
                minWidth: MomentoTheme.contentMinWidth + MomentoTheme.contentSidebarGap * 2,
                maxWidth: .infinity,
                maxHeight: .infinity
            )

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
        .overlay {
            shellToast
        }
        .animation(.smooth(duration: 0.18), value: isToastVisible)
        .onChange(of: toastRequest) { _, request in
            guard let request else {
                return
            }

            showToast(request.message)
        }
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
            InspectorTitlebarSpacerConfigurator(
                isInspectorPresented: $isInspectorPresented,
                isVisible: showsChromeControls,
                inspectorWidth: effectiveInspectorWidth,
                label: localization.string("Toggle Inspector")
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
            onRenameFolder: onRenameFolder,
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
            notes: $inspectorNotes,
            onColorCopied: showColorCopyToast
        )
        .frame(width: effectiveInspectorWidth)
        .fixedSize(horizontal: true, vertical: false)
        .layoutPriority(1)
        .overlay(alignment: .leading) {
            inspectorResizeHandle()
        }
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                isInspectorHovered = hovering
            }
        }
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

    private func inspectorResizeHandle() -> some View {
        VStack {
            Spacer()

            Capsule()
                .fill(inspectorResizeHandleColor)
                .frame(width: 7, height: 76)
                .opacity(isInspectorResizeHandleVisible ? 1 : 0)
                .contentShape(Capsule())
                .pointerStyle(.columnResize(directions: .all))
                .onHover { hovering in
                    withAnimation(.smooth(duration: 0.14)) {
                        isInspectorResizeHandleHovered = hovering
                    }
                }
                .gesture(
                    DragGesture(minimumDistance: 0, coordinateSpace: .global)
                        .onChanged { value in
                            let startWidth = inspectorResizeStartWidth ?? effectiveInspectorWidth
                            let widthRange = inspectorWidthRange
                            let proposedWidth = startWidth - value.translation.width

                            inspectorResizeStartWidth = startWidth

                            inspectorWidth = proposedWidth.clamped(to: widthRange)
                        }
                        .onEnded { _ in
                            inspectorResizeStartWidth = nil
                        }
                )

            Spacer()
        }
        .frame(width: 7)
        .frame(maxHeight: .infinity)
    }

    private var effectiveSidebarWidth: CGFloat {
        sidebarWidth.clamped(to: sidebarWidthRange)
    }

    private var sidebarWidthRange: ClosedRange<CGFloat> {
        MomentoTheme.sidebarMinWidth...MomentoTheme.sidebarMaxWidth
    }

    private var effectiveInspectorWidth: CGFloat {
        inspectorWidth.clamped(to: inspectorWidthRange)
    }

    private var inspectorWidthRange: ClosedRange<CGFloat> {
        MomentoTheme.inspectorMinWidth...MomentoTheme.inspectorMaxWidth
    }

    private var isInspectorResizeHandleVisible: Bool {
        isInspectorHovered || isInspectorResizeHandleHovered || inspectorResizeStartWidth != nil
    }

    private var inspectorResizeHandleColor: Color {
        if isInspectorResizeHandleHovered || inspectorResizeStartWidth != nil {
            return .white.opacity(0.48)
        }

        return .white.opacity(0.24)
    }

    @ViewBuilder
    private var shellToast: some View {
        if isToastVisible, let activeToastMessage {
            Label(activeToastMessage, systemImage: "checkmark")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(.white)
                .labelStyle(.titleAndIcon)
                .padding(.horizontal, 16)
                .frame(height: 42)
                .background {
                    MomentoGlassBackground(
                        glass: .regular.tint(Color.black.opacity(0.36)),
                        cornerRadius: 14
                    )
                    .overlay {
                        RoundedRectangle(cornerRadius: 14, style: .continuous)
                            .strokeBorder(Color.white.opacity(0.16), lineWidth: 1)
                    }
                }
                .shadow(color: Color.black.opacity(0.32), radius: 18, y: 10)
                .transition(.opacity.combined(with: .scale(scale: 0.96)))
                .allowsHitTesting(false)
                .zIndex(10)
        }
    }

    private func showColorCopyToast() {
        showToast(localization.string("Color copied"))
    }

    private func showToast(_ message: String) {
        let token = UUID()
        activeToastMessage = message
        toastToken = token

        withAnimation(.smooth(duration: 0.18)) {
            isToastVisible = true
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 1.45) {
            guard toastToken == token else {
                return
            }

            withAnimation(.smooth(duration: 0.18)) {
                isToastVisible = false
            }
        }
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
