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
- Local libraries continue to use the existing `.momento` package with no iCloud behavior.

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
- CloudKit schema and indexes need production planning. Apple documents that production schemas can only evolve forward, and queryable fields need indexes.
  - Source: [Designing apps using CloudKit](https://developer.apple.com/icloud/cloudkit/designing/)
  - Source: [Inspecting and Editing an iCloud Container's Schema](https://developer.apple.com/documentation/cloudkit/inspecting-and-editing-an-icloud-container-s-schema)
- `CKSyncEngine` is Apple's current sync helper for local/remote CloudKit records. It requires persisting sync-engine state, CloudKit and remote notification entitlements, and accepting that background sync timing is indeterminate.
  - Source: [CKSyncEngine](https://developer.apple.com/documentation/cloudkit/cksyncengine-5sie5)
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
- Requires a local pending-change queue and conflict handling.
- Requires CloudKit entitlements, schema deployment, telemetry, and real multi-device tests.

Verdict: use this.

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

### CloudKit Scope

Use the user's private CloudKit database. This matches the requested behavior: the same Apple ID sees the same cloud libraries on Mac and iOS.

Use one custom private zone for Momento cloud data in the first version:

```text
zone: MomentoCloud
```

Reasons:

- One zone keeps discovery and subscription setup simpler.
- All cloud libraries are listed by querying/syncing `CloudLibrary` records.
- Deleting a library can be represented by tombstoning the library and cascading records through app logic.
- Per-library zones can be revisited later if library sharing becomes a product goal. Sharing is not part of this requirement.

### Local Cache

Cloud libraries still need a local store:

```text
Application Support/Momento/CloudLibraries/<libraryID>/
├── cache.sqlite
├── assets/<hashPrefix>/<sha256>.<ext>       # downloaded originals only
├── thumbnails/<sha256>.png                  # downloaded/generated cache
└── sync/
    ├── engine-state
    └── pending-operations
```

This cache is not a user-facing `.momento` package. It is an implementation detail for offline support and fast UI.

Rules:

- UI reads local cache through the same value-model boundary as local packages.
- Writes first apply to local cache in a transaction, then enqueue a sync operation.
- A failed upload must leave a visible pending/error state. No silent success.
- Cached originals can be evicted later, but metadata and thumbnails needed for browsing should remain.

### Cloud Record Schema

Use explicit CloudKit records rather than mirrored Core Data entities.

```text
CloudLibrary
- id
- displayName
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

- Record names should be deterministic and include `libraryID` where needed, for example `library:<uuid>`, `asset:<libraryID>:<contentHash>`, `blob:<libraryID>:<contentHash>`, and `folder-membership:<libraryID>:<assetID>:<folderID>`.
- Do not store large blobs on `CloudAsset`. Metadata updates should not require touching the original file.
- Use membership records instead of arrays on `CloudAsset`; this reduces conflicts and avoids large-array update problems.
- Keep `deletedAt` tombstones for sync correctness. Hard deletion can be delayed.
- Keep color analysis local initially. Palette colors can be added later if needed, but they are derived and should not block first sync.
- Define queryable/sortable CloudKit indexes before production promotion for fields used in discovery and sync-related queries, especially `libraryID`, `deletedAt`, `updatedAt`, `contentHash`, and normalized tag/folder names where queried.

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
- If a file is too large or CloudKit rejects it, do not create a fake synced asset. Mark the local import as "not uploaded" and surface the failure.
- The first production version may restrict cloud libraries to common image formats and a documented maximum file size.
- Do not implement chunked uploads until there is a proven requirement. Chunking would add substantial complexity to dedupe, retry, delete, and download behavior.

This is a deliberate correction to the earlier simple "store every original as CKAsset" idea. CKAsset is still the default, but upload eligibility must be enforced.

### Sync Engine

Use `CKSyncEngine` for cloud libraries when deployment targets allow it.

Responsibilities:

- Initialize the sync engine early for the private database.
- Persist sync-engine state in the local cache.
- Maintain a local pending-change queue.
- Map local changes to CloudKit records in `nextRecordZoneChangeBatch`.
- Apply remote record changes into local cache.
- Handle account changes, zone deletion, partial failures, retries, and server conflicts.
- Provide observable sync state for UI.

Fallback if CKSyncEngine is unavailable on the chosen iOS target:

- Use `CKFetchDatabaseChangesOperation`, `CKFetchRecordZoneChangesOperation`, server change tokens, and record-zone subscriptions manually.
- Keep the same local cache and record schema.

Given the current macOS target is modern, CKSyncEngine should be the default unless the iOS deployment target forces otherwise.

## Mutation Flow

All user writes should use the same command pattern on Mac and iOS.

### Import Image

1. User chooses image from Files/Photos/Finder.
2. App copies it into local cache.
3. App computes SHA-256.
4. App creates or reuses local `Asset` by `contentHash`.
5. App creates thumbnail locally.
6. App writes metadata/blob state in one local transaction.
7. App enqueues `save CloudAsset` and, if missing remotely, `save CloudAssetBlob`.
8. UI shows the asset immediately with sync status.
9. Sync uploads metadata and blob.
10. Other devices receive metadata first, then download thumbnail/original on demand.

### Edit Metadata

Examples: rename asset, edit note, favorite/unfavorite, tag changes.

Rules:

- Apply locally first.
- Enqueue a semantic record save.
- Use CloudKit record metadata / change tags to detect server conflicts.
- Merge simple scalar conflicts with last-writer-wins using `updatedAt`.
- Keep membership changes as independent records so adding a tag on one device does not overwrite another tag added elsewhere.

### Folder Operations

Folder create, rename, move, and delete are record-level operations.

Rules:

- Folder rename: last-writer-wins by `updatedAt`.
- Folder move: last-writer-wins for `parentID` and `sortIndex`.
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
| Display name / note / favorite | Last writer wins by `updatedAt` |
| Tags/folder membership | Independent membership tombstone records |
| Folder rename/move | Last writer wins |
| Trash vs restore | Later `updatedAt` wins while `deletedAt` is nil |
| Permanent delete | `deletedAt` wins over restore |
| Missing blob after metadata arrives | Show placeholder and retry download/upload repair |
| Account signed out/switched | Disable cloud writes; protect cached data by account identity |

More advanced CRDT-style merging is not justified for the first implementation. It would be over-designed for this product stage and would complicate debugging. The design instead chooses smaller records and clear tombstones.

## iCloud Account and Offline Behavior

Cases:

- No iCloud account: cloud libraries cannot be created or synced.
- Same iCloud account, offline network: allow local cloud-library writes and queue uploads.
- Account temporarily unavailable: keep cached cloud libraries visible, but show sync blocked state.
- Account switched: do not open previous account's cloud cache as writable. Require resync or explicit cache reset.
- iCloud quota exceeded: keep local changes pending, show quota/upload failure, and do not claim sync success.

The UI needs at least these sync states:

- Synced
- Syncing
- Waiting for network
- Waiting for iCloud sign-in
- Upload failed
- Download failed
- Quota/storage blocked
- Conflict resolved

## Migration and Conversion

Do not mutate a local `.momento` package into a cloud library in place.

Supported first path:

1. User chooses "Copy Library to iCloud".
2. App opens the local package.
3. App creates a new cloud library with a new `libraryID`.
4. App imports local assets into the cloud cache using existing content hashes.
5. App uploads metadata and blobs.
6. The original local library remains unchanged.

Why:

- Avoids destroying the user's local-only source.
- Avoids changing security-scoped bookmarks.
- Makes failures recoverable.
- Keeps local and cloud lifecycle rules separate.

Export remains the inverse: cloud library can be exported to a local `.momento` package after all required originals are downloaded.

## Implementation Phasing

### Phase 1: Storage Mode Foundation

- Add storage-mode concept to library descriptors.
- Keep existing local package behavior unchanged.
- Add cloud library registry/listing with empty cloud libraries.
- Add iCloud capability checks and account-state UI.

### Phase 2: Cloud Metadata Sync

- Add CloudKit schema and one private custom zone.
- Add local cloud cache and sync-engine state persistence.
- Sync libraries, folders, tags, and asset metadata without original download beyond thumbnails.
- Add deterministic record IDs and tombstones.

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

- Unit tests for storage-mode registry: local libraries stay local; cloud libraries are discovered from cloud cache.
- Unit tests for deterministic record IDs and dedupe.
- Unit tests for tombstone rules: delete vs restore, folder delete vs membership arrival, blob GC grace period.
- Unit tests for pending operation retry and partial failure handling.
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

### Finding 6: Per-library zones may be premature

Counter case:

- A user creates many libraries. Per-library zones and subscriptions increase operational overhead before sharing is needed.

Fix in design:

- Use one custom private zone for the first version.
- Keep per-library zones as a future option only if sharing or performance requires it.

### Finding 7: Offline-first writes can hide sync failures

Counter case:

- iOS imports an image offline. The UI shows the asset normally. Later upload fails due to quota. Without a durable pending state, the user assumes the image is on Mac too.

Fix in design:

- Every local cloud-library write creates a durable pending operation.
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

## Final Recommendation

Build cloud libraries as a separate CloudKit-backed storage mode while preserving local `.momento` libraries unchanged.

Use CKSyncEngine plus explicit CloudKit records, a local cache, deterministic record IDs, membership records, tombstones, and visible sync states. This is more work than package sync, but it is the smallest design that correctly supports Mac/iOS read-write behavior without hiding data-loss risks.

Do not implement sharing, chunked uploads, CRDTs, or package-level iCloud Drive sync in the first version. Those are either outside the stated requirement or unjustified until real usage proves the need.
