// 中文注释：本视图绘制素材预览浮层，具体窗口生命周期由 AppKit 面板控制器管理。
import AppKit
import ImageIO
import SwiftUI

private enum MomentoAssetPreviewMetrics {
    static let imagePanelCornerRadius: CGFloat = 30
    static let imagePanelPadding: CGFloat = 14
    static let imagePanelTintOpacity = 0.5
    static let navigationButtonSize: CGFloat = 44
    static let navigationHorizontalInset: CGFloat = 22
    static let navigationRevealDelayNanoseconds: UInt64 = 240_000_000
    static let navigationToastDuration: TimeInterval = 1.35
}

struct MomentoAssetPreviewOverlay: View {
    @Environment(\.appLocalization) private var localization

    var asset: AssetItem
    var previewURL: URL
    var closesOnSpaceKeyUp = false
    var usesWindowTransition = false
    var animatesPresentation = true
    var showsNavigationControls = false
    var canNavigatePrevious = false
    var canNavigateNext = false
    var onNavigatePrevious: (() -> Void)?
    var onNavigateNext: (() -> Void)?
    var onDismiss: () -> Void

    @State private var previewImage: NSImage?
    @State private var previewImageTask: Task<Void, Never>?
    @State private var isPresented = false
    @State private var isClosing = false
    @State private var areNavigationControlsVisible = false
    @State private var activeNavigationToastMessage: String?
    @State private var navigationToastToken = UUID()

    var body: some View {
        GeometryReader { proxy in
            ZStack {
                Color.clear
                    .ignoresSafeArea()
                    .contentShape(Rectangle())
                    .onTapGesture {
                        dismiss()
                    }

                previewContent(in: proxy.size)
                    .scaleEffect(presentationScale)
                    .opacity(presentationOpacity)
                    .animation(presentationAnimation, value: isPresented)

                if showsNavigationControls, areNavigationControlsVisible {
                    navigationControls
                        .transaction { transaction in
                            transaction.animation = nil
                        }
                }

                if let activeNavigationToastMessage {
                    navigationToast(activeNavigationToastMessage)
                        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
                        .padding(.top, 52)
                        .allowsHitTesting(false)
                        .zIndex(2)
                }
            }
            .frame(width: proxy.size.width, height: proxy.size.height)
        }
        .background {
            MomentoPreviewKeyboardCapture(
                closesOnSpaceKeyUp: closesOnSpaceKeyUp,
                onNavigatePrevious: keyboardNavigatePreviousAction,
                onNavigateNext: keyboardNavigateNextAction,
                onDismiss: dismiss
            )
                .frame(width: 0, height: 0)
        }
        .onAppear {
            loadPreviewImage()
        }
        .onChange(of: previewURL) { _, _ in
            loadPreviewImage()
        }
        .onDisappear {
            previewImageTask?.cancel()
            previewImageTask = nil
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
                            cornerRadius: MomentoTheme.assetImageCornerRadius,
                            style: .continuous
                        )
                    )
                    .padding(MomentoAssetPreviewMetrics.imagePanelPadding)
                    .background {
                        MomentoGlassBackground(
                            style: .regular.tint(MomentoTheme.surfaceGlassTint(darkOpacity: MomentoAssetPreviewMetrics.imagePanelTintOpacity)),
                            cornerRadius: MomentoAssetPreviewMetrics.imagePanelCornerRadius
                        )
                    }
                    .shadow(color: .black.opacity(0.36), radius: 32, y: 18)
            }

            previewMetadata
        }
        .padding(18)
    }

    private var navigationControls: some View {
        HStack {
            navigationButton(
                systemName: "chevron.left",
                help: previousNavigationHelp,
                isAvailable: canNavigatePrevious,
                action: navigatePrevious
            )

            Spacer()

            navigationButton(
                systemName: "chevron.right",
                help: nextNavigationHelp,
                isAvailable: canNavigateNext,
                action: navigateNext
            )
        }
        .padding(.horizontal, MomentoAssetPreviewMetrics.navigationHorizontalInset)
    }

    private func navigationButton(
        systemName: String,
        help: String,
        isAvailable: Bool,
        action: (() -> Void)?
    ) -> some View {
        Button {
            action?()
        } label: {
            Image(systemName: systemName)
                .font(.system(size: 18, weight: .semibold))
                .frame(
                    width: MomentoAssetPreviewMetrics.navigationButtonSize,
                    height: MomentoAssetPreviewMetrics.navigationButtonSize
                )
        }
        .momentoGlassButtonStyle()
        .buttonBorderShape(.circle)
        .opacity(isAvailable ? 1 : 0.38)
        .help(help)
        .contentShape(Circle())
        .pointerStyle(.link)
    }

    private var keyboardNavigatePreviousAction: (() -> Void)? {
        guard showsNavigationControls else {
            return nil
        }

        return { navigatePrevious() }
    }

    private var keyboardNavigateNextAction: (() -> Void)? {
        guard showsNavigationControls else {
            return nil
        }

        return { navigateNext() }
    }

    private var previousNavigationHelp: String {
        canNavigatePrevious
            ? localization.string("Previous Image")
            : localization.string("No previous image")
    }

    private var nextNavigationHelp: String {
        canNavigateNext
            ? localization.string("Next Image")
            : localization.string("No next image")
    }

    private func navigationToast(_ message: String) -> some View {
        Text(message)
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(.white)
            .padding(.horizontal, 16)
            .frame(height: 38)
            .background {
                MomentoGlassBackground(
                    style: .regular.tint(MomentoTheme.surfaceGlassTint(darkOpacity: 0.4)),
                    cornerRadius: 14
                )
            }
    }

    private func navigatePrevious() {
        guard canNavigatePrevious else {
            showNavigationBoundaryToast(localization.string("No previous image"))
            return
        }

        onNavigatePrevious?()
    }

    private func navigateNext() {
        guard canNavigateNext else {
            showNavigationBoundaryToast(localization.string("No next image"))
            return
        }

        onNavigateNext?()
    }

    private func showNavigationBoundaryToast(_ message: String) {
        let token = UUID()
        navigationToastToken = token
        activeNavigationToastMessage = message

        DispatchQueue.main.asyncAfter(deadline: .now() + MomentoAssetPreviewMetrics.navigationToastDuration) {
            guard navigationToastToken == token else {
                return
            }

            activeNavigationToastMessage = nil
        }
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
                .momentoGlassButtonStyle()
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
            MomentoGlassBackground(style: .regular.tint(MomentoTheme.surfaceGlassTint(darkOpacity: 0.12)), cornerRadius: 16)
        }
    }

    private var subtitle: String {
        if let dimensions = asset.dimensions {
            return "\(dimensions.width) × \(dimensions.height)"
        }

        return previewURL.lastPathComponent
    }

    private var presentationScale: CGFloat {
        animatesPresentation && !isPresented ? 0.96 : 1
    }

    private var presentationOpacity: Double {
        animatesPresentation && !isPresented ? 0 : 1
    }

    private var presentationAnimation: Animation? {
        animatesPresentation ? .spring(response: 0.24, dampingFraction: 0.86) : nil
    }

    private func loadPreviewImage() {
        previewImageTask?.cancel()
        if previewImage == nil {
            areNavigationControlsVisible = false
        }

        let url = previewURL
        let maxPixelSize = previewMaxPixelSize
        previewImageTask = Task {
            let image = await PreviewImageLoader.image(for: url, maxPixelSize: maxPixelSize)

            guard !Task.isCancelled else {
                return
            }

            previewImage = image ?? NSWorkspace.shared.icon(forFile: url.path)
            withAnimation(presentationAnimation) {
                isPresented = true
            }

            if animatesPresentation {
                try? await Task.sleep(nanoseconds: MomentoAssetPreviewMetrics.navigationRevealDelayNanoseconds)
                guard !Task.isCancelled else {
                    return
                }
            }

            areNavigationControlsVisible = true
        }
    }

    private var previewMaxPixelSize: Int {
        let screen = NSScreen.main
        let scale = screen?.backingScaleFactor ?? 2
        let visibleFrame = screen?.visibleFrame ?? NSRect(x: 0, y: 0, width: 1600, height: 1000)
        return max(Int(max(visibleFrame.width * 0.86, visibleFrame.height * 0.82) * scale), 1)
    }

    private func dismiss() {
        guard !isClosing else {
            return
        }

        isClosing = true
        guard !usesWindowTransition else {
            onDismiss()
            return
        }

        withAnimation(.smooth(duration: 0.16)) {
            isPresented = false
        }

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.16) {
            onDismiss()
        }
    }
}

private enum PreviewImageLoader {
    static func image(for url: URL, maxPixelSize: Int) async -> NSImage? {
        await Task.detached(priority: .userInitiated) {
            guard let source = CGImageSourceCreateWithURL(url as CFURL, [
                kCGImageSourceShouldCache: false
            ] as CFDictionary) else {
                return nil
            }

            let options = [
                kCGImageSourceCreateThumbnailFromImageAlways: true,
                kCGImageSourceCreateThumbnailWithTransform: true,
                kCGImageSourceShouldCacheImmediately: true,
                kCGImageSourceThumbnailMaxPixelSize: maxPixelSize
            ] as CFDictionary

            guard let image = CGImageSourceCreateThumbnailAtIndex(source, 0, options) else {
                return nil
            }

            return NSImage(
                cgImage: image,
                size: NSSize(width: image.width, height: image.height)
            )
        }.value
    }
}

private struct MomentoPreviewKeyboardCapture: NSViewRepresentable {
    var closesOnSpaceKeyUp: Bool
    var onNavigatePrevious: (() -> Void)?
    var onNavigateNext: (() -> Void)?
    var onDismiss: () -> Void

    func makeNSView(context: Context) -> KeyboardCaptureView {
        let view = KeyboardCaptureView()
        view.closesOnSpaceKeyUp = closesOnSpaceKeyUp
        view.onNavigatePrevious = onNavigatePrevious
        view.onNavigateNext = onNavigateNext
        view.onDismiss = onDismiss
        return view
    }

    func updateNSView(_ nsView: KeyboardCaptureView, context: Context) {
        nsView.closesOnSpaceKeyUp = closesOnSpaceKeyUp
        nsView.onNavigatePrevious = onNavigatePrevious
        nsView.onNavigateNext = onNavigateNext
        nsView.onDismiss = onDismiss
        DispatchQueue.main.async {
            nsView.window?.makeFirstResponder(nsView)
        }
    }

    final class KeyboardCaptureView: NSView {
        private static let leftArrowKeyCode: UInt16 = 123
        private static let rightArrowKeyCode: UInt16 = 124

        var onNavigatePrevious: (() -> Void)?
        var onNavigateNext: (() -> Void)?
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
            if let onNavigatePrevious,
               event.keyCode == Self.leftArrowKeyCode,
               acceptsNavigationModifiers(event) {
                onNavigatePrevious()
                return
            }

            if let onNavigateNext,
               event.keyCode == Self.rightArrowKeyCode,
               acceptsNavigationModifiers(event) {
                onNavigateNext()
                return
            }

            if event.charactersIgnoringModifiers == "\u{1b}" {
                onDismiss?()
                return
            }

            if event.charactersIgnoringModifiers == " " {
                if !closesOnSpaceKeyUp {
                    onDismiss?()
                }
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

        private func acceptsNavigationModifiers(_ event: NSEvent) -> Bool {
            event.modifierFlags.intersection([.command, .control, .option]).isEmpty
        }
    }
}
