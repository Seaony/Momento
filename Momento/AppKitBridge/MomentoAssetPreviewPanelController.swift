// 中文注释：本控制器管理长按预览浮窗的 AppKit 生命周期和 SwiftUI 内容承载。
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
        sourceFrame: NSRect? = nil,
        showsNavigationControls: Bool = false,
        canNavigatePrevious: Bool = false,
        canNavigateNext: Bool = false,
        onNavigatePrevious: (() -> Void)? = nil,
        onNavigateNext: (() -> Void)? = nil
    ) {
        let panel = panel ?? makePanel()
        let targetFrame = previewFrame()
        let transitionFrame = sanitizedTransitionFrame(sourceFrame)
        let rootView = previewOverlay(
            asset: asset,
            previewURL: previewURL,
            localization: localization,
            closesOnSpaceKeyUp: closesOnSpaceKeyUp,
            usesWindowTransition: transitionFrame != nil,
            animatesPresentation: true,
            showsNavigationControls: showsNavigationControls,
            canNavigatePrevious: canNavigatePrevious,
            canNavigateNext: canNavigateNext,
            onNavigatePrevious: onNavigatePrevious,
            onNavigateNext: onNavigateNext
        )

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

    func update(
        asset: AssetItem,
        previewURL: URL,
        localization: AppLocalization,
        closesOnSpaceKeyUp: Bool,
        showsNavigationControls: Bool,
        canNavigatePrevious: Bool,
        canNavigateNext: Bool,
        onNavigatePrevious: (() -> Void)?,
        onNavigateNext: (() -> Void)?
    ) {
        guard let panel, let hostingController else {
            show(
                asset: asset,
                previewURL: previewURL,
                localization: localization,
                closesOnSpaceKeyUp: closesOnSpaceKeyUp,
                showsNavigationControls: showsNavigationControls,
                canNavigatePrevious: canNavigatePrevious,
                canNavigateNext: canNavigateNext,
                onNavigatePrevious: onNavigatePrevious,
                onNavigateNext: onNavigateNext
            )
            return
        }

        returnFrame = nil
        hostingController.rootView = AnyView(
            previewOverlay(
                asset: asset,
                previewURL: previewURL,
                localization: localization,
                closesOnSpaceKeyUp: closesOnSpaceKeyUp,
                usesWindowTransition: false,
                animatesPresentation: false,
                showsNavigationControls: showsNavigationControls,
                canNavigatePrevious: canNavigatePrevious,
                canNavigateNext: canNavigateNext,
                onNavigatePrevious: onNavigatePrevious,
                onNavigateNext: onNavigateNext
            )
        )
        panel.makeKeyAndOrderFront(nil)
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

    private func previewOverlay(
        asset: AssetItem,
        previewURL: URL,
        localization: AppLocalization,
        closesOnSpaceKeyUp: Bool,
        usesWindowTransition: Bool,
        animatesPresentation: Bool,
        showsNavigationControls: Bool,
        canNavigatePrevious: Bool,
        canNavigateNext: Bool,
        onNavigatePrevious: (() -> Void)?,
        onNavigateNext: (() -> Void)?
    ) -> some View {
        MomentoAssetPreviewOverlay(
            asset: asset,
            previewURL: previewURL,
            closesOnSpaceKeyUp: closesOnSpaceKeyUp,
            usesWindowTransition: usesWindowTransition,
            animatesPresentation: animatesPresentation,
            showsNavigationControls: showsNavigationControls,
            canNavigatePrevious: canNavigatePrevious,
            canNavigateNext: canNavigateNext,
            onNavigatePrevious: onNavigatePrevious,
            onNavigateNext: onNavigateNext,
            onDismiss: { [weak self] in
                self?.close()
            }
        )
        .environment(\.appLocalization, localization)
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
