import CloudKit
import XCTest
@testable import Momento

final class CloudRecordIDBuilderTests: XCTestCase {
    func testGeneratedRecordNamesStayInsideCloudKitConstraints() {
        let longIdentifier = String(repeating: "library/素材🚀", count: 80)
        let names = [
            CloudRecordNaming.libraryRecordName(libraryID: longIdentifier),
            CloudRecordNaming.assetRecordName(contentHash: longIdentifier),
            CloudRecordNaming.blobRecordName(contentHash: longIdentifier),
            CloudRecordNaming.assetColorRecordName(assetID: longIdentifier, sortIndex: 7),
            CloudRecordNaming.folderRecordName(folderID: longIdentifier),
            CloudRecordNaming.tagRecordName(tagID: longIdentifier),
            CloudRecordNaming.folderMembershipRecordName(assetID: longIdentifier, folderID: longIdentifier),
            CloudRecordNaming.tagMembershipRecordName(assetID: longIdentifier, tagID: longIdentifier)
        ]

        for name in names {
            XCTAssertTrue(CloudRecordNaming.isValidRecordName(name), name)
            XCTAssertLessThanOrEqual(name.count, CloudRecordNaming.maximumRecordNameLength)
            XCTAssertTrue(name.unicodeScalars.allSatisfy { $0.value <= 127 })
        }
    }

    func testRecordNamesAreDeterministicAndMatchIOSCompatibilityValues() {
        let assetID = "asset/with unicode 素材"
        let folderID = "folder/with emoji 🚀"

        XCTAssertEqual(
            CloudRecordNaming.folderMembershipRecordName(assetID: assetID, folderID: folderID),
            CloudRecordNaming.folderMembershipRecordName(assetID: assetID, folderID: folderID)
        )
        XCTAssertEqual(
            CloudRecordNaming.tagMembershipRecordName(assetID: assetID, tagID: folderID),
            CloudRecordNaming.tagMembershipRecordName(assetID: assetID, tagID: folderID)
        )
        XCTAssertEqual(
            CloudRecordNaming.assetColorRecordName(assetID: assetID, sortIndex: 7),
            CloudRecordNaming.assetColorRecordName(assetID: assetID, sortIndex: 7)
        )

        XCTAssertEqual(
            CloudRecordNaming.folderMembershipRecordName(assetID: assetID, folderID: folderID),
            "folder-membership:b927d8b3f8ce7ca6"
        )
        XCTAssertEqual(
            CloudRecordNaming.tagMembershipRecordName(assetID: assetID, tagID: folderID),
            "tag-membership:b927d8b3f8ce7ca6"
        )
        XCTAssertEqual(
            CloudRecordNaming.assetColorRecordName(assetID: assetID, sortIndex: 7),
            "asset-color:9abc1799b2c2bc94"
        )
    }

    func testInvalidRecordNamesAreRejected() {
        XCTAssertFalse(CloudRecordNaming.isValidRecordName(""))
        XCTAssertFalse(CloudRecordNaming.isValidRecordName(String(repeating: "a", count: 256)))
        XCTAssertFalse(CloudRecordNaming.isValidRecordName("asset:素材"))
    }

    func testRecordIDsUseCatalogAndPerLibraryZones() {
        let ownerName = "test-owner"
        let libraryID = "library/素材🚀"
        let assetID = "asset-1"
        let folderID = "folder-1"
        let tagID = "tag-1"
        let contentHash = "abcdef123456"

        let libraryRecordID = CloudRecordIDBuilder.cloudLibraryRecordID(
            libraryID: libraryID,
            ownerName: ownerName
        )
        XCTAssertEqual(libraryRecordID.zoneID.zoneName, "MomentoCatalog")
        XCTAssertEqual(libraryRecordID.zoneID.ownerName, ownerName)
        XCTAssertEqual(libraryRecordID.recordName, "library:library----")

        let expectedLibraryZoneName = "MomentoLibrary-library----"
        let recordIDs = [
            CloudRecordIDBuilder.assetRecordID(libraryID: libraryID, contentHash: contentHash, ownerName: ownerName),
            CloudRecordIDBuilder.blobRecordID(libraryID: libraryID, contentHash: contentHash, ownerName: ownerName),
            CloudRecordIDBuilder.assetColorRecordID(libraryID: libraryID, assetID: assetID, sortIndex: 0, ownerName: ownerName),
            CloudRecordIDBuilder.folderRecordID(libraryID: libraryID, folderID: folderID, ownerName: ownerName),
            CloudRecordIDBuilder.tagRecordID(libraryID: libraryID, tagID: tagID, ownerName: ownerName),
            CloudRecordIDBuilder.folderMembershipRecordID(libraryID: libraryID, assetID: assetID, folderID: folderID, ownerName: ownerName),
            CloudRecordIDBuilder.tagMembershipRecordID(libraryID: libraryID, assetID: assetID, tagID: tagID, ownerName: ownerName)
        ]

        for recordID in recordIDs {
            XCTAssertEqual(recordID.zoneID.zoneName, expectedLibraryZoneName)
            XCTAssertEqual(recordID.zoneID.ownerName, ownerName)
            XCTAssertTrue(CloudRecordNaming.isValidRecordName(recordID.recordName), recordID.recordName)
        }
    }

    func testSchemaConstantsMatchIOSCloudRecordTypes() {
        XCTAssertEqual(
            Set(CloudRecordType.allCases.map(\.rawValue)),
            [
                "CloudLibrary",
                "CloudAsset",
                "CloudAssetColor",
                "CloudAssetBlob",
                "CloudFolder",
                "CloudTag",
                "CloudFolderMembership",
                "CloudTagMembership"
            ]
        )
        XCTAssertEqual(CloudKitConfiguration.containerIdentifier, "iCloud.com.seaony.Momento")
        XCTAssertEqual(CloudRecordNaming.catalogZoneName, "MomentoCatalog")
    }
}
