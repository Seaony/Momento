// 中文注释：本测试文件覆盖资源库、导入、元数据、回收站、文件夹等数据生命周期的端到端烟雾路径。
import CoreGraphics
import CoreData
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
    func testExportingAndImportingLibraryPackages() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage(applicationSupportRoot: environment.rootURL)
        let sourceStore = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try sourceStore.createLibrary(at: environment.packageURL)
        let sourceLibrary = try XCTUnwrap(sourceStore.currentLibrary)

        let source = try environment.writeOnePixelPNG(named: "portable.png")
        try await sourceStore.importItems(from: [source])

        let previewURL = environment.packageURL
            .appendingPathComponent("previews", isDirectory: true)
            .appendingPathComponent("portable.preview")
        try FileManager.default.createDirectory(at: previewURL.deletingLastPathComponent(), withIntermediateDirectories: true)
        try Data([1, 2, 3]).write(to: previewURL)

        let exportURL = environment.rootURL.appendingPathComponent("Exported.momento", isDirectory: true)
        let exportedURL = try storage.exportLibraryPackage(sourceLibrary, to: exportURL)

        XCTAssertEqual(exportedURL, exportURL)
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURL.appendingPathComponent("manifest.json").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURL.appendingPathComponent("database/library.sqlite").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURL.appendingPathComponent("assets").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURL.appendingPathComponent("thumbnails").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: exportedURL.appendingPathComponent("previews/portable.preview").path))

        let exportedManifest = try JSONDecoder.momento.decode(
            LibraryManifest.self,
            from: Data(contentsOf: exportedURL.appendingPathComponent("manifest.json"))
        )
        XCTAssertEqual(exportedManifest.libraryID, sourceLibrary.id)
        XCTAssertEqual(try storage.validateLibraryPackage(at: exportedURL).id, sourceLibrary.id)

        let importRoot = environment.rootURL.appendingPathComponent("Imported", isDirectory: true)
        let importedLibrary = try storage.importLibraryPackage(from: exportedURL, to: importRoot)
        let importedPackageURL = importRoot.appendingPathComponent(exportedURL.lastPathComponent, isDirectory: true)
        XCTAssertEqual(importedLibrary.id, sourceLibrary.id)
        XCTAssertEqual(importedLibrary.packageURL, importedPackageURL)
        XCTAssertTrue(FileManager.default.fileExists(atPath: importedPackageURL.appendingPathComponent("database/library.sqlite").path))

        XCTAssertThrowsError(try storage.importLibraryPackage(from: exportedURL, to: importRoot)) { error in
            guard case LibraryStorageError.libraryPackageAlreadyExists = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }

        let badPackageURL = environment.rootURL.appendingPathComponent("Bad.momento", isDirectory: true)
        try FileManager.default.createDirectory(at: badPackageURL, withIntermediateDirectories: true)
        var badManifest = LibraryManifest(library: sourceLibrary)
        badManifest.schemaVersion = 999
        try JSONEncoder.momento.encode(badManifest)
            .write(to: badPackageURL.appendingPathComponent("manifest.json"), options: .atomic)

        XCTAssertThrowsError(
            try storage.importLibraryPackage(
                from: badPackageURL,
                to: environment.rootURL.appendingPathComponent("BadImport", isDirectory: true)
            )
        ) { error in
            guard case LibraryStorageError.unsupportedSchemaVersion(999) = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }

        XCTAssertThrowsError(
            try sourceStore.importLibrary(
                from: exportedURL,
                destinationRootURL: environment.rootURL.appendingPathComponent("DuplicateImport", isDirectory: true)
            )
        ) { error in
            guard case LibraryStoreError.duplicateLibraryID = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }

        let importedDefaultsSuiteName = "\(environment.defaultsSuiteName).imported"
        let importedDefaults = try XCTUnwrap(UserDefaults(suiteName: importedDefaultsSuiteName))
        defer {
            importedDefaults.removePersistentDomain(forName: importedDefaultsSuiteName)
        }

        let importedStore = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: importedDefaults),
            loadRecentLibrary: false
        )
        try importedStore.importLibrary(
            from: exportedURL,
            destinationRootURL: environment.rootURL.appendingPathComponent("StoreImport", isDirectory: true)
        )

        XCTAssertEqual(importedStore.currentLibrary?.id, sourceLibrary.id)
        XCTAssertEqual(importedStore.recentLibraries.first?.id, sourceLibrary.id)
        XCTAssertEqual(importedStore.assets.map(\.displayName), ["portable"])
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
    func testImportedAssetPersistsCoreMetadata() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "core-metadata.png")
        try await store.importItems(from: [source])
        let imported = try XCTUnwrap(store.assets.first)

        XCTAssertEqual(imported.displayName, "core-metadata")
        XCTAssertEqual(imported.originalFileName, "core-metadata.png")
        XCTAssertEqual(imported.utiIdentifier, UTType.png.identifier)
        XCTAssertNil(imported.note)
        XCTAssertFalse(imported.isTrashed)
        XCTAssertNil(imported.trashedAt)
        XCTAssertEqual(imported.updatedAt.timeIntervalSince1970, imported.importedAt.timeIntervalSince1970, accuracy: 0.01)

        try await Task.sleep(nanoseconds: 10_000_000)
        try store.renameAsset(id: imported.id, to: "Renamed Metadata")
        let renamed = try XCTUnwrap(store.assets.first)

        XCTAssertEqual(renamed.displayName, "Renamed Metadata")
        XCTAssertEqual(renamed.originalFileName, "core-metadata.png")
        XCTAssertGreaterThan(renamed.updatedAt, imported.updatedAt)

        try await Task.sleep(nanoseconds: 10_000_000)
        try store.toggleFavorite(for: imported.id)
        let favorited = try XCTUnwrap(store.assets.first)

        XCTAssertTrue(favorited.isFavorite)
        XCTAssertGreaterThan(favorited.updatedAt, renamed.updatedAt)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        let reloaded = try XCTUnwrap(reopened.assets.first)

        XCTAssertEqual(reloaded.displayName, "Renamed Metadata")
        XCTAssertEqual(reloaded.originalFileName, "core-metadata.png")
        XCTAssertEqual(reloaded.utiIdentifier, UTType.png.identifier)
        XCTAssertNil(reloaded.note)
        XCTAssertFalse(reloaded.isTrashed)
        XCTAssertNil(reloaded.trashedAt)
        XCTAssertEqual(reloaded.updatedAt.timeIntervalSince1970, favorited.updatedAt.timeIntervalSince1970, accuracy: 0.01)
    }

    @MainActor
    func testImportedAssetPersistsSourcePageURL() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "linked-source.png")
        let sourcePageURL = try XCTUnwrap(URL(string: "https://example.com/articles/reference"))
        try await store.importItems(from: [source], sourcePageURL: sourcePageURL)

        XCTAssertEqual(store.assets.first?.sourcePageURL, sourcePageURL)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)

        XCTAssertEqual(reopened.assets.first?.sourcePageURL, sourcePageURL)
    }

    @MainActor
    func testImportItemsReportsProgressForFolderImport() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let folderURL = environment.inputURL.appendingPathComponent("Batch", isDirectory: true)
        _ = try environment.writeOnePixelPNG(named: "first.png", in: folderURL)
        _ = try environment.writeOnePixelJPEGWithExif(named: "second.jpg", in: folderURL)
        let recorder = ImportProgressRecorder()

        try await store.importItems(
            from: [folderURL],
            progressHandler: { progress in
                await recorder.append(progress)
            }
        )

        let values = await recorder.snapshot()
        XCTAssertEqual(values.first?.phase, .preparing)
        XCTAssertTrue(values.contains { progress in
            progress.phase == .importing
                && progress.totalFileCount == 2
                && progress.processedFileCount == 0
        })

        let finalImportingProgress = try XCTUnwrap(values.last { $0.phase == .importing })
        XCTAssertEqual(finalImportingProgress.totalFileCount, 2)
        XCTAssertEqual(finalImportingProgress.processedFileCount, 2)
        XCTAssertEqual(finalImportingProgress.importedFileCount, 2)
        XCTAssertEqual(finalImportingProgress.skippedFileCount, 0)
        XCTAssertEqual(values.last?.phase, .finalizing)
        XCTAssertEqual(store.assets.count, 2)
    }

    func testOpeningPreV3LibraryMigratesMetadata() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let storage = LibraryStorage(applicationSupportRoot: environment.rootURL)
        let library = try storage.createLibraryPackage(at: environment.packageURL, name: "Test")
        let fixture = try environment.writePreV3MetadataStore(for: library, storage: storage)

        let metadataStore = try LibraryMetadataStore(library: library, storage: storage)
        let asset = try XCTUnwrap(try metadataStore.loadAssets().first)

        XCTAssertEqual(asset.id, fixture.assetID)
        XCTAssertEqual(asset.updatedAt.timeIntervalSince1970, fixture.importedAt.timeIntervalSince1970, accuracy: 0.01)
        XCTAssertEqual(asset.utiIdentifier, "public.data")
        XCTAssertNil(asset.note)
        XCTAssertFalse(asset.isTrashed)
        XCTAssertNil(asset.trashedAt)
        XCTAssertEqual(asset.tags.map(\.name), ["Legacy"])
        XCTAssertEqual(asset.folderIDs, [fixture.folderID])
        XCTAssertEqual(asset.paletteColors.map(\.hex), ["#123456"])

        XCTAssertEqual(try metadataStore.loadTags().map(\.name), ["Legacy"])
        let untagged = try metadataStore.setTagNames([], forAssetID: fixture.assetID)
        XCTAssertTrue(untagged.tags.isEmpty)
        XCTAssertEqual(try metadataStore.loadTags().map(\.name), ["Legacy"])
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
        let moodID = try XCTUnwrap(store.tagSummaries.first { $0.tag.name == "Mood" }?.tag.id)
        store.selectSidebarItem(id: "tag-\(moodID)")
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(reopened.assets.first?.tags.map(\.name), ["Reference", "Mood"])

        let referenceID = try XCTUnwrap(reopened.tagSummaries.first { $0.tag.name == "Reference" }?.tag.id)
        reopened.selectSidebarItem(id: "tag-\(referenceID)")
        XCTAssertEqual(reopened.visibleAssets.map(\.id), [asset.id])
    }

    @MainActor
    func testUpdatingAssetNotePersistsAcrossReloads() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let firstSource = try environment.writeOnePixelPNG(named: "note-a.png")
        let secondSource = try environment.writeOnePixelJPEGWithExif(named: "note-b.jpg")
        try await store.importItems(from: [firstSource, secondSource])
        let firstAsset = try XCTUnwrap(store.assets.first { $0.displayName == "note-a" })
        let secondAsset = try XCTUnwrap(store.assets.first { $0.displayName == "note-b" })

        try store.updateNote("First note", forAssetID: firstAsset.id)
        store.selectAsset(id: secondAsset.id)
        try store.updateSelectedNote("Second note")

        XCTAssertEqual(store.assets.first { $0.id == firstAsset.id }?.note, "First note")
        XCTAssertEqual(store.assets.first { $0.id == secondAsset.id }?.note, "Second note")

        store.selectAsset(id: firstAsset.id)
        XCTAssertEqual(store.selectedAsset?.note, "First note")
        store.selectAsset(id: secondAsset.id)
        XCTAssertEqual(store.selectedAsset?.note, "Second note")

        try store.updateNote("   ", forAssetID: firstAsset.id)
        XCTAssertNil(store.assets.first { $0.id == firstAsset.id }?.note)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)

        XCTAssertNil(reopened.assets.first { $0.id == firstAsset.id }?.note)
        XCTAssertEqual(reopened.assets.first { $0.id == secondAsset.id }?.note, "Second note")
    }

    @MainActor
    func testTagRecordsRenameAndDeleteAcrossAssets() async throws {
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

        let moodID = try XCTUnwrap(store.tagSummaries.first { $0.tag.name == "Mood" }?.tag.id)
        let travelID = try XCTUnwrap(store.tagSummaries.first { $0.tag.name == "Travel" }?.tag.id)

        XCTAssertThrowsError(try store.renameTag(id: travelID, to: " mood "))

        try store.renameTag(id: moodID, to: "Vibe")
        let vibeID = try XCTUnwrap(store.tagSummaries.first { $0.tag.name == "Vibe" }?.tag.id)

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

        reopened.selectAsset(id: secondAsset.id)
        try reopened.updateSelectedTags(["Vibe"])
        XCTAssertEqual(reopened.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Travel:0",
            "Vibe:2"
        ])

        let withZeroCountTag = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try withZeroCountTag.openLibrary(at: environment.packageURL)
        XCTAssertEqual(withZeroCountTag.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Travel:0",
            "Vibe:2"
        ])

        try withZeroCountTag.deleteTag(id: vibeID)
        XCTAssertEqual(withZeroCountTag.tagSummaries.map { "\($0.tag.name):\($0.assetCount)" }, [
            "Travel:0"
        ])
        XCTAssertEqual(withZeroCountTag.assets.count, 2)
        XCTAssertTrue(withZeroCountTag.assets.allSatisfy { asset in
            !asset.tags.contains { $0.name == "Vibe" }
        })
    }

    @MainActor
    func testAssigningMultipleDraggedAssetsToFolderIsIdempotent() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let firstSource = try environment.writeOnePixelPNG(named: "dragged-a.png")
        let secondSource = try environment.writeOnePixelJPEGWithExif(named: "dragged-b.jpg")
        try await store.importItems(from: [firstSource, secondSource])

        let firstAsset = try XCTUnwrap(store.assets.first { $0.displayName == "dragged-a" })
        let secondAsset = try XCTUnwrap(store.assets.first { $0.displayName == "dragged-b" })
        let assetIDs = Set([firstAsset.id, secondAsset.id])

        try store.createFolder(name: "Inbox")
        let inboxID = try XCTUnwrap(store.folders.first { $0.name == "Inbox" }?.id)
        try store.createFolder(name: "Archive")
        let archiveFolderID = try XCTUnwrap(store.folders.first { $0.name == "Archive" }?.id)

        store.selectSidebarItem(id: "all-assets")
        store.selectAsset(id: firstAsset.id)
        try store.updateSelectedTags(["Drop", "Archive"])
        let dropTagID = try XCTUnwrap(store.tagSummaries.first { $0.tag.name == "Drop" }?.tag.id)
        let archiveTagID = try XCTUnwrap(store.tagSummaries.first { $0.tag.name == "Archive" }?.tag.id)

        store.selectSidebarItem(id: "all-assets")
        store.selectAssets(ids: assetIDs)
        XCTAssertEqual(store.selectedAssetIDs, assetIDs)

        try store.assignAssets(ids: assetIDs, to: inboxID)
        try store.assignAssets(ids: assetIDs, to: inboxID)

        XCTAssertEqual(store.selectedAssetIDs, assetIDs)
        XCTAssertTrue(try XCTUnwrap(store.assets.first { $0.id == firstAsset.id }).folderIDs.contains(inboxID))
        XCTAssertTrue(try XCTUnwrap(store.assets.first { $0.id == secondAsset.id }).folderIDs.contains(inboxID))

        try store.addTag(id: dropTagID, toAssets: assetIDs)
        try store.addTag(id: dropTagID, toAssets: assetIDs)

        XCTAssertEqual(store.selectedAssetIDs, assetIDs)
        XCTAssertEqual(store.tagSummaries.first { $0.tag.id == dropTagID }?.assetCount, 2)
        XCTAssertTrue(store.assets.allSatisfy { asset in
            asset.tags.map(\.id).count == Set(asset.tags.map(\.id)).count
        })

        try store.addTag(named: "Batch", toAssets: assetIDs)
        XCTAssertEqual(store.tagSummaries.first { $0.tag.name == "Batch" }?.assetCount, 2)

        try store.removeTag(named: "Batch", fromAssets: [firstAsset.id])
        XCTAssertFalse(try XCTUnwrap(store.assets.first { $0.id == firstAsset.id }).tags.contains { $0.name == "Batch" })
        XCTAssertTrue(try XCTUnwrap(store.assets.first { $0.id == secondAsset.id }).tags.contains { $0.name == "Batch" })

        try store.removeTag(named: "Batch", fromAssets: assetIDs)
        XCTAssertTrue(store.assets.allSatisfy { asset in
            !asset.tags.contains { $0.name == "Batch" }
        })

        try store.moveAssetToTrash(id: secondAsset.id)
        try store.assignAssets(ids: assetIDs, to: archiveFolderID)
        try store.addTag(id: archiveTagID, toAssets: assetIDs)

        let updatedFirst = try XCTUnwrap(store.assets.first { $0.id == firstAsset.id })
        let trashedSecond = try XCTUnwrap(store.assets.first { $0.id == secondAsset.id })
        XCTAssertTrue(updatedFirst.folderIDs.contains(archiveFolderID))
        XCTAssertFalse(trashedSecond.folderIDs.contains(archiveFolderID))
        XCTAssertTrue(updatedFirst.tags.contains { $0.id == archiveTagID })
        XCTAssertFalse(trashedSecond.tags.contains { $0.id == archiveTagID })
        XCTAssertTrue(trashedSecond.isTrashed)
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
    func testMovingAssetToTrashSoftDeletesAndRestores() async throws {
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
        try store.createFolder(name: "References")
        let folderID = try XCTUnwrap(store.folders.first?.id)
        try store.assignAssets(ids: [asset.id], to: folderID)
        store.selectSidebarItem(id: "all-assets")
        store.selectAsset(id: asset.id)

        let assignedAsset = try XCTUnwrap(store.assets.first)
        let storedAssetURL = asset.storageURL
        let thumbnailURL = try XCTUnwrap(assignedAsset.thumbnailURL)

        try store.moveAssetToTrash(id: asset.id)

        let trashedAsset = try XCTUnwrap(store.assets.first)
        XCTAssertTrue(trashedAsset.isTrashed)
        XCTAssertNotNil(trashedAsset.trashedAt)
        XCTAssertEqual(trashedAsset.folderIDs, [folderID])
        XCTAssertTrue(FileManager.default.fileExists(atPath: storedAssetURL.path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: thumbnailURL.path))
        XCTAssertTrue(store.visibleAssets.isEmpty)
        XCTAssertNil(store.selectedAssetID)

        store.selectSidebarItem(id: "trash")
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])

        try store.restoreAssets(ids: [asset.id])
        let restoredAsset = try XCTUnwrap(store.assets.first)
        XCTAssertFalse(restoredAsset.isTrashed)
        XCTAssertNil(restoredAsset.trashedAt)
        XCTAssertEqual(restoredAsset.folderIDs, [folderID])
        XCTAssertTrue(store.visibleAssets.isEmpty)

        store.selectSidebarItem(id: "all-assets")
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            storage: storage,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        let reopenedAsset = try XCTUnwrap(reopened.assets.first)
        XCTAssertFalse(reopenedAsset.isTrashed)
        XCTAssertNil(reopenedAsset.trashedAt)
        XCTAssertEqual(reopenedAsset.folderIDs, [folderID])
    }

    @MainActor
    func testMovingMultipleAssetsToTrashSoftDeletesInBatch() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let firstSource = try environment.writeOnePixelPNG(named: "batch-trash-1.png")
        let secondSource = try environment.writeOnePixelPNG(named: "batch-trash-2.png")
        try await store.importItems(from: [firstSource, secondSource])
        let assetIDs = Set(store.assets.map(\.id))
        store.selectSidebarItem(id: "all-assets")
        store.selectAssets(ids: assetIDs)

        try store.moveAssetsToTrash(ids: assetIDs)

        XCTAssertEqual(Set(store.assets.map(\.id)), assetIDs)
        XCTAssertTrue(store.assets.allSatisfy(\.isTrashed))
        XCTAssertTrue(store.assets.allSatisfy { $0.trashedAt != nil })
        XCTAssertTrue(store.visibleAssets.isEmpty)
        XCTAssertTrue(store.selectedAssetIDs.isEmpty)
        XCTAssertNil(store.selectedAssetID)

        store.selectSidebarItem(id: "trash")
        XCTAssertEqual(Set(store.visibleAssets.map(\.id)), assetIDs)
    }

    @MainActor
    func testTrashedAssetReimportRestoresExistingRecord() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "restore-duplicate.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)
        try store.createFolder(name: "Duplicates")
        let folderID = try XCTUnwrap(store.folders.first?.id)
        try store.assignAssets(ids: [asset.id], to: folderID)

        try store.moveAssetToTrash(id: asset.id)
        XCTAssertEqual(store.assets.count, 1)
        XCTAssertTrue(try XCTUnwrap(store.assets.first).isTrashed)

        try await store.importItems(from: [source])

        let restoredAsset = try XCTUnwrap(store.assets.first)
        XCTAssertEqual(store.assets.count, 1)
        XCTAssertEqual(restoredAsset.id, asset.id)
        XCTAssertFalse(restoredAsset.isTrashed)
        XCTAssertNil(restoredAsset.trashedAt)
        XCTAssertEqual(restoredAsset.folderIDs, [folderID])
        XCTAssertEqual(store.visibleAssets.map(\.id), [asset.id])
    }

    @MainActor
    func testEmptyTrashPermanentlyDeletesMetadataAndStoredFiles() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let source = try environment.writeOnePixelPNG(named: "empty-trash.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)
        store.selectAsset(id: asset.id)
        try store.updateSelectedTags(["Disposable"])
        let storedAssetURL = asset.storageURL
        let thumbnailURL = try XCTUnwrap(asset.thumbnailURL)
        let previewURL = environment.packageURL
            .appendingPathComponent("previews", isDirectory: true)
            .appendingPathComponent("\(asset.contentHash)-preview.dat")
        try Data("preview".utf8).write(to: previewURL)

        try store.moveAssetToTrash(id: asset.id)
        try store.emptyTrash()

        XCTAssertTrue(store.assets.isEmpty)
        XCTAssertTrue(store.tagSummaries.allSatisfy { $0.assetCount == 0 })
        XCTAssertFalse(FileManager.default.fileExists(atPath: storedAssetURL.path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: thumbnailURL.path))
        XCTAssertFalse(FileManager.default.fileExists(atPath: previewURL.path))

        let reopened = LibraryStore(
            defaultViewMode: .grid,
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
    func testVisibleAssetsApplyToolbarFiltersAndSorting() throws {
        let library = AssetLibrary(id: "library", name: "Library", createdAt: Date(timeIntervalSince1970: 0))
        let warmTag = TagItem(id: "warm", name: "Warm")
        let coolTag = TagItem(id: "cool", name: "Cool")
        let smallPNG = toolbarFilterAsset(
            id: "small-png",
            libraryID: library.id,
            displayName: "Beta",
            fileExtension: "png",
            byteSize: 100,
            tag: warmTag,
            colorHex: "#FF0000",
            importedAt: Date(timeIntervalSince1970: 10)
        )
        let largeJPG = toolbarFilterAsset(
            id: "large-jpg",
            libraryID: library.id,
            displayName: "Alpha",
            fileExtension: "jpg",
            byteSize: 300,
            tag: coolTag,
            colorHex: "#0000FF",
            importedAt: Date(timeIntervalSince1970: 20)
        )
        let mediumPNG = toolbarFilterAsset(
            id: "medium-png",
            libraryID: library.id,
            displayName: "Gamma",
            fileExtension: "png",
            byteSize: 200,
            tag: warmTag,
            colorHex: "#00FF00",
            importedAt: Date(timeIntervalSince1970: 30)
        )
        let darkPNG = toolbarFilterAsset(
            id: "dark-png",
            libraryID: library.id,
            displayName: "Dark",
            fileExtension: "png",
            byteSize: 150,
            tag: coolTag,
            colorHex: "#101010",
            importedAt: Date(timeIntervalSince1970: 40),
            paletteColors: [
                (hex: "#101010", coverage: 0.7),
                (hex: "#FF0000", coverage: 0.05)
            ]
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [mediumPNG, largeJPG, smallPNG, darkPNG],
            loadRecentLibrary: false
        )

        XCTAssertEqual(store.availableFilterFileExtensions, ["jpg", "png"])
        XCTAssertEqual(store.availableFilterColorCategories, AssetColorCategory.allCases)

        store.toggleFilterFileExtension("PNG")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["dark-png", "medium-png", "small-png"])

        store.toggleFilterTag(id: warmTag.id)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["medium-png", "small-png"])

        store.toggleFilterColorCategory(.green)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["medium-png"])

        store.clearAssetFilters()
        store.toggleFilterColorCategory(.red)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["small-png"])

        store.clearAssetFilters()
        store.selectAsset(id: smallPNG.id)
        store.toggleFilterFileExtension("jpg")
        XCTAssertEqual(store.visibleAssets.map(\.id), ["large-jpg"])
        XCTAssertNil(store.selectedAssetID)

        store.clearAssetFilters()
        store.setSortOption(.fileSize)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["large-jpg", "medium-png", "dark-png", "small-png"])

        store.setSortOption(.fileSize)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["small-png", "dark-png", "medium-png", "large-jpg"])

        store.setSortOption(.name)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["medium-png", "dark-png", "small-png", "large-jpg"])

        store.setSortOption(.name)
        XCTAssertEqual(store.visibleAssets.map(\.id), ["large-jpg", "small-png", "dark-png", "medium-png"])
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
    func testMovingFoldersPersistsManualOrderAndHierarchy() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        try store.createFolder(name: "Alpha")
        let alphaID = try XCTUnwrap(store.folders.first { $0.name == "Alpha" }?.id)
        try store.createFolder(name: "Beta")
        let betaID = try XCTUnwrap(store.folders.first { $0.name == "Beta" }?.id)
        try store.createFolder(name: "Gamma")
        let gammaID = try XCTUnwrap(store.folders.first { $0.name == "Gamma" }?.id)

        try store.moveFolder(id: gammaID, toParentID: nil, relativeTo: alphaID, insertAfterTarget: false)
        XCTAssertEqual(folderNames(in: store.folders, parentID: nil), ["Gamma", "Alpha", "Beta"])

        try store.moveFolder(id: alphaID, toParentID: betaID, relativeTo: nil, insertAfterTarget: false)
        XCTAssertEqual(folderNames(in: store.folders, parentID: nil), ["Gamma", "Beta"])
        XCTAssertEqual(folderNames(in: store.folders, parentID: betaID), ["Alpha"])

        try store.moveFolder(id: alphaID, toParentID: nil, relativeTo: betaID, insertAfterTarget: true)
        XCTAssertEqual(folderNames(in: store.folders, parentID: nil), ["Gamma", "Beta", "Alpha"])
        XCTAssertNil(store.folders.first { $0.id == alphaID }?.parentID)

        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(folderNames(in: reopened.folders, parentID: nil), ["Gamma", "Beta", "Alpha"])
    }

    @MainActor
    func testMovingFolderIntoDescendantIsRejected() throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        try store.createFolder(name: "Parent")
        let parentID = try XCTUnwrap(store.folders.first { $0.name == "Parent" }?.id)
        try store.createFolder(name: "Child", parentID: parentID)
        let childID = try XCTUnwrap(store.folders.first { $0.name == "Child" }?.id)

        XCTAssertThrowsError(
            try store.moveFolder(id: parentID, toParentID: childID, relativeTo: nil, insertAfterTarget: false)
        ) { error in
            guard case LibraryMetadataError.invalidFolderMove = error else {
                return XCTFail("Unexpected error: \(error)")
            }
        }
        XCTAssertEqual(store.folders.first { $0.id == parentID }?.parentID, nil)
        XCTAssertEqual(store.folders.first { $0.id == childID }?.parentID, parentID)
    }

    @MainActor
    func testImportingFolderPreservesHierarchyAsVirtualFolders() async throws {
        let environment = try TestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let sourceRoot = environment.inputURL.appendingPathComponent("Source", isDirectory: true)
        let postersURL = sourceRoot.appendingPathComponent("Posters", isDirectory: true)
        let referencesURL = sourceRoot.appendingPathComponent("References", isDirectory: true)
        try FileManager.default.createDirectory(at: postersURL, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: referencesURL, withIntermediateDirectories: true)
        _ = try environment.writeOnePixelPNG(named: "cover.png", in: postersURL)
        _ = try environment.writeOnePixelJPEGWithExif(named: "nested.jpg", in: referencesURL)

        try await store.importItems(from: [sourceRoot])

        let postersFolderID = try XCTUnwrap(store.folders.first { $0.name == "Posters" }?.id)
        let referencesFolderID = try XCTUnwrap(store.folders.first { $0.name == "References" }?.id)
        XCTAssertNil(store.folders.first { $0.name == "Source" })
        XCTAssertEqual(store.assets.count, 2)
        XCTAssertTrue(try XCTUnwrap(store.assets.first { $0.displayName == "cover" }).folderIDs.contains(postersFolderID))
        XCTAssertTrue(try XCTUnwrap(store.assets.first { $0.displayName == "nested" }).folderIDs.contains(referencesFolderID))

        let storedAssetCount = try environment.storedAssetFiles().count
        let reopened = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try reopened.openLibrary(at: environment.packageURL)
        XCTAssertEqual(Set(reopened.folders.map(\.name)), ["Posters", "References"])
        XCTAssertTrue(try XCTUnwrap(reopened.assets.first { $0.displayName == "cover" }).folderIDs.contains(postersFolderID))
        XCTAssertTrue(try XCTUnwrap(reopened.assets.first { $0.displayName == "nested" }).folderIDs.contains(referencesFolderID))

        try await reopened.importItems(from: [sourceRoot])
        XCTAssertEqual(reopened.assets.count, 2)
        XCTAssertEqual(try environment.storedAssetFiles().count, storedAssetCount)
        XCTAssertTrue(try XCTUnwrap(reopened.assets.first { $0.displayName == "cover" }).folderIDs.contains(postersFolderID))

        let coverID = try XCTUnwrap(reopened.assets.first { $0.displayName == "cover" }?.id)
        try reopened.moveAssetToTrash(id: coverID)
        try await reopened.importItems(from: [sourceRoot])
        let restoredCover = try XCTUnwrap(reopened.assets.first { $0.id == coverID })
        XCTAssertFalse(restoredCover.isTrashed)
        XCTAssertTrue(restoredCover.folderIDs.contains(postersFolderID))
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

    private func folderNames(in folders: [AssetFolder], parentID: AssetFolder.ID?) -> [String] {
        folders
            .filter { $0.parentID == parentID }
            .sorted {
                if $0.sortIndex == $1.sortIndex {
                    return $0.createdAt < $1.createdAt
                }
                return $0.sortIndex < $1.sortIndex
            }
            .map(\.name)
    }

    private func toolbarFilterAsset(
        id: String,
        libraryID: String,
        displayName: String,
        fileExtension: String,
        byteSize: Int64,
        tag: TagItem,
        colorHex: String,
        importedAt: Date,
        paletteColors: [(hex: String, coverage: Double)]? = nil
    ) -> AssetItem {
        let colors = paletteColors ?? [(hex: colorHex, coverage: 0.5)]

        return AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: displayName,
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/Samples/\(displayName).\(fileExtension)"),
            kind: .image,
            fileExtension: fileExtension,
            byteSize: byteSize,
            contentHash: id,
            dimensions: nil,
            tags: [tag],
            paletteColors: colors.enumerated().map { index, color in
                AssetColor(
                    id: "\(id)-color-\(index)",
                    libraryID: libraryID,
                    assetID: id,
                    hex: color.hex,
                    coverage: color.coverage,
                    sortIndex: index
                )
            },
            isFavorite: false,
            importedAt: importedAt
        )
    }
}

private actor ImportProgressRecorder {
    private var values: [AssetImportProgress] = []

    func append(_ progress: AssetImportProgress) {
        values.append(progress)
    }

    func snapshot() -> [AssetImportProgress] {
        values
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

    func writeOnePixelPNG(named fileName: String, in directory: URL? = nil) throws -> URL {
        let data = Data(base64Encoded: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")!
        let directory = directory ?? inputURL
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        let url = directory.appendingPathComponent(fileName)
        try data.write(to: url)
        return url
    }

    func writeOnePixelJPEGWithExif(named fileName: String, in directory: URL? = nil) throws -> URL {
        let directory = directory ?? inputURL
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        let url = directory.appendingPathComponent(fileName)
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

    func writePreV3MetadataStore(
        for library: AssetLibrary,
        storage: LibraryStorage
    ) throws -> (assetID: String, folderID: String, importedAt: Date) {
        let model = try MomentoCoreDataStack.managedObjectModel(versionName: "MomentoModel v2")
        let coordinator = NSPersistentStoreCoordinator(managedObjectModel: model)
        try coordinator.addPersistentStore(
            ofType: NSSQLiteStoreType,
            configurationName: nil,
            at: storage.databaseURL(for: library),
            options: nil
        )

        let context = NSManagedObjectContext(concurrencyType: .privateQueueConcurrencyType)
        context.persistentStoreCoordinator = coordinator

        let assetID = "legacy-asset"
        let folderID = "legacy-folder"
        let importedAt = Date(timeIntervalSince1970: 1_700_000_000)

        try context.performAndWait {
            let asset = NSManagedObject(
                entity: NSEntityDescription.entity(forEntityName: "AssetRecord", in: context)!,
                insertInto: context
            )
            asset.setValue(assetID, forKey: "id")
            asset.setValue(library.id, forKey: "libraryID")
            asset.setValue("Legacy Asset", forKey: "displayName")
            asset.setValue("assets/legacy.png", forKey: "storageRelativePath")
            asset.setValue("image", forKey: "kindRaw")
            asset.setValue("png", forKey: "fileExtension")
            asset.setValue(Int64(128), forKey: "byteSize")
            asset.setValue("legacy-hash", forKey: "contentHash")
            asset.setValue(1, forKey: "pixelWidth")
            asset.setValue(1, forKey: "pixelHeight")
            asset.setValue(false, forKey: "isFavorite")
            asset.setValue(importedAt, forKey: "importedAt")
            asset.setValue(try JSONEncoder.momento.encode([TagItem(id: "legacy", name: "Legacy")]), forKey: "tagsData")

            let color = NSManagedObject(
                entity: NSEntityDescription.entity(forEntityName: "AssetColorRecord", in: context)!,
                insertInto: context
            )
            color.setValue("legacy-color", forKey: "id")
            color.setValue(library.id, forKey: "libraryID")
            color.setValue(assetID, forKey: "assetID")
            color.setValue("#123456", forKey: "hex")
            color.setValue(0.8, forKey: "coverage")
            color.setValue(0, forKey: "sortIndex")

            let folder = NSManagedObject(
                entity: NSEntityDescription.entity(forEntityName: "FolderRecord", in: context)!,
                insertInto: context
            )
            folder.setValue(folderID, forKey: "id")
            folder.setValue(library.id, forKey: "libraryID")
            folder.setValue("Legacy Folder", forKey: "name")
            folder.setValue(0, forKey: "sortIndex")
            folder.setValue(importedAt, forKey: "createdAt")
            folder.setValue(importedAt, forKey: "updatedAt")

            let membership = NSManagedObject(
                entity: NSEntityDescription.entity(forEntityName: "AssetFolderMembershipRecord", in: context)!,
                insertInto: context
            )
            membership.setValue("legacy-membership", forKey: "id")
            membership.setValue(library.id, forKey: "libraryID")
            membership.setValue(assetID, forKey: "assetID")
            membership.setValue(folderID, forKey: "folderID")
            membership.setValue(importedAt, forKey: "createdAt")

            try context.save()
        }

        return (assetID, folderID, importedAt)
    }

    func cleanup() {
        defaults.removePersistentDomain(forName: defaultsSuiteName)
        try? FileManager.default.removeItem(at: rootURL)
    }
}
