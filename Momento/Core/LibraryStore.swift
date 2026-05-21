import Foundation
import Observation

@MainActor
@Observable
final class LibraryStore {
    var libraries: [AssetLibrary]
    var currentLibrary: AssetLibrary?
    var assets: [AssetItem]
    var selectedAssetID: AssetItem.ID?
    var searchQuery = ""
    var viewMode: AssetViewMode = .masonry
    var sidebarSelection: SidebarSelection = .library("")
    var recentLibraries: [RecentLibraryReference]
    var libraryErrorMessage: String?

    private let importService: AssetImportService
    private let storage: LibraryStorage
    private let recentStore: RecentLibraryStore
    private var metadataStore: LibraryMetadataStore?
    private var libraryAccessScope: LibraryAccessScope?

    init(
        libraries: [AssetLibrary]? = nil,
        assets: [AssetItem]? = nil,
        defaultViewMode: AssetViewMode = .masonry,
        importService: AssetImportService = AssetImportService(),
        storage: LibraryStorage = LibraryStorage(),
        recentStore: RecentLibraryStore = RecentLibraryStore(),
        loadRecentLibrary: Bool = true
    ) {
        self.libraries = []
        self.currentLibrary = nil
        self.assets = []
        self.selectedAssetID = nil
        self.viewMode = defaultViewMode
        self.importService = importService
        self.storage = storage
        self.recentStore = recentStore
        self.recentLibraries = recentStore.load()
        self.libraryErrorMessage = nil

        if let libraries {
            let currentLibrary = libraries.first ?? .defaultLibrary
            let initialAssets = assets ?? Self.sampleAssets(for: currentLibrary)
            self.libraries = libraries.isEmpty ? [currentLibrary] : libraries
            self.currentLibrary = currentLibrary
            self.assets = initialAssets
            self.selectedAssetID = initialAssets.first?.id
            self.sidebarSelection = .library(currentLibrary.id)
        } else if loadRecentLibrary, let reference = recentLibraries.first {
            do {
                try openRecentLibrary(reference)
            } catch {
                libraryErrorMessage = libraryErrorMessage(for: error)
            }
        }
    }

    var visibleAssets: [AssetItem] {
        guard let currentLibrary else {
            return []
        }

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
        guard let currentLibrary else {
            return []
        }

        var seen = Set<String>()
        return assets
            .filter { $0.libraryID == currentLibrary.id }
            .flatMap(\.tags)
            .filter { seen.insert($0.id).inserted }
            .sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
    }

    func createLibrary(at packageURL: URL) throws {
        let name = packageURL.deletingPathExtension().lastPathComponent
        let library = try storage.createLibraryPackage(at: packageURL, name: name)
        try activateLibrary(library, accessScope: LibraryAccessScope(url: storage.rootURL(for: library)))
        try saveRecentLibrary(library)
    }

    func openLibrary(at packageURL: URL) throws {
        guard LibraryStorage.supportedPackageExtensions.contains(packageURL.pathExtension) else {
            throw LibraryStoreError.unsupportedLibraryURL
        }

        let accessScope = LibraryAccessScope(url: packageURL)
        let library = try storage.openLibraryPackage(at: packageURL)
        try activateLibrary(library, accessScope: accessScope)
        try saveRecentLibrary(library)
    }

    func openRecentLibrary(id: RecentLibraryReference.ID) throws {
        guard let reference = recentLibraries.first(where: { $0.id == id }) else {
            throw LibraryStoreError.missingRecentLibrary
        }
        try openRecentLibrary(reference)
    }

    func validateCurrentLibraryAvailability() throws {
        guard let currentLibrary else {
            return
        }

        do {
            _ = try storage.openLibraryPackage(at: storage.rootURL(for: currentLibrary))
        } catch LibraryStorageError.missingLibraryPackage {
            let libraryID = currentLibrary.id
            closeCurrentLibrary()
            try recentStore.remove(id: libraryID)
            recentLibraries = recentStore.load()
            libraryErrorMessage = LibraryStorageError.missingLibraryPackage.errorDescription
        }
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

    func closeCurrentLibrary() {
        libraryAccessScope = nil
        metadataStore = nil
        currentLibrary = nil
        libraries = []
        assets = []
        selectedAssetID = nil
        searchQuery = ""
        sidebarSelection = .library("")
        libraryErrorMessage = nil
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
            sidebarSelection = .library(currentLibrary?.id ?? "")
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
        guard let currentLibrary, let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        let importingLibraryID = currentLibrary.id
        let importAccessScope = LibraryAccessScope(url: storage.rootURL(for: currentLibrary))
        defer {
            _ = importAccessScope
        }

        let imported = try await importService.importItems(
            from: urls,
            into: currentLibrary,
            excludingContentHashes: metadataStore.existingContentHashes()
        )
        let savedAssets = try metadataStore.saveImportedAssets(imported)
        guard self.currentLibrary?.id == importingLibraryID else {
            return
        }

        var existingIDs = Set(assets.map(\.id))

        for asset in savedAssets where existingIDs.insert(asset.id).inserted {
            assets.append(asset)
        }

        if selectedAssetID == nil {
            selectedAssetID = visibleAssets.first?.id
        }
    }

    private func openRecentLibrary(_ reference: RecentLibraryReference) throws {
        let resolved: (url: URL, isStale: Bool)
        do {
            resolved = try recentStore.resolve(reference)
        } catch {
            try removeRecentLibrary(reference)
            throw LibraryStoreError.missingRecentLibrary
        }

        let accessScope = LibraryAccessScope(url: resolved.url)
        let library: AssetLibrary
        do {
            library = try storage.openLibraryPackage(at: resolved.url)
        } catch LibraryStorageError.missingLibraryPackage {
            try removeRecentLibrary(reference)
            throw LibraryStoreError.missingRecentLibrary
        }

        try activateLibrary(library, accessScope: accessScope)

        if resolved.isStale {
            try saveRecentLibrary(library)
        }
    }

    private func removeRecentLibrary(_ reference: RecentLibraryReference) throws {
        try recentStore.remove(id: reference.id)
        recentLibraries = recentStore.load()
    }

    private func activateLibrary(_ library: AssetLibrary, accessScope: LibraryAccessScope) throws {
        let metadataStore = try LibraryMetadataStore(library: library, storage: storage)
        let loadedAssets = try metadataStore.loadAssets()

        libraryAccessScope = accessScope
        self.metadataStore = metadataStore
        currentLibrary = library
        libraries = [library]
        assets = loadedAssets
        selectedAssetID = loadedAssets.first?.id
        searchQuery = ""
        sidebarSelection = .library(library.id)
        libraryErrorMessage = nil
    }

    private func saveRecentLibrary(_ library: AssetLibrary) throws {
        try recentStore.save(library)
        recentLibraries = recentStore.load()
    }

    private func libraryErrorMessage(for error: Error) -> String {
        if let storeError = error as? LibraryStoreError {
            return storeError.errorDescription ?? error.localizedDescription
        }

        if let storageError = error as? LibraryStorageError {
            return storageError.errorDescription ?? error.localizedDescription
        }

        return error.localizedDescription
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

enum LibraryStoreError: LocalizedError {
    case noCurrentLibrary
    case missingRecentLibrary
    case unsupportedLibraryURL

    var errorDescription: String? {
        switch self {
        case .noCurrentLibrary:
            "Create or open a Momento library before importing assets."
        case .missingRecentLibrary:
            "This recent library is no longer available."
        case .unsupportedLibraryURL:
            "Choose a .momento package."
        }
    }
}
