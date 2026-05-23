import Foundation

nonisolated enum MomentoExternalImportRequest: Equatable, Sendable {
    case remoteImage(URL)

    init?(url: URL) {
        guard url.scheme?.lowercased() == "momento",
              url.host?.lowercased() == "import",
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
              let sourceValue = components.queryItems?.first(where: { $0.name == "url" })?.value,
              let sourceURL = URL(string: sourceValue) else {
            return nil
        }

        self = .remoteImage(sourceURL)
    }
}
