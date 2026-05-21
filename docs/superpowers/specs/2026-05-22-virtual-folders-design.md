# Virtual Folders And Import Metadata Design

## 背景

Momento 目前已经有资源库包、资源导入、最近资源库列表、侧边栏导航和资源网格，但“文件夹”和“图片导入衍生元数据”还没有成为真实数据能力：

- `Momento/Core/AssetModels.swift` 只有 `AssetItem`、`TagItem` 和 `SidebarSelection.folderManagement`，没有文件夹模型。
- `Momento/Storage/MomentoModel.xcdatamodeld` 目前只有 `AssetRecord` 一个 Core Data 实体。
- `Momento/Storage/LibraryMetadataStore.swift` 只负责读取和保存资源，没有保存文件夹、资源-文件夹关系或图片主色板。
- `Momento/Core/LibraryStore.swift` 的 `.uncategorized` 当前仍返回全部资源，这只是临时行为，不符合真实文件夹语义。
- `Momento/Services/AssetImportService.swift` 已经会读取图片尺寸，但还不会像 Eagle 一样在导入时分析图片颜色构成和占比，也不会为大图生成列表页可用的缩略图。

这次要实现的是虚拟文件夹、导入图片时的主色板元数据，以及导入图片时的缩略图生成。虚拟文件夹不移动 `.momento/assets` 里的物理文件；颜色元数据不能混进 tags；列表页也不能直接加载大图原图。

## 外部参考

- Eagle Plugin API 的 item 数据上有 `folders: string[]`，并提供 `isUnfiled` 查询条件。这说明一项资源可以属于多个文件夹，未分类是由文件夹关联为空推导出来的。
- Eagle Plugin API 的 folder 数据有 `id`、`name`、`parent`、`children`，说明文件夹本身是独立元数据，和磁盘目录不是一回事。
- SwiftUI 官方文档中 `fileImporter` 用于系统文件选择，并要求处理 security-scoped resource；Momento 当前导入服务已经遵循这个方向。
- Core Data 官方文档要求通过自动迁移选项处理版本化 store 的 lightweight migration。本次不能直接原地改唯一模型文件后假设旧库能打开。
- Apple Accelerate 示例使用 k-means 提取图片 dominant colors。本次不引入第三方库，优先用系统框架完成导入时主色分析。
- Image I/O 的 `CGImageSourceCreateThumbnailAtIndex` 支持按最大像素尺寸创建缩略图。本次缩略图生成优先使用 Image I/O，而不是手写全图缩放。

参考链接：

- [Eagle item API](https://developer.eagle.cool/plugin-api/api/item)
- [Eagle folder API](https://developer.eagle.cool/plugin-api/api/folder)
- [SwiftUI fileImporter](https://developer.apple.com/documentation/swiftui/view/fileimporter%28ispresented%3Aallowedcontenttypes%3Aallowsmultipleselection%3Aoncompletion%3Aoncancellation%3A%29)
- [Core Data automatic migration](https://developer.apple.com/documentation/coredata/migrating-your-data-model-automatically)
- [Apple Accelerate dominant colors sample](https://developer.apple.com/documentation/accelerate/vimage/calculating_the_dominant_colors_in_an_image)
- [CGImageSourceCreateThumbnailAtIndex](https://developer.apple.com/documentation/imageio/cgimagesourcecreatethumbnailatindex%28_%3A_%3A_%3A%29)

## 目标

1. 资源库内支持创建和删除文件夹，文件夹允许无限层级嵌套。
2. 文件夹是虚拟分类，一张资源可以关联多个文件夹。
3. 点击侧边栏文件夹时，只显示关联到该文件夹的资源。
4. `未分类` 是筛选状态，不是真实文件夹；它只显示没有任何文件夹关联的资源。
5. 无论当前正在查看哪个侧边栏项，外部文件导入后默认都没有文件夹关联，因此会被 `未分类` 筛选出来。
6. 资源关联到文件夹的具体 UI 交互本轮不设计，后续单独处理；数据层仍保留关联和移除关联能力。
7. 如果导入的是库内已有资源，保持现有去重语义，不复制物理文件，也不自动改变它的文件夹关联。
8. 删除文件夹只删除文件夹元数据和关联关系，不删除资源文件，也不删除资源记录。
9. 导入可解码的图片时，分析主色板，保存颜色 hex 和占比，用于 Inspector 展示类似 Eagle 的颜色条。
10. 导入可解码的图片时，生成一档缩略图，列表页和检查器优先使用缩略图，不直接加载原图。
11. 已有资源库可以通过明确的 Core Data 轻量迁移继续打开。
12. 资源库 manifest schema 和 Core Data model migration 必须协同处理，不能让旧库在 manifest 检查阶段提前失败。
13. `thumbnails/` 和 `previews/` 仍然是可删除缓存；清除缓存后必须有可控的缩略图重建路径。
14. 用户可以把已关联到文件夹的资源从该文件夹移除，避免错误归类只能靠删除整个文件夹修正。
15. 搜索需要覆盖资源名称、扩展名、标签、文件夹名称和图片主色 hex。

## 非目标

- 不实现 Finder 真实目录同步。
- 不移动 `.momento/assets` 下的物理文件。
- 不实现文件夹拖拽排序、拖拽改父级、重命名、颜色、图标选择。
- 不实现智能文件夹、筛选规则、批量移动、右键完整菜单。
- 不实现跨资源库复制文件夹结构。
- 不实现外部 Finder 文件拖到文件夹后自动归类。外部导入始终保持无文件夹关联。
- 不设计资源网格资源拖到文件夹的关联交互；这块后续单独做。
- 不对旧资源做后台颜色回填。
- 不在普通打开旧库时自动全量回填缩略图；清除缓存触发的缩略图重建属于显式修复路径。
- 不把 SVG、PDF、视频纳入本轮导入、颜色分析或缩略图生成范围。
- 不新增持久化 SearchIndex 实体，不实现模糊搜索、语义搜索或颜色相似度搜索。
- 不实现缩略图生成状态表。第一版用确定性路径和文件存在性判断缩略图是否可用。

这些能力可以在虚拟文件夹和颜色元数据基础上继续扩展，但不放进这次实现。

## 核心决策

### 文件夹使用虚拟模型

资源文件继续由内容 hash 管理，路径仍是：

```text
.momento/assets/<hash-prefix>/<sha256>.<ext>
```

文件夹只存在于数据库中，资源和文件夹通过关联表连接。

这样做的原因：

- 同一资源可以同时属于多个文件夹，不需要复制物理文件。
- 删除文件夹不会误删资源。
- 资源去重仍然按 content hash 工作。
- 后续导出时可以按虚拟文件夹结构生成真实目录，但那是导出层逻辑，不污染库内存储。

### 导入后保持无文件夹关联

导入只负责把外部文件放进当前资源库，不负责分类：

- 新导入资源的 `folderIDs` 必须是空数组。
- 导入重复文件时保持 no-op，不复制物理文件，也不修改已有资源的 `folderIDs`。
- `未分类` 不对应任何 `FolderRecord`，也不创建 membership。
- `未分类` 只是 `folderIDs.isEmpty` 的筛选结果。

这个规则比“按当前文件夹自动归类”更稳定：导入行为不会因为当前侧边栏选区变化而产生隐式副作用，`未分类` 只是无文件夹关联资源的状态视图。

### 资源关联交互后置

资源和文件夹的 membership 是真实数据能力，但本轮不规定具体交互。

本轮只要求：

- 存储层和 Store 提供 `assignAssets(ids:to:)` 与 `unassignAssets(ids:from:)`。
- 文件夹视图能按已有 membership 过滤资源。
- 导入不会自动创建 membership。

具体的“如何把资源放进文件夹”交互后续单独设计，避免现在把拖拽、右键菜单、批量操作一起塞进当前实现。

### 文件夹支持无限层级

`AssetFolder.parentID` 是真实结构字段，本轮 UI 需要支持创建子文件夹和渲染任意深度树。

设计规则：

- `parentID == nil` 表示根级文件夹。
- 子文件夹通过父文件夹行上的新增入口创建。
- 文件夹树没有人为层级上限。
- 渲染时按 `parentID` 建树，再按 `sortIndex` 和创建时间稳定排序。
- 删除文件夹时递归删除所有后代文件夹和这些文件夹的 membership。
- 不能允许循环父子关系；创建新文件夹天然不会产生循环，后续如果支持移动文件夹，必须额外验证。

### 删除文件夹采用“删除分类，不删资源”

删除文件夹时：

- 删除该文件夹记录。
- 删除该文件夹以及所有后代文件夹的资源关联记录。
- 资源记录和物理文件保留。
- 如果某个资源因此不再属于任何文件夹，它会自然出现在 `未分类`。

如果当前正在查看被删除的文件夹，删除完成后切回 `全部`。

### 资源可以从文件夹移除

文件夹关联必须是可逆操作。第一版需要支持把库内资源从某个文件夹移除：

- 移除 folder membership，不删除资源记录。
- 不删除 `.momento/assets` 中的物理文件。
- 如果资源不再属于任何文件夹，它会显示在 `未分类`。
- 对不存在的 asset id 保持和关联操作一致：忽略不存在项，不创建或删除脏数据。
- 如果目标 folder 不存在，抛 `missingFolder`。

这个能力可以先通过当前文件夹视图里的资源操作入口暴露，不要求本次做完整右键菜单体系；但数据层和 Store API 必须存在，避免错误归类没有恢复路径。

### 文件夹名称规则

第一版采用最少规则：

- 创建文件夹时先 trim 首尾空白和换行。
- trim 后为空则报错。
- 允许同一个父级下出现相同名称的文件夹。
- 不做大小写不敏感重名校验。

重名是否造成识别困难，后续可以通过图标、颜色、路径提示或重命名能力解决，不在本轮通过数据约束禁止。

### 图片色彩分析在导入时完成

导入图片时提取主色板：

- 本轮只处理图片资源，例如 PNG、JPEG、HEIC、WebP，以及当前导入管线已支持的 GIF 首帧。
- SVG、PDF、视频不纳入本轮。
- 色彩分析失败不阻止资源导入，但必须得到空 palette，而不是伪造颜色。
- 颜色按占比从高到低排序，最多保留 8 个颜色。
- 占比保存为 `0...1` 的 `Double`，UI 展示时格式化成百分比。

色彩分析不能写进 `TagItem.colorHex`。tags 是用户语义标签，palette 是导入元数据，两者必须分开。

### 图片缩略图在导入时生成

导入图片时同时生成一档缩略图：

- 本轮只处理图片资源，例如 PNG、JPEG、HEIC、WebP，以及当前导入管线已支持的 GIF 首帧。
- SVG、PDF、视频不纳入本轮。
- 缩略图是派生缓存，不是资源原文件；原文件仍保存在 `.momento/assets`。
- 列表页、瀑布流、Inspector 预览优先使用缩略图。
- 如果缩略图缺失，UI 显示占位或通用图标，不在列表页直接回退加载原图。

缩略图文件放在现有库包目录中：

```text
.momento/thumbnails/<contentHash>.png
```

第一版统一输出 PNG，最长边建议 512 px，保留透明图的 alpha。虽然照片类 PNG 可能比 JPEG 大，但缩略图尺寸受控，且不需要额外持久化“这个缩略图到底是 jpg 还是 png”的路径状态。后续如果要优化磁盘占用，可以再引入格式选择和 thumbnail metadata。

### 缩略图缓存必须可重建

`thumbnails/` 和 `previews/` 是派生缓存，不是权威数据。清除缓存后不能让资源永久停留在无缩略图状态。

第一版采用显式修复路径：

- `clearCachesAndReloadCurrentLibrary()` 删除缓存后，触发一次当前库缩略图重建。
- 重建只处理本轮支持的图片资源。
- 重建过程中 UI 可以先显示占位或通用图标。
- 重建失败不影响资源库打开，但失败资源继续保持无缩略图，不回退到列表页加载原图。
- 重建逻辑复用 `AssetThumbnailService`，不能复制一份独立缩放实现。

暂时不新增 `ThumbnailRecord` 或生成队列表。第一版通过确定性路径判断单张 PNG 缩略图是否存在，足够支撑清除缓存后的修复。

### Manifest schema 和数据库迁移必须协同

资源库包打开时不能只看 `LibraryManifest.currentSchemaVersion` 的等值判断。否则 v1 manifest 会在 Core Data 轻量迁移之前被拒绝，导致“迁移测试通过但真实旧资源库打不开”。

第一版规则：

- `LibraryManifest.currentSchemaVersion` 可以升级到 `2`，表示包内数据库模型新增了文件夹、颜色和缩略图派生能力。
- `LibraryStorage.openLibraryPackage` 接受当前 app 支持的 schema version，例如 `1...2`。
- 只有未来版本或未知版本才抛 `unsupportedSchemaVersion`。
- 对 v1 manifest，先允许进入 Core Data 打开流程，由 model version 执行 lightweight migration。
- 迁移和首次成功打开后，可以把 manifest 写回当前 schema version；写回必须发生在数据库能成功打开之后。

这个规则把“包格式兼容性”和“数据库模型迁移”分开处理，避免 manifest gate 抢在数据库迁移前失败。

### 第一版搜索使用现有内存数据

本次不新增持久化 `SearchIndex`。搜索先基于 `LibraryStore` 当前内存中的 `AssetItem` 和 `folders` 做确定性匹配：

- 文件名和扩展名。
- 标签名称。
- 资源关联的文件夹名称。
- 图片主色 hex，支持带 `#` 和不带 `#` 的查询。

推荐先不做持久化 SearchIndex，原因是当前架构已经会把资源、标签、文件夹和颜色元数据读入 `LibraryStore`，内存过滤可以直接复用这份权威状态。持久化索引会复制一份派生数据，导入资源、删除文件夹、重命名文件夹、修改标签、重建颜色时都要同步更新，第一版更容易引入一致性 bug。

后续出现以下需求时，再单独设计持久化索引或 SQLite FTS：

- 资源量明显变大，内存过滤出现可感知卡顿。
- 需要跨库搜索。
- 需要全文、模糊、拼音、分词或复杂排序。
- 需要颜色相似度，而不只是 hex 文本命中。

## 数据模型

### Swift 值类型

在 `Momento/Core/AssetModels.swift` 增加：

```swift
nonisolated struct AssetFolder: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var libraryID: String
    var name: String
    var parentID: String?
    var sortIndex: Int
    var createdAt: Date
    var updatedAt: Date
}

nonisolated struct AssetColor: Identifiable, Hashable, Codable, Sendable {
    var id: String
    var libraryID: String
    var assetID: String
    var hex: String
    var coverage: Double
    var sortIndex: Int
}
```

修改 `AssetItem`：

```swift
var folderIDs: [String]
var paletteColors: [AssetColor]
var thumbnailURL: URL?
```

修改 `SidebarSelection`：

```swift
case folder(String)
```

保留 `folderManagement` 的现有 case，以免命令面板或旧 UI 引用立刻断掉；但侧边栏这次会优先展示真实文件夹列表。

### Core Data 实体

必须先创建新的 Core Data model version，再修改新版本。不要只原地编辑当前唯一的 `MomentoModel.xcdatamodel`。

预期结构：

```text
MomentoModel.xcdatamodeld/
  MomentoModel.xcdatamodel/       # v1，保留现状
  MomentoModel v2.xcdatamodel/    # v2，新增文件夹和颜色实体，设为 current
  .xccurrentversion
```

`FolderRecord`：

- `id: String`
- `libraryID: String`
- `name: String`
- `parentID: String?`
- `sortIndex: Integer 64`
- `createdAt: Date`
- `updatedAt: Date`

唯一约束：

- `libraryID + id`

`AssetFolderMembershipRecord`：

- `id: String`
- `libraryID: String`
- `assetID: String`
- `folderID: String`
- `createdAt: Date`

唯一约束：

- `libraryID + assetID + folderID`

`AssetColorRecord`：

- `id: String`
- `libraryID: String`
- `assetID: String`
- `hex: String`
- `coverage: Double`
- `sortIndex: Integer 64`

唯一约束：

- `libraryID + assetID + sortIndex`

暂时不加 Core Data relationship，保持和现有 `LibraryMetadataStore` 一样的 key-value `NSManagedObject` 风格。这样实现简单，也避免一口气把现有存储层改成 managed object graph。

### 迁移

`MomentoCoreDataStack` 需要明确打开轻量迁移：

```swift
storeDescription.shouldMigrateStoreAutomatically = true
storeDescription.shouldInferMappingModelAutomatically = true
```

这次只新增实体，不修改 `AssetRecord` 现有字段语义，适合 lightweight migration。

迁移验证不能只用当前模型新建数据库。必须覆盖真实旧库路径：

1. 用 v1 model 创建一个只含 `AssetRecord` 的测试库，或提交一个最小 v1 sqlite fixture。
2. 用 v2 app/model 打开它。
3. 验证旧资源仍能读取，且 `folderIDs == []`、`paletteColors == []`、`thumbnailURL == nil`。

## 存储层设计

`Momento/Storage/LibraryStorage.swift` 增加 deterministic thumbnail URL helper：

```swift
func thumbnailURL(
    forContentHash contentHash: String,
    in library: AssetLibrary
) -> URL
```

`prepareLibraryDirectories(for:)` 已经创建 `thumbnails` 目录，这里只补路径生成能力。

`Momento/Storage/LibraryMetadataStore.swift` 增加以下能力：

```swift
func loadFolders() throws -> [AssetFolder]
func createFolder(name: String, parentID: AssetFolder.ID?) throws -> AssetFolder
func deleteFolder(id: AssetFolder.ID) throws -> [AssetFolder.ID]
func saveImportedAssets(_ assets: [AssetItem]) throws -> [AssetItem]
func assignAssets(ids: Set<AssetItem.ID>, to folderID: AssetFolder.ID) throws -> [AssetItem]
func unassignAssets(ids: Set<AssetItem.ID>, from folderID: AssetFolder.ID) throws -> [AssetItem]
```

建议新增错误：

```swift
enum LibraryMetadataError: LocalizedError {
    case invalidFolderName
    case missingFolder
}
```

读取资源时：

1. 查询当前库的所有 `AssetFolderMembershipRecord`。
2. 按 `assetID` 组成 `[String: [String]]`。
3. 查询当前库的所有 `AssetColorRecord`。
4. 按 `assetID` 组成 `[String: [AssetColor]]`。
5. 构造 `AssetItem` 时填入 `folderIDs` 和 `paletteColors`。
6. `folderIDs` 排序稳定，优先按 folder `sortIndex`，再按 folder `createdAt`，最后按 `id`。
7. `paletteColors` 按 `sortIndex` 排序，实际含义是 coverage 从高到低的顺序。
8. `thumbnailURL` 不需要写入 Core Data；按 `contentHash` 的确定性路径检查 PNG 是否存在，存在则填入 URL，否则为 `nil`。

保存导入资源时：

1. 先按 `contentHash` 查已有 `AssetRecord`。
2. 已存在则复用已有资源值，不新增文件夹关联，不覆盖已有 palette。
3. 不存在则创建新的 `AssetRecord`。
4. 为新资源写入 import service 生成的 `AssetColorRecord`。
5. 新资源的 `folderIDs` 固定为空。
6. 新资源的 `thumbnailURL` 来自 import service 生成的确定性缩略图路径。
7. 保存后返回带 `folderIDs`、`paletteColors` 和 `thumbnailURL` 的 `AssetItem`。

关联已有资源时：

1. `assignAssets(ids:to:)` 只接受当前库中真实存在的 asset id。
2. 如果 folder 不存在，抛 `missingFolder`。
3. 如果部分 asset id 不存在，忽略不存在的 id，不创建脏 membership。
4. membership 已存在时保持幂等。
5. 返回被更新的资源，供 `LibraryStore` 替换内存状态。

移除资源关联时：

1. `unassignAssets(ids:from:)` 只删除当前库中真实存在的 membership。
2. 如果 folder 不存在，抛 `missingFolder`。
3. 如果部分 asset id 不存在，忽略不存在的 id。
4. membership 不存在时保持幂等。
5. 返回被更新的资源，供 `LibraryStore` 替换内存状态。

删除文件夹时：

1. 验证目标文件夹存在。
2. 找到目标文件夹以及所有后代文件夹 ID。
3. 删除这些 `FolderRecord`。
4. 删除这些 folderID 对应的 `AssetFolderMembershipRecord`。
5. 返回实际删除的 folderID 列表，供 `LibraryStore` 更新内存状态。

### 缩略图缓存修复

`LibraryMetadataStore` 不负责生成缩略图，但需要提供当前库资源列表和 content hash，让上层服务可以重建缺失缓存。

建议新增一个小型协调服务，而不是把修复逻辑塞进 UI：

```swift
struct AssetDerivativeRepairService: Sendable {
    nonisolated func regenerateMissingThumbnails(
        for assets: [AssetItem],
        in library: AssetLibrary
    ) async
}
```

规则：

- 只检查 `AssetItem.thumbnailURL == nil` 或对应文件不存在的资源。
- 只对本轮支持的图片资源调用 `AssetThumbnailService`。
- 生成完成后重新读取资源或返回更新后的 `AssetItem`，让 Store 刷新 `thumbnailURL`。
- 单个文件失败不终止整个修复过程。

## 导入、缩略图和色彩分析设计

### AssetImportService

`AssetImportService` 继续负责：

- security-scoped source access 生命周期。
- 文件夹递归收集。
- 支持格式过滤，本轮只允许图片类型进入导入流程。
- SHA-256 hash。
- 复制物理文件到 `.momento/assets`。
- 图片尺寸读取。
- 缩略图生成。
- 主色板分析。

保留现有库内去重：

```swift
excludingContentHashes: metadataStore.existingContentHashes()
```

如果 hash 已存在，这个文件不返回新 `AssetItem`，也不修改已有资源的 folder、palette 或 thumbnail。

### AssetThumbnailService

新增独立服务生成缩略图：

```swift
struct AssetThumbnailService: Sendable {
    nonisolated func generateThumbnail(
        for sourceURL: URL,
        contentHash: String,
        in library: AssetLibrary
    ) throws -> URL?
}
```

实现原则：

- 使用 `CGImageSourceCreateThumbnailAtIndex`。
- 设置 `kCGImageSourceCreateThumbnailFromImageAlways = true`。
- 设置 `kCGImageSourceCreateThumbnailWithTransform = true`，保留 EXIF orientation。
- 设置 `kCGImageSourceThumbnailMaxPixelSize` 控制尺寸。
- 输出 PNG 到 `LibraryStorage` 提供的 deterministic thumbnail URL。
- 使用 atomic write，避免导入中断时留下半文件。

建议尺寸：最长边 512 px。

使用方式：

- 资源网格、列表和 Inspector 预览都优先用这张缩略图。
- QuickLook 和双击预览仍打开原始资源文件。

如果缩略图生成失败，导入继续成功并返回 `thumbnailURL == nil`。UI 不能因此去列表里加载原图，只能显示占位或文件类型图标。

### AssetColorAnalysisService

新增一个独立服务，避免把颜色算法塞进 `AssetImportService`：

```swift
struct AssetColorAnalysisService: Sendable {
    nonisolated func paletteColors(
        for url: URL,
        libraryID: String,
        assetID: String,
        maxColorCount: Int
    ) -> [AssetColor]
}
```

实现原则：

- 使用 `CGImageSource` 解码图片，或复用缩略图生成阶段得到的小尺寸图像输入。
- 使用最长边 96 或 128 像素的小图做采样，避免对大图全量采样。
- 转换到稳定的 sRGB/RGBA8 像素格式。
- 忽略透明像素，例如 alpha 低于 5% 的像素。
- 用确定性的颜色量化直方图生成最多 8 个颜色。
- 对相近颜色做一次合并，避免同一主色被拆成多个非常接近的 swatch。
- `coverage = clusterPixelCount / consideredPixelCount`。
- 输出 hex 使用大写 `#RRGGBB`。

如果图片无法解码、没有有效像素或算法失败，返回空数组。导入仍然成功，因为资源持久化是主流程，颜色只是可缺省的导入元数据。

## Store 设计

`Momento/Core/LibraryStore.swift` 增加：

```swift
var folders: [AssetFolder]
```

打开资源库时：

```swift
folders = try metadataStore.loadFolders()
assets = try metadataStore.loadAssets()
```

关闭资源库时必须清空：

```swift
folders = []
```

新增方法：

```swift
func createFolder(named name: String, parentID: AssetFolder.ID?) throws
func deleteFolder(id: AssetFolder.ID) throws
func assignAssets(ids: Set<AssetItem.ID>, to folderID: AssetFolder.ID) throws
func unassignAssets(ids: Set<AssetItem.ID>, from folderID: AssetFolder.ID) throws
func rebuildMissingThumbnails() async
```

`importItems(from:)` 保持“不接收 folderID”的语义。导入永远创建无文件夹关联资源：

```swift
func importItems(from urls: [URL]) async throws
```

`LibraryStore` 需要一个小 helper 来同步资源数组：

```swift
private func mergeAssets(_ updatedAssets: [AssetItem]) {
    for asset in updatedAssets {
        if let index = assets.firstIndex(where: { $0.id == asset.id }) {
            assets[index] = asset
        } else {
            assets.append(asset)
        }
    }
}
```

这个 helper 是必要的，因为 `assignAssets(ids:to:)` 会返回已有资源的新 `folderIDs`。只 append 新 ID 会导致 UI 不刷新。

`unassignAssets(ids:from:)` 复用同一个 `mergeAssets(_:)`，保证移除文件夹关联后当前视图立即更新。

`clearCachesAndReloadCurrentLibrary()` 的顺序：

1. 删除 `thumbnails/` 和 `previews/`。
2. 重新创建缓存目录。
3. 重新读取资源，此时 `thumbnailURL` 为空或缺失。
4. 调用 `rebuildMissingThumbnails()`。
5. 缩略图生成后重新合并或重载资源。

UI 在第 3 步到第 5 步之间只能显示占位或文件类型图标，不能直接加载原图。

删除文件夹后的 Store 同步规则：

1. 从 `folders` 删除被删 folderID。
2. 从每个 `AssetItem.folderIDs` 删除这些 folderID。
3. 如果 `sidebarSelection` 是被删 folderID 或后代 folderID，切回 `.library(currentLibrary.id)`。
4. 如果当前 `selectedAssetID` 不在新的 `visibleAssets` 里，选中第一项或清空。

`visibleAssets` 规则：

- `.library`: 当前库全部资源。
- `.favorites`: `isFavorite == true`。
- `.uncategorized`: `folderIDs.isEmpty`。
- `.untagged`: `tags.isEmpty`。
- `.folder(id)`: `folderIDs.contains(id)`。
- `.tag(id)`: 保持现有 tag 过滤。
- `.tagManagement`、`.folderManagement`、`.trash`: 暂时为空。

搜索规则叠加在侧边栏过滤之后：

- 空搜索词返回当前侧边栏过滤结果。
- 搜索词 trim 并做大小写不敏感匹配。
- 文件名、扩展名、标签名称、关联文件夹名称命中任一项即可显示。
- `paletteColors.hex` 支持 `#1B1D1D` 和 `1B1D1D` 两种输入。
- 第一版不做模糊匹配和颜色距离计算。

`selectSidebarItem(id:)` 规则：

- `folder-<folderID>` 映射到 `.folder(folderID)`。
- 如果传入的 folderID 不存在，回退到 `.library(currentLibrary.id)`。

## UI 设计

### 侧边栏文件夹区

`Momento/Features/Sidebar/MomentoSidebarView.swift` 增加输入：

```swift
var folders: [AssetFolder]
var onCreateFolder: (AssetFolder.ID?) -> Void
var onDeleteFolder: (AssetFolder.ID) -> Void
```

文件夹标题区保持当前视觉规则：

- 标题“文件夹”常亮。
- hover 后显示 `+` 和展开/收起按钮。
- 标题区 `+` 创建根级文件夹。

文件夹列表：

- 按 `parentID` 渲染任意层级树。
- 子级按层级缩进，缩进不设置人为最大层级。
- 有文件夹时渲染真实文件夹行和展开/收起状态。
- 没有文件夹时保持当前的“暂无文件夹”空状态。
- 文件夹行 hover 使用和侧边栏其他项一致的 `MomentoTheme.sidebarIconHoverBackground`。
- 选中时文字更亮，并使用同样的背景逻辑。
- 每行右侧 hover 时显示新增子文件夹和删除入口。
- 文件夹行不接收 Finder URL。外部文件导入仍走全局 importer/drop，保持无文件夹关联。
- App 内资源关联到文件夹的交互本轮不做。

### 新建文件夹弹窗

新增一个小型输入弹窗，复用当前资源库创建弹窗的视觉语言：

- Liquid Glass 背景。
- 黑色透明模糊遮罩。
- 标题：`新建文件夹`。
- 输入框 placeholder：`文件夹名称`。
- 按钮：`取消`、`创建`。

新建成功后：

- 关闭弹窗。
- 选中新文件夹。
- 如果新文件夹为空，主区域显示空状态。

如果名称为空：

- 不关闭弹窗。
- 在弹窗内显示错误文案。
- 不静默 fallback 成其它名称。

同级重名允许创建，不显示错误。

### 删除文件夹确认

删除前必须确认，避免误操作。

文案方向：

```text
删除“Jobs”文件夹？

这个操作只会删除文件夹和它的资源关联，不会删除资源库中的图片文件。
如果某些资源不再属于任何文件夹，它们会显示在“未分类”里。
```

按钮：

- `取消`
- `删除文件夹`，红色 Liquid Glass 按钮

### Inspector 颜色展示

`MomentoInspectorAsset` 应该从 `asset.paletteColors` 构建颜色条，而不是继续用 `asset.tags.compactMap(\.colorHex)`。

展示规则：

- 显示最多 8 个 swatch。
- swatch 顺序按 `coverage` 从高到低。
- hover 或辅助文本展示 `#RRGGBB (85.0%)`。
- 没有 palette 时不显示颜色条，不用 fake 颜色。

## 导入和关联行为

### 导入

所有导入入口都保持一致：

- 点击主区域空状态的导入按钮。
- 从命令面板执行导入。
- 从 toolbar/search 外的全局 `fileImporter` 选择文件。
- 拖 Finder 文件到主内容区域。

这些入口都调用 `store.importItems(from:)`。导入结果保持无文件夹关联，因此会被 `未分类` 筛选出来，不因为当前选中的文件夹而自动关联。

### 关联到文件夹

本轮只定义数据层能力，不实现具体 UI 交互：

- `store.assignAssets(ids:to:)` 创建资源到文件夹的 membership。
- Store 用返回的资源替换内存中的旧资源。
- 目标文件夹视图按已有 membership 展示资源。
- 拖拽、右键菜单、批量操作等关联交互后续单独设计。

### 从文件夹移除

用户在某个文件夹视图中发现资源归类错误时，需要有一个最小可用的移除路径：

- 只在当前侧边栏选中真实文件夹时显示“从当前文件夹移除”动作。
- 调用 `store.unassignAssets(ids: selectedAssetIDs, from: folderID)`。
- Store 用返回的资源替换内存中的旧资源。
- 如果资源被移除后不再属于任何文件夹，它会出现在 `未分类`。
- 移除动作不删除资源、不移动物理文件、不修改标签。

### 重复导入

如果导入的文件已经存在：

- 不复制物理文件。
- 不新增 `AssetRecord`。
- 不新增 membership。
- 不覆盖 palette。
- 不覆盖 thumbnail。

如果用户想把已有资源放入文件夹，必须通过显式关联动作。

### 搜索

搜索框的第一版行为保持轻量：

- 搜索当前侧边栏范围内的资源，而不是强制全库搜索。
- 搜索命中文件名、扩展名、标签名、文件夹名、主色 hex。
- 主色 hex 查询允许用户输入 `#1B1D1D` 或 `1B1D1D`。
- 搜索不改变资源的文件夹关联。

## 本地化

需要新增或确认以下文案：

- `New Folder`
- `Folder Name`
- `Create`
- `Delete Folder`
- `Folder name is required.`
- `This folder no longer exists.`
- `Delete “%@” folder?`
- `This only removes the folder and its asset associations. Assets remain in the library.`

注意：按当前约定，如果 `Momento/Localizable.xcstrings` 因 Xcode 字符串提取发生变化，直接随任务一起提交，不单独调查或清理。

## 验证计划

这次不是纯 UI 微调，涉及持久化、迁移和导入元数据，所以需要保留少量高价值测试。

建议新增或调整 `MomentoTests/ImportServiceSmokeTests.swift`：

1. 创建文件夹后重开资源库，文件夹仍存在。
2. 创建多层子文件夹后重开资源库，父子关系仍正确。
3. 空文件夹名会失败，同级重名允许创建。
4. 导入图片后，资源的 `folderIDs` 为空，资源会被 `未分类` 筛选出来。
5. 在文件夹视图导入图片后，资源仍然不自动关联当前文件夹。
6. 对已有资源调用 `assignAssets(ids:to:)` 后，内存和重开库后的 `folderIDs` 都更新。
7. 删除父文件夹后，所有后代文件夹和 membership 都删除，资源记录和物理文件保留。
8. 删除当前正在查看的文件夹后，选择切回 `全部`，且 selected asset 不残留到不可见资源。
9. 导入一张已知单色 PNG 后，`paletteColors` 至少包含该颜色，coverage 接近 1。
10. 导入一张已知 PNG 后，单张缩略图文件存在，且最长边不超过 512 px。
11. 重复导入同一张图片不新增资源、不新增 membership、不覆盖 palette 或 thumbnail。
12. 用 v1 store fixture 验证轻量迁移：旧资源库能打开，旧资源默认 `folderIDs == []`、`paletteColors == []`、`thumbnailURL == nil`。
13. v1 manifest + v1 database 能打开并迁移；未来版本 manifest 仍然被拒绝。
14. 清除缓存并重新加载后，缺失缩略图会被重建或进入可恢复的占位状态，列表页不会加载原图。
15. 对已有资源调用 `unassignAssets(ids:from:)` 后，内存和重开库后的 `folderIDs` 都移除对应文件夹。
16. 搜索文件夹名称和主色 hex 时，能命中对应资源；无关资源不被误匹配。

验证命令：

```bash
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-virtual-folders test SWIFT_EMIT_LOC_STRINGS=NO
git diff --check
```

不会启动 App 做视觉检查，UI 由你手动查看。

## 实施顺序

1. 为 Core Data 创建 v2 model version，保留 v1，并设置 v2 current。
2. 扩展模型：`AssetFolder`、`AssetColor`、`AssetItem.folderIDs`、`AssetItem.paletteColors`、`AssetItem.thumbnailURL`、`SidebarSelection.folder`。
3. 扩展 Core Data v2 model：新增 `FolderRecord`、`AssetFolderMembershipRecord`、`AssetColorRecord`。
4. 打开 Core Data 轻量迁移。
5. 扩展 `LibraryStorage` 的 thumbnail URL helper。
6. 新增 `AssetThumbnailService`，并接入 `AssetImportService`。
7. 新增 `AssetColorAnalysisService`，并接入 `AssetImportService`。
8. 扩展 `LibraryMetadataStore` 的 folder/membership/color 读写、空名称校验、已有资源关联。
9. 扩展 `LibraryMetadataStore` 的 `unassignAssets(ids:from:)`。
10. 调整 manifest schema 兼容检查，让 v1 manifest 可以进入 Core Data 迁移流程。
11. 新增缩略图缓存修复协调逻辑，接入清除缓存并重新加载流程。
12. 扩展 `LibraryStore`：`folders` 状态、创建/删除文件夹、文件夹过滤、显式资源关联、移除关联、搜索匹配、`mergeAssets(_:)`。
13. 调整 `ContentView`：新建/删除文件夹弹窗状态、文件夹操作传递、导入保持无文件夹关联。
14. 调整 `MomentoShellView` 和 `MomentoSidebarView`：传入 folders 和文件夹操作回调。
15. 调整 `AssetCollectionGridView`：优先使用 `thumbnailURL`。
16. 增加可展开的多层文件夹列表 UI、空状态、创建子文件夹入口、删除入口。
17. 调整 Inspector，用 `thumbnailURL` 展示预览，用 `paletteColors` 展示颜色条和百分比。
18. 增加针对存储、导入、缩略图、颜色分析、迁移、搜索和 Store 内存同步的最小可信测试。
19. 运行验证并提交。

## 边界风险

- Core Data model 如果不做版本化，旧资源库迁移测试没有意义，用户已有库可能打不开。
- 如果 manifest schema 等值检查先于 Core Data 迁移拒绝 v1 资源库，真实旧库会打不开。
- 如果导入自动关联当前文件夹，会和“导入默认未分类”的产品语义冲突。
- 如果 `LibraryStore` 只 append 新资源，不替换已有资源，显式关联已有资源后 UI 不会立即更新。
- 如果删除文件夹只删数据库，不同步内存 `folderIDs`，`未分类` 和文件夹视图会短时间显示错误数据。
- 色彩分析如果全量处理大图，导入性能会变差；必须先 downsample。
- 色彩分析如果使用随机算法，测试和 UI 结果会抖动；第一版使用确定性颜色量化直方图。
- 如果列表页缩略图缺失时直接加载原图，大图资源会造成滚动和内存问题；缺失时必须显示占位。
- 如果清除缓存后没有重建缩略图路径，用户会误以为资源预览坏掉。
- 如果没有 `unassignAssets(ids:from:)`，错误归类只能通过删除文件夹修正，数据操作不完整。
- 如果搜索不包含文件夹和主色，用户导入后看到的分类和颜色元数据无法被检索。

## 已确认的产品决策

1. 删除文件夹只删除文件夹和关联，不删除资源。
2. 文件夹支持无限层级嵌套，不只做顶层。
3. 文件夹允许重名，只禁止空名称。
4. `未分类` 是状态筛选，不是真实文件夹。
5. 导入图片后默认无文件夹关联，因此会被 `未分类` 筛选出来。
6. 删除当前正在查看的文件夹后切回 `全部`。
7. 本轮不设计资源网格拖拽到文件夹的关联交互，后续单独做。
8. 本轮导入、颜色分析和缩略图只覆盖图片资源，PDF、视频等先不管。
9. 缩略图只生成一档，建议最长边 512 px。
10. 清除缓存后立即尝试重建缺失缩略图。
11. 从文件夹移除资源只删除关联，不删除资源、不改标签。
12. 第一版搜索使用现有内存数据，不新增持久化 SearchIndex；后续规模或搜索能力需要时再单独设计持久化索引。
