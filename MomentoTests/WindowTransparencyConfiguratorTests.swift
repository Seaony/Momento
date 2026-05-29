import AppKit
@testable import Momento
import XCTest

final class WindowTransparencyConfiguratorTests: XCTestCase {
    @MainActor
    func testBackingColorResolvesAgainstCurrentAppearance() throws {
        let color = WindowTransparencyConfigurator.adaptiveBackingColor()
        let lightComponents = try components(of: color, appearanceName: .aqua)
        let darkComponents = try components(of: color, appearanceName: .darkAqua)
        let expectedDarkComponents = try components(
            of: WindowTransparencyConfigurator.systemWindowBackingColor(for: try XCTUnwrap(NSAppearance(named: .darkAqua))),
            appearanceName: .darkAqua
        )

        XCTAssertEqual(lightComponents.red, WindowTransparencyConfigurator.lightBackingWhite, accuracy: 0.001)
        XCTAssertEqual(lightComponents.green, WindowTransparencyConfigurator.lightBackingWhite, accuracy: 0.001)
        XCTAssertEqual(lightComponents.blue, WindowTransparencyConfigurator.lightBackingWhite, accuracy: 0.001)
        XCTAssertEqual(darkComponents.red, expectedDarkComponents.red, accuracy: 0.001)
        XCTAssertEqual(darkComponents.green, expectedDarkComponents.green, accuracy: 0.001)
        XCTAssertEqual(darkComponents.blue, expectedDarkComponents.blue, accuracy: 0.001)
        XCTAssertEqual(lightComponents.alpha, WindowTransparencyConfigurator.backingOpacity, accuracy: 0.001)
        XCTAssertEqual(darkComponents.alpha, WindowTransparencyConfigurator.backingOpacity, accuracy: 0.001)
    }

    private func components(
        of color: NSColor,
        appearanceName: NSAppearance.Name
    ) throws -> (red: CGFloat, green: CGFloat, blue: CGFloat, alpha: CGFloat) {
        let appearance = try XCTUnwrap(NSAppearance(named: appearanceName))
        var resolvedColor: NSColor?

        appearance.performAsCurrentDrawingAppearance {
            resolvedColor = color.usingColorSpace(.deviceRGB)
        }

        let resolved = try XCTUnwrap(resolvedColor)
        return (
            red: resolved.redComponent,
            green: resolved.greenComponent,
            blue: resolved.blueComponent,
            alpha: resolved.alphaComponent
        )
    }
}
