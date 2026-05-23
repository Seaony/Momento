// 中文注释：本控制器把当前素材 URL 暴露给系统 Quick Look 面板。
import AppKit
import Quartz

@MainActor
final class QuickLookPreviewController: NSObject, QLPreviewPanelDataSource, QLPreviewPanelDelegate {
    static let shared = QuickLookPreviewController()

    private var previewURL: URL?

    func show(url: URL) {
        previewURL = url

        guard let panel = QLPreviewPanel.shared() else {
            return
        }

        panel.dataSource = self
        panel.delegate = self
        panel.reloadData()
        panel.makeKeyAndOrderFront(nil)
    }

    func numberOfPreviewItems(in panel: QLPreviewPanel!) -> Int {
        previewURL == nil ? 0 : 1
    }

    func previewPanel(_ panel: QLPreviewPanel!, previewItemAt index: Int) -> QLPreviewItem! {
        previewURL as NSURL?
    }
}
