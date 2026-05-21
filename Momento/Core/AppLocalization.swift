import Foundation
import SwiftUI

struct AppLocalization: Equatable {
    var language: AppLanguage

    var locale: Locale {
        language.locale
    }

    func string(_ key: String) -> String {
        bundle.localizedString(forKey: key, value: key, table: nil)
    }

    func format(_ key: String, _ arguments: CVarArg...) -> String {
        String(format: string(key), locale: locale, arguments: arguments)
    }

    func itemCount(_ count: Int) -> String {
        if count == 1 {
            return string("1 item")
        }
        return format("%d items", count)
    }

    func title(for language: AppLanguage) -> String {
        switch language {
        case .system:
            string("System")
        case .english:
            string("English")
        case .simplifiedChinese:
            string("Simplified Chinese")
        }
    }

    func title(for viewMode: AssetViewMode) -> String {
        switch viewMode {
        case .masonry:
            string("Masonry View")
        case .grid:
            string("Grid View")
        case .list:
            string("List View")
        }
    }

    func kindTitle(for kind: AssetKind) -> String {
        switch kind {
        case .image:
            string("Image")
        case .gif:
            string("GIF")
        case .svg:
            string("SVG")
        case .video:
            string("Video")
        case .pdf:
            string("PDF")
        }
    }

    func dateTime(_ date: Date) -> String {
        date.formatted(
            Date.FormatStyle(date: .abbreviated, time: .shortened)
                .locale(locale)
        )
    }

    func fileSize(_ byteSize: Int64) -> String {
        byteSize.formatted(
            ByteCountFormatStyle(style: .file)
                .locale(locale)
        )
    }

    private var bundle: Bundle {
        guard let resourceIdentifier = language.resourceIdentifier,
              let path = Bundle.main.path(forResource: resourceIdentifier, ofType: "lproj"),
              let bundle = Bundle(path: path) else {
            return .main
        }
        return bundle
    }
}

private struct AppLocalizationKey: EnvironmentKey {
    static let defaultValue = AppLocalization(language: .system)
}

extension EnvironmentValues {
    var appLocalization: AppLocalization {
        get { self[AppLocalizationKey.self] }
        set { self[AppLocalizationKey.self] = newValue }
    }
}
