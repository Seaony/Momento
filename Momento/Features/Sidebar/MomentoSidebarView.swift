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
    @Environment(\.openSettings) private var openSettings

    var sections: [MomentoSidebarSection]
    @Binding var selection: MomentoSidebarItem.ID?
    var libraryName: String?
    var recentLibraries: [RecentLibraryReference]
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void
    var onCloseLibrary: () -> Void
    var onItemContextMenu: ((MomentoSidebarItem) -> AnyView)?

    @State private var collapsedSectionIDs: Set<MomentoSidebarSection.ID> = []

    init(
        sections: [MomentoSidebarSection] = .momentoDefaultSections,
        selection: Binding<MomentoSidebarItem.ID?>,
        libraryName: String? = nil,
        recentLibraries: [RecentLibraryReference] = [],
        onCreateLibrary: @escaping () -> Void = {},
        onOpenLibrary: @escaping () -> Void = {},
        onSwitchLibrary: @escaping (RecentLibraryReference.ID) -> Void = { _ in },
        onCloseLibrary: @escaping () -> Void = {},
        onItemContextMenu: ((MomentoSidebarItem) -> AnyView)? = nil
    ) {
        self.sections = sections
        self._selection = selection
        self.libraryName = libraryName
        self.recentLibraries = recentLibraries
        self.onCreateLibrary = onCreateLibrary
        self.onOpenLibrary = onOpenLibrary
        self.onSwitchLibrary = onSwitchLibrary
        self.onCloseLibrary = onCloseLibrary
        self.onItemContextMenu = onItemContextMenu
    }

    var body: some View {
        VStack(spacing: 0) {
            libraryMenu
            .padding(.horizontal, 14)
            .padding(.top, MomentoTheme.floatingSidebarTitlebarContentInset)
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

            sidebarBottomSeparator
            bottomActionBar
        }
        .frame(
            minWidth: MomentoTheme.sidebarMinWidth,
            idealWidth: MomentoTheme.sidebarWidth,
            maxWidth: MomentoTheme.sidebarMaxWidth
        )
        .background {
            MomentoGlassBackground(cornerRadius: MomentoTheme.floatingSidebarRadius)
        }
        .clipShape(sidebarShape)
        .overlay {
            sidebarShape.strokeBorder(MomentoTheme.subtleStroke.opacity(0.42), lineWidth: 1)
        }
    }

    private var sidebarShape: RoundedRectangle {
        RoundedRectangle(cornerRadius: MomentoTheme.floatingSidebarRadius, style: .continuous)
    }

    private var sidebarBottomSeparator: some View {
        Rectangle()
            .fill(MomentoTheme.subtleStroke.opacity(0.24))
            .frame(height: 0.5)
            .padding(.horizontal, 14)
    }

    private var bottomActionBar: some View {
        HStack(spacing: 10) {
            sidebarFooterButton(
                systemImage: "trash",
                label: localization.string("Trash")
            ) {
                withAnimation(.smooth(duration: 0.16)) {
                    selection = "trash"
                }
            }

            sidebarFooterButton(
                systemImage: "gear",
                label: localization.string("Settings"),
                action: openSettings.callAsFunction
            )

            sidebarFooterIcon(
                systemImage: "questionmark.circle",
                label: localization.string("Help Center")
            )
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 9)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func sidebarFooterButton(
        systemImage: String,
        label: String,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            Image(systemName: systemImage)
                .font(.system(size: 14, weight: .medium))
                .foregroundStyle(MomentoTheme.secondaryText)
                .frame(width: 28, height: 28)
                .contentShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
        }
        .buttonStyle(.plain)
        .help(label)
        .accessibilityLabel(label)
    }

    private func sidebarFooterIcon(systemImage: String, label: String) -> some View {
        Image(systemName: systemImage)
            .font(.system(size: 14, weight: .medium))
            .foregroundStyle(MomentoTheme.secondaryText)
            .frame(width: 28, height: 28)
            .help(label)
            .accessibilityLabel(label)
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

    private var libraryMenu: some View {
        Menu {
            Button(localization.string("Create Library"), action: onCreateLibrary)
            Button(localization.string("Open Library"), action: onOpenLibrary)

            if libraryName != nil {
                Divider()

                Button {
                    onCloseLibrary()
                } label: {
                    Label(localization.string("Close Library"), systemImage: "xmark.circle")
                }
            }

            if !recentLibraries.isEmpty {
                Divider()

                ForEach(recentLibraries) { library in
                    Button(library.name) {
                        onSwitchLibrary(library.id)
                    }
                }
            }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "square.grid.2x2")
                    .font(.system(size: 15, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 28, height: 28)
                    .background(.blue.gradient, in: RoundedRectangle(cornerRadius: 7, style: .continuous))

                VStack(alignment: .leading, spacing: 1) {
                    Text("Momento")
                        .font(.system(size: 14, weight: .semibold))
                    Text(libraryName ?? localization.string("No library selected"))
                        .font(.system(size: 11))
                        .foregroundStyle(MomentoTheme.secondaryText)
                        .lineLimit(1)
                }

                Spacer()

                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 9, weight: .semibold))
                    .foregroundStyle(MomentoTheme.tertiaryText)
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
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
                rowBackground
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

    @ViewBuilder
    private var rowBackground: some View {
        let shape = RoundedRectangle(cornerRadius: MomentoTheme.rowRadius, style: .continuous)

        if isSelected {
            Color.clear
                .glassEffect(.regular.tint(Color.accentColor), in: shape)
        } else if isHovered {
            Color.clear
                .glassEffect(.regular, in: shape)
        } else {
            Color.clear
        }
    }
}

extension Array where Element == MomentoSidebarSection {
    static func momentoDefaultSections(localization: AppLocalization) -> [MomentoSidebarSection] {
        [
            MomentoSidebarSection(
                id: "favorites",
                title: localization.string("Favorites"),
                items: [
                    MomentoSidebarItem(id: "favorites", title: localization.string("Starred Assets"), systemImage: "star", count: 0, tint: .yellow)
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
