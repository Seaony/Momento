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

        XCTAssertTrue(shellSource.contains("""
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
"""))
        XCTAssertFalse(shellSource.contains("HSplitView"))
        XCTAssertTrue(shellSource.contains("HStack(spacing: 0)"))
        XCTAssertTrue(shellSource.contains("trailingInspector"))
        XCTAssertTrue(shellSource.contains("if isInspectorPresented {"))
        XCTAssertFalse(shellSource.contains(".inspector(isPresented: $isInspectorPresented)"))
        XCTAssertFalse(shellSource.contains(".inspectorColumnWidth("))
        XCTAssertFalse(contentSource.contains("            .background {\n                MomentoGlassBackground(cornerRadius: 0)\n            }\n"))
        XCTAssertFalse(shellSource.contains("                    .background {\n                        MomentoGlassBackground(cornerRadius: 0)\n                    }\n"))
        XCTAssertFalse(inspectorSource.contains("        .background {\n            MomentoGlassBackground(cornerRadius: 0)\n                .ignoresSafeArea()\n        }\n"))
        XCTAssertTrue(sidebarSource.contains("MomentoGlassBackground(cornerRadius: MomentoTheme.floatingSidebarRadius)"))
        XCTAssertTrue(settingsSource.contains("MomentoGlassBackground(cornerRadius: 0)"))
    }

    func testWindowBackingOpacityControlsSingleGlobalMainGlassSurface() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let inspectorSource = try String(contentsOf: inspectorViewURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(shellSource.contains("""
        .background {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()
        }
"""))
        XCTAssertFalse(shellSource.contains("HSplitView"))
        XCTAssertTrue(shellSource.contains("HStack(spacing: 0)"))
        XCTAssertTrue(shellSource.contains("trailingInspector"))
        XCTAssertFalse(shellSource.contains(".inspector(isPresented: $isInspectorPresented)"))
        XCTAssertFalse(contentSource.contains("            .background {\n                MomentoGlassBackground(cornerRadius: 0)\n            }\n"))
        XCTAssertFalse(shellSource.contains("                    .background {\n                        MomentoGlassBackground(cornerRadius: 0)\n                    }\n"))
        XCTAssertFalse(inspectorSource.contains("        .background {\n            MomentoGlassBackground(cornerRadius: 0)\n                .ignoresSafeArea()\n        }\n"))
        XCTAssertTrue(windowSource.contains("static let backingOpacity: CGFloat ="))
        XCTAssertTrue(windowSource.contains("NSColor.windowBackgroundColor"))
        XCTAssertTrue(windowSource.contains(".withAlphaComponent(Self.backingOpacity)"))
        XCTAssertFalse(windowSource.contains("window.backgroundColor = .clear"))
    }

    func testMainAppKitGridDoesNotDrawOpaqueBackgrounds() throws {
        let source = try String(contentsOf: assetCollectionURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("collectionView.backgroundColors = [.clear]"))
        XCTAssertTrue(source.contains("scrollView.drawsBackground = false"))
    }

    func testSidebarUsesFloatingLiquidGlassPanel() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let designSource = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(shellSource.contains("floatingSidebar"))
        XCTAssertTrue(designSource.contains("static let floatingSidebarInset: CGFloat = 6"))
        XCTAssertTrue(shellSource.contains(".padding(.leading, MomentoTheme.floatingSidebarInset)"))
        XCTAssertTrue(shellSource.contains(".padding(.trailing, MomentoTheme.floatingSidebarInset)"))
        XCTAssertTrue(shellSource.contains(".padding(.vertical, MomentoTheme.floatingSidebarInset)"))
        XCTAssertTrue(sidebarSource.contains("MomentoGlassBackground(cornerRadius: MomentoTheme.floatingSidebarRadius)"))
        XCTAssertTrue(sidebarSource.contains("RoundedRectangle(cornerRadius: MomentoTheme.floatingSidebarRadius"))
        XCTAssertTrue(sidebarSource.contains("private var sidebarShape: RoundedRectangle"))
        XCTAssertTrue(sidebarSource.contains(".clipShape(sidebarShape)"))
        XCTAssertTrue(sidebarSource.contains("sidebarShape.strokeBorder"))
        XCTAssertFalse(sidebarSource.contains(".ignoresSafeArea()"))
    }

    func testSidebarFooterUsesInsetHairlineAndIconOnlyActions() throws {
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)

        XCTAssertTrue(sidebarSource.contains("sidebarBottomSeparator"))
        XCTAssertTrue(sidebarSource.contains("MomentoTheme.subtleStroke.opacity(0.24)"))
        XCTAssertTrue(sidebarSource.contains(".frame(height: 0.5)"))
        XCTAssertTrue(sidebarSource.contains(".padding(.horizontal, 14)"))
        XCTAssertTrue(sidebarSource.contains("bottomActionBar"))
        XCTAssertTrue(sidebarSource.contains("systemImage: \"trash\""))
        XCTAssertTrue(sidebarSource.contains("systemImage: \"gearshape\""))
        XCTAssertTrue(sidebarSource.contains("systemImage: \"questionmark.circle\""))
        XCTAssertFalse(sidebarSource.contains("systemImage: \"externaldrive\""))
    }

    func testFloatingSidebarWidthIsUserResizableWithoutSplitViewBorders() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(shellSource.contains("@State private var sidebarWidth = MomentoTheme.sidebarWidth"))
        XCTAssertTrue(shellSource.contains("@State private var sidebarResizeStartWidth: CGFloat?"))
        XCTAssertTrue(shellSource.contains("sidebarResizeHandle"))
        XCTAssertTrue(shellSource.contains(".overlay(alignment: .trailing)"))
        XCTAssertTrue(shellSource.contains("DragGesture(minimumDistance: 0, coordinateSpace: .global)"))
        XCTAssertTrue(shellSource.contains("value.translation.width"))
        XCTAssertTrue(shellSource.contains(".clamped(to: MomentoTheme.sidebarMinWidth...MomentoTheme.sidebarMaxWidth)"))
        XCTAssertTrue(shellSource.contains(".frame(width: sidebarWidth)"))
        XCTAssertTrue(shellSource.contains(".frame(width: 14)"))
        XCTAssertTrue(shellSource.contains(".pointerStyle(.columnResize(directions: .all))"))
        XCTAssertFalse(shellSource.contains("HSplitView"))
    }

    func testFloatingSidebarExtendsIntoWindowTitlebarArea() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let designSource = try String(contentsOf: designSystemURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(shellSource.contains(".ignoresSafeArea(.container, edges: .top)"))
        XCTAssertTrue(sidebarSource.contains(".padding(.top, MomentoTheme.floatingSidebarTitlebarContentInset)"))
        XCTAssertTrue(designSource.contains("static let floatingSidebarTitlebarContentInset"))
        XCTAssertTrue(windowSource.contains("window.styleMask.insert(.fullSizeContentView)"))
        XCTAssertTrue(windowSource.contains("window.titlebarAppearsTransparent = true"))
        XCTAssertTrue(windowSource.contains("window.titleVisibility = .hidden"))
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

    private func windowTransparencyURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/WindowTransparencyConfigurator.swift")
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }
}
