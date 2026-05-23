# 全项目实现质量 Review 报告

日期：2026-05-23

## 文档依据

本次 review 和注释补充前先查询了 Apple 官方文档。当前项目采用 SwiftUI 为主、AppKit 补充的结构，Apple 对应文档建议通过 `NSViewRepresentable` / `NSViewControllerRepresentable` 把 AppKit 对象包装进 SwiftUI，并通过 coordinator/context 管理两边通信：

- https://developer.apple.com/documentation/swiftui/appkit-integration
- https://developer.apple.com/documentation/SwiftUI/NSViewRepresentable

本次没有引入新框架，也没有改变架构，只按现有 SwiftUI、AppKit、Core Data 边界做源码 review 和中文注释补充。

## 范围

本次覆盖当前仓库的源码、测试、Core Data 模型版本、资源库配置、字符串目录、Xcode 工程配置和现有文档。

已补中文注释的范围：

- 所有 Swift 源文件和测试文件：补充文件职责说明。
- 关键非直观逻辑：补充少量中文行内注释，覆盖可见素材计算顺序、NSCollectionView 原地刷新策略、重复导入时来源页面链接的持久化规则。

没有补注释的范围：

- `*.xcstrings`、`*.plist`、`*.json`、`*.xcdatamodel/contents`、`*.pbxproj` 和图片资源。这些文件格式不适合直接写注释，强行写入会破坏解析或增加工程风险。

## 当前结论

未发现需要立即阻断合并的 P0/P1 新问题。当前分支的核心数据路径已经具备较好的基础保护：

- 资源库包、manifest、assets、database 的目录职责清楚。
- Core Data 模型已拆出 `TagRecord`、`AssetTagRecord`、`AssetFolderMembershipRecord`，并保留旧模型版本用于轻量迁移。
- 素材删除使用 `isTrashed` / `trashedAt` 软删除路径，永久清理再移除文件和记录。
- 导入路径包含内容哈希去重、缩略图生成、颜色分析、EXIF 提取和目录层级映射。
- 浏览器导入现在能把图片 URL 与来源页面 URL 一起传到导入管线，并持久化到素材详情。
- AppKit 素材列表通过 `NSViewRepresentable` 接入 SwiftUI，符合当前项目需要高性能瀑布流和系统拖拽的约束。

## 主要风险和建议

### P2：大型文件需要在后续功能迭代中拆分

这些文件职责仍然清楚，但体量已经偏大：

- `Momento/AppKitBridge/AssetCollectionGridView.swift`
- `Momento/ContentView.swift`
- `Momento/Storage/LibraryMetadataStore.swift`
- `MomentoTests/ImportServiceSmokeTests.swift`
- `Momento/Features/Sidebar/MomentoSidebarView.swift`
- `Momento/Features/Inspector/MomentoInspectorView.swift`

本次没有为了 review 做大规模拆分，因为拆分会触碰布局、选择状态、拖拽和持久化边界。建议后续只在对应功能变更时顺手按真实边界拆，例如把素材列表的 drag/drop、preview、layout、cell 状态分别拆出。

### P2：缩略图读取仍有同步路径

`AssetCollectionGridView` 里仍然存在基于文件 URL 的同步图片读取，并通过缓存缓解重复解码。当前小库可接受，但大资源库滚动时仍可能带来主线程 I/O 或解码压力。

建议后续单独做缩略图管线：

- small / medium / large 多级缩略图。
- 明确 pending / ready / failed 状态。
- 滚动时异步加载和取消。
- 失败占位和缓存失效策略。

### P2：Core Data store 打开仍是同步等待

`MomentoCoreDataStack` 当前用同步等待方式打开 persistent store。现阶段逻辑简单、测试可控，但如果库很大、磁盘很慢或迁移失败，窗口打开路径可能被阻塞。

建议后续只在资源库打开体验需要优化时改成异步打开，并同步调整错误展示和 loading 状态。不要在当前稳定路径上做猜测性重构。

### P3：检查器中存在一个未使用的 file section helper

`Momento/Features/Inspector/MomentoInspectorView.swift` 中 `fileSection(_:)` 当前没有调用。考虑到用户已经明确不希望备注 UI 回到右侧栏，且文件 section 之前也经历过产品调整，本次只记录为死代码候选，不直接删除。后续如果确认文件区不会恢复，可以单独删除。

### P3：源码字符串护栏测试仍偏脆

`MomentoTests/ArchitectureGuardTests.swift` 这类测试能保护 Liquid Glass、窗口 chrome、Info.plist 等容易回归的规则，但它依赖源码字符串。当前符合项目测试策略，后续应优先保留少量粗粒度护栏，避免扩展成大量纯视觉源码测试。

## 本次已做的注释补充

- 为 44 个 Swift 文件补充中文文件职责说明。
- 为 `LibraryStore.visibleAssets` 补充可见素材计算顺序注释，明确侧边栏范围、筛选、搜索、排序的执行顺序。
- 为 `AssetCollectionGridView.applyAssetChanges` 和 `canUpdateItemInPlace` 补充注释，说明为什么允许轻量字段原地刷新，以及何时回退到完整 reload。
- 为 `LibraryMetadataStore.saveImportedBatch` 补充注释，说明重复导入时只补写缺失的来源页面链接，不覆盖用户编辑过的素材信息。

## 未做的事项

- 未给 JSON、plist、xcdatamodel、pbxproj、图片资源写注释，避免破坏文件格式。
- 未启动 App，也未做人工视觉截图验收。
- 未做多格式导入支持。用户已明确当前不需要 SVG/PDF/视频导入。
- 未把备注 UI 放回右侧栏。用户已明确不需要。
- 未拆分大文件，避免把 review 任务变成高风险重构。

## 验证结果

已执行：

```bash
git diff --check
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoReviewCommentsDerivedData -only-testing:MomentoTests/ArchitectureGuardTests -only-testing:MomentoTests/BrowserImportHTTPTests test
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoReviewCommentsDerivedData -only-testing:MomentoTests/LibraryPackagePersistenceTests/testImportedAssetPersistsSourcePageURL test
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoReviewCommentsDerivedData build
```

结果：以上命令均通过。

未运行：

```text
not run: 人工视觉截图验收。原因是本轮没有启动 App，遵守本仓库不主动启动本地 App 的约束。
```
