// 中文注释：manifest 是资源库包的轻量元信息入口，用来识别库 ID、名称和 schema 版本。
import Foundation

nonisolated struct LibraryManifest: Codable, Equatable, Sendable {
    static let fileName = "manifest.json"
    static let currentSchemaVersion = 1

    var schemaVersion: Int
    var libraryID: String
    var displayName: String
    var createdAt: Date
    var updatedAt: Date

    init(library: AssetLibrary) {
        self.schemaVersion = Self.currentSchemaVersion
        self.libraryID = library.id
        self.displayName = library.name
        self.createdAt = library.createdAt
        self.updatedAt = library.createdAt
    }
}
