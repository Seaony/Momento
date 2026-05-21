import CoreData
import Foundation

nonisolated final class LibraryMetadataStore: @unchecked Sendable {
    private let library: AssetLibrary
    private let storage: LibraryStorage
    private let context: NSManagedObjectContext

    init(library: AssetLibrary, storage: LibraryStorage = LibraryStorage()) throws {
        self.library = library
        self.storage = storage
        // UI 层只消费 AssetItem 值类型，不直接持有 NSManagedObject。这里固定使用
        // background context，并通过同步的 performAndWait 把 Core Data 的队列约束
        // 封装在存储层内部，避免 managed object 泄漏到 SwiftUI 状态树里。
        self.context = try MomentoCoreDataStack(library: library, storage: storage).container.newBackgroundContext()
        self.context.mergePolicy = NSMergePolicy(merge: .mergeByPropertyStoreTrumpMergePolicyType)
    }

    func loadAssets() throws -> [AssetItem] {
        try context.performAndWait {
            let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
            request.predicate = NSPredicate(format: "libraryID == %@", library.id)
            request.sortDescriptors = [NSSortDescriptor(key: "importedAt", ascending: true)]
            request.fetchBatchSize = 200
            return try context.fetch(request).compactMap(asset(from:))
        }
    }

    func existingContentHashes() throws -> Set<String> {
        try context.performAndWait {
            let request = NSFetchRequest<NSDictionary>(entityName: "AssetRecord")
            request.resultType = .dictionaryResultType
            request.propertiesToFetch = ["contentHash"]
            request.predicate = NSPredicate(format: "libraryID == %@", library.id)
            return Set(try context.fetch(request).compactMap { $0["contentHash"] as? String })
        }
    }

    func saveImportedAssets(_ assets: [AssetItem]) throws -> [AssetItem] {
        try context.performAndWait {
            var savedAssets: [AssetItem] = []

            for asset in assets {
                if let existing = try assetRecord(withContentHash: asset.contentHash) {
                    savedAssets.append(existing)
                    continue
                }

                let record = NSManagedObject(entity: entity(), insertInto: context)
                record.setValue(asset.id, forKey: "id")
                record.setValue(asset.libraryID, forKey: "libraryID")
                record.setValue(asset.displayName, forKey: "displayName")
                // 数据库只保存库包内的相对路径。用户移动整个 .momento 包后，
                // 只要 manifest 和 database 仍在同一个包里，资源路径仍可重新解析。
                record.setValue(try storage.relativePath(for: asset.storageURL, in: library), forKey: "storageRelativePath")
                record.setValue(asset.kind.rawValue, forKey: "kindRaw")
                record.setValue(asset.fileExtension, forKey: "fileExtension")
                record.setValue(asset.byteSize, forKey: "byteSize")
                record.setValue(asset.contentHash, forKey: "contentHash")
                record.setValue(asset.dimensions?.width, forKey: "pixelWidth")
                record.setValue(asset.dimensions?.height, forKey: "pixelHeight")
                record.setValue(asset.isFavorite, forKey: "isFavorite")
                record.setValue(asset.importedAt, forKey: "importedAt")
                savedAssets.append(asset)
            }

            if context.hasChanges {
                try context.save()
            }

            return savedAssets
        }
    }

    private func assetRecord(withContentHash contentHash: String) throws -> AssetItem? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
        request.fetchLimit = 1
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND contentHash == %@",
            library.id,
            contentHash
        )
        return try context.fetch(request).first.flatMap(asset(from:))
    }

    private func asset(from record: NSManagedObject) -> AssetItem? {
        guard let id = record.value(forKey: "id") as? String,
              let libraryID = record.value(forKey: "libraryID") as? String,
              let displayName = record.value(forKey: "displayName") as? String,
              let storageRelativePath = record.value(forKey: "storageRelativePath") as? String,
              let kindRaw = record.value(forKey: "kindRaw") as? String,
              let kind = AssetKind(rawValue: kindRaw),
              let fileExtension = record.value(forKey: "fileExtension") as? String,
              let contentHash = record.value(forKey: "contentHash") as? String,
              let importedAt = record.value(forKey: "importedAt") as? Date else {
            return nil
        }

        let width = record.value(forKey: "pixelWidth") as? Int
        let height = record.value(forKey: "pixelHeight") as? Int
        let dimensions: AssetDimensions?
        if let width, let height {
            dimensions = AssetDimensions(width: width, height: height)
        } else {
            dimensions = nil
        }

        return AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: displayName,
            originalURL: nil,
            storageURL: storage.resolveAssetURL(relativePath: storageRelativePath, in: library),
            kind: kind,
            fileExtension: fileExtension,
            byteSize: record.value(forKey: "byteSize") as? Int64 ?? 0,
            contentHash: contentHash,
            dimensions: dimensions,
            tags: [],
            isFavorite: record.value(forKey: "isFavorite") as? Bool ?? false,
            importedAt: importedAt
        )
    }

    private func entity() -> NSEntityDescription {
        NSEntityDescription.entity(forEntityName: "AssetRecord", in: context)!
    }
}
