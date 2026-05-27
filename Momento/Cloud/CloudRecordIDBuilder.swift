import CloudKit
import Foundation

nonisolated enum CloudRecordIDBuilder {
    static func catalogZoneID(ownerName: String = CKCurrentUserDefaultName) -> CKRecordZone.ID {
        CKRecordZone.ID(zoneName: CloudRecordNaming.catalogZoneName, ownerName: ownerName)
    }

    static func libraryZoneID(
        libraryID: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecordZone.ID {
        CKRecordZone.ID(
            zoneName: CloudRecordNaming.libraryZoneName(libraryID: libraryID),
            ownerName: ownerName
        )
    }

    static func cloudLibraryRecordID(
        libraryID: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.libraryRecordName(libraryID: libraryID),
            zoneID: catalogZoneID(ownerName: ownerName)
        )
    }

    static func assetRecordID(
        libraryID: String,
        contentHash: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.assetRecordName(contentHash: contentHash),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }

    static func blobRecordID(
        libraryID: String,
        contentHash: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.blobRecordName(contentHash: contentHash),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }

    static func assetColorRecordID(
        libraryID: String,
        assetID: String,
        sortIndex: Int,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.assetColorRecordName(assetID: assetID, sortIndex: sortIndex),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }

    static func folderRecordID(
        libraryID: String,
        folderID: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.folderRecordName(folderID: folderID),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }

    static func tagRecordID(
        libraryID: String,
        tagID: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.tagRecordName(tagID: tagID),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }

    static func folderMembershipRecordID(
        libraryID: String,
        assetID: String,
        folderID: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.folderMembershipRecordName(assetID: assetID, folderID: folderID),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }

    static func tagMembershipRecordID(
        libraryID: String,
        assetID: String,
        tagID: String,
        ownerName: String = CKCurrentUserDefaultName
    ) -> CKRecord.ID {
        CKRecord.ID(
            recordName: CloudRecordNaming.tagMembershipRecordName(assetID: assetID, tagID: tagID),
            zoneID: libraryZoneID(libraryID: libraryID, ownerName: ownerName)
        )
    }
}
