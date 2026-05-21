import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct ContentView: View {
    @State private var store = LibraryStore()
    @State private var isImporterPresented = false
    @State private var isCommandPalettePresented = false
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
        MomentoShellView(
            sidebarSelection: sidebarSelection,
            searchQuery: $store.searchQuery,
            isCommandPalettePresented: $isCommandPalettePresented,
            sidebarSections: sidebarSections,
            title: title,
            subtitle: "\(store.visibleAssets.count) items",
            inspectorAsset: store.selectedAsset.map(MomentoInspectorAsset.init(asset:)),
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
            .background(Color(nsColor: .windowBackgroundColor))
        }
        .overlay(alignment: .bottom) {
            if let importError {
                importErrorBanner(importError)
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
        .frame(minWidth: 1080, minHeight: 680)
    }

    private var title: String {
        switch store.sidebarSelection {
        case .library:
            "All Assets"
        case .favorites:
            "Favorites"
        case .tag(let tagID):
            store.tags.first { $0.id == tagID }?.name ?? "Tag"
        case .trash:
            "Trash"
        }
    }

    private var sidebarSections: [MomentoSidebarSection] {
        [
            MomentoSidebarSection(
                id: "library",
                title: "Library",
                items: [
                    MomentoSidebarItem(
                        id: "all-assets",
                        title: store.currentLibrary.name,
                        systemImage: "photo.on.rectangle.angled",
                        count: store.assets.count
                    ),
                    MomentoSidebarItem(
                        id: "recent",
                        title: "Recent",
                        systemImage: "clock",
                        count: store.assets.count
                    ),
                    MomentoSidebarItem(
                        id: "favorites",
                        title: "Favorites",
                        systemImage: "star",
                        count: store.assets.filter(\.isFavorite).count,
                        tint: .yellow
                    )
                ],
                isCollapsible: false
            ),
            MomentoSidebarSection(
                id: "folders",
                title: "Folders",
                items: [
                    MomentoSidebarItem(id: "folder-inspiration", title: "Inspiration", systemImage: "folder", count: 0),
                    MomentoSidebarItem(id: "folder-screenshots", title: "Screenshots", systemImage: "folder", count: 0)
                ]
            ),
            MomentoSidebarSection(
                id: "tags",
                title: "Tags",
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
                title: "Trash",
                items: [
                    MomentoSidebarItem(id: "trash", title: "Trash", systemImage: "trash", count: 0)
                ],
                isCollapsible: false
            )
        ]
    }

    private var commands: [MomentoCommand] {
        [
            MomentoCommand(id: "import", title: "Import Assets", subtitle: "Choose files or folders", systemImage: "square.and.arrow.down", shortcut: "I"),
            MomentoCommand(id: "view-masonry", title: "Masonry View", subtitle: "Show adaptive visual grid", systemImage: "rectangle.grid.2x2", shortcut: "1"),
            MomentoCommand(id: "view-grid", title: "Grid View", subtitle: "Show uniform thumbnails", systemImage: "square.grid.3x3", shortcut: "2"),
            MomentoCommand(id: "view-list", title: "List View", subtitle: "Show compact rows", systemImage: "list.bullet", shortcut: "3"),
            MomentoCommand(id: "quick-preview", title: "Quick Preview", subtitle: "Preview the selected asset", systemImage: "eye", shortcut: "Space")
        ]
    }

    private var emptyGridState: some View {
        VStack(spacing: 12) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 34, weight: .regular))
                .foregroundStyle(MomentoTheme.tertiaryText)

            Text("Drop assets here")
                .font(.system(size: 15, weight: .semibold))

            Text("Import images, GIFs, SVG, videos, PDF files, or folders to start building this library.")
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)

            Button {
                isImporterPresented = true
            } label: {
                Label("Import Assets", systemImage: "square.and.arrow.down")
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
        }
        .padding(28)
        .background {
            MomentoGlassBackground(material: .hudWindow, cornerRadius: 18)
        }
    }

    private func importErrorBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Image(systemName: "exclamationmark.triangle")
            Text(message)
                .lineLimit(2)
            Button("Dismiss") {
                withAnimation(.smooth(duration: 0.16)) {
                    importError = nil
                }
            }
            .buttonStyle(.plain)
        }
        .font(.system(size: 12, weight: .medium))
        .padding(.horizontal, 14)
        .frame(minHeight: 34)
        .background {
            MomentoGlassBackground(material: .hudWindow, cornerRadius: 11)
        }
    }

    private func selectAssets(_ ids: Set<AssetItem.ID>) {
        store.selectAsset(id: ids.first)
    }

    private func preview(_ asset: AssetItem) {
        let previewURL = FileManager.default.fileExists(atPath: asset.storageURL.path)
            ? asset.storageURL
            : asset.originalURL
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
        default:
            break
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
            importError = error.localizedDescription
        }
    }
}

private extension MomentoInspectorAsset {
    init(asset: AssetItem) {
        let previewURL = FileManager.default.fileExists(atPath: asset.storageURL.path)
            ? asset.storageURL
            : asset.originalURL

        self.init(
            id: asset.id,
            title: asset.displayName,
            fileName: previewURL.lastPathComponent,
            previewImage: asset.kind == .image || asset.kind == .gif ? NSImage(contentsOf: previewURL) : nil,
            dimensions: asset.dimensions.map { "\($0.width) × \($0.height)" },
            colorHexes: asset.tags.compactMap(\.colorHex),
            filePath: previewURL.path,
            fileSize: ByteCountFormatter.string(fromByteCount: asset.byteSize, countStyle: .file),
            addedDate: asset.importedAt,
            kind: asset.kind.rawValue.uppercased()
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
    ContentView()
}
