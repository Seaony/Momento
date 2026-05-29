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
    var browserExtensionAction: (() -> Void)?
    var browserExtensionLabel: String?

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
                importLabel: importLabel,
                browserExtensionAction: browserExtensionAction,
                browserExtensionLabel: browserExtensionLabel
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
        var browserExtensionAction: (() -> Void)?
        var browserExtensionLabel: String?
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
                browserExtensionAction: configuration.browserExtensionAction,
                browserExtensionLabel: configuration.browserExtensionLabel,
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
                browserExtensionAction: configuration.browserExtensionAction,
                browserExtensionLabel: configuration.browserExtensionLabel,
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
            let actionButtonCount = titlebarActionButtonCount(configuration: configuration)
            if actionButtonCount == 0 {
                return MomentoTheme.titlebarControlHitSize
            }

            return importButtonOffset(configuration: configuration)
                + MomentoTheme.sidebarTitlebarButtonHitInset
                - MomentoTheme.toolbarIconButtonHitInset
                + CGFloat(actionButtonCount) * MomentoTheme.titlebarControlHitSize
        }

        private func importButtonOffset(configuration: Configuration) -> CGFloat {
            guard titlebarActionButtonCount(configuration: configuration) > 0 else {
                return 0
            }

            let requestedMinX = configuration.importButtonMinX ?? configuration.buttonMinX + MomentoTheme.sidebarTitlebarButtonSize + 10
            return max(
                MomentoTheme.sidebarTitlebarButtonSize + 10,
                requestedMinX - configuration.buttonMinX
            )
        }

        private func titlebarActionButtonCount(configuration: Configuration) -> Int {
            (configuration.importAction == nil ? 0 : 1)
                + (configuration.browserExtensionAction == nil ? 0 : 1)
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
    var browserExtensionAction: (() -> Void)?
    var browserExtensionLabel: String?
    var importButtonOffset: CGFloat

    @State private var isToggleHovered = false
    @State private var hoveredAction: TitlebarAction?

    private enum TitlebarAction {
        case importAssets
        case browserExtension
    }

    var body: some View {
        ZStack(alignment: .leading) {
            sidebarToggleButton

            if let importAction, let importLabel {
                titlebarActionButton(
                    action: importAction,
                    label: importLabel,
                    systemImage: "square.and.arrow.down",
                    hoverID: .importAssets
                )
                    .offset(
                        x: importButtonOffset
                            + MomentoTheme.sidebarTitlebarButtonHitInset
                            - MomentoTheme.toolbarIconButtonHitInset
                    )
            }

            if let browserExtensionAction, let browserExtensionLabel {
                titlebarActionButton(
                    action: browserExtensionAction,
                    label: browserExtensionLabel,
                    systemImage: "backpack",
                    hoverID: .browserExtension
                )
                    .offset(
                        x: browserExtensionButtonOffset
                            + MomentoTheme.sidebarTitlebarButtonHitInset
                            - MomentoTheme.toolbarIconButtonHitInset
                    )
            }
        }
        .frame(width: titlebarControlsWidth, height: titlebarControlsHeight, alignment: .leading)
    }

    private var sidebarToggleButton: some View {
        Button {
            withAnimation(.smooth(duration: 0.18)) {
                isCollapsed.toggle()
            }
        } label: {
            ZStack {
                Image(systemName: "sidebar.left")
                    .font(.system(size: MomentoTheme.toolbarIconSize, weight: .medium))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .frame(width: MomentoTheme.sidebarTitlebarButtonSize, height: MomentoTheme.sidebarTitlebarButtonSize)
                    .background {
                        MomentoGlassBackground(
                            style: .regular.interactive(isToggleHovered),
                            cornerRadius: 9
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
                isToggleHovered = hovering
            }
        }
        .help(label)
        .accessibilityLabel(label)
    }

    private func titlebarActionButton(
        action: @escaping () -> Void,
        label: String,
        systemImage: String,
        hoverID: TitlebarAction
    ) -> some View {
        let isHovered = hoveredAction == hoverID
        let shape = RoundedRectangle(cornerRadius: MomentoTheme.toolbarControlRadius, style: .continuous)

        return Button(action: action) {
            ZStack {
                Image(systemName: systemImage)
                    .font(.system(size: MomentoTheme.toolbarIconSize, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .frame(width: MomentoTheme.toolbarIconButtonWidth, height: MomentoTheme.toolbarControlHeight)
                    .background {
                        MomentoGlassBackground(
                            style: .regular.interactive(isHovered),
                            cornerRadius: MomentoTheme.toolbarControlRadius
                        )
                    }
                    .overlay {
                        if isHovered {
                            shape.fill(MomentoTheme.sidebarIconHoverBackground)
                        }
                    }
                    .contentShape(shape)
                    .offset(y: MomentoTheme.toolbarControlHitInset)
            }
            .frame(width: MomentoTheme.titlebarControlHitSize, height: MomentoTheme.titlebarControlHitSize, alignment: .top)
            .contentShape(.interaction, Rectangle())
        }
        .buttonStyle(.plain)
        .frame(width: MomentoTheme.titlebarControlHitSize, height: MomentoTheme.titlebarControlHitSize)
        .contentShape(.interaction, Rectangle())
        .pointerStyle(.link)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                if hovering {
                    hoveredAction = hoverID
                } else if hoveredAction == hoverID {
                    hoveredAction = nil
                }
            }
        }
        .help(label)
        .accessibilityLabel(label)
    }

    private var titlebarControlsWidth: CGFloat {
        if titlebarActionButtonCount == 0 {
            return MomentoTheme.titlebarControlHitSize
        }

        return importButtonOffset
            + MomentoTheme.sidebarTitlebarButtonHitInset
            - MomentoTheme.toolbarIconButtonHitInset
            + CGFloat(titlebarActionButtonCount) * MomentoTheme.titlebarControlHitSize
    }

    private var titlebarControlsHeight: CGFloat {
        MomentoTheme.titlebarControlHitSize
    }

    var titlebarActionButtonCount: Int {
        (importAction == nil ? 0 : 1)
            + (browserExtensionAction == nil ? 0 : 1)
    }

    private var browserExtensionButtonOffset: CGFloat {
        importButtonOffset + (importAction == nil ? 0 : MomentoTheme.titlebarControlHitSize)
    }
}

private final class SidebarTitlebarToggleContainerView: NSView {
    private let hostingView: SidebarTitlebarToggleHostingView
    private var buttonMinX: CGFloat = 0
    private var controlsWidth: CGFloat = MomentoTheme.titlebarControlHitSize
    private var controlsHeight: CGFloat = MomentoTheme.titlebarControlHitSize
    private var actionButtonOffset: CGFloat?
    private var actionButtonCount = 0

    override var isFlipped: Bool {
        true
    }

    init(hostingView: SidebarTitlebarToggleHostingView) {
        self.hostingView = hostingView
        super.init(frame: .zero)
        clipsToBounds = false
        hostingView.clipsToBounds = false
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
        self.controlsHeight = MomentoTheme.titlebarControlHitSize
        self.actionButtonOffset = rootView.titlebarActionButtonCount == 0 ? nil : rootView.importButtonOffset
        self.actionButtonCount = rootView.titlebarActionButtonCount
        needsLayout = true
    }

    override func layout() {
        super.layout()

        // AppKit 会把 titlebar accessory 放进自己的容器；容器原点不一定等于窗口原点。
        // 先把当前容器原点转换到窗口坐标，再用外部传入的 buttonMinX 抵消偏移，
        // 才能保证按钮展开/收起时始终和左侧玻璃侧边栏的右上角对齐。
        let titlebarOriginX = convert(.zero, to: nil).x
        let buttonX = max(0, buttonMinX - titlebarOriginX - MomentoTheme.sidebarTitlebarButtonHitInset)
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
            y: hostingView.frame.minY,
            width: MomentoTheme.titlebarControlHitSize,
            height: MomentoTheme.titlebarControlHitSize
        )
        let actionFrames = actionButtonOffset.map { offset in
            (0..<actionButtonCount).map { index in
                NSRect(
                    x: hostingView.frame.minX
                        + offset
                        + CGFloat(index) * MomentoTheme.titlebarControlHitSize
                        + MomentoTheme.sidebarTitlebarButtonHitInset
                        - MomentoTheme.toolbarIconButtonHitInset,
                    y: hostingView.frame.minY,
                    width: MomentoTheme.titlebarControlHitSize,
                    height: MomentoTheme.titlebarControlHitSize
                )
            }
        } ?? []

        guard toggleFrame.contains(point) || actionFrames.contains(where: { $0.contains(point) }) else {
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
