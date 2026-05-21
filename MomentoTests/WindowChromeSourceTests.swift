import Foundation
import XCTest

final class WindowChromeSourceTests: XCTestCase {
    func testContentViewInstallsGlobalWindowCornerRadiusConfigurator() throws {
        let source = try String(contentsOf: contentViewURL(), encoding: .utf8)

        XCTAssertTrue(source.contains("MomentoWindowCornerRadiusConfigurator()"))
        XCTAssertTrue(source.contains("momentoWindowCornerRadius: CGFloat = 28"))
        XCTAssertTrue(source.contains("contentView.layer?.cornerRadius = momentoWindowCornerRadius"))
        XCTAssertTrue(source.contains("contentView.layer?.cornerCurve = .continuous"))
        XCTAssertTrue(source.contains("contentView.layer?.masksToBounds = true"))
        XCTAssertTrue(source.contains("restoreContentViewConfiguration()"))
    }

    private func contentViewURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/ContentView.swift")
    }
}
