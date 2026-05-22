import SwiftUI
import AppKit

struct MomentoInspectorColor: Identifiable, Hashable {
    var hex: String
    var coverage: Double?

    var id: String {
        if let coverage {
            return "\(hex)-\(coverage)"
        }
        return hex
    }

    var normalizedHex: String {
        let trimmedHex = hex.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        if trimmedHex.hasPrefix("#") {
            return trimmedHex
        }

        return "#\(trimmedHex)"
    }

    var helpText: String {
        guard let coverage else {
            return normalizedHex
        }
        return "\(normalizedHex) (\((coverage * 100).formatted(.number.precision(.fractionLength(1))))%)"
    }
}

struct MomentoInspectorInfoItem: Identifiable, Hashable {
    var label: String
    var value: String

    var id: String { label }
}

struct MomentoInspectorAsset: Identifiable, Hashable {
    var id: String
    var title: String
    var fileName: String
    var previewImage: NSImage?
    var dimensions: String?
    var colors: [MomentoInspectorColor]
    var filePath: String?
    var fileSize: String?
    var addedDate: Date?
    var kind: String?
    var exifItems: [MomentoInspectorInfoItem]

    init(
        id: String,
        title: String,
        fileName: String,
        previewImage: NSImage? = nil,
        dimensions: String? = nil,
        colorHexes: [String] = [],
        colors: [MomentoInspectorColor]? = nil,
        filePath: String? = nil,
        fileSize: String? = nil,
        addedDate: Date? = nil,
        kind: String? = nil,
        exifItems: [MomentoInspectorInfoItem] = []
    ) {
        self.id = id
        self.title = title
        self.fileName = fileName
        self.previewImage = previewImage
        self.dimensions = dimensions
        self.colors = colors ?? colorHexes.map { MomentoInspectorColor(hex: $0, coverage: nil) }
        self.filePath = filePath
        self.fileSize = fileSize
        self.addedDate = addedDate
        self.kind = kind
        self.exifItems = exifItems
    }
}

struct MomentoInspectorView: View {
    private static let tagPickerMinimumContentHeight: CGFloat = 92
    private static let tagPickerMaximumContentHeight: CGFloat = 220

    @Environment(\.appLocalization) private var localization

    var asset: MomentoInspectorAsset?
    @Binding var tags: [String]
    var availableTags: [String]
    @Binding var notes: String
    var onRevealInFinder: (() -> Void)?
    var onTitleCommit: ((MomentoInspectorAsset.ID, String) -> Void)?
    var onColorCopied: (() -> Void)?

    @State private var hoveredColorID: String?
    @State private var hoveredTag: String?
    @State private var hoveredTagChoice: String?
    @State private var isCreateTagRowHovered = false
    @State private var isTagPickerPresented = false
    @State private var tagSearchQuery = ""
    @State private var tagPickerContentHeight: CGFloat = 0
    @State private var hoveredTitleID: MomentoInspectorAsset.ID?
    @State private var editingTitleID: MomentoInspectorAsset.ID?
    @State private var draftTitle = ""
    @FocusState private var isTitleFieldFocused: Bool
    @FocusState private var isTagSearchFocused: Bool

    init(
        asset: MomentoInspectorAsset?,
        tags: Binding<[String]> = .constant([]),
        availableTags: [String] = [],
        notes: Binding<String> = .constant(""),
        onRevealInFinder: (() -> Void)? = nil,
        onTitleCommit: ((MomentoInspectorAsset.ID, String) -> Void)? = nil,
        onColorCopied: (() -> Void)? = nil
    ) {
        self.asset = asset
        self._tags = tags
        self.availableTags = availableTags
        self._notes = notes
        self.onRevealInFinder = onRevealInFinder
        self.onTitleCommit = onTitleCommit
        self.onColorCopied = onColorCopied
    }

    var body: some View {
        VStack(spacing: 0) {
            if let asset {
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        preview(asset)
                        metadata(asset)
                        tagEditor
                        exifMetadata(asset)
                    }
                    .padding(
                        EdgeInsets(
                            top: MomentoTheme.inspectorContentTopInset + 16,
                            leading: 16,
                            bottom: 16,
                            trailing: 16
                        )
                    )
                }
                .contentMargins(.top, 0, for: .scrollContent)
                .ignoresSafeArea(.container, edges: .top)
                .scrollIndicators(.never)
            } else {
                emptyState
            }
        }
        .frame(
            minWidth: MomentoTheme.inspectorMinWidth,
            idealWidth: MomentoTheme.inspectorWidth,
            maxWidth: MomentoTheme.inspectorMaxWidth,
            maxHeight: .infinity,
            alignment: .top
        )
        .onChange(of: asset?.id) { _, _ in
            cancelTitleEditing()
            hoveredTitleID = nil
            hoveredTag = nil
            closeTagPicker()
        }
        .onChange(of: isTagPickerPresented) { _, isPresented in
            if !isPresented {
                resetTagPicker()
            }
        }
    }

    private func preview(_ asset: MomentoInspectorAsset) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            previewImage(asset)

            colorSection(asset.colors)

            titleEditor(asset)
        }
    }

    private func beginTitleEditing(_ asset: MomentoInspectorAsset) {
        editingTitleID = asset.id
        draftTitle = asset.title
        isTitleFieldFocused = true
    }

    private func commitTitleEditing(for asset: MomentoInspectorAsset) {
        guard editingTitleID == asset.id else {
            return
        }

        let trimmedTitle = draftTitle.trimmingCharacters(in: .whitespacesAndNewlines)
        editingTitleID = nil
        isTitleFieldFocused = false

        guard !trimmedTitle.isEmpty, trimmedTitle != asset.title else {
            return
        }

        onTitleCommit?(asset.id, trimmedTitle)
    }

    private func cancelTitleEditing() {
        editingTitleID = nil
        draftTitle = ""
        isTitleFieldFocused = false
    }

    @ViewBuilder
    private func titleEditor(_ asset: MomentoInspectorAsset) -> some View {
        let isEditing = editingTitleID == asset.id
        let isHovered = hoveredTitleID == asset.id
        let titleShape = RoundedRectangle(cornerRadius: 7, style: .continuous)

        if isEditing {
            TextField("", text: $draftTitle, axis: .vertical)
                .textFieldStyle(.plain)
                .font(.system(size: 13, weight: .regular))
                .lineLimit(1...6)
                .focused($isTitleFieldFocused)
                .padding(.horizontal, 6)
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background {
                    titleShape
                        .fill(Color.white.opacity(0.08))
                        .overlay {
                            titleShape.strokeBorder(Color.white.opacity(0.14), lineWidth: 1)
                        }
                }
                .onSubmit {
                    commitTitleEditing(for: asset)
                }
                .onChange(of: isTitleFieldFocused) { _, isFocused in
                    if !isFocused {
                        commitTitleEditing(for: asset)
                    }
                }
                .task {
                    isTitleFieldFocused = true
                }
        } else {
            Text(asset.title)
                .font(.system(size: 13, weight: .regular))
                .lineLimit(nil)
                .fixedSize(horizontal: false, vertical: true)
                .padding(.horizontal, 6)
                .padding(.vertical, 4)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background {
                    titleShape.fill(isHovered ? Color.white.opacity(0.06) : Color.clear)
                }
                .contentShape(titleShape)
                .pointerStyle(.link)
                .onTapGesture {
                    beginTitleEditing(asset)
                }
                .onHover { hovering in
                    withAnimation(.smooth(duration: 0.12)) {
                        hoveredTitleID = hovering ? asset.id : nil
                    }
                }
        }
    }

    @ViewBuilder
    private func previewImage(_ asset: MomentoInspectorAsset) -> some View {
        if let image = asset.previewImage {
            Image(nsImage: image)
                .resizable()
                .scaledToFit()
                .frame(maxWidth: .infinity, alignment: .center)
                .clipShape(RoundedRectangle(cornerRadius: MomentoTheme.assetImageCornerRadius, style: .continuous))
        } else {
            ZStack {
                MomentoGlassBackground(cornerRadius: MomentoTheme.assetImageCornerRadius)

                VStack(spacing: 8) {
                    Image(systemName: "photo")
                        .font(.system(size: 32))
                        .foregroundStyle(MomentoTheme.tertiaryText)
                    Text(asset.fileName)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(MomentoTheme.secondaryText)
                        .lineLimit(1)
                }
                .padding()
            }
            .frame(maxWidth: .infinity)
            .aspectRatio(1.25, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: MomentoTheme.assetImageCornerRadius, style: .continuous))
        }
    }

    private func metadata(_ asset: MomentoInspectorAsset) -> some View {
        inspectorSection(localization.string("Details")) {
            if let kind = asset.kind {
                infoRow(localization.string("Kind"), kind)
            }
            if let dimensions = asset.dimensions {
                infoRow(localization.string("Dimensions"), dimensions)
            }
            if let fileSize = asset.fileSize {
                infoRow(localization.string("Size"), fileSize)
            }
            if let addedDate = asset.addedDate {
                infoRow(localization.string("Added"), localization.dateTime(addedDate))
            }
        }
    }

    @ViewBuilder
    private func exifMetadata(_ asset: MomentoInspectorAsset) -> some View {
        if !asset.exifItems.isEmpty {
            inspectorSection(localization.string("EXIF")) {
                ForEach(asset.exifItems) { item in
                    infoRow(item.label, item.value)
                }
            }
        }
    }

    private var tagEditor: some View {
        inspectorSection(localization.string("Tags")) {
            FlowLayout(spacing: 6) {
                ForEach(tags, id: \.self) { tag in
                    tagChip(tag)
                }

                addTagChip
            }
        }
    }

    private func tagChip(_ tag: String) -> some View {
        let isHovered = hoveredTag == tag

        return HStack(spacing: isHovered ? 5 : 0) {
            Text(tag)
                .lineLimit(1)

            if isHovered {
                Button {
                    removeTag(tag)
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 8, weight: .bold))
                        .frame(width: 14, height: 14)
                }
                .buttonStyle(.plain)
                .foregroundStyle(MomentoTheme.primaryText)
                .transition(.opacity.combined(with: .scale(scale: 0.82)))
                .accessibilityLabel(localization.format("Remove %@", tag))
            }
        }
        .font(.system(size: 12, weight: .medium))
        .foregroundStyle(MomentoTheme.primaryText)
        .padding(.leading, 9)
        .padding(.trailing, isHovered ? 5 : 9)
        .frame(height: 26)
        .glassEffect(
            .regular.tint(Color.white.opacity(isHovered ? 0.12 : 0.04)),
            in: RoundedRectangle(cornerRadius: 9, style: .continuous)
        )
        .contentShape(RoundedRectangle(cornerRadius: 9, style: .continuous))
        .animation(.smooth(duration: 0.14), value: isHovered)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                hoveredTag = hovering ? tag : nil
            }
        }
    }

    private var addTagChip: some View {
        Button {
            withAnimation(.smooth(duration: 0.16)) {
                isTagPickerPresented = true
            }
        } label: {
            Image(systemName: "plus")
                .font(.system(size: 12, weight: .semibold))
                .frame(width: 34, height: 26)
                .glassEffect(.regular, in: RoundedRectangle(cornerRadius: 9, style: .continuous))
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .accessibilityLabel(localization.string("Add tag"))
        .popover(isPresented: $isTagPickerPresented, arrowEdge: .bottom) {
            tagPicker
        }
    }

    private var tagPicker: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(MomentoTheme.primaryText)

                TextField(localization.string("Search tags"), text: $tagSearchQuery)
                    .textFieldStyle(.plain)
                    .font(.system(size: 13, weight: .medium))
                    .focused($isTagSearchFocused)
                    .onSubmit(submitTagSearch)
            }
            .padding(.horizontal, 10)
            .frame(height: 34)
            .glassEffect(.regular, in: RoundedRectangle(cornerRadius: 12, style: .continuous))

            ScrollView {
                VStack(alignment: .leading, spacing: 8) {
                    if !filteredTagChoices.isEmpty {
                        Text("\(localization.string("All Tags")) (\(filteredTagChoices.count))")
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .padding(.horizontal, 2)

                        FlowLayout(spacing: 6) {
                            ForEach(filteredTagChoices, id: \.self) { tag in
                                tagChoiceButton(tag)
                            }
                        }
                    } else {
                        Text(localization.string("No matching tags"))
                            .font(.system(size: 12))
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .padding(.vertical, 4)
                    }

                    if shouldShowCreateTag {
                        Button {
                            addTag(trimmedTagSearchQuery)
                        } label: {
                            HStack(spacing: 8) {
                                Image(systemName: "plus")
                                    .font(.system(size: 12, weight: .bold))
                                Text(localization.format("Create %@", trimmedTagSearchQuery))
                                    .lineLimit(1)
                                Spacer(minLength: 0)
                            }
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(MomentoTheme.primaryText)
                            .padding(.horizontal, 10)
                            .frame(height: 34)
                            .glassEffect(
                                .regular.tint(Color.accentColor.opacity(isCreateTagRowHovered ? 0.32 : 0.18)),
                                in: RoundedRectangle(cornerRadius: 11, style: .continuous)
                            )
                        }
                        .buttonStyle(.plain)
                        .pointerStyle(.link)
                        .onHover { hovering in
                            withAnimation(.smooth(duration: 0.12)) {
                                isCreateTagRowHovered = hovering
                            }
                        }
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.bottom, shouldShowCreateTag ? 8 : 0)
                .background {
                    GeometryReader { proxy in
                        Color.clear.preference(
                            key: TagPickerContentHeightKey.self,
                            value: proxy.size.height
                        )
                    }
                }
            }
            .frame(height: tagPickerScrollHeight)
            .scrollIndicators(.never)
            .onPreferenceChange(TagPickerContentHeightKey.self) { height in
                tagPickerContentHeight = height
            }
        }
        .padding(10)
        .frame(width: 292)
        .background {
            MomentoGlassBackground(
                glass: .regular.tint(Color.black.opacity(0.16)),
                cornerRadius: 16
            )
        }
        .task {
            isTagSearchFocused = true
        }
    }

    private func tagChoiceButton(_ tag: String) -> some View {
        let isSelected = containsTag(tag)
        let isHovered = hoveredTagChoice == tag

        return Button {
            guard !isSelected else {
                return
            }

            addTag(tag)
        } label: {
            HStack(spacing: 6) {
                Image(systemName: isSelected ? "checkmark.circle.fill" : "tag")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(isSelected ? Color.accentColor : MomentoTheme.secondaryText)
                Text(tag)
                    .lineLimit(1)
            }
            .font(.system(size: 12, weight: .medium))
            .foregroundStyle(isSelected ? MomentoTheme.secondaryText : MomentoTheme.primaryText)
            .padding(.horizontal, 10)
            .frame(height: 28)
            .glassEffect(
                .regular.tint(Color.white.opacity(isHovered || isSelected ? 0.14 : 0.06)),
                in: Capsule()
            )
        }
        .buttonStyle(.plain)
        .disabled(isSelected)
        .pointerStyle(.link)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.12)) {
                hoveredTagChoice = hovering ? tag : nil
            }
        }
    }

    @ViewBuilder
    private func colorSection(_ colors: [MomentoInspectorColor]) -> some View {
        if !colors.isEmpty {
            GeometryReader { proxy in
                ScrollView(.horizontal) {
                    HStack(spacing: 1) {
                        ForEach(colors) { color in
                            colorSwatchButton(color)
                                .zIndex(hoveredColorID == color.id ? 1 : 0)
                        }
                    }
                    .padding(.horizontal, 5)
                    .padding(.vertical, 3)
                    .background {
                        RoundedRectangle(cornerRadius: 10, style: .continuous)
                            .fill(Color.black.opacity(0.24))
                            .overlay {
                                RoundedRectangle(cornerRadius: 10, style: .continuous)
                                    .strokeBorder(Color.white.opacity(0.08), lineWidth: 1)
                            }
                    }
                    .frame(minWidth: proxy.size.width, alignment: .center)
                }
                .scrollIndicators(.never)
            }
            .frame(height: 28)
            .frame(maxWidth: .infinity)
            .overlayPreferenceValue(ColorSwatchBoundsKey.self) { bounds in
                GeometryReader { proxy in
                    if
                        let hoveredColor = colors.first(where: { $0.id == hoveredColorID }),
                        let anchor = bounds[hoveredColor.id]
                    {
                        let swatchFrame = proxy[anchor]

                        colorTooltip(hoveredColor)
                            .position(x: swatchFrame.midX, y: swatchFrame.minY - 20)
                            .transition(.opacity.combined(with: .scale(scale: 0.96, anchor: .bottom)))
                            .zIndex(2)
                    }
                }
                .allowsHitTesting(false)
            }
            .zIndex(1)
        }
    }

    private func colorSwatchButton(_ color: MomentoInspectorColor) -> some View {
        let isHovered = hoveredColorID == color.id
        let swatchShape = RoundedRectangle(cornerRadius: 5, style: .continuous)
        let hoverShape = RoundedRectangle(cornerRadius: 7, style: .continuous)

        return Button {
            copyColor(color)
        } label: {
            swatchShape
                .fill(Color(hex: color.hex) ?? .clear)
                .frame(width: 16, height: 16)
                .overlay {
                    swatchShape.strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                }
                .padding(2)
                .background {
                    if isHovered {
                        hoverShape.fill(MomentoTheme.sidebarIconHoverBackground)
                    } else {
                        Color.clear
                    }
                }
                .contentShape(hoverShape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateColorHover(id: color.id, isHovering: hovering)
        }
        .anchorPreference(key: ColorSwatchBoundsKey.self, value: .bounds) { anchor in
            [color.id: anchor]
        }
        .accessibilityLabel(color.helpText)
    }

    private func colorTooltip(_ color: MomentoInspectorColor) -> some View {
        Text(color.helpText)
            .font(.system(size: 13, weight: .medium))
            .foregroundStyle(.white)
            .lineLimit(1)
            .fixedSize()
            .padding(.horizontal, 10)
            .frame(height: 30)
            .background {
                MomentoGlassBackground(
                    glass: .regular.tint(Color.black.opacity(0.34)),
                    cornerRadius: 9
                )
                    .overlay {
                        RoundedRectangle(cornerRadius: 9, style: .continuous)
                            .strokeBorder(Color.white.opacity(0.16), lineWidth: 1)
                    }
            }
            .shadow(color: Color.black.opacity(0.28), radius: 12, y: 6)
            .allowsHitTesting(false)
    }

    private var notesEditor: some View {
        inspectorSection(localization.string("Notes")) {
            TextEditor(text: $notes)
                .font(.system(size: 12))
                .scrollContentBackground(.hidden)
                .frame(minHeight: 86)
                .padding(6)
                .background {
                    MomentoGlassBackground(cornerRadius: 8)
                        .overlay {
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                        }
                }
        }
    }

    private func fileSection(_ asset: MomentoInspectorAsset) -> some View {
        inspectorSection(localization.string("File")) {
            infoRow(localization.string("Name"), asset.fileName)
            if let filePath = asset.filePath {
                infoRow(localization.string("Path"), filePath)
            }

            if onRevealInFinder != nil {
                Button(action: { onRevealInFinder?() }) {
                    Label(localization.string("Reveal in Finder"), systemImage: "finder")
                        .frame(maxWidth: .infinity)
                        .frame(height: 30)
                }
                .buttonStyle(.glass)
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "sidebar.right")
                .font(.system(size: 30))
                .foregroundStyle(MomentoTheme.tertiaryText)
            Text(localization.string("No asset selected"))
                .font(.system(size: 13, weight: .medium))
            Text(localization.string("Select an item to inspect metadata, tags, colors, and file details."))
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 220)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(24)
    }

    private func inspectorSection<Content: View>(
        _ title: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title.uppercased())
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(MomentoTheme.tertiaryText)
            content()
        }
    }

    private func infoRow(_ label: String, _ value: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .frame(width: 74, alignment: .leading)
            Text(value)
                .font(.system(size: 12))
                .textSelection(.enabled)
                .lineLimit(2)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var trimmedTagSearchQuery: String {
        tagSearchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var availableTagChoices: [String] {
        var seen = Set<String>()
        return (availableTags + tags)
            .compactMap { tag in
                let trimmedTag = tag.trimmingCharacters(in: .whitespacesAndNewlines)
                guard !trimmedTag.isEmpty else {
                    return nil
                }

                let key = trimmedTag.lowercased()
                guard seen.insert(key).inserted else {
                    return nil
                }
                return trimmedTag
            }
            .sorted { lhs, rhs in
                lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
            }
    }

    private var filteredTagChoices: [String] {
        let query = trimmedTagSearchQuery
        guard !query.isEmpty else {
            return availableTagChoices
        }

        return availableTagChoices.filter {
            $0.localizedCaseInsensitiveContains(query)
        }
    }

    private var shouldShowCreateTag: Bool {
        let query = trimmedTagSearchQuery
        guard !query.isEmpty else {
            return false
        }

        return !availableTagChoices.contains {
            $0.caseInsensitiveCompare(query) == .orderedSame
        }
    }

    private var tagPickerScrollHeight: CGFloat {
        let measuredHeight = tagPickerContentHeight > 0
            ? tagPickerContentHeight
            : Self.tagPickerMinimumContentHeight

        return min(
            max(measuredHeight, Self.tagPickerMinimumContentHeight),
            Self.tagPickerMaximumContentHeight
        )
    }

    private func submitTagSearch() {
        let query = trimmedTagSearchQuery
        guard !query.isEmpty else {
            return
        }

        if let exactMatch = availableTagChoices.first(where: { $0.caseInsensitiveCompare(query) == .orderedSame }) {
            addTag(exactMatch)
        } else if let firstMatch = filteredTagChoices.first(where: { !containsTag($0) }) {
            addTag(firstMatch)
        } else if shouldShowCreateTag {
            addTag(query)
        }
    }

    private func addTag(_ tag: String) {
        let trimmedTag = tag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedTag.isEmpty else {
            return
        }

        if !containsTag(trimmedTag) {
            tags.append(trimmedTag)
        }
        closeTagPicker()
    }

    private func removeTag(_ tag: String) {
        tags.removeAll {
            $0.caseInsensitiveCompare(tag) == .orderedSame
        }
    }

    private func containsTag(_ tag: String) -> Bool {
        tags.contains {
            $0.caseInsensitiveCompare(tag) == .orderedSame
        }
    }

    private func closeTagPicker() {
        isTagPickerPresented = false
        resetTagPicker()
    }

    private func resetTagPicker() {
        tagSearchQuery = ""
        tagPickerContentHeight = 0
        hoveredTagChoice = nil
        isCreateTagRowHovered = false
        isTagSearchFocused = false
    }

    private func updateColorHover(id: String, isHovering: Bool) {
        withAnimation(.spring(response: 0.22, dampingFraction: 0.9)) {
            if isHovering {
                hoveredColorID = id
            } else if hoveredColorID == id {
                hoveredColorID = nil
            }
        }
    }

    private func copyColor(_ color: MomentoInspectorColor) {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()

        guard pasteboard.setString(color.normalizedHex, forType: .string) else {
            return
        }

        onColorCopied?()
    }
}

private struct ColorSwatchBoundsKey: PreferenceKey {
    static var defaultValue: [String: Anchor<CGRect>] = [:]

    static func reduce(value: inout [String: Anchor<CGRect>], nextValue: () -> [String: Anchor<CGRect>]) {
        value.merge(nextValue(), uniquingKeysWith: { $1 })
    }
}

private struct TagPickerContentHeightKey: PreferenceKey {
    static var defaultValue: CGFloat = 0

    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = nextValue()
    }
}

private struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(
        proposal: ProposedViewSize,
        subviews: Subviews,
        cache: inout ()
    ) -> CGSize {
        let maxWidth = proposal.width ?? 260
        let rows = rows(maxWidth: maxWidth, subviews: subviews)
        return CGSize(width: maxWidth, height: rows.reduce(0) { $0 + $1.height } + CGFloat(max(rows.count - 1, 0)) * spacing)
    }

    func placeSubviews(
        in bounds: CGRect,
        proposal: ProposedViewSize,
        subviews: Subviews,
        cache: inout ()
    ) {
        let rows = rows(maxWidth: bounds.width, subviews: subviews)
        var y = bounds.minY

        for row in rows {
            var x = bounds.minX
            for item in row.items {
                item.subview.place(
                    at: CGPoint(x: x, y: y),
                    proposal: ProposedViewSize(item.size)
                )
                x += item.size.width + spacing
            }
            y += row.height + spacing
        }
    }

    private func rows(maxWidth: CGFloat, subviews: Subviews) -> [FlowRow] {
        var rows: [FlowRow] = []
        var currentItems: [FlowItem] = []
        var currentWidth: CGFloat = 0
        var currentHeight: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            let nextWidth = currentItems.isEmpty ? size.width : currentWidth + spacing + size.width

            if nextWidth > maxWidth, !currentItems.isEmpty {
                rows.append(FlowRow(items: currentItems, height: currentHeight))
                currentItems = [FlowItem(subview: subview, size: size)]
                currentWidth = size.width
                currentHeight = size.height
            } else {
                currentItems.append(FlowItem(subview: subview, size: size))
                currentWidth = nextWidth
                currentHeight = max(currentHeight, size.height)
            }
        }

        if !currentItems.isEmpty {
            rows.append(FlowRow(items: currentItems, height: currentHeight))
        }

        return rows
    }

    private struct FlowRow {
        var items: [FlowItem]
        var height: CGFloat
    }

    private struct FlowItem {
        var subview: LayoutSubview
        var size: CGSize
    }
}

private extension Color {
    init?(hex: String) {
        var normalized = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        if normalized.hasPrefix("#") {
            normalized.removeFirst()
        }

        guard normalized.count == 6, let value = UInt64(normalized, radix: 16) else {
            return nil
        }

        let red = Double((value & 0xFF0000) >> 16) / 255
        let green = Double((value & 0x00FF00) >> 8) / 255
        let blue = Double(value & 0x0000FF) / 255

        self.init(red: red, green: green, blue: blue)
    }
}

#Preview {
    MomentoInspectorView(
        asset: MomentoInspectorAsset(
            id: "sample",
            title: "Landing Page Reference",
            fileName: "landing-reference.png",
            dimensions: "2400 × 1600",
            colorHexes: ["#315CFF", "#F2C94C", "#111827"],
            filePath: "~/Pictures/Momento/landing-reference.png",
            fileSize: "4.8 MB",
            addedDate: .now,
            kind: "PNG Image"
        ),
        tags: .constant(["UI", "Reference"]),
        notes: .constant("Strong card rhythm and useful spacing reference.")
    )
    .frame(width: 308, height: 720)
}
