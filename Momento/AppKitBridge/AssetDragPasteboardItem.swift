import AppKit
import Foundation

final class AssetDragPasteboardItem: NSObject, NSPasteboardWriting {
    private let payloadData: Data
    private let filePromise: AssetFilePromiseProvider

    init?(asset: AssetItem, libraryID: AssetLibrary.ID, assetIDs: [AssetItem.ID], primaryAssetID: AssetItem.ID) {
        guard let payloadData = AssetDragPasteboardWriter.encodedPayload(
            libraryID: libraryID,
            assetIDs: assetIDs,
            primaryAssetID: primaryAssetID
        ) else {
            return nil
        }

        self.payloadData = payloadData
        self.filePromise = AssetFilePromiseProvider(asset: asset)
        super.init()
    }

    func writableTypes(for pasteboard: NSPasteboard) -> [NSPasteboard.PasteboardType] {
        [AssetDragPasteboardWriter.assetIDsPasteboardType] + filePromise.provider.writableTypes(for: pasteboard)
    }

    func pasteboardPropertyList(forType type: NSPasteboard.PasteboardType) -> Any? {
        if type == AssetDragPasteboardWriter.assetIDsPasteboardType {
            return payloadData
        }

        return filePromise.provider.pasteboardPropertyList(forType: type)
    }

    func writingOptions(
        forType type: NSPasteboard.PasteboardType,
        pasteboard: NSPasteboard
    ) -> NSPasteboard.WritingOptions {
        if type == AssetDragPasteboardWriter.assetIDsPasteboardType {
            return []
        }

        return filePromise.provider.writingOptions(forType: type, pasteboard: pasteboard)
    }
}
