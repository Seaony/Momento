import AppKit
import SwiftUI

private let inactiveBackdropOpacity = 1.0
private let focusedBackdropOpacity = 0.64
private let welcomeGlassTintOpacity = 0.28
private let welcomeButtonWidth: CGFloat = 124
private let welcomeButtonHeight: CGFloat = 42

struct MomentoLibraryWelcomeView: View {
    @Environment(\.appLocalization) private var localization

    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void

    var body: some View {
        ZStack {
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
                    .buttonStyle(.glassProminent)
                    .buttonBorderShape(.capsule)
                    .tint(Color.accentColor)
                    .foregroundStyle(Color.white)
                    .contentShape(Capsule(style: .continuous))
                    .environment(\.appearsActive, true)
                    .pointerStyle(.link)

                    Button {
                        onOpenLibrary()
                    } label: {
                        Label(localization.string("Open Library"), systemImage: "folder.fill.badge.plus")
                            .labelStyle(.titleAndIcon)
                            .font(.system(size: 14, weight: .semibold))
                            .frame(width: welcomeButtonWidth, height: welcomeButtonHeight)
                    }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)
                    .foregroundStyle(Color.white)
                    .contentShape(Capsule(style: .continuous))
                    .environment(\.appearsActive, true)
                    .pointerStyle(.link)
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

private struct WelcomeGlassBackdrop: View {
    @Environment(\.appearsActive) private var appearsActive

    private var windowBackgroundOpacity: Double {
        appearsActive ? focusedBackdropOpacity : inactiveBackdropOpacity
    }

    var body: some View {
        ZStack {
            Color.clear
                .glassEffect(.regular.tint(Color(nsColor: .windowBackgroundColor).opacity(welcomeGlassTintOpacity)), in: Rectangle())

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
