import Foundation
import XCTest

final class ArchitectureGuardTests: XCTestCase {
    func testGlassBackgroundUsesNativeSwiftUIGlassEffect() throws {
        let source = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".glassEffect(glass, in: RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))"))
        XCTAssertFalse(source.contains("MomentoVisualEffectView"))
        XCTAssertFalse(source.contains("NSVisualEffectView"))
    }

    func testWindowToolbarStaysTransparentWithoutHidingControls() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
        XCTAssertTrue(contentSource.contains(".toolbarBackgroundVisibility(.hidden, for: .windowToolbar)"))
        XCTAssertFalse(contentSource.contains(".toolbarVisibility("))
        XCTAssertTrue(windowSource.contains("window.titlebarAppearsTransparent = true"))
        XCTAssertTrue(windowSource.contains("window.titleVisibility = .hidden"))
    }

    func testMainWindowMinimumSizeUsesAdaptiveShellBounds() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let themeSource = try String(contentsOf: designSystemURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellURL(), encoding: .utf8)

        XCTAssertTrue(themeSource.contains("static let mainWindowMinWidth: CGFloat = 800"))
        XCTAssertTrue(themeSource.contains("static let compactContentMinWidth: CGFloat = 300"))
        XCTAssertTrue(contentSource.contains(".frame(minWidth: MomentoTheme.mainWindowMinWidth, minHeight: MomentoTheme.mainWindowMinHeight)"))
        XCTAssertTrue(appSource.contains(".windowResizability(.contentMinSize)"))
        XCTAssertTrue(shellSource.contains("availableShellWidth"))
        XCTAssertTrue(shellSource.contains("MomentoTheme.compactContentMinWidth"))
    }

    func testContentViewValidatesCurrentLibraryWhenWindowAppears() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("store.validateCurrentLibraryAvailability()"))
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }

    private func appURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/MomentoApp.swift")
    }

    private func contentViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/ContentView.swift")
    }

    private func designSystemURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/DesignSystem/MomentoGlass.swift")
    }

    private func shellURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Shell/MomentoShellView.swift")
    }

    private func windowTransparencyURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/WindowTransparencyConfigurator.swift")
    }
}
