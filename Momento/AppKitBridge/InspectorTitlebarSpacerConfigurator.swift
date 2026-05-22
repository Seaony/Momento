import AppKit
import SwiftUI

struct InspectorTitlebarSpacerConfigurator: NSViewRepresentable {
    @Binding var isInspectorPresented: Bool
    var isVisible: Bool
    var inspectorWidth: CGFloat
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
                inspectorWidth: inspectorWidth,
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
        var inspectorWidth: CGFloat
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

            accessoryController?.isHidden = !configuration.isVisible

            guard configuration.isVisible else {
                return
            }

            let rootView = InspectorTitlebarControlAccessoryView(
                isInspectorPresented: configuration.isInspectorPresented,
                label: configuration.label
            )
            let size = accessorySize(configuration: configuration)

            containerView?.update(
                rootView: rootView,
                inspectorWidth: configuration.isInspectorPresented.wrappedValue ? configuration.inspectorWidth : 0
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
                rootView: rootView,
                inspectorWidth: configuration.isInspectorPresented.wrappedValue ? configuration.inspectorWidth : 0
            )
            containerView.frame = NSRect(origin: .zero, size: accessorySize(configuration: configuration))

            let accessoryController = NSTitlebarAccessoryViewController()
            accessoryController.layoutAttribute = .right
            accessoryController.view = containerView

            self.containerView = containerView
            self.accessoryController = accessoryController
            window.addTitlebarAccessoryViewController(accessoryController)
        }

        private func accessorySize(configuration: Configuration) -> NSSize {
            let reservedInspectorWidth = configuration.isInspectorPresented.wrappedValue ? configuration.inspectorWidth : 0

            return NSSize(
                width: MomentoTheme.sidebarTitlebarButtonTrailingInset
                    + MomentoTheme.sidebarTitlebarButtonSize
                    + MomentoTheme.sidebarTitlebarButtonTrailingInset
                    + reservedInspectorWidth,
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
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        Button {
            withAnimation(.spring(response: 0.28, dampingFraction: 0.88)) {
                isInspectorPresented.toggle()
            }
        } label: {
            Image(systemName: "sidebar.right")
                .font(.system(size: 17, weight: .medium))
                .foregroundStyle(isHovered ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
                .frame(width: MomentoTheme.sidebarTitlebarButtonSize, height: MomentoTheme.sidebarTitlebarButtonSize)
                .background {
                    if isHovered {
                        shape.fill(MomentoTheme.sidebarIconHoverBackground)
                    } else {
                        Color.clear
                    }
                }
        }
        .buttonStyle(.plain)
        .frame(width: MomentoTheme.sidebarTitlebarButtonSize, height: MomentoTheme.sidebarTitlebarButtonSize)
        .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
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
    private var inspectorWidth: CGFloat = 0

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

    func update(rootView: InspectorTitlebarControlAccessoryView, inspectorWidth: CGFloat) {
        hostingView.rootView = rootView
        self.inspectorWidth = inspectorWidth
        needsLayout = true
    }

    override func layout() {
        super.layout()

        let buttonX = max(
            MomentoTheme.sidebarTitlebarButtonTrailingInset,
            bounds.width
                - inspectorWidth
                - MomentoTheme.sidebarTitlebarButtonTrailingInset
                - MomentoTheme.sidebarTitlebarButtonSize
        )
        let nextFrame = NSRect(
            x: buttonX,
            y: MomentoTheme.sidebarTitlebarButtonTopInset,
            width: MomentoTheme.sidebarTitlebarButtonSize,
            height: MomentoTheme.sidebarTitlebarButtonSize
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
