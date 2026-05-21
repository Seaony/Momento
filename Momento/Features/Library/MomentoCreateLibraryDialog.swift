import SwiftUI

private let createLibraryDialogWidth: CGFloat = 460
private let createLibraryDialogIconSize: CGFloat = 48
private let createLibraryDialogFieldHeight: CGFloat = 36

struct MomentoCreateLibraryDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var initialName: String
    var onContinue: (String) -> Void

    @State private var libraryName: String
    @State private var isCancelButtonHovered = false
    @State private var isChooseLocationButtonHovered = false
    @FocusState private var isNameFocused: Bool

    private var trimmedLibraryName: String {
        libraryName.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    init(
        isPresented: Binding<Bool>,
        initialName: String,
        onContinue: @escaping (String) -> Void
    ) {
        self._isPresented = isPresented
        self.initialName = initialName
        self.onContinue = onContinue
        self._libraryName = State(initialValue: initialName)
    }

    var body: some View {
        ZStack {
            Rectangle()
                .fill(.ultraThinMaterial)
                .overlay(Color.black.opacity(0.18))
                .ignoresSafeArea()
                .contentShape(Rectangle())
                .onTapGesture {
                    dismiss()
                }

            HStack(alignment: .top, spacing: 16) {
                dialogIcon

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(localization.string("Create Library"))
                            .font(.system(size: 18, weight: .semibold))

                        Text(localization.string("Enter a name for this library, then choose where to save it."))
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
                        .onSubmit(continueToDestination)

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
                            continueToDestination()
                        } label: {
                            Text(localization.string("Choose Save Location"))
                                .font(.system(size: 14, weight: .semibold))
                                .padding(.horizontal, 18)
                                .padding(.vertical, 6)
                        }
                        .buttonStyle(.glassProminent)
                        .buttonBorderShape(.capsule)
                        .controlSize(.large)
                        .contentShape(Capsule(style: .continuous))
                        .pointerStyle(.link)
                        .createLibraryDialogButtonHoverFeedback(isHovered: isChooseLocationButtonHovered, reduceMotion: reduceMotion)
                        .onHover { isHovered in
                            isChooseLocationButtonHovered = isHovered
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

    private func continueToDestination() {
        guard !trimmedLibraryName.isEmpty else {
            return
        }

        let name = trimmedLibraryName
        dismiss()
        onContinue(name)
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
        onContinue: { _ in }
    )
    .frame(width: 640, height: 420)
}
