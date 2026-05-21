import SwiftUI

struct MomentoSidebarItem: Identifiable, Hashable {
    var id: String
    var title: String
    var systemImage: String
    var count: Int?
    var tint: Color?

    init(
        id: String,
        title: String,
        systemImage: String,
        count: Int? = nil,
        tint: Color? = nil
    ) {
        self.id = id
        self.title = title
        self.systemImage = systemImage
        self.count = count
        self.tint = tint
    }
}

struct MomentoSidebarSection: Identifiable, Hashable {
    var id: String
    var title: String
    var items: [MomentoSidebarItem]
    var isCollapsible: Bool

    init(
        id: String,
        title: String,
        items: [MomentoSidebarItem],
        isCollapsible: Bool = true
    ) {
        self.id = id
        self.title = title
        self.items = items
        self.isCollapsible = isCollapsible
    }
}

struct MomentoSidebarView: View {
    @Environment(\.appLocalization) private var localization

    var sections: [MomentoSidebarSection]
    @Binding var selection: MomentoSidebarItem.ID?
    var onItemContextMenu: ((MomentoSidebarItem) -> AnyView)?

    @State private var collapsedSectionIDs: Set<MomentoSidebarSection.ID> = []

    init(
        sections: [MomentoSidebarSection] = .momentoDefaultSections,
        selection: Binding<MomentoSidebarItem.ID?>,
        onItemContextMenu: ((MomentoSidebarItem) -> AnyView)? = nil
    ) {
        self.sections = sections
        self._selection = selection
        self.onItemContextMenu = onItemContextMenu
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Image(systemName: "square.grid.2x2")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 28, height: 28)
                    .background(.blue.gradient, in: RoundedRectangle(cornerRadius: 7, style: .continuous))

                VStack(alignment: .leading, spacing: 1) {
                    Text("Momento")
                        .font(.system(size: 14, weight: .semibold))
                    Text(localization.string("Local library"))
                        .font(.system(size: 11))
                        .foregroundStyle(MomentoTheme.secondaryText)
                }

                Spacer()
            }
            .padding(.horizontal, 14)
            .padding(.top, 14)
            .padding(.bottom, 10)

            ScrollView {
                LazyVStack(alignment: .leading, spacing: 14) {
                    ForEach(sections) { section in
                        sectionView(section)
                    }
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
            }
            .scrollIndicators(.never)

            Divider()

            HStack(spacing: 8) {
                Image(systemName: "externaldrive")
                Text(localization.string("No library selected"))
                Spacer()
            }
            .font(.system(size: 12))
            .foregroundStyle(MomentoTheme.secondaryText)
            .padding(12)
        }
        .frame(
            minWidth: MomentoTheme.sidebarMinWidth,
            idealWidth: MomentoTheme.sidebarWidth,
            maxWidth: MomentoTheme.sidebarMaxWidth
        )
        .background {
            MomentoVisualEffectView(material: .sidebar)
                .ignoresSafeArea()
        }
    }

    private func sectionView(_ section: MomentoSidebarSection) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            Button {
                guard section.isCollapsible else { return }
                withAnimation(.smooth(duration: 0.18)) {
                    if collapsedSectionIDs.contains(section.id) {
                        collapsedSectionIDs.remove(section.id)
                    } else {
                        collapsedSectionIDs.insert(section.id)
                    }
                }
            } label: {
                HStack(spacing: 5) {
                    Image(systemName: "chevron.down")
                        .font(.system(size: 9, weight: .bold))
                        .rotationEffect(.degrees(collapsedSectionIDs.contains(section.id) ? -90 : 0))
                        .opacity(section.isCollapsible ? 0.75 : 0)

                    Text(section.title.uppercased())
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(MomentoTheme.tertiaryText)
                    Spacer()
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 6)

            if !collapsedSectionIDs.contains(section.id) {
                VStack(spacing: 2) {
                    ForEach(section.items) { item in
                        MomentoSidebarRow(item: item, isSelected: selection == item.id) {
                            withAnimation(.smooth(duration: 0.16)) {
                                selection = item.id
                            }
                        }
                        .contextMenu {
                            if let menu = onItemContextMenu?(item) {
                                menu
                            }
                        }
                    }
                }
                .transition(.opacity.combined(with: .move(edge: .top)))
            }
        }
    }
}

private struct MomentoSidebarRow: View {
    @Environment(\.appLocalization) private var localization

    var item: MomentoSidebarItem
    var isSelected: Bool
    var action: () -> Void

    @State private var isHovered = false

    var body: some View {
        Button(action: action) {
            HStack(spacing: 8) {
                Image(systemName: item.systemImage)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(item.tint ?? (isSelected ? Color.accentColor : MomentoTheme.secondaryText))
                    .frame(width: 18)

                Text(item.title)
                    .font(.system(size: 13, weight: isSelected ? .medium : .regular))
                    .lineLimit(1)

                Spacer(minLength: 8)

                if let count = item.count {
                    Text(count.formatted(.number.locale(localization.locale)))
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(MomentoTheme.tertiaryText)
                }
            }
            .padding(.horizontal, 8)
            .frame(height: 28)
            .background {
                RoundedRectangle(cornerRadius: MomentoTheme.rowRadius, style: .continuous)
                    .fill(backgroundStyle)
            }
            .contentShape(RoundedRectangle(cornerRadius: MomentoTheme.rowRadius, style: .continuous))
        }
        .buttonStyle(.plain)
        .onHover { hovering in
            withAnimation(.smooth(duration: 0.14)) {
                isHovered = hovering
            }
        }
    }

    private var backgroundStyle: Color {
        if isSelected {
            return Color.accentColor.opacity(0.16)
        }
        if isHovered {
            return Color.primary.opacity(0.06)
        }
        return .clear
    }
}

extension Array where Element == MomentoSidebarSection {
    static func momentoDefaultSections(localization: AppLocalization) -> [MomentoSidebarSection] {
        [
            MomentoSidebarSection(
                id: "library",
                title: localization.string("Library"),
                items: [
                    MomentoSidebarItem(id: "all-assets", title: localization.string("All Assets"), systemImage: "photo.on.rectangle.angled", count: 0),
                    MomentoSidebarItem(id: "recent", title: localization.string("Recent"), systemImage: "clock", count: 0)
                ],
                isCollapsible: false
            ),
            MomentoSidebarSection(
                id: "folders",
                title: localization.string("Folders"),
                items: [
                    MomentoSidebarItem(id: "folder-inspiration", title: localization.string("Inspiration"), systemImage: "folder", count: 0),
                    MomentoSidebarItem(id: "folder-screenshots", title: localization.string("Screenshots"), systemImage: "folder", count: 0)
                ]
            ),
            MomentoSidebarSection(
                id: "tags",
                title: localization.string("Tags"),
                items: [
                    MomentoSidebarItem(id: "tag-ui", title: "UI", systemImage: "tag", count: 0, tint: .blue),
                    MomentoSidebarItem(id: "tag-brand", title: "Brand", systemImage: "tag", count: 0, tint: .purple)
                ]
            ),
            MomentoSidebarSection(
                id: "favorites",
                title: localization.string("Favorites"),
                items: [
                    MomentoSidebarItem(id: "favorites", title: localization.string("Starred Assets"), systemImage: "star", count: 0, tint: .yellow)
                ],
                isCollapsible: false
            ),
            MomentoSidebarSection(
                id: "trash",
                title: localization.string("Trash"),
                items: [
                    MomentoSidebarItem(id: "trash", title: localization.string("Trash"), systemImage: "trash", count: 0)
                ],
                isCollapsible: false
            )
        ]
    }

    static var momentoDefaultSections: [MomentoSidebarSection] {
        momentoDefaultSections(localization: AppLocalization(language: .system))
    }
}

#Preview {
    MomentoSidebarView(selection: .constant("all-assets"))
        .frame(width: 236, height: 620)
}
