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
        XCTAssertTrue(source.contains("inactiveBackdropOpacity = 1.0"))
        XCTAssertTrue(source.contains("focusedBackdropOpacity = 0.56"))
        XCTAssertTrue(source.contains("appearsActive ? focusedBackdropOpacity : inactiveBackdropOpacity"))
        XCTAssertFalse(source.contains("@Environment(\\.controlActiveState)"))
    }

    func testWelcomeConfiguresTransparentWindowForBehindWindowGlass() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("WelcomeWindowTransparencyConfigurator"))
        XCTAssertTrue(source.contains("window.isOpaque = false"))
        XCTAssertTrue(source.contains("window.backgroundColor = .clear"))
        XCTAssertTrue(source.contains("restoreWindowConfiguration()"))
    }

    func testWelcomeDoesNotOwnWindowCornerRadius() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("welcomeWindowCornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerCurve"))
        XCTAssertFalse(source.contains("contentView.layer?.masksToBounds"))
    }

    func testWelcomeButtonsUseSharedFixedMetrics() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("welcomeButtonWidth: CGFloat = 144"))
        XCTAssertTrue(source.contains("welcomeButtonHeight: CGFloat = 42"))
        let fixedSizeFrameCount = source
            .components(separatedBy: ".frame(width: welcomeButtonWidth, height: welcomeButtonHeight)")
            .count - 1
        XCTAssertGreaterThanOrEqual(fixedSizeFrameCount, 2)
    }

    func testWelcomeButtonsUseRequestedSymbols() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("systemImage: \"plus.circle.fill\""))
        XCTAssertTrue(source.contains("systemImage: \"folder.fill.badge.plus\""))
    }

    func testWelcomeButtonsUsePointingHandCursorOnHover() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("updatePointingHandCursor(isHovered: hovering)"))
        XCTAssertTrue(source.contains("NSCursor.pointingHand.set()"))
        let pointingHandUpdateCount = source
            .components(separatedBy: "updatePointingHandCursor(isHovered: hovering)")
            .count - 1
        XCTAssertGreaterThanOrEqual(pointingHandUpdateCount, 2)
    }

    func testOpenLibraryButtonKeepsWhiteGlassAppearance() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".foregroundStyle(Color.white)"))
        XCTAssertTrue(source.contains(".glassEffect(.regular.interactive(), in: Capsule(style: .continuous))"))
        XCTAssertFalse(source.contains("isOpenHovered ? Color.primary : MomentoTheme.secondaryText"))
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }
}
