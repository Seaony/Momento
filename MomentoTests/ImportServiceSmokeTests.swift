import Foundation
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
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("thumbnails/small").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("thumbnails/medium").path))
        XCTAssertTrue(FileManager.default.fileExists(atPath: environment.packageURL.appendingPathComponent("thumbnails/large").path))
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
        XCTAssertEqual(try environment.storedAssetFiles().count, 1)
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
}

private struct TestEnvironment {
    let rootURL: URL
    let packageURL: URL
    let inputURL: URL
    let defaults: UserDefaults
    let defaultsSuiteName: String

    init() throws {
        rootURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        inputURL = rootURL.appendingPathComponent("input", isDirectory: true)
        packageURL = rootURL.appendingPathComponent("Test.momentolibrary", isDirectory: true)
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
