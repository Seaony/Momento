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
        XCTAssertTrue(source.contains(".glassEffect(.regular.interactive(), in: Capsule(style: .continuous))"))
        XCTAssertFalse(source.contains("WelcomeGlassButtonStyle"))
    }

    func testWelcomeBackdropAdjustsOpacityWhenWindowIsFocused() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("@Environment(\\.appearsActive)"))
        XCTAssertTrue(source.contains("inactiveBackdropOpacity = 0.88"))
        XCTAssertTrue(source.contains("focusedBackdropOpacity = 0.76"))
        XCTAssertTrue(source.contains("appearsActive ? focusedBackdropOpacity : inactiveBackdropOpacity"))
        XCTAssertFalse(source.contains("@Environment(\\.controlActiveState)"))
    }

    func testWelcomeButtonsUseSharedFixedMetrics() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("welcomeButtonWidth: CGFloat = 152"))
        XCTAssertTrue(source.contains("welcomeButtonHeight: CGFloat = 36"))
        let fixedSizeFrameCount = source
            .components(separatedBy: ".frame(width: welcomeButtonWidth, height: welcomeButtonHeight)")
            .count - 1
        XCTAssertGreaterThanOrEqual(fixedSizeFrameCount, 2)
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }
}
