import SwiftUI
import AppKit

struct MomentoVisualEffectView: NSViewRepresentable {
    var material: NSVisualEffectView.Material
    var blendingMode: NSVisualEffectView.BlendingMode
    var state: NSVisualEffectView.State
    var emphasized: Bool

    init(
        material: NSVisualEffectView.Material = .sidebar,
        blendingMode: NSVisualEffectView.BlendingMode = .behindWindow,
        state: NSVisualEffectView.State = .active,
        emphasized: Bool = false
    ) {
        self.material = material
        self.blendingMode = blendingMode
        self.state = state
        self.emphasized = emphasized
    }

    func makeNSView(context: Context) -> NSVisualEffectView {
        let view = NSVisualEffectView()
        view.isEmphasized = emphasized
        view.material = material
        view.blendingMode = blendingMode
        view.state = state
        return view
    }

    func updateNSView(_ view: NSVisualEffectView, context: Context) {
        view.isEmphasized = emphasized
        view.material = material
        view.blendingMode = blendingMode
        view.state = state
    }
}

struct MomentoGlassBackground: View {
    var material: NSVisualEffectView.Material
    var cornerRadius: CGFloat
    var strokeOpacity: Double

    init(
        material: NSVisualEffectView.Material = .hudWindow,
        cornerRadius: CGFloat = 16,
        strokeOpacity: Double = 0.16
    ) {
        self.material = material
        self.cornerRadius = cornerRadius
        self.strokeOpacity = strokeOpacity
    }

    var body: some View {
        MomentoVisualEffectView(material: material)
            .clipShape(RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))
            .overlay {
                RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                    .strokeBorder(.white.opacity(strokeOpacity), lineWidth: 1)
            }
            .shadow(color: .black.opacity(0.12), radius: 18, x: 0, y: 12)
    }
}

struct MomentoGlassPanelModifier: ViewModifier {
    var material: NSVisualEffectView.Material
    var cornerRadius: CGFloat
    var padding: CGFloat

    func body(content: Content) -> some View {
        content
            .padding(padding)
            .background {
                MomentoGlassBackground(material: material, cornerRadius: cornerRadius)
            }
    }
}

extension View {
    func momentoGlassPanel(
        material: NSVisualEffectView.Material = .hudWindow,
        cornerRadius: CGFloat = 16,
        padding: CGFloat = 12
    ) -> some View {
        modifier(
            MomentoGlassPanelModifier(
                material: material,
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
    static let inspectorMinWidth: CGFloat = 260
    static let inspectorWidth: CGFloat = 308
    static let inspectorMaxWidth: CGFloat = 460
    static let contentMinWidth: CGFloat = 520
    static let toolbarHeight: CGFloat = 56
    static let rowRadius: CGFloat = 8
    static let panelRadius: CGFloat = 14
    static let subtleStroke = Color(nsColor: .separatorColor).opacity(0.5)
    static let secondaryText = Color(nsColor: .secondaryLabelColor)
    static let tertiaryText = Color(nsColor: .tertiaryLabelColor)
}
