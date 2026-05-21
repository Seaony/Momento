# Library, Preview, Settings, and Localization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn Momento into an Eagle-style library-based macOS app with real library creation/opening/switching, image previews with resolution subtitles, a native Settings window, and English/Simplified Chinese localization.

**Architecture:** Implement this as several small, separately committed milestones. Keep the current SwiftUI + AppKit bridge structure, introduce localization before adding more UI text, then add native Settings, grid preview metadata, and finally the library package/session/persistence path. Library metadata uses Core Data + SQLite inside a user-selected `.momentolibrary` package; sandbox access uses read-write security-scoped bookmarks.

**Tech Stack:** Swift 6, SwiftUI, AppKit, NSCollectionView, Core Data/SQLite, String Catalogs (`Localizable.xcstrings`), macOS Settings scene.

---

## Current State

- `LibraryStore` is an in-memory `@Observable` object with sample assets and no persisted library metadata.
- `LibraryStorage` creates `Libraries/<id>/.library` under Application Support, not a user-selected `.momentolibrary` package.
- `AssetImportService` copies files into `assets/` and returns in-memory `AssetItem` values; it does not write library metadata.
- `AssetCollectionGridView` uses `NSCollectionView`, but item cells only have one label and use synchronous `NSImage(contentsOf:)` fallback behavior.
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
   - After copy import, browsing must not depend on the original path.
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
   - Library path/recent libraries move into Settings only after the library package foundation exists.

---

## Milestone 1: Localization Foundation and App Language

**Goal:** Add English and Simplified Chinese localization infrastructure before adding more UI surfaces.

**Files:**
- Create: `Momento/Localizable.xcstrings`
- Create: `Momento/Core/AppLanguage.swift`
- Modify: `Momento.xcodeproj/project.pbxproj`
- Modify: `Momento/MomentoApp.swift`
- Modify user-facing text in:
  - `Momento/ContentView.swift`
  - `Momento/Features/Sidebar/MomentoSidebarView.swift`
  - `Momento/Features/Search/MomentoSearchBar.swift`
  - `Momento/Features/CommandPalette/MomentoCommandPalette.swift`
  - `Momento/Features/Inspector/MomentoInspectorView.swift`
  - `Momento/Features/Shell/MomentoShellView.swift`

**Behavior:**
- Add `en` and `zh-Hans` to `Localizable.xcstrings`.
- Add `zh-Hans` to project known localizations.
- Add an `AppLanguage` enum with `system`, `english`, and `simplifiedChinese`.
- Store the selected app language in `@AppStorage`.
- Apply the selected language at the app root using SwiftUI environment locale for SwiftUI `Text`.
- Add one helper for non-SwiftUI strings, backed by the selected language, so AppKit cells and imperative code do not call `NSLocalizedString` with the system locale by accident.
- Dynamic app-level language switching must update SwiftUI views without relaunch.
- AppKit-backed cells must read localized strings during configuration so reloads pick up the selected language.
- Avoid string concatenation for translatable sentences.
- For dynamic counts, use localizable format strings.

**Known limits:**
- System-owned menu titles and some Info.plist-localized values may continue following system language until InfoPlist localization is added.
- AppKit cell content may need collection reload after language changes.

**References:**
- Apple Localization: https://developer.apple.com/documentation/xcode/localization
- Apple String Catalogs: https://developer.apple.com/documentation/xcode/localizing-and-varying-text-with-a-string-catalog
- SwiftUI localization: https://developer.apple.com/documentation/swiftui/preparing-views-for-localization

**Steps:**

- [ ] Add `Localizable.xcstrings` with `en` and `zh-Hans`.
- [ ] Add `zh-Hans` to project known regions.
- [ ] Add `AppLanguage`, `@AppStorage` key, and non-SwiftUI localization helper.
- [ ] Apply `.environment(\.locale, selectedLocale)` at the app root.
- [ ] Localize major visible strings.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify English, Simplified Chinese, and System modes.
- [ ] Commit: `feat: add english and chinese localization`.

---

## Milestone 2: Native Settings Window

**Goal:** Make `Command + ,` open a useful native Settings window with only working settings.

**Files:**
- Modify: `Momento/MomentoApp.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Create: `Momento/Features/Settings/MomentoSettingsView.swift`
- Optional modify: `Momento/DesignSystem/MomentoGlass.swift`

**Behavior:**
- Add a SwiftUI `Settings` scene in `MomentoApp`.
- Settings includes:
  - Language picker: `System`, `English`, `Simplified Chinese`.
  - Default view mode picker backed by `@AppStorage`.
  - About/version section.
- Do not include library path or recent-library controls until Milestone 4.
- Use native macOS Settings behavior, not a custom modal.
- Default view mode must affect new library/store sessions and should update the current view mode when changed during the session.

**Steps:**

- [ ] Add `MomentoSettingsView`.
- [ ] Add real `@AppStorage` values for language and default view mode.
- [ ] Wire default view mode to `LibraryStore` instead of leaving it as a saved-but-unused preference.
- [ ] Add `Settings { MomentoSettingsView() }` to `MomentoApp`.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify `Command + ,` opens Settings and changes language/default view.
- [ ] Commit: `feat: add native settings window`.

---

## Milestone 3: Grid Preview and Resolution Subtitle

**Goal:** Grid/list cells show a real image preview when available, and show image resolution under the title.

**Files:**
- Modify: `Momento/AppKitBridge/AssetCollectionGridView.swift`
- Reuse: `Momento/Core/AssetModels.swift`
- Reuse: `Momento/Services/AssetImportService.swift`

**Behavior:**
- Add a subtitle label to `AssetCollectionViewItem`.
- For assets with dimensions, subtitle displays `WIDTH x HEIGHT`.
- For assets without dimensions, subtitle can show file extension or file size only if already available; avoid inventing metadata.
- Image/GIF cells show `NSImage` from copied storage path when available.
- Non-image formats keep system icon fallback.

**Constraints:**
- First investigate why current imported images may be falling back to file icons even though `previewImage(for:)` tries `NSImage(contentsOf:)`.
- Keep this pass small. Do not build full async thumbnail cache here.
- Do not degrade list mode layout.

**Steps:**

- [ ] Inspect one imported raster image and confirm `kind`, `storageURL`, `FileManager.fileExists`, and `NSImage(contentsOf:)` behavior.
- [ ] Fix the root cause if previews currently fall back to icons because the stored URL/kind/path is wrong.
- [ ] Add subtitle label and layout constraints.
- [ ] Update `configure(with:viewMode:)` to populate title and subtitle.
- [ ] Ensure `prepareForReuse` resets subtitle.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually import a PNG/JPG and verify preview + resolution in Grid/Masonry/List.
- [ ] Commit: `feat: show asset preview metadata in grid`.

---

## Milestone 4: Library Package Foundation

**Goal:** Support Eagle-style user-selected library packages without replacing all metadata internals at once.

**Files:**
- Modify: `Momento.xcodeproj/project.pbxproj`
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Storage/LibraryStorage.swift`
- Create: `Momento/Storage/LibraryManifest.swift`
- Create: `Momento/Features/Library/MomentoLibraryWelcomeView.swift`
- Modify: `Momento/ContentView.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`

**Behavior:**
- First launch with no recent library shows a welcome view:
  - Create Library
  - Open Library
- Create Library:
  - User chooses a folder/location.
  - App creates `<Name>.momentolibrary`.
  - App writes `manifest.json`.
  - App creates `database/`, `assets/`, `thumbnails/`, `previews/`, `metadata/import-sessions/`.
- Register `.momentolibrary` as an app document/package type.
- Change sandbox user-selected file access from read-only to read-write (`ENABLE_USER_SELECTED_FILES = readwrite`).
- Store security-scoped bookmarks for recent libraries and keep access active while reading or writing package contents.
- Open Library:
  - User selects an existing `.momentolibrary`.
  - App validates `manifest.json`.
  - App switches current library.
- Sidebar top library header becomes a switcher button/menu.
- Recent libraries are saved in `UserDefaults` as bookmark data plus display names.
- Until Milestone 5 lands, importing from a selected library session is disabled instead of performing copy-only imports that disappear on relaunch.

**Constraints:**
- Do not add sample assets when no library exists.
- Do not silently create a default library in Application Support.
- Do not require Core Data to be complete before the welcome flow works.
- Do not leave an import path that writes new assets to the old Application Support `.library` location.

**Steps:**

- [ ] Register `.momentolibrary` document/package metadata in the Xcode project / generated Info.plist settings.
- [ ] Change sandbox user-selected file access from read-only to read-write.
- [ ] Define `LibraryManifest`.
- [ ] Update `LibraryStorage` to create/read/validate `.momentolibrary` packages.
- [ ] Add bookmark persistence and resolution for recent libraries.
- [ ] Add library welcome view.
- [ ] Add create/open actions using `NSOpenPanel` / `NSSavePanel` or SwiftUI file importer/exporter where appropriate.
- [ ] Change `LibraryStore` initialization so it can represent "no current library".
- [ ] Update `ContentView` to show welcome view when no library is selected.
- [ ] Update sidebar header to show current library name and switcher affordance.
- [ ] Disable import for selected library sessions until metadata persistence exists, and ensure no import path writes to an old non-selected library path.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify first launch, create, open, switch.
- [ ] Commit: `feat: add library package workflow`.

---

## Milestone 5: Library Metadata Persistence

**Goal:** Persist imported assets inside the selected library instead of keeping them only in memory.

**Files:**
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Services/AssetImportService.swift`
- Modify: `Momento/Storage/LibraryStorage.swift`
- Create: `Momento/Storage/LibraryMetadataStore.swift`
- Create: `Momento/Storage/MomentoCoreDataStack.swift` if no equivalent Core Data stack exists.
- Update: `MomentoTests/ImportServiceSmokeTests.swift` only if tests are intentionally brought into the build/test flow.

**Behavior:**
- Store metadata in `database/library.sqlite` within the current `.momentolibrary`.
- Treat the entire `database/` directory as Core Data-owned, including SQLite sidecar files such as `-wal` and `-shm`.
- Asset records use stable IDs and `contentHash`.
- Physical files copy to `assets/<first-two-hash-chars>/<sha256>.<ext>`.
- `(libraryID, contentHash)` is unique.
- Importing duplicate files returns/reuses existing asset metadata.
- `LibraryStore` loads current library assets from metadata store.
- Import only runs when a writable current library is selected and bookmark access is active.
- Resolve and start the selected library security-scoped bookmark before loading the Core Data store.

**Constraints:**
- Avoid implementing full Tags/Folders/SearchIndex/AssetColor in this milestone unless required for compile.
- Keep compatibility DTOs if needed so existing UI does not need a full rewrite.
- Do not copy or export only `library.sqlite`; preserve the whole `database/` directory.

**Steps:**

- [ ] Add metadata store abstraction.
- [ ] Add Core Data stack configured for the current library package path.
- [ ] Add minimal Asset persistence.
- [ ] Update import pipeline to write metadata.
- [ ] Update `LibraryStore` load/import path.
- [ ] Remove sample asset dependency from normal app launch.
- [ ] Add or wire smoke tests for library package creation, duplicate import, and reload-after-relaunch behavior.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify create library -> import -> quit/relaunch -> assets remain.
- [ ] Commit: `feat: persist library assets`.

---

## Recommended Execution Order

1. Milestone 1: Localization foundation and app language.
2. Milestone 2: Settings window.
3. Milestone 3: Grid preview and resolution subtitle.
4. Milestone 4: Library package foundation.
5. Milestone 5: Library metadata persistence.

This order avoids adding new hard-coded UI text before localization exists, keeps early changes small and verifiable, then handles the larger library workflow in two stages. Library package creation and metadata persistence are separate, but imports must be disabled during the package-only stage so data cannot be written to the wrong location or disappear on relaunch.

## Review Checklist

- [x] Core Data + SQLite inside `.momentolibrary`.
- [x] `.momentolibrary` package extension.
- [x] Duplicate import reuses existing asset metadata.
- [x] Original source paths/bookmarks are not needed for normal browsing in first pass.
- [x] First thumbnail scope is raster images/GIFs; PDF/video/SVG remain icon fallback.
- [x] Support `en` and `zh-Hans`.
- [x] Add App-level language switch.
- [x] Settings first version only includes working settings.
- [x] Each implementation milestone should be committed separately.

## Not In Scope For This Plan

- Full 100k asset performance target.
- Async thumbnail generation queue for all file types.
- Finder Sync extension.
- Watched folders.
- Full Core Data model for tags, folders, search index, and colors unless explicitly pulled into Milestone 5.
