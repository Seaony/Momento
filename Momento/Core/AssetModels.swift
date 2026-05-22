import Foundation
import UniformTypeIdentifiers

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

nonisolated struct TagSummary: Identifiable, Hashable, Sendable {
    var tag: TagItem
    var assetCount: Int

    var id: TagItem.ID {
        tag.id
    }
}

nonisolated struct AssetDimensions: Hashable, Codable, Sendable {
    var width: Int
    var height: Int
}

nonisolated struct AssetExifMetadata: Hashable, Codable, Sendable {
    var fileCreatedAt: Date?
    var fileModifiedAt: Date?
    var contentCreatedAt: Date?
    var pixelWidth: Int?
    var pixelHeight: Int?
    var dpiWidth: Double?
    var dpiHeight: Double?
    var colorModel: String?
    var profileName: String?
    var cameraMake: String?
    var cameraModel: String?
    var lensModel: String?
    var exposureTime: Double?
    var focalLength: Double?
    var isoSpeedRatings: [Int]
    var flashFired: Bool?
    var fNumber: Double?
    var exposureProgram: Int?
    var meteringMode: Int?
    var whiteBalance: Int?
    var creator: String?

    init(
        fileCreatedAt: Date? = nil,
        fileModifiedAt: Date? = nil,
        contentCreatedAt: Date? = nil,
        pixelWidth: Int? = nil,
        pixelHeight: Int? = nil,
        dpiWidth: Double? = nil,
        dpiHeight: Double? = nil,
        colorModel: String? = nil,
        profileName: String? = nil,
        cameraMake: String? = nil,
        cameraModel: String? = nil,
        lensModel: String? = nil,
        exposureTime: Double? = nil,
        focalLength: Double? = nil,
        isoSpeedRatings: [Int] = [],
        flashFired: Bool? = nil,
        fNumber: Double? = nil,
        exposureProgram: Int? = nil,
        meteringMode: Int? = nil,
        whiteBalance: Int? = nil,
        creator: String? = nil
    ) {
        self.fileCreatedAt = fileCreatedAt
        self.fileModifiedAt = fileModifiedAt
        self.contentCreatedAt = contentCreatedAt
        self.pixelWidth = pixelWidth
        self.pixelHeight = pixelHeight
        self.dpiWidth = dpiWidth
        self.dpiHeight = dpiHeight
        self.colorModel = colorModel
        self.profileName = profileName
        self.cameraMake = cameraMake
        self.cameraModel = cameraModel
        self.lensModel = lensModel
        self.exposureTime = exposureTime
        self.focalLength = focalLength
        self.isoSpeedRatings = isoSpeedRatings
        self.flashFired = flashFired
        self.fNumber = fNumber
        self.exposureProgram = exposureProgram
        self.meteringMode = meteringMode
        self.whiteBalance = whiteBalance
        self.creator = creator
    }

    var isEmpty: Bool {
        fileCreatedAt == nil
            && fileModifiedAt == nil
            && contentCreatedAt == nil
            && pixelWidth == nil
            && pixelHeight == nil
            && dpiWidth == nil
            && dpiHeight == nil
            && colorModel == nil
            && profileName == nil
            && cameraMake == nil
            && cameraModel == nil
            && lensModel == nil
            && exposureTime == nil
            && focalLength == nil
            && isoSpeedRatings.isEmpty
            && flashFired == nil
            && fNumber == nil
            && exposureProgram == nil
            && meteringMode == nil
            && whiteBalance == nil
            && creator == nil
    }
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
    var originalFileName: String
    var originalURL: URL?
    var storageURL: URL
    var kind: AssetKind
    var fileExtension: String
    var utiIdentifier: String
    var byteSize: Int64
    var contentHash: String
    var dimensions: AssetDimensions?
    var exifMetadata: AssetExifMetadata?
    var orientation: Int?
    var colorProfileName: String?
    var note: String?
    var tags: [TagItem]
    var folderIDs: [String]
    var paletteColors: [AssetColor]
    var thumbnailURL: URL?
    var isFavorite: Bool
    var isTrashed: Bool
    var trashedAt: Date?
    var importedAt: Date
    var updatedAt: Date

    init(
        id: String,
        libraryID: String,
        displayName: String,
        originalFileName: String? = nil,
        originalURL: URL?,
        storageURL: URL,
        kind: AssetKind,
        fileExtension: String,
        utiIdentifier: String? = nil,
        byteSize: Int64,
        contentHash: String,
        dimensions: AssetDimensions?,
        exifMetadata: AssetExifMetadata? = nil,
        orientation: Int? = nil,
        colorProfileName: String? = nil,
        note: String? = nil,
        tags: [TagItem],
        folderIDs: [String] = [],
        paletteColors: [AssetColor] = [],
        thumbnailURL: URL? = nil,
        isFavorite: Bool,
        isTrashed: Bool = false,
        trashedAt: Date? = nil,
        importedAt: Date,
        updatedAt: Date? = nil
    ) {
        self.id = id
        self.libraryID = libraryID
        self.displayName = displayName
        self.originalFileName = originalFileName
            ?? originalURL?.lastPathComponent
            ?? storageURL.lastPathComponent
        self.originalURL = originalURL
        self.storageURL = storageURL
        self.kind = kind
        self.fileExtension = fileExtension
        self.utiIdentifier = utiIdentifier
            ?? UTType(filenameExtension: fileExtension)?.identifier
            ?? "public.data"
        self.byteSize = byteSize
        self.contentHash = contentHash
        self.dimensions = dimensions
        self.exifMetadata = exifMetadata
        self.orientation = orientation
        self.colorProfileName = colorProfileName
        self.note = note
        self.tags = tags
        self.folderIDs = folderIDs
        self.paletteColors = paletteColors
        self.thumbnailURL = thumbnailURL
        self.isFavorite = isFavorite
        self.isTrashed = isTrashed
        self.trashedAt = trashedAt
        self.importedAt = importedAt
        self.updatedAt = updatedAt ?? importedAt
    }
}

nonisolated enum AssetColorCategory: String, CaseIterable, Identifiable, Hashable, Codable, Sendable {
    case black
    case white
    case gray
    case red
    case rose
    case pink
    case magenta
    case purple
    case violet
    case indigo
    case blue
    case sky
    case cyan
    case teal
    case mint
    case green
    case lime
    case olive
    case yellow
    case amber
    case orange
    case coral
    case brown
    case beige

    var id: String { rawValue }
}

nonisolated struct AssetFilterState: Hashable, Codable, Sendable {
    var colorCategories: Set<AssetColorCategory> = []
    var tagIDs: Set<TagItem.ID> = []
    var fileExtensions: Set<String> = []

    var isActive: Bool {
        !colorCategories.isEmpty || !tagIDs.isEmpty || !fileExtensions.isEmpty
    }
}

nonisolated enum AssetSortOption: String, CaseIterable, Identifiable, Hashable, Codable, Sendable {
    case addedTime
    case name
    case fileSize

    var id: String { rawValue }
}

nonisolated enum AssetSortDirection: String, CaseIterable, Identifiable, Hashable, Codable, Sendable {
    case ascending
    case descending

    var id: String { rawValue }
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
