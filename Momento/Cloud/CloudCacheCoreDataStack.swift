import CoreData
import Foundation

nonisolated final class CloudCacheCoreDataStack {
    let container: NSPersistentContainer

    init(
        cloudAccountID: String,
        libraryID: String,
        paths: CloudLibraryCachePaths = CloudLibraryCachePaths()
    ) throws {
        try paths.prepareLibraryDirectories(cloudAccountID: cloudAccountID, libraryID: libraryID)

        container = NSPersistentContainer(
            name: "MomentoCloudModel",
            managedObjectModel: try Self.managedObjectModel()
        )

        let storeDescription = NSPersistentStoreDescription(
            url: try paths.cacheDatabaseURL(cloudAccountID: cloudAccountID, libraryID: libraryID)
        )
        storeDescription.type = NSSQLiteStoreType
        storeDescription.shouldMigrateStoreAutomatically = true
        storeDescription.shouldInferMappingModelAutomatically = true
        container.persistentStoreDescriptions = [storeDescription]

        var loadError: (any Error)?
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
            guard let modelURL = bundle.url(forResource: "MomentoCloudModel", withExtension: "momd"),
                  let model = managedObjectModel(in: modelURL, versionName: versionName) else {
                continue
            }
            return model
        }

        throw CloudCacheCoreDataStackError.missingManagedObjectModel
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

nonisolated enum CloudCacheCoreDataStackError: LocalizedError {
    case missingManagedObjectModel

    var errorDescription: String? {
        switch self {
        case .missingManagedObjectModel:
            "MomentoCloudModel could not be loaded from the app bundle."
        }
    }
}
