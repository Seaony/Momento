import AppKit
import SwiftUI

struct SidebarTitlebarToggleConfigurator: NSViewRepresentable {
    @Binding var isCollapsed: Bool
    var leadingInset: CGFloat
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
                leadingInset: leadingInset,
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
        var leadingInset: CGFloat
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
        private var hostingView: SidebarTitlebarToggleHostingView?
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
            hostingView = nil
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
                leadingInset: configuration.leadingInset,
                label: configuration.label
            )
            let size = accessorySize(leadingInset: configuration.leadingInset)

            hostingView?.rootView = rootView
            hostingView?.setFrameSize(size)
            accessoryController?.view.setFrameSize(size)
        }

        private func installAccessoryView(configuration: Configuration) {
            guard let window else {
                return
            }

            let rootView = SidebarTitlebarToggleAccessoryView(
                isCollapsed: configuration.isCollapsed,
                leadingInset: configuration.leadingInset,
                label: configuration.label
            )
            let hostingView = SidebarTitlebarToggleHostingView(rootView: rootView)
            hostingView.frame = NSRect(origin: .zero, size: accessorySize(leadingInset: configuration.leadingInset))

            let accessoryController = NSTitlebarAccessoryViewController()
            accessoryController.layoutAttribute = .left
            accessoryController.view = hostingView

            self.hostingView = hostingView
            self.accessoryController = accessoryController
            window.addTitlebarAccessoryViewController(accessoryController)
        }

        private func accessorySize(leadingInset: CGFloat) -> NSSize {
            NSSize(
                width: leadingInset + MomentoTheme.sidebarTitlebarButtonSize,
                height: MomentoTheme.floatingSidebarTitlebarContentInset
            )
        }
    }
}

private struct SidebarTitlebarToggleAccessoryView: View {
    @Binding var isCollapsed: Bool
    var leadingInset: CGFloat
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
        .padding(.leading, leadingInset)
        .padding(.top, MomentoTheme.sidebarTitlebarButtonTopInset)
        .frame(
            width: leadingInset + MomentoTheme.sidebarTitlebarButtonSize,
            height: MomentoTheme.floatingSidebarTitlebarContentInset,
            alignment: .topLeading
        )
    }
}

private final class SidebarTitlebarToggleHostingView: NSHostingView<SidebarTitlebarToggleAccessoryView> {
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool {
        true
    }
}
