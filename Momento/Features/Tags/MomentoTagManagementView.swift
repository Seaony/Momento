import SwiftUI

struct MomentoTagManagementView: View {
    private static let rowCornerRadius: CGFloat = 10
    private static let actionColumnWidth: CGFloat = 86
    private static let actionButtonWidth: CGFloat = 34
    private static let actionButtonHeight: CGFloat = 30

    @Environment(\.appLocalization) private var localization

    var tags: [TagSummary]
    var onRenameTag: (TagItem.ID, String) -> Void
    var onDeleteTag: (TagItem.ID) -> Void

    @State private var editingTagID: TagItem.ID?
    @State private var draftTagName = ""
    @State private var deletingTag: TagSummary?
    @State private var hoveredRowID: TagItem.ID?
    @State private var hoveredMenuID: TagItem.ID?
    @FocusState private var isNameFieldFocused: Bool

    var body: some View {
        Group {
            if tags.isEmpty {
                emptyState
            } else {
                tagList
            }
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 14)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        .alert(
            localization.string("Delete Tag"),
            isPresented: deleteAlertIsPresented
        ) {
            Button(localization.string("Cancel"), role: .cancel) {
                deletingTag = nil
            }
            Button(localization.string("Delete Tag"), role: .destructive) {
                if let deletingTag {
                    onDeleteTag(deletingTag.id)
                }
                deletingTag = nil
            }
        } message: {
            if let deletingTag {
                Text(localization.format("Delete tag warning: %@", deletingTag.tag.name))
            }
        }
    }

    private var tagList: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 0) {
                headerRow

                ForEach(tags) { tag in
                    tagRow(tag)

                    if tag.id != tags.last?.id {
                        Rectangle()
                            .fill(Color.white.opacity(0.08))
                            .frame(height: 1)
                            .padding(.leading, 42)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .topLeading)
        }
        .scrollIndicators(.never)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private var headerRow: some View {
        HStack(spacing: 12) {
            Text(localization.string("Name"))
                .frame(maxWidth: .infinity, alignment: .leading)

            Text(localization.string("Linked Assets"))
                .frame(width: 112, alignment: .trailing)

            Color.clear
                .frame(width: Self.actionColumnWidth)
        }
        .font(.system(size: 11, weight: .semibold))
        .foregroundStyle(MomentoTheme.tertiaryText)
        .padding(.horizontal, 14)
        .frame(height: 38)
        .frame(maxWidth: .infinity)
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
                        RoundedRectangle(cornerRadius: 9, style: .continuous)
                            .fill(Color.white.opacity(0.08))
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
                .frame(width: 112, alignment: .trailing)

            HStack(spacing: 6) {
                if isEditing {
                    actionButton(
                        systemImage: "checkmark",
                        accessibilityLabel: localization.string("Save Changes"),
                        foregroundColor: Color.accentColor
                    ) {
                        commitEditing(summary)
                    }
                    .disabled(trimmedDraftTagName.isEmpty)

                    actionButton(
                        systemImage: "xmark",
                        accessibilityLabel: localization.string("Cancel"),
                        foregroundColor: MomentoTheme.secondaryText
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
            rowShape.fill(isHovered ? Color.white.opacity(0.05) : Color.clear)
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
                deletingTag = summary
            } label: {
                Label(localization.string("Delete Tag"), systemImage: "trash")
            }
        } label: {
            Image(systemName: "ellipsis")
                .font(.system(size: 14, weight: .semibold))
                .frame(width: 30, height: 28)
                .background {
                    if isHovered {
                        shape.fill(MomentoTheme.sidebarIconHoverBackground)
                    }
                }
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
        systemImage: String,
        accessibilityLabel: String,
        foregroundColor: Color? = nil,
        role: ButtonRole? = nil,
        action: @escaping () -> Void
    ) -> some View {
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)
        let resolvedForegroundColor = foregroundColor ?? (role == .destructive ? Color.red.opacity(0.9) : MomentoTheme.primaryText)

        return Button(role: role, action: action) {
            Image(systemName: systemImage)
                .symbolRenderingMode(.monochrome)
                .font(.system(size: 13, weight: .bold))
                .foregroundStyle(resolvedForegroundColor)
                .frame(width: Self.actionButtonWidth, height: Self.actionButtonHeight)
        }
        .buttonStyle(.glass)
        .buttonBorderShape(.roundedRectangle(radius: 10))
        .controlSize(.small)
        .contentShape(shape)
        .pointerStyle(.link)
        .accessibilityLabel(accessibilityLabel)
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "number")
                .font(.system(size: 34, weight: .regular))
                .foregroundStyle(MomentoTheme.tertiaryText)

            Text(localization.string("No tags"))
                .font(.system(size: 15, weight: .semibold))
        }
        .padding(28)
        .frame(maxWidth: 320)
        .background {
            MomentoGlassBackground(cornerRadius: 18)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var trimmedDraftTagName: String {
        draftTagName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var deleteAlertIsPresented: Binding<Bool> {
        Binding {
            deletingTag != nil
        } set: { isPresented in
            if !isPresented {
                deletingTag = nil
            }
        }
    }

    private func beginEditing(_ summary: TagSummary) {
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
}

#Preview {
    MomentoTagManagementView(
        tags: [
            TagSummary(tag: TagItem(name: "Travel"), assetCount: 18),
            TagSummary(tag: TagItem(name: "Reference"), assetCount: 7)
        ],
        onRenameTag: { _, _ in },
        onDeleteTag: { _ in }
    )
    .frame(width: 820, height: 520)
}
