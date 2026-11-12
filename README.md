<div align="center">
<img src="app/resources/icon.png" width="112" alt="Momento">

# Momento

A local image library for macOS, Windows and Linux.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![mygo](https://img.shields.io/badge/mygo-v0.9.0-222222)
![Native UI](https://img.shields.io/badge/UI-native_GPU-222222)

[English](README.md) · [简体中文](README.zh-CN.md) · [Browser extension](https://github.com/Seaony/Momento-Chomre-Extension/releases/latest)
</div>

Momento keeps images on your own disk, organized by color, tags and nested folders. One Go codebase draws a native GPU interface on all three desktop platforms.

## Features

- Browse in masonry, grid or list view. Search names, file types and tags, then filter by color, tag or extension.
- Organize images with favorites, nested folders and batch tag edits. Inspect palettes, source links and EXIF data.
- Import files or whole folders with SHA-256 deduplication. Supported formats: JPEG, PNG, GIF, WebP, TIFF, BMP, HEIC and AVIF. GIF previews show the first frame.
- Export originals, JPEG or PNG. Deleted images stay in the library's trash until restored or permanently deleted.
- Use English, Simplified Chinese or Japanese, with system, light or dark appearance.

Folder counts and the images shown when opening a folder include only its direct members, excluding images in subfolders.

## Quick start

Install Go 1.27.3 or later, then run:

```sh
git clone https://github.com/Seaony/Momento.git
cd Momento/app
go tool mygo dev
```

The CLI is pinned in `go.mod`; no Bun, Node, web frontend or cgo is needed. Linux requires GTK 3. The development command opens a real application window and rebuilds it when Go files change.

During window capture or sharing, macOS may replace the traffic lights with a purple `Window sharing` system control. Its menu still offers close, minimize and full screen. This is not a development badge; the normal traffic lights appear when the system control is absent.

## Settings and shortcuts

Open the separate Settings window from the sidebar, from **Momento > Settings…** on macOS (`⌘,`), or from **File > Settings…** on Windows/Linux (`Ctrl+,`). Opening it again focuses the existing window.

- **General:** launch at login, appearance, language and automatic update checks. Launch at login is off by default; its switch reflects the operating system's actual state and shows any error. Automatic checks are on by default and run once at startup; turning them off leaves manual checks available.
- **Browser Extension:** allow imports and inspect the local service status. Enabled by default, this switch starts or stops the listener at `127.0.0.1:47641`. A port conflict or other service error is shown in Settings and recorded in the log.
- **About:** view the version, check for updates and open the log directory. Update checks show progress, the result or the reason they cannot run, including missing update configuration in development builds.

Appearance and language changes apply immediately to both windows and application menus. Following the system uses English, Simplified Chinese or Japanese, falling back to English for other languages. On macOS, native AppKit file-dialog buttons usually pick up a changed language on the next process launch.

The application remembers the view mode, sort order and direction, sidebar and inspector visibility, and sidebar width between launches.

| Action | macOS | Windows/Linux |
|---|---|---|
| New / open library | `⌘N` / `⌘O` | `Ctrl+N` / `Ctrl+O` |
| Import files / folder | `⌘I` / `⌘⇧I` | `Ctrl+I` / `Ctrl+Shift+I` |
| Export selected images | `⌘E` | `Ctrl+E` |
| Masonry / grid / list | `⌘1` / `⌘2` / `⌘3` | `Ctrl+1` / `Ctrl+2` / `Ctrl+3` |
| Search | `⌘F` | `Ctrl+F` |
| Filter / sort | `⌘⌥F` / `⌘⌥S` | `Ctrl+Alt+F` / `Ctrl+Alt+S` |
| Toggle inspector | `⌘⌥I` | `Ctrl+Alt+I` |
| Settings | `⌘,` | `Ctrl+,` |

## Development

From `app/`:

```sh
go build ./...
go vet ./...
go test ./internal/...
go test ./...
```

Generate the offline UI screenshot set with:

```sh
MOMENTO_SCREENSHOTS=1 go test ./ui -run '^TestStateScreenshots$' -count=1
MOMENTO_SETTINGS_SCREENSHOTS=1 go test ./ui -run '^TestSettingsWindowScreenshots$' -count=1
```

The main state set is written to `/tmp/momento-titlebar-settings-preview/`; Settings states go to `/tmp/momento-settings-window/`, each with a manifest. Both cover English, Simplified Chinese and Japanese in light and dark appearance. These tests render the interface without opening a window.

To create a sample library for development:

```sh
go run ./cmd/sample-library -path /tmp/momento-sample.momento -count 14
```

The default is 14 images: 10 JPEG and 4 PNG. Use `-count 100000` for a large library. The parent directory must exist; the command refuses to overwrite an existing library and removes only its new package if generation fails or is cancelled.

## Configuration, libraries and logs

Settings and remembered UI state are saved in `preferences.json` in the application configuration directory:

- macOS: `~/Library/Application Support/Momento/`
- Windows: `%APPDATA%\Momento\`
- Linux: `$XDG_CONFIG_HOME/Momento/` (default `~/.config/Momento/`)

New fields use their defaults when absent from a saved file. The initial appearance and language follow the system; the initial view is masonry, sorted by newest added first, with both the sidebar and inspector visible. Recent library paths are stored separately in `recent-libraries.json` in the same directory.

Application logs rotate at 5 MB, keeping three previous files:

- macOS: `~/Library/Logs/Momento/`
- Windows: `%LOCALAPPDATA%\Momento\Logs\`
- Linux: `$XDG_STATE_HOME/momento/` (default `~/.local/state/momento/`)

A `.momento` library is a self-contained directory package with a manifest, SQLite database, originals and thumbnails. The database stores only relative paths, so the package can be moved or backed up as a whole. This is a new application format.

The library menu can copy a library into the application's data directory, export a verified copy, or rebuild its thumbnail cache. On macOS, working libraries must be outside iCloud Drive; importing from or exporting to it is allowed.

The browser extension uses `127.0.0.1:47641`; the application also accepts `momento://import?url=<image URL>`.

## Build packages

From `app/`, build for the current platform or cross-compile all desktop targets:

```sh
go tool mygo build
go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

Packages go to `app/build/`: a macOS app and DMG, Windows installers, and Linux DEB/RPM packages. macOS DMG creation needs a Mac; Windows installer creation needs NSIS. Linux native UI depends on GTK 3 and does not need WebKitGTK.

## Release setup

[The release workflow](.github/workflows/release.yml) runs for version tags or manually. Tags must match the version in `app/mygo.json`. Without the signing configuration, it saves unsigned build artifacts. With all required keys and a requested draft release, it signs packages and uploads them to a draft. It does not publish the draft automatically.

Before distributing a release:

1. Run `go tool mygo keygen`. It stores the update key pair in `mygo/update-keys` under the user's configuration directory. Keep the private key outside the repository and back it up securely. Add the public key to `updates.publicKey` in `app/mygo.json`, with `updates.github` set to `Seaony/Momento`.
2. Configure repository secrets: `MYGO_UPDATER_PRIVATE_KEY`; `MACOS_CERTIFICATE`, `MACOS_CERTIFICATE_PASSWORD`, `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_PASSWORD`; and `WINDOWS_CERTIFICATE`, `WINDOWS_CERTIFICATE_PASSWORD`.
3. Verify signed installation, macOS notarization and signed updates. Test Windows/Linux installation, title bars, window controls, import and export on real machines. Check native file dragging in Finder, Explorer and a Linux file manager, and actual launch at login on all three platforms.

The update public key is not configured yet. Signing, notarization, signed-update acceptance and Windows/Linux GUI acceptance remain release work. Windows/Linux title bars and window controls on real machines, and actual launch at login on all three platforms, are **not run**. A successful cross-build or offline screenshot does not prove these behaviors.

Project conventions are in [AGENTS.md](AGENTS.md).
