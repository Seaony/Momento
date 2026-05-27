import CloudKit
import Foundation

nonisolated protocol CloudLibraryManaging: Sendable {
    func createLibrary(named name: String) async throws -> CloudLibraryDescriptor
    func renameLibrary(_ library: AssetLibrary, to name: String) async throws -> CloudLibraryDescriptor
    func deleteLibrary(_ library: AssetLibrary) async throws -> CloudLibraryDescriptor
}

nonisolated protocol CloudLibraryManagementProviding: Sendable {
    func saveZones(_ zones: [CKRecordZone]) async throws -> [CKRecordZone]
    func saveCatalogRecord(_ record: CKRecord) async throws -> CKRecord
}

actor CloudLibraryManagementService: CloudLibraryManaging {
    private let provider: CloudLibraryManagementProviding

    init(provider: CloudLibraryManagementProviding = CloudKitLibraryManagementProvider()) {
        self.provider = provider
    }

    func createLibrary(named name: String) async throws -> CloudLibraryDescriptor {
        let displayName = try Self.validDisplayName(name)
        let libraryID = UUID().uuidString
        let now = Date()
        let descriptor = CloudLibraryDescriptor(
            id: libraryID,
            displayName: displayName,
            libraryZoneName: CloudRecordNaming.libraryZoneName(libraryID: libraryID),
            createdAt: now,
            updatedAt: now,
            deletedAt: nil,
            schemaVersion: CloudLibraryCatalogService.clientMaxSupportedSchemaVersion,
            syncState: .synced,
            lastError: nil
        )

        _ = try await provider.saveZones([
            CKRecordZone(zoneID: CloudRecordIDBuilder.catalogZoneID()),
            CKRecordZone(zoneID: CloudRecordIDBuilder.libraryZoneID(libraryID: descriptor.id))
        ])
        _ = try await provider.saveCatalogRecord(Self.catalogRecord(from: descriptor))
        return descriptor
    }

    func renameLibrary(_ library: AssetLibrary, to name: String) async throws -> CloudLibraryDescriptor {
        try Self.requireSupportedLibrary(library)
        let descriptor = CloudLibraryDescriptor(
            id: library.id,
            displayName: try Self.validDisplayName(name),
            libraryZoneName: library.libraryZoneName,
            createdAt: library.createdAt,
            updatedAt: Date(),
            deletedAt: nil,
            schemaVersion: CloudLibraryCatalogService.clientMaxSupportedSchemaVersion,
            syncState: .synced,
            lastError: nil
        )

        _ = try await provider.saveCatalogRecord(Self.catalogRecord(from: descriptor))
        return descriptor
    }

    func deleteLibrary(_ library: AssetLibrary) async throws -> CloudLibraryDescriptor {
        try Self.requireSupportedLibrary(library)
        let deletedAt = Date()
        let descriptor = CloudLibraryDescriptor(
            id: library.id,
            displayName: library.name,
            libraryZoneName: library.libraryZoneName,
            createdAt: library.createdAt,
            updatedAt: deletedAt,
            deletedAt: deletedAt,
            schemaVersion: CloudLibraryCatalogService.clientMaxSupportedSchemaVersion,
            syncState: .synced,
            lastError: nil
        )

        _ = try await provider.saveCatalogRecord(Self.catalogRecord(from: descriptor))
        return descriptor
    }

    private static func catalogRecord(from descriptor: CloudLibraryDescriptor) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.library.rawValue,
            recordID: CloudRecordIDBuilder.cloudLibraryRecordID(libraryID: descriptor.id)
        )
        record[CloudLibraryRecordField.id] = descriptor.id
        record[CloudLibraryRecordField.displayName] = descriptor.displayName
        record[CloudLibraryRecordField.libraryZoneName] = descriptor.libraryZoneName
        record[CloudLibraryRecordField.createdAt] = descriptor.createdAt
        record[CloudLibraryRecordField.updatedAt] = descriptor.updatedAt
        record[CloudLibraryRecordField.deletedAt] = descriptor.deletedAt
        record[CloudLibraryRecordField.schemaVersion] = descriptor.schemaVersion
        return record
    }

    private static func validDisplayName(_ name: String) throws -> String {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty else {
            throw CloudLibraryManagementError.invalidLibraryName
        }
        return trimmedName
    }

    private static func requireSupportedLibrary(_ library: AssetLibrary) throws {
        guard library.storageMode == .cloud,
              library.syncState != .unsupportedSchema else {
            throw CloudLibraryManagementError.unsupportedSchemaVersion
        }
        guard !library.id.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty,
              !library.libraryZoneName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            throw CloudLibraryManagementError.activeCloudLibraryRequired
        }
    }
}

nonisolated struct CloudKitLibraryManagementProvider: CloudLibraryManagementProviding, @unchecked Sendable {
    private let containerIdentifier: String

    init(containerIdentifier: String = CloudKitConfiguration.containerIdentifier) {
        self.containerIdentifier = containerIdentifier
    }

    func saveZones(_ zones: [CKRecordZone]) async throws -> [CKRecordZone] {
        let response = try await database.modifyRecordZones(saving: zones, deleting: [])
        return try response.saveResults.map { try $0.1.get() }
    }

    func saveCatalogRecord(_ record: CKRecord) async throws -> CKRecord {
        let response = try await database.modifyRecords(
            saving: [record],
            deleting: [],
            savePolicy: .changedKeys,
            atomically: true
        )
        guard let savedRecord = try response.saveResults.first?.value.get() else {
            throw CloudLibraryManagementError.missingSavedCatalogRecord
        }
        return savedRecord
    }

    private var database: CKDatabase {
        CKContainer(identifier: containerIdentifier).privateCloudDatabase
    }
}

enum CloudLibraryManagementError: LocalizedError, Equatable {
    case invalidLibraryName
    case activeCloudLibraryRequired
    case unsupportedSchemaVersion
    case missingSavedCatalogRecord

    var errorDescription: String? {
        switch self {
        case .invalidLibraryName:
            "Library name cannot be empty."
        case .activeCloudLibraryRequired:
            "This action requires an active iCloud library."
        case .unsupportedSchemaVersion:
            "Open this library on a newer Momento to edit."
        case .missingSavedCatalogRecord:
            "CloudKit did not return the saved library catalog record."
        }
    }
}
