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
        // 打开的资源库通常来自用户选择的 sandbox 外部目录。把 security-scoped
        // resource 的生命周期绑定到对象生命周期，可以让 LibraryStore 通过持有
        // libraryAccessScope 明确表达“当前库仍在被访问”。
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
        try saveReferences(Array(references.prefix(10)))
    }

    func updateName(id: RecentLibraryReference.ID, name: String) throws {
        var references = load()
        guard let index = references.firstIndex(where: { $0.id == id }) else {
            return
        }

        references[index].name = name
        try saveReferences(references)
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

    func move(id: RecentLibraryReference.ID, relativeTo targetID: RecentLibraryReference.ID, insertAfterTarget: Bool) throws {
        var references = load()
        guard let sourceIndex = references.firstIndex(where: { $0.id == id }),
              references.contains(where: { $0.id == targetID }),
              id != targetID else {
            return
        }

        let reference = references.remove(at: sourceIndex)
        guard let targetIndex = references.firstIndex(where: { $0.id == targetID }) else {
            return
        }

        let insertionIndex = insertAfterTarget ? targetIndex + 1 : targetIndex
        references.insert(reference, at: min(insertionIndex, references.endIndex))
        try saveReferences(references)
    }

    private func saveReferences(_ references: [RecentLibraryReference]) throws {
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
