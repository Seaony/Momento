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
        XCTAssertTrue(designSource.contains("static let sidebarWidth: CGFloat = 280"))
        XCTAssertTrue(designSource.contains("static let floatingSidebarInset: CGFloat = 8"))
        XCTAssertTrue(designSource.contains("static let floatingSidebarRadius: CGFloat = 20"))
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
        let designSource = try String(contentsOf: designSystemURL(), encoding: .utf8)
        let footerStart = try XCTUnwrap(sidebarSource.range(of: "private var bottomActionBar: some View {"))
        let footerEnd = try XCTUnwrap(sidebarSource[footerStart.lowerBound...].range(of: "    private func sidebarFooterButton("))
        let footerSource = String(sidebarSource[footerStart.lowerBound..<footerEnd.lowerBound])
        let backgroundStart = try XCTUnwrap(sidebarSource.range(of: "    private func sidebarFooterIconBackground("))
        let backgroundEnd = try XCTUnwrap(sidebarSource[backgroundStart.lowerBound...].range(of: "    private func updateFooterHover("))
        let backgroundSource = String(sidebarSource[backgroundStart.lowerBound..<backgroundEnd.lowerBound])

        XCTAssertTrue(sidebarSource.contains("sidebarBottomSeparator"))
        XCTAssertTrue(sidebarSource.contains("MomentoTheme.subtleStroke.opacity(1)"))
        XCTAssertTrue(sidebarSource.contains(".frame(height: 0.5)"))
        XCTAssertTrue(sidebarSource.contains(".padding(.horizontal, 14)"))
        XCTAssertTrue(sidebarSource.contains("bottomActionBar"))
        XCTAssertTrue(footerSource.contains("HStack(spacing: 6)"))
        XCTAssertTrue(footerSource.contains(".frame(maxWidth: .infinity, alignment: .leading)"))
        XCTAssertFalse(footerSource.contains("Spacer()"))
        XCTAssertTrue(footerSource.contains("systemImage: \"trash\""))
        XCTAssertTrue(footerSource.contains("systemImage: \"gear\""))
        XCTAssertTrue(footerSource.contains("systemImage: \"questionmark.circle\""))
        XCTAssertFalse(sidebarSource.contains("systemImage: \"externaldrive\""))
        XCTAssertTrue(designSource.contains("static let primaryText = Color(nsColor: .labelColor)"))
        XCTAssertTrue(sidebarSource.contains("@State private var hoveredFooterActionID: String?"))
        XCTAssertTrue(footerSource.contains("isSelected: selection == \"trash\""))
        XCTAssertTrue(sidebarSource.contains("hoveredFooterActionID == id"))
        XCTAssertTrue(sidebarSource.contains("updateFooterHover(id: id, isHovering: hovering)"))
        XCTAssertTrue(sidebarSource.contains("MomentoTheme.primaryText"))
        XCTAssertTrue(sidebarSource.contains("sidebarFooterIconBackground(id: id, isSelected: isSelected)"))
        XCTAssertTrue(backgroundSource.contains("MomentoTheme.sidebarIconHoverBackground"))
        XCTAssertFalse(backgroundSource.contains(".glassEffect(.regular, in: shape)"))
        XCTAssertTrue(sidebarSource.contains(".pointerStyle(.link)"))
    }

    func testFloatingSidebarWidthIsUserResizableWithoutSplitViewBorders() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(shellSource.contains("@State private var sidebarWidth = MomentoTheme.sidebarWidth"))
        XCTAssertTrue(shellSource.contains("@State private var sidebarResizeStartWidth: CGFloat?"))
        XCTAssertTrue(shellSource.contains("sidebarResizeHandle"))
        XCTAssertTrue(shellSource.contains("private var shellContent: some View"))
        XCTAssertTrue(shellSource.contains(".frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .leading)"))
        XCTAssertTrue(shellSource.contains("floatingSidebar"))
        XCTAssertTrue(shellSource.contains("private var floatingSidebar: some View"))
        XCTAssertTrue(shellSource.contains("private func sidebarResizeHandle() -> some View"))
        XCTAssertTrue(shellSource.contains("private var effectiveSidebarWidth: CGFloat"))
        XCTAssertTrue(shellSource.contains("private var sidebarWidthRange: ClosedRange<CGFloat>"))
        XCTAssertTrue(shellSource.contains("MomentoTheme.sidebarMinWidth...MomentoTheme.sidebarMaxWidth"))
        XCTAssertFalse(shellSource.contains("availableWidth - MomentoTheme.contentMinWidth"))
        XCTAssertFalse(shellSource.contains("min(MomentoTheme.sidebarMaxWidth, max(MomentoTheme.sidebarMinWidth, availableSidebarWidth))"))
        XCTAssertTrue(shellSource.contains(".overlay(alignment: .trailing)"))
        XCTAssertTrue(shellSource.contains("DragGesture(minimumDistance: 0, coordinateSpace: .global)"))
        XCTAssertTrue(shellSource.contains("value.translation.width"))
        XCTAssertTrue(shellSource.contains(".clamped(to: widthRange)"))
        XCTAssertTrue(shellSource.contains(".frame(width: effectiveSidebarWidth)"))
        XCTAssertTrue(shellSource.contains(".fixedSize(horizontal: true, vertical: false)"))
        XCTAssertTrue(shellSource.contains(".layoutPriority(2)"))
        XCTAssertTrue(shellSource.contains(".frame(width: 14)"))
        XCTAssertTrue(shellSource.contains(".pointerStyle(.columnResize(directions: .all))"))
        XCTAssertFalse(shellSource.contains("HSplitView"))
    }

    func testDraggingSidebarBelowMinimumCollapsesFloatingSidebar() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let designSource = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(designSource.contains("static let sidebarCollapseDragOvershoot: CGFloat = 24"))
        XCTAssertTrue(shellSource.contains("let widthRange = sidebarWidthRange"))
        XCTAssertTrue(shellSource.contains("let proposedWidth = startWidth + value.translation.width"))
        XCTAssertTrue(shellSource.contains("if proposedWidth < widthRange.lowerBound - MomentoTheme.sidebarCollapseDragOvershoot {"))
        XCTAssertTrue(shellSource.contains("sidebarWidth = widthRange.lowerBound"))
        XCTAssertTrue(shellSource.contains("collapseSidebarFromResize()"))
        XCTAssertTrue(shellSource.contains("private func collapseSidebarFromResize()"))
        XCTAssertTrue(shellSource.contains("guard !isSidebarCollapsed else {"))
        XCTAssertTrue(shellSource.contains("sidebarResizeStartWidth = nil"))
        XCTAssertTrue(shellSource.contains("withAnimation(.smooth(duration: 0.18))"))
        XCTAssertTrue(shellSource.contains("isSidebarCollapsed = true"))
    }

    func testFloatingSidebarCanCollapseFromTitlebarButton() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let designSource = try String(contentsOf: designSystemURL(), encoding: .utf8)
        let titlebarSource = try String(contentsOf: titlebarToggleConfiguratorURL(), encoding: .utf8)

        XCTAssertFalse(designSource.contains("collapsedSidebarWidth"))
        XCTAssertTrue(designSource.contains("static let sidebarTitlebarButtonSize: CGFloat = 28"))
        XCTAssertTrue(designSource.contains("static let sidebarTitlebarButtonTopInset: CGFloat = 13"))
        XCTAssertTrue(designSource.contains("static let sidebarTitlebarButtonTrailingInset: CGFloat = 14"))
        XCTAssertTrue(designSource.contains("static let collapsedSidebarToggleLeadingInset: CGFloat = 92"))
        XCTAssertTrue(designSource.contains("static let sidebarIconHoverBackground = Color.white.opacity(0.08)"))
        XCTAssertTrue(shellSource.contains("@State private var isSidebarCollapsed = false"))
        XCTAssertTrue(shellSource.contains("if !isSidebarCollapsed {"))
        XCTAssertTrue(shellSource.contains("floatingSidebar"))
        XCTAssertTrue(shellSource.contains(".transition(.move(edge: .leading).combined(with: .opacity))"))
        XCTAssertTrue(shellSource.contains("SidebarTitlebarToggleConfigurator("))
        XCTAssertTrue(shellSource.contains("isCollapsed: $isSidebarCollapsed"))
        XCTAssertTrue(shellSource.contains("showsChromeControls: Bool = true"))
        XCTAssertTrue(shellSource.contains("isVisible: showsChromeControls"))
        XCTAssertTrue(shellSource.contains("buttonMinX: sidebarToggleButtonMinX"))
        XCTAssertTrue(shellSource.contains("label: sidebarToggleLabel"))
        XCTAssertTrue(shellSource.contains("private var sidebarToggleLabel: String"))
        XCTAssertTrue(shellSource.contains("private var sidebarToggleButtonMinX: CGFloat"))
        XCTAssertTrue(shellSource.contains("return MomentoTheme.collapsedSidebarToggleLeadingInset"))
        XCTAssertTrue(shellSource.contains("return MomentoTheme.floatingSidebarInset + effectiveSidebarWidth - MomentoTheme.sidebarTitlebarButtonTrailingInset - MomentoTheme.sidebarTitlebarButtonSize"))
        XCTAssertTrue(shellSource.contains(".animation(.smooth(duration: 0.18), value: isSidebarCollapsed)"))
        XCTAssertFalse(shellSource.contains("@State private var isSidebarToggleHovered = false"))
        XCTAssertFalse(shellSource.contains("private func sidebarTitlebarControls(availableWidth: CGFloat) -> some View"))
        XCTAssertFalse(shellSource.contains("private var sidebarToggleButton: some View"))
        XCTAssertFalse(shellSource.contains("isSidebarCollapsed ? MomentoTheme.collapsedSidebarWidth : sidebarWidth"))
        XCTAssertTrue(titlebarSource.contains("private struct SidebarTitlebarToggleAccessoryView: View"))
        XCTAssertTrue(titlebarSource.contains("var isVisible: Bool"))
        XCTAssertTrue(titlebarSource.contains("guard configuration.isVisible else"))
        XCTAssertTrue(titlebarSource.contains("accessoryController?.isHidden = !configuration.isVisible"))
        XCTAssertTrue(titlebarSource.contains("MomentoTheme.sidebarIconHoverBackground"))
        XCTAssertTrue(titlebarSource.contains("isHovered ? MomentoTheme.primaryText : MomentoTheme.secondaryText"))
        XCTAssertTrue(titlebarSource.contains(".onHover { hovering in"))
        XCTAssertTrue(titlebarSource.contains("isHovered = hovering"))
        XCTAssertTrue(titlebarSource.contains(".contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))"))
        XCTAssertTrue(titlebarSource.contains("withAnimation(.smooth(duration: 0.18))"))
        XCTAssertTrue(titlebarSource.contains("isCollapsed.toggle()"))

        XCTAssertFalse(sidebarSource.contains("var isCollapsed: Bool"))
        XCTAssertFalse(sidebarSource.contains("var onToggleCollapsed: () -> Void"))
        XCTAssertFalse(sidebarSource.contains("sidebarCollapseButton"))
        XCTAssertFalse(sidebarSource.contains("expandedSidebarContent"))
    }

    func testSidebarTitlebarToggleUsesNativeTitlebarAccessoryForHitTesting() throws {
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let titlebarSource = try String(contentsOf: titlebarToggleConfiguratorURL(), encoding: .utf8)

        XCTAssertTrue(shellSource.contains("SidebarTitlebarToggleConfigurator("))
        XCTAssertFalse(shellSource.contains("private func sidebarTitlebarControls(availableWidth: CGFloat) -> some View"))
        XCTAssertTrue(titlebarSource.contains("NSTitlebarAccessoryViewController"))
        XCTAssertTrue(titlebarSource.contains("window.addTitlebarAccessoryViewController"))
        XCTAssertTrue(titlebarSource.contains("layoutAttribute = .left"))
        XCTAssertTrue(titlebarSource.contains("NSHostingView"))
        XCTAssertTrue(titlebarSource.contains("@Binding var isCollapsed: Bool"))
        XCTAssertTrue(titlebarSource.contains("isCollapsed.toggle()"))
    }

    func testSidebarTitlebarTogglePositionAccountsForAccessoryOrigin() throws {
        let titlebarSource = try String(contentsOf: titlebarToggleConfiguratorURL(), encoding: .utf8)

        XCTAssertTrue(titlebarSource.contains("var buttonMinX: CGFloat"))
        XCTAssertTrue(titlebarSource.contains("convert(.zero, to: nil).x"))
        XCTAssertTrue(titlebarSource.contains("buttonMinX - titlebarOriginX"))
        XCTAssertTrue(titlebarSource.contains("SidebarTitlebarToggleContainerView"))
        XCTAssertTrue(titlebarSource.contains("override func hitTest(_ point: NSPoint) -> NSView?"))
        XCTAssertFalse(titlebarSource.contains(".padding(.leading, leadingInset)"))
    }

    func testSidebarTitlebarToggleUsesAppKitCursorRect() throws {
        let titlebarSource = try String(contentsOf: titlebarToggleConfiguratorURL(), encoding: .utf8)

        XCTAssertTrue(titlebarSource.contains("override func resetCursorRects()"))
        XCTAssertTrue(titlebarSource.contains("addCursorRect(bounds, cursor: .pointingHand)"))
        XCTAssertTrue(titlebarSource.contains("window?.invalidateCursorRects(for: hostingView)"))
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

    private func titlebarToggleConfiguratorURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/SidebarTitlebarToggleConfigurator.swift")
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }
}
