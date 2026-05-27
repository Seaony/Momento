import XCTest
@testable import Momento

final class SidebarFolderDropPlacementResolverTests: XCTestCase {
    func testFolderDropSurfacesIncludeInsertionTargetBeforeEachVisibleFolder() {
        XCTAssertEqual(
            MomentoSidebarFolderDropSurfaceResolver.surfaces(folderIDs: ["alpha", "beta"]),
            [
                .insertionBefore("alpha"),
                .folder("alpha"),
                .insertionBefore("beta"),
                .folder("beta"),
                .rootEnd
            ]
        )
    }

    func testTopEdgeMapsToBeforePlacement() {
        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.placement(
                localY: 1,
                rowHeight: 30,
                edgeZoneHeight: 8
            ),
            .before
        )
    }

    func testBottomEdgeMapsToAfterPlacement() {
        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.placement(
                localY: 29,
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

    func testSameParentDragFromLowerSiblingDefaultsToBeforeTarget() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1)
        ]

        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.effectivePlacement(
                rawPlacement: .into,
                draggedID: "beta",
                targetFolder: folders[0],
                folders: folders,
                prefersNesting: false
            ),
            .before
        )
    }

    func testExplicitAfterPlacementIsPreservedForSiblingDrop() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1)
        ]

        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.effectivePlacement(
                rawPlacement: .after,
                draggedID: "beta",
                targetFolder: folders[0],
                folders: folders,
                prefersNesting: false
            ),
            .after
        )
    }

    func testExplicitBeforePlacementIsPreservedForSiblingDrop() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1),
            folder(id: "gamma", sortIndex: 2)
        ]

        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.effectivePlacement(
                rawPlacement: .before,
                draggedID: "alpha",
                targetFolder: folders[2],
                folders: folders,
                prefersNesting: false
            ),
            .before
        )
    }

    func testSameParentDragFromUpperSiblingDefaultsToAfterTarget() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1)
        ]

        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.effectivePlacement(
                rawPlacement: .into,
                draggedID: "alpha",
                targetFolder: folders[1],
                folders: folders,
                prefersNesting: false
            ),
            .after
        )
    }

    func testSameParentDragCanStillNestWhenPointerPrefersNesting() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1)
        ]

        XCTAssertEqual(
            MomentoSidebarFolderDropPlacementResolver.effectivePlacement(
                rawPlacement: .into,
                draggedID: "beta",
                targetFolder: folders[0],
                folders: folders,
                prefersNesting: true
            ),
            .into
        )
    }

    private func folder(id: AssetFolder.ID, parentID: AssetFolder.ID? = nil, sortIndex: Int) -> AssetFolder {
        AssetFolder(
            id: id,
            libraryID: "library",
            name: id.capitalized,
            parentID: parentID,
            sortIndex: sortIndex,
            createdAt: Date(timeIntervalSince1970: TimeInterval(sortIndex)),
            updatedAt: Date(timeIntervalSince1970: TimeInterval(sortIndex))
        )
    }
}
