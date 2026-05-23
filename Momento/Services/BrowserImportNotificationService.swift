// 中文注释：浏览器导入成功反馈是非关键路径，失败时不影响素材已导入的主流程。
import Foundation

nonisolated enum BrowserImportNotificationService {
    static func playImageSavedFeedback() {
        AssetDeletionSoundPlayer.playDeletionSound()
    }
}
