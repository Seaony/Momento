# P0 P1 Core Gap Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close Momento's P0/P1 product gaps without turning the app into a broad Eagle clone: soft trash, persistent notes, drag organization, first-class tags, folder hierarchy import, and library import/export.

**Architecture:** Make Core Data the single source of truth before expanding UI workflows. Land model changes first, then wire user workflows on top of stable APIs. Keep derived data rebuildable and postpone P2 systems such as persistent SearchIndex and ThumbnailRecord until the model is stable.

**Tech Stack:** Swift 6, SwiftUI, AppKit `NSCollectionView`, Core Data lightweight migration, UniformTypeIdentifiers, ImageIO.

---

## Review Decisions

The plan below is implementable as written, but these product choices should be confirmed before execution:

1. **Notes UI:** Notes were previously removed from the inspector UI. This plan persists `Asset.note` and reintroduces a compact editable Notes section only if review approves it. If not approved, land the data/API layer and keep the UI hidden.
2. **Tag migration:** Keep `tagsData` as a temporary legacy field in the next model version, backfill `TagRecord`/`AssetTagRecord`, and ignore `tagsData` afterward. Removing `tagsData` can be a later cleanup after the new model is proven.
3. **Library import semantics:** "Open Library" references a package in place. "Import Library" copies an existing `.momento`/legacy `.momentolibrary` package into a chosen destination or app-managed location, validates it, and adds it to Recent Libraries.
4. **Multi-format import:** SVG/PDF/video import is explicitly out of scope for P0/P1. Keep the current image/GIF scope and do not add placeholder model or thumbnail work just for future formats.

## References Checked

- Apple Core Data automatic migration: lightweight migration can infer common model changes; nonoptional additions need defaults, and larger changes should be staged.
- Apple AppKit collection view drag/drop: `NSCollectionViewDelegate` provides `canDragItemsAt`, `pasteboardWriterForItemAt`, `validateDrop`, and `acceptDrop`.
- Apple `NSFilePromiseProvider`: use file promises when dragging files from the app to Finder or other apps.
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
- `note` can be persisted after a later task adds the write API.
- `isTrashed == false` by default.
- `updatedAt` exists and changes on title/note/tag updates.

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testImportedAssetPersistsCoreMetadata
```

Expected before implementation: FAIL because fields do not exist.

- [ ] **Step 2: Create Core Data model version v3**

Create `MomentoModel v3.xcdatamodel` and set `.xccurrentversion` to v3.

Add to `AssetRecord`:

- `originalFileName: String`, nonoptional, default `""`
- `utiIdentifier: String`, nonoptional, default `public.data`
- `orientation: Integer 64`, optional
- `colorProfileName: String`, optional
- `note: String`, optional
- `isTrashed: Boolean`, nonoptional, default `NO`
- `trashedAt: Date`, optional
- `updatedAt: Date`, nonoptional

Keep existing fields for compatibility:

- `tagsData` remains temporarily as legacy data.
- `exifMetadataData` remains as the serialized EXIF payload.

- [ ] **Step 3: Verify lightweight migration assumptions**

Because this adds mostly optional fields and nonoptional fields with defaults, it should stay in lightweight migration territory.

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testImportPersistsAndDeduplicatesAssets
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

Update initializers and all call sites. Avoid optional `updatedAt` in app state; old data should fall back to `importedAt` during mapping.

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
- `updatedAt` changes whenever user-visible metadata changes.

- [ ] **Step 6: Commit**

```bash
git add Momento/Storage/MomentoModel.xcdatamodeld Momento/Core/AssetModels.swift Momento/Storage/LibraryMetadataStore.swift Momento/Storage/MomentoCoreDataStack.swift MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: align asset metadata model"
```

### Task 2: Promote tags to first-class records

**Files:**
- Modify: `Momento/Storage/MomentoModel.xcdatamodeld`
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

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testTagRecordsRenameAndDeleteAcrossAssets
```

Expected before implementation: FAIL.

- [ ] **Step 2: Add `TagRecord` and `AssetTagRecord`**

Add entities:

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

Use string UUIDs to stay consistent with existing value types.

- [ ] **Step 3: Add tag APIs in `LibraryMetadataStore`**

Replace blob mutation with record mutation:

```swift
func loadTags() throws -> [TagItem]
func updateTags(_ tags: [TagItem], forAssetID assetID: AssetItem.ID) throws -> AssetItem
func renameTag(id tagID: TagItem.ID, to name: String) throws -> [AssetItem]
func deleteTag(id tagID: TagItem.ID) throws -> [AssetItem]
```

`loadAssets()` should fetch asset tag links and hydrate `AssetItem.tags` from `TagRecord`.

- [ ] **Step 4: Backfill legacy `tagsData` idempotently**

On library open/save path, run a private migration:

```swift
private func migrateLegacyTagsIfNeeded() throws
```

Rules:

- If `AssetTagRecord` already exists for an asset, do not duplicate links.
- Decode `tagsData` only as migration input.
- Create missing `TagRecord`s by normalized name.
- Keep `tagsData` untouched for one model version but stop writing new values.

- [ ] **Step 5: Update `LibraryStore` tag state**

`LibraryStore.tags`, `tagSummaries`, `updateSelectedTags`, `renameTag`, and `deleteTag` should rely on `TagRecord`-backed data. Keep existing sidebar IDs (`tag-\(id)`) stable enough for UI.

- [ ] **Step 6: Commit**

```bash
git add Momento/Storage/MomentoModel.xcdatamodeld Momento/Core/AssetModels.swift Momento/Storage/LibraryMetadataStore.swift Momento/Core/LibraryStore.swift MomentoTests/ImportServiceSmokeTests.swift
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
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Replace current trash test**

Replace `testMovingImportedAssetToTrashRemovesMetadataAndStoredFile` with tests for:

- `moveAssetToTrash` marks `isTrashed = true` and sets `trashedAt`.
- Default visible assets exclude trashed assets.
- Trash sidebar shows trashed assets.
- Restore clears `isTrashed`/`trashedAt`.
- Folder membership survives trash/restore.
- Empty trash removes metadata, tag links, colors, memberships, asset file, thumbnail.

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testMovingAssetToTrashSoftDeletesAndRestores
```

Expected before implementation: FAIL because metadata/file are deleted immediately.

- [ ] **Step 2: Add store APIs**

In `LibraryMetadataStore`:

```swift
func moveAssetToTrash(id assetID: AssetItem.ID) throws -> AssetItem
func restoreAssets(ids: Set<AssetItem.ID>) throws -> [AssetItem]
func emptyTrash() throws -> [AssetItem.ID]
```

Rules:

- Moving to trash does not delete the asset file.
- Moving to trash does not delete folder memberships.
- Empty trash performs the destructive cleanup.
- `trashedAt` is set once per move-to-trash action.

- [ ] **Step 3: Update `LibraryStore` view scoping**

Rules:

- `.trash` selection returns only trashed assets.
- `.all`, `.favorites`, `.unfiled`, `.untagged`, `.tag`, `.folder`, search and filters exclude trashed assets.
- Selection clears if the selected asset leaves the current scope.
- Command palette "Move to Trash" remains soft delete.
- Add restore and empty trash actions in store first; UI can expose them in the same task or next task.

- [ ] **Step 4: Update UI affordances**

Minimum P0 UI:

- Trash sidebar count is real.
- Context menu in Trash offers "Restore" and "Delete Permanently" or "Empty Trash".
- Non-Trash context menu keeps "Move to Trash".
- Do not hide toolbar chrome during delete/empty dialogs.

- [ ] **Step 5: Commit**

```bash
git add Momento/Core/LibraryStore.swift Momento/Storage/LibraryMetadataStore.swift Momento/Storage/LibraryStorage.swift Momento/Features/Sidebar/MomentoSidebarView.swift Momento/AppKitBridge/AssetCollectionGridView.swift MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: soft delete trashed assets"
```

### Task 4: Persist inspector notes

**Files:**
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Storage/LibraryMetadataStore.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/ContentView.swift`
- Modify: `Momento/Features/Shell/MomentoShellView.swift`
- Modify: `Momento/Features/Inspector/MomentoInspectorView.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Add failing note tests**

Add tests for:

- Update selected asset note.
- Reload library and verify note persists.
- Switch between two selected assets and verify each asset shows its own note.

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testUpdatingAssetNotePersistsAcrossReloads
```

Expected before implementation: FAIL because notes are local UI state.

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

- [ ] **Step 4: Replace local notes dictionary**

Remove `ContentView.inspectorNotesByAssetID`. Bind inspector notes to `store.selectedAsset?.note ?? ""`.

For `TextEditor`, avoid writing on every keystroke if it causes grid flashes:

- Keep a local draft inside `MomentoInspectorView`.
- Reset draft on `asset.id` change.
- Commit on focus loss and explicit keyboard submit if available.
- Do not reload all assets for note edits.

- [ ] **Step 5: UI review decision**

If Notes section is approved, add `notesEditor` back into the inspector below Folders and above EXIF with the existing section separator style. If not approved, keep the editor hidden but keep data/API/tests.

- [ ] **Step 6: Commit**

```bash
git add Momento/Core/AssetModels.swift Momento/Storage/LibraryMetadataStore.swift Momento/Core/LibraryStore.swift Momento/ContentView.swift Momento/Features/Shell/MomentoShellView.swift Momento/Features/Inspector/MomentoInspectorView.swift MomentoTests/ImportServiceSmokeTests.swift
git commit -m "feat: persist asset notes"
```

---

## Chunk 3: P0 Drag Organization

### Task 5: Add asset drag source and file promises

**Files:**
- Create: `Momento/AppKitBridge/AssetDragPasteboardWriter.swift`
- Create: `Momento/AppKitBridge/AssetFilePromiseProvider.swift`
- Modify: `Momento/AppKitBridge/AssetCollectionGridView.swift`
- Test: `MomentoTests/ArchitectureGuardTests.swift`

- [ ] **Step 1: Add architecture guard**

Add a source-level test asserting:

- `AssetCollectionGridView` implements `collectionView(_:canDragItemsAt:with:)`.
- It implements `collectionView(_:pasteboardWriterForItemAt:)`.
- It references `NSFilePromiseProvider`.
- It defines an internal pasteboard type for Momento asset IDs.

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ArchitectureGuardTests/testAssetGridSupportsDraggingAndFilePromises
```

Expected before implementation: FAIL.

- [ ] **Step 2: Create internal pasteboard writer**

`AssetDragPasteboardWriter` should encode:

- library ID
- ordered asset IDs
- primary asset ID

Use a custom pasteboard type:

```swift
static let assetIDsPasteboardType = NSPasteboard.PasteboardType("com.seaony.momento.asset-ids")
```

- [ ] **Step 3: Create file promise provider**

`AssetFilePromiseProvider` should use `NSFilePromiseProvider` with:

- `fileType` from `asset.utiIdentifier`
- `userInfo` containing source asset URL and display file name
- delegate writes a copy into destination URL

Rules:

- Dragging to Finder copies files out, never moves the library's content-addressed source file.
- File names use `displayName + "." + fileExtension`, resolving collisions at destination.
- Multi-select creates one provider per selected asset.

- [ ] **Step 4: Wire drag source in `AssetCollectionGridView`**

Rules:

- Drag selected assets if the pointer starts on a selected item.
- If dragging an unselected item, select it first and drag only that item.
- Drag image should use existing selected item snapshots where practical.
- Long-press QuickLook must not conflict with drag start.

- [ ] **Step 5: Manual validation checklist**

Because AppKit dragging is hard to unit test fully, validate manually after implementation:

- Drag one asset to Finder creates a copied file.
- Drag multiple selected assets to Finder creates multiple copied files.
- Dragging does not remove assets from Momento.
- Drag start does not trigger the prior input-error sound.

- [ ] **Step 6: Commit**

```bash
git add Momento/AppKitBridge/AssetDragPasteboardWriter.swift Momento/AppKitBridge/AssetFilePromiseProvider.swift Momento/AppKitBridge/AssetCollectionGridView.swift MomentoTests/ArchitectureGuardTests.swift
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
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testAssigningMultipleDraggedAssetsToFolderIsIdempotent
```

- [ ] **Step 2: Add store bulk APIs**

```swift
func assignAssets(ids: Set<AssetItem.ID>, to folderID: AssetFolder.ID) throws
func addTag(id tagID: TagItem.ID, toAssets assetIDs: Set<AssetItem.ID>) throws
```

Keep these methods merge-based; do not reload the whole asset list.

- [ ] **Step 3: Add SwiftUI drop handling**

Sidebar folder rows and tag rows should accept the internal Momento pasteboard type.

Rules:

- Hover state makes the row visibly droppable.
- Drop on `.unfiled`, `.all`, `.favorites`, `.trash`, and management rows is not accepted.
- Drop on tag adds tag.
- Drop on folder adds membership.
- Multi-select is preserved after drop.

- [ ] **Step 4: Add inspector drop handling if useful**

If implementation remains small, allow dropping assets onto tag/folder chips in the inspector too. If this adds complexity, defer to sidebar-only drag organization.

- [ ] **Step 5: Commit**

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

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testImportingFolderPreservesHierarchyAsVirtualFolders
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
    var assets: [AssetItem]
    var folderAssignments: [AssetItem.ID: [[String]]]
}
```

For duplicates:

- Do not copy the physical file again.
- Do create folder membership if a duplicate is imported through a folder hierarchy.

- [ ] **Step 4: Add metadata store batch save**

Add:

```swift
func saveImportedBatch(_ batch: AssetImportBatch) throws -> [AssetItem]
```

Rules:

- Create missing folders by `(libraryID, parentID, name)`.
- Reuse existing folders with matching sibling name.
- Save asset records first, then memberships.
- Preserve folder memberships for duplicates.

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
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

- [ ] **Step 1: Add storage tests**

Add tests for:

- Export copies the current package and preserves `manifest.json`, `database/library.sqlite`, `assets/`, `thumbnails/`, `previews/`.
- Import validates manifest and database before adding to recent libraries.
- Import refuses unknown manifest schema.
- Importing into an existing destination does not overwrite silently.

Run:

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests/testExportingAndImportingLibraryPackages
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

- [ ] **Step 3: Add LibraryStore flows**

```swift
func exportCurrentLibrary(to destinationURL: URL) throws
func importLibrary(from sourceURL: URL, destinationRootURL: URL?) throws
```

Import should append to Recent Libraries and switch to the imported library.

- [ ] **Step 4: Wire UI**

Minimum UI:

- Library menu: "Import Library..." and "Export Library...".
- Command palette entries for import/export.
- Use native file panels.
- Do not hide toolbar/sidebar during dialogs.

- [ ] **Step 5: Commit**

```bash
git add Momento/Storage/LibraryStorage.swift Momento/Core/LibraryStore.swift Momento/ContentView.swift Momento/Features/Sidebar/MomentoSidebarView.swift Momento/Features/CommandPalette/MomentoCommandPalette.swift MomentoTests/ImportServiceSmokeTests.swift
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
xcodebuild test -scheme Momento -only-testing:MomentoTests/ImportServiceSmokeTests
```

Expected: `TEST SUCCEEDED`.

- [ ] **Step 3: Architecture guard tests**

```bash
xcodebuild test -scheme Momento -only-testing:MomentoTests/ArchitectureGuardTests
```

Expected: `TEST SUCCEEDED`.

- [ ] **Step 4: Full build**

```bash
xcodebuild build -scheme Momento
```

Expected: `BUILD SUCCEEDED`.

- [ ] **Step 5: Manual app validation by user**

The agent must not launch the app. User should manually check:

- Trash: move, restore, empty.
- Notes: edit, switch selection, restart app.
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
| Trash empty deletes files that restore still needs | Only delete physical files in `emptyTrash`; soft trash never moves/removes asset files. |
| Notes cause grid flash | Merge single updated asset into memory; do not reload library on note writes. |
| Drag/drop hard to unit test | Add source-level guard tests and explicit manual checklist. |
| Library export accidentally overwrites | Refuse existing destination in P1. |

## Review Checklist

- [ ] Approve whether Notes section should return to the inspector UI.
- [ ] Approve keeping `tagsData` as a temporary legacy field for one model version.
- [ ] Approve "Import Library" as copy-in rather than open-in-place.
- [ ] Approve not doing P2 features in this pass.
