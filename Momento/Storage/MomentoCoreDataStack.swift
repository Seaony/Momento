import CoreData
import Foundation

nonisolated final class MomentoCoreDataStack {
    let container: NSPersistentContainer

    init(library: AssetLibrary, storage: LibraryStorage = LibraryStorage()) throws {
        container = NSPersistentContainer(name: "MomentoModel", managedObjectModel: try Self.managedObjectModel())

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
