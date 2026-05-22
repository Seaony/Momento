import AppKit
import SwiftUI
import UniformTypeIdentifiers

private struct FolderCreationRequest: Identifiable {
    let id = UUID()
    var parentID: AssetFolder.ID?
}

struct ContentView: View {
    @Environment(\.appLocalization) private var localization
    @AppStorage(AppSettingsKeys.defaultViewMode) private var defaultViewModeRawValue = AssetViewMode.masonry.rawValue
    @Bindable var store: LibraryStore

    @State private var isImporterPresented = false
    @State private var isCommandPalettePresented = false
    @State private var isCreateLibraryDialogPresented = false
    @State private var editingLibrary: RecentLibraryReference?
    @State private var deletingLibrary: RecentLibraryReference?
    @State private var creatingFolder: FolderCreationRequest?
    @State private var editingFolder: AssetFolder?
    @State private var deletingFolder: AssetFolder?
    @State private var isInspectorPresented = false
    @State private var inspectorNotesByAssetID: [AssetItem.ID: String] = [:]
    @State private var importError: String?
    @State private var isToolbarSearchExpanded = false
    @State private var hoveredToolbarViewMode: AssetViewMode?
    @State private var shellToastRequest: MomentoToastRequest?
    @FocusState private var isToolbarSearchFocused: Bool

    private var sidebarSelection: Binding<String?> {
        Binding {
            store.sidebarItemID()
        } set: { id in
            store.selectSidebarItem(id: id)
        }
    }

    private var selectedAssetIDs: Set<AssetItem.ID> {
        if let selectedAssetID = store.selectedAssetID {
            return [selectedAssetID]
        }
        return []
    }

    private var selectedTags: Binding<[String]> {
        Binding {
            store.selectedAsset?.tags.map(\.name) ?? []
        } set: { names in
            store.updateSelectedTags(names)
        }
    }

    private var inspectorNotes: Binding<String> {
        Binding {
            guard let selectedAssetID = store.selectedAssetID else {
                return ""
            }
            return inspectorNotesByAssetID[selectedAssetID, default: ""]
        } set: { value in
            guard let selectedAssetID = store.selectedAssetID else {
                return
            }
            inspectorNotesByAssetID[selectedAssetID] = value
        }
    }

    var body: some View {
        Group {
            if store.currentLibrary == nil {
                MomentoLibraryWelcomeView(
                    onCreateLibrary: createLibrary,
                    onOpenLibrary: openLibrary
                )
                .navigationTitle("")
            } else {
                libraryBody
            }
        }
        .blur(radius: isModalOverlayVisible ? 8 : 0, opaque: false)
        .animation(.smooth(duration: 0.16), value: isModalOverlayVisible)
        .overlay(alignment: .bottom) {
            if let errorMessage {
                importErrorBanner(errorMessage)
                    .padding(.bottom, 16)
            }
        }
        .overlay {
            libraryDialogOverlay
        }
        .fileImporter(
            isPresented: $isImporterPresented,
            allowedContentTypes: [.image, .folder],
            allowsMultipleSelection: true,
            onCompletion: handleImportResult
        )
        .dropDestination(for: URL.self) { urls, _ in
            importURLs(urls)
            return true
        }
        .background {
            Button("") {
                withAnimation(.smooth(duration: 0.18)) {
                    isCommandPalettePresented = true
                }
            }
            .keyboardShortcut("k", modifiers: .command)
            .frame(width: 0, height: 0)
            .opacity(0)
        }
        .background {
            WindowTransparencyConfigurator()
        }
        .toolbarBackgroundVisibility(.hidden, for: .windowToolbar)
        .onAppear {
            validateCurrentLibraryAvailability()
        }
        .onChange(of: defaultViewMode) { _, newValue in
            store.setViewMode(newValue)
        }
        .frame(minWidth: MomentoTheme.mainWindowMinWidth, minHeight: MomentoTheme.mainWindowMinHeight)
    }

    private var libraryBody: some View {
        let visibleAssets = store.visibleAssets

        return MomentoShellView(
            sidebarSelection: sidebarSelection,
            searchQuery: $store.searchQuery,
            isCommandPalettePresented: $isCommandPalettePresented,
            isInspectorPresented: $isInspectorPresented,
            libraryName: store.currentLibrary?.name,
            currentLibraryID: store.currentLibrary?.id,
            recentLibraries: store.recentLibraries,
            folders: store.folders,
            sidebarCounts: sidebarAssetCounts,
            onCreateLibrary: createLibrary,
            onOpenLibrary: openLibrary,
            onSwitchLibrary: switchLibrary,
            onRenameLibrary: renameLibrary,
            onDeleteLibrary: deleteLibrary,
            onRevealLibrary: revealLibraryInFinder,
            onMoveLibrary: moveLibrary,
            onReloadLibrary: reloadLibrary,
            onCloseLibrary: closeLibrary,
            onImportAssets: { isImporterPresented = true },
            onCreateFolder: presentCreateFolderDialog,
            onRenameFolder: presentRenameFolderDialog,
            onDeleteFolder: presentDeleteFolderDialog,
            title: title,
            subtitle: localization.itemCount(visibleAssets.count),
            showsChromeControls: !isModalOverlayVisible,
            inspectorAsset: store.selectedAsset.map { MomentoInspectorAsset(asset: $0, localization: localization) },
            inspectorTags: selectedTags,
            inspectorNotes: inspectorNotes,
            toastRequest: $shellToastRequest,
            commands: commands,
            onCommandSelected: handleCommand
        ) {
            ZStack {
                AssetCollectionGridView(
                    assets: visibleAssets,
                    selectedAssetIDs: selectedAssetIDs,
                    viewMode: store.viewMode,
                    localization: localization,
                    onSelectionChange: selectAssets,
                    onDoubleClick: { asset in
                        preview(asset)
                    },
                    onSpacePreviewStart: previewWhileSpaceIsPressed,
                    onSpacePreviewEnd: endSpacePreview,
                    onContextMenuAction: handleAssetContextMenuAction
                )

                if visibleAssets.isEmpty {
                    emptyGridState
                }
            }
        }
        .toolbar {
            if !isModalOverlayVisible {
                ToolbarItem(placement: .confirmationAction) {
                    toolbarViewModeSwitcher
                        .padding(.trailing, 6)
                }
                .sharedBackgroundVisibility(.hidden)

                ToolbarItem(placement: .confirmationAction) {
                    toolbarSearchControl
                }
                .sharedBackgroundVisibility(.hidden)
            }
        }
        .navigationTitle("")
        .onChange(of: isToolbarSearchFocused) { _, isFocused in
            if !isFocused {
                collapseEmptyToolbarSearch()
            }
        }
        .onChange(of: store.searchQuery) { _, _ in
            collapseEmptyToolbarSearch()
        }
        .onChange(of: isModalOverlayVisible) { _, isVisible in
            if isVisible {
                collapseToolbarSearch(ignoresQuery: true)
            }
        }
    }

    private var toolbarViewModeSwitcher: some View {
        HStack(spacing: 2) {
            ForEach(AssetViewMode.allCases) { viewMode in
                let isSelected = store.viewMode == viewMode
                let isHovered = hoveredToolbarViewMode == viewMode

                Button {
                    store.setViewMode(viewMode)
                } label: {
                    Image(systemName: systemImage(for: viewMode))
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(isSelected ? Color.white : MomentoTheme.primaryText)
                        .frame(width: 34, height: 28)
                        .background {
                            toolbarSegmentBackground(isSelected: isSelected, isHovered: isHovered)
                        }
                        .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                }
                .buttonStyle(.plain)
                .pointerStyle(.link)
                .help(localization.title(for: viewMode))
                .accessibilityLabel(localization.title(for: viewMode))
                .onHover { hovering in
                    withAnimation(.smooth(duration: 0.14)) {
                        if hovering {
                            hoveredToolbarViewMode = viewMode
                        } else if hoveredToolbarViewMode == viewMode {
                            hoveredToolbarViewMode = nil
                        }
                    }
                }
            }
        }
        .padding(3)
        .frame(height: 34)
        .background {
            toolbarControlBackground(cornerRadius: 10)
        }
    }

    @ViewBuilder
    private var toolbarSearchControl: some View {
        if isToolbarSearchExpanded {
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: MomentoTheme.toolbarIconSize, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)

                TextField(localization.string("Search assets, tags, colors..."), text: $store.searchQuery)
                    .textFieldStyle(.plain)
                    .focused($isToolbarSearchFocused)
                    .frame(width: 260)
            }
            .padding(.horizontal, 11)
            .frame(height: MomentoTheme.toolbarControlHeight)
            .background {
                toolbarControlBackground(cornerRadius: MomentoTheme.toolbarControlRadius)
            }
            .fixedSize(horizontal: true, vertical: false)
            .layoutPriority(1)
            .onAppear {
                isToolbarSearchFocused = true
            }
        } else {
            Button {
                withAnimation(.smooth(duration: 0.16)) {
                    isToolbarSearchExpanded = true
                }
                isToolbarSearchFocused = true
            } label: {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: MomentoTheme.toolbarIconSize, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .frame(width: MomentoTheme.toolbarIconButtonWidth, height: MomentoTheme.toolbarControlHeight)
                    .background {
                        toolbarControlBackground(cornerRadius: MomentoTheme.toolbarControlRadius)
                    }
                    .contentShape(RoundedRectangle(cornerRadius: MomentoTheme.toolbarControlRadius, style: .continuous))
            }
            .buttonStyle(.plain)
            .pointerStyle(.link)
            .help(localization.string("Search assets, tags, colors..."))
        }
    }

    private func collapseEmptyToolbarSearch() {
        guard !isToolbarSearchFocused, store.searchQuery.isEmpty else {
            return
        }

        collapseToolbarSearch(ignoresQuery: true)
    }

    private func collapseToolbarSearch(ignoresQuery: Bool = false) {
        guard isToolbarSearchExpanded else {
            return
        }
        guard ignoresQuery || store.searchQuery.isEmpty else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            isToolbarSearchExpanded = false
            isToolbarSearchFocused = false
        }
    }

    @ViewBuilder
    private func toolbarSegmentBackground(isSelected: Bool, isHovered: Bool) -> some View {
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        if isSelected {
            shape.fill(Color.accentColor)
        } else if isHovered {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func toolbarControlBackground(cornerRadius: CGFloat) -> some View {
        MomentoGlassBackground(glass: .regular.interactive(true), cornerRadius: cornerRadius)
    }

    @ViewBuilder
    private var libraryDialogOverlay: some View {
        if isCreateLibraryDialogPresented {
            MomentoCreateLibraryDialog(
                mode: .create,
                isPresented: $isCreateLibraryDialogPresented,
                initialName: localization.string("Untitled Library"),
                onSubmit: chooseLibraryDestination
            )
            .zIndex(30)
        }

        if let editingLibrary {
            MomentoCreateLibraryDialog(
                mode: .edit,
                isPresented: editingLibraryDialogIsPresented,
                initialName: editingLibrary.name,
                onSubmit: { name in
                    renameLibrary(editingLibrary.id, to: name)
                }
            )
            .zIndex(31)
        }

        if let deletingLibrary {
            MomentoDeleteLibraryDialog(
                isPresented: deletingLibraryDialogIsPresented,
                libraryName: deletingLibrary.name,
                onConfirm: {
                    confirmDeleteLibrary(deletingLibrary.id)
                }
            )
            .zIndex(32)
        }

        if let creatingFolder {
            MomentoFolderNameDialog(
                mode: .create,
                isPresented: creatingFolderDialogIsPresented,
                initialName: localization.string("Untitled Folder"),
                onSubmit: { name in
                    createFolder(name: name, parentID: creatingFolder.parentID)
                }
            )
            .zIndex(33)
        }

        if let editingFolder {
            MomentoFolderNameDialog(
                mode: .edit,
                isPresented: editingFolderDialogIsPresented,
                initialName: editingFolder.name,
                onSubmit: { name in
                    renameFolder(editingFolder.id, to: name)
                }
            )
            .zIndex(34)
        }

        if let deletingFolder {
            MomentoDeleteFolderDialog(
                isPresented: deletingFolderDialogIsPresented,
                folderName: deletingFolder.name,
                onConfirm: {
                    deleteFolder(deletingFolder.id)
                }
            )
            .zIndex(35)
        }
    }

    private var editingLibraryDialogIsPresented: Binding<Bool> {
        Binding {
            editingLibrary != nil
        } set: { isPresented in
            if !isPresented {
                editingLibrary = nil
            }
        }
    }

    private var deletingLibraryDialogIsPresented: Binding<Bool> {
        Binding {
            deletingLibrary != nil
        } set: { isPresented in
            if !isPresented {
                deletingLibrary = nil
            }
        }
    }

    private var creatingFolderDialogIsPresented: Binding<Bool> {
        Binding {
            creatingFolder != nil
        } set: { isPresented in
            if !isPresented {
                creatingFolder = nil
            }
        }
    }

    private var editingFolderDialogIsPresented: Binding<Bool> {
        Binding {
            editingFolder != nil
        } set: { isPresented in
            if !isPresented {
                editingFolder = nil
            }
        }
    }

    private var deletingFolderDialogIsPresented: Binding<Bool> {
        Binding {
            deletingFolder != nil
        } set: { isPresented in
            if !isPresented {
                deletingFolder = nil
            }
        }
    }

    private var defaultViewMode: AssetViewMode {
        AssetViewMode(rawValue: defaultViewModeRawValue) ?? .masonry
    }

    private var isLibraryDialogVisible: Bool {
        isCreateLibraryDialogPresented || editingLibrary != nil || deletingLibrary != nil
    }

    private var isFolderDialogVisible: Bool {
        creatingFolder != nil || editingFolder != nil || deletingFolder != nil
    }

    private var isModalOverlayVisible: Bool {
        isLibraryDialogVisible || isFolderDialogVisible
    }

    private var errorMessage: String? {
        importError ?? store.libraryErrorMessage.map(localization.string)
    }

    private var title: String {
        switch store.sidebarSelection {
        case .library:
            store.currentLibrary?.name ?? localization.string("Library")
        case .favorites:
            localization.string("Favorites")
        case .uncategorized:
            localization.string("Uncategorized")
        case .untagged:
            localization.string("Untagged")
        case .tagManagement:
            localization.string("Tag Management")
        case .folderManagement:
            localization.string("Folder Management")
        case .folder(let folderID):
            store.folder(id: folderID)?.name ?? localization.string("Folder")
        case .tag(let tagID):
            store.tags.first { $0.id == tagID }?.name ?? localization.string("Tag")
        case .trash:
            localization.string("Trash")
        }
    }

    private var sidebarAssetCounts: MomentoSidebarAssetCounts {
        guard let currentLibrary = store.currentLibrary else {
            return .empty
        }

        let libraryAssets = store.assets.filter { $0.libraryID == currentLibrary.id }
        var folderCounts: [AssetFolder.ID: Int] = [:]

        for asset in libraryAssets {
            for folderID in asset.folderIDs {
                folderCounts[folderID, default: 0] += 1
            }
        }

        return MomentoSidebarAssetCounts(
            all: libraryAssets.count,
            favorites: libraryAssets.filter(\.isFavorite).count,
            uncategorized: libraryAssets.filter(\.folderIDs.isEmpty).count,
            untagged: libraryAssets.filter(\.tags.isEmpty).count,
            folders: folderCounts
        )
    }

    private var commands: [MomentoCommand] {
        [
            MomentoCommand(id: "import", title: localization.string("Import Assets"), subtitle: localization.string("Choose files or folders"), systemImage: "square.and.arrow.down", shortcut: "I"),
            MomentoCommand(id: "view-masonry", title: localization.string("Masonry View"), subtitle: localization.string("Show adaptive visual grid"), systemImage: "circle.hexagongrid", shortcut: "1"),
            MomentoCommand(id: "view-grid", title: localization.string("Grid View"), subtitle: localization.string("Show uniform thumbnails"), systemImage: "square.grid.2x2", shortcut: "2"),
            MomentoCommand(id: "view-list", title: localization.string("List View"), subtitle: localization.string("Show compact rows"), systemImage: "rectangle.grid.1x2", shortcut: "3"),
            MomentoCommand(id: "toggle-inspector", title: localization.string("Toggle Inspector"), subtitle: localization.string("Show or hide asset details"), systemImage: "sidebar.right", shortcut: "⌥⌘I"),
            MomentoCommand(id: "quick-preview", title: localization.string("Quick Preview"), subtitle: localization.string("Preview the selected asset"), systemImage: "eye", shortcut: "Space")
        ]
    }

    private var emptyGridState: some View {
        VStack(spacing: 12) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 34, weight: .regular))
                .foregroundStyle(MomentoTheme.tertiaryText)

            Text(localization.string("Drop assets here"))
                .font(.system(size: 15, weight: .semibold))

            Text(localization.string("Import images or GIFs to start building this library."))
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)

            Button {
                isImporterPresented = true
            } label: {
                Label(localization.string("Import Assets"), systemImage: "square.and.arrow.down")
            }
            .buttonStyle(.glassProminent)
            .controlSize(.large)
        }
        .padding(28)
        .background {
            MomentoGlassBackground(cornerRadius: 18)
        }
    }

    private func importErrorBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Image(systemName: "exclamationmark.triangle")
            Text(message)
                .lineLimit(2)
            Button(localization.string("Dismiss")) {
                withAnimation(.smooth(duration: 0.16)) {
                    importError = nil
                    store.libraryErrorMessage = nil
                }
            }
            .buttonStyle(.glass)
        }
        .font(.system(size: 12, weight: .medium))
        .padding(.horizontal, 14)
        .frame(minHeight: 34)
        .background {
            MomentoGlassBackground(cornerRadius: 11)
        }
    }

    private func selectAssets(_ ids: Set<AssetItem.ID>) {
        guard let id = ids.first else {
            return
        }

        store.selectAsset(id: id)

        if !isInspectorPresented {
            withAnimation(.smooth(duration: 0.22)) {
                isInspectorPresented = true
            }
        }
    }

    private func systemImage(for viewMode: AssetViewMode) -> String {
        switch viewMode {
        case .masonry:
            "circle.hexagongrid"
        case .grid:
            "square.grid.2x2"
        case .list:
            "rectangle.grid.1x2"
        }
    }

    @discardableResult
    private func preview(_ asset: AssetItem) -> Bool {
        showPreview(asset, closesOnSpaceKeyUp: false)
    }

    private func previewWhileSpaceIsPressed(_ asset: AssetItem, sourceFrame: NSRect?) {
        showPreview(asset, closesOnSpaceKeyUp: true, sourceFrame: sourceFrame)
    }

    private func endSpacePreview() {
        MomentoAssetPreviewPanelController.shared.close()
    }

    @discardableResult
    private func showPreview(_ asset: AssetItem, closesOnSpaceKeyUp: Bool, sourceFrame: NSRect? = nil) -> Bool {
        guard let previewURL = previewURL(for: asset) else {
            return false
        }

        MomentoAssetPreviewPanelController.shared.show(
            asset: asset,
            previewURL: previewURL,
            localization: localization,
            closesOnSpaceKeyUp: closesOnSpaceKeyUp,
            sourceFrame: sourceFrame
        )
        return true
    }

    private func previewURL(for asset: AssetItem) -> URL? {
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            return asset.storageURL
        }

        return asset.originalURL
    }

    private func handleCommand(_ command: MomentoCommand) {
        switch command.id {
        case "import":
            isImporterPresented = true
        case "view-masonry":
            store.setViewMode(.masonry)
        case "view-grid":
            store.setViewMode(.grid)
        case "view-list":
            store.setViewMode(.list)
        case "quick-preview":
            if let selectedAsset = store.selectedAsset {
                preview(selectedAsset)
            }
        case "toggle-inspector":
            withAnimation(.smooth(duration: 0.18)) {
                isInspectorPresented.toggle()
            }
        default:
            break
        }
    }

    private func handleAssetContextMenuAction(_ asset: AssetItem, action: AssetContextMenuAction) {
        switch action {
        case .previewOriginal:
            if preview(asset) {
                showAssetActionToast(action)
            }
        case .refreshThumbnail:
            if refreshThumbnail(for: asset) {
                showAssetActionToast(action)
            }
        case .reanalyzeColors:
            if reanalyzeColors(for: asset) {
                showAssetActionToast(action)
            }
        case .revealInFinder:
            if revealInFinder(asset) {
                showAssetActionToast(action)
            }
        case .moveToTrash:
            if moveAssetToTrash(asset) {
                showAssetActionToast(action)
            }
        }
    }

    private func refreshThumbnail(for asset: AssetItem) -> Bool {
        do {
            AssetCollectionGridView.invalidatePreviewCache(for: asset)
            if let updatedAsset = try store.refreshThumbnail(for: asset.id) {
                AssetCollectionGridView.invalidatePreviewCache(for: updatedAsset)
            }
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func reanalyzeColors(for asset: AssetItem) -> Bool {
        do {
            try store.reanalyzeColors(for: asset.id)
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func revealInFinder(_ asset: AssetItem) -> Bool {
        guard let fileURL = previewURL(for: asset) else {
            return false
        }

        NSWorkspace.shared.activateFileViewerSelecting([fileURL])
        return true
    }

    private func moveAssetToTrash(_ asset: AssetItem) -> Bool {
        do {
            AssetCollectionGridView.invalidatePreviewCache(for: asset)
            try store.moveAssetToTrash(id: asset.id)
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func showAssetActionToast(_ action: AssetContextMenuAction) {
        shellToastRequest = MomentoToastRequest(message: localization.string(action.titleKey))
    }

    private func createLibrary() {
        withAnimation(.smooth(duration: 0.16)) {
            isCreateLibraryDialogPresented = true
        }
    }

    private func chooseLibraryDestination(named libraryName: String) {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.canCreateDirectories = true
        panel.allowsMultipleSelection = false
        panel.prompt = localization.string("Create Library")

        guard panel.runModal() == .OK, let destinationURL = panel.url else {
            return
        }

        let packageURL = destinationURL.appendingPathComponent(libraryName, isDirectory: true)

        do {
            try store.createLibrary(at: packageURL)
        } catch {
            showImportError(error)
        }
    }

    private func openLibrary() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.momentoLibrary]
        panel.canChooseFiles = true
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false

        guard panel.runModal() == .OK, let url = panel.url else {
            return
        }

        do {
            try store.openLibrary(at: url)
        } catch {
            showImportError(error)
        }
    }

    private func switchLibrary(_ id: RecentLibraryReference.ID) {
        do {
            try store.openRecentLibrary(id: id)
        } catch {
            showImportError(error)
        }
    }

    private func renameLibrary(_ id: RecentLibraryReference.ID) {
        guard let library = store.recentLibraries.first(where: { $0.id == id }) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            editingLibrary = library
        }
    }

    private func renameLibrary(_ id: RecentLibraryReference.ID, to name: String) {
        do {
            try store.renameRecentLibrary(id: id, to: name)
        } catch {
            showImportError(error)
        }
    }

    private func deleteLibrary(_ id: RecentLibraryReference.ID) {
        guard let library = store.recentLibraries.first(where: { $0.id == id }) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            deletingLibrary = library
        }
    }

    private func revealLibraryInFinder(_ id: RecentLibraryReference.ID) {
        do {
            let url = try store.recentLibraryURL(id: id)
            let didStartAccessing = url.startAccessingSecurityScopedResource()
            defer {
                if didStartAccessing {
                    url.stopAccessingSecurityScopedResource()
                }
            }

            NSWorkspace.shared.activateFileViewerSelecting([url])
        } catch {
            showImportError(error)
        }
    }

    private func confirmDeleteLibrary(_ id: RecentLibraryReference.ID) {
        do {
            try store.deleteRecentLibrary(id: id)
        } catch {
            showImportError(error)
        }
    }

    private func moveLibrary(
        _ id: RecentLibraryReference.ID,
        relativeTo targetID: RecentLibraryReference.ID,
        insertAfterTarget: Bool
    ) {
        do {
            try store.moveRecentLibrary(id: id, relativeTo: targetID, insertAfterTarget: insertAfterTarget)
        } catch {
            showImportError(error)
        }
    }

    private func reloadLibrary() {
        do {
            try store.clearCachesAndReloadCurrentLibrary()
        } catch {
            showImportError(error)
        }
    }

    private func presentCreateFolderDialog(parentID: AssetFolder.ID?) {
        withAnimation(.smooth(duration: 0.16)) {
            creatingFolder = FolderCreationRequest(parentID: parentID)
        }
    }

    private func presentRenameFolderDialog(_ id: AssetFolder.ID) {
        guard let folder = store.folder(id: id) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            editingFolder = folder
        }
    }

    private func presentDeleteFolderDialog(_ id: AssetFolder.ID) {
        guard let folder = store.folder(id: id) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            deletingFolder = folder
        }
    }

    private func createFolder(name: String, parentID: AssetFolder.ID?) {
        do {
            try store.createFolder(name: name, parentID: parentID)
        } catch {
            showImportError(error)
        }
    }

    private func renameFolder(_ id: AssetFolder.ID, to name: String) {
        do {
            try store.renameFolder(id: id, to: name)
        } catch {
            showImportError(error)
        }
    }

    private func deleteFolder(_ id: AssetFolder.ID) {
        do {
            try store.deleteFolder(id: id)
        } catch {
            showImportError(error)
        }
    }

    private func closeLibrary() {
        withAnimation(.smooth(duration: 0.18)) {
            importError = nil
            store.closeCurrentLibrary()
        }
    }

    private func validateCurrentLibraryAvailability() {
        do {
            try store.validateCurrentLibraryAvailability()
        } catch {
            showImportError(error)
        }
    }

    private func handleImportResult(_ result: Result<[URL], Error>) {
        switch result {
        case .success(let urls):
            importURLs(urls)
        case .failure(let error):
            showImportError(error)
        }
    }

    private func importURLs(_ urls: [URL]) {
        guard !urls.isEmpty else {
            return
        }

        Task {
            do {
                try await store.importItems(from: urls)
            } catch {
                showImportError(error)
            }
        }
    }

    private func showImportError(_ error: Error) {
        withAnimation(.smooth(duration: 0.16)) {
            importError = localization.errorMessage(error)
        }
    }
}

private extension MomentoInspectorAsset {
    init(asset: AssetItem, localization: AppLocalization) {
        let fileURL: URL?
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            fileURL = asset.storageURL
        } else {
            fileURL = asset.originalURL
        }

        let previewURL: URL?
        if let thumbnailURL = asset.thumbnailURL, FileManager.default.fileExists(atPath: thumbnailURL.path) {
            previewURL = thumbnailURL
        } else {
            previewURL = fileURL
        }

        self.init(
            id: asset.id,
            title: asset.displayName,
            fileName: fileURL?.lastPathComponent ?? asset.displayName,
            previewImage: previewURL.flatMap { asset.kind == .image || asset.kind == .gif ? NSImage(contentsOf: $0) : nil },
            dimensions: asset.dimensions.map { "\($0.width) × \($0.height)" },
            colors: asset.paletteColors.map { color in
                MomentoInspectorColor(hex: color.hex, coverage: color.coverage)
            },
            filePath: fileURL?.path,
            fileSize: localization.fileSize(asset.byteSize),
            addedDate: asset.importedAt,
            kind: localization.kindTitle(for: asset.kind)
        )
    }
}

private extension Color {
    init?(hex: String?) {
        guard var normalized = hex?.trimmingCharacters(in: .whitespacesAndNewlines), !normalized.isEmpty else {
            return nil
        }

        if normalized.hasPrefix("#") {
            normalized.removeFirst()
        }

        guard normalized.count == 6, let value = UInt64(normalized, radix: 16) else {
            return nil
        }

        self.init(
            red: Double((value & 0xFF0000) >> 16) / 255,
            green: Double((value & 0x00FF00) >> 8) / 255,
            blue: Double(value & 0x0000FF) / 255
        )
    }
}

#Preview {
    ContentView(store: LibraryStore(libraries: [.defaultLibrary]))
        .environment(\.appLocalization, AppLocalization(language: .system))
}
