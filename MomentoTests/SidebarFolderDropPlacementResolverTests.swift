import XCTest
@testable import Momento

final class SidebarFolderDropPlacementResolverTests: XCTestCase {
    func testTopEdgeMapsToBeforePlacement() {
        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.placement(
                localY: 29,
                rowHeight: 30,
                edgeZoneHeight: 8
            ),
            .before
        )
    }

    func testBottomEdgeMapsToAfterPlacement() {
        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.placement(
                localY: 1,
                rowHeight: 30,
                edgeZoneHeight: 8
            ),
            .after
        )
    }

    func testMiddleMapsToIntoPlacement() {
        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.placement(
                localY: 15,
                rowHeight: 30,
                edgeZoneHeight: 8
            ),
            .into
        )
    }
}
