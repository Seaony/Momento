import Foundation
import XCTest

final class WindowChromeSourceTests: XCTestCase {
    func testContentViewInstallsGlobalWindowChromeConfigurator() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("MomentoWindowChromeConfigurator()"))
        XCTAssertTrue(source.contains("momentoWindowCornerRadius: CGFloat = 36"))
        XCTAssertTrue(source.contains("window.isOpaque = false"))
        XCTAssertTrue(source.contains("window.backgroundColor = .clear"))
        XCTAssertTrue(source.contains("contentView.layer?.cornerRadius = momentoWindowCornerRadius"))
        XCTAssertTrue(source.contains("contentView.layer?.cornerCurve = .continuous"))
        XCTAssertTrue(source.contains("contentView.layer?.masksToBounds = true"))
        XCTAssertTrue(source.contains("window.invalidateShadow()"))
        XCTAssertTrue(source.contains("restoreWindowConfiguration()"))
    }

    private func contentViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/ContentView.swift")
    }
}
