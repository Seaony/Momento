<div align="center">
<img src="app/resources/icon.png" width="112" alt="Momento">

# Momento

A local image library for macOS, Windows and Linux.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white) ![mygo](https://img.shields.io/badge/mygo-v0.9.0-222222) ![Native UI](https://img.shields.io/badge/UI-native_GPU-222222)

[English](README.md) · [简体中文](README.zh-CN.md) · [Browser extension](https://github.com/Seaony/Momento-Chomre-Extension/releases/latest)
</div>

Momento keeps images on your own disk, organized by color, tags and nested folders. The application is being rebuilt with Go and mygo native UI for three desktop platforms.

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

Application logs rotate at 5 MB, keeping three previous files:

- macOS: `~/Library/Logs/Momento/`
- Windows: `%LOCALAPPDATA%\Momento\Logs\`
- Linux: `$XDG_STATE_HOME/momento/` (default `~/.local/state/momento/`)

A `.momento` library is a self-contained directory package with a manifest, SQLite database, originals and thumbnails. The database stores only relative paths, so the package can be moved or backed up as a whole. This is a new application format.

The browser extension uses `127.0.0.1:47641`; the application also accepts `momento://import?url=<image URL>`.

Project conventions are in [AGENTS.md](AGENTS.md).
