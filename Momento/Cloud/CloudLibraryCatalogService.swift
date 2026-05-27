import CloudKit
import Foundation

nonisolated protocol CloudLibraryCatalogProviding: Sendable {
    func fetchCatalogLibraryRecords() async throws -> [CKRecord]
}

actor CloudLibraryCatalogService {
    static let clientMaxSupportedSchemaVersion = 1

    private let provider: CloudLibraryCatalogProviding

    init(provider: CloudLibraryCatalogProviding = CloudKitLibraryCatalogProvider()) {
        self.provider = provider
    }

    func fetchLibraries() async throws -> [CloudLibraryDescriptor] {
        let records: [CKRecord]
        do {
            records = try await provider.fetchCatalogLibraryRecords()
        } catch {
            if Self.isMissingCatalogZone(error) {
                return []
            }
            throw error
        }

        return try records
            .map(Self.descriptor(from:))
            .filter { $0.deletedAt == nil }
            .sorted { $0.updatedAt > $1.updatedAt }
    }

    private static func descriptor(from record: CKRecord) throws -> CloudLibraryDescriptor {
        let schemaVersion = try requiredInt(CloudLibraryRecordField.schemaVersion, in: record)
        let syncState: CloudLibrarySyncState = schemaVersion > clientMaxSupportedSchemaVersion
            ? .unsupportedSchema
            : .synced
        let lastError = schemaVersion > clientMaxSupportedSchemaVersion
            ? "Open this library on a newer Momento to edit."
            : nil

        return CloudLibraryDescriptor(
            id: try requiredString(CloudLibraryRecordField.id, in: record),
            displayName: try requiredString(CloudLibraryRecordField.displayName, in: record),
            libraryZoneName: try requiredString(CloudLibraryRecordField.libraryZoneName, in: record),
            createdAt: try requiredDate(CloudLibraryRecordField.createdAt, in: record),
            updatedAt: try requiredDate(CloudLibraryRecordField.updatedAt, in: record),
            deletedAt: dateField(CloudLibraryRecordField.deletedAt, in: record),
            schemaVersion: schemaVersion,
            syncState: syncState,
            lastError: lastError
        )
    }

    private static func requiredString(_ key: String, in record: CKRecord) throws -> String {
        guard let value = record[key] as? String, !value.isEmpty else {
            throw CloudLibraryCatalogError.missingRequiredField(
                recordName: record.recordID.recordName,
                fieldName: key
            )
        }
        return value
    }

    private static func requiredDate(_ key: String, in record: CKRecord) throws -> Date {
        guard let value = dateField(key, in: record) else {
            throw CloudLibraryCatalogError.missingRequiredField(
                recordName: record.recordID.recordName,
                fieldName: key
            )
        }
        return value
    }

    private static func requiredInt(_ key: String, in record: CKRecord) throws -> Int {
        guard let value = intField(key, in: record) else {
            throw CloudLibraryCatalogError.missingRequiredField(
                recordName: record.recordID.recordName,
                fieldName: key
            )
        }
        return value
    }

    private static func dateField(_ key: String, in record: CKRecord) -> Date? {
        record[key] as? Date
    }

    private static func intField(_ key: String, in record: CKRecord) -> Int? {
        if let value = record[key] as? Int {
            return value
        }
        if let value = record[key] as? NSNumber {
            return value.intValue
        }
        return nil
    }

    private static func isMissingCatalogZone(_ error: any Error) -> Bool {
        guard let cloudKitError = error as? CKError else {
            return false
        }
        return cloudKitError.code == .zoneNotFound
    }
}

nonisolated struct CloudKitLibraryCatalogProvider: CloudLibraryCatalogProviding, @unchecked Sendable {
    private let containerIdentifier: String

    init(containerIdentifier: String = CloudKitConfiguration.containerIdentifier) {
        self.containerIdentifier = containerIdentifier
    }

    func fetchCatalogLibraryRecords() async throws -> [CKRecord] {
        let query = CKQuery(
            recordType: CloudRecordType.library.rawValue,
            predicate: NSPredicate(value: true)
        )
        var records: [CKRecord] = []
        var response = try await database.records(
            matching: query,
            inZoneWith: CloudRecordIDBuilder.catalogZoneID(),
            desiredKeys: CloudLibraryRecordField.metadataKeys,
            resultsLimit: CKQueryOperation.maximumResults
        )
        try appendSuccessfulRecords(from: response.matchResults, to: &records)

        while let cursor = response.queryCursor {
            response = try await database.records(
                continuingMatchFrom: cursor,
                desiredKeys: CloudLibraryRecordField.metadataKeys,
                resultsLimit: CKQueryOperation.maximumResults
            )
            try appendSuccessfulRecords(from: response.matchResults, to: &records)
        }

        return records
    }

    private func appendSuccessfulRecords(
        from matchResults: [(CKRecord.ID, Result<CKRecord, any Error>)],
        to records: inout [CKRecord]
    ) throws {
        for (_, result) in matchResults {
            records.append(try result.get())
        }
    }

    private var database: CKDatabase {
        CKContainer(identifier: containerIdentifier).privateCloudDatabase
    }
}

enum CloudLibraryCatalogError: LocalizedError, Equatable {
    case missingRequiredField(recordName: String, fieldName: String)

    var errorDescription: String? {
        switch self {
        case .missingRequiredField(let recordName, let fieldName):
            "Cloud library record \(recordName) is missing required field \(fieldName)."
        }
    }
}
