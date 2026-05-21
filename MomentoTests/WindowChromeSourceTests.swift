import Foundation
import XCTest

final class WindowChromeSourceTests: XCTestCase {
    func testContentViewDoesNotOverrideSystemWindowCornerRadius() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("MomentoWindowCornerRadiusConfigurator"))
        XCTAssertFalse(source.contains("momentoWindowCornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerCurve"))
        XCTAssertFalse(source.contains("contentView.layer?.masksToBounds"))
        XCTAssertFalse(source.contains("import QuartzCore"))
    }

    func testMainWindowUsesSystemToolbarChromeForLargeTahoeCorners() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
        XCTAssertFalse(appSource.contains(".windowToolbarStyle(.unifiedCompact)"))

        XCTAssertTrue(contentSource.contains(".toolbar {"))
        XCTAssertFalse(contentSource.contains("ToolbarItemGroup(placement: .navigation)"))
        XCTAssertFalse(contentSource.contains("ToolbarItem(placement: .principal)"))
        XCTAssertTrue(contentSource.contains(".libraryToolbarSearch(isVisible: !isLibraryDialogVisible"))
        XCTAssertFalse(shellSource.contains("MomentoTopBar("))
    }

    func testMainToolbarDoesNotShowAllAssetsTitleOrAdjacentImportButton() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(contentSource.contains("MomentoToolbarTitle("))
        XCTAssertFalse(contentSource.contains("private struct MomentoToolbarTitle"))
        XCTAssertFalse(contentSource.contains("ToolbarItemGroup(placement: .navigation)"))
        XCTAssertFalse(contentSource.contains("case .library:\n            localization.string(\"All Assets\")"))
    }

    func testLibraryWindowHidesTitleTextButKeepsToolbarControls() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains(".navigationTitle(\"\")"))
        XCTAssertFalse(contentSource.contains(".navigationTitle(title)"))
        XCTAssertTrue(windowSource.contains("window.titleVisibility = .hidden"))
        XCTAssertFalse(contentSource.contains(".toolbarVisibility("))
    }

    func testInspectorUsesCustomTransparentTrailingColumnAndToolbarToggle() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("@State private var isInspectorPresented = true"))
        XCTAssertTrue(contentSource.contains("isInspectorPresented: $isInspectorPresented"))
        XCTAssertTrue(contentSource.contains("Toggle(isOn: $isInspectorPresented)"))
        XCTAssertTrue(contentSource.contains("Label(localization.string(\"Toggle Inspector\"), systemImage: \"sidebar.right\")"))
        XCTAssertTrue(contentSource.contains("MomentoCommand(id: \"toggle-inspector\""))
        XCTAssertTrue(contentSource.contains("case \"toggle-inspector\":"))
        XCTAssertTrue(contentSource.contains("isInspectorPresented.toggle()"))

        XCTAssertTrue(shellSource.contains("@Binding var isInspectorPresented: Bool"))
        XCTAssertTrue(shellSource.contains("trailingInspector"))
        XCTAssertTrue(shellSource.contains("if isInspectorPresented {"))
        XCTAssertFalse(shellSource.contains(".inspector(isPresented: $isInspectorPresented)"))
        XCTAssertFalse(shellSource.contains(".inspectorColumnWidth("))
        XCTAssertFalse(shellSource.contains("HSplitView"))
    }

    func testWindowToolbarBackgroundIsTransparentWithoutHidingControls() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        // Hiding the window toolbar in the `.unified` style collapses the
        // title bar and takes the close/minimize/zoom controls with it. Keep
        // the controls and toolbar items, but let content show through the
        // toolbar background across the app.
        XCTAssertTrue(contentSource.contains(".toolbarBackgroundVisibility(.hidden, for: .windowToolbar)"))
        XCTAssertFalse(contentSource.contains(".toolbarVisibility("))
        XCTAssertTrue(contentSource.contains(".toolbar {"))
        XCTAssertTrue(contentSource.contains("if !isLibraryDialogVisible {\n                ToolbarItemGroup(placement: .primaryAction)"))
        XCTAssertTrue(contentSource.contains(".libraryToolbarSearch(isVisible: !isLibraryDialogVisible"))
        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
    }

    func testSidebarLibraryMenuUsesCustomGlassLibrarySwitcher() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarViewURL(), encoding: .utf8)
        let themeSource = try String(contentsOf: themeURL(), encoding: .utf8)
        let switcherOverlayStart = try XCTUnwrap(sidebarSource.range(of: "    @ViewBuilder\n    private var librarySwitcherOverlay: some View {"))
        let switcherOverlayEnd = try XCTUnwrap(sidebarSource[switcherOverlayStart.lowerBound...].range(of: "    private var sidebarShape: RoundedRectangle"))
        let switcherOverlaySource = String(sidebarSource[switcherOverlayStart.lowerBound..<switcherOverlayEnd.lowerBound])
        let libraryMenuStart = try XCTUnwrap(sidebarSource.range(of: "    private var libraryMenu: some View {"))
        let libraryMenuEnd = try XCTUnwrap(sidebarSource[libraryMenuStart.lowerBound...].range(of: "    private var libraryMenuBackground: some View"))
        let libraryMenuSource = String(sidebarSource[libraryMenuStart.lowerBound..<libraryMenuEnd.lowerBound])
        let switcherMenuStart = try XCTUnwrap(sidebarSource.range(of: "private struct MomentoLibrarySwitcherMenu: View {"))
        let switcherMenuEnd = try XCTUnwrap(sidebarSource[switcherMenuStart.lowerBound...].range(of: "private struct MomentoSidebarRow: View {"))
        let switcherMenuSource = String(sidebarSource[switcherMenuStart.lowerBound..<switcherMenuEnd.lowerBound])

        XCTAssertTrue(sidebarSource.contains("@State private var isLibrarySwitcherPresented = false"))
        XCTAssertTrue(sidebarSource.contains("@State private var isLibraryMenuHovered = false"))
        XCTAssertTrue(libraryMenuSource.contains("Text(libraryName ?? localization.string(\"No library selected\"))"))
        XCTAssertTrue(libraryMenuSource.contains("HStack(spacing: 8)"))
        XCTAssertTrue(libraryMenuSource.contains(".font(.system(size: 13, weight: .semibold))"))
        XCTAssertTrue(libraryMenuSource.contains(".font(.system(size: 11, weight: .semibold))"))
        XCTAssertTrue(libraryMenuSource.contains(".frame(width: 20, height: 20)"))
        XCTAssertTrue(libraryMenuSource.contains(".background(.blue.gradient, in: RoundedRectangle(cornerRadius: 5, style: .continuous))"))
        XCTAssertTrue(libraryMenuSource.contains(".padding(.vertical, 3)"))
        XCTAssertTrue(libraryMenuSource.contains(".frame(maxWidth: .infinity, alignment: .leading)"))
        XCTAssertTrue(libraryMenuSource.contains(".background { libraryMenuBackground }"))
        XCTAssertTrue(libraryMenuSource.contains(".onHover { hovering in\n            updateLibraryMenuHover(hovering)"))
        XCTAssertTrue(sidebarSource.contains("private var libraryMenuBackground: some View"))
        XCTAssertTrue(sidebarSource.contains("isLibraryMenuHovered || isLibrarySwitcherPresented"))
        XCTAssertTrue(sidebarSource.contains("shape.fill(MomentoTheme.sidebarIconHoverBackground)"))
        XCTAssertFalse(libraryMenuSource.contains("Text(verbatim: \"Momento\")"))
        XCTAssertFalse(libraryMenuSource.contains("VStack(alignment: .leading"))
        XCTAssertTrue(sidebarSource.contains("MomentoLibrarySwitcherMenu("))
        XCTAssertTrue(sidebarSource.contains("MomentoGlassBackground(glass: .regular, cornerRadius: 14)"))
        XCTAssertTrue(switcherMenuSource.contains("ForEach(visibleLibraries) { library in"))
        XCTAssertFalse(sidebarSource.contains("systemName: \"circle.grid.2x3.fill\""))
        XCTAssertTrue(sidebarSource.contains("isSelected || isHovered ? MomentoTheme.primaryText : MomentoTheme.secondaryText"))
        XCTAssertTrue(sidebarSource.contains("systemName: \"checkmark\""))
        XCTAssertTrue(sidebarSource.contains("systemName: \"ellipsis\""))
        XCTAssertTrue(sidebarSource.contains("localization.string(\"Create Library\")"))
        XCTAssertTrue(sidebarSource.contains("localization.string(\"Open Other Library\")"))
        XCTAssertTrue(sidebarSource.contains("localization.string(\"Clear Cache and Reload\")"))
        XCTAssertTrue(sidebarSource.contains("localization.string(\"Edit Library\")"))
        XCTAssertTrue(sidebarSource.contains("onRenameLibrary(library.id)"))
        XCTAssertTrue(sidebarSource.contains("onDeleteLibrary(library.id)"))
        XCTAssertFalse(switcherMenuSource.contains(".disabled(true)"))
        XCTAssertFalse(switcherMenuSource.contains("Menu {"))
        XCTAssertTrue(switcherMenuSource.contains("@State private var activeMoreLibraryID: RecentLibraryReference.ID?"))
        XCTAssertTrue(switcherMenuSource.contains("private func libraryMoreMenu(_ library: RecentLibraryReference) -> some View"))
        XCTAssertTrue(switcherMenuSource.contains("MomentoGlassBackground(glass: .regular.tint(Color.white.opacity(0.08)), cornerRadius: 12)"))
        XCTAssertFalse(switcherMenuSource.contains("MomentoGlassBackground(glass: .regular, cornerRadius: 12)"))
        XCTAssertTrue(switcherOverlaySource.contains("LibrarySwitcherDismissMonitor(isPresented: $isLibrarySwitcherPresented)"))
        XCTAssertTrue(sidebarSource.contains("NSEvent.addLocalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown])"))
        XCTAssertFalse(sidebarSource.contains("private var libraryMenu: some View {\n        Menu {"))
        XCTAssertFalse(sidebarSource.contains("ZStack(alignment: .topLeading) {\n            sidebarPanel\n            librarySwitcherOverlay"))
        XCTAssertFalse(sidebarSource.contains(".frame(width: 380)"))
        XCTAssertTrue(sidebarSource.contains(".frame(width: MomentoTheme.librarySwitcherWidth, alignment: .topLeading)"))
        XCTAssertTrue(themeSource.contains("static let librarySwitcherWidth: CGFloat = 300"))
        XCTAssertTrue(sidebarSource.contains(".overlay(alignment: .topLeading) {\n            librarySwitcherOverlay\n        }"))
        XCTAssertTrue(switcherOverlaySource.contains(".transition(.opacity.combined(with: .scale(scale: 0.98, anchor: .topLeading)))"))
        XCTAssertFalse(switcherOverlaySource.contains(".transition(.opacity.combined(with: .move(edge: .top)))"))
        XCTAssertTrue(switcherMenuSource.contains("VStack(spacing: 2)"))
        XCTAssertTrue(switcherMenuSource.contains(".frame(height: 42)"))
        XCTAssertTrue(switcherMenuSource.contains(".frame(height: 30)"))
        XCTAssertTrue(switcherMenuSource.contains("Image(systemName: \"archivebox.fill\")"))
        XCTAssertTrue(switcherMenuSource.contains("private var menuPanel: some View"))
        XCTAssertTrue(switcherMenuSource.contains("libraryMoreMenu(activeLibrary)\n                    .offset(libraryMoreMenuOffset(for: activeIndex))"))
        XCTAssertTrue(switcherMenuSource.contains("private func libraryMoreMenuOffset(for index: Int) -> CGSize"))
        XCTAssertFalse(switcherMenuSource.contains(".overlay(alignment: .topLeading) {\n                if activeMoreLibraryID == library.id"))
        XCTAssertTrue(switcherMenuSource.contains("private func libraryDragHandle(isActive: Bool) -> some View"))
        XCTAssertTrue(switcherMenuSource.contains("@State private var displayedLibraries: [RecentLibraryReference] = []"))
        XCTAssertTrue(switcherMenuSource.contains("@State private var draggingLibraryID: RecentLibraryReference.ID?"))
        XCTAssertTrue(switcherMenuSource.contains("private var visibleLibraries: [RecentLibraryReference]"))
        XCTAssertTrue(switcherMenuSource.contains(".onAppear(perform: syncDisplayedLibraries)"))
        XCTAssertTrue(switcherMenuSource.contains(".onChange(of: recentLibraries) { _, libraries in"))
        XCTAssertTrue(switcherMenuSource.contains(".dragHandleCursor()"))
        XCTAssertTrue(switcherMenuSource.contains(".contentShape(.dragPreview, RoundedRectangle(cornerRadius: 10, style: .continuous))"))
        XCTAssertTrue(switcherMenuSource.contains(".onDrag {\n            draggingLibraryID = library.id"))
        XCTAssertTrue(switcherMenuSource.contains("NSItemProvider(object: library.id as NSString)"))
        XCTAssertTrue(switcherMenuSource.contains("private func libraryDragPreview(_ library: RecentLibraryReference) -> some View"))
        XCTAssertTrue(switcherMenuSource.contains("private struct LibraryReorderDropDelegate: DropDelegate"))
        XCTAssertTrue(switcherMenuSource.contains("@Binding var displayedLibraries: [RecentLibraryReference]"))
        XCTAssertTrue(sidebarSource.contains("addCursorRect(bounds, cursor: .openHand)"))
        XCTAssertTrue(switcherMenuSource.contains(".onDrop(of: [.plainText], delegate: LibraryReorderDropDelegate("))
        XCTAssertFalse(switcherMenuSource.contains(".onDrop(of: [.text]"))
        XCTAssertFalse(switcherMenuSource.contains(".dropDestination(for: RecentLibraryReference.ID.self)"))
        XCTAssertTrue(switcherMenuSource.contains("dropEntered(info: DropInfo) {\n        reorderIfNeeded(info: info)"))
        XCTAssertTrue(switcherMenuSource.contains("dropUpdated(info: DropInfo) -> DropProposal? {\n        reorderIfNeeded(info: info)"))
        XCTAssertTrue(switcherMenuSource.contains("private func reorderIfNeeded(info: DropInfo)"))
        XCTAssertTrue(switcherMenuSource.contains("displayedLibraries.remove(at: sourceIndex)"))
        XCTAssertTrue(switcherMenuSource.contains("displayedLibraries.insert(movedLibrary, at: min(insertionIndex, displayedLibraries.endIndex))"))
        XCTAssertTrue(switcherMenuSource.contains("onMoveLibrary(draggedID, targetLibrary.id, insertAfterTarget)"))
        XCTAssertTrue(switcherMenuSource.contains("DropProposal(operation: .move)"))
        XCTAssertTrue(switcherMenuSource.contains("CGSize(width: MomentoTheme.librarySwitcherWidth - 11, height: 15 + CGFloat(index) * 44)"))
        XCTAssertTrue(switcherMenuSource.contains("VStack(spacing: 2)"))
        XCTAssertTrue(switcherMenuSource.contains("HStack(spacing: 2)"))
        XCTAssertTrue(switcherMenuSource.contains(".frame(width: 2.4, height: 2.4)"))
        XCTAssertTrue(switcherMenuSource.contains("Circle()"))
        XCTAssertTrue(switcherMenuSource.contains(".font(.system(size: 11))"))
        XCTAssertTrue(switcherMenuSource.contains(".foregroundStyle(isSelected ? .white : MomentoTheme.primaryText)"))
        XCTAssertTrue(switcherMenuSource.contains("isSelected ? MomentoTheme.primaryText.opacity(0.82) : MomentoTheme.secondaryText"))
        XCTAssertTrue(switcherMenuSource.contains("if isSelected {"))
        XCTAssertTrue(switcherMenuSource.contains(".glassEffect(.regular.tint(Color.accentColor), in: shape)"))
        XCTAssertTrue(switcherMenuSource.contains("} else if isHovered {"))
        XCTAssertTrue(switcherMenuSource.contains("shape.fill(MomentoTheme.sidebarIconHoverBackground)"))
        XCTAssertTrue(contentSource.contains("onCreateLibrary: createLibrary"))
        XCTAssertTrue(contentSource.contains("onRenameLibrary: renameLibrary"))
        XCTAssertTrue(contentSource.contains("onDeleteLibrary: deleteLibrary"))
        XCTAssertTrue(contentSource.contains("onMoveLibrary: moveLibrary"))
        XCTAssertTrue(contentSource.contains("onReloadLibrary: reloadLibrary"))
        XCTAssertTrue(shellSource.contains("onCreateLibrary"))
        XCTAssertTrue(shellSource.contains("onRenameLibrary"))
        XCTAssertTrue(shellSource.contains("onDeleteLibrary"))
        XCTAssertTrue(shellSource.contains("onMoveLibrary"))
        XCTAssertTrue(shellSource.contains("onReloadLibrary"))
        XCTAssertTrue(contentSource.contains("store.closeCurrentLibrary()"))
    }

    func testContentViewValidatesCurrentLibraryWhenWindowAppears() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("store.validateCurrentLibraryAvailability()"))
    }

    private func contentViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/ContentView.swift")
    }

    private func appURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/MomentoApp.swift")
    }

    private func shellViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Shell/MomentoShellView.swift")
    }

    private func sidebarViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Sidebar/MomentoSidebarView.swift")
    }

    private func windowTransparencyURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/AppKitBridge/WindowTransparencyConfigurator.swift")
    }

    private func themeURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/DesignSystem/MomentoGlass.swift")
    }
}
