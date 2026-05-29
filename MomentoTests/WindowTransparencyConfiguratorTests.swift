import AppKit
@testable import Momento
import XCTest

final class WindowTransparencyConfiguratorTests: XCTestCase {
    @MainActor
    func testBackingColorResolvesAgainstCurrentAppearance() throws {
        let color = WindowTransparencyConfigurator.adaptiveBackingColor()
        let lightComponents = try components(of: color, appearanceName: .aqua)
        let darkComponents = try components(of: color, appearanceName: .darkAqua)

        XCTAssertGreaterThan(lightComponents.red, darkComponents.red)
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
