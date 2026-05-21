import SwiftUI

struct MomentoSearchBar: View {
    @Environment(\.appLocalization) private var localization

    @Binding var query: String
    var placeholder: String
    var onSubmit: () -> Void
    var onCommandPalette: (() -> Void)?

    @FocusState private var isFocused: Bool

    init(
        query: Binding<String>,
        placeholder: String = "Search assets, tags, colors...",
        onSubmit: @escaping () -> Void = {},
        onCommandPalette: (() -> Void)? = nil
    ) {
        self._query = query
        self.placeholder = placeholder
        self.onSubmit = onSubmit
        self.onCommandPalette = onCommandPalette
    }

    var body: some View {
        HStack(spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: "magnifyingglass")
                    .foregroundStyle(MomentoTheme.secondaryText)

                TextField(placeholder, text: $query)
                    .textFieldStyle(.plain)
                    .focused($isFocused)
                    .onSubmit(onSubmit)

                if !query.isEmpty {
                    Button {
                        query = ""
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(MomentoTheme.tertiaryText)
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(localization.string("Clear search"))
                }
            }
            .font(.system(size: 14))
            .padding(.horizontal, 11)
            .frame(height: 34)
            .background {
                RoundedRectangle(cornerRadius: 10, style: .continuous)
                    .fill(Color(nsColor: .controlBackgroundColor).opacity(isFocused ? 0.9 : 0.68))
                    .overlay {
                        RoundedRectangle(cornerRadius: 10, style: .continuous)
                            .strokeBorder(isFocused ? Color.accentColor.opacity(0.55) : MomentoTheme.subtleStroke, lineWidth: 1)
                    }
            }

            if let onCommandPalette {
                Button(action: onCommandPalette) {
                    HStack(spacing: 6) {
                        Image(systemName: "command")
                        Text("K")
                    }
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(MomentoTheme.secondaryText)
                    .frame(width: 48, height: 34)
                    .background {
                        RoundedRectangle(cornerRadius: 10, style: .continuous)
                            .fill(Color.primary.opacity(0.045))
                            .overlay {
                                RoundedRectangle(cornerRadius: 10, style: .continuous)
                                    .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                            }
                    }
                }
                .buttonStyle(.plain)
                .help(localization.string("Open command palette"))
            }
        }
    }
}

struct MomentoTopBar: View {
    @Environment(\.appLocalization) private var localization

    @Binding var query: String
    var title: String
    var subtitle: String?
    var onSubmitSearch: () -> Void
    var onCommandPalette: () -> Void

    init(
        title: String = "All Assets",
        subtitle: String? = "0 items",
        query: Binding<String>,
        onSubmitSearch: @escaping () -> Void = {},
        onCommandPalette: @escaping () -> Void = {}
    ) {
        self.title = title
        self.subtitle = subtitle
        self._query = query
        self.onSubmitSearch = onSubmitSearch
        self.onCommandPalette = onCommandPalette
    }

    var body: some View {
        HStack(spacing: 18) {
            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(size: 18, weight: .semibold))
                    .lineLimit(1)
                if let subtitle {
                    Text(subtitle)
                        .font(.system(size: 12))
                        .foregroundStyle(MomentoTheme.secondaryText)
                }
            }
            .frame(minWidth: 140, alignment: .leading)

            MomentoSearchBar(
                query: $query,
                placeholder: localization.string("Search assets, tags, colors..."),
                onSubmit: onSubmitSearch,
                onCommandPalette: onCommandPalette
            )
            .frame(maxWidth: 560)

            Spacer()
        }
        .padding(.horizontal, 18)
        .frame(height: MomentoTheme.toolbarHeight)
        .background {
            MomentoVisualEffectView(material: .headerView)
                .ignoresSafeArea()
        }
        .overlay(alignment: .bottom) {
            Divider()
        }
    }
}

#Preview {
    MomentoTopBar(query: .constant(""))
        .frame(width: 860)
}
