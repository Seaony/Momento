// 中文注释：本桥接只负责窗口级透明 backing，业务视图继续保持 SwiftUI 声明式结构。
import AppKit
import SwiftUI

/// 在窗口层统一开启透明 backing，让 SwiftUI 的原生 Liquid Glass
/// 能折射到同一层窗口背景，而不是让每个功能视图各自处理 AppKit 窗口状态。
///
/// 这个桥接只放在 app shell：业务视图保持纯 SwiftUI，避免不同页面重复改
/// NSWindow 属性导致标题栏、圆角、透明度互相覆盖。
struct WindowTransparencyConfigurator: NSViewRepresentable {
    /// 所有 Liquid Glass 面板背后的统一窗口底色透明度。单个面板不再叠自己的
    /// opaque 背景，因此这个值就是整 App 视觉密度的唯一入口。
    static let backingOpacity: CGFloat = 0.9
    static let lightBackingWhite: CGFloat = 0.94

    static func adaptiveBackingColor() -> NSColor {
        NSColor(name: nil) { appearance in
            if !appearance.momentoUsesDarkWindowBacking {
                return NSColor(deviceWhite: Self.lightBackingWhite, alpha: Self.backingOpacity)
            }
            return systemWindowBackingColor(for: appearance)
        }
    }

    static func systemWindowBackingColor(for appearance: NSAppearance) -> NSColor {
        var color = NSColor.windowBackgroundColor.withAlphaComponent(Self.backingOpacity)
        appearance.performAsCurrentDrawingAppearance {
            color = NSColor.windowBackgroundColor.withAlphaComponent(Self.backingOpacity)
        }
        return color
    }

    var fixedContentSize: CGSize?

    init(fixedContentSize: CGSize? = nil) {
        self.fixedContentSize = fixedContentSize
    }

    func makeNSView(context: Context) -> NSView {
        let view = NSView()
        DispatchQueue.main.async { applyTransparency(to: view.window) }
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {
        guard let window = nsView.window else {
            DispatchQueue.main.async { applyTransparency(to: nsView.window) }
            return
        }
        applyTransparency(to: window)
    }

    private func applyTransparency(to window: NSWindow?) {
        guard let window else { return }
        window.styleMask.insert(.fullSizeContentView)
        window.titlebarAppearsTransparent = true
        window.titleVisibility = .hidden
        window.isOpaque = false
        window.backgroundColor = Self.adaptiveBackingColor()

        if let fixedContentSize {
            window.contentMinSize = fixedContentSize
            window.contentMaxSize = fixedContentSize
            window.setContentSize(fixedContentSize)
        }
    }
}

private extension NSAppearance {
    var momentoUsesDarkWindowBacking: Bool {
        let match = bestMatch(from: [
            .accessibilityHighContrastDarkAqua,
            .darkAqua,
            .accessibilityHighContrastAqua,
            .aqua
        ])
        return match == .accessibilityHighContrastDarkAqua || match == .darkAqua
    }
}
