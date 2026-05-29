// 中文注释：欢迎页只承载创建/打开/最近资源库入口，不直接操作底层文件系统。
import SwiftUI

private let welcomeButtonWidth: CGFloat = 116
private let welcomeButtonHeight: CGFloat = 36
private let welcomeButtonFontSize: CGFloat = 13

struct MomentoLibraryWelcomeView: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var isCreateButtonHovered = false
    @State private var isOpenButtonHovered = false

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
                            .font(.system(size: welcomeButtonFontSize, weight: .semibold))
                            .frame(width: welcomeButtonWidth, height: welcomeButtonHeight)
                    }
                    .momentoGlassButtonStyle(.prominent)
                    .buttonBorderShape(.capsule)
                    .tint(Color.accentColor)
                    .foregroundStyle(Color.white)
                    .contentShape(Capsule(style: .continuous))
                    .environment(\.appearsActive, true)
                    .pointerStyle(.link)
                    .welcomeButtonHoverFeedback(isHovered: isCreateButtonHovered, reduceMotion: reduceMotion)
                    .onHover { isHovered in
                        isCreateButtonHovered = isHovered
                    }

                    Button {
                        onOpenLibrary()
                    } label: {
                        Label(localization.string("Open Library"), systemImage: "folder.fill.badge.plus")
                            .labelStyle(.titleAndIcon)
                            .font(.system(size: welcomeButtonFontSize, weight: .semibold))
                            .frame(width: welcomeButtonWidth, height: welcomeButtonHeight)
                    }
                    .momentoGlassButtonStyle()
                    .buttonBorderShape(.capsule)
                    .foregroundStyle(Color.white)
                    .contentShape(Capsule(style: .continuous))
                    .environment(\.appearsActive, true)
                    .pointerStyle(.link)
                    .welcomeButtonHoverFeedback(isHovered: isOpenButtonHovered, reduceMotion: reduceMotion)
                    .onHover { isHovered in
                        isOpenButtonHovered = isHovered
                    }
                }
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

private struct WelcomeGlassBackdrop: View {
    var body: some View {
        MomentoGlassBackground(cornerRadius: 0)
            .ignoresSafeArea()
    }
}

private extension View {
    func welcomeButtonHoverFeedback(isHovered: Bool, reduceMotion: Bool) -> some View {
        scaleEffect(isHovered && !reduceMotion ? 1.035 : 1)
            .brightness(isHovered ? 0.08 : 0)
            .animation(reduceMotion ? nil : .smooth(duration: 0.16), value: isHovered)
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
