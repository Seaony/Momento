//
//  MomentoApp.swift
//  Momento
//
//  Created by Seaony on 2026/5/21.
//

import SwiftUI

@main
struct MomentoApp: App {
    @AppStorage(AppSettingsKeys.appLanguage) private var appLanguageRawValue = AppLanguage.system.rawValue
    @AppStorage(AppSettingsKeys.defaultViewMode) private var defaultViewModeRawValue = AssetViewMode.masonry.rawValue

    var body: some Scene {
        let language = appLanguage
        let localization = AppLocalization(language: language)

        WindowGroup {
            ContentView()
                .environment(\.locale, language.locale)
                .environment(\.appLocalization, localization)
        }
        .windowStyle(.hiddenTitleBar)
        .windowToolbarStyle(.unifiedCompact)

        Settings {
            MomentoSettingsView(
                appLanguage: appLanguageBinding,
                defaultViewMode: defaultViewModeBinding
            )
            .environment(\.locale, language.locale)
            .environment(\.appLocalization, localization)
        }
    }

    private var appLanguage: AppLanguage {
        AppLanguage(rawValue: appLanguageRawValue) ?? .system
    }

    private var defaultViewMode: AssetViewMode {
        AssetViewMode(rawValue: defaultViewModeRawValue) ?? .masonry
    }

    private var appLanguageBinding: Binding<AppLanguage> {
        Binding {
            appLanguage
        } set: { newValue in
            appLanguageRawValue = newValue.rawValue
        }
    }

    private var defaultViewModeBinding: Binding<AssetViewMode> {
        Binding {
            defaultViewMode
        } set: { newValue in
            defaultViewModeRawValue = newValue.rawValue
        }
    }
}
