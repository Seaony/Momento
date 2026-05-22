import AppKit
import SwiftUI

struct InspectorTitlebarSpacerConfigurator: NSViewRepresentable {
    var isVisible: Bool
    var width: CGFloat

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
        context.coordinator.update(isVisible: isVisible, width: width)
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
}

extension InspectorTitlebarSpacerConfigurator {
    final class Coordinator {
        private weak var window: NSWindow?
        private var accessoryController: NSTitlebarAccessoryViewController?
        private var spacerView: InspectorTitlebarSpacerView?
        private var isVisible = false
        private var width: CGFloat = 0

        func update(isVisible: Bool, width: CGFloat) {
            self.isVisible = isVisible
            self.width = width
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
            spacerView = nil
            window = nil
        }

        private func updateAccessoryView() {
            guard width > 0 else {
                remove()
                return
            }

            if accessoryController == nil {
                installAccessoryView()
            }

            accessoryController?.isHidden = !isVisible

            guard isVisible else {
                return
            }

            let size = NSSize(width: width, height: MomentoTheme.floatingSidebarTitlebarContentInset)
            spacerView?.setFrameSize(size)
            accessoryController?.view.setFrameSize(size)
        }

        private func installAccessoryView() {
            guard let window else {
                return
            }

            let size = NSSize(width: width, height: MomentoTheme.floatingSidebarTitlebarContentInset)
            let spacerView = InspectorTitlebarSpacerView(frame: NSRect(origin: .zero, size: size))
            let accessoryController = NSTitlebarAccessoryViewController()
            accessoryController.layoutAttribute = .right
            accessoryController.view = spacerView

            self.spacerView = spacerView
            self.accessoryController = accessoryController
            window.addTitlebarAccessoryViewController(accessoryController)
        }
    }
}

private final class InspectorTitlebarSpacerView: NSView {
    override func hitTest(_ point: NSPoint) -> NSView? {
        nil
    }
}
