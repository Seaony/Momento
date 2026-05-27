import CloudKit
import Foundation
import XCTest
@testable import Momento

final class CloudLibraryManagementServiceTests: XCTestCase {
    func testCreateLibraryCreatesCatalogAndLibraryZonesThenSavesCatalogRecord() async throws {
        let provider = FakeCloudLibraryManagementProvider()
        let service = CloudLibraryManagementService(provider: provider)

        let descriptor = try await service.createLibrary(named: " Design Archive ")
        let savedZoneNames = await provider.savedZoneNames()
        let savedCatalogRecordSnapshots = await provider.savedCatalogRecordSnapshots()

        XCTAssertEqual(
            savedZoneNames,
            [
                CloudRecordNaming.catalogZoneName,
                descriptor.libraryZoneName
            ]
        )
        XCTAssertEqual(descriptor.displayName, "Design Archive")
        XCTAssertEqual(descriptor.libraryZoneName, CloudRecordNaming.libraryZoneName(libraryID: descriptor.id))
        XCTAssertEqual(descriptor.schemaVersion, CloudLibraryCatalogService.clientMaxSupportedSchemaVersion)
        XCTAssertEqual(descriptor.syncState, .synced)
        XCTAssertNil(descriptor.deletedAt)

        let record = try XCTUnwrap(savedCatalogRecordSnapshots.first)
        XCTAssertEqual(record.zoneName, CloudRecordNaming.catalogZoneName)
        XCTAssertEqual(record.recordName, CloudRecordNaming.libraryRecordName(libraryID: descriptor.id))
        XCTAssertEqual(record.id, descriptor.id)
        XCTAssertEqual(record.displayName, "Design Archive")
        XCTAssertEqual(record.libraryZoneName, descriptor.libraryZoneName)
        XCTAssertNil(record.deletedAt)
    }

    func testRenameLibraryUpdatesExistingCatalogRecordWithoutCreatingZones() async throws {
        let provider = FakeCloudLibraryManagementProvider()
        let service = CloudLibraryManagementService(provider: provider)
        let library = makeAssetLibrary()

        let descriptor = try await service.renameLibrary(library, to: " Brand Vault ")
        let savedZoneNames = await provider.savedZoneNames()
        let savedCatalogRecordSnapshots = await provider.savedCatalogRecordSnapshots()

        XCTAssertTrue(savedZoneNames.isEmpty)
        XCTAssertEqual(descriptor.id, "library-a")
        XCTAssertEqual(descriptor.displayName, "Brand Vault")
        XCTAssertEqual(descriptor.libraryZoneName, "MomentoLibrary-library-a")
        XCTAssertEqual(descriptor.createdAt, library.createdAt)
        XCTAssertNil(descriptor.deletedAt)

        let record = try XCTUnwrap(savedCatalogRecordSnapshots.first)
        XCTAssertEqual(record.displayName, "Brand Vault")
        XCTAssertNil(record.deletedAt)
    }

    func testDeleteLibrarySoftDeletesCatalogRecord() async throws {
        let provider = FakeCloudLibraryManagementProvider()
        let service = CloudLibraryManagementService(provider: provider)
        let library = makeAssetLibrary()

        let descriptor = try await service.deleteLibrary(library)
        let savedZoneNames = await provider.savedZoneNames()
        let savedCatalogRecordSnapshots = await provider.savedCatalogRecordSnapshots()

        XCTAssertTrue(savedZoneNames.isEmpty)
        XCTAssertEqual(descriptor.id, "library-a")
        XCTAssertNotNil(descriptor.deletedAt)

        let record = try XCTUnwrap(savedCatalogRecordSnapshots.first)
        XCTAssertEqual(record.id, "library-a")
        XCTAssertEqual(record.deletedAt, descriptor.deletedAt)
    }

}

final class CloudAssetServiceTests: XCTestCase {
    func testAssetCatalogMarksFetchedAssetsAsRemoteOnlyAndSynced() async throws {
        let descriptor = makeDescriptor()
        let library = makeAssetLibrary(from: descriptor)
        let provider = FakeCloudAssetCatalogProvider(
            assetRecords: [Self.assetRecord(library: descriptor)],
            colorRecords: []
        )
        let service = CloudAssetCatalogService(
            cachePaths: CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot()),
            provider: provider
        )

        let assets = try await service.fetchAssets(in: library)
        let asset = try XCTUnwrap(assets.first)

        XCTAssertEqual(asset.availability.original, .remoteOnly)
        XCTAssertEqual(asset.availability.thumbnail, .remoteOnly)
        XCTAssertNil(asset.availability.lastError)
        XCTAssertEqual(asset.syncState, .synced)
    }

    func testAssetUploadMarksSavedAssetsAsLocalAndSynced() async throws {
        let library = makeAssetLibrary()
        let provider = FakeCloudAssetUploadProvider()
        let service = CloudAssetUploadService(provider: provider)
        let localFileURL = temporaryRoot().appendingPathComponent("asset.png")
        try Data("asset".utf8).write(to: localFileURL)

        let asset = Self.assetItem(
            libraryID: library.id,
            storageURL: localFileURL,
            availability: AssetFileAvailability(original: .uploadPending, thumbnail: .local, lastError: "pending"),
            syncState: .syncing
        )

        let uploadedAssets = try await service.uploadAssets([asset], to: library)
        let uploadedAsset = try XCTUnwrap(uploadedAssets.first)

        XCTAssertEqual(uploadedAsset.availability.original, .local)
        XCTAssertEqual(uploadedAsset.availability.thumbnail, .local)
        XCTAssertNil(uploadedAsset.availability.lastError)
        XCTAssertEqual(uploadedAsset.syncState, .synced)
    }

    func testAssetMetadataSaveClearsNilOptionalCloudFields() async throws {
        let library = makeAssetLibrary()
        let provider = FakeCloudAssetUploadProvider()
        let service = CloudAssetUploadService(provider: provider)
        let localFileURL = temporaryRoot().appendingPathComponent("asset.png")
        let asset = Self.assetItem(
            libraryID: library.id,
            storageURL: localFileURL,
            availability: AssetFileAvailability(original: .local, thumbnail: .local, lastError: nil),
            syncState: .synced
        )

        _ = try await service.saveAssetMetadata([asset], to: library)
        let savedRecords = await provider.savedAssetRecords()
        let metadataRecord = try XCTUnwrap(savedRecords.first { record in
            record.recordType == CloudRecordType.asset.rawValue
        })

        XCTAssertTrue(metadataRecord.changedKeys().contains(CloudAssetRecordField.sourcePageURL))
        XCTAssertTrue(metadataRecord.changedKeys().contains(CloudAssetRecordField.note))
        XCTAssertTrue(metadataRecord.changedKeys().contains(CloudAssetRecordField.trashedAt))
        XCTAssertNil(metadataRecord[CloudAssetRecordField.sourcePageURL])
        XCTAssertNil(metadataRecord[CloudAssetRecordField.note])
        XCTAssertNil(metadataRecord[CloudAssetRecordField.trashedAt])
    }

    private static func assetRecord(library: CloudLibraryDescriptor) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.asset.rawValue,
            recordID: CKRecord.ID(
                recordName: CloudRecordNaming.assetRecordName(contentHash: "hash-a"),
                zoneID: CloudRecordIDBuilder.libraryZoneID(libraryID: library.id)
            )
        )
        record[CloudAssetRecordField.id] = "asset-a"
        record[CloudAssetRecordField.libraryID] = library.id
        record[CloudAssetRecordField.contentHash] = "hash-a"
        record[CloudAssetRecordField.displayName] = "Asset A"
        record[CloudAssetRecordField.originalFileName] = "asset-a.png"
        record[CloudAssetRecordField.fileExtension] = "png"
        record[CloudAssetRecordField.utiIdentifier] = "public.png"
        record[CloudAssetRecordField.kind] = AssetKind.image.rawValue
        record[CloudAssetRecordField.byteSize] = NSNumber(value: 5)
        record[CloudAssetRecordField.importedAt] = Date(timeIntervalSince1970: 1_000)
        record[CloudAssetRecordField.updatedAt] = Date(timeIntervalSince1970: 2_000)
        return record
    }

    private static func assetItem(
        libraryID: String,
        storageURL: URL,
        availability: AssetFileAvailability,
        syncState: CloudLibrarySyncState
    ) -> AssetItem {
        AssetItem(
            id: "asset-a",
            libraryID: libraryID,
            displayName: "Asset A",
            originalFileName: "asset-a.png",
            originalURL: storageURL,
            storageURL: storageURL,
            kind: .image,
            fileExtension: "png",
            utiIdentifier: "public.png",
            byteSize: 5,
            contentHash: "hash-a",
            dimensions: nil,
            tags: [],
            paletteColors: [],
            thumbnailURL: storageURL,
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 1_000),
            updatedAt: Date(timeIntervalSince1970: 2_000),
            availability: availability,
            syncState: syncState
        )
    }
}

private struct CatalogRecordSnapshot: Sendable {
    var zoneName: String
    var recordName: String
    var id: String?
    var displayName: String?
    var libraryZoneName: String?
    var deletedAt: Date?
}

private func makeDescriptor() -> CloudLibraryDescriptor {
    CloudLibraryDescriptor(
        id: "library-a",
        displayName: "Design Archive",
        libraryZoneName: "MomentoLibrary-library-a",
        createdAt: Date(timeIntervalSince1970: 1_000),
        updatedAt: Date(timeIntervalSince1970: 2_000),
        deletedAt: nil,
        schemaVersion: CloudLibraryCatalogService.clientMaxSupportedSchemaVersion,
        syncState: .synced,
        lastError: nil
    )
}

private func makeAssetLibrary(
    from descriptor: CloudLibraryDescriptor = makeDescriptor(),
    cloudAccountID: String = "account"
) -> AssetLibrary {
    AssetLibrary(
        id: descriptor.id,
        name: descriptor.displayName,
        createdAt: descriptor.createdAt,
        packageURL: nil,
        storageMode: .cloud,
        libraryZoneName: descriptor.libraryZoneName,
        cloudAccountID: cloudAccountID,
        syncState: descriptor.syncState
    )
}

private actor FakeCloudLibraryManagementProvider: CloudLibraryManagementProviding {
    private var savedZones: [CKRecordZone] = []
    private var savedCatalogRecords: [CKRecord] = []

    func saveZones(_ zones: [CKRecordZone]) async throws -> [CKRecordZone] {
        savedZones.append(contentsOf: zones)
        return zones
    }

    func saveCatalogRecord(_ record: CKRecord) async throws -> CKRecord {
        savedCatalogRecords.append(record)
        return record
    }

    func savedZoneNames() -> [String] {
        savedZones.map { $0.zoneID.zoneName }
    }

    func savedCatalogRecordSnapshots() -> [CatalogRecordSnapshot] {
        savedCatalogRecords.map { record in
            CatalogRecordSnapshot(
                zoneName: record.recordID.zoneID.zoneName,
                recordName: record.recordID.recordName,
                id: record[CloudLibraryRecordField.id] as? String,
                displayName: record[CloudLibraryRecordField.displayName] as? String,
                libraryZoneName: record[CloudLibraryRecordField.libraryZoneName] as? String,
                deletedAt: record[CloudLibraryRecordField.deletedAt] as? Date
            )
        }
    }
}

private struct FakeCloudAssetCatalogProvider: CloudAssetCatalogProviding {
    var assetRecords: [CKRecord]
    var colorRecords: [CKRecord]

    func fetchAssetRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        assetRecords
    }

    func fetchAssetColorRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        colorRecords
    }
}

private actor FakeCloudAssetUploadProvider: CloudAssetUploadProviding {
    private var savedRecords: [CKRecord] = []

    func saveRecords(_ records: [CKRecord], in libraryZoneName: String) async throws -> [CKRecord] {
        savedRecords.append(contentsOf: records)
        return records
    }

    func savedAssetRecords() -> [CKRecord] {
        savedRecords
    }
}

private func temporaryRoot() -> URL {
    let root = FileManager.default.temporaryDirectory
        .appendingPathComponent("MomentoTests", isDirectory: true)
        .appendingPathComponent(UUID().uuidString, isDirectory: true)
    try? FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
    return root
}
