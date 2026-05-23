// 中文注释：本文件解析外部 URL scheme，只把受支持的请求转换成应用内部意图。
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
