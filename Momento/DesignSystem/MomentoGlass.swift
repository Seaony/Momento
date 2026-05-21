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
    static let floatingSidebarRadius: CGFloat = 22
    static let floatingSidebarTitlebarContentInset: CGFloat = 54
    static let sidebarTitlebarButtonSize: CGFloat = 28
    static let sidebarTitlebarButtonTopInset: CGFloat = -18
    static let sidebarTitlebarButtonTrailingInset: CGFloat = 14
    static let collapsedSidebarToggleLeadingInset: CGFloat = 92
    static let inspectorMinWidth: CGFloat = 260
    static let inspectorWidth: CGFloat = 308
    static let inspectorMaxWidth: CGFloat = 460
    static let contentMinWidth: CGFloat = 520
    static let toolbarHeight: CGFloat = 56
    static let rowRadius: CGFloat = 8
    static let panelRadius: CGFloat = 14
    static let subtleStroke = Color(nsColor: .separatorColor)
    static let primaryText = Color(nsColor: .labelColor)
    static let secondaryText = Color(nsColor: .secondaryLabelColor)
    static let tertiaryText = Color(nsColor: .tertiaryLabelColor)
}
