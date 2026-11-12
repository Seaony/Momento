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
- Use English or Simplified Chinese, with system, light or dark appearance.

## Quick start

Install Go 1.27.3 or later, then run:

```sh
git clone https://github.com/Seaony/Momento.git
cd Momento/app
go tool mygo dev
```

The CLI is pinned in `go.mod`; no Bun, Node, web frontend or cgo is needed. Linux requires GTK 3. The development command opens a real application window and rebuilds it when Go files change.

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
```

The images and their manifest are written to `/tmp/momento-ui-preview/`. These tests render the interface without opening a window.

To create a sample library for development:

```sh
go run ./cmd/sample-library -path /tmp/momento-sample.momento -count 14
```

The default is 14 images: 10 JPEG and 4 PNG. Use `-count 100000` for a large library. The parent directory must exist; the command refuses to overwrite an existing library and removes only its new package if generation fails or is cancelled.

## Libraries and logs

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
3. Verify signed installation, macOS notarization and signed updates. Test Windows/Linux installation, import and export on real machines. Check native file dragging in Finder, Explorer and a Linux file manager.

The update public key is not configured yet. Signing, notarization, signed-update acceptance and Windows/Linux GUI acceptance remain release work; a successful cross-build does not prove them.

Project conventions are in [AGENTS.md](AGENTS.md).
