// 中文注释：命令面板只负责展示和触发当前上下文命令，不持久化业务数据。
import SwiftUI

struct MomentoCommand: Identifiable, Hashable {
    var id: String
    var title: String
    var subtitle: String?
    var systemImage: String
    var shortcut: String?

    init(
        id: String,
        title: String,
        subtitle: String? = nil,
        systemImage: String,
        shortcut: String? = nil
    ) {
        self.id = id
        self.title = title
        self.subtitle = subtitle
        self.systemImage = systemImage
        self.shortcut = shortcut
    }
}

struct MomentoCommandPalette: View {
    @Environment(\.appLocalization) private var localization

    @Binding var isPresented: Bool
    var commands: [MomentoCommand]
    var onSelect: (MomentoCommand) -> Void

    @State private var query = ""
    @State private var selectionID: MomentoCommand.ID?
    @FocusState private var isSearchFocused: Bool

    private var filteredCommands: [MomentoCommand] {
        let trimmedQuery = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedQuery.isEmpty else { return commands }

        return commands.filter { command in
            command.title.localizedCaseInsensitiveContains(trimmedQuery)
                || command.subtitle?.localizedCaseInsensitiveContains(trimmedQuery) == true
        }
    }

    var body: some View {
        ZStack {
            Color.clear
                .ignoresSafeArea()
                .contentShape(Rectangle())
                .onTapGesture {
                    dismiss()
                }

            VStack(spacing: 0) {
                HStack(spacing: 10) {
                    Image(systemName: "command")
                        .foregroundStyle(MomentoTheme.secondaryText)

                    TextField(localization.string("Type a command or search..."), text: $query)
                        .textFieldStyle(.plain)
                        .font(.system(size: 16))
                        .focused($isSearchFocused)
                }
                .padding(.horizontal, 16)
                .frame(height: 54)

                Divider()

                ScrollView {
                    LazyVStack(spacing: 4) {
                        ForEach(filteredCommands) { command in
                            MomentoCommandRow(command: command, isSelected: selectionID == command.id) {
                                select(command)
                            }
                            .onHover { hovering in
                                guard hovering else { return }
                                selectionID = command.id
                            }
                        }

                        if filteredCommands.isEmpty {
                            VStack(spacing: 8) {
                                Image(systemName: "magnifyingglass")
                                    .font(.system(size: 24))
                                    .foregroundStyle(MomentoTheme.tertiaryText)
                                Text(localization.string("No commands found"))
                                    .font(.system(size: 13, weight: .medium))
                                Text(localization.string("Try a different action or keyword."))
                                    .font(.system(size: 12))
                                    .foregroundStyle(MomentoTheme.secondaryText)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 36)
                        }
                    }
                    .padding(8)
                }
                .frame(maxHeight: 360)
            }
            .frame(width: 560)
            .background {
                MomentoGlassBackground(cornerRadius: 18)
            }
            .onAppear {
                selectionID = filteredCommands.first?.id
                isSearchFocused = true
            }
            .onChange(of: query) {
                selectionID = filteredCommands.first?.id
            }
        }
        .transition(.opacity)
        .onExitCommand {
            dismiss()
        }
    }

    private func select(_ command: MomentoCommand) {
        onSelect(command)
        dismiss()
    }

    private func dismiss() {
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }
    }
}

private struct MomentoCommandRow: View {
    var command: MomentoCommand
    var isSelected: Bool
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                Image(systemName: command.systemImage)
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(isSelected ? Color.accentColor : MomentoTheme.secondaryText)
                    .frame(width: 26, height: 26)
                    .background {
                        MomentoGlassBackground(cornerRadius: 7)
                    }

                VStack(alignment: .leading, spacing: 2) {
                    Text(command.title)
                        .font(.system(size: 13, weight: .medium))
                    if let subtitle = command.subtitle {
                        Text(subtitle)
                            .font(.system(size: 12))
                            .foregroundStyle(MomentoTheme.secondaryText)
                    }
                }

                Spacer()

                if let shortcut = command.shortcut {
                    Text(shortcut)
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(MomentoTheme.tertiaryText)
                        .padding(.horizontal, 7)
                        .frame(height: 22)
                        .background {
                            MomentoGlassBackground(cornerRadius: 6)
                        }
                }
            }
            .padding(.horizontal, 10)
            .frame(height: 48)
            .background {
                rowBackground
            }
            .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        }
        .buttonStyle(.plain)
    }

    @ViewBuilder
    private var rowBackground: some View {
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        if isSelected {
            Color.clear
                .glassEffect(.regular.tint(Color.accentColor), in: shape)
        } else {
            Color.clear
        }
    }
}

struct MomentoCommandPaletteModifier: ViewModifier {
    @Binding var isPresented: Bool
    var commands: [MomentoCommand]
    var onSelect: (MomentoCommand) -> Void

    func body(content: Content) -> some View {
        content
            .overlay {
                if isPresented {
                    MomentoCommandPalette(
                        isPresented: $isPresented,
                        commands: commands,
                        onSelect: onSelect
                    )
                    .zIndex(20)
                }
            }
    }
}

extension View {
    func momentoCommandPalette(
        isPresented: Binding<Bool>,
        commands: [MomentoCommand],
        onSelect: @escaping (MomentoCommand) -> Void
    ) -> some View {
        modifier(
            MomentoCommandPaletteModifier(
                isPresented: isPresented,
                commands: commands,
                onSelect: onSelect
            )
        )
    }
}

extension Array where Element == MomentoCommand {
    static func momentoDefaultCommands(localization: AppLocalization) -> [MomentoCommand] {
        [
            MomentoCommand(id: "import", title: localization.string("Import Assets"), subtitle: localization.string("Choose files or folders"), systemImage: "square.and.arrow.down", shortcut: "I"),
            MomentoCommand(id: "import-library", title: localization.string("Import Library"), subtitle: localization.string("Copy a library package into Momento"), systemImage: "square.and.arrow.down.on.square"),
            MomentoCommand(id: "export-library", title: localization.string("Export Library"), subtitle: localization.string("Copy the current library package"), systemImage: "square.and.arrow.up.on.square"),
            MomentoCommand(id: "new-folder", title: localization.string("New Folder"), subtitle: localization.string("Create a folder in the current library"), systemImage: "folder.badge.plus", shortcut: "N"),
            MomentoCommand(id: "toggle-inspector", title: localization.string("Toggle Inspector"), subtitle: localization.string("Show or hide asset details"), systemImage: "sidebar.right", shortcut: "⌥⌘I"),
            MomentoCommand(id: "quick-preview", title: localization.string("Quick Preview"), subtitle: localization.string("Preview the selected asset"), systemImage: "eye", shortcut: "Space"),
            MomentoCommand(id: "trash", title: localization.string("Move to Trash"), subtitle: localization.string("Remove selected assets from the library"), systemImage: "trash", shortcut: "⌫")
        ]
    }

    static var momentoDefaultCommands: [MomentoCommand] {
        momentoDefaultCommands(localization: AppLocalization(language: .system))
    }
}

#Preview {
    MomentoCommandPalette(
        isPresented: .constant(true),
        commands: .momentoDefaultCommands,
        onSelect: { _ in }
    )
    .frame(width: 760, height: 520)
}
