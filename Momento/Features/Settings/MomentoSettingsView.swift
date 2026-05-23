// 中文注释：设置页只绑定用户偏好状态，网格等主工作区设置不在这里重复维护。
import SwiftUI

struct MomentoSettingsView: View {
    @Environment(\.appLocalization) private var localization

    @Binding var appLanguage: AppLanguage

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
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
                    .buttonStyle(.glass(.clear))
                    .frame(width: 190)
                }
            }

            settingsSection(localization.string("About")) {
                settingsInfoRow(label: localization.string("Name"), value: "Momento")
                settingsInfoRow(label: localization.string("Version"), value: versionText)
            }

            Spacer(minLength: 0)
        }
        .padding(20)
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
        .frame(width: 420, alignment: .topLeading)
        .frame(minHeight: 220, alignment: .topLeading)
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
                    glass: .regular.tint(Color.black.opacity(0.12)),
                    cornerRadius: 16
                )
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
        .frame(minHeight: 34)
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
        .frame(minHeight: 30)
    }
}

#Preview {
    @Previewable @State var language = AppLanguage.system

    MomentoSettingsView(appLanguage: $language)
        .environment(\.appLocalization, AppLocalization(language: language))
}
