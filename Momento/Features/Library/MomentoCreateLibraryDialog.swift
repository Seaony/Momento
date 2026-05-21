import SwiftUI

private let createLibraryDialogWidth: CGFloat = 520
private let createLibraryDialogIconSize: CGFloat = 52
private let createLibraryDialogFieldHeight: CGFloat = 44

struct MomentoCreateLibraryDialog: View {
    @Environment(\.appLocalization) private var localization
    @Binding var isPresented: Bool

    var initialName: String
    var onContinue: (String) -> Void

    @State private var libraryName: String
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
            Color.clear
                .ignoresSafeArea()
                .contentShape(Rectangle())
                .onTapGesture {
                    dismiss()
                }

            HStack(alignment: .top, spacing: 18) {
                dialogIcon

                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 8) {
                        Text(localization.string("Create Library"))
                            .font(.system(size: 20, weight: .semibold))

                        Text(localization.string("Enter a name for this library, then choose where to save it."))
                            .font(.system(size: 14, weight: .regular))
                            .lineSpacing(4)
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .fixedSize(horizontal: false, vertical: true)
                    }

                    TextField(localization.string("Library Name"), text: $libraryName)
                        .textFieldStyle(.plain)
                        .font(.system(size: 15, weight: .regular))
                        .foregroundStyle(MomentoTheme.secondaryText)
                        .padding(.horizontal, 14)
                        .frame(height: createLibraryDialogFieldHeight)
                        .background {
                            createLibraryNameFieldBackground
                        }
                        .focused($isNameFocused)
                        .onSubmit(continueToDestination)

                    HStack(spacing: 12) {
                        Button {
                            dismiss()
                        } label: {
                            Text(localization.string("Cancel"))
                                .font(.system(size: 15, weight: .semibold))
                                .frame(width: 132, height: 44)
                        }
                        .buttonStyle(.glass)

                        Button {
                            continueToDestination()
                        } label: {
                            Text(localization.string("Choose Save Location"))
                                .font(.system(size: 15, weight: .semibold))
                                .frame(width: 220, height: 44)
                        }
                        .buttonStyle(.glassProminent)
                        .disabled(trimmedLibraryName.isEmpty)
                    }
                }
            }
            .padding(.horizontal, 28)
            .padding(.vertical, 24)
            .frame(width: createLibraryDialogWidth)
            .background {
                MomentoGlassBackground(cornerRadius: 14)
            }
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
            .font(.system(size: 23, weight: .semibold))
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

#Preview {
    MomentoCreateLibraryDialog(
        isPresented: .constant(true),
        initialName: "Untitled Library",
        onContinue: { _ in }
    )
    .frame(width: 640, height: 420)
}
