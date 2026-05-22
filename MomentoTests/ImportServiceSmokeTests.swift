import CoreGraphics
import Foundation
import ImageIO
import UniformTypeIdentifiers
import XCTest
@testable import Momento

final class LibraryPackagePersistenceTests: XCTestCase {
    func testLibraryPackageCreationWritesManifestAndFolders() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage()
        let library = try storage.createLibraryPackage(at: environment.packageURL, name: "Test")

        XCTAssertEqual(library.name, "Test")
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("manifest.json").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("database").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("assets").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("thumbnails").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("previews").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("metadata/import-sessions").path))

        let manifestURL = environment.packageURL.appendingPathComponent("manifest.json")
        let manifest = try JSONDecoder.momento.decode(LibraryManifest.self, from: Data(contentsOf: manifestURL))
        XCTAssertEqual(manifest.schemaVersion, LibraryManifest.currentSchemaVersion)
        XCTAssertEqual(manifest.libraryID, library.id)
        XCTAssertEqual(manifest.displayName, "Test")
        XCTAssertEqual(manifest.createdAt.timeIntervalSince1970, library.createdAt.timeIntervalSince1970, accuracy: 1)
        XCTAssertEqual(manifest.updatedAt.timeIntervalSince1970, library.createdAt.timeIntervalSince1970, accuracy: 1)
    }

    func testLibraryPackageCreationAppendsMomentoExtensionWhenMissing() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage()
        let inputURL = environment.rootURL.appendingPathComponent("Workspace", isDirectory: true)
        let expectedPackageURL = environment.rootURL.appendingPathComponent("Workspace.momento", isDirectory: true)

        let library = try storage.createLibraryPackage(at: inputURL, name: "Workspace")

        XCTAssertEqual(library.packageURL, expectedPackageURL)
        XCTAssertTrue(FileManager.default.fileExists(atPath: expectedPackageURL.path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: inputURL.path))
    }

    func testLibraryPackageCreationRefusesExistingPackage() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage()
        _ = try storage.createLibraryPackage(at: environment.packageURL, name: "Test")

        XCTAssertThrowsError(try storage.createLibraryPackage(at: environment.packageURL, name: "Test")) { error in
            guard case LibraryStorageError.libraryPackageAlreadyExists = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }

        let manifestURL = environment.packageURL.appendingPathComponent("manifest.json")
        let manifest = try JSONDecoder.momento.decode(LibraryManifest.self, from: Data(contentsOf: manifestURL))
        XCTAssertEqual(manifest.displayName, "Test")
    }

    func testOpeningUnsupportedManifestVersionFails() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage()
        let library = try storage.createLibraryPackage(at: environment.packageURL, name: "Test")
        var manifest = LibraryManifest(library: library)
        manifest.schemaVersion = 999

        let manifestData = try JSONEncoder.momento.encode(manifest)
        try manifestData.write(to: environment.packageURL.appendingPathComponent("manifest.json"), options: .atomic)

        XCTAssertThrowsError(try storage.openLibraryPackage(at: environment.packageURL)) { error in
            guard case LibraryStorageError.unsupportedSchemaVersion(999) = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }
    }

    @MainActor
    func testImportPersistsAndDeduplicatesAssets() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "first.png")
        try await store.importItems(from: [source])
        try await store.importItems(from: [source])

        XCTAssertEqual(store.assets.count, 1)
        XCTAssertEqual(store.assets.first?.dimensions, AssetDimensions(width: 1, height: 1))
        XCTAssertNotNil(store.assets.first?.thumbnailURL)
        XCTAssertTrue(FileManager.default.fileExists(atPath: try XCTUnwrap(store.assets.first?.thumbnailURL).path))
        XCTAssertLessThanOrEqual(store.assets.first?.paletteColors.count ?? 0, 8)
        XCTAssertEqual(try environment.storedAssetFiles().count, 1)
    }

    @MainActor
    func testRefreshingImportedAssetThumbnailRegeneratesCacheFile() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "thumbnail.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)
        let originalThumbnailURL = try XCTUnwrap(asset.thumbnailURL)
        try FileManager.default.removeItem(at: originalThumbnailURL)

        let refreshedAsset = try XCTUnwrap(try store.refreshThumbnail(for: asset.id))

        XCTAssertEqual(refreshedAsset.id, asset.id)
        XCTAssertTrue(FileManager.default.fileExists(atPath: try XCTUnwrap(refreshedAsset.thumbnailURL).path))
    }

    @MainActor
    func testTogglingFavoritePersistsAcrossReloads() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "favorite.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)

        try store.toggleFavorite(for: asset.id)
        XCTAssertTrue(try XCTUnwrap(store.assets.first).isFavorite)
        store.selectSidebarItem(id: "favorites")
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertTrue(try XCTUnwrap(reopened.assets.first).isFavorite)

        store.selectAsset(id: asset.id)
        try store.toggleFavorite(for: asset.id)
        XCTAssertFalse(try XCTUnwrap(store.assets.first).isFavorite)
        XCTAssertTrue(store.visibleAssets.isEmpty)
        XCTAssertNil(store.selectedAssetID)
    }

    @MainActor
    func testRenamingImportedAssetDisplayNamePersistsAcrossReloads() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "original-title.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)

        try store.renameAsset(id: asset.id, to: "Renamed Asset")

        XCTAssertEqual(try XCTUnwrap(store.assets.first).displayName, "Renamed Asset")
        store.searchQuery = "Renamed"
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(try XCTUnwrap(reopened.assets.first).displayName, "Renamed Asset")
    }

    @MainActor
    func testUpdatingImportedAssetTagsPersistsAcrossReloads() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "tagged.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)

        store.selectAsset(id: asset.id)
        try store.updateSelectedTags(["Reference", "Mood"])

        XCTAssertEqual(store.assets.first?.tags.map(\.name), ["Reference", "Mood"])
        store.selectSidebarItem(id: "tag-mood")
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(reopened.assets.first?.tags.map(\.name), ["Reference", "Mood"])

        reopened.selectSidebarItem(id: "tag-reference")
        XCTAssertEqual(reopened.visibleAssets.map(\.id), [asset.id])
    }

    @MainActor
    func testTagManagementRenamesAndDeletesTagsAcrossAssets() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let firstSource = try environment.writeOnePixelPNG(named: "first-tagged.png")
        let secondSource = try environment.writeOnePixelJPEGWithExif(named: "second-tagged.jpg")
        try await store.importItems(from: [firstSource, secondSource])

        let firstAsset = try XCTUnwrap(store.assets.first { $0.displayName == "first-tagged" })
        let secondAsset = try XCTUnwrap(store.assets.first { $0.displayName == "second-tagged" })

        store.selectAsset(id: firstAsset.id)
        try store.updateSelectedTags(["Mood"])
        store.selectAsset(id: secondAsset.id)
        try store.updateSelectedTags(["Mood", "Travel"])

        XCTAssertEqual(store.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Mood:2",
            "Travel:1"
        ])

        try store.renameTag(id: "mood", to: "Vibe")

        XCTAssertEqual(store.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Travel:1",
            "Vibe:2"
        ])
        XCTAssertTrue(store.assets.allSatisfy { asset in
            !asset.tags.contains { $0.name == "Mood" }
        })

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(reopened.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Travel:1",
            "Vibe:2"
        ])

        try reopened.deleteTag(id: "mood")
        XCTAssertEqual(reopened.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Travel:1"
        ])
        XCTAssertTrue(reopened.assets.allSatisfy { asset in
            !asset.tags.contains { $0.name == "Vibe" }
        })
    }

    @MainActor
    func testImportPersistsExifMetadataAcrossReloads() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelJPEGWithExif(named: "exif.jpg")
        try await store.importItems(from: [source])
        let metadata = try XCTUnwrap(store.assets.first?.exifMetadata)

        XCTAssertEqual(metadata.pixelWidth, 1)
        XCTAssertEqual(metadata.pixelHeight, 1)
        XCTAssertEqual(metadata.dpiWidth, 300)
        XCTAssertEqual(metadata.dpiHeight, 300)
        XCTAssertEqual(metadata.colorModel, "RGB")
        XCTAssertEqual(metadata.profileName, "sRGB IEC61966-2.1")
        XCTAssertEqual(metadata.cameraMake, "NIKON CORPORATION")
        XCTAssertEqual(metadata.cameraModel, "NIKON Z 7_2")
        XCTAssertEqual(metadata.lensModel, "50.0 mm f/1.8")
        XCTAssertEqual(metadata.exposureTime, 0.004)
        XCTAssertEqual(metadata.focalLength, 50)
        XCTAssertEqual(metadata.isoSpeedRatings, [64])
        XCTAssertEqual(metadata.fNumber, 1.8)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(reopened.assets.first?.exifMetadata, metadata)
    }

    @MainActor
    func testMovingImportedAssetToTrashRemovesMetadataAndStoredFile() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage(applicationSupportRoot: environment.rootURL, trashURLs: [environment.trashURL])
        let store = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "trashed.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)
        let storedAssetURL = asset.storageURL
        let thumbnailURL = try XCTUnwrap(asset.thumbnailURL)

        try store.moveAssetToTrash(id: asset.id)

        XCTAssertTrue(store.assets.isEmpty)
        XCTAssertNil(store.selectedAssetID)
        XCTAssertFalse(FileManager.default.fileExists(atPath: storedAssetURL.path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: thumbnailURL.path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.trashURL.appendingPathComponent(storedAssetURL.lastPathComponent).path))

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertTrue(reopened.assets.isEmpty)
    }

    @MainActor
    func testReloadsImportedAssetsWithoutOriginalSource() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "reload.png")
        try await store.importItems(from: [source])
        try FileManager.default.removeItem(at: source)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)

        XCTAssertEqual(reopened.assets.count, 1)
        XCTAssertNil(reopened.assets.first?.originalURL)
        XCTAssertEqual(reopened.assets.first?.dimensions, AssetDimensions(width: 1, height: 1))
        XCTAssertNotNil(reopened.assets.first?.thumbnailURL)
        XCTAssertTrue(FileManager.default.fileExists(atPath: reopened.assets[0].storageURL.path))
    }

    @MainActor
    func testClosingCurrentLibraryReturnsToWelcomeStateWithoutDeletingPackage() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        store.searchQuery = "asset"

        store.closeCurrentLibrary()

        XCTAssertNil(store.currentLibrary)
        XCTAssertTrue(store.libraries.isEmpty)
        XCTAssertTrue(store.assets.isEmpty)
        XCTAssertNil(store.selectedAssetID)
        XCTAssertEqual(store.searchQuery, "")
        XCTAssertEqual(store.sidebarItemID(), "all-assets")
        XCTAssertNil(store.libraryErrorMessage)
        XCTAssertEqual(store.recentLibraries.first?.name, "Test")
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.path))
    }

    @MainActor
    func testSidebarNavigationSelectionsAreStable() throws {
        let library = AssetLibrary(id: "library", name: "Library", createdAt: Date(timeIntervalSince1970: 0))
        let taggedAsset = AssetItem(
            id: "tagged",
            libraryID: library.id,
            displayName: "Tagged",
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/Samples/tagged.png"),
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: "tagged",
            dimensions: nil,
            tags: [TagItem(name: "Reference")],
            folderIDs: ["folder"],
            isFavorite: true,
            importedAt: Date(timeIntervalSince1970: 0)
        )
        let untaggedAsset = AssetItem(
            id: "untagged",
            libraryID: library.id,
            displayName: "Untagged",
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/Samples/untagged.png"),
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: "untagged",
            dimensions: nil,
            tags: [],
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [taggedAsset, untaggedAsset],
            loadRecentLibrary: false
        )

        store.selectSidebarItem(id: "untagged")
        XCTAssertEqual(store.sidebarItemID(), "untagged")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["untagged"])

        store.selectSidebarItem(id: "uncategorized")
        XCTAssertEqual(store.sidebarItemID(), "uncategorized")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["untagged"])

        store.selectSidebarItem(id: "folder-folder")
        XCTAssertEqual(store.sidebarItemID(), "folder-folder")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["tagged"])

        store.selectSidebarItem(id: "tag-management")
        XCTAssertEqual(store.sidebarItemID(), "tag-management")
        XCTAssertTrue(store.visibleAssets.isEmpty)

        store.selectSidebarItem(id: "folder-management")
        XCTAssertEqual(store.sidebarItemID(), "folder-management")
        XCTAssertTrue(store.visibleAssets.isEmpty)
    }

    @MainActor
    func testMissingRecentLibraryIsPrunedOnLaunchAndReportsError() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        store.closeCurrentLibrary()
        try FileManager.default.removeItem(at: environment.packageURL)

        let relaunched = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults)
        )

        XCTAssertNil(relaunched.currentLibrary)
        XCTAssertTrue(relaunched.recentLibraries.isEmpty)
        XCTAssertEqual(relaunched.libraryErrorMessage, LibraryStoreError.missingRecentLibrary.errorDescription)
    }

    @MainActor
    func testTrashedRecentLibraryIsPrunedOnLaunchAndReportsError() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        try FileManager.default.createDirectory(at: environment.trashURL, withIntermediateDirectories: true)
        let trashedPackageURL = environment.trashURL.appendingPathComponent("Test.momento", isDirectory: true)
        let storage = LibraryStorage(applicationSupportRoot: environment.rootURL, trashURLs: [environment.trashURL])
        let store = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: trashedPackageURL)
        let recentID = try XCTUnwrap(store.recentLibraries.first?.id)
        store.closeCurrentLibrary()

        let relaunched = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults)
        )

        XCTAssertNil(relaunched.currentLibrary)
        XCTAssertTrue(FileManager.default.fileExists(atPath: trashedPackageURL.path))
        XCTAssertTrue(relaunched.recentLibraries.isEmpty)
        XCTAssertFalse(relaunched.recentLibraries.contains { $0.id == recentID })
        XCTAssertEqual(relaunched.libraryErrorMessage, LibraryStoreError.missingRecentLibrary.errorDescription)
    }

    @MainActor
    func testOpeningMissingRecentLibraryPrunesReferenceAndThrowsUnavailableError() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        let recentID = try XCTUnwrap(store.recentLibraries.first?.id)
        store.closeCurrentLibrary()
        try FileManager.default.removeItem(at: environment.packageURL)

        XCTAssertThrowsError(try store.openRecentLibrary(id: recentID)) { error in
            guard let storeError = error as? LibraryStoreError,
                  case .missingRecentLibrary = storeError else {
                return XCTFail("Unexpected error: \(error)")
            }
        }
        XCTAssertTrue(store.recentLibraries.isEmpty)
    }

    @MainActor
    func testValidatingMissingCurrentLibraryClosesLibraryAndReportsError() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        store.searchQuery = "asset"
        try FileManager.default.removeItem(at: environment.packageURL)

        try store.validateCurrentLibraryAvailability()

        XCTAssertNil(store.currentLibrary)
        XCTAssertTrue(store.libraries.isEmpty)
        XCTAssertTrue(store.assets.isEmpty)
        XCTAssertNil(store.selectedAssetID)
        XCTAssertEqual(store.searchQuery, "")
        XCTAssertTrue(store.recentLibraries.isEmpty)
        XCTAssertEqual(store.libraryErrorMessage, LibraryStorageError.missingLibraryPackage.errorDescription)
    }

    @MainActor
    func testClearingCurrentLibraryCachesRemovesGeneratedCacheFilesAndReloadsLibrary() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let thumbnailURL = environment.packageURL
            .appendingPathComponent("thumbnails", isDirectory: true)
            .appendingPathComponent("cached-thumb.dat")
        let previewURL = environment.packageURL
            .appendingPathComponent("previews", isDirectory: true)
            .appendingPathComponent("cached-preview.dat")
        try Data("thumbnail".utf8).write(to: thumbnailURL)
        try Data("preview".utf8).write(to: previewURL)

        try store.clearCachesAndReloadCurrentLibrary()

        XCTAssertEqual(store.currentLibrary?.name, "Test")
        XCTAssertFalse(FileManager.default.fileExists(atPath: thumbnailURL.path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: previewURL.path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("thumbnails").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("previews").path))
    }

    @MainActor
    func testFoldersPersistNestedHierarchyAndDeleteDescendants() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        try store.createFolder(name: "Jobs")
        let rootID = try XCTUnwrap(store.folders.first?.id)
        try store.createFolder(name: "Mantis", parentID: rootID)

        XCTAssertEqual(store.folders.map(\.name), ["Jobs", "Mantis"])
        XCTAssertEqual(store.folders.first { $0.name == "Mantis" }?.parentID, rootID)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(reopened.folders.map(\.name), ["Jobs", "Mantis"])
        XCTAssertEqual(reopened.folders.first { $0.name == "Mantis" }?.parentID, rootID)

        let childID = try XCTUnwrap(reopened.folders.first { $0.name == "Mantis" }?.id)
        try reopened.renameFolder(id: childID, to: "Design")
        XCTAssertEqual(reopened.folders.map(\.name), ["Jobs", "Design"])

        let renamed = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try renamed.openLibrary(at: environment.packageURL)
        XCTAssertEqual(renamed.folders.map(\.name), ["Jobs", "Design"])

        try renamed.deleteFolder(id: rootID)
        XCTAssertTrue(renamed.folders.isEmpty)
        XCTAssertEqual(renamed.sidebarItemID(), "all-assets")
    }

    @MainActor
    func testRenamingRecentLibraryUpdatesManifestCurrentLibraryAndRecentReference() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        let libraryID = try XCTUnwrap(store.currentLibrary?.id)

        try store.renameRecentLibrary(id: libraryID, to: "Renamed")

        XCTAssertEqual(store.currentLibrary?.name, "Renamed")
        XCTAssertEqual(store.recentLibraries.first?.name, "Renamed")

        let manifestURL = environment.packageURL.appendingPathComponent("manifest.json")
        let manifest = try JSONDecoder.momento.decode(LibraryManifest.self, from: Data(contentsOf: manifestURL))
        XCTAssertEqual(manifest.libraryID, libraryID)
        XCTAssertEqual(manifest.displayName, "Renamed")
    }

    @MainActor
    func testDeletingRecentLibraryRemovesPackageAndClosesCurrentLibrary() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage(applicationSupportRoot: environment.rootURL, trashURLs: [environment.trashURL])
        let store = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        let libraryID = try XCTUnwrap(store.currentLibrary?.id)

        try store.deleteRecentLibrary(id: libraryID)

        XCTAssertNil(store.currentLibrary)
        XCTAssertTrue(store.recentLibraries.isEmpty)
        XCTAssertFalse(FileManager.default.fileExists(atPath: environment.packageURL.path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: environment.trashURL.appendingPathComponent("Test.momento").path))
    }

    @MainActor
    func testMovingRecentLibraryPersistsManualOrder() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        let alphaURL = environment.rootURL.appendingPathComponent("Alpha.momento", isDirectory: true)
        let betaURL = environment.rootURL.appendingPathComponent("Beta.momento", isDirectory: true)
        let gammaURL = environment.rootURL.appendingPathComponent("Gamma.momento", isDirectory: true)

        try store.createLibrary(at: alphaURL)
        try store.createLibrary(at: betaURL)
        try store.createLibrary(at: gammaURL)

        let alphaID = try XCTUnwrap(store.recentLibraries.first { $0.name == "Alpha" }?.id)
        let gammaID = try XCTUnwrap(store.recentLibraries.first { $0.name == "Gamma" }?.id)

        try store.moveRecentLibrary(id: gammaID, relativeTo: alphaID, insertAfterTarget: true)

        XCTAssertEqual(store.recentLibraries.map(\.name), ["Beta", "Alpha", "Gamma"])

        let relaunched = LibraryStore(
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        XCTAssertEqual(relaunched.recentLibraries.map(\.name), ["Beta", "Alpha", "Gamma"])
    }
}

private struct TestEnvironment {
    let rootURL: URL
    let packageURL: URL
    let inputURL: URL
    let trashURL: URL
    let defaults: UserDefaults
    let defaultsSuiteName: String

    init() throws {
        rootURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        inputURL = rootURL.appendingPathComponent("input", isDirectory: true)
        packageURL = rootURL.appendingPathComponent("Test.momento", isDirectory: true)
        trashURL = rootURL.appendingPathComponent(".Trash", isDirectory: true)
        defaultsSuiteName = "MomentoTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: defaultsSuiteName)!

        try FileManager.default.createDirectory(at: inputURL, withIntermediateDirectories: true)
    }

    func writeOnePixelPNG(named fileName: String) throws -> URL {
        let data = Data(base64Encoded: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")!
        let url = inputURL.appendingPathComponent(fileName)
        try data.write(to: url)
        return url
    }

    func writeOnePixelJPEGWithExif(named fileName: String) throws -> URL {
        let url = inputURL.appendingPathComponent(fileName)
        let colorSpace = CGColorSpaceCreateDeviceRGB()
        let pixel = Data([255, 0, 0, 255])
        guard
            let provider = CGDataProvider(data: pixel as CFData),
            let image = CGImage(
                width: 1,
                height: 1,
                bitsPerComponent: 8,
                bitsPerPixel: 32,
                bytesPerRow: 4,
                space: colorSpace,
                bitmapInfo: CGBitmapInfo(rawValue: CGImageAlphaInfo.premultipliedLast.rawValue),
                provider: provider,
                decode: nil,
                shouldInterpolate: false,
                intent: .defaultIntent
            ),
            let destination = CGImageDestinationCreateWithURL(
                url as CFURL,
                UTType.jpeg.identifier as CFString,
                1,
                nil
            )
        else {
            throw CocoaError(.fileWriteUnknown)
        }

        let properties: [CFString: Any] = [
            kCGImagePropertyDPIWidth: 300,
            kCGImagePropertyDPIHeight: 300,
            kCGImagePropertyColorModel: kCGImagePropertyColorModelRGB,
            kCGImagePropertyProfileName: "sRGB IEC61966-2.1",
            kCGImagePropertyTIFFDictionary: [
                kCGImagePropertyTIFFMake: "NIKON CORPORATION",
                kCGImagePropertyTIFFModel: "NIKON Z 7_2"
            ],
            kCGImagePropertyExifDictionary: [
                kCGImagePropertyExifLensModel: "50.0 mm f/1.8",
                kCGImagePropertyExifExposureTime: 0.004,
                kCGImagePropertyExifFocalLength: 50.0,
                kCGImagePropertyExifISOSpeedRatings: [64],
                kCGImagePropertyExifFNumber: 1.8
            ]
        ]

        CGImageDestinationAddImage(destination, image, properties as CFDictionary)
        guard CGImageDestinationFinalize(destination) else {
            throw CocoaError(.fileWriteUnknown)
        }
        return url
    }

    func storedAssetFiles() throws -> [URL] {
        let assetsURL = packageURL.appendingPathComponent("assets", isDirectory: true)
        guard let enumerator = FileManager.default.enumerator(
            at: assetsURL,
            includingPropertiesForKeys: [.isRegularFileKey]
        ) else {
            return []
        }

        return try enumerator.compactMap { item in
            guard let url = item as? URL else {
                return nil
            }
            let values = try url.resourceValues(forKeys: [.isRegularFileKey])
            return values.isRegularFile == true ? url : nil
        }
    }

    func cleanup() {
        defaults.removePersistentDomain(forName: defaultsSuiteName)
        try? FileManager.default.removeItem(at: rootURL)
    }
}
