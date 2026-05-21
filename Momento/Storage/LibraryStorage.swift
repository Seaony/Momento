import Foundation

struct LibraryStorage: Sendable {
    var applicationSupportRoot: URL

    init(applicationSupportRoot: URL? = nil) {
        self.applicationSupportRoot = applicationSupportRoot ?? Self.defaultApplicationSupportRoot()
    }

    nonisolated func rootURL(for library: AssetLibrary) -> URL {
        applicationSupportRoot
            .appendingPathComponent("Libraries", isDirectory: true)
            .appendingPathComponent(library.id, isDirectory: true)
            .appendingPathComponent(".library", isDirectory: true)
    }

    nonisolated func assetsURL(for library: AssetLibrary) -> URL {
        rootURL(for: library).appendingPathComponent("assets", isDirectory: true)
    }

    nonisolated func prepareLibraryDirectories(for library: AssetLibrary) throws {
        let root = rootURL(for: library)
        for folder in ["database", "assets", "thumbnails", "metadata"] {
            try FileManager.default.createDirectory(
                at: root.appendingPathComponent(folder, isDirectory: true),
                withIntermediateDirectories: true
            )
        }
    }

    nonisolated private static func defaultApplicationSupportRoot() -> URL {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Library/Application Support", isDirectory: true)
        return base.appendingPathComponent("Momento", isDirectory: true)
    }
}
