import SwiftUI

struct MomentoLibraryWelcomeView: View {
    @Environment(\.appLocalization) private var localization

    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void

    var body: some View {
        VStack(spacing: 18) {
            Image(systemName: "books.vertical")
                .font(.system(size: 44, weight: .regular))
                .foregroundStyle(Color.accentColor)

            VStack(spacing: 6) {
                Text(localization.string("No Library Open"))
                    .font(.system(size: 22, weight: .semibold))
                Text(localization.string("Create or open a Momento library to start organizing assets."))
                    .font(.system(size: 13))
                    .foregroundStyle(MomentoTheme.secondaryText)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 360)
            }

            HStack(spacing: 10) {
                Button {
                    onCreateLibrary()
                } label: {
                    Label(localization.string("Create Library"), systemImage: "plus")
                }
                .buttonStyle(.borderedProminent)

                Button {
                    onOpenLibrary()
                } label: {
                    Label(localization.string("Open Library"), systemImage: "folder")
                }
                .buttonStyle(.bordered)
            }
            .controlSize(.large)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(nsColor: .windowBackgroundColor))
    }
}

#Preview {
    MomentoLibraryWelcomeView(
        onCreateLibrary: {},
        onOpenLibrary: {}
    )
        .environment(\.appLocalization, AppLocalization(language: .system))
        .frame(width: 760, height: 520)
}
