import Foundation

struct AssetLibrary: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var name: String
    var createdAt: Date

    static let defaultLibrary = AssetLibrary(
        id: "default",
        name: "Momento Library",
        createdAt: Date(timeIntervalSince1970: 0)
    )
}

struct TagItem: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var name: String
    var colorHex: String?

    init(id: String? = nil, name: String, colorHex: String? = nil) {
        self.id = id ?? name.lowercased()
        self.name = name
        self.colorHex = colorHex
    }
}

struct AssetDimensions: Hashable, Codable, Sendable {
    var width: Int
    var height: Int
}

enum AssetKind: String, CaseIterable, Hashable, Codable, Sendable {
    case image
    case gif
    case svg
    case video
    case pdf
}

struct AssetItem: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var libraryID: String
    var displayName: String
    var originalURL: URL
    var storageURL: URL
    var kind: AssetKind
    var fileExtension: String
    var byteSize: Int64
    var contentHash: String
    var dimensions: AssetDimensions?
    var tags: [TagItem]
    var isFavorite: Bool
    var importedAt: Date
}

enum AssetViewMode: String, CaseIterable, Identifiable, Hashable, Codable, Sendable {
    case masonry
    case grid
    case list

    var id: String { rawValue }
}

enum SidebarSelection: Hashable, Codable, Sendable {
    case library(String)
    case favorites
    case tag(String)
    case trash
}
