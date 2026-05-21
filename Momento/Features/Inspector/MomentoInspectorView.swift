import SwiftUI
import AppKit

struct MomentoInspectorAsset: Identifiable, Hashable {
    var id: String
    var title: String
    var fileName: String
    var previewImage: NSImage?
    var dimensions: String?
    var colorHexes: [String]
    var filePath: String?
    var fileSize: String?
    var addedDate: Date?
    var kind: String?

    init(
        id: String,
        title: String,
        fileName: String,
        previewImage: NSImage? = nil,
        dimensions: String? = nil,
        colorHexes: [String] = [],
        filePath: String? = nil,
        fileSize: String? = nil,
        addedDate: Date? = nil,
        kind: String? = nil
    ) {
        self.id = id
        self.title = title
        self.fileName = fileName
        self.previewImage = previewImage
        self.dimensions = dimensions
        self.colorHexes = colorHexes
        self.filePath = filePath
        self.fileSize = fileSize
        self.addedDate = addedDate
        self.kind = kind
    }
}

struct MomentoInspectorView: View {
    var asset: MomentoInspectorAsset?
    @Binding var tags: [String]
    @Binding var notes: String
    var onRevealInFinder: (() -> Void)?

    @State private var pendingTag = ""

    init(
        asset: MomentoInspectorAsset?,
        tags: Binding<[String]> = .constant([]),
        notes: Binding<String> = .constant(""),
        onRevealInFinder: (() -> Void)? = nil
    ) {
        self.asset = asset
        self._tags = tags
        self._notes = notes
        self.onRevealInFinder = onRevealInFinder
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text("Inspector")
                    .font(.system(size: 14, weight: .semibold))
                Spacer()
                Image(systemName: "sidebar.right")
                    .foregroundStyle(MomentoTheme.secondaryText)
            }
            .padding(.horizontal, 16)
            .frame(height: MomentoTheme.toolbarHeight)

            Divider()

            if let asset {
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        preview(asset)
                        metadata(asset)
                        tagEditor
                        colorSection(asset.colorHexes)
                        notesEditor
                        fileSection(asset)
                    }
                    .padding(16)
                }
                .scrollIndicators(.never)
            } else {
                emptyState
            }
        }
        .frame(
            minWidth: MomentoTheme.inspectorMinWidth,
            idealWidth: MomentoTheme.inspectorWidth,
            maxWidth: MomentoTheme.inspectorMaxWidth
        )
        .background {
            MomentoVisualEffectView(material: .contentBackground)
                .ignoresSafeArea()
        }
    }

    private func preview(_ asset: MomentoInspectorAsset) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            ZStack {
                RoundedRectangle(cornerRadius: MomentoTheme.panelRadius, style: .continuous)
                    .fill(Color.primary.opacity(0.045))

                if let image = asset.previewImage {
                    Image(nsImage: image)
                        .resizable()
                        .scaledToFit()
                        .padding(12)
                } else {
                    VStack(spacing: 8) {
                        Image(systemName: "photo")
                            .font(.system(size: 32))
                            .foregroundStyle(MomentoTheme.tertiaryText)
                        Text(asset.fileName)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(MomentoTheme.secondaryText)
                            .lineLimit(1)
                    }
                    .padding()
                }
            }
            .aspectRatio(1.25, contentMode: .fit)
            .overlay {
                RoundedRectangle(cornerRadius: MomentoTheme.panelRadius, style: .continuous)
                    .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
            }

            Text(asset.title)
                .font(.system(size: 15, weight: .semibold))
                .lineLimit(2)
        }
    }

    private func metadata(_ asset: MomentoInspectorAsset) -> some View {
        inspectorSection("Details") {
            if let kind = asset.kind {
                infoRow("Kind", kind)
            }
            if let dimensions = asset.dimensions {
                infoRow("Dimensions", dimensions)
            }
            if let fileSize = asset.fileSize {
                infoRow("Size", fileSize)
            }
            if let addedDate = asset.addedDate {
                infoRow("Added", addedDate.formatted(date: .abbreviated, time: .shortened))
            }
        }
    }

    private var tagEditor: some View {
        inspectorSection("Tags") {
            if tags.isEmpty {
                Text("No tags")
                    .font(.system(size: 12))
                    .foregroundStyle(MomentoTheme.secondaryText)
            } else {
                FlowLayout(spacing: 6) {
                    ForEach(tags, id: \.self) { tag in
                        HStack(spacing: 5) {
                            Text(tag)
                            Button {
                                tags.removeAll { $0 == tag }
                            } label: {
                                Image(systemName: "xmark")
                                    .font(.system(size: 9, weight: .bold))
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel("Remove \(tag)")
                        }
                        .font(.system(size: 12, weight: .medium))
                        .padding(.horizontal, 8)
                        .frame(height: 24)
                        .background(Color.accentColor.opacity(0.12), in: Capsule())
                    }
                }
            }

            HStack(spacing: 8) {
                TextField("Add tag", text: $pendingTag)
                    .textFieldStyle(.plain)
                    .font(.system(size: 12))
                    .onSubmit(addPendingTag)

                Button(action: addPendingTag) {
                    Image(systemName: "plus")
                }
                .buttonStyle(.plain)
                .disabled(pendingTag.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
            .padding(.horizontal, 9)
            .frame(height: 30)
            .background {
                RoundedRectangle(cornerRadius: 8, style: .continuous)
                    .fill(Color.primary.opacity(0.045))
                    .overlay {
                        RoundedRectangle(cornerRadius: 8, style: .continuous)
                            .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                    }
            }
        }
    }

    private func colorSection(_ colors: [String]) -> some View {
        inspectorSection("Colors") {
            if colors.isEmpty {
                Text("No colors extracted")
                    .font(.system(size: 12))
                    .foregroundStyle(MomentoTheme.secondaryText)
            } else {
                HStack(spacing: 7) {
                    ForEach(colors, id: \.self) { hex in
                        Circle()
                            .fill(Color(hex: hex) ?? .clear)
                            .frame(width: 22, height: 22)
                            .overlay {
                                Circle()
                                    .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                            }
                            .help(hex)
                    }
                    Spacer()
                }
            }
        }
    }

    private var notesEditor: some View {
        inspectorSection("Notes") {
            TextEditor(text: $notes)
                .font(.system(size: 12))
                .scrollContentBackground(.hidden)
                .frame(minHeight: 86)
                .padding(6)
                .background {
                    RoundedRectangle(cornerRadius: 8, style: .continuous)
                        .fill(Color.primary.opacity(0.045))
                        .overlay {
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                        }
                }
        }
    }

    private func fileSection(_ asset: MomentoInspectorAsset) -> some View {
        inspectorSection("File") {
            infoRow("Name", asset.fileName)
            if let filePath = asset.filePath {
                infoRow("Path", filePath)
            }

            if onRevealInFinder != nil {
                Button(action: { onRevealInFinder?() }) {
                    Label("Reveal in Finder", systemImage: "finder")
                        .frame(maxWidth: .infinity)
                        .frame(height: 30)
                }
                .buttonStyle(.bordered)
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "sidebar.right")
                .font(.system(size: 30))
                .foregroundStyle(MomentoTheme.tertiaryText)
            Text("No asset selected")
                .font(.system(size: 13, weight: .medium))
            Text("Select an item to inspect metadata, tags, colors, and file details.")
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .multilineTextAlignment(.center)
                .frame(maxWidth: 220)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(24)
    }

    private func inspectorSection<Content: View>(
        _ title: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(title.uppercased())
                .font(.system(size: 10, weight: .semibold))
                .foregroundStyle(MomentoTheme.tertiaryText)
            content()
        }
    }

    private func infoRow(_ label: String, _ value: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            Text(label)
                .font(.system(size: 12))
                .foregroundStyle(MomentoTheme.secondaryText)
                .frame(width: 74, alignment: .leading)
            Text(value)
                .font(.system(size: 12))
                .textSelection(.enabled)
                .lineLimit(2)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func addPendingTag() {
        let tag = pendingTag.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !tag.isEmpty, !tags.contains(tag) else { return }
        tags.append(tag)
        pendingTag = ""
    }
}

private struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(
        proposal: ProposedViewSize,
        subviews: Subviews,
        cache: inout ()
    ) -> CGSize {
        let maxWidth = proposal.width ?? 260
        let rows = rows(maxWidth: maxWidth, subviews: subviews)
        return CGSize(width: maxWidth, height: rows.reduce(0) { $0 + $1.height } + CGFloat(max(rows.count - 1, 0)) * spacing)
    }

    func placeSubviews(
        in bounds: CGRect,
        proposal: ProposedViewSize,
        subviews: Subviews,
        cache: inout ()
    ) {
        let rows = rows(maxWidth: bounds.width, subviews: subviews)
        var y = bounds.minY

        for row in rows {
            var x = bounds.minX
            for item in row.items {
                item.subview.place(
                    at: CGPoint(x: x, y: y),
                    proposal: ProposedViewSize(item.size)
                )
                x += item.size.width + spacing
            }
            y += row.height + spacing
        }
    }

    private func rows(maxWidth: CGFloat, subviews: Subviews) -> [FlowRow] {
        var rows: [FlowRow] = []
        var currentItems: [FlowItem] = []
        var currentWidth: CGFloat = 0
        var currentHeight: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            let nextWidth = currentItems.isEmpty ? size.width : currentWidth + spacing + size.width

            if nextWidth > maxWidth, !currentItems.isEmpty {
                rows.append(FlowRow(items: currentItems, height: currentHeight))
                currentItems = [FlowItem(subview: subview, size: size)]
                currentWidth = size.width
                currentHeight = size.height
            } else {
                currentItems.append(FlowItem(subview: subview, size: size))
                currentWidth = nextWidth
                currentHeight = max(currentHeight, size.height)
            }
        }

        if !currentItems.isEmpty {
            rows.append(FlowRow(items: currentItems, height: currentHeight))
        }

        return rows
    }

    private struct FlowRow {
        var items: [FlowItem]
        var height: CGFloat
    }

    private struct FlowItem {
        var subview: LayoutSubview
        var size: CGSize
    }
}

private extension Color {
    init?(hex: String) {
        var normalized = hex.trimmingCharacters(in: .whitespacesAndNewlines)
        if normalized.hasPrefix("#") {
            normalized.removeFirst()
        }

        guard normalized.count == 6, let value = UInt64(normalized, radix: 16) else {
            return nil
        }

        let red = Double((value & 0xFF0000) >> 16) / 255
        let green = Double((value & 0x00FF00) >> 8) / 255
        let blue = Double(value & 0x0000FF) / 255

        self.init(red: red, green: green, blue: blue)
    }
}

#Preview {
    MomentoInspectorView(
        asset: MomentoInspectorAsset(
            id: "sample",
            title: "Landing Page Reference",
            fileName: "landing-reference.png",
            dimensions: "2400 × 1600",
            colorHexes: ["#315CFF", "#F2C94C", "#111827"],
            filePath: "~/Pictures/Momento/landing-reference.png",
            fileSize: "4.8 MB",
            addedDate: .now,
            kind: "PNG Image"
        ),
        tags: .constant(["UI", "Reference"]),
        notes: .constant("Strong card rhythm and useful spacing reference.")
    )
    .frame(width: 308, height: 720)
}
