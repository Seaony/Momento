import AppKit
import XCTest
@testable import Momento

final class AssetPreviewImageProviderTests: XCTestCase {
    func testCancelledVisibleThumbnailLoadDoesNotCacheFallbackImage() async {
        let decoderStarted = DispatchSemaphore(value: 0)
        let finishDecoder = DispatchSemaphore(value: 0)
        let provider = AssetPreviewImageProvider(
            thumbnailDecoder: { _ in
                decoderStarted.signal()
                _ = finishDecoder.wait(timeout: .now() + 2)
                return Self.makeImage()
            },
            fallbackImageProvider: { _ in
                Self.makeImage()
            }
        )
        let asset = Self.makeAsset()

        let task = Task {
            await provider.imageAsync(for: asset)
        }

        XCTAssertEqual(decoderStarted.wait(timeout: .now() + 1), .success)
        task.cancel()
        finishDecoder.signal()
        _ = await task.value

        XCTAssertNil(provider.cachedImage(for: asset))
    }

    func testCancelledPrefetchDoesNotCacheDecodedImage() async throws {
        let decoderStarted = DispatchSemaphore(value: 0)
        let finishDecoder = DispatchSemaphore(value: 0)
        let provider = AssetPreviewImageProvider(
            thumbnailDecoder: { _ in
                decoderStarted.signal()
                _ = finishDecoder.wait(timeout: .now() + 2)
                return Self.makeImage()
            },
            fallbackImageProvider: { _ in
                Self.makeImage()
            }
        )
        let asset = Self.makeAsset()

        provider.prefetchImage(for: asset)
        XCTAssertEqual(decoderStarted.wait(timeout: .now() + 1), .success)
        provider.cancelPrefetch(for: asset)
        finishDecoder.signal()
        try await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertNil(provider.cachedImage(for: asset))
    }

    private static func makeAsset() -> AssetItem {
        let url = URL(fileURLWithPath: "/tmp/momento-preview-test.png")
        return AssetItem(
            id: UUID().uuidString,
            libraryID: "library",
            displayName: "Preview Test",
            originalURL: nil,
            storageURL: url,
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: UUID().uuidString,
            dimensions: AssetDimensions(width: 16, height: 16),
            tags: [],
            thumbnailURL: url,
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
    }

    private static func makeImage() -> NSImage {
        NSImage(size: NSSize(width: 16, height: 16))
    }
}
