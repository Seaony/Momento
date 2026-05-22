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
    var sidebarCounts: MomentoSidebarAssetCounts
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onRenameLibrary: (RecentLibraryReference.ID) -> Void
    var onDeleteLibrary: (RecentLibraryReference.ID) -> Void
    var onRevealLibrary: (RecentLibraryReference.ID) -> Void
    var onMoveLibrary: (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void
    var onReloadLibrary: () -> Void
    var onCloseLibrary: () -> Void
    var onImportAssets: () -> Void
    var onCreateFolder: (AssetFolder.ID?) -> Void
    var onRenameFolder: (AssetFolder.ID) -> Void
    var onDeleteFolder: (AssetFolder.ID) -> Void
    var title: String
    var subtitle: String?
    var showsChromeControls: Bool
    var inspectorAsset: MomentoInspectorAsset?
    @Binding var inspectorTags: [String]
    var inspectorAvailableTags: [String]
    @Binding var inspectorFolderIDs: [AssetFolder.ID]
    var inspectorFolders: [AssetFolder]
    @Binding var inspectorNotes: String
    @Binding var toastRequest: MomentoToastRequest?
    var onRenameInspectorAsset: (MomentoInspectorAsset.ID, String) -> Void
    var commands: [MomentoCommand]
    var onCommandSelected: (MomentoCommand) -> Void
    var content: () -> Content

    @State private var sidebarWidth = MomentoTheme.sidebarWidth
    @State private var sidebarResizeStartWidth: CGFloat?
    @State private var isSidebarCollapsed = false
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
        sidebarCounts: MomentoSidebarAssetCounts = .empty,
        onCreateLibrary: @escaping () -> Void = {},
        onOpenLibrary: @escaping () -> Void = {},
        onSwitchLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onRenameLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onDeleteLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onRevealLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onMoveLibrary: @escaping (RecentLibraryReference.ID, RecentLibraryReference.ID, Bool) -> Void = { _, _, _ in },
        onReloadLibrary: @escaping () -> Void = {},
        onCloseLibrary: @escaping () -> Void = {},
        onImportAssets: @escaping () -> Void = {},
        onCreateFolder: @escaping (AssetFolder.ID?) -> Void = { _ in },
        onRenameFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        onDeleteFolder: @escaping (AssetFolder.ID) -> Void = { _ in },
        title: String = "All Assets",
        subtitle: String? = "0 items",
        showsChromeControls: Bool = true,
        inspectorAsset: MomentoInspectorAsset? = nil,
        inspectorTags: Binding<[String]> = .constant([]),
        inspectorAvailableTags: [String] = [],
        inspectorFolderIDs: Binding<[AssetFolder.ID]> = .constant([]),
        inspectorFolders: [AssetFolder] = [],
        inspectorNotes: Binding<String> = .constant(""),
        toastRequest: Binding<MomentoToastRequest?> = .constant(nil),
        onRenameInspectorAsset: @escaping (MomentoInspectorAsset.ID, String) -> Void = { _, _ in },
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
        self.sidebarCounts = sidebarCounts
        self.onCreateLibrary = onCreateLibrary
        self.onOpenLibrary = onOpenLibrary
        self.onSwitchLibrary = onSwitchLibrary
        self.onRenameLibrary = onRenameLibrary
        self.onDeleteLibrary = onDeleteLibrary
        self.onRevealLibrary = onRevealLibrary
        self.onMoveLibrary = onMoveLibrary
        self.onReloadLibrary = onReloadLibrary
        self.onCloseLibrary = onCloseLibrary
        self.onImportAssets = onImportAssets
        self.onCreateFolder = onCreateFolder
        self.onRenameFolder = onRenameFolder
        self.onDeleteFolder = onDeleteFolder
        self.title = title
        self.subtitle = subtitle
        self.showsChromeControls = showsChromeControls
        self.inspectorAsset = inspectorAsset
        self._inspectorTags = inspectorTags
        self.inspectorAvailableTags = inspectorAvailableTags
        self._inspectorFolderIDs = inspectorFolderIDs
        self.inspectorFolders = inspectorFolders
        self._inspectorNotes = inspectorNotes
        self._toastRequest = toastRequest
        self.onRenameInspectorAsset = onRenameInspectorAsset
        self.commands = commands
        self.onCommandSelected = onCommandSelected
        self.content = content
    }

    var body: some View {
        shellContent
    }

    private var shellContent: some View {
        HStack(alignment: .top, spacing: 0) {
            if !isSidebarCollapsed {
                floatingSidebar
                    .transition(.move(edge: .leading).combined(with: .opacity))
            }

            VStack(spacing: 0) {
                content()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
            .padding(.horizontal, MomentoTheme.contentSidebarGap)
            .frame(maxWidth: .infinity, maxHeight: .infinity)

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
                importButtonMinX: titlebarImportButtonMinX,
                label: sidebarToggleLabel,
                importAction: onImportAssets,
                importLabel: localization.string("Import Assets")
            )
            .frame(width: 0, height: 0)
        }
        .background {
            InspectorTitlebarSpacerConfigurator(
                isInspectorPresented: $isInspectorPresented,
                isVisible: showsChromeControls,
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
            counts: sidebarCounts,
            onCreateLibrary: onCreateLibrary,
            onOpenLibrary: onOpenLibrary,
            onSwitchLibrary: onSwitchLibrary,
            onRenameLibrary: onRenameLibrary,
            onDeleteLibrary: onDeleteLibrary,
            onRevealLibrary: onRevealLibrary,
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
            availableTags: inspectorAvailableTags,
            folderIDs: $inspectorFolderIDs,
            folders: inspectorFolders,
            notes: $inspectorNotes,
            onTitleCommit: onRenameInspectorAsset,
            onColorCopied: showColorCopyToast
        )
        .frame(width: MomentoTheme.inspectorWidth)
        .frame(maxHeight: .infinity, alignment: .top)
        .fixedSize(horizontal: true, vertical: false)
        .layoutPriority(1)
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

    private var titlebarImportButtonMinX: CGFloat {
        let contentStartX = if isSidebarCollapsed {
            MomentoTheme.contentSidebarGap
        } else {
            MomentoTheme.floatingSidebarInset * 2 + effectiveSidebarWidth + MomentoTheme.contentSidebarGap
        }
        let minimumAfterSidebarToggle = sidebarToggleButtonMinX + MomentoTheme.sidebarTitlebarButtonSize + 10

        return max(contentStartX, minimumAfterSidebarToggle)
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
