import AppKit
import SwiftUI
import UniformTypeIdentifiers

private struct FolderCreationRequest: Identifiable {
    let id = UUID()
    var parentID: AssetFolder.ID?
}

private enum AssetFilterFacet: String, CaseIterable, Hashable, Identifiable {
    case colors
    case tags
    case fileTypes

    var id: String { rawValue }
}

private enum ContentToolbarMetrics {
    // 搜索框使用弹性宽度（min...max）而不是写死宽度：窄窗口下工具栏会优先把
    // 它压窄，而不是因为无法压缩而整体收进溢出菜单、退化成一个按钮。
    static let searchControlMinWidth: CGFloat = 140
    static let searchControlMaxWidth: CGFloat = 214
    static let iconButtonWidth: CGFloat = 38
    static let viewModeSwitcherWidth: CGFloat = 112
    static let filterPopoverWidth: CGFloat = 340
    static let filterScrollableContentMaxHeight: CGFloat = 220
    static let sortPopoverWidth: CGFloat = 156
    static let popoverSectionSpacing: CGFloat = 12
}

struct ContentView: View {
    @Environment(\.appLocalization) private var localization
    @AppStorage(AppSettingsKeys.defaultViewMode) private var defaultViewModeRawValue = AssetViewMode.masonry.rawValue
    @Bindable var store: LibraryStore

    @State private var isImporterPresented = false
    @State private var isCommandPalettePresented = false
    @State private var isCreateLibraryDialogPresented = false
    @State private var editingLibrary: RecentLibraryReference?
    @State private var deletingLibrary: RecentLibraryReference?
    @State private var creatingFolder: FolderCreationRequest?
    @State private var editingFolder: AssetFolder?
    @State private var deletingFolder: AssetFolder?
    @State private var isInspectorPresented = false
    @State private var importError: String?
    @State private var hoveredToolbarViewMode: AssetViewMode?
    @State private var hoveredToolbarActionID: String?
    @State private var hoveredFilterFacet: AssetFilterFacet?
    @State private var hoveredFilterOptionID: String?
    @State private var hoveredSortOptionID: String?
    @State private var selectedFilterFacet: AssetFilterFacet = .colors
    @State private var filterTagSearchQuery = ""
    @State private var isFilterPopoverPresented = false
    @State private var isSortPopoverPresented = false
    @State private var shellToastRequest: MomentoToastRequest?

    private var sidebarSelection: Binding<String?> {
        Binding {
            store.sidebarItemID()
        } set: { id in
            store.selectSidebarItem(id: id)
        }
    }

    private var selectedAssetIDs: Set<AssetItem.ID> {
        if let selectedAssetID = store.selectedAssetID {
            return [selectedAssetID]
        }
        return []
    }

    private var selectedTags: Binding<[String]> {
        Binding {
            store.selectedAsset?.tags.map(\.name) ?? []
        } set: { names in
            do {
                try store.updateSelectedTags(names)
            } catch {
                showImportError(error)
            }
        }
    }

    private var availableTagNames: [String] {
        store.tags.map(\.name)
    }

    private var selectedFolderIDs: Binding<[AssetFolder.ID]> {
        Binding {
            store.selectedAsset?.folderIDs ?? []
        } set: { folderIDs in
            updateSelectedFolders(folderIDs)
        }
    }

    var body: some View {
        Group {
            if store.currentLibrary == nil {
                MomentoLibraryWelcomeView(
                    onCreateLibrary: createLibrary,
                    onOpenLibrary: openLibrary
                )
                .navigationTitle("")
            } else {
                libraryBody
            }
        }
        .overlay(alignment: .bottom) {
            if let errorMessage {
                importErrorBanner(errorMessage)
                    .padding(.bottom, 16)
            }
        }
        .overlay {
            libraryDialogOverlay
        }
        .fileImporter(
            isPresented: $isImporterPresented,
            allowedContentTypes: [.image, .folder],
            allowsMultipleSelection: true,
            onCompletion: handleImportResult
        )
        .dropDestination(for: URL.self) { urls, _ in
            importURLs(urls)
            return true
        }
        .background {
            Button("") {
                withAnimation(.smooth(duration: 0.18)) {
                    isCommandPalettePresented = true
                }
            }
            .keyboardShortcut("k", modifiers: .command)
            .frame(width: 0, height: 0)
            .opacity(0)
        }
        .background {
            WindowTransparencyConfigurator()
        }
        .toolbarBackgroundVisibility(.hidden, for: .windowToolbar)
        .onAppear {
            validateCurrentLibraryAvailability()
        }
        .onChange(of: defaultViewMode) { _, newValue in
            store.setViewMode(newValue)
        }
        .frame(minWidth: MomentoTheme.mainWindowMinWidth, minHeight: MomentoTheme.mainWindowMinHeight)
    }

    private var libraryBody: some View {
        let visibleAssets = store.visibleAssets

        return MomentoShellView(
            sidebarSelection: sidebarSelection,
            searchQuery: $store.searchQuery,
            isCommandPalettePresented: $isCommandPalettePresented,
            isInspectorPresented: $isInspectorPresented,
            libraryName: store.currentLibrary?.name,
            currentLibraryID: store.currentLibrary?.id,
            recentLibraries: store.recentLibraries,
            folders: store.folders,
            sidebarCounts: sidebarAssetCounts,
            onCreateLibrary: createLibrary,
            onOpenLibrary: openLibrary,
            onSwitchLibrary: switchLibrary,
            onRenameLibrary: renameLibrary,
            onDeleteLibrary: deleteLibrary,
            onRevealLibrary: revealLibraryInFinder,
            onMoveLibrary: moveLibrary,
            onReloadLibrary: reloadLibrary,
            onCloseLibrary: closeLibrary,
            onImportAssets: { isImporterPresented = true },
            onCreateFolder: presentCreateFolderDialog,
            onRenameFolder: presentRenameFolderDialog,
            onDeleteFolder: presentDeleteFolderDialog,
            title: title,
            subtitle: localization.itemCount(visibleAssets.count),
            inspectorAsset: store.selectedAsset.map { MomentoInspectorAsset(asset: $0, localization: localization) },
            inspectorTags: selectedTags,
            inspectorAvailableTags: availableTagNames,
            inspectorFolderIDs: selectedFolderIDs,
            inspectorFolders: store.folders,
            toastRequest: $shellToastRequest,
            onRenameInspectorAsset: renameAssetTitle,
            commands: commands,
            onCommandSelected: handleCommand
        ) {
            if isTagManagementSelected {
                tagManagementContent
            } else {
                assetGridContent(visibleAssets)
            }
        }
        .toolbar {
            ToolbarSpacer(.flexible)
            ToolbarItemGroup(placement: .automatic) {
                toolbarFilterButton
                toolbarSortButton
                    .padding(.trailing, 6)
                toolbarViewModeSwitcher
                    .padding(.trailing, 6)
                toolbarSearchControl
            }
            .sharedBackgroundVisibility(.hidden)
        }
        .navigationTitle("")
    }

    private var isTagManagementSelected: Bool {
        if case .tagManagement = store.sidebarSelection {
            return true
        }
        return false
    }

    @ViewBuilder
    private func assetGridContent(_ visibleAssets: [AssetItem]) -> some View {
        ZStack {
            AssetCollectionGridView(
                assets: visibleAssets,
                selectedAssetIDs: selectedAssetIDs,
                viewMode: store.viewMode,
                localization: localization,
                onSelectionChange: selectAssets,
                onDoubleClick: { asset in
                    preview(asset)
                },
                onSpacePreviewStart: previewWhileSpaceIsPressed,
                onSpacePreviewEnd: endSpacePreview,
                onFavoriteToggle: toggleFavorite,
                onContextMenuAction: handleAssetContextMenuAction
            )

            if visibleAssets.isEmpty {
                emptyGridState
            }
        }
    }

    private var tagManagementContent: some View {
        MomentoTagManagementView(
            tags: store.tagSummaries,
            onRenameTag: renameTag,
            onDeleteTag: deleteTag
        )
    }

    private var toolbarViewModeSwitcher: some View {
        HStack(spacing: 2) {
            ForEach(AssetViewMode.allCases) { viewMode in
                let isSelected = store.viewMode == viewMode
                let isHovered = hoveredToolbarViewMode == viewMode

                Button {
                    store.setViewMode(viewMode)
                } label: {
                    Image(systemName: systemImage(for: viewMode))
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(isSelected ? Color.white : MomentoTheme.primaryText)
                        .frame(width: 34, height: 28)
                        .background {
                            toolbarSegmentBackground(isSelected: isSelected, isHovered: isHovered)
                        }
                        .contentShape(RoundedRectangle(cornerRadius: MomentoTheme.toolbarControlRadius - 3, style: .continuous))
                }
                .buttonStyle(.plain)
                .pointerStyle(.link)
                .help(localization.title(for: viewMode))
                .accessibilityLabel(localization.title(for: viewMode))
                .onHover { hovering in
                    withAnimation(.smooth(duration: 0.14)) {
                        if hovering {
                            hoveredToolbarViewMode = viewMode
                        } else if hoveredToolbarViewMode == viewMode {
                            hoveredToolbarViewMode = nil
                        }
                    }
                }
            }
        }
        .padding(3)
        .frame(width: ContentToolbarMetrics.viewModeSwitcherWidth, height: MomentoTheme.toolbarControlHeight)
        .background {
            toolbarControlBackground(cornerRadius: MomentoTheme.toolbarControlRadius)
        }
    }

    @ViewBuilder
    private var toolbarSearchControl: some View {
        let placeholder = localization.string("Search image name")

        HStack(spacing: 8) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: MomentoTheme.toolbarIconSize, weight: .semibold))
                .foregroundStyle(MomentoTheme.primaryText)

            TextField(placeholder, text: $store.searchQuery)
                .textFieldStyle(.plain)
                .frame(maxWidth: .infinity)
        }
        .padding(.horizontal, 11)
        .frame(
            minWidth: ContentToolbarMetrics.searchControlMinWidth,
            maxWidth: ContentToolbarMetrics.searchControlMaxWidth
        )
        .frame(height: MomentoTheme.toolbarControlHeight)
        .background {
            toolbarControlBackground(cornerRadius: MomentoTheme.toolbarControlRadius)
        }
        .layoutPriority(10)
        .help(placeholder)
    }

    private var toolbarFilterButton: some View {
        toolbarIconButton(
            id: "filter",
            systemImage: store.filterState.isActive ? "line.3.horizontal.decrease.circle.fill" : "line.3.horizontal.decrease.circle",
            label: localization.string("Filter"),
            isActive: store.filterState.isActive
        ) {
            withAnimation(.smooth(duration: 0.16)) {
                isSortPopoverPresented = false
                isFilterPopoverPresented.toggle()
            }
        }
        .popover(isPresented: $isFilterPopoverPresented, arrowEdge: .bottom) {
            filterPopover
        }
    }

    private var toolbarSortButton: some View {
        toolbarIconButton(
            id: "sort",
            systemImage: "arrow.up.arrow.down.circle",
            label: localization.string("Sort"),
            isActive: true
        ) {
            withAnimation(.smooth(duration: 0.16)) {
                isFilterPopoverPresented = false
                isSortPopoverPresented.toggle()
            }
        }
        .popover(isPresented: $isSortPopoverPresented, arrowEdge: .bottom) {
            sortPopover
        }
    }

    private func toolbarIconButton(
        id: String,
        systemImage: String,
        label: String,
        isActive: Bool,
        action: @escaping () -> Void
    ) -> some View {
        let isHovered = hoveredToolbarActionID == id
        let shape = RoundedRectangle(cornerRadius: MomentoTheme.toolbarControlRadius, style: .continuous)

        return Button(action: action) {
            Image(systemName: systemImage)
                .font(.system(size: MomentoTheme.toolbarIconSize, weight: .semibold))
                .foregroundStyle(isActive || isHovered ? MomentoTheme.primaryText : MomentoTheme.secondaryText)
                .frame(width: ContentToolbarMetrics.iconButtonWidth, height: MomentoTheme.toolbarControlHeight)
                .background {
                    MomentoGlassBackground(glass: .regular.interactive(true), cornerRadius: MomentoTheme.toolbarControlRadius)
                }
                .overlay {
                    if isHovered {
                        shape.fill(MomentoTheme.sidebarIconHoverBackground)
                    }
                }
                .contentShape(shape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .help(label)
        .accessibilityLabel(label)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                if hovering {
                    hoveredToolbarActionID = id
                } else if hoveredToolbarActionID == id {
                    hoveredToolbarActionID = nil
                }
            }
        }
    }

    private var filterPopover: some View {
        GlassEffectContainer(spacing: 10) {
            VStack(alignment: .leading, spacing: 14) {
                filterFacetPicker
                filterFacetContent
                    .frame(maxWidth: .infinity, alignment: .topLeading)
            }
        }
        .padding(14)
        .frame(width: ContentToolbarMetrics.filterPopoverWidth)
        .fixedSize(horizontal: false, vertical: true)
        .background {
            MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.18)), cornerRadius: 20)
        }
    }

    private var sortPopover: some View {
        GlassEffectContainer(spacing: 10) {
            VStack(alignment: .leading, spacing: ContentToolbarMetrics.popoverSectionSpacing) {
                VStack(alignment: .leading, spacing: 7) {
                    ForEach(AssetSortOption.allCases) { option in
                        sortChoiceButton(
                            id: "sort-option-\(option.id)",
                            title: localization.title(for: option),
                            isSelected: store.sortOption == option
                        ) {
                            store.setSortOption(option)
                        }
                    }
                }
            }
        }
        .padding(12)
        .frame(width: ContentToolbarMetrics.sortPopoverWidth)
        .background {
            MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.16)), cornerRadius: 18)
        }
    }

    private var filterFacetPicker: some View {
        let shape = RoundedRectangle(cornerRadius: 14, style: .continuous)

        return HStack(spacing: 6) {
            ForEach(AssetFilterFacet.allCases) { facet in
                filterFacetButton(facet)
            }
        }
        .padding(5)
        .background {
            ZStack {
                shape.fill(.regularMaterial)
                shape.fill(Color.black.opacity(0.14))
            }
        }
        .overlay {
            shape.strokeBorder(Color.white.opacity(0.08), lineWidth: 1)
        }
        .transaction(value: selectedFilterFacet) { transaction in
            transaction.disablesAnimations = true
        }
    }

    private func filterFacetButton(_ facet: AssetFilterFacet) -> some View {
        let isSelected = selectedFilterFacet == facet
        let isHovered = hoveredFilterFacet == facet
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        return Button {
            selectedFilterFacet = facet
        } label: {
            Text(filterFacetSegmentTitle(for: facet))
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(Color.white.opacity(isSelected ? 0.96 : 0.72))
                .lineLimit(1)
                .minimumScaleFactor(0.86)
                .frame(maxWidth: .infinity)
                .frame(height: 34)
                .contentShape(shape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .background {
            if isSelected || isHovered {
                shape.fill(Color.white.opacity(isSelected ? 0.15 : 0.07))
            }
        }
        .overlay {
            if isSelected {
                shape.strokeBorder(Color.white.opacity(0.10), lineWidth: 1)
            }
        }
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.12)) {
                if hovering {
                    hoveredFilterFacet = facet
                } else if hoveredFilterFacet == facet {
                    hoveredFilterFacet = nil
                }
            }
        }
    }

    @ViewBuilder
    private var filterFacetContent: some View {
        switch selectedFilterFacet {
        case .colors:
            filterColorsContent
        case .tags:
            filterTagsContent
        case .fileTypes:
            filterFileTypesContent
        }
    }

    private var filterColorsContent: some View {
        let colorCategories = store.availableFilterColorCategories
        let columns = Array(repeating: GridItem(.fixed(34), spacing: 5), count: 8)

        return LazyVGrid(
            columns: columns,
            alignment: .leading,
            spacing: 7
        ) {
            ForEach(colorCategories) { category in
                colorCategoryFilterButton(category)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var filterTagsContent: some View {
        let tags = filteredFilterTags

        return VStack(alignment: .leading, spacing: 8) {
            if store.tags.count > 12 || !filterTagSearchQuery.isEmpty {
                filterTagSearchField
            }

            ScrollView {
                if tags.isEmpty {
                    filterEmptyText(localization.string("No tags"))
                } else {
                    LazyVGrid(
                        columns: [GridItem(.adaptive(minimum: 88), spacing: 7)],
                        alignment: .leading,
                        spacing: 7
                    ) {
                        ForEach(tags) { tag in
                            filterChoiceButton(
                                id: "filter-tag-\(tag.id)",
                                title: tag.name,
                                systemImage: "number",
                                isSelected: store.filterState.tagIDs.contains(tag.id)
                            ) {
                                store.toggleFilterTag(id: tag.id)
                            }
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .frame(maxHeight: ContentToolbarMetrics.filterScrollableContentMaxHeight)
        }
        .frame(maxWidth: .infinity, alignment: .topLeading)
    }

    private var filterFileTypesContent: some View {
        let fileExtensions = sortedFilterFileExtensions

        return ScrollView {
            if fileExtensions.isEmpty {
                filterEmptyText(localization.string("No file types available"))
            } else {
                LazyVGrid(
                    columns: [GridItem(.adaptive(minimum: 72), spacing: 7)],
                    alignment: .leading,
                    spacing: 7
                ) {
                    ForEach(fileExtensions, id: \.self) { fileExtension in
                        filterChoiceButton(
                            id: "filter-file-\(fileExtension)",
                            title: fileExtension.uppercased(),
                            systemImage: "doc",
                            isSelected: store.filterState.fileExtensions.contains(fileExtension)
                        ) {
                            store.toggleFilterFileExtension(fileExtension)
                        }
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
        .frame(maxHeight: ContentToolbarMetrics.filterScrollableContentMaxHeight)
        .frame(maxWidth: .infinity, alignment: .topLeading)
    }

    private var filteredFilterTags: [TagItem] {
        let query = filterTagSearchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        let tags = query.isEmpty ? store.tags : store.tags.filter {
            $0.name.localizedCaseInsensitiveContains(query)
        }

        return tags.sorted { lhs, rhs in
            lhs.name.localizedCaseInsensitiveCompare(rhs.name) == .orderedAscending
        }
    }

    private var sortedFilterFileExtensions: [String] {
        store.availableFilterFileExtensions.sorted { lhs, rhs in
            lhs.localizedStandardCompare(rhs) == .orderedAscending
        }
    }

    private var filterTagSearchField: some View {
        HStack(spacing: 7) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 12, weight: .semibold))
                .foregroundStyle(MomentoTheme.secondaryText)

            TextField(localization.string("Search tags"), text: $filterTagSearchQuery)
                .textFieldStyle(.plain)
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(MomentoTheme.primaryText)
        }
        .padding(.horizontal, 9)
        .frame(height: 30)
        .background {
            MomentoGlassBackground(glass: .regular.interactive(true), cornerRadius: 10)
        }
    }

    private func filterFacetSegmentTitle(for facet: AssetFilterFacet) -> String {
        let title = filterFacetTitle(for: facet)
        let count = selectedFilterCount(for: facet)
        return count > 0 ? "\(title) \(count)" : title
    }

    private func selectedFilterCount(for facet: AssetFilterFacet) -> Int {
        switch facet {
        case .colors:
            store.filterState.colorCategories.count
        case .tags:
            store.filterState.tagIDs.count
        case .fileTypes:
            store.filterState.fileExtensions.count
        }
    }

    private func filterFacetTitle(for facet: AssetFilterFacet) -> String {
        switch facet {
        case .colors:
            localization.string("Colors")
        case .tags:
            localization.string("Tags")
        case .fileTypes:
            localization.string("File Types")
        }
    }

    private func filterEmptyText(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 12, weight: .medium))
            .foregroundStyle(MomentoTheme.tertiaryText)
            .frame(height: 28)
            .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func colorCategoryFilterButton(_ category: AssetColorCategory) -> some View {
        let isSelected = store.filterState.colorCategories.contains(category)
        let optionID = "filter-color-\(category.id)"
        let isHovered = hoveredFilterOptionID == optionID
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)
        let swatchShape = RoundedRectangle(cornerRadius: 7, style: .continuous)

        return Button {
            store.toggleFilterColorCategory(category)
        } label: {
            ZStack {
                swatchShape
                    .fill(colorSwatch(for: category))
                    .frame(width: 22, height: 22)
                    .overlay {
                        swatchShape.strokeBorder(Color.white.opacity(0.28), lineWidth: 1)
                    }
                if isSelected {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(Color.white)
                        .shadow(color: Color.black.opacity(0.32), radius: 2, x: 0, y: 1)
                }
            }
            .frame(width: 34, height: 34)
            .glassEffect(
                .regular.tint(Color.white.opacity(isSelected || isHovered ? 0.16 : 0.06)).interactive(),
                in: shape
            )
            .contentShape(shape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .help(localization.title(for: category))
        .onHover { hovering in
            updateFilterOptionHover(id: optionID, hovering: hovering)
        }
    }

    private func filterChoiceButton(
        id: String,
        title: String,
        systemImage: String,
        isSelected: Bool,
        action: @escaping () -> Void
    ) -> some View {
        let isHovered = hoveredFilterOptionID == id
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        return Button(action: action) {
            HStack(spacing: 6) {
                Image(systemName: isSelected ? "checkmark.circle.fill" : systemImage)
                    .font(.system(size: 11, weight: .semibold))
                Text(title)
                    .lineLimit(1)
                    .minimumScaleFactor(0.82)
            }
            .font(.system(size: 12, weight: .medium))
            .foregroundStyle(MomentoTheme.primaryText)
            .padding(.horizontal, 9)
            .frame(height: 30)
            .frame(maxWidth: .infinity, alignment: .leading)
            .glassEffect(
                .regular.tint(Color.white.opacity(isSelected || isHovered ? 0.16 : 0.06)).interactive(),
                in: shape
            )
            .contentShape(shape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateFilterOptionHover(id: id, hovering: hovering)
        }
    }

    private func sortChoiceButton(
        id: String,
        title: String,
        isSelected: Bool,
        action: @escaping () -> Void
    ) -> some View {
        let isHovered = hoveredSortOptionID == id
        let shape = RoundedRectangle(cornerRadius: 11, style: .continuous)

        return Button(action: action) {
            HStack(spacing: 8) {
                Text(title)
                    .lineLimit(1)
                Spacer(minLength: 0)
                if isSelected {
                    Image(systemName: sortDirectionSystemImage())
                        .font(.system(size: 12, weight: .semibold))
                        .frame(width: 16)
                }
            }
            .font(.system(size: 12, weight: .medium))
            .foregroundStyle(MomentoTheme.primaryText)
            .padding(.horizontal, 10)
            .frame(height: 32)
            .glassEffect(
                .regular.tint(Color.white.opacity(isSelected || isHovered ? 0.16 : 0.06)).interactive(),
                in: shape
            )
            .contentShape(shape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateSortOptionHover(id: id, hovering: hovering)
        }
    }

    private func sortDirectionSystemImage() -> String {
        switch store.sortDirection {
        case .ascending:
            "arrow.up.circle.fill"
        case .descending:
            "arrow.down.circle.fill"
        }
    }

    private func updateFilterOptionHover(id: String, hovering: Bool) {
        withAnimation(.smooth(duration: 0.12)) {
            if hovering {
                hoveredFilterOptionID = id
            } else if hoveredFilterOptionID == id {
                hoveredFilterOptionID = nil
            }
        }
    }

    private func updateSortOptionHover(id: String, hovering: Bool) {
        withAnimation(.smooth(duration: 0.12)) {
            if hovering {
                hoveredSortOptionID = id
            } else if hoveredSortOptionID == id {
                hoveredSortOptionID = nil
            }
        }
    }

    private func colorSwatch(for category: AssetColorCategory) -> Color {
        switch category {
        case .black:
            Color(red: 0.05, green: 0.05, blue: 0.06)
        case .white:
            Color(red: 0.94, green: 0.94, blue: 0.9)
        case .gray:
            Color(red: 0.5, green: 0.5, blue: 0.5)
        case .red:
            Color(red: 0.92, green: 0.16, blue: 0.16)
        case .rose:
            Color(red: 0.93, green: 0.23, blue: 0.34)
        case .pink:
            Color(red: 0.88, green: 0.28, blue: 0.58)
        case .magenta:
            Color(red: 0.78, green: 0.18, blue: 0.78)
        case .purple:
            Color(red: 0.5, green: 0.28, blue: 0.82)
        case .violet:
            Color(red: 0.38, green: 0.28, blue: 0.85)
        case .indigo:
            Color(red: 0.22, green: 0.28, blue: 0.72)
        case .blue:
            Color(red: 0.16, green: 0.42, blue: 0.86)
        case .sky:
            Color(red: 0.22, green: 0.62, blue: 0.9)
        case .cyan:
            Color(red: 0.14, green: 0.72, blue: 0.84)
        case .teal:
            Color(red: 0.1, green: 0.64, blue: 0.68)
        case .mint:
            Color(red: 0.32, green: 0.78, blue: 0.62)
        case .green:
            Color(red: 0.22, green: 0.66, blue: 0.32)
        case .lime:
            Color(red: 0.58, green: 0.78, blue: 0.2)
        case .olive:
            Color(red: 0.48, green: 0.5, blue: 0.18)
        case .yellow:
            Color(red: 0.95, green: 0.78, blue: 0.16)
        case .amber:
            Color(red: 0.88, green: 0.58, blue: 0.16)
        case .orange:
            Color(red: 0.95, green: 0.45, blue: 0.12)
        case .coral:
            Color(red: 0.94, green: 0.34, blue: 0.24)
        case .brown:
            Color(red: 0.48, green: 0.28, blue: 0.13)
        case .beige:
            Color(red: 0.78, green: 0.67, blue: 0.5)
        }
    }

    @ViewBuilder
    private func toolbarSegmentBackground(isSelected: Bool, isHovered: Bool) -> some View {
        let shape = RoundedRectangle(cornerRadius: MomentoTheme.toolbarControlRadius - 3, style: .continuous)

        if isSelected {
            shape.fill(Color.accentColor)
        } else if isHovered {
            shape.fill(MomentoTheme.sidebarIconHoverBackground)
        } else {
            Color.clear
        }
    }

    private func toolbarControlBackground(cornerRadius: CGFloat) -> some View {
        MomentoGlassBackground(glass: .regular.interactive(true), cornerRadius: cornerRadius)
    }

    @ViewBuilder
    private var libraryDialogOverlay: some View {
        if isCreateLibraryDialogPresented {
            MomentoCreateLibraryDialog(
                mode: .create,
                isPresented: $isCreateLibraryDialogPresented,
                initialName: localization.string("Untitled Library"),
                onSubmit: chooseLibraryDestination
            )
            .zIndex(30)
        }

        if let editingLibrary {
            MomentoCreateLibraryDialog(
                mode: .edit,
                isPresented: editingLibraryDialogIsPresented,
                initialName: editingLibrary.name,
                onSubmit: { name in
                    renameLibrary(editingLibrary.id, to: name)
                }
            )
            .zIndex(31)
        }

        if let deletingLibrary {
            MomentoDeleteLibraryDialog(
                isPresented: deletingLibraryDialogIsPresented,
                libraryName: deletingLibrary.name,
                onConfirm: {
                    confirmDeleteLibrary(deletingLibrary.id)
                }
            )
            .zIndex(32)
        }

        if let creatingFolder {
            MomentoFolderNameDialog(
                mode: .create,
                isPresented: creatingFolderDialogIsPresented,
                initialName: localization.string("Untitled Folder"),
                onSubmit: { name in
                    createFolder(name: name, parentID: creatingFolder.parentID)
                }
            )
            .zIndex(33)
        }

        if let editingFolder {
            MomentoFolderNameDialog(
                mode: .edit,
                isPresented: editingFolderDialogIsPresented,
                initialName: editingFolder.name,
                onSubmit: { name in
                    renameFolder(editingFolder.id, to: name)
                }
            )
            .zIndex(34)
        }

        if let deletingFolder {
            MomentoDeleteFolderDialog(
                isPresented: deletingFolderDialogIsPresented,
                folderName: deletingFolder.name,
                onConfirm: {
                    deleteFolder(deletingFolder.id)
                }
            )
            .zIndex(35)
        }
    }

    private var editingLibraryDialogIsPresented: Binding<Bool> {
        Binding {
            editingLibrary != nil
        } set: { isPresented in
            if !isPresented {
                editingLibrary = nil
            }
        }
    }

    private var deletingLibraryDialogIsPresented: Binding<Bool> {
        Binding {
            deletingLibrary != nil
        } set: { isPresented in
            if !isPresented {
                deletingLibrary = nil
            }
        }
    }

    private var creatingFolderDialogIsPresented: Binding<Bool> {
        Binding {
            creatingFolder != nil
        } set: { isPresented in
            if !isPresented {
                creatingFolder = nil
            }
        }
    }

    private var editingFolderDialogIsPresented: Binding<Bool> {
        Binding {
            editingFolder != nil
        } set: { isPresented in
            if !isPresented {
                editingFolder = nil
            }
        }
    }

    private var deletingFolderDialogIsPresented: Binding<Bool> {
        Binding {
            deletingFolder != nil
        } set: { isPresented in
            if !isPresented {
                deletingFolder = nil
            }
        }
    }

    private var defaultViewMode: AssetViewMode {
        AssetViewMode(rawValue: defaultViewModeRawValue) ?? .masonry
    }

    private var errorMessage: String? {
        importError ?? store.libraryErrorMessage.map(localization.string)
    }

    private var title: String {
        switch store.sidebarSelection {
        case .library:
            store.currentLibrary?.name ?? localization.string("Library")
        case .favorites:
            localization.string("Favorites")
        case .uncategorized:
            localization.string("Uncategorized")
        case .untagged:
            localization.string("Untagged")
        case .tagManagement:
            localization.string("Tag Management")
        case .folderManagement:
            localization.string("Folder Management")
        case .folder(let folderID):
            store.folder(id: folderID)?.name ?? localization.string("Folder")
        case .tag(let tagID):
            store.tags.first { $0.id == tagID }?.name ?? localization.string("Tag")
        case .trash:
            localization.string("Trash")
        }
    }

    private var sidebarAssetCounts: MomentoSidebarAssetCounts {
        guard let currentLibrary = store.currentLibrary else {
            return .empty
        }

        let allLibraryAssets = store.assets.filter { $0.libraryID == currentLibrary.id }
        let libraryAssets = allLibraryAssets.filter { !$0.isTrashed }
        var folderCounts: [AssetFolder.ID: Int] = [:]

        for asset in libraryAssets {
            for folderID in asset.folderIDs {
                folderCounts[folderID, default: 0] += 1
            }
        }

        return MomentoSidebarAssetCounts(
            all: libraryAssets.count,
            favorites: libraryAssets.filter(\.isFavorite).count,
            uncategorized: libraryAssets.filter(\.folderIDs.isEmpty).count,
            untagged: libraryAssets.filter(\.tags.isEmpty).count,
            trash: allLibraryAssets.filter(\.isTrashed).count,
            folders: folderCounts
        )
    }

    private var commands: [MomentoCommand] {
        [
            MomentoCommand(id: "import", title: localization.string("Import Assets"), subtitle: localization.string("Choose files or folders"), systemImage: "square.and.arrow.down", shortcut: "I"),
            MomentoCommand(id: "view-masonry", title: localization.string("Masonry View"), subtitle: localization.string("Show adaptive visual grid"), systemImage: "circle.hexagongrid", shortcut: "1"),
            MomentoCommand(id: "view-grid", title: localization.string("Grid View"), subtitle: localization.string("Show uniform thumbnails"), systemImage: "square.grid.2x2", shortcut: "2"),
            MomentoCommand(id: "view-list", title: localization.string("List View"), subtitle: localization.string("Show compact rows"), systemImage: "rectangle.grid.1x2", shortcut: "3"),
            MomentoCommand(id: "toggle-inspector", title: localization.string("Toggle Inspector"), subtitle: localization.string("Show or hide asset details"), systemImage: "sidebar.right", shortcut: "⌥⌘I"),
            MomentoCommand(id: "quick-preview", title: localization.string("Quick Preview"), subtitle: localization.string("Preview the selected asset"), systemImage: "eye", shortcut: "Space")
        ]
    }

    private var emptyGridState: some View {
        VStack(spacing: 12) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.system(size: 34, weight: .regular))
                .foregroundStyle(MomentoTheme.tertiaryText)

            Text(localization.string("Drop assets here"))
                .font(.system(size: 15, weight: .semibold))

            Text(localization.string("Import images or GIFs to start building this library."))
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 360)

            Button {
                isImporterPresented = true
            } label: {
                Label(localization.string("Import Assets"), systemImage: "square.and.arrow.down")
            }
            .buttonStyle(.glassProminent)
            .controlSize(.large)
        }
        .padding(28)
        .background {
            MomentoGlassBackground(cornerRadius: 18)
        }
    }

    private func importErrorBanner(_ message: String) -> some View {
        HStack(spacing: 10) {
            Image(systemName: "exclamationmark.triangle")
            Text(message)
                .lineLimit(2)
            Button(localization.string("Dismiss")) {
                withAnimation(.smooth(duration: 0.16)) {
                    importError = nil
                    store.libraryErrorMessage = nil
                }
            }
            .buttonStyle(.glass)
        }
        .font(.system(size: 12, weight: .medium))
        .padding(.horizontal, 14)
        .frame(minHeight: 34)
        .background {
            MomentoGlassBackground(cornerRadius: 11)
        }
    }

    private func selectAssets(_ ids: Set<AssetItem.ID>) {
        guard let id = ids.first else {
            return
        }

        store.selectAsset(id: id)

        if !isInspectorPresented {
            withAnimation(.smooth(duration: 0.22)) {
                isInspectorPresented = true
            }
        }
    }

    private func systemImage(for viewMode: AssetViewMode) -> String {
        switch viewMode {
        case .masonry:
            "circle.hexagongrid"
        case .grid:
            "square.grid.2x2"
        case .list:
            "rectangle.grid.1x2"
        }
    }

    @discardableResult
    private func preview(_ asset: AssetItem) -> Bool {
        showPreview(asset, closesOnSpaceKeyUp: false)
    }

    private func previewWhileSpaceIsPressed(_ asset: AssetItem, sourceFrame: NSRect?) {
        showPreview(asset, closesOnSpaceKeyUp: true, sourceFrame: sourceFrame)
    }

    private func endSpacePreview() {
        MomentoAssetPreviewPanelController.shared.close()
    }

    @discardableResult
    private func showPreview(_ asset: AssetItem, closesOnSpaceKeyUp: Bool, sourceFrame: NSRect? = nil) -> Bool {
        guard let previewURL = previewURL(for: asset) else {
            return false
        }

        MomentoAssetPreviewPanelController.shared.show(
            asset: asset,
            previewURL: previewURL,
            localization: localization,
            closesOnSpaceKeyUp: closesOnSpaceKeyUp,
            sourceFrame: sourceFrame
        )
        return true
    }

    private func previewURL(for asset: AssetItem) -> URL? {
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            return asset.storageURL
        }

        return asset.originalURL
    }

    private func handleCommand(_ command: MomentoCommand) {
        switch command.id {
        case "import":
            isImporterPresented = true
        case "view-masonry":
            store.setViewMode(.masonry)
        case "view-grid":
            store.setViewMode(.grid)
        case "view-list":
            store.setViewMode(.list)
        case "quick-preview":
            if let selectedAsset = store.selectedAsset {
                preview(selectedAsset)
            }
        case "toggle-inspector":
            withAnimation(.smooth(duration: 0.18)) {
                isInspectorPresented.toggle()
            }
        default:
            break
        }
    }

    private func handleAssetContextMenuAction(_ asset: AssetItem, action: AssetContextMenuAction) {
        switch action {
        case .previewOriginal:
            if preview(asset) {
                showAssetActionToast(action)
            }
        case .refreshThumbnail:
            if refreshThumbnail(for: asset) {
                showAssetActionToast(action)
            }
        case .reanalyzeColors:
            if reanalyzeColors(for: asset) {
                showAssetActionToast(action)
            }
        case .revealInFinder:
            if revealInFinder(asset) {
                showAssetActionToast(action)
            }
        case .moveToTrash:
            if moveAssetToTrash(asset) {
                showAssetActionToast(action)
            }
        case .restore:
            if restoreAsset(asset) {
                showAssetActionToast(action)
            }
        case .deletePermanently:
            if deleteAssetPermanently(asset) {
                showAssetActionToast(action)
            }
        }
    }

    private func toggleFavorite(_ asset: AssetItem) {
        let isAddingFavorite = !asset.isFavorite

        do {
            try store.toggleFavorite(for: asset.id)
            shellToastRequest = MomentoToastRequest(
                message: localization.string(isAddingFavorite ? "Added to Favorites" : "Removed from Favorites")
            )
        } catch {
            showImportError(error)
        }
    }

    private func renameAssetTitle(_ assetID: AssetItem.ID, to title: String) {
        do {
            try store.renameAsset(id: assetID, to: title)
        } catch {
            showImportError(error)
        }
    }

    private func renameTag(_ tagID: TagItem.ID, to name: String) {
        do {
            try store.renameTag(id: tagID, to: name)
        } catch {
            showImportError(error)
        }
    }

    private func deleteTag(_ tagID: TagItem.ID) {
        do {
            try store.deleteTag(id: tagID)
        } catch {
            showImportError(error)
        }
    }

    private func updateSelectedFolders(_ folderIDs: [AssetFolder.ID]) {
        guard let selectedAsset = store.selectedAsset else {
            return
        }

        let assetIDs = Set([selectedAsset.id])
        let currentFolderIDs = Set(selectedAsset.folderIDs)
        let nextFolderIDs = Set(folderIDs)

        do {
            for folderID in nextFolderIDs.subtracting(currentFolderIDs) {
                try store.assignAssets(ids: assetIDs, to: folderID)
            }
            for folderID in currentFolderIDs.subtracting(nextFolderIDs) {
                try store.unassignAssets(ids: assetIDs, from: folderID)
            }
        } catch {
            showImportError(error)
        }
    }

    private func refreshThumbnail(for asset: AssetItem) -> Bool {
        do {
            AssetCollectionGridView.invalidatePreviewCache(for: asset)
            if let updatedAsset = try store.refreshThumbnail(for: asset.id) {
                AssetCollectionGridView.invalidatePreviewCache(for: updatedAsset)
            }
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func reanalyzeColors(for asset: AssetItem) -> Bool {
        do {
            try store.reanalyzeColors(for: asset.id)
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func revealInFinder(_ asset: AssetItem) -> Bool {
        guard let fileURL = previewURL(for: asset) else {
            return false
        }

        NSWorkspace.shared.activateFileViewerSelecting([fileURL])
        return true
    }

    private func moveAssetToTrash(_ asset: AssetItem) -> Bool {
        do {
            AssetCollectionGridView.invalidatePreviewCache(for: asset)
            try store.moveAssetToTrash(id: asset.id)
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func restoreAsset(_ asset: AssetItem) -> Bool {
        do {
            try store.restoreAssets(ids: [asset.id])
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func deleteAssetPermanently(_ asset: AssetItem) -> Bool {
        do {
            AssetCollectionGridView.invalidatePreviewCache(for: asset)
            try store.deleteAssetPermanently(id: asset.id)
            return true
        } catch {
            showImportError(error)
            return false
        }
    }

    private func showAssetActionToast(_ action: AssetContextMenuAction) {
        shellToastRequest = MomentoToastRequest(message: localization.string(action.titleKey))
    }

    private func createLibrary() {
        withAnimation(.smooth(duration: 0.16)) {
            isCreateLibraryDialogPresented = true
        }
    }

    private func chooseLibraryDestination(named libraryName: String) {
        let panel = NSOpenPanel()
        panel.canChooseFiles = false
        panel.canChooseDirectories = true
        panel.canCreateDirectories = true
        panel.allowsMultipleSelection = false
        panel.prompt = localization.string("Create Library")

        guard panel.runModal() == .OK, let destinationURL = panel.url else {
            return
        }

        let packageURL = destinationURL.appendingPathComponent(libraryName, isDirectory: true)

        do {
            try store.createLibrary(at: packageURL)
        } catch {
            showImportError(error)
        }
    }

    private func openLibrary() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.momentoLibrary]
        panel.canChooseFiles = true
        panel.canChooseDirectories = true
        panel.allowsMultipleSelection = false

        guard panel.runModal() == .OK, let url = panel.url else {
            return
        }

        do {
            try store.openLibrary(at: url)
        } catch {
            showImportError(error)
        }
    }

    private func switchLibrary(_ id: RecentLibraryReference.ID) {
        do {
            try store.openRecentLibrary(id: id)
        } catch {
            showImportError(error)
        }
    }

    private func renameLibrary(_ id: RecentLibraryReference.ID) {
        guard let library = store.recentLibraries.first(where: { $0.id == id }) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            editingLibrary = library
        }
    }

    private func renameLibrary(_ id: RecentLibraryReference.ID, to name: String) {
        do {
            try store.renameRecentLibrary(id: id, to: name)
        } catch {
            showImportError(error)
        }
    }

    private func deleteLibrary(_ id: RecentLibraryReference.ID) {
        guard let library = store.recentLibraries.first(where: { $0.id == id }) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            deletingLibrary = library
        }
    }

    private func revealLibraryInFinder(_ id: RecentLibraryReference.ID) {
        do {
            let url = try store.recentLibraryURL(id: id)
            let didStartAccessing = url.startAccessingSecurityScopedResource()
            defer {
                if didStartAccessing {
                    url.stopAccessingSecurityScopedResource()
                }
            }

            NSWorkspace.shared.activateFileViewerSelecting([url])
        } catch {
            showImportError(error)
        }
    }

    private func confirmDeleteLibrary(_ id: RecentLibraryReference.ID) {
        do {
            try store.deleteRecentLibrary(id: id)
        } catch {
            showImportError(error)
        }
    }

    private func moveLibrary(
        _ id: RecentLibraryReference.ID,
        relativeTo targetID: RecentLibraryReference.ID,
        insertAfterTarget: Bool
    ) {
        do {
            try store.moveRecentLibrary(id: id, relativeTo: targetID, insertAfterTarget: insertAfterTarget)
        } catch {
            showImportError(error)
        }
    }

    private func reloadLibrary() {
        do {
            try store.clearCachesAndReloadCurrentLibrary()
        } catch {
            showImportError(error)
        }
    }

    private func presentCreateFolderDialog(parentID: AssetFolder.ID?) {
        withAnimation(.smooth(duration: 0.16)) {
            creatingFolder = FolderCreationRequest(parentID: parentID)
        }
    }

    private func presentRenameFolderDialog(_ id: AssetFolder.ID) {
        guard let folder = store.folder(id: id) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            editingFolder = folder
        }
    }

    private func presentDeleteFolderDialog(_ id: AssetFolder.ID) {
        guard let folder = store.folder(id: id) else {
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            deletingFolder = folder
        }
    }

    private func createFolder(name: String, parentID: AssetFolder.ID?) {
        do {
            try store.createFolder(name: name, parentID: parentID)
        } catch {
            showImportError(error)
        }
    }

    private func renameFolder(_ id: AssetFolder.ID, to name: String) {
        do {
            try store.renameFolder(id: id, to: name)
        } catch {
            showImportError(error)
        }
    }

    private func deleteFolder(_ id: AssetFolder.ID) {
        do {
            try store.deleteFolder(id: id)
        } catch {
            showImportError(error)
        }
    }

    private func closeLibrary() {
        withAnimation(.smooth(duration: 0.18)) {
            importError = nil
            store.closeCurrentLibrary()
        }
    }

    private func validateCurrentLibraryAvailability() {
        do {
            try store.validateCurrentLibraryAvailability()
        } catch {
            showImportError(error)
        }
    }

    private func handleImportResult(_ result: Result<[URL], Error>) {
        switch result {
        case .success(let urls):
            importURLs(urls)
        case .failure(let error):
            showImportError(error)
        }
    }

    private func importURLs(_ urls: [URL]) {
        guard !urls.isEmpty else {
            return
        }

        Task {
            do {
                try await store.importItems(from: urls)
            } catch {
                showImportError(error)
            }
        }
    }

    private func showImportError(_ error: Error) {
        withAnimation(.smooth(duration: 0.16)) {
            importError = localization.errorMessage(error)
        }
    }
}

private extension MomentoInspectorAsset {
    init(asset: AssetItem, localization: AppLocalization) {
        let fileURL: URL?
        if FileManager.default.fileExists(atPath: asset.storageURL.path) {
            fileURL = asset.storageURL
        } else {
            fileURL = asset.originalURL
        }

        let previewURL: URL?
        if let thumbnailURL = asset.thumbnailURL, FileManager.default.fileExists(atPath: thumbnailURL.path) {
            previewURL = thumbnailURL
        } else {
            previewURL = fileURL
        }

        self.init(
            id: asset.id,
            title: asset.displayName,
            fileName: fileURL?.lastPathComponent ?? asset.displayName,
            previewImage: previewURL.flatMap { asset.kind == .image || asset.kind == .gif ? NSImage(contentsOf: $0) : nil },
            dimensions: asset.dimensions.map { "\($0.width) × \($0.height)" },
            colors: asset.paletteColors.map { color in
                MomentoInspectorColor(hex: color.hex, coverage: color.coverage)
            },
            filePath: fileURL?.path,
            fileSize: localization.fileSize(asset.byteSize),
            addedDate: asset.importedAt,
            kind: localization.kindTitle(for: asset.kind),
            exifItems: Self.exifItems(for: asset.exifMetadata, localization: localization)
        )
    }

    private static func exifItems(
        for metadata: AssetExifMetadata?,
        localization: AppLocalization
    ) -> [MomentoInspectorInfoItem] {
        guard let metadata else {
            return []
        }

        var items: [MomentoInspectorInfoItem] = []
        appendDate(metadata.fileCreatedAt, label: "Created", to: &items, localization: localization)
        appendDate(metadata.fileModifiedAt, label: "Modified", to: &items, localization: localization)
        appendDate(metadata.contentCreatedAt, label: "Content Created", to: &items, localization: localization)

        if let dimensions = dimensionsValue(width: metadata.pixelWidth, height: metadata.pixelHeight) {
            append(dimensions, label: "Dimensions", to: &items, localization: localization)
        }
        if let resolution = resolutionValue(width: metadata.dpiWidth, height: metadata.dpiHeight, localization: localization) {
            append(resolution, label: "Resolution", to: &items, localization: localization)
        }

        append(metadata.colorModel, label: "Color Space", to: &items, localization: localization)
        append(metadata.profileName, label: "Color Profile", to: &items, localization: localization)
        append(metadata.cameraMake, label: "Camera Maker", to: &items, localization: localization)
        append(metadata.cameraModel, label: "Camera Model", to: &items, localization: localization)
        append(metadata.lensModel, label: "Lens Model", to: &items, localization: localization)

        if let exposureTime = metadata.exposureTime {
            append(exposureTimeValue(exposureTime, localization: localization), label: "Exposure Time", to: &items, localization: localization)
        }
        if let focalLength = metadata.focalLength {
            append(localization.format("%@ mm", numberString(focalLength, localization: localization)), label: "Focal Length", to: &items, localization: localization)
        }
        if !metadata.isoSpeedRatings.isEmpty {
            let iso = metadata.isoSpeedRatings.map { integerString($0, localization: localization) }.joined(separator: ", ")
            append(iso, label: "ISO", to: &items, localization: localization)
        }
        if let flashFired = metadata.flashFired {
            append(localization.string(flashFired ? "Yes" : "No"), label: "Flash", to: &items, localization: localization)
        }
        if let fNumber = metadata.fNumber {
            append("f/\(numberString(fNumber, localization: localization))", label: "Aperture", to: &items, localization: localization)
        }
        if let exposureProgram = metadata.exposureProgram {
            append(exposureProgramValue(exposureProgram, localization: localization), label: "Exposure Program", to: &items, localization: localization)
        }
        if let meteringMode = metadata.meteringMode {
            append(meteringModeValue(meteringMode, localization: localization), label: "Metering Mode", to: &items, localization: localization)
        }
        if let whiteBalance = metadata.whiteBalance {
            append(whiteBalanceValue(whiteBalance, localization: localization), label: "White Balance", to: &items, localization: localization)
        }
        append(metadata.creator, label: "Creator", to: &items, localization: localization)
        return items
    }

    private static func appendDate(
        _ date: Date?,
        label: String,
        to items: inout [MomentoInspectorInfoItem],
        localization: AppLocalization
    ) {
        guard let date else {
            return
        }
        append(localization.dateTime(date), label: label, to: &items, localization: localization)
    }

    private static func append(
        _ value: String?,
        label: String,
        to items: inout [MomentoInspectorInfoItem],
        localization: AppLocalization
    ) {
        guard let value, !value.isEmpty else {
            return
        }

        items.append(MomentoInspectorInfoItem(label: localization.string(label), value: value))
    }

    private static func dimensionsValue(width: Int?, height: Int?) -> String? {
        guard let width, let height else {
            return nil
        }
        return "\(width) × \(height)"
    }

    private static func resolutionValue(
        width: Double?,
        height: Double?,
        localization: AppLocalization
    ) -> String? {
        guard let width, let height else {
            return nil
        }
        return "\(numberString(width, localization: localization)) × \(numberString(height, localization: localization))"
    }

    private static func exposureTimeValue(
        _ seconds: Double,
        localization: AppLocalization
    ) -> String {
        guard seconds > 0 else {
            return numberString(seconds, localization: localization, maximumFractionDigits: 3)
        }

        if seconds < 1 {
            let denominator = Int((1 / seconds).rounded())
            if denominator > 1 {
                return "1/\(integerString(denominator, localization: localization))"
            }
        }

        return numberString(seconds, localization: localization, maximumFractionDigits: 3)
    }

    private static func exposureProgramValue(
        _ value: Int,
        localization: AppLocalization
    ) -> String {
        let key: String? = switch value {
        case 0: "Not defined"
        case 1: "Manual"
        case 2: "Normal program"
        case 3: "Aperture priority"
        case 4: "Shutter priority"
        case 5: "Creative program"
        case 6: "Action program"
        case 7: "Portrait mode"
        case 8: "Landscape mode"
        default: nil
        }

        return key.map(localization.string) ?? integerString(value, localization: localization)
    }

    private static func meteringModeValue(
        _ value: Int,
        localization: AppLocalization
    ) -> String {
        let key: String? = switch value {
        case 0: "Unknown"
        case 1: "Average"
        case 2: "Center-weighted average"
        case 3: "Spot"
        case 4: "Multi-spot"
        case 5: "Pattern"
        case 6: "Partial"
        case 255: "Other"
        default: nil
        }

        return key.map(localization.string) ?? integerString(value, localization: localization)
    }

    private static func whiteBalanceValue(
        _ value: Int,
        localization: AppLocalization
    ) -> String {
        let key: String? = switch value {
        case 0: "Auto"
        case 1: "Manual"
        default: nil
        }

        return key.map(localization.string) ?? integerString(value, localization: localization)
    }

    private static func integerString(
        _ value: Int,
        localization: AppLocalization
    ) -> String {
        value.formatted(.number.locale(localization.locale))
    }

    private static func numberString(
        _ value: Double,
        localization: AppLocalization,
        maximumFractionDigits: Int = 1
    ) -> String {
        let formatter = NumberFormatter()
        formatter.locale = localization.locale
        formatter.minimumFractionDigits = 0
        formatter.maximumFractionDigits = maximumFractionDigits
        return formatter.string(from: NSNumber(value: value)) ?? "\(value)"
    }
}

private extension Color {
    init?(hex: String?) {
        guard var normalized = hex?.trimmingCharacters(in: .whitespacesAndNewlines), !normalized.isEmpty else {
            return nil
        }

        if normalized.hasPrefix("#") {
            normalized.removeFirst()
        }

        guard normalized.count == 6, let value = UInt64(normalized, radix: 16) else {
            return nil
        }

        self.init(
            red: Double((value & 0xFF0000) >> 16) / 255,
            green: Double((value & 0x00FF00) >> 8) / 255,
            blue: Double(value & 0x0000FF) / 255
        )
    }
}

#Preview {
    ContentView(store: LibraryStore(libraries: [.defaultLibrary]))
        .environment(\.appLocalization, AppLocalization(language: .system))
}
