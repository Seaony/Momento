import Foundation
import XCTest

final class LiquidGlassSourceTests: XCTestCase {
    func testGlassBackgroundUsesNativeSwiftUIGlassEffect() throws {
        let source = try String(contentsOf: designSystemURL(), encoding: .utf8)

        XCTAssertTrue(source.contains(".glassEffect(glass, in: RoundedRectangle(cornerRadius: cornerRadius, style: .continuous))"))
        XCTAssertFalse(source.contains("MomentoVisualEffectView(material: material)"))
        XCTAssertFalse(source.contains("strokeOpacity"))
        XCTAssertFalse(source.contains(".strokeBorder(.white.opacity"))
        XCTAssertFalse(source.contains(".shadow(color: .black.opacity"))
    }

    private func designSystemURL() -> URL {
        URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("Momento/DesignSystem/MomentoGlass.swift")
    }
}
