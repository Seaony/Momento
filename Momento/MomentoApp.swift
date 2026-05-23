//
//  MomentoApp.swift
//  Momento
//
//  Created by Seaony on 2026/5/21.
//

// 中文注释：应用入口负责注入全局 store、本地化环境、窗口命令和浏览器导入服务。
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
            try browserImportServer.start { request in
                try await store.importRemoteImage(
                    from: request.imageURL,
                    sourcePageURL: request.sourcePageURL
                )
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
