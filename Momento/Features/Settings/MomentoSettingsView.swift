import SwiftUI

struct MomentoSettingsView: View {
    @Environment(\.appLocalization) private var localization

    @Binding var appLanguage: AppLanguage
    @Binding var defaultViewMode: AssetViewMode

    var body: some View {
        Form {
            Section(localization.string("Language")) {
                Picker(localization.string("App Language"), selection: $appLanguage) {
                    ForEach(AppLanguage.allCases) { language in
                        Text(localization.title(for: language))
                            .tag(language)
                    }
                }
                .pickerStyle(.menu)
            }

            Section(localization.string("Appearance")) {
                Picker(localization.string("Default View"), selection: $defaultViewMode) {
                    ForEach(AssetViewMode.allCases) { viewMode in
                        Text(localization.title(for: viewMode))
                            .tag(viewMode)
                    }
                }
                .pickerStyle(.segmented)
            }

            Section(localization.string("About")) {
                LabeledContent(localization.string("Name"), value: "Momento")
                LabeledContent(localization.string("Version"), value: versionText)
            }
        }
        .formStyle(.grouped)
        .padding(20)
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
        .frame(width: 420)
    }

    private var versionText: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }
}

#Preview {
    @Previewable @State var language = AppLanguage.system
    @Previewable @State var viewMode = AssetViewMode.masonry

    MomentoSettingsView(appLanguage: $language, defaultViewMode: $viewMode)
        .environment(\.appLocalization, AppLocalization(language: language))
}
