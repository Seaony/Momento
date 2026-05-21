import SwiftUI

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
                        Label(localization.string("Create Library"), systemImage: "plus")
                    }
                    .buttonStyle(WelcomeGlassButtonStyle(isProminent: true))

                    Button {
                        onOpenLibrary()
                    } label: {
                        Label(localization.string("Open Library"), systemImage: "folder")
                    }
                    .buttonStyle(WelcomeGlassButtonStyle())
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

            LinearGradient(
                colors: [
                    Color.white.opacity(0.10),
                    Color.black.opacity(0.06),
                    Color.accentColor.opacity(0.08)
                ],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            RadialGradient(
                colors: [
                    Color.white.opacity(0.12),
                    .clear
                ],
                center: .center,
                startRadius: 40,
                endRadius: 380
            )
        }
        .ignoresSafeArea()
    }
}

private struct WelcomeGlassButtonStyle: ButtonStyle {
    var isProminent = false

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .labelStyle(.titleAndIcon)
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(isProminent ? Color.white : Color.primary)
            .padding(.horizontal, 18)
            .frame(height: 36)
            .background {
                Capsule(style: .continuous)
                    .fill(.clear)
                    .background {
                        MomentoVisualEffectView(
                            material: .hudWindow,
                            blendingMode: .withinWindow,
                            state: .active,
                            emphasized: isProminent
                        )
                        .clipShape(Capsule(style: .continuous))
                    }
                    .overlay {
                        Capsule(style: .continuous)
                            .fill(
                                isProminent
                                    ? Color.accentColor.opacity(configuration.isPressed ? 0.46 : 0.36)
                                    : Color.white.opacity(configuration.isPressed ? 0.13 : 0.08)
                            )
                    }
                    .overlay(alignment: .top) {
                        Capsule(style: .continuous)
                            .fill(Color.white.opacity(isProminent ? 0.22 : 0.16))
                            .frame(height: 1)
                            .padding(.horizontal, 13)
                    }
                    .overlay {
                        Capsule(style: .continuous)
                            .strokeBorder(Color.white.opacity(isProminent ? 0.28 : 0.18), lineWidth: 1)
                    }
                    .shadow(color: .black.opacity(configuration.isPressed ? 0.12 : 0.20), radius: configuration.isPressed ? 6 : 14, x: 0, y: configuration.isPressed ? 3 : 8)
            }
            .scaleEffect(configuration.isPressed ? 0.98 : 1)
            .animation(.smooth(duration: 0.16), value: configuration.isPressed)
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
