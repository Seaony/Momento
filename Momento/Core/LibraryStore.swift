import Foundation
import Observation

@MainActor
@Observable
final class LibraryStore {
    var libraries: [AssetLibrary]
    var currentLibrary: AssetLibrary?
    var assets: [AssetItem]
    var folders: [AssetFolder]
    var tags: [TagItem]
    var selectedAssetID: AssetItem.ID?
    var searchQuery = ""
    var viewMode: AssetViewMode = .masonry
    var filterState = AssetFilterState()
    var sortOption: AssetSortOption = .addedTime
    var sortDirection: AssetSortDirection = .descending
    var sidebarSelection: SidebarSelection = .library("")
    var recentLibraries: [RecentLibraryReference]
    var libraryErrorMessage: String?

    private let importService: AssetImportService
    private let thumbnailService: AssetThumbnailService
    private let colorAnalysisService: AssetColorAnalysisService
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
        self.folders = []
        self.tags = []
        self.selectedAssetID = nil
        self.viewMode = defaultViewMode
        self.importService = importService
        self.storage = storage
        self.thumbnailService = AssetThumbnailService(storage: storage)
        self.colorAnalysisService = AssetColorAnalysisService()
        self.recentStore = recentStore
        self.recentLibraries = recentStore.load()
        self.libraryErrorMessage = nil

        if let libraries {
            let currentLibrary = libraries.first ?? .defaultLibrary
            let initialAssets = assets ?? Self.sampleAssets(for: currentLibrary)
            self.libraries = libraries.isEmpty ? [currentLibrary] : libraries
            self.currentLibrary = currentLibrary
            self.assets = initialAssets
            self.folders = []
            self.tags = Self.uniqueTags(from: initialAssets, libraryID: currentLibrary.id)
            self.selectedAssetID = nil
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
        guard currentLibrary != nil else {
            return []
        }

        let libraryAssets = currentLibraryAssets
        let scopedAssets: [AssetItem]

        switch sidebarSelection {
        case .library:
            scopedAssets = libraryAssets
        case .favorites:
            scopedAssets = libraryAssets.filter(\.isFavorite)
        case .uncategorized:
            scopedAssets = libraryAssets.filter(\.folderIDs.isEmpty)
        case .untagged:
            scopedAssets = libraryAssets.filter(\.tags.isEmpty)
        case .tagManagement, .folderManagement:
            scopedAssets = []
        case .folder(let folderID):
            scopedAssets = libraryAssets.filter { $0.folderIDs.contains(folderID) }
        case .tag(let tagID):
            scopedAssets = libraryAssets.filter { asset in
                asset.tags.contains { $0.id == tagID }
            }
        case .trash:
            scopedAssets = allCurrentLibraryAssets.filter(\.isTrashed)
        }

        let filteredAssets = applyFilters(to: scopedAssets)
        let query = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !query.isEmpty else {
            return sortAssets(filteredAssets)
        }

        return sortAssets(filteredAssets.filter { asset in
            asset.displayName.localizedCaseInsensitiveContains(query)
                || asset.fileExtension.localizedCaseInsensitiveContains(query)
                || asset.tags.contains { $0.name.localizedCaseInsensitiveContains(query) }
        })
    }

    var selectedAsset: AssetItem? {
        guard let selectedAssetID else {
            return nil
        }
        guard visibleAssets.contains(where: { $0.id == selectedAssetID }) else {
            return nil
        }
        return assets.first { $0.id == selectedAssetID }
    }

    func folder(id: AssetFolder.ID) -> AssetFolder? {
        folders.first { $0.id == id }
    }

    var availableFilterColorCategories: [AssetColorCategory] {
        AssetColorCategory.allCases
    }

    var availableFilterFileExtensions: [String] {
        Array(
            Set(
                currentLibraryAssets
                    .map { Self.normalizedFileExtension($0.fileExtension) }
                    .filter { !$0.isEmpty }
            )
        )
        .sorted { $0.localizedStandardCompare($1) == .orderedAscending }
    }

    var tagSummaries: [TagSummary] {
        guard currentLibrary != nil else {
            return []
        }

        var countsByID: [TagItem.ID: Int] = [:]
        for asset in currentLibraryAssets {
            var seenTagIDs = Set<TagItem.ID>()
            for tag in asset.tags where seenTagIDs.insert(tag.id).inserted {
                countsByID[tag.id, default: 0] += 1
            }
        }

        return tags.map { tag in
            TagSummary(tag: tag, assetCount: countsByID[tag.id, default: 0])
        }.sorted {
            $0.tag.name.localizedCaseInsensitiveCompare($1.tag.name) == .orderedAscending
        }
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

    func clearCachesAndReloadCurrentLibrary() throws {
        guard let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        let packageURL = storage.rootURL(for: currentLibrary)
        try storage.clearTransientCaches(for: currentLibrary)
        try openLibrary(at: packageURL)
        try rebuildMissingThumbnails()
    }

    func renameRecentLibrary(id: RecentLibraryReference.ID, to name: String) throws {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty else {
            throw LibraryStoreError.invalidLibraryName
        }

        guard let reference = recentLibraries.first(where: { $0.id == id }) else {
            throw LibraryStoreError.missingRecentLibrary
        }

        let library = try withSecurityScopedRecentLibraryURL(reference) { resolvedURL in
            try storage.renameLibraryPackage(at: resolvedURL, to: trimmedName)
        }
        try recentStore.updateName(id: id, name: trimmedName)
        recentLibraries = recentStore.load()

        if currentLibrary?.id == id {
            currentLibrary = library
            libraries = [library]
        }
    }

    func deleteRecentLibrary(id: RecentLibraryReference.ID) throws {
        guard let reference = recentLibraries.first(where: { $0.id == id }) else {
            throw LibraryStoreError.missingRecentLibrary
        }

        try withSecurityScopedRecentLibraryURL(reference) { resolvedURL in
            try storage.deleteLibraryPackage(at: resolvedURL)
        }

        if currentLibrary?.id == id {
            closeCurrentLibrary()
        }

        try recentStore.remove(id: id)
        recentLibraries = recentStore.load()
    }

    func recentLibraryURL(id: RecentLibraryReference.ID) throws -> URL {
        guard let reference = recentLibraries.first(where: { $0.id == id }) else {
            throw LibraryStoreError.missingRecentLibrary
        }

        return try recentStore.resolve(reference).url
    }

    func moveRecentLibrary(
        id: RecentLibraryReference.ID,
        relativeTo targetID: RecentLibraryReference.ID,
        insertAfterTarget: Bool
    ) throws {
        try recentStore.move(id: id, relativeTo: targetID, insertAfterTarget: insertAfterTarget)
        recentLibraries = recentStore.load()
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

    func toggleFilterColorCategory(_ category: AssetColorCategory) {
        if filterState.colorCategories.contains(category) {
            filterState.colorCategories.remove(category)
        } else {
            filterState.colorCategories.insert(category)
        }
        pruneSelectedAssetIfNeeded()
    }

    func toggleFilterTag(id tagID: TagItem.ID) {
        if filterState.tagIDs.contains(tagID) {
            filterState.tagIDs.remove(tagID)
        } else {
            filterState.tagIDs.insert(tagID)
        }
        pruneSelectedAssetIfNeeded()
    }

    func toggleFilterFileExtension(_ fileExtension: String) {
        let normalized = Self.normalizedFileExtension(fileExtension)
        guard !normalized.isEmpty else {
            return
        }

        if filterState.fileExtensions.contains(normalized) {
            filterState.fileExtensions.remove(normalized)
        } else {
            filterState.fileExtensions.insert(normalized)
        }
        pruneSelectedAssetIfNeeded()
    }

    func clearAssetFilters() {
        filterState = AssetFilterState()
        pruneSelectedAssetIfNeeded()
    }

    func setSortOption(_ option: AssetSortOption) {
        if sortOption == option {
            sortDirection = sortDirection == .descending ? .ascending : .descending
        } else {
            sortOption = option
            sortDirection = .descending
        }
    }

    func setSortDirection(_ direction: AssetSortDirection) {
        sortDirection = direction
    }

    func closeCurrentLibrary() {
        libraryAccessScope = nil
        metadataStore = nil
        currentLibrary = nil
        libraries = []
        assets = []
        folders = []
        tags = []
        selectedAssetID = nil
        searchQuery = ""
        filterState = AssetFilterState()
        sortOption = .addedTime
        sortDirection = .descending
        sidebarSelection = .library("")
        libraryErrorMessage = nil
    }

    func selectSidebarItem(id: String?) {
        switch id {
        case "favorites":
            sidebarSelection = .favorites
        case "uncategorized":
            sidebarSelection = .uncategorized
        case "untagged":
            sidebarSelection = .untagged
        case "tag-management":
            sidebarSelection = .tagManagement
        case "folder-management":
            sidebarSelection = .folderManagement
        case let id? where id.hasPrefix("folder-"):
            sidebarSelection = .folder(String(id.dropFirst(7)))
        case "trash":
            sidebarSelection = .trash
        case let id? where id.hasPrefix("tag-"):
            sidebarSelection = .tag(String(id.dropFirst(4)))
        default:
            sidebarSelection = .library(currentLibrary?.id ?? "")
        }

        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = nil
        }
    }

    func sidebarItemID() -> String {
        switch sidebarSelection {
        case .library:
            "all-assets"
        case .favorites:
            "favorites"
        case .uncategorized:
            "uncategorized"
        case .untagged:
            "untagged"
        case .tagManagement:
            "tag-management"
        case .folderManagement:
            "folder-management"
        case .folder(let folderID):
            "folder-\(folderID)"
        case .tag(let tagID):
            "tag-\(tagID)"
        case .trash:
            "trash"
        }
    }

    func updateSelectedTags(_ names: [String]) throws {
        guard let selectedAssetID,
              let index = assets.firstIndex(where: { $0.id == selectedAssetID }) else {
            return
        }

        if let metadataStore {
            mergeAssets([try metadataStore.setTagNames(names, forAssetID: selectedAssetID)])
            tags = try metadataStore.loadTags()
        } else {
            let updatedTags = normalizedTags(from: names)
            assets[index].tags = updatedTags
            if let currentLibrary {
                tags = Self.uniqueTags(from: assets, libraryID: currentLibrary.id)
            }
        }
    }

    func renameTag(id tagID: TagItem.ID, to name: String) throws {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty else {
            throw LibraryStoreError.invalidTagName
        }

        guard let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        if let metadataStore {
            mergeAssets(try metadataStore.renameTag(id: tagID, to: trimmedName))
            tags = try metadataStore.loadTags()
            if case .tag(let selectedTagID) = sidebarSelection, selectedTagID == tagID {
                sidebarSelection = .tag(tagID)
            }
            pruneSelectedAssetIfNeeded()
            return
        }

        let libraryAssets = assets.filter { $0.libraryID == currentLibrary.id }
        guard let existingTag = libraryAssets.flatMap(\.tags).first(where: { $0.id == tagID }) else {
            throw LibraryStoreError.missingTag
        }

        let replacementTag = libraryAssets
            .flatMap(\.tags)
            .first {
                $0.id != tagID && $0.name.caseInsensitiveCompare(trimmedName) == .orderedSame
            }
            ?? TagItem(id: existingTag.id, name: trimmedName, colorHex: existingTag.colorHex)

        try updateAssetsWithTag(id: tagID) { tags in
            deduplicatedTags(
                tags.map { tag in
                    tag.id == tagID ? replacementTag : tag
                }
            )
        }

        if case .tag(let selectedTagID) = sidebarSelection, selectedTagID == tagID {
            sidebarSelection = .tag(replacementTag.id)
        }
        tags = Self.uniqueTags(from: assets, libraryID: currentLibrary.id)
    }

    func deleteTag(id tagID: TagItem.ID) throws {
        guard let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        if let metadataStore {
            mergeAssets(try metadataStore.deleteTag(id: tagID))
            tags = try metadataStore.loadTags()
            if case .tag(let selectedTagID) = sidebarSelection, selectedTagID == tagID {
                sidebarSelection = .tagManagement
            }
            pruneSelectedAssetIfNeeded()
            return
        }

        guard assets.contains(where: { asset in
            asset.libraryID == currentLibrary.id &&
            asset.tags.contains { $0.id == tagID }
        }) else {
            throw LibraryStoreError.missingTag
        }

        try updateAssetsWithTag(id: tagID) { tags in
            tags.filter { $0.id != tagID }
        }

        if case .tag(let selectedTagID) = sidebarSelection, selectedTagID == tagID {
            sidebarSelection = .tagManagement
        }
        tags = Self.uniqueTags(from: assets, libraryID: currentLibrary.id)
    }

    func createFolder(name: String? = nil, parentID: AssetFolder.ID? = nil) throws {
        guard let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        let folder = try metadataStore.createFolder(
            name: name ?? nextFolderName(parentID: parentID),
            parentID: parentID
        )
        folders = try metadataStore.loadFolders()
        sidebarSelection = .folder(folder.id)
        selectedAssetID = nil
    }

    func renameFolder(id: AssetFolder.ID, to name: String) throws {
        guard let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        _ = try metadataStore.renameFolder(id: id, to: name)
        folders = try metadataStore.loadFolders()
    }

    func deleteFolder(id: AssetFolder.ID) throws {
        guard let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        let deletedIDs = Set(try metadataStore.deleteFolder(id: id))
        folders = try metadataStore.loadFolders()
        assets = try metadataStore.loadAssets()

        if case .folder(let selectedFolderID) = sidebarSelection, deletedIDs.contains(selectedFolderID) {
            sidebarSelection = .library(currentLibrary?.id ?? "")
        }

        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = nil
        }
    }

    func assignAssets(ids: Set<AssetItem.ID>, to folderID: AssetFolder.ID) throws {
        guard let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        mergeAssets(try metadataStore.assignAssets(ids: ids, to: folderID))
    }

    func unassignAssets(ids: Set<AssetItem.ID>, from folderID: AssetFolder.ID) throws {
        guard let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        mergeAssets(try metadataStore.unassignAssets(ids: ids, from: folderID))
    }

    func toggleFavorite(for assetID: AssetItem.ID) throws {
        guard let metadataStore,
              let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        guard let asset = assets.first(where: { $0.id == assetID && $0.libraryID == currentLibrary.id }) else {
            throw LibraryStoreError.missingAsset
        }

        mergeAssets([try metadataStore.setFavorite(!asset.isFavorite, forAssetID: asset.id)])
        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = nil
        }
    }

    func renameAsset(id assetID: AssetItem.ID, to displayName: String) throws {
        let trimmedName = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty else {
            throw LibraryStoreError.invalidAssetName
        }

        guard let metadataStore,
              let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        guard assets.contains(where: { $0.id == assetID && $0.libraryID == currentLibrary.id }) else {
            throw LibraryStoreError.missingAsset
        }

        mergeAssets([try metadataStore.renameAsset(id: assetID, to: trimmedName)])
        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = nil
        }
    }

    func refreshThumbnail(for assetID: AssetItem.ID) throws -> AssetItem? {
        guard let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        guard let index = assets.firstIndex(where: { $0.id == assetID && $0.libraryID == currentLibrary.id }) else {
            throw LibraryStoreError.missingAsset
        }

        var asset = assets[index]
        guard asset.kind == .image || asset.kind == .gif else {
            return asset
        }

        let thumbnailURL = try thumbnailService.generateThumbnail(
            for: asset.storageURL,
            contentHash: asset.contentHash,
            in: currentLibrary
        )
        asset.thumbnailURL = thumbnailURL
        assets[index] = asset
        return asset
    }

    func reanalyzeColors(for assetID: AssetItem.ID) throws {
        guard let metadataStore,
              let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        guard let asset = assets.first(where: { $0.id == assetID && $0.libraryID == currentLibrary.id }) else {
            throw LibraryStoreError.missingAsset
        }

        let sourceURL = asset.thumbnailURL ?? asset.storageURL
        let colors = colorAnalysisService.paletteColors(
            for: sourceURL,
            libraryID: currentLibrary.id,
            assetID: asset.id
        )
        mergeAssets([try metadataStore.replaceColors(colors, forAssetID: asset.id)])
    }

    func moveAssetToTrash(id assetID: AssetItem.ID) throws {
        guard let metadataStore,
              let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        guard let asset = assets.first(where: { $0.id == assetID && $0.libraryID == currentLibrary.id }) else {
            throw LibraryStoreError.missingAsset
        }

        mergeAssets([try metadataStore.moveAssetToTrash(id: asset.id)])
        pruneSelectedAssetIfNeeded()
    }

    func restoreAssets(ids assetIDs: Set<AssetItem.ID>) throws {
        guard let metadataStore else {
            throw LibraryStoreError.noCurrentLibrary
        }

        mergeAssets(try metadataStore.restoreAssets(ids: assetIDs))
        pruneSelectedAssetIfNeeded()
    }

    func deleteAssetPermanently(id assetID: AssetItem.ID) throws {
        guard let metadataStore,
              let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        guard let asset = allCurrentLibraryAssets.first(where: { $0.id == assetID }) else {
            throw LibraryStoreError.missingAsset
        }

        try storage.removeStoredAssetFiles(for: asset, in: currentLibrary)
        try metadataStore.deleteAsset(id: assetID)
        assets.removeAll { $0.id == assetID }
        pruneSelectedAssetIfNeeded()
    }

    func emptyTrash() throws {
        guard let metadataStore,
              let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        let trashedAssets = allCurrentLibraryAssets.filter(\.isTrashed)
        for asset in trashedAssets {
            try storage.removeStoredAssetFiles(for: asset, in: currentLibrary)
        }

        let deletedIDs = Set(try metadataStore.emptyTrash())
        assets.removeAll { deletedIDs.contains($0.id) }
        pruneSelectedAssetIfNeeded()
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

        mergeAssets(savedAssets)
        pruneSelectedAssetIfNeeded()
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

    private func withSecurityScopedRecentLibraryURL<T>(
        _ reference: RecentLibraryReference,
        perform action: (URL) throws -> T
    ) throws -> T {
        let resolved = try recentStore.resolve(reference)
        let accessScope = LibraryAccessScope(url: resolved.url)
        return try withExtendedLifetime(accessScope) {
            try action(resolved.url)
        }
    }

    private var currentLibraryAssets: [AssetItem] {
        allCurrentLibraryAssets.filter { !$0.isTrashed }
    }

    private var allCurrentLibraryAssets: [AssetItem] {
        guard let currentLibrary else {
            return []
        }

        return assets.filter { $0.libraryID == currentLibrary.id }
    }

    private func applyFilters(to assets: [AssetItem]) -> [AssetItem] {
        guard filterState.isActive else {
            return assets
        }

        return assets.filter { asset in
            let colorCategories = Self.colorCategories(for: asset)
            let matchesColors = filterState.colorCategories.isEmpty
                || colorCategories.contains { filterState.colorCategories.contains($0) }
            let matchesTags = filterState.tagIDs.isEmpty || asset.tags.contains { tag in
                filterState.tagIDs.contains(tag.id)
            }
            let matchesFileTypes = filterState.fileExtensions.isEmpty
                || filterState.fileExtensions.contains(Self.normalizedFileExtension(asset.fileExtension))

            return matchesColors && matchesTags && matchesFileTypes
        }
    }

    private func sortAssets(_ assets: [AssetItem]) -> [AssetItem] {
        assets.sorted { lhs, rhs in
            let comparison: ComparisonResult
            switch sortOption {
            case .addedTime:
                comparison = lhs.importedAt.compare(rhs.importedAt)
            case .name:
                comparison = lhs.displayName.localizedStandardCompare(rhs.displayName)
            case .fileSize:
                comparison = lhs.byteSize == rhs.byteSize
                    ? lhs.displayName.localizedStandardCompare(rhs.displayName)
                    : (lhs.byteSize < rhs.byteSize ? .orderedAscending : .orderedDescending)
            }

            if comparison == .orderedSame {
                return lhs.id.localizedStandardCompare(rhs.id) == .orderedAscending
            }

            switch sortDirection {
            case .ascending:
                return comparison == .orderedAscending
            case .descending:
                return comparison == .orderedDescending
            }
        }
    }

    private func pruneSelectedAssetIfNeeded() {
        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = nil
        }
    }

    private func activateLibrary(_ library: AssetLibrary, accessScope: LibraryAccessScope) throws {
        let metadataStore = try LibraryMetadataStore(library: library, storage: storage)
        let loadedAssets = try metadataStore.loadAssets()
        let loadedFolders = try metadataStore.loadFolders()
        let loadedTags = try metadataStore.loadTags()

        libraryAccessScope = accessScope
        self.metadataStore = metadataStore
        currentLibrary = library
        libraries = [library]
        assets = loadedAssets
        folders = loadedFolders
        tags = loadedTags
        selectedAssetID = nil
        searchQuery = ""
        filterState = AssetFilterState()
        sortOption = .addedTime
        sortDirection = .descending
        sidebarSelection = .library(library.id)
        libraryErrorMessage = nil
    }

    private func saveRecentLibrary(_ library: AssetLibrary) throws {
        try recentStore.save(library)
        recentLibraries = recentStore.load()
    }

    private func mergeAssets(_ updatedAssets: [AssetItem]) {
        for asset in updatedAssets {
            if let index = assets.firstIndex(where: { $0.id == asset.id }) {
                assets[index] = asset
            } else {
                assets.append(asset)
            }
        }
    }

    private func updateAssetsWithTag(
        id tagID: TagItem.ID,
        transform: ([TagItem]) -> [TagItem]
    ) throws {
        guard let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
        }

        let updatedAssetIDs = assets
            .filter { asset in
                asset.libraryID == currentLibrary.id &&
                asset.tags.contains { $0.id == tagID }
            }
            .map(\.id)

        guard !updatedAssetIDs.isEmpty else {
            return
        }

        if let metadataStore {
            var updatedAssets: [AssetItem] = []
            for assetID in updatedAssetIDs {
                guard let asset = assets.first(where: { $0.id == assetID }) else {
                    continue
                }
                updatedAssets.append(
                    try metadataStore.updateTags(transform(asset.tags), forAssetID: assetID)
                )
            }
            mergeAssets(updatedAssets)
        } else {
            for assetID in updatedAssetIDs {
                guard let index = assets.firstIndex(where: { $0.id == assetID }) else {
                    continue
                }
                assets[index].tags = transform(assets[index].tags)
            }
        }

        if let selectedAssetID, !visibleAssets.contains(where: { $0.id == selectedAssetID }) {
            self.selectedAssetID = nil
        }
    }

    private func deduplicatedTags(_ tags: [TagItem]) -> [TagItem] {
        var seen = Set<TagItem.ID>()
        return tags.filter { tag in
            seen.insert(tag.id).inserted
        }
    }

    private static func normalizedFileExtension(_ fileExtension: String) -> String {
        fileExtension
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
    }

    private static func colorCategories(for asset: AssetItem) -> [AssetColorCategory] {
        let colors = asset.paletteColors.sorted {
            if $0.coverage == $1.coverage {
                return $0.hex.localizedStandardCompare($1.hex) == .orderedAscending
            }
            return $0.coverage > $1.coverage
        }

        guard let topCoverage = colors.first?.coverage else {
            return []
        }

        var seen = Set<AssetColorCategory>()
        return colors
            .enumerated()
            .filter { index, color in
                index == 0 || color.coverage >= 0.18 || color.coverage >= topCoverage * 0.55
            }
            .prefix(3)
            .compactMap { _, color in
                guard let category = colorCategory(for: color.hex),
                      seen.insert(category).inserted else {
                    return nil
                }
                return category
            }
    }

    private static func colorCategory(for hex: String) -> AssetColorCategory? {
        let cleaned = hex
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .trimmingCharacters(in: CharacterSet(charactersIn: "#"))
        guard cleaned.count == 6,
              let value = UInt64(cleaned, radix: 16) else {
            return nil
        }

        let red = Double((value >> 16) & 0xFF) / 255
        let green = Double((value >> 8) & 0xFF) / 255
        let blue = Double(value & 0xFF) / 255
        let maximum = max(red, green, blue)
        let minimum = min(red, green, blue)
        let delta = maximum - minimum
        let saturation = maximum == 0 ? 0 : delta / maximum
        let brightness = maximum

        if brightness <= 0.18 {
            return .black
        }

        let hue: Double
        if delta == 0 {
            hue = 0
        } else if maximum == red {
            hue = 60 * ((green - blue) / delta).truncatingRemainder(dividingBy: 6)
        } else if maximum == green {
            hue = 60 * ((blue - red) / delta + 2)
        } else {
            hue = 60 * ((red - green) / delta + 4)
        }

        let normalizedHue = hue < 0 ? hue + 360 : hue

        if saturation <= 0.14 {
            if brightness >= 0.82 {
                return .white
            }
            return brightness <= 0.28 ? .black : .gray
        }

        if saturation <= 0.34,
           brightness >= 0.62,
           normalizedHue >= 35,
           normalizedHue < 70 {
            return .beige
        }

        if normalizedHue >= 15, normalizedHue < 50, brightness < 0.58, saturation > 0.22 {
            return .brown
        }

        if normalizedHue >= 50, normalizedHue < 90, brightness < 0.58, saturation > 0.22 {
            return .olive
        }

        switch normalizedHue {
        case 0..<12, 350..<360:
            return .red
        case 12..<25:
            return .coral
        case 25..<38:
            return .orange
        case 38..<50:
            return .amber
        case 50..<65:
            return .yellow
        case 65..<85:
            return .lime
        case 85..<145:
            return .green
        case 145..<165:
            return .mint
        case 165..<195:
            return .teal
        case 195..<205:
            return .cyan
        case 205..<220:
            return .sky
        case 220..<250:
            return .blue
        case 250..<265:
            return .indigo
        case 265..<285:
            return .violet
        case 285..<305:
            return .purple
        case 305..<325:
            return .magenta
        case 325..<340:
            return .pink
        case 340..<350:
            return .rose
        default:
            return .red
        }
    }

    private func nextFolderName(parentID: AssetFolder.ID?) -> String {
        let baseName = "New Folder"
        let siblingNames = Set(folders.filter { $0.parentID == parentID }.map(\.name))
        guard siblingNames.contains(baseName) else {
            return baseName
        }

        var index = 2
        while siblingNames.contains("\(baseName) \(index)") {
            index += 1
        }
        return "\(baseName) \(index)"
    }

    private func rebuildMissingThumbnails() throws {
        guard let currentLibrary, let metadataStore else {
            return
        }

        var repairedAssets: [AssetItem] = []
        for asset in assets where (asset.kind == .image || asset.kind == .gif) && asset.thumbnailURL == nil {
            guard let thumbnailURL = try? thumbnailService.generateThumbnail(
                for: asset.storageURL,
                contentHash: asset.contentHash,
                in: currentLibrary
            ) else {
                continue
            }

            var repairedAsset = asset
            repairedAsset.thumbnailURL = thumbnailURL
            repairedAssets.append(repairedAsset)
        }

        mergeAssets(repairedAssets)
        assets = try metadataStore.loadAssets()
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

    private func normalizedTags(from names: [String]) -> [TagItem] {
        let existingTags = Dictionary(
            tags.map { ($0.name.lowercased(), $0) },
            uniquingKeysWith: { first, _ in first }
        )
        var seen = Set<String>()

        return names.compactMap { name in
            let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmedName.isEmpty else {
                return nil
            }

            let key = trimmedName.lowercased()
            guard seen.insert(key).inserted else {
                return nil
            }

            return existingTags[key] ?? TagItem(name: trimmedName)
        }
    }

    private static func uniqueTags(from assets: [AssetItem], libraryID: AssetLibrary.ID) -> [TagItem] {
        var seen = Set<TagItem.ID>()
        return assets
            .filter { $0.libraryID == libraryID }
            .flatMap(\.tags)
            .filter { seen.insert($0.id).inserted }
            .sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
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
    case invalidLibraryName
    case invalidAssetName
    case invalidTagName
    case missingAsset
    case missingTag

    var errorDescription: String? {
        switch self {
        case .noCurrentLibrary:
            "Create or open a Momento library before importing assets."
        case .missingRecentLibrary:
            "This recent library is no longer available."
        case .unsupportedLibraryURL:
            "Choose a .momento package."
        case .invalidLibraryName:
            "Enter a library name."
        case .invalidAssetName:
            "Enter an asset title."
        case .invalidTagName:
            "Enter a tag name."
        case .missingAsset:
            "This asset is no longer available."
        case .missingTag:
            "This tag is no longer available."
        }
    }
}
