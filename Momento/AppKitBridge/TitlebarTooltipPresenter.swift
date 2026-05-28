// 中文注释：标题栏 accessory 的 SwiftUI overlay 会被 titlebar hosting 区域裁剪，因此 tooltip 通过 child panel 显示。
import AppKit
import SwiftUI

extension View {
    func momentoTitlebarTooltip(_ text: String, isPresented: Bool) -> some View {
        background {
            MomentoTitlebarTooltipAnchor(text: text, isPresented: isPresented)
                .frame(
                    width: MomentoTheme.titlebarControlHitSize,
                    height: MomentoTheme.titlebarControlHitSize
                )
        }
    }
}

private struct MomentoTitlebarTooltipAnchor: NSViewRepresentable {
    var text: String
    var isPresented: Bool

    func makeNSView(context: Context) -> MomentoTitlebarTooltipAnchorView {
        MomentoTitlebarTooltipAnchorView()
    }

    func updateNSView(_ nsView: MomentoTitlebarTooltipAnchorView, context: Context) {
        nsView.update(text: text, isPresented: isPresented)
    }

    static func dismantleNSView(_ nsView: MomentoTitlebarTooltipAnchorView, coordinator: ()) {
        nsView.hideTooltip()
    }
}

private final class MomentoTitlebarTooltipAnchorView: NSView {
    private var text = ""
    private var isPresented = false

    func update(text: String, isPresented: Bool) {
        self.text = text
        self.isPresented = isPresented
        updateTooltipVisibility()
    }

    func hideTooltip() {
        MomentoTitlebarTooltipPresenter.shared.hide(from: self)
    }

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()
        updateTooltipVisibility()
    }

    override func setFrameSize(_ newSize: NSSize) {
        super.setFrameSize(newSize)
        updateTooltipVisibility()
    }

    override func hitTest(_ point: NSPoint) -> NSView? {
        nil
    }

    private func updateTooltipVisibility() {
        guard isPresented, !text.isEmpty, window != nil else {
            hideTooltip()
            return
        }

        MomentoTitlebarTooltipPresenter.shared.show(text: text, from: self)
    }
}

private final class MomentoTitlebarTooltipPresenter {
    static let shared = MomentoTitlebarTooltipPresenter()

    private weak var sourceView: NSView?
    private weak var parentWindow: NSWindow?
    private var panel: NSPanel?
    private var hostingView: NSHostingView<MomentoTooltipBubble>?

    private init() {}

    func show(text: String, from sourceView: NSView) {
        guard let window = sourceView.window else {
            hide(from: sourceView)
            return
        }

        let panel = tooltipPanel()
        let hostingView = tooltipHostingView(text: text)
        let size = hostingView.fittingSize
        hostingView.frame = NSRect(origin: .zero, size: size)
        panel.contentView = hostingView
        panel.setContentSize(size)

        if panel.parent !== window {
            parentWindow?.removeChildWindow(panel)
            window.addChildWindow(panel, ordered: .above)
            parentWindow = window
        }

        self.sourceView = sourceView
        position(panel: panel, size: size, from: sourceView, in: window)

        guard !panel.isVisible else {
            panel.alphaValue = 1
            return
        }

        panel.alphaValue = 0
        panel.orderFront(nil)
        NSAnimationContext.runAnimationGroup { context in
            context.duration = 0.12
            panel.animator().alphaValue = 1
        }
    }

    func hide(from sourceView: NSView) {
        guard self.sourceView === sourceView else {
            return
        }

        hideCurrent()
    }

    private func hideCurrent() {
        if let panel {
            parentWindow?.removeChildWindow(panel)
            panel.orderOut(nil)
        }

        sourceView = nil
        parentWindow = nil
    }

    private func tooltipPanel() -> NSPanel {
        if let panel {
            return panel
        }

        let panel = NSPanel(
            contentRect: .zero,
            styleMask: [.borderless, .nonactivatingPanel],
            backing: .buffered,
            defer: false
        )
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = false
        panel.hidesOnDeactivate = true
        panel.ignoresMouseEvents = true
        panel.collectionBehavior = [.transient, .ignoresCycle]

        self.panel = panel
        return panel
    }

    private func tooltipHostingView(text: String) -> NSHostingView<MomentoTooltipBubble> {
        if let hostingView {
            hostingView.rootView = MomentoTooltipBubble(text: text)
            return hostingView
        }

        let hostingView = NSHostingView(rootView: MomentoTooltipBubble(text: text))
        hostingView.wantsLayer = true
        hostingView.layer?.masksToBounds = false
        self.hostingView = hostingView
        return hostingView
    }

    private func position(panel: NSPanel, size: NSSize, from sourceView: NSView, in window: NSWindow) {
        let anchorInWindow = sourceView.convert(sourceView.bounds, to: nil)
        let anchorOnScreen = window.convertToScreen(anchorInWindow)
        let visibleFrame = window.screen?.visibleFrame ?? NSScreen.main?.visibleFrame
        let preferredX = anchorOnScreen.midX - size.width / 2
        let minX = (visibleFrame?.minX ?? preferredX) + 8
        let maxX = (visibleFrame?.maxX ?? preferredX + size.width) - size.width - 8
        let originX = min(max(preferredX.rounded(), minX), maxX)
        let originY = (anchorOnScreen.minY - 7 - size.height).rounded()

        panel.setFrameOrigin(NSPoint(x: originX, y: originY))
    }
}
