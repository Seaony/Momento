import CloudKit
import Foundation

nonisolated protocol CloudAssetUploading: Sendable {
    func uploadAssets(_ assets: [AssetItem], to library: AssetLibrary) async throws -> [AssetItem]
}

nonisolated protocol CloudAssetMetadataWriting: Sendable {
    func saveAssetMetadata(_ assets: [AssetItem], to library: AssetLibrary) async throws -> [AssetItem]
}

nonisolated protocol CloudAssetDeleting: Sendable {
    func deleteAssets(_ assets: [AssetItem], from library: AssetLibrary) async throws
}

nonisolated protocol CloudAssetUploadProviding: Sendable {
    func saveRecords(_ records: [CKRecord], in libraryZoneName: String) async throws -> [CKRecord]
}

actor CloudAssetUploadService: CloudAssetUploading, CloudAssetMetadataWriting, CloudAssetDeleting {
    private static let maximumPaletteColorCount = 8

    private let provider: CloudAssetUploadProviding

    init(provider: CloudAssetUploadProviding = CloudKitAssetUploadProvider()) {
        self.provider = provider
    }

    func uploadAssets(_ assets: [AssetItem], to library: AssetLibrary) async throws -> [AssetItem] {
        guard !assets.isEmpty else {
            return []
        }

        var recordsToSave: [CKRecord] = []
        for asset in assets {
            guard FileManager.default.fileExists(atPath: asset.storageURL.path) else {
                throw CloudAssetUploadError.missingLocalOriginal(assetID: asset.id)
            }
            recordsToSave.append(metadataRecord(for: asset, in: library))
            recordsToSave.append(blobRecord(for: asset, in: library))
            recordsToSave.append(contentsOf: colorRecords(for: asset, in: library))
        }

        _ = try await provider.saveRecords(recordsToSave, in: library.libraryZoneName)
        return assets.map { asset in
            var uploadedAsset = asset
            uploadedAsset.availability.original = .local
            uploadedAsset.availability.lastError = nil
            uploadedAsset.syncState = .synced
            return uploadedAsset
        }
    }

    func saveAssetMetadata(_ assets: [AssetItem], to library: AssetLibrary) async throws -> [AssetItem] {
        guard !assets.isEmpty else {
            return []
        }

        let recordsToSave = assets.map { metadataRecord(for: $0, in: library) }
        _ = try await provider.saveRecords(recordsToSave, in: library.libraryZoneName)
        return assets.map { asset in
            var savedAsset = asset
            savedAsset.availability.lastError = nil
            savedAsset.syncState = .synced
            return savedAsset
        }
    }

    func deleteAssets(_ assets: [AssetItem], from library: AssetLibrary) async throws {
        guard !assets.isEmpty else {
            return
        }

        let tombstoneDate = Date()
        let recordsToSave = assets.flatMap { asset in
            deletedRecords(for: asset, in: library, deletedAt: tombstoneDate)
        }
        _ = try await provider.saveRecords(recordsToSave, in: library.libraryZoneName)
    }

    private func metadataRecord(for asset: AssetItem, in library: AssetLibrary) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.asset.rawValue,
            recordID: recordID(
                name: CloudRecordNaming.assetRecordName(contentHash: asset.contentHash),
                in: library
            )
        )
        record[CloudAssetRecordField.id] = asset.id
        record[CloudAssetRecordField.libraryID] = asset.libraryID
        record[CloudAssetRecordField.contentHash] = asset.contentHash
        record[CloudAssetRecordField.displayName] = asset.displayName
        record[CloudAssetRecordField.originalFileName] = asset.originalFileName
        record[CloudAssetRecordField.fileExtension] = asset.fileExtension
        record[CloudAssetRecordField.utiIdentifier] = asset.utiIdentifier
        record[CloudAssetRecordField.kind] = asset.kind.rawValue
        record[CloudAssetRecordField.byteSize] = NSNumber(value: asset.byteSize)
        record[CloudAssetRecordField.isFavorite] = asset.isFavorite
        record[CloudAssetRecordField.isTrashed] = asset.isTrashed
        record[CloudAssetRecordField.importedAt] = asset.importedAt
        record[CloudAssetRecordField.updatedAt] = asset.updatedAt

        if let width = asset.dimensions?.width {
            record[CloudAssetRecordField.pixelWidth] = width
        }
        if let height = asset.dimensions?.height {
            record[CloudAssetRecordField.pixelHeight] = height
        }
        if let orientation = asset.orientation {
            record[CloudAssetRecordField.orientation] = orientation
        }
        if let colorProfileName = asset.colorProfileName {
            record[CloudAssetRecordField.colorProfileName] = colorProfileName
        }
        record[CloudAssetRecordField.sourcePageURL] = asset.sourcePageURL?.absoluteString
        record[CloudAssetRecordField.note] = asset.note
        record[CloudAssetRecordField.trashedAt] = asset.trashedAt
        record[CloudAssetRecordField.deletedAt] = nil

        return record
    }

    private func blobRecord(for asset: AssetItem, in library: AssetLibrary) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.assetBlob.rawValue,
            recordID: recordID(
                name: CloudRecordNaming.blobRecordName(contentHash: asset.contentHash),
                in: library
            )
        )
        record[CloudAssetBlobRecordField.id] = asset.id
        record[CloudAssetBlobRecordField.libraryID] = asset.libraryID
        record[CloudAssetBlobRecordField.contentHash] = asset.contentHash
        record[CloudAssetBlobRecordField.originalFile] = CKAsset(fileURL: asset.storageURL)
        record[CloudAssetBlobRecordField.byteSize] = NSNumber(value: asset.byteSize)
        record[CloudAssetBlobRecordField.uploadedAt] = Date()
        record[CloudAssetBlobRecordField.deletedAt] = nil
        return record
    }

    private func deletedRecords(for asset: AssetItem, in library: AssetLibrary, deletedAt: Date) -> [CKRecord] {
        var records: [CKRecord] = [
            deletedAssetRecord(for: asset, in: library, deletedAt: deletedAt),
            deletedBlobRecord(for: asset, in: library, deletedAt: deletedAt)
        ]
        records.append(contentsOf: deletedColorRecords(for: asset, in: library, deletedAt: deletedAt))
        return records
    }

    private func deletedAssetRecord(for asset: AssetItem, in library: AssetLibrary, deletedAt: Date) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.asset.rawValue,
            recordID: recordID(
                name: CloudRecordNaming.assetRecordName(contentHash: asset.contentHash),
                in: library
            )
        )
        record[CloudAssetRecordField.id] = asset.id
        record[CloudAssetRecordField.libraryID] = library.id
        record[CloudAssetRecordField.contentHash] = asset.contentHash
        record[CloudAssetRecordField.isTrashed] = true
        record[CloudAssetRecordField.trashedAt] = asset.trashedAt ?? deletedAt
        record[CloudAssetRecordField.updatedAt] = deletedAt
        record[CloudAssetRecordField.deletedAt] = deletedAt
        return record
    }

    private func deletedBlobRecord(for asset: AssetItem, in library: AssetLibrary, deletedAt: Date) -> CKRecord {
        let record = CKRecord(
            recordType: CloudRecordType.assetBlob.rawValue,
            recordID: recordID(
                name: CloudRecordNaming.blobRecordName(contentHash: asset.contentHash),
                in: library
            )
        )
        record[CloudAssetBlobRecordField.id] = asset.id
        record[CloudAssetBlobRecordField.libraryID] = library.id
        record[CloudAssetBlobRecordField.contentHash] = asset.contentHash
        record[CloudAssetBlobRecordField.deletedAt] = deletedAt
        return record
    }

    private func deletedColorRecords(for asset: AssetItem, in library: AssetLibrary, deletedAt: Date) -> [CKRecord] {
        (0..<Self.maximumPaletteColorCount).map { sortIndex in
            let record = CKRecord(
                recordType: CloudRecordType.assetColor.rawValue,
                recordID: recordID(
                    name: CloudRecordNaming.assetColorRecordName(assetID: asset.id, sortIndex: sortIndex),
                    in: library
                )
            )
            record[CloudAssetColorRecordField.id] = "\(asset.id)-color-\(sortIndex)"
            record[CloudAssetColorRecordField.libraryID] = library.id
            record[CloudAssetColorRecordField.assetID] = asset.id
            record[CloudAssetColorRecordField.sortIndex] = sortIndex
            record[CloudAssetColorRecordField.deletedAt] = deletedAt
            return record
        }
    }

    private func colorRecords(for asset: AssetItem, in library: AssetLibrary) -> [CKRecord] {
        let activeRecords = asset.paletteColors.map { color in
            let record = CKRecord(
                recordType: CloudRecordType.assetColor.rawValue,
                recordID: recordID(
                    name: CloudRecordNaming.assetColorRecordName(assetID: asset.id, sortIndex: color.sortIndex),
                    in: library
                )
            )
            record[CloudAssetColorRecordField.id] = color.id
            record[CloudAssetColorRecordField.libraryID] = color.libraryID
            record[CloudAssetColorRecordField.assetID] = color.assetID
            record[CloudAssetColorRecordField.hex] = color.hex
            record[CloudAssetColorRecordField.coverage] = color.coverage
            record[CloudAssetColorRecordField.sortIndex] = color.sortIndex
            record[CloudAssetColorRecordField.deletedAt] = nil
            return record
        }

        guard shouldReplacePaletteRecords(for: asset) else {
            return activeRecords
        }

        let activeSortIndexes = Set(asset.paletteColors.map(\.sortIndex))
        let tombstoneDate = Date()
        let tombstoneRecords = (0..<Self.maximumPaletteColorCount)
            .filter { !activeSortIndexes.contains($0) }
            .map { sortIndex in
                let record = CKRecord(
                    recordType: CloudRecordType.assetColor.rawValue,
                    recordID: recordID(
                        name: CloudRecordNaming.assetColorRecordName(assetID: asset.id, sortIndex: sortIndex),
                        in: library
                    )
                )
                record[CloudAssetColorRecordField.id] = "\(asset.id)-color-\(sortIndex)"
                record[CloudAssetColorRecordField.libraryID] = library.id
                record[CloudAssetColorRecordField.assetID] = asset.id
                record[CloudAssetColorRecordField.sortIndex] = sortIndex
                record[CloudAssetColorRecordField.deletedAt] = tombstoneDate
                return record
            }
        return activeRecords + tombstoneRecords
    }

    private func shouldReplacePaletteRecords(for asset: AssetItem) -> Bool {
        switch asset.kind {
        case .image, .gif:
            return true
        case .svg, .video, .pdf:
            return !asset.paletteColors.isEmpty
        }
    }

    private func recordID(name: String, in library: AssetLibrary) -> CKRecord.ID {
        CKRecord.ID(
            recordName: name,
            zoneID: CKRecordZone.ID(zoneName: library.libraryZoneName, ownerName: CKCurrentUserDefaultName)
        )
    }
}

nonisolated struct CloudKitAssetUploadProvider: CloudAssetUploadProviding, @unchecked Sendable {
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

enum CloudAssetUploadError: LocalizedError, Equatable {
    case missingLocalOriginal(assetID: String)

    var errorDescription: String? {
        switch self {
        case .missingLocalOriginal(let assetID):
            "Asset \(assetID) is missing a local original file for CloudKit upload."
        }
    }
}
