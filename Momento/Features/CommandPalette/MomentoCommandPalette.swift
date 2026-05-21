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
            Color.black.opacity(0.18)
                .ignoresSafeArea()
                .onTapGesture {
                    dismiss()
                }

            VStack(spacing: 0) {
                HStack(spacing: 10) {
                    Image(systemName: "command")
                        .foregroundStyle(MomentoTheme.secondaryText)

                    TextField("Type a command or search...", text: $query)
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
                                Text("No commands found")
                                    .font(.system(size: 13, weight: .medium))
                                Text("Try a different action or keyword.")
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
                MomentoGlassBackground(material: .hudWindow, cornerRadius: 18, strokeOpacity: 0.2)
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
                        RoundedRectangle(cornerRadius: 7, style: .continuous)
                            .fill(Color.primary.opacity(0.055))
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
                            RoundedRectangle(cornerRadius: 6, style: .continuous)
                                .fill(Color.primary.opacity(0.05))
                        }
                }
            }
            .padding(.horizontal, 10)
            .frame(height: 48)
            .background {
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .fill(isSelected ? Color.accentColor.opacity(0.14) : .clear)
            }
            .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        }
        .buttonStyle(.plain)
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
    static var momentoDefaultCommands: [MomentoCommand] {
        [
            MomentoCommand(id: "import", title: "Import Assets", subtitle: "Choose files or folders", systemImage: "square.and.arrow.down", shortcut: "I"),
            MomentoCommand(id: "new-folder", title: "New Folder", subtitle: "Create a folder in the current library", systemImage: "folder.badge.plus", shortcut: "N"),
            MomentoCommand(id: "toggle-inspector", title: "Toggle Inspector", subtitle: "Show or hide asset details", systemImage: "sidebar.right", shortcut: "⌥⌘I"),
            MomentoCommand(id: "quick-preview", title: "Quick Preview", subtitle: "Preview the selected asset", systemImage: "eye", shortcut: "Space"),
            MomentoCommand(id: "trash", title: "Move to Trash", subtitle: "Remove selected assets from the library", systemImage: "trash", shortcut: "⌫")
        ]
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
