# Virtual Folders Design

## 背景

Momento 目前已经有资源库包、资源导入、最近资源库列表、侧边栏导航和资源网格，但“文件夹”仍只是侧边栏里的空 UI 壳：

- `Momento/Core/AssetModels.swift` 只有 `AssetItem`、`TagItem` 和 `SidebarSelection.folderManagement`，没有文件夹模型。
- `Momento/Storage/MomentoModel.xcdatamodeld` 目前只有 `AssetRecord` 一个 Core Data 实体。
- `Momento/Storage/LibraryMetadataStore.swift` 只负责读取和保存资源，没有保存文件夹或资源-文件夹关系。
- `Momento/Core/LibraryStore.swift` 的 `.uncategorized` 当前仍返回全部资源，这只是临时行为，不符合真实文件夹语义。
- `Momento/Services/AssetImportService.swift` 会用已有 hash 跳过库内重复资源；如果用户把同一张图再次导入到某个文件夹，当前流程无法只新增“文件夹关联”。

这次要实现的是虚拟文件夹，而不是把 `.momento/assets` 里的物理文件按 Finder 目录移动。

## 外部参考

- Eagle Plugin API 的 item 数据上有 `folders: string[]`，并提供 `isUnfiled` 查询条件。这说明一项资源可以属于多个文件夹，未分类是由文件夹关联为空推导出来的。
- Eagle Plugin API 的 folder 数据有 `id`、`name`、`parent`、`children`，说明文件夹本身是独立元数据，和磁盘目录不是一回事。
- SwiftUI 官方文档中 `fileImporter` 用于系统文件选择，并要求处理 security-scoped resource；Momento 当前导入服务已经遵循这个方向。
- SwiftUI 官方文档中列表排序可通过 `onMove` 或自定义 drop 流程更新底层数组；本次先不做文件夹拖拽排序，只保留 `sortIndex` 数据字段，避免把 UI 交互复杂度一次拉满。

参考链接：

- [Eagle item API](https://developer.eagle.cool/plugin-api/api/item)
- [Eagle folder API](https://developer.eagle.cool/plugin-api/api/folder)
- [SwiftUI fileImporter](https://developer.apple.com/documentation/swiftui/view/fileimporter%28ispresented%3Aallowedcontenttypes%3Aallowsmultipleselection%3Aoncompletion%3Aoncancellation%3A%29)
- [SwiftUI lists and move actions](https://developer.apple.com/documentation/swiftui/lists)

## 目标

1. 资源库内支持创建和删除文件夹。
2. 文件夹是虚拟分类，一张资源可以关联多个文件夹。
3. 点击侧边栏文件夹时，只显示关联到该文件夹的资源。
4. `未分类` 只显示没有任何文件夹关联的资源。
5. 用户在某个文件夹上下文中导入图片、GIF、SVG、视频、PDF 或文件夹时，新导入资源会自动关联当前文件夹。
6. 用户把 Finder 文件拖到侧边栏某个文件夹上时，资源会导入并关联这个文件夹。
7. 删除文件夹只删除文件夹元数据和关联关系，不删除资源文件，也不删除资源记录。
8. 已有资源库可以通过轻量迁移继续打开。

## 非目标

- 不实现 Finder 真实目录同步。
- 不移动 `.momento/assets` 下的物理文件。
- 不实现文件夹拖拽排序、拖拽嵌套、重命名、颜色、图标选择。
- 不实现智能文件夹、筛选规则、批量移动、右键完整菜单。
- 不实现跨资源库复制文件夹结构。

这些能力可以在虚拟文件夹的数据基础上继续扩展，但不放进这次实现。

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

### 删除文件夹采用“删除分类，不删资源”

删除文件夹时：

- 删除该文件夹记录。
- 删除该文件夹以及子文件夹的所有资源关联记录。
- 资源记录和物理文件保留。
- 如果某个资源因此不再属于任何文件夹，它会自然出现在 `未分类`。

如果当前正在查看被删除的文件夹，删除完成后切回 `全部`。

### 导入流程要允许“重复资源补关联”

当前导入服务会用已有 hash 跳过库内重复资源。虚拟文件夹上线后，这个行为需要调整：

- 导入服务仍然做批内去重，避免同一次导入重复处理同一个 hash。
- 不再因为“库里已有相同 hash”直接跳过结果。
- 持久化层如果发现 `AssetRecord` 已存在，返回已有资源。
- 持久化层再为这些新资源或已有资源创建文件夹关联。

这样用户把已经存在的图片导入到新文件夹时，不会复制文件，但会把这张图片加入该文件夹。

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
```

修改 `AssetItem`：

```swift
var folderIDs: [String]
```

修改 `SidebarSelection`：

```swift
case folder(String)
```

保留 `folderManagement` 的现有 case，以免命令面板或旧 UI 引用立刻断掉；但侧边栏这次会优先展示真实文件夹列表。

### Core Data 实体

在 `MomentoModel.xcdatamodeld` 新增两个实体。

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

暂时不加 Core Data relationship，保持和现有 `LibraryMetadataStore` 一样的 key-value `NSManagedObject` 风格。这样实现简单，迁移风险也低。

### 迁移

`MomentoCoreDataStack` 需要打开轻量迁移：

```swift
storeDescription.shouldMigrateStoreAutomatically = true
storeDescription.shouldInferMappingModelAutomatically = true
```

这次只新增实体和字段，不修改 `AssetRecord` 现有字段语义，适合轻量迁移。

## 存储层设计

`Momento/Storage/LibraryMetadataStore.swift` 增加以下能力：

```swift
func loadFolders() throws -> [AssetFolder]
func createFolder(name: String, parentID: String?) throws -> AssetFolder
func deleteFolder(id: AssetFolder.ID) throws -> [AssetFolder.ID]
func saveImportedAssets(_ assets: [AssetItem], assigningTo folderID: AssetFolder.ID?) throws -> [AssetItem]
```

读取资源时：

1. 查询当前库的所有 `AssetFolderMembershipRecord`。
2. 按 `assetID` 组成 `[String: [String]]`。
3. 构造 `AssetItem` 时填入 `folderIDs`。

保存导入资源时：

1. 先按 `contentHash` 查已有 `AssetRecord`。
2. 已存在则复用已有资源值。
3. 不存在则创建新的 `AssetRecord`。
4. 如果传入 `folderID`，为每个保存后的资源创建 membership。
5. 保存后返回带最新 `folderIDs` 的 `AssetItem`。

删除文件夹时：

1. 找到目标文件夹以及所有后代文件夹 ID。
2. 删除这些 `FolderRecord`。
3. 删除这些 folderID 对应的 `AssetFolderMembershipRecord`。
4. 返回实际删除的 folderID 列表，供 `LibraryStore` 更新内存状态。

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

新增方法：

```swift
func createFolder(named name: String, parentID: AssetFolder.ID?) throws
func deleteFolder(id: AssetFolder.ID) throws
func importItems(from urls: [URL], assigningTo folderID: AssetFolder.ID?) async throws
```

现有 `importItems(from:)` 保留，内部根据当前选择推导默认目标文件夹：

```swift
let selectedFolderID: String? = {
    if case .folder(let id) = sidebarSelection { id } else { nil }
}()
try await importItems(from: urls, assigningTo: selectedFolderID)
```

`visibleAssets` 规则：

- `.library`: 当前库全部资源。
- `.favorites`: `isFavorite == true`。
- `.uncategorized`: `folderIDs.isEmpty`。
- `.untagged`: `tags.isEmpty`。
- `.folder(id)`: `folderIDs.contains(id)`。
- `.tag(id)`: 保持现有 tag 过滤。
- `.tagManagement`、`.folderManagement`、`.trash`: 暂时为空。

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
var onImportIntoFolder: (AssetFolder.ID, [URL]) -> Void
```

文件夹标题区保持当前视觉规则：

- 标题“文件夹”常亮。
- hover 后显示 `+` 和展开/收起按钮。
- `+` 打开新建文件夹弹窗。

文件夹列表：

- 有文件夹时渲染真实文件夹行。
- 没有文件夹时保持当前的“暂无文件夹”空状态。
- 文件夹行 hover 使用和侧边栏其他项一致的 `MomentoTheme.sidebarIconHoverBackground`。
- 选中时文字更亮，并使用同样的背景逻辑。
- 每行右侧 hover 时显示删除按钮或更多按钮；本次只需要删除文件夹。

拖入文件：

- 文件夹行支持 `dropDestination(for: URL.self)`。
- 用户从 Finder 拖文件或文件夹到某个文件夹行时，调用 `onImportIntoFolder(folderID, urls)`。

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

### 删除文件夹确认

删除前必须确认，避免误操作。

文案方向：

```text
删除“Jobs”文件夹？

这个操作只会删除文件夹和它的资源关联，不会删除资源库中的图片、视频或其它文件。
如果某些资源不再属于任何文件夹，它们会显示在“未分类”里。
```

按钮：

- `取消`
- `删除文件夹`，红色 Liquid Glass 按钮

## 导入行为

### 在文件夹视图导入

当当前侧边栏选中 `.folder(id)`：

- 点击主区域空状态的导入按钮。
- 从命令面板执行导入。
- 从 toolbar/search 外的全局 `fileImporter` 选择文件。
- 拖文件到主内容区域。

这些入口都应把资源关联到当前文件夹。

### 拖到文件夹行导入

当用户把 Finder 文件拖到某个文件夹行：

- 不改变当前选区也可以导入。
- 导入完成后可以选中目标文件夹，让用户立刻看到结果。

### 重复资源

如果导入的文件已经存在：

- 不复制物理文件。
- 不新增 `AssetRecord`。
- 只补齐该资源到目标文件夹的 membership。
- 如果 membership 已存在，保持幂等。

## 本地化

需要新增或确认以下文案：

- `New Folder`
- `Folder Name`
- `Create`
- `Delete Folder`
- `Delete “%@” folder?`
- `This only removes the folder and its asset associations. Assets remain in the library.`

注意：按当前约定，如果 `Momento/Localizable.xcstrings` 因 Xcode 字符串提取发生变化，直接随任务一起提交，不单独调查或清理。

## 验证计划

这次不是纯 UI 微调，涉及持久化和迁移，所以需要保留少量高价值测试。

建议新增或调整 `MomentoTests/ImportServiceSmokeTests.swift`：

1. 创建文件夹后重开资源库，文件夹仍存在。
2. 在文件夹上下文导入图片后，资源的 `folderIDs` 包含该文件夹。
3. 重复导入同一图片到另一个文件夹，不复制资源文件，但新增 folder membership。
4. 删除文件夹后，资源仍存在，`folderIDs` 移除该文件夹；无其它文件夹的资源出现在 `未分类`。
5. 老资源库轻量迁移后仍能打开，旧资源默认 `folderIDs == []`。

验证命令：

```bash
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-virtual-folders test SWIFT_EMIT_LOC_STRINGS=NO
git diff --check
```

不会启动 App 做视觉检查，UI 由你手动查看。

## 实施顺序

1. 扩展模型：`AssetFolder`、`AssetItem.folderIDs`、`SidebarSelection.folder`。
2. 扩展 Core Data model：新增 `FolderRecord` 和 `AssetFolderMembershipRecord`。
3. 打开 Core Data 轻量迁移。
4. 扩展 `LibraryMetadataStore` 的 folder/membership 读写。
5. 调整 `AssetImportService`，允许已有 hash 返回给持久化层补 membership。
6. 扩展 `LibraryStore`：`folders` 状态、创建/删除文件夹、文件夹过滤、按上下文导入。
7. 调整 `ContentView`：新建/删除文件夹弹窗状态和导入目标传递。
8. 调整 `MomentoShellView` 和 `MomentoSidebarView`：传入 folders 和文件夹操作回调。
9. 增加文件夹列表 UI、文件夹行 drop import、空状态。
10. 增加针对存储和导入的最小可信测试。
11. 运行验证并提交。

## 需要你 review 的点

1. 删除文件夹是否确认采用“只删文件夹和关联，不删资源”的语义？
2. 本次是否先只做新建顶层文件夹和删除文件夹？数据模型会预留 `parentID`，但不做创建子文件夹和拖拽嵌套。
3. 重复导入同一张图片到不同文件夹时，是否按“补关联，不复制文件”的方式处理？
4. 文件夹删除后，如果当前正在查看它，是否切回 `全部`？
