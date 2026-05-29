// 中文注释：本视图展示素材导出格式确认弹窗，背景使用系统原生 Liquid Glass。
import SwiftUI

private let assetExportDialogWidth: CGFloat = 430
private let assetExportDialogIconSize: CGFloat = 46

struct MomentoAssetExportDialog: View {
    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var isPresented: Bool

    var assetCount: Int
    var onSubmit: (AssetExportConfiguration) -> Void

    @State private var selectedFormat = AssetExportFormat.original
    @State private var jpegQuality = AssetExportConfiguration.default.jpegQuality
    @State private var isCancelButtonHovered = false
    @State private var isPrimaryButtonHovered = false

    var body: some View {
        ZStack {
            MomentoDialogBackdrop(dismiss: dismiss)

            HStack(alignment: .top, spacing: 16) {
                dialogIcon

                VStack(alignment: .leading, spacing: 18) {
                    header
                    formatOptions

                    if selectedFormat == .jpeg {
                        jpegQualityControl
                    }

                    footer
                }
            }
            .padding(.horizontal, 30)
            .padding(.vertical, 28)
            .frame(width: assetExportDialogWidth)
            .background {
                MomentoGlassBackground(style: .regular.tint(MomentoTheme.surfaceGlassTint(darkOpacity: 0.18)), cornerRadius: 14)
            }
            .contentShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .onTapGesture {}
        }
        .transition(reduceMotion ? .opacity : .scale(scale: 0.96).combined(with: .opacity))
        .onExitCommand {
            dismiss()
        }
    }

    private var dialogIcon: some View {
        Image(systemName: "square.and.arrow.up.fill")
            .font(.system(size: 21, weight: .semibold))
            .foregroundStyle(.white)
            .frame(width: assetExportDialogIconSize, height: assetExportDialogIconSize)
            .background {
                MomentoGlassBackground(style: .regular.tint(Color.accentColor), cornerRadius: 14)
            }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(localization.string("Export Assets"))
                .font(.system(size: 18, weight: .semibold))

            Text(subtitle)
                .font(.system(size: 13, weight: .regular))
                .lineSpacing(2)
                .foregroundStyle(MomentoTheme.secondaryText)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    private var formatOptions: some View {
        VStack(spacing: 8) {
            ForEach(AssetExportFormat.allCases) { format in
                formatButton(format)
            }
        }
    }

    private func formatButton(_ format: AssetExportFormat) -> some View {
        Button {
            selectedFormat = format
        } label: {
            HStack(spacing: 10) {
                Image(systemName: format.systemImageName)
                    .font(.system(size: 14, weight: .semibold))
                    .frame(width: 20)

                VStack(alignment: .leading, spacing: 2) {
                    Text(localization.string(format.titleKey))
                        .font(.system(size: 13, weight: .semibold))
                    Text(localization.string(format.subtitleKey))
                        .font(.system(size: 11, weight: .regular))
                        .foregroundStyle(MomentoTheme.secondaryText)
                }

                Spacer(minLength: 8)

                Image(systemName: selectedFormat == format ? "checkmark.circle.fill" : "circle")
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(selectedFormat == format ? Color.accentColor : MomentoTheme.secondaryText)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
            .contentShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        }
        .buttonStyle(.plain)
        .background {
            MomentoGlassBackground(
                style: .regular
                    .tint(selectedFormat == format
                        ? Color.accentColor.opacity(0.16)
                        : MomentoTheme.contrastTint(lightOpacity: 0.03, darkOpacity: 0.04)
                    )
                    .interactive(true),
                cornerRadius: 10
            )
        }
        .pointerStyle(.link)
    }

    private var jpegQualityControl: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(localization.string("JPEG Quality"))
                    .font(.system(size: 12, weight: .semibold))
                Spacer()
                Text("\(Int((jpegQuality * 100).rounded()))%")
                    .font(.system(size: 12, weight: .semibold))
                    .monospacedDigit()
                    .foregroundStyle(MomentoTheme.secondaryText)
            }

            Slider(value: $jpegQuality, in: 0.1...1, step: 0.01)
                .controlSize(.small)
        }
        .padding(12)
        .background {
            MomentoGlassBackground(style: .regular.interactive(true), cornerRadius: 10)
        }
    }

    private var footer: some View {
        HStack(spacing: 14) {
            Button {
                dismiss()
            } label: {
                Text(localization.string("Cancel"))
                    .font(.system(size: 14, weight: .semibold))
                    .padding(.horizontal, 18)
                    .padding(.vertical, 6)
            }
            .momentoGlassButtonStyle()
            .buttonBorderShape(.capsule)
            .controlSize(.large)
            .contentShape(Capsule(style: .continuous))
            .pointerStyle(.link)
            .assetExportDialogButtonHoverFeedback(isHovered: isCancelButtonHovered, reduceMotion: reduceMotion)
            .onHover { isHovered in
                isCancelButtonHovered = isHovered
            }

            Button {
                submit()
            } label: {
                Text(localization.string("Export"))
                    .font(.system(size: 14, weight: .semibold))
                    .padding(.horizontal, 18)
                    .padding(.vertical, 6)
            }
            .momentoGlassButtonStyle(.prominent)
            .buttonBorderShape(.capsule)
            .controlSize(.large)
            .contentShape(Capsule(style: .continuous))
            .pointerStyle(.link)
            .assetExportDialogButtonHoverFeedback(isHovered: isPrimaryButtonHovered, reduceMotion: reduceMotion)
            .onHover { isHovered in
                isPrimaryButtonHovered = isHovered
            }
        }
    }

    private var subtitle: String {
        assetCount == 1
            ? localization.string("Choose the format for this asset.")
            : localization.format("Choose the format for %d selected assets.", assetCount)
    }

    private func submit() {
        let configuration = AssetExportConfiguration(format: selectedFormat, jpegQuality: jpegQuality)
        dismiss()
        DispatchQueue.main.async {
            onSubmit(configuration)
        }
    }

    private func dismiss() {
        withAnimation(.smooth(duration: reduceMotion ? 0.08 : 0.16)) {
            isPresented = false
        }
    }
}

private extension View {
    func assetExportDialogButtonHoverFeedback(isHovered: Bool, reduceMotion: Bool) -> some View {
        scaleEffect(isHovered && !reduceMotion ? 1.035 : 1)
            .brightness(isHovered ? 0.08 : 0)
            .animation(reduceMotion ? nil : .smooth(duration: 0.16), value: isHovered)
    }
}
