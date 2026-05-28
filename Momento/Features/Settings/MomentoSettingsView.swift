// 中文注释：设置页只绑定用户偏好状态，网格等主工作区设置不在这里重复维护。
import SwiftUI

private enum MomentoSettingsMetrics {
    static let windowWidth: CGFloat = 460
    static let minWindowHeight: CGFloat = 312
    static let sectionSpacing: CGFloat = 14
    static let rowHeight: CGFloat = 38
    static let controlWidth: CGFloat = 196
    static let controlHeight: CGFloat = 34
    static let sectionRadius: CGFloat = 16
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

            VStack(alignment: .leading, spacing: 18) {
                Text(localization.string("Settings"))
                    .font(.system(size: 20, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)

                VStack(alignment: .leading, spacing: MomentoSettingsMetrics.sectionSpacing) {
                    settingsSection(localization.string("Language")) {
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
                    }

                    settingsSection(localization.string("Updates")) {
                        settingsRow(label: localization.string("App Updates")) {
                            Button {
                                updateService.checkForUpdates()
                            } label: {
                                Label(localization.string("Check for Updates..."), systemImage: "arrow.triangle.2.circlepath")
                                    .labelStyle(.titleAndIcon)
                                    .font(.system(size: 13, weight: .semibold))
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
                    }

                    settingsSection(localization.string("About")) {
                        settingsInfoRow(label: localization.string("Name"), value: "Momento")
                        settingsDivider
                        settingsInfoRow(label: localization.string("Version"), value: versionText)
                    }
                }

                Spacer(minLength: 0)
            }
            .padding(.top, 46)
            .padding(.horizontal, 22)
            .padding(.bottom, 22)
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

    private func settingsSection<Content: View>(
        _ title: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(MomentoTheme.secondaryText)
                .padding(.horizontal, 2)

            VStack(spacing: 0) {
                content()
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .background {
                MomentoGlassBackground(
                    glass: .regular.tint(Color.white.opacity(0.05)).interactive(true),
                    cornerRadius: MomentoSettingsMetrics.sectionRadius
                )
            }
            .overlay {
                RoundedRectangle(cornerRadius: MomentoSettingsMetrics.sectionRadius, style: .continuous)
                    .strokeBorder(MomentoTheme.subtleStroke.opacity(0.38), lineWidth: 0.6)
            }
        }
    }

    private func settingsRow<Content: View>(
        label: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        HStack(spacing: 12) {
            Text(label)
                .foregroundStyle(MomentoTheme.primaryText)

            Spacer(minLength: 12)

            content()
        }
        .font(.system(size: 13, weight: .medium))
        .frame(height: MomentoSettingsMetrics.rowHeight)
    }

    private func settingsInfoRow(label: String, value: String) -> some View {
        HStack(spacing: 12) {
            Text(label)
                .foregroundStyle(MomentoTheme.secondaryText)

            Spacer(minLength: 12)

            Text(value)
                .foregroundStyle(MomentoTheme.primaryText)
                .multilineTextAlignment(.trailing)
        }
        .font(.system(size: 13, weight: .medium))
        .frame(height: MomentoSettingsMetrics.rowHeight)
    }

    private var settingsDivider: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(0.34))
            .frame(height: 0.6)
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
        .frame(width: 460, height: 312)
}
