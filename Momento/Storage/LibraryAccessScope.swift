import Foundation

struct RecentLibraryReference: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var name: String
    var bookmarkData: Data
}

final class LibraryAccessScope {
    let url: URL
    private let didStartAccessing: Bool

    init(url: URL) {
        self.url = url
        self.didStartAccessing = url.startAccessingSecurityScopedResource()
    }

    deinit {
        if didStartAccessing {
            url.stopAccessingSecurityScopedResource()
        }
    }
}

struct RecentLibraryStore {
    private let defaults: UserDefaults
    private let key = "recentLibraries"

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    func load() -> [RecentLibraryReference] {
        guard let data = defaults.data(forKey: key),
              let references = try? JSONDecoder.momento.decode([RecentLibraryReference].self, from: data) else {
            return []
        }
        return references
    }

    func save(_ library: AssetLibrary) throws {
        guard let packageURL = library.packageURL else {
            return
        }

        let bookmarkData = try packageURL.bookmarkData(
            options: [.withSecurityScope],
            includingResourceValuesForKeys: nil,
            relativeTo: nil
        )
        let reference = RecentLibraryReference(id: library.id, name: library.name, bookmarkData: bookmarkData)
        var references = load().filter { $0.id != library.id }
        references.insert(reference, at: 0)
        let data = try JSONEncoder.momento.encode(Array(references.prefix(10)))
        defaults.set(data, forKey: key)
    }

    func remove(id: RecentLibraryReference.ID) throws {
        let references = load().filter { $0.id != id }
        guard !references.isEmpty else {
            defaults.removeObject(forKey: key)
            return
        }

        let data = try JSONEncoder.momento.encode(references)
        defaults.set(data, forKey: key)
    }

    func resolve(_ reference: RecentLibraryReference) throws -> (url: URL, isStale: Bool) {
        var isStale = false
        let url = try URL(
            resolvingBookmarkData: reference.bookmarkData,
            options: [.withSecurityScope],
            relativeTo: nil,
            bookmarkDataIsStale: &isStale
        )
        return (url, isStale)
    }
}
