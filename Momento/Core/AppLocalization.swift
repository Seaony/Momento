// 中文注释：本文件集中封装字符串、本地化格式化和错误文案映射，避免界面层散落 Bundle 查询。
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

    func title(for sortOption: AssetSortOption) -> String {
        switch sortOption {
        case .addedTime:
            string("Added Time")
        case .name:
            string("File Name")
        case .fileSize:
            string("File Size")
        }
    }

    func title(for sortDirection: AssetSortDirection) -> String {
        switch sortDirection {
        case .ascending:
            string("Ascending")
        case .descending:
            string("Descending")
        }
    }

    func title(for colorCategory: AssetColorCategory) -> String {
        switch colorCategory {
        case .black:
            string("Black")
        case .white:
            string("White")
        case .gray:
            string("Gray")
        case .red:
            string("Red")
        case .rose:
            string("Rose")
        case .pink:
            string("Pink")
        case .magenta:
            string("Magenta")
        case .purple:
            string("Purple")
        case .violet:
            string("Violet")
        case .indigo:
            string("Indigo")
        case .blue:
            string("Blue")
        case .sky:
            string("Sky")
        case .cyan:
            string("Cyan")
        case .teal:
            string("Teal")
        case .mint:
            string("Mint")
        case .green:
            string("Green")
        case .lime:
            string("Lime")
        case .olive:
            string("Olive")
        case .yellow:
            string("Yellow")
        case .amber:
            string("Amber")
        case .orange:
            string("Orange")
        case .coral:
            string("Coral")
        case .brown:
            string("Brown")
        case .beige:
            string("Beige")
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

    func errorMessage(_ error: Error) -> String {
        guard let libraryError = error as? LibraryStoreError else {
            if let storageError = error as? LibraryStorageError {
                return errorMessage(storageError)
            }
            if let remoteImageError = error as? RemoteImageImportError {
                return errorMessage(remoteImageError)
            }
            if let assetExportError = error as? AssetExportError {
                return errorMessage(assetExportError)
            }
            return error.localizedDescription
        }

        return switch libraryError {
        case .noCurrentLibrary:
            string("Create or open a Momento library before importing assets.")
        case .missingRecentLibrary:
            string("This recent library is no longer available.")
        case .unsupportedLibraryURL:
            string("Choose a .momento package.")
        case .duplicateLibraryID:
            string("This library is already in your recent libraries.")
        case .invalidLibraryName:
            string("Enter a library name.")
        case .invalidAssetName:
            string("Enter an asset title.")
        case .invalidTagName:
            string("Enter a tag name.")
        case .missingAsset:
            string("This asset is no longer available.")
        case .missingTag:
            string("This tag is no longer available.")
        }
    }

    private func errorMessage(_ error: AssetExportError) -> String {
        switch error {
        case .emptySelection:
            string("Select one or more assets to export.")
        case .unsupportedFormat:
            string("The selected asset cannot be exported in that format.")
        }
    }

    private func errorMessage(_ error: RemoteImageImportError) -> String {
        switch error {
        case .unsupportedURLScheme:
            string("Only HTTP or HTTPS image URLs can be imported from Chrome.")
        case .downloadFailed:
            string("Momento could not download the image from Chrome.")
        case .unsupportedImageContent:
            string("The selected Chrome image is not a supported image format.")
        }
    }

    private func errorMessage(_ error: LibraryStorageError) -> String {
        switch error {
        case .assetOutsideLibrary:
            string("Asset storage must stay inside the selected library package.")
        case .libraryPackageAlreadyExists:
            string("A library already exists at the selected location.")
        case .missingLibraryDatabase:
            string("The selected library database is missing.")
        case .missingLibraryPackage:
            string("The selected library no longer exists.")
        case .unsupportedSchemaVersion(let version):
            format("Unsupported library schema version: %d.", version)
        }
    }

    func dateTime(_ date: Date) -> String {
        date.formatted(
            Date.FormatStyle(date: .abbreviated, time: .shortened)
                .locale(locale)
        )
    }

    func relativeOrDateTime(_ date: Date, relativeTo referenceDate: Date = .now) -> String {
        let elapsedYears = Calendar.current.dateComponents([.year], from: date, to: referenceDate).year ?? 0
        if elapsedYears >= 1 {
            return dateTime(date)
        }

        let formatter = RelativeDateTimeFormatter()
        formatter.locale = locale
        formatter.unitsStyle = .full
        formatter.dateTimeStyle = .numeric
        return formatter.localizedString(for: date, relativeTo: referenceDate)
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
