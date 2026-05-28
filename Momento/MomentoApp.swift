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
    @StateObject private var updateService = AppUpdateService()
    @State private var store = LibraryStore(defaultViewMode: AppSettings.defaultViewMode())
    @State private var browserImportServer = BrowserImportServer()
    @AppStorage(AppSettingsKeys.appLanguage) private var appLanguageRawValue = AppLanguage.system.rawValue

    var body: some Scene {
        let language = appLanguage
        let localization = AppLocalization(language: language)

        WindowGroup {
            ContentView(store: store, updateService: updateService)
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
        .commands {
            MomentoMenuCommands(localization: localization)
            MomentoUpdateCommands(localization: localization, updateService: updateService)
        }

        Settings {
            MomentoSettingsView(
                appLanguage: appLanguageBinding,
                updateService: updateService
            )
            .environment(\.locale, language.locale)
            .environment(\.appLocalization, localization)
        }
        .defaultSize(MomentoSettingsView.preferredSize)
        .windowResizability(.contentSize)
        .windowStyle(.hiddenTitleBar)
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
                if request.playFeedback {
                    BrowserImportNotificationService.playImageSavedFeedback()
                }
            }
        } catch {
            store.libraryErrorMessage = error.localizedDescription
        }
    }
}

private struct MomentoUpdateCommands: Commands {
    var localization: AppLocalization
    @ObservedObject var updateService: AppUpdateService

    var body: some Commands {
        CommandGroup(after: .appInfo) {
            Button(localization.string("Check for Updates...")) {
                updateService.checkForUpdates()
            }
            .disabled(!updateService.canCheckForUpdates)
        }
    }
}

typealias MomentoMenuCommandAction = (String) -> Void

extension FocusedValues {
    @Entry var momentoMenuCommandAction: MomentoMenuCommandAction?
}

private struct MomentoMenuCommands: Commands {
    var localization: AppLocalization

    @FocusedValue(\.momentoMenuCommandAction) private var commandAction

    var body: some Commands {
        CommandMenu(localization.string("Library")) {
            commandButton(localization.string("Import Assets"), id: "import")
                .keyboardShortcut("i", modifiers: .command)
            commandButton(localization.string("Import Library"), id: "import-library")
            commandButton(localization.string("Export Library"), id: "export-library")
        }

        CommandMenu(localization.string("View")) {
            commandButton(localization.string("Masonry View"), id: "view-masonry")
                .keyboardShortcut("1", modifiers: .command)
            commandButton(localization.string("Grid View"), id: "view-grid")
                .keyboardShortcut("2", modifiers: .command)
            commandButton(localization.string("List View"), id: "view-list")
                .keyboardShortcut("3", modifiers: .command)

            Divider()

            commandButton(localization.string("Focus Search"), id: "focus-search")
                .keyboardShortcut("f", modifiers: .command)
            commandButton(localization.string("Toggle Filter"), id: "toggle-filter")
                .keyboardShortcut("f", modifiers: [.command, .option])
            commandButton(localization.string("Toggle Sort"), id: "toggle-sort")
                .keyboardShortcut("s", modifiers: [.command, .option])
            commandButton(localization.string("Toggle Inspector"), id: "toggle-inspector")
                .keyboardShortcut("i", modifiers: [.command, .option])
        }

        CommandMenu(localization.string("Asset")) {
            commandButton(localization.string("Quick Preview"), id: "quick-preview")
                .keyboardShortcut(.space, modifiers: [])
            commandButton(localization.string("Move to Trash"), id: "move-to-trash")
                .keyboardShortcut(.delete, modifiers: .command)
        }
    }

    private func commandButton(_ title: String, id: String) -> some View {
        Button(title) {
            commandAction?(id)
        }
        .disabled(commandAction == nil)
    }
}
