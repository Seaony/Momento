// 中文注释：标签管理页负责标签列表、重命名和删除入口，关联数量来自 store 汇总。
import SwiftUI

struct MomentoTagManagementView: View {
    private static let rowCornerRadius: CGFloat = 10
    private static let assetCountColumnWidth: CGFloat = 118
    private static let actionColumnWidth: CGFloat = 196
    private static let actionButtonCornerRadius: CGFloat = 13
    private static let actionButtonHeight: CGFloat = 34
    private static let toolbarHeight: CGFloat = 38

    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var tags: [TagSummary]
    var onCreateTag: (String) -> Void
    var onRenameTag: (TagItem.ID, String) -> Void
    var onDeleteTag: (TagItem.ID) -> Void

    @State private var editingTagID: TagItem.ID?
    @State private var draftTagName = ""
    @State private var searchQuery = ""
    @State private var isCreatingTag = false
    @State private var draftNewTagName = ""
    @State private var deletingTag: TagSummary?
    @State private var hoveredRowID: TagItem.ID?
    @State private var hoveredMenuID: TagItem.ID?
    @State private var hoveredActionID: String?
    @State private var isCreateButtonHovered = false
    @FocusState private var isNameFieldFocused: Bool
    @FocusState private var isCreateFieldFocused: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            managementToolbar

            if filteredTags.isEmpty && !isCreatingTag {
                emptyState
            } else {
                tagList(filteredTags)
            }
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 14)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .overlay {
            if let deletingTag {
                MomentoDestructiveConfirmationDialog(
                    isPresented: deleteDialogIsPresented,
                    iconName: "trash.fill",
                    title: localization.string("Delete Tag"),
                    message: localization.format("Delete tag warning: %@", deletingTag.tag.name),
                    confirmTitle: localization.string("Delete Tag"),
                    onConfirm: {
                        onDeleteTag(deletingTag.id)
                    }
                )
                .zIndex(30)
            }
        }
        .animation(.smooth(duration: reduceMotion ? 0.08 : 0.18), value: deletingTag != nil)
    }

    private var managementToolbar: some View {
        HStack(spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(MomentoTheme.secondaryText)

                TextField(localization.string("Search tags"), text: $searchQuery)
                    .textFieldStyle(.plain)
                    .font(.system(size: 13, weight: .medium))
            }
            .padding(.horizontal, 11)
            .frame(maxWidth: 260)
            .frame(height: Self.toolbarHeight)
            .background {
                MomentoGlassBackground(
                    style: .regular.interactive(true),
                    cornerRadius: 13
                )
            }

            Spacer(minLength: 12)

            Button(action: beginCreatingTag) {
                Label(localization.string("New Tag"), systemImage: "plus")
                    .font(.system(size: 13, weight: .semibold))
                    .labelStyle(.titleAndIcon)
                    .padding(.horizontal, 14)
                    .frame(height: Self.toolbarHeight)
                    .background {
                        MomentoGlassBackground(
                            style: .regular.tint(MomentoTheme.contrastTint(
                                lightOpacity: isCreateButtonHovered ? 0.07 : 0.03,
                                darkOpacity: isCreateButtonHovered ? 0.12 : 0.04
                            )).interactive(true),
                            cornerRadius: 13
                        )
                    }
            }
            .buttonStyle(.plain)
            .pointerStyle(.link)
            .accessibilityLabel(localization.string("New Tag"))
            .onHover { hovering in
                withAnimation(.smooth(duration: 0.12)) {
                    isCreateButtonHovered = hovering
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func tagList(_ visibleTags: [TagSummary]) -> some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 0) {
                headerRow

                if isCreatingTag {
                    tagCreationRow

                    if !visibleTags.isEmpty {
                        Rectangle()
                            .fill(MomentoTheme.subtleGlassStroke)
                            .frame(height: 1)
                            .padding(.leading, 42)
                    }
                }

                ForEach(visibleTags) { tag in
                    tagRow(tag)

                    if tag.id != visibleTags.last?.id {
                        Rectangle()
                            .fill(MomentoTheme.subtleGlassStroke)
                            .frame(height: 1)
                            .padding(.leading, 42)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .topLeading)
        }
        .scrollIndicators(.automatic)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var headerRow: some View {
        HStack(spacing: 12) {
            Text(localization.string("Name"))
                .frame(maxWidth: .infinity, alignment: .leading)

            Text(localization.string("Linked Assets"))
                .frame(width: Self.assetCountColumnWidth, alignment: .leading)

            Color.clear
                .frame(width: Self.actionColumnWidth)
        }
        .font(.system(size: 11, weight: .semibold))
        .foregroundStyle(MomentoTheme.tertiaryText)
        .padding(.horizontal, 14)
        .frame(height: 38)
        .frame(maxWidth: .infinity)
    }

    private var tagCreationRow: some View {
        HStack(spacing: 12) {
            Image(systemName: "number")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(MomentoTheme.secondaryText)
                .frame(width: 18)

            TextField(localization.string("New Tag"), text: $draftNewTagName)
                .textFieldStyle(.plain)
                .font(.system(size: 13, weight: .medium))
                .focused($isCreateFieldFocused)
                .padding(.horizontal, 9)
                .frame(height: 30)
                .frame(maxWidth: .infinity, alignment: .leading)
                .layoutPriority(1)
                .background {
                    MomentoGlassBackground(
                        style: .regular.interactive(true),
                        cornerRadius: 9
                    )
                    .overlay {
                        RoundedRectangle(cornerRadius: 9, style: .continuous)
                            .strokeBorder(Color.accentColor.opacity(0.75), lineWidth: 1)
                    }
                }
                .onSubmit(commitNewTag)

            Color.clear
                .frame(width: Self.assetCountColumnWidth, alignment: .leading)

            HStack(spacing: 8) {
                actionButton(
                    id: "new-tag-save",
                    title: localization.string("Save Changes"),
                    accessibilityLabel: localization.string("Save Changes"),
                    isProminent: true
                ) {
                    commitNewTag()
                }
                .disabled(trimmedNewTagName.isEmpty)

                actionButton(
                    id: "new-tag-cancel",
                    title: localization.string("Cancel"),
                    accessibilityLabel: localization.string("Cancel"),
                    isProminent: false
                ) {
                    cancelCreatingTag()
                }
            }
            .frame(width: Self.actionColumnWidth, alignment: .trailing)
        }
        .foregroundStyle(MomentoTheme.primaryText)
        .padding(.horizontal, 14)
        .frame(height: 52)
        .frame(maxWidth: .infinity)
        .task {
            isCreateFieldFocused = true
        }
    }

    private func tagRow(_ summary: TagSummary) -> some View {
        let isEditing = editingTagID == summary.id
        let isHovered = hoveredRowID == summary.id
        let rowShape = RoundedRectangle(cornerRadius: Self.rowCornerRadius, style: .continuous)

        return HStack(spacing: 12) {
            Image(systemName: "number")
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(MomentoTheme.secondaryText)
                .frame(width: 18)

            if isEditing {
                TextField(localization.string("Tag"), text: $draftTagName)
                    .textFieldStyle(.plain)
                    .font(.system(size: 13, weight: .medium))
                    .focused($isNameFieldFocused)
                    .padding(.horizontal, 9)
                    .frame(height: 30)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .layoutPriority(1)
                    .background {
                        MomentoGlassBackground(
                            style: .regular.interactive(true),
                            cornerRadius: 9
                        )
                            .overlay {
                                RoundedRectangle(cornerRadius: 9, style: .continuous)
                                    .strokeBorder(Color.accentColor.opacity(0.75), lineWidth: 1)
                            }
                    }
                    .onSubmit {
                        commitEditing(summary)
                    }
            } else {
                Text(summary.tag.name)
                    .font(.system(size: 13, weight: .medium))
                    .lineLimit(1)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .layoutPriority(1)
            }

            Text(localization.itemCount(summary.assetCount))
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(MomentoTheme.secondaryText)
                .frame(width: Self.assetCountColumnWidth, alignment: .leading)

            HStack(spacing: 8) {
                if isEditing {
                    actionButton(
                        id: "\(summary.id)-save",
                        title: localization.string("Save Changes"),
                        accessibilityLabel: localization.string("Save Changes"),
                        isProminent: true
                    ) {
                        commitEditing(summary)
                    }
                    .disabled(trimmedDraftTagName.isEmpty)

                    actionButton(
                        id: "\(summary.id)-cancel",
                        title: localization.string("Cancel"),
                        accessibilityLabel: localization.string("Cancel"),
                        isProminent: false
                    ) {
                        cancelEditing()
                    }
                } else {
                    tagMenu(summary)
                }
            }
            .frame(width: Self.actionColumnWidth, alignment: .trailing)
        }
        .foregroundStyle(MomentoTheme.primaryText)
        .padding(.horizontal, 14)
        .frame(height: 52)
        .frame(maxWidth: .infinity)
        .background {
            rowShape.fill(isHovered ? MomentoTheme.contrastTint(lightOpacity: 0.04, darkOpacity: 0.05) : Color.clear)
        }
        .contentShape(rowShape)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.12)) {
                hoveredRowID = hovering ? summary.id : nil
            }
        }
        .task(id: editingTagID) {
            if editingTagID == summary.id {
                isNameFieldFocused = true
            }
        }
    }

    private func tagMenu(_ summary: TagSummary) -> some View {
        let isHovered = hoveredMenuID == summary.id
        let shape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        return Menu {
            Button {
                beginEditing(summary)
            } label: {
                Label(localization.string("Edit Tag"), systemImage: "pencil")
            }

            Button(role: .destructive) {
                withAnimation(.smooth(duration: reduceMotion ? 0.08 : 0.18)) {
                    deletingTag = summary
                }
            } label: {
                Label(localization.string("Delete Tag"), systemImage: "trash")
            }
        } label: {
            Image(systemName: "ellipsis")
                .font(.system(size: 14, weight: .semibold))
                .frame(width: 30, height: 28)
                .momentoSurface(
                    .regular.tint(MomentoTheme.contrastTint(
                        lightOpacity: isHovered ? 0.07 : 0.03,
                        darkOpacity: isHovered ? 0.12 : 0.04
                    )).interactive(true),
                    in: shape
                )
                .contentShape(shape)
        }
        .menuStyle(.button)
        .buttonStyle(.plain)
        .foregroundStyle(MomentoTheme.primaryText)
        .contentShape(shape)
        .pointerStyle(.link)
        .accessibilityLabel(localization.string("More Actions"))
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.12)) {
                hoveredMenuID = hovering ? summary.id : nil
            }
        }
    }

    private func actionButton(
        id: String,
        title: String,
        accessibilityLabel: String,
        isProminent: Bool,
        role: ButtonRole? = nil,
        action: @escaping () -> Void
    ) -> some View {
        let isHovered = hoveredActionID == id
        let shape = RoundedRectangle(cornerRadius: Self.actionButtonCornerRadius, style: .continuous)
        let foregroundColor = role == .destructive ? Color.red.opacity(0.9) : MomentoTheme.primaryText

        return Button(role: role, action: action) {
            Text(title)
                .font(.system(size: 13, weight: .semibold))
                .lineLimit(1)
                .minimumScaleFactor(0.82)
                .foregroundStyle(foregroundColor)
                .padding(.horizontal, 16)
                .frame(height: Self.actionButtonHeight)
                .background {
                    Color.clear
                        .momentoSurface(
                            isProminent
                                ? .regular.tint(Color.accentColor).interactive(true)
                                : .regular.interactive(true),
                            in: shape
                        )
                }
                .overlay {
                    if isHovered {
                        shape.fill(MomentoTheme.contrastTint(
                            lightOpacity: isProminent ? 0.08 : 0.055,
                            darkOpacity: isProminent ? 0.16 : 0.1
                        ))
                    }
                }
                .overlay {
                    shape.strokeBorder(
                        MomentoTheme.contrastTint(
                            lightOpacity: isHovered ? 0.16 : 0.09,
                            darkOpacity: isHovered ? 0.28 : 0.14
                        ),
                        lineWidth: 1
                    )
                }
                .contentShape(shape)
        }
        .buttonStyle(.plain)
        .scaleEffect(isHovered ? 1.035 : 1)
        .shadow(color: isHovered ? Color.black.opacity(0.2) : Color.clear, radius: 8, y: 3)
        .animation(.smooth(duration: 0.14), value: isHovered)
        .contentShape(shape)
        .pointerStyle(.link)
        .accessibilityLabel(accessibilityLabel)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                if hovering {
                    hoveredActionID = id
                } else if hoveredActionID == id {
                    hoveredActionID = nil
                }
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "number")
                .font(.system(size: 34, weight: .regular))
                .foregroundStyle(MomentoTheme.tertiaryText)

            Text(tags.isEmpty ? localization.string("No tags") : localization.string("No matching tags"))
                .font(.system(size: 15, weight: .semibold))
        }
        .padding(28)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var trimmedDraftTagName: String {
        draftTagName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var trimmedNewTagName: String {
        draftNewTagName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var filteredTags: [TagSummary] {
        let trimmedQuery = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedQuery.isEmpty else { return tags }

        return tags.filter {
            $0.tag.name.localizedCaseInsensitiveContains(trimmedQuery)
        }
    }

    private var deleteDialogIsPresented: Binding<Bool> {
        Binding {
            deletingTag != nil
        } set: { isPresented in
            if !isPresented {
                deletingTag = nil
            }
        }
    }

    private func beginEditing(_ summary: TagSummary) {
        cancelCreatingTag()
        editingTagID = summary.id
        draftTagName = summary.tag.name
        isNameFieldFocused = true
    }

    private func commitEditing(_ summary: TagSummary) {
        let name = trimmedDraftTagName
        guard !name.isEmpty else {
            return
        }

        editingTagID = nil
        isNameFieldFocused = false

        guard name != summary.tag.name else {
            return
        }

        onRenameTag(summary.id, name)
    }

    private func cancelEditing() {
        editingTagID = nil
        draftTagName = ""
        isNameFieldFocused = false
    }

    private func beginCreatingTag() {
        cancelEditing()
        isCreatingTag = true
        draftNewTagName = ""
        isCreateFieldFocused = true
    }

    private func commitNewTag() {
        let name = trimmedNewTagName
        guard !name.isEmpty else {
            return
        }

        onCreateTag(name)
        isCreatingTag = false
        draftNewTagName = ""
        searchQuery = ""
        isCreateFieldFocused = false
    }

    private func cancelCreatingTag() {
        isCreatingTag = false
        draftNewTagName = ""
        isCreateFieldFocused = false
    }
}

#Preview {
    MomentoTagManagementView(
        tags: [
            TagSummary(tag: TagItem(name: "Travel"), assetCount: 18),
            TagSummary(tag: TagItem(name: "Reference"), assetCount: 7)
        ],
        onCreateTag: { _ in },
        onRenameTag: { _, _ in },
        onDeleteTag: { _ in }
    )
    .frame(width: 820, height: 520)
}
