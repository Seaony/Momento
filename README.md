<div align="center">
  <br />
  <img src="Momento/Assets.xcassets/AppIcon.appiconset/icon_256x256.png" width="128" alt="Momento App Icon" />
  <h1>Momento</h1>
  <p><strong>一个安静、漂亮、原生的 macOS 图片素材库。</strong></p>
  <p>
    把散落在 Finder、浏览器和项目文件夹里的灵感图片收进一个本地素材库，
    用标签、文件夹、颜色和视图把它们重新整理好。
  </p>
  <p>
    <a href="https://github.com/Seaony/Momento/releases/latest"><strong>下载最新版</strong></a>
    ·
    <a href="#功能亮点">功能亮点</a>
    ·
    <a href="#适合谁">适合谁</a>
    ·
    <a href="#自动更新">自动更新</a>
  </p>
  <p>
    <img alt="macOS 15+" src="https://img.shields.io/badge/macOS-15%2B-111111?style=flat-square&labelColor=222222" />
    <img alt="Swift" src="https://img.shields.io/badge/Swift-6-FA7343?style=flat-square&labelColor=222222" />
    <img alt="Native App" src="https://img.shields.io/badge/Native-macOS-0A84FF?style=flat-square&labelColor=222222" />
    <img alt="Updates" src="https://img.shields.io/badge/Updates-Sparkle-7A6FF0?style=flat-square&labelColor=222222" />
  </p>
  <br />
</div>

## 为什么是 Momento

Momento 想解决的是一个很日常的问题：图片素材越来越多，但真正要找的时候却总是散在下载目录、桌面、聊天记录和项目文件夹里。

它把素材管理做成本地优先的 macOS App。素材保存在你自己的磁盘上，界面尽量克制，操作尽量直接：导入、浏览、筛选、整理、预览、导出，都围绕图片素材的日常工作流展开。

## 功能亮点

<table>
  <tr>
    <td width="33%">
      <h3>集中收纳</h3>
      <p>创建独立素材库，把图片、GIF 和文件夹里的素材批量导入。相同文件会自动去重，不会反复占用空间。</p>
    </td>
    <td width="33%">
      <h3>三种视图</h3>
      <p>瀑布流适合灵感浏览，网格适合快速扫描，列表适合查看名称、尺寸和信息。不同整理场景可以随时切换。</p>
    </td>
    <td width="33%">
      <h3>快速整理</h3>
      <p>用收藏、标签和文件夹整理图片。导入文件夹时会保留原始层级，后续也可以继续手动归类。</p>
    </td>
  </tr>
  <tr>
    <td width="33%">
      <h3>颜色筛选</h3>
      <p>导入时自动分析图片调色板，并归入 24 个常用色系。找黑色、蓝色、橙色或绿色氛围图时更直接。</p>
    </td>
    <td width="33%">
      <h3>详细信息</h3>
      <p>检查器展示预览、标题、色板、标签、文件夹、基础信息和 EXIF。能读到的相机、镜头、曝光数据会自动展示。</p>
    </td>
    <td width="33%">
      <h3>拖拽导出</h3>
      <p>选中图片后可以直接拖到 Finder 或桌面。也可以通过导出面板选择原文件、JPEG 或 PNG。</p>
    </td>
  </tr>
</table>

## 适合谁

- 设计师：管理参考图、Moodboard、界面截图、品牌素材。
- 摄影师：快速筛选样片、查看 EXIF、按颜色和标签归类。
- 内容创作者：保存封面参考、素材图、灵感图片和可复用视觉资产。
- 独立开发者：整理 App 截图、产品素材、社媒配图和发布资源。

## 日常体验

- 从 Finder 选择图片或文件夹导入。
- 从浏览器右键把远程图片保存到 Momento。
- 用瀑布流浏览整组素材，用搜索快速定位图片名称。
- 用颜色、标签、文件类型筛选当前素材库。
- 用空格快速预览，用 `Command + Delete` 删除选中的图片。
- 把选中的图片直接拖出到桌面或项目文件夹。

## 支持的内容

当前版本聚焦图片素材：

- 支持 macOS 可识别的常见图片格式和 GIF。
- RAW 是否可导入取决于当前系统的图片解码能力。
- SVG、PDF、视频暂不作为素材导入格式。

素材库是本地 package，扩展名为 `.momento`。原始文件、缩略图和数据库都保存在素材库里，不需要云端账号。

## 自动更新

Momento 集成 Sparkle 2。发布新版本后，App 可以通过 GitHub Releases 和 appcast 检查更新、下载更新包并完成安装。

如果你只是使用 App，直接下载最新版 DMG 即可：

<p align="center">
  <a href="https://github.com/Seaony/Momento/releases/latest"><strong>前往 Releases 下载 Momento</strong></a>
</p>

<details>
  <summary>开发者信息</summary>

### 技术栈

- Swift 6
- SwiftUI + AppKit
- Core Data + SQLite
- ImageIO / UniformTypeIdentifiers
- Sparkle 2
- XCTest

### 本地构建

```bash
open Momento.xcodeproj
```

```bash
xcodebuild \
  -project Momento.xcodeproj \
  -scheme Momento \
  -configuration Debug \
  -destination 'platform=macOS' \
  build
```

### 测试

```bash
xcodebuild \
  -project Momento.xcodeproj \
  -scheme Momento \
  -destination 'platform=macOS' \
  test
```

### 发布

```bash
scripts/prepare-release.sh <marketing-version> <build-number>
```

发布脚本会构建 Release、生成 DMG、签名 Sparkle 更新包、更新 `appcast.xml`，并创建 GitHub Release。

### 相关文档

- [FEATURE.md](FEATURE.md)
- [docs/update-release-flow.md](docs/update-release-flow.md)
- [docs/full-project-review-report.md](docs/full-project-review-report.md)
- [AGENTS.md](AGENTS.md)

</details>
