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
  - Source: [CKSyncEngineDelegate.nextRecordZoneChangeBatch](https://developer.apple.com/documentation/cloudkit/cksyncenginedelegate-1q7g8/nextrecordzonechangebatch%28_%3Asyncengine%3A%29)
  - Source: [CKSyncEngine.State.pendingRecordZoneChanges](https://developer.apple.com/documentation/cloudkit/cksyncenginestate/pendingrecordzonechanges)
  - Source: [Apple CKSyncEngine sample](https://github.com/apple/sample-cloudkit-sync-engine)
- Core Data with CloudKit only works if the model is compatible with CloudKit limitations. Apple documents that unique constraints are unsupported, relationships must be optional, and production CloudKit schemas need careful forward-compatibility planning.
  - Source: [Mirroring a Core Data store with CloudKit](https://developer.apple.com/documentation/CoreData/mirroring-a-core-data-store-with-cloudkit)
  - Source: [Creating a Core Data Model for CloudKit](https://developer.apple.com/documentation/CoreData/creating-a-core-data-model-for-cloudkit)
- iCloud Documents can sync document packages, but multi-device document writes require file coordinators/file presenters and conflict handling. Package syncing is a document workflow, not a good fit for a live multi-device SQLite package with caches.
  - Source: [iCloud File Management](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/iCloud/iCloud.html)
- iOS local cache files have backup and file-protection consequences. Re-downloadable support files should be excluded from backup, and files needed by background sync must use a protection class that remains accessible after first unlock.
  - Source: [NSURLIsExcludedFromBackupKey](https://developer.apple.com/documentation/foundation/urlresourcekey/isexcludedfrombackupkey)
  - Source: [FileProtectionType.completeUntilFirstUserAuthentication](https://developer.apple.com/documentation/foundation/fileprotectiontype/completeuntilfirstuserauthentication)

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
├── assets/<hashPrefix>/<sha256>.<ext>       # downloaded or upload-pending originals
└── thumbnails/<sha256>.png                  # downloaded/generated cache
```

This cache is not a user-facing `.momento` package. It is an implementation detail for offline support and fast UI.

The CKSyncEngine serialized state is not stored per library. It belongs to the private CloudKit database for the signed-in CloudKit account:

```text
Application Support/Momento/CloudSync/<cloudAccountID>/private-database-engine-state
```

Rules:

- UI reads local cache through the same value-model boundary as local packages.
- Writes first apply to local cache in a transaction, then register the affected CloudKit record IDs with `CKSyncEngine`.
- A failed upload must leave a visible pending/error state. No silent success.
- Cached originals can be evicted later, but metadata and thumbnails needed for browsing should remain.
- Each cached record stores the latest known CloudKit record change tag needed for conflict detection.
- Do not introduce a separate general-purpose operation log in the first version. Start with per-record dirty/error state plus persisted CKSyncEngine state. Add a semantic operation table only if a real recovery case cannot be represented by dirty records and tombstones.

### Cloud Cache Write Boundary

Cloud mode introduces background CloudKit callbacks, local UI commands, file copies, hashing, and thumbnail generation. These must not write the same Core Data rows or asset files from unrelated execution contexts.

Use one explicit cloud-cache write boundary, for example a `CloudLibraryRepository` actor or an equivalent serial executor. The name is not important; the ownership is.

Responsibilities:

- Own the cloud-cache Core Data context, cloud-cache file moves/copies, availability updates, dirty/error state, and CKRecord system-field persistence.
- Accept UI write commands from the `@MainActor` `LibraryStore`, apply them to the local cloud cache, then enqueue `CKSyncEngine` pending changes.
- Accept CKSyncEngine delegate events, materialize outgoing records from local state, and apply remote changes into the local cache.
- Publish value snapshots or an async change stream back to `LibraryStore`; do not mutate SwiftUI-observed UI state directly from CloudKit callbacks.
- Serialize file and database state transitions. For example, a CKAsset download must copy the staged file, verify hash, update `originalAvailability`, and persist record system fields through one write boundary.
- Keep long hashing, thumbnail generation, and file IO off the main actor. Only publish small value snapshots to UI.

This boundary is required before Phase 2 metadata sync. Without it, a remote record change, local edit, and asset download can race and leave `isDirty`, availability, and filesystem contents inconsistent.

### iOS Cache Durability, Backup, and File Protection

The cloud cache is not a disposable `Caches` directory. Metadata, pending writes, and upload-pending originals must survive app relaunch and low-storage cleanup.

Storage rules:

- Store `cache.sqlite`, CKSyncEngine state, dirty metadata, and upload-pending originals under Application Support, not `Library/Caches`.
- Do not mark upload-pending originals as excluded from backup until CloudKit confirms the upload. Before that confirmation, the local file may be the only durable copy of the user's import.
- After a blob is confirmed uploaded and can be re-downloaded from CloudKit, mark downloaded/synced originals and thumbnails with `NSURLIsExcludedFromBackupKey = true`. Set this attribute after every copy/write because common file operations can reset it.
- Thumbnails are always reconstructable and should be excluded from backup. They can still live in Application Support if the product wants stable offline browsing; exclusion from backup is separate from purgeability.
- Do not evict a local original while its `CloudAssetBlob` is dirty, uploadPending, uploadFailed, or referenced by a user-triggered export/drag/download operation.
- On iOS, use a file protection policy that allows background sync after first device unlock, such as `.completeUntilFirstUserAuthentication`, for the cloud cache database, sync-engine state, and sync-managed asset files. Do not use `.complete` for files that CKSyncEngine may need while the device is locked unless the product explicitly accepts locked-device sync blocking.
- After device restore from backup, revalidate account binding before opening the restored cloud cache as writable. If the CloudKit account does not match, use the account-mismatch flow.

### Local Cache Schema

The cloud cache should use a dedicated Core Data model such as `MomentoCloudModel.xcdatamodeld`, separate from the existing local-package `MomentoModel`. This separation is recommended because:

- The current `MomentoModel` relies on local-package uniqueness constraints such as `(libraryID, contentHash)`. Keeping it unchanged avoids destabilizing the local path. The cloud cache is not an `NSPersistentCloudKitContainer` mirrored store, so local-only constraints remain allowed when they protect cache integrity; CloudKit uniqueness must still come from deterministic record IDs.
- The cloud cache needs additional columns (availability, dirty/error state, record change tag, system field blob) that do not exist on the local-package model. Adding them to a shared schema would force migrations on existing local libraries for no reason.

Cloud cache entities mirror cloud records one-to-one, plus per-record sync state. Indexed fields are noted. Add local constraints only where they mirror deterministic CloudKit identity and have tests, for example `(libraryID, assetID)` on cached assets.

```text
CachedCloudLibrary
- libraryID (indexed)
- displayName, zoneName, createdAt, updatedAt, deletedAt?, schemaVersion
- syncState: synced | syncing | error
- lastError?

CachedCloudAsset
- libraryID (indexed)
- assetID = contentHash (indexed; local constraint allowed if it mirrors CloudKit record identity)
- displayName, originalFileName, fileExtension, utiIdentifier, kind
- byteSize, pixelWidth?, pixelHeight?, orientation?, colorProfileName?
- sourcePageURL?, note?, isFavorite, isTrashed, trashedAt?
- importedAt, updatedAt, deletedAt?
- ckRecordChangeTag?         // refreshed after every successful save
- ckSystemFieldsBlob?        // encoded CKRecord system fields
- isDirty                    // set on local edit, cleared on CKSyncEngine confirmation
- dirtyFields                // structured dirty field set for server-versioned merge
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
- plus: ckRecordChangeTag?, ckSystemFieldsBlob?, isDirty, pendingIntent: present | tombstoned, lastError?
- no dirtyFields column: membership conflict handling uses the intended membership state (`deletedAt == nil` or non-nil), not scalar field merging
```

Rules:

- `ckSystemFieldsBlob` stores `CKRecord.encodeSystemFields(with:)` output so records can be reconstructed with server metadata, including record ID and change tag, after app relaunch.
- `ckRecordChangeTag` is persisted for diagnostics and local conflict reasoning; the encoded system fields remain the authoritative source when reconstructing a `CKRecord`.
- `isDirty` plus structured dirty field tracking are read by both the UI (to show pending state) and the sync code (to know what to merge on conflict). The storage format can be a normalized table, bitset, or encoded field list; do not lock this to comma-joined strings until implementation.
- Membership rows use `pendingIntent` instead of `dirtyFields` because their only user-visible states are "present" and "tombstoned". They still require change-tag conflict handling when the same membership is concurrently added and removed on different devices.
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
- **Unknown fields on known record types**: preserve them by merging into the current server record and writing only locally dirty fields. Do not use a local stale record as the save base. If a manual `CKModifyRecordsOperation` is introduced for a spike, use `.ifServerRecordUnchanged`; `.changedKeys` bypasses change-tag comparison and `.allKeys` overwrites unchanged fields.
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
- After copy, recompute SHA-256 and verify it matches the `contentHash` field on the cloud record. This is Momento's application-level identity check for content-addressed blobs; a mismatch means the downloaded file does not match the referenced asset and must be discarded and re-fetched.

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

- Proposed container identifier: `iCloud.com.seaony.Momento` (single container shared by Mac + iOS). Verify the final value in Xcode and the Apple Developer portal before implementation.
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
    <string>iCloud.com.seaony.Momento</string> <!-- proposed; verify before use -->
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

The current macOS deployment target is 26. The iOS deployment target is not present in this repository yet and must be confirmed in Phase 0. Use `CKSyncEngine` as the preferred sync transport when the chosen iOS target supports it; do not implement a manual `CKFetchRecordZoneChangesOperation`+token fallback in parallel.

Responsibilities:

- Initialize the sync engine early for the private database after CloudKit account status and cache identity checks pass (see "Account Identity Binding").
- Use one sync engine for the user's private database in production. Do not create one engine per library zone.
- Persist sync-engine state at the account/private-database level, not inside an individual library cache.
- Ensure `MomentoCatalog` and any opened library zones exist before writing records.
- Use CKSyncEngine pending record-zone changes as the CloudKit upload queue.
- Track local dirty/error state per affected record so UI can explain pending work and failures.
- Map local changes to CloudKit records in `nextRecordZoneChangeBatch`.
- Apply remote record changes into local cache.
- Handle account changes, zone deletion, partial failures, retryable errors, non-retryable errors, and server conflicts.
- Provide observable sync state for UI.

If Phase 0 chooses an iOS target where CKSyncEngine is unavailable, the manual operation/token transport becomes a separate design decision. Do not stub both transports into v1 "just in case".

### Remote Notifications Setup

CKSyncEngine relies on silent remote notifications to drive incremental sync. This is a Phase 2 prerequisite.

Setup:

- `aps-environment` entitlement (see "Entitlements & CloudKit Container").
- macOS: call `NSApplication.shared.registerForRemoteNotifications()` after app launch. Implement `application(_:didRegisterForRemoteNotificationsWithDeviceToken:)` (CloudKit does not require the token to be forwarded anywhere, but registration must complete).
- iOS: same registration via `UIApplication`, plus `UIBackgroundModes` in `Info.plist` containing `remote-notification`.
- Both targets: implement the appropriate remote-notification application delegate entry point and validate the CKSyncEngine integration against Apple's sample and current SDK. Do not invent manual record-zone subscriptions unless Phase 0 proves the engine does not create the needed database subscription.

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
- Keep membership changes as independent records so adding a tag on one device does not overwrite a different tag added elsewhere. The same membership's add/remove race still needs change-tag conflict handling; see "Membership Add/Delete Semantics".

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
| Tags/folder membership | Independent membership records; different membership IDs commute, same membership add/remove resolves through change tags |
| Folder rename/move | Server-versioned last writer wins |
| Trash vs restore | Server-versioned last writer wins while `deletedAt` is nil |
| Permanent delete | `deletedAt` wins over restore |
| Missing blob after metadata arrives | Show placeholder and retry download/upload repair |
| Account signed out/switched | Disable cloud writes; protect cached data by account identity |

More advanced CRDT-style merging is not justified for the first implementation. It would be over-designed for this product stage and would complicate debugging. The design instead chooses smaller records and clear tombstones.

### Membership Add/Delete Semantics

Folder and tag membership records reduce conflict scope, but they do not make every conflict disappear.

Rules:

- The deterministic membership record ID represents exactly one relationship: `assetID + folderID` or `assetID + tagID` within a library zone.
- Adding membership writes the record with `deletedAt = nil` and `pendingIntent = present`.
- Removing membership writes the same record with `deletedAt = now` and `pendingIntent = tombstoned`.
- A user-visible remove is a tombstone save, not an immediate CloudKit hard delete. Hard deletion is GC only after the grace period.
- Concurrent changes to different membership record IDs commute naturally because they are separate records.
- Concurrent add/remove of the same membership record ID does **not** commute. Resolve it with the same CloudKit change-tag flow as scalar records: merge the local intended state into the current server record and retry. The last accepted CloudKit save wins for that exact membership.
- Parent deletion wins over membership presence. If the asset, folder, or tag is tombstoned or missing, incoming membership-present records are kept tombstoned locally or ignored; they must not resurrect the parent.

Required tests:

- Device A adds asset X to folder F while Device B removes X from F; the final membership state matches the last accepted CloudKit save and both caches converge.
- Device A adds X to folder F while Device B adds X to folder G; both memberships survive.
- Device A deletes folder F while Device B adds X to F offline; folder deletion wins and the membership does not resurrect F.

### Server-Versioned Merge Algorithm

Server-versioned last-writer-wins is implemented through CloudKit record change tags plus field-level merging. With CKSyncEngine, the app materializes records in `nextRecordZoneChangeBatch`; successful saves return updated system fields, and failed saves surface errors through `sentRecordZoneChanges`. If a manual `CKModifyRecordsOperation` is used in a spike, use `.ifServerRecordUnchanged`.

```text
saveDirtyRecord(localRecord):
  add CKSyncEngine.PendingRecordZoneChange.saveRecord(recordID)

saveHardDeletedRecord(recordID):
  # GC path only; normal user deletes are tombstone saves.
  add CKSyncEngine.PendingRecordZoneChange.deleteRecord(recordID)

nextRecordZoneChangeBatch(context, syncEngine):
  pending = syncEngine.state.pendingRecordZoneChanges filtered to context.options.scope
  if pending is empty:
    return nil
  return CKSyncEngine.RecordZoneChangeBatch(pendingChanges: pending) { recordID in
    return recordProvider(recordID)
  }

recordProvider(recordID):
  localRecord = load local dirty record for recordID
  if missing:
    remove stale pending change from CKSyncEngine state
    return nil
  ckRecord = reconstruct from localRecord.ckSystemFieldsBlob,
             or create a new CKRecord with deterministic recordID if no system fields exist
  if localRecord is membership:
    apply pendingIntent by setting deletedAt nil/non-nil
  else:
    apply only localRecord.dirtyFields to ckRecord
  return ckRecord

on sentRecordZoneChanges.savedRecords(savedRecord):
  persist savedRecord.encodeSystemFields(...)
  clear local isDirty / dirtyFields / lastError
  clear membership pendingIntent when relevant

on sentRecordZoneChanges.deletedRecordIDs(recordID):
  clear local system fields for recordID
  mark GC delete complete

on sentRecordZoneChanges.failedRecordSaves(error: serverRecordChanged):
  serverRecord = error.serverRecord
  merged = serverRecord.copy()
  if localRecord is membership:
    apply pendingIntent by setting deletedAt nil/non-nil
  else:
    apply only localRecord.dirtyFields to merged
  persist merged system fields locally
  keep localRecord.isDirty = true
  add/keep pending save so CKSyncEngine sends the merged record

on sentRecordZoneChanges.failedRecordSaves(error: zoneNotFound):
  add CKSyncEngine.PendingDatabaseChange.saveZone(recordID.zoneID)
  clear stale local ckSystemFieldsBlob for that record
  keep localRecord.isDirty = true
  add pending save after zone recreation

on sentRecordZoneChanges.failedRecordSaves(error: unknownItem):
  if local intent is present or tombstoned:
    clear stale local ckSystemFieldsBlob
    keep localRecord.isDirty = true
    add pending save to recreate the tombstone/present record
  else if local intent is hard-delete GC:
    treat delete as complete and clear local sync state

on retryable error (network, throttle, zoneBusy):
  keep localRecord.isDirty = true
  do not duplicate pending changes; CKSyncEngine keeps retryable pending changes according to system conditions

on limitExceeded:
  keep affected records dirty
  reduce future batch size for the affected zone/type and retry; do not mark records synced

on non-retryable error (quotaExceeded, badContainer, permissionFailure):
  keep localRecord.isDirty = true
  localRecord.lastError = error
  surface in UI

on account/notAuthenticated error:
  block cloud writes until CKContainer.accountStatus and cloudAccountID are revalidated
  keep local dirty state visible
```

Required invariants:

- **Conflict detection is change-tag based.** Never use client `updatedAt` as the conflict authority.
- **Manual save policy, if used, is `.ifServerRecordUnchanged`.** `.changedKeys` does not compare record change tags, and `.allKeys` overwrites unchanged fields.
- **Field-level dirty tracking is required.** A record cannot just save "everything I have" because that clobbers server fields the client never observed. Each `CachedCloud*` entity stores dirty field state and clears it on success.
- **Unknown fields on the server record pass through unchanged** because the merge only writes locally-dirty fields and leaves the rest as the server returned them.
- **Membership records skip scalar dirty-field merging, not conflict detection.** `CloudFolderMembership` / `CloudTagMembership` merge by applying the local membership intent to the current server record, then retrying with CloudKit change tags.
- **The next batch must respect CKSyncEngine's requested scope.** Returning pending changes outside `context.options.scope` can produce invalid-argument failures.

## iCloud Account and Offline Behavior

### Account Identity Binding

Cloud cache and CKSyncEngine state must be bound to the signed-in CloudKit account so that switching accounts cannot mix identities. Use two checks:

- `CKContainer.accountStatus` or `fetchUserRecordID` is the authority for CloudKit account availability.
- The cache binding key should be a stable, app-local representation of the CloudKit user identity, derived after CloudKit account validation. For example, hash the `CKContainer.fetchUserRecordID` record name and store that as `cloudAccountID`.
- `FileManager.default.ubiquityIdentityToken` is only a fast local signal for detecting possible iCloud identity changes and protecting local cache reuse. Apple documents that CloudKit clients should not use it as the login-status authority.
- If the token is stored, serialize it via `NSKeyedArchiver` as an auxiliary preflight value, not as the primary CloudKit account ID.
- On every app launch and on `NSUbiquityIdentityDidChangeNotification`, compare the current token against the persisted token if available, then revalidate CloudKit account status and CloudKit user record ID.

State transitions:

| Persisted cloudAccountID | Current CloudKit account | Behavior |
|---|---|---|
| absent | available | First sign-in. Persist `cloudAccountID`, create cloud cache directory and sync engine state. |
| A | A | Continue as normal. |
| A | B | Account switched. Block cloud writes immediately. Surface "iCloud account changed" UI with two options: "Sign back in to the previous iCloud account" (read-only access to existing cache until restored) or "Reset cloud cache" (destructive, clearly labeled). |
| A | unavailable or unverified | Block cloud writes. Keep cloud cache read-only and visible. Restore writable sync only after CloudKit account validation confirms the same `cloudAccountID`. |

`NSUbiquityIdentityDidChangeNotification` and CloudKit account-change signals must be observed for the entire app lifetime, not just at launch. Mid-session account changes must trigger the same block and revalidation.

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

## Implementation Entry Points

The implementation should start from the existing local-library seams and avoid broad rewrites.

Current files to protect:

- `Momento/Storage/LibraryStorage.swift`: remains the local `.momento` package owner. Do not make it a CloudKit transport.
- `Momento/Storage/LibraryMetadataStore.swift`: remains the local-package Core Data store. Do not retrofit CloudKit sync columns into this model.
- `Momento/Storage/LibraryAccessScope.swift`: currently stores recent local libraries as security-scoped bookmarks. Evolve this into a storage-mode-aware descriptor without breaking existing bookmark decoding.
- `Momento/Core/LibraryStore.swift`: remains the `@MainActor` UI state aggregator. Add storage-mode routing here, but keep cloud cache writes inside the cloud-cache write boundary described above.
- `Momento/Core/AssetModels.swift`: `AssetItem.storageURL` is currently non-optional. Cloud mode must not fake a local original URL for remote-only assets; introduce a cloud-aware availability/resolution path before exposing cloud assets through UI workflows that preview, export, or drag originals.
- `Momento/Services/AssetImportService.swift`, `AssetThumbnailService.swift`, and `AssetExportService.swift`: reuse hashing, thumbnail, and export behavior where possible, but route cloud imports/exports through the cloud eligibility and resolver rules.

Preferred shape:

- Add small storage-mode types and adapters first.
- Add a dedicated cloud cache stack/repository for cloud libraries.
- Keep local package APIs stable unless a call site genuinely needs storage-mode dispatch.
- Do not introduce a large generic persistence abstraction before both local and cloud paths have real shared behavior.

## Implementation Phasing

Each phase has an exit gate. Do not start the next phase until the gate is met, because later phases depend on storage identity, account identity, and sync-state correctness.

### Phase 0: Technical Spikes and Release Gates

Pre-Phase-0 facts and decisions:

- macOS deployment target: 26, from `Momento.xcodeproj`.
- iOS deployment target: not present in this repository yet; confirm before implementation.
- Preferred sync transport: CKSyncEngine, if the chosen iOS deployment target supports it.
- iOS v1 consumes cloud libraries only; iOS does not implement `.momento` package opening.
- Account identity: use CloudKit account status as authority; use `ubiquityIdentityToken` only as a fast local cache-mismatch signal.
- Proposed CloudKit container identifier: `iCloud.com.seaony.Momento`; verify before wiring entitlements.

Phase 0 spikes:

- Wire up CloudKit entitlements, container, remote notifications, and provisioning on both macOS and iOS targets; verify a smoke-test record save/load round-trips between the two.
- Confirm iOS deployment target and CKSyncEngine availability. If unavailable, write a separate manual-sync design instead of adding both transports.
- Confirm the final CloudKit container identifier in Xcode and the Apple Developer portal.
- Validate catalog-zone plus per-library-zone creation, deletion, and sync behavior.
- Validate one private-database sync engine handling catalog and per-library zones.
- Validate account binding flow: sign in, sign out, account switch, mid-session change, `CKContainer.accountStatus`, and `NSUbiquityIdentityDidChangeNotification`.
- Measure CKAsset upload/download behavior with realistic file sizes and failure modes (including CKAsset re-hash verification after download).
- Measure initial sync and local cache behavior with large synthetic libraries.
- Measure multi-library zone behavior (not just one big library) — see Finding 18.
- Define first-release cloud limits for file type, file size, cloud-library count, per-library asset count, and total cloud bytes. Start the spike with conservative seed values such as 5k assets per library, 100 MB per file, and 20 cloud libraries per account, but do not ship these limits without measurement.
- Define CloudKit schema/index checklist before production promotion.

Exit gate:

- A written Phase 0 result exists with the chosen iOS deployment target, final container identifier, CKSyncEngine availability result, entitlement/provisioning status, account-switch findings, CKAsset size findings, and proposed v1 limits.
- A real-device smoke test proves a record can save on one platform and fetch on the other through the selected container.
- If CKSyncEngine is not available for the selected iOS target, stop this plan and write the manual-sync design before implementation.

### Phase 1: Storage Mode Foundation

Outputs:

- Storage-mode-aware `LibraryDescriptor` / recent-library registry with backward-compatible migration from existing `RecentLibraryReference`.
- Local libraries still create, open, rename, reveal, reorder, delete, import, and export as `.momento` packages.
- Cloud library descriptors can be represented locally as empty placeholders after iCloud account availability is known; real CloudKit discovery waits for Phase 2.
- Library picker/sidebar separates "On This Mac" and "iCloud" without exposing local-only libraries on iOS.
- Account-state service exposes at least: available, unavailable, restricted/error, switched/mismatch.

Exit gate:

- Existing local-library lifecycle tests still pass.
- New tests cover migration of existing local recent libraries, local-only invisibility on iOS registry code paths, and storage-mode selection at create time.
- No CloudKit metadata or blob sync is implemented in this phase beyond account/status checks.

### Phase 2: Cloud Metadata Sync

Outputs:

- CloudKit record schema constants and deterministic record-name builders with ASCII/length tests.
- Catalog zone and per-library zone creation through one account/private-database CKSyncEngine.
- Dedicated cloud-cache Core Data model/stack and cloud-cache write boundary.
- Cloud library create, discover, open, rename, and tombstone delete.
- Metadata sync for folders, tags, memberships, trash state, favorite, note, display name, and asset metadata without original download.
- Server-versioned conflict handling for scalar fields and same-membership add/remove.
- Development schema/index checklist for every query used by discovery and sync.

Exit gate:

- Protocol-backed fake CloudKit/CKSyncEngine tests simulate two devices with separate local caches.
- Tests cover deterministic record IDs, zone lifecycle, metadata dirty retry, serverRecordChanged merge, zoneNotFound recovery, unknownItem handling, and same-membership add/remove convergence.
- Manual real-device test proves cloud library creation/rename and metadata edit sync between Mac and iOS.
- Local `.momento` lifecycle tests still pass.

### Phase 3: Blob Upload/Download

Outputs:

- Cloud import eligibility checks for file type, byte size, and local readability before normal cloud asset creation.
- Import copies originals into the cloud cache, computes SHA-256, creates thumbnails, and marks `CloudAsset`/`CloudAssetBlob` dirty in one local transaction.
- CKAsset upload uses canonical cloud-cache file URLs that remain readable until send completion.
- CKAsset download copies staged files into the cloud cache during the record-change callback, verifies SHA-256, and only then marks originals local.
- iOS cache files apply the backup-exclusion and file-protection policy from this document.
- Preview/export/drag uses the cloud-aware original resolver and surfaces download failure.
- Upload/download failure states are visible and durable.

Exit gate:

- Tests cover import eligibility rejection, uploadFailed state, download hash mismatch, staged-file copy, backup-exclusion marking, file-protection assignment on iOS paths, and no eviction of upload-pending originals.
- Manual real-device test covers Mac import visible on iOS, iOS import visible on Mac, on-demand original download, offline import then upload, and quota/upload failure handling if a safe test path is available.
- Large-file spike result is written before enabling file types/sizes beyond the measured v1 limits.

### Phase 4: Full Read-Write Cross-Device Behavior

Outputs:

- Shared Mac/iOS command layer for rename, note, favorite, tags, folders, trash, restore, permanent delete, and cloud-library delete.
- Visible per-library and per-record sync status in the UI.
- Copy local library to iCloud is resumable and cancellable before it is enabled for non-trivial libraries.
- Export cloud library to local `.momento` package downloads required originals first and remains Mac-only in v1.
- Multi-device integration suite runs against fake/protocol CloudKit with separate caches.

Exit gate:

- Multi-device tests cover concurrent rename, concurrent folder move, same-membership add/remove, different-membership add/add, offline edits, account sign-out, account switch, quota/upload failure, permanent delete vs restore, and blob GC grace period.
- Manual matrix passes on two real devices with the same Apple ID.
- Schema/index checklist is complete and production promotion is explicitly approved before release builds point at production CloudKit schema.
- Build/typecheck passes for macOS and iOS targets.

## Validation Plan

Minimum validation for the design implementation later:

- Spike evidence for CKSyncEngine availability, CKAsset file behavior, and large-library scale before enabling cloud import broadly.
- Unit tests for storage-mode registry: local libraries stay local; cloud libraries are discovered from cloud cache.
- Unit tests for deterministic record IDs and dedupe.
- Unit tests for zone naming and library-zone lifecycle.
- Unit tests for tombstone rules: delete vs restore, folder delete vs membership arrival, blob GC grace period.
- Unit tests for dirty-state retry and partial failure handling.
- Unit tests for CKSyncEngine batch scope filtering, `serverRecordChanged`, `zoneNotFound`, `unknownItem`, and `limitExceeded` handling.
- Unit tests for same-membership add/remove conflict convergence.
- Unit tests for iOS cache backup-exclusion, file-protection policy, and upload-pending original durability.
- Unit tests or actor-isolation tests for the cloud-cache write boundary so UI writes and sync callbacks cannot mutate the same cache directly.
- Unit tests for record-name generation: ASCII-only, deterministic, under 255 characters.
- Integration tests using CloudKit test environment or protocol-backed fake CloudKit service.
- Manual tests on two devices with the same Apple ID: Mac import, iOS import, concurrent rename, concurrent folder move, same-membership add/remove, offline edits, account sign-out, account switch, quota/upload failure.
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

- CKSyncEngine is the preferred v1 transport if the chosen iOS target supports it.
- The manual `CKFetchRecordZoneChangesOperation`+token path is not implemented in parallel "just in case".
- If the iOS target cannot use CKSyncEngine, treat the manual path as a separate design before implementation.

### Finding 18: Per-library zones need a library-count release gate

Counter case:

- A user creates many small cloud libraries. The per-library zone model remains semantically correct, but zone creation, change fetching, and local cache bookkeeping may become the bottleneck before asset count does.

Fix in design:

- Measure multi-library zone behavior in Phase 0, not just one large library.
- Define a first-release cloud-library count limit if measurement shows operational overhead.

### Finding 19: Same-membership add/remove is not naturally commutative

Counter case:

- Mac adds asset X to folder F while iOS removes asset X from folder F offline. Both actions target the same deterministic membership record, so treating membership records as "insert-or-tombstone, no conflict" can leave clients disagreeing about whether the relationship exists.

Fix in design:

- Add `pendingIntent` for membership rows.
- Resolve same-membership add/remove through CloudKit change tags by applying the local intended `deletedAt` state to the current server record and retrying.
- Add convergence tests for same-membership add/remove and different-membership add/add.

### Finding 20: CKSyncEngine failure handling needs executable branches

Counter case:

- The app tries to save a record in a missing zone or with stale system fields. If the plan only says "keep pending and retry", CKSyncEngine may keep failing because the app never recreates the zone, clears stale system fields, or re-adds the repaired pending save.

Fix in design:

- Add explicit branches for `serverRecordChanged`, `zoneNotFound`, `unknownItem`, retryable errors, `limitExceeded`, account/notAuthenticated, and non-retryable blocking errors.
- Require `nextRecordZoneChangeBatch` to filter pending changes to CKSyncEngine's requested scope.
- Add tests for each branch before relying on real CloudKit behavior.

### Finding 21: iOS cache durability is separate from sync cache design

Counter case:

- iOS imports an image, stores it in a cache-like directory, marks it excluded from backup, then upload fails or the app is restored before upload succeeds. The imported original can be lost even though the UI showed it locally.

Fix in design:

- Keep dirty metadata, sync-engine state, and upload-pending originals in Application Support.
- Do not exclude upload-pending originals from backup until upload success.
- Exclude re-downloadable synced blobs and thumbnails from backup after every write/copy.
- Specify a file-protection policy compatible with background sync after first unlock.

### Finding 22: CloudKit callbacks need a single write owner

Counter case:

- A user edits metadata while a remote change callback and CKAsset download are also updating the same rows. If UI, sync delegate, and file IO mutate the cache independently, `isDirty`, system fields, and availability can diverge.

Fix in design:

- Add a required cloud-cache write boundary.
- Keep `LibraryStore` as the `@MainActor` presentation layer.
- Route UI commands and CKSyncEngine delegate events through the same serialized cloud-cache writer.

### Finding 23: Phase bullets were not enough for execution

Counter case:

- An implementer starts Phase 2 before Phase 0 proves CKSyncEngine availability or before Phase 1 protects local-library registry migration. The design remains correct in theory but creates rework and untestable partial states.

Fix in design:

- Add implementation entry points tied to current Momento files.
- Add per-phase outputs and exit gates.
- Add validation coverage for the newly identified conflict, CKSyncEngine, cache durability, and concurrency risks.

## Final Recommendation

Build cloud libraries as a separate CloudKit-backed storage mode while preserving local `.momento` libraries unchanged.

Use CKSyncEngine plus explicit CloudKit records, a local cache, deterministic record IDs, membership records, tombstones, and visible sync states. This is more work than package sync, but it is the smallest design that correctly supports Mac/iOS read-write behavior without hiding data-loss risks.

Do not implement sharing, chunked uploads, CRDTs, a second manual sync transport, or package-level iCloud Drive sync in the first version. Those are either outside the stated requirement or unjustified until real usage proves the need.
