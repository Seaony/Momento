// 中文注释：本桥接在系统标题栏里预留检查器按钮空间，避免 SwiftUI 工具栏被右侧栏挤压。
import AppKit
import SwiftUI

struct InspectorTitlebarSpacerConfigurator: NSViewRepresentable {
    @Binding var isInspectorPresented: Bool
    var isVisible: Bool
    var label: String

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> TitlebarSpacerAnchorView {
        let view = TitlebarSpacerAnchorView(frame: .zero)
        let coordinator = context.coordinator

        view.onWindowChange = { [weak coordinator] window in
            coordinator?.attach(to: window)
        }

        return view
    }

    func updateNSView(_ nsView: TitlebarSpacerAnchorView, context: Context) {
        context.coordinator.update(
            configuration: Configuration(
                isInspectorPresented: $isInspectorPresented,
                isVisible: isVisible,
                label: label
            )
        )
        context.coordinator.attach(to: nsView.window)
    }

    static func dismantleNSView(_ nsView: TitlebarSpacerAnchorView, coordinator: Coordinator) {
        nsView.onWindowChange = nil
        coordinator.remove()
    }

    final class TitlebarSpacerAnchorView: NSView {
        var onWindowChange: ((NSWindow?) -> Void)?

        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            onWindowChange?(window)
        }
    }

    struct Configuration {
        var isInspectorPresented: Binding<Bool>
        var isVisible: Bool
        var label: String
    }
}

extension InspectorTitlebarSpacerConfigurator {
    final class Coordinator {
        private weak var window: NSWindow?
        private var accessoryController: NSTitlebarAccessoryViewController?
        private var containerView: InspectorTitlebarControlContainerView?
        private var configuration: Configuration?

        func update(configuration: Configuration) {
            self.configuration = configuration
            updateAccessoryView()
        }

        func attach(to window: NSWindow?) {
            guard self.window !== window else {
                updateAccessoryView()
                return
            }

            remove()
            self.window = window
            updateAccessoryView()
        }

        func remove() {
            if
                let window,
                let accessoryController,
                let index = window.titlebarAccessoryViewControllers.firstIndex(where: { $0 === accessoryController })
            {
                window.removeTitlebarAccessoryViewController(at: index)
            }

            accessoryController = nil
            containerView = nil
            window = nil
        }

        private func updateAccessoryView() {
            guard let configuration else {
                return
            }

            if accessoryController == nil {
                installAccessoryView(configuration: configuration)
            }

            let isHidden = !configuration.isVisible
            accessoryController?.isHidden = isHidden
            accessoryController?.view.isHidden = isHidden
            containerView?.isHidden = isHidden

            guard configuration.isVisible else {
                return
            }

            let rootView = InspectorTitlebarControlAccessoryView(
                isInspectorPresented: configuration.isInspectorPresented,
                label: configuration.label
            )
            let size = accessorySize()

            containerView?.update(
                rootView: rootView
            )
            containerView?.setFrameSize(size)
            accessoryController?.view.setFrameSize(size)
        }

        private func installAccessoryView(configuration: Configuration) {
            guard let window else {
                return
            }

            let rootView = InspectorTitlebarControlAccessoryView(
                isInspectorPresented: configuration.isInspectorPresented,
                label: configuration.label
            )
            let hostingView = InspectorTitlebarControlHostingView(rootView: rootView)
            let containerView = InspectorTitlebarControlContainerView(hostingView: hostingView)
            containerView.update(
                rootView: rootView
            )
            containerView.frame = NSRect(origin: .zero, size: accessorySize())

            let accessoryController = NSTitlebarAccessoryViewController()
            accessoryController.layoutAttribute = .right
            if #available(macOS 26.1, *) {
                accessoryController.preferredScrollEdgeEffectStyle = .soft
            }
            accessoryController.view = containerView

            self.containerView = containerView
            self.accessoryController = accessoryController
            window.addTitlebarAccessoryViewController(accessoryController)
        }

        private func accessorySize() -> NSSize {
            return NSSize(
                width: MomentoTheme.sidebarTitlebarButtonTrailingInset
                    + MomentoTheme.toolbarIconButtonWidth
                    + MomentoTheme.sidebarTitlebarButtonTrailingInset,
                height: MomentoTheme.floatingSidebarTitlebarContentInset
            )
        }
    }
}

private struct InspectorTitlebarControlAccessoryView: View {
    @Binding var isInspectorPresented: Bool
    var label: String

    @State private var isHovered = false

    var body: some View {
        Button {
            withAnimation(.spring(response: 0.28, dampingFraction: 0.88)) {
                isInspectorPresented.toggle()
            }
        } label: {
            ZStack {
                Image(systemName: "sidebar.right")
                    .font(.system(size: 15, weight: .medium))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .frame(width: MomentoTheme.toolbarIconButtonWidth, height: MomentoTheme.toolbarControlHeight)
                    .background {
                        MomentoGlassBackground(
                            glass: .regular.interactive(isHovered),
                            cornerRadius: MomentoTheme.toolbarControlRadius
                        )
                    }
            }
            .frame(width: MomentoTheme.titlebarControlHitSize, height: MomentoTheme.titlebarControlHitSize)
            .contentShape(.interaction, Rectangle())
        }
        .buttonStyle(.plain)
        .frame(width: MomentoTheme.titlebarControlHitSize, height: MomentoTheme.titlebarControlHitSize)
        .contentShape(.interaction, Rectangle())
        .pointerStyle(.link)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                isHovered = hovering
            }
        }
        .help(label)
        .accessibilityLabel(label)
    }
}

private final class InspectorTitlebarControlContainerView: NSView {
    private let hostingView: InspectorTitlebarControlHostingView

    override var isFlipped: Bool {
        true
    }

    init(hostingView: InspectorTitlebarControlHostingView) {
        self.hostingView = hostingView
        super.init(frame: .zero)
        addSubview(hostingView)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func update(rootView: InspectorTitlebarControlAccessoryView) {
        hostingView.rootView = rootView
        needsLayout = true
    }

    override func layout() {
        super.layout()

        let buttonX = max(
            MomentoTheme.sidebarTitlebarButtonTrailingInset,
            bounds.width
                - MomentoTheme.sidebarTitlebarButtonTrailingInset
                - MomentoTheme.toolbarIconButtonWidth
        ) - MomentoTheme.toolbarIconButtonHitInset
        let buttonY = (
            MomentoTheme.floatingSidebarTitlebarContentInset
                - MomentoTheme.toolbarControlHeight
        ) / 2 - MomentoTheme.toolbarControlHitInset
        let nextFrame = NSRect(
            x: buttonX,
            y: buttonY,
            width: MomentoTheme.titlebarControlHitSize,
            height: MomentoTheme.titlebarControlHitSize
        )

        if hostingView.frame != nextFrame {
            hostingView.frame = nextFrame
            window?.invalidateCursorRects(for: hostingView)
        }
    }

    override func hitTest(_ point: NSPoint) -> NSView? {
        guard hostingView.frame.contains(point) else {
            return nil
        }

        return super.hitTest(point)
    }
}

private final class InspectorTitlebarControlHostingView: NSHostingView<InspectorTitlebarControlAccessoryView> {
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool {
        true
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }
}
