import SwiftUI

struct MomentoLibraryWelcomeView: View {
    @Environment(\.appLocalization) private var localization
    @State private var isCreateHovered = false
    @State private var isOpenHovered = false

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
                        Label(localization.string("Create Library"), systemImage: "plus")
                    }
                    .buttonStyle(WelcomePrimaryButtonStyle(isHovered: isCreateHovered))
                    .onHover { isCreateHovered = $0 }

                    Button {
                        onOpenLibrary()
                    } label: {
                        Label(localization.string("Open Library"), systemImage: "folder")
                    }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)
                    .controlSize(.large)
                    .foregroundStyle(isOpenHovered ? Color.primary : MomentoTheme.secondaryText)
                    .scaleEffect(isOpenHovered ? 1.025 : 1)
                    .shadow(
                        color: .black.opacity(isOpenHovered ? 0.18 : 0.10),
                        radius: isOpenHovered ? 14 : 8,
                        x: 0,
                        y: isOpenHovered ? 8 : 4
                    )
                    .onHover { isOpenHovered = $0 }
                    .animation(.smooth(duration: 0.16), value: isOpenHovered)
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

private struct WelcomeGlassBackdrop: View {
    var body: some View {
        ZStack {
            MomentoVisualEffectView(
                material: .underWindowBackground,
                blendingMode: .behindWindow,
                state: .active,
                emphasized: true
            )

            Color(nsColor: .windowBackgroundColor)
                .opacity(0.88)

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

private struct WelcomePrimaryButtonStyle: ButtonStyle {
    var isHovered: Bool

    func makeBody(configuration: Configuration) -> some View {
        let isPressed = configuration.isPressed
        let fillOpacity = isPressed ? 0.82 : (isHovered ? 1.0 : 0.94)
        let shadowOpacity = isPressed ? 0.12 : (isHovered ? 0.24 : 0.18)

        configuration.label
            .labelStyle(.titleAndIcon)
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(Color.white)
            .padding(.horizontal, 18)
            .frame(height: 36)
            .background {
                Capsule(style: .continuous)
                    .fill(Color.accentColor.opacity(fillOpacity))
                    .overlay(alignment: .top) {
                        Capsule(style: .continuous)
                            .fill(Color.white.opacity(isHovered ? 0.24 : 0.18))
                            .frame(height: 1)
                            .padding(.horizontal, 13)
                    }
                    .overlay {
                        Capsule(style: .continuous)
                            .strokeBorder(Color.white.opacity(isHovered ? 0.28 : 0.20), lineWidth: 1)
                    }
                    .shadow(
                        color: .black.opacity(shadowOpacity),
                        radius: isPressed ? 6 : (isHovered ? 14 : 10),
                        x: 0,
                        y: isPressed ? 3 : (isHovered ? 8 : 5)
                    )
            }
            .scaleEffect(isPressed ? 0.98 : (isHovered ? 1.025 : 1))
            .animation(.smooth(duration: 0.16), value: isHovered)
            .animation(.smooth(duration: 0.12), value: isPressed)
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
