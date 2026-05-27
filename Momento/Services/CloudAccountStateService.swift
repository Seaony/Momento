// 中文注释：本服务只解析 CloudKit 账户可用性和身份变化，不创建 CloudKit 记录。
import CloudKit
import CryptoKit
import Foundation

nonisolated struct CloudAccountStateService {
    private let fetchAccountStatus: @Sendable () async throws -> CKAccountStatus
    private let fetchUserRecordName: @Sendable () async throws -> String
    private let currentUbiquityIdentityTokenData: @Sendable () -> Data?

    init(
        fetchAccountStatus: @escaping @Sendable () async throws -> CKAccountStatus,
        fetchUserRecordName: @escaping @Sendable () async throws -> String,
        currentUbiquityIdentityTokenData: @escaping @Sendable () -> Data? = { nil }
    ) {
        self.fetchAccountStatus = fetchAccountStatus
        self.fetchUserRecordName = fetchUserRecordName
        self.currentUbiquityIdentityTokenData = currentUbiquityIdentityTokenData
    }

    init(
        container: CKContainer = .default(),
        currentUbiquityIdentityTokenData: @escaping @Sendable () -> Data? = Self.currentUbiquityIdentityTokenData
    ) {
        self.init(
            fetchAccountStatus: {
                try await Self.accountStatus(for: container)
            },
            fetchUserRecordName: {
                try await Self.userRecordName(for: container)
            },
            currentUbiquityIdentityTokenData: currentUbiquityIdentityTokenData
        )
    }

    func currentState(expectedCloudAccountID: String? = nil) async -> CloudAccountState {
        let status: CKAccountStatus
        do {
            status = try await fetchAccountStatus()
        } catch {
            return .error(error.localizedDescription)
        }

        switch status {
        case .available:
            return await availableState(expectedCloudAccountID: expectedCloudAccountID)
        case .noAccount:
            return .unavailable(.noAccount)
        case .restricted:
            return .restricted
        case .couldNotDetermine:
            return .unavailable(.couldNotDetermine)
        case .temporarilyUnavailable:
            return .unavailable(.temporarilyUnavailable)
        @unknown default:
            return .unavailable(.unknown)
        }
    }

    func observeAccountChanges(
        notificationCenter: NotificationCenter = .default,
        handler: @escaping @Sendable () -> Void
    ) -> CloudAccountChangeObservation {
        CloudAccountChangeObservation(
            notificationCenter: notificationCenter,
            tokens: [
                notificationCenter.addObserver(
                    forName: .CKAccountChanged,
                    object: nil,
                    queue: .main
                ) { _ in
                    handler()
                },
                notificationCenter.addObserver(
                    forName: .NSUbiquityIdentityDidChange,
                    object: nil,
                    queue: .main
                ) { _ in
                    handler()
                }
            ]
        )
    }

    private func availableState(expectedCloudAccountID: String?) async -> CloudAccountState {
        let userRecordName: String
        do {
            userRecordName = try await fetchUserRecordName()
        } catch {
            return .error(error.localizedDescription)
        }

        let identity = CloudAccountIdentity(
            cloudAccountID: Self.sha256Hex(userRecordName),
            ubiquityIdentityTokenHash: currentUbiquityIdentityTokenData().map(Self.sha256Hex(data:))
        )
        if let expectedCloudAccountID,
           expectedCloudAccountID != identity.cloudAccountID {
            return .mismatch(
                expectedCloudAccountID: expectedCloudAccountID,
                actualCloudAccountID: identity.cloudAccountID
            )
        }
        return .available(identity)
    }

    private static func accountStatus(for container: CKContainer) async throws -> CKAccountStatus {
        try await withCheckedThrowingContinuation { continuation in
            container.accountStatus { status, error in
                if let error {
                    continuation.resume(throwing: error)
                } else {
                    continuation.resume(returning: status)
                }
            }
        }
    }

    private static func userRecordName(for container: CKContainer) async throws -> String {
        try await withCheckedThrowingContinuation { continuation in
            container.fetchUserRecordID { recordID, error in
                if let error {
                    continuation.resume(throwing: error)
                } else if let recordName = recordID?.recordName {
                    continuation.resume(returning: recordName)
                } else {
                    continuation.resume(throwing: CloudAccountStateServiceError.missingUserRecordID)
                }
            }
        }
    }

    private static func archivedUbiquityIdentityTokenData(
        from token: (any NSCoding & NSCopying & NSObjectProtocol)?
    ) -> Data? {
        guard let token else {
            return nil
        }
        return try? NSKeyedArchiver.archivedData(withRootObject: token, requiringSecureCoding: false)
    }

    private static func currentUbiquityIdentityTokenData() -> Data? {
        archivedUbiquityIdentityTokenData(from: FileManager.default.ubiquityIdentityToken)
    }

    static func sha256Hex(_ string: String) -> String {
        sha256Hex(data: Data(string.utf8))
    }

    static func sha256Hex(data: Data) -> String {
        SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
    }
}

nonisolated struct CloudAccountChangeObservation {
    private let notificationCenter: NotificationCenter
    private let tokens: [NSObjectProtocol]

    init(notificationCenter: NotificationCenter, tokens: [NSObjectProtocol]) {
        self.notificationCenter = notificationCenter
        self.tokens = tokens
    }

    func invalidate() {
        tokens.forEach(notificationCenter.removeObserver)
    }
}

private enum CloudAccountStateServiceError: LocalizedError {
    case missingUserRecordID

    var errorDescription: String? {
        switch self {
        case .missingUserRecordID:
            "CloudKit did not return a current user record ID."
        }
    }
}
