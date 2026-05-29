import AppKit
import SwiftUI

enum MomentoSurfaceStyle {
    case regular
    case regularInteractive
    case tinted(Color, interactive: Bool)

    func tint(_ color: Color) -> MomentoSurfaceStyle {
        switch self {
        case .regular, .regularInteractive:
            .tinted(color, interactive: isInteractive)
        case .tinted(_, let interactive):
            .tinted(color, interactive: interactive)
        }
    }

    func interactive(_ enabled: Bool = true) -> MomentoSurfaceStyle {
        switch self {
        case .regular, .regularInteractive:
            enabled ? .regularInteractive : .regular
        case .tinted(let color, _):
            .tinted(color, interactive: enabled)
        }
    }

    private var isInteractive: Bool {
        switch self {
        case .regular:
            false
        case .regularInteractive:
            true
        case .tinted(_, let interactive):
            interactive
        }
    }

    var fallbackTint: Color? {
        switch self {
        case .regular, .regularInteractive:
            nil
        case .tinted(let color, _):
            color
        }
    }
}

enum MomentoButtonProminence {
    case regular
    case prominent
}

struct MomentoGlassEffectContainer<Content: View>: View {
    let spacing: CGFloat
    let content: Content

    init(
        spacing: CGFloat,
        @ViewBuilder content: () -> Content
    ) {
        self.spacing = spacing
        self.content = content()
    }

    var body: some View {
        if #available(macOS 26.0, *) {
            GlassEffectContainer(spacing: spacing) {
                content
            }
        } else {
            content
        }
    }
}

extension View {
    @ViewBuilder
    func momentoSurface<S: Shape>(
        _ style: MomentoSurfaceStyle = .regular,
        in shape: S
    ) -> some View {
        if #available(macOS 26.0, *) {
            glassEffect(style.momentoGlass, in: shape)
        } else {
            background {
                MomentoFallbackSurface(style: style, shape: shape)
            }
        }
    }

    @ViewBuilder
    func momentoGlassButtonStyle(_ prominence: MomentoButtonProminence = .regular) -> some View {
        if #available(macOS 26.0, *) {
            switch prominence {
            case .regular:
                buttonStyle(.glass)
            case .prominent:
                buttonStyle(.glassProminent)
            }
        } else {
            switch prominence {
            case .regular:
                buttonStyle(.bordered)
            case .prominent:
                buttonStyle(.borderedProminent)
            }
        }
    }
}

struct MomentoFallbackSurface<S: Shape>: View {
    var style: MomentoSurfaceStyle
    var shape: S

    var body: some View {
        shape
            .fill(.regularMaterial)
            .overlay {
                if let tint = style.fallbackTint {
                    shape.fill(tint)
                }
            }
    }
}

enum MomentoAppKitMaterialViewFactory {
    static func updateGlassTint(of view: NSView, tintColor: NSColor) {
        if #available(macOS 26.0, *), let glassView = view as? NSGlassEffectView {
            glassView.tintColor = tintColor
        }
    }

    static func makeContextMenuBackgroundView(cornerRadius: CGFloat) -> NSView {
        if #available(macOS 26.0, *) {
            let view = NSGlassEffectView()
            view.style = .regular
            view.cornerRadius = cornerRadius
            return view
        }

        return makeVisualEffectView(material: .popover, cornerRadius: cornerRadius)
    }

    static func makeSelectionBackgroundView(cornerRadius: CGFloat) -> NSView {
        if #available(macOS 26.0, *) {
            let view = NSGlassEffectView()
            view.style = .regular
            view.cornerRadius = cornerRadius
            return view
        }

        return makeVisualEffectView(material: .selection, cornerRadius: cornerRadius)
    }

    private static func makeVisualEffectView(
        material: NSVisualEffectView.Material,
        cornerRadius: CGFloat
    ) -> NSVisualEffectView {
        let view = NSVisualEffectView()
        view.material = material
        view.blendingMode = .withinWindow
        view.state = .active
        view.wantsLayer = true
        view.layer?.cornerRadius = cornerRadius
        view.layer?.cornerCurve = .continuous
        view.layer?.masksToBounds = true
        return view
    }
}

@available(macOS 26.0, *)
private extension MomentoSurfaceStyle {
    var momentoGlass: Glass {
        switch self {
        case .regular:
            .regular
        case .regularInteractive:
            .regular.interactive(true)
        case .tinted(let color, let interactive):
            .regular.tint(color).interactive(interactive)
        }
    }
}
