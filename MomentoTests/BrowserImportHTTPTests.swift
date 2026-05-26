// 中文注释：本测试覆盖浏览器本地导入服务的 HTTP 路由、JSON 解析和监听启动行为。
import Foundation
import XCTest
@testable import Momento

final class BrowserImportHTTPTests: XCTestCase {
    func testServerStartsWithLocalListener() throws {
        let server = BrowserImportServer(port: 0)
        try server.start { _ in }
        server.stop()
    }

    func testParsesChromeExtensionImageImportRequest() throws {
        let body = #"{"url":"https://example.com/source.png"}"#
        let imageURL = try XCTUnwrap(URL(string: "https://example.com/source.png"))
        let request = httpRequest(
            method: "POST",
            path: "/api/v1/import/image",
            headers: [
                "Host": "127.0.0.1:47641",
                "Origin": "chrome-extension://example/",
                "Content-Type": "application/json",
                "Content-Length": "\(Data(body.utf8).count)"
            ],
            body: body
        )

        XCTAssertEqual(
            BrowserImportHTTP.parseRequest(Data(request.utf8)),
            .request(.importImage(BrowserImageImportRequest(imageURL: imageURL, sourcePageURL: nil)))
        )
    }

    func testParsesChromeExtensionImageImportSourcePageURL() throws {
        let body = #"{"url":"https://example.com/source.png","pageUrl":"https://example.com/articles/reference"}"#
        let imageURL = try XCTUnwrap(URL(string: "https://example.com/source.png"))
        let sourcePageURL = try XCTUnwrap(URL(string: "https://example.com/articles/reference"))
        let request = httpRequest(
            method: "POST",
            path: "/api/v1/import/image",
            headers: [
                "Host": "127.0.0.1:47641",
                "Origin": "chrome-extension://example/",
                "Content-Type": "application/json",
                "Content-Length": "\(Data(body.utf8).count)"
            ],
            body: body
        )

        XCTAssertEqual(
            BrowserImportHTTP.parseRequest(Data(request.utf8)),
            .request(.importImage(BrowserImageImportRequest(imageURL: imageURL, sourcePageURL: sourcePageURL)))
        )
    }

    func testParsesChromeExtensionImageImportFeedbackFlag() throws {
        let body = #"{"url":"https://example.com/source.png","playFeedback":false}"#
        let imageURL = try XCTUnwrap(URL(string: "https://example.com/source.png"))
        let request = httpRequest(
            method: "POST",
            path: "/api/v1/import/image",
            headers: [
                "Host": "127.0.0.1:47641",
                "Origin": "chrome-extension://example/",
                "Content-Type": "application/json",
                "Content-Length": "\(Data(body.utf8).count)"
            ],
            body: body
        )

        XCTAssertEqual(
            BrowserImportHTTP.parseRequest(Data(request.utf8)),
            .request(.importImage(BrowserImageImportRequest(
                imageURL: imageURL,
                sourcePageURL: nil,
                playFeedback: false
            )))
        )
    }

    func testRejectsWebPageOrigin() {
        let body = #"{"url":"https://example.com/source.png"}"#
        let request = httpRequest(
            method: "POST",
            path: "/api/v1/import/image",
            headers: [
                "Origin": "https://example.com",
                "Content-Length": "\(Data(body.utf8).count)"
            ],
            body: body
        )

        XCTAssertEqual(BrowserImportHTTP.parseRequest(Data(request.utf8)), .invalid)
    }

    func testParsesStatusRequest() {
        let request = httpRequest(
            method: "GET",
            path: "/api/v1/status",
            headers: ["Host": "127.0.0.1:47641"]
        )

        XCTAssertEqual(BrowserImportHTTP.parseRequest(Data(request.utf8)), .request(.status))
    }

    private func httpRequest(
        method: String,
        path: String,
        headers: [String: String],
        body: String = ""
    ) -> String {
        let headerLines = headers.map { "\($0.key): \($0.value)" }.joined(separator: "\r\n")
        return "\(method) \(path) HTTP/1.1\r\n\(headerLines)\r\n\r\n\(body)"
    }
}
