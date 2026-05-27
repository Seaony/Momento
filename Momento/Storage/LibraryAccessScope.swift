// 中文注释：本文件封装资源库 security-scoped bookmark 的解析和访问生命周期。
import Foundation

enum LibraryStorageMode: String, Codable, Sendable {
    case local
    case cloud
}

struct RecentLibraryReference: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var name: String
    var storageMode: LibraryStorageMode
    var localPackageBookmarkData: Data?
    var cloudLibraryID: String?
    var lastOpenedAt: Date?
    var lastKnownSyncState: String?

    var bookmarkData: Data? {
        localPackageBookmarkData
    }

    var isValidStorageDescriptor: Bool {
        switch storageMode {
        case .local:
            localPackageBookmarkData?.isEmpty == false
        case .cloud:
            cloudLibraryID?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false
        }
    }

    init(id: String, name: String, bookmarkData: Data) {
        self.init(
            id: id,
            name: name,
            storageMode: .local,
            localPackageBookmarkData: bookmarkData,
            cloudLibraryID: nil,
            lastOpenedAt: nil,
            lastKnownSyncState: nil
        )
    }

    init(
        id: String,
        name: String,
        storageMode: LibraryStorageMode,
        localPackageBookmarkData: Data?,
        cloudLibraryID: String?,
        lastOpenedAt: Date?,
        lastKnownSyncState: String?
    ) {
        self.id = id
        self.name = name
        self.storageMode = storageMode
        self.localPackageBookmarkData = localPackageBookmarkData
        self.cloudLibraryID = cloudLibraryID
        self.lastOpenedAt = lastOpenedAt
        self.lastKnownSyncState = lastKnownSyncState
    }

    private enum CodingKeys: String, CodingKey {
        case id
        case name
        case storageMode
        case localPackageBookmarkData
        case cloudLibraryID
        case lastOpenedAt
        case lastKnownSyncState
        case bookmarkData
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        storageMode = try container.decodeIfPresent(LibraryStorageMode.self, forKey: .storageMode) ?? .local
        localPackageBookmarkData = try container.decodeIfPresent(Data.self, forKey: .localPackageBookmarkData)
            ?? container.decodeIfPresent(Data.self, forKey: .bookmarkData)
        cloudLibraryID = try container.decodeIfPresent(String.self, forKey: .cloudLibraryID)
        lastOpenedAt = try container.decodeIfPresent(Date.self, forKey: .lastOpenedAt)
        lastKnownSyncState = try container.decodeIfPresent(String.self, forKey: .lastKnownSyncState)
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(name, forKey: .name)
        try container.encode(storageMode, forKey: .storageMode)
        try container.encodeIfPresent(localPackageBookmarkData, forKey: .localPackageBookmarkData)
        try container.encodeIfPresent(cloudLibraryID, forKey: .cloudLibraryID)
        try container.encodeIfPresent(lastOpenedAt, forKey: .lastOpenedAt)
        try container.encodeIfPresent(lastKnownSyncState, forKey: .lastKnownSyncState)
        if storageMode == .local {
            try container.encodeIfPresent(localPackageBookmarkData, forKey: .bookmarkData)
        }
    }
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

    func load(includeLocalLibraries: Bool = true) -> [RecentLibraryReference] {
        guard let data = defaults.data(forKey: key),
              let references = try? JSONDecoder.momento.decode([RecentLibraryReference].self, from: data) else {
            return []
        }
        let validReferences = references.filter(\.isValidStorageDescriptor)
        guard !includeLocalLibraries else {
            return validReferences
        }
        return validReferences.filter { $0.storageMode != .local }
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
        let reference = RecentLibraryReference(
            id: library.id,
            name: library.name,
            storageMode: .local,
            localPackageBookmarkData: bookmarkData,
            cloudLibraryID: nil,
            lastOpenedAt: Date(),
            lastKnownSyncState: nil
        )
        var references = load().filter { $0.id != library.id }
        references.insert(reference, at: 0)
        try saveReferences(Array(references.prefix(10)))
    }

    func saveCloudPlaceholder(
        id: String,
        name: String,
        cloudLibraryID: String,
        lastKnownSyncState: String? = nil
    ) throws {
        let reference = RecentLibraryReference(
            id: id,
            name: name,
            storageMode: .cloud,
            localPackageBookmarkData: nil,
            cloudLibraryID: cloudLibraryID,
            lastOpenedAt: nil,
            lastKnownSyncState: lastKnownSyncState
        )
        var references = load().filter { $0.id != id }
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
              let targetIndexBeforeMove = references.firstIndex(where: { $0.id == targetID }),
              references[sourceIndex].storageMode == references[targetIndexBeforeMove].storageMode,
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
        guard reference.storageMode == .local, let bookmarkData = reference.localPackageBookmarkData else {
            throw CocoaError(.fileReadUnsupportedScheme)
        }

        var isStale = false
        let url = try URL(
            resolvingBookmarkData: bookmarkData,
            options: [.withSecurityScope],
            relativeTo: nil,
            bookmarkDataIsStale: &isStale
        )
        return (url, isStale)
    }
}
