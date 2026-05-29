// 中文注释：本测试用粗粒度源码护栏保护容易回归的架构约束和平台集成点。
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
        let mainWindowSceneSource = try sourceBlock(in: appSource, from: "WindowGroup {", to: "Settings {")
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let windowSource = try String(contentsOf: windowTransparencyURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(mainWindowSceneSource.contains(".windowStyle(.hiddenTitleBar)"))
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

    func testTitlebarImportAndExtensionButtonsUseFullHitTargets() throws {
        let source = try String(contentsOf: sidebarTitlebarURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let actionButtonStart = try XCTUnwrap(source.range(of: "private func titlebarActionButton("))
        let actionButtonEnd = try XCTUnwrap(source.range(of: "private var titlebarControlsWidth", range: actionButtonStart.upperBound..<source.endIndex))
        let actionButtonSource = String(source[actionButtonStart.lowerBound..<actionButtonEnd.lowerBound])

        XCTAssertTrue(source.contains("browserExtensionAction"))
        XCTAssertTrue(source.contains("systemImage: \"square.and.arrow.down\""))
        XCTAssertTrue(source.contains("hoverID: .importAssets"))
        XCTAssertTrue(source.contains("systemImage: \"backpack\""))
        XCTAssertFalse(source.contains("systemImage: \"puzzlepiece.extension\""))
        XCTAssertTrue(source.contains("hoverID: .browserExtension"))
        XCTAssertTrue(actionButtonSource.contains(".frame(width: MomentoTheme.titlebarControlHitSize, height: MomentoTheme.titlebarControlHitSize)"))
        XCTAssertTrue(actionButtonSource.contains(".contentShape(.interaction, Rectangle())"))
        XCTAssertTrue(actionButtonSource.contains("shape.fill(MomentoTheme.sidebarIconHoverBackground)"))
        XCTAssertTrue(actionButtonSource.contains(".offset(y: MomentoTheme.sidebarTitlebarButtonHitInset)"))
        XCTAssertTrue(shellSource.contains("browserExtensionAction: onInstallBrowserExtension"))
        XCTAssertTrue(contentSource.contains("onInstallBrowserExtension: installBrowserExtension"))
    }

    func testToolbarControlsUseNativeHelpTooltips() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let sidebarTitlebarSource = try String(contentsOf: sidebarTitlebarURL(), encoding: .utf8)
        let inspectorTitlebarSource = try String(contentsOf: inspectorTitlebarURL(), encoding: .utf8)
        let updateButtonSource = try sourceBlock(
            in: contentSource,
            from: "private var toolbarUpdateButton",
            to: "private var isTagManagementSelected"
        )
        let viewModeSource = try sourceBlock(
            in: contentSource,
            from: "private var toolbarViewModeSwitcher",
            to: "private func toolbarSearchControl"
        )
        let toolbarIconButtonSource = try sourceBlock(
            in: contentSource,
            from: "private func toolbarIconButton(",
            to: "private var filterPopover"
        )

        XCTAssertTrue(updateButtonSource.contains(".help("))
        XCTAssertTrue(viewModeSource.contains(".help("))
        XCTAssertTrue(toolbarIconButtonSource.contains(".help(label)"))
        XCTAssertTrue(sidebarTitlebarSource.contains(".help(label)"))
        XCTAssertTrue(inspectorTitlebarSource.contains(".help(label)"))
        XCTAssertFalse(contentSource.contains(".momentoTooltip("))
        XCTAssertFalse(sidebarTitlebarSource.contains(".momentoTitlebarTooltip("))
        XCTAssertFalse(inspectorTitlebarSource.contains(".momentoTitlebarTooltip("))
        XCTAssertFalse(FileManager.default.fileExists(atPath: titlebarTooltipURL().path))
    }

    func testToolbarSearchControlDoesNotShowTooltip() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let searchStart = try XCTUnwrap(source.range(of: "private func toolbarSearchControl(resultCount: Int) -> some View"))
        let searchEnd = try XCTUnwrap(source.range(of: "private var toolbarFilterButton", range: searchStart.upperBound..<source.endIndex))
        let searchSource = String(source[searchStart.lowerBound..<searchEnd.lowerBound])

        XCTAssertFalse(source.contains("isToolbarSearchHovered"))
        XCTAssertFalse(searchSource.contains(".momentoTooltip("))
    }

    func testEmptyGridBrowserExtensionButtonUsesVisibleGlassAppearance() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let emptyStateSource = try sourceBlock(
            in: source,
            from: "private var emptyGridState",
            to: "private func importErrorBanner"
        )
        let buttonRowStart = try XCTUnwrap(emptyStateSource.range(of: "HStack(spacing: 12)"))
        let buttonRowEnd = try XCTUnwrap(emptyStateSource.range(of: ".padding(.top, 30)", range: buttonRowStart.upperBound..<emptyStateSource.endIndex))
        let buttonSource = String(emptyStateSource[buttonRowStart.lowerBound..<buttonRowEnd.upperBound])

        XCTAssertTrue(buttonSource.contains("Label(localization.string(\"Import Assets\"), systemImage: \"square.and.arrow.down\")"))
        XCTAssertTrue(buttonSource.contains("Label(localization.string(\"Install Browser Extension\"), systemImage: \"backpack\")"))
        XCTAssertEqual(buttonSource.components(separatedBy: ".frame(height: 38)").count - 1, 2)
        XCTAssertFalse(buttonSource.contains(".padding(.top, 12)"))
        XCTAssertFalse(buttonSource.contains("puzzlepiece.extension"))
        XCTAssertTrue(buttonSource.contains(".buttonStyle(.glass)"))
        XCTAssertTrue(buttonSource.contains(".foregroundStyle(MomentoTheme.primaryText)"))
        XCTAssertTrue(buttonSource.contains(".environment(\\.appearsActive, true)"))
    }

    func testSettingsWindowUsesTransparentGlassChromeAndControls() throws {
        let source = try String(contentsOf: settingsURL(), encoding: .utf8)
        let settingsRowSource = try sourceBlock(
            in: source,
            from: "private func settingsRow",
            to: "private var languagePicker"
        )
        let languagePickerSource = try sourceBlock(
            in: source,
            from: "private var languagePicker",
            to: "private var appearancePicker"
        )
        let appearancePickerSource = try sourceBlock(
            in: source,
            from: "private var appearancePicker",
            to: "// MARK: - 检查更新"
        )

        XCTAssertTrue(source.contains("WindowTransparencyConfigurator(fixedContentSize: Self.preferredSize)"))
        XCTAssertTrue(source.contains(".toolbarBackgroundVisibility(.hidden, for: .windowToolbar)"))
        XCTAssertTrue(source.contains("MomentoGlassBackground(cornerRadius: 0)"))
        XCTAssertTrue(source.contains(".buttonStyle(.glass)"))
        XCTAssertTrue(settingsRowSource.contains(".frame(width: MomentoSettingsMetrics.rowControlSlotWidth, alignment: .trailing)"))
        XCTAssertTrue(settingsRowSource.contains(".frame(maxWidth: .infinity)"))
        XCTAssertTrue(languagePickerSource.contains(".pickerStyle(.menu)"))
        XCTAssertFalse(languagePickerSource.contains(".frame(width: MomentoSettingsMetrics.pickerWidth)"))
        XCTAssertTrue(appearancePickerSource.contains("HStack(spacing: MomentoSettingsMetrics.appearancePickerSpacing)"))
        XCTAssertTrue(appearancePickerSource.contains("appearanceOptionButton(for: appearance)"))
        XCTAssertTrue(appearancePickerSource.contains("cornerRadius: MomentoSettingsMetrics.appearancePickerRadius"))
        XCTAssertTrue(appearancePickerSource.contains("cornerRadius: MomentoSettingsMetrics.appearancePickerSegmentRadius"))
        XCTAssertTrue(appearancePickerSource.contains("shape.fill(Color.accentColor)"))
        XCTAssertFalse(appearancePickerSource.contains("Picker(\"\", selection: $appAppearance)"))
        XCTAssertFalse(appearancePickerSource.contains(".pickerStyle(.segmented)"))
        XCTAssertFalse(appearancePickerSource.contains(".pickerStyle(.menu)"))
        XCTAssertFalse(appearancePickerSource.contains("Text(localization.title(for: appearance))"))
        XCTAssertTrue(appearancePickerSource.contains(".help(localization.title(for: appearance))"))
        XCTAssertTrue(source.contains(".environment(\\.appearsActive, true)"))
        XCTAssertFalse(source.contains(".buttonStyle(.glass(.clear))"))
        XCTAssertFalse(source.contains("Form("))
    }

    func testAppAppearancePreferenceDoesNotForceDarkMode() throws {
        let source = try String(contentsOf: appURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("import AppKit"))
        XCTAssertTrue(source.contains("@AppStorage(AppSettingsKeys.appAppearance)"))
        XCTAssertTrue(source.contains("Self.applyAppearance(AppSettings.appAppearance())"))
        XCTAssertTrue(source.contains("NSApplication.shared.appearance = appearance.appKitAppearanceName.flatMap(NSAppearance.init(named:))"))
        XCTAssertEqual(source.components(separatedBy: ".preferredColorScheme(appearance.colorScheme)").count - 1, 2)
        XCTAssertFalse(source.contains("NSApplication.shared.appearance = NSAppearance(named: .darkAqua)"))
        XCTAssertFalse(source.contains(".preferredColorScheme(.dark)"))
    }

    func testDesignSystemUsesAdaptiveAppearanceTokens() throws {
        let source = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("static let sidebarIconHoverBackground = contrastTint"))
        XCTAssertTrue(source.contains("static let appBackgroundLightGlassTint"))
        XCTAssertFalse(source.contains("appBackgroundGlassTint = adaptiveColor"))
        XCTAssertTrue(source.contains("adaptiveNSColor(light: NSColor, dark: NSColor)"))
        XCTAssertTrue(source.contains("bestMatch(from:"))
    }

    func testAppShellUsesUntintedDarkBackground() throws {
        let source = try String(contentsOf: shellURL(), encoding: .utf8)
        let backgroundSource = try sourceBlock(
            in: source,
            from: "@ViewBuilder\n    private var appBackground",
            to: "private var floatingSidebar"
        )

        XCTAssertTrue(source.contains("@Environment(\\.colorScheme)"))
        XCTAssertTrue(backgroundSource.contains("if colorScheme == .light"))
        XCTAssertTrue(backgroundSource.contains("MomentoTheme.appBackgroundLightGlassTint"))
        XCTAssertTrue(backgroundSource.contains("MomentoGlassBackground(cornerRadius: 0)"))
        XCTAssertFalse(backgroundSource.contains("appBackgroundGlassTint"))
        XCTAssertFalse(backgroundSource.contains("dark: .clear"))
    }

    func testSidebarSelectedNavigationForegroundUsesAdaptiveTextColor() throws {
        let source = try String(contentsOf: sidebarURL(), encoding: .utf8)
        let foregroundSource = try sourceBlock(
            in: source,
            from: "private func sidebarNavigationForeground",
            to: "@ViewBuilder\n    private func sidebarNavigationItemBackground"
        )

        XCTAssertTrue(foregroundSource.contains("if isSelected {\n            return MomentoTheme.primaryText\n        }"))
        XCTAssertFalse(foregroundSource.contains("return .white"))
    }

    func testLibrarySwitcherSelectedSubtitleUsesLightText() throws {
        let source = try String(contentsOf: sidebarURL(), encoding: .utf8)
        let libraryRowSource = try sourceBlock(
            in: source,
            from: "private func libraryRow",
            to: "private func libraryDragPreview"
        )

        XCTAssertTrue(libraryRowSource.contains(".foregroundStyle(isSelected ? .white.opacity(0.72) : MomentoTheme.secondaryText)"))
        XCTAssertFalse(libraryRowSource.contains("MomentoTheme.primaryText.opacity(0.82)"))
    }

    func testSparkleUpdatesAreConfiguredForGitHubAppcast() throws {
        let infoData = try Data(contentsOf: infoPlistURL())
        let infoPlist = try XCTUnwrap(PropertyListSerialization.propertyList(from: infoData, options: [], format: nil) as? [String: Any])
        let entitlementsData = try Data(contentsOf: entitlementsURL())
        let entitlements = try XCTUnwrap(PropertyListSerialization.propertyList(from: entitlementsData, options: [], format: nil) as? [String: Any])
        let projectSource = try String(contentsOf: projectURL(), encoding: .utf8)
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let settingsSource = try String(contentsOf: settingsURL(), encoding: .utf8)
        let updateServiceSource = try String(contentsOf: updateServiceURL(), encoding: .utf8)

        XCTAssertEqual(infoPlist["SUFeedURL"] as? String, "https://seaony.github.io/Momento/appcast.xml")
        XCTAssertEqual(infoPlist["SUPublicEDKey"] as? String, "GPqtCAsJ50slxZrQHqwhjvY5V6XTjxhFQFNnLq8sNu0=")
        XCTAssertEqual(infoPlist["SUEnableInstallerLauncherService"] as? Bool, true)
        XCTAssertEqual(infoPlist["SUEnableAutomaticChecks"] as? Bool, true)
        XCTAssertTrue(projectSource.contains("https://github.com/sparkle-project/Sparkle"))
        XCTAssertTrue(projectSource.contains("productName = Sparkle;"))
        XCTAssertTrue(projectSource.contains("CODE_SIGN_ENTITLEMENTS = Momento/Momento.entitlements;"))
        XCTAssertEqual(entitlements["com.apple.security.app-sandbox"] as? Bool, true)
        XCTAssertEqual(entitlements["com.apple.security.network.client"] as? Bool, true)
        XCTAssertEqual(entitlements["com.apple.security.network.server"] as? Bool, true)
        let machLookupExceptions = try XCTUnwrap(
            entitlements["com.apple.security.temporary-exception.mach-lookup.global-name"] as? [String]
        )
        XCTAssertTrue(machLookupExceptions.contains("$(PRODUCT_BUNDLE_IDENTIFIER)-spks"))
        XCTAssertTrue(machLookupExceptions.contains("$(PRODUCT_BUNDLE_IDENTIFIER)-spki"))
        XCTAssertTrue(appSource.contains("@StateObject private var updateService = AppUpdateService()"))
        XCTAssertTrue(appSource.contains("MomentoUpdateCommands(localization: localization, updateService: updateService)"))
        XCTAssertTrue(settingsSource.contains("updateService.checkForUpdates()"))
        XCTAssertTrue(updateServiceSource.contains("updaterController.updater.automaticallyChecksForUpdates"))
        XCTAssertTrue(updateServiceSource.contains("updaterController.updater.checkForUpdatesInBackground()"))
        XCTAssertTrue(updateServiceSource.contains("return immediateFocus"))
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

    func testTagManagementSelectionCollapsesInspector() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains(".onChange(of: store.sidebarSelection) { _, selection in"))
        XCTAssertTrue(contentSource.contains("collapseInspectorForTagManagementIfNeeded(selection)"))
        XCTAssertTrue(contentSource.contains("private func collapseInspectorForTagManagementIfNeeded(_ selection: SidebarSelection)"))
        XCTAssertTrue(contentSource.contains("guard case .tagManagement = selection, isInspectorPresented else"))
        XCTAssertTrue(contentSource.contains("isInspectorPresented = false"))
    }

    func testAssetGridSupportsDraggingAndFilePromises() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(assetGridSource.contains("collectionView(_ collectionView: NSCollectionView, canDragItemsAt indexPaths: Set<IndexPath>, with event: NSEvent)"))
        XCTAssertTrue(assetGridSource.contains("collectionView(_ collectionView: NSCollectionView, pasteboardWriterForItemAt indexPath: IndexPath)"))
        XCTAssertTrue(assetGridSource.contains("showDragSourcePlaceholders(for: indexPaths.union(collectionView.selectionIndexPaths), in: collectionView)"))
        XCTAssertTrue(assetGridSource.contains("deferAssetChangesUntilDragEnds()"))
        XCTAssertTrue(assetGridSource.contains("DispatchQueue.main.async { [weak self] in\n                self?.applyDeferredAssetChangesAfterDragIfNeeded()"))
        XCTAssertTrue(assetGridSource.contains("return [primaryAssetID]"))
        XCTAssertTrue(assetGridSource.contains("collectionView.onPrimaryMouseUp"))
        XCTAssertTrue(assetGridSource.contains("publishSelectionAfterCurrentMouseGesture(from: collectionView)"))
        XCTAssertTrue(assetGridSource.contains("cancelPendingSelectionPublish()\n            activeDragPrimaryAssetID = asset.id"))
        XCTAssertTrue(assetGridSource.contains("cancelPendingSelectionPublish()\n            isAssetDragSessionActive = true"))
        XCTAssertFalse(assetGridSource.contains("activeDragPrimaryAssetID = asset.id\n            if !collectionView.selectionIndexPaths.contains(indexPath)"))
        XCTAssertFalse(assetGridSource.contains("collapseSelectionAfterPlainClickIfNeeded(indexPaths, in: collectionView)\n            publishSelection(from: collectionView)"))
        XCTAssertTrue(bridgeSource.contains("NSFilePromiseProvider"))
        XCTAssertTrue(bridgeSource.contains("com.seaony.momento.asset-ids"))
    }

    func testAssetGridReflowKeepsLayoutMutationInsideBatchUpdates() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let reflowStart = try XCTUnwrap(
            assetGridSource.range(
                of: "prepareAnimatedReflowLayout(for: collectionView, deletedIndexPaths: changeSet.deletedIndexPaths)"
            )
        )
        let batchStart = try XCTUnwrap(
            assetGridSource.range(
                of: "collectionView.animator().performBatchUpdates {",
                range: reflowStart.upperBound..<assetGridSource.endIndex
            )
        )
        let currentAssetsUpdate = try XCTUnwrap(
            assetGridSource.range(
                of: "coordinator.currentAssets = assets",
                range: batchStart.upperBound..<assetGridSource.endIndex
            )
        )
        let layoutPreparation = try XCTUnwrap(
            assetGridSource.range(
                of: "prepareLayout(for: collectionView)",
                range: currentAssetsUpdate.upperBound..<assetGridSource.endIndex
            )
        )
        let firstDeleteOperation = try XCTUnwrap(
            assetGridSource.range(
                of: "collectionView.deleteItems(at: changeSet.deletedIndexPaths)",
                range: currentAssetsUpdate.upperBound..<assetGridSource.endIndex
            )
        )

        XCTAssertTrue(batchStart.lowerBound > reflowStart.lowerBound)
        XCTAssertTrue(currentAssetsUpdate.lowerBound > batchStart.lowerBound)
        XCTAssertTrue(firstDeleteOperation.lowerBound > currentAssetsUpdate.lowerBound)
        XCTAssertTrue(layoutPreparation.lowerBound > firstDeleteOperation.lowerBound)

        let preBatchSource = String(assetGridSource[reflowStart.upperBound..<batchStart.lowerBound])
        XCTAssertFalse(preBatchSource.contains("coordinator.currentAssets = assets"))
        XCTAssertFalse(preBatchSource.contains("prepareLayout(for: collectionView)"))

        let preOperationSource = String(assetGridSource[currentAssetsUpdate.upperBound..<firstDeleteOperation.lowerBound])
        XCTAssertFalse(preOperationSource.contains("prepareLayout(for: collectionView)"))
    }

    func testAssetGridSupportsCommandDeleteAssetShortcut() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(assetGridSource.contains("onCommandDeleteShortcut"))
        XCTAssertTrue(assetGridSource.contains("isCommandDelete(event)"))
        XCTAssertTrue(assetGridSource.contains("modifiers == .command"))
        XCTAssertTrue(assetGridSource.contains("event.keyCode == Self.deleteKeyCode"))
        XCTAssertTrue(assetGridSource.contains("parent.onCommandDelete(selectedIDs)"))
        XCTAssertTrue(contentSource.contains("onCommandDelete: commandDeleteSelectedAssets"))
        XCTAssertTrue(contentSource.contains("if case .trash = store.sidebarSelection"))
        XCTAssertTrue(contentSource.contains("return presentPermanentAssetDeletionConfirmation(for: assetIDs)"))
        XCTAssertTrue(contentSource.contains("return moveSelectedAssetsToTrash(assetIDs)"))
        XCTAssertTrue(contentSource.contains("try store.moveAssetsToTrash(ids: Set(selectedAssets.map(\\.id)))"))
        XCTAssertTrue(contentSource.contains("try store.deleteAssetPermanently(id: asset.id)"))
    }

    func testSpacePreviewNavigationUsesSessionSnapshotAndArrowControls() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let panelSource = try String(contentsOf: previewPanelURL(), encoding: .utf8)
        let overlaySource = try String(contentsOf: previewOverlayURL(), encoding: .utf8)
        let navigationSource = try sourceBlock(
            in: contentSource,
            from: "private func navigateSpacePreview(",
            to: "private func canNavigateSpacePreview("
        )

        XCTAssertTrue(contentSource.contains("private struct SpacePreviewSession"))
        XCTAssertTrue(contentSource.contains("orderedAssetIDs: visibleAssetIDs.contains(asset.id) ? visibleAssetIDs : [asset.id]"))
        XCTAssertTrue(contentSource.contains("updatesExistingPanel: true"))
        XCTAssertFalse(navigationSource.contains("store.selectAsset"))
        XCTAssertFalse(navigationSource.contains("selectedAssetID"))
        XCTAssertTrue(panelSource.contains("func update("))
        XCTAssertTrue(panelSource.contains("returnFrame = nil"))
        XCTAssertTrue(panelSource.contains("animatesPresentation: false"))
        XCTAssertTrue(overlaySource.contains("var animatesPresentation = true"))
        XCTAssertTrue(overlaySource.contains("@State private var areNavigationControlsVisible = false"))
        XCTAssertTrue(overlaySource.contains("private var presentationAnimation: Animation?"))
        XCTAssertTrue(overlaySource.contains(".scaleEffect(presentationScale)"))
        XCTAssertTrue(overlaySource.contains(".animation(presentationAnimation, value: isPresented)"))
        XCTAssertTrue(overlaySource.contains("withAnimation(presentationAnimation)"))
        XCTAssertTrue(overlaySource.contains("if showsNavigationControls, areNavigationControlsVisible"))
        XCTAssertTrue(overlaySource.contains("transaction.animation = nil"))
        XCTAssertTrue(overlaySource.contains("try? await Task.sleep(nanoseconds: MomentoAssetPreviewMetrics.navigationRevealDelayNanoseconds)"))
        XCTAssertTrue(overlaySource.contains("systemName: \"chevron.left\""))
        XCTAssertTrue(overlaySource.contains("systemName: \"chevron.right\""))
        XCTAssertTrue(overlaySource.contains("localization.string(\"Previous Image\")"))
        XCTAssertTrue(overlaySource.contains("localization.string(\"Next Image\")"))
        XCTAssertTrue(overlaySource.contains("private static let leftArrowKeyCode: UInt16 = 123"))
        XCTAssertTrue(overlaySource.contains("private static let rightArrowKeyCode: UInt16 = 124"))
    }

    func testAssetDragWriterIsFilePromiseProviderItself() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(bridgeSource.contains("final class AssetFilePromiseProvider: NSFilePromiseProvider, NSFilePromiseProviderDelegate"))
        XCTAssertTrue(assetGridSource.contains("return AssetFilePromiseProvider("))
        XCTAssertFalse(bridgeSource.contains("final class AssetDragPasteboardItem"))
    }

    func testAssetDragOutPlaysTrashSoundOnceAfterDragBatchCompletes() throws {
        let assetGridSource = try String(contentsOf: assetGridURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(assetGridSource.contains("AssetDragExportBatch(expectedFileCount: selectedAssetIDs.count)"))
        XCTAssertTrue(assetGridSource.contains("exportBatch: exportBatch"))
        XCTAssertTrue(bridgeSource.contains("exportBatch.promiseDidFinish(success: success)"))
        XCTAssertTrue(bridgeSource.contains("AssetDeletionSoundPlayer.playDeletionSound()"))
        XCTAssertTrue(bridgeSource.contains("/System/Library/Components/CoreAudio.component/Contents/SharedSupport/SystemSounds/finder/move to trash.aif"))
        XCTAssertTrue(bridgeSource.contains("private static let playbackDurationNanoseconds: UInt64 = 500_000_000"))
        XCTAssertTrue(bridgeSource.contains("NSSound(contentsOfFile: moveToTrashSoundPath, byReference: true)"))
        XCTAssertTrue(bridgeSource.contains("successSound?.currentTime = 0"))
        XCTAssertTrue(bridgeSource.contains("successSound?.play()"))
        XCTAssertTrue(bridgeSource.contains("successSound?.stop()"))
        XCTAssertFalse(bridgeSource.contains("import AudioToolbox"))
        XCTAssertFalse(bridgeSource.contains("private static let bundledSoundName = \"MomentoActionSuccess\""))
        XCTAssertFalse(bridgeSource.contains("Bundle.main.url("))
        XCTAssertFalse(bridgeSource.contains("AudioServicesPlaySystemSound"))
        XCTAssertTrue(bridgeSource.contains("completionHandler(nil)"))
    }

    func testBrowserImportSuccessPlaysSoundInsteadOfNotification() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let feedbackSource = try String(contentsOf: browserImportFeedbackURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains("BrowserImportNotificationService.playImageSavedFeedback()"))
        XCTAssertTrue(feedbackSource.contains("AssetDeletionSoundPlayer.playDeletionSound()"))
        XCTAssertFalse(feedbackSource.contains("import UserNotifications"))
        XCTAssertFalse(feedbackSource.contains("UNUserNotificationCenter"))
        XCTAssertFalse(feedbackSource.contains("UNNotificationRequest"))
        XCTAssertFalse(feedbackSource.contains("requestAuthorization"))
    }

    func testCommandDeletePlaysDeletionSoundAfterSuccessfulDeleteActions() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("try store.moveAssetsToTrash(ids: Set(selectedAssets.map(\\.id)))"))
        XCTAssertTrue(contentSource.contains("try store.deleteAssetPermanently(id: asset.id)"))
        XCTAssertTrue(contentSource.contains("AssetDeletionSoundPlayer.playDeletionSound()"))
    }

    func testPermanentAssetDeletionRequiresConfirmationBeforeStoreDelete() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("private struct PermanentAssetDeletionRequest: Identifiable"))
        XCTAssertTrue(contentSource.contains("@State private var pendingPermanentAssetDeletion: PermanentAssetDeletionRequest?"))
        XCTAssertTrue(contentSource.contains("MomentoDestructiveConfirmationDialog("))
        XCTAssertTrue(contentSource.contains("isPresented: permanentAssetDeletionDialogIsPresented"))
        XCTAssertTrue(contentSource.contains("title: localization.string(\"Delete Permanently\")"))
        XCTAssertTrue(contentSource.contains("confirmTitle: localization.string(\"Delete Permanently\")"))
        XCTAssertTrue(contentSource.contains("confirmPermanentAssetDeletion(pendingPermanentAssetDeletion)"))
        XCTAssertTrue(contentSource.contains("presentPermanentAssetDeletionConfirmation(for: assetIDs)"))
        XCTAssertTrue(contentSource.contains("presentPermanentAssetDeletionConfirmation(for: asset)"))
        XCTAssertTrue(contentSource.contains("pendingPermanentAssetDeletion = PermanentAssetDeletionRequest(assets: selectedAssets)"))
        XCTAssertTrue(contentSource.contains("private func confirmPermanentAssetDeletion(_ request: PermanentAssetDeletionRequest)"))
        XCTAssertTrue(contentSource.contains("deleteSelectedAssetsPermanently(request.assets)"))
    }

    func testInternalDragUTTypesAreDeclaredInInfoPlist() throws {
        let data = try Data(contentsOf: infoPlistURL())
        let plist = try XCTUnwrap(PropertyListSerialization.propertyList(from: data, options: [], format: nil) as? [String: Any])
        let exportedTypes = try XCTUnwrap(plist["UTExportedTypeDeclarations"] as? [[String: Any]])

        let assetIDsType = try exportedType("com.seaony.momento.asset-ids", in: exportedTypes)
        XCTAssertEqual(assetIDsType["UTTypeDescription"] as? String, "Momento Asset Drag Payload")
        XCTAssertEqual(assetIDsType["UTTypeConformsTo"] as? [String], ["public.json"])

        let folderIDType = try exportedType("com.seaony.momento.folder-id", in: exportedTypes)
        XCTAssertEqual(folderIDType["UTTypeDescription"] as? String, "Momento Folder Drag Payload")
        XCTAssertEqual(folderIDType["UTTypeConformsTo"] as? [String], ["public.json"])
    }

    func testSidebarAcceptsInternalAssetDropsForOrganization() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(sidebarSource.contains("SidebarFolderAssetDropView("))
        XCTAssertTrue(sidebarSource.contains(".overlay {\n            SidebarFolderAssetDropView("))
        XCTAssertTrue(sidebarSource.contains("onAssignDroppedAssetsToFolder(assetIDs, folder.id)"))
        XCTAssertTrue(bridgeSource.contains("registerForDraggedTypes([AssetDragPasteboardWriter.assetIDsPasteboardType])"))
        XCTAssertTrue(bridgeSource.contains("override func hitTest(_ point: NSPoint) -> NSView?"))
        XCTAssertTrue(bridgeSource.contains("performAssetDrop(from pasteboard: NSPasteboard) -> Bool"))
        XCTAssertFalse(sidebarSource.contains(".background(\n            SidebarFolderAssetDropView("))
        XCTAssertFalse(sidebarSource.contains("MomentoSidebarAssetDropDelegate"))
        XCTAssertFalse(sidebarSource.contains("sidebarTagSection"))
        XCTAssertFalse(sidebarSource.contains("onAssignDroppedAssetsToTag"))
        XCTAssertTrue(contentSource.contains("try store.assignAssets(ids: assetIDs, to: folderID)"))
    }

    func testToolbarSearchUsesDebouncedDraftState() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(contentSource.contains("@State private var toolbarSearchDraft = \"\""))
        XCTAssertTrue(contentSource.contains("@State private var toolbarSearchDebounceTask: Task<Void, Never>?"))
        XCTAssertTrue(contentSource.contains("static let searchDebounceDelay = Duration.milliseconds(300)"))
        XCTAssertTrue(contentSource.contains("TextField(placeholder, text: toolbarSearchText)"))
        XCTAssertTrue(contentSource.contains("Task.sleep(for: ContentToolbarMetrics.searchDebounceDelay)"))
        XCTAssertTrue(contentSource.contains("syncToolbarSearchDraftFromStore()"))
        XCTAssertFalse(contentSource.contains("TextField(placeholder, text: $store.searchQuery)"))
    }

    func testCustomDialogsUseAnimatedTransitionAndDarkerBackdrop() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let libraryDialogSource = try String(
            contentsOf: repositoryRoot().appendingPathComponent("Momento/Features/Library/MomentoCreateLibraryDialog.swift"),
            encoding: .utf8
        )
        let exportDialogSource = try String(
            contentsOf: repositoryRoot().appendingPathComponent("Momento/Features/Export/MomentoAssetExportDialog.swift"),
            encoding: .utf8
        )
        let tagManagementSource = try String(
            contentsOf: repositoryRoot().appendingPathComponent("Momento/Features/Tags/MomentoTagManagementView.swift"),
            encoding: .utf8
        )

        XCTAssertTrue(contentSource.contains("private var modalDialogAnimation: Animation"))
        XCTAssertTrue(contentSource.contains(".animation(modalDialogAnimation, value: isCreateLibraryDialogPresented)"))
        XCTAssertTrue(contentSource.contains(".animation(modalDialogAnimation, value: pendingPermanentAssetDeletion != nil)"))
        XCTAssertTrue(libraryDialogSource.contains("Color.black.opacity(0.50)"))
        XCTAssertFalse(libraryDialogSource.contains("Color.black.opacity(0.35)"))
        XCTAssertTrue(libraryDialogSource.contains(".scale(scale: 0.96).combined(with: .opacity)"))
        XCTAssertTrue(exportDialogSource.contains(".scale(scale: 0.96).combined(with: .opacity)"))
        XCTAssertTrue(tagManagementSource.contains(".animation(.smooth(duration: reduceMotion ? 0.08 : 0.18), value: deletingTag != nil)"))
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }

    private func exportedType(_ identifier: String, in exportedTypes: [[String: Any]]) throws -> [String: Any] {
        try XCTUnwrap(exportedTypes.first { type in
            type["UTTypeIdentifier"] as? String == identifier
        })
    }

    private func sourceBlock(in source: String, from startMarker: String, to endMarker: String) throws -> String {
        let start = try XCTUnwrap(source.range(of: startMarker))
        let end = try XCTUnwrap(source.range(of: endMarker, range: start.upperBound..<source.endIndex))

        return String(source[start.lowerBound..<end.lowerBound])
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

    private func entitlementsURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Momento.entitlements")
    }

    private func projectURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento.xcodeproj/project.pbxproj")
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

    private func settingsURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Settings/MomentoSettingsView.swift")
    }

    private func assetGridURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/AssetCollectionGridView.swift")
    }

    private func sidebarURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Sidebar/MomentoSidebarView.swift")
    }

    private func browserImportFeedbackURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Services/BrowserImportNotificationService.swift")
    }

    private func updateServiceURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Services/AppUpdateService.swift")
    }

    private func sidebarTitlebarURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/SidebarTitlebarToggleConfigurator.swift")
    }

    private func inspectorTitlebarURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/InspectorTitlebarSpacerConfigurator.swift")
    }

    private func titlebarTooltipURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/TitlebarTooltipPresenter.swift")
    }

    private func previewPanelURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/AppKitBridge/MomentoAssetPreviewPanelController.swift")
    }

    private func previewOverlayURL() -> URL {
        repositoryRoot().appendingPathComponent("Momento/Features/Preview/MomentoAssetPreviewOverlay.swift")
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
