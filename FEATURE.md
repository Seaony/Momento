# Momento PRD

## Product Positioning

Momento 是一个 macOS 原生素材管理应用。

目标：

- UI 风格参考 Craft
- 功能体验 1:1 对标 Eagle
- 强调：
  - 极致 macOS 原生体验
  - 精致动画
  - 液态玻璃 / 毛玻璃视觉
  - 高性能图片管理
  - 快速拖拽整理素材

不是 Web App 壳。
不是 Electron。
必须是纯原生 macOS App。

---

# Core Stack

## Language

- Swift 6+

## UI

- SwiftUI 为主
- AppKit 补充复杂能力

## Architecture

- MVVM
- Feature Modularization
- Observable + async/await

## Storage

- Core Data + SQLite store 作为素材 metadata 的唯一权威来源
- 文件以内容寻址方式本地存储
- 缩略图 / 预览图是可重建缓存，不作为源数据
- security-scoped bookmarks 仅用于外部来源或引用模式，不用于库内文件

## Rendering

- NSCollectionView 用于大规模素材渲染
- SwiftUI 包装 AppKit View

## Minimum macOS

- macOS 14+

---

# Design Rules

严格参考：

- Craft macOS App
- Raycast
- Notion Mac Client
- Arc Browser

视觉要求：

- 大量使用：
  - Liquid Glass
  - NSVisualEffectView
  - Gaussian Blur
  - Vibrancy
  - Smooth Shadow
  - Layer Depth

动画要求：

- 所有 hover 必须有反馈
- 所有 panel transition 必须有 easing
- 所有 sidebar interaction 必须丝滑
- 不允许生硬闪烁
- 不允许默认 SwiftUI 丑陋动画

整体气质：

- 安静
- 高级
- 克制
- 轻盈
- 专业

禁止：

- Material 风格
- Web 风格 UI
- Element Plus 风格
- 传统后台系统风格

---

# Primary Features

## 1. Library Management

支持：

- 创建素材库
- 切换素材库
- 导入素材库
- 导出素材库

素材库存储结构：

```text
<LibraryName>.momentolibrary/
├── manifest.json
├── database/
│   └── library.sqlite
├── assets/
│   └── ab/
│       └── <sha256>.<ext>
├── thumbnails/
│   ├── small/
│   ├── medium/
│   └── large/
├── previews/
└── metadata/
    └── import-sessions/
```

结构规则：

- `database/library.sqlite` 是 metadata 的唯一权威来源。
- `manifest.json` 只保存库级别信息：schemaVersion、libraryID、displayName、createdAt、updatedAt。
- `assets/` 只保存原始文件副本，按 SHA-256 content hash 分桶，避免单目录过大。
- `thumbnails/` 和 `previews/` 是缓存，可删除并从 `assets/` 重建。
- `metadata/` 不保存每个素材的权威字段，避免和数据库形成双写冲突。
- 导出素材库时保留同样目录结构；导入素材库时以 `manifest.json` 和数据库 schemaVersion 做兼容检查。

## 1.1 Core Data Model

第一阶段必须明确以下实体和关系。

### Library

- `id: UUID`
- `displayName: String`
- `createdAt: Date`
- `updatedAt: Date`
- `schemaVersion: Int`

### Asset

逻辑素材条目。一个素材对应一个内容文件；重复导入相同文件时复用已有 asset，不重复复制物理文件。

- `id: UUID`
- `libraryID: UUID`
- `contentHash: String`
- `displayName: String`
- `originalFileName: String`
- `fileExtension: String`
- `utiIdentifier: String`
- `kind: image | gif | svg | video | pdf`
- `relativeAssetPath: String`
- `byteSize: Int64`
- `pixelWidth: Int?`
- `pixelHeight: Int?`
- `duration: Double?`
- `orientation: Int?`
- `colorProfileName: String?`
- `exifJSON: Data?`
- `note: String?`
- `isFavorite: Bool`
- `isTrashed: Bool`
- `importedAt: Date`
- `updatedAt: Date`
- `trashedAt: Date?`

约束：

- `(libraryID, contentHash)` 唯一。
- `contentHash` 必须来自文件内容 SHA-256。
- UI 展示路径使用 `relativeAssetPath` 解析到当前素材库根目录，不在数据库中保存库内绝对路径。
- 不支持 EXIF 的格式允许 `exifJSON = nil`，不要用空字典伪装成已解析。

### Folder

用户手动管理的层级分类。

- `id: UUID`
- `libraryID: UUID`
- `parentID: UUID?`
- `name: String`
- `sortOrder: Int`
- `createdAt: Date`
- `updatedAt: Date`

约束：

- 同一 `parentID` 下 `name` 唯一。
- Folder 不是文件系统目录；它只是一组数据库关系。

### FolderMembership

素材与 Folder 的多对多关系，用于支持一个素材出现在多个分类中。

- `assetID: UUID`
- `folderID: UUID`
- `sortOrder: Int`
- `addedAt: Date`

约束：

- `(assetID, folderID)` 唯一。
- Trash 不删除 membership；恢复时可以回到原分类。

### Tag

- `id: UUID`
- `libraryID: UUID`
- `name: String`
- `normalizedName: String`
- `colorHex: String?`
- `createdAt: Date`
- `updatedAt: Date`

约束：

- `(libraryID, normalizedName)` 唯一。
- `normalizedName` 用于去重和自动补全，UI 展示保留 `name`。

### AssetTag

- `assetID: UUID`
- `tagID: UUID`
- `addedAt: Date`

约束：

- `(assetID, tagID)` 唯一。

### AssetColor

用于颜色展示和颜色搜索。

- `assetID: UUID`
- `hex: String`
- `red: Int`
- `green: Int`
- `blue: Int`
- `population: Double`
- `sortOrder: Int`

约束：

- 颜色从缩略图或预览图提取。
- `sortOrder = 0` 表示主色。

### ThumbnailRecord

记录缩略图缓存状态。实际图片文件仍保存在 `thumbnails/`。

- `assetID: UUID`
- `sizeClass: small | medium | large`
- `relativePath: String`
- `pixelWidth: Int`
- `pixelHeight: Int`
- `generatorVersion: Int`
- `status: pending | ready | failed`
- `updatedAt: Date`

约束：

- `(assetID, sizeClass, generatorVersion)` 唯一。
- 缩略图可删除重建，不能承载用户编辑信息。

### SearchIndex

搜索索引是派生数据，可以从 Asset、Tag、Folder、AssetColor 重建。

- `assetID: UUID`
- `nameText: String`
- `tagText: String`
- `folderText: String`
- `colorText: String`
- `updatedAt: Date`

要求：

- 第一阶段支持文件名、扩展名、标签名、Folder 名搜索。
- 模糊搜索和颜色相似度搜索可以基于该结构后续扩展，不在 Asset 表里塞临时字段。

### FileBookmark

仅用于外部来源权限，不用于库内副本。

- `id: UUID`
- `libraryID: UUID`
- `assetID: UUID?`
- `scope: originalFile | sourceFolder | referencedFile`
- `bookmarkData: Data`
- `createdAt: Date`
- `lastResolvedAt: Date?`

要求：

- 默认导入模式是复制到 `assets/`，因此导入完成后浏览素材不依赖外部 bookmark。
- 只有需要 Reveal Original、引用模式或 watched folder 时才保存 bookmark。

---

# 2. Asset Import

支持导入：

- 图片
- GIF
- SVG
- 视频
- PDF
- 文件夹

导入方式：

- Drag & Drop
- Finder 右键
- 粘贴
- 文件选择器

要求：

- 自动生成缩略图
- 自动读取 EXIF
- 自动计算颜色
- 自动 Hash 去重：同一素材库内 `(libraryID, contentHash)` 唯一，重复导入返回已有 asset
- 导入文件夹时，原始文件夹层级映射为 Folder / FolderMembership，不直接映射为库内物理目录

---

# 3. Masonry Grid

核心模块。

要求：

- 超高性能
- 支持 10 万级素材
- 虚拟滚动
- 惰性加载
- 缩略图缓存
- 平滑缩放

支持：

- Grid
- Masonry
- List

禁止：

- SwiftUI LazyVGrid 直接硬写大数据
- 出现滚动掉帧

必须：

- 使用 NSCollectionView

---

# 4. Sidebar

参考 Craft。

支持：

- Library
- Folder
- Tags
- Favorites
- Trash

要求：

- 可折叠
- 支持拖拽排序
- 支持 hover 动效
- 支持 contextual menu

---

# 5. Detail Inspector

右侧详情面板。

展示：

- 预览
- 尺寸
- EXIF
- 标签
- 颜色
- 文件路径
- 文件大小
- 添加时间

支持：

- 编辑标签
- 编辑备注
- 修改分类

数据来源：

- 备注写入 `Asset.note`。
- 分类通过 `FolderMembership` 修改。
- 标签通过 `AssetTag` 修改。
- EXIF 读取自 `Asset.exifJSON`。
- 颜色读取自 `AssetColor`。

---

# 6. Search

支持：

- 全局搜索
- 标签搜索
- 颜色搜索
- 文件名搜索
- 模糊搜索

要求：

- 毫秒级响应
- Spotlight 风格 UI

---

# 7. Tag System

支持：

- 多标签
- 自动补全
- 标签颜色
- 标签拖拽

---

# 8. Quick Preview

支持：

- Space 快速预览
- 全屏查看
- 左右切换

要求：

- 使用 QuickLook

---

# 9. Drag System

这是核心能力。

支持：

- App 内拖拽
- 拖到 Finder
- 从 Finder 拖入
- 多选拖拽
- 拖拽动画

必须：

- 完全原生
- 不允许 Web 风格拖拽体验

---

# 10. Command Palette

参考：

- Raycast
- Spotlight

快捷键：

```text
⌘K
```

支持：

- 搜索素材
- 搜索标签
- 执行动作

---

# Technical Requirements

## Performance

必须满足：

- 启动时间 < 1.5s
- 10万素材滚动不卡顿
- 内存占用可控
- 缩略图异步生成
- 不阻塞主线程

---

# File Handling

必须：

- 支持 sandbox
- 支持 security-scoped bookmarks
- 支持 Finder Sync

---

# Thumbnail System

要求：

- 多尺寸缓存：small / medium / large
- 后台生成，并写入 `ThumbnailRecord.status`
- 内存 LRU Cache 只缓存当前滚动窗口附近图片
- 磁盘缓存位于 `thumbnails/`，可删除重建
- 视频 / GIF 使用第一帧或指定 poster frame
- PDF 使用第一页预览
- SVG 优先用矢量渲染生成位图缩略图

---

# Database

要求：

- Core Data SQLite store 是 metadata 的唯一权威来源
- 写入、导入、缩略图状态更新使用 background context
- UI 列表查询必须分页或 batch fetch
- Asset、Tag、Folder、FolderMembership、SearchIndex 建必要索引
- 所有派生数据必须可重建：ThumbnailRecord、SearchIndex、AssetColor

禁止：

- 在主线程执行大批量查询、导入写入或缩略图状态批量更新
- 在数据库中保存库内文件绝对路径
- 在 `metadata/` 中重复保存 Asset / Tag / Folder 的权威字段

---

# Architecture Rules

## Strict Rules

禁止：

- Massive View
- God Object
- 单文件超 1000 行
- 过度抽象
- 无意义 Protocol
- 为未来需求过度设计

优先：

- 小而稳定
- 可维护
- 清晰边界

---

# SwiftUI Rules

SwiftUI 仅负责：

- 页面结构
- 状态绑定
- 动画
- UI 组合

复杂高性能区域：

必须 AppKit。

---

# AppKit Rules

以下模块必须优先考虑 AppKit：

- Grid Rendering
- Drag System
- Finder Integration
- Context Menu
- Keyboard System
- Collection Rendering

---

# Animation Rules

禁止默认 SwiftUI 生硬动画。

要求：

- Spring 动画
- Hover interpolation
- Smooth opacity transition
- Smooth scale transition

参考：

- Craft
- Arc
- Linear

---

# Development Rules

## Git

禁止自动 commit。

所有 commit：
必须先询问用户。

---

# Dev Server

禁止自动启动：

- npm run dev
- vite
- next dev
- 本地 server

必须由用户手动启动。

---

# Coding Rules

禁止：

- 过度防御性代码
- 为极低概率错误写大量逻辑
- 无意义 Optional chaining
- 无意义 abstraction

优先：

- 简洁
- 可读
- 原生
- 稳定

---

# UI Rules

禁止：

- 默认 SwiftUI Button 样式
- 默认 List 样式
- iPad 风格 UI
- UIKit 风格 UI

必须：

- 完全 macOS Desktop 气质

---

# First Milestone

第一阶段必须完成：

1. Library System
2. Asset Import
3. Masonry Grid
4. Sidebar
5. Detail Inspector
6. Search
7. Tag System
8. Quick Preview

完成标准：

- 可以替代 Eagle 基础使用
- UI 达到 Craft 级完成度
- 动画达到 macOS 原生高级应用水准

---

# Folder Structure

```text
Momento/
├── App/
├── Core/
├── Features/
├── Shared/
├── DesignSystem/
├── Services/
├── Storage/
├── AppKitBridge/
└── Resources/
```

---

# Final Goal

最终目标不是“能用”。

而是：

做出一个：

- 真正 macOS Native
- 有 Craft 级 UI 质感
- 有 Eagle 级素材管理能力
- 有 Raycast 级交互细节
- 能长期使用的专业工具
