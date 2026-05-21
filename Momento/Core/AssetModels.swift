import Foundation

nonisolated struct AssetLibrary: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var name: String
    var createdAt: Date
    var packageURL: URL?

    init(id: String, name: String, createdAt: Date, packageURL: URL? = nil) {
        self.id = id
        self.name = name
        self.createdAt = createdAt
        self.packageURL = packageURL
    }

    static let defaultLibrary = AssetLibrary(
        id: "default",
        name: "Momento Library",
        createdAt: Date(timeIntervalSince1970: 0),
        packageURL: nil
    )
}

nonisolated struct TagItem: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var name: String
    var colorHex: String?

    init(id: String? = nil, name: String, colorHex: String? = nil) {
        self.id = id ?? name.lowercased()
        self.name = name
        self.colorHex = colorHex
    }
}

nonisolated struct AssetDimensions: Hashable, Codable, Sendable {
    var width: Int
    var height: Int
}

nonisolated struct AssetFolder: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var libraryID: String
    var name: String
    var parentID: String?
    var sortIndex: Int
    var createdAt: Date
    var updatedAt: Date

    init(
        id: String = UUID().uuidString,
        libraryID: String,
        name: String,
        parentID: String? = nil,
        sortIndex: Int,
        createdAt: Date = Date(),
        updatedAt: Date = Date()
    ) {
        self.id = id
        self.libraryID = libraryID
        self.name = name
        self.parentID = parentID
        self.sortIndex = sortIndex
        self.createdAt = createdAt
        self.updatedAt = updatedAt
    }
}

nonisolated struct AssetColor: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var libraryID: String
    var assetID: String
    var hex: String
    var coverage: Double
    var sortIndex: Int

    init(
        id: String = UUID().uuidString,
        libraryID: String,
        assetID: String,
        hex: String,
        coverage: Double,
        sortIndex: Int
    ) {
        self.id = id
        self.libraryID = libraryID
        self.assetID = assetID
        self.hex = hex
        self.coverage = coverage
        self.sortIndex = sortIndex
    }
}

nonisolated enum AssetKind: String, CaseIterable, Hashable, Codable, Sendable {
    case image
    case gif
    case svg
    case video
    case pdf
}

nonisolated struct AssetItem: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var libraryID: String
    var displayName: String
    var originalURL: URL?
    var storageURL: URL
    var kind: AssetKind
    var fileExtension: String
    var byteSize: Int64
    var contentHash: String
    var dimensions: AssetDimensions?
    var tags: [TagItem]
    var folderIDs: [String]
    var paletteColors: [AssetColor]
    var thumbnailURL: URL?
    var isFavorite: Bool
    var importedAt: Date

    init(
        id: String,
        libraryID: String,
        displayName: String,
        originalURL: URL?,
        storageURL: URL,
        kind: AssetKind,
        fileExtension: String,
        byteSize: Int64,
        contentHash: String,
        dimensions: AssetDimensions?,
        tags: [TagItem],
        folderIDs: [String] = [],
        paletteColors: [AssetColor] = [],
        thumbnailURL: URL? = nil,
        isFavorite: Bool,
        importedAt: Date
    ) {
        self.id = id
        self.libraryID = libraryID
        self.displayName = displayName
        self.originalURL = originalURL
        self.storageURL = storageURL
        self.kind = kind
        self.fileExtension = fileExtension
        self.byteSize = byteSize
        self.contentHash = contentHash
        self.dimensions = dimensions
        self.tags = tags
        self.folderIDs = folderIDs
        self.paletteColors = paletteColors
        self.thumbnailURL = thumbnailURL
        self.isFavorite = isFavorite
        self.importedAt = importedAt
    }
}

nonisolated enum AssetViewMode: String, CaseIterable, Identifiable, Hashable, Codable, Sendable {
    case masonry
    case grid
    case list

    var id: String { rawValue }
}

nonisolated enum SidebarSelection: Hashable, Codable, Sendable {
    case library(String)
    case favorites
    case uncategorized
    case untagged
    case tagManagement
    case folderManagement
    case folder(String)
    case tag(String)
    case trash
}
