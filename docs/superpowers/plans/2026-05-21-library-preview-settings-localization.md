# Library, Preview, Settings, and Localization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn Momento into an Eagle-style library-based macOS app with real library creation/opening/switching, image previews with resolution subtitles, a native Settings window, and English/Simplified Chinese localization.

**Architecture:** Implement this as several small, separately committed milestones. Keep the current SwiftUI + AppKit bridge structure, but introduce a library session layer before changing import and grid behavior. Do not combine the Core Data migration, thumbnail work, Settings scene, and localization into one large change.

**Tech Stack:** Swift 6, SwiftUI, AppKit, NSCollectionView, Core Data/SQLite, String Catalogs (`Localizable.xcstrings`), macOS Settings scene.

---

## Current State

- `LibraryStore` is an in-memory `@Observable` object with sample assets and no persisted library metadata.
- `LibraryStorage` creates `Libraries/<id>/.library` under Application Support, not a user-selected `.momentolibrary` package.
- `AssetImportService` copies files into `assets/` and returns in-memory `AssetItem` values; it does not write library metadata.
- `AssetCollectionGridView` uses `NSCollectionView`, but item cells only have one label and use synchronous `NSImage(contentsOf:)` fallback behavior.
- `MomentoApp` only defines a `WindowGroup`; there is no native `Settings` scene for `Command + ,`.
- User-facing strings are hard-coded across `ContentView`, Sidebar, Search, Inspector, Command Palette, and previews.
- Existing uncommitted items before this plan: `FEATURE.md` and `MomentoTests/` are untracked and should not be mixed into implementation commits unless explicitly requested.

## Decisions Needed Before Implementation

1. **Library persistence approach**
   - Recommended: Core Data + SQLite inside the `.momentolibrary` package.
   - Alternative: `manifest.json + assets.json` first, then migrate later.
   - Decision required before Tasks 3-5.

2. **Library package extension**
   - Recommended: `<Name>.momentolibrary`.
   - This matches the updated `FEATURE.md` direction and makes the resource library visible as a single package in Finder.

3. **Duplicate import behavior**
   - Recommended: same `contentHash` in one library reuses the existing asset record and does not copy another physical file.
   - If a duplicate is imported into a folder later, add folder membership only.

4. **Original source URL**
   - Recommended: after copy import, browsing must not depend on the original path.
   - Store original source path/bookmark later only for "Reveal Original" or watched-folder features.

5. **Thumbnail scope**
   - Recommended first pass: raster images and GIFs show real image previews; PDF/video/SVG can use system icons until dedicated preview generation lands.
   - Follow-up: PDF first page, video/GIF poster frame, SVG rendered bitmap thumbnails.

6. **Localization language scope**
   - Recommended: English `en` and Simplified Chinese `zh-Hans`.
   - Follow system language by default; do not build in-app language switching in the first pass.

7. **Settings first version**
   - Recommended first pass: Library section showing current library path and recent libraries, Appearance section with default view mode, About section.
   - Avoid adding settings that do nothing.

---

## Milestone 1: Native Settings Window

**Goal:** Make `Command + ,` open a useful native Settings window.

**Files:**
- Modify: `Momento/MomentoApp.swift`
- Create: `Momento/Features/Settings/MomentoSettingsView.swift`
- Optional modify: `Momento/DesignSystem/MomentoGlass.swift`

**Behavior:**
- Add a SwiftUI `Settings` scene in `MomentoApp`.
- Settings window includes:
  - Library section: current library status placeholder.
  - Appearance section: default view mode placeholder or real `@AppStorage` value.
  - About section: app name/version placeholder.
- Use native macOS Settings behavior, not a custom modal.

**Steps:**

- [ ] Add `MomentoSettingsView`.
- [ ] Add `Settings { MomentoSettingsView() }` to `MomentoApp`.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify `Command + ,` opens Settings.
- [ ] Commit: `feat: add native settings window`.

---

## Milestone 2: Grid Preview and Resolution Subtitle

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
- Keep this pass small. Do not build full async thumbnail cache here.
- Do not degrade list mode layout.

**Steps:**

- [ ] Add a failing smoke-level UI layout test if a practical test harness exists; otherwise document manual verification.
- [ ] Add subtitle label and layout constraints.
- [ ] Update `configure(with:viewMode:)` to populate title and subtitle.
- [ ] Ensure `prepareForReuse` resets subtitle.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually import a PNG/JPG and verify preview + resolution in Grid/Masonry/List.
- [ ] Commit: `feat: show asset preview metadata in grid`.

---

## Milestone 3: Library Package Foundation

**Goal:** Support Eagle-style user-selected library packages without replacing all metadata internals at once.

**Files:**
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
- Open Library:
  - User selects an existing `.momentolibrary`.
  - App validates `manifest.json`.
  - App switches current library.
- Sidebar top library header becomes a switcher button/menu.
- Recent libraries are saved in `UserDefaults` as bookmarks or paths depending on sandbox decision.

**Constraints:**
- Do not add sample assets when no library exists.
- Do not silently create a default library in Application Support.
- Do not require Core Data to be complete before the welcome flow works.

**Steps:**

- [ ] Define `LibraryManifest`.
- [ ] Update `LibraryStorage` to create/read/validate `.momentolibrary` packages.
- [ ] Add library welcome view.
- [ ] Add create/open actions using `NSOpenPanel` / `NSSavePanel` or SwiftUI file importer/exporter where appropriate.
- [ ] Change `LibraryStore` initialization so it can represent "no current library".
- [ ] Update `ContentView` to show welcome view when no library is selected.
- [ ] Update sidebar header to show current library name and switcher affordance.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify first launch, create, open, switch.
- [ ] Commit: `feat: add library package workflow`.

---

## Milestone 4: Library Metadata Persistence

**Goal:** Persist imported assets inside the selected library instead of keeping them only in memory.

**Files:**
- Modify: `Momento/Core/AssetModels.swift`
- Modify: `Momento/Core/LibraryStore.swift`
- Modify: `Momento/Services/AssetImportService.swift`
- Modify: `Momento/Storage/LibraryStorage.swift`
- Create: `Momento/Storage/LibraryMetadataStore.swift`
- Optional create: `Momento/Storage/MomentoCoreDataStack.swift`
- Update: `MomentoTests/ImportServiceSmokeTests.swift` only if tests are intentionally brought into the build/test flow.

**Behavior, assuming Core Data approach is approved:**
- Store metadata in `database/library.sqlite` within the current `.momentolibrary`.
- Asset records use stable IDs and `contentHash`.
- Physical files copy to `assets/<first-two-hash-chars>/<sha256>.<ext>`.
- `(libraryID, contentHash)` is unique.
- Importing duplicate files returns/reuses existing asset metadata.
- `LibraryStore` loads current library assets from metadata store.

**Constraints:**
- Avoid implementing full Tags/Folders/SearchIndex/AssetColor in this milestone unless required for compile.
- Keep compatibility DTOs if needed so existing UI does not need a full rewrite.

**Steps:**

- [ ] Add metadata store abstraction.
- [ ] Add minimal Asset persistence.
- [ ] Update import pipeline to write metadata.
- [ ] Update `LibraryStore` load/import path.
- [ ] Remove sample asset dependency from normal app launch.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify create library -> import -> quit/relaunch -> assets remain.
- [ ] Commit: `feat: persist library assets`.

---

## Milestone 5: Localization Foundation

**Goal:** Add English and Simplified Chinese localization using native Xcode String Catalogs.

**Files:**
- Create: `Momento/Localizable.xcstrings`
- Modify: `Momento.xcodeproj/project.pbxproj`
- Modify: user-facing text in:
  - `Momento/ContentView.swift`
  - `Momento/Features/Sidebar/MomentoSidebarView.swift`
  - `Momento/Features/Search/MomentoSearchBar.swift`
  - `Momento/Features/CommandPalette/MomentoCommandPalette.swift`
  - `Momento/Features/Inspector/MomentoInspectorView.swift`
  - `Momento/Features/Shell/MomentoShellView.swift`
  - `Momento/Features/Settings/MomentoSettingsView.swift`

**Behavior:**
- Add `zh-Hans` to known localizations.
- Use `LocalizedStringKey` / `String(localized:)` where appropriate.
- Avoid string concatenation for translatable sentences.
- For dynamic counts, use localizable format strings.
- Follow system language. No in-app language switch in first pass.

**References:**
- Apple Localization: https://developer.apple.com/documentation/xcode/localization
- Apple String Catalogs: https://developer.apple.com/documentation/xcode/localizing-and-varying-text-with-a-string-catalog
- SwiftUI localization: https://developer.apple.com/documentation/swiftui/preparing-views-for-localization

**Steps:**

- [ ] Add `Localizable.xcstrings` with `en` and `zh-Hans`.
- [ ] Add `zh-Hans` to project known regions.
- [ ] Localize major visible strings.
- [ ] Build with `xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build`.
- [ ] Manually verify English and Simplified Chinese UI by changing app/system language.
- [ ] Commit: `feat: add english and chinese localization`.

---

## Recommended Execution Order

1. Milestone 1: Settings window.
2. Milestone 2: Grid preview and resolution subtitle.
3. Milestone 3: Library package foundation.
4. Milestone 4: Library metadata persistence.
5. Milestone 5: Localization foundation.

This order keeps early changes small and verifiable. The library workflow should be split from metadata persistence because first-launch UX and package creation can be validated before the full database path is finished.

## Review Checklist

- [ ] Confirm Core Data vs JSON-first persistence approach.
- [ ] Confirm `.momentolibrary` package extension.
- [ ] Confirm duplicate import behavior.
- [ ] Confirm whether original source paths/bookmarks are needed in first pass.
- [ ] Confirm thumbnail scope for first pass.
- [ ] Confirm `en` + `zh-Hans` only.
- [ ] Confirm Settings first-version sections.
- [ ] Confirm each milestone should be committed separately.

## Not In Scope For This Plan

- Full 100k asset performance target.
- Async thumbnail generation queue for all file types.
- Finder Sync extension.
- Watched folders.
- In-app language switcher.
- Full Core Data model for tags, folders, search index, and colors unless explicitly pulled into Milestone 4.
