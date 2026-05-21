import Foundation
import XCTest

final class LocalizationCatalogTests: XCTestCase {
    func testEveryLocalizationKeyUsedByCodeExistsInStringCatalog() throws {
        let sourceKeys = try localizationKeysUsedBySource()
        let catalog = try stringCatalog()
        let missingKeys = sourceKeys.filter { catalog[$0] == nil }.sorted()

        XCTAssertTrue(missingKeys.isEmpty, "Missing Localizable.xcstrings keys: \(missingKeys)")
    }

    func testCatalogEntriesHaveEnglishAndSimplifiedChineseValues() throws {
        let catalog = try stringCatalog()
        var missingLocalizations: [String] = []

        for key in try localizationKeysUsedBySource().sorted() {
            guard let entry = catalog[key],
                  let localizations = entry["localizations"] as? [String: Any] else {
                continue
            }

            for locale in ["en", "zh-Hans"] {
                guard
                    let localeEntry = localizations[locale] as? [String: Any],
                    let stringUnit = localeEntry["stringUnit"] as? [String: Any],
                    let value = stringUnit["value"] as? String,
                    !value.isEmpty
                else {
                    missingLocalizations.append("\(key) [\(locale)]")
                    continue
                }
            }
        }

        XCTAssertTrue(missingLocalizations.isEmpty, "Missing Localizable.xcstrings values: \(missingLocalizations)")
    }

    private func localizationKeysUsedBySource() throws -> Set<String> {
        let sourceRoot = repositoryRoot().appendingPathComponent("Momento", isDirectory: true)
        let pattern = #"localization\.(?:string|format)\("([^"]+)""#
        let regularExpression = try NSRegularExpression(pattern: pattern)
        var keys = Set<String>()

        guard let enumerator = FileManager.default.enumerator(
            at: sourceRoot,
            includingPropertiesForKeys: [.isRegularFileKey],
            options: [.skipsHiddenFiles]
        ) else {
            return keys
        }

        for case let fileURL as URL in enumerator where fileURL.pathExtension == "swift" {
            let source = try String(contentsOf: fileURL, encoding: .utf8)
            let range = NSRange(source.startIndex..<source.endIndex, in: source)

            regularExpression.enumerateMatches(in: source, range: range) { match, _, _ in
                guard let match, match.numberOfRanges > 1,
                      let keyRange = Range(match.range(at: 1), in: source) else {
                    return
                }

                keys.insert(String(source[keyRange]))
            }
        }

        return keys
    }

    private func stringCatalog() throws -> [String: [String: Any]] {
        let catalogURL = repositoryRoot().appendingPathComponent("Momento/Localizable.xcstrings")
        let data = try Data(contentsOf: catalogURL)
        let root = try XCTUnwrap(JSONSerialization.jsonObject(with: data) as? [String: Any])
        return try XCTUnwrap(root["strings"] as? [String: [String: Any]])
    }

    private func repositoryRoot() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
    }
}
