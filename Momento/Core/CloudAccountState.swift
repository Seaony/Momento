// 中文注释：Cloud account state 是 cloud library 写入前的身份边界，不负责具体同步。
import Foundation

nonisolated struct CloudAccountIdentity: Codable, Equatable, Hashable, Sendable {
    var cloudAccountID: String
    var ubiquityIdentityTokenHash: String?

    init(cloudAccountID: String, ubiquityIdentityTokenHash: String? = nil) {
        self.cloudAccountID = cloudAccountID
        self.ubiquityIdentityTokenHash = ubiquityIdentityTokenHash
    }
}

nonisolated enum CloudAccountUnavailableReason: String, Equatable, Sendable {
    case noAccount
    case temporarilyUnavailable
    case couldNotDetermine
    case unknown
}

nonisolated enum CloudAccountState: Equatable, Sendable {
    case available(CloudAccountIdentity)
    case unavailable(CloudAccountUnavailableReason)
    case restricted
    case error(String)
    case mismatch(expectedCloudAccountID: String, actualCloudAccountID: String?)

    var cloudAccountID: String? {
        switch self {
        case .available(let identity):
            identity.cloudAccountID
        case .unavailable, .restricted, .error, .mismatch:
            nil
        }
    }

    var canCreateCloudLibraryPlaceholder: Bool {
        if case .available(let identity) = self {
            return identity.cloudAccountID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
        }
        return false
    }
}
