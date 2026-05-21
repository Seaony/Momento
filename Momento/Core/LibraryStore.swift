import Foundation
import Observation

@MainActor
@Observable
final class LibraryStore {
    var libraries: [AssetLibrary]
    var currentLibrary: AssetLibrary
    var assets: [AssetItem]
    var selectedAssetID: AssetItem.ID?
    var searchQuery = ""
    var viewMode: AssetViewMode = .masonry
    var sidebarSelection: SidebarSelection

    private let importService: AssetImportService

    init(
        libraries: [AssetLibrary] = [.defaultLibrary],
        assets: [AssetItem]? = nil,
        defaultViewMode: AssetViewMode = .masonry,
        importService: AssetImportService = AssetImportService()
    ) {
        let currentLibrary = libraries.first ?? .defaultLibrary
        let initialAssets = assets ?? Self.sampleAssets(for: currentLibrary)
        self.libraries = libraries.isEmpty ? [.defaultLibrary] : libraries
        self.currentLibrary = currentLibrary
        self.assets = initialAssets
        self.selectedAssetID = initialAssets.first?.id
        self.viewMode = defaultViewMode
        self.sidebarSelection = .library(currentLibrary.id)
        self.importService = importService
    }

    var visibleAssets: [AssetItem] {
        let libraryAssets = assets.filter { $0.libraryID == currentLibrary.id }
        let scopedAssets: [AssetItem]

        switch sidebarSelection {
        case .library:
            scopedAssets = libraryAssets
        case .favorites:
            scopedAssets = libraryAssets.filter(\.isFavorite)
        case .tag(let tagID):
            scopedAssets = libraryAssets.filter { asset in
                asset.tags.contains { $0.id == tagID }
            }
        case .trash:
            scopedAssets = []
        }

        let query = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !query.isEmpty else {
            return scopedAssets
        }

        return scopedAssets.filter { asset in
            asset.displayName.localizedCaseInsensitiveContains(query)
                || asset.fileExtension.localizedCaseInsensitiveContains(query)
                || asset.tags.contains { $0.name.localizedCaseInsensitiveContains(query) }
        }
    }

    var selectedAsset: AssetItem? {
        guard let selectedAssetID else {
            return nil
        }
        return assets.first { $0.id == selectedAssetID }
    }

    var tags: [TagItem] {
        var seen = Set<String>()
        return assets
            .flatMap(\.tags)
            .filter { seen.insert($0.id).inserted }
            .sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
    }

    func selectAsset(_ asset: AssetItem?) {
        selectedAssetID = asset?.id
    }

    func selectAsset(id: AssetItem.ID?) {
        selectedAssetID = id
    }

    func setViewMode(_ mode: AssetViewMode) {
        viewMode = mode
    }

    func selectSidebarItem(id: String?) {
        switch id {
        case "favorites":
            sidebarSelection = .favorites
        case "trash":
            sidebarSelection = .trash
        case let id? where id.hasPrefix("tag-"):
            sidebarSelection = .tag(String(id.dropFirst(4)))
        default:
            sidebarSelection = .library(currentLibrary.id)
        }

        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = visibleAssets.first?.id
        }
    }

    func sidebarItemID() -> String {
        switch sidebarSelection {
        case .library:
            "all-assets"
        case .favorites:
            "favorites"
        case .tag(let tagID):
            "tag-\(tagID)"
        case .trash:
            "trash"
        }
    }

    func updateSelectedTags(_ names: [String]) {
        guard let selectedAssetID,
              let index = assets.firstIndex(where: { $0.id == selectedAssetID }) else {
            return
        }

        assets[index].tags = names.map { TagItem(name: $0) }
    }

    func importItems(from urls: [URL]) async throws {
        let imported = try await importService.importItems(from: urls, into: currentLibrary)
        var existingHashes = Set(assets.map(\.contentHash))

        for asset in imported where existingHashes.insert(asset.contentHash).inserted {
            assets.append(asset)
        }

        if selectedAssetID == nil {
            selectedAssetID = visibleAssets.first?.id
        }
    }

    private static func sampleAssets(for library: AssetLibrary) -> [AssetItem] {
        let now = Date(timeIntervalSince1970: 0)
        let design = TagItem(name: "Design", colorHex: "#5E8CFF")
        let poster = TagItem(name: "Poster", colorHex: "#E06C75")

        return [
            AssetItem(
                id: "sample-brand-board",
                libraryID: library.id,
                displayName: "Brand Board",
                originalURL: URL(fileURLWithPath: "/Samples/Brand Board.png"),
                storageURL: URL(fileURLWithPath: "/Samples/Brand Board.png"),
                kind: .image,
                fileExtension: "png",
                byteSize: 2_048_000,
                contentHash: "sample-brand-board",
                dimensions: AssetDimensions(width: 1600, height: 1200),
                tags: [design],
                isFavorite: true,
                importedAt: now
            ),
            AssetItem(
                id: "sample-poster-study",
                libraryID: library.id,
                displayName: "Poster Study",
                originalURL: URL(fileURLWithPath: "/Samples/Poster Study.jpg"),
                storageURL: URL(fileURLWithPath: "/Samples/Poster Study.jpg"),
                kind: .image,
                fileExtension: "jpg",
                byteSize: 1_536_000,
                contentHash: "sample-poster-study",
                dimensions: AssetDimensions(width: 1080, height: 1440),
                tags: [poster],
                isFavorite: false,
                importedAt: now
            ),
            AssetItem(
                id: "sample-motion-reference",
                libraryID: library.id,
                displayName: "Motion Reference",
                originalURL: URL(fileURLWithPath: "/Samples/Motion Reference.mp4"),
                storageURL: URL(fileURLWithPath: "/Samples/Motion Reference.mp4"),
                kind: .video,
                fileExtension: "mp4",
                byteSize: 8_192_000,
                contentHash: "sample-motion-reference",
                dimensions: AssetDimensions(width: 1920, height: 1080),
                tags: [design],
                isFavorite: false,
                importedAt: now
            )
        ]
    }
}
