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

    func testWelcomeButtonsUseNativeGlassEffects() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".glassEffect(.regular.tint(Color.accentColor).interactive(), in: Capsule(style: .continuous))"))
        XCTAssertTrue(source.contains(".glassEffect(.regular.interactive(), in: Capsule(style: .continuous))"))
        XCTAssertFalse(source.contains("WelcomePrimaryButtonStyle"))
        XCTAssertFalse(source.contains("WelcomeGlassButtonStyle"))
        XCTAssertFalse(source.contains(".overlay(alignment: .top)"))
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

        XCTAssertTrue(source.contains("welcomeButtonWidth: CGFloat = 136"))
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

        XCTAssertTrue(source.contains("pointingHandCursor()"))
        XCTAssertTrue(source.contains("NSCursor.pointingHand.push()"))
        XCTAssertTrue(source.contains("NSCursor.pop()"))
        let pointingHandUpdateCount = source
            .components(separatedBy: ".pointingHandCursor()")
            .count - 1
        XCTAssertGreaterThanOrEqual(pointingHandUpdateCount, 2)
        XCTAssertFalse(source.contains("PointingHandCursorView"))
        XCTAssertFalse(source.contains("addCursorRect(bounds, cursor: .pointingHand)"))
        XCTAssertFalse(source.contains("updatePointingHandCursor(isHovered: hovering)"))
    }

    func testOpenLibraryButtonKeepsWhiteGlassAppearance() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".foregroundStyle(Color.white)"))
        XCTAssertTrue(source.contains(".glassEffect(.regular.interactive(), in: Capsule(style: .continuous))"))
        XCTAssertTrue(source.contains(".environment(\\.appearsActive, true)"))
        XCTAssertFalse(source.contains("isOpenHovered ? Color.primary : MomentoTheme.secondaryText"))
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }
}
