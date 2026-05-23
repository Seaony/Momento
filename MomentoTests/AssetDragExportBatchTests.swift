// 中文注释：本测试覆盖拖拽导出批次的完成计数，确保一次拖拽只触发一次完成反馈。
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
