import Foundation

nonisolated enum CloudKitConfiguration {
    static let containerIdentifier = "iCloud.com.seaony.Momento"
}

enum CloudRecordType: String, CaseIterable, Hashable, Codable, Sendable {
    case library = "CloudLibrary"
    case asset = "CloudAsset"
    case assetColor = "CloudAssetColor"
    case assetBlob = "CloudAssetBlob"
    case folder = "CloudFolder"
    case tag = "CloudTag"
    case folderMembership = "CloudFolderMembership"
    case tagMembership = "CloudTagMembership"
}

struct CloudLibraryDescriptor: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var displayName: String
    var libraryZoneName: String
    var createdAt: Date
    var updatedAt: Date
    var deletedAt: Date?
    var schemaVersion: Int
    var syncState: CloudLibrarySyncState
    var lastError: String?
}

enum CloudLibrarySyncState: String, Hashable, Codable, Sendable {
    case synced
    case syncing
    case waitingForNetwork
    case waitingForICloudSignIn
    case uploadFailed
    case downloadFailed
    case quotaBlocked
    case unsupportedSchema
}

nonisolated enum CloudLibraryRecordField {
    static let id = "id"
    static let displayName = "displayName"
    static let libraryZoneName = "libraryZoneName"
    static let createdAt = "createdAt"
    static let updatedAt = "updatedAt"
    static let deletedAt = "deletedAt"
    static let schemaVersion = "schemaVersion"

    static let metadataKeys = [
        id,
        displayName,
        libraryZoneName,
        createdAt,
        updatedAt,
        deletedAt,
        schemaVersion
    ]
}

nonisolated enum CloudAssetRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let contentHash = "contentHash"
    static let displayName = "displayName"
    static let originalFileName = "originalFileName"
    static let fileExtension = "fileExtension"
    static let utiIdentifier = "utiIdentifier"
    static let kind = "kind"
    static let byteSize = "byteSize"
    static let pixelWidth = "pixelWidth"
    static let pixelHeight = "pixelHeight"
    static let orientation = "orientation"
    static let colorProfileName = "colorProfileName"
    static let sourcePageURL = "sourcePageURL"
    static let note = "note"
    static let isFavorite = "isFavorite"
    static let isTrashed = "isTrashed"
    static let trashedAt = "trashedAt"
    static let importedAt = "importedAt"
    static let updatedAt = "updatedAt"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        contentHash,
        displayName,
        originalFileName,
        fileExtension,
        utiIdentifier,
        kind,
        byteSize,
        pixelWidth,
        pixelHeight,
        orientation,
        colorProfileName,
        sourcePageURL,
        note,
        isFavorite,
        isTrashed,
        trashedAt,
        importedAt,
        updatedAt,
        deletedAt
    ]
}

nonisolated enum CloudAssetColorRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let assetID = "assetID"
    static let hex = "hex"
    static let coverage = "coverage"
    static let sortIndex = "sortIndex"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        assetID,
        hex,
        coverage,
        sortIndex,
        deletedAt
    ]
}

nonisolated enum CloudAssetBlobRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let contentHash = "contentHash"
    static let originalFile = "originalFile"
    static let byteSize = "byteSize"
    static let uploadedAt = "uploadedAt"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        contentHash,
        byteSize,
        uploadedAt,
        deletedAt
    ]
}

nonisolated enum CloudFolderRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let name = "name"
    static let parentID = "parentID"
    static let sortIndex = "sortIndex"
    static let createdAt = "createdAt"
    static let updatedAt = "updatedAt"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        name,
        parentID,
        sortIndex,
        createdAt,
        updatedAt,
        deletedAt
    ]
}

nonisolated enum CloudTagRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let name = "name"
    static let normalizedName = "normalizedName"
    static let colorHex = "colorHex"
    static let createdAt = "createdAt"
    static let updatedAt = "updatedAt"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        name,
        normalizedName,
        colorHex,
        createdAt,
        updatedAt,
        deletedAt
    ]
}

nonisolated enum CloudFolderMembershipRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let assetID = "assetID"
    static let folderID = "folderID"
    static let createdAt = "createdAt"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        assetID,
        folderID,
        createdAt,
        deletedAt
    ]
}

nonisolated enum CloudTagMembershipRecordField {
    static let id = "id"
    static let libraryID = "libraryID"
    static let assetID = "assetID"
    static let tagID = "tagID"
    static let createdAt = "createdAt"
    static let deletedAt = "deletedAt"

    static let metadataKeys = [
        id,
        libraryID,
        assetID,
        tagID,
        createdAt,
        deletedAt
    ]
}

struct CloudRecordIdentity: Hashable, Codable, Sendable {
    var recordType: CloudRecordType
    var recordName: String
    var zoneName: String
}

nonisolated enum CloudRecordNaming {
    static let catalogZoneName = "MomentoCatalog"
    static let maximumRecordNameLength = 255

    static func libraryZoneName(libraryID: String) -> String {
        "MomentoLibrary-\(asciiRecordComponent(libraryID))"
    }

    static func libraryRecordName(libraryID: String) -> String {
        "library:\(asciiRecordComponent(libraryID))"
    }

    static func assetRecordName(contentHash: String) -> String {
        "asset:\(asciiRecordComponent(contentHash))"
    }

    static func blobRecordName(contentHash: String) -> String {
        "blob:\(asciiRecordComponent(contentHash))"
    }

    static func assetColorRecordName(assetID: String, sortIndex: Int) -> String {
        "asset-color:\(shortStableKey(assetID, String(sortIndex)))"
    }

    static func folderRecordName(folderID: String) -> String {
        "folder:\(asciiRecordComponent(folderID))"
    }

    static func tagRecordName(tagID: String) -> String {
        "tag:\(asciiRecordComponent(tagID))"
    }

    static func folderMembershipRecordName(assetID: String, folderID: String) -> String {
        "folder-membership:\(shortStableKey(assetID, folderID))"
    }

    static func tagMembershipRecordName(assetID: String, tagID: String) -> String {
        "tag-membership:\(shortStableKey(assetID, tagID))"
    }

    static func isValidRecordName(_ recordName: String) -> Bool {
        !recordName.isEmpty
            && recordName.count <= maximumRecordNameLength
            && recordName.unicodeScalars.allSatisfy { $0.value <= 127 }
    }

    private static func asciiRecordComponent(_ value: String) -> String {
        let allowed = Set("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_")
        let mapped = value.map { character in
            allowed.contains(character) ? character : "-"
        }
        let result = String(mapped)
        if result.count <= 180 {
            return result
        }
        return shortStableKey(result)
    }

    private static func shortStableKey(_ values: String...) -> String {
        let joined = values.joined(separator: "|")
        let scalars = joined.unicodeScalars.map(\.value)
        let hash = scalars.reduce(UInt64(14_695_981_039_346_656_037)) { partial, scalar in
            (partial ^ UInt64(scalar)).multipliedReportingOverflow(by: 1_099_511_628_211).partialValue
        }
        return String(hash, radix: 16)
    }
}
