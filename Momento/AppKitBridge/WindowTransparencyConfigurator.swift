import AppKit
import SwiftUI

/// Makes the hosting window non-opaque so the app's native Liquid Glass
/// surfaces refract the desktop behind the window instead of a solid window
/// background.
///
/// Owned at the app shell level — individual feature views stay pure SwiftUI
/// and never touch the window.
struct WindowTransparencyConfigurator: NSViewRepresentable {
    /// Opacity of the window backing that sits behind every Liquid Glass
    /// surface. `0` lets the desktop fully bleed through, `1` is a solid
    /// opaque window. Lower it for more transparency, raise it for less.
    static let backingOpacity: CGFloat = 0.9

    func makeNSView(context: Context) -> NSView {
        let view = NSView()
        DispatchQueue.main.async { applyTransparency(to: view.window) }
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        guard let window = nsView.window, window.isOpaque else { return }
        applyTransparency(to: window)
    }

    private func applyTransparency(to window: NSWindow?) {
        guard let window else { return }
        window.isOpaque = false
        window.backgroundColor = NSColor.windowBackgroundColor
            .withAlphaComponent(Self.backingOpacity)
    }
}
