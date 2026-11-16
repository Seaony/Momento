<div align="center">
<img src="app/resources/icon.png" width="112" alt="Momento">

# Momento

适用于 macOS、Windows 和 Linux 的本地图片素材库。

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![mygo](https://img.shields.io/badge/mygo-v0.9.0-222222)
![Native UI](https://img.shields.io/badge/UI-native_GPU-222222)
![License](https://img.shields.io/badge/license-MIT-222222)

[English](README.md) · [简体中文](README.zh-CN.md) · [浏览器扩展](https://github.com/Seaony/Momento-Chomre-Extension)
</div>

Momento 用来收集和整理参考图、截图、照片和短视频。所有文件都放在你自己的磁盘上，一个图库就是一个 `.momento` 文件夹，可以整体移动、复制和备份。不需要账号，也不联网同步。

整个应用是一套 Go 代码，用 [mygo](https://github.com/egoist/mygo) 在三个桌面平台上绘制原生 GPU 界面，没有网页前端，也不用 cgo。

## 功能

**浏览**：瀑布流、等高行和列表三种视图，缩略图大小可调。空格打开预览窗口，可以缩放、平移、播放动图和视频，按 H 水平翻转、按 G 看灰度。也可以全屏放映当前列表。

**整理**：嵌套文件夹、标签、收藏、1 到 5 星评分和备注，都能批量修改；标签可以复制后粘贴到别的图片上。智能文件夹按名称、备注、来源、标签、颜色、格式、尺寸、文件大小、评分、时长和添加时间自动收集图片。批量重命名可以按模板编号，也可以查找替换。

**查找**：按名称、标签、备注、来源链接和格式搜索，再按颜色、标签和扩展名筛选。「查找相似图片」会把看起来一样的图分成一组，选一张保留，其余几张的标签、文件夹、备注和评分并进这一张，然后移到回收站。

**检查器**：显示调色板、尺寸、大小、路径、来源链接和 EXIF。可以在图上框出区域写批注，批注内容也能被搜索到。

**导入**：拖入、粘贴、选文件或整个文件夹，按内容去重。可以监视几个文件夹，放进去的图片自动导入，原文件移到废纸篓。图片里自带的关键字可以直接变成标签。

**导出**：导出原文件，或转成 JPEG、PNG；可以限制最长边，常用的组合存成预设。实况照片可以连同视频一起导出。删除的素材先进图库回收站，可以恢复，也可以永久删除。

**其他**：浏览器扩展可以把网页上的图片直接存进来。可以设置密码锁、在菜单栏显示图标、开机启动、改快捷键。界面支持简体中文、英文和日语，以及浅色、深色外观。

## 支持的格式

| 类型 | 格式 | 说明 |
|---|---|---|
| 图片 | JPEG、PNG、GIF、WebP、TIFF、BMP、HEIC、AVIF | 三个平台都支持 |
| 相机 RAW、PSD 等 | DNG、CR2、CR3、NEF、ARW、RAF、ORF、RW2 等，以及 PSD、ICO、ICNS、JPEG 2000、TGA、EXR、HDR | macOS 用系统解码；Windows 只能拿到系统缩略图，预览最长 2560 像素；Linux 暂不支持 |
| 视频 | MP4、MOV、M4V、WebM | 封面由系统截取；macOS 上的 WebM 和 Linux 上的所有视频需要装 ffmpeg |
| 实况照片 | 同名的 HEIC 或 JPEG 加 MOV | 导入时自动配对 |

## 安装

还没有发布安装包，目前需要从源码运行。安装 Go 1.27.3 或更高版本，然后：

```sh
git clone https://github.com/Seaony/Momento.git
cd Momento/app
go tool mygo dev
```

命令行工具的版本锁在 `go.mod` 里，不需要 Bun、Node 或 cgo。Linux 需要 GTK 3。这条命令会打开真正的应用窗口，改了 Go 文件会自动重新构建。

## 浏览器扩展

[Momento Image Saver](https://github.com/Seaony/Momento-Chomre-Extension) 是配套的 Chrome 扩展。在网页图片上右键就能存进 Momento，在小红书、Instagram、Pixiv、微博、X 等网站上还能一次存下整帖。扩展只和本机的 Momento 通信（`127.0.0.1:47641`），可以在设置的「浏览器扩展」页关掉。

应用也接受 `momento://import?url=<图片地址>` 这样的链接。

## 快捷键

下面是默认按键，都可以在设置的「快捷键」页修改。

| 操作 | macOS | Windows/Linux |
|---|---|---|
| 新建 / 打开图库 | `⌘N` / `⌘O` | `Ctrl+N` / `Ctrl+O` |
| 导入文件 / 文件夹 | `⌘I` / `⌘⇧I` | `Ctrl+I` / `Ctrl+Shift+I` |
| 导出 | `⌘E` | `Ctrl+E` |
| 瀑布流 / 列表 / 等高行 | `⌘1` / `⌘2` / `⌘3` | `Ctrl+1` / `Ctrl+2` / `Ctrl+3` |
| 搜索 | `⌘K` | `Ctrl+K` |
| 筛选 / 排序 | `⌘⌥F` / `⌘⌥S` | `Ctrl+Alt+F` / `Ctrl+Alt+S` |
| 显示或隐藏检查器 | `⌘⌥I` | `Ctrl+Alt+I` |
| 放大 / 缩小缩略图 | `⌘=` / `⌘-` | `Ctrl+=` / `Ctrl+-` |
| 复制 / 粘贴标签 | `⌘⇧C` / `⌘⇧V` | `Ctrl+Shift+C` / `Ctrl+Shift+V` |
| 加入文件夹 | `⇧F` | `Shift+F` |
| 预览 | `Space` | `Space` |
| 预览里翻转 / 灰度 / 批注 | `H` / `G` / `A` | `H` / `G` / `A` |
| 锁定 | `⌘L` | `Ctrl+L` |
| 设置 | `⌘,` | `Ctrl+,` |

## 数据放在哪里

`.momento` 图库里有 manifest、SQLite 数据库、原图和缩略图。数据库只记相对路径，所以整个包可以随意移动或备份。macOS 上正在使用的图库不能放在 iCloud Drive 里，但可以从 iCloud Drive 导入，或者导出到那里。图库菜单里还可以把图库复制到应用数据目录、导出一份校验过的副本，或者重建缩略图。

设置和界面状态存在应用配置目录的 `preferences.json` 里，最近打开的图库存在同一目录的 `recent-libraries.json`：

- macOS：`~/Library/Application Support/Momento/`
- Windows：`%APPDATA%\Momento\`
- Linux：`$XDG_CONFIG_HOME/Momento/`，默认 `~/.config/Momento/`

忘了密码锁的密码：先退出 Momento，再删掉 `preferences.json` 里的 `passwordHash` 字段。密码锁只锁界面，不加密图库文件。

日志满 5 MB 就轮换，保留三份旧文件：

- macOS：`~/Library/Logs/Momento/`
- Windows：`%LOCALAPPDATA%\Momento\Logs\`
- Linux：`$XDG_STATE_HOME/momento/`，默认 `~/.local/state/momento/`

## 开发

在 `app/` 里执行：

```sh
go build ./...
go vet ./...
go test ./...
```

不打开窗口，直接渲染界面截图：

```sh
MOMENTO_SCREENSHOTS=1 go test ./ui -run '^TestStateScreenshots$' -count=1
MOMENTO_SETTINGS_SCREENSHOTS=1 go test ./ui -run '^TestSettingsWindowScreenshots$' -count=1
```

主界面截图写到 `/tmp/momento-titlebar-settings-preview/`，设置窗口写到 `/tmp/momento-settings-window/`，覆盖中英日三种语言和浅深两种外观。

生成一个样本图库：

```sh
go run ./cmd/sample-library -path /tmp/momento-sample.momento -count 14
```

默认 14 张图（10 张 JPEG、4 张 PNG），测大图库可以用 `-count 100000`。父目录必须已经存在；命令不会覆盖已有图库，失败或取消时只删掉这次新建的包。

项目约定见 [AGENTS.md](AGENTS.md)。

## 打包

在 `app/` 里为当前平台打包，或交叉编译全部桌面平台：

```sh
go tool mygo build
go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

产物在 `app/build/`：macOS 的 App 和 DMG、Windows 安装包、Linux 的 DEB 和 RPM。打 DMG 需要 Mac，打 Windows 安装包需要 NSIS。Linux 界面依赖 GTK 3，不需要 WebKitGTK。

## 发布

[发布流程](.github/workflows/release.yml) 由版本标签或手动触发，标签必须和 `app/mygo.json` 里的版本一致。签名配置不全时只保存未签名的构建产物；配置齐全并要求建草稿时，才签名并上传到草稿，不会自动公开。

正式发布前要做的：

1. 执行 `go tool mygo keygen`，密钥对会写到用户配置目录下的 `mygo/update-keys`。私钥不进仓库，另外妥善备份。把公钥写进 `app/mygo.json` 的 `updates.publicKey`，`updates.github` 设为 `Seaony/Momento`。
2. 配置仓库密钥：`MYGO_UPDATER_PRIVATE_KEY`；`MACOS_CERTIFICATE`、`MACOS_CERTIFICATE_PASSWORD`、`APPLE_ID`、`APPLE_TEAM_ID`、`APPLE_APP_PASSWORD`；`WINDOWS_CERTIFICATE`、`WINDOWS_CERTIFICATE_PASSWORD`。
3. 验证签名安装、macOS 公证和签名更新；在 Windows、Linux 真机上检查安装、标题栏、窗口按钮、导入导出、往文件管理器拖文件，以及三个平台的开机启动。

目前更新公钥还没配置，上面这些验证都还没做过。交叉编译成功或离线截图正常，不能代替真机验证。

## 许可证

[MIT](LICENSE)
