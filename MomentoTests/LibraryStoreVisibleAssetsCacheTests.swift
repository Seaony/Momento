import Foundation
import XCTest
@testable import Momento

@MainActor
final class LibraryStoreVisibleAssetsCacheTests: XCTestCase {
    func testVisibleAssetsRevisionAdvancesForDerivedInputChangesButNotCacheHits() {
        let library = AssetLibrary(
            id: "library",
            name: "Library",
            createdAt: Date(timeIntervalSince1970: 0)
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [
                Self.makeAsset(id: "alpha", libraryID: library.id, displayName: "Alpha", fileExtension: "png"),
                Self.makeAsset(id: "beta", libraryID: library.id, displayName: "Beta", fileExtension: "jpg", isFavorite: true)
            ],
            loadRecentLibrary: false
        )

        XCTAssertEqual(store.visibleAssets.map(\.id), ["beta", "alpha"])
        let initialRevision = store.visibleAssetsRevision

        XCTAssertEqual(store.visibleAssets.map(\.id), ["beta", "alpha"])
        XCTAssertEqual(store.visibleAssetsRevision, initialRevision)

        store.searchQuery = "alpha"
        XCTAssertEqual(store.visibleAssets.map(\.id), ["alpha"])
        let afterSearchRevision = store.visibleAssetsRevision
        XCTAssertGreaterThan(afterSearchRevision, initialRevision)

        store.searchQuery = ""
        store.toggleFilterFileExtension("jpg")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["beta"])
        let afterFilterRevision = store.visibleAssetsRevision
        XCTAssertGreaterThan(afterFilterRevision, afterSearchRevision)

        store.clearAssetFilters()
        store.setSortOption(.name)
        store.setSortDirection(.ascending)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["alpha", "beta"])
        let afterSortRevision = store.visibleAssetsRevision
        XCTAssertGreaterThan(afterSortRevision, afterFilterRevision)

        store.selectSidebarItem(id: "favorites")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["beta"])
        XCTAssertGreaterThan(store.visibleAssetsRevision, afterSortRevision)
    }

    func testVisibleAssetsAndSidebarCountsInvalidateForPersistentAssetMutations() async throws {
        let environment = try AssetsVersionTestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        XCTAssertEqual(store.sidebarAssetCounts.all, 0)
        XCTAssertTrue(store.visibleAssets.isEmpty)

        let source = try environment.writeOnePixelPNG(named: "cached.png")
        try await store.importItems(from: [source])
        let assetID = try XCTUnwrap(store.assets.first?.id)
        XCTAssertEqual(store.visibleAssets.map(\.id), [assetID])
        XCTAssertEqual(store.sidebarAssetCounts.all, 1)

        store.selectSidebarItem(id: "favorites")
        XCTAssertTrue(store.visibleAssets.isEmpty)
        try store.toggleFavorite(for: assetID)
        XCTAssertEqual(store.visibleAssets.map(\.id), [assetID])
        XCTAssertEqual(store.sidebarAssetCounts.favorites, 1)

        try store.createFolder(name: "Cached Folder")
        let folderID = try XCTUnwrap(store.folders.first?.id)
        store.selectSidebarItem(id: "folder-\(folderID)")
        XCTAssertTrue(store.visibleAssets.isEmpty)
        try store.assignAssets(ids: [assetID], to: folderID)
        XCTAssertEqual(store.visibleAssets.map(\.id), [assetID])
        XCTAssertEqual(store.sidebarAssetCounts.folders[folderID], 1)

        store.selectAsset(id: assetID)
        try store.updateSelectedTags(["Cached Tag"])
        let tagID = try XCTUnwrap(store.assets.first?.tags.first?.id)
        store.selectSidebarItem(id: "tag-\(tagID)")
        XCTAssertEqual(store.visibleAssets.map(\.id), [assetID])

        try store.updateSelectedTags(["Other Tag"])
        XCTAssertTrue(store.visibleAssets.isEmpty)

        store.selectSidebarItem(id: "trash")
        XCTAssertTrue(store.visibleAssets.isEmpty)
        try store.moveAssetToTrash(id: assetID)
        XCTAssertEqual(store.visibleAssets.map(\.id), [assetID])
        XCTAssertEqual(store.sidebarAssetCounts.trash, 1)

        try store.restoreAssets(ids: [assetID])
        XCTAssertTrue(store.visibleAssets.isEmpty)
        XCTAssertEqual(store.sidebarAssetCounts.all, 1)
        XCTAssertEqual(store.sidebarAssetCounts.trash, 0)
    }

    private static func makeAsset(
        id: AssetItem.ID,
        libraryID: AssetLibrary.ID,
        displayName: String,
        fileExtension: String,
        isFavorite: Bool = false
    ) -> AssetItem {
        AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: displayName,
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/Samples/\(id).\(fileExtension)"),
            kind: .image,
            fileExtension: fileExtension,
            byteSize: 1,
            contentHash: id,
            dimensions: AssetDimensions(width: 1, height: 1),
            tags: [],
            isFavorite: isFavorite,
            importedAt: Date(timeIntervalSince1970: id == "alpha" ? 0 : 1)
        )
    }
}
