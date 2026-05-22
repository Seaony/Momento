import Foundation
import Observation

@MainActor
@Observable
final class LibraryStore {
    var libraries: [AssetLibrary]
    var currentLibrary: AssetLibrary?
    var assets: [AssetItem]
    var folders: [AssetFolder]
    var selectedAssetID: AssetItem.ID?
    var searchQuery = ""
    var viewMode: AssetViewMode = .masonry
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

    func folder(id: AssetFolder.ID) -> AssetFolder? {
        folders.first { $0.id == id }
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

    var tagSummaries: [TagSummary] {
        guard let currentLibrary else {
            return []
        }

        var summariesByID: [TagItem.ID: TagSummary] = [:]
        for asset in assets where asset.libraryID == currentLibrary.id {
            var seenTagIDs = Set<TagItem.ID>()
            for tag in asset.tags where seenTagIDs.insert(tag.id).inserted {
                var summary = summariesByID[tag.id] ?? TagSummary(tag: tag, assetCount: 0)
                summary.assetCount += 1
                summariesByID[tag.id] = summary
            }
        }

        return summariesByID.values.sorted {
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

        let resolved = try recentStore.resolve(reference)
        let library = try storage.renameLibraryPackage(at: resolved.url, to: trimmedName)
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

        let resolved = try recentStore.resolve(reference)
        try storage.deleteLibraryPackage(at: resolved.url)

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

    func closeCurrentLibrary() {
        libraryAccessScope = nil
        metadataStore = nil
        currentLibrary = nil
        libraries = []
        assets = []
        folders = []
        selectedAssetID = nil
        searchQuery = ""
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

        let updatedTags = normalizedTags(from: names)
        if let metadataStore {
            mergeAssets([try metadataStore.updateTags(updatedTags, forAssetID: selectedAssetID)])
        } else {
            assets[index].tags = updatedTags
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
    }

    func deleteTag(id tagID: TagItem.ID) throws {
        guard let currentLibrary else {
            throw LibraryStoreError.noCurrentLibrary
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

        try metadataStore.deleteAsset(id: assetID)

        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            try storage.trashAssetFile(at: asset.storageURL)
        }

        if let thumbnailURL = asset.thumbnailURL,
           FileManager.default.fileExists(atPath: thumbnailURL.path) {
            try? FileManager.default.removeItem(at: thumbnailURL)
        }

        assets.removeAll { $0.id == assetID }
        if selectedAssetID == assetID {
            selectedAssetID = nil
        }
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
        let loadedFolders = try metadataStore.loadFolders()

        libraryAccessScope = accessScope
        self.metadataStore = metadataStore
        currentLibrary = library
        libraries = [library]
        assets = loadedAssets
        folders = loadedFolders
        selectedAssetID = nil
        searchQuery = ""
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
