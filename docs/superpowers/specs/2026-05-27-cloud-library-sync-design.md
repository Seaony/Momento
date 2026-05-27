# Momento Cloud Library Sync Design

Date: 2026-05-27

## Summary

Momento should support two library storage modes:

- Local library: Mac-only, stored as the current `.momento` package. It is not visible on iOS and is not synced.
- Cloud library: Mac and iOS read-write library, synced through the user's iCloud account.

The cloud mode should not sync the existing `.momento` package through iCloud Drive. The package contains a SQLite Core Data store, original files, thumbnails, previews, and transient caches. That shape is a good local document format, but it is fragile as a multi-device write protocol.

The recommended design is a CloudKit-backed library model:

- CloudKit private database is the cloud source of truth for cloud libraries.
- A local SQLite/Core Data cache keeps each device usable offline and keeps the existing UI model mostly intact.
- Original files are immutable content-addressed blobs keyed by SHA-256.
- Thumbnails/previews are caches. A cloud thumbnail may be uploaded for first-screen speed, but it is never authoritative.
- Local libraries continue to use the existing `.momento` package with no iCloud behavior. iOS does not implement `.momento` package opening in v1; iOS only consumes cloud libraries.

## Current Project Baseline

The current macOS app stores a library as a local package:

```text
<Name>.momento/
├── manifest.json
├── database/library.sqlite
├── assets/<hashPrefix>/<sha256>.<ext>
├── thumbnails/
├── previews/
└── metadata/import-sessions/
```

Relevant source:

- `Momento/Storage/LibraryStorage.swift` owns package structure, manifest IO, asset paths, cache clearing, import/export, and delete behavior.
- `Momento/Storage/MomentoCoreDataStack.swift` opens one `NSSQLiteStoreType` Core Data store at `database/library.sqlite`.
- `Momento/Storage/LibraryMetadataStore.swift` wraps the Core Data store and exposes value-type `AssetItem` models to `LibraryStore`.
- `Momento/Storage/LibraryAccessScope.swift` stores recent local libraries as security-scoped bookmarks.
- `Momento/Momento.entitlements` currently has sandbox, file access, and network entitlements, but no iCloud or CloudKit entitlements.

The current Core Data model also uses uniqueness constraints, for example `(libraryID, contentHash)`. That is appropriate for the local package, but it matters for CloudKit model choices.

## Official Documentation Findings

The design below is based on Apple documentation checked during this review:

- CloudKit is a transfer service for app data, not a replacement for an app's local model/cache. Apple notes that CloudKit has minimal offline caching support and relies on network plus a valid iCloud account for user-specific saves.
  - Source: [CloudKit framework overview](https://developer.apple.com/documentation/CloudKit)
- The private CloudKit database is available only with an iCloud account and counts toward the user's iCloud storage quota.
  - Source: [CKDatabase.Scope.private](https://developer.apple.com/documentation/cloudkit/ckdatabase/scope/private)
- `CKAsset` is the official mechanism for photos, videos, and other binary files in CloudKit records. CloudKit stores only asset data, so filename and metadata must live in separate fields. Fetched assets are staged and must be moved into the app container if the app needs to keep them.
  - Source: [CKAsset](https://developer.apple.com/documentation/cloudkit/ckasset)
- CloudKit fetch/query APIs support `desiredKeys`, which should be used to avoid fetching large asset fields when only metadata is needed.
  - Source: [Local Records](https://developer.apple.com/documentation/cloudkit/local-records)
- CloudKit record IDs are made from a record name and zone ID. Custom record names should be ASCII strings no longer than 255 characters, and record IDs must be unique within the zone.
  - Source: [CKRecord.ID](https://developer.apple.com/documentation/cloudkit/ckrecord/id)
- CloudKit errors such as `limitExceeded` require splitting oversized operations. Some errors expose retry timing; sync code must classify retryable and non-retryable failures instead of retrying blindly.
  - Source: [CKError.limitExceeded](https://developer.apple.com/documentation/cloudkit/ckerror/limitexceeded)
- CloudKit record conflicts are versioned by server-managed record change tags. `serverRecordChanged` returns client, server, and ancestor records; conflict resolution must merge into the server record and save that version.
  - Source: [CKRecord.recordChangeTag](https://developer.apple.com/documentation/cloudkit/ckrecord/recordchangetag)
  - Source: [CKError.serverRecordChanged](https://developer.apple.com/documentation/cloudkit/ckerror/serverrecordchanged)
- CloudKit schema and indexes need production planning. Apple documents that production schemas can only evolve forward, and queryable fields need indexes.
  - Source: [Designing apps using CloudKit](https://developer.apple.com/icloud/cloudkit/designing/)
  - Source: [Inspecting and Editing an iCloud Container's Schema](https://developer.apple.com/documentation/cloudkit/inspecting-and-editing-an-icloud-container-s-schema)
- CloudKit custom zones are intended to encapsulate related records and allow zone-by-zone syncing. Deleting a zone deletes its records. Sharing later also requires custom zones or record hierarchies.
  - Source: [CKRecordZone](https://developer.apple.com/documentation/cloudkit/ckrecordzone)
  - Source: [Local Records](https://developer.apple.com/documentation/cloudkit/local-records)
- Cloud libraries need explicit account-state checks. `CKContainer.accountStatus` reports whether the system can access the user's iCloud account, and account changes must block cloud writes until revalidated.
  - Source: [CKContainer](https://developer.apple.com/documentation/cloudkit/ckcontainer)
- `CKSyncEngine` is Apple's current sync helper for local/remote CloudKit records. It requires persisting sync-engine state, CloudKit and remote notification entitlements, and accepting that background sync timing is indeterminate.
  - Source: [CKSyncEngine](https://developer.apple.com/documentation/cloudkit/cksyncengine-5sie5)
  - Source: [CKSyncEngine.Configuration.database](https://developer.apple.com/documentation/cloudkit/cksyncengineconfiguration/database)
  - Source: [CKSyncEngine.Configuration.stateSerialization](https://developer.apple.com/documentation/cloudkit/cksyncengine-5sie5/configuration/stateserialization)
  - Source: [Apple CKSyncEngine sample](https://github.com/apple/sample-cloudkit-sync-engine)
- Core Data with CloudKit only works if the model is compatible with CloudKit limitations. Apple documents that unique constraints are unsupported, relationships must be optional, and production CloudKit schemas need careful forward-compatibility planning.
  - Source: [Mirroring a Core Data store with CloudKit](https://developer.apple.com/documentation/CoreData/mirroring-a-core-data-store-with-cloudkit)
  - Source: [Creating a Core Data Model for CloudKit](https://developer.apple.com/documentation/CoreData/creating-a-core-data-model-for-cloudkit)
- iCloud Documents can sync document packages, but multi-device document writes require file coordinators/file presenters and conflict handling. Package syncing is a document workflow, not a good fit for a live multi-device SQLite package with caches.
  - Source: [iCloud File Management](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/iCloud/iCloud.html)

## Alternatives Reviewed

### Option 1: Put `.momento` packages in iCloud Drive

This is the smallest implementation on paper, but it is not suitable for the requested Mac/iOS read-write behavior.

Problems:

- SQLite store, WAL files, thumbnails, previews, and original files would all sync as package contents.
- Multi-device edits would become file-version conflicts, not semantic asset/folder/tag conflicts.
- The app would need `NSFileCoordinator` and file presenter coverage for every package read/write.
- Cache files could trigger unnecessary sync churn.
- iOS would need to open and edit the same package structure, including database migrations and package conflict resolution.

Verdict: reject for production read-write sync. It may be acceptable only as an explicit "open an iCloud Drive document" import/export path.

### Option 2: Directly switch current Core Data store to `NSPersistentCloudKitContainer`

This looks attractive because the project already uses Core Data, but it is not a clean fit.

Problems:

- The current model uses uniqueness constraints; Apple's Core Data + CloudKit mirroring does not support unique constraints.
- The existing local package mixes authoritative data and file/cache layout. Mirroring metadata does not solve original-file sync.
- The app needs per-library storage-mode control. CloudKit mirroring is store-level and would require a separate compatible cloud store anyway.
- Cloud conflict behavior would be mostly owned by Core Data, while this product needs explicit semantics for import deduplication, folder membership, trash, restore, and blob cleanup.

Verdict: reject as the primary plan. It can be reconsidered only for a separate, simplified cloud-cache store after a model redesign, but custom CloudKit records are clearer for this domain.

### Option 3: Custom CloudKit records with CKSyncEngine and local cache

This is the recommended option.

Benefits:

- Local and cloud libraries can coexist cleanly.
- Cloud libraries work on both Mac and iOS.
- Original files are immutable and deduplicated within a cloud library.
- Metadata changes are semantic records, not file conflicts.
- The existing UI can continue consuming value types from a local store.
- Sync state, retries, offline writes, and conflict rules are explicit.

Costs:

- More initial design work than iCloud Drive package sync.
- Requires durable local dirty/error state and conflict handling.
- Requires CloudKit entitlements, schema deployment, telemetry, and real multi-device tests.

Verdict: use this.

## First Shippable Scope

The first cloud-library release should be intentionally narrow:

Included:

- Per-library storage mode: local or cloud.
- Cloud library creation, discovery, open, rename, and tombstone delete.
- Mac and iOS import of common image formats that pass a documented cloud upload eligibility check.
- Metadata sync for display name, favorite, note, trash, folder structure, tags, and memberships.
- Local thumbnail generation and optional small uploaded thumbnail for remote browsing.
- On-demand original download for preview/export/drag.
- Visible sync state and explicit upload/download failure states.

Deferred:

- Sharing a library with another Apple ID.
- Cross-library blob dedupe.
- Chunked upload protocol for very large files.
- CRDT-level merge logic.
- RAW/video/PDF cloud support beyond whatever the large-file spike explicitly validates.
- iCloud Drive package editing as a cloud-library backend.
- Automatic conversion of very large existing libraries before scale testing proves the path.

This boundary keeps the design aligned with the user's concrete goal: some libraries sync across the user's own Mac/iOS devices, while some remain local-only.

## Recommended Architecture

### Library Registry

Add a device-local library registry that can represent both modes:

```text
LibraryDescriptor
- id
- displayName
- storageMode: local | cloud
- localPackageBookmarkData?     // local only
- cloudLibraryID?               // cloud only
- lastOpenedAt
- lastKnownSyncState
```

Rules:

- Local libraries are only registered on the Mac that created/opened them.
- Cloud libraries are discovered from CloudKit and cached locally on Mac/iOS.
- The library picker should visually separate "On This Mac" and "iCloud" libraries.
- Creating a library must ask for storage mode up front. Converting a local library to cloud should be an explicit "copy to iCloud" operation, not an in-place mutation.

### CloudKit Zones

Use the user's private CloudKit database. This matches the requested behavior: the same Apple ID sees the same cloud libraries on Mac and iOS.

Use a small two-level zone model:

```text
zone: MomentoCatalog
- CloudLibrary records only

zone: MomentoLibrary-<libraryID>
- CloudAsset
- CloudAssetBlob
- CloudFolder
- CloudTag
- CloudFolderMembership
- CloudTagMembership
```

Reasons:

- `MomentoCatalog` is small and lets devices discover cloud libraries without syncing every asset record.
- Each library zone is the sync unit for that library's assets, folders, tags, blobs, and memberships.
- Opening a cloud library can start syncing its library zone; listing libraries only needs the catalog zone.
- Deleting a cloud library first tombstones its `CloudLibrary` record, then deletes the per-library zone after the delete grace period and local export/recovery windows have passed.
- Per-library zones also keep the future sharing path open without implementing sharing in v1.
- This adds more zone setup than a single-zone model, but it is aligned with the product's library boundary and avoids syncing all libraries as one undifferentiated dataset.

### Local Cache

Cloud libraries still need a local store:

```text
Application Support/Momento/CloudLibraries/<libraryID>/
├── cache.sqlite
├── assets/<hashPrefix>/<sha256>.<ext>       # downloaded originals only
└── thumbnails/<sha256>.png                  # downloaded/generated cache
```

This cache is not a user-facing `.momento` package. It is an implementation detail for offline support and fast UI.

The CKSyncEngine serialized state is not stored per library. It belongs to the private CloudKit database for the signed-in account:

```text
Application Support/Momento/CloudSync/<accountIdentity>/private-database-engine-state
```

Rules:

- UI reads local cache through the same value-model boundary as local packages.
- Writes first apply to local cache in a transaction, then register the affected CloudKit record IDs with `CKSyncEngine`.
- A failed upload must leave a visible pending/error state. No silent success.
- Cached originals can be evicted later, but metadata and thumbnails needed for browsing should remain.
- Each cached record stores the latest known CloudKit record change tag needed for conflict detection.
- Do not introduce a separate general-purpose operation log in the first version. Start with per-record dirty/error state plus persisted CKSyncEngine state. Add a semantic operation table only if a real recovery case cannot be represented by dirty records and tombstones.

### Local Cache Schema

The cloud cache uses a dedicated Core Data model `MomentoCloudModel.xcdatamodeld`, separate from the existing local-package `MomentoModel`. This separation is required because:

- CloudKit-mirrored data cannot use Core Data uniqueness constraints (Apple documents this restriction for `NSPersistentCloudKitContainer`-compatible models). The current `MomentoModel` relies on `(libraryID, contentHash)` uniqueness; keeping it for local packages and using a constraint-free schema for the cloud cache avoids destabilizing the local path.
- The cloud cache needs additional columns (availability, dirty/error state, record change tag, system field blob) that do not exist on the local-package model. Adding them to a shared schema would force migrations on existing local libraries for no reason.

Cloud cache entities mirror cloud records one-to-one, plus per-record sync state. Indexed fields are noted; nothing has a uniqueness constraint.

```text
CachedCloudLibrary
- libraryID (indexed)
- displayName, zoneName, createdAt, updatedAt, deletedAt?, schemaVersion
- syncState: synced | syncing | error
- lastError?

CachedCloudAsset
- libraryID (indexed)
- assetID = contentHash (indexed, NOT a constraint; dedupe enforced at insert time)
- displayName, originalFileName, fileExtension, utiIdentifier, kind
- byteSize, pixelWidth?, pixelHeight?, orientation?, colorProfileName?
- sourcePageURL?, note?, isFavorite, isTrashed, trashedAt?
- importedAt, updatedAt, deletedAt?
- ckRecordChangeTag?         // refreshed after every successful save
- ckSystemFieldsBlob?        // encoded CKRecord system fields
- isDirty                    // set on local edit, cleared on CKSyncEngine confirmation
- dirtyFields                // comma-joined field names since last sync (for LWW merge)
- lastError?
- originalAvailability       // local | downloading | remoteOnly | uploadPending | uploadFailed | missing
- thumbnailAvailability      // local | downloading | remoteOnly | generationPending | failed | missing

CachedCloudAssetBlob
- libraryID, contentHash (composite index)
- byteSize, fileExtension, utiIdentifier
- originalLocalPath?         // assets/<prefix>/<hash>.<ext> when downloaded
- thumbnailLocalPath?
- createdAt, lastReferencedAt
- ckRecordChangeTag?, ckSystemFieldsBlob?, isDirty, lastError?

CachedCloudFolder / CachedCloudTag
- mirror the cloud schema fields exactly
- plus: ckRecordChangeTag?, ckSystemFieldsBlob?, isDirty, dirtyFields, lastError?

CachedCloudFolderMembership / CachedCloudTagMembership
- mirror the cloud schema fields exactly
- plus: ckRecordChangeTag?, ckSystemFieldsBlob?, isDirty, lastError?
- no dirtyFields column: memberships are insert-or-tombstone, no scalar merge
```

Rules:

- `ckSystemFieldsBlob` stores `CKRecord.encodeSystemFields(with:)` output; CKSyncEngine requires it to reconstruct records after app relaunch without refetching from the server.
- `ckRecordChangeTag` is read for `.ifServerRecordUnchanged` save policy on every save (see "LWW Merge Algorithm").
- `isDirty` plus `dirtyFields` are read by both the UI (to show pending state) and the sync code (to know what to merge on conflict).
- `originalAvailability` / `thumbnailAvailability` are persisted (not derived) so UI does not need to re-stat the filesystem on every list refresh.
- Deletions are tombstones (`deletedAt`), not row removals; GC is co-scheduled with the corresponding cloud record GC after the grace period.

UI continues to read value-type `AssetItem`s through a cloud-aware reader (see "Local Model Adjustments for Cloud Mode" below).

### Cloud Record Schema

Use explicit CloudKit records rather than mirrored Core Data entities.

```text
CloudLibrary
- id
- displayName
- libraryZoneName
- createdAt
- updatedAt
- deletedAt?
- schemaVersion

CloudAsset
- id                         // deterministic: contentHash for dedupe within library
- libraryID
- contentHash
- displayName
- originalFileName
- fileExtension
- utiIdentifier
- kind
- byteSize
- pixelWidth?
- pixelHeight?
- orientation?
- colorProfileName?
- sourcePageURL?
- note?
- isFavorite
- isTrashed
- trashedAt?
- importedAt
- updatedAt
- deletedAt?

CloudAssetBlob
- libraryID
- contentHash
- byteSize
- fileExtension
- utiIdentifier
- originalAsset: CKAsset
- thumbnailAsset: CKAsset?
- createdAt
- lastReferencedAt

CloudFolder
- id
- libraryID
- name
- parentID?
- sortIndex
- createdAt
- updatedAt
- deletedAt?

CloudTag
- id
- libraryID
- name
- normalizedName
- colorHex?
- createdAt
- updatedAt
- deletedAt?

CloudFolderMembership
- id                         // deterministic: libraryID + assetID + folderID
- libraryID
- assetID
- folderID
- createdAt
- deletedAt?

CloudTagMembership
- id                         // deterministic: libraryID + assetID + tagID
- libraryID
- assetID
- tagID
- createdAt
- deletedAt?
```

Design choices:

- Record names should be deterministic, ASCII-only, and no longer than 255 characters. Use user-independent identifiers only, never user-visible names. If a composed ID risks exceeding the limit, hash the composed key.
- Record names should include `libraryID` where needed, for example `library:<uuid>` in `MomentoCatalog`, and `asset:<contentHash>`, `blob:<contentHash>`, and `folder-membership:<hash(assetID,folderID)>` inside a per-library zone.
- Do not use `CKReference` across zones. Store `libraryID`, `assetID`, `folderID`, and `tagID` as scalar fields because CloudKit references must stay in the same zone.
- Do not store large blobs on `CloudAsset`. Metadata updates should not require touching the original file.
- Use membership records instead of arrays on `CloudAsset`; this reduces conflicts and avoids large-array update problems.
- Keep `deletedAt` tombstones for sync correctness. Hard deletion can be delayed.
- Keep color analysis local initially. Palette colors can be added later if needed, but they are derived and should not block first sync.
- Define queryable/sortable CloudKit indexes before production promotion for fields used in discovery and sync-related queries, especially `libraryID`, `deletedAt`, `updatedAt`, `contentHash`, and normalized tag/folder names where queried.
- Deduplicate blobs only within one cloud library in the first version. Cross-library blob dedupe would complicate ownership, delete rules, and future sharing, and it is not required for the Mac/iOS sync goal.

### Schema Forward Compatibility

CloudKit production schemas can only evolve forward (see Official Documentation Findings). Multiple Momento versions will coexist on the user's devices; the schema rules below ensure an older client meeting a newer record does not corrupt or drop data.

- **Unknown record types**: render as inert placeholder in lists ("This asset requires a newer Momento") and never apply local mutations. CloudKit retains unknown record types as-is on the server.
- **Unknown fields on known record types**: CloudKit preserves unknown fields automatically **only when the save policy is `.ifServerRecordUnchanged`** and the client merges into the server record. Save policies `.allKeys` and `.changedKeys` discard unknown fields. The implementation must use `.ifServerRecordUnchanged` for all saves; this is a hard rule, not a default. See "LWW Merge Algorithm" for the exact merge procedure.
- **Per-library schema version**: the `schemaVersion` field on `CloudLibrary` is the per-library schema. If `library.schemaVersion > clientMaxSupportedSchemaVersion`, the library is read-only on this client with a UI warning ("Open <Library> on a newer Momento to edit").
- **Field deprecation**: removing a field from the cloud production schema is not allowed by CloudKit. Deprecate semantically (stop writing, document the move) but keep the field readable.
- **Index changes**: adding queryable/sortable indexes is allowed and must go through the Phase 2 schema/index checklist before production promotion.
- **Breaking changes**: would require a new container identifier and explicit data migration; treat this as a v2-major version, not a routine schema change.

### Local Model Adjustments for Cloud Mode

The current UI-facing `AssetItem` has `storageURL` and optional `thumbnailURL`, which works for local packages because the original file exists inside the opened `.momento` package. Cloud metadata can arrive before the original file is downloaded, so cloud mode needs an explicit availability layer.

Add a cloud-cache-only availability model:

```text
AssetFileAvailability
- original: local | downloading | remoteOnly | uploadPending | uploadFailed | missing
- thumbnail: local | downloading | remoteOnly | generationPending | failed | missing
- lastError?
```

Rules:

- Do not fake a local `storageURL` for a remote-only original.
- Preview/export/drag must request original availability and trigger download if needed.
- Grid/list UI can render from local thumbnail first, remote thumbnail second, and placeholder only when neither is available.
- Existing local-library code can keep assuming package-local files; cloud mode should route file access through a cloud-aware resolver.

### Blob Storage Policy

Default cloud blob storage uses `CKAsset` because it is Apple's CloudKit binary-file mechanism.

However, large image workflows need an explicit guardrail:

- Before shipping cloud import, run a spike with realistic images, GIFs, and RAW files to verify native CloudKit upload behavior, limits, latency, retry behavior, and quota impact.
- If a file fails local upload eligibility checks, reject it before creating a normal cloud asset.
- If CloudKit rejects an otherwise eligible upload after local import, keep the asset in `uploadFailed` state and surface the failure. Do not create a fake synced asset.
- The first production version may restrict cloud libraries to common image formats and a documented maximum file size.
- Do not implement chunked uploads until there is a proven requirement. Chunking would add substantial complexity to dedupe, retry, delete, and download behavior.

This is a deliberate correction to the earlier simple "store every original as CKAsset" idea. CKAsset is still the default, but upload eligibility must be enforced.

#### CKAsset Lifecycle Rules

CKAsset file handling has narrow contracts. The implementation must enforce all of the following:

**Upload**

- `CKAsset(fileURL:)` requires the file to remain readable at the supplied URL until `CKModifyRecordsOperation` completes. Point `CKAsset` at the canonical `assets/<prefix>/<hash>.<ext>` path inside the cloud cache, not a temporary copy that may be deleted concurrently.
- If a `CachedCloudAssetBlob` row is deleted or replaced mid-upload, the upload must either complete or fail atomically before the underlying file is removed. Wrap delete with a check on `isDirty` / in-flight upload state.

**Download**

- CloudKit downloads stage asset files in a system-managed temporary location. The `fileURL` returned in `recordChanged` events is **not persistent across app launches** and may be cleaned up by the system at any time.
- Synchronously copy downloaded `CKAsset` files into `assets/<prefix>/<hash>.<ext>` inside the cloud cache during the same `recordChanged` callback, before signalling `originalAvailability = local`.
- After copy, recompute SHA-256 and verify it matches the `contentHash` field on the cloud record. CloudKit does not guarantee binary integrity end-to-end; a mismatched download must be discarded and re-fetched.

**Failure handling**

- Hash mismatch on download: discard the file, keep `originalAvailability = downloading` with a `lastError`, schedule retry with backoff.
- Download timeout: keep `originalAvailability = remoteOnly`; surface the failure only if the user explicitly triggered the download (preview/export/drag).
- Upload failure after the local file is created: set `originalAvailability = uploadFailed` with a `lastError`. Do not silently retry — wait for next CKSyncEngine send event or user-triggered retry.

### Scale Policy

Momento's local UI is designed for large libraries, but cloud sync must prove its own scale separately.

Rules:

- Do not promise 100k-asset cloud libraries until initial sync, incremental sync, local cache size, iOS memory, and CloudKit operation behavior are measured.
- "Copy Library to iCloud" must be resumable and cancellable before it is exposed for non-trivial libraries.
- The first cloud release may set conservative limits for cloud-library count, per-library asset count, total byte size, and per-file byte size. These limits should be documented in the UI and adjusted only after measurement.
- Cloud browsing should support metadata pagination/incremental loading from the local cache. Do not block opening a cloud library until every original has downloaded.

### Entitlements & CloudKit Container

CloudKit access requires explicit entitlements on both macOS and iOS app targets. The current `Momento.entitlements` only has sandbox/network keys; cloud mode is blocked until the following are wired up. This is a Phase 1 prerequisite.

CloudKit container:

- Container identifier: `iCloud.com.seaony.Momento` (single container shared by Mac + iOS).
- Configured in Xcode → Signing & Capabilities → iCloud → CloudKit, and managed in the CloudKit Console.
- Container schema must be promoted from Development to Production before public release; this is a one-way operation and a hard release gate.

Required entitlements (both macOS and iOS targets):

```xml
<key>com.apple.developer.icloud-services</key>
<array>
    <string>CloudKit</string>
</array>
<key>com.apple.developer.icloud-container-identifiers</key>
<array>
    <string>iCloud.com.seaony.Momento</string>
</array>
<key>aps-environment</key>
<string>development</string>
```

For release builds the `aps-environment` value becomes `production`. This is done at build configuration level, not at runtime.

Sandbox interaction:

- macOS keeps `com.apple.security.app-sandbox`. CloudKit access is compatible with App Sandbox; no additional exception is needed.
- `com.apple.security.network.client` is already present and is required for CloudKit traffic.

Provisioning:

- The CloudKit container must be added to both macOS and iOS app IDs in the developer portal.
- Provisioning profile regeneration is required whenever the entitlements change.

Background remote notifications:

- Required for CKSyncEngine to receive push-driven sync. See "Remote Notifications Setup" below.

### Sync Engine

Deployment targets are macOS 26 and iOS 26, both confirmed before Phase 0. `CKSyncEngine` is the sole sync transport; no manual `CKFetchRecordZoneChangesOperation`+token fallback is implemented.

Responsibilities:

- Initialize the sync engine early for the private database, after `ubiquityIdentityToken` validation (see "Account Identity Binding").
- Use one sync engine for the user's private database in production. Do not create one engine per library zone.
- Persist sync-engine state at the account/private-database level, not inside an individual library cache.
- Ensure `MomentoCatalog` and any opened library zones exist before writing records.
- Use CKSyncEngine pending record-zone changes as the CloudKit upload queue.
- Track local dirty/error state per affected record so UI can explain pending work and failures.
- Map local changes to CloudKit records in `nextRecordZoneChangeBatch`.
- Apply remote record changes into local cache.
- Handle account changes, zone deletion, partial failures, retryable errors, non-retryable errors, and server conflicts.
- Provide observable sync state for UI.

If a future deployment-target reduction ever makes CKSyncEngine unavailable, that would be a separate v2 design — do not stub the manual transport into v1 "just in case".

### Remote Notifications Setup

CKSyncEngine relies on silent remote notifications to drive incremental sync. This is a Phase 2 prerequisite.

Setup:

- `aps-environment` entitlement (see "Entitlements & CloudKit Container").
- macOS: call `NSApplication.shared.registerForRemoteNotifications()` after app launch. Implement `application(_:didRegisterForRemoteNotificationsWithDeviceToken:)` (CloudKit does not require the token to be forwarded anywhere, but registration must complete).
- iOS: same registration via `UIApplication`, plus `UIBackgroundModes` in `Info.plist` containing `remote-notification`.
- Both targets: implement `application(_:didReceiveRemoteNotification:fetchCompletionHandler:)` and forward the payload to CKSyncEngine via its standard push handler.

Subscriptions:

- CKSyncEngine creates and maintains database/zone subscriptions automatically when sync engine state is persisted. Do not create `CKQuerySubscription` or `CKRecordZoneSubscription` manually in v1.
- If CKSyncEngine reports a missing subscription error, treat it as a recoverable initialization step, not a hard failure — let the engine recreate it.

User permission:

- Silent CloudKit pushes do not require the user-facing notification permission. The app must not prompt for notification permission solely for sync purposes.
- If the app later adds user-visible notifications, that requires a separate permission flow.

## Mutation Flow

All user writes should use the same command pattern on Mac and iOS.

### Import Image

1. User chooses image from Files/Photos/Finder.
2. App copies it into local cache.
3. App computes SHA-256.
4. App creates or reuses local `Asset` by `contentHash`.
5. App creates thumbnail locally.
6. App writes metadata/blob state in one local transaction.
7. App marks `CloudAsset` dirty and, if missing remotely, marks `CloudAssetBlob` dirty.
8. UI shows the asset immediately with sync status.
9. CKSyncEngine asks for pending records and uploads metadata/blob records.
10. Other devices receive metadata first, then download thumbnail/original on demand.

### Edit Metadata

Examples: rename asset, edit note, favorite/unfavorite, tag changes.

Rules:

- Apply locally first.
- Mark the affected record dirty and let CKSyncEngine request the CloudKit record save.
- Use CloudKit record metadata / change tags to detect server conflicts.
- Do not use device-local `updatedAt` as the authority for conflict detection. It is useful for UI and local ordering, but device clocks can skew.
- For v1 scalar conflicts, use server-versioned last-writer-wins: save with the latest known record change tag, handle `serverRecordChanged`, merge the intended scalar changes into the server record, and retry. The winning version is the last write CloudKit accepts, not the largest client timestamp.
- Keep membership changes as independent records so adding a tag on one device does not overwrite another tag added elsewhere.

### Folder Operations

Folder create, rename, move, and delete are record-level operations.

Rules:

- Folder rename: server-versioned last-writer-wins for `name`.
- Folder move: server-versioned last-writer-wins for `parentID` and `sortIndex`.
- Folder delete: write `deletedAt`, remove/hide memberships pointing to that folder locally.
- If a remote membership arrives for a deleted/missing folder, keep the membership tombstoned or ignore it. Do not resurrect the folder.

### Trash and Permanent Delete

Trash is a reversible metadata state. Permanent delete is delayed.

Rules:

- Move to trash: set `isTrashed = true`, `trashedAt`, `updatedAt`.
- Restore: set `isTrashed = false` only if `deletedAt` is nil.
- Permanent delete: set `deletedAt` tombstone on `CloudAsset` and memberships.
- Blob cleanup: delete/orphan `CloudAssetBlob` only after no non-deleted `CloudAsset` references the same `contentHash`, and only after a grace period.

This avoids a counter case where one device deletes a blob while another device has not yet downloaded the metadata change.

## Conflict Policy

The first version should use simple, explicit conflict rules:

| Area | Rule |
|---|---|
| Asset original file | Immutable by `contentHash`; no merge needed |
| Duplicate import | Deterministic asset/blob record IDs collapse duplicates |
| Display name / note / favorite | Server-versioned last writer wins through CloudKit record change tags |
| Tags/folder membership | Independent membership tombstone records |
| Folder rename/move | Server-versioned last writer wins |
| Trash vs restore | Server-versioned last writer wins while `deletedAt` is nil |
| Permanent delete | `deletedAt` wins over restore |
| Missing blob after metadata arrives | Show placeholder and retry download/upload repair |
| Account signed out/switched | Disable cloud writes; protect cached data by account identity |

More advanced CRDT-style merging is not justified for the first implementation. It would be over-designed for this product stage and would complicate debugging. The design instead chooses smaller records and clear tombstones.

### LWW Merge Algorithm

Server-versioned last-writer-wins is implemented through CloudKit's `.ifServerRecordUnchanged` save policy plus field-level merging. The algorithm is identical for every scalar conflict (display name, note, favorite, folder rename/move, trash flag):

```text
saveDirtyRecord(localRecord):
  for attempt in 1...3:
    ckRecord = localRecord.toCKRecord(systemFieldsBlob: localRecord.ckSystemFieldsBlob)
    operation = CKModifyRecordsOperation(recordsToSave: [ckRecord])
    operation.savePolicy = .ifServerRecordUnchanged

    on success(savedRecord):
      localRecord.ckRecordChangeTag = savedRecord.recordChangeTag
      localRecord.ckSystemFieldsBlob = encode(savedRecord)
      localRecord.isDirty = false
      localRecord.dirtyFields = []
      localRecord.lastError = nil
      return

    on CKError.serverRecordChanged(serverRecord, _):
      # take server as base, apply only locally-dirty fields
      merged = serverRecord.copy()
      for field in localRecord.dirtyFields:
        merged[field] = localRecord[field]
      ckRecord = merged
      # loop continues with refreshed change tag

    on retryable error (network, throttle, zoneBusy):
      schedule retry after CKError.retryAfterSeconds (or default backoff)
      return  # CKSyncEngine will re-drive

    on non-retryable error (quotaExceeded, badContainer, permissionFailure):
      localRecord.isDirty = true  (keep)
      localRecord.lastError = error
      surface in UI
      return

  # exhausted attempts
  localRecord.lastError = "merge conflict, please retry"
  surface in UI for manual intervention
```

Required invariants:

- **Save policy is always `.ifServerRecordUnchanged`.** Never use `.allKeys` or `.changedKeys`; those discard unknown fields and break schema forward compatibility.
- **Field-level dirty tracking is required.** A record cannot just save "everything I have" because that clobbers server fields the client never observed. Each `CachedCloud*` entity stores `dirtyFields` and clears it on success.
- **Unknown fields on the server record pass through unchanged** because the merge only writes locally-dirty fields and leaves the rest as the server returned them.
- **Membership records skip this algorithm.** `CloudFolderMembership` / `CloudTagMembership` are insert-with-tombstone, so concurrent inserts/deletes commute naturally — no field merging needed.

## iCloud Account and Offline Behavior

### Account Identity Binding

Cloud cache and CKSyncEngine state must be bound to the signed-in iCloud account so that switching accounts cannot mix identities. The binding key is `FileManager.default.ubiquityIdentityToken`:

- The token is an opaque `NSCoding` value that uniquely identifies the iCloud account on the device.
- Serialize via `NSKeyedArchiver` to `Application Support/Momento/CloudSync/account-identity.bin`.
- On every app launch and on `NSUbiquityIdentityDidChangeNotification`, compare the current token against the persisted one.
- `CKContainer.userRecordID()` is **not** used as the binding key — it is async, container-scoped, slower than the system token, and unavailable before CloudKit network calls succeed.

State transitions:

| Persisted token | Current token | Behavior |
|---|---|---|
| absent | present | First sign-in. Persist token, create cloud cache directory and sync engine state. |
| present, A | present, A | Continue as normal. |
| present, A | present, B | Account switched. Block cloud writes immediately. Surface "iCloud account changed" UI with two options: "Sign back in to <previous>" (read-only access to existing cache until restored) or "Reset cloud cache" (destructive, clearly labeled). |
| present, A | absent | Signed out. Block cloud writes. Keep cloud cache read-only and visible. Restore on re-sign-in if token matches. |

`NSUbiquityIdentityDidChangeNotification` must be observed for the entire app lifetime, not just at launch. Mid-session account changes must trigger the same block.

### Connectivity and Quota Cases

Cases:

- No iCloud account: cloud libraries cannot be created or synced.
- Same iCloud account, offline network: allow local cloud-library writes and keep CKSyncEngine pending changes until network returns.
- Account temporarily unavailable: keep cached cloud libraries visible, but show sync blocked state.
- Account switched: do not open previous account's cloud cache as writable. See Account Identity Binding table above.
- iCloud quota exceeded: keep local changes pending, show quota/upload failure, and do not claim sync success. Note: CloudKit does not provide a quota query API; quota state is detected only on `CKError.quotaExceeded` from a write attempt.

The UI needs at least these sync states:

- Synced
- Syncing
- Waiting for network
- Waiting for iCloud sign-in
- Upload failed
- Download failed
- Quota/storage blocked
- Conflict resolved

Retry policy:

- Retry network and service-throttle failures using CloudKit-provided retry timing when available.
- Split oversized record batches on `limitExceeded`.
- Treat missing entitlements, bad container/database, unsupported file type, and account mismatch as blocking errors that require user or build/configuration action.
- Never mark a record as synced until the relevant CKSyncEngine send event confirms success.

## Migration and Conversion

Do not mutate a local `.momento` package into a cloud library in place.

Supported first path:

1. User chooses "Copy Library to iCloud".
2. App opens the local package.
3. App creates a new cloud library with a new `libraryID`.
4. App creates the `MomentoLibrary-<libraryID>` zone.
5. App imports local assets into the cloud cache using existing content hashes.
6. App uploads metadata and blobs.
7. The original local library remains unchanged.

Why:

- Avoids destroying the user's local-only source.
- Avoids changing security-scoped bookmarks.
- Makes failures recoverable.
- Keeps local and cloud lifecycle rules separate.

Export remains the inverse: cloud library can be exported to a local `.momento` package after all required originals are downloaded. Export is Mac-only in v1, matching local library access scope.

## Implementation Phasing

### Phase 0: Technical Spikes and Release Gates

Pre-Phase-0 decisions (already made, recorded here for traceability):

- macOS deployment target: 26.
- iOS deployment target: 26.
- CKSyncEngine is the only sync transport.
- iOS v1 consumes cloud libraries only; iOS does not implement `.momento` package opening.
- Account identity binding key: `FileManager.default.ubiquityIdentityToken`.
- CloudKit container identifier: `iCloud.com.seaony.Momento`.

Phase 0 spikes:

- Wire up CloudKit entitlements, container, remote notifications, and provisioning on both macOS and iOS targets; verify a smoke-test record save/load round-trips between the two.
- Validate catalog-zone plus per-library-zone creation, deletion, and sync behavior.
- Validate one private-database sync engine handling catalog and per-library zones.
- Validate `ubiquityIdentityToken` binding flow: sign in, sign out, account switch, mid-session change via `NSUbiquityIdentityDidChangeNotification`.
- Measure CKAsset upload/download behavior with realistic file sizes and failure modes (including CKAsset re-hash verification after download).
- Measure initial sync and local cache behavior with large synthetic libraries.
- Measure multi-library zone behavior (not just one big library) — see Finding 18.
- Define first-release cloud limits for file type, file size, cloud-library count, per-library asset count, and total cloud bytes. Recommended starting baselines (revise after spike): ≤ 5k assets per library, ≤ 100 MB per file, ≤ 20 cloud libraries per account.
- Define CloudKit schema/index checklist before production promotion.

### Phase 1: Storage Mode Foundation

- Add storage-mode concept to library descriptors.
- Keep existing local package behavior unchanged.
- Add cloud library registry/listing with empty cloud libraries.
- Add iCloud capability checks and account-state UI.

### Phase 2: Cloud Metadata Sync

- Add CloudKit schema, catalog zone, and per-library zone creation.
- Add local cloud cache and account-level sync-engine state persistence.
- Sync libraries, folders, tags, and asset metadata without original download beyond thumbnails.
- Add deterministic record IDs and tombstones.
- Persist CloudKit record change tags in the local cloud cache and use them for conflict detection.
- Add CloudKit schema/index checklist and a development-to-production promotion gate.

### Phase 3: Blob Upload/Download

- Add import into cloud libraries.
- Upload original `CKAsset` and thumbnail.
- Add on-demand original download.
- Add blob repair and upload failure states.
- Run large-file spike before enabling broad file support.

### Phase 4: Full Read-Write Cross-Device Behavior

- Implement Mac/iOS write commands for rename, note, favorite, tags, folders, trash, restore, and permanent delete.
- Add conflict handling and visible sync status.
- Add multi-device integration tests with separate local caches.

## Validation Plan

Minimum validation for the design implementation later:

- Spike evidence for CKSyncEngine availability, CKAsset file behavior, and large-library scale before enabling cloud import broadly.
- Unit tests for storage-mode registry: local libraries stay local; cloud libraries are discovered from cloud cache.
- Unit tests for deterministic record IDs and dedupe.
- Unit tests for zone naming and library-zone lifecycle.
- Unit tests for tombstone rules: delete vs restore, folder delete vs membership arrival, blob GC grace period.
- Unit tests for dirty-state retry and partial failure handling.
- Unit tests for record-name generation: ASCII-only, deterministic, under 255 characters.
- Integration tests using CloudKit test environment or protocol-backed fake CloudKit service.
- Manual tests on two devices with the same Apple ID: Mac import, iOS import, concurrent rename, concurrent folder move, offline edits, account sign-out, quota/upload failure.
- `git diff --check`, targeted tests, and build/typecheck for each implementation phase.

## Deep Review Findings and Fixes

This section records the self-review pass requested before implementation.

### Finding 1: Direct iCloud package sync is simpler but unsafe

Counter case:

- Mac and iOS both edit `database/library.sqlite` while iCloud Drive is syncing the package. iCloud sees file versions, not asset-level operations. A conflict can preserve one SQLite version while losing the other device's semantic edits.

Fix in design:

- Reject package sync for cloud libraries.
- Keep `.momento` as local/export/import format only.

### Finding 2: Direct `NSPersistentCloudKitContainer` mirroring is tempting but mismatched

Counter case:

- Current local Core Data model depends on uniqueness constraints for dedupe. Apple's Core Data + CloudKit mirroring does not support unique constraints. Removing those constraints just for CloudKit would weaken local data safety unless a separate model is introduced.

Fix in design:

- Use custom CloudKit records for cloud libraries.
- Keep local package model unchanged.

### Finding 3: "Everything as CKAsset" ignores large-file and quota behavior

Counter case:

- User imports a very large GIF or RAW file. Upload fails due to CloudKit limits, account quota, or throttling. If the app still marks the asset as synced, iOS sees metadata but can never download the original.

Fix in design:

- Add explicit blob upload eligibility checks.
- Add upload failure states.
- Require a large-file spike before broad cloud-file support.
- Avoid chunked upload until proven necessary.

### Finding 4: Immediate hard delete can corrupt other devices' view

Counter case:

- Device A permanently deletes an asset and removes the blob. Device B is offline and later uploads a membership or metadata edit for the same asset.

Fix in design:

- Use `deletedAt` tombstones.
- Garbage collect blobs only when no non-deleted asset references the content hash and after a grace period.

### Finding 5: Folder/tag arrays would create unnecessary conflicts

Counter case:

- Mac adds asset X to folder A while iOS adds the same asset to folder B. If folder membership is stored as one array on `CloudAsset`, one save can overwrite the other.

Fix in design:

- Use independent `CloudFolderMembership` and `CloudTagMembership` records.

### Finding 6: A single CloudKit zone conflicts with the library boundary

Counter case:

- The user has several cloud libraries but only opens one on iOS. With one zone, the device has to process one large mixed change stream for every library's assets, folders, tags, and blobs.

Fix in design:

- Use a small `MomentoCatalog` zone for library discovery.
- Use one per-library custom zone for each cloud library's records.
- Keep sharing deferred, but do not choose a zone layout that makes future library sharing or selective sync harder.

### Finding 7: Offline-first writes can hide sync failures

Counter case:

- iOS imports an image offline. The UI shows the asset normally. Later upload fails due to quota. Without a durable pending state, the user assumes the image is on Mac too.

Fix in design:

- Every local cloud-library write creates durable dirty/error state and a CKSyncEngine pending record change.
- UI must show sync status and failures.
- No silent success.

### Finding 8: Account switching can leak stale cache assumptions

Counter case:

- User signs out of iCloud account A and signs into account B. The app opens cached account-A cloud libraries and allows writes, mixing identities.

Fix in design:

- Bind cloud cache to iCloud account identity.
- Disable writes on account mismatch and require resync/cache reset.

### Finding 9: Current file URL assumptions do not hold for cloud metadata

Counter case:

- iOS receives a remote `CloudAsset` record, but the original `CKAsset` has not been downloaded yet. If the UI treats `storageURL` as a valid local file path, preview/export/drag can fail or silently show stale placeholders.

Fix in design:

- Add explicit file availability state for cloud libraries.
- Route preview/export/drag through a cloud-aware resolver instead of directly trusting `storageURL`.
- Keep local-package behavior unchanged.

### Finding 10: CloudKit schema/index planning is easy to defer too long

Counter case:

- Development succeeds with ad hoc records, but production schema promotion misses a queryable index for `libraryID` or `deletedAt`. Later clients cannot efficiently list cloud libraries or filter active records without a schema/index update cycle.

Fix in design:

- Add schema/index planning to the design.
- Treat production schema promotion as a release gate, not a background setup task.

### Finding 11: A custom operation log is unnecessary until proven

Counter case:

- The first implementation builds a domain-specific pending operation queue plus CKSyncEngine state. The two queues diverge after a partial failure, making it unclear which source owns retry decisions.

Fix in design:

- Use CKSyncEngine pending changes as the CloudKit upload queue.
- Keep only per-record dirty/error state for UI and recovery in the first version.
- Add a semantic operation table later only if dirty records and tombstones cannot represent a real failure mode.

### Finding 12: Cross-library blob dedupe is a hidden sharing/delete problem

Counter case:

- Two cloud libraries reference the same global blob. The user deletes one library, later adds sharing to the other, and blob ownership becomes ambiguous.

Fix in design:

- Deduplicate only within a cloud library for v1.
- Defer cross-library dedupe until sharing and storage pressure justify the added ownership model.

### Finding 13: Large local libraries can make a "correct" cloud design unusable

Counter case:

- A user copies a 100k-asset local library to iCloud. The app creates huge upload work, iOS receives a massive first sync, and the user cannot tell whether the process is progressing or stuck.

Fix in design:

- Add Phase 0 scale spikes.
- Require resumable/cancellable conversion before large-library copy is exposed.
- Allow conservative first-release cloud limits until measured behavior supports larger libraries.

### Finding 14: Creating metadata before eligibility checks can create broken cloud assets

Counter case:

- The user imports a file that is locally known to be outside the cloud-library limits. The app creates `CloudAsset` metadata anyway, so other devices see an item that will never upload.

Fix in design:

- Reject locally ineligible files before creating normal cloud assets.
- Reserve `uploadFailed` for files that pass local checks but fail during actual CloudKit upload.

### Finding 15: Client timestamps are not safe conflict authority

Counter case:

- Mac and iOS both edit the same note while offline, but one device clock is several minutes ahead. If the app blindly picks the larger `updatedAt`, the older user action can win just because the local clock is wrong.

Fix in design:

- Use CloudKit record change tags and `serverRecordChanged` as the conflict-detection mechanism.
- Treat v1 last-writer-wins as the last write accepted by CloudKit after merging into the server record, not the largest client timestamp.
- Keep `updatedAt` for UI/local ordering only.

### Finding 16: Per-library sync-engine state would split one database incorrectly

Counter case:

- The app opens two cloud libraries and creates one CKSyncEngine state file per library. Both engines target the same private database, so database changes, subscriptions, and pending zone changes can diverge or duplicate work.

Fix in design:

- Use one production CKSyncEngine for the private database.
- Store CKSyncEngine serialization under the signed-in account/private-database scope.
- Keep per-library caches for records, blobs, thumbnails, availability, dirty/error state, and record change tags only.

### Finding 17: Fallback sync can become accidental double implementation

Counter case:

- The project supports an iOS target where CKSyncEngine is available, but implementation still builds the manual operation/token fallback "just in case". The app now has two sync transports, two failure surfaces, and twice the test matrix before the first cloud release.

Fix in design:

- iOS 26 deployment target is confirmed pre-Phase-0; CKSyncEngine is the only transport.
- The manual `CKFetchRecordZoneChangesOperation`+token path is not implemented in v1.
- If a future minimum-OS reduction ever requires the manual path, treat it as a separate v2 design.

### Finding 18: Per-library zones need a library-count release gate

Counter case:

- A user creates many small cloud libraries. The per-library zone model remains semantically correct, but zone creation, change fetching, and local cache bookkeeping may become the bottleneck before asset count does.

Fix in design:

- Measure multi-library zone behavior in Phase 0, not just one large library.
- Define a first-release cloud-library count limit if measurement shows operational overhead.

## Final Recommendation

Build cloud libraries as a separate CloudKit-backed storage mode while preserving local `.momento` libraries unchanged.

Use CKSyncEngine plus explicit CloudKit records, a local cache, deterministic record IDs, membership records, tombstones, and visible sync states. This is more work than package sync, but it is the smallest design that correctly supports Mac/iOS read-write behavior without hiding data-loss risks.

Do not implement sharing, chunked uploads, CRDTs, a second manual sync transport, or package-level iCloud Drive sync in the first version. Those are either outside the stated requirement or unjustified until real usage proves the need.
