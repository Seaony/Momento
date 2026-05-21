import SwiftUI

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

            VStack(alignment: .leading, spacing: 18) {
                Text(localization.string("Create Library"))
                    .font(.system(size: 17, weight: .semibold))

                VStack(alignment: .leading, spacing: 8) {
                    Text(localization.string("Name"))
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(MomentoTheme.secondaryText)

                    TextField(localization.string("Untitled Library"), text: $libraryName)
                        .textFieldStyle(.roundedBorder)
                        .focused($isNameFocused)
                        .onSubmit(continueToDestination)
                }

                HStack(spacing: 10) {
                    Spacer()

                    Button(localization.string("Cancel")) {
                        dismiss()
                    }
                    .buttonStyle(.glass)

                    Button(localization.string("Continue")) {
                        continueToDestination()
                    }
                    .buttonStyle(.glassProminent)
                    .disabled(trimmedLibraryName.isEmpty)
                }
            }
            .padding(22)
            .frame(width: 380)
            .background {
                MomentoGlassBackground(cornerRadius: 18)
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
