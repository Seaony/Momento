// 中文注释：浏览器导入通知是非关键反馈路径，失败时不影响素材已导入的主流程。
import Foundation
import UserNotifications

nonisolated enum BrowserImportNotificationService {
    @discardableResult
    static func notifyImageSaved(title: String, body: String) async -> Bool {
        let center = UNUserNotificationCenter.current()

        do {
            let granted = try await center.requestAuthorization(options: [.alert, .sound])
            guard granted else {
                return false
            }

            let content = UNMutableNotificationContent()
            content.title = title
            content.body = body
            content.sound = .default

            let request = UNNotificationRequest(
                identifier: "browser-import-\(UUID().uuidString)",
                content: content,
                trigger: nil
            )
            try await center.add(request)
            return true
        } catch {
            return false
        }
    }
}
