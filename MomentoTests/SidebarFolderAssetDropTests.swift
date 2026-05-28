import AppKit
import XCTest
@testable import Momento

final class SidebarFolderAssetDropTests: XCTestCase {
    @MainActor
    func testPerformsSameLibraryAssetDropFromPasteboard() throws {
        var capturedAssetIDs: Set<AssetItem.ID>?
        let view = SidebarFolderAssetDropView(
            currentLibraryID: "library",
            onTargetedChange: { _ in },
            onDropAssetIDs: { capturedAssetIDs = $0 }
        )
        let coordinator = view.makeCoordinator()
        let pasteboard = try makePasteboard(
            libraryID: "library",
            assetIDs: ["asset-a", "asset-b"],
            primaryAssetID: "asset-a"
        )

        XCTAssertTrue(coordinator.performAssetDrop(from: pasteboard))
        XCTAssertEqual(try XCTUnwrap(capturedAssetIDs), ["asset-a", "asset-b"])
    }

    @MainActor
    func testRejectsCrossLibraryAssetDropFromPasteboard() throws {
        var capturedAssetIDs: Set<AssetItem.ID>?
        let view = SidebarFolderAssetDropView(
            currentLibraryID: "library-b",
            onTargetedChange: { _ in },
            onDropAssetIDs: { capturedAssetIDs = $0 }
        )
        let coordinator = view.makeCoordinator()
        let pasteboard = try makePasteboard(
            libraryID: "library-a",
            assetIDs: ["asset-a"],
            primaryAssetID: "asset-a"
        )

        XCTAssertFalse(coordinator.performAssetDrop(from: pasteboard))
        XCTAssertNil(capturedAssetIDs)
    }

    @MainActor
    private func makePasteboard(
        libraryID: AssetLibrary.ID,
        assetIDs: [AssetItem.ID],
        primaryAssetID: AssetItem.ID
    ) throws -> NSPasteboard {
        let pasteboard = NSPasteboard(name: NSPasteboard.Name("SidebarFolderAssetDropTests.\(UUID().uuidString)"))
        pasteboard.clearContents()
        let data = try XCTUnwrap(AssetDragPasteboardWriter.encodedPayload(
            libraryID: libraryID,
            assetIDs: assetIDs,
            primaryAssetID: primaryAssetID
        ))
        pasteboard.setData(data, forType: AssetDragPasteboardWriter.assetIDsPasteboardType)
        return pasteboard
    }
}
