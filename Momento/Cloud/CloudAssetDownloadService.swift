import CloudKit
import Foundation
import ImageIO
import UniformTypeIdentifiers

nonisolated protocol CloudAssetDownloading: Sendable {
    func downloadOriginals(
        for assets: [AssetItem],
        in library: AssetLibrary
    ) async throws -> [AssetItem]
}

nonisolated protocol CloudAssetDownloadProviding: Sendable {
    func fetchAssetBlobRecords(recordIDs: [CKRecord.ID]) async throws -> [CKRecord]
}

actor CloudAssetDownloadService: CloudAssetDownloading {
    private let cachePaths: CloudLibraryCachePaths
    private let provider: CloudAssetDownloadProviding
    private let colorAnalysisService: AssetColorAnalysisService

    init(
        cachePaths: CloudLibraryCachePaths = CloudLibraryCachePaths(),
        provider: CloudAssetDownloadProviding = CloudKitAssetDownloadProvider()
    ) {
        self.cachePaths = cachePaths
        self.provider = provider
        self.colorAnalysisService = AssetColorAnalysisService()
    }

    func downloadOriginals(
        for assets: [AssetItem],
        in library: AssetLibrary
    ) async throws -> [AssetItem] {
        guard !assets.isEmpty else {
            return []
        }
        let cloudAccountID = try Self.requiredCloudAccountID(in: library)

        let recordIDs = assets.map { asset in
            CKRecord.ID(
                recordName: CloudRecordNaming.blobRecordName(contentHash: asset.contentHash),
                zoneID: CKRecordZone.ID(zoneName: library.libraryZoneName, ownerName: CKCurrentUserDefaultName)
            )
        }
        let records = try await provider.fetchAssetBlobRecords(recordIDs: recordIDs)
        let recordsByContentHash = Dictionary(uniqueKeysWithValues: try records.map { record in
            (try contentHash(in: record), record)
        })

        try cachePaths.prepareLibraryDirectories(cloudAccountID: cloudAccountID, libraryID: library.id)
        return try assets.map { asset in
            guard let record = recordsByContentHash[asset.contentHash] else {
                throw CloudAssetDownloadError.missingBlobRecord(assetID: asset.id)
            }
            guard let originalFile = record[CloudAssetBlobRecordField.originalFile] as? CKAsset,
                  let stagedURL = originalFile.fileURL else {
                throw CloudAssetDownloadError.missingOriginalFile(assetID: asset.id)
            }

            let destinationURL = try cachePaths.originalURL(
                cloudAccountID: cloudAccountID,
                libraryID: library.id,
                contentHash: asset.contentHash,
                fileExtension: asset.fileExtension
            )
            try FileManager.default.createDirectory(
                at: destinationURL.deletingLastPathComponent(),
                withIntermediateDirectories: true
            )
            if !FileManager.default.fileExists(atPath: destinationURL.path) {
                try FileManager.default.copyItem(at: stagedURL, to: destinationURL)
                try cachePaths.applyDurabilityAttributes(to: destinationURL, role: .syncedOriginal)
            }

            let thumbnailURL = try generateThumbnail(
                for: destinationURL,
                asset: asset,
                library: library,
                cloudAccountID: cloudAccountID
            )

            var downloadedAsset = asset
            downloadedAsset.storageURL = destinationURL
            downloadedAsset.thumbnailURL = thumbnailURL
            downloadedAsset.paletteColors = colorAnalysisService.paletteColors(
                for: thumbnailURL ?? destinationURL,
                libraryID: asset.libraryID,
                assetID: asset.id,
                maxColorCount: 8
            )
            downloadedAsset.availability.original = .local
            downloadedAsset.availability.thumbnail = thumbnailURL == nil ? .generationPending : .local
            downloadedAsset.availability.lastError = nil
            downloadedAsset.syncState = .synced
            return downloadedAsset
        }
    }

    private static func requiredCloudAccountID(in library: AssetLibrary) throws -> String {
        guard let cloudAccountID = library.cloudAccountID?.trimmingCharacters(in: .whitespacesAndNewlines),
              !cloudAccountID.isEmpty else {
            throw CloudAssetDownloadError.missingCloudAccountID(libraryID: library.id)
        }
        return cloudAccountID
    }

    private func generateThumbnail(
        for sourceURL: URL,
        asset: AssetItem,
        library: AssetLibrary,
        cloudAccountID: String
    ) throws -> URL? {
        guard asset.kind == .image || asset.kind == .gif,
              let source = CGImageSourceCreateWithURL(sourceURL as CFURL, nil) else {
            return nil
        }

        let options: [CFString: Any] = [
            kCGImageSourceCreateThumbnailFromImageAlways: true,
            kCGImageSourceCreateThumbnailWithTransform: true,
            kCGImageSourceThumbnailMaxPixelSize: 512
        ]
        guard let thumbnail = CGImageSourceCreateThumbnailAtIndex(source, 0, options as CFDictionary) else {
            return nil
        }

        let destinationURL = try cachePaths.thumbnailURL(
            cloudAccountID: cloudAccountID,
            libraryID: library.id,
            contentHash: asset.contentHash
        )
        try FileManager.default.createDirectory(
            at: destinationURL.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        let temporaryURL = destinationURL
            .deletingLastPathComponent()
            .appendingPathComponent(".\(destinationURL.lastPathComponent).tmp-\(UUID().uuidString)")

        guard let destination = CGImageDestinationCreateWithURL(
            temporaryURL as CFURL,
            UTType.png.identifier as CFString,
            1,
            nil
        ) else {
            return nil
        }
        CGImageDestinationAddImage(destination, thumbnail, nil)
        guard CGImageDestinationFinalize(destination) else {
            try? FileManager.default.removeItem(at: temporaryURL)
            return nil
        }

        if FileManager.default.fileExists(atPath: destinationURL.path) {
            try FileManager.default.removeItem(at: destinationURL)
        }
        try FileManager.default.moveItem(at: temporaryURL, to: destinationURL)
        try cachePaths.applyDurabilityAttributes(to: destinationURL, role: .thumbnail)
        return destinationURL
    }

    private func contentHash(in record: CKRecord) throws -> String {
        guard let contentHash = record[CloudAssetBlobRecordField.contentHash] as? String,
              !contentHash.isEmpty else {
            throw CloudAssetDownloadError.missingContentHash(recordName: record.recordID.recordName)
        }
        return contentHash
    }
}

nonisolated struct CloudKitAssetDownloadProvider: CloudAssetDownloadProviding, @unchecked Sendable {
    private let containerIdentifier: String

    init(containerIdentifier: String = CloudKitConfiguration.containerIdentifier) {
        self.containerIdentifier = containerIdentifier
    }

    func fetchAssetBlobRecords(recordIDs: [CKRecord.ID]) async throws -> [CKRecord] {
        let response = try await database.records(
            for: recordIDs,
            desiredKeys: [
                CloudAssetBlobRecordField.id,
                CloudAssetBlobRecordField.libraryID,
                CloudAssetBlobRecordField.contentHash,
                CloudAssetBlobRecordField.originalFile,
                CloudAssetBlobRecordField.byteSize,
                CloudAssetBlobRecordField.uploadedAt
            ]
        )
        return try response.map { try $0.1.get() }
    }

    private var database: CKDatabase {
        CKContainer(identifier: containerIdentifier).privateCloudDatabase
    }
}

enum CloudAssetDownloadError: LocalizedError, Equatable {
    case missingBlobRecord(assetID: String)
    case missingOriginalFile(assetID: String)
    case missingContentHash(recordName: String)
    case missingCloudAccountID(libraryID: String)

    var errorDescription: String? {
        switch self {
        case .missingBlobRecord(let assetID):
            "Asset \(assetID) does not have a CloudKit blob record."
        case .missingOriginalFile(let assetID):
            "Asset \(assetID) does not have a CloudKit original file."
        case .missingContentHash(let recordName):
            "Cloud asset blob record \(recordName) is missing its content hash."
        case .missingCloudAccountID(let libraryID):
            "Cloud library \(libraryID) is missing its iCloud account identity."
        }
    }
}
