import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct ContentView: View {
    @Environment(\.appLocalization) private var localization
    @AppStorage(AppSettingsKeys.defaultViewMode) private var defaultViewModeRawValue = AssetViewMode.masonry.rawValue
    @Bindable var store: LibraryStore

    @State private var isImporterPresented = false
    @State private var isCommandPalettePresented = false
    @State private var isInspectorPresented = true
    @State private var inspectorNotesByAssetID: [AssetItem.ID: String] = [:]
    @State private var importError: String?

    private var sidebarSelection: Binding<MomentoSidebarItem.ID?> {
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

    private var viewModeSelection: Binding<AssetViewMode> {
        Binding {
            store.viewMode
        } set: { mode in
            store.setViewMode(mode)
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
        .overlay(alignment: .bottom) {
            if let errorMessage {
                importErrorBanner(errorMessage)
                    .padding(.bottom, 16)
            }
        }
        .fileImporter(
            isPresented: $isImporterPresented,
            allowedContentTypes: [.item],
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
        .onChange(of: defaultViewMode) { _, newValue in
            store.setViewMode(newValue)
        }
        .frame(minWidth: 540, minHeight: 340)
    }

    private var libraryBody: some View {
        MomentoShellView(
            sidebarSelection: sidebarSelection,
            searchQuery: $store.searchQuery,
            isCommandPalettePresented: $isCommandPalettePresented,
            isInspectorPresented: $isInspectorPresented,
            sidebarSections: sidebarSections,
            libraryName: store.currentLibrary?.name,
            recentLibraries: store.recentLibraries,
            onCreateLibrary: createLibrary,
            onOpenLibrary: openLibrary,
            onSwitchLibrary: switchLibrary,
            onCloseLibrary: closeLibrary,
            title: title,
            subtitle: localization.itemCount(store.visibleAssets.count),
            inspectorAsset: store.selectedAsset.map { MomentoInspectorAsset(asset: $0, localization: localization) },
            inspectorTags: selectedTags,
            inspectorNotes: inspectorNotes,
            commands: commands,
            onCommandSelected: handleCommand
        ) {
            ZStack {
                AssetCollectionGridView(
                    assets: store.visibleAssets,
                    selectedAssetIDs: selectedAssetIDs,
                    viewMode: store.viewMode,
                    onSelectionChange: selectAssets,
                    onDoubleClick: preview
                )

                if store.visibleAssets.isEmpty {
                    emptyGridState
                }
            }
        }
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                Picker(localization.string("View"), selection: viewModeSelection) {
                    ForEach(AssetViewMode.allCases) { viewMode in
                        Text(localization.title(for: viewMode))
                            .tag(viewMode)
                    }
                }
                .labelsHidden()
                .pickerStyle(.segmented)

                Toggle(isOn: $isInspectorPresented) {
                    Label(localization.string("Toggle Inspector"), systemImage: "sidebar.right")
                }
                .toggleStyle(.button)
                .help(localization.string("Toggle Inspector"))

                Button {
                    withAnimation(.smooth(duration: 0.18)) {
                        isCommandPalettePresented = true
                    }
                } label: {
                    Label(localization.string("Open command palette"), systemImage: "command")
                }
                .help(localization.string("Open command palette"))
            }
        }
        .searchable(text: $store.searchQuery, placement: .toolbar, prompt: Text(localization.string("Search assets, tags, colors...")))
        .navigationTitle("")
    }

    private var defaultViewMode: AssetViewMode {
        AssetViewMode(rawValue: defaultViewModeRawValue) ?? .masonry
    }

    private var errorMessage: String? {
        importError ?? store.libraryErrorMessage
    }

    private var title: String {
        switch store.sidebarSelection {
        case .library:
            store.currentLibrary?.name ?? localization.string("Library")
        case .favorites:
            localization.string("Favorites")
        case .tag(let tagID):
            store.tags.first { $0.id == tagID }?.name ?? localization.string("Tag")
        case .trash:
            localization.string("Trash")
        }
    }

    private var sidebarSections: [MomentoSidebarSection] {
        [
            MomentoSidebarSection(
                id: "favorites",
                title: localization.string("Favorites"),
                items: [
                    MomentoSidebarItem(
                        id: "favorites",
                        title: localization.string("Favorites"),
                        systemImage: "star",
                        count: store.assets.filter(\.isFavorite).count,
                        tint: .yellow
                    )
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
                items: store.tags.map { tag in
                    MomentoSidebarItem(
                        id: "tag-\(tag.id)",
                        title: tag.name,
                        systemImage: "tag",
                        count: store.assets.filter { $0.tags.contains(tag) }.count,
                        tint: Color(hex: tag.colorHex)
                    )
                }
            ),
            MomentoSidebarSection(
                id: "trash",
                title: localization.string("Trash"),
                items: [
                    MomentoSidebarItem(id: "trash", title: localization.string("Trash"), systemImage: "trash", count: 0)
                ],
                isCollapsible: false
            )
        ]
    }

    private var commands: [MomentoCommand] {
        [
            MomentoCommand(id: "import", title: localization.string("Import Assets"), subtitle: localization.string("Choose files or folders"), systemImage: "square.and.arrow.down", shortcut: "I"),
            MomentoCommand(id: "view-masonry", title: localization.string("Masonry View"), subtitle: localization.string("Show adaptive visual grid"), systemImage: "rectangle.grid.2x2", shortcut: "1"),
            MomentoCommand(id: "view-grid", title: localization.string("Grid View"), subtitle: localization.string("Show uniform thumbnails"), systemImage: "square.grid.3x3", shortcut: "2"),
            MomentoCommand(id: "view-list", title: localization.string("List View"), subtitle: localization.string("Show compact rows"), systemImage: "list.bullet", shortcut: "3"),
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

            Text(localization.string("Import images, GIFs, SVG, videos, PDF files, or folders to start building this library."))
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
        store.selectAsset(id: ids.first)
    }

    private func preview(_ asset: AssetItem) {
        let previewURL: URL?
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            previewURL = asset.storageURL
        } else {
            previewURL = asset.originalURL
        }

        guard let previewURL else {
            return
        }

        QuickLookPreviewController.shared.show(url: previewURL)
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

    private func createLibrary() {
        let panel = NSSavePanel()
        panel.allowedContentTypes = [.momentoLibrary]
        panel.canCreateDirectories = true
        panel.nameFieldStringValue = localization.string("Untitled Library") + ".\(LibraryStorage.packageExtension)"

        guard panel.runModal() == .OK, let url = panel.url else {
            return
        }

        do {
            try store.createLibrary(at: url)
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

    private func closeLibrary() {
        withAnimation(.smooth(duration: 0.18)) {
            importError = nil
            store.closeCurrentLibrary()
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
        let previewURL: URL?
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            previewURL = asset.storageURL
        } else {
            previewURL = asset.originalURL
        }

        self.init(
            id: asset.id,
            title: asset.displayName,
            fileName: previewURL?.lastPathComponent ?? asset.displayName,
            previewImage: previewURL.flatMap { asset.kind == .image || asset.kind == .gif ? NSImage(contentsOf: $0) : nil },
            dimensions: asset.dimensions.map { "\($0.width) × \($0.height)" },
            colorHexes: asset.tags.compactMap(\.colorHex),
            filePath: previewURL?.path,
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
