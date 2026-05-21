import Foundation
import XCTest

final class WindowChromeSourceTests: XCTestCase {
    func testContentViewDoesNotOverrideSystemWindowCornerRadius() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("MomentoWindowCornerRadiusConfigurator"))
        XCTAssertFalse(source.contains("momentoWindowCornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerCurve"))
        XCTAssertFalse(source.contains("contentView.layer?.masksToBounds"))
        XCTAssertFalse(source.contains("import QuartzCore"))
    }

    func testMainWindowUsesSystemToolbarChromeForLargeTahoeCorners() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
        XCTAssertFalse(appSource.contains(".windowToolbarStyle(.unifiedCompact)"))

        XCTAssertTrue(contentSource.contains(".toolbar {"))
        XCTAssertFalse(contentSource.contains("ToolbarItemGroup(placement: .navigation)"))
        XCTAssertFalse(contentSource.contains("ToolbarItem(placement: .principal)"))
        XCTAssertTrue(contentSource.contains(".searchable(text: $store.searchQuery, placement: .toolbar"))
        XCTAssertFalse(shellSource.contains("MomentoTopBar("))
    }

    func testMainToolbarDoesNotShowAllAssetsTitleOrAdjacentImportButton() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(contentSource.contains("MomentoToolbarTitle("))
        XCTAssertFalse(contentSource.contains("private struct MomentoToolbarTitle"))
        XCTAssertFalse(contentSource.contains("ToolbarItemGroup(placement: .navigation)"))
        XCTAssertFalse(contentSource.contains("case .library:\n            localization.string(\"All Assets\")"))
    }

    func testLibraryWindowHidesTitleTextButKeepsToolbarControls() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains(".navigationTitle(\"\")"))
        XCTAssertFalse(contentSource.contains(".navigationTitle(title)"))
        XCTAssertTrue(windowSource.contains("window.titleVisibility = .hidden"))
        XCTAssertFalse(contentSource.contains(".toolbarVisibility("))
    }

    func testInspectorUsesCollapsibleTrailingColumnAndToolbarToggle() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("@State private var isInspectorPresented = true"))
        XCTAssertTrue(contentSource.contains("isInspectorPresented: $isInspectorPresented"))
        XCTAssertTrue(contentSource.contains("Toggle(isOn: $isInspectorPresented)"))
        XCTAssertTrue(contentSource.contains("Label(localization.string(\"Toggle Inspector\"), systemImage: \"sidebar.right\")"))
        XCTAssertTrue(contentSource.contains("MomentoCommand(id: \"toggle-inspector\""))
        XCTAssertTrue(contentSource.contains("case \"toggle-inspector\":"))
        XCTAssertTrue(contentSource.contains("isInspectorPresented.toggle()"))

        XCTAssertTrue(shellSource.contains("@Binding var isInspectorPresented: Bool"))
        XCTAssertTrue(shellSource.contains(".inspector(isPresented: $isInspectorPresented)"))
        XCTAssertTrue(shellSource.contains(".inspectorColumnWidth("))
        XCTAssertFalse(shellSource.contains("HSplitView"))
    }

    func testWindowToolbarBackgroundIsTransparentWithoutHidingControls() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        // Hiding the window toolbar in the `.unified` style collapses the
        // title bar and takes the close/minimize/zoom controls with it. Keep
        // the controls and toolbar items, but let content show through the
        // toolbar background across the app.
        XCTAssertTrue(contentSource.contains(".toolbarBackgroundVisibility(.hidden, for: .windowToolbar)"))
        XCTAssertFalse(contentSource.contains(".toolbarVisibility("))
        XCTAssertTrue(contentSource.contains(".toolbar {"))
        XCTAssertTrue(contentSource.contains(".searchable(text: $store.searchQuery, placement: .toolbar"))
        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
    }

    func testLibraryMenuExposesCloseLibraryAction() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("onCloseLibrary: closeLibrary"))
        XCTAssertTrue(contentSource.contains("store.closeCurrentLibrary()"))
        XCTAssertTrue(shellSource.contains("onCloseLibrary"))
        XCTAssertTrue(sidebarSource.contains("onCloseLibrary"))
        XCTAssertTrue(sidebarSource.contains("Close Library"))
    }

    private func contentViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/ContentView.swift")
    }

    private func appURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/MomentoApp.swift")
    }

    private func shellViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Shell/MomentoShellView.swift")
    }

    private func sidebarViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Sidebar/MomentoSidebarView.swift")
    }

    private func windowTransparencyURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/AppKitBridge/WindowTransparencyConfigurator.swift")
    }
}
