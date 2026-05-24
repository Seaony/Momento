// 中文注释：Sparkle 负责下载、签名校验和替换 App；这里仅保留 Momento 需要暴露给菜单和设置页的最小控制面。
import Combine
import Foundation
import Sparkle

@MainActor
final class AppUpdateService: NSObject, ObservableObject {
    @Published private(set) var canCheckForUpdates = false
    @Published private(set) var availableUpdateDisplayVersion: String?

    private var updaterController: SPUStandardUpdaterController!
    private var canCheckObservation: NSKeyValueObservation?

    override init() {
        super.init()

        updaterController = SPUStandardUpdaterController(
            startingUpdater: true,
            updaterDelegate: self,
            userDriverDelegate: self
        )

        canCheckObservation = updaterController.updater.observe(
            \.canCheckForUpdates,
            options: [.initial, .new]
        ) { [weak self] updater, _ in
            Task { @MainActor [weak self] in
                self?.canCheckForUpdates = updater.canCheckForUpdates
            }
        }
    }

    func checkForUpdates() {
        updaterController.checkForUpdates(nil)
    }
}

extension AppUpdateService: SPUUpdaterDelegate {
    func updater(_ updater: SPUUpdater, didFindValidUpdate item: SUAppcastItem) {
        availableUpdateDisplayVersion = item.displayVersionString
    }

    func updaterDidNotFindUpdate(_ updater: SPUUpdater, error: Error) {
        availableUpdateDisplayVersion = nil
    }

    func updater(
        _ updater: SPUUpdater,
        userDidMake choice: SPUUserUpdateChoice,
        forUpdate updateItem: SUAppcastItem,
        state: SPUUserUpdateState
    ) {
        switch choice {
        case .dismiss:
            availableUpdateDisplayVersion = updateItem.displayVersionString
        case .install, .skip:
            availableUpdateDisplayVersion = nil
        @unknown default:
            break
        }
    }
}

extension AppUpdateService: SPUStandardUserDriverDelegate {
    var supportsGentleScheduledUpdateReminders: Bool {
        true
    }

    func standardUserDriverShouldHandleShowingScheduledUpdate(
        _ update: SUAppcastItem,
        andInImmediateFocus immediateFocus: Bool
    ) -> Bool {
        availableUpdateDisplayVersion = update.displayVersionString
        return false
    }

    func standardUserDriverWillHandleShowingUpdate(
        _ handleShowingUpdate: Bool,
        forUpdate update: SUAppcastItem,
        state: SPUUserUpdateState
    ) {
        availableUpdateDisplayVersion = update.displayVersionString
    }
}
