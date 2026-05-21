import Foundation
import XCTest

final class WindowChromeSourceTests: XCTestCase {
    func testContentViewDoesNotOverrideSystemWindowCornerRadius() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("MomentoWindowCornerRadiusConfigurator"))
        XCTAssertFalse(source.contains("momentoWindowCornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerRadius"))
        XCTAssertFalse(source.contains("contentView.layer?.cornerCurve"))
        XCTAssertFalse(source.contains("contentView.layer?.masksToBounds"))
        XCTAssertFalse(source.contains("import QuartzCore"))
    }

    func testMainWindowUsesSystemToolbarChromeForLargeTahoeCorners() throws {
        let appSource = try String(contentsOf: appURL(), encoding: .utf8)
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let shellSource = try String(contentsOf: shellViewURL(), encoding: .utf8)

        XCTAssertTrue(appSource.contains(".windowToolbarStyle(.unified)"))
        XCTAssertFalse(appSource.contains(".windowStyle(.hiddenTitleBar)"))
        XCTAssertFalse(appSource.contains(".windowToolbarStyle(.unifiedCompact)"))

        XCTAssertTrue(contentSource.contains(".toolbar {"))
        XCTAssertTrue(contentSource.contains("ToolbarItemGroup(placement: .navigation)"))
        XCTAssertTrue(contentSource.contains("ToolbarItem(placement: .principal)"))
        XCTAssertTrue(contentSource.contains(".searchable(text: $store.searchQuery, placement: .toolbar"))
        XCTAssertFalse(shellSource.contains("MomentoTopBar("))
    }

    private func contentViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/ContentView.swift")
    }

    private func appURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/MomentoApp.swift")
    }

    private func shellViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Shell/MomentoShellView.swift")
    }
}
