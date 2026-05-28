# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Working constraints

`AGENTS.md` is the authoritative source for working style, hard constraints, testing policy, validation, and Git rules. Read it first. This file only covers the build/test commands and the cross-file architecture; it does not restate those rules.

## Commands

```bash
# Build (Debug)
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination 'platform=macOS' build

# Run the full test suite
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test

# Run a single test (class or method)
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' \
  test -only-testing:MomentoTests/LibraryPackagePersistenceTests/testImportPersistsAndDeduplicatesAssets

# Whitespace check used as a minimum validation gate
git diff --check

# Release: build Release, make DMG, sign Sparkle update, update appcast.xml, create GitHub Release
scripts/prepare-release.sh <marketing-version> <build-number>
```

Targets: macOS 26+, Swift 6 with strict concurrency. Sparkle 2 is the only third-party dependency (via SPM, `XCRemoteSwiftPackageReference`). Do not launch the app to verify UI — the user runs it and judges visuals.

## Architecture

Native macOS app (SwiftUI shell + AppKit for the heavy grid). No web tech. Entry chain: `MomentoApp.swift` → `ContentView.swift` → `MomentoShellView`.

### State flow

`Core/LibraryStore.swift` (`@MainActor @Observable`) is the single source of truth and the only thing the UI mutates — views never touch the storage layer directly. The pattern that matters across files:

- `store.assets` holds the value objects for the **currently open library only** (one library loaded at a time; `activateLibrary` replaces `assets`/`folders`/`tags` wholesale).
- `store.visibleAssets` is a computed derivation (sidebar scope → filter → search → sort) consumed by the grid and item-count UI. It recomputes on every access; `ContentView.libraryBody` reads it once and threads the result down. Treat it as a hot path (see `docs/performance-deep-review.md`).
- Mutations (favorite, tags, folders, trash, import) write back into `store.assets`, which fires Observation and re-renders `ContentView.body`. A single edit cascades into full re-derivations — keep that in mind before adding more per-render scans.

`Core/AssetModels.swift` holds `nonisolated Sendable` value types (`AssetItem`, `TagItem`, `AssetFolder`, `AssetColor`, …). These cross actor boundaries between background services and the MainActor store; mutating store state means replacing these values, not mutating shared references.

### Storage layer (`Storage/`)

A library is a local `.momento` package (legacy `.momentolibrary` opens too). Layout:

```text
<Name>.momento/
├── manifest.json
├── database/library.sqlite        # Core Data store
├── assets/<hashPrefix>/<sha256>.<ext>
├── thumbnails/<sha256>.png        # pre-generated at 512px
└── metadata/import-sessions/
```

- `LibraryMetadataStore` is the Core Data boundary. It runs on a **background context via `performAndWait`**, returns Sendable value objects, loads the whole library on open (`loadAssets` + batched `IN` relationship fetches, no N+1), and persists imports in one batch save.
- The DB stores **package-relative paths only**, never absolute paths — so moving the whole `.momento` package keeps assets resolvable.
- `LibraryStorage` / `LibraryAccessScope` manage the package on disk and **security-scoped bookmarks** for libraries outside the sandbox; scoped access must stay alive across the whole async import, not just the URL-collection phase.
- Content `sha256` is both the `AssetItem.id` and the dedup key (in-batch + in-library + Core Data uniqueness).

### Grid rendering (`AppKitBridge/`)

`AssetCollectionGridView.swift` is an `NSViewRepresentable` wrapping `NSCollectionView` — the 100k-asset render path. **Do not replace it with `LazyVGrid`.** It owns custom masonry/grid layouts (cached frames + binary-search visible range), in-place cell updates that avoid `reloadData` for lightweight field changes, and `AssetPreviewImageProvider` (bounded-concurrency thumbnail decode, in-flight coalescing, prefetch, bounded `NSCache`).

### Services (`Services/`)

Background work runs in `Task.detached` and hands Sendable results back to the MainActor store: `AssetImportService` (collect → hash → copy → thumbnail → palette → EXIF/dimensions, with throttled progress), `AssetThumbnailService`, `AssetColorAnalysisService` (24 color categories), plus a local-only HTTP listener for browser imports.

### UI conventions

- Liquid Glass uses SwiftUI-native `.glassEffect` / `.buttonStyle(.glass)` / `.glassProminent` — never fake it with custom blur/stroke/shadow. `DesignSystem/MomentoGlass.swift` is the theme/token entry point.
- `Momento/Localizable.xcstrings` is the only string catalog; all user-facing text goes through `AppLocalization`.
- New `.swift` files under `Momento/` are auto-included by Xcode's file-system synchronized group — don't edit `project.pbxproj` to add them.

## Key references

- `FEATURE.md` — full product spec; read before building new features.
- `docs/performance-deep-review.md` — concrete performance hotspots with file:line.
- `docs/README.md` — index marking each doc as current / historical / future.
