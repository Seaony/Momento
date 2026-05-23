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
    @State private var browserImportServer = BrowserImportServer()
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
                    startBrowserImportServer()
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

    private func startBrowserImportServer() {
        do {
            try browserImportServer.start { url in
                try await store.importRemoteImage(from: url)
                let localization = AppLocalization(language: appLanguage)
                await BrowserImportNotificationService.notifyImageSaved(
                    title: localization.string("Saved to Momento"),
                    body: localization.string("The image has been imported into your current library.")
                )
            }
        } catch {
            store.libraryErrorMessage = error.localizedDescription
        }
    }
}
