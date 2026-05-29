// 中文注释：设置页采用「品牌标识 + 玻璃偏好卡片 + 独立动作按钮」的层次结构，只呈现真实存在的偏好（语言）与动作（检查更新），不堆砌占位项。
import AppKit
import SwiftUI

private enum MomentoSettingsMetrics {
    static let windowWidth: CGFloat = 400
    static let windowHeight: CGFloat = 370

    static let topInset: CGFloat = 42          // 让出隐藏标题栏与窗口关闭按钮的区域
    static let horizontalInset: CGFloat = 20
    static let bottomInset: CGFloat = 24

    static let iconSize: CGFloat = 76
    static let cardRadius: CGFloat = 18
    static let rowHeight: CGFloat = 44
    static let rowHorizontalInset: CGFloat = 16
    static let pickerWidth: CGFloat = 172
    static let appearancePickerWidth: CGFloat = 118
    static let actionButtonHeight: CGFloat = 30
    static let actionButtonRadius: CGFloat = 10

    static var cardWidth: CGFloat {
        windowWidth - horizontalInset * 2
    }
}

struct MomentoSettingsView: View {
    static let preferredSize = CGSize(
        width: MomentoSettingsMetrics.windowWidth,
        height: MomentoSettingsMetrics.windowHeight
    )

    @Environment(\.appLocalization) private var localization
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    @Binding var appLanguage: AppLanguage
    @Binding var appAppearance: AppAppearanceMode
    @ObservedObject var updateService: AppUpdateService

    @State private var isUpdateHovered = false

    var body: some View {
        ZStack {
            MomentoGlassBackground(cornerRadius: 0)
                .ignoresSafeArea()

            VStack(spacing: 0) {
                identityHeader
                    .padding(.bottom, 22)
                settingsCard
                    .padding(.bottom, 16)
                updateButton
            }
            .padding(.top, MomentoSettingsMetrics.topInset)
            .padding(.horizontal, MomentoSettingsMetrics.horizontalInset)
            .padding(.bottom, MomentoSettingsMetrics.bottomInset)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
        }
        .background {
            WindowTransparencyConfigurator(fixedContentSize: Self.preferredSize)
        }
        .toolbarBackgroundVisibility(.hidden, for: .windowToolbar)
        .frame(
            width: MomentoSettingsMetrics.windowWidth,
            height: MomentoSettingsMetrics.windowHeight,
            alignment: .top
        )
    }

    // MARK: - 品牌标识

    private var identityHeader: some View {
        VStack(spacing: 0) {
            Image(nsImage: NSApplication.shared.applicationIconImage)
                .resizable()
                .interpolation(.high)
                .frame(width: MomentoSettingsMetrics.iconSize, height: MomentoSettingsMetrics.iconSize)
                .shadow(color: Color.black.opacity(0.22), radius: 10, y: 5)
                .padding(.bottom, 12)

            Text(verbatim: "Momento")
                .font(.system(size: 22, weight: .semibold))
                .foregroundStyle(MomentoTheme.primaryText)
                .tracking(0.2)

            Text(versionText)
                .font(.system(size: 12, weight: .regular))
                .foregroundStyle(MomentoTheme.secondaryText)
                .padding(.top, 3)
        }
        .frame(maxWidth: .infinity)
    }

    private var versionText: String {
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(localization.string("Version")) \(version) (\(build))"
    }

    // MARK: - 偏好卡片

    private var settingsCard: some View {
        VStack(spacing: 0) {
            settingsRow(label: localization.string("Language")) {
                languagePicker
            }

            Divider()
                .padding(.leading, MomentoSettingsMetrics.rowHorizontalInset)

            settingsRow(label: localization.string("Appearance")) {
                appearancePicker
            }
        }
        .padding(.vertical, 6)
        .frame(width: MomentoSettingsMetrics.cardWidth)
        .background {
            MomentoGlassBackground(
                glass: .regular.tint(MomentoTheme.contrastTint(lightOpacity: 0.04, darkOpacity: 0.05)).interactive(true),
                cornerRadius: MomentoSettingsMetrics.cardRadius
            )
        }
        .overlay {
            RoundedRectangle(cornerRadius: MomentoSettingsMetrics.cardRadius, style: .continuous)
                .strokeBorder(MomentoTheme.subtleStroke.opacity(0.35), lineWidth: 0.6)
        }
        .shadow(color: Color.black.opacity(0.1), radius: 12, y: 4)
    }

    private func settingsRow<Content: View>(
        label: String,
        @ViewBuilder content: () -> Content
    ) -> some View {
        HStack(spacing: 12) {
            Text(label)
                .font(.system(size: 13, weight: .medium))
                .foregroundStyle(MomentoTheme.primaryText)
            Spacer(minLength: 8)
            content()
        }
        .padding(.horizontal, MomentoSettingsMetrics.rowHorizontalInset)
        .frame(maxWidth: .infinity)
        .frame(height: MomentoSettingsMetrics.rowHeight)
    }

    private var languagePicker: some View {
        Picker("", selection: $appLanguage) {
            ForEach(AppLanguage.allCases) { language in
                Text(localization.title(for: language))
                    .tag(language)
            }
        }
        .labelsHidden()
        .pickerStyle(.menu)
        .buttonStyle(.glass)
        .controlSize(.regular)
        .frame(width: MomentoSettingsMetrics.pickerWidth)
        .environment(\.appearsActive, true)
    }

    private var appearancePicker: some View {
        Picker("", selection: $appAppearance) {
            ForEach(AppAppearanceMode.allCases) { appearance in
                Label(localization.title(for: appearance), systemImage: appearanceIconName(for: appearance))
                    .labelStyle(.iconOnly)
                    .help(localization.title(for: appearance))
                    .tag(appearance)
            }
        }
        .labelsHidden()
        .pickerStyle(.segmented)
        .controlSize(.regular)
        .frame(width: MomentoSettingsMetrics.appearancePickerWidth)
        .environment(\.appearsActive, true)
    }

    private func appearanceIconName(for appearance: AppAppearanceMode) -> String {
        switch appearance {
        case .system:
            "circle.lefthalf.filled"
        case .light:
            "sun.max.fill"
        case .dark:
            "moon.fill"
        }
    }

    // MARK: - 检查更新

    private var updateButton: some View {
        let hasUpdate = updateService.availableUpdateDisplayVersion != nil
        let shape = RoundedRectangle(cornerRadius: MomentoSettingsMetrics.actionButtonRadius, style: .continuous)

        return Button {
            updateService.checkForUpdates()
        } label: {
            HStack(spacing: 6) {
                Image(systemName: hasUpdate ? "arrow.down.circle.fill" : "arrow.triangle.2.circlepath")
                Text(updateButtonTitle)
                    .lineLimit(1)
            }
            .font(.system(size: 12, weight: .semibold))
            .foregroundStyle(hasUpdate ? Color.white : MomentoTheme.primaryText)
            .fixedSize()
            .padding(.horizontal, 16)
            .frame(height: MomentoSettingsMetrics.actionButtonHeight)
            .glassEffect(
                hasUpdate
                    ? .regular.tint(Color.accentColor).interactive(true)
                    : .regular.interactive(true),
                in: shape
            )
            .contentShape(shape)
        }
        .buttonStyle(.plain)
        .scaleEffect(isUpdateHovered && !reduceMotion ? 1.03 : 1)
        .brightness(isUpdateHovered ? 0.06 : 0)
        .shadow(
            color: Color.black.opacity(isUpdateHovered ? 0.2 : 0),
            radius: isUpdateHovered ? 7 : 0,
            y: isUpdateHovered ? 3 : 0
        )
        .environment(\.appearsActive, true)
        .disabled(!updateService.canCheckForUpdates)
        .opacity(updateService.canCheckForUpdates ? 1 : 0.55)
        .pointerStyle(.link)
        .animation(.smooth(duration: 0.15), value: isUpdateHovered)
        .onHover { hovering in
            isUpdateHovered = hovering
        }
    }

    private var updateButtonTitle: String {
        if let version = updateService.availableUpdateDisplayVersion {
            return "\(localization.string("Update Available")) · \(version)"
        }
        return localization.string("Check for Updates")
    }
}

#Preview {
    @Previewable @State var language = AppLanguage.system
    @Previewable @State var appearance = AppAppearanceMode.system

    MomentoSettingsView(
        appLanguage: $language,
        appAppearance: $appearance,
        updateService: AppUpdateService()
    )
    .environment(\.appLocalization, AppLocalization(language: language))
    .frame(width: MomentoSettingsView.preferredSize.width, height: MomentoSettingsView.preferredSize.height)
}
