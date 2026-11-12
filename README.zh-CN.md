<div align="center">
<img src="app/resources/icon.png" width="112" alt="Momento">

# Momento

适用于 macOS、Windows 和 Linux 的本地图片素材库。

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white) ![mygo](https://img.shields.io/badge/mygo-v0.9.0-222222) ![Native UI](https://img.shields.io/badge/UI-native_GPU-222222)

[English](README.md) · [简体中文](README.zh-CN.md) · [浏览器扩展](https://github.com/Seaony/Momento-Chomre-Extension/releases/latest)
</div>

Momento 把图片保存在自己的磁盘上，按颜色、标签和嵌套文件夹整理。目前正用 Go 和 mygo 原生界面重写为三端桌面应用。

## 快速开始

安装 Go 1.27.3 或更高版本，然后运行：

```sh
git clone https://github.com/Seaony/Momento.git
cd Momento/app
go tool mygo dev
```

命令行工具版本锁定在 `go.mod`，无需 Bun、Node、网页前端或 cgo。Linux 需要 GTK 3。开发命令打开真正的应用窗口，Go 文件修改后自动重新构建。

## 开发

在 `app/` 中执行：

```sh
go build ./...
go vet ./...
go test ./internal/...
go test ./...
```

日志达到 5 MB 时轮转，保留三份历史文件：

- macOS：`~/Library/Logs/Momento/`
- Windows：`%LOCALAPPDATA%\Momento\Logs\`
- Linux：`$XDG_STATE_HOME/momento/`，默认 `~/.local/state/momento/`

`.momento` 图库是自包含目录包，内含 manifest、SQLite 数据库、原图和缩略图。数据库只存相对路径，可整体移动或备份。这是全新应用的数据格式。

浏览器扩展通过 `127.0.0.1:47641` 与应用通信，同时保留 `momento://import?url=<图片地址>`。

项目规则见 [AGENTS.md](AGENTS.md)。
