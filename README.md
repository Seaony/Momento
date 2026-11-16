<div align="center">
<img src="app/resources/icon.png" width="112" alt="Momento">

# Momento

A local image library for macOS, Windows and Linux.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![mygo](https://img.shields.io/badge/mygo-v0.9.0-222222)
![Native UI](https://img.shields.io/badge/UI-native_GPU-222222)
![License](https://img.shields.io/badge/license-MIT-222222)

[English](README.md) · [简体中文](README.zh-CN.md) · [Browser extension](https://github.com/Seaony/Momento-Chomre-Extension)
</div>

Momento collects and organizes reference images, screenshots, photos and short videos. Everything stays on your own disk. A library is a single `.momento` folder that you can move, copy or back up as a whole. There is no account and no cloud sync.

The whole app is one Go codebase. It uses [mygo](https://github.com/egoist/mygo) to draw a native GPU interface on all three desktop platforms, with no web frontend and no cgo.

## Features

**Browse** in masonry, justified rows or list view, with adjustable thumbnail size. Press Space to open a preview window that zooms, pans and plays animated images and videos; press H to flip it horizontally and G to view it in grayscale. You can also run a full-screen slideshow of the current list.

**Organize** with nested folders, tags, favorites, 1 to 5 star ratings and notes, all editable in batches. Copy tags from one image and paste them onto others. Smart folders collect images by name, note, source, tag, color, format, dimensions, file size, rating, duration and date added. Batch rename numbers files from a template or finds and replaces text.

**Find** images by name, tag, note, source link or format, then filter by color, tag or extension. Find Similar Images groups images that look the same; pick one to keep, and the tags, folders, notes and ratings of the others are merged into it before they go to the trash.

**Inspect** palettes, dimensions, file size, path, source link and EXIF data. Draw a box on an image to leave an annotation; annotations are searchable too.

**Import** by dragging, pasting, or choosing files or whole folders, with duplicates detected by content. Watched folders import anything you put in them and move the originals to the trash. Keywords embedded in images can become tags.

**Export** originals, or convert to JPEG or PNG with an optional longest-edge limit, and save the settings you use often as presets. Live Photos can be exported together with their videos. Deleted images go to the library's trash first, where you can restore them or delete them for good.

**Also**: a browser extension that saves images from web pages, a password lock, a menu bar icon, launch at login and customizable shortcuts. The interface comes in English, Simplified Chinese and Japanese, with light and dark appearances.

## Supported formats

| Kind | Formats | Notes |
|---|---|---|
| Images | JPEG, PNG, GIF, WebP, TIFF, BMP, HEIC, AVIF | All platforms |
| Camera RAW, PSD and more | DNG, CR2, CR3, NEF, ARW, RAF, ORF, RW2 and others, plus PSD, ICO, ICNS, JPEG 2000, TGA, EXR, HDR | macOS decodes them with the system; Windows only gets the system thumbnail, up to 2560 pixels; not supported on Linux yet |
| Videos | MP4, MOV, M4V, WebM | The system captures the cover frame; WebM on macOS and all videos on Linux need ffmpeg |
| Live Photos | A HEIC or JPEG with a MOV of the same name | Paired automatically on import |

## Install

There are no installers yet, so run it from source. Install Go 1.27.3 or later, then:

```sh
git clone https://github.com/Seaony/Momento.git
cd Momento/app
go tool mygo dev
```

The CLI is pinned in `go.mod`; you do not need Bun, Node or cgo. Linux needs GTK 3. The command opens the real app window and rebuilds it when Go files change.

## Browser extension

[Momento Image Saver](https://github.com/Seaony/Momento-Chomre-Extension) is the companion Chrome extension. Right-click an image on a web page to save it to Momento; on Xiaohongshu, Instagram, Pixiv, Weibo, X and similar sites it can save a whole post at once. The extension only talks to Momento on your own computer (`127.0.0.1:47641`), and you can turn it off on the Browser Extension page in Settings.

The app also accepts links such as `momento://import?url=<image URL>`.

## Shortcuts

These are the defaults; change any of them on the Shortcuts page in Settings.

| Action | macOS | Windows/Linux |
|---|---|---|
| New / open library | `⌘N` / `⌘O` | `Ctrl+N` / `Ctrl+O` |
| Import files / folder | `⌘I` / `⌘⇧I` | `Ctrl+I` / `Ctrl+Shift+I` |
| Export | `⌘E` | `Ctrl+E` |
| Masonry / list / justified rows | `⌘1` / `⌘2` / `⌘3` | `Ctrl+1` / `Ctrl+2` / `Ctrl+3` |
| Search | `⌘K` | `Ctrl+K` |
| Filter / sort | `⌘⌥F` / `⌘⌥S` | `Ctrl+Alt+F` / `Ctrl+Alt+S` |
| Show or hide inspector | `⌘⌥I` | `Ctrl+Alt+I` |
| Larger / smaller thumbnails | `⌘=` / `⌘-` | `Ctrl+=` / `Ctrl+-` |
| Copy / paste tags | `⌘⇧C` / `⌘⇧V` | `Ctrl+Shift+C` / `Ctrl+Shift+V` |
| Add to folder | `⇧F` | `Shift+F` |
| Preview | `Space` | `Space` |
| Flip / grayscale / annotate in preview | `H` / `G` / `A` | `H` / `G` / `A` |
| Lock | `⌘L` | `Ctrl+L` |
| Settings | `⌘,` | `Ctrl+,` |

## Where your data lives

A `.momento` library holds a manifest, a SQLite database, the originals and their thumbnails. The database stores only relative paths, so the package can be moved or backed up as a whole. On macOS a library in use cannot live in iCloud Drive, though you can import from it or export to it. The library menu can also copy a library into the app's data directory, export a verified copy, or rebuild thumbnails.

Settings and interface state are saved in `preferences.json` in the app's configuration directory, and recently opened libraries in `recent-libraries.json` next to it:

- macOS: `~/Library/Application Support/Momento/`
- Windows: `%APPDATA%\Momento\`
- Linux: `$XDG_CONFIG_HOME/Momento/` (default `~/.config/Momento/`)

If you forget the password lock password, quit Momento and delete the `passwordHash` field from `preferences.json`. The lock only covers the interface; it does not encrypt library files.

Logs rotate at 5 MB and keep three older files:

- macOS: `~/Library/Logs/Momento/`
- Windows: `%LOCALAPPDATA%\Momento\Logs\`
- Linux: `$XDG_STATE_HOME/momento/` (default `~/.local/state/momento/`)

## Development

From `app/`:

```sh
go build ./...
go vet ./...
go test ./...
```

Render interface screenshots without opening a window:

```sh
MOMENTO_SCREENSHOTS=1 go test ./ui -run '^TestStateScreenshots$' -count=1
MOMENTO_SETTINGS_SCREENSHOTS=1 go test ./ui -run '^TestSettingsWindowScreenshots$' -count=1
```

Main window screenshots go to `/tmp/momento-titlebar-settings-preview/` and settings window screenshots to `/tmp/momento-settings-window/`, in English, Chinese and Japanese, light and dark.

Generate a sample library:

```sh
go run ./cmd/sample-library -path /tmp/momento-sample.momento -count 14
```

The default is 14 images (10 JPEG, 4 PNG); use `-count 100000` to test a large library. The parent directory must exist. The command never overwrites an existing library, and on failure or cancellation it removes only the package it created.

Project conventions are in [AGENTS.md](AGENTS.md).

## Build packages

From `app/`, build for the current platform or cross-compile every desktop target:

```sh
go tool mygo build
go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

Packages go to `app/build/`: the macOS app and DMG, Windows installers, and Linux DEB and RPM packages. Building a DMG needs a Mac; building Windows installers needs NSIS. The Linux interface depends on GTK 3 and does not need WebKitGTK.

## Releasing

[The release workflow](.github/workflows/release.yml) runs for version tags or by hand, and the tag must match the version in `app/mygo.json`. Without the full signing setup it only keeps unsigned build artifacts. With every key in place and a draft requested, it signs the packages and uploads them to a draft release; it never publishes the draft by itself.

Before the first real release:

1. Run `go tool mygo keygen`. It writes the update key pair to `mygo/update-keys` in your user configuration directory. Keep the private key out of the repository and back it up. Put the public key in `updates.publicKey` in `app/mygo.json`, with `updates.github` set to `Seaony/Momento`.
2. Add the repository secrets `MYGO_UPDATER_PRIVATE_KEY`; `MACOS_CERTIFICATE`, `MACOS_CERTIFICATE_PASSWORD`, `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_PASSWORD`; and `WINDOWS_CERTIFICATE`, `WINDOWS_CERTIFICATE_PASSWORD`.
3. Check signed installation, macOS notarization and signed updates. On real Windows and Linux machines, check installation, title bars, window buttons, import and export, dragging files out to the file manager, and launch at login on all three platforms.

The update public key is not set yet, and none of these checks have been run. A clean cross-build or offline screenshot does not replace testing on real machines.

## License

[MIT](LICENSE)
