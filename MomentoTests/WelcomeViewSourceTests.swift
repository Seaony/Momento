import Foundation
import XCTest

final class WelcomeViewSourceTests: XCTestCase {
    func testWelcomeViewDoesNotExposeLibrarySwitcherMenu() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("Menu {"))
        XCTAssertFalse(source.contains("libraryMenu"))
        XCTAssertFalse(source.contains("recentLibraries"))
        XCTAssertFalse(source.contains("onSwitchLibrary"))
        XCTAssertFalse(source.contains("No library selected"))
    }

    func testWelcomeViewUsesPrimaryAndNativeGlassButtons() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("WelcomePrimaryButtonStyle"))
        XCTAssertTrue(source.contains(".buttonStyle(.glass)"))
        XCTAssertFalse(source.contains("WelcomeGlassButtonStyle"))
    }

    func testWelcomeBackdropAdjustsOpacityWhenWindowIsFocused() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("@Environment(\\.controlActiveState)"))
        XCTAssertTrue(source.contains("inactiveBackdropOpacity = 0.88"))
        XCTAssertTrue(source.contains("focusedBackdropOpacity = 0.84"))
        XCTAssertTrue(source.contains("controlActiveState == .inactive ? inactiveBackdropOpacity : focusedBackdropOpacity"))
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }
}
