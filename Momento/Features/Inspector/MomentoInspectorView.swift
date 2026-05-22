import SwiftUI
import AppKit

struct MomentoInspectorColor: Identifiable, Hashable {
    var hex: String
    var coverage: Double?

    var id: String {
        if let coverage {
            return "\(hex)-\(coverage)"
        }
        return hex
    }

    var normalizedHex: String {
        let trimmedHex = hex.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        if trimmedHex.hasPrefix("#") {
            return trimmedHex
        }

        return "#\(trimmedHex)"
    }

    var helpText: String {
        guard let coverage else {
            return normalizedHex
        }
        return "\(normalizedHex) (\((coverage * 100).formatted(.number.precision(.fractionLength(1))))%)"
    }
}

struct MomentoInspectorAsset: Identifiable, Hashable {
    var id: String
    var title: String
    var fileName: String
    var previewImage: NSImage?
    var dimensions: String?
    var colors: [MomentoInspectorColor]
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
        colors: [MomentoInspectorColor]? = nil,
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
        self.colors = colors ?? colorHexes.map { MomentoInspectorColor(hex: $0, coverage: nil) }
        self.filePath = filePath
        self.fileSize = fileSize
        self.addedDate = addedDate
        self.kind = kind
    }
}

struct MomentoInspectorView: View {
    @Environment(\.appLocalization) private var localization

    var asset: MomentoInspectorAsset?
    @Binding var tags: [String]
    @Binding var notes: String
    var onRevealInFinder: (() -> Void)?

    @State private var pendingTag = ""
    @State private var hoveredColorID: String?
    @State private var isColorCopyToastVisible = false
    @State private var colorCopyToastToken = UUID()

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
            if let asset {
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        preview(asset)
                        metadata(asset)
                        tagEditor
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
        .overlay(alignment: .bottom) {
            colorCopyToast
        }
        .animation(.smooth(duration: 0.16), value: isColorCopyToastVisible)
    }

    private func preview(_ asset: MomentoInspectorAsset) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            previewImage(asset)

            colorSection(asset.colors)

            Text(asset.title)
                .font(.system(size: 13, weight: .semibold))
                .lineLimit(2)
        }
    }

    @ViewBuilder
    private func previewImage(_ asset: MomentoInspectorAsset) -> some View {
        if let image = asset.previewImage {
            Image(nsImage: image)
                .resizable()
                .scaledToFit()
                .frame(maxWidth: .infinity, alignment: .center)
                .clipShape(RoundedRectangle(cornerRadius: MomentoTheme.assetImageCornerRadius, style: .continuous))
        } else {
            ZStack {
                MomentoGlassBackground(cornerRadius: MomentoTheme.assetImageCornerRadius)

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
            .frame(maxWidth: .infinity)
            .aspectRatio(1.25, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: MomentoTheme.assetImageCornerRadius, style: .continuous))
        }
    }

    private func metadata(_ asset: MomentoInspectorAsset) -> some View {
        inspectorSection(localization.string("Details")) {
            if let kind = asset.kind {
                infoRow(localization.string("Kind"), kind)
            }
            if let dimensions = asset.dimensions {
                infoRow(localization.string("Dimensions"), dimensions)
            }
            if let fileSize = asset.fileSize {
                infoRow(localization.string("Size"), fileSize)
            }
            if let addedDate = asset.addedDate {
                infoRow(localization.string("Added"), localization.dateTime(addedDate))
            }
        }
    }

    private var tagEditor: some View {
        inspectorSection(localization.string("Tags")) {
            if tags.isEmpty {
                Text(localization.string("No tags"))
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
                            .accessibilityLabel(localization.format("Remove %@", tag))
                        }
                        .font(.system(size: 12, weight: .medium))
                        .padding(.horizontal, 8)
                        .frame(height: 24)
                        .glassEffect(.regular.tint(Color.accentColor), in: Capsule())
                    }
                }
            }

            HStack(spacing: 8) {
                TextField(localization.string("Add tag"), text: $pendingTag)
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
                MomentoGlassBackground(cornerRadius: 8)
                    .overlay {
                        RoundedRectangle(cornerRadius: 8, style: .continuous)
                            .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                    }
            }
        }
    }

    @ViewBuilder
    private func colorSection(_ colors: [MomentoInspectorColor]) -> some View {
        if !colors.isEmpty {
            GeometryReader { proxy in
                ScrollView(.horizontal) {
                    HStack(spacing: 4) {
                        ForEach(colors) { color in
                            colorSwatchButton(color)
                                .zIndex(hoveredColorID == color.id ? 1 : 0)
                        }
                    }
                    .frame(minWidth: proxy.size.width, alignment: .center)
                }
                .scrollIndicators(.never)
                .overlay(alignment: .top) {
                    if let hoveredColor = colors.first(where: { $0.id == hoveredColorID }) {
                        colorTooltip(hoveredColor)
                            .offset(y: -34)
                            .transition(.opacity.combined(with: .move(edge: .bottom)))
                            .zIndex(2)
                    }
                }
            }
            .frame(height: 24)
            .frame(maxWidth: .infinity)
            .zIndex(1)
        }
    }

    private func colorSwatchButton(_ color: MomentoInspectorColor) -> some View {
        let isHovered = hoveredColorID == color.id
        let swatchShape = RoundedRectangle(cornerRadius: 6, style: .continuous)
        let hoverShape = RoundedRectangle(cornerRadius: 8, style: .continuous)

        return Button {
            copyColor(color)
        } label: {
            swatchShape
                .fill(Color(hex: color.hex) ?? .clear)
                .frame(width: 18, height: 18)
                .overlay {
                    swatchShape.strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                }
                .padding(3)
                .background {
                    if isHovered {
                        hoverShape.fill(MomentoTheme.sidebarIconHoverBackground)
                    } else {
                        Color.clear
                    }
                }
                .contentShape(hoverShape)
        }
        .buttonStyle(.plain)
        .pointerStyle(.link)
        .onHover { hovering in
            updateColorHover(id: color.id, isHovering: hovering)
        }
        .accessibilityLabel(color.helpText)
    }

    private func colorTooltip(_ color: MomentoInspectorColor) -> some View {
        Text(color.helpText)
            .font(.system(size: 13, weight: .medium))
            .foregroundStyle(.white)
            .lineLimit(1)
            .fixedSize()
            .padding(.horizontal, 10)
            .frame(height: 30)
            .background {
                RoundedRectangle(cornerRadius: 6, style: .continuous)
                    .fill(Color.black.opacity(0.82))
                    .overlay {
                        RoundedRectangle(cornerRadius: 6, style: .continuous)
                            .strokeBorder(Color.white.opacity(0.18), lineWidth: 1)
                    }
            }
            .shadow(color: Color.black.opacity(0.32), radius: 8, y: 4)
            .allowsHitTesting(false)
    }

    private var notesEditor: some View {
        inspectorSection(localization.string("Notes")) {
            TextEditor(text: $notes)
                .font(.system(size: 12))
                .scrollContentBackground(.hidden)
                .frame(minHeight: 86)
                .padding(6)
                .background {
                    MomentoGlassBackground(cornerRadius: 8)
                        .overlay {
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                        }
                }
        }
    }

    private func fileSection(_ asset: MomentoInspectorAsset) -> some View {
        inspectorSection(localization.string("File")) {
            infoRow(localization.string("Name"), asset.fileName)
            if let filePath = asset.filePath {
                infoRow(localization.string("Path"), filePath)
            }

            if onRevealInFinder != nil {
                Button(action: { onRevealInFinder?() }) {
                    Label(localization.string("Reveal in Finder"), systemImage: "finder")
                        .frame(maxWidth: .infinity)
                        .frame(height: 30)
                }
                .buttonStyle(.glass)
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "sidebar.right")
                .font(.system(size: 30))
                .foregroundStyle(MomentoTheme.tertiaryText)
            Text(localization.string("No asset selected"))
                .font(.system(size: 13, weight: .medium))
            Text(localization.string("Select an item to inspect metadata, tags, colors, and file details."))
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

    @ViewBuilder
    private var colorCopyToast: some View {
        if isColorCopyToastVisible {
            Text(localization.string("Color copied"))
                .font(.system(size: 12, weight: .medium))
                .foregroundStyle(MomentoTheme.primaryText)
                .padding(.horizontal, 12)
                .frame(height: 30)
                .background {
                    MomentoGlassBackground(cornerRadius: 10)
                        .overlay {
                            RoundedRectangle(cornerRadius: 10, style: .continuous)
                                .strokeBorder(MomentoTheme.subtleStroke, lineWidth: 1)
                        }
                }
                .padding(.bottom, 14)
                .transition(.opacity.combined(with: .move(edge: .bottom)))
                .allowsHitTesting(false)
        }
    }

    private func updateColorHover(id: String, isHovering: Bool) {
        withAnimation(.smooth(duration: 0.14)) {
            if isHovering {
                hoveredColorID = id
            } else if hoveredColorID == id {
                hoveredColorID = nil
            }
        }
    }

    private func copyColor(_ color: MomentoInspectorColor) {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()

        guard pasteboard.setString(color.normalizedHex, forType: .string) else {
            return
        }

        let token = UUID()
        colorCopyToastToken = token

        withAnimation(.smooth(duration: 0.16)) {
            isColorCopyToastVisible = true
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 1.4) {
            guard colorCopyToastToken == token else {
                return
            }

            withAnimation(.smooth(duration: 0.16)) {
                isColorCopyToastVisible = false
            }
        }
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
