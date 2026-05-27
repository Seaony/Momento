import XCTest
@testable import Momento

final class SidebarFolderDropResolverTests: XCTestCase {
    func testBeforeDropMovesFolderBeforeTargetSibling() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1),
            folder(id: "gamma", sortIndex: 2)
        ]

        let command = MomentoSidebarFolderDropResolver.moveCommand(
            draggedID: "gamma",
            targetFolder: folders[1],
            folders: folders,
            placement: .before
        )

        XCTAssertEqual(command?.parentID, nil)
        XCTAssertEqual(command?.targetID, "beta")
        XCTAssertEqual(command?.insertAfterTarget, false)
    }

    func testAfterDropMovesFolderAfterTargetSibling() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1),
            folder(id: "gamma", sortIndex: 2)
        ]

        let command = MomentoSidebarFolderDropResolver.moveCommand(
            draggedID: "alpha",
            targetFolder: folders[1],
            folders: folders,
            placement: .after
        )

        XCTAssertEqual(command?.parentID, nil)
        XCTAssertEqual(command?.targetID, "beta")
        XCTAssertEqual(command?.insertAfterTarget, true)
    }

    func testIntoDropMovesFolderUnderTarget() {
        let folders = [
            folder(id: "alpha", sortIndex: 0),
            folder(id: "beta", sortIndex: 1)
        ]

        let command = MomentoSidebarFolderDropResolver.moveCommand(
            draggedID: "alpha",
            targetFolder: folders[1],
            folders: folders,
            placement: .into
        )

        XCTAssertEqual(command?.parentID, "beta")
        XCTAssertEqual(command?.targetID, nil)
        XCTAssertEqual(command?.insertAfterTarget, false)
    }

    func testRootEndDropMovesFolderToRootEnd() {
        let folders = [
            folder(id: "parent", sortIndex: 0),
            folder(id: "child", parentID: "parent", sortIndex: 0)
        ]

        let command = MomentoSidebarFolderDropResolver.moveCommand(
            draggedID: "child",
            targetFolder: nil,
            folders: folders,
            placement: .rootEnd
        )

        XCTAssertEqual(command?.parentID, nil)
        XCTAssertEqual(command?.targetID, nil)
        XCTAssertEqual(command?.insertAfterTarget, false)
    }

    func testDropOntoSelfIsRejected() {
        let folders = [folder(id: "alpha", sortIndex: 0)]

        let command = MomentoSidebarFolderDropResolver.moveCommand(
            draggedID: "alpha",
            targetFolder: folders[0],
            folders: folders,
            placement: .into
        )

        XCTAssertNil(command)
    }

    func testDropIntoDescendantIsRejected() {
        let folders = [
            folder(id: "parent", sortIndex: 0),
            folder(id: "child", parentID: "parent", sortIndex: 0),
            folder(id: "grandchild", parentID: "child", sortIndex: 0)
        ]

        let command = MomentoSidebarFolderDropResolver.moveCommand(
            draggedID: "parent",
            targetFolder: folders[2],
            folders: folders,
            placement: .into
        )

        XCTAssertNil(command)
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
