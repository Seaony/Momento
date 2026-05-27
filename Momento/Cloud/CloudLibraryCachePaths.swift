import Foundation

nonisolated enum CloudLibraryCachePathError: LocalizedError, Equatable {
    case invalidCloudAccountID
    case invalidLibraryID
    case invalidContentHash
    case invalidFileExtension

    var errorDescription: String? {
        switch self {
        case .invalidCloudAccountID:
            "Cloud account ID is not a valid cache path component."
        case .invalidLibraryID:
            "Cloud library ID is not a valid cache path component."
        case .invalidContentHash:
            "Cloud asset content hash is not a valid cache path component."
        case .invalidFileExtension:
            "Cloud asset file extension is not a valid cache path component."
        }
    }
}

nonisolated enum CloudCacheFileRole: Sendable {
    case uploadPendingOriginal
    case syncedOriginal
    case thumbnail
}

nonisolated struct CloudLibraryCachePaths: Sendable {
    let applicationSupportRoot: URL

    init(applicationSupportRoot: URL = Self.defaultApplicationSupportRoot()) {
        self.applicationSupportRoot = applicationSupportRoot
    }

    func accountRoot(cloudAccountID: String) throws -> URL {
        applicationSupportRoot
            .appendingPathComponent("CloudLibraries", isDirectory: true)
            .appendingPathComponent(try safePathComponent(cloudAccountID, error: .invalidCloudAccountID), isDirectory: true)
    }

    func libraryRoot(cloudAccountID: String, libraryID: String) throws -> URL {
        try accountRoot(cloudAccountID: cloudAccountID)
            .appendingPathComponent(try safePathComponent(libraryID, error: .invalidLibraryID), isDirectory: true)
    }

    func cacheDatabaseURL(cloudAccountID: String, libraryID: String) throws -> URL {
        try libraryRoot(cloudAccountID: cloudAccountID, libraryID: libraryID)
            .appendingPathComponent("cache.sqlite", isDirectory: false)
    }

    func assetsRoot(cloudAccountID: String, libraryID: String) throws -> URL {
        try libraryRoot(cloudAccountID: cloudAccountID, libraryID: libraryID)
            .appendingPathComponent("assets", isDirectory: true)
    }

    func thumbnailsRoot(cloudAccountID: String, libraryID: String) throws -> URL {
        try libraryRoot(cloudAccountID: cloudAccountID, libraryID: libraryID)
            .appendingPathComponent("thumbnails", isDirectory: true)
    }

    func originalURL(
        cloudAccountID: String,
        libraryID: String,
        contentHash: String,
        fileExtension: String
    ) throws -> URL {
        let safeHash = try safePathComponent(contentHash, error: .invalidContentHash)
        let safeExtension = try safeFileExtension(fileExtension)
        return try assetsRoot(cloudAccountID: cloudAccountID, libraryID: libraryID)
            .appendingPathComponent(String(safeHash.prefix(2)), isDirectory: true)
            .appendingPathComponent(safeHash, isDirectory: false)
            .appendingPathExtension(safeExtension)
    }

    func thumbnailURL(
        cloudAccountID: String,
        libraryID: String,
        contentHash: String
    ) throws -> URL {
        let safeHash = try safePathComponent(contentHash, error: .invalidContentHash)
        return try thumbnailsRoot(cloudAccountID: cloudAccountID, libraryID: libraryID)
            .appendingPathComponent(safeHash, isDirectory: false)
            .appendingPathExtension("png")
    }

    func prepareLibraryDirectories(cloudAccountID: String, libraryID: String) throws {
        for directory in [
            try libraryRoot(cloudAccountID: cloudAccountID, libraryID: libraryID),
            try assetsRoot(cloudAccountID: cloudAccountID, libraryID: libraryID),
            try thumbnailsRoot(cloudAccountID: cloudAccountID, libraryID: libraryID)
        ] {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        }
    }

    func applyDurabilityAttributes(to fileURL: URL, role: CloudCacheFileRole) throws {
        var values = URLResourceValues()
        switch role {
        case .uploadPendingOriginal:
            values.isExcludedFromBackup = false
        case .syncedOriginal, .thumbnail:
            values.isExcludedFromBackup = true
        }
        var mutableURL = fileURL
        try mutableURL.setResourceValues(values)
    }

    private func safePathComponent(
        _ value: String,
        error: CloudLibraryCachePathError
    ) throws -> String {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty,
              trimmed != ".",
              trimmed != "..",
              trimmed.range(of: #"[A-Za-z0-9_-]+"#, options: .regularExpression)?.lowerBound == trimmed.startIndex,
              trimmed.range(of: #"[A-Za-z0-9_-]+"#, options: .regularExpression)?.upperBound == trimmed.endIndex else {
            throw error
        }
        return trimmed
    }

    private func safeFileExtension(_ value: String) throws -> String {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !trimmed.isEmpty,
              trimmed.count <= 16,
              trimmed.range(of: #"[A-Za-z0-9]+"#, options: .regularExpression)?.lowerBound == trimmed.startIndex,
              trimmed.range(of: #"[A-Za-z0-9]+"#, options: .regularExpression)?.upperBound == trimmed.endIndex else {
            throw CloudLibraryCachePathError.invalidFileExtension
        }
        return trimmed
    }

    private static func defaultApplicationSupportRoot() -> URL {
        let root = try? FileManager.default.url(
            for: .applicationSupportDirectory,
            in: .userDomainMask,
            appropriateFor: nil,
            create: true
        )
        return (root ?? FileManager.default.temporaryDirectory).appendingPathComponent("Momento", isDirectory: true)
    }
}
