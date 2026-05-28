import Foundation
import XCTest
@testable import Momento

@MainActor
final class LibraryStoreAssetsVersionTests: XCTestCase {
    func testAssetsVersionAdvancesForPersistentAssetMutations() async throws {
        let environment = try AssetsVersionTestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let afterCreate = store.assetsVersion
        let source = try environment.writeOnePixelPNG(named: "versioned.png")
        try await store.importItems(from: [source])
        XCTAssertEqual(store.assetsVersion, afterCreate + 1)

        let asset = try XCTUnwrap(store.assets.first)

        let afterImport = store.assetsVersion
        try store.toggleFavorite(for: asset.id)
        XCTAssertEqual(store.assetsVersion, afterImport + 1)

        let afterFavorite = store.assetsVersion
        try store.renameAsset(id: asset.id, to: "Versioned Rename")
        XCTAssertEqual(store.assetsVersion, afterFavorite + 1)

        let afterRename = store.assetsVersion
        try store.updateNote("Versioned note", forAssetID: asset.id)
        XCTAssertEqual(store.assetsVersion, afterRename + 1)

        let afterNote = store.assetsVersion
        _ = try store.refreshThumbnail(for: asset.id)
        XCTAssertEqual(store.assetsVersion, afterNote + 1)

        try store.createFolder(name: "Versioned Folder")
        let folder = try XCTUnwrap(store.folders.first)

        let afterFolderCreate = store.assetsVersion
        try store.assignAssets(ids: [asset.id], to: folder.id)
        XCTAssertEqual(store.assetsVersion, afterFolderCreate + 1)

        let afterAssign = store.assetsVersion
        try store.deleteFolder(id: folder.id)
        XCTAssertEqual(store.assetsVersion, afterAssign + 1)

        let afterFolderDelete = store.assetsVersion
        try store.moveAssetToTrash(id: asset.id)
        XCTAssertEqual(store.assetsVersion, afterFolderDelete + 1)

        let afterMoveToTrash = store.assetsVersion
        try store.restoreAssets(ids: [asset.id])
        XCTAssertEqual(store.assetsVersion, afterMoveToTrash + 1)

        let afterRestore = store.assetsVersion
        try store.moveAssetToTrash(id: asset.id)
        XCTAssertEqual(store.assetsVersion, afterRestore + 1)

        let afterSecondMoveToTrash = store.assetsVersion
        try store.deleteAssetPermanently(id: asset.id)
        XCTAssertEqual(store.assetsVersion, afterSecondMoveToTrash + 1)
    }

    func testAssetsVersionAdvancesForInMemoryTagMutations() throws {
        let library = AssetLibrary(
            id: "library",
            name: "Library",
            createdAt: Date(timeIntervalSince1970: 0)
        )
        let asset = Self.makeAsset(
            id: "asset",
            libraryID: library.id,
            tags: [TagItem(id: "old", name: "Old")]
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [asset],
            loadRecentLibrary: false
        )

        store.selectAsset(id: asset.id)

        let afterSelect = store.assetsVersion
        try store.updateSelectedTags(["New"])
        XCTAssertEqual(store.assetsVersion, afterSelect + 1)

        let tagID = try XCTUnwrap(store.assets.first?.tags.first?.id)
        let afterUpdate = store.assetsVersion
        try store.renameTag(id: tagID, to: "Renamed")
        XCTAssertEqual(store.assetsVersion, afterUpdate + 1)

        let afterRename = store.assetsVersion
        try store.deleteTag(id: tagID)
        XCTAssertEqual(store.assetsVersion, afterRename + 1)
    }

    func testAssetsVersionAdvancesWhenClosingLibrary() {
        let library = AssetLibrary(
            id: "library",
            name: "Library",
            createdAt: Date(timeIntervalSince1970: 0)
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [Self.makeAsset(id: "asset", libraryID: library.id)],
            loadRecentLibrary: false
        )

        let beforeClose = store.assetsVersion
        store.closeCurrentLibrary()
        XCTAssertEqual(store.assetsVersion, beforeClose + 1)
    }

    private static func makeAsset(
        id: AssetItem.ID,
        libraryID: AssetLibrary.ID,
        tags: [TagItem] = []
    ) -> AssetItem {
        AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: id,
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/Samples/\(id).png"),
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: id,
            dimensions: AssetDimensions(width: 1, height: 1),
            tags: tags,
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
    }
}

private struct AssetsVersionTestEnvironment {
    let rootURL: URL
    let packageURL: URL
    let inputURL: URL
    let defaults: UserDefaults
    let defaultsSuiteName: String

    init() throws {
        rootURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        inputURL = rootURL.appendingPathComponent("input", isDirectory: true)
        packageURL = rootURL.appendingPathComponent("Test.momento", isDirectory: true)
        defaultsSuiteName = "MomentoAssetsVersionTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: defaultsSuiteName)!

        try FileManager.default.createDirectory(at: inputURL, withIntermediateDirectories: true)
    }

    func writeOnePixelPNG(named fileName: String) throws -> URL {
        let data = Data(base64Encoded: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")!
        let url = inputURL.appendingPathComponent(fileName)
        try data.write(to: url)
        return url
    }

    func cleanup() {
        try? FileManager.default.removeItem(at: rootURL)
        UserDefaults.standard.removePersistentDomain(forName: defaultsSuiteName)
    }
}
