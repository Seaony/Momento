import Foundation

nonisolated struct LibraryStorage: Sendable {
    static let packageExtension = "momento"
    static let legacyPackageExtension = "momentolibrary"
    static let supportedPackageExtensions = [packageExtension, legacyPackageExtension]

    var applicationSupportRoot: URL
    var trashURLs: [URL]

    init(applicationSupportRoot: URL? = nil, trashURLs: [URL]? = nil) {
        self.applicationSupportRoot = applicationSupportRoot ?? Self.defaultApplicationSupportRoot()
        self.trashURLs = trashURLs ?? FileManager.default.urls(for: .trashDirectory, in: .userDomainMask)
    }

    nonisolated func rootURL(for library: AssetLibrary) -> URL {
        if let packageURL = library.packageURL {
            return packageURL
        }

        return applicationSupportRoot
            .appendingPathComponent("Libraries", isDirectory: true)
            .appendingPathComponent(library.id, isDirectory: true)
            .appendingPathComponent(".library", isDirectory: true)
    }

    nonisolated func assetsURL(for library: AssetLibrary) -> URL {
        rootURL(for: library).appendingPathComponent("assets", isDirectory: true)
    }

    nonisolated func databaseURL(for library: AssetLibrary) -> URL {
        rootURL(for: library)
            .appendingPathComponent("database", isDirectory: true)
            .appendingPathComponent("library.sqlite")
    }

    nonisolated func assetStorageURL(
        forContentHash contentHash: String,
        fileExtension: String,
        in library: AssetLibrary
    ) -> URL {
        let prefix = String(contentHash.prefix(2))
        return assetsURL(for: library)
            .appendingPathComponent(prefix, isDirectory: true)
            .appendingPathComponent(contentHash)
            .appendingPathExtension(fileExtension)
    }

    nonisolated func thumbnailURL(forContentHash contentHash: String, in library: AssetLibrary) -> URL {
        rootURL(for: library)
            .appendingPathComponent("thumbnails", isDirectory: true)
            .appendingPathComponent(contentHash)
            .appendingPathExtension("png")
    }

    nonisolated func prepareLibraryDirectories(for library: AssetLibrary) throws {
        let root = rootURL(for: library)
        for folder in [
            "database",
            "assets",
            "thumbnails",
            "previews",
            "metadata/import-sessions"
        ] {
            try FileManager.default.createDirectory(
                at: root.appendingPathComponent(folder, isDirectory: true),
                withIntermediateDirectories: true
            )
        }
    }

    nonisolated func clearTransientCaches(for library: AssetLibrary) throws {
        let root = rootURL(for: library)
        for folder in ["thumbnails", "previews"] {
            let url = root.appendingPathComponent(folder, isDirectory: true)
            if FileManager.default.fileExists(atPath: url.path) {
                try FileManager.default.removeItem(at: url)
            }
        }

        try prepareLibraryDirectories(for: library)
    }

    nonisolated func createLibraryPackage(at packageURL: URL, name: String) throws -> AssetLibrary {
        let normalizedURL = packageURL.pathExtension == Self.packageExtension
            ? packageURL
            : packageURL.appendingPathExtension(Self.packageExtension)

        guard !FileManager.default.fileExists(atPath: normalizedURL.path) else {
            throw LibraryStorageError.libraryPackageAlreadyExists
        }

        try FileManager.default.createDirectory(at: normalizedURL, withIntermediateDirectories: true)
        let library = AssetLibrary(id: UUID().uuidString, name: name, createdAt: Date(), packageURL: normalizedURL)
        try prepareLibraryDirectories(for: library)
        try writeManifest(LibraryManifest(library: library), in: library)
        return library
    }

    nonisolated func openLibraryPackage(at packageURL: URL) throws -> AssetLibrary {
        guard !isInTrash(packageURL) else {
            throw LibraryStorageError.missingLibraryPackage
        }

        var isDirectory = ObjCBool(false)
        guard FileManager.default.fileExists(atPath: packageURL.path, isDirectory: &isDirectory),
              isDirectory.boolValue else {
            throw LibraryStorageError.missingLibraryPackage
        }

        let manifestURL = packageURL.appendingPathComponent(LibraryManifest.fileName, isDirectory: false)
        guard FileManager.default.fileExists(atPath: manifestURL.path) else {
            throw LibraryStorageError.missingLibraryPackage
        }

        let data = try Data(contentsOf: manifestURL)
        let manifest = try JSONDecoder.momento.decode(LibraryManifest.self, from: data)
        guard manifest.schemaVersion == LibraryManifest.currentSchemaVersion else {
            throw LibraryStorageError.unsupportedSchemaVersion(manifest.schemaVersion)
        }

        return AssetLibrary(
            id: manifest.libraryID,
            name: manifest.displayName,
            createdAt: manifest.createdAt,
            packageURL: packageURL
        )
    }

    nonisolated func renameLibraryPackage(at packageURL: URL, to name: String) throws -> AssetLibrary {
        let library = try openLibraryPackage(at: packageURL)
        let renamedLibrary = AssetLibrary(
            id: library.id,
            name: name,
            createdAt: library.createdAt,
            packageURL: packageURL
        )
        try writeManifest(LibraryManifest(library: renamedLibrary), in: renamedLibrary)
        return renamedLibrary
    }

    nonisolated func deleteLibraryPackage(at packageURL: URL) throws {
        try FileManager.default.removeItem(at: packageURL)
    }

    nonisolated func trashAssetFile(at fileURL: URL) throws {
        guard let trashURL = trashURLs.first else {
            try FileManager.default.trashItem(at: fileURL, resultingItemURL: nil)
            return
        }

        try FileManager.default.createDirectory(at: trashURL, withIntermediateDirectories: true)
        try FileManager.default.moveItem(at: fileURL, to: availableTrashURL(for: fileURL, in: trashURL))
    }

    nonisolated func relativePath(for url: URL, in library: AssetLibrary) throws -> String {
        let packageURL = rootURL(for: library).standardizedFileURL
        let assetURL = url.standardizedFileURL
        let packagePath = packageURL.path.hasSuffix("/") ? packageURL.path : packageURL.path + "/"

        guard assetURL.path.hasPrefix(packagePath) else {
            throw LibraryStorageError.assetOutsideLibrary
        }

        return String(assetURL.path.dropFirst(packagePath.count))
    }

    nonisolated func resolveAssetURL(relativePath: String, in library: AssetLibrary) -> URL {
        rootURL(for: library).appendingPathComponent(relativePath, isDirectory: false)
    }

    nonisolated private func writeManifest(_ manifest: LibraryManifest, in library: AssetLibrary) throws {
        let data = try JSONEncoder.momento.encode(manifest)
        try data.write(to: rootURL(for: library).appendingPathComponent(LibraryManifest.fileName), options: .atomic)
    }

    nonisolated private func availableTrashURL(for itemURL: URL, in trashURL: URL) -> URL {
        let isDirectory = (try? itemURL.resourceValues(forKeys: [.isDirectoryKey]).isDirectory) == true
        let baseName = itemURL.deletingPathExtension().lastPathComponent
        let pathExtension = itemURL.pathExtension
        var candidate = trashURL.appendingPathComponent(itemURL.lastPathComponent, isDirectory: isDirectory)
        var index = 2

        while FileManager.default.fileExists(atPath: candidate.path) {
            candidate = trashURL
                .appendingPathComponent("\(baseName) \(index)", isDirectory: isDirectory)
                .appendingPathExtension(pathExtension)
            index += 1
        }

        return candidate
    }

    nonisolated private func isInTrash(_ url: URL) -> Bool {
        let itemPath = pathWithTrailingSlash(url)
        return trashURLs.contains { trashURL in
            itemPath.hasPrefix(pathWithTrailingSlash(trashURL))
        }
    }

    nonisolated private func pathWithTrailingSlash(_ url: URL) -> String {
        let path = url.standardizedFileURL.resolvingSymlinksInPath().path
        return path.hasSuffix("/") ? path : path + "/"
    }

    nonisolated private static func defaultApplicationSupportRoot() -> URL {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Library/Application Support", isDirectory: true)
        return base.appendingPathComponent("Momento", isDirectory: true)
    }
}

nonisolated enum LibraryStorageError: LocalizedError {
    case assetOutsideLibrary
    case libraryPackageAlreadyExists
    case missingLibraryPackage
    case unsupportedSchemaVersion(Int)

    var errorDescription: String? {
        switch self {
        case .assetOutsideLibrary:
            "Asset storage must stay inside the selected library package."
        case .libraryPackageAlreadyExists:
            "A library already exists at the selected location."
        case .missingLibraryPackage:
            "The selected library no longer exists."
        case .unsupportedSchemaVersion(let version):
            "Unsupported library schema version: \(version)."
        }
    }
}

extension JSONEncoder {
    nonisolated static var momento: JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        return encoder
    }
}

extension JSONDecoder {
    nonisolated static var momento: JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return decoder
    }
}
