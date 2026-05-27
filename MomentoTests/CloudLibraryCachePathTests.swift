import Foundation
import XCTest
@testable import Momento

final class CloudLibraryCachePathTests: XCTestCase {
    private var temporaryRoot: URL!

    override func setUpWithError() throws {
        temporaryRoot = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString, isDirectory: true)
        try FileManager.default.createDirectory(at: temporaryRoot, withIntermediateDirectories: true)
    }

    override func tearDownWithError() throws {
        if let temporaryRoot {
            try? FileManager.default.removeItem(at: temporaryRoot)
        }
        temporaryRoot = nil
    }

    func testBuildsAccountScopedCloudLibraryPaths() throws {
        let paths = CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)
        let cloudAccountID = "account_abc123"
        let libraryID = "library-001"

        XCTAssertEqual(
            try paths.libraryRoot(cloudAccountID: cloudAccountID, libraryID: libraryID).path,
            temporaryRoot
                .appendingPathComponent("CloudLibraries", isDirectory: true)
                .appendingPathComponent(cloudAccountID, isDirectory: true)
                .appendingPathComponent(libraryID, isDirectory: true)
                .path
        )
        XCTAssertEqual(
            try paths.cacheDatabaseURL(cloudAccountID: cloudAccountID, libraryID: libraryID).lastPathComponent,
            "cache.sqlite"
        )
        XCTAssertEqual(
            try paths.originalURL(
                cloudAccountID: cloudAccountID,
                libraryID: libraryID,
                contentHash: "abcdef123456",
                fileExtension: "JPG"
            ).path,
            temporaryRoot
                .appendingPathComponent("CloudLibraries", isDirectory: true)
                .appendingPathComponent(cloudAccountID, isDirectory: true)
                .appendingPathComponent(libraryID, isDirectory: true)
                .appendingPathComponent("assets", isDirectory: true)
                .appendingPathComponent("ab", isDirectory: true)
                .appendingPathComponent("abcdef123456.jpg", isDirectory: false)
                .path
        )
        XCTAssertEqual(
            try paths.thumbnailURL(
                cloudAccountID: cloudAccountID,
                libraryID: libraryID,
                contentHash: "abcdef123456"
            ).lastPathComponent,
            "abcdef123456.png"
        )
    }

    func testRejectsUnsafePathComponents() throws {
        let paths = CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)

        XCTAssertThrowsError(
            try paths.libraryRoot(cloudAccountID: "../other", libraryID: "library")
        ) { error in
            XCTAssertEqual(error as? CloudLibraryCachePathError, .invalidCloudAccountID)
        }
        XCTAssertThrowsError(
            try paths.libraryRoot(cloudAccountID: "account", libraryID: "library/other")
        ) { error in
            XCTAssertEqual(error as? CloudLibraryCachePathError, .invalidLibraryID)
        }
        XCTAssertThrowsError(
            try paths.originalURL(
                cloudAccountID: "account",
                libraryID: "library",
                contentHash: "hash/other",
                fileExtension: "jpg"
            )
        ) { error in
            XCTAssertEqual(error as? CloudLibraryCachePathError, .invalidContentHash)
        }
        XCTAssertThrowsError(
            try paths.originalURL(
                cloudAccountID: "account",
                libraryID: "library",
                contentHash: "abcdef",
                fileExtension: "../jpg"
            )
        ) { error in
            XCTAssertEqual(error as? CloudLibraryCachePathError, .invalidFileExtension)
        }
    }

    func testPrepareLibraryDirectoriesCreatesStableRoots() throws {
        let paths = CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)

        try paths.prepareLibraryDirectories(cloudAccountID: "account", libraryID: "library")

        XCTAssertTrue(
            FileManager.default.fileExists(
                atPath: try paths.libraryRoot(cloudAccountID: "account", libraryID: "library").path
            )
        )
        XCTAssertTrue(
            FileManager.default.fileExists(
                atPath: try paths.assetsRoot(cloudAccountID: "account", libraryID: "library").path
            )
        )
        XCTAssertTrue(
            FileManager.default.fileExists(
                atPath: try paths.thumbnailsRoot(cloudAccountID: "account", libraryID: "library").path
            )
        )
    }

    func testDurabilityAttributesKeepPendingUploadsBackedUp() throws {
        let paths = CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)
        let fileURL = temporaryRoot.appendingPathComponent("pending.jpg")
        try Data("image".utf8).write(to: fileURL)

        try paths.applyDurabilityAttributes(to: fileURL, role: .uploadPendingOriginal)

        let values = try fileURL.resourceValues(forKeys: [.isExcludedFromBackupKey])
        XCTAssertEqual(values.isExcludedFromBackup, false)
    }

    func testDurabilityAttributesExcludeSyncedFilesAndThumbnailsFromBackup() throws {
        let paths = CloudLibraryCachePaths(applicationSupportRoot: temporaryRoot)
        let syncedOriginalURL = temporaryRoot.appendingPathComponent("synced.jpg")
        let thumbnailURL = temporaryRoot.appendingPathComponent("thumb.png")
        try Data("image".utf8).write(to: syncedOriginalURL)
        try Data("thumbnail".utf8).write(to: thumbnailURL)

        try paths.applyDurabilityAttributes(to: syncedOriginalURL, role: .syncedOriginal)
        try paths.applyDurabilityAttributes(to: thumbnailURL, role: .thumbnail)

        let syncedValues = try syncedOriginalURL.resourceValues(forKeys: [.isExcludedFromBackupKey])
        let thumbnailValues = try thumbnailURL.resourceValues(forKeys: [.isExcludedFromBackupKey])
        XCTAssertEqual(syncedValues.isExcludedFromBackup, true)
        XCTAssertEqual(thumbnailValues.isExcludedFromBackup, true)
    }
}
