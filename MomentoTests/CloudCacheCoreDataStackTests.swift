import CoreData
import Foundation
import XCTest
@testable import Momento

final class CloudCacheCoreDataStackTests: XCTestCase {
    private var temporaryRoot: URL!

    override func setUpWithError() throws {
        temporaryRoot = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        try FileManager.default.createDirectory(at: temporaryRoot, withIntermediateDirectories: true)
    }

    override func tearDownWithError() throws {
        if let temporaryRoot {
            try? FileManager.default.removeItem(at: temporaryRoot)
        }
        temporaryRoot = nil
    }

    func testManagedObjectModelContainsOnlyCloudCacheEntities() throws {
        let model = try CloudCacheCoreDataStack.managedObjectModel()
        let entityNames = Set(model.entities.compactMap(\.name))

        XCTAssertEqual(
            entityNames,
            [
                "CachedCloudLibrary",
                "CachedCloudAsset",
                "CachedCloudAssetBlob",
                "CachedCloudAssetColor",
                "CachedCloudFolder",
                "CachedCloudTag",
                "CachedCloudFolderMembership",
                "CachedCloudTagMembership"
            ]
        )
        XCTAssertNil(model.entitiesByName["AssetRecord"])
    }

    func testRecordBackedEntitiesExposeRecordIdentityAndSyncStateFields() throws {
        let model = try CloudCacheCoreDataStack.managedObjectModel()

        for entityName in Self.recordBackedEntityNames {
            let entity = try XCTUnwrap(model.entitiesByName[entityName], entityName)
            XCTAssertAttribute(entity, "recordName", type: .stringAttributeType, optional: false)
            XCTAssertAttribute(entity, "zoneName", type: .stringAttributeType, optional: false)
            XCTAssertAttribute(entity, "ckSystemFieldsBlob", type: .binaryDataAttributeType, optional: true)
            XCTAssertAttribute(entity, "ckRecordChangeTag", type: .stringAttributeType, optional: true)
            XCTAssertAttribute(entity, "dirtyFields", type: .stringAttributeType, optional: true)
            XCTAssertAttribute(entity, "lastError", type: .stringAttributeType, optional: true)
            XCTAssertAttribute(entity, "deletedAt", type: .dateAttributeType, optional: true)
            XCTAssertAttribute(entity, "isDirty", type: .booleanAttributeType, optional: false)
            XCTAssertTrue(
                uniquenessConstraints(for: entity).contains(["zoneName", "recordName"]),
                "\(entityName) must be uniquely addressable by CloudKit record identity."
            )
        }
    }

    func testModelAllowsTombstonesWithOnlyRecordIdentity() throws {
        let stack = try CloudCacheCoreDataStack(
            cloudAccountID: "account",
            libraryID: "library",
            paths: CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)
        )
        let context = stack.container.viewContext
        let deletedAt = Date()

        for entityName in Self.recordBackedEntityNames {
            let entity = try XCTUnwrap(NSEntityDescription.entity(forEntityName: entityName, in: context))
            let object = NSManagedObject(entity: entity, insertInto: context)
            object.setValue("record-\(entityName)", forKey: "recordName")
            object.setValue("MomentoLibrary-library", forKey: "zoneName")
            object.setValue(deletedAt, forKey: "deletedAt")
        }

        XCTAssertNoThrow(try context.save())
    }

    func testStackStoresDatabaseInCloudCachePath() throws {
        let paths = CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)
        let stack = try CloudCacheCoreDataStack(
            cloudAccountID: "account",
            libraryID: "library",
            paths: paths
        )

        let storeURL = try XCTUnwrap(stack.container.persistentStoreCoordinator.persistentStores.first?.url)
        let expectedURL = try paths.cacheDatabaseURL(cloudAccountID: "account", libraryID: "library")
        XCTAssertEqual(storeURL.standardizedFileURL.path, expectedURL.standardizedFileURL.path)
        XCTAssertNotNil(NSEntityDescription.entity(forEntityName: "CachedCloudAsset", in: stack.container.viewContext))
        XCTAssertNil(NSEntityDescription.entity(forEntityName: "AssetRecord", in: stack.container.viewContext))
    }

    func testCloudSpecificEntitiesIncludeColorAndTagLookupFields() throws {
        let model = try CloudCacheCoreDataStack.managedObjectModel()

        XCTAssertAttribute(
            try XCTUnwrap(model.entitiesByName["CachedCloudAssetColor"]),
            "sortIndex",
            type: .integer64AttributeType,
            optional: true
        )
        XCTAssertAttribute(
            try XCTUnwrap(model.entitiesByName["CachedCloudTag"]),
            "normalizedName",
            type: .stringAttributeType,
            optional: true
        )
    }

    private static let recordBackedEntityNames = [
        "CachedCloudLibrary",
        "CachedCloudAsset",
        "CachedCloudAssetBlob",
        "CachedCloudAssetColor",
        "CachedCloudFolder",
        "CachedCloudTag",
        "CachedCloudFolderMembership",
        "CachedCloudTagMembership"
    ]

    private func XCTAssertAttribute(
        _ entity: NSEntityDescription,
        _ name: String,
        type: NSAttributeType,
        optional: Bool,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        guard let attribute = entity.attributesByName[name] else {
            XCTFail("\(entity.name ?? "<unknown>").\(name) is missing.", file: file, line: line)
            return
        }
        XCTAssertEqual(attribute.attributeType, type, file: file, line: line)
        XCTAssertEqual(attribute.isOptional, optional, file: file, line: line)
    }

    private func uniquenessConstraints(for entity: NSEntityDescription) -> [[String]] {
        entity.uniquenessConstraints.map { constraint in
            constraint.compactMap { $0 as? String }
        }
    }
}
