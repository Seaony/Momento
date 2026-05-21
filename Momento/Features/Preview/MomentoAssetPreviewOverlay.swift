import AppKit
import SwiftUI

private enum MomentoAssetPreviewMetrics {
    static let imagePanelCornerRadius: CGFloat = 30
    static let imageCornerRadius: CGFloat = 24
    static let imagePanelPadding: CGFloat = 14
}

struct MomentoAssetPreviewOverlay: View {
    @Environment(\.appLocalization) private var localization

    var asset: AssetItem
    var previewURL: URL
    var closesOnSpaceKeyUp = false
    var onDismiss: () -> Void

    @State private var previewImage: NSImage?
    @State private var isPresented = false
    @State private var isClosing = false

    var body: some View {
        GeometryReader { proxy in
            ZStack {
                Color.black.opacity(0.28)
                    .ignoresSafeArea()
                    .background(.ultraThinMaterial)
                    .contentShape(Rectangle())
                    .onTapGesture {
                        dismiss()
                    }

                previewContent(in: proxy.size)
                    .scaleEffect(isPresented ? 1 : 0.96)
                    .opacity(isPresented ? 1 : 0)
                    .animation(.spring(response: 0.24, dampingFraction: 0.86), value: isPresented)
            }
            .frame(width: proxy.size.width, height: proxy.size.height)
        }
        .background {
            MomentoPreviewKeyboardCapture(
                closesOnSpaceKeyUp: closesOnSpaceKeyUp,
                onDismiss: dismiss
            )
                .frame(width: 0, height: 0)
        }
        .onAppear {
            previewImage = NSImage(contentsOf: previewURL) ?? NSWorkspace.shared.icon(forFile: previewURL.path)
            withAnimation(.spring(response: 0.24, dampingFraction: 0.86)) {
                isPresented = true
            }
        }
    }

    private func previewContent(in size: CGSize) -> some View {
        VStack(spacing: 12) {
            if let previewImage {
                Image(nsImage: previewImage)
                    .resizable()
                    .interpolation(.high)
                    .scaledToFit()
                    .frame(
                        maxWidth: max(size.width * 0.86, 360),
                        maxHeight: max(size.height * 0.82, 320)
                    )
                    .clipShape(
                        RoundedRectangle(
                            cornerRadius: MomentoAssetPreviewMetrics.imageCornerRadius,
                            style: .continuous
                        )
                    )
                    .padding(MomentoAssetPreviewMetrics.imagePanelPadding)
                    .background {
                        MomentoGlassBackground(
                            glass: .regular.tint(Color.black.opacity(0.10)),
                            cornerRadius: MomentoAssetPreviewMetrics.imagePanelCornerRadius
                        )
                    }
                    .shadow(color: .black.opacity(0.36), radius: 32, y: 18)
            }

            previewMetadata
        }
        .padding(18)
    }

    private var previewMetadata: some View {
        ZStack {
            VStack(alignment: .center, spacing: 3) {
                Text(asset.displayName)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(MomentoTheme.primaryText)
                    .lineLimit(1)
                    .truncationMode(.tail)
                    .multilineTextAlignment(.center)

                Text(subtitle)
                    .font(.system(size: 11))
                    .foregroundStyle(MomentoTheme.secondaryText)
                    .lineLimit(1)
                    .truncationMode(.tail)
                    .multilineTextAlignment(.center)
            }
            .padding(.horizontal, 48)
            .frame(maxWidth: .infinity)

            HStack {
                Spacer()

                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 12, weight: .semibold))
                        .frame(width: 28, height: 28)
                }
                .buttonStyle(.glass)
                .buttonBorderShape(.circle)
                .help(localization.string("Dismiss"))
                .contentShape(Circle())
                .pointerStyle(.link)
            }
        }
        .frame(maxWidth: 560)
        .padding(.horizontal, 14)
        .padding(.vertical, 10)
        .background {
            MomentoGlassBackground(glass: .regular.tint(Color.black.opacity(0.12)), cornerRadius: 16)
        }
    }

    private var subtitle: String {
        if let dimensions = asset.dimensions {
            return "\(dimensions.width) × \(dimensions.height)"
        }

        return previewURL.lastPathComponent
    }

    private func dismiss() {
        guard !isClosing else {
            return
        }

        isClosing = true
        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.16) {
            onDismiss()
        }
    }
}

private struct MomentoPreviewKeyboardCapture: NSViewRepresentable {
    var closesOnSpaceKeyUp: Bool
    var onDismiss: () -> Void

    func makeNSView(context: Context) -> KeyboardCaptureView {
        let view = KeyboardCaptureView()
        view.closesOnSpaceKeyUp = closesOnSpaceKeyUp
        view.onDismiss = onDismiss
        return view
    }

    func updateNSView(_ nsView: KeyboardCaptureView, context: Context) {
        nsView.closesOnSpaceKeyUp = closesOnSpaceKeyUp
        nsView.onDismiss = onDismiss
        DispatchQueue.main.async {
            nsView.window?.makeFirstResponder(nsView)
        }
    }

    final class KeyboardCaptureView: NSView {
        var onDismiss: (() -> Void)?
        var closesOnSpaceKeyUp = false

        override var acceptsFirstResponder: Bool {
            true
        }

        override func viewDidMoveToWindow() {
            super.viewDidMoveToWindow()
            window?.makeFirstResponder(self)
        }

        override func keyDown(with event: NSEvent) {
            if event.charactersIgnoringModifiers == "\u{1b}" {
                onDismiss?()
                return
            }

            if event.charactersIgnoringModifiers == " ", !closesOnSpaceKeyUp {
                onDismiss?()
                return
            }

            super.keyDown(with: event)
        }

        override func keyUp(with event: NSEvent) {
            if event.charactersIgnoringModifiers == " ", closesOnSpaceKeyUp {
                onDismiss?()
                return
            }

            super.keyUp(with: event)
        }
    }
}
