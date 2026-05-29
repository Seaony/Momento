// 中文注释：本文件集中定义 Momento 的 Liquid Glass 背景、按钮样式和视觉 token。
import AppKit
import SwiftUI

struct MomentoGlassBackground: View {
    var style: MomentoSurfaceStyle
    var cornerRadius: CGFloat

    init(
        style: MomentoSurfaceStyle = .regular,
        cornerRadius: CGFloat = 16
    ) {
        self.style = style
        self.cornerRadius = cornerRadius
    }

    var body: some View {
        let shape = RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
        Color.clear
            .momentoSurface(style, in: shape)
    }
}

struct MomentoGlassPanelModifier: ViewModifier {
    var style: MomentoSurfaceStyle
    var cornerRadius: CGFloat
    var padding: CGFloat

    func body(content: Content) -> some View {
        content
            .padding(padding)
            .background {
                MomentoGlassBackground(style: style, cornerRadius: cornerRadius)
            }
    }
}

struct MomentoTooltipBubble: View {
    var text: String

    var body: some View {
        Text(text)
            .font(.system(size: 13, weight: .medium))
            .foregroundStyle(.white)
            .lineLimit(1)
            .fixedSize()
            .padding(.horizontal, 10)
            .frame(height: 30)
            .background {
                MomentoGlassBackground(
                    style: .tinted(Color.black.opacity(0.34), interactive: false),
                    cornerRadius: 9
                )
                    .overlay {
                        RoundedRectangle(cornerRadius: 9, style: .continuous)
                            .strokeBorder(Color.white.opacity(0.16), lineWidth: 1)
                    }
            }
            .shadow(color: Color.black.opacity(0.28), radius: 12, y: 6)
            .allowsHitTesting(false)
    }
}

extension View {
    func momentoGlassPanel(
        style: MomentoSurfaceStyle = .regular,
        cornerRadius: CGFloat = 16,
        padding: CGFloat = 12
    ) -> some View {
        modifier(
            MomentoGlassPanelModifier(
                style: style,
                cornerRadius: cornerRadius,
                padding: padding
            )
        )
    }
}

enum MomentoTheme {
    static let sidebarMinWidth: CGFloat = 196
    static let sidebarWidth: CGFloat = 236
    static let sidebarMaxWidth: CGFloat = 340
    static let floatingSidebarInset: CGFloat = 8
    static let floatingSidebarRadius: CGFloat = 20
    static let floatingSidebarTitlebarContentInset: CGFloat = 54
    static let inspectorContentTopInset = floatingSidebarTitlebarContentInset
    static let librarySelectorHeight: CGFloat = 26
    static let librarySwitcherVerticalGap: CGFloat = 8
    static let sidebarTitlebarButtonSize: CGFloat = 26
    static let titlebarControlHitSize: CGFloat = 44
    static var sidebarTitlebarButtonHitInset: CGFloat {
        (titlebarControlHitSize - sidebarTitlebarButtonSize) / 2
    }
    static var toolbarIconButtonHitInset: CGFloat {
        (titlebarControlHitSize - toolbarIconButtonWidth) / 2
    }
    static var toolbarControlHitInset: CGFloat {
        (titlebarControlHitSize - toolbarControlHeight) / 2
    }
    static let sidebarTitlebarButtonTopInset: CGFloat = 14
    static let sidebarTitlebarButtonTrailingInset: CGFloat = 14
    static let collapsedSidebarToggleLeadingInset: CGFloat = 92
    static let sidebarIconHoverBackground = contrastTint(lightOpacity: 0.06, darkOpacity: 0.08)
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
    static let glassStroke = contrastTint(lightOpacity: 0.10, darkOpacity: 0.12)
    static let subtleGlassStroke = contrastTint(lightOpacity: 0.08, darkOpacity: 0.08)
    static let inspectorSectionSeparator = contrastTint(lightOpacity: 0.08, darkOpacity: 0.06)
    static let primaryText = Color(nsColor: .labelColor)
    static let secondaryText = Color(nsColor: .secondaryLabelColor)
    static let tertiaryText = Color(nsColor: .tertiaryLabelColor)

    static func surfaceGlassTint(darkOpacity: CGFloat) -> Color {
        adaptiveColor(
            light: NSColor.white.withAlphaComponent(max(darkOpacity, 0.18)),
            dark: NSColor.black.withAlphaComponent(darkOpacity)
        )
    }

    static func contrastTint(lightOpacity: CGFloat, darkOpacity: CGFloat) -> Color {
        adaptiveColor(
            light: NSColor.black.withAlphaComponent(lightOpacity),
            dark: NSColor.white.withAlphaComponent(darkOpacity)
        )
    }

    static func adaptiveColor(light: NSColor, dark: NSColor) -> Color {
        Color(nsColor: adaptiveNSColor(light: light, dark: dark))
    }

    static func adaptiveNSColor(light: NSColor, dark: NSColor) -> NSColor {
        NSColor(name: nil) { appearance in
            appearance.momentoUsesDarkAppearance ? dark : light
        }
    }
}

private extension NSAppearance {
    var momentoUsesDarkAppearance: Bool {
        let match = bestMatch(from: [
            .accessibilityHighContrastDarkAqua,
            .darkAqua,
            .accessibilityHighContrastAqua,
            .aqua
        ])
        return match == .accessibilityHighContrastDarkAqua || match == .darkAqua
    }
}
