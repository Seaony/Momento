// 中文注释：设置页只绑定用户偏好状态，网格等主工作区设置不在这里重复维护。
import SwiftUI

private enum MomentoSettingsMetrics {
    static let windowWidth: CGFloat = 386
    static let windowHeight: CGFloat = 232
    static let contentTopInset: CGFloat = 52
    static let contentHorizontalInset: CGFloat = 20
    static let contentBottomInset: CGFloat = 18
    static let rowHeight: CGFloat = 33
    static let labelWidth: CGFloat = 108
    static let controlWidth: CGFloat = 188
    static let controlHeight: CGFloat = 28
    static let panelRadius: CGFloat = 18
}

struct MomentoSettingsView: View {
    static let preferredSize = CGSize(
        width: MomentoSettingsMetrics.windowWidth,
        height: MomentoSettingsMetrics.windowHeight
    )

    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    @Binding var appLanguage: AppLanguage
    @ObservedObject var updateService: AppUpdateService

    @State private var isUpdateButtonHovered = false

    var body: some View {
        ZStack {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()

            VStack(alignment: .leading, spacing: 14) {
                settingsPanel
            }
            .padding(.top, MomentoSettingsMetrics.contentTopInset)
            .padding(.horizontal, MomentoSettingsMetrics.contentHorizontalInset)
            .padding(.bottom, MomentoSettingsMetrics.contentBottomInset)
        }
        .background {
            WindowTransparencyConfigurator()
        }
        .toolbarBackgroundVisibility(.hidden, for: .windowToolbar)
        .frame(
            width: MomentoSettingsMetrics.windowWidth,
            height: MomentoSettingsMetrics.windowHeight,
            alignment: .topLeading
        )
    }

    private var versionText: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }

    private var settingsPanel: some View {
        VStack(spacing: 0) {
            settingsRow(label: localization.string("Language")) {
                Picker("", selection: $appLanguage) {
                    ForEach(AppLanguage.allCases) { language in
                        Text(localization.title(for: language))
                            .tag(language)
                    }
                }
                .labelsHidden()
                .pickerStyle(.menu)
                .buttonStyle(.glass)
                .controlSize(.regular)
                .frame(
                    width: MomentoSettingsMetrics.controlWidth,
                    height: MomentoSettingsMetrics.controlHeight
                )
                .environment(\.appearsActive, true)
            }

            settingsDivider

            settingsRow(label: localization.string("Updates")) {
                Button {
                    updateService.checkForUpdates()
                } label: {
                    Label(localization.string("Check for Updates..."), systemImage: "arrow.triangle.2.circlepath")
                        .labelStyle(.titleAndIcon)
                        .font(.system(size: 12, weight: .semibold))
                        .frame(
                            width: MomentoSettingsMetrics.controlWidth,
                            height: MomentoSettingsMetrics.controlHeight
                        )
                }
                .buttonStyle(.glass)
                .controlSize(.regular)
                .foregroundStyle(MomentoTheme.primaryText)
                .environment(\.appearsActive, true)
                .disabled(!updateService.canCheckForUpdates)
                .pointerStyle(.link)
                .settingsButtonHoverFeedback(isHovered: isUpdateButtonHovered, reduceMotion: reduceMotion)
                .onHover { isHovered in
                    isUpdateButtonHovered = isHovered
                }
            }

            settingsDivider

            settingsInfoRow(label: localization.string("Name"), value: "Momento")
            settingsInsetDivider
            settingsInfoRow(label: localization.string("Version"), value: versionText)
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background {
            MomentoGlassBackground(
                glass: .regular.tint(Color.white.opacity(0.045)).interactive(true),
                cornerRadius: MomentoSettingsMetrics.panelRadius
            )
        }
        .overlay {
            RoundedRectangle(cornerRadius: MomentoSettingsMetrics.panelRadius, style: .continuous)
                .strokeBorder(MomentoTheme.subtleStroke.opacity(0.34), lineWidth: 0.6)
        }
    }

    private var settingsDivider: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(0.26))
            .frame(height: 0.5)
    }

    private var settingsInsetDivider: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(0.18))
            .frame(height: 0.5)
            .padding(.leading, MomentoSettingsMetrics.labelWidth)
    }

    private func settingsLabel(_ label: String, color: Color) -> some View {
        Text(label)
            .foregroundStyle(color)
            .frame(width: MomentoSettingsMetrics.labelWidth, alignment: .leading)
            .lineLimit(1)
            .minimumScaleFactor(0.86)
    }

    private func controlSlot<Content: View>(@ViewBuilder content: () -> Content) -> some View {
        HStack(spacing: 0) {
            Spacer(minLength: 10)
            content()
                .frame(width: MomentoSettingsMetrics.controlWidth, alignment: .trailing)
        }
        .frame(maxWidth: .infinity, alignment: .trailing)
    }

    private func settingsRow<Content: View>(
        label: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        HStack(spacing: 12) {
            settingsLabel(label, color: MomentoTheme.primaryText)
            controlSlot {
                content()
            }
        }
        .font(.system(size: 12, weight: .medium))
        .frame(height: MomentoSettingsMetrics.rowHeight)
    }

    private func settingsInfoRow(label: String, value: String) -> some View {
        HStack(spacing: 12) {
            settingsLabel(label, color: MomentoTheme.secondaryText)
            controlSlot {
                Text(value)
                    .foregroundStyle(MomentoTheme.primaryText)
                    .lineLimit(1)
                    .minimumScaleFactor(0.82)
                    .multilineTextAlignment(.trailing)
            }
        }
        .font(.system(size: 12, weight: .medium))
        .frame(height: MomentoSettingsMetrics.rowHeight)
    }
}

private extension View {
    func settingsButtonHoverFeedback(isHovered: Bool, reduceMotion: Bool) -> some View {
        scaleEffect(isHovered && !reduceMotion ? 1.025 : 1)
            .brightness(isHovered ? 0.06 : 0)
            .animation(reduceMotion ? nil : .smooth(duration: 0.16), value: isHovered)
    }
}

#Preview {
    @Previewable @State var language = AppLanguage.system

    MomentoSettingsView(
        appLanguage: $language,
        updateService: AppUpdateService()
    )
        .environment(\.appLocalization, AppLocalization(language: language))
        .frame(width: MomentoSettingsView.preferredSize.width, height: MomentoSettingsView.preferredSize.height)
}
