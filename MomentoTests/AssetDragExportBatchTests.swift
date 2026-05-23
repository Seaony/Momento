import XCTest
@testable import Momento

final class AssetDragExportBatchTests: XCTestCase {
    func testBatchCompletesOnceAfterAllPromisesSucceed() {
        let batch = AssetDragExportBatch(expectedFileCount: 3)

        XCTAssertFalse(batch.promiseDidFinish(success: true))
        XCTAssertFalse(batch.promiseDidFinish(success: true))
        XCTAssertTrue(batch.promiseDidFinish(success: true))
        XCTAssertFalse(batch.promiseDidFinish(success: true))
    }

    func testBatchDoesNotCompleteSuccessfullyAfterAnyPromiseFails() {
        let batch = AssetDragExportBatch(expectedFileCount: 2)

        XCTAssertFalse(batch.promiseDidFinish(success: true))
        XCTAssertFalse(batch.promiseDidFinish(success: false))
    }
}
