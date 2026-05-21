import CoreData
import Foundation

nonisolated final class MomentoCoreDataStack {
    let container: NSPersistentContainer

    init(library: AssetLibrary, storage: LibraryStorage = LibraryStorage()) throws {
        container = NSPersistentContainer(name: "MomentoModel", managedObjectModel: try Self.managedObjectModel())

        let storeDescription = NSPersistentStoreDescription(url: storage.databaseURL(for: library))
        storeDescription.type = NSSQLiteStoreType
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

    private static func managedObjectModel() throws -> NSManagedObjectModel {
        for bundle in [Bundle.main] + Bundle.allBundles + Bundle.allFrameworks {
            guard let modelURL = bundle.url(forResource: "MomentoModel", withExtension: "momd"),
                  let model = NSManagedObjectModel(contentsOf: modelURL) else {
                continue
            }
            return model
        }

        throw MomentoCoreDataStackError.missingManagedObjectModel
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
