# Library, Preview, Settings, and Localization Implementation Plan

> Status: historical execution plan. It records the implementation context from 2026-05-21; current behavior and constraints are defined by `README.md`, `AGENTS.md`, and the current review docs.

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn Momento into an Eagle-style library-based macOS app with real library creation/opening/switching, image previews with resolution subtitles, a native Settings window, and English/Simplified Chinese localization.

**Architecture:** Implement this as small, separately committed, shippable milestones. Keep the current SwiftUI + AppKit bridge structure, but avoid intermediate states where the app can create a library yet cannot safely import or reload assets. Library metadata uses Core Data + SQLite inside a user-selected `.momentolibrary` package; persisted asset paths are relative to the package, and sandbox access uses read-write security-scoped bookmarks.

**Tech Stack:** Swift 6, SwiftUI, AppKit, NSCollectionView, Core Data/SQLite, String Catalogs (`Localizable.xcstrings`), macOS Settings scene.

---

## Current State

- `LibraryStore` is an in-memory `@Observable` object with sample assets and no persisted library metadata.
- `LibraryStorage` creates `Libraries/<id>/.library` under Application Support, not a user-selected `.momentolibrary` package.
- `AssetImportService` copies files into `assets/` and returns in-memory `AssetItem` values; it does not write library metadata.
- `AssetItem.originalURL` is non-optional, and preview paths can fall back to original source files.
- `AssetCollectionGridView` uses `NSCollectionView`; item cells only have one label and use synchronous `NSImage(contentsOf:)` fallback behavior.
- `MomentoApp` only defines a `WindowGroup`; there is no native `Settings` scene for `Command + ,`.
- User-facing strings are hard-coded across `ContentView`, Sidebar, Search, Inspector, Command Palette, and previews.
- The project currently has sandbox enabled with user-selected files set to read-only. Creating and reopening user-selected libraries requires read-write access and persisted bookmarks.
- Existing uncommitted items before this plan: `FEATURE.md` and `MomentoTests/` are untracked and should not be mixed into implementation commits unless explicitly requested.

## Confirmed Decisions

1. **Library persistence approach**
   - Use Core Data + SQLite inside the `.momentolibrary` package.
   - Do not implement a JSON-first metadata store.

2. **Library package extension**
   - Use `<Name>.momentolibrary`.
   - Register it as a macOS document/package type so Finder treats the library as one package and the app can open it directly.

3. **Duplicate import behavior**
   - Same `contentHash` in one library reuses the existing asset record and does not copy another physical file.
   - If a duplicate is imported into a folder later, add folder membership only.

4. **Original source URL**
   - After copy import, browsing and preview must not depend on the original source path.
   - Change `AssetItem.originalURL` to `URL?`.
   - Persisted assets loaded from `.momentolibrary` set `originalURL` to `nil`.
   - Store original source path/bookmark later only for "Reveal Original" or watched-folder features.

5. **Thumbnail scope**
   - First pass: raster images and GIFs show real image previews; PDF/video/SVG can use system icons until dedicated preview generation lands.
   - Follow-up: PDF first page, video/GIF poster frame, SVG rendered bitmap thumbnails.

6. **Localization language scope**
   - Support English `en` and Simplified Chinese `zh-Hans`.
   - Add an in-app language switch in Settings with `System`, `English`, and `Simplified Chinese`.
   - `System` remains the default.

7. **Settings first version**
   - Only include settings that work immediately.
   - First version includes app language, default view mode, and About/version.
   - Library path/recent libraries move into Settings only after the library package workflow exists.

## Implementation Rules

- Each milestone must leave the app in a usable state and must be committed separately.
- Do not create a milestone where library creation works but import/reload is disabled or data disappears on relaunch.
- Do not silently create a default library in Application Support after the library-based workflow exists.
- Do not persist absolute asset storage paths. Persist paths relative to the selected `.momentolibrary` package.
- Do not add full tags, folders, search index, watched folders, source bookmarks, Finder Sync, or async thumbnail queues in this plan.
- Do not stage or modify existing untracked `FEATURE.md` or `MomentoTests/` unless that is intentionally pulled into the implementation.
- When `MomentoTests/` is pulled into Milestone 3, inspect any existing untracked files first and only stage test files that are part of that milestone.

---

## Milestone 1: Localization and Native Settings

**Goal:** Make `Command + ,` open a native Settings window and add working English/Simplified Chinese app language switching.

**Files:**
- Create: `Momento/Localizable.xcstrings`
- Create: `Momento/Core/AppLanguage.swift`
- Create: `Momento/Core/AppLocalization.swift`
- Create: `Momento/Features/Settings/MomentoSettingsView.swift`
- Modify: `Momento.xcodeproj/project.pbxproj`
- Modify: `Momento/MomentoApp.swift`
- Modify: `Momento/ContentView.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify user-facing text in:
  - `Momento/Features/Sidebar/MomentoSidebarView.swift`
  - `Momento/Features/Search/MomentoSearchBar.swift`
  - `Momento/Features/CommandPalette/MomentoCommandPalette.swift`
  - `Momento/Features/Inspector/MomentoInspectorView.swift`
  - `Momento/Features/Shell/MomentoShellView.swift`

**Behavior:**
- Add `en` and `zh-Hans` to `Localizable.xcstrings`.
- Add `zh-Hans` to project known localizations.
- Add an `AppLanguage` enum with `system`, `english`, and `simplifiedChinese`.
- Store selected app language in `@AppStorage`.
- Apply selected language at the app root using SwiftUI environment locale for SwiftUI `Text`.
- Add `AppLocalization` for model-backed and AppKit strings so code does not accidentally use the system locale through `NSLocalizedString`.
- Localize model-backed strings, including sidebar item titles, command palette command titles/subtitles, top bar title/subtitle, empty states, import errors, Inspector labels, and Settings labels.
- Add a SwiftUI `Settings` scene in `MomentoApp`.
- `MomentoApp` owns the shared app state:
  - `@State private var store = LibraryStore(...)`.
  - `@AppStorage` values for app language and default view mode.
  - `ContentView` receives the shared `LibraryStore` instead of creating its own `@State private var store`.
  - `MomentoSettingsView` receives bindings/actions for app language and default view mode, plus access to the same `LibraryStore` when changing the current view mode.
- Do not introduce a singleton store or a second independent `LibraryStore` for Settings.
- Settings includes:
  - Language picker: `System`, `English`, `Simplified Chinese`.
  - Default view mode picker backed by `@AppStorage`.
  - About/version section.
- Default view mode setting is separate from current session view mode:
  - `LibraryStore` initializes from the default setting.
  - Changing Settings updates both the stored default and current `LibraryStore.viewMode`.
  - Command palette view changes update the current session view mode only.
- Avoid string concatenation for translatable sentences.
- For dynamic counts, use localizable format strings.

**Known limits:**
- System-owned menu titles and some Info.plist-localized values may continue following system language until InfoPlist localization is added.
- AppKit cell content must be reloaded after language changes if visible AppKit-backed cells contain localized text.

**Steps:**

- [ ] Add `Localizable.xcstrings` with `en` and `zh-Hans`.
- [ ] Add `zh-Hans` to project known regions.
- [ ] Add `AppLanguage`, `@AppStorage` key, and `AppLocalization`.
- [ ] Add `MomentoSettingsView` with language, default view mode, and About/version only.
- [ ] Move `LibraryStore` ownership from `ContentView` into `MomentoApp` and pass the same store into `ContentView` and `MomentoSettingsView`.
- [ ] Add `Settings { MomentoSettingsView(...) }` to `MomentoApp`.
- [ ] Apply `.environment(\.locale, selectedLocale)` at the app root.
- [ ] Wire default view mode into `LibraryStore` without making `@AppStorage` and `LibraryStore.viewMode` compete as uncontrolled sources.
- [ ] Localize major visible strings, including model-backed strings.
- [ ] Reload AppKit-backed visible content when selected app language changes.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify `Command + ,` opens Settings.
- [ ] Manually verify `System`, `English`, and `Simplified Chinese` modes from Settings without relaunch.
- [ ] Manually verify command palette, sidebar, top bar, empty state, and Inspector labels switch language consistently.
- [ ] Commit: `feat: add settings and localization`.

---

## Milestone 2: Grid Preview and Resolution Subtitle

**Goal:** Grid/list cells show a real image preview when available, and show image resolution under the title.

**Files:**
- Modify: `Momento/AppKitBridge/AssetCollectionGridView.swift`
- Reuse/modify if needed: `Momento/Core/AssetModels.swift`
- Reuse/modify if needed: `Momento/Services/AssetImportService.swift`

**Behavior:**
- Add a subtitle label to `AssetCollectionViewItem`.
- For assets with dimensions, subtitle displays `WIDTH x HEIGHT`.
- For assets without dimensions, subtitle can show file extension or file size only if already available; avoid inventing metadata.
- Image/GIF cells show `NSImage` from copied storage path when available.
- Non-image formats keep system icon fallback.
- `prepareForReuse` clears title, subtitle, image, hover state, and any mode-specific state.

**Constraints:**
- First investigate why current imported images may be falling back to file icons even though `previewImage(for:)` tries `NSImage(contentsOf:)`.
- Fix stored URL/kind/path root causes before changing UI fallback behavior.
- Keep this pass small. Do not build full async thumbnail cache here.
- Do not degrade list mode layout.

**Steps:**

- [ ] Inspect one imported raster image and confirm `kind`, `storageURL`, `FileManager.fileExists`, and `NSImage(contentsOf:)` behavior.
- [ ] Fix the root cause if previews currently fall back to icons because stored URL/kind/path is wrong.
- [ ] Add subtitle label and layout constraints for grid, masonry, and list modes.
- [ ] Update `configure(with:viewMode:)` to populate title, subtitle, and preview image.
- [ ] Ensure `prepareForReuse` resets subtitle and image state.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually import a PNG/JPG and verify preview + resolution in Grid/Masonry/List.
- [ ] Manually verify non-image formats still show system icon fallback.
- [ ] Commit: `feat: show asset preview metadata in grid`.

---

## Milestone 3: Library Package and Minimal Persistence

**Goal:** Support Eagle-style user-selected `.momentolibrary` packages with working create/open/switch/import/relaunch behavior in one shippable milestone.

**Files:**
- Modify: `Momento.xcodeproj/project.pbxproj`
- Create: `Momento/Info.plist`
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Storage/LibraryStorage.swift`
- Create: `Momento/Storage/LibraryManifest.swift`
- Create: `Momento/Storage/LibraryAccessScope.swift`
- Create: `Momento/Storage/LibraryMetadataStore.swift`
- Create: `Momento/Storage/MomentoCoreDataStack.swift`
- Create: `Momento/Storage/MomentoModel.xcdatamodeld`
- Modify: `Momento/Services/AssetImportService.swift`
- Create: `Momento/AppOpenHandler.swift`
- Create: `Momento/Features/Library/MomentoLibraryWelcomeView.swift`
- Modify: `Momento/ContentView.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Create/modify: `MomentoTests/LibraryPackagePersistenceTests.swift`
- Modify: `Momento.xcodeproj/project.pbxproj` to create a `MomentoTests` target and attach it to a scheme that can run `xcodebuild ... test`.

**Package Structure:**

```text
<Name>.momentolibrary/
  manifest.json
  database/
    library.sqlite
    library.sqlite-wal
    library.sqlite-shm
  assets/
    <first-two-hash-chars>/
      <sha256>.<ext>
  thumbnails/
  previews/
  metadata/
    import-sessions/
```

**Core Data Model:**
- Create a minimal model file, `MomentoModel.xcdatamodeld`.
- First version includes only the asset metadata needed for current UI:
  - `id: String`
  - `libraryID: String`
  - `displayName: String`
  - `storageRelativePath: String`
  - `kindRaw: String`
  - `fileExtension: String`
  - `byteSize: Int64`
  - `contentHash: String`
  - `pixelWidth: Int64?`
  - `pixelHeight: Int64?`
  - `isFavorite: Bool`
  - `importedAt: Date`
- Use fetch-before-insert on `contentHash` within the current library for duplicate reuse.
- Add a Core Data uniqueness constraint for `libraryID + contentHash`.
- Use a merge policy that preserves the existing asset record on duplicate import instead of creating a second visible asset.
- Do not model tags, folders, search index, asset colors, trash, or source bookmarks yet.

**Document and Package Registration:**
- Register `.momentolibrary` as an exported package type and app document type.
- Include a stable UTI such as `com.seaony.momento.library`.
- Use explicit `Momento/Info.plist` as the source of truth for `CFBundleDocumentTypes` and `UTExportedTypeDeclarations`; nested document/package metadata should not be hidden in unreadable project build settings.
- Configure the target to use `Momento/Info.plist` and preserve existing generated values through build setting placeholders such as `$(PRODUCT_BUNDLE_IDENTIFIER)`, `$(PRODUCT_NAME)`, `$(MARKETING_VERSION)`, and `$(CURRENT_PROJECT_VERSION)`.
- Ensure Finder treats `.momentolibrary` as a package and the app can open it.

**External Open Behavior:**
- Add one explicit app-open entry point for `.momentolibrary` URLs.
- Implement `AppOpenHandler` using `NSApplicationDelegateAdaptor` to handle macOS file-open events and forward accepted `.momentolibrary` URLs to the shared `LibraryStore`.
- If an external open event arrives before `LibraryStore` is ready, queue the URL in app state and open it once the shared store is initialized.
- Reject non-`.momentolibrary` URLs with a visible error instead of silently ignoring them.

**Sandbox and Bookmark Behavior:**
- Change sandbox user-selected file access from read-only to read-write (`ENABLE_USER_SELECTED_FILES = readwrite`).
- Store recent libraries as security-scoped bookmark data plus display names in `UserDefaults`.
- Add `LibraryAccessScope` to resolve bookmarks, detect stale bookmarks, refresh stale bookmarks when possible, and keep package access active while reading/writing.
- Call `startAccessingSecurityScopedResource()` before opening the Core Data store, validating a package, importing into a package, or reading package assets for preview.
- Call `stopAccessingSecurityScopedResource()` when switching libraries, closing a session, or deinitializing the access scope.
- Source URLs from file import/drop must also be accessed for the duration of the copy/hash operation.
- Do not start a detached import task after source or destination security scope has already been stopped.

**Library Workflow Behavior:**
- First launch with no recent library shows a welcome view:
  - Create Library
  - Open Library
- Create Library:
  - User chooses a folder/location.
  - App creates `<Name>.momentolibrary`.
  - App writes `manifest.json`.
  - App creates `database/`, `assets/`, `thumbnails/`, `previews/`, `metadata/import-sessions/`.
  - App initializes `database/library.sqlite`.
  - App switches to the new library and stores its bookmark.
- Open Library:
  - User selects an existing `.momentolibrary`.
  - App validates `manifest.json`.
  - App resolves/starts bookmark access.
  - App opens `database/library.sqlite`.
  - App loads persisted assets.
  - App switches current library.
- Open From Finder/Open With:
  - User opens a `.momentolibrary` package from Finder or Open With.
  - App receives the URL through the external open entry point.
  - App validates and opens the library through the same `LibraryStore.openLibrary(at:)` path as the Open Library action.
- Sidebar top library header becomes a switcher button/menu.
- Recent libraries are saved and can be reopened on next launch.
- Normal app launch should not show sample assets after library workflow exists.
- Import copies files into the selected package and writes metadata in the same operation.
- After import, browsing and preview use package-internal storage paths only.
- `AssetItem.originalURL` becomes optional; loaded persisted assets set it to `nil` and do not require original source URLs.

**Constraints:**
- Do not create a package-only milestone that disables import.
- Do not leave any import path that writes new assets to the old Application Support `.library` location.
- Do not persist absolute package paths inside Core Data asset records.
- Treat the entire `database/` directory as Core Data-owned, including SQLite sidecar files such as `-wal` and `-shm`.
- Preserve compatibility DTOs only where needed to avoid rewriting unrelated UI.

**Steps:**

- [ ] Register `.momentolibrary` document/package metadata in the target.
- [ ] Add explicit `Momento/Info.plist` and preserve current generated plist behavior through build setting placeholders.
- [ ] Add the external `.momentolibrary` open handler and route Finder/Open With URLs to the same open-library path used by the Open Library button.
- [ ] Change sandbox user-selected file access from read-only to read-write.
- [ ] Define `LibraryManifest`.
- [ ] Add `LibraryAccessScope` for bookmark resolution, stale handling, and scoped access lifetime.
- [ ] Update `LibraryStorage` to create/read/validate `.momentolibrary` packages and resolve package-relative asset paths.
- [ ] Add `MomentoModel.xcdatamodeld` with the minimal Asset entity.
- [ ] Add the `libraryID + contentHash` Core Data uniqueness constraint and duplicate merge behavior.
- [ ] Add `MomentoCoreDataStack` configured for `database/library.sqlite` inside the current package.
- [ ] Add `LibraryMetadataStore` for loading, inserting, and duplicate lookup by `contentHash`.
- [ ] Change `AssetItem.originalURL` to `URL?` and update preview/import call sites so persisted assets do not require original source paths.
- [ ] Update `AssetImportService` to hash, copy to `assets/<prefix>/<sha256>.<ext>`, and write metadata while source and destination security scopes are active.
- [ ] Change `LibraryStore` initialization so it can represent "no current library".
- [ ] Remove sample asset dependency from normal app launch.
- [ ] Add library welcome view.
- [ ] Add Create Library with `NSSavePanel` and Open Library with `NSOpenPanel`, configured for `.momentolibrary` packages.
- [ ] Update `ContentView` to show welcome view when no library is selected.
- [ ] Update sidebar header to show current library name and switcher affordance.
- [ ] Ensure Quick Look, Inspector preview, and grid preview resolve package-relative storage URLs and do not fall back to original source paths for persisted assets.
- [ ] Create a `MomentoTests` target.
- [ ] Ensure the test target is included in a scheme that supports `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test`.
- [ ] Add automated smoke tests for library package creation, duplicate import, and reload-after-relaunch behavior.
- [ ] Run `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test`.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify first launch -> create library -> import PNG/JPG -> quit/relaunch -> assets remain.
- [ ] Manually verify opening an existing `.momentolibrary` from Finder/Open panel.
- [ ] Manually verify duplicate import does not create a second physical copy or duplicate visible asset.
- [ ] Manually verify imported assets still preview after the original source file is moved or deleted.
- [ ] Commit: `feat: add library package persistence`.

---

## Recommended Execution Order

1. Milestone 1: Localization and native Settings.
2. Milestone 2: Grid preview and resolution subtitle.
3. Milestone 3: Library package and minimal persistence.

This order fixes `Command + ,` and app-level language switching first, then fixes the currently visible grid preview issue, then moves to the larger library architecture change as one complete, shippable slice. The library work is intentionally not split into package-only and persistence-only commits because that would create an unusable intermediate app state.

## Review Checklist

- [x] Core Data + SQLite inside `.momentolibrary`.
- [x] `.momentolibrary` package extension.
- [x] Duplicate import reuses existing asset metadata.
- [x] Original source paths/bookmarks are not needed for normal browsing in first pass.
- [x] Persist package-relative asset paths, not absolute asset storage paths.
- [x] First thumbnail scope is raster images/GIFs; PDF/video/SVG remain icon fallback.
- [x] Support `en` and `zh-Hans`.
- [x] Add app-level language switch.
- [x] `MomentoApp` owns shared app state; Settings and main window do not create separate stores.
- [x] Settings first version only includes working settings.
- [x] Finder/Open With `.momentolibrary` URLs route through the same open-library path.
- [x] Core Data enforces duplicate asset uniqueness with `libraryID + contentHash`.
- [x] Milestone 3 creates/runs automated smoke tests through a test target.
- [x] Each implementation milestone should be committed separately.
- [x] Each milestone leaves the app in a usable state.

## Not In Scope For This Plan

- Full 100k asset performance target.
- Async thumbnail generation queue for all file types.
- Finder Sync extension.
- Watched folders.
- "Reveal Original" source bookmarks.
- Full Core Data model for tags, folders, search index, trash, and colors.
