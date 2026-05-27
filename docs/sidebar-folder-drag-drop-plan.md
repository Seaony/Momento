# 左侧边栏文件夹拖拽整理需求与执行说明

日期：2026-05-27

## 目标

把左侧边栏的文件夹整理做成可靠、可验证、符合 macOS 拖拽习惯的交互：

- 文件夹可以拖拽调整同级顺序。
- 文件夹可以拖到另一个文件夹上，成为它的子文件夹。
- 拖到折叠文件夹上方并悬停时自动展开，方便继续拖到更深层级。
- 资源库内的图片可以拖到文件夹上，并自动关联到该文件夹。

最终标准不是“拖拽时看起来有反馈”，而是操作必须落到 `LibraryStore` / `LibraryMetadataStore`，重新打开资源库后仍能保留文件夹顺序、嵌套结构和图片归属。

## 非目标

- 不改 Core Data schema。当前 `AssetFolder.parentID`、`sortIndex` 和 `AssetFolderMembershipRecord` 已经能表达本需求。
- 不引入第三方拖拽库。
- 不重写整个侧栏。
- 不支持跨资源库拖文件夹或跨资源库分配图片。跨库 payload 必须拒绝。
- 不把 Finder 文件拖到文件夹视为本轮目标。Finder 导入仍走现有导入路径。
- 不新增标签拖拽、智能文件夹、批量重命名等扩展功能。
- 不新增缓存。拖拽状态必须是短生命周期 UI state，drop 结束或取消后清理。

## 当前代码事实

- `Momento/Features/Sidebar/MomentoSidebarView.swift`
  - 已有 `MomentoSidebarFolderDropDelegate`，支持 `.before`、`.into`、`.after` 三种 placement。
  - 已有 `MomentoSidebarAssetDropDelegate`，目标是接收资源拖拽并调用 `onAssignDroppedAssetsToFolder(assetIDs, folder.id)`。
  - 已有 `expandedFolderIDs` 和 `scheduleFolderExpansionAfterDropHover(_:)`。
  - 没有明确的 root drop zone，用户不容易把子文件夹拖回顶层末尾。
- `Momento/AppKitBridge/AssetDragPasteboardWriter.swift`
  - 已定义 asset payload：`com.seaony.momento.asset-ids`。
  - 已定义 folder payload：`com.seaony.momento.folder-id`。
- `Momento/Info.plist`
  - 已声明 asset payload 为 exported UTI，conforms to `public.json`。
  - 尚未声明 folder payload。
- `Momento/AppKitBridge/AssetFilePromiseProvider.swift`
  - `NSCollectionView` 资源拖出时通过 `NSPasteboardWriting` 写入 asset payload，同时保留 Finder file promise。
- `Momento/Core/LibraryStore.swift`
  - `moveFolder(id:toParentID:relativeTo:insertAfterTarget:)` 已存在。
  - `assignAssets(ids:to:)` 已存在，并会过滤非当前活跃资源。
- `Momento/Storage/LibraryMetadataStore.swift`
  - `moveFolder` 已做目标存在性校验、循环嵌套防护、同级 sort index 归一化和持久化。
  - `targetID == nil` 时，当前实现会把移动项追加到同级末尾。
  - `assignAssets` 已通过 membership 表建立资源和文件夹关系。
- `MomentoTests/ImportServiceSmokeTests.swift`
  - 已有文件夹排序/嵌套持久化测试。
  - 已有资源分配到文件夹的存储层测试。

结论：数据层基本具备能力。需要补强的是侧栏拖拽协议声明、drop 目标判定、root 落点、payload 读取可靠性和验证闭环。

## 官方文档依据

- Apple 的 Uniform Type Identifiers 文档要求 App 自有的专有数据类型在 `Info.plist` 中声明，并定义 conformance，例如 `public.json`：https://developer.apple.com/documentation/uniformtypeidentifiers/defining-file-and-data-types-for-your-app
- SwiftUI `onDrop(of:delegate:)` 只会为传入的 supported content types 创建 drop destination： https://developer.apple.com/documentation/swiftui/view/ondrop%28of%3Adelegate%3A%29
- SwiftUI `DropInfo.itemProviders(for:)` 文档说明该读取只在 `performDrop()` 行为中有效。当前代码在 `performDrop` 中读取 provider，这个方向是对的；不要把异步解码挪到 `dropUpdated`：https://developer.apple.com/documentation/swiftui/dropinfo/itemproviders%28for%3A%29-b6fo
- `DropDelegate.dropUpdated(info:)` 应返回当前 drop proposal，用于表达 copy/move 等操作： https://developer.apple.com/documentation/swiftui/dropdelegate/dropupdated%28info%3A%29-2mktz
- AppKit `NSView.registerForDraggedTypes(_:)` 会让 view 成为对应 pasteboard 类型的拖拽目标；如果使用 AppKit receiver，必须注册类型并实现 `NSDraggingDestination`：https://developer.apple.com/documentation/appkit/nsview/registerfordraggedtypes%28_%3A%29
- AppKit `NSDraggingInfo.draggingPasteboard` 是拖拽操作实际使用的数据源： https://developer.apple.com/documentation/appkit/nsdragginginfo/draggingpasteboard
- AppKit `NSPasteboardWriting.pasteboardPropertyList(forType:)` 支持自定义类型返回 `Data`：https://developer.apple.com/documentation/appkit/nspasteboardwriting/pasteboardpropertylist%28fortype%3A%29

## Review 结论

### 原文档中过度设计的点

原文档把 AppKit drop receiver 写成了默认方案，这不够严谨。当前代码的文件夹拖拽源和目标都在 SwiftUI 侧，Apple 文档也支持 `onDrop(of:delegate:)` 这条路径。最佳实践不是一开始就叠一个 AppKit receiver，而是先修正已知声明缺口和 SwiftUI drop 逻辑，再用手动复现确认 asset payload 是否真的无法通过 SwiftUI `DropInfo.itemProviders` 到达。

调整后方案：

- 文件夹拖拽优先走现有 SwiftUI `onDrag` / `onDrop`。
- 图片拖到文件夹也优先保留现有 SwiftUI delegate。
- 只有在验证证明 `DropInfo.itemProviders(for:)` 拿不到 `AssetFilePromiseProvider` 写入的 asset payload，而 AppKit dragging pasteboard 能拿到时，才增加 AppKit receiver。
- 如果引入 AppKit receiver，必须替换对应 SwiftUI drop handler，不能双路径并存。

### 不合理或冲突点

- 不能把 folder UTI 未声明写成“唯一根因”。它是明确缺口，但拖拽失败还可能来自 SwiftUI state、drop hit area、provider 桥接或 handler 重复。
- 不能在 `dropUpdated` 解码 `NSItemProvider`。Apple 文档限定 `itemProviders(for:)` 只在 `performDrop` 中有效。
- 不能靠额外缓存或 session token 遮住双路径 drop。正确做法是保留一个权威 drop path。
- 不能让 AppKit receiver 直接写 store，否则会绕开 `ContentView` 当前错误处理和数据流。

### 是否符合最佳实践

修订后的方案符合：

- 自有 UTI 在 `Info.plist` 声明。
- SwiftUI 内部拖拽优先使用 SwiftUI delegate。
- AppKit fallback 只用于 AppKit drag pasteboard 到 SwiftUI bridge 被证实不可靠的场景。
- drop placement 计算抽成小型纯逻辑，解决可测试性，而不是新增通用树框架。
- store 和 storage 继续作为唯一数据写入边界。

## 期望交互

### 文件夹排序

1. 用户拖动任意文件夹 row。
2. 拖到目标文件夹上边缘时显示细 drop line。
3. 松手后，被拖文件夹移动到目标前面，parentID 等于目标 parentID。
4. 拖到目标下边缘时同理，移动到目标后面。
5. 操作后选中状态保持在被拖文件夹；列表按新的 sortIndex 刷新。

### 文件夹嵌套

1. 用户把文件夹拖到另一个文件夹 row 中间区域。
2. 目标 row 高亮，表示“放入”。
3. 松手后，被拖文件夹 parentID 变成目标文件夹 id。
4. 目标文件夹自动展开，让用户立即看到新子项。
5. 拒绝把文件夹拖进自己或自己的后代。

### 悬停自动展开

1. 用户拖着文件夹悬停在折叠文件夹的中间区域。
2. 约 0.55 秒后目标文件夹展开。
3. 如果用户移到上/下边缘或离开 row，不展开。
4. drop 结束、drop 退出或 drag 取消时清理 pending expansion。

### 图片拖到文件夹

1. 用户在资源网格、列表或瀑布流里选中一张或多张图片。
2. 拖到左侧某个文件夹 row。
3. 目标 row 高亮，drop proposal 为 copy/organize。
4. 松手后所有当前库内、非废纸篓的 asset id 通过 `store.assignAssets(ids:to:)` 关联到该文件夹。
5. 文件夹计数更新；如果当前正在查看该文件夹，内容列表包含新关联资源。

## 执行计划

### Task 0：复现和定位，不先写大改动

Files:

- Inspect: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Inspect: `Momento/AppKitBridge/AssetFilePromiseProvider.swift`
- Inspect: `Momento/AppKitBridge/AssetDragPasteboardWriter.swift`

Steps:

- 手动验证当前三条路径：
  - 文件夹拖到文件夹上/中/下区域。
  - 文件夹拖到文件夹列表空白处。
  - 资源从 `NSCollectionView` 拖到文件夹 row。
- 观察是 drop target 完全不激活，还是激活后 `performDrop` 没写入。
- 如需临时诊断，只在本地加 Debug-only logging；诊断日志不提交，除非后续决定做正式 telemetry。
- 如果执行者无法启动 App 做手动复现，不要停在这里；先执行 Task 1-4 的自动化修复和测试，最后把手动 QA 作为交付前验证项。

Gate:

- 如果文件夹 drop target 不激活，先进入 Task 1。
- 如果图片 drop target 激活但 `assetIDs` 为空或 provider 为空，进入 Task 5 的 fallback 判定。

### Task 1：补齐内部拖拽 UTI 声明

Files:

- Modify: `Momento/Info.plist`
- Modify: `MomentoTests/ArchitectureGuardTests.swift`

Implementation:

- 增加 `com.seaony.momento.folder-id` 到 `UTExportedTypeDeclarations`。
- 设置：
  - `UTTypeIdentifier = com.seaony.momento.folder-id`
  - `UTTypeDescription = Momento Folder Drag Payload`
  - `UTTypeConformsTo = public.json`
- 更新 guard 测试，确保 asset 和 folder payload 都声明为 `public.json`。

Validation:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/ArchitectureGuardTests
```

### Task 2：抽出最小 folder drop resolver

Files:

- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Create: `MomentoTests/SidebarFolderDropResolverTests.swift`

Implementation:

- 把现有 `MomentoSidebarFolderDropDelegate.moveCommand` 的规则抽成小型 resolver，例如 `MomentoSidebarFolderDropResolver`。
- resolver 和测试需要用到的 placement/command 值类型保持 `internal`，让 `@testable import Momento` 能覆盖规则；SwiftUI delegate 和 row 视图仍保持 `private`。
- 输入：
  - dragged folder id
  - target row folder，可选
  - all folders
  - placement：before / into / after / rootEnd
- 输出沿用现有 command：
  - `parentID`
  - `targetID`
  - `insertAfterTarget`
- 不创建通用树编辑框架，不改变 `LibraryStore.moveFolder` 签名。

Test cases:

- before：parentID 等于目标 parentID，targetID 为目标，`insertAfterTarget = false`。
- after：parentID 等于目标 parentID，targetID 为目标，`insertAfterTarget = true`。
- into：parentID 为目标 id，targetID nil。
- rootEnd：parentID nil，targetID nil；按 storage 当前实现追加到顶层末尾。
- 自己拖到自己：nil。
- 父拖到子孙：nil。
- 跨库不在 resolver 里处理；由 payload decode 阶段拒绝。

Validation:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/SidebarFolderDropResolverTests
```

### Task 3：完善 SwiftUI folder drop path

Files:

- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`

Implementation:

- 保留 row 级 `.onDrag` / `.onDrop`。
- row 上/下边缘显示 `folderDropIndicator`。
- row 中间区域高亮表示 `.into`。
- 新增 folder section root drop zone：
  - 文件夹列表空白处或空状态处接收 folder payload。
  - 生成 resolver 的 `.rootEnd` command。
  - 显示一条底部 drop line。
- `dropUpdated` 只做同步可得的 state/placement 判断，不解码 `NSItemProvider`。
- `performDrop` 再读取 provider，并回到 `MainActor` 调用 `onMoveFolder`。
- drop 完成、退出和已知失败路径清空：
  - `draggingFolderID`
  - `targetedFolderDrop`
  - `pendingFolderExpansionID`
- SwiftUI `onDrag` 没有可靠的全局 drag ended 回调，因此不要把 `draggingFolderID` 当作最终权限判断；`performDrop` 解码出的 payload 和 storage 层校验才是最终事实。

Validation:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/SidebarFolderDropResolverTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/LibraryPackagePersistenceTests/testMovingFoldersPersistsManualOrderAndHierarchy -only-testing:MomentoTests/LibraryPackagePersistenceTests/testMovingFolderIntoDescendantIsRejected
```

Manual QA:

- 文件夹拖到同级前/后。
- 文件夹拖入另一个文件夹。
- 文件夹拖到空白 root drop zone。
- 父文件夹拖到子文件夹时无写入。

### Task 4：修通图片拖到文件夹的 SwiftUI path

Files:

- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Inspect/Test: `Momento/AppKitBridge/AssetFilePromiseProvider.swift`
- Create: `MomentoTests/DragPasteboardWriterTests.swift`

Implementation:

- 先不新增 AppKit receiver。
- 确认 `AssetFilePromiseProvider.writableTypes(for:)` 包含 `AssetDragPasteboardWriter.assetIDsPasteboardType`。
- 确认 `pasteboardPropertyList(forType:)` 返回的 Data 能解码为 `AssetDragPasteboardPayload`。
- 只有测试证明 payload 写入不完整时才修改 `AssetFilePromiseProvider`；如果现有实现已经正确，不做无关改动。
- `MomentoSidebarAssetDropDelegate.performDrop` 继续只在 `performDrop` 中读取 provider。
- 解码后必须校验：
  - payload.libraryID == currentLibraryID
  - assetIDs 非空
- UI 回调必须回到 `MainActor`。
- `assignDroppedAssetsToFolder` 继续调用 `store.assignAssets(ids:to:)`，不绕过 ContentView。

Validation:

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/DragPasteboardWriterTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/LibraryPackagePersistenceTests
```

Manual QA:

- 单选图片拖到文件夹。
- 多选图片拖到文件夹。
- 当前正在查看目标文件夹时，drop 后内容更新。
- 废纸篓资源不被重新关联。

### Task 5：只有证据充分时增加 AppKit fallback

Files:

- Create only if needed: `Momento/AppKitBridge/SidebarPasteboardDropReceiver.swift`
- Modify only if needed: `Momento/Features/Sidebar/MomentoSidebarView.swift`

进入条件：

- Task 1-4 后，手动 QA 仍证明图片 drop 失败。
- 失败点明确为 SwiftUI `DropInfo.itemProviders(for:)` 在 `performDrop` 中拿不到 asset payload。
- 同一拖拽的 AppKit pasteboard 可以读取 `com.seaony.momento.asset-ids`。

Implementation if needed:

- 用 `NSViewRepresentable` 包一个薄 AppKit receiver。
- receiver 只负责：
  - `registerForDraggedTypes`
  - 从 `NSDraggingInfo.draggingPasteboard` 读取 Data
  - 解析 payload
  - 回调 SwiftUI closure
- receiver 不持有 `LibraryStore`。
- receiver 不改业务状态。
- 如果 receiver 接管 asset drop，则移除同 row 上 asset 类型的 SwiftUI `.onDrop`，避免双写。
- folder drop 不默认迁移到 AppKit receiver，除非 Task 3 仍被证据证明失败。

Validation:

```bash
xcodebuild build -project Momento.xcodeproj -scheme Momento -destination platform=macOS
git diff --check
```

Manual QA:

- 图片拖到文件夹只触发一次关联。
- Finder 拖文件到文件夹不触发内部 assign。
- 文件夹拖拽仍走同一权威路径，不重复调用 move。

## 验证矩阵

| 场景 | 预期 |
| --- | --- |
| Folder A 拖到 Folder B 上边缘 | A 与 B 同 parent，A 排在 B 前 |
| Folder A 拖到 Folder B 下边缘 | A 与 B 同 parent，A 排在 B 后 |
| Folder A 拖到 Folder B 中间 | A.parentID == B.id，B 展开 |
| Parent 拖到 Child 中间 | drop 被拒绝，store 不写入 |
| Child 拖到空白 root drop zone | Child.parentID == nil，排到顶层末尾 |
| 拖到折叠 folder 中间并悬停 | folder 自动展开 |
| 图片单选拖到 folder | asset.folderIDs 包含 folder.id |
| 图片多选拖到 folder | 所有当前库内、非废纸篓选中项都关联 |
| 跨资源库 payload | drop 被拒绝 |
| Finder 文件拖到 folder | 本轮不处理，不调用 assignAssets |
| 同一次 drop | 不重复调用 move/assign |

## 推荐验证命令

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/ArchitectureGuardTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/SidebarFolderDropResolverTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/DragPasteboardWriterTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/LibraryPackagePersistenceTests/testMovingFoldersPersistsManualOrderAndHierarchy -only-testing:MomentoTests/LibraryPackagePersistenceTests/testMovingFolderIntoDescendantIsRejected
xcodebuild build -project Momento.xcodeproj -scheme Momento -destination platform=macOS
git diff --check
```

## 风险与约束

- 双路径 drop：同一个 payload 类型只能有一个最终处理路径。
- 坐标判断：如果实现 AppKit receiver，`draggingLocation` 是窗口坐标，必须转换到 row 本地坐标再判断 before/into/after。
- 异步解码：`NSItemProvider.loadDataRepresentation` completion 不保证在主线程；UI/store 回调必须回到 `MainActor`。
- 视觉误导：无效 drop 不能高亮成可接受状态。
- 手动 QA 必须覆盖快速拖拽、拖出侧栏后取消、折叠 section 内拖拽。

## 完成定义

- 自动测试覆盖 UTI 声明、folder resolver、payload 编解码、持久化写入。
- 手动 QA 覆盖文件夹排序、嵌套、root drop、悬停展开、图片拖到文件夹。
- 没有新增 schema、依赖或缓存。
- 没有扩大业务 API、store public contract 或持久化格式。
- 没有两个 drop handler 同时处理同一种 payload。
- `xcodebuild build` 和 `git diff --check` 通过。
