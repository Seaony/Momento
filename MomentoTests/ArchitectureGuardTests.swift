import Foundation
import XCTest

final class ArchitectureGuardTests: XCTestCase {
    func testGlassBackgroundUsesNativeSwiftUIGlassEffect() throws {
        let source = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".glassEffect(glass, in: RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))"))
        XCTAssertFalse(source.contains("MomentoVisualEffectView"))
        XCTAssertFalse(source.contains("NSVisualEffectView"))
    }

    func testWindowToolbarStaysTransparentWithoutHidingControls() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
        XCTAssertTrue(contentSource.contains(".toolbarBackgroundVisibility(.hidden, for: .windowToolbar)"))
        XCTAssertFalse(contentSource.contains(".toolbarVisibility("))
        XCTAssertTrue(windowSource.contains("window.titlebarAppearsTransparent = true"))
        XCTAssertTrue(windowSource.contains("window.titleVisibility = .hidden"))
    }

    func testMainWindowMinimumSizeUsesFixedSidebarsAndFlexibleContent() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let themeSource = try String(contentsOf: designSystemURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellURL(), encoding: .utf8)

        XCTAssertTrue(themeSource.contains("static let mainWindowMinWidth: CGFloat = 1100"))
        XCTAssertTrue(themeSource.contains("static let defaultWindowWidth: CGFloat = 1100"))
        XCTAssertTrue(contentSource.contains(".frame(minWidth: MomentoTheme.mainWindowMinWidth, minHeight: MomentoTheme.mainWindowMinHeight)"))
        XCTAssertTrue(appSource.contains(".defaultSize(width: MomentoTheme.defaultWindowWidth, height: MomentoTheme.defaultWindowHeight)"))
        XCTAssertTrue(appSource.contains(".windowResizability(.contentMinSize)"))
        XCTAssertTrue(shellSource.contains(".frame(width: MomentoTheme.inspectorWidth)"))
        XCTAssertTrue(shellSource.contains("MomentoTheme.sidebarMinWidth...MomentoTheme.sidebarMaxWidth"))
        XCTAssertFalse(shellSource.contains("effectiveContentMinWidth"))
        XCTAssertFalse(shellSource.contains("inspectorResizeHandle()"))
    }

    func testToolbarSearchUsesFlexibleWidthSoItCompressesInsteadOfCollapsing() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("ToolbarItemGroup(placement: .automatic)"))
        XCTAssertTrue(source.contains("static let searchControlMinWidth"))
        XCTAssertTrue(source.contains("static let searchControlMaxWidth"))
        XCTAssertTrue(source.contains("minWidth: ContentToolbarMetrics.searchControlMinWidth"))
        XCTAssertTrue(source.contains("maxWidth: ContentToolbarMetrics.searchControlMaxWidth"))
        // 搜索框不能再写死宽度或对水平方向 fixedSize，否则窄窗口下整组会被收进工具栏溢出菜单。
        XCTAssertFalse(source.contains("static let searchControlWidth"))
        XCTAssertFalse(source.contains(".fixedSize(horizontal: true"))
        XCTAssertFalse(source.contains("toolbarControlCluster"))
        XCTAssertFalse(source.contains("static let controlClusterWidth"))
    }

    func testContentViewValidatesCurrentLibraryWhenWindowAppears() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("store.validateCurrentLibraryAvailability()"))
    }

    func testRecentLibraryMutationsUseSecurityScopedAccess() throws {
        let source = try String(contentsOf: libraryStoreURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("private func withSecurityScopedRecentLibraryURL"))
        XCTAssertTrue(source.contains("let accessScope = LibraryAccessScope(url: resolved.url)"))
        XCTAssertTrue(source.contains("withExtendedLifetime(accessScope)"))
        XCTAssertTrue(source.contains("let library = try withSecurityScopedRecentLibraryURL(reference) { resolvedURL in"))
        XCTAssertTrue(source.contains("try storage.renameLibraryPackage(at: resolvedURL, to: trimmedName)"))
        XCTAssertTrue(source.contains("try withSecurityScopedRecentLibraryURL(reference) { resolvedURL in"))
        XCTAssertTrue(source.contains("try storage.deleteLibraryPackage(at: resolvedURL)"))
    }

    func testCustomDialogsDoNotHideToolbarChrome() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("showsChromeControls: !isModalOverlayVisible"))
        XCTAssertFalse(source.contains("if !isModalOverlayVisible {"))
        XCTAssertFalse(source.contains(".blur(radius: isModalOverlayVisible"))
    }

    func testInspectorDoesNotExposeNotesEditor() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellURL(), encoding: .utf8)
        let inspectorSource = try String(contentsOf: inspectorURL(), encoding: .utf8)

        XCTAssertFalse(contentSource.contains("inspectorNotesByAssetID"))
        XCTAssertFalse(contentSource.contains("private var inspectorNotes"))
        XCTAssertFalse(shellSource.contains("inspectorNotes"))
        XCTAssertFalse(inspectorSource.contains("notesEditor"))
        XCTAssertFalse(inspectorSource.contains("TextEditor(text: $notes)"))
        XCTAssertFalse(inspectorSource.contains("localization.string(\"Notes\")"))
    }

    func testAssetGridSupportsDraggingAndFilePromises() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(assetGridSource.contains("collectionView(_ collectionView: NSCollectionView, canDragItemsAt indexPaths: Set<IndexPath>, with event: NSEvent)"))
        XCTAssertTrue(assetGridSource.contains("collectionView(_ collectionView: NSCollectionView, pasteboardWriterForItemAt indexPath: IndexPath)"))
        XCTAssertTrue(bridgeSource.contains("NSFilePromiseProvider"))
        XCTAssertTrue(bridgeSource.contains("com.seaony.momento.asset-ids"))
    }

    func testAssetDragWriterIsFilePromiseProviderItself() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(bridgeSource.contains("final class AssetFilePromiseProvider: NSFilePromiseProvider, NSFilePromiseProviderDelegate"))
        XCTAssertTrue(assetGridSource.contains("return AssetFilePromiseProvider("))
        XCTAssertFalse(bridgeSource.contains("final class AssetDragPasteboardItem"))
    }

    func testInternalAssetDragUTTypeIsDeclaredInInfoPlist() throws {
        let data = try Data(contentsOf: infoPlistURL())
        let plist = try XCTUnwrap(PropertyListSerialization.propertyList(from: data, options: [], format: nil) as? [String: Any])
        let exportedTypes = try XCTUnwrap(plist["UTExportedTypeDeclarations"] as? [[String: Any]])
        let matchingType: [String: Any]? = exportedTypes.first { type in
            type["UTTypeIdentifier"] as? String == "com.seaony.momento.asset-ids"
        }
        let assetIDsType = try XCTUnwrap(matchingType)

        XCTAssertEqual(assetIDsType["UTTypeDescription"] as? String, "Momento Asset Drag Payload")
        XCTAssertEqual(assetIDsType["UTTypeConformsTo"] as? [String], ["public.json"])
    }

    func testSidebarAcceptsInternalAssetDropsForOrganization() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarURL(), encoding: .utf8)

        XCTAssertTrue(sidebarSource.contains("MomentoSidebarAssetDropDelegate"))
        XCTAssertTrue(sidebarSource.contains("AssetDragPasteboardWriter.assetIDsUTType"))
        XCTAssertTrue(sidebarSource.contains("onAssignDroppedAssetsToFolder(assetIDs, folder.id)"))
        XCTAssertTrue(sidebarSource.contains("onAssignDroppedAssetsToTag(assetIDs, tag.id)"))
        XCTAssertTrue(contentSource.contains("try store.assignAssets(ids: assetIDs, to: folderID)"))
        XCTAssertTrue(contentSource.contains("try store.addTag(id: tagID, toAssets: assetIDs)"))
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }

    private func appURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/MomentoApp.swift")
    }

    private func contentViewURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/ContentView.swift")
    }

    private func infoPlistURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Info.plist")
    }

    private func designSystemURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/DesignSystem/MomentoGlass.swift")
    }

    private func shellURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Shell/MomentoShellView.swift")
    }

    private func libraryStoreURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Core/LibraryStore.swift")
    }

    private func inspectorURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Inspector/MomentoInspectorView.swift")
    }

    private func assetGridURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/AssetCollectionGridView.swift")
    }

    private func sidebarURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Sidebar/MomentoSidebarView.swift")
    }

    private func appKitBridgeSource() throws -> String {
        let bridgeURL = repositoryRoot().appendingPathComponent("Momento/AppKitBridge", isDirectory: true)
        let urls = try FileManager.default.contentsOfDirectory(
            at: bridgeURL,
            includingPropertiesForKeys: nil
        )

        return try urls
            .filter { $0.pathExtension == "swift" }
            .sorted { $0.lastPathComponent < $1.lastPathComponent }
            .map { try String(contentsOf: $0, encoding: .utf8) }
            .joined(separator: "\n")
    }

    private func windowTransparencyURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/WindowTransparencyConfigurator.swift")
    }
}
