// 中文注释：本文件集中实现资源库、文件夹等名称输入和确认弹窗。
import SwiftUI

private let createLibraryDialogWidth: CGFloat = 460
private let createLibraryDialogIconSize: CGFloat = 48
private let createLibraryDialogFieldHeight: CGFloat = 36
private let deleteLibraryDialogWidth: CGFloat = 430
private let folderNameDialogWidth: CGFloat = 420
private let destructiveConfirmationDialogWidth: CGFloat = 430

enum MomentoLibraryNameDialogMode {
    case create
    case edit

    var titleKey: String {
        switch self {
        case .create:
            "Create Library"
        case .edit:
            "Edit Library"
        }
    }

    var subtitleKey: String {
        switch self {
        case .create:
            "Enter a name for this library, then choose where to save it."
        case .edit:
            "Change this library name. Assets and files stay in the same library package."
        }
    }

    var primaryActionKey: String {
        switch self {
        case .create:
            "Choose Save Location"
        case .edit:
            "Save Changes"
        }
    }
}

struct MomentoCreateLibraryDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var mode: MomentoLibraryNameDialogMode
    var initialName: String
    var onSubmit: (String) -> Void

    @State private var libraryName: String
    @State private var isCancelButtonHovered = false
    @State private var isPrimaryButtonHovered = false
    @FocusState private var isNameFocused: Bool

    private var trimmedLibraryName: String {
        libraryName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    init(
        mode: MomentoLibraryNameDialogMode = .create,
        isPresented: Binding<Bool>,
        initialName: String,
        onSubmit: @escaping (String) -> Void
    ) {
        self.mode = mode
        self._isPresented = isPresented
        self.initialName = initialName
        self.onSubmit = onSubmit
        self._libraryName = State(initialValue: initialName)
    }

    var body: some View {
        ZStack {
            MomentoDialogBackdrop(dismiss: dismiss)

            HStack(alignment: .top, spacing: 16) {
                dialogIcon

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(localization.string(mode.titleKey))
                            .font(.system(size: 18, weight: .semibold))

                        Text(localization.string(mode.subtitleKey))
                            .font(.system(size: 13, weight: .regular))
                            .lineSpacing(2)
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    TextField(text: $libraryName, prompt: Text(localization.string("Library Name")).foregroundStyle(MomentoTheme.secondaryText)) {
                        Text(localization.string("Library Name"))
                    }
                        .textFieldStyle(.plain)
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(libraryName.isEmpty ? MomentoTheme.secondaryText : MomentoTheme.primaryText)
                        .padding(.horizontal, 12)
                        .frame(height: createLibraryDialogFieldHeight)
                        .background {
                            createLibraryNameFieldBackground
                        }
                        .focused($isNameFocused)
                        .onSubmit(submit)

                    HStack(spacing: 14) {
                        Button {
                            dismiss()
                        } label: {
                            Text(localization.string("Cancel"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glass)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isCancelButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isCancelButtonHovered = isHovered
                        }

                        Button {
                            submit()
                        } label: {
                            Text(localization.string(mode.primaryActionKey))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glassProminent)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isPrimaryButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isPrimaryButtonHovered = isHovered
                        }
                        .disabled(trimmedLibraryName.isEmpty)
                    }
                }
            }
            .padding(.horizontal, 34)
            .padding(.vertical, 30)
            .frame(width: createLibraryDialogWidth)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.18)), cornerRadius: 14)
            }
            .contentShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onTapGesture {}
            .onAppear {
                isNameFocused = true
            }
        }
        .transition(.opacity)
        .onExitCommand {
            dismiss()
        }
    }

    private var dialogIcon: some View {
        Image(systemName: "archivebox.fill")
            .font(.system(size: 22, weight: .semibold))
            .foregroundStyle(.white)
            .frame(width: createLibraryDialogIconSize, height: createLibraryDialogIconSize)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.accentColor), cornerRadius: 14)
            }
    }

    private var createLibraryNameFieldBackground: some View {
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        return Color.clear
            .glassEffect(.regular, in: shape)
            .overlay {
                shape.strokeBorder(isNameFocused ? Color.accentColor : MomentoTheme.subtleStroke, lineWidth: 2)
            }
    }

    private func submit() {
        guard !trimmedLibraryName.isEmpty else {
            return
        }

        let name = trimmedLibraryName
        dismiss()
        onSubmit(name)
    }

    private func dismiss() {
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }
    }
}

struct MomentoDeleteLibraryDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var libraryName: String
    var onConfirm: () -> Void

    @State private var isCancelButtonHovered = false
    @State private var isDeleteButtonHovered = false

    var body: some View {
        ZStack {
            MomentoDialogBackdrop(dismiss: dismiss)

            HStack(alignment: .top, spacing: 16) {
                deleteIcon

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(localization.string("Delete Library"))
                            .font(.system(size: 18, weight: .semibold))

                        Text(localization.format("Delete library warning: %@", libraryName))
                            .font(.system(size: 13, weight: .regular))
                            .lineSpacing(2)
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    HStack(spacing: 14) {
                        Button {
                            dismiss()
                        } label: {
                            Text(localization.string("Cancel"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glass)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isCancelButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isCancelButtonHovered = isHovered
                        }

                        Button(role: .destructive) {
                            confirmDelete()
                        } label: {
                            Text(localization.string("Delete"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glassProminent)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .tint(.red)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isDeleteButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isDeleteButtonHovered = isHovered
                        }
                    }
                }
            }
            .padding(.horizontal, 34)
            .padding(.vertical, 30)
            .frame(width: deleteLibraryDialogWidth)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.18)), cornerRadius: 14)
            }
            .contentShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onTapGesture {}
        }
        .transition(.opacity)
        .onExitCommand {
            dismiss()
        }
    }

    private var deleteIcon: some View {
        Image(systemName: "trash.fill")
            .font(.system(size: 22, weight: .semibold))
            .foregroundStyle(.white)
            .frame(width: createLibraryDialogIconSize, height: createLibraryDialogIconSize)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.red), cornerRadius: 14)
            }
    }

    private func confirmDelete() {
        dismiss()
        onConfirm()
    }

    private func dismiss() {
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }
    }
}

enum MomentoFolderNameDialogMode {
    case create
    case edit

    var titleKey: String {
        switch self {
        case .create:
            "New Folder"
        case .edit:
            "Edit Folder"
        }
    }

    var subtitleKey: String {
        switch self {
        case .create:
            "Enter a name for this folder."
        case .edit:
            "Change this folder name. Assets stay in the same folder."
        }
    }

    var primaryActionKey: String {
        switch self {
        case .create:
            "Create Folder"
        case .edit:
            "Save Changes"
        }
    }

    var iconName: String {
        switch self {
        case .create:
            "folder.fill.badge.plus"
        case .edit:
            "folder.fill"
        }
    }
}

struct MomentoFolderNameDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var mode: MomentoFolderNameDialogMode
    var initialName: String
    var onSubmit: (String) -> Void

    @State private var folderName: String
    @State private var isCancelButtonHovered = false
    @State private var isPrimaryButtonHovered = false
    @FocusState private var isNameFocused: Bool

    private var trimmedFolderName: String {
        folderName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    init(
        mode: MomentoFolderNameDialogMode = .create,
        isPresented: Binding<Bool>,
        initialName: String,
        onSubmit: @escaping (String) -> Void
    ) {
        self.mode = mode
        self._isPresented = isPresented
        self.initialName = initialName
        self.onSubmit = onSubmit
        self._folderName = State(initialValue: initialName)
    }

    var body: some View {
        ZStack {
            MomentoDialogBackdrop(dismiss: dismiss)

            HStack(alignment: .top, spacing: 16) {
                dialogIcon

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(localization.string(mode.titleKey))
                            .font(.system(size: 18, weight: .semibold))

                        Text(localization.string(mode.subtitleKey))
                            .font(.system(size: 13, weight: .regular))
                            .lineSpacing(2)
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    TextField(text: $folderName, prompt: Text(localization.string("Folder Name")).foregroundStyle(MomentoTheme.secondaryText)) {
                        Text(localization.string("Folder Name"))
                    }
                    .textFieldStyle(.plain)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(folderName.isEmpty ? MomentoTheme.secondaryText : MomentoTheme.primaryText)
                    .padding(.horizontal, 12)
                    .frame(height: createLibraryDialogFieldHeight)
                    .background {
                        folderNameFieldBackground
                    }
                    .focused($isNameFocused)
                    .onSubmit(submit)

                    HStack(spacing: 14) {
                        Button {
                            dismiss()
                        } label: {
                            Text(localization.string("Cancel"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glass)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isCancelButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isCancelButtonHovered = isHovered
                        }

                        Button {
                            submit()
                        } label: {
                            Text(localization.string(mode.primaryActionKey))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glassProminent)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isPrimaryButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isPrimaryButtonHovered = isHovered
                        }
                        .disabled(trimmedFolderName.isEmpty)
                    }
                }
            }
            .padding(.horizontal, 34)
            .padding(.vertical, 30)
            .frame(width: folderNameDialogWidth)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.18)), cornerRadius: 14)
            }
            .contentShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onTapGesture {}
            .onAppear {
                isNameFocused = true
            }
        }
        .transition(.opacity)
        .onExitCommand {
            dismiss()
        }
    }

    private var dialogIcon: some View {
        Image(systemName: mode.iconName)
            .font(.system(size: 22, weight: .semibold))
            .foregroundStyle(.white)
            .frame(width: createLibraryDialogIconSize, height: createLibraryDialogIconSize)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.accentColor), cornerRadius: 14)
            }
    }

    private var folderNameFieldBackground: some View {
        let shape = RoundedRectangle(cornerRadius: 10, style: .continuous)

        return Color.clear
            .glassEffect(.regular, in: shape)
            .overlay {
                shape.strokeBorder(isNameFocused ? Color.accentColor : MomentoTheme.subtleStroke, lineWidth: 2)
            }
    }

    private func submit() {
        guard !trimmedFolderName.isEmpty else {
            return
        }

        let name = trimmedFolderName
        dismiss()
        onSubmit(name)
    }

    private func dismiss() {
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }
    }
}

struct MomentoDeleteFolderDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var folderName: String
    var onConfirm: () -> Void

    @State private var isCancelButtonHovered = false
    @State private var isDeleteButtonHovered = false

    var body: some View {
        ZStack {
            MomentoDialogBackdrop(dismiss: dismiss)

            HStack(alignment: .top, spacing: 16) {
                deleteIcon

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(localization.string("Delete Folder"))
                            .font(.system(size: 18, weight: .semibold))

                        Text(localization.format("Delete folder warning: %@", folderName))
                            .font(.system(size: 13, weight: .regular))
                            .lineSpacing(2)
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    HStack(spacing: 14) {
                        Button {
                            dismiss()
                        } label: {
                            Text(localization.string("Cancel"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glass)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isCancelButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isCancelButtonHovered = isHovered
                        }

                        Button(role: .destructive) {
                            confirmDelete()
                        } label: {
                            Text(localization.string("Delete Folder"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glassProminent)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .tint(.red)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isDeleteButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isDeleteButtonHovered = isHovered
                        }
                    }
                }
            }
            .padding(.horizontal, 34)
            .padding(.vertical, 30)
            .frame(width: deleteLibraryDialogWidth)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.18)), cornerRadius: 14)
            }
            .contentShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onTapGesture {}
        }
        .transition(.opacity)
        .onExitCommand {
            dismiss()
        }
    }

    private var deleteIcon: some View {
        Image(systemName: "folder.fill.badge.minus")
            .font(.system(size: 22, weight: .semibold))
            .foregroundStyle(.white)
            .frame(width: createLibraryDialogIconSize, height: createLibraryDialogIconSize)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.red), cornerRadius: 14)
            }
    }

    private func confirmDelete() {
        dismiss()
        onConfirm()
    }

    private func dismiss() {
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }
    }
}

struct MomentoDialogBackdrop: View {
    var dismiss: () -> Void

    var body: some View {
        Rectangle()
            .fill(Color.black.opacity(0.10))
            .ignoresSafeArea()
            .contentShape(Rectangle())
            .onTapGesture(perform: dismiss)
    }
}

struct MomentoDestructiveConfirmationDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var iconName: String
    var title: String
    var message: String
    var confirmTitle: String
    var onConfirm: () -> Void

    @State private var isCancelButtonHovered = false
    @State private var isConfirmButtonHovered = false

    var body: some View {
        ZStack {
            MomentoDialogBackdrop(dismiss: dismiss)

            HStack(alignment: .top, spacing: 16) {
                Image(systemName: iconName)
                    .font(.system(size: 22, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: createLibraryDialogIconSize, height: createLibraryDialogIconSize)
                    .background {
                        MomentoGlassBackground(glass: .regular.tint(Color.red), cornerRadius: 14)
                    }

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(title)
                            .font(.system(size: 18, weight: .semibold))

                        Text(message)
                            .font(.system(size: 13, weight: .regular))
                            .lineSpacing(2)
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    HStack(spacing: 14) {
                        Button {
                            dismiss()
                        } label: {
                            Text(localization.string("Cancel"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glass)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isCancelButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isCancelButtonHovered = isHovered
                        }

                        Button(role: .destructive) {
                            confirm()
                        } label: {
                            Text(confirmTitle)
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glassProminent)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .tint(.red)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isConfirmButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isConfirmButtonHovered = isHovered
                        }
                    }
                }
            }
            .padding(.horizontal, 34)
            .padding(.vertical, 30)
            .frame(width: destructiveConfirmationDialogWidth)
            .background {
                MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.18)), cornerRadius: 14)
            }
            .contentShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onTapGesture {}
        }
        .transition(.opacity)
        .onExitCommand {
            dismiss()
        }
    }

    private func confirm() {
        dismiss()
        onConfirm()
    }

    private func dismiss() {
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }
    }
}

private extension View {
    func createLibraryDialogButtonHoverFeedback(isHovered: Bool, reduceMotion: Bool) -> some View {
        scaleEffect(isHovered && !reduceMotion ? 1.035 : 1)
            .brightness(isHovered ? 0.08 : 0)
            .animation(reduceMotion ? nil : .smooth(duration: 0.16), value: isHovered)
    }
}

#Preview {
    MomentoCreateLibraryDialog(
        isPresented: .constant(true),
        initialName: "Untitled Library",
        onSubmit: { _ in }
    )
    .frame(width: 640, height: 420)
}
