import Foundation
import XCTest

final class LiquidGlassSourceTests: XCTestCase {
    func testGlassBackgroundUsesNativeSwiftUIGlassEffect() throws {
        let source = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".glassEffect(glass, in: RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))"))
        XCTAssertFalse(source.contains("MomentoVisualEffectView"))
        XCTAssertFalse(source.contains("NSVisualEffectView"))
        XCTAssertFalse(source.contains("strokeOpacity"))
        XCTAssertFalse(source.contains(".strokeBorder(.white.opacity"))
        XCTAssertFalse(source.contains(".shadow(color: .black.opacity"))
    }

    func testMainSurfacesUseNativeLiquidGlassBackgrounds() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let inspectorSource = try String(contentsOf: inspectorViewURL(), encoding: .utf8)
        let commandPaletteSource = try String(contentsOf: commandPaletteURL(), encoding: .utf8)
        let settingsSource = try String(contentsOf: settingsViewURL(), encoding: .utf8)

        for source in [contentSource, shellSource, sidebarSource, inspectorSource, commandPaletteSource, settingsSource] {
            XCTAssertTrue(source.contains("MomentoGlassBackground"))
            XCTAssertFalse(source.contains("MomentoVisualEffectView"))
            XCTAssertFalse(source.contains("Color(nsColor: .windowBackgroundColor)"))
            XCTAssertFalse(source.contains(".regularMaterial"))
            XCTAssertFalse(source.contains(".ultraThinMaterial"))
            XCTAssertFalse(source.contains(".thinMaterial"))
        }

        XCTAssertTrue(shellSource.contains("MomentoGlassBackground(cornerRadius: 0)"))
        XCTAssertTrue(sidebarSource.contains("MomentoGlassBackground(cornerRadius: 0)"))
        XCTAssertTrue(inspectorSource.contains("MomentoGlassBackground(cornerRadius: 0)"))
        XCTAssertTrue(settingsSource.contains("MomentoGlassBackground(cornerRadius: 0)"))
    }

    func testInteractiveBackgroundsDoNotUseManualOpacity() throws {
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let inspectorSource = try String(contentsOf: inspectorViewURL(), encoding: .utf8)
        let commandPaletteSource = try String(contentsOf: commandPaletteURL(), encoding: .utf8)

        for source in [sidebarSource, inspectorSource, commandPaletteSource] {
            XCTAssertTrue(source.contains(".glassEffect("))
            XCTAssertFalse(source.contains("Color.primary.opacity"))
            XCTAssertFalse(source.contains("Color.accentColor.opacity"))
            XCTAssertFalse(source.contains("Color.black.opacity"))
        }
    }

    func testAppKitCollectionUsesNativeGlassForSelectionBackgrounds() throws {
        let source = try String(contentsOf: assetCollectionURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("NSGlassEffectView"))
        XCTAssertTrue(source.contains("glassBackgroundView"))
        XCTAssertFalse(source.contains("withAlphaComponent"))
        XCTAssertFalse(source.contains("layer?.backgroundColor = NSColor.controlAccentColor"))
    }

    private func designSystemURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/DesignSystem/MomentoGlass.swift")
    }

    private func contentViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/ContentView.swift")
    }

    private func shellViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Shell/MomentoShellView.swift")
    }

    private func sidebarViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Sidebar/MomentoSidebarView.swift")
    }

    private func inspectorViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Inspector/MomentoInspectorView.swift")
    }

    private func commandPaletteURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/CommandPalette/MomentoCommandPalette.swift")
    }

    private func settingsViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Settings/MomentoSettingsView.swift")
    }

    private func assetCollectionURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/AssetCollectionGridView.swift")
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }
}
