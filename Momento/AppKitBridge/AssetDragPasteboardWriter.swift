// 中文注释：本文件定义应用内拖拽使用的 pasteboard payload，Finder 导出另走 file promise。
import AppKit
import Foundation
import UniformTypeIdentifiers

nonisolated struct AssetDragPasteboardPayload: Codable, Hashable, Sendable {
    var libraryID: AssetLibrary.ID
    var assetIDs: [AssetItem.ID]
    var primaryAssetID: AssetItem.ID
}

nonisolated enum AssetDragPasteboardWriter {
    static let assetIDsTypeIdentifier = "com.seaony.momento.asset-ids"
    static let assetIDsPasteboardType = NSPasteboard.PasteboardType(assetIDsTypeIdentifier)
    static let assetIDsUTType = UTType(exportedAs: assetIDsTypeIdentifier)

    static func encodedPayload(
        libraryID: AssetLibrary.ID,
        assetIDs: [AssetItem.ID],
        primaryAssetID: AssetItem.ID
    ) -> Data? {
        let payload = AssetDragPasteboardPayload(
            libraryID: libraryID,
            assetIDs: assetIDs,
            primaryAssetID: primaryAssetID
        )
        return try? JSONEncoder.momento.encode(payload)
    }
}
