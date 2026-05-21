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

    func testMainSidebarDoesNotListAllAssetsEntry() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("id: \"all-assets\""))
        XCTAssertFalse(source.contains("id: \"recent\""))
    }

    func testMainSidebarDoesNotListTrashEntry() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let contentSidebarStart = try XCTUnwrap(contentSource.range(of: "    private var sidebarSections: [MomentoSidebarSection] {"))
        let contentSidebarEnd = try XCTUnwrap(contentSource[contentSidebarStart.lowerBound...].range(of: "    private var commands: [MomentoCommand] {"))
        let contentSidebarSource = String(contentSource[contentSidebarStart.lowerBound..<contentSidebarEnd.lowerBound])
        let defaultSidebarStart = try XCTUnwrap(sidebarSource.range(of: "    static func momentoDefaultSections(localization: AppLocalization) -> [MomentoSidebarSection] {"))
        let defaultSidebarEnd = try XCTUnwrap(sidebarSource[defaultSidebarStart.lowerBound...].range(of: "    static var momentoDefaultSections: [MomentoSidebarSection] {"))
        let defaultSidebarSource = String(sidebarSource[defaultSidebarStart.lowerBound..<defaultSidebarEnd.lowerBound])

        XCTAssertFalse(contentSidebarSource.contains("id: \"trash\""))
        XCTAssertFalse(contentSidebarSource.contains("systemImage: \"trash\""))
        XCTAssertFalse(defaultSidebarSource.contains("id: \"trash\""))
        XCTAssertFalse(defaultSidebarSource.contains("systemImage: \"trash\""))
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

    func testWelcomeBackdropDoesNotUseManualOpacityState() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("@Environment(\\.appearsActive)"))
        XCTAssertFalse(source.contains("inactiveBackdropOpacity"))
        XCTAssertFalse(source.contains("focusedBackdropOpacity"))
        XCTAssertFalse(source.contains("windowBackgroundOpacity"))
        XCTAssertFalse(source.contains("@Environment(\\.controlActiveState)"))
    }

    func testWelcomeDoesNotConfigureManualWindowTransparency() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("WelcomeWindowTransparencyConfigurator"))
        XCTAssertFalse(source.contains("window.isOpaque"))
        XCTAssertFalse(source.contains("window.backgroundColor"))
        XCTAssertFalse(source.contains("restoreWindowConfiguration()"))
        XCTAssertFalse(source.contains("NSViewRepresentable"))
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

        XCTAssertTrue(source.contains("welcomeButtonWidth: CGFloat = 116"))
        XCTAssertTrue(source.contains("welcomeButtonHeight: CGFloat = 36"))
        XCTAssertTrue(source.contains("welcomeButtonFontSize: CGFloat = 13"))
        let fixedSizeFrameCount = source
            .components(separatedBy: ".frame(width: welcomeButtonWidth, height: welcomeButtonHeight)")
            .count - 1
        XCTAssertGreaterThanOrEqual(fixedSizeFrameCount, 2)

        let buttonFontCount = source
            .components(separatedBy: ".font(.system(size: welcomeButtonFontSize, weight: .semibold))")
            .count - 1
        XCTAssertGreaterThanOrEqual(buttonFontCount, 2)
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

    func testWelcomeButtonsProvideVisibleHoverFeedback() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("@Environment(\\.accessibilityReduceMotion) private var reduceMotion"))
        XCTAssertTrue(source.contains("@State private var isCreateButtonHovered = false"))
        XCTAssertTrue(source.contains("@State private var isOpenButtonHovered = false"))

        let hoverCount = source
            .components(separatedBy: ".onHover { isHovered in")
            .count - 1
        XCTAssertGreaterThanOrEqual(hoverCount, 2)
        XCTAssertTrue(source.contains(".welcomeButtonHoverFeedback(isHovered: isCreateButtonHovered, reduceMotion: reduceMotion)"))
        XCTAssertTrue(source.contains(".welcomeButtonHoverFeedback(isHovered: isOpenButtonHovered, reduceMotion: reduceMotion)"))
        XCTAssertTrue(source.contains("scaleEffect(isHovered && !reduceMotion ? 1.035 : 1)"))
        XCTAssertTrue(source.contains(".brightness(isHovered ? 0.08 : 0)"))
        XCTAssertTrue(source.contains(".animation(reduceMotion ? nil : .smooth(duration: 0.16), value: isHovered)"))
    }

    func testCreateLibraryUsesNameDialogBeforeDirectoryPicker() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("@State private var isCreateLibraryDialogPresented = false"))
        XCTAssertTrue(source.contains("MomentoCreateLibraryDialog("))
        XCTAssertTrue(source.contains("isPresented: $isCreateLibraryDialogPresented"))
        XCTAssertTrue(source.contains("onContinue: chooseLibraryDestination"))
        XCTAssertTrue(source.contains("isCreateLibraryDialogPresented = true"))
        XCTAssertFalse(source.contains("NSSavePanel()"))
        XCTAssertTrue(source.contains("private func chooseLibraryDestination(named libraryName: String)"))
        XCTAssertTrue(source.contains("let panel = NSOpenPanel()"))
        XCTAssertTrue(source.contains("panel.canChooseFiles = false"))
        XCTAssertTrue(source.contains("panel.canChooseDirectories = true"))
        XCTAssertTrue(source.contains("panel.canCreateDirectories = true"))
        XCTAssertTrue(source.contains("let packageURL = destinationURL.appendingPathComponent(libraryName, isDirectory: true)"))
        XCTAssertTrue(source.contains("try store.createLibrary(at: packageURL)"))
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

    func testWelcomeBackdropUsesNativeGlassEffectWithoutTintOpacity() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("MomentoGlassBackground(cornerRadius: 0)"))
        XCTAssertFalse(source.contains("welcomeGlassTintOpacity"))
        XCTAssertFalse(source.contains(".opacity("))
        XCTAssertFalse(source.contains("LinearGradient("))
        XCTAssertFalse(source.contains("MomentoVisualEffectView("))
        XCTAssertFalse(source.contains("NSVisualEffectView"))
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }

    private func contentViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/ContentView.swift")
    }

    private func sidebarViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Sidebar/MomentoSidebarView.swift")
    }
}
