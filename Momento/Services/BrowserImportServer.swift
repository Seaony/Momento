import Foundation
import Network

nonisolated enum BrowserImportServerError: LocalizedError, Sendable {
    case invalidRequest
    case unsupportedEndpoint

    var errorDescription: String? {
        switch self {
        case .invalidRequest:
            "Momento received an invalid browser import request."
        case .unsupportedEndpoint:
            "Momento does not support this browser import endpoint."
        }
    }
}

nonisolated enum BrowserImportRoute: Equatable, Sendable {
    case status
    case importImage(URL)
}

nonisolated enum BrowserImportParseResult: Equatable, Sendable {
    case incomplete
    case request(BrowserImportRoute)
    case invalid
}

nonisolated final class BrowserImportServer: @unchecked Sendable {
    static let port: UInt16 = 47641

    private let queue = DispatchQueue(label: "com.seaony.Momento.browser-import-server")
    private let listenPort: UInt16
    private var listener: NWListener?
    private var importHandler: (@MainActor @Sendable (URL) async throws -> Void)?

    init(port: UInt16 = BrowserImportServer.port) {
        listenPort = port
    }

    func start(importHandler: @escaping @MainActor @Sendable (URL) async throws -> Void) throws {
        self.importHandler = importHandler
        guard listener == nil else {
            return
        }

        let port = NWEndpoint.Port(rawValue: listenPort)!
        let parameters = NWParameters.tcp
        parameters.allowLocalEndpointReuse = true
        parameters.acceptLocalOnly = true

        let listener = try NWListener(using: parameters, on: port)
        listener.newConnectionHandler = { [weak self] connection in
            self?.accept(connection)
        }
        listener.start(queue: queue)
        self.listener = listener
    }

    func stop() {
        listener?.cancel()
        listener = nil
    }

    private func accept(_ connection: NWConnection) {
        connection.start(queue: queue)
        receive(on: connection, buffer: Data())
    }

    private func receive(on connection: NWConnection, buffer: Data) {
        connection.receive(minimumIncompleteLength: 1, maximumLength: 64 * 1024) { [weak self] data, _, isComplete, error in
            guard let self else {
                connection.cancel()
                return
            }

            guard error == nil else {
                connection.cancel()
                return
            }

            var nextBuffer = buffer
            if let data {
                nextBuffer.append(data)
            }

            switch BrowserImportHTTP.parseRequest(nextBuffer) {
            case .incomplete where !isComplete && nextBuffer.count < 512 * 1024:
                self.receive(on: connection, buffer: nextBuffer)
            case .request(let route):
                self.handle(route, on: connection)
            case .incomplete, .invalid:
                self.respond(
                    on: connection,
                    statusCode: 400,
                    body: ["ok": false, "error": BrowserImportServerError.invalidRequest.localizedDescription]
                )
            }
        }
    }

    private func handle(_ route: BrowserImportRoute, on connection: NWConnection) {
        switch route {
        case .status:
            respond(on: connection, statusCode: 200, body: ["ok": true, "status": "ready"])
        case .importImage(let url):
            guard let importHandler else {
                respond(on: connection, statusCode: 503, body: ["ok": false, "error": "Momento is not ready."])
                return
            }

            Task {
                do {
                    try await importHandler(url)
                    respond(on: connection, statusCode: 200, body: ["ok": true])
                } catch {
                    respond(on: connection, statusCode: 500, body: ["ok": false, "error": error.localizedDescription])
                }
            }
        }
    }

    private func respond(on connection: NWConnection, statusCode: Int, body: [String: Any]) {
        let response = BrowserImportHTTP.response(statusCode: statusCode, body: body)
        connection.send(content: response, completion: .contentProcessed { _ in
            connection.cancel()
        })
    }
}

nonisolated enum BrowserImportHTTP {
    static func parseRequest(_ data: Data) -> BrowserImportParseResult {
        guard let headerEnd = data.range(of: Data("\r\n\r\n".utf8)) else {
            return .incomplete
        }

        guard let headerText = String(data: data[..<headerEnd.lowerBound], encoding: .utf8) else {
            return .invalid
        }

        let lines = headerText.components(separatedBy: "\r\n")
        guard let requestLine = lines.first else {
            return .invalid
        }

        let requestParts = requestLine.split(separator: " ", maxSplits: 2).map(String.init)
        guard requestParts.count >= 2 else {
            return .invalid
        }

        let method = requestParts[0]
        let path = requestParts[1]
        let headers = lines.dropFirst().reduce(into: [String: String]()) { result, line in
            guard let separator = line.firstIndex(of: ":") else {
                return
            }

            let key = String(line[..<separator]).lowercased()
            let value = line[line.index(after: separator)...].trimmingCharacters(in: .whitespaces)
            result[key] = value
        }

        guard isAllowedOrigin(headers["origin"]) else {
            return .invalid
        }

        let bodyStart = headerEnd.upperBound
        let contentLength = Int(headers["content-length"] ?? "0") ?? 0
        guard data.count >= bodyStart + contentLength else {
            return .incomplete
        }

        if method == "GET", path == "/api/v1/status" {
            return .request(.status)
        }

        guard method == "POST", path == "/api/v1/import/image" else {
            return .invalid
        }

        let body = data[bodyStart..<(bodyStart + contentLength)]
        guard let payload = try? JSONDecoder().decode(BrowserImportImagePayload.self, from: body),
              let url = URL(string: payload.url) else {
            return .invalid
        }

        return .request(.importImage(url))
    }

    static func response(statusCode: Int, body: [String: Any]) -> Data {
        let reason = statusReason(for: statusCode)
        let bodyData = (try? JSONSerialization.data(withJSONObject: body)) ?? Data("{}".utf8)
        var response = Data()
        response.append(Data("HTTP/1.1 \(statusCode) \(reason)\r\n".utf8))
        response.append(Data("Content-Type: application/json\r\n".utf8))
        response.append(Data("Content-Length: \(bodyData.count)\r\n".utf8))
        response.append(Data("Connection: close\r\n".utf8))
        response.append(Data("\r\n".utf8))
        response.append(bodyData)
        return response
    }

    private static func isAllowedOrigin(_ origin: String?) -> Bool {
        guard let origin else {
            return true
        }

        return origin.hasPrefix("chrome-extension://")
    }

    private static func statusReason(for statusCode: Int) -> String {
        switch statusCode {
        case 200:
            "OK"
        case 400:
            "Bad Request"
        case 500:
            "Internal Server Error"
        case 503:
            "Service Unavailable"
        default:
            "OK"
        }
    }
}

nonisolated private struct BrowserImportImagePayload: Decodable {
    var url: String
}
