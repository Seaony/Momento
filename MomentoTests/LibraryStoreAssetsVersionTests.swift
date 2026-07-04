import AppKit
import Foundation
import XCTest
@testable import Momento

@MainActor
final class LibraryStoreAssetsVersionTests: XCTestCase {
    func testAssetsVersionAdvancesForPersistentAssetMutations() async throws {
        let environment = try AssetsVersionTestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let afterCreate = store.assetsVersion
        let source = try environment.writeOnePixelPNG(named: "versioned.png")
        try await store.importItems(from: [source])
        XCTAssertEqual(store.assetsVersion, afterCreate + 1)

        let asset = try XCTUnwrap(store.assets.first)

        let afterImport = store.assetsVersion
        try store.toggleFavorite(for: asset.id)
        XCTAssertEqual(store.assetsVersion, afterImport + 1)

        let afterFavorite = store.assetsVersion
        try store.renameAsset(id: asset.id, to: "Versioned Rename")
        XCTAssertEqual(store.assetsVersion, afterFavorite + 1)

        let afterRename = store.assetsVersion
        try store.updateNote("Versioned note", forAssetID: asset.id)
        XCTAssertEqual(store.assetsVersion, afterRename + 1)

        let afterNote = store.assetsVersion
        _ = try store.refreshThumbnail(for: asset.id)
        XCTAssertEqual(store.assetsVersion, afterNote + 1)

        try store.createFolder(name: "Versioned Folder")
        let folder = try XCTUnwrap(store.folders.first)

        let afterFolderCreate = store.assetsVersion
        try store.assignAssets(ids: [asset.id], to: folder.id)
        XCTAssertEqual(store.assetsVersion, afterFolderCreate + 1)

        let afterAssign = store.assetsVersion
        try store.deleteFolder(id: folder.id)
        XCTAssertEqual(store.assetsVersion, afterAssign + 1)

        let afterFolderDelete = store.assetsVersion
        try store.moveAssetToTrash(id: asset.id)
        XCTAssertEqual(store.assetsVersion, afterFolderDelete + 1)

        let afterMoveToTrash = store.assetsVersion
        try store.restoreAssets(ids: [asset.id])
        XCTAssertEqual(store.assetsVersion, afterMoveToTrash + 1)

        let afterRestore = store.assetsVersion
        try store.moveAssetToTrash(id: asset.id)
        XCTAssertEqual(store.assetsVersion, afterRestore + 1)

        let afterSecondMoveToTrash = store.assetsVersion
        try store.deleteAssetPermanently(id: asset.id)
        XCTAssertEqual(store.assetsVersion, afterSecondMoveToTrash + 1)
    }

    // 中文注释：验证「本次 visibleAssets 重算对应哪些 asset id 变更」的权威集合契约——
    // 单条编辑/缩略图刷新精确记录被改 id；结构性删除回退为 nil（下游据此全量深比较，绝不漏刷新）。
    func testChangedAssetIDsSnapshotTracksSingleEditsAndFallsBackForBulk() async throws {
        let environment = try AssetsVersionTestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)
        let source = try environment.writeOnePixelPNG(named: "tracked.png")
        try await store.importItems(from: [source])
        let asset = try XCTUnwrap(store.assets.first)

        // 先消费导入产生的变更，重置累积
        _ = store.visibleAssets

        // 单条收藏切换：应精确记录被改 id
        try store.toggleFavorite(for: asset.id)
        _ = store.visibleAssets
        XCTAssertEqual(store.changedAssetIDsForVisibleRevision, [asset.id])

        // 缩略图刷新（只改 thumbnailURL、不改 updatedAt）：仍需精确记录，供网格 reload 刷新缩略图
        _ = try store.refreshThumbnail(for: asset.id)
        _ = store.visibleAssets
        XCTAssertEqual(store.changedAssetIDsForVisibleRevision, [asset.id])

        // 移到废纸篓是软删除（只改 isTrashed 字段，仍走 mergeAssets），属单条精确变更
        try store.moveAssetToTrash(id: asset.id)
        _ = store.visibleAssets
        XCTAssertEqual(store.changedAssetIDsForVisibleRevision, [asset.id])

        // 永久删除直接从 assets 数组移除（bumpAssetsVersion 不带 changedIDs），无法精确追踪，应回退为 nil
        try store.deleteAssetPermanently(id: asset.id)
        _ = store.visibleAssets
        XCTAssertNil(store.changedAssetIDsForVisibleRevision)
    }

    // 中文注释：用两张内容不同的图片（不去重）验证变更集的「多 id 累积」与「revision 之间清空」，
    // 这是单资产测试（{id}∪{id}={id}）无法覆盖的：能捕获 pending 未清空、跨 id 残留等回归。
    func testChangedAssetIDsAccumulatesDistinctIDsAndResetsBetweenRevisions() async throws {
        let environment = try AssetsVersionTestEnvironment()
        defer { environment.cleanup() }

        let store = LibraryStore(
            defaultViewMode: .grid,
            recentStore: RecentLibraryStore(defaults: environment.defaults),
            loadRecentLibrary: false
        )
        try store.createLibrary(at: environment.packageURL)

        let sourceA = environment.inputURL.appendingPathComponent("a.png")
        let sourceB = environment.inputURL.appendingPathComponent("b.png")
        try Self.distinctPNGData(red: 40).write(to: sourceA)
        try Self.distinctPNGData(red: 200).write(to: sourceB)
        try await store.importItems(from: [sourceA, sourceB])
        XCTAssertEqual(store.assets.count, 2)

        let idA = try XCTUnwrap(store.assets.first).id
        let idB = try XCTUnwrap(store.assets.last).id
        XCTAssertNotEqual(idA, idB)
        _ = store.visibleAssets

        // 连续编辑不同资产：第二次变更集恰为 {idB}，证明第一次的 {idA} 已在 revision 之间清空、不残留
        try store.toggleFavorite(for: idA)
        _ = store.visibleAssets
        XCTAssertEqual(store.changedAssetIDsForVisibleRevision, [idA])

        try store.toggleFavorite(for: idB)
        _ = store.visibleAssets
        XCTAssertEqual(store.changedAssetIDsForVisibleRevision, [idB])

        // 一次操作改多个资产：变更集应为两者并集
        try store.moveAssetsToTrash(ids: [idA, idB])
        _ = store.visibleAssets
        XCTAssertEqual(store.changedAssetIDsForVisibleRevision, [idA, idB])
    }

    func testAssetsVersionAdvancesForInMemoryTagMutations() throws {
        let library = AssetLibrary(
            id: "library",
            name: "Library",
            createdAt: Date(timeIntervalSince1970: 0)
        )
        let asset = Self.makeAsset(
            id: "asset",
            libraryID: library.id,
            tags: [TagItem(id: "old", name: "Old")]
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [asset],
            loadRecentLibrary: false
        )

        store.selectAsset(id: asset.id)

        let afterSelect = store.assetsVersion
        try store.updateSelectedTags(["New"])
        XCTAssertEqual(store.assetsVersion, afterSelect + 1)

        let tagID = try XCTUnwrap(store.assets.first?.tags.first?.id)
        let afterUpdate = store.assetsVersion
        try store.renameTag(id: tagID, to: "Renamed")
        XCTAssertEqual(store.assetsVersion, afterUpdate + 1)

        let afterRename = store.assetsVersion
        try store.deleteTag(id: tagID)
        XCTAssertEqual(store.assetsVersion, afterRename + 1)
    }

    func testAssetsVersionAdvancesWhenClosingLibrary() {
        let library = AssetLibrary(
            id: "library",
            name: "Library",
            createdAt: Date(timeIntervalSince1970: 0)
        )
        let store = LibraryStore(
            libraries: [library],
            assets: [Self.makeAsset(id: "asset", libraryID: library.id)],
            loadRecentLibrary: false
        )

        let beforeClose = store.assetsVersion
        store.closeCurrentLibrary()
        XCTAssertEqual(store.assetsVersion, beforeClose + 1)
    }

    // 中文注释：生成 2x2、指定红色分量的有效 PNG；不同 red 值 → 不同像素 → 不同 contentHash，导入时不会被去重。
    private static func distinctPNGData(red: Int) throws -> Data {
        guard let rep = NSBitmapImageRep(
            bitmapDataPlanes: nil,
            pixelsWide: 2,
            pixelsHigh: 2,
            bitsPerSample: 8,
            samplesPerPixel: 4,
            hasAlpha: true,
            isPlanar: false,
            colorSpaceName: .deviceRGB,
            bytesPerRow: 0,
            bitsPerPixel: 0
        ) else {
            throw CocoaError(.fileWriteUnknown)
        }
        let color = NSColor(deviceRed: CGFloat(red) / 255.0, green: 0, blue: 0, alpha: 1)
        for x in 0..<2 {
            for y in 0..<2 {
                rep.setColor(color, atX: x, y: y)
            }
        }
        guard let data = rep.representation(using: .png, properties: [:]) else {
            throw CocoaError(.fileWriteUnknown)
        }
        return data
    }

    private static func makeAsset(
        id: AssetItem.ID,
        libraryID: AssetLibrary.ID,
        tags: [TagItem] = []
    ) -> AssetItem {
        AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: id,
            originalURL: nil,
            storageURL: URL(fileURLWithPath: "/Samples/\(id).png"),
            kind: .image,
            fileExtension: "png",
            byteSize: 1,
            contentHash: id,
            dimensions: AssetDimensions(width: 1, height: 1),
            tags: tags,
            isFavorite: false,
            importedAt: Date(timeIntervalSince1970: 0)
        )
    }
}

struct AssetsVersionTestEnvironment {
    let rootURL: URL
    let packageURL: URL
    let inputURL: URL
    let defaults: UserDefaults
    let defaultsSuiteName: String

    init() throws {
        rootURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        inputURL = rootURL.appendingPathComponent("input", isDirectory: true)
        packageURL = rootURL.appendingPathComponent("Test.momento", isDirectory: true)
        defaultsSuiteName = "MomentoAssetsVersionTests.\(UUID().uuidString)"
        defaults = UserDefaults(suiteName: defaultsSuiteName)!

        try FileManager.default.createDirectory(at: inputURL, withIntermediateDirectories: true)
    }

    func writeOnePixelPNG(named fileName: String) throws -> URL {
        let data = Data(base64Encoded: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")!
        let url = inputURL.appendingPathComponent(fileName)
        try data.write(to: url)
        return url
    }

    func cleanup() {
        try? FileManager.default.removeItem(at: rootURL)
        UserDefaults.standard.removePersistentDomain(forName: defaultsSuiteName)
    }
}
