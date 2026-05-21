import AppKit
import QuartzCore
import SwiftUI

@MainActor
final class MomentoAssetPreviewPanelController: NSObject, NSWindowDelegate {
    static let shared = MomentoAssetPreviewPanelController()

    private var panel: MomentoAssetPreviewPanel?
    private var hostingController: NSHostingController<AnyView>?
    private var returnFrame: NSRect?
    private var isClosing = false

    func show(
        asset: AssetItem,
        previewURL: URL,
        localization: AppLocalization,
        closesOnSpaceKeyUp: Bool,
        sourceFrame: NSRect? = nil
    ) {
        let panel = panel ?? makePanel()
        let targetFrame = previewFrame()
        let transitionFrame = sanitizedTransitionFrame(sourceFrame)
        let rootView = MomentoAssetPreviewOverlay(
            asset: asset,
            previewURL: previewURL,
            closesOnSpaceKeyUp: closesOnSpaceKeyUp,
            usesWindowTransition: transitionFrame != nil,
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

        isClosing = false
        returnFrame = transitionFrame
        panel.alphaValue = 1
        panel.setFrame(transitionFrame ?? targetFrame, display: false)
        panel.makeKeyAndOrderFront(nil)
        self.panel = panel

        guard transitionFrame != nil else {
            panel.setFrame(targetFrame, display: true)
            return
        }

        NSAnimationContext.runAnimationGroup { context in
            context.duration = 0.22
            context.timingFunction = CAMediaTimingFunction(name: .easeOut)
            panel.animator().setFrame(targetFrame, display: true)
        }
    }

    func close() {
        guard let panel, !isClosing else {
            return
        }

        isClosing = true

        guard let returnFrame else {
            finishClose(panel)
            return
        }

        NSAnimationContext.runAnimationGroup { context in
            context.duration = 0.18
            context.timingFunction = CAMediaTimingFunction(name: .easeInEaseOut)
            panel.animator().setFrame(returnFrame, display: true)
            panel.animator().alphaValue = 0
        } completionHandler: {
            Task { @MainActor in
                self.finishClose(panel)
            }
        }
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

    private func sanitizedTransitionFrame(_ sourceFrame: NSRect?) -> NSRect? {
        guard let sourceFrame,
              sourceFrame.width > 1,
              sourceFrame.height > 1 else {
            return nil
        }

        return sourceFrame
    }

    private func finishClose(_ panel: MomentoAssetPreviewPanel) {
        panel.orderOut(nil)
        panel.alphaValue = 1
        panel.contentViewController = nil
        panel.delegate = nil
        self.panel = nil
        hostingController = nil
        returnFrame = nil
        isClosing = false
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
