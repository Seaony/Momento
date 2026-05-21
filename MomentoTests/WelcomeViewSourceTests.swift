import Foundation
import XCTest

final class WelcomeViewSourceTests: XCTestCase {
    func testWelcomeViewDoesNotExposeLibrarySwitcherMenu() throws {
        let source = try String(contentsOf: welcomeViewURL(), encoding: .utf8)

        XCTAssertFalse(source.contains("Menu {"))
        XCTAssertFalse(source.contains("libraryMenu"))
        XCTAssertFalse(source.contains("recentLibraries"))
        XCTAssertFalse(source.contains("onSwitchLibrary"))
        XCTAssertFalse(source.contains("No library selected"))
    }

    private func welcomeViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/Features/Library/MomentoLibraryWelcomeView.swift")
    }
}
