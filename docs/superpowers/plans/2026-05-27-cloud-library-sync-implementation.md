# Cloud Library Sync Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build CloudKit-backed read-write cloud libraries for Momento while preserving existing local `.momento` libraries unchanged.

**Architecture:** Local libraries remain package-based and Mac-only. Cloud libraries use CloudKit private database records, one account-scoped CKSyncEngine, a dedicated local cloud cache, deterministic record IDs, tombstones, durable dirty/error state, and cloud-aware original-file resolution.

**Tech Stack:** Swift, SwiftUI, AppKit, Core Data, CloudKit, CKSyncEngine, CKAsset, Xcode entitlements, XCTest.

---

## Non-Negotiable Scope Boundary

This plan must not enable the iCloud create option until the Phase 0 gates pass.

Current repository facts:

- macOS target exists and uses `MACOSX_DEPLOYMENT_TARGET = 26.0`.
- No iOS target exists in this repository. The iOS app is a separate worktree at `/Users/seaony/code/Momento-iOS`.
- `/Users/seaony/code/Momento-iOS` has an iOS `Momento` target with `IPHONEOS_DEPLOYMENT_TARGET = 26.5` and existing CloudKit model/service code.
- `/Users/seaony/code/Momento-iOS/Momento/Core/CloudLibrarySyncTypes.swift` currently defines `CloudKitConfiguration.containerIdentifier = "iCloud.com.seaony.Momento"` and is the compatibility source for record names and schema fields.
- `Momento/Momento.entitlements` has no CloudKit, iCloud container, or `aps-environment` key.
- `Momento/Features/Library/MomentoCreateLibraryDialog.swift` intentionally disables the iCloud row.
- `Momento/Core/LibraryStore.swift` rejects `.cloud` libraries with `cloudLibraryUnavailable`.

External facts that code cannot safely invent:

- Final CloudKit container identifier in Apple Developer portal and Xcode Signing & Capabilities. The iOS code value alone is not enough.
- Apple Developer portal container/app ID setup.
- Provisioning profile regeneration.
- iOS entitlements/provisioning status in `/Users/seaony/code/Momento-iOS`.
- Real-device smoke test evidence.

Stop and ask the user before changing entitlements or project signing settings if those values are not confirmed.

## Cross-Repo Compatibility Rule

macOS and iOS must write identical CloudKit record names, record types, zones, and field keys.

Rules:

- Treat `/Users/seaony/code/Momento-iOS/Momento/Core/CloudLibrarySyncTypes.swift` as the current iOS compatibility source until the shared model is extracted.
- Do not modify `/Users/seaony/code/Momento-iOS` from this macOS implementation branch unless the user explicitly asks for cross-repo edits.
- Any macOS CloudKit schema change must include a compatibility test proving it matches the iOS record naming behavior.
- iOS currently has a dirty worktree. Read it for compatibility, but do not overwrite or revert its changes.

## Official Sources Checked

- Apple CloudKit overview: https://developer.apple.com/documentation/cloudkit
- Apple CKSyncEngine overview: https://developer.apple.com/documentation/cloudkit/cksyncengine-5sie5
- Apple CKSyncEngine state serialization: https://developer.apple.com/documentation/cloudkit/cksyncenginestateupdateevent/4155542-stateserialization
- Apple CKSyncEngine pending record changes: https://developer.apple.com/documentation/cloudkit/cksyncenginestate/pendingrecordzonechanges
- Apple CKAsset: https://developer.apple.com/documentation/cloudkit/ckasset
- Apple CKContainer account status and account change notification: https://developer.apple.com/documentation/cloudkit/ckcontainer
- Apple `serverRecordChanged`: https://developer.apple.com/documentation/cloudkit/ckerror/serverrecordchanged
- Apple `quotaExceeded`: https://developer.apple.com/documentation/cloudkit/ckerror/quotaexceeded
- Apple CKSyncEngine sample: https://github.com/apple/sample-cloudkit-sync-engine

Key constraints from the sources:

- CKSyncEngine state must be persisted across launches alongside app data.
- CKSyncEngine requires CloudKit and remote notification entitlements.
- CKSyncEngine sync timing is system-controlled; explicit send/fetch is needed only when the UI must force freshness.
- CKAsset staged download files must be copied into the app container if the app needs to keep them.
- `serverRecordChanged` must merge local intent into the server record and retry with that server record.
- Private database quota failures mean the user lacks iCloud storage.

## File Map

### Keep Stable

- `Momento/Storage/LibraryStorage.swift`: local `.momento` package owner only.
- `Momento/Storage/LibraryMetadataStore.swift`: local-package Core Data store only.
- `Momento/Storage/MomentoModel.xcdatamodeld`: local-package model only.
- `Momento/Storage/LibraryAccessScope.swift`: storage-mode descriptor and local bookmark registry.
- `Momento/Core/LibraryStore.swift`: main-actor presentation and command router.

### Create

- `Momento/Cloud/CloudKitConfiguration.swift`: container identifier, database scope, zone names, record type constants.
- `Momento/Cloud/CloudRecordIDBuilder.swift`: deterministic ASCII record-name generation.
- `Momento/Cloud/CloudLibrarySchema.swift`: record field constants and record materialization helpers.
- `Momento/Cloud/CloudLibraryCachePaths.swift`: Application Support paths, backup-exclusion and protection helpers.
- `Momento/Cloud/CloudCacheCoreDataStack.swift`: cloud-cache persistent container.
- `Momento/Cloud/MomentoCloudModel.xcdatamodeld`: cache-only Core Data model.
- `Momento/Cloud/CloudLibraryRepository.swift`: single serialized write boundary for UI commands and sync callbacks.
- `Momento/Cloud/CloudSyncEngineController.swift`: account-scoped CKSyncEngine owner and delegate bridge.
- `Momento/Cloud/CloudSyncErrorClassifier.swift`: retryable, blocking, quota, account, conflict, and batch-size classification.
- `Momento/Cloud/CloudAssetEligibility.swift`: file type, byte-size, readability, and hash preflight.
- `Momento/Cloud/CloudOriginalFileResolver.swift`: preview/export/drag download and local file resolution.
- `MomentoTests/CloudRecordIDBuilderTests.swift`
- `MomentoTests/CloudLibraryCachePathTests.swift`
- `MomentoTests/CloudAssetEligibilityTests.swift`
- `MomentoTests/CloudSyncEngineControllerTests.swift`
- `MomentoTests/CloudLibraryRepositoryTests.swift`
- `MomentoTests/CloudOriginalFileResolverTests.swift`
- `docs/superpowers/specs/2026-05-27-cloud-library-sync-phase0-status.md`: update evidence as gates pass.
- `docs/cloudkit-schema-checklist.md`: development and production CloudKit schema/index checklist.

### Modify Later, Only After Gates Pass

- `Momento/Momento.entitlements`
- `Momento/Info.plist`
- `Momento.xcodeproj/project.pbxproj`
- `Momento/Features/Library/MomentoCreateLibraryDialog.swift`
- `Momento/Features/Sidebar/MomentoSidebarView.swift`
- `Momento/Core/LibraryStore.swift`
- `Momento/Core/AssetModels.swift`
- `Momento/Services/AssetImportService.swift`
- `Momento/Services/AssetExportService.swift`

## Chunk 1: Phase 0 Gates and Smoke Harness

**Purpose:** Prove the app has a real CloudKit target before any cloud-library UI is enabled.

**Files:**

- Modify: `docs/superpowers/specs/2026-05-27-cloud-library-sync-phase0-status.md`
- Create: `docs/cloudkit-schema-checklist.md`
- Create: `Momento/Cloud/CloudKitConfiguration.swift`
- Create: `Momento/Cloud/CloudKitSmokeTestService.swift`
- Test: `MomentoTests/CloudKitConfigurationTests.swift`

- [ ] Step 1: Confirm final container identifier.

  Required user-provided value: exact CloudKit container ID from Xcode and Apple Developer portal.

  Expected format: `iCloud.<team-or-domain>.<app-name>`.

  Stop if the value is still only proposed.

- [ ] Step 2: Confirm iOS target exists or create it as a separate app target.

  Required decisions: target name, bundle identifier, deployment target, shared source membership policy, iOS entitlements file.

  Stop if the product does not yet want an iOS target in this repository.

- [ ] Step 3: Add `CloudKitConfiguration`.

  It should expose:

  ```swift
  enum CloudKitConfiguration {
      static let containerIdentifier = "<confirmed container>"
      static let catalogZoneName = "MomentoCatalog"
      static let libraryZonePrefix = "MomentoLibrary-"
      static let databaseScope: CKDatabase.Scope = .private
  }
  ```

  Do not hard-code the proposed container until Step 1 is confirmed.

- [ ] Step 4: Add CloudKit and remote notification entitlements.

  macOS keys:

  ```xml
  <key>com.apple.developer.icloud-services</key>
  <array>
      <string>CloudKit</string>
  </array>
  <key>com.apple.developer.icloud-container-identifiers</key>
  <array>
      <string><confirmed container></string>
  </array>
  <key>aps-environment</key>
  <string>development</string>
  ```

  iOS target needs the same CloudKit/container keys and remote-notification background mode in its `Info.plist`.

- [ ] Step 5: Add a debug-only smoke service.

  The service saves a tiny record in `MomentoCatalog`, fetches it by record ID, then deletes it. It must be excluded from normal app flow.

- [ ] Step 6: Run local build.

  Run:

  ```bash
  xcodebuild build -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-cloud-phase0
  ```

  Expected: build succeeds. If signing fails, stop and fix provisioning instead of disabling signing checks.

- [ ] Step 7: Run real-device smoke test.

  Required evidence:

  - Mac saves and fetches a smoke record through the confirmed private database.
  - iOS saves and fetches a smoke record through the same private database.
  - Mac can fetch the iOS-created smoke record and iOS can fetch the Mac-created smoke record.

- [ ] Step 8: Update Phase 0 status.

  Record the actual container identifier, target names, deployment targets, entitlement status, provisioning status, and smoke-test result.

- [ ] Step 9: Commit.

  ```bash
  git add Momento MomentoTests docs
  git commit -m "feat: add CloudKit phase 0 smoke harness"
  ```

## Chunk 2: Cloud Record Identity and Schema Checklist

**Purpose:** Make cloud record identity deterministic before cache or sync code depends on it.

**Files:**

- Create: `Momento/Cloud/CloudRecordIDBuilder.swift`
- Create: `Momento/Cloud/CloudLibrarySchema.swift`
- Create: `docs/cloudkit-schema-checklist.md`
- Test: `MomentoTests/CloudRecordIDBuilderTests.swift`

- [x] Step 1: Write tests for record names.

  Cover:

  - ASCII only.
  - Deterministic for same input.
  - Under 255 characters.
  - Membership IDs hash long composed keys.
  - Catalog library records live in `MomentoCatalog`.
  - Asset/blob/folder/tag/membership records live in `MomentoLibrary-<libraryID>`.

- [x] Step 2: Implement record ID builder.

  Required functions:

  ```swift
  static func catalogZoneID(ownerName: String = CKCurrentUserDefaultName) -> CKRecordZone.ID
  static func libraryZoneID(libraryID: String, ownerName: String = CKCurrentUserDefaultName) -> CKRecordZone.ID
  static func cloudLibraryRecordID(libraryID: String) -> CKRecord.ID
  static func assetRecordID(libraryID: String, contentHash: String) -> CKRecord.ID
  static func blobRecordID(libraryID: String, contentHash: String) -> CKRecord.ID
  static func folderRecordID(libraryID: String, folderID: String) -> CKRecord.ID
  static func tagRecordID(libraryID: String, tagID: String) -> CKRecord.ID
  static func folderMembershipRecordID(libraryID: String, assetID: String, folderID: String) -> CKRecord.ID
  static func tagMembershipRecordID(libraryID: String, assetID: String, tagID: String) -> CKRecord.ID
  ```

- [x] Step 3: Add schema constants.

  Record types:

  - `CloudLibrary`
  - `CloudAsset`
  - `CloudAssetBlob`
  - `CloudFolder`
  - `CloudTag`
  - `CloudFolderMembership`
  - `CloudTagMembership`

- [x] Step 4: Write CloudKit schema checklist.

  Include every record type, field name, field type, queryable index, sortable index, and production-promotion gate.

- [x] Step 5: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudRecordIDBuilderTests -derivedDataPath /tmp/MomentoDerivedData-cloud-records
  ```

- [x] Step 6: Commit.

  ```bash
  git add Momento/Cloud MomentoTests docs/cloudkit-schema-checklist.md
  git commit -m "feat: add CloudKit record identity schema"
  ```

## Chunk 3: Cloud Cache Storage

**Purpose:** Add a local cache that can survive offline writes without touching local `.momento` packages.

**Files:**

- Create: `Momento/Cloud/CloudLibraryCachePaths.swift`
- Create: `Momento/Cloud/CloudCacheCoreDataStack.swift`
- Create: `Momento/Cloud/MomentoCloudModel.xcdatamodeld`
- Test: `MomentoTests/CloudLibraryCachePathTests.swift`
- Test: `MomentoTests/CloudCacheCoreDataStackTests.swift`

- [x] Step 1: Write cache path tests.

  Cover:

  - Account-scoped root under Application Support.
  - Per-library cache path.
  - Account ID path traversal rejection.
  - Library ID path traversal rejection.
  - Thumbnail paths are marked excluded from backup.
  - Synced downloaded originals are excluded from backup.
  - Upload-pending originals are not excluded from backup.

- [x] Step 2: Create cache path helper.

  Layout:

  ```text
  Application Support/Momento/CloudLibraries/<cloudAccountID>/<libraryID>/
  ├── cache.sqlite
  ├── assets/<hashPrefix>/<sha256>.<ext>
  └── thumbnails/<sha256>.png
  ```

- [x] Step 3: Create cloud Core Data model.

  Entities:

  - `CachedCloudLibrary`
  - `CachedCloudAsset`
  - `CachedCloudAssetBlob`
  - `CachedCloudAssetColor`
  - `CachedCloudFolder`
  - `CachedCloudTag`
  - `CachedCloudFolderMembership`
  - `CachedCloudTagMembership`

  Required sync fields on record-backed entities:

  - `ckSystemFieldsBlob`
  - `ckRecordChangeTag`
  - `isDirty`
  - `dirtyFields`
  - `lastError`
  - `deletedAt`
  - `recordName`
  - `zoneName`

- [x] Step 4: Add cloud cache stack.

  It must open `cache.sqlite` under the cloud cache root and must not reuse `MomentoCoreDataStack`.

- [x] Step 5: Run tests and local lifecycle regression tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudCacheCoreDataStackTests -only-testing:MomentoTests/CloudLibraryCachePathTests -only-testing:MomentoTests/LibraryPackagePersistenceTests -derivedDataPath /tmp/MomentoDerivedData-cloud-cache
  ```

- [x] Step 6: Commit.

  ```bash
  git add Momento/Cloud MomentoTests
  git commit -m "feat: add cloud library cache storage"
  ```

## Chunk 4: Serialized Cloud Repository

**Purpose:** Ensure UI writes and CloudKit callbacks cannot race over the same cache rows/files.

**Files:**

- Create: `Momento/Cloud/CloudLibraryRepository.swift`
- Create: `Momento/Cloud/CloudLibraryRepositoryModels.swift`
- Test: `MomentoTests/CloudLibraryRepositoryTests.swift`

- [ ] Step 1: Write repository tests with a temporary cache.

  Cover:

  - Create cloud library locally.
  - Rename cloud library marks `CloudLibrary.displayName` dirty.
  - Folder create/rename/move marks only the intended dirty fields.
  - Tag add/remove creates or tombstones membership records.
  - Permanent delete uses `deletedAt` tombstones and does not hard-delete rows.

- [ ] Step 2: Implement repository as an actor.

  Required responsibilities:

  - Own cloud cache Core Data context.
  - Own cloud cache file moves/copies for cloud-managed files.
  - Convert cache rows into UI value snapshots.
  - Persist CKRecord system fields.
  - Persist dirty field sets.
  - Keep local package APIs out of cloud cache writes.

- [ ] Step 3: Add read model snapshots.

  Do not fake a local `storageURL` for remote-only originals. Represent original and thumbnail availability explicitly.

- [ ] Step 4: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudLibraryRepositoryTests -derivedDataPath /tmp/MomentoDerivedData-cloud-repository
  ```

- [ ] Step 5: Commit.

  ```bash
  git add Momento/Cloud MomentoTests
  git commit -m "feat: add cloud library repository"
  ```

## Chunk 5: CKSyncEngine Controller

**Purpose:** Sync metadata through one private-database engine, with explicit error branches.

**Files:**

- Create: `Momento/Cloud/CloudSyncEngineController.swift`
- Create: `Momento/Cloud/CloudSyncErrorClassifier.swift`
- Test: `MomentoTests/CloudSyncEngineControllerTests.swift`

- [ ] Step 1: Define protocol seams for tests.

  Do not make production code depend directly on a fake-only abstraction. Keep protocols narrow:

  - Engine state read/add pending changes.
  - Record provider callback.
  - Event handler input.

- [ ] Step 2: Persist CKSyncEngine state.

  Store state under:

  ```text
  Application Support/Momento/CloudSync/<cloudAccountID>/private-database-engine-state
  ```

- [ ] Step 3: Implement pending batch scope filtering.

  `nextRecordZoneChangeBatch` must return only pending changes matching the engine-requested scope.

- [ ] Step 4: Implement success handling.

  On saved records:

  - Persist encoded system fields.
  - Clear `isDirty`.
  - Clear dirty field sets.
  - Clear last error.

- [ ] Step 5: Implement failure handling.

  Required branches:

  - `serverRecordChanged`: merge local dirty fields into server record, persist server fields, keep dirty, requeue.
  - `zoneNotFound`: queue zone save, clear stale system fields, keep dirty, requeue.
  - `unknownItem`: clear stale system fields and recreate tombstone/present record when needed.
  - Retryable network/throttle errors: keep dirty and do not duplicate pending changes.
  - `limitExceeded`: keep dirty, reduce future batch size.
  - `quotaExceeded`: keep dirty, store quota error, surface to UI.
  - `notAuthenticated` or account mismatch: block writes until `CloudAccountStateService` validates the same account.

- [ ] Step 6: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudSyncEngineControllerTests -derivedDataPath /tmp/MomentoDerivedData-cloud-sync-engine
  ```

- [ ] Step 7: Commit.

  ```bash
  git add Momento/Cloud MomentoTests
  git commit -m "feat: add cloud sync engine controller"
  ```

## Chunk 6: Cloud Library Registry and UI Enablement

**Purpose:** Connect cloud descriptors to the existing library picker without breaking local libraries.

**Files:**

- Modify: `Momento/Storage/LibraryAccessScope.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Features/Library/MomentoCreateLibraryDialog.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`
- Test: `MomentoTests/ArchitectureGuardTests.swift`

- [ ] Step 1: Add tests before enabling UI.

  Cover:

  - Existing local recent-library JSON still decodes.
  - Local library create/open/rename/delete still works.
  - Cloud create is blocked when account unavailable.
  - Cloud create is blocked when Phase 0 smoke status is missing.
  - Cloud create succeeds only when account is available and CloudKit controller is initialized.
  - iOS registry path excludes local-only libraries.

- [ ] Step 2: Add cloud create/open routes in `LibraryStore`.

  Route `.local` to existing package flow.

  Route `.cloud` to:

  - Validate account state.
  - Create catalog record.
  - Create library zone.
  - Save cloud descriptor.
  - Open repository snapshot.

- [ ] Step 3: Enable iCloud row in create dialog only behind a real availability state.

  Do not use a static `isDisabled: false`.

  The row must show unavailable when:

  - no iCloud account,
  - restricted account,
  - container/provisioning not working,
  - account mismatch,
  - CloudKit smoke gate failed.

- [ ] Step 4: Add sidebar cloud library switching.

  Only cloud descriptors for the current cloud account should be visible.

- [ ] Step 5: Run focused tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests -only-testing:MomentoTests/ArchitectureGuardTests -derivedDataPath /tmp/MomentoDerivedData-cloud-ui
  ```

- [ ] Step 6: Commit.

  ```bash
  git add Momento MomentoTests
  git commit -m "feat: enable gated cloud libraries"
  ```

## Chunk 7: Metadata Sync

**Purpose:** Sync cloud-library metadata before original-file transfer.

**Files:**

- Modify: `Momento/Cloud/CloudLibraryRepository.swift`
- Modify: `Momento/Cloud/CloudLibrarySchema.swift`
- Modify: `Momento/Cloud/CloudSyncEngineController.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Test: `MomentoTests/CloudLibraryRepositoryTests.swift`
- Test: `MomentoTests/CloudSyncEngineControllerTests.swift`

- [ ] Step 1: Add two-cache fake sync tests.

  Simulate Mac and iOS with separate repositories and a fake CloudKit record store.

  Cover:

  - Rename cloud library.
  - Rename asset.
  - Edit note.
  - Favorite/unfavorite.
  - Folder create/rename/move.
  - Tag create/rename.
  - Different membership add/add converges.
  - Same membership add/remove converges by last accepted save.
  - Folder delete prevents membership from resurrecting the folder.

- [ ] Step 2: Implement record materialization.

  Each dirty local row must produce a CKRecord based on encoded system fields or deterministic record ID.

- [ ] Step 3: Implement remote application.

  Fetched remote records must update local cache through `CloudLibraryRepository`, not directly from the sync delegate.

- [ ] Step 4: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudLibraryRepositoryTests -only-testing:MomentoTests/CloudSyncEngineControllerTests -derivedDataPath /tmp/MomentoDerivedData-cloud-metadata
  ```

- [ ] Step 5: Run real-device metadata test.

  Required evidence:

  - Mac creates cloud library, iOS sees it.
  - iOS creates cloud library, Mac sees it.
  - Rename, folder, tag, trash, and note changes sync both directions.

- [ ] Step 6: Commit.

  ```bash
  git add Momento MomentoTests docs
  git commit -m "feat: sync cloud library metadata"
  ```

## Chunk 8: Blob Upload, Download, and Original Resolution

**Purpose:** Add image import and original-file access without pretending remote files are local.

**Files:**

- Create: `Momento/Cloud/CloudAssetEligibility.swift`
- Create: `Momento/Cloud/CloudOriginalFileResolver.swift`
- Modify: `Momento/Services/AssetImportService.swift`
- Modify: `Momento/Services/AssetExportService.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Test: `MomentoTests/CloudAssetEligibilityTests.swift`
- Test: `MomentoTests/CloudOriginalFileResolverTests.swift`
- Test: `MomentoTests/AssetExportServiceTests.swift`

- [ ] Step 1: Define measured v1 import limits.

  Do not ship seed values without Phase 0 measurement.

  Required values:

  - Allowed UTTypes/extensions.
  - Max file size.
  - Max asset count per cloud library.
  - Max cloud libraries per account if needed.

- [ ] Step 2: Test eligibility.

  Cover unreadable file, unsupported type, oversized file, path outside selected source, and valid common image.

- [ ] Step 3: Implement cloud import.

  Required order:

  - Validate eligibility.
  - Copy original into cloud cache.
  - Compute SHA-256.
  - Generate thumbnail.
  - Write `CloudAsset` and `CloudAssetBlob` dirty state in one transaction.
  - Queue pending changes.

- [ ] Step 4: Implement CKAsset upload.

  `CKAsset(fileURL:)` must point to the canonical cloud-cache file and the file must not be evicted while upload is pending.

- [ ] Step 5: Implement CKAsset download.

  During fetched-record handling:

  - Copy staged asset file to canonical cache path immediately.
  - Recompute SHA-256.
  - If hash matches, mark original local.
  - If hash mismatches, discard file and keep retryable error state.

- [ ] Step 6: Implement cloud original resolver.

  Preview/export/drag must:

  - Return local path if original is local.
  - Trigger download if original is remote-only.
  - Surface download failure.
  - Never fabricate `storageURL`.

- [ ] Step 7: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudAssetEligibilityTests -only-testing:MomentoTests/CloudOriginalFileResolverTests -only-testing:MomentoTests/AssetExportServiceTests -derivedDataPath /tmp/MomentoDerivedData-cloud-blobs
  ```

- [ ] Step 8: Run real-device blob tests.

  Required evidence:

  - Mac import appears on iOS.
  - iOS import appears on Mac.
  - Remote-only original downloads on preview/export.
  - Offline import stays pending and later uploads.
  - Quota/upload failure is visible and durable.

- [ ] Step 9: Commit.

  ```bash
  git add Momento MomentoTests docs
  git commit -m "feat: sync cloud library originals"
  ```

## Chunk 9: Account Changes, Offline States, and Quota UI

**Purpose:** Make the cloud feature honest under real CloudKit failure modes.

**Files:**

- Modify: `Momento/Services/CloudAccountStateService.swift`
- Modify: `Momento/Cloud/CloudSyncEngineController.swift`
- Modify: `Momento/Cloud/CloudLibraryRepository.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Test: `MomentoTests/CloudAccountStateServiceTests.swift`
- Test: `MomentoTests/CloudSyncEngineControllerTests.swift`

- [ ] Step 1: Test account transitions.

  Cover:

  - available account opens writable.
  - no account blocks cloud create and writes.
  - restricted account blocks cloud create and writes.
  - temporarily unavailable keeps cache visible but read-only.
  - account mismatch blocks writes and does not clear dirty data.

- [ ] Step 2: Observe account changes app-wide.

  Revalidate on:

  - `CKAccountChanged`
  - `NSUbiquityIdentityDidChange`
  - app launch
  - app foreground

- [ ] Step 3: Add visible sync state.

  Minimum states:

  - synced
  - syncing
  - waiting for network
  - waiting for iCloud sign-in
  - account mismatch
  - upload failed
  - download failed
  - quota blocked

- [ ] Step 4: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudAccountStateServiceTests -only-testing:MomentoTests/CloudSyncEngineControllerTests -derivedDataPath /tmp/MomentoDerivedData-cloud-account
  ```

- [ ] Step 5: Commit.

  ```bash
  git add Momento MomentoTests
  git commit -m "fix: harden cloud account sync states"
  ```

## Chunk 10: Local-to-Cloud Copy and Cloud Export

**Purpose:** Let users move data intentionally without mutating local packages in place.

**Files:**

- Create: `Momento/Cloud/CloudLibraryCopyService.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Storage/LibraryStorage.swift`
- Test: `MomentoTests/CloudLibraryCopyServiceTests.swift`
- Test: `MomentoTests/LibraryPackagePersistenceTests.swift`

- [ ] Step 1: Implement cancellable local-to-cloud copy for small libraries first.

  Required behavior:

  - Creates a new cloud library ID.
  - Original local package remains unchanged.
  - Uses existing content hashes.
  - Writes cloud cache state incrementally.
  - Can resume after app relaunch.

- [ ] Step 2: Keep large-library copy disabled until measured.

  If library exceeds measured v1 limits, show a blocking message instead of starting a doomed upload.

- [ ] Step 3: Implement cloud export to local `.momento`.

  Mac-only in v1. Download required originals first. Fail visibly if required originals cannot be fetched.

- [ ] Step 4: Run tests.

  ```bash
  xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/CloudLibraryCopyServiceTests -only-testing:MomentoTests/LibraryPackagePersistenceTests -derivedDataPath /tmp/MomentoDerivedData-cloud-copy
  ```

- [ ] Step 5: Commit.

  ```bash
  git add Momento MomentoTests
  git commit -m "feat: add cloud library copy and export"
  ```

## Final Validation Matrix

Run before declaring the feature complete:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-cloud-full
git diff --check
```

Manual matrix:

- Two Macs or Mac plus iOS device signed into the same Apple ID.
- Mac creates cloud library, iOS sees it.
- iOS creates cloud library, Mac sees it.
- Mac imports image, iOS previews it.
- iOS imports image, Mac previews it.
- Offline edit on one device syncs after reconnect.
- Concurrent rename resolves without data loss.
- Same membership add/remove converges.
- Account sign-out blocks writes.
- Account switch blocks previous account cache writes.
- Quota or upload failure remains visible and durable.
- Local `.momento` create/open/import/export/delete still works.

## Self Review

### Finding 1: The request says "implement everything", but the repository lacks an iOS target.

Risk: pretending this can be fully completed in the current macOS-only project would produce a Mac-only CloudKit path that does not prove the user's Mac/iOS requirement.

Fix in plan: iOS is tracked as a separate repo at `/Users/seaony/code/Momento-iOS`; Phase 0 now requires cross-repo compatibility checks plus real Mac/iOS smoke tests before cloud library UI is enabled.

### Finding 2: Entitlements and provisioning are external release gates.

Risk: directly editing `Momento.entitlements` with a guessed container can break signing or create code that compiles only on this machine.

Fix in plan: container ID and provisioning are hard gates. Code must stop if the values are unconfirmed.

### Finding 3: CKSyncEngine can be over-abstracted.

Risk: building both CKSyncEngine and a manual operation-token transport would double the implementation and test matrix.

Fix in plan: v1 uses CKSyncEngine only. Manual sync requires a separate design only if the chosen iOS target cannot use CKSyncEngine.

### Finding 4: A generic storage abstraction would be premature.

Risk: wrapping local packages and cloud cache behind a large shared interface before both paths exist would churn stable local code.

Fix in plan: local package classes stay stable. Cloud gets a separate repository and `LibraryStore` routes by storage mode.

### Finding 5: Metadata sync without blob state would create broken previews.

Risk: other devices could show an asset with no honest original availability.

Fix in plan: cloud read models must carry file availability, and preview/export/drag must go through `CloudOriginalFileResolver`.

### Finding 6: Local timestamps are not a conflict authority.

Risk: device clock skew can make older edits win.

Fix in plan: conflict handling uses CloudKit change tags and merges local dirty fields into the server record on `serverRecordChanged`.

### Finding 7: Upload-pending originals must be durable.

Risk: excluding upload-pending originals from backup or storing them in purgeable cache can lose the user's only copy.

Fix in plan: Application Support is required; upload-pending originals are not backup-excluded until upload success.

### Finding 8: The first cloud release needs measured limits.

Risk: enabling arbitrary large libraries creates a "correct but unusable" sync system.

Fix in plan: file size, library count, asset count, and total byte limits are Phase 0/Chunk 8 gates.

### Finding 9: The plan could still be too broad for one branch.

Risk: ten chunks may produce a long-running feature branch and harder review.

Fix in plan: every chunk has a concrete commit boundary and validation command. Do not batch multiple chunks into one unreviewed change.

### Verdict

This plan is executable only if Phase 0 external gates are satisfied first. It is intentionally not a promise that the current repository can become a complete Mac/iOS CloudKit product without adding or confirming an iOS target, signing configuration, provisioning, and real-device CloudKit evidence.
