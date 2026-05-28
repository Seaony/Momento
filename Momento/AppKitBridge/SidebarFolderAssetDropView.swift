// 中文注释：文件夹行的素材 drop 用 AppKit 原生 drag destination 接收，
// 规避 SwiftUI .onDrop 收不到 NSFilePromiseProvider 自定义类型的问题。
import AppKit
import SwiftUI

struct SidebarFolderAssetDropView: NSViewRepresentable {
    var currentLibraryID: AssetLibrary.ID?
    var onTargetedChange: (Bool) -> Void
    var onDropAssetIDs: (Set<AssetItem.ID>) -> Void

    func makeNSView(context: Context) -> AssetFolderDropTargetView {
        let view = AssetFolderDropTargetView()
        view.registerForDraggedTypes([AssetDragPasteboardWriter.assetIDsPasteboardType])
        view.handlers = context.coordinator
        return view
    }

    func updateNSView(_ nsView: AssetFolderDropTargetView, context: Context) {
        context.coordinator.parent = self
        nsView.handlers = context.coordinator
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    @MainActor
    final class Coordinator: AssetFolderDropTargetHandlers {
        var parent: SidebarFolderAssetDropView

        init(_ parent: SidebarFolderAssetDropView) {
            self.parent = parent
        }

        func canAcceptAssetDrop() -> Bool {
            parent.currentLibraryID != nil
        }

        func assetDropTargetedChanged(_ isTargeted: Bool) {
            parent.onTargetedChange(isTargeted)
        }

        func performAssetDrop(from pasteboard: NSPasteboard) -> Bool {
            guard let data = pasteboard.data(forType: AssetDragPasteboardWriter.assetIDsPasteboardType),
                  let payload = try? JSONDecoder.momento.decode(AssetDragPasteboardPayload.self, from: data),
                  payload.libraryID == parent.currentLibraryID else {
                return false
            }

            parent.onDropAssetIDs(Set(payload.assetIDs))
            return true
        }
    }
}

@MainActor
protocol AssetFolderDropTargetHandlers: AnyObject {
    func canAcceptAssetDrop() -> Bool
    func assetDropTargetedChanged(_ isTargeted: Bool)
    func performAssetDrop(from pasteboard: NSPasteboard) -> Bool
}

@MainActor
final class AssetFolderDropTargetView: NSView {
    weak var handlers: AssetFolderDropTargetHandlers?

    override func hitTest(_ point: NSPoint) -> NSView? {
        // Let SwiftUI keep handling normal row clicks while this view remains a drag destination.
        nil
    }

    override func draggingEntered(_ sender: NSDraggingInfo) -> NSDragOperation {
        guard handlers?.canAcceptAssetDrop() == true,
              sender.draggingPasteboard.availableType(from: [AssetDragPasteboardWriter.assetIDsPasteboardType]) != nil else {
            return []
        }

        handlers?.assetDropTargetedChanged(true)
        return .copy
    }

    override func draggingUpdated(_ sender: NSDraggingInfo) -> NSDragOperation {
        guard handlers?.canAcceptAssetDrop() == true,
              sender.draggingPasteboard.availableType(from: [AssetDragPasteboardWriter.assetIDsPasteboardType]) != nil else {
            return []
        }

        return .copy
    }

    override func draggingExited(_ sender: NSDraggingInfo?) {
        handlers?.assetDropTargetedChanged(false)
    }

    override func draggingEnded(_ sender: NSDraggingInfo) {
        handlers?.assetDropTargetedChanged(false)
    }

    override func performDragOperation(_ sender: NSDraggingInfo) -> Bool {
        let accepted = handlers?.performAssetDrop(from: sender.draggingPasteboard) ?? false
        handlers?.assetDropTargetedChanged(false)
        return accepted
    }
}
