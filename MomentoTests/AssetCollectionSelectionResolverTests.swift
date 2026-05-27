import AppKit
import XCTest
@testable import Momento

final class AssetCollectionSelectionResolverTests: XCTestCase {
    func testPlainClickOnUnselectedItemCollapsesExistingMultiSelection() {
        let clicked = IndexPath(item: 2, section: 0)
        let selected: Set<IndexPath> = [
            IndexPath(item: 0, section: 0),
            IndexPath(item: 1, section: 0),
            clicked
        ]

        XCTAssertEqual(
            AssetCollectionSelectionResolver.replacementIndexPathForPlainClick(
                changedIndexPaths: [clicked],
                selectedIndexPaths: selected,
                mouseDownIndexPath: clicked,
                modifierFlags: []
            ),
            clicked
        )
    }

    func testCommandClickKeepsMultiSelectionBehavior() {
        let clicked = IndexPath(item: 2, section: 0)
        let selected: Set<IndexPath> = [
            IndexPath(item: 0, section: 0),
            clicked
        ]

        XCTAssertNil(
            AssetCollectionSelectionResolver.replacementIndexPathForPlainClick(
                changedIndexPaths: [clicked],
                selectedIndexPaths: selected,
                mouseDownIndexPath: clicked,
                modifierFlags: .command
            )
        )
    }

    func testPlainClickFromNonMouseSelectionDoesNotCollapseSelection() {
        let clicked = IndexPath(item: 2, section: 0)
        let selected: Set<IndexPath> = [
            IndexPath(item: 0, section: 0),
            clicked
        ]

        XCTAssertNil(
            AssetCollectionSelectionResolver.replacementIndexPathForPlainClick(
                changedIndexPaths: [clicked],
                selectedIndexPaths: selected,
                mouseDownIndexPath: nil,
                modifierFlags: []
            )
        )
    }
}
