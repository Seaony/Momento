// 中文注释：设置页只绑定用户偏好状态，网格等主工作区设置不在这里重复维护。
import SwiftUI

private enum MomentoSettingsMetrics {
    static let windowWidth: CGFloat = 420
    static let minWindowHeight: CGFloat = 276
    static let rowHeight: CGFloat = 36
    static let labelWidth: CGFloat = 122
    static let controlWidth: CGFloat = 178
    static let controlHeight: CGFloat = 32
    static let panelRadius: CGFloat = 16
    static let headerIconSize: CGFloat = 28
}

struct MomentoSettingsView: View {
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
                settingsHeader
                settingsPanel
            }
            .padding(.top, 30)
            .padding(.horizontal, 18)
            .padding(.bottom, 18)
        }
        .background {
            WindowTransparencyConfigurator()
        }
        .toolbarBackgroundVisibility(.hidden, for: .windowToolbar)
        .frame(width: MomentoSettingsMetrics.windowWidth, alignment: .topLeading)
        .frame(minHeight: MomentoSettingsMetrics.minWindowHeight, alignment: .topLeading)
    }

    private var versionText: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }

    private var settingsHeader: some View {
        HStack(spacing: 9) {
            Image(systemName: "gearshape")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(MomentoTheme.primaryText)
                .frame(
                    width: MomentoSettingsMetrics.headerIconSize,
                    height: MomentoSettingsMetrics.headerIconSize
                )
                .background {
                    MomentoGlassBackground(
                        glass: .regular.tint(Color.white.opacity(0.04)).interactive(true),
                        cornerRadius: 9
                    )
                }

            Text(localization.string("Settings"))
                .font(.system(size: 15, weight: .semibold))
                .foregroundStyle(MomentoTheme.primaryText)
        }
    }

    private var settingsPanel: some View {
        VStack(spacing: 0) {
            settingsRow(label: localization.string("App Language")) {
                Picker("", selection: $appLanguage) {
                    ForEach(AppLanguage.allCases) { language in
                        Text(localization.title(for: language))
                            .tag(language)
                    }
                }
                .labelsHidden()
                .pickerStyle(.menu)
                .buttonStyle(.glass)
                .controlSize(.large)
                .frame(
                    width: MomentoSettingsMetrics.controlWidth,
                    height: MomentoSettingsMetrics.controlHeight
                )
                .environment(\.appearsActive, true)
            }

            settingsDivider

            settingsRow(label: localization.string("App Updates")) {
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
                .controlSize(.large)
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
        .padding(.vertical, 8)
        .background {
            MomentoGlassBackground(
                glass: .regular.tint(Color.white.opacity(0.035)).interactive(true),
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
            .fill(MomentoTheme.subtleStroke.opacity(0.3))
            .frame(height: 0.6)
            .padding(.vertical, 6)
    }

    private var settingsInsetDivider: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(0.24))
            .frame(height: 0.6)
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
        HStack {
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
        .font(.system(size: 13, weight: .medium))
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
        .font(.system(size: 13, weight: .medium))
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
        .frame(width: 420, height: 276)
}
