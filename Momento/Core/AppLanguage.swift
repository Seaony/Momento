// 中文注释：本枚举定义用户可选语言和对应 Locale，供 AppStorage 与环境 locale 共用。
import Foundation

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

enum AppSettingsKeys {
    static let appLanguage = "appLanguage"
    static let defaultViewMode = "defaultViewMode"
}

enum AppSettings {
    static func defaultViewMode(defaults: UserDefaults = .standard) -> AssetViewMode {
        let rawValue = defaults.string(forKey: AppSettingsKeys.defaultViewMode)
        return rawValue.flatMap(AssetViewMode.init(rawValue:)) ?? .masonry
    }
}
