<div align="center">
<img src="app/resources/icon.png" width="112" alt="Momento">

# Momento

适用于 macOS、Windows 和 Linux 的本地图片素材库。

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![mygo](https://img.shields.io/badge/mygo-v0.9.0-222222)
![Native UI](https://img.shields.io/badge/UI-native_GPU-222222)

[English](README.md) · [简体中文](README.zh-CN.md) · [浏览器扩展](https://github.com/Seaony/Momento-Chomre-Extension/releases/latest)
</div>

Momento 把图片保存在自己的磁盘上，按颜色、标签和嵌套文件夹整理。一套 Go 代码，在三个桌面平台绘制原生 GPU 界面。

## 功能

- 瀑布流、网格和列表三种视图。搜索名称、文件类型和标签，再按颜色、标签或扩展名筛选。
- 收藏、嵌套文件夹和批量标签编辑。检查器显示调色板、来源链接和 EXIF 数据。
- 导入文件或整个文件夹，按 SHA-256 去重。支持 JPEG、PNG、GIF、WebP、TIFF、BMP、HEIC 和 AVIF；GIF 预览显示首帧。
- 导出原文件、JPEG 或 PNG。删除的素材先进入图库回收站，可以恢复或永久删除。
- 支持英文和简体中文，以及跟随系统、浅色、深色外观。

文件夹旁的数量和打开文件夹后的结果都只包含直接归入该文件夹的素材，不包含子文件夹中的素材。

## 快速开始

安装 Go 1.27.3 或更高版本，然后运行：

```sh
git clone https://github.com/Seaony/Momento.git
cd Momento/app
go tool mygo dev
```

命令行工具版本锁定在 `go.mod`，无需 Bun、Node、网页前端或 cgo。Linux 需要 GTK 3。开发命令打开真正的应用窗口，Go 文件修改后自动重新构建。

macOS 捕获或共享窗口时，左上角可能显示紫色的 `Window sharing` 系统控件，点击仍可关闭、最小化或全屏。这不是开发版标记；未被该控件替代时显示正常的红黄绿按钮。

## 开发

在 `app/` 中执行：

```sh
go build ./...
go vet ./...
go test ./internal/...
go test ./...
```

生成离线界面截图：

```sh
MOMENTO_SCREENSHOTS=1 go test ./ui -run '^TestStateScreenshots$' -count=1
```

截图和清单写入 `/tmp/momento-ui-preview/`。测试直接渲染界面，不会打开应用窗口。

生成开发用样本图库：

```sh
go run ./cmd/sample-library -path /tmp/momento-sample.momento -count 14
```

默认生成 14 张图片，其中 10 张 JPEG、4 张 PNG。大图库用 `-count 100000`。父目录必须存在；命令拒绝覆盖已有图库，失败或取消时只清理本次新建的包。

## 图库与日志

日志达到 5 MB 时轮转，保留三份历史文件：

- macOS：`~/Library/Logs/Momento/`
- Windows：`%LOCALAPPDATA%\Momento\Logs\`
- Linux：`$XDG_STATE_HOME/momento/`，默认 `~/.local/state/momento/`

`.momento` 图库是自包含目录包，内含 manifest、SQLite 数据库、原图和缩略图。数据库只存相对路径，可整体移动或备份。这是全新应用的数据格式。

图库菜单可以把图库复制到应用数据目录、导出经过校验的副本，或重建缩略图缓存。macOS 工作图库必须放在 iCloud Drive 之外；可以从 iCloud Drive 导入或向其导出图库。

浏览器扩展通过 `127.0.0.1:47641` 与应用通信，同时保留 `momento://import?url=<图片地址>`。

## 打包

在 `app/` 中，为当前平台打包或交叉编译全部桌面目标：

```sh
go tool mygo build
go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

产物位于 `app/build/`：macOS 应用和 DMG、Windows 安装包、Linux DEB/RPM。生成 macOS DMG 需要 Mac，生成 Windows 安装包需要 NSIS。Linux 原生界面依赖 GTK 3，不需要 WebKitGTK。

## 发布准备

[发布流程](.github/workflows/release.yml) 由版本标签或手动触发，标签必须与 `app/mygo.json` 的版本一致。缺少签名配置时，只保存未签名构建产物；配置齐全且请求建立草稿时，才签名并上传到草稿发布，不会自动公开草稿。

正式分发前需要完成：

1. 执行 `go tool mygo keygen`，密钥对会写入用户配置目录下的 `mygo/update-keys`。私钥不进仓库，并另行安全备份。在 `app/mygo.json` 的 `updates.publicKey` 写入公钥，`updates.github` 设为 `Seaony/Momento`。
2. 配置仓库密钥：`MYGO_UPDATER_PRIVATE_KEY`；`MACOS_CERTIFICATE`、`MACOS_CERTIFICATE_PASSWORD`、`APPLE_ID`、`APPLE_TEAM_ID`、`APPLE_APP_PASSWORD`；`WINDOWS_CERTIFICATE`、`WINDOWS_CERTIFICATE_PASSWORD`。
3. 验证正式签名安装、macOS 公证和签名更新。在 Windows/Linux 真机安装运行，检查导入、导出和拖出到文件管理器；macOS 也要实际检查拖出到 Finder。

更新公钥尚未配置。正式签名、公证、签名更新验收和 Windows/Linux 图形界面验收仍待完成，交叉编译成功不能代替这些验证。

项目规则见 [AGENTS.md](AGENTS.md)。
