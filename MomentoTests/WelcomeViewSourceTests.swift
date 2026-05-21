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

        XCTAssertTrue(source.contains(".buttonStyle(.glassProminent)"))
        XCTAssertTrue(source.contains(".buttonStyle(.glass)"))
        XCTAssertFalse(source.contains("WelcomePrimaryButtonStyle"))
        XCTAssertFalse(source.contains("WelcomeGlassButtonStyle"))
        XCTAssertFalse(source.contains(".buttonStyle(.plain)"))
        XCTAssertFalse(source.contains(".overlay(alignment: .top)"))
        XCTAssertFalse(source.contains(".shadow("))
    }

    func testWelcomeBackdropAdjustsOpacityWhenWindowIsFocused() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("@Environment(\\.appearsActive)"))
        XCTAssertTrue(source.contains("inactiveBackdropOpacity = 1.0"))
        XCTAssertTrue(source.contains("focusedBackdropOpacity = 0.64"))
        XCTAssertTrue(source.contains("appearsActive ? focusedBackdropOpacity : inactiveBackdropOpacity"))
        XCTAssertFalse(source.contains("@Environment(\\.controlActiveState)"))
    }

    func testWelcomeDoesNotOwnWindowTransparency() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("WelcomeWindowTransparencyConfigurator"))
        XCTAssertFalse(source.contains("window.isOpaque = false"))
        XCTAssertFalse(source.contains("window.backgroundColor = .clear"))
        XCTAssertFalse(source.contains("restoreWindowConfiguration()"))
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

        XCTAssertTrue(source.contains("welcomeButtonWidth: CGFloat = 124"))
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

        let pointerStyleCount = source
            .components(separatedBy: ".pointerStyle(.link)")
            .count - 1
        XCTAssertGreaterThanOrEqual(pointerStyleCount, 2)
        XCTAssertFalse(source.contains("pointingHandCursor()"))
        XCTAssertFalse(source.contains("PointingHandCursorView"))
        XCTAssertFalse(source.contains("addCursorRect(bounds, cursor: .pointingHand)"))
        XCTAssertFalse(source.contains("NSCursor.pointingHand.push()"))
        XCTAssertFalse(source.contains("NSCursor.pop()"))
        XCTAssertFalse(source.contains("updatePointingHandCursor(isHovered: hovering)"))
    }

    func testWelcomeButtonsUseNativeCapsuleBorderShape() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        let capsuleShapeCount = source
            .components(separatedBy: ".buttonBorderShape(.capsule)")
            .count - 1
        XCTAssertGreaterThanOrEqual(capsuleShapeCount, 2)
        XCTAssertFalse(source.contains("ClipShape(Capsule"))
        XCTAssertFalse(source.contains(".clipShape(Capsule"))
    }

    func testOpenLibraryButtonKeepsWhiteGlassAppearance() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".foregroundStyle(Color.white)"))
        XCTAssertTrue(source.contains(".buttonStyle(.glass)"))
        XCTAssertTrue(source.contains(".environment(\\.appearsActive, true)"))
        XCTAssertFalse(source.contains("isOpenHovered ? Color.primary : MomentoTheme.secondaryText"))
    }

    func testWelcomeBackdropUsesNativeGlassEffect() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("welcomeGlassTintOpacity = 0.28"))
        XCTAssertTrue(source.contains(".glassEffect(.regular.tint(Color(nsColor: .windowBackgroundColor).opacity(welcomeGlassTintOpacity)), in: Rectangle())"))
        XCTAssertFalse(source.contains("MomentoVisualEffectView("))
        XCTAssertFalse(source.contains("NSVisualEffectView"))
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }
}
