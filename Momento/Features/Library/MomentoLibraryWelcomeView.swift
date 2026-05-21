import SwiftUI

struct MomentoLibraryWelcomeView: View {
    @Environment(\.appLocalization) private var localization

    var recentLibraries: [RecentLibraryReference]
    var onCreateLibrary: () -> Void
    var onOpenLibrary: () -> Void
    var onSwitchLibrary: (RecentLibraryReference.ID) -> Void

    var body: some View {
        ZStack(alignment: .topLeading) {
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

            libraryMenu
                .padding(.top, 14)
                .padding(.leading, 14)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(nsColor: .windowBackgroundColor))
    }

    private var libraryMenu: some View {
        Menu {
            Button(localization.string("Create Library"), action: onCreateLibrary)
            Button(localization.string("Open Library"), action: onOpenLibrary)

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
                    Text(localization.string("No library selected"))
                        .font(.system(size: 11))
                        .foregroundStyle(MomentoTheme.secondaryText)
                        .lineLimit(1)
                }

                Image(systemName: "chevron.up.chevron.down")
                    .font(.system(size: 9, weight: .semibold))
                    .foregroundStyle(MomentoTheme.tertiaryText)
            }
            .padding(.vertical, 6)
            .padding(.horizontal, 7)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

#Preview {
    MomentoLibraryWelcomeView(
        recentLibraries: [],
        onCreateLibrary: {},
        onOpenLibrary: {},
        onSwitchLibrary: { _ in }
    )
        .environment(\.appLocalization, AppLocalization(language: .system))
        .frame(width: 760, height: 520)
}
