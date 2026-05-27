import CloudKit
import Foundation

nonisolated protocol CloudAssetCatalogProviding: Sendable {
    func fetchAssetRecords(in libraryZoneName: String) async throws -> [CKRecord]
    func fetchAssetColorRecords(in libraryZoneName: String) async throws -> [CKRecord]
}

actor CloudAssetCatalogService {
    private let cachePaths: CloudLibraryCachePaths
    private let provider: CloudAssetCatalogProviding

    init(
        cachePaths: CloudLibraryCachePaths = CloudLibraryCachePaths(),
        provider: CloudAssetCatalogProviding = CloudKitAssetCatalogProvider()
    ) {
        self.cachePaths = cachePaths
        self.provider = provider
    }

    func fetchAssets(in library: AssetLibrary) async throws -> [AssetItem] {
        let cloudAccountID = try Self.requiredCloudAccountID(in: library)
        async let fetchedAssetRecords = provider.fetchAssetRecords(in: library.libraryZoneName)
        async let fetchedColorRecords = provider.fetchAssetColorRecords(in: library.libraryZoneName)
        let (records, colorRecords) = try await (fetchedAssetRecords, fetchedColorRecords)
        let colorsByAssetID = try Self.colorsByAssetID(from: colorRecords)
        let assets = try records
            .filter { Self.dateField(CloudAssetRecordField.deletedAt, in: $0) == nil }
            .map { record in
                try Self.asset(
                    from: record,
                    colorsByAssetID: colorsByAssetID,
                    cloudAccountID: cloudAccountID,
                    cachePaths: cachePaths
                )
        }
        return assets.sorted { $0.importedAt > $1.importedAt }
    }

    private static func requiredCloudAccountID(in library: AssetLibrary) throws -> String {
        guard let cloudAccountID = library.cloudAccountID?.trimmingCharacters(in: .whitespacesAndNewlines),
              !cloudAccountID.isEmpty else {
            throw CloudAssetCatalogError.missingCloudAccountID(libraryID: library.id)
        }
        return cloudAccountID
    }

    private static func asset(
        from record: CKRecord,
        colorsByAssetID: [String: [AssetColor]],
        cloudAccountID: String,
        cachePaths: CloudLibraryCachePaths
    ) throws -> AssetItem {
        let id = try requiredString(CloudAssetRecordField.id, in: record)
        let libraryID = try requiredString(CloudAssetRecordField.libraryID, in: record)
        let contentHash = try requiredString(CloudAssetRecordField.contentHash, in: record)
        let fileExtension = try requiredString(CloudAssetRecordField.fileExtension, in: record)
        let kindRawValue = try requiredString(CloudAssetRecordField.kind, in: record)
        guard let kind = AssetKind(rawValue: kindRawValue) else {
            throw CloudAssetCatalogError.unsupportedKind(recordName: record.recordID.recordName, kind: kindRawValue)
        }

        let pixelWidth = intField(CloudAssetRecordField.pixelWidth, in: record)
        let pixelHeight = intField(CloudAssetRecordField.pixelHeight, in: record)
        let dimensions: AssetDimensions?
        if let pixelWidth, let pixelHeight {
            dimensions = AssetDimensions(width: pixelWidth, height: pixelHeight)
        } else {
            dimensions = nil
        }

        let colorProfileName = stringField(CloudAssetRecordField.colorProfileName, in: record)
        let storageURL = try cachePaths.originalURL(
            cloudAccountID: cloudAccountID,
            libraryID: libraryID,
            contentHash: contentHash,
            fileExtension: fileExtension
        )
        let thumbnailURL = try cachePaths.thumbnailURL(
            cloudAccountID: cloudAccountID,
            libraryID: libraryID,
            contentHash: contentHash
        )

        return AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: try requiredString(CloudAssetRecordField.displayName, in: record),
            originalFileName: try requiredString(CloudAssetRecordField.originalFileName, in: record),
            originalURL: nil,
            sourcePageURL: urlField(CloudAssetRecordField.sourcePageURL, in: record),
            storageURL: storageURL,
            kind: kind,
            fileExtension: fileExtension,
            utiIdentifier: try requiredString(CloudAssetRecordField.utiIdentifier, in: record),
            byteSize: try requiredInt64(CloudAssetRecordField.byteSize, in: record),
            contentHash: contentHash,
            dimensions: dimensions,
            exifMetadata: colorProfileName.map(Self.exifMetadata(colorProfileName:)),
            orientation: intField(CloudAssetRecordField.orientation, in: record),
            colorProfileName: colorProfileName,
            note: stringField(CloudAssetRecordField.note, in: record),
            tags: [],
            folderIDs: [],
            paletteColors: colorsByAssetID[id, default: []],
            thumbnailURL: FileManager.default.fileExists(atPath: thumbnailURL.path) ? thumbnailURL : nil,
            isFavorite: boolField(CloudAssetRecordField.isFavorite, in: record) ?? false,
            isTrashed: boolField(CloudAssetRecordField.isTrashed, in: record) ?? false,
            trashedAt: dateField(CloudAssetRecordField.trashedAt, in: record),
            importedAt: try requiredDate(CloudAssetRecordField.importedAt, in: record),
            updatedAt: try requiredDate(CloudAssetRecordField.updatedAt, in: record),
            availability: AssetFileAvailability(original: .remoteOnly, thumbnail: .remoteOnly, lastError: nil),
            syncState: .synced
        )
    }

    private static func exifMetadata(colorProfileName: String) -> AssetExifMetadata {
        AssetExifMetadata(profileName: colorProfileName)
    }

    private static func colorsByAssetID(from records: [CKRecord]) throws -> [String: [AssetColor]] {
        let colors = try records
            .filter { dateField(CloudAssetColorRecordField.deletedAt, in: $0) == nil }
            .map(color(from:))

        var colorsByAssetID: [String: [AssetColor]] = [:]
        for color in colors {
            colorsByAssetID[color.assetID, default: []].append(color)
        }

        return colorsByAssetID.mapValues { colors in
            colors.sorted {
                if $0.sortIndex == $1.sortIndex {
                    return $0.hex.localizedStandardCompare($1.hex) == .orderedAscending
                }
                return $0.sortIndex < $1.sortIndex
            }
        }
    }

    private static func color(from record: CKRecord) throws -> AssetColor {
        AssetColor(
            id: try requiredString(CloudAssetColorRecordField.id, in: record),
            libraryID: try requiredString(CloudAssetColorRecordField.libraryID, in: record),
            assetID: try requiredString(CloudAssetColorRecordField.assetID, in: record),
            hex: try requiredString(CloudAssetColorRecordField.hex, in: record),
            coverage: try requiredDouble(CloudAssetColorRecordField.coverage, in: record),
            sortIndex: try requiredInt(CloudAssetColorRecordField.sortIndex, in: record)
        )
    }

    private static func requiredString(_ key: String, in record: CKRecord) throws -> String {
        guard let value = stringField(key, in: record), !value.isEmpty else {
            throw CloudAssetCatalogError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func requiredDate(_ key: String, in record: CKRecord) throws -> Date {
        guard let value = dateField(key, in: record) else {
            throw CloudAssetCatalogError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func requiredInt64(_ key: String, in record: CKRecord) throws -> Int64 {
        guard let value = int64Field(key, in: record) else {
            throw CloudAssetCatalogError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func requiredInt(_ key: String, in record: CKRecord) throws -> Int {
        guard let value = intField(key, in: record) else {
            throw CloudAssetCatalogError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func requiredDouble(_ key: String, in record: CKRecord) throws -> Double {
        guard let value = doubleField(key, in: record) else {
            throw CloudAssetCatalogError.missingRequiredField(recordName: record.recordID.recordName, fieldName: key)
        }
        return value
    }

    private static func stringField(_ key: String, in record: CKRecord) -> String? {
        record[key] as? String
    }

    private static func dateField(_ key: String, in record: CKRecord) -> Date? {
        record[key] as? Date
    }

    private static func urlField(_ key: String, in record: CKRecord) -> URL? {
        guard let value = stringField(key, in: record) else {
            return nil
        }
        return URL(string: value)
    }

    private static func boolField(_ key: String, in record: CKRecord) -> Bool? {
        if let value = record[key] as? Bool {
            return value
        }
        if let value = record[key] as? NSNumber {
            return value.boolValue
        }
        return nil
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

    private static func int64Field(_ key: String, in record: CKRecord) -> Int64? {
        if let value = record[key] as? Int64 {
            return value
        }
        if let value = record[key] as? Int {
            return Int64(value)
        }
        if let value = record[key] as? NSNumber {
            return value.int64Value
        }
        return nil
    }

    private static func doubleField(_ key: String, in record: CKRecord) -> Double? {
        if let value = record[key] as? Double {
            return value
        }
        if let value = record[key] as? NSNumber {
            return value.doubleValue
        }
        return nil
    }
}

nonisolated struct CloudKitAssetCatalogProvider: CloudAssetCatalogProviding, @unchecked Sendable {
    private let containerIdentifier: String

    init(containerIdentifier: String = CloudKitConfiguration.containerIdentifier) {
        self.containerIdentifier = containerIdentifier
    }

    func fetchAssetRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        try await fetchRecords(
            recordType: .asset,
            desiredKeys: CloudAssetRecordField.metadataKeys,
            in: libraryZoneName
        )
    }

    func fetchAssetColorRecords(in libraryZoneName: String) async throws -> [CKRecord] {
        try await fetchRecords(
            recordType: .assetColor,
            desiredKeys: CloudAssetColorRecordField.metadataKeys,
            in: libraryZoneName
        )
    }

    private func fetchRecords(
        recordType: CloudRecordType,
        desiredKeys: [String],
        in libraryZoneName: String
    ) async throws -> [CKRecord] {
        let zoneID = CKRecordZone.ID(zoneName: libraryZoneName, ownerName: CKCurrentUserDefaultName)
        let query = CKQuery(recordType: recordType.rawValue, predicate: NSPredicate(value: true))
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

enum CloudAssetCatalogError: LocalizedError, Equatable {
    case missingRequiredField(recordName: String, fieldName: String)
    case unsupportedKind(recordName: String, kind: String)
    case missingCloudAccountID(libraryID: String)

    var errorDescription: String? {
        switch self {
        case .missingRequiredField(let recordName, let fieldName):
            "Cloud asset record \(recordName) is missing required field \(fieldName)."
        case .unsupportedKind(let recordName, let kind):
            "Cloud asset record \(recordName) has unsupported kind \(kind)."
        case .missingCloudAccountID(let libraryID):
            "Cloud library \(libraryID) is missing its iCloud account identity."
        }
    }
}
