// 中文注释：本测试覆盖 CloudKit account 状态边界，不访问真实 iCloud 账户。
import CloudKit
import Foundation
import XCTest
@testable import Momento

final class CloudAccountStateServiceTests: XCTestCase {
    private struct TestCloudAccountError: LocalizedError {
        var errorDescription: String? {
            "Test CloudKit account error"
        }
    }

    func testAvailableStateUsesCloudKitUserRecordAsAccountIdentity() async {
        let tokenData = Data("ubiquity-token".utf8)
        let service = CloudAccountStateService(
            fetchAccountStatus: { .available },
            fetchUserRecordName: { "cloudkit-user-record" },
            currentUbiquityIdentityTokenData: { tokenData }
        )

        let expectedIdentity = CloudAccountIdentity(
            cloudAccountID: CloudAccountStateService.sha256Hex("cloudkit-user-record"),
            ubiquityIdentityTokenHash: CloudAccountStateService.sha256Hex(data: tokenData)
        )

        let state = await service.currentState()

        XCTAssertEqual(state, .available(expectedIdentity))
        XCTAssertTrue(state.canCreateCloudLibraryPlaceholder)
        XCTAssertEqual(state.cloudAccountID, expectedIdentity.cloudAccountID)
    }

    func testAvailableStateReadsUbiquityTokenOnEachValidation() async {
        let tokenData = LockedValue<Data?>(Data("first-token".utf8))
        let service = CloudAccountStateService(
            fetchAccountStatus: { .available },
            fetchUserRecordName: { "cloudkit-user-record" },
            currentUbiquityIdentityTokenData: {
                tokenData.get()
            }
        )

        let firstState = await service.currentState()
        tokenData.set(Data("second-token".utf8))
        let secondState = await service.currentState()

        guard case .available(let firstIdentity) = firstState,
              case .available(let secondIdentity) = secondState else {
            return XCTFail("Expected available states.")
        }
        XCTAssertEqual(firstIdentity.cloudAccountID, secondIdentity.cloudAccountID)
        XCTAssertNotEqual(
            firstIdentity.ubiquityIdentityTokenHash,
            secondIdentity.ubiquityIdentityTokenHash
        )
    }

    func testBlankAccountIdentityIsNotWritable() {
        let state = CloudAccountState.available(CloudAccountIdentity(cloudAccountID: " "))

        XCTAssertFalse(state.canCreateCloudLibraryPlaceholder)
    }

    func testUnavailableAccountDoesNotFetchUserRecordID() async {
        let service = CloudAccountStateService(
            fetchAccountStatus: { .noAccount },
            fetchUserRecordName: { throw TestCloudAccountError() }
        )

        let state = await service.currentState()

        XCTAssertEqual(state, .unavailable(.noAccount))
        XCTAssertFalse(state.canCreateCloudLibraryPlaceholder)
    }

    func testRestrictedAccountMapsToRestrictedState() async {
        let service = CloudAccountStateService(
            fetchAccountStatus: { .restricted },
            fetchUserRecordName: { throw TestCloudAccountError() }
        )

        let state = await service.currentState()

        XCTAssertEqual(state, .restricted)
        XCTAssertFalse(state.canCreateCloudLibraryPlaceholder)
    }

    func testAccountStatusFailureMapsToErrorState() async {
        let service = CloudAccountStateService(
            fetchAccountStatus: { throw TestCloudAccountError() },
            fetchUserRecordName: { "unexpected" }
        )

        let state = await service.currentState()

        XCTAssertEqual(state, .error("Test CloudKit account error"))
        XCTAssertFalse(state.canCreateCloudLibraryPlaceholder)
    }

    func testUserRecordFailureMapsToErrorState() async {
        let service = CloudAccountStateService(
            fetchAccountStatus: { .available },
            fetchUserRecordName: { throw TestCloudAccountError() }
        )

        let state = await service.currentState()

        XCTAssertEqual(state, .error("Test CloudKit account error"))
        XCTAssertFalse(state.canCreateCloudLibraryPlaceholder)
    }

    func testExpectedAccountMismatchBlocksWritableState() async {
        let service = CloudAccountStateService(
            fetchAccountStatus: { .available },
            fetchUserRecordName: { "new-cloudkit-user-record" }
        )
        let actualCloudAccountID = CloudAccountStateService.sha256Hex("new-cloudkit-user-record")

        let state = await service.currentState(expectedCloudAccountID: "previous-account")

        XCTAssertEqual(
            state,
            .mismatch(
                expectedCloudAccountID: "previous-account",
                actualCloudAccountID: actualCloudAccountID
            )
        )
        XCTAssertFalse(state.canCreateCloudLibraryPlaceholder)
        XCTAssertEqual(state.cloudAccountID, actualCloudAccountID)
    }

    func testAccountChangeObservationHandlesCloudKitAndUbiquityChanges() {
        let notificationCenter = NotificationCenter()
        let service = CloudAccountStateService(
            fetchAccountStatus: { .available },
            fetchUserRecordName: { "cloudkit-user-record" }
        )
        let expectation = expectation(description: "Account changes observed")
        expectation.expectedFulfillmentCount = 2

        let observation = service.observeAccountChanges(notificationCenter: notificationCenter) {
            expectation.fulfill()
        }

        notificationCenter.post(name: .CKAccountChanged, object: nil)
        notificationCenter.post(name: .NSUbiquityIdentityDidChange, object: nil)

        wait(for: [expectation], timeout: 1)
        observation.invalidate()
    }
}

private final class LockedValue<Value>: @unchecked Sendable {
    private let lock = NSLock()
    private var value: Value

    init(_ value: Value) {
        self.value = value
    }

    func get() -> Value {
        lock.lock()
        defer { lock.unlock() }
        return value
    }

    func set(_ newValue: Value) {
        lock.lock()
        defer { lock.unlock() }
        value = newValue
    }
}
