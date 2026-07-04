// 中文注释：本文件负责为单个资源库打开 Core Data SQLite store，并启用轻量迁移。
import CoreData
import Foundation

nonisolated final class MomentoCoreDataStack {
    let container: NSPersistentContainer

    init(library: AssetLibrary, storage: LibraryStorage = LibraryStorage()) throws {
        container = NSPersistentContainer(name: "MomentoModel", managedObjectModel: try Self.sharedManagedObjectModel())

        let storeDescription = NSPersistentStoreDescription(url: storage.databaseURL(for: library))
        storeDescription.type = NSSQLiteStoreType
        storeDescription.shouldMigrateStoreAutomatically = true
        storeDescription.shouldInferMappingModelAutomatically = true
        container.persistentStoreDescriptions = [storeDescription]

        var loadError: Error?
        let semaphore = DispatchSemaphore(value: 0)
        container.loadPersistentStores { _, error in
            loadError = error
            semaphore.signal()
        }
        semaphore.wait()

        if let loadError {
            throw loadError
        }

        container.viewContext.mergePolicy = NSMergePolicy(merge: .mergeByPropertyStoreTrumpMergePolicyType)
    }

    // 中文注释：默认模型在整个进程内共享同一个 NSManagedObjectModel 实例。
    // 否则每次打开/校验/导入导出库都会重新扫描全部 bundle/framework 并反序列化整个 .momd，
    // 且同一进程内多个模型实例并存会触发 Core Data "Multiple NSEntityDescriptions claim the NSManagedObject subclass" 警告。
    // NSManagedObjectModel 加载后按只读方式使用，跨库共享是 Core Data 官方推荐做法。
    nonisolated(unsafe) private static let cachedDefaultModel = try? managedObjectModel()

    private static func sharedManagedObjectModel() throws -> NSManagedObjectModel {
        if let cachedDefaultModel {
            return cachedDefaultModel
        }
        // 缓存构建失败时回退到实时构建，以便把真实错误抛给调用方，而不是静默使用空模型。
        return try managedObjectModel()
    }

    static func managedObjectModel(versionName: String? = nil) throws -> NSManagedObjectModel {
        for bundle in [Bundle.main] + Bundle.allBundles + Bundle.allFrameworks {
            guard let modelURL = bundle.url(forResource: "MomentoModel", withExtension: "momd"),
                  let model = managedObjectModel(in: modelURL, versionName: versionName) else {
                continue
            }
            return model
        }

        throw MomentoCoreDataStackError.missingManagedObjectModel
    }

    private static func managedObjectModel(in modelURL: URL, versionName: String?) -> NSManagedObjectModel? {
        guard let versionName else {
            return NSManagedObjectModel(contentsOf: modelURL)
        }

        let versionURL = modelURL
            .appendingPathComponent(versionName)
            .appendingPathExtension("mom")
        return NSManagedObjectModel(contentsOf: versionURL)
    }
}

enum MomentoCoreDataStackError: LocalizedError {
    case missingManagedObjectModel

    var errorDescription: String? {
        switch self {
        case .missingManagedObjectModel:
            "MomentoModel could not be loaded from the app bundle."
        }
    }
}
