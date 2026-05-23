// 中文注释：本桥接把左侧栏折叠和导入按钮放进 AppKit 标题栏坐标系，保证窗口 chrome 命中稳定。
import AppKit
import SwiftUI

struct SidebarTitlebarToggleConfigurator: NSViewRepresentable {
    @Binding var isCollapsed: Bool
    var isVisible: Bool
    var buttonMinX: CGFloat
    var importButtonMinX: CGFloat?
    var label: String
    var importAction: (() -> Void)?
    var importLabel: String?

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
                isVisible: isVisible,
                buttonMinX: buttonMinX,
                importButtonMinX: importButtonMinX,
                label: label,
                importAction: importAction,
                importLabel: importLabel
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
        var isVisible: Bool
        var buttonMinX: CGFloat
        var importButtonMinX: CGFloat?
        var label: String
        var importAction: (() -> Void)?
        var importLabel: String?
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

        // 左侧标题栏控制组必须和系统红黄绿窗口按钮处在同一个 titlebar 坐标系里。
        // 如果直接用 SwiftUI overlay 盖在内容区上，按钮会被内容布局、safe area、
        // 窗口缩放和命中测试共同影响，之前出现过 hover/click 不稳定的问题。
        // 这里用 NSTitlebarAccessoryViewController 让 AppKit 负责 titlebar 命中区域，
        // SwiftUI 只负责按钮内容和 hover 动画。
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

            let rootView = SidebarTitlebarToggleAccessoryView(
                isCollapsed: configuration.isCollapsed,
                label: configuration.label,
                importAction: configuration.importAction,
                importLabel: configuration.importLabel,
                importButtonOffset: importButtonOffset(configuration: configuration)
            )
            let controlsWidth = titlebarControlsWidth(configuration: configuration)
            let size = accessorySize(buttonMinX: configuration.buttonMinX, controlsWidth: controlsWidth)

            containerView?.update(
                rootView: rootView,
                buttonMinX: configuration.buttonMinX,
                controlsWidth: controlsWidth
            )
            containerView?.setFrameSize(size)
            accessoryController?.view.setFrameSize(size)
        }

        private func installAccessoryView(configuration: Configuration) {
            guard let window else {
                return
            }

            let rootView = SidebarTitlebarToggleAccessoryView(
                isCollapsed: configuration.isCollapsed,
                label: configuration.label,
                importAction: configuration.importAction,
                importLabel: configuration.importLabel,
                importButtonOffset: importButtonOffset(configuration: configuration)
            )
            let controlsWidth = titlebarControlsWidth(configuration: configuration)
            let hostingView = SidebarTitlebarToggleHostingView(rootView: rootView)
            let containerView = SidebarTitlebarToggleContainerView(hostingView: hostingView)
            containerView.update(
                rootView: rootView,
                buttonMinX: configuration.buttonMinX,
                controlsWidth: controlsWidth
            )
            containerView.frame = NSRect(
                origin: .zero,
                size: accessorySize(buttonMinX: configuration.buttonMinX, controlsWidth: controlsWidth)
            )

            let accessoryController = NSTitlebarAccessoryViewController()
            accessoryController.layoutAttribute = .left
            accessoryController.view = containerView

            self.containerView = containerView
            self.accessoryController = accessoryController
            window.addTitlebarAccessoryViewController(accessoryController)
        }

        private func titlebarControlsWidth(configuration: Configuration) -> CGFloat {
            if configuration.importAction == nil {
                return MomentoTheme.sidebarTitlebarButtonSize
            }

            return importButtonOffset(configuration: configuration) + MomentoTheme.toolbarIconButtonWidth
        }

        private func importButtonOffset(configuration: Configuration) -> CGFloat {
            guard configuration.importAction != nil else {
                return 0
            }

            let requestedMinX = configuration.importButtonMinX ?? configuration.buttonMinX + MomentoTheme.sidebarTitlebarButtonSize + 10
            return max(
                MomentoTheme.sidebarTitlebarButtonSize + 10,
                requestedMinX - configuration.buttonMinX
            )
        }

        private func accessorySize(buttonMinX: CGFloat, controlsWidth: CGFloat) -> NSSize {
            NSSize(
                width: buttonMinX + controlsWidth,
                height: MomentoTheme.floatingSidebarTitlebarContentInset
            )
        }
    }
}

private struct SidebarTitlebarToggleAccessoryView: View {
    @Binding var isCollapsed: Bool
    var label: String
    var importAction: (() -> Void)?
    var importLabel: String?
    var importButtonOffset: CGFloat

    @State private var isToggleHovered = false

    var body: some View {
        ZStack(alignment: .leading) {
            sidebarToggleButton

            if let importAction, let importLabel {
                importButton(action: importAction, label: importLabel)
                    .offset(x: importButtonOffset)
            }
        }
        .frame(width: titlebarControlsWidth, height: titlebarControlsHeight, alignment: .leading)
    }

    private var sidebarToggleButton: some View {
        let shape = RoundedRectangle(cornerRadius: 7, style: .continuous)

        return Button {
            withAnimation(.smooth(duration: 0.18)) {
                isCollapsed.toggle()
            }
        } label: {
            Image(systemName: "sidebar.left")
                .font(.system(size: MomentoTheme.toolbarIconSize, weight: .medium))
                .foregroundStyle(MomentoTheme.primaryText)
                .frame(width: MomentoTheme.sidebarTitlebarButtonSize, height: MomentoTheme.sidebarTitlebarButtonSize)
                .background {
                    if isToggleHovered {
                        shape.fill(MomentoTheme.sidebarIconHoverBackground)
                    } else {
                        Color.clear
                    }
                }
        }
        .buttonStyle(.plain)
        .frame(width: MomentoTheme.sidebarTitlebarButtonSize, height: MomentoTheme.sidebarTitlebarButtonSize)
        .contentShape(RoundedRectangle(cornerRadius: 7, style: .continuous))
        .pointerStyle(.link)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                isToggleHovered = hovering
            }
        }
        .help(label)
        .accessibilityLabel(label)
    }

    private func importButton(action: @escaping () -> Void, label: String) -> some View {
        return Button(action: action) {
            Image(systemName: "square.and.arrow.down")
                .font(.system(size: MomentoTheme.toolbarIconSize, weight: .semibold))
                .foregroundStyle(MomentoTheme.primaryText)
                .frame(width: MomentoTheme.toolbarIconButtonWidth, height: MomentoTheme.toolbarControlHeight)
                .background {
                    MomentoGlassBackground(
                        glass: .regular.interactive(true),
                        cornerRadius: MomentoTheme.toolbarControlRadius
                    )
                }
        }
        .buttonStyle(.plain)
        .frame(width: MomentoTheme.toolbarIconButtonWidth, height: MomentoTheme.toolbarControlHeight)
        .contentShape(RoundedRectangle(cornerRadius: MomentoTheme.toolbarControlRadius, style: .continuous))
        .pointerStyle(.link)
        .help(label)
        .accessibilityLabel(label)
    }

    private var titlebarControlsWidth: CGFloat {
        if importAction == nil {
            return MomentoTheme.sidebarTitlebarButtonSize
        }

        return importButtonOffset + MomentoTheme.toolbarIconButtonWidth
    }

    private var titlebarControlsHeight: CGFloat {
        if importAction == nil {
            return MomentoTheme.sidebarTitlebarButtonSize
        }

        return MomentoTheme.toolbarControlHeight
    }
}

private final class SidebarTitlebarToggleContainerView: NSView {
    private let hostingView: SidebarTitlebarToggleHostingView
    private var buttonMinX: CGFloat = 0
    private var controlsWidth: CGFloat = MomentoTheme.sidebarTitlebarButtonSize
    private var controlsHeight: CGFloat = MomentoTheme.sidebarTitlebarButtonSize
    private var importButtonOffset: CGFloat?

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

    func update(rootView: SidebarTitlebarToggleAccessoryView, buttonMinX: CGFloat, controlsWidth: CGFloat) {
        hostingView.rootView = rootView
        self.buttonMinX = buttonMinX
        self.controlsWidth = controlsWidth
        self.controlsHeight = rootView.importAction == nil ? MomentoTheme.sidebarTitlebarButtonSize : MomentoTheme.toolbarControlHeight
        self.importButtonOffset = rootView.importAction == nil ? nil : rootView.importButtonOffset
        needsLayout = true
    }

    override func layout() {
        super.layout()

        // AppKit 会把 titlebar accessory 放进自己的容器；容器原点不一定等于窗口原点。
        // 先把当前容器原点转换到窗口坐标，再用外部传入的 buttonMinX 抵消偏移，
        // 才能保证按钮展开/收起时始终和左侧玻璃侧边栏的右上角对齐。
        let titlebarOriginX = convert(.zero, to: nil).x
        let buttonX = max(0, buttonMinX - titlebarOriginX)
        let buttonY = (MomentoTheme.floatingSidebarTitlebarContentInset - controlsHeight) / 2
        let nextFrame = NSRect(
            x: buttonX,
            y: buttonY,
            width: controlsWidth,
            height: controlsHeight
        )

        if hostingView.frame != nextFrame {
            hostingView.frame = nextFrame
            window?.invalidateCursorRects(for: hostingView)
        }
    }

    override func hitTest(_ point: NSPoint) -> NSView? {
        let toggleFrame = NSRect(
            x: hostingView.frame.minX,
            y: hostingView.frame.minY + (controlsHeight - MomentoTheme.sidebarTitlebarButtonSize) / 2,
            width: MomentoTheme.sidebarTitlebarButtonSize,
            height: MomentoTheme.sidebarTitlebarButtonSize
        )
        let importFrame = importButtonOffset.map { offset in
            NSRect(
                x: hostingView.frame.minX + offset,
                y: hostingView.frame.minY,
                width: MomentoTheme.toolbarIconButtonWidth,
                height: MomentoTheme.toolbarControlHeight
            )
        }

        guard toggleFrame.contains(point) || importFrame?.contains(point) == true else {
            return nil
        }

        return super.hitTest(point)
    }
}

private final class SidebarTitlebarToggleHostingView: NSHostingView<SidebarTitlebarToggleAccessoryView> {
    override func acceptsFirstMouse(for event: NSEvent?) -> Bool {
        true
    }

    override func resetCursorRects() {
        super.resetCursorRects()
        addCursorRect(bounds, cursor: .pointingHand)
    }
}
