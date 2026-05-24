# Momento

Momento 是一个原生 macOS 素材管理应用，用来集中保存、浏览、筛选、整理和导出本地图片素材。项目目标是保持 macOS 原生体验：SwiftUI 负责主要界面，AppKit 负责高性能素材列表、系统拖拽、Quick Look 和窗口细节。

当前版本重点覆盖图片素材管理，不是 Electron / Web App 壳，也不依赖远端服务保存用户素材。

## 功能概览

- 素材库管理：创建、打开、导入、导出、重命名、删除、最近素材库列表。
- 图片导入：支持 macOS `UTType` / ImageIO 可识别的图片与 GIF，支持文件夹导入并映射原始目录层级。
- 去重存储：导入时计算 SHA-256，按内容寻址保存原始文件，重复文件不重复复制。
- 元数据：导入时提取尺寸、文件信息、EXIF、颜色调色板和 24 个颜色分类。
- 浏览模式：瀑布流、网格、列表三种视图，基于 `NSCollectionView` 渲染大规模素材。
- 整理能力：收藏、文件夹归类、标签关联、标签管理、软删除回收站。
- 筛选和排序：按颜色、标签、文件类型筛选；按添加时间、文件名称、文件大小排序。
- 检查器：查看预览图、调色板、标题、标签、文件夹、基础信息和 EXIF 信息。
- 导出：支持拖出到 Finder / 桌面，也支持导出原文件、JPEG、PNG。
- 快速预览：支持 Quick Look / 空格预览。
- 浏览器导入：App 内启动本机导入服务，供 Chrome 扩展把远程图片发送到 Momento。
- 自动更新：集成 Sparkle 2，通过 GitHub Releases 托管 DMG 和 appcast。

## 当前边界

- 当前导入范围是图片与 GIF；SVG / PDF / 视频暂不作为素材导入格式。
- RAW 是否可导入取决于当前系统的 `UTType` / ImageIO 支持，项目内没有单独的 RAW 工作流。
- 缩略图和预览图是可重建缓存，原始素材文件和 Core Data 数据库才是权威数据。
- UI 主要面向 macOS 26 Liquid Glass 视觉；旧系统不是当前主要兼容目标。

## 技术栈

- Swift 6
- SwiftUI + AppKit
- Observation / async-await
- Core Data + SQLite
- ImageIO / UniformTypeIdentifiers
- Sparkle 2
- XCTest

## 环境要求

- macOS 26 或更新版本
- Xcode 17 或更新版本
- Git
- 发布版本需要 GitHub CLI：

```bash
brew install gh
gh auth login
```

## 快速开始

克隆仓库后，直接用 Xcode 打开工程：

```bash
open Momento.xcodeproj
```

也可以使用命令行构建：

```bash
xcodebuild \
  -project Momento.xcodeproj \
  -scheme Momento \
  -configuration Debug \
  -destination 'platform=macOS' \
  build
```

运行测试：

```bash
xcodebuild \
  -project Momento.xcodeproj \
  -scheme Momento \
  -destination 'platform=macOS' \
  test
```

基础 diff 校验：

```bash
git diff --check
```

## 目录结构

```text
.
├── Momento/                         # App 源码
│   ├── AppKitBridge/                # NSCollectionView、Quick Look、拖拽、窗口桥接
│   ├── Core/                        # 值类型模型、本地化、应用设置
│   ├── DesignSystem/                # Liquid Glass 样式和视觉 token
│   ├── Features/                    # Sidebar、Shell、Inspector、Settings 等功能界面
│   ├── Services/                    # 导入、导出、缩略图、颜色分析、更新、浏览器导入
│   └── Storage/                     # 素材库包、Core Data、manifest、security-scoped access
├── MomentoTests/                    # XCTest 测试和架构护栏
├── docs/                            # 设计、review、发布流程文档
├── scripts/                         # 发布脚本
├── appcast.xml                      # Sparkle 更新 feed
├── FEATURE.md                       # 产品和架构目标
└── AGENTS.md                        # 当前仓库的 agent 工作约束
```

## 素材库结构

Momento 素材库是一个本地 package，扩展名为 `.momento`，并兼容旧的 `.momentolibrary`。

```text
<LibraryName>.momento/
├── manifest.json
├── database/
│   └── library.sqlite
├── assets/
│   └── ab/
│       └── <sha256>.<ext>
├── thumbnails/
├── previews/
└── metadata/
    └── import-sessions/
```

规则：

- `manifest.json` 保存库级信息，例如库 ID、展示名称、schema 版本和创建时间。
- `database/library.sqlite` 是素材 metadata 的唯一权威来源。
- `assets/` 保存原始素材副本，路径由内容 SHA-256 决定。
- `thumbnails/` 和 `previews/` 是缓存，可以删除后重建。
- 最近素材库通过 security-scoped bookmark 重新获得访问权限。

## 核心数据模型

当前 Core Data 模型围绕这些概念组织：

- `AssetRecord`：素材条目、文件路径、尺寸、EXIF、颜色、收藏、回收站状态等。
- `TagRecord` / `AssetTagRecord`：标签和素材的多对多关系。
- `FolderRecord` / `AssetFolderMembershipRecord`：文件夹树和素材归类关系。
- `AssetColorRecord`：调色板颜色与颜色筛选数据。

UI 层使用 `AssetItem`、`TagItem`、`AssetFolder` 等值类型，避免 SwiftUI 直接持有 Core Data 对象。

## 开发说明

常用入口：

- 主窗口协调层：[Momento/ContentView.swift](Momento/ContentView.swift)
- 全局状态聚合：[Momento/Core/LibraryStore.swift](Momento/Core/LibraryStore.swift)
- 素材列表：[Momento/AppKitBridge/AssetCollectionGridView.swift](Momento/AppKitBridge/AssetCollectionGridView.swift)
- 右侧检查器：[Momento/Features/Inspector/MomentoInspectorView.swift](Momento/Features/Inspector/MomentoInspectorView.swift)
- 视觉 token：[Momento/DesignSystem/MomentoGlass.swift](Momento/DesignSystem/MomentoGlass.swift)
- 导入服务：[Momento/Services/AssetImportService.swift](Momento/Services/AssetImportService.swift)
- 持久化层：[Momento/Storage/LibraryMetadataStore.swift](Momento/Storage/LibraryMetadataStore.swift)

开发原则：

- 保持 SwiftUI 为主，AppKit 只用于 SwiftUI 不适合承载的能力。
- 视觉组件优先复用 `DesignSystem` 中的 Liquid Glass 样式。
- 不把缩略图、预览图当作源数据保存。
- 数据安全路径优先使用软删除，再由用户确认永久删除。
- 纯视觉微调不新增过细测试；数据、导入、删除、更新、架构护栏需要保留测试覆盖。

## 自动更新和发布

Momento 使用 Sparkle 2 做 App 内更新：

- appcast: `https://seaony.github.io/Momento/appcast.xml`
- Release 托管：GitHub Releases
- 发布产物：`dist/Momento-<version>.dmg`

发布脚本：

```bash
scripts/prepare-release.sh 1.0.1 2
```

脚本会自动完成：

- 如有未提交的非忽略改动，先自动提交。
- 更新 Xcode 版本号和 build number。
- 构建 Release App。
- 生成 DMG。
- 使用 Sparkle `sign_update` 签名更新包。
- 更新 `appcast.xml`。
- 提交 release metadata、创建 tag、push 当前分支和 tag。
- 创建 GitHub Release 并上传 DMG。

更多细节见 [docs/update-release-flow.md](docs/update-release-flow.md)。

## 常用命令

```bash
# 构建 Debug
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination 'platform=macOS' build

# 运行全部测试
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test

# 只跑架构护栏测试
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -only-testing:MomentoTests/ArchitectureGuardTests test

# 检查 whitespace / conflict marker
git diff --check

# 准备并发布新版本
scripts/prepare-release.sh <marketing-version> <build-number>
```

## 相关文档

- [FEATURE.md](FEATURE.md)：产品目标、架构目标和长期功能范围。
- [docs/full-project-review-report.md](docs/full-project-review-report.md)：当前实现质量 review。
- [docs/update-release-flow.md](docs/update-release-flow.md)：Sparkle + GitHub Release 发布流程。
- [AGENTS.md](AGENTS.md)：当前仓库的工程约束。
