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

    func testAssetFilePromiseProviderRejectsCopyWhenSourceValidatorFails() throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(
            "momento-drag-promise-tests-\(UUID().uuidString)",
            isDirectory: true
        )
        defer { try? FileManager.default.removeItem(at: directory) }
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)

        let sourceURL = directory.appendingPathComponent("source.png")
        try Data("asset".utf8).write(to: sourceURL)
        let destinationURL = directory.appendingPathComponent("destination.png")
        let asset = makeAsset(storageURL: sourceURL)
        let provider = try XCTUnwrap(AssetFilePromiseProvider(
            asset: asset,
            libraryID: "library",
            assetIDs: ["asset-a"],
            primaryAssetID: "asset-a",
            exportBatch: AssetDragExportBatch(expectedFileCount: 1),
            sourceAccessValidator: {
                throw LibraryStorageError.ubiquitousLibraryPackageUnsupported
            }
        ))

        let promiseWritten = expectation(description: "Promise write completes")
        var capturedError: Error?
        provider.filePromiseProvider(provider, writePromiseTo: destinationURL) { error in
            capturedError = error
            promiseWritten.fulfill()
        }
        wait(for: [promiseWritten], timeout: 1)

        guard let storageError = capturedError as? LibraryStorageError,
              case .ubiquitousLibraryPackageUnsupported = storageError else {
            return XCTFail("Unexpected error: \(String(describing: capturedError))")
        }
        XCTAssertFalse(FileManager.default.fileExists(atPath: destinationURL.path))
    }

    func testAssetFilePromiseProviderReportsSourceValidatorFailureToApp() throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(
            "momento-drag-promise-report-tests-\(UUID().uuidString)",
            isDirectory: true
        )
        defer { try? FileManager.default.removeItem(at: directory) }
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)

        let sourceURL = directory.appendingPathComponent("source.png")
        try Data("asset".utf8).write(to: sourceURL)
        let destinationURL = directory.appendingPathComponent("destination.png")
        let sourceAccessRejected = expectation(description: "Source access error is reported")
        var reportedError: Error?
        let provider = try XCTUnwrap(AssetFilePromiseProvider(
            asset: makeAsset(storageURL: sourceURL),
            libraryID: "library",
            assetIDs: ["asset-a"],
            primaryAssetID: "asset-a",
            exportBatch: AssetDragExportBatch(expectedFileCount: 1),
            sourceAccessValidator: {
                throw LibraryStorageError.ubiquitousLibraryPackageUnsupported
            },
            onSourceAccessError: { error in
                reportedError = error
                sourceAccessRejected.fulfill()
            }
        ))

        let promiseWritten = expectation(description: "Promise write completes")
        provider.filePromiseProvider(provider, writePromiseTo: destinationURL) { _ in
            promiseWritten.fulfill()
        }
        wait(for: [sourceAccessRejected, promiseWritten], timeout: 1)

        guard let storageError = reportedError as? LibraryStorageError,
              case .ubiquitousLibraryPackageUnsupported = storageError else {
            return XCTFail("Unexpected error: \(String(describing: reportedError))")
        }
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

    private func makeAsset(storageURL: URL = URL(fileURLWithPath: "/tmp/momento-drag-pasteboard-writer-test.png")) -> AssetItem {
        AssetItem(
            id: "asset-a",
            libraryID: "library",
            displayName: "Asset A",
            originalURL: nil,
            storageURL: storageURL,
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
