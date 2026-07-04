import XCTest
@testable import Momento

final class AssetCollectionGridUpdateDecisionTests: XCTestCase {
    func testAssetUpdatesAreDrivenByVisibleAssetsRevision() {
        XCTAssertFalse(
            AssetCollectionGridUpdateDecision.shouldApplyAssetChanges(
                previousRevision: 4,
                nextRevision: 4
            )
        )

        XCTAssertTrue(
            AssetCollectionGridUpdateDecision.shouldApplyAssetChanges(
                previousRevision: 4,
                nextRevision: 5
            )
        )

        XCTAssertTrue(
            AssetCollectionGridUpdateDecision.shouldApplyAssetChanges(
                previousRevision: 5,
                nextRevision: 4
            )
        )
    }

    func testChangeSetAnimatesDeletionWithoutRedundantMoves() throws {
        let changeSet = try XCTUnwrap(AssetCollectionGridChangeSet.make(
            from: ["a", "b", "c", "d"].map { Self.makeAsset(id: $0) },
            to: ["a", "c", "d"].map { Self.makeAsset(id: $0) }
        ))

        XCTAssertEqual(changeSet.deletedIndexPaths, [IndexPath(item: 1, section: 0)])
        XCTAssertTrue(changeSet.insertedIndexPaths.isEmpty)
        XCTAssertTrue(changeSet.movedIndexPaths.isEmpty)
    }

    func testChangeSetAnimatesInsertionWithoutRedundantMoves() throws {
        let changeSet = try XCTUnwrap(AssetCollectionGridChangeSet.make(
            from: ["a", "c"].map { Self.makeAsset(id: $0) },
            to: ["a", "b", "c"].map { Self.makeAsset(id: $0) }
        ))

        XCTAssertTrue(changeSet.deletedIndexPaths.isEmpty)
        XCTAssertEqual(changeSet.insertedIndexPaths, [IndexPath(item: 1, section: 0)])
        XCTAssertTrue(changeSet.movedIndexPaths.isEmpty)
    }

    func testChangeSetAnimatesRealReorderingAsMoves() throws {
        let changeSet = try XCTUnwrap(AssetCollectionGridChangeSet.make(
            from: ["a", "b", "c"].map { Self.makeAsset(id: $0) },
            to: ["c", "b", "a"].map { Self.makeAsset(id: $0) }
        ))

        XCTAssertTrue(changeSet.deletedIndexPaths.isEmpty)
        XCTAssertTrue(changeSet.insertedIndexPaths.isEmpty)
        XCTAssertEqual(changeSet.movedIndexPaths, [
            AssetCollectionGridChangeSet.Move(
                from: IndexPath(item: 2, section: 0),
                to: IndexPath(item: 0, section: 0)
            ),
            AssetCollectionGridChangeSet.Move(
                from: IndexPath(item: 0, section: 0),
                to: IndexPath(item: 2, section: 0)
            )
        ])
        XCTAssertTrue(changeSet.isAnimationPractical)
    }

    func testChangeSetRejectsDuplicateIDs() {
        XCTAssertNil(AssetCollectionGridChangeSet.make(
            from: ["a", "a"].map { Self.makeAsset(id: $0) },
            to: ["a"].map { Self.makeAsset(id: $0) }
        ))

        XCTAssertNil(AssetCollectionGridChangeSet.make(
            from: ["a"].map { Self.makeAsset(id: $0) },
            to: ["a", "a"].map { Self.makeAsset(id: $0) }
        ))
    }

    func testChangeSetSkipsPerItemAnimationForLargeReorders() throws {
        let oldIDs = (0..<514).map { "asset-\($0)" }
        let newIDs = oldIDs.reversed()
        let changeSet = try XCTUnwrap(AssetCollectionGridChangeSet.make(
            from: oldIDs.map { Self.makeAsset(id: $0) },
            to: newIDs.map { Self.makeAsset(id: $0) }
        ))

        XCTAssertFalse(changeSet.isAnimationPractical)
    }

    private static func makeAsset(id: AssetItem.ID) -> AssetItem {
        AssetItem(
            id: id,
            libraryID: "library",
            displayName: id,
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/tmp/\(id).png"),
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: id,
            dimensions: AssetDimensions(width: 100, height: 100),
            tags: [],
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
    }
}
