// 中文注释：本文件定义用户偏好枚举和持久化 key，供 AppStorage 与全局环境共用。
import AppKit
import Foundation
import SwiftUI

enum AppLanguage: String, CaseIterable, Identifiable, Sendable {
    case system
    case english
    case simplifiedChinese

    var id: String { rawValue }

    var localeIdentifier: String? {
        switch self {
        case .system:
            nil
        case .english:
            "en"
        case .simplifiedChinese:
            "zh-Hans"
        }
    }

    var locale: Locale {
        if let localeIdentifier {
            return Locale(identifier: localeIdentifier)
        }
        return .autoupdatingCurrent
    }

    var resourceIdentifier: String? {
        localeIdentifier
    }
}

enum AppAppearanceMode: String, CaseIterable, Identifiable, Sendable {
    case system
    case light
    case dark

    var id: String { rawValue }

    var colorScheme: ColorScheme? {
        switch self {
        case .system:
            nil
        case .light:
            .light
        case .dark:
            .dark
        }
    }

    var appKitAppearanceName: NSAppearance.Name? {
        switch self {
        case .system:
            nil
        case .light:
            .aqua
        case .dark:
            .darkAqua
        }
    }
}

enum AppSettingsKeys {
    static let appLanguage = "appLanguage"
    static let appAppearance = "appAppearance"
    static let defaultViewMode = "defaultViewMode"
}

enum AppSettings {
    static func appAppearance(defaults: UserDefaults = .standard) -> AppAppearanceMode {
        let rawValue = defaults.string(forKey: AppSettingsKeys.appAppearance)
        return rawValue.flatMap(AppAppearanceMode.init(rawValue:)) ?? .system
    }

    static func defaultViewMode(defaults: UserDefaults = .standard) -> AssetViewMode {
        let rawValue = defaults.string(forKey: AppSettingsKeys.defaultViewMode)
        return rawValue.flatMap(AssetViewMode.init(rawValue:)) ?? .masonry
    }
}
