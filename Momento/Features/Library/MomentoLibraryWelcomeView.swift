import AppKit
import SwiftUI

private let inactiveBackdropOpacity = 1.0
private let focusedBackdropOpacity = 0.56
private let welcomeButtonWidth: CGFloat = 136
private let welcomeButtonHeight: CGFloat = 42

struct MomentoLibraryWelcomeView: View {
    @Environment(\.appLocalization) private var localization
    @State private var isCreateHovered = false
    @State private var isOpenHovered = false

    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void

    var body: some View {
        ZStack {
            WelcomeWindowTransparencyConfigurator()
                .frame(width: 0, height: 0)

            WelcomeGlassBackdrop()

            VStack(spacing: 18) {
                Image(systemName: "books.vertical")
                    .font(.system(size: 44, weight: .regular))
                    .foregroundStyle(Color.accentColor)

                VStack(spacing: 6) {
                    Text(localization.string("No Library Open"))
                        .font(.system(size: 22, weight: .semibold))
                    Text(localization.string("Create or open a Momento library to start organizing assets."))
                        .font(.system(size: 13))
                        .foregroundStyle(MomentoTheme.secondaryText)
                        .multilineTextAlignment(.center)
                        .frame(maxWidth: 360)
                }

                HStack(spacing: 10) {
                    Button {
                        onCreateLibrary()
                    } label: {
                        Label(localization.string("Create Library"), systemImage: "plus.circle.fill")
                            .labelStyle(.titleAndIcon)
                            .font(.system(size: 14, weight: .semibold))
                            .frame(width: welcomeButtonWidth, height: welcomeButtonHeight)
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(Color.white)
                    .contentShape(Capsule(style: .continuous))
                    .glassEffect(.regular.tint(Color.accentColor).interactive(), in: Capsule(style: .continuous))
                    .environment(\.appearsActive, true)
                    .scaleEffect(isCreateHovered ? 1.025 : 1)
                    .shadow(
                        color: .black.opacity(isCreateHovered ? 0.24 : 0.18),
                        radius: isCreateHovered ? 14 : 10,
                        x: 0,
                        y: isCreateHovered ? 8 : 5
                    )
                    .onHover { hovering in
                        isCreateHovered = hovering
                    }
                    .pointerStyle(.link)
                    .animation(.smooth(duration: 0.16), value: isCreateHovered)

                    Button {
                        onOpenLibrary()
                    } label: {
                        Label(localization.string("Open Library"), systemImage: "folder.fill.badge.plus")
                            .labelStyle(.titleAndIcon)
                            .font(.system(size: 14, weight: .semibold))
                            .frame(width: welcomeButtonWidth, height: welcomeButtonHeight)
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(Color.white)
                    .contentShape(Capsule(style: .continuous))
                    .glassEffect(.regular.interactive(), in: Capsule(style: .continuous))
                    .environment(\.appearsActive, true)
                    .scaleEffect(isOpenHovered ? 1.025 : 1)
                    .shadow(
                        color: .black.opacity(isOpenHovered ? 0.18 : 0.10),
                        radius: isOpenHovered ? 14 : 8,
                        x: 0,
                        y: isOpenHovered ? 8 : 4
                    )
                    .onHover { hovering in
                        isOpenHovered = hovering
                    }
                    .pointerStyle(.link)
                    .animation(.smooth(duration: 0.16), value: isOpenHovered)
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

private struct WelcomeWindowTransparencyConfigurator: NSViewRepresentable {
    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> WelcomeWindowObserverView {
        let view = WelcomeWindowObserverView()
        view.onWindowChanged = { [weak coordinator = context.coordinator] window in
            coordinator?.configureWindow(window)
        }
        return view
    }

    func updateNSView(_ view: WelcomeWindowObserverView, context: Context) {
        context.coordinator.configureWindow(view.window)
    }

    static func dismantleNSView(_ view: WelcomeWindowObserverView, coordinator: Coordinator) {
        coordinator.restoreWindowConfiguration()
    }

    final class Coordinator {
        private weak var configuredWindow: NSWindow?
        private var originalIsOpaque: Bool?
        private var originalBackgroundColor: NSColor?

        func configureWindow(_ window: NSWindow?) {
            guard let window else {
                return
            }

            if configuredWindow !== window {
                restoreWindowConfiguration()
                configuredWindow = window
                originalIsOpaque = window.isOpaque
                originalBackgroundColor = window.backgroundColor
            }

            window.isOpaque = false
            window.backgroundColor = .clear
        }

        func restoreWindowConfiguration() {
            guard let configuredWindow else {
                return
            }

            if let originalIsOpaque {
                configuredWindow.isOpaque = originalIsOpaque
            }
            if let originalBackgroundColor {
                configuredWindow.backgroundColor = originalBackgroundColor
            }

            self.configuredWindow = nil
            originalIsOpaque = nil
            originalBackgroundColor = nil
        }
    }
}

private final class WelcomeWindowObserverView: NSView {
    var onWindowChanged: ((NSWindow?) -> Void)?

    override func viewDidMoveToWindow() {
        super.viewDidMoveToWindow()
        onWindowChanged?(window)
    }
}

private struct WelcomeGlassBackdrop: View {
    @Environment(\.appearsActive) private var appearsActive

    private var windowBackgroundOpacity: Double {
        appearsActive ? focusedBackdropOpacity : inactiveBackdropOpacity
    }

    var body: some View {
        ZStack {
            MomentoVisualEffectView(
                material: .underWindowBackground,
                blendingMode: .behindWindow,
                state: .active,
                emphasized: true
            )

            Color(nsColor: .windowBackgroundColor)
                .opacity(windowBackgroundOpacity)
                .animation(.smooth(duration: 0.18), value: appearsActive)

            LinearGradient(
                colors: [
                    Color.white.opacity(0.035),
                    Color.accentColor.opacity(0.030),
                    Color.black.opacity(0.025)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )
        }
        .ignoresSafeArea()
    }
}

#Preview {
    MomentoLibraryWelcomeView(
        onCreateLibrary: {},
        onOpenLibrary: {}
    )
        .environment(\.appLocalization, AppLocalization(language: .system))
        .frame(width: 760, height: 520)
}
