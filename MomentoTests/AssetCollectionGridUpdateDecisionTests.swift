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
}
