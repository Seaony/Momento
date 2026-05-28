# P0 P1 Core Gap Implementation Plan

> Status: historical execution plan. It records the implementation context from 2026-05-23; current behavior and constraints are defined by `README.md`, `AGENTS.md`, and the current review docs.

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close Momento's P0/P1 product gaps without turning the app into a broad Eagle clone: soft trash, persistent notes, drag organization, first-class tags, folder hierarchy import, and library import/export.

**Architecture:** Make Core Data the single source of truth before expanding UI workflows. Land model changes first, then wire user workflows on top of stable APIs. Keep derived data rebuildable and postpone P2 systems such as persistent SearchIndex and ThumbnailRecord until the model is stable.

**Tech Stack:** Swift 6, SwiftUI, AppKit `NSCollectionView`, Core Data lightweight migration, UniformTypeIdentifiers, ImageIO.

---

## Review Decisions

This revision bakes in the technical review fixes needed before implementation. Remaining product confirmations are listed in the review checklist:

1. **Notes UI:** Notes must not return to the right inspector. This plan persists `Asset.note` for data correctness and future use, but does not add a Notes section back to the UI.
2. **Core Data migration:** Use exactly one new model version for this pass. Add all v3 fields and entities in that one version, then keep API/UI work in later commits without changing the schema again. `updatedAt` must not be a migration trap: the persistent field is optional in v3 and app state maps missing values to `importedAt`.
3. **Tag migration:** Keep `tagsData` as a temporary legacy field in v3, backfill `TagRecord`/`AssetTagRecord`, and ignore `tagsData` afterward. Preserve legacy tag IDs during backfill; new tags may use generated stable IDs, and UI/tests must stop deriving tag IDs from names.
4. **Trash duplicate imports:** Re-importing a file whose content hash belongs to a trashed asset restores that asset instead of silently skipping the import.
5. **Library import semantics:** "Open Library" references a package in place. "Import Library" copies an existing `.momento`/legacy `.momentolibrary` package, validates it, saves a security-scoped recent-library bookmark for the copied package, and refuses duplicate library IDs until a future "duplicate as new library" migration exists.
6. **Multi-format import:** SVG/PDF/video import is explicitly out of scope for P0/P1. Keep the current image/GIF scope and do not add placeholder model or thumbnail work just for future formats.

## References Checked

- Apple Core Data automatic migration: lightweight migration can infer common model changes; nonoptional additions need defaults, and larger changes should be staged.
- Apple AppKit collection view drag/drop: `NSCollectionViewDelegate` provides `collectionView(_:pasteboardWriterForItemAt:)` for drag sources.
- Apple `NSFilePromiseProvider`: use one provider per promised file when dragging files from the app to Finder or other apps, and write promised files in the delegate rather than moving source files.
- Apple UniformTypeIdentifiers: use UTTypes to classify importable files and pasteboard/file-promise content.
- Eagle API references confirm folders, tags, extension filtering, rating, annotation, URL/bookmark, and trash are core comparison dimensions, but this plan only covers P0/P1.

## Non-Goals

- No scoring/rating in this plan.
- No smart folders or saved searches.
- No perceptual duplicate detection.
- No SVG/PDF/video multi-format import in this pass.
- No browser extension, screenshot capture, Finder Sync, watched folders, cloud sync, or image annotation.
- No persistent `SearchIndex` or `ThumbnailRecord` yet.
- No major visual redesign beyond UI needed to expose the new behavior.

---

## Chunk 1: Core Data Model Alignment

### Task 1: Add v3 model fields for real asset metadata

**Files:**
- Modify: `Momento/Storage/MomentoModel.xcdatamodeld`
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Modify: `Momento/Storage/MomentoCoreDataStack.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Add failing persistence tests**

Add tests that import one image, reload the library, and assert:
- `originalFileName` is preserved separately from `displayName`.
- `utiIdentifier` is stored.
- `note` hydrates as `nil` by default. Do not test note writes in Task 1.
- `isTrashed == false` by default.
- `updatedAt` exists, falls back to `importedAt` for old rows, and changes on title/favorite updates. Do not test note/tag-driven `updatedAt` until Task 2/Task 4.
- A library created with the current pre-v3 model opens under v3 without migration failure.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testImportedAssetPersistsCoreMetadata
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testOpeningPreV3LibraryMigratesMetadata
```

Expected before implementation: FAIL because fields do not exist.

Pre-v3 migration fixture rule:

- Build the fixture with `MomentoModel v2.xcdatamodel` in the test helper before opening it with the v3 stack, or check in a tiny SQLite fixture created from the current v2 model.
- Do not create the "pre-v3" fixture through the normal `MomentoCoreDataStack` after v3 becomes current; that would only test a fresh v3 library.
- The fixture should contain at least one `AssetRecord` with `tagsData`, one color record, and one folder membership so v3 opening validates old rows do not break. Task 2 owns assertions that legacy `tagsData` is backfilled into `TagRecord`/`AssetTagRecord`.

- [ ] **Step 2: Create one Core Data model version v3**

Create `MomentoModel v3.xcdatamodel` and set `.xccurrentversion` to v3.

Schema rule: this plan must not create v4 just to add tags. Add every new field/entity needed by Chunk 1 in v3 before committing schema changes.

Add to `AssetRecord`:

- `originalFileName: String`, nonoptional, default `""`
- `utiIdentifier: String`, nonoptional, default `public.data`
- `orientation: Integer 64`, optional
- `colorProfileName: String`, optional
- `note: String`, optional
- `isTrashed: Boolean`, nonoptional, default `NO`
- `trashedAt: Date`, optional
- `updatedAt: Date`, optional in Core Data v3

Keep existing fields for compatibility:

- `tagsData` remains temporarily as legacy data.
- `exifMetadataData` remains as the serialized EXIF payload.

Also add the v3 tag entities now so Task 2 can wire behavior without another model version:

`TagRecord`
- `id: String`
- `libraryID: String`
- `name: String`
- `normalizedName: String`
- `colorHex: String`, optional
- `createdAt: Date`
- `updatedAt: Date`
- uniqueness: `libraryID + normalizedName`

`AssetTagRecord`
- `id: String`
- `libraryID: String`
- `assetID: String`
- `tagID: String`
- `createdAt: Date`
- uniqueness: `libraryID + assetID + tagID`

- [ ] **Step 3: Verify lightweight migration assumptions**

Because this adds optional fields, new entities, and nonoptional fields with defaults, it should stay in lightweight migration territory. Do not trust this assumption without opening a pre-v3 library in a test.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testImportPersistsAndDeduplicatesAssets
```

Expected after v3 model wiring: PASS.

- [ ] **Step 4: Extend `AssetItem`**

Add fields to `AssetItem`:

```swift
var originalFileName: String
var utiIdentifier: String
var orientation: Int?
var colorProfileName: String?
var note: String?
var isTrashed: Bool
var trashedAt: Date?
var updatedAt: Date
```

Update initializers and all call sites. Avoid optional `updatedAt` in app state; old data and nil Core Data values must fall back to `importedAt` during mapping.

- [ ] **Step 5: Map v3 fields in `LibraryMetadataStore`**

Update:

- `saveImportedAssets(_:)`
- `asset(from:folderIDs:paletteColors:)`
- `renameAsset(id:to:)`
- `setFavorite(_:forAssetID:)`
- later tag/note/trash mutators

Rules:

- `originalFileName` is the source file's full last path component, not the title without extension.
- `displayName` remains the editable title.
- `utiIdentifier` comes from `UTType(filenameExtension:)?.identifier`, with `public.data` fallback only if UTType cannot be inferred.
- In Task 1, `updatedAt` changes for title and favorite updates only. Task 2 covers tag updates; Task 4 covers note updates; Task 3 covers trash updates.
- Existing records whose `updatedAt` is nil hydrate as `updatedAt = importedAt`, then get a real `updatedAt` the next time metadata changes.

- [ ] **Step 6: Commit**

```bash
git add Momento/Storage/MomentoModel.xcdatamodeld Momento/Core/AssetModels.swift Momento/Storage/LibraryMetadataStore.swift Momento/Storage/MomentoCoreDataStack.swift MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: align asset metadata model"
```

### Task 2: Promote tags to first-class records

**Files:**
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Add failing tag entity tests**

Add tests for:

- Creating tags through inspector persists `TagRecord`.
- Renaming a tag updates one `TagRecord` and keeps `AssetTagRecord` links.
- Deleting a tag removes links but does not delete assets.
- Two tags that normalize to the same lowercase trimmed name cannot coexist.
- Tests resolve tag IDs from `store.tags`/`tagSummaries`; do not assume `name.lowercased()` is the ID for newly created tags.
- Removing the last asset link from a tag keeps the `TagRecord` with `assetCount == 0`.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testTagRecordsRenameAndDeleteAcrossAssets
```

Expected before implementation: FAIL.

- [ ] **Step 2: Use the v3 tag schema without changing the model again**

Task 1 already added `TagRecord` and `AssetTagRecord` to v3. Do not create v4 for this task.

ID rules:

- During legacy backfill, preserve `TagItem.id` from `tagsData` so existing sidebar selections like `tag-\(id)` keep working.
- New tags get generated stable string IDs; do not derive new IDs from tag names.
- Renaming a tag changes `name`, `normalizedName`, and `updatedAt`, but never changes `id`.
- `TagItem.init(name:)` can keep its current default for sample/in-memory compatibility, but persistent tag creation must go through `LibraryMetadataStore`.

Lifecycle rules:

- `TagRecord` is a first-class user-managed object. Removing the last `AssetTagRecord` link does not delete the tag.
- `deleteTag(id:)` is the only P1 operation that deletes a `TagRecord`; it also removes all `AssetTagRecord` links for that tag.
- `tagSummaries` and the tag management page should show zero-count tags. The Add Tag picker should also be able to show zero-count tags.

- [ ] **Step 3: Add tag APIs in `LibraryMetadataStore`**

Replace blob mutation with record mutation:

```swift
func loadTags() throws -> [TagItem]
func resolveOrCreateTags(named names: [String]) throws -> [TagItem]
func setTagNames(_ names: [String], forAssetID assetID: AssetItem.ID) throws -> AssetItem
func renameTag(id tagID: TagItem.ID, to name: String) throws -> [AssetItem]
func deleteTag(id tagID: TagItem.ID) throws -> [AssetItem]
```

`loadAssets()` should fetch asset tag links and hydrate `AssetItem.tags` from `TagRecord`.

Rules:

- Do not expose a primary mutation API that asks callers to construct `[TagItem]` for names. That repeats the current bug where `TagItem(name:)` fabricates `name.lowercased()` IDs.
- `ContentView.selectedTags` and `LibraryStore.updateSelectedTags(_:)` may still traffic in names, but `LibraryMetadataStore` must resolve those names to real persisted tag IDs.
- Task 6 can add the bulk `addTag(id:toAssets:)` API once drag/drop needs it.

- [ ] **Step 4: Backfill legacy `tagsData` idempotently**

On library open/save path, run a private migration:

```swift
private func migrateLegacyTagsIfNeeded() throws
```

Rules:

- If `AssetTagRecord` already exists for an asset, do not duplicate links.
- Decode `tagsData` only as migration input.
- Create missing `TagRecord`s by normalized name.
- If multiple legacy blobs contain the same normalized name with different IDs, pick the first stable ID deterministically and rewrite links to that one `TagRecord`.
- Keep `tagsData` untouched for one model version but stop writing new values.
- Backfilled tags remain even if later all links are removed; do not treat zero linked rows as orphan cleanup.

- [ ] **Step 5: Update `LibraryStore` tag state**

`LibraryStore.tags`, `tagSummaries`, `updateSelectedTags`, `renameTag`, and `deleteTag` should rely on `TagRecord`-backed data. Keep sidebar IDs as `tag-\(id)`, but make callers use actual tag IDs returned by the store. Update tests that currently call `renameTag(id: "mood", ...)` unless the test is explicitly verifying legacy backfill preserves an existing legacy ID.

- [ ] **Step 6: Commit**

```bash
git add Momento/Core/AssetModels.swift Momento/Storage/LibraryMetadataStore.swift Momento/Core/LibraryStore.swift MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: store tags as first-class records"
```

---

## Chunk 2: P0 Data Safety

### Task 3: Replace hard delete asset trash with soft trash

**Files:**
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Modify: `Momento/Storage/LibraryStorage.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Modify: `Momento/AppKitBridge/AssetCollectionGridView.swift`
- Modify: `Momento/Localizable.xcstrings`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Replace current trash test**

Replace `testMovingImportedAssetToTrashRemovesMetadataAndStoredFile` with tests for:

- `moveAssetToTrash` marks `isTrashed = true` and sets `trashedAt`.
- Default visible assets exclude trashed assets.
- Trash sidebar shows trashed assets.
- Restore clears `isTrashed`/`trashedAt`.
- Folder membership survives trash/restore.
- Empty trash removes metadata, tag links, colors, memberships, asset file, thumbnail, and preview cache files.
- Re-importing the same file while its asset is in Trash restores the existing asset instead of silently skipping it.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testMovingAssetToTrashSoftDeletesAndRestores
```

Expected before implementation: FAIL because metadata/file are deleted immediately.

- [ ] **Step 2: Add store APIs**

In `LibraryMetadataStore`:

```swift
func moveAssetToTrash(id assetID: AssetItem.ID) throws -> AssetItem
func restoreAssets(ids: Set<AssetItem.ID>) throws -> [AssetItem]
func emptyTrash() throws -> [AssetItem.ID]
func duplicateAssetReferences(forContentHashes hashes: Set<String>, includeTrashed: Bool) throws -> [String: DuplicateAssetReference]
```

Add a small value type:

```swift
struct DuplicateAssetReference: Hashable {
    var assetID: AssetItem.ID
    var isTrashed: Bool
}
```

Rules:

- Moving to trash does not delete the asset file.
- Moving to trash does not delete folder memberships.
- Empty trash performs the destructive cleanup, including `assets/`, `thumbnails/`, and any preview files under `previews/` for each deleted content hash.
- `trashedAt` is set once per move-to-trash action.
- Duplicate import resolution must be metadata-driven, not hidden inside `AssetImportService`'s current `existingContentHashes` skip.
- If a duplicate hash maps to a trashed asset, restore that asset, merge any requested folder membership, and return the restored asset through `mergeAssets`.

- [ ] **Step 3: Update `LibraryStore` view scoping**

Rules:

- `.trash` selection returns only trashed assets.
- `.all`, `.favorites`, `.unfiled`, `.untagged`, `.tag`, `.folder`, search and filters exclude trashed assets.
- Selection clears if the selected asset leaves the current scope.
- `selectedAsset` should not expose a trashed asset while the current scope excludes trash.
- Command palette "Move to Trash" remains soft delete.
- Add restore and empty trash actions in store first; UI can expose them in the same task or next task.

- [ ] **Step 4: Update UI affordances**

Minimum P0 UI:

- Trash sidebar count is real.
- Context menu in Trash offers "Restore" and "Delete Permanently" or "Empty Trash".
- Non-Trash context menu keeps "Move to Trash".
- Do not hide toolbar chrome during delete/empty dialogs.
- Add/update localized strings for restore, delete permanently, and empty trash.

- [ ] **Step 5: Commit**

```bash
git add Momento/Core/LibraryStore.swift Momento/Storage/LibraryMetadataStore.swift Momento/Storage/LibraryStorage.swift Momento/Features/Sidebar/MomentoSidebarView.swift Momento/AppKitBridge/AssetCollectionGridView.swift Momento/Localizable.xcstrings MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: soft delete trashed assets"
```

### Task 4: Persist inspector notes

**Files:**
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/ContentView.swift` only if stale local notes state is still wired
- Test: `MomentoTests/ImportServiceSmokeTests.swift`
- Test: `MomentoTests/ArchitectureGuardTests.swift`

- [ ] **Step 1: Add failing note tests**

Add tests for:

- Update selected asset note.
- Reload library and verify note persists.
- Switch between two selected assets through store APIs and verify each asset keeps its own note.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testUpdatingAssetNotePersistsAcrossReloads
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ArchitectureGuardTests/testInspectorDoesNotExposeNotesEditor
```

Expected before implementation: FAIL because notes are local UI state or missing from persistent metadata.

Add `testInspectorDoesNotExposeNotesEditor` as a source-level guard in `MomentoTests/ArchitectureGuardTests.swift`; it asserts no inspector Notes section/editor is reintroduced. Do not put UI source checks in `ImportServiceSmokeTests`.

- [ ] **Step 2: Add metadata write API**

In `LibraryMetadataStore`:

```swift
func updateNote(_ note: String?, forAssetID assetID: AssetItem.ID) throws -> AssetItem
```

Normalize empty/whitespace-only strings to `nil`. Update `updatedAt`.

- [ ] **Step 3: Add LibraryStore API**

```swift
func updateSelectedNote(_ note: String) throws
func updateNote(_ note: String, forAssetID assetID: AssetItem.ID) throws
```

Use `mergeAssets` to update in-memory state without reloading the whole library.

- [ ] **Step 4: Remove fake local notes state if present**

Remove `ContentView.inspectorNotesByAssetID` and any stale binding that suggests notes are saved when they are not. Do not add a visible notes editor.

- [ ] **Step 5: Keep Notes out of the inspector**

Do not add `notesEditor` back into `MomentoInspectorView`. The only UI-related work in this task is removing misleading local state if it still exists.

- [ ] **Step 6: Commit**

```bash
git add Momento/Core/AssetModels.swift Momento/Storage/LibraryMetadataStore.swift Momento/Core/LibraryStore.swift Momento/ContentView.swift MomentoTests/ImportServiceSmokeTests.swift MomentoTests/ArchitectureGuardTests.swift
git commit -m "feat: persist asset notes"
```

---

## Chunk 3: P0 Drag Organization

### Task 5: Add asset drag source and file promises

**Files:**
- Create: `Momento/AppKitBridge/AssetDragPasteboardWriter.swift`
- Create: `Momento/AppKitBridge/AssetFilePromiseProvider.swift`
- Create: `Momento/AppKitBridge/AssetDragPasteboardItem.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/ContentView.swift`
- Modify: `Momento/AppKitBridge/AssetCollectionGridView.swift`
- Test: `MomentoTests/ArchitectureGuardTests.swift`

- [ ] **Step 1: Add real multi-selection state**

Current `NSCollectionView` can publish a `Set<AssetItem.ID>`, but `ContentView` collapses it to the first ID. Fix this before drag work.

Rules:

- Add `selectedAssetIDs: Set<AssetItem.ID>` to `LibraryStore`.
- Keep `selectedAssetID` or add `primarySelectedAssetID` only as the inspector/preview anchor.
- `ContentView.selectAssets(_:)` stores the full set and updates the primary anchor deterministically.
- Scope changes, trash, search, folder/tag filters, and library switching prune the full selected set, not only one ID.

- [ ] **Step 2: Add coarse architecture guard**

Add a source-level test asserting the behavior has a native AppKit drag path without over-specifying helper names:

- `AssetCollectionGridView` implements `collectionView(_:canDragItemsAt:with:)`.
- It implements `collectionView(_:pasteboardWriterForItemAt:)`.
- It references `NSFilePromiseProvider`.
- It defines an internal pasteboard type for Momento asset IDs.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ArchitectureGuardTests/testAssetGridSupportsDraggingAndFilePromises
```

Expected before implementation: FAIL.

- [ ] **Step 3: Create internal pasteboard payload**

`AssetDragPasteboardWriter` should encode:

- library ID
- ordered asset IDs
- primary asset ID

Use a custom pasteboard type:

```swift
static let assetIDsPasteboardType = NSPasteboard.PasteboardType("com.seaony.momento.asset-ids")
```

Also define the matching `UTType` string for SwiftUI/sidebar drop loading. Keep the encoded payload small JSON data; do not serialize full `AssetItem` values.

- [ ] **Step 4: Create file promise provider**

`AssetFilePromiseProvider` should use `NSFilePromiseProvider` with:

- `fileType` from `asset.utiIdentifier`
- `userInfo` containing source asset URL and display file name
- delegate writes a copy into destination URL

Rules:

- Dragging to Finder copies files out, never moves the library's content-addressed source file.
- File names use `displayName + "." + fileExtension`, resolving collisions at destination.
- Multi-select creates one provider per selected asset.
- File promise writes must use the destination URL passed by the delegate and call the completion handler with any error.

- [ ] **Step 5: Create a combined pasteboard item**

`collectionView(_:pasteboardWriterForItemAt:)` can return only one `NSPasteboardWriting` per item. Do not choose between internal IDs and file promises. Create `AssetDragPasteboardItem` that writes both representations:

- internal Momento asset-ID payload for app-internal drops
- `NSFilePromiseProvider`/file-promise representation for Finder and other apps

Implementation options:

- Preferred: subclass/wrap `NSFilePromiseProvider` only if it can also advertise/write the internal pasteboard type reliably.
- Otherwise create a custom `NSPasteboardWriting` object whose `writableTypes(for:)` includes the internal type and a file-promise type, delegating promise fulfillment to `AssetFilePromiseProvider`.
- If AppKit cannot support the combined writer cleanly, split behavior explicitly: use file promises for external dragging and use an AppKit sidebar drop target that can read the same dragging session's pasteboard. Do not leave one path broken silently.

- [ ] **Step 6: Wire drag source in `AssetCollectionGridView`**

Rules:

- Drag selected assets if the pointer starts on a selected item.
- If dragging an unselected item, select it first and drag only that item.
- Drag image should use existing selected item snapshots where practical.
- Long-press QuickLook must not conflict with drag start.

- [ ] **Step 7: Manual validation checklist**

Because AppKit dragging is hard to unit test fully, validate manually after implementation:

- Drag one asset to Finder creates a copied file.
- Drag multiple selected assets to Finder creates multiple copied files.
- Dragging does not remove assets from Momento.
- Drag start does not trigger the prior input-error sound.

- [ ] **Step 8: Commit**

```bash
git add Momento/AppKitBridge/AssetDragPasteboardWriter.swift Momento/AppKitBridge/AssetFilePromiseProvider.swift Momento/AppKitBridge/AssetDragPasteboardItem.swift Momento/Core/LibraryStore.swift Momento/ContentView.swift Momento/AppKitBridge/AssetCollectionGridView.swift MomentoTests/ArchitectureGuardTests.swift
git commit -m "feat: drag assets out with file promises"
```

### Task 6: Drop assets onto folders and tags

**Files:**
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Modify: `Momento/Features/Inspector/MomentoInspectorView.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`
- Test: `MomentoTests/ArchitectureGuardTests.swift`

- [ ] **Step 1: Add data tests**

Add tests for:

- Assign multiple assets to a folder from asset IDs.
- Add an existing tag to multiple assets.
- Drop duplicate membership is idempotent.
- Dropping trashed assets is rejected or ignored.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testAssigningMultipleDraggedAssetsToFolderIsIdempotent
```

- [ ] **Step 2: Add store bulk APIs**

```swift
func assignAssets(ids: Set<AssetItem.ID>, to folderID: AssetFolder.ID) throws
func addTag(id tagID: TagItem.ID, toAssets assetIDs: Set<AssetItem.ID>) throws
```

Keep these methods merge-based; do not reload the whole asset list.

- [ ] **Step 3: Add sidebar drop handling**

Sidebar folder rows and tag rows should accept the internal Momento pasteboard type.

Do not assume AppKit `NSPasteboardWriting` automatically works with SwiftUI's `dropDestination(for:)`. Choose one explicit bridge:

- Preferred: define a custom `UTType` and make `AssetDragPasteboardWriter` expose data that a SwiftUI `DropDelegate` can load through `NSItemProvider.loadDataRepresentation`.
- Fallback: add a small AppKit drop target wrapper for sidebar rows if SwiftUI cannot reliably read the AppKit pasteboard type.

Rules:

- Hover state makes the row visibly droppable.
- Drop on `.unfiled`, `.all`, `.favorites`, `.trash`, and management rows is not accepted.
- Drop on tag adds tag.
- Drop on folder adds membership.
- Multi-select is preserved after drop.

- [ ] **Step 4: Add inspector drop handling if useful**

If implementation remains small, allow dropping assets onto tag/folder chips in the inspector too. If this adds complexity, defer to sidebar-only drag organization.

- [ ] **Step 5: Manual validation checklist**

Because cross-view drag/drop is hard to test fully, validate manually after implementation:

- Drag one or more selected assets to a sidebar folder assigns those assets.
- Drag one or more selected assets to a sidebar tag links that tag.
- Drop hover state appears only on accepted targets.
- Dropping onto `.all`, `.favorites`, `.trash`, and management rows is rejected.
- Selection remains stable after a successful drop.

- [ ] **Step 6: Commit**

```bash
git add Momento/Features/Sidebar/MomentoSidebarView.swift Momento/Features/Inspector/MomentoInspectorView.swift Momento/Core/LibraryStore.swift Momento/Storage/LibraryMetadataStore.swift MomentoTests/ImportServiceSmokeTests.swift MomentoTests/ArchitectureGuardTests.swift
git commit -m "feat: organize dragged assets"
```

---

## Chunk 4: P1 Folder Import Model

### Task 7: Preserve folder hierarchy during folder import

**Files:**
- Modify: `Momento/Services/AssetImportService.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Add failing hierarchy import test**

Create a temp folder:

```text
Source/
├── Posters/
│   └── cover.png
└── References/
    └── nested.jpg
```

Import `Source/`, then assert:

- `Posters` and `References` folders exist.
- Imported assets have matching folder IDs.
- Reopening the library preserves folders and memberships.
- Importing the same folder again does not copy duplicate files but preserves idempotent folder membership.
- If a duplicate file is currently trashed, folder import restores it and assigns the matching folder.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testImportingFolderPreservesHierarchyAsVirtualFolders
```

Expected before implementation: FAIL because import flattens directory trees.

- [ ] **Step 2: Replace flat URL collection with import candidates**

Introduce an internal value:

```swift
struct AssetImportCandidate {
    var sourceURL: URL
    var rootURL: URL?
    var relativeFolderComponents: [String]
}
```

Rules:

- A selected file has no relative folder components.
- A selected folder preserves the path from selected folder root to each file's parent directory.
- Hidden files and package descendants remain skipped.

- [ ] **Step 3: Return/import a batch, not only assets**

Introduce:

```swift
struct AssetImportBatch {
    var newAssets: [AssetItem]
    var folderAssignmentsByContentHash: [String: [[String]]]
}
```

For duplicates:

- Do not copy the physical file again.
- Do create folder membership if a duplicate is imported through a folder hierarchy.
- Do not let `AssetImportService` drop duplicates before the metadata layer can resolve folder assignments.
- The metadata layer resolves each content hash to either a new asset or an existing asset, including trashed assets that should be restored.

- [ ] **Step 4: Add metadata store batch save**

Add:

```swift
func saveImportedBatch(_ batch: AssetImportBatch) throws -> [AssetItem]
```

Rules:

- Create missing folders by `(libraryID, parentID, name)`.
- Reuse existing folders with matching sibling name.
- Save new asset records first, then resolve `folderAssignmentsByContentHash` against both new and existing records.
- Preserve folder memberships for duplicates and restore duplicate trashed assets before returning.

- [ ] **Step 5: Commit**

```bash
git add Momento/Services/AssetImportService.swift Momento/Core/LibraryStore.swift Momento/Storage/LibraryMetadataStore.swift MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: preserve imported folder hierarchy"
```

## Chunk 5: P1 Library Portability

### Task 8: Add library import and export

**Files:**
- Modify: `Momento/Storage/LibraryStorage.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/ContentView.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Modify: `Momento/Features/CommandPalette/MomentoCommandPalette.swift`
- Modify: `Momento/Localizable.xcstrings`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Add storage tests**

Add tests for:

- Export copies the current package and preserves `manifest.json`, `database/library.sqlite`, `assets/`, `thumbnails/`, `previews/`.
- Import validates manifest and database before adding to recent libraries.
- Import refuses unknown manifest schema.
- Importing into an existing destination does not overwrite silently.
- Import refuses a package whose `libraryID` already exists in Recent Libraries or is the current library.
- Import/export keeps security-scoped access alive for the full copy/validation operation.

Run:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests/testExportingAndImportingLibraryPackages
```

- [ ] **Step 2: Add storage APIs**

In `LibraryStorage`:

```swift
func exportLibraryPackage(_ library: AssetLibrary, to destinationURL: URL) throws -> URL
func importLibraryPackage(from sourceURL: URL, to destinationRootURL: URL?) throws -> AssetLibrary
func validateLibraryPackage(at packageURL: URL) throws -> AssetLibrary
```

Rules:

- Export is copy-only.
- Import is copy-only unless product review decides "import" should reference in place.
- Never overwrite an existing package without explicit future UI.
- Validate `manifest.json` first, then open the Core Data stack enough to confirm the store is readable.
- Do not rewrite `manifest.libraryID` in P1. Rewriting the package ID would also require rewriting Core Data `libraryID` fields and is out of scope.
- Refuse duplicate library IDs for now instead of silently replacing the Recent Library bookmark.

- [ ] **Step 3: Add LibraryStore flows**

```swift
func exportCurrentLibrary(to destinationURL: URL) throws
func importLibrary(from sourceURL: URL, destinationRootURL: URL?) throws
```

Import should save a security-scoped bookmark for the copied package, append it to Recent Libraries, and switch to the imported library.

- [ ] **Step 4: Wire UI**

Minimum UI:

- Library menu: "Import Library..." and "Export Library...".
- Command palette entries for import/export.
- Use native file panels.
- Do not hide toolbar/sidebar during dialogs.
- Add/update localized strings for the new menu and command labels.

- [ ] **Step 5: Commit**

```bash
git add Momento/Storage/LibraryStorage.swift Momento/Core/LibraryStore.swift Momento/ContentView.swift Momento/Features/Sidebar/MomentoSidebarView.swift Momento/Features/CommandPalette/MomentoCommandPalette.swift Momento/Localizable.xcstrings MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: import and export libraries"
```

---

## Chunk 6: Final Validation And Cleanup

### Task 9: Full regression pass

**Files:**
- Modify only if validation exposes defects.

- [ ] **Step 1: Source check**

```bash
git diff --check
```

Expected: no output.

- [ ] **Step 2: Focused persistence tests**

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ImportServiceSmokeTests
```

Expected: `TEST SUCCEEDED`.

- [ ] **Step 3: Architecture guard tests**

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ArchitectureGuardTests
```

Expected: `TEST SUCCEEDED`.

- [ ] **Step 4: Full build**

```bash
xcodebuild build -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS'
```

Expected: `BUILD SUCCEEDED`.

- [ ] **Step 5: Manual app validation by user**

The agent must not launch the app. User should manually check:

- Trash: move, restore, empty.
- Notes: no inspector Notes section is visible; persistence is covered by automated tests.
- Drag: grid to Finder, grid to folder, grid to tag, multi-select.
- Folder import hierarchy.
- Library import/export.

- [ ] **Step 6: Final commit if fixes were needed**

```bash
git status --short
git commit -m "fix: stabilize p0 p1 core workflows"
```

Only commit if there are actual follow-up fixes.

## Recommended Execution Order

1. Chunk 1 first. Do not build UI features on top of the old tag/blob model.
2. Chunk 2 next. Trash and notes are user-trust issues.
3. Chunk 3 after soft trash. Drag actions should not be able to act on trashed assets accidentally.
4. Chunk 4 after drag. Folder hierarchy import depends on stable folder membership APIs.
5. Chunk 5 last. Library import/export should copy the final package structure and database schema.

## Risk Register

| Risk | Mitigation |
|---|---|
| Core Data migration failure | Keep v3 additions lightweight where possible; add migration tests before UI work. |
| Tag backfill duplicates tags | Normalize by trimmed lowercase name and enforce `libraryID + normalizedName` uniqueness. |
| Tag IDs change unexpectedly | Preserve legacy IDs during backfill and make new code use actual returned IDs instead of name-derived IDs. |
| Zero-count tags disappear unexpectedly | Treat `TagRecord` as user-managed; only explicit delete removes it. |
| Trash empty deletes files that restore still needs | Only delete physical files in `emptyTrash`; soft trash never moves/removes asset files. |
| Trash empty leaves preview cache files | Delete thumbnails and preview files for each permanently deleted content hash. |
| Re-import of trashed duplicate appears to do nothing | Resolve duplicates in metadata and restore trashed duplicate assets on import. |
| Notes cause grid flash | Merge single updated asset into memory; do not reload library on note writes; do not add a visible inspector editor. |
| Multi-select gets lost before drag/drop | Add real selection-set state before file promise and sidebar drop work. |
| Drag/drop hard to unit test | Add source-level guard tests and explicit manual checklist. |
| SwiftUI drop cannot read AppKit pasteboard writer | Use an explicit `UTType`/`NSItemProvider` bridge, or a small AppKit drop target wrapper if needed. |
| Library export accidentally overwrites | Refuse existing destination in P1. |
| Importing a copied library replaces an existing recent entry | Refuse duplicate library IDs until a future package-clone migration can rewrite manifest and Core Data library IDs together. |

## Review Checklist

- [ ] Approve keeping `tagsData` as a temporary legacy field for one model version.
- [ ] Approve "Import Library" as copy-in rather than open-in-place.
- [ ] Approve refusing duplicate library IDs during import rather than cloning them as new libraries.
- [ ] Approve not doing P2 features in this pass.
