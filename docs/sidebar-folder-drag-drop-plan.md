# 左侧边栏文件夹拖拽整理需求与执行说明

日期：2026-05-27

## 目标

把左侧边栏的文件夹整理做成可靠的原生 macOS 拖拽体验：

- 文件夹可以直接拖拽调整同级顺序。
- 文件夹可以拖到另一个文件夹上，成为它的子文件夹。
- 拖拽悬停在可展开文件夹上时，文件夹自动展开，方便拖到更深层级。
- 资源库中的图片可以拖到文件夹上，并自动关联到该文件夹。

实现标准是修通现有数据流，而不是新增一套临时拖拽状态或伪成功 UI。拖拽成功必须落到 `LibraryStore` / `LibraryMetadataStore`，重新打开资源库后仍然保持排序、层级和资源归属。

## 非目标

- 不改 Core Data schema。当前 `AssetFolder.parentID`、`sortIndex` 和 `AssetFolderMembershipRecord` 已经能表达排序、嵌套和资源归属。
- 不引入第三方拖拽库。
- 不重写整个侧边栏，也不把文件夹树抽成新的大型架构。
- 不支持跨资源库拖文件夹或跨资源库分配图片。跨库 payload 必须拒绝。
- 不把 Finder 文件拖到文件夹视为本轮目标。Finder 导入仍走现有导入路径，库内资源整理只接受 Momento 内部 asset payload。
- 不新增“为了顺手”的标签拖拽、批量重命名、智能文件夹等扩展功能。

## 当前代码事实

- `Momento/Features/Sidebar/MomentoSidebarView.swift`
  - 已有 `MomentoSidebarFolderDropDelegate`，支持 `.before`、`.into`、`.after` 三种 placement。
  - 已有 `MomentoSidebarAssetDropDelegate`，意图接收资源拖拽并调用 `onAssignDroppedAssetsToFolder(assetIDs, folder.id)`。
  - 已有 `expandedFolderIDs` 和 `scheduleFolderExpansionAfterDropHover(_:)`，自动展开的状态基础存在。
- `Momento/AppKitBridge/AssetDragPasteboardWriter.swift`
  - 已定义 asset payload：`com.seaony.momento.asset-ids`。
  - 已定义 folder payload：`com.seaony.momento.folder-id`。
- `Momento/AppKitBridge/AssetFilePromiseProvider.swift`
  - 资源从 `NSCollectionView` 拖出时通过 `NSPasteboardWriting` 写入 asset payload，同时保留 Finder 文件承诺。
- `Momento/Core/LibraryStore.swift`
  - `moveFolder(id:toParentID:relativeTo:insertAfterTarget:)` 已存在。
  - `assignAssets(ids:to:)` 已存在，并且会过滤非当前活跃资源。
- `Momento/Storage/LibraryMetadataStore.swift`
  - `moveFolder` 已做循环嵌套防护、目标存在性校验、同级 sort index 归一化和持久化。
  - `assignAssets` 已通过 membership 表建立资源和文件夹关系。
- `MomentoTests/ImportServiceSmokeTests.swift`
  - 已有文件夹排序/嵌套持久化测试。
  - 已有资源分配到文件夹的存储层测试。

结论：数据层不是缺失点。要修的是左侧边栏拖拽接收和 payload 桥接的可靠性。

## 主要问题判断

### 1. 文件夹拖拽的 payload 没有完整注册

asset payload 已在 `Info.plist` 的 `UTExportedTypeDeclarations` 中声明为 `public.json`，folder payload 目前只在代码里 `UTType(exportedAs:)`，没有同等 Info.plist 声明。这个缺口不一定是唯一根因，但它让 folder payload 和 asset payload 的系统声明不一致，也让后续测试无法明确守住两类内部拖拽类型。

执行要求：

- 在 `Info.plist` 增加 `com.seaony.momento.folder-id`，描述为 `Momento Folder Drag Payload`，conforms to `public.json`。
- 给这个声明补 architecture guard，避免之后删掉。

### 2. 资源拖拽跨 AppKit 源和 SwiftUI 目标，不能只依赖 `DropInfo.itemProviders`

资源网格是 `NSCollectionView`，拖出对象是 `AssetFilePromiseProvider`，它通过 `NSPasteboardWriting` 提供自定义 asset payload。左侧边栏是 SwiftUI `onDrop`，当前 delegate 通过 `DropInfo.itemProviders(for:)` 读取 payload。这个桥接路径在 macOS 上不够稳：AppKit drag pasteboard 上明明有自定义类型时，SwiftUI 的 item provider 视图层也可能拿不到预期 provider，导致用户看到“能拖动，但放到文件夹没有关联”。

执行要求：

- 保留 `AssetFilePromiseProvider` 的 Finder file promise 行为。
- 对左侧边栏文件夹 drop target 增加一个很薄的 AppKit 接收层，直接从 `NSDraggingInfo.draggingPasteboard` 读取 Momento 自定义 payload。
- 这个接收层只负责 drag hit testing、payload decode、drop proposal 和把解码后的命令回调给 SwiftUI，不直接改 store。
- SwiftUI row 继续负责视觉状态、展开状态和调用 `onMoveFolder` / `onAssignDroppedAssetsToFolder`。

### 3. 文件夹 drop 需要根级落点和更明确的 placement

当前 row 级 placement 能表达放到某个 folder 前、后、内部，但没有一个清晰的 folder section 根级落点。用户把子文件夹拖回顶层最后一个位置，或在空文件夹区拖动时，很容易没有可用目标。

执行要求：

- 保留 row 内三段式 placement：
  - 上边缘：放到目标前面。
  - 中间：放入目标文件夹。
  - 下边缘：放到目标后面。
- 增加 folder section 级 root drop target：
  - 拖到文件夹列表空白处或空状态处，移动为顶层最后一个文件夹。
  - 拖到 section 顶部空白处，移动为顶层首位可以作为第二阶段；第一阶段先保证顶层末尾可用。
- 对无效 drop 给出“不接收”的视觉状态，不要调用 store 后静默失败。

### 4. 自动展开要绑定 drop target 生命周期

已有 `pendingFolderExpansionID` 和 0.55 秒 delay，但需要确保只在 `.into` placement 时触发，并且在离开目标、切换 placement、performDrop 或取消拖拽时清理。

执行要求：

- 只有拖到中心区域 `.into` 时 schedule expand。
- 目标从 `.into` 变成 `.before` / `.after` 时取消 pending expand。
- drop 完成、drop 退出、drag 结束都清理 pending state。
- 已展开文件夹不重复 schedule。

## 期望交互

### 文件夹排序

1. 用户按住任意文件夹 row 拖动。
2. 拖到同级或其他层级文件夹的上边缘，显示一条细 accent drop line。
3. 松手后，被拖文件夹移动到目标前面，parentID 等于目标 parentID。
4. 拖到下边缘同理，移动到目标后面。
5. 操作后选中状态保持在原文件夹；列表按新的 sortIndex 刷新。

### 文件夹嵌套

1. 用户把文件夹拖到另一个文件夹 row 的中间区域。
2. 目标 row 高亮，显示这是“放入”而不是“排序线”。
3. 松手后，被拖文件夹 parentID 变成目标文件夹 id。
4. 目标文件夹自动展开，让用户立即看到新子项。
5. 拒绝把文件夹拖进自己或自己的后代。

### 悬停自动展开

1. 用户拖着文件夹悬停在一个折叠且有子项的文件夹中间区域。
2. 约 0.55 秒后目标文件夹展开。
3. 如果用户移到上/下边缘或离开 row，不展开。

### 图片拖到文件夹

1. 用户在资源网格、列表或瀑布流里选中一张或多张图片。
2. 拖到左侧某个文件夹 row。
3. 目标 row 高亮，drop proposal 表示 copy/organize。
4. 松手后所有当前库内、非废纸篓的 asset id 通过 `store.assignAssets(ids:to:)` 关联到该文件夹。
5. 文件夹计数更新；如果当前正在查看该文件夹，内容列表包含新关联资源。

## 实现说明

### Task 1：补齐拖拽 payload 声明和测试

Files:

- Modify: `Momento/Info.plist`
- Modify: `MomentoTests/ArchitectureGuardTests.swift`
- Test: `MomentoTests/ArchitectureGuardTests.swift`

Steps:

- 增加 `com.seaony.momento.folder-id` 的 `UTExportedTypeDeclarations`。
- 在 architecture guard 中检查：
  - asset payload 仍是 `public.json`。
  - folder payload 也是 `public.json`。
  - 两者 description 不混淆。
- 运行：
  - `xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/ArchitectureGuardTests/testDragPayloadTypesAreExported`

### Task 2：把 drop 命令计算抽成可测试的纯逻辑

Files:

- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Create: `MomentoTests/SidebarFolderDropResolverTests.swift`

Steps:

- 从现有 `MomentoSidebarFolderDropDelegate.moveCommand` 提取小型 resolver，输入为：
  - dragged folder id
  - target row folder
  - all folders
  - placement
  - optional root drop intent
- 输出沿用现有命令形状：
  - `parentID`
  - `targetID`
  - `insertAfterTarget`
- 测试覆盖：
  - before：同 parent，targetID 为目标，`insertAfterTarget = false`。
  - after：同 parent，targetID 为目标，`insertAfterTarget = true`。
  - into：parentID 为目标，targetID 为 nil。
  - root end：parentID nil，targetID nil；按当前 `LibraryMetadataStore` 约定，`targetID == nil` 表示追加到同级末尾。
  - 自己拖到自己：nil。
  - 父拖到子孙：nil。
- 不改变 `LibraryStore.moveFolder` 的签名。

### Task 3：增加 AppKit drop receiver，稳定读取自定义 pasteboard

Files:

- Create: `Momento/AppKitBridge/SidebarDropReceiverView.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Test: `MomentoTests/DragPasteboardWriterTests.swift`

Steps:

- 新增 `NSViewRepresentable`，内部 `NSView` 注册：
  - `AssetDragPasteboardWriter.assetIDsPasteboardType`
  - `NSPasteboard.PasteboardType(FolderDragPasteboardWriter.folderIDTypeIdentifier)`
- 在 `draggingEntered` / `draggingUpdated` 中：
  - 从 `draggingPasteboard` 判断当前 payload 类型。
  - 计算 row placement。
  - 回调 SwiftUI 更新 `targetedAssetDropID` 或 `targetedFolderDrop`。
  - 返回 `.copy` 给 asset drop，`.move` 给 folder drop。
- 在 `performDragOperation` 中：
  - 解码 asset payload 后验证 libraryID，再回调 `onAssignDroppedAssetsToFolder`。
  - 解码 folder payload 后验证 libraryID，再通过 resolver 生成 move command 并回调 `onMoveFolder`。
- AppKit receiver 不直接持有 `LibraryStore`，不吞错误，不伪造成功。
- 原有 SwiftUI `.onDrop` 可以先保留到 AppKit receiver 验证通过；若两个路径会重复触发，则移除 SwiftUI `.onDrop`，只保留一个权威 drop path。
- 单元测试至少验证：
  - `FolderDragPasteboardWriter.itemProvider` 能产出可解码 payload。
  - `AssetFilePromiseProvider.writableTypes` 包含 asset payload。
  - `AssetFilePromiseProvider.pasteboardPropertyList` 可解码 libraryID、assetIDs、primaryAssetID。

### Task 4：完善 row 和 section 级视觉反馈

Files:

- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`

Steps:

- row 中心 drop：沿用 row 背景高亮。
- row 上/下 drop：沿用 `folderDropIndicator`。
- root drop：在文件夹列表底部或空状态处显示一条细 drop line。
- asset drop：row 高亮即可，不显示排序线，避免和 folder move 混淆。
- pending expand 只在 folder `.into` drop target 生效。

### Task 5：接入 store，并验证持久化闭环

Files:

- Modify: `Momento/ContentView.swift`
- Modify: `Momento/Features/Sidebar/MomentoSidebarView.swift`
- Test: `MomentoTests/ImportServiceSmokeTests.swift`

Steps:

- 文件夹 move 仍调用现有 `moveFolder(_:toParentID:relativeTo:insertAfterTarget:)`。
- 图片 drop 仍调用现有 `assignDroppedAssetsToFolder(_:folderID:)`。
- 如果 drop 后目标文件夹是 `.into`，确保目标 folder id 进入 `expandedFolderIDs`。
- 复用已有存储层测试，并补一组业务闭环测试：
  - move folder 后 reopen package，层级和顺序保持。
  - assign asset 后 reopen package，asset.folderIDs 包含目标 folder id。
- 手动验证：
  - 拖文件夹到同级前/后。
  - 拖文件夹到另一个文件夹内部。
  - 拖文件夹悬停展开后放到更深层。
  - 拖图片到文件夹，计数和当前文件夹内容更新。
  - 拖父文件夹到子文件夹时没有任何写入。

## 验证矩阵

| 场景 | 预期 |
| --- | --- |
| Folder A 拖到 Folder B 上边缘 | A 与 B 同 parent，A 排在 B 前 |
| Folder A 拖到 Folder B 下边缘 | A 与 B 同 parent，A 排在 B 后 |
| Folder A 拖到 Folder B 中间 | A.parentID == B.id |
| Parent 拖到 Child 中间 | drop 被拒绝，store 不写入 |
| Child 拖到空白 root drop zone | Child.parentID == nil，排到顶层末尾 |
| 拖到折叠 folder 中间并悬停 | folder 自动展开 |
| 图片单选拖到 folder | asset.folderIDs 包含 folder.id |
| 图片多选拖到 folder | 所有当前库内、非废纸篓选中项都关联 |
| 跨资源库 payload | drop 被拒绝 |
| Finder 文件拖到 folder | 本轮不处理，不调用 assignAssets |

## 推荐验证命令

```bash
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/ArchitectureGuardTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/SidebarFolderDropResolverTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/DragPasteboardWriterTests
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS -only-testing:MomentoTests/LibraryPackagePersistenceTests/testMovingFoldersPersistsManualOrderAndHierarchy -only-testing:MomentoTests/LibraryPackagePersistenceTests/testMovingFolderIntoDescendantIsRejected
xcodebuild build -project Momento.xcodeproj -scheme Momento -destination platform=macOS
git diff --check
```

## 官方文档依据

- SwiftUI `onDrop(of:delegate:)`：drop destination 只对传入的 supported content types 激活，并把行为交给 `DropDelegate`。https://developer.apple.com/documentation/swiftui/view/ondrop%28of%3Adelegate%3A%29
- SwiftUI `DropDelegate.dropUpdated(info:)`：用于返回当前 drop proposal。https://developer.apple.com/documentation/swiftui/dropdelegate/dropupdated%28info%3A%29-2mktz
- Foundation `NSItemProvider.registerDataRepresentation`：用于注册 data-backed representation。https://developer.apple.com/documentation/foundation/nsitemprovider/registerdatarepresentation%28fortypeidentifier%3Avisibility%3Aloadhandler%3A%29
- AppKit `NSPasteboardWriting` / `pasteboardPropertyList(forType:)`：AppKit drag pasteboard 可以直接写入和读取自定义类型数据。https://developer.apple.com/documentation/appkit/nspasteboardwriting/pasteboardpropertylist%28fortype%3A%29
- AppKit `NSPasteboard`：drag pasteboard 是 AppKit 拖拽传输数据的直接接口。https://developer.apple.com/documentation/appkit/nspasteboard/

## 自我 Review

### 是否过度设计

结论：不过度，但必须控制 AppKit receiver 的边界。

- 不新增数据模型、不改 store public contract、不引入依赖，符合最小改动。
- AppKit receiver 是为了解决 AppKit source 到 SwiftUI target 的桥接不稳定，不是为了抽象而抽象。
- resolver 抽取只承接现有 `moveCommand` 逻辑，目的是让 placement 和非法移动可测试；不能扩展成通用树编辑框架。

### 是否有边界风险

风险存在，执行时必须处理：

- 双路径 drop 风险：SwiftUI `.onDrop` 和 AppKit receiver 如果同时成功，可能重复调用 move/assign。实现中必须选一个权威路径；推荐验证 AppKit receiver 后移除 row 上相同类型的 SwiftUI `.onDrop`，不要靠额外缓存或 session token 掩盖重复触发。
- 坐标风险：AppKit receiver 的 `draggingLocation` 需要转换到 row 本地坐标，再复用同一套 placement 阈值。不能用全局窗口坐标直接判断上/中/下。
- 异步风险：payload decode 后回调 SwiftUI/store 必须回到 MainActor。
- 跨库风险：libraryID 不一致必须拒绝，不得 fallback 到当前库。
- 循环嵌套风险：UI resolver 和 storage 层都要拒绝，storage 层是最终防线。

### 是否会和当前逻辑冲突

主要冲突点是现有 SwiftUI `.onDrop`。文档明确要求最终只保留一个权威 drop path，避免双写。其他路径应保持兼容：

- `LibraryStore.moveFolder` 和 `assignAssets` 不改签名。
- `AssetFilePromiseProvider` 继续保留 Finder file promise。
- `expandedFolderIDs` 仍由 sidebar view 持有，不上移到 store。
- 文件夹计数继续由 `sidebarAssetCounts` 派生，不新增缓存。

### 是否存在无意义缓存

没有。该方案不新增拖拽缓存，也不缓存文件夹树。自动展开只用短生命周期 pending state，drop 结束后清理。

### 是否达到执行标准

达到。文档给出了：

- 明确用户可感知行为。
- 当前代码事实和根因方向。
- 具体文件、任务拆分和测试入口。
- 验证矩阵。
- 风险和非目标。

执行时仍需先做一个最小复现验证：确认 folder payload 声明补齐后，文件夹 SwiftUI drop 是否恢复。如果恢复，AppKit receiver 可以先只用于 asset drop；如果没有恢复，则 folder 和 asset drop 都走 AppKit receiver。
