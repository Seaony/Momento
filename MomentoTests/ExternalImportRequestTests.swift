import Foundation
import XCTest
@testable import Momento

final class ExternalImportRequestTests: XCTestCase {
    func testParsesRemoteImageImportURL() throws {
        let importURL = try XCTUnwrap(URL(string: "momento://import?url=https%3A%2F%2Fexample.com%2Fimage.png%3Fsize%3Dlarge"))
        let sourceURL = try XCTUnwrap(URL(string: "https://example.com/image.png?size=large"))

        XCTAssertEqual(MomentoExternalImportRequest(url: importURL), .remoteImage(sourceURL))
    }

    func testIgnoresUnknownExternalURL() throws {
        let url = try XCTUnwrap(URL(string: "momento://open?url=https%3A%2F%2Fexample.com%2Fimage.png"))

        XCTAssertNil(MomentoExternalImportRequest(url: url))
    }
}
