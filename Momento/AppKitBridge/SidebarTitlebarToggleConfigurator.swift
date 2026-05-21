import AppKit
import SwiftUI

struct SidebarTitlebarToggleConfigurator: NSViewRepresentable {
    @Binding var isCollapsed: Bool
    var buttonMinX: CGFloat
    var label: String

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> TitlebarToggleAnchorView {
        let view = TitlebarToggleAnchorView(frame: .zero)
        let coordinator = context.coordinator

        view.onWindowChange = { [weak coordinator] window in
            coordinator?.attach(to: window)
        }

        return view
    }

    func updateNSView(_ nsView: TitlebarToggleAnchorView, context: Context) {
        context.coordinator.update(
            configuration: Configuration(
                isCollapsed: $isCollapsed,
                buttonMinX: buttonMinX,
                label: label
            )
        )
        context.coordinator.attach(to: nsView.window)
    }

    static func dismantleNSView(_ nsView: TitlebarToggleAnchorView, coordinator: Coordinator) {
        nsView.onWindowChange = nil
        coordinator.remove()
    }

    struct Configuration {
        var isCollapsed: Binding<Bool>
        var buttonMinX: CGFloat
        var label: String
    }

    final class TitlebarToggleAnchorView: NSView {
        var onWindowChange: ((NSWindow?) -> Void)?

        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            onWindowChange?(window)
        }
    }
}

extension SidebarTitlebarToggleConfigurator {
    final class Coordinator {
        private weak var window: NSWindow?
        private var accessoryController: NSTitlebarAccessoryViewController?
        private var containerView: SidebarTitlebarToggleContainerView?
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

            let rootView = SidebarTitlebarToggleAccessoryView(
                isCollapsed: configuration.isCollapsed,
                label: configuration.label
            )
            let size = accessorySize(buttonMinX: configuration.buttonMinX)

            containerView?.update(rootView: rootView, buttonMinX: configuration.buttonMinX)
            containerView?.setFrameSize(size)
            accessoryController?.view.setFrameSize(size)
        }

        private func installAccessoryView(configuration: Configuration) {
            guard let window else {
                return
            }

            let rootView = SidebarTitlebarToggleAccessoryView(
                isCollapsed: configuration.isCollapsed,
                label: configuration.label
            )
            let hostingView = SidebarTitlebarToggleHostingView(rootView: rootView)
            let containerView = SidebarTitlebarToggleContainerView(hostingView: hostingView)
            containerView.update(rootView: rootView, buttonMinX: configuration.buttonMinX)
            containerView.frame = NSRect(origin: .zero, size: accessorySize(buttonMinX: configuration.buttonMinX))

            let accessoryController = NSTitlebarAccessoryViewController()
            accessoryController.layoutAttribute = .left
            accessoryController.view = containerView

            self.containerView = containerView
            self.accessoryController = accessoryController
            window.addTitlebarAccessoryViewController(accessoryController)
        }

        private func accessorySize(buttonMinX: CGFloat) -> NSSize {
            NSSize(
                width: buttonMinX + MomentoTheme.sidebarTitlebarButtonSize,
                height: MomentoTheme.floatingSidebarTitlebarContentInset
            )
        }
    }
}

private struct SidebarTitlebarToggleAccessoryView: View {
    @Binding var isCollapsed: Bool
    var label: String

    @State private var isHovered = false

    var body: some View {
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        Button {
            withAnimation(.smooth(duration: 0.18)) {
                isCollapsed.toggle()
            }
        } label: {
            Image(systemName: "sidebar.left")
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
        .frame(
            width: MomentoTheme.sidebarTitlebarButtonSize,
            height: MomentoTheme.sidebarTitlebarButtonSize
        )
    }
}

private final class SidebarTitlebarToggleContainerView: NSView {
    private let hostingView: SidebarTitlebarToggleHostingView
    private var buttonMinX: CGFloat = 0

    override var isFlipped: Bool {
        true
    }

    init(hostingView: SidebarTitlebarToggleHostingView) {
        self.hostingView = hostingView
        super.init(frame: .zero)
        addSubview(hostingView)
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func update(rootView: SidebarTitlebarToggleAccessoryView, buttonMinX: CGFloat) {
        hostingView.rootView = rootView
        self.buttonMinX = buttonMinX
        needsLayout = true
    }

    override func layout() {
        super.layout()

        let titlebarOriginX = convert(.zero, to: nil).x
        let buttonX = max(0, buttonMinX - titlebarOriginX)
        hostingView.frame = NSRect(
            x: buttonX,
            y: MomentoTheme.sidebarTitlebarButtonTopInset,
            width: MomentoTheme.sidebarTitlebarButtonSize,
            height: MomentoTheme.sidebarTitlebarButtonSize
        )
    }

    override func hitTest(_ point: NSPoint) -> NSView? {
        guard hostingView.frame.contains(point) else {
            return nil
        }

        return super.hitTest(point)
    }
}

private final class SidebarTitlebarToggleHostingView: NSHostingView<SidebarTitlebarToggleAccessoryView> {
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool {
        true
    }
}
