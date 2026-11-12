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
- 支持英文、简体中文和日语，以及跟随系统、浅色、深色外观。

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

## 设置与快捷键

从侧栏打开独立设置窗口，也可以在 macOS 的 **Momento > 设置…** 中打开（`⌘,`）；Windows/Linux 位于 **文件 > 设置…**（`Ctrl+,`）。再次打开会聚焦已有窗口。

- **通用：** 开机时启动、外观、语言和自动检查更新。开机启动默认关闭，开关以操作系统的真实状态为准，失败时显示错误。自动检查更新默认开启，启动后检查一次；关闭后仍可手动检查。
- **浏览器扩展：** 允许导入并查看本机服务状态。默认开启，开关实际启动或停止 `127.0.0.1:47641` 的监听。端口占用和其他服务错误会显示在设置中并记入日志。
- **关于：** 查看版本、检查更新和打开日志文件夹。更新检查显示进度、结果或无法检查的原因，包括开发构建缺少更新配置的情况。

外观和语言切换立即作用于两个窗口及应用菜单。跟随系统支持英文、简体中文和日语，其他语言回落到英文。在 macOS 上，AppKit 原生文件对话框按钮通常在下次启动进程时采用更改后的语言。

应用自动记住上次的视图模式、排序方式和方向、侧栏与检查器是否显示，以及侧栏宽度。

| 操作 | macOS | Windows/Linux |
|---|---|---|
| 新建 / 打开图库 | `⌘N` / `⌘O` | `Ctrl+N` / `Ctrl+O` |
| 导入文件 / 文件夹 | `⌘I` / `⌘⇧I` | `Ctrl+I` / `Ctrl+Shift+I` |
| 导出选中素材 | `⌘E` | `Ctrl+E` |
| 瀑布流 / 网格 / 列表 | `⌘1` / `⌘2` / `⌘3` | `Ctrl+1` / `Ctrl+2` / `Ctrl+3` |
| 搜索 | `⌘F` | `Ctrl+F` |
| 筛选 / 排序 | `⌘⌥F` / `⌘⌥S` | `Ctrl+Alt+F` / `Ctrl+Alt+S` |
| 切换检查器 | `⌘⌥I` | `Ctrl+Alt+I` |
| 设置 | `⌘,` | `Ctrl+,` |

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
MOMENTO_SETTINGS_SCREENSHOTS=1 go test ./ui -run '^TestSettingsWindowScreenshots$' -count=1
```

主界面状态截图写入 `/tmp/momento-titlebar-settings-preview/`，设置窗口状态写入 `/tmp/momento-settings-window/`，各自附带清单。两组均覆盖中英日三语和浅深色。测试直接渲染界面，不会打开应用窗口。

生成开发用样本图库：

```sh
go run ./cmd/sample-library -path /tmp/momento-sample.momento -count 14
```

默认生成 14 张图片，其中 10 张 JPEG、4 张 PNG。大图库用 `-count 100000`。父目录必须存在；命令拒绝覆盖已有图库，失败或取消时只清理本次新建的包。

## 配置、图库与日志

设置和界面状态保存在应用配置目录的 `preferences.json` 中：

- macOS：`~/Library/Application Support/Momento/`
- Windows：`%APPDATA%\Momento\`
- Linux：`$XDG_CONFIG_HOME/Momento/`，默认 `~/.config/Momento/`

已有配置文件缺少新字段时使用默认值。初始主题和语言跟随系统；初始视图为瀑布流，按添加时间从新到旧排序，侧栏与检查器均显示。最近图库路径单独保存在同目录的 `recent-libraries.json` 中。

忘记密码锁的密码时，先退出 Momento，再删除上述配置目录中 `preferences.json` 的 `passwordHash` 字段，即可解除。密码锁只锁界面，不加密图库文件。

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
3. 验证正式签名安装、macOS 公证和签名更新。在 Windows/Linux 真机安装运行，检查标题栏、窗口按钮、导入和导出。实际检查拖出到 Finder、资源管理器和 Linux 文件管理器，以及三端开机启动的效果。

更新公钥尚未配置。正式签名、公证、签名更新验收和 Windows/Linux 图形界面验收仍待完成。Windows/Linux 真机标题栏与窗口按钮、三端开机启动实效均为 **not run（未执行）**。交叉编译或离线截图成功不能代替这些验证。

项目规则见 [AGENTS.md](AGENTS.md)。
