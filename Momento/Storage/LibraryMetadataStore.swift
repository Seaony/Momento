// 中文注释：本文件封装单个资源库的 Core Data 读写，并只向 UI 层暴露值类型模型。
import CoreData
import Foundation

nonisolated struct DuplicateAssetReference: Hashable {
    var assetID: AssetItem.ID
    var isTrashed: Bool
}

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
        try migrateLegacyTagsIfNeeded()
    }

    func loadAssets() throws -> [AssetItem] {
        try context.performAndWait {
            let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
            request.predicate = NSPredicate(format: "libraryID == %@", library.id)
            request.sortDescriptors = [NSSortDescriptor(key: "importedAt", ascending: true)]
            request.fetchBatchSize = 200

            let records = try context.fetch(request)
            let assetIDs = records.compactMap { $0.value(forKey: "id") as? String }
            let folderIDsByAssetID = try folderIDsByAssetID(for: Set(assetIDs))
            let colorsByAssetID = try colorsByAssetID(for: Set(assetIDs))
            let tagsByAssetID = try tagsByAssetID(for: Set(assetIDs))

            return records.compactMap { record in
                guard let id = record.value(forKey: "id") as? String else {
                    return nil
                }
                return asset(
                    from: record,
                    folderIDs: folderIDsByAssetID[id, default: []],
                    paletteColors: colorsByAssetID[id, default: []],
                    tags: tagsByAssetID[id, default: []]
                )
            }
        }
    }

    func loadTags() throws -> [TagItem] {
        try context.performAndWait {
            try loadTagRecords().compactMap(tag(from:)).sorted {
                $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending
            }
        }
    }

    func loadFolders() throws -> [AssetFolder] {
        try context.performAndWait {
            let request = NSFetchRequest<NSManagedObject>(entityName: "FolderRecord")
            request.predicate = NSPredicate(format: "libraryID == %@", library.id)
            request.sortDescriptors = [
                NSSortDescriptor(key: "sortIndex", ascending: true),
                NSSortDescriptor(key: "createdAt", ascending: true),
                NSSortDescriptor(key: "id", ascending: true)
            ]
            return try context.fetch(request).compactMap(folder(from:))
        }
    }

    func existingContentHashes(includeTrashed: Bool = false) throws -> Set<String> {
        try context.performAndWait {
            let request = NSFetchRequest<NSDictionary>(entityName: "AssetRecord")
            request.resultType = .dictionaryResultType
            request.propertiesToFetch = ["contentHash"]
            request.predicate = includeTrashed
                ? NSPredicate(format: "libraryID == %@", library.id)
                : NSPredicate(format: "libraryID == %@ AND isTrashed == NO", library.id)
            return Set(try context.fetch(request).compactMap { $0["contentHash"] as? String })
        }
    }

    func saveImportedAssets(_ assets: [AssetItem]) throws -> [AssetItem] {
        try saveImportedBatch(
            AssetImportBatch(
                newAssets: assets,
                folderAssignmentsByContentHash: [:]
            )
        )
    }

    func saveImportedBatch(_ batch: AssetImportBatch) throws -> [AssetItem] {
        try context.performAndWait {
            var affectedAssetIDs = Set<AssetItem.ID>()

            for importedAsset in batch.newAssets {
                if let existingRecord = try assetRecord(contentHash: importedAsset.contentHash) {
                    var didUpdateExistingRecord = false
                    let now = Date()

                    if (existingRecord.value(forKey: "isTrashed") as? Bool) == true {
                        existingRecord.setValue(false, forKey: "isTrashed")
                        existingRecord.setValue(nil, forKey: "trashedAt")
                        didUpdateExistingRecord = true
                    }
                    // 中文注释：内容哈希命中已有素材时不覆盖用户编辑过的标题、标签等信息；
                    // 只在旧记录还没有来源页面时补写浏览器导入传来的上下文链接。
                    if storedURL(existingRecord.value(forKey: "sourcePageURL")) == nil,
                       let sourcePageURL = importedAsset.sourcePageURL {
                        existingRecord.setValue(sourcePageURL.absoluteString, forKey: "sourcePageURL")
                        didUpdateExistingRecord = true
                    }
                    if didUpdateExistingRecord {
                        existingRecord.setValue(now, forKey: "updatedAt")
                    }
                    if let existingID = existingRecord.value(forKey: "id") as? String {
                        affectedAssetIDs.insert(existingID)
                    }
                    continue
                }

                let record = NSManagedObject(entity: entity(named: "AssetRecord"), insertInto: context)
                record.setValue(importedAsset.id, forKey: "id")
                record.setValue(importedAsset.libraryID, forKey: "libraryID")
                record.setValue(importedAsset.displayName, forKey: "displayName")
                record.setValue(importedAsset.originalFileName, forKey: "originalFileName")
                record.setValue(importedAsset.sourcePageURL?.absoluteString, forKey: "sourcePageURL")
                // 数据库只保存库包内的相对路径。用户移动整个 .momento 包后，
                // 只要 manifest 和 database 仍在同一个包里，资源路径仍可重新解析。
                record.setValue(try storage.relativePath(for: importedAsset.storageURL, in: library), forKey: "storageRelativePath")
                record.setValue(importedAsset.kind.rawValue, forKey: "kindRaw")
                record.setValue(importedAsset.fileExtension, forKey: "fileExtension")
                record.setValue(importedAsset.utiIdentifier, forKey: "utiIdentifier")
                record.setValue(importedAsset.byteSize, forKey: "byteSize")
                record.setValue(importedAsset.contentHash, forKey: "contentHash")
                record.setValue(importedAsset.dimensions?.width, forKey: "pixelWidth")
                record.setValue(importedAsset.dimensions?.height, forKey: "pixelHeight")
                record.setValue(importedAsset.orientation, forKey: "orientation")
                record.setValue(importedAsset.colorProfileName, forKey: "colorProfileName")
                let storedExifMetadataData = exifMetadataData(importedAsset.exifMetadata)
                record.setValue(storedExifMetadataData, forKey: "exifMetadataData")
                record.setValue(importedAsset.note, forKey: "note")
                record.setValue(importedAsset.isFavorite, forKey: "isFavorite")
                record.setValue(importedAsset.isTrashed, forKey: "isTrashed")
                record.setValue(importedAsset.trashedAt, forKey: "trashedAt")
                record.setValue(importedAsset.importedAt, forKey: "importedAt")
                record.setValue(importedAsset.updatedAt, forKey: "updatedAt")
                record.setValue(nil, forKey: "tagsData")
                saveColors(importedAsset.paletteColors, forAssetID: importedAsset.id)

                affectedAssetIDs.insert(importedAsset.id)
            }

            try saveFolderAssignments(
                batch.folderAssignmentsByContentHash,
                affectedAssetIDs: &affectedAssetIDs
            )

            if context.hasChanges {
                try context.save()
            }

            return try assets(ids: affectedAssetIDs)
        }
    }

    func createFolder(name: String, parentID: AssetFolder.ID?) throws -> AssetFolder {
        try context.performAndWait {
            let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmedName.isEmpty else {
                throw LibraryMetadataError.invalidFolderName
            }

            if let parentID, try folderRecord(id: parentID) == nil {
                throw LibraryMetadataError.missingFolder
            }

            let now = Date()
            let folder = AssetFolder(
                libraryID: library.id,
                name: trimmedName,
                parentID: parentID,
                sortIndex: try nextFolderSortIndex(parentID: parentID),
                createdAt: now,
                updatedAt: now
            )

            let record = NSManagedObject(entity: entity(named: "FolderRecord"), insertInto: context)
            record.setValue(folder.id, forKey: "id")
            record.setValue(folder.libraryID, forKey: "libraryID")
            record.setValue(folder.name, forKey: "name")
            record.setValue(folder.parentID, forKey: "parentID")
            record.setValue(folder.sortIndex, forKey: "sortIndex")
            record.setValue(folder.createdAt, forKey: "createdAt")
            record.setValue(folder.updatedAt, forKey: "updatedAt")

            try context.save()
            return folder
        }
    }

    func renameFolder(id: AssetFolder.ID, to name: String) throws -> AssetFolder {
        try context.performAndWait {
            let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmedName.isEmpty else {
                throw LibraryMetadataError.invalidFolderName
            }

            guard let record = try folderRecord(id: id) else {
                throw LibraryMetadataError.missingFolder
            }

            let updatedAt = Date()
            record.setValue(trimmedName, forKey: "name")
            record.setValue(updatedAt, forKey: "updatedAt")

            try context.save()
            guard let folder = folder(from: record) else {
                throw LibraryMetadataError.missingFolder
            }
            return folder
        }
    }

    func moveFolder(
        id: AssetFolder.ID,
        toParentID parentID: AssetFolder.ID?,
        relativeTo targetID: AssetFolder.ID?,
        insertAfterTarget: Bool
    ) throws -> AssetFolder {
        try context.performAndWait {
            guard let movingRecord = try folderRecord(id: id) else {
                throw LibraryMetadataError.missingFolder
            }

            if parentID == id {
                throw LibraryMetadataError.invalidFolderMove
            }

            if let parentID, try folderRecord(id: parentID) == nil {
                throw LibraryMetadataError.missingFolder
            }

            let descendantIDs = try descendantFolderIDs(startingAt: id)
            if let parentID, descendantIDs.contains(parentID) {
                throw LibraryMetadataError.invalidFolderMove
            }
            if let targetID, descendantIDs.contains(targetID) {
                throw LibraryMetadataError.invalidFolderMove
            }

            if let targetID {
                guard let targetRecord = try folderRecord(id: targetID) else {
                    throw LibraryMetadataError.missingFolder
                }
                guard (targetRecord.value(forKey: "parentID") as? String) == parentID else {
                    throw LibraryMetadataError.invalidFolderMove
                }
            }

            let previousParentID = movingRecord.value(forKey: "parentID") as? String
            let updatedAt = Date()

            if previousParentID != parentID {
                movingRecord.setValue(parentID, forKey: "parentID")
                movingRecord.setValue(updatedAt, forKey: "updatedAt")
                try normalizeFolderSortIndexes(parentID: previousParentID, updatedAt: updatedAt)
            }

            try normalizeFolderSortIndexes(
                parentID: parentID,
                movingRecord: movingRecord,
                relativeTo: targetID,
                insertAfterTarget: insertAfterTarget,
                updatedAt: updatedAt
            )

            if context.hasChanges {
                try context.save()
            }

            guard let folder = folder(from: movingRecord) else {
                throw LibraryMetadataError.missingFolder
            }
            return folder
        }
    }

    func deleteFolder(id: AssetFolder.ID) throws -> [AssetFolder.ID] {
        try context.performAndWait {
            guard try folderRecord(id: id) != nil else {
                throw LibraryMetadataError.missingFolder
            }

            let deletedIDs = try descendantFolderIDs(startingAt: id)
            for record in try folderRecords(ids: deletedIDs) {
                context.delete(record)
            }
            for record in try membershipRecords(folderIDs: deletedIDs) {
                context.delete(record)
            }

            if context.hasChanges {
                try context.save()
            }
            return Array(deletedIDs)
        }
    }

    func assignAssets(ids: Set<AssetItem.ID>, to folderID: AssetFolder.ID) throws -> [AssetItem] {
        try context.performAndWait {
            guard try folderRecord(id: folderID) != nil else {
                throw LibraryMetadataError.missingFolder
            }

            let validAssetIDs = try existingAssetIDs(in: ids, includeTrashed: false)
            let existing = try existingMembershipAssetIDs(folderID: folderID, assetIDs: validAssetIDs)
            let now = Date()

            for assetID in validAssetIDs.subtracting(existing) {
                let record = NSManagedObject(entity: entity(named: "AssetFolderMembershipRecord"), insertInto: context)
                record.setValue("\(assetID)-\(folderID)", forKey: "id")
                record.setValue(library.id, forKey: "libraryID")
                record.setValue(assetID, forKey: "assetID")
                record.setValue(folderID, forKey: "folderID")
                record.setValue(now, forKey: "createdAt")
            }

            if context.hasChanges {
                try context.save()
            }
            return try assets(ids: validAssetIDs)
        }
    }

    func unassignAssets(ids: Set<AssetItem.ID>, from folderID: AssetFolder.ID) throws -> [AssetItem] {
        try context.performAndWait {
            guard try folderRecord(id: folderID) != nil else {
                throw LibraryMetadataError.missingFolder
            }

            let validAssetIDs = try existingAssetIDs(in: ids)
            for record in try membershipRecords(folderID: folderID, assetIDs: validAssetIDs) {
                context.delete(record)
            }

            if context.hasChanges {
                try context.save()
            }
            return try assets(ids: validAssetIDs)
        }
    }

    func setFavorite(_ isFavorite: Bool, forAssetID assetID: AssetItem.ID) throws -> AssetItem {
        try context.performAndWait {
            guard let record = try assetRecord(id: assetID) else {
                throw LibraryMetadataError.missingAsset
            }

            if (record.value(forKey: "isFavorite") as? Bool) != isFavorite {
                record.setValue(isFavorite, forKey: "isFavorite")
                record.setValue(Date(), forKey: "updatedAt")
            }
            if context.hasChanges {
                try context.save()
            }

            guard let asset = try assets(ids: [assetID]).first else {
                throw LibraryMetadataError.missingAsset
            }
            return asset
        }
    }

    func renameAsset(id assetID: AssetItem.ID, to displayName: String) throws -> AssetItem {
        try context.performAndWait {
            let trimmedName = displayName.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmedName.isEmpty else {
                throw LibraryMetadataError.invalidAssetName
            }

            guard let record = try assetRecord(id: assetID) else {
                throw LibraryMetadataError.missingAsset
            }

            if (record.value(forKey: "displayName") as? String) != trimmedName {
                record.setValue(trimmedName, forKey: "displayName")
                record.setValue(Date(), forKey: "updatedAt")
            }
            if context.hasChanges {
                try context.save()
            }

            guard let asset = try assets(ids: [assetID]).first else {
                throw LibraryMetadataError.missingAsset
            }
            return asset
        }
    }

    func updateNote(_ note: String?, forAssetID assetID: AssetItem.ID) throws -> AssetItem {
        try context.performAndWait {
            guard let record = try assetRecord(id: assetID) else {
                throw LibraryMetadataError.missingAsset
            }

            let storedNote = note.flatMap { value in
                value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : value
            }

            if (record.value(forKey: "note") as? String) != storedNote {
                record.setValue(storedNote, forKey: "note")
                record.setValue(Date(), forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            guard let asset = try assets(ids: [assetID]).first else {
                throw LibraryMetadataError.missingAsset
            }
            return asset
        }
    }

    func resolveOrCreateTags(named names: [String]) throws -> [TagItem] {
        try context.performAndWait {
            let records = try resolveOrCreateTagRecords(named: names)
            if context.hasChanges {
                try context.save()
            }
            return records.compactMap(tag(from:))
        }
    }

    func addTag(id tagID: TagItem.ID, toAssets assetIDs: Set<AssetItem.ID>) throws -> [AssetItem] {
        try context.performAndWait {
            guard try tagRecord(id: tagID) != nil else {
                throw LibraryMetadataError.missingTag
            }

            let validAssetIDs = try existingAssetIDs(in: assetIDs, includeTrashed: false)
            let existingAssetIDs = Set(
                try tagLinkRecords(tagID: tagID, assetIDs: validAssetIDs).compactMap {
                    $0.value(forKey: "assetID") as? String
                }
            )
            let now = Date()

            for assetID in validAssetIDs.subtracting(existingAssetIDs) {
                let link = NSManagedObject(entity: entity(named: "AssetTagRecord"), insertInto: context)
                link.setValue("\(assetID)-\(tagID)", forKey: "id")
                link.setValue(library.id, forKey: "libraryID")
                link.setValue(assetID, forKey: "assetID")
                link.setValue(tagID, forKey: "tagID")
                link.setValue(now, forKey: "createdAt")
            }

            let linkedAssetIDs = validAssetIDs.subtracting(existingAssetIDs)
            for assetRecord in try assetRecords(ids: linkedAssetIDs) {
                assetRecord.setValue(now, forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            return try assets(ids: validAssetIDs)
        }
    }

    func setTagNames(_ names: [String], forAssetID assetID: AssetItem.ID) throws -> AssetItem {
        try context.performAndWait {
            guard let assetRecord = try assetRecord(id: assetID) else {
                throw LibraryMetadataError.missingAsset
            }

            let tagRecords = try resolveOrCreateTagRecords(named: names)
            let orderedTagIDs = tagRecords.compactMap { $0.value(forKey: "id") as? String }
            let tagIDs = Set(orderedTagIDs)
            let existingLinks = try tagLinkRecords(assetID: assetID)
            let existingOrderedTagIDs = existingLinks.compactMap { $0.value(forKey: "tagID") as? String }
            let existingTagIDs = Set(existingLinks.compactMap { $0.value(forKey: "tagID") as? String })
            let existingLinksByTagID = Dictionary(
                uniqueKeysWithValues: existingLinks.compactMap { link -> (String, NSManagedObject)? in
                    guard let tagID = link.value(forKey: "tagID") as? String else {
                        return nil
                    }
                    return (tagID, link)
                }
            )

            for link in existingLinks {
                guard let tagID = link.value(forKey: "tagID") as? String else {
                    continue
                }
                if !tagIDs.contains(tagID) {
                    context.delete(link)
                }
            }

            let now = Date()
            let shouldUpdateOrder = existingOrderedTagIDs != orderedTagIDs
            for (index, tagID) in orderedTagIDs.enumerated() {
                let link = existingLinksByTagID[tagID]
                    ?? NSManagedObject(entity: entity(named: "AssetTagRecord"), insertInto: context)
                if !existingTagIDs.contains(tagID) {
                    link.setValue("\(assetID)-\(tagID)", forKey: "id")
                    link.setValue(library.id, forKey: "libraryID")
                    link.setValue(assetID, forKey: "assetID")
                    link.setValue(tagID, forKey: "tagID")
                }

                if shouldUpdateOrder || !existingTagIDs.contains(tagID) {
                    link.setValue(now.addingTimeInterval(TimeInterval(index) / 1_000), forKey: "createdAt")
                }
            }

            if existingOrderedTagIDs != orderedTagIDs {
                assetRecord.setValue(now, forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            guard let asset = try assets(ids: [assetID]).first else {
                throw LibraryMetadataError.missingAsset
            }
            return asset
        }
    }

    func renameTag(id tagID: TagItem.ID, to name: String) throws -> [AssetItem] {
        try context.performAndWait {
            let normalized = normalizedTagName(name)
            guard let normalized else {
                throw LibraryMetadataError.invalidTagName
            }

            guard let record = try tagRecord(id: tagID) else {
                throw LibraryMetadataError.missingTag
            }

            if let existing = try tagRecord(normalizedName: normalized.key),
               (existing.value(forKey: "id") as? String) != tagID {
                throw LibraryMetadataError.duplicateTagName
            }

            let now = Date()
            record.setValue(normalized.displayName, forKey: "name")
            record.setValue(normalized.key, forKey: "normalizedName")
            record.setValue(now, forKey: "updatedAt")

            let linkedAssetIDs = try assetIDs(linkedToTagID: tagID)
            for assetRecord in try assetRecords(ids: linkedAssetIDs) {
                assetRecord.setValue(now, forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            return try assets(ids: linkedAssetIDs)
        }
    }

    func deleteTag(id tagID: TagItem.ID) throws -> [AssetItem] {
        try context.performAndWait {
            guard let tagRecord = try tagRecord(id: tagID) else {
                throw LibraryMetadataError.missingTag
            }

            let linkedAssetIDs = try assetIDs(linkedToTagID: tagID)
            let now = Date()
            for link in try tagLinkRecords(tagID: tagID) {
                context.delete(link)
            }
            for assetRecord in try assetRecords(ids: linkedAssetIDs) {
                assetRecord.setValue(now, forKey: "updatedAt")
            }
            context.delete(tagRecord)

            if context.hasChanges {
                try context.save()
            }

            return try assets(ids: linkedAssetIDs)
        }
    }

    func updateTags(_ tags: [TagItem], forAssetID assetID: AssetItem.ID) throws -> AssetItem {
        try setTagNames(tags.map(\.name), forAssetID: assetID)
    }

    func replaceColors(_ colors: [AssetColor], forAssetID assetID: AssetItem.ID) throws -> AssetItem {
        try context.performAndWait {
            guard try assetRecord(id: assetID) != nil else {
                throw LibraryMetadataError.missingAsset
            }

            for record in try colorRecords(assetID: assetID) {
                context.delete(record)
            }
            saveColors(colors, forAssetID: assetID)

            if context.hasChanges {
                try context.save()
            }

            guard let asset = try assets(ids: [assetID]).first else {
                throw LibraryMetadataError.missingAsset
            }
            return asset
        }
    }

    func deleteAsset(id assetID: AssetItem.ID) throws {
        try context.performAndWait {
            guard let assetRecord = try assetRecord(id: assetID) else {
                throw LibraryMetadataError.missingAsset
            }

            for record in try colorRecords(assetID: assetID) {
                context.delete(record)
            }
            for record in try membershipRecords(assetIDs: [assetID]) {
                context.delete(record)
            }
            for record in try tagLinkRecords(assetID: assetID) {
                context.delete(record)
            }
            context.delete(assetRecord)

            if context.hasChanges {
                try context.save()
            }
        }
    }

    func moveAssetToTrash(id assetID: AssetItem.ID) throws -> AssetItem {
        try context.performAndWait {
            guard let record = try assetRecord(id: assetID) else {
                throw LibraryMetadataError.missingAsset
            }

            let now = Date()
            if (record.value(forKey: "isTrashed") as? Bool) != true {
                record.setValue(true, forKey: "isTrashed")
                record.setValue(now, forKey: "trashedAt")
                record.setValue(now, forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            guard let asset = try asset(from: record) else {
                throw LibraryMetadataError.missingAsset
            }
            return asset
        }
    }

    func moveAssetsToTrash(ids assetIDs: Set<AssetItem.ID>) throws -> [AssetItem] {
        try context.performAndWait {
            let records = try assetRecords(ids: assetIDs)
            guard records.count == assetIDs.count else {
                throw LibraryMetadataError.missingAsset
            }

            let now = Date()
            for record in records where (record.value(forKey: "isTrashed") as? Bool) != true {
                record.setValue(true, forKey: "isTrashed")
                record.setValue(now, forKey: "trashedAt")
                record.setValue(now, forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            return try assets(ids: assetIDs)
        }
    }

    func restoreAssets(ids assetIDs: Set<AssetItem.ID>) throws -> [AssetItem] {
        try context.performAndWait {
            let records = try assetRecords(ids: assetIDs)
            let now = Date()

            for record in records where (record.value(forKey: "isTrashed") as? Bool) == true {
                record.setValue(false, forKey: "isTrashed")
                record.setValue(nil, forKey: "trashedAt")
                record.setValue(now, forKey: "updatedAt")
            }

            if context.hasChanges {
                try context.save()
            }

            return try assets(ids: Set(records.compactMap { $0.value(forKey: "id") as? String }))
        }
    }

    func emptyTrash() throws -> [AssetItem.ID] {
        try context.performAndWait {
            let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
            request.predicate = NSPredicate(format: "libraryID == %@ AND isTrashed == YES", library.id)
            let records = try context.fetch(request)
            let assetIDs = records.compactMap { $0.value(forKey: "id") as? String }

            for assetID in assetIDs {
                for record in try colorRecords(assetID: assetID) {
                    context.delete(record)
                }
                for record in try membershipRecords(assetIDs: [assetID]) {
                    context.delete(record)
                }
                for record in try tagLinkRecords(assetID: assetID) {
                    context.delete(record)
                }
            }
            for record in records {
                context.delete(record)
            }

            if context.hasChanges {
                try context.save()
            }

            return assetIDs
        }
    }

    func duplicateAssetReferences(
        forContentHashes hashes: Set<String>,
        includeTrashed: Bool
    ) throws -> [String: DuplicateAssetReference] {
        try context.performAndWait {
            guard !hashes.isEmpty else {
                return [:]
            }

            let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
            request.predicate = includeTrashed
                ? NSPredicate(format: "libraryID == %@ AND contentHash IN %@", library.id, Array(hashes))
                : NSPredicate(
                    format: "libraryID == %@ AND contentHash IN %@ AND isTrashed == NO",
                    library.id,
                    Array(hashes)
                )

            var references: [String: DuplicateAssetReference] = [:]
            for record in try context.fetch(request) {
                guard let contentHash = record.value(forKey: "contentHash") as? String,
                      let assetID = record.value(forKey: "id") as? String else {
                    continue
                }
                references[contentHash] = DuplicateAssetReference(
                    assetID: assetID,
                    isTrashed: record.value(forKey: "isTrashed") as? Bool ?? false
                )
            }
            return references
        }
    }

    private func assets(ids: Set<AssetItem.ID>) throws -> [AssetItem] {
        guard !ids.isEmpty else {
            return []
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND id IN %@", library.id, Array(ids))
        request.sortDescriptors = [NSSortDescriptor(key: "importedAt", ascending: true)]
        let records = try context.fetch(request)
        let folderIDsByAssetID = try folderIDsByAssetID(for: ids)
        let colorsByAssetID = try colorsByAssetID(for: ids)
        let tagsByAssetID = try tagsByAssetID(for: ids)

        return records.compactMap { record in
            guard let id = record.value(forKey: "id") as? String else {
                return nil
            }
            return asset(
                from: record,
                folderIDs: folderIDsByAssetID[id, default: []],
                paletteColors: colorsByAssetID[id, default: []],
                tags: tagsByAssetID[id, default: []]
            )
        }
    }

    private func asset(withContentHash contentHash: String) throws -> AssetItem? {
        guard let record = try assetRecord(contentHash: contentHash) else {
            return nil
        }

        return try asset(from: record)
    }

    private func asset(from record: NSManagedObject) throws -> AssetItem? {
        guard let id = record.value(forKey: "id") as? String else {
            return nil
        }

        return asset(
            from: record,
            folderIDs: try folderIDsByAssetID(for: [id])[id, default: []],
            paletteColors: try colorsByAssetID(for: [id])[id, default: []],
            tags: try tagsByAssetID(for: [id])[id, default: []]
        )
    }

    private func assetRecord(contentHash: String) throws -> NSManagedObject? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
        request.fetchLimit = 1
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND contentHash == %@",
            library.id,
            contentHash
        )

        return try context.fetch(request).first
    }

    private func assetRecord(id: AssetItem.ID) throws -> NSManagedObject? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
        request.fetchLimit = 1
        request.predicate = NSPredicate(format: "libraryID == %@ AND id == %@", library.id, id)
        return try context.fetch(request).first
    }

    private func migrateLegacyTagsIfNeeded() throws {
        try context.performAndWait {
            let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
            request.predicate = NSPredicate(format: "libraryID == %@", library.id)
            request.sortDescriptors = [
                NSSortDescriptor(key: "importedAt", ascending: true),
                NSSortDescriptor(key: "id", ascending: true)
            ]

            let assetRecords = try context.fetch(request)
            guard !assetRecords.isEmpty else {
                return
            }

            var tagRecordsByNormalizedName = Dictionary(
                uniqueKeysWithValues: try loadTagRecords().compactMap { record -> (String, NSManagedObject)? in
                    guard let normalizedName = record.value(forKey: "normalizedName") as? String else {
                        return nil
                    }
                    return (normalizedName, record)
                }
            )
            var usedTagIDs = Set(
                tagRecordsByNormalizedName.values.compactMap { $0.value(forKey: "id") as? String }
            )
            var existingLinkKeys = Set(
                try allTagLinkRecords().compactMap { record -> String? in
                    guard let assetID = record.value(forKey: "assetID") as? String,
                          let tagID = record.value(forKey: "tagID") as? String else {
                        return nil
                    }
                    return tagLinkKey(assetID: assetID, tagID: tagID)
                }
            )

            for assetRecord in assetRecords {
                guard let assetID = assetRecord.value(forKey: "id") as? String else {
                    continue
                }

                for (legacyIndex, legacyTag) in tags(from: assetRecord.value(forKey: "tagsData")).enumerated() {
                    guard let normalized = normalizedTagName(legacyTag.name) else {
                        continue
                    }

                    let tagRecord: NSManagedObject
                    if let existing = tagRecordsByNormalizedName[normalized.key] {
                        tagRecord = existing
                    } else {
                        let record = NSManagedObject(entity: entity(named: "TagRecord"), insertInto: context)
                        let legacyID = legacyTag.id.trimmingCharacters(in: .whitespacesAndNewlines)
                        let tagID = !legacyID.isEmpty && !usedTagIDs.contains(legacyID)
                            ? legacyID
                            : UUID().uuidString
                        let now = assetRecord.value(forKey: "importedAt") as? Date ?? Date()

                        record.setValue(tagID, forKey: "id")
                        record.setValue(library.id, forKey: "libraryID")
                        record.setValue(normalized.displayName, forKey: "name")
                        record.setValue(normalized.key, forKey: "normalizedName")
                        record.setValue(legacyTag.colorHex, forKey: "colorHex")
                        record.setValue(now, forKey: "createdAt")
                        record.setValue(now, forKey: "updatedAt")

                        tagRecordsByNormalizedName[normalized.key] = record
                        usedTagIDs.insert(tagID)
                        tagRecord = record
                    }

                    guard let tagID = tagRecord.value(forKey: "id") as? String else {
                        continue
                    }

                    let linkKey = tagLinkKey(assetID: assetID, tagID: tagID)
                    guard existingLinkKeys.insert(linkKey).inserted else {
                        continue
                    }

                    let link = NSManagedObject(entity: entity(named: "AssetTagRecord"), insertInto: context)
                    link.setValue("\(assetID)-\(tagID)", forKey: "id")
                    link.setValue(library.id, forKey: "libraryID")
                    link.setValue(assetID, forKey: "assetID")
                    link.setValue(tagID, forKey: "tagID")
                    let importedAt = assetRecord.value(forKey: "importedAt") as? Date ?? Date()
                    link.setValue(importedAt.addingTimeInterval(TimeInterval(legacyIndex) / 1_000), forKey: "createdAt")
                }
            }

            if context.hasChanges {
                try context.save()
            }
        }
    }

    private func asset(
        from record: NSManagedObject,
        folderIDs: [String],
        paletteColors: [AssetColor],
        tags: [TagItem]
    ) -> AssetItem? {
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

        let width = intValue(record.value(forKey: "pixelWidth"))
        let height = intValue(record.value(forKey: "pixelHeight"))
        let dimensions: AssetDimensions?
        if let width, let height {
            dimensions = AssetDimensions(width: width, height: height)
        } else {
            dimensions = nil
        }

        let storageURL = storage.resolveAssetURL(relativePath: storageRelativePath, in: library)
        let exifMetadata = exifMetadata(from: record.value(forKey: "exifMetadataData"))
        let thumbnailURL = storage.thumbnailURL(forContentHash: contentHash, in: library)
        let resolvedThumbnailURL = FileManager.default.fileExists(atPath: thumbnailURL.path) ? thumbnailURL : nil
        let originalFileName = storedOriginalFileName(
            record.value(forKey: "originalFileName"),
            displayName: displayName,
            fileExtension: fileExtension,
            storageURL: storageURL
        )

        return AssetItem(
            id: id,
            libraryID: libraryID,
            displayName: displayName,
            originalFileName: originalFileName,
            originalURL: nil,
            sourcePageURL: storedURL(record.value(forKey: "sourcePageURL")),
            storageURL: storageURL,
            kind: kind,
            fileExtension: fileExtension,
            utiIdentifier: storedUTIIdentifier(record.value(forKey: "utiIdentifier")),
            byteSize: int64Value(record.value(forKey: "byteSize")) ?? 0,
            contentHash: contentHash,
            dimensions: dimensions,
            exifMetadata: exifMetadata,
            orientation: intValue(record.value(forKey: "orientation")),
            colorProfileName: record.value(forKey: "colorProfileName") as? String ?? exifMetadata?.profileName,
            note: record.value(forKey: "note") as? String,
            tags: tags,
            folderIDs: folderIDs,
            paletteColors: paletteColors,
            thumbnailURL: resolvedThumbnailURL,
            isFavorite: record.value(forKey: "isFavorite") as? Bool ?? false,
            isTrashed: record.value(forKey: "isTrashed") as? Bool ?? false,
            trashedAt: record.value(forKey: "trashedAt") as? Date,
            importedAt: importedAt,
            updatedAt: record.value(forKey: "updatedAt") as? Date ?? importedAt
        )
    }

    private func storedOriginalFileName(
        _ value: Any?,
        displayName: String,
        fileExtension: String,
        storageURL: URL
    ) -> String {
        if let value = value as? String, !value.isEmpty {
            return value
        }

        if !displayName.isEmpty {
            return fileExtension.isEmpty ? displayName : "\(displayName).\(fileExtension)"
        }

        return storageURL.lastPathComponent
    }

    private func storedUTIIdentifier(_ value: Any?) -> String {
        if let value = value as? String, !value.isEmpty {
            return value
        }

        return "public.data"
    }

    private func storedURL(_ value: Any?) -> URL? {
        guard let value = value as? String, !value.isEmpty else {
            return nil
        }

        return URL(string: value)
    }

    private func loadMemberships() throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetFolderMembershipRecord")
        request.predicate = NSPredicate(format: "libraryID == %@", library.id)
        request.sortDescriptors = [
            NSSortDescriptor(key: "createdAt", ascending: true),
            NSSortDescriptor(key: "id", ascending: true)
        ]
        return try context.fetch(request)
    }

    private func folderIDsByAssetID(for assetIDs: Set<String>) throws -> [String: [String]] {
        guard !assetIDs.isEmpty else {
            return [:]
        }

        let folders = try loadFolders()
        let sortOrder = Dictionary(uniqueKeysWithValues: folders.enumerated().map { ($0.element.id, $0.offset) })
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetFolderMembershipRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND assetID IN %@", library.id, Array(assetIDs))
        let records = try context.fetch(request)

        var grouped: [String: [String]] = [:]
        for record in records {
            guard let assetID = record.value(forKey: "assetID") as? String,
                  let folderID = record.value(forKey: "folderID") as? String else {
                continue
            }
            grouped[assetID, default: []].append(folderID)
        }

        return grouped.mapValues { folderIDs in
            folderIDs.sorted { lhs, rhs in
                let lhsOrder = sortOrder[lhs] ?? Int.max
                let rhsOrder = sortOrder[rhs] ?? Int.max
                if lhsOrder == rhsOrder {
                    return lhs < rhs
                }
                return lhsOrder < rhsOrder
            }
        }
    }

    private func colorsByAssetID(for assetIDs: Set<String>) throws -> [String: [AssetColor]] {
        guard !assetIDs.isEmpty else {
            return [:]
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetColorRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND assetID IN %@", library.id, Array(assetIDs))
        request.sortDescriptors = [NSSortDescriptor(key: "sortIndex", ascending: true)]

        var grouped: [String: [AssetColor]] = [:]
        for record in try context.fetch(request) {
            guard let color = color(from: record) else {
                continue
            }
            grouped[color.assetID, default: []].append(color)
        }
        return grouped
    }

    private func tagsByAssetID(for assetIDs: Set<String>) throws -> [String: [TagItem]] {
        guard !assetIDs.isEmpty else {
            return [:]
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetTagRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND assetID IN %@", library.id, Array(assetIDs))
        request.sortDescriptors = [
            NSSortDescriptor(key: "createdAt", ascending: true),
            NSSortDescriptor(key: "id", ascending: true)
        ]
        let records = try context.fetch(request)
        let tagIDs = Set(records.compactMap { $0.value(forKey: "tagID") as? String })
        let tagsByID = try tagItemsByID(for: tagIDs)

        var grouped: [String: [TagItem]] = [:]
        for record in records {
            guard let assetID = record.value(forKey: "assetID") as? String,
                  let tagID = record.value(forKey: "tagID") as? String,
                  let tag = tagsByID[tagID] else {
                continue
            }
            grouped[assetID, default: []].append(tag)
        }

        return grouped.mapValues { tags in
            var seen = Set<TagItem.ID>()
            return tags.filter { seen.insert($0.id).inserted }
        }
    }

    private func tagItemsByID(for tagIDs: Set<String>) throws -> [String: TagItem] {
        guard !tagIDs.isEmpty else {
            return [:]
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "TagRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND id IN %@", library.id, Array(tagIDs))
        request.sortDescriptors = [NSSortDescriptor(key: "name", ascending: true)]

        var result: [String: TagItem] = [:]
        for record in try context.fetch(request) {
            guard let tag = tag(from: record) else {
                continue
            }
            result[tag.id] = tag
        }
        return result
    }

    private func loadTagRecords() throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "TagRecord")
        request.predicate = NSPredicate(format: "libraryID == %@", library.id)
        request.sortDescriptors = [
            NSSortDescriptor(key: "name", ascending: true),
            NSSortDescriptor(key: "id", ascending: true)
        ]
        return try context.fetch(request)
    }

    private func resolveOrCreateTagRecords(named names: [String]) throws -> [NSManagedObject] {
        var recordsByNormalizedName = Dictionary(
            uniqueKeysWithValues: try loadTagRecords().compactMap { record -> (String, NSManagedObject)? in
                guard let normalizedName = record.value(forKey: "normalizedName") as? String else {
                    return nil
                }
                return (normalizedName, record)
            }
        )

        var resolved: [NSManagedObject] = []
        var seen = Set<String>()
        let now = Date()

        for name in names {
            guard let normalized = normalizedTagName(name),
                  seen.insert(normalized.key).inserted else {
                continue
            }

            if let existing = recordsByNormalizedName[normalized.key] {
                resolved.append(existing)
                continue
            }

            let record = NSManagedObject(entity: entity(named: "TagRecord"), insertInto: context)
            record.setValue(UUID().uuidString, forKey: "id")
            record.setValue(library.id, forKey: "libraryID")
            record.setValue(normalized.displayName, forKey: "name")
            record.setValue(normalized.key, forKey: "normalizedName")
            record.setValue(now, forKey: "createdAt")
            record.setValue(now, forKey: "updatedAt")
            recordsByNormalizedName[normalized.key] = record
            resolved.append(record)
        }

        return resolved
    }

    private func tagRecord(id tagID: TagItem.ID) throws -> NSManagedObject? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "TagRecord")
        request.fetchLimit = 1
        request.predicate = NSPredicate(format: "libraryID == %@ AND id == %@", library.id, tagID)
        return try context.fetch(request).first
    }

    private func tagRecord(normalizedName: String) throws -> NSManagedObject? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "TagRecord")
        request.fetchLimit = 1
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND normalizedName == %@",
            library.id,
            normalizedName
        )
        return try context.fetch(request).first
    }

    private func tagLinkRecords(assetID: AssetItem.ID) throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetTagRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND assetID == %@", library.id, assetID)
        request.sortDescriptors = [NSSortDescriptor(key: "createdAt", ascending: true)]
        return try context.fetch(request)
    }

    private func allTagLinkRecords() throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetTagRecord")
        request.predicate = NSPredicate(format: "libraryID == %@", library.id)
        return try context.fetch(request)
    }

    private func tagLinkRecords(tagID: TagItem.ID) throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetTagRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND tagID == %@", library.id, tagID)
        return try context.fetch(request)
    }

    private func tagLinkRecords(tagID: TagItem.ID, assetIDs: Set<AssetItem.ID>) throws -> [NSManagedObject] {
        guard !assetIDs.isEmpty else {
            return []
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetTagRecord")
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND tagID == %@ AND assetID IN %@",
            library.id,
            tagID,
            Array(assetIDs)
        )
        return try context.fetch(request)
    }

    private func assetIDs(linkedToTagID tagID: TagItem.ID) throws -> Set<AssetItem.ID> {
        Set(try tagLinkRecords(tagID: tagID).compactMap { $0.value(forKey: "assetID") as? String })
    }

    private func assetRecords(ids: Set<AssetItem.ID>) throws -> [NSManagedObject] {
        guard !ids.isEmpty else {
            return []
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND id IN %@", library.id, Array(ids))
        return try context.fetch(request)
    }

    private func tagLinkKey(assetID: AssetItem.ID, tagID: TagItem.ID) -> String {
        "\(assetID)\u{0}\(tagID)"
    }

    private func saveColors(_ colors: [AssetColor], forAssetID assetID: String) {
        for color in colors {
            let record = NSManagedObject(entity: entity(named: "AssetColorRecord"), insertInto: context)
            record.setValue(color.id, forKey: "id")
            record.setValue(color.libraryID, forKey: "libraryID")
            record.setValue(assetID, forKey: "assetID")
            record.setValue(color.hex, forKey: "hex")
            record.setValue(color.coverage, forKey: "coverage")
            record.setValue(color.sortIndex, forKey: "sortIndex")
        }
    }

    private func color(from record: NSManagedObject) -> AssetColor? {
        guard let id = record.value(forKey: "id") as? String,
              let libraryID = record.value(forKey: "libraryID") as? String,
              let assetID = record.value(forKey: "assetID") as? String,
              let hex = record.value(forKey: "hex") as? String,
              let coverage = doubleValue(record.value(forKey: "coverage")),
              let sortIndex = intValue(record.value(forKey: "sortIndex")) else {
            return nil
        }

        return AssetColor(
            id: id,
            libraryID: libraryID,
            assetID: assetID,
            hex: hex,
            coverage: coverage,
            sortIndex: sortIndex
        )
    }

    private func tag(from record: NSManagedObject) -> TagItem? {
        guard let id = record.value(forKey: "id") as? String,
              let name = record.value(forKey: "name") as? String else {
            return nil
        }

        return TagItem(
            id: id,
            name: name,
            colorHex: record.value(forKey: "colorHex") as? String
        )
    }

    private func normalizedTagName(_ name: String) -> (displayName: String, key: String)? {
        let displayName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !displayName.isEmpty else {
            return nil
        }

        return (displayName, displayName.lowercased())
    }

    private func folder(from record: NSManagedObject) -> AssetFolder? {
        guard let id = record.value(forKey: "id") as? String,
              let libraryID = record.value(forKey: "libraryID") as? String,
              let name = record.value(forKey: "name") as? String,
              let sortIndex = intValue(record.value(forKey: "sortIndex")),
              let createdAt = record.value(forKey: "createdAt") as? Date,
              let updatedAt = record.value(forKey: "updatedAt") as? Date else {
            return nil
        }

        return AssetFolder(
            id: id,
            libraryID: libraryID,
            name: name,
            parentID: record.value(forKey: "parentID") as? String,
            sortIndex: sortIndex,
            createdAt: createdAt,
            updatedAt: updatedAt
        )
    }

    private func nextFolderSortIndex(parentID: String?) throws -> Int {
        let request = NSFetchRequest<NSManagedObject>(entityName: "FolderRecord")
        if let parentID {
            request.predicate = NSPredicate(
                format: "libraryID == %@ AND parentID == %@",
                library.id,
                parentID
            )
        } else {
            request.predicate = NSPredicate(format: "libraryID == %@ AND parentID == nil", library.id)
        }

        let records = try context.fetch(request)
        let maxSortIndex = records.compactMap { intValue($0.value(forKey: "sortIndex")) }.max() ?? -1
        return maxSortIndex + 1
    }

    private func normalizeFolderSortIndexes(parentID: String?, updatedAt: Date) throws {
        try updateFolderSortIndexes(try folderRecords(parentID: parentID), updatedAt: updatedAt)
    }

    private func normalizeFolderSortIndexes(
        parentID: String?,
        movingRecord: NSManagedObject,
        relativeTo targetID: AssetFolder.ID?,
        insertAfterTarget: Bool,
        updatedAt: Date
    ) throws {
        guard let movingID = movingRecord.value(forKey: "id") as? String else {
            throw LibraryMetadataError.missingFolder
        }

        var siblingRecords = try folderRecords(parentID: parentID).filter {
            ($0.value(forKey: "id") as? String) != movingID
        }

        let insertionIndex: Int
        if let targetID {
            guard let targetIndex = siblingRecords.firstIndex(where: { ($0.value(forKey: "id") as? String) == targetID }) else {
                throw LibraryMetadataError.missingFolder
            }
            insertionIndex = insertAfterTarget ? targetIndex + 1 : targetIndex
        } else {
            insertionIndex = siblingRecords.endIndex
        }

        siblingRecords.insert(movingRecord, at: min(insertionIndex, siblingRecords.endIndex))
        try updateFolderSortIndexes(siblingRecords, updatedAt: updatedAt)
    }

    private func updateFolderSortIndexes(_ records: [NSManagedObject], updatedAt: Date) throws {
        for (index, record) in records.enumerated() {
            guard intValue(record.value(forKey: "sortIndex")) != index else {
                continue
            }

            record.setValue(index, forKey: "sortIndex")
            record.setValue(updatedAt, forKey: "updatedAt")
        }
    }

    private func saveFolderAssignments(
        _ assignmentsByContentHash: [String: [[String]]],
        affectedAssetIDs: inout Set<AssetItem.ID>
    ) throws {
        guard !assignmentsByContentHash.isEmpty else {
            return
        }

        var folderPathCache: [String: AssetFolder.ID] = [:]

        for (contentHash, assignments) in assignmentsByContentHash {
            guard let assetRecord = try assetRecord(contentHash: contentHash),
                  let assetID = assetRecord.value(forKey: "id") as? String else {
                continue
            }

            affectedAssetIDs.insert(assetID)
            let now = Date()
            var didChangeAsset = false

            if (assetRecord.value(forKey: "isTrashed") as? Bool) == true {
                assetRecord.setValue(false, forKey: "isTrashed")
                assetRecord.setValue(nil, forKey: "trashedAt")
                didChangeAsset = true
            }

            var seenPaths = Set<[String]>()
            for assignment in assignments {
                let path = assignment.map {
                    $0.trimmingCharacters(in: .whitespacesAndNewlines)
                }.filter {
                    !$0.isEmpty
                }

                guard !path.isEmpty, seenPaths.insert(path).inserted else {
                    continue
                }

                let folderID = try folderID(forPath: path, cache: &folderPathCache)
                let existing = try existingMembershipAssetIDs(folderID: folderID, assetIDs: [assetID])
                guard !existing.contains(assetID) else {
                    continue
                }

                let membership = NSManagedObject(
                    entity: entity(named: "AssetFolderMembershipRecord"),
                    insertInto: context
                )
                membership.setValue("\(assetID)-\(folderID)", forKey: "id")
                membership.setValue(library.id, forKey: "libraryID")
                membership.setValue(assetID, forKey: "assetID")
                membership.setValue(folderID, forKey: "folderID")
                membership.setValue(now, forKey: "createdAt")
                didChangeAsset = true
            }

            if didChangeAsset {
                assetRecord.setValue(now, forKey: "updatedAt")
            }
        }
    }

    private func folderID(
        forPath components: [String],
        cache: inout [String: AssetFolder.ID]
    ) throws -> AssetFolder.ID {
        var parentID: AssetFolder.ID?
        var currentPath: [String] = []

        for component in components {
            currentPath.append(component)
            let cacheKey = currentPath.joined(separator: "\u{0}")
            if let cachedID = cache[cacheKey] {
                parentID = cachedID
                continue
            }

            if let existingRecord = try folderRecord(parentID: parentID, name: component),
               let existingID = existingRecord.value(forKey: "id") as? String {
                cache[cacheKey] = existingID
                parentID = existingID
                continue
            }

            let now = Date()
            let folder = AssetFolder(
                libraryID: library.id,
                name: component,
                parentID: parentID,
                sortIndex: try nextFolderSortIndex(parentID: parentID),
                createdAt: now,
                updatedAt: now
            )
            let record = NSManagedObject(entity: entity(named: "FolderRecord"), insertInto: context)
            record.setValue(folder.id, forKey: "id")
            record.setValue(folder.libraryID, forKey: "libraryID")
            record.setValue(folder.name, forKey: "name")
            record.setValue(folder.parentID, forKey: "parentID")
            record.setValue(folder.sortIndex, forKey: "sortIndex")
            record.setValue(folder.createdAt, forKey: "createdAt")
            record.setValue(folder.updatedAt, forKey: "updatedAt")

            cache[cacheKey] = folder.id
            parentID = folder.id
        }

        guard let folderID = parentID else {
            throw LibraryMetadataError.missingFolder
        }
        return folderID
    }

    private func folderRecord(id: String) throws -> NSManagedObject? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "FolderRecord")
        request.fetchLimit = 1
        request.predicate = NSPredicate(format: "libraryID == %@ AND id == %@", library.id, id)
        return try context.fetch(request).first
    }

    private func folderRecord(parentID: String?, name: String) throws -> NSManagedObject? {
        let request = NSFetchRequest<NSManagedObject>(entityName: "FolderRecord")
        request.fetchLimit = 1
        if let parentID {
            request.predicate = NSPredicate(
                format: "libraryID == %@ AND parentID == %@ AND name == %@",
                library.id,
                parentID,
                name
            )
        } else {
            request.predicate = NSPredicate(
                format: "libraryID == %@ AND parentID == nil AND name == %@",
                library.id,
                name
            )
        }
        return try context.fetch(request).first
    }

    private func folderRecords(parentID: String?) throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "FolderRecord")
        if let parentID {
            request.predicate = NSPredicate(
                format: "libraryID == %@ AND parentID == %@",
                library.id,
                parentID
            )
        } else {
            request.predicate = NSPredicate(format: "libraryID == %@ AND parentID == nil", library.id)
        }
        request.sortDescriptors = [
            NSSortDescriptor(key: "sortIndex", ascending: true),
            NSSortDescriptor(key: "createdAt", ascending: true),
            NSSortDescriptor(key: "id", ascending: true)
        ]
        return try context.fetch(request)
    }

    private func folderRecords(ids: Set<String>) throws -> [NSManagedObject] {
        guard !ids.isEmpty else {
            return []
        }
        let request = NSFetchRequest<NSManagedObject>(entityName: "FolderRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND id IN %@", library.id, Array(ids))
        return try context.fetch(request)
    }

    private func descendantFolderIDs(startingAt id: String) throws -> Set<String> {
        let folders = try loadFolders()
        let childrenByParentID = Dictionary(grouping: folders.compactMap { folder -> (String, String)? in
            guard let parentID = folder.parentID else {
                return nil
            }
            return (parentID, folder.id)
        }, by: \.0).mapValues { pairs in pairs.map(\.1) }

        var result: Set<String> = []
        var pending = [id]
        while let next = pending.popLast() {
            guard result.insert(next).inserted else {
                continue
            }
            pending.append(contentsOf: childrenByParentID[next, default: []])
        }
        return result
    }

    private func existingAssetIDs(in ids: Set<AssetItem.ID>, includeTrashed: Bool = true) throws -> Set<String> {
        guard !ids.isEmpty else {
            return []
        }
        let request = NSFetchRequest<NSDictionary>(entityName: "AssetRecord")
        request.resultType = .dictionaryResultType
        request.propertiesToFetch = ["id"]
        request.predicate = includeTrashed
            ? NSPredicate(format: "libraryID == %@ AND id IN %@", library.id, Array(ids))
            : NSPredicate(format: "libraryID == %@ AND id IN %@ AND isTrashed == NO", library.id, Array(ids))
        return Set(try context.fetch(request).compactMap { $0["id"] as? String })
    }

    private func existingMembershipAssetIDs(folderID: String, assetIDs: Set<String>) throws -> Set<String> {
        Set(try membershipRecords(folderID: folderID, assetIDs: assetIDs).compactMap {
            $0.value(forKey: "assetID") as? String
        })
    }

    private func membershipRecords(folderID: String, assetIDs: Set<String>) throws -> [NSManagedObject] {
        guard !assetIDs.isEmpty else {
            return []
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetFolderMembershipRecord")
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND folderID == %@ AND assetID IN %@",
            library.id,
            folderID,
            Array(assetIDs)
        )
        return try context.fetch(request)
    }

    private func membershipRecords(assetIDs: Set<String>) throws -> [NSManagedObject] {
        guard !assetIDs.isEmpty else {
            return []
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetFolderMembershipRecord")
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND assetID IN %@",
            library.id,
            Array(assetIDs)
        )
        return try context.fetch(request)
    }

    private func membershipRecords(folderIDs: Set<String>) throws -> [NSManagedObject] {
        guard !folderIDs.isEmpty else {
            return []
        }

        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetFolderMembershipRecord")
        request.predicate = NSPredicate(
            format: "libraryID == %@ AND folderID IN %@",
            library.id,
            Array(folderIDs)
        )
        return try context.fetch(request)
    }

    private func colorRecords(assetID: AssetItem.ID) throws -> [NSManagedObject] {
        let request = NSFetchRequest<NSManagedObject>(entityName: "AssetColorRecord")
        request.predicate = NSPredicate(format: "libraryID == %@ AND assetID == %@", library.id, assetID)
        return try context.fetch(request)
    }

    private func entity(named name: String) -> NSEntityDescription {
        NSEntityDescription.entity(forEntityName: name, in: context)!
    }

    private func intValue(_ value: Any?) -> Int? {
        if let value = value as? Int {
            return value
        }
        if let value = value as? NSNumber {
            return value.intValue
        }
        return nil
    }

    private func exifMetadataData(_ metadata: AssetExifMetadata?) -> Data? {
        guard let metadata else {
            return nil
        }

        return try? JSONEncoder.momento.encode(metadata)
    }

    private func exifMetadata(from value: Any?) -> AssetExifMetadata? {
        guard let data = value as? Data else {
            return nil
        }

        return try? JSONDecoder.momento.decode(AssetExifMetadata.self, from: data)
    }

    private func tagsData(_ tags: [TagItem]) -> Data? {
        guard !tags.isEmpty else {
            return nil
        }

        return try? JSONEncoder.momento.encode(tags)
    }

    private func tags(from value: Any?) -> [TagItem] {
        guard let data = value as? Data,
              let tags = try? JSONDecoder.momento.decode([TagItem].self, from: data) else {
            return []
        }

        return tags
    }

    private func int64Value(_ value: Any?) -> Int64? {
        if let value = value as? Int64 {
            return value
        }
        if let value = value as? NSNumber {
            return value.int64Value
        }
        return nil
    }

    private func doubleValue(_ value: Any?) -> Double? {
        if let value = value as? Double {
            return value
        }
        if let value = value as? NSNumber {
            return value.doubleValue
        }
        return nil
    }
}

enum LibraryMetadataError: LocalizedError {
    case invalidFolderName
    case invalidFolderMove
    case invalidAssetName
    case invalidTagName
    case duplicateTagName
    case missingFolder
    case missingAsset
    case missingTag

    var errorDescription: String? {
        switch self {
        case .invalidFolderName:
            "Enter a folder name."
        case .invalidFolderMove:
            "Move the folder to a different location."
        case .invalidAssetName:
            "Enter an asset title."
        case .invalidTagName:
            "Enter a tag name."
        case .duplicateTagName:
            "A tag with this name already exists."
        case .missingFolder:
            "This folder no longer exists."
        case .missingAsset:
            "This asset is no longer available."
        case .missingTag:
            "This tag no longer exists."
        }
    }
}
