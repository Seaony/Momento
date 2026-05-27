// 中文注释：这些粗粒度 benchmark 为资源列表派生数据优化提供稳定的本地对比入口。
import Foundation
import XCTest
@testable import Momento

@MainActor
final class LibraryStorePerformanceTests: XCTestCase {
    func testVisibleAssetsFilteringSearchingAndSortingPerformance() {
        let library = AssetLibrary(id: "library", name: "Library", createdAt: Date(timeIntervalSince1970: 0))
        let store = LibraryStore(
            libraries: [library],
            assets: Self.makeAssets(count: 10_000, libraryID: library.id),
            loadRecentLibrary: false
        )
        store.filterState.fileExtensions = ["png"]
        store.searchQuery = "asset"
        store.sortOption = .name
        store.sortDirection = .ascending

        XCTAssertEqual(store.visibleAssets.count, 5_000)

        measure(metrics: [XCTClockMetric(), XCTCPUMetric(), XCTMemoryMetric()]) {
            _ = store.visibleAssets.count
        }
    }

    private static func makeAssets(count: Int, libraryID: AssetLibrary.ID) -> [AssetItem] {
        let baseDate = Date(timeIntervalSince1970: 0)

        return (0..<count).map { index in
            let isPNG = index.isMultiple(of: 2)
            let fileExtension = isPNG ? "png" : "jpg"
            let tag = TagItem(id: "tag-\(index % 12)", name: "Tag \(index % 12)")

            return AssetItem(
                id: "asset-\(index)",
                libraryID: libraryID,
                displayName: "Asset \(String(format: "%05d", index))",
                originalURL: nil,
                storageURL: URL(fileURLWithPath: "/Samples/asset-\(index).\(fileExtension)"),
                kind: .image,
                fileExtension: fileExtension,
                byteSize: Int64(index + 1),
                contentHash: "hash-\(index)",
                dimensions: AssetDimensions(width: 100 + index % 10, height: 120 + index % 10),
                tags: [tag],
                paletteColors: [
                    AssetColor(
                        id: "asset-\(index)-color",
                        libraryID: libraryID,
                        assetID: "asset-\(index)",
                        hex: isPNG ? "#ff0000" : "#0000ff",
                        coverage: 0.5,
                        sortIndex: 0
                    )
                ],
                isFavorite: index.isMultiple(of: 5),
                importedAt: baseDate.addingTimeInterval(TimeInterval(index))
            )
        }
    }
}
