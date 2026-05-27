import AppKit
import XCTest
@testable import Momento

final class DragPasteboardWriterTests: XCTestCase {
    func testFolderItemProviderEncodesFolderPayload() async throws {
        let provider = FolderDragPasteboardWriter.itemProvider(libraryID: "library", folderID: "folder")

        let data = try await loadDataRepresentation(
            from: provider,
            typeIdentifier: FolderDragPasteboardWriter.folderIDTypeIdentifier
        )
        let payload = try JSONDecoder.momento.decode(FolderDragPasteboardPayload.self, from: data)

        XCTAssertEqual(payload.libraryID, "library")
        XCTAssertEqual(payload.folderID, "folder")
    }

    func testAssetFilePromiseProviderWritesAssetPayload() throws {
        let asset = makeAsset()
        let provider = try XCTUnwrap(AssetFilePromiseProvider(
            asset: asset,
            libraryID: "library",
            assetIDs: ["asset-a", "asset-b"],
            primaryAssetID: "asset-a",
            exportBatch: AssetDragExportBatch(expectedFileCount: 2)
        ))
        let pasteboard = NSPasteboard(name: NSPasteboard.Name("DragPasteboardWriterTests"))

        XCTAssertTrue(provider.writableTypes(for: pasteboard).contains(AssetDragPasteboardWriter.assetIDsPasteboardType))

        let data = try XCTUnwrap(
            provider.pasteboardPropertyList(forType: AssetDragPasteboardWriter.assetIDsPasteboardType) as? Data
        )
        let payload = try JSONDecoder.momento.decode(AssetDragPasteboardPayload.self, from: data)

        XCTAssertEqual(payload.libraryID, "library")
        XCTAssertEqual(payload.assetIDs, ["asset-a", "asset-b"])
        XCTAssertEqual(payload.primaryAssetID, "asset-a")
    }

    private func loadDataRepresentation(
        from provider: NSItemProvider,
        typeIdentifier: String
    ) async throws -> Data {
        try await withCheckedThrowingContinuation { continuation in
            _ = provider.loadDataRepresentation(forTypeIdentifier: typeIdentifier) { data, error in
                if let error {
                    continuation.resume(throwing: error)
                    return
                }

                guard let data else {
                    continuation.resume(throwing: DragPasteboardWriterTestError.missingData)
                    return
                }

                continuation.resume(returning: data)
            }
        }
    }

    private func makeAsset() -> AssetItem {
        AssetItem(
            id: "asset-a",
            libraryID: "library",
            displayName: "Asset A",
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/tmp/momento-drag-pasteboard-writer-test.png"),
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: "asset-a-hash",
            dimensions: AssetDimensions(width: 16, height: 16),
            tags: [],
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
    }

    private enum DragPasteboardWriterTestError: Error {
        case missingData
    }
}
