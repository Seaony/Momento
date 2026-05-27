import CloudKit
import Foundation

nonisolated protocol CloudLibraryRelationshipProviding: Sendable {
    func fetchFolderRecords(in libraryZoneName: String) async throws -> [CKRecord]
    func fetchTagRecords(in libraryZoneName: String) async throws -> [CKRecord]
    func fetchFolderMembershipRecords(in libraryZoneName: String) async throws -> [CKRecord]
    func fetchTagMembershipRecords(in libraryZoneName: String) async throws -> [CKRecord]
}

nonisolated protocol CloudAssetRelationshipWriting: Sendable {
    func saveTagMembership(
        assetID: AssetItem.ID,
        tag: TagItem,
        isMember: Bool,
        in library: AssetLibrary
    ) async throws
    func saveFolderMembership(
        assetID: AssetItem.ID,
        folder: AssetFolder,
        isMember: Bool,
        in library: AssetLibrary
    ) async throws
    func saveTag(
        _ tag: TagItem,
        in library: AssetLibrary,
        createdAt: Date?,
        updatedAt: Date,
        deletedAt: Date?
    ) async throws
    func saveFolder(_ folder: AssetFolder, in library: AssetLibrary, deletedAt: Date?) async throws
    func deleteTag(
        _ tag: TagItem,
        affectedAssetIDs: [AssetItem.ID],
        in library: AssetLibrary,
        deletedAt: Date
    ) async throws
    func deleteFolders(
        _ folders: [AssetFolder],
        affectedMemberships: [CloudFolderMembershipMutation],
        in library: AssetLibrary,
        deletedAt: Date
    ) async throws
}

nonisolated protocol CloudLibraryRelationshipSaveProviding: Sendable {
    func saveRecords(_ records: [CKRecord], in libraryZoneName: String) async throws -> [CKRecord]
}

struct CloudLibraryRelationships: Hashable, Sendable {
    var folders: [AssetFolder]
    var tags: [TagItem]
    var folderIDsByAssetID: [AssetItem.ID: [AssetFolder.ID]]
    var tagsByAssetID: [AssetItem.ID: [TagItem]]
}

struct CloudFolderMembershipMutation: Hashable, Sendable {
    var assetID: AssetItem.ID
    var folder: AssetFolder
    var isMember: Bool
}

actor CloudLibraryRelationshipService {
    private let provider: CloudLibraryRelationshipProviding

    init(provider: CloudLibraryRelationshipProviding = CloudKitLibraryRelationshipProvider()) {
        self.provider = provider
    }

    func fetchRelationships(in libraryZoneName: String) async throws -> CloudLibraryRelationships {
        async let folderRecords = provider.fetchFolderRecords(in: libraryZoneName)
        async let tagRecords = provider.fetchTagRecords(in: libraryZoneName)
        async let folderMembershipRecords = provider.fetchFolderMembershipRecords(in: libraryZoneName)
        async let tagMembershipRecords = provider.fetchTagMembershipRecords(in: libraryZoneName)

        let fetchedFolderRecords = try await folderRecords
        let fetchedTagRecords = try await tagRecords
        let fetchedFolderMembershipRecords = try await folderMembershipRecords
        let fetchedTagMembershipRecords = try await tagMembershipRecords

        var folders: [AssetFolder] = []
        for record in fetchedFolderRecords where Self.dateField(CloudFolderRecordField.deletedAt, in: record) == nil {
            folders.append(try Self.folder(from: record))
        }
        folders.sort { lhs, rhs in
            lhs.sortIndex == rhs.sortIndex ? lhs.name < rhs.name : lhs.sortIndex < rhs.sortIndex
        }

        var tags: [TagItem] = []
        for record in fetchedTagRecords where Self.dateField(CloudTagRecordField.deletedAt, in: record) == nil {
            tags.append(try Self.tag(from: record))
        }
        tags.sort { lhs, rhs in
            lhs.name.localizedCaseInsensitiveCompare(rhs.name) == .orderedAscending
        }

        var folderMemberships: [FolderMembership] = []
        for record in fetchedFolderMembershipRecords where Self.dateField(CloudFolderMembershipRecordField.deletedAt, in: record) == nil {
            folderMemberships.append(try Self.folderMembership(from: record))
        }

        var tagMemberships: [TagMembership] = []
        for record in fetchedTagMembershipRecords where Self.dateField(CloudTagMembershipRecordField.deletedAt, in: record) == nil {
            tagMemberships.append(try Self.tagMembership(from: record))
        }

        let tagsByID = Dictionary(uniqueKeysWithValues: tags.map { ($0.id, $0) })
        let folderIDsByAssetID = Dictionary(grouping: folderMemberships, by: \.assetID)
            .mapValues { memberships in
                memberships.map(\.folderID).sorted()
            }
        let tagsByAssetID = Dictionary(grouping: tagMemberships, by: \.assetID)
            .mapValues { memberships in
                memberships.compactMap { tagsByID[$0.tagID] }
            }

        return CloudLibraryRelationships(
            folders: folders,
            tags: tags,
            folderIDsByAssetID: folderIDsByAssetID,
            tagsByAssetID: tagsByAssetID
        )
    }

    private static func folder(from record: CKRecord) throws -> AssetFolder {
        AssetFolder(
            id: try requiredString(CloudFolderRecordField.id, in: record),
            libraryID: try requiredString(CloudFolderRecordField.libraryID, in: record),
            name: try requiredString(CloudFolderRecordField.name, in: record),
            parentID: stringField(CloudFolderRecordField.parentID, in: record),
            sortIndex: intField(CloudFolderRecordField.sortIndex, in: record) ?? 0,
            createdAt: try requiredDate(CloudFolderRecordField.createdAt, in: record),
            updatedAt: try requiredDate(CloudFolderRecordField.updatedAt, in: record)
        )
    }

    private static func tag(from record: CKRecord) throws -> TagItem {
        TagItem(
            id: try requiredString(CloudTagRecordField.id, in: record),
            name: try requiredString(CloudTagRecordField.name, in: record),
            colorHex: stringField(CloudTagRecordField.colorHex, in: record)
        )
    }

    private static func folderMembership(from record: CKRecord) throws -> FolderMembership {
        FolderMembership(
            assetID: try requiredString(CloudFolderMembershipRecordField.assetID, in: record),
            folderID: try requiredString(CloudFolderMembershipRecordField.folderID, in: record)
        )
    }

    private static func tagMembership(from record: CKRecord) throws -> TagMembership {
        TagMembership(
            assetID: try requiredString(CloudTagMembershipRecordField.assetID, in: record),
            tagID: try requiredString(CloudTagMembershipRecordField.tagID, in: record)
        )
    }

    private static func requiredString(_ key: String, in record: CKRecord) throws -> String {
        guard let value = stringField(key, in: record), !value.isEmpty else {
            throw CloudLibraryRelationshipError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func requiredDate(_ key: String, in record: CKRecord) throws -> Date {
        guard let value = dateField(key, in: record) else {
            throw CloudLibraryRelationshipError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func stringField(_ key: String, in record: CKRecord) -> String? {
        record[key] as? String
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

    private struct FolderMembership {
        var assetID: AssetItem.ID
        var folderID: AssetFolder.ID
    }

    private struct TagMembership {
        var assetID: AssetItem.ID
        var tagID: TagItem.ID
    }
}

actor CloudLibraryRelationshipWriteService: CloudAssetRelationshipWriting {
    private let provider: CloudLibraryRelationshipSaveProviding

    init(provider: CloudLibraryRelationshipSaveProviding = CloudKitLibraryRelationshipSaveProvider()) {
        self.provider = provider
    }

    func saveTagMembership(
        assetID: AssetItem.ID,
        tag: TagItem,
        isMember: Bool,
        in library: AssetLibrary
    ) async throws {
        _ = try await provider.saveRecords(
            [tagMembershipRecord(assetID: assetID, tag: tag, isMember: isMember, in: library, updatedAt: Date())],
            in: library.libraryZoneName
        )
    }

    func saveFolderMembership(
        assetID: AssetItem.ID,
        folder: AssetFolder,
        isMember: Bool,
        in library: AssetLibrary
    ) async throws {
        _ = try await provider.saveRecords(
            [folderMembershipRecord(assetID: assetID, folder: folder, isMember: isMember, in: library, updatedAt: Date())],
            in: library.libraryZoneName
        )
    }

    func saveTag(
        _ tag: TagItem,
        in library: AssetLibrary,
        createdAt: Date?,
        updatedAt: Date,
        deletedAt: Date?
    ) async throws {
        _ = try await provider.saveRecords(
            [tagRecord(tag, in: library, createdAt: createdAt, updatedAt: updatedAt, deletedAt: deletedAt)],
            in: library.libraryZoneName
        )
    }

    func saveFolder(_ folder: AssetFolder, in library: AssetLibrary, deletedAt: Date?) async throws {
        _ = try await provider.saveRecords(
            [folderRecord(folder, in: library, deletedAt: deletedAt)],
            in: library.libraryZoneName
        )
    }

    func deleteTag(
        _ tag: TagItem,
        affectedAssetIDs: [AssetItem.ID],
        in library: AssetLibrary,
        deletedAt: Date
    ) async throws {
        let tagRecord = tagRecord(tag, in: library, createdAt: nil, updatedAt: deletedAt, deletedAt: deletedAt)
        let membershipRecords = affectedAssetIDs.map { assetID in
            tagMembershipRecord(assetID: assetID, tag: tag, isMember: false, in: library, updatedAt: deletedAt)
        }
        _ = try await provider.saveRecords([tagRecord] + membershipRecords, in: library.libraryZoneName)
    }

    func deleteFolders(
        _ folders: [AssetFolder],
        affectedMemberships: [CloudFolderMembershipMutation],
        in library: AssetLibrary,
        deletedAt: Date
    ) async throws {
        let folderRecords = folders.map { folder in
            var deletedFolder = folder
            deletedFolder.updatedAt = deletedAt
            return folderRecord(deletedFolder, in: library, deletedAt: deletedAt)
        }
        let membershipRecords = affectedMemberships.map { membership in
            folderMembershipRecord(
                assetID: membership.assetID,
                folder: membership.folder,
                isMember: membership.isMember,
                in: library,
                updatedAt: deletedAt
            )
        }
        _ = try await provider.saveRecords(folderRecords + membershipRecords, in: library.libraryZoneName)
    }

    private func tagRecord(
        _ tag: TagItem,
        in library: AssetLibrary,
        createdAt: Date?,
        updatedAt: Date,
        deletedAt: Date?
    ) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.tag.rawValue,
            recordID: recordID(name: CloudRecordNaming.tagRecordName(tagID: tag.id), in: library)
        )
        record[CloudTagRecordField.id] = tag.id
        record[CloudTagRecordField.libraryID] = library.id
        record[CloudTagRecordField.name] = tag.name
        record[CloudTagRecordField.normalizedName] = tag.name.lowercased()
        record[CloudTagRecordField.colorHex] = tag.colorHex
        if let createdAt {
            record[CloudTagRecordField.createdAt] = createdAt
        }
        record[CloudTagRecordField.updatedAt] = updatedAt
        record[CloudTagRecordField.deletedAt] = deletedAt
        return record
    }

    private func folderRecord(
        _ folder: AssetFolder,
        in library: AssetLibrary,
        deletedAt: Date?
    ) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.folder.rawValue,
            recordID: recordID(name: CloudRecordNaming.folderRecordName(folderID: folder.id), in: library)
        )
        record[CloudFolderRecordField.id] = folder.id
        record[CloudFolderRecordField.libraryID] = folder.libraryID
        record[CloudFolderRecordField.name] = folder.name
        record[CloudFolderRecordField.parentID] = folder.parentID
        record[CloudFolderRecordField.sortIndex] = folder.sortIndex
        record[CloudFolderRecordField.createdAt] = folder.createdAt
        record[CloudFolderRecordField.updatedAt] = folder.updatedAt
        record[CloudFolderRecordField.deletedAt] = deletedAt
        return record
    }

    private func tagMembershipRecord(
        assetID: AssetItem.ID,
        tag: TagItem,
        isMember: Bool,
        in library: AssetLibrary,
        updatedAt: Date
    ) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.tagMembership.rawValue,
            recordID: recordID(
                name: CloudRecordNaming.tagMembershipRecordName(assetID: assetID, tagID: tag.id),
                in: library
            )
        )
        record[CloudTagMembershipRecordField.id] = "\(assetID)-\(tag.id)"
        record[CloudTagMembershipRecordField.libraryID] = library.id
        record[CloudTagMembershipRecordField.assetID] = assetID
        record[CloudTagMembershipRecordField.tagID] = tag.id
        record[CloudTagMembershipRecordField.createdAt] = updatedAt
        record[CloudTagMembershipRecordField.deletedAt] = isMember ? nil : updatedAt
        return record
    }

    private func folderMembershipRecord(
        assetID: AssetItem.ID,
        folder: AssetFolder,
        isMember: Bool,
        in library: AssetLibrary,
        updatedAt: Date
    ) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.folderMembership.rawValue,
            recordID: recordID(
                name: CloudRecordNaming.folderMembershipRecordName(assetID: assetID, folderID: folder.id),
                in: library
            )
        )
        record[CloudFolderMembershipRecordField.id] = "\(assetID)-\(folder.id)"
        record[CloudFolderMembershipRecordField.libraryID] = library.id
        record[CloudFolderMembershipRecordField.assetID] = assetID
        record[CloudFolderMembershipRecordField.folderID] = folder.id
        record[CloudFolderMembershipRecordField.createdAt] = updatedAt
        record[CloudFolderMembershipRecordField.deletedAt] = isMember ? nil : updatedAt
        return record
    }

    private func recordID(name: String, in library: AssetLibrary) -> CKRecord.ID {
        CKRecord.ID(
            recordName: name,
            zoneID: CKRecordZone.ID(zoneName: library.libraryZoneName, ownerName: CKCurrentUserDefaultName)
        )
    }
}

nonisolated struct CloudKitLibraryRelationshipProvider: CloudLibraryRelationshipProviding, @unchecked Sendable {
    private let containerIdentifier: String

    init(containerIdentifier: String = CloudKitConfiguration.containerIdentifier) {
        self.containerIdentifier = containerIdentifier
    }

    func fetchFolderRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        try await fetchRecords(type: .folder, in: libraryZoneName, desiredKeys: CloudFolderRecordField.metadataKeys)
    }

    func fetchTagRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        try await fetchRecords(type: .tag, in: libraryZoneName, desiredKeys: CloudTagRecordField.metadataKeys)
    }

    func fetchFolderMembershipRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        try await fetchRecords(
            type: .folderMembership,
            in: libraryZoneName,
            desiredKeys: CloudFolderMembershipRecordField.metadataKeys
        )
    }

    func fetchTagMembershipRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        try await fetchRecords(
            type: .tagMembership,
            in: libraryZoneName,
            desiredKeys: CloudTagMembershipRecordField.metadataKeys
        )
    }

    private func fetchRecords(
        type: CloudRecordType,
        in libraryZoneName: String,
        desiredKeys: [String]
    ) async throws -> [CKRecord] {
        let zoneID = CKRecordZone.ID(zoneName: libraryZoneName, ownerName: CKCurrentUserDefaultName)
        let query = CKQuery(recordType: type.rawValue, predicate: NSPredicate(value: true))
        var records: [CKRecord] = []
        var response = try await database.records(
            matching: query,
            inZoneWith: zoneID,
            desiredKeys: desiredKeys,
            resultsLimit: CKQueryOperation.maximumResults
        )
        try appendSuccessfulRecords(from: response.matchResults, to: &records)

        while let cursor = response.queryCursor {
            response = try await database.records(
                continuingMatchFrom: cursor,
                desiredKeys: desiredKeys,
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

nonisolated struct CloudKitLibraryRelationshipSaveProvider: CloudLibraryRelationshipSaveProviding, @unchecked Sendable {
    private let containerIdentifier: String

    init(containerIdentifier: String = CloudKitConfiguration.containerIdentifier) {
        self.containerIdentifier = containerIdentifier
    }

    func saveRecords(_ records: [CKRecord], in libraryZoneName: String) async throws -> [CKRecord] {
        let response = try await database.modifyRecords(
            saving: records,
            deleting: [],
            savePolicy: .changedKeys,
            atomically: true
        )
        return try response.saveResults.map { try $0.1.get() }
    }

    private var database: CKDatabase {
        CKContainer(identifier: containerIdentifier).privateCloudDatabase
    }
}

enum CloudLibraryRelationshipError: LocalizedError, Equatable {
    case missingRequiredField(recordName: String, fieldName: String)

    var errorDescription: String? {
        switch self {
        case .missingRequiredField(let recordName, let fieldName):
            "Cloud relationship record \(recordName) is missing required field \(fieldName)."
        }
    }
}
