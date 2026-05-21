import AppKit
import SwiftUI

@MainActor
final class MomentoAssetPreviewPanelController: NSObject, NSWindowDelegate {
    static let shared = MomentoAssetPreviewPanelController()

    private var panel: MomentoAssetPreviewPanel?
    private var hostingController: NSHostingController<AnyView>?

    func show(
        asset: AssetItem,
        previewURL: URL,
        localization: AppLocalization,
        closesOnSpaceKeyUp: Bool
    ) {
        let panel = panel ?? makePanel()
        let rootView = MomentoAssetPreviewOverlay(
            asset: asset,
            previewURL: previewURL,
            closesOnSpaceKeyUp: closesOnSpaceKeyUp,
            onDismiss: { [weak self] in
                self?.close()
            }
        )
        .id("\(asset.id)-\(previewURL.path)-\(closesOnSpaceKeyUp)")
        .environment(\.appLocalization, localization)

        if let hostingController {
            hostingController.rootView = AnyView(rootView)
        } else {
            let hostingController = NSHostingController(rootView: AnyView(rootView))
            hostingController.view.wantsLayer = true
            hostingController.view.layer?.backgroundColor = NSColor.clear.cgColor
            panel.contentViewController = hostingController
            self.hostingController = hostingController
        }

        panel.setFrame(previewFrame(), display: true)
        panel.makeKeyAndOrderFront(nil)
        self.panel = panel
    }

    func close() {
        panel?.orderOut(nil)
        panel?.contentViewController = nil
        panel?.delegate = nil
        panel = nil
        hostingController = nil
    }

    func windowWillClose(_ notification: Notification) {
        close()
    }

    private func makePanel() -> MomentoAssetPreviewPanel {
        let panel = MomentoAssetPreviewPanel(
            contentRect: previewFrame(),
            styleMask: [.borderless],
            backing: .buffered,
            defer: false
        )
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = false
        panel.isReleasedWhenClosed = false
        panel.level = .floating
        panel.collectionBehavior = [.fullScreenAuxiliary, .moveToActiveSpace, .transient]
        panel.delegate = self
        return panel
    }

    private func previewFrame() -> NSRect {
        let screen = NSApp.keyWindow?.screen ?? NSScreen.main
        let visibleFrame = screen?.visibleFrame ?? NSRect(x: 0, y: 0, width: 1280, height: 800)
        let horizontalInset = min(36, visibleFrame.width * 0.02)
        let verticalInset = min(32, visibleFrame.height * 0.025)

        return visibleFrame.insetBy(dx: horizontalInset, dy: verticalInset)
    }
}

private final class MomentoAssetPreviewPanel: NSPanel {
    override var canBecomeKey: Bool {
        true
    }

    override var canBecomeMain: Bool {
        false
    }
}
