//
//  MomentoApp.swift
//  Momento
//
//  Created by Seaony on 2026/5/21.
//

import SwiftUI

@main
struct MomentoApp: App {
    @NSApplicationDelegateAdaptor(AppOpenHandler.self) private var appOpenHandler
    @State private var store = LibraryStore(defaultViewMode: AppSettings.defaultViewMode())
    @AppStorage(AppSettingsKeys.appLanguage) private var appLanguageRawValue = AppLanguage.system.rawValue

    var body: some Scene {
        let language = appLanguage
        let localization = AppLocalization(language: language)

        WindowGroup {
            ContentView(store: store)
                .environment(\.locale, language.locale)
                .environment(\.appLocalization, localization)
                .onAppear {
                    appOpenHandler.onOpenLibraryURLs = openLibraryURLs
                    appOpenHandler.flushPendingLibraryURLs()
                }
        }
        .windowToolbarStyle(.unified)
        .defaultSize(width: MomentoTheme.defaultWindowWidth, height: MomentoTheme.defaultWindowHeight)
        .windowResizability(.contentMinSize)

        Settings {
            MomentoSettingsView(appLanguage: appLanguageBinding)
            .environment(\.locale, language.locale)
            .environment(\.appLocalization, localization)
        }
    }

    private var appLanguage: AppLanguage {
        AppLanguage(rawValue: appLanguageRawValue) ?? .system
    }

    private var appLanguageBinding: Binding<AppLanguage> {
        Binding {
            appLanguage
        } set: { newValue in
            appLanguageRawValue = newValue.rawValue
        }
    }

    private func openLibraryURLs(_ urls: [URL]) -> Bool {
        var didOpenLibrary = false

        for url in urls {
            do {
                try store.openLibrary(at: url)
                didOpenLibrary = true
            } catch {
                store.libraryErrorMessage = AppLocalization(language: appLanguage).errorMessage(error)
            }
        }

        return didOpenLibrary
    }
}
