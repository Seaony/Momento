# CloudKit Schema Checklist

Status: draft, development schema only.

This checklist tracks the CloudKit records that macOS must write compatibly with `/Users/seaony/code/Momento-iOS`. Do not promote the production schema until real Mac/iOS smoke tests pass with the same container.

## Container and Zones

- Container: `iCloud.com.seaony.Momento`
- Database: private database
- Catalog zone: `MomentoCatalog`
- Library zone: `MomentoLibrary-<libraryID component>`

## Record Types

### CloudLibrary

Fields:

- `id`: String, queryable
- `displayName`: String, queryable
- `libraryZoneName`: String
- `createdAt`: Date, sortable
- `updatedAt`: Date, sortable
- `deletedAt`: Date, queryable
- `schemaVersion`: Int64

Indexes:

- Queryable: `id`, `deletedAt`
- Sortable: `updatedAt`, `createdAt`

### CloudAsset

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `contentHash`: String, queryable
- `displayName`: String
- `originalFileName`: String
- `fileExtension`: String
- `utiIdentifier`: String
- `kind`: String
- `byteSize`: Int64
- `pixelWidth`: Int64
- `pixelHeight`: Int64
- `orientation`: Int64
- `colorProfileName`: String
- `sourcePageURL`: String
- `note`: String
- `isFavorite`: Int64/Bool
- `isTrashed`: Int64/Bool, queryable
- `trashedAt`: Date
- `importedAt`: Date, sortable
- `updatedAt`: Date, sortable
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `contentHash`, `deletedAt`, `isTrashed`
- Sortable: `updatedAt`, `importedAt`

### CloudAssetColor

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `assetID`: String, queryable
- `hex`: String
- `coverage`: Double
- `sortIndex`: Int64
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `assetID`, `deletedAt`
- Sortable: `sortIndex`

### CloudAssetBlob

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `contentHash`: String, queryable
- `originalFile`: Asset
- `byteSize`: Int64
- `uploadedAt`: Date, sortable
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `contentHash`, `deletedAt`
- Sortable: `uploadedAt`

### CloudFolder

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `name`: String
- `parentID`: String, queryable
- `sortIndex`: Int64, sortable
- `createdAt`: Date
- `updatedAt`: Date, sortable
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `parentID`, `deletedAt`
- Sortable: `sortIndex`, `updatedAt`

### CloudTag

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `name`: String
- `normalizedName`: String, queryable
- `colorHex`: String
- `createdAt`: Date
- `updatedAt`: Date, sortable
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `normalizedName`, `deletedAt`
- Sortable: `updatedAt`

### CloudFolderMembership

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `assetID`: String, queryable
- `folderID`: String, queryable
- `createdAt`: Date
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `assetID`, `folderID`, `deletedAt`

### CloudTagMembership

Fields:

- `id`: String, queryable
- `libraryID`: String, queryable
- `assetID`: String, queryable
- `tagID`: String, queryable
- `createdAt`: Date
- `deletedAt`: Date, queryable

Indexes:

- Queryable: `libraryID`, `assetID`, `tagID`, `deletedAt`

## Promotion Gates

- Development schema contains every field above.
- Queryable/sortable indexes above exist in the CloudKit Console.
- Mac can create a cloud library and iOS can discover it.
- iOS can create a cloud library and Mac can discover it.
- Mac and iOS agree on record names for library, asset, blob, color, folder, tag, and membership records.
- Upload/download smoke test passes with one small image from Mac to iOS and one small image from iOS to Mac.
- Account sign-out and account-switch tests block writes instead of writing into a stale cache.
- No production schema promotion until the container, provisioning profiles, and real-device smoke evidence are recorded in `docs/superpowers/specs/2026-05-27-cloud-library-sync-phase0-status.md`.
