// 中文注释：本文件集中定义 Momento 的 Liquid Glass 背景、按钮样式和视觉 token。
import SwiftUI

struct MomentoGlassBackground: View {
    var glass: Glass
    var cornerRadius: CGFloat

    init(
        glass: Glass = .regular,
        cornerRadius: CGFloat = 16
    ) {
        self.glass = glass
        self.cornerRadius = cornerRadius
    }

    var body: some View {
        Color.clear
            .glassEffect(glass, in: RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
    }
}

struct MomentoGlassPanelModifier: ViewModifier {
    var glass: Glass
    var cornerRadius: CGFloat
    var padding: CGFloat

    func body(content: Content) -> some View {
        content
            .padding(padding)
            .background {
                MomentoGlassBackground(glass: glass, cornerRadius: cornerRadius)
            }
    }
}

extension View {
    func momentoGlassPanel(
        glass: Glass = .regular,
        cornerRadius: CGFloat = 16,
        padding: CGFloat = 12
    ) -> some View {
        modifier(
            MomentoGlassPanelModifier(
                glass: glass,
                cornerRadius: cornerRadius,
                padding: padding
            )
        )
    }
}

enum MomentoTheme {
    static let sidebarMinWidth: CGFloat = 196
    static let sidebarWidth: CGFloat = 280
    static let sidebarMaxWidth: CGFloat = 340
    static let floatingSidebarInset: CGFloat = 8
    static let floatingSidebarRadius: CGFloat = 20
    static let floatingSidebarTitlebarContentInset: CGFloat = 54
    static let inspectorContentTopInset = floatingSidebarTitlebarContentInset
    static let librarySelectorHeight: CGFloat = 26
    static let librarySwitcherVerticalGap: CGFloat = 8
    static let sidebarTitlebarButtonSize: CGFloat = 26
    static let sidebarTitlebarButtonTopInset: CGFloat = 14
    static let sidebarTitlebarButtonTrailingInset: CGFloat = 14
    static let collapsedSidebarToggleLeadingInset: CGFloat = 92
    static let sidebarIconHoverBackground = Color.white.opacity(0.08)
    static let toolbarIconButtonWidth: CGFloat = 38
    static let toolbarControlHeight: CGFloat = 34
    static let toolbarControlRadius: CGFloat = 14
    static let toolbarIconSize: CGFloat = 15
    static let librarySwitcherWidth: CGFloat = 300
    static let inspectorMinWidth: CGFloat = 260
    static let inspectorWidth: CGFloat = inspectorMinWidth
    static let inspectorMaxWidth: CGFloat = 460
    static let contentMinWidth: CGFloat = 520
    static let contentSidebarGap: CGFloat = 12
    static let toolbarHeight: CGFloat = 56
    static let mainWindowMinWidth: CGFloat = 1100
    static let mainWindowMinHeight: CGFloat = 640
    static let defaultWindowWidth: CGFloat = 1100
    static let defaultWindowHeight: CGFloat = 800
    static let assetImageCornerRadius: CGFloat = 12
    static let panelRadius: CGFloat = 14
    static let subtleStroke = Color(nsColor: .separatorColor)
    static let primaryText = Color(nsColor: .labelColor)
    static let secondaryText = Color(nsColor: .secondaryLabelColor)
    static let tertiaryText = Color(nsColor: .tertiaryLabelColor)
}
