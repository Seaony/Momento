# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Momento is a native macOS asset-management app (Eagle-class functionality, Craft-class UI). Pure SwiftUI + AppKit, no web tech. The full product spec lives in `FEATURE.md` — read it before touching new features. Highlights:

- macOS 26+, Swift 6, strict concurrency.
- Liquid Glass UI: use SwiftUI's native `.glassEffect`, `.buttonStyle(.glass)` / `.buttonStyle(.glassProminent)`. Do **not** fake glass with `NSVisualEffectView` + custom blur/stroke/shadow.
- Files must stay under 1000 lines; avoid speculative abstraction.

## Build / Test Commands

There is no Makefile or shell-script tooling — it's a pure Xcode project (`Momento.xcodeproj`).

```bash
# Build
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' build

# Run all tests
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test

# Run a single test (replace ClassName/methodName)
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' \
  test -only-testing:MomentoTests/LibraryPackagePersistenceTests/testImportPersistsAndDeduplicatesAssets
```

The Momento target uses Xcode's "file-system synchronized group" — new `.swift` files placed under `Momento/` are picked up automatically; no `project.pbxproj` edits required.

## High-Level Architecture

The app is organized as a feature-module layout around a single observable store. Trace flow from `MomentoApp.swift` (Scene) → `ContentView.swift` (root view) → `MomentoShellView` (sidebar / content / inspector layout).

### Core state
- **`Core/LibraryStore.swift`** — the `@MainActor @Observable` central state object. Owns `currentLibrary`, `assets`, `selectedAssetID`, `sidebarSelection`, `viewMode`, etc. Derives `visibleAssets` (search + sidebar filtering) and `tags` reactively. All UI mutates state through this store; views never talk to `Storage/` directly.
- **`Core/AssetModels.swift`** — `nonisolated Sendable` plain structs (`AssetLibrary`, `AssetItem`, `TagItem`, `AssetDimensions`, `AssetKind`, `AssetViewMode`, `SidebarSelection`). These cross actor boundaries.
- **`Core/AppLanguage.swift` + `Core/AppLocalization.swift`** — language switcher. `AppLocalization` is injected via `EnvironmentValues.appLocalization` and resolves strings from a per-locale `.lproj` bundle, falling back to `Bundle.main`. The single string catalog is `Momento/Localizable.xcstrings` — edit it via Xcode, not raw `.strings` files. Settings persists `appLanguage` and `defaultViewMode` via `@AppStorage` (keys in `AppSettingsKeys`).

### Persistence layer (`Storage/`)
A Momento library is a **directory bundle** with extension `.momentolibrary` (UTI `com.seaony.momento.library`, exported in `Info.plist`). Internal layout follows `FEATURE.md §1`:
```
<Name>.momentolibrary/
├── manifest.json            # schemaVersion, libraryID, displayName, createdAt, updatedAt
├── database/library.sqlite  # Core Data store — authoritative metadata
├── assets/<hashPrefix>/<sha256>.<ext>   # content-addressable
├── thumbnails/{small,medium,large}/
├── previews/
└── metadata/import-sessions/
```

- **`LibraryStorage`** — pure path/IO helpers; creates package directories, reads/writes `manifest.json`, computes asset paths from content hash. Stays `nonisolated Sendable`.
- **`MomentoCoreDataStack`** — loads `MomentoModel` (Core Data model, file `Storage/MomentoModel.xcdatamodeld`), pointing the SQLite store at the opened library's `database/library.sqlite`. One stack instance per opened library.
- **`LibraryMetadataStore`** — wraps a `newBackgroundContext()`; converts between `AssetRecord` Core Data entities and `AssetItem` value types. Uses `performAndWait`. Enforces `(libraryID, contentHash)` uniqueness through a Core Data uniqueness constraint and dedup-on-import.
- **`LibraryAccessScope`** / **`RecentLibraryStore`** — security-scoped bookmark plumbing for the recent-libraries list (UserDefaults key `recentLibraries`).
- **`LibraryManifest`** — `schemaVersion` checked on open; bump `currentSchemaVersion` only when adding migration logic.

**Important Core Data note:** the live model currently has *only* `AssetRecord`. `FEATURE.md` specifies a richer schema (Folder, Tag, AssetTag, AssetColor, ThumbnailRecord, SearchIndex, FileBookmark, FolderMembership). New entities must be added to `MomentoModel.xcdatamodeld` and migrations handled — don't shoehorn fields onto `AssetRecord`.

### Import pipeline (`Services/`)
- **`AssetImportService`** — `Task.detached(priority: .userInitiated)` walks input URLs (files or directory trees), filters by extension/UTType, computes SHA-256 streaming hash, copies into `assets/<prefix>/<hash>.<ext>` (skips if file already exists), reads `CGImageSource` for image dimensions. Returns `[AssetItem]` for the store to persist. Manages `startAccessingSecurityScopedResource` lifetimes via `SourceAccessScope`.
- Dedup happens twice: (1) within the batch via the hash set, (2) at persistence via the Core Data uniqueness constraint. `LibraryStore.importItems` passes `metadataStore.existingContentHashes()` as the pre-existing seed.

### AppKit bridge (`AppKitBridge/`)
SwiftUI hosts AppKit for the high-performance and platform-integration paths:
- **`AssetCollectionGridView`** — `NSViewRepresentable` wrapping `NSCollectionView` + `NSCollectionViewFlowLayout`. Handles masonry/grid/list modes, hover/selection styling on `HoverSelectionView`, double-click → `onDoubleClick` callback. **This is the rendering path for 100k+ asset support; do not replace with `LazyVGrid`.**
- **`QuickLookPreviewController`** — QuickLook bridge for Space-preview.

### Design system (`DesignSystem/`)
- **`MomentoGlass.swift`** — `MomentoGlassBackground` (uses native SwiftUI `.glassEffect`), `MomentoVisualEffectView` (only for legitimate AppKit semantic materials), `MomentoTheme` size/color tokens (sidebar widths, panel radii, etc.). When adding glass surfaces, route through `MomentoGlassBackground` / `.momentoGlassPanel(...)`.

### Feature modules (`Features/`)
One folder per surface — `Sidebar`, `Settings`, `Inspector`, `Shell` (HSplitView shell composing sidebar + content + inspector + command palette), `Library` (Welcome screen), `Search` (top-bar search field), `CommandPalette` (⌘K), `Assets`. Cross-feature wiring lives in `ContentView` (binds store ↔ shell, builds sidebar sections, handles ⌘K shortcut and file importer).

### App lifecycle
- **`MomentoApp.swift`** — `WindowGroup` + `Settings` scene; uses `.windowStyle(.hiddenTitleBar)` + `.windowToolbarStyle(.unifiedCompact)`. Injects `\.locale` and `\.appLocalization`.
- **`AppOpenHandler.swift`** — `NSApplicationDelegateAdaptor` that receives `application(_:openFiles:)` for double-clicked `.momentolibrary` packages and forwards to `LibraryStore.openLibrary`. Queues URLs that arrive before the SwiftUI view is ready and flushes them via `flushPendingLibraryURLs()`.

## Concurrency & Sendable conventions

- The project is Swift 6 strict-concurrency clean. `LibraryStore` is `@MainActor`; almost everything in `Storage/`, `Services/`, and `Core/AssetModels.swift` is marked `nonisolated` + `Sendable`.
- Heavy work (import, hashing, Core Data writes) runs on detached tasks or background contexts — keep it that way. UI never touches a Core Data managed object directly; it consumes the `AssetItem` value type.

## Localization

- Single string catalog at `Momento/Localizable.xcstrings`. Use `localization.string("Key")` / `localization.format("…%d…", n)` everywhere user-facing. The fallback `value` passed to `localizedString` is the key itself, so English keys double as English copy.
- Adding a new user-visible string: add it to `Localizable.xcstrings` (Xcode handles `en` and `zh-Hans` columns), then call `localization.string("…")` from the view.

## Testing policy

Do not default to strict TDD for this project. Unless the user explicitly asks for it, or the change touches high-risk data/state/filesystem behavior, implement directly and verify afterward.

Keep tests focused on durable behavior:

- Library package creation/opening/renaming/deletion, recent-library persistence, missing-library pruning, import/dedup/cache behavior, and other user-data safety paths.
- Localization catalog integrity.
- A small number of coarse architecture guards for rules that have repeatedly regressed, such as native Liquid Glass usage, transparent window toolbar behavior, current-library validation on appear, and main-window minimum sizing.

Do not add tests for pure visual tuning: padding, font size, icon size, hover brightness, menu spacing, or exact SwiftUI view structure. If a source-level test starts locking those details, delete it or collapse it into a coarse architecture guard.

For visual UI changes, the default validation is: compile/build if needed, run the most relevant existing tests, and run `git diff --check`. The user will launch the app and judge visual output; do not proactively open or launch Momento for inspection.

## Project Settings

- macOS Deployment Target: **26.0** (required for native Liquid Glass APIs — `FEATURE.md` lists 14+ aspirationally but the build is 26).
- Swift: 6.0.
- App Sandbox: enabled. File access flows through security-scoped bookmarks (`LibraryAccessScope`, `RecentLibraryStore`).
- Bundle id: `com.seaony.Momento`. Exported UTI: `com.seaony.momento.library` (extension `momentolibrary`, conforms to `public.package`).

## Working Rules (carried from `FEATURE.md` and global rules)

- **Never auto-start a dev server or launch the app for the user.** Build commands above are for verification only.
- **Commit completed updates after basic validation.** The current project preference is to commit each finished user-requested update. Commit messages follow `<type>: <summary>`.
- **Strict scope control.** Don't refactor adjacent code while fixing a bug. Surface unrelated issues separately.
- **No fake placeholders.** Don't return `[]` / `nil` / empty dicts to silence the type checker — finish the read/write path or stop.
- **Never hardcode library-absolute paths** into the database — only relative paths under the library root (see `LibraryStorage.relativePath` / `resolveAssetURL`).
- **Sensitive fields in Settings (API keys etc.)** stay plaintext by default; do not add masking.
