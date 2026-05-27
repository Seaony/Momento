# Momento 性能优化调研与路线

日期：2026-05-27

## 目标

这份文档面向 Momento 的整体流畅度优化，重点覆盖：

- 瀑布流、网格、列表的滚动稳定性。
- 窗口 resize、左右侧栏展开收起、侧栏宽度拖拽时的响应。
- 资源库打开、视图切换、筛选、搜索、选择状态变化时的主线程压力。
- 导入、缩略图、颜色分析、图片元数据读取对 UI 的影响。
- 大资源库下 Core Data 查询、内存占用和状态派生数据的增长风险。

原则是先测量、再做小范围优化；优先消除明确的主线程重复计算和重复 IO，不做没有数据支撑的大重构。

## 非目标

- 不重写 SwiftUI/AppKit 的整体架构。
- 不立刻引入新的数据库层、搜索引擎或外部依赖。
- 不为纯视觉问题新增脆弱的 UI 结构测试。
- 不用无限并发、静默 fallback 或空 catch 掩盖生命周期问题。
- 不提前实现完整的持久化 SearchIndex/FTS，除非后续数据证明内存过滤已成为瓶颈。

## 当前结论

Momento 的主要性能风险不是单一问题，而是几个路径叠加：

1. `LibraryStore.visibleAssets` 是计算属性，每次访问都会按当前资源库、侧栏范围、筛选、搜索和排序重新生成数组。SwiftUI body、选择、Inspector、批量操作等路径会重复访问它，资源数量变大时容易把滚动、resize、搜索输入都拖慢。
2. `AssetCollectionGridView` 已经用了 `NSCollectionView`、自定义 layout cache、二分查找可见元素，以及最近新增的缩略图解码并发限制，这是正确方向。但 asset 结构变化、模式切换和 resize 仍可能触发较重的 reload / layout prepare。
3. 窗口和侧栏 resize 会连续触发 Shell、ContentView 和 CollectionView 的更新。当前 masonry layout 在尺寸变化时需要重新计算全部 item frame，大资源库下 live resize 容易出现卡顿。
4. 导入 pipeline 已在后台任务中执行，但单文件内会分别读取 hash、缩略图、颜色、EXIF、尺寸，并且每个文件都会报告 progress。大量导入时，频繁进度更新和重复图片属性读取会带来 UI 和 IO 压力。
5. Core Data 当前打开资源库时会一次性加载全部 Asset、Folder、Tag、关系和颜色数据。小中型库体验简单稳定，但在 10k-100k 资产目标下，需要先建立性能基线，再考虑分阶段加载和索引优化。

## 代码证据

- `Momento/Core/LibraryStore.swift`
  - `visibleAssets` 每次访问都会执行 scope、filter、search、sort。
  - `primarySelectedAssetID`、`pruneSelectedAssetsIfNeeded` 等选择路径也会重新读 `visibleAssets`。
  - `activateLibrary` 打开资源库后同步装载 assets、folders、tags。
- `Momento/ContentView.swift`
  - `libraryBody` 中先读取 `store.visibleAssets`，同时其他 derived state 也会再次间接读取。
  - `selectedInspectorAssets`、批量复制、批量标签/文件夹操作等路径会从可见资产中再过滤选中项。
- `Momento/AppKitBridge/AssetCollectionGridView.swift`
  - `updateNSView` 每次 SwiftUI 更新都会调用 `configureScrollView`。
  - 资源结构变化、模式切换、语言变化会走 reload 或 layout invalidation。
  - masonry layout 对滚动 origin 变化做了避免失效处理，但尺寸变化仍需要重新 prepare 全部 frame。
  - 缩略图已有 `AssetPreviewDecodeLimiter` 和 in-flight coalescing，不过仍需要度量取消、命中率和并发数。
- `Momento/Storage/LibraryMetadataStore.swift`
  - `loadAssets()` 一次性 fetch 当前库所有 asset，再按所有 assetID 拉取 folder、color、tag 映射。
  - Core Data model 主要依赖唯一约束，热查询上的显式索引需要结合实际 fetch profile 再加。
- `Momento/Services/AssetImportService.swift`
  - import 在后台任务执行，但每个文件都会 hash、copy、thumbnail、palette、EXIF、dimensions。
  - `imageExifMetadata` 和 `imageDimensions` 分别创建 image source 并读取 properties，可以合并。
  - progress callback 按文件触发，批量导入时可能导致 UI 高频刷新。

## 测量基线

优化前先加轻量测量，避免靠感觉改代码。

### Instruments 场景

- Time Profiler：滚动、resize、搜索输入、视图切换、导入 100/1000 张图片。
- Core Animation：观察滚动和 resize 期间 frame hitch。
- Allocations / Memory Graph：缩略图 cache、导入、快速切换资源库时的内存增长。
- File Activity：导入和资源库打开期间的图片读取、Core Data 读取。

### 建议加 signpost 的位置

- `LibraryStore.visibleAssets`：记录 asset count、filter/search/sort 耗时。
- `LibraryStore.activateLibrary`：分段记录 loadAssets/loadFolders/loadTags。
- `LibraryMetadataStore.loadAssets`：记录 Core Data fetch、relationship mapping、value object mapping。
- `AssetCollectionGridView.updateNSView`：记录 reload、in-place update、layout invalidation。
- `MomentoMasonryLayout.prepare`：记录 item count、column count、live resize 状态。
- `AssetImportService.importBatch`：记录 enumerate/hash/copy/thumbnail/color/metadata/save。
- `AssetPreviewImageProvider.image`：记录 cache hit、in-flight hit、decode latency、cancellation。

### 手动验证矩阵

| 场景 | 数据规模 | 关注指标 |
| --- | --- | --- |
| 打开资源库 | 1k / 10k / 50k asset | 首屏可操作时间、主线程最长阻塞 |
| 瀑布流滚动 | 10k / 50k asset | hitch 次数、主线程 >50ms 阻塞 |
| 窗口 resize | 10k / 50k asset | live resize 是否连续、layout prepare 耗时 |
| 左右侧栏展开收起 | 10k asset | 动画期间是否触发重复 filter/sort |
| 视图切换 | masonry/grid/list | reload 耗时、滚动位置是否稳定 |
| 搜索输入 | 10 字连续输入 | 每次输入响应时间、visibleAssets 耗时 |
| 批量导入 | 100 / 1000 images | UI progress 刷新频率、后台 CPU、主线程阻塞 |
| Inspector 切换 | 单选/多选 100 个 | 选择响应时间、重复 visibleAssets 次数 |

## 优化路线

### Phase 0：先建立可重复测量

目标：1 天内拿到能比较优化前后的数据。

- 增加 `os_signpost` 或统一轻量 logger，覆盖上面的热点路径。
- 为 `LibraryStore.visibleAssets` 增加 deterministic performance test，用合成资产覆盖 1k、10k、50k 的过滤、搜索、排序。
- 建一个本地大资源库测试流程，记录数据规模、图片尺寸、视图模式和机器环境。
- 保留 `git diff --check`、相关测试、build 作为每轮修改的最低验证。

判断标准：

- 能回答一次滚动卡顿到底花在 layout、decode、filter/sort、Core Data 还是 SwiftUI 更新。
- 能比较每个小改动前后的耗时差异。

### Phase 1：低风险、短路径优化

这些改动不改变外部行为，也不需要大重构，建议优先做。

1. 缓存 `visibleAssets` 的派生结果
   - 给 assets/filter/search/sort/sidebar scope 增加版本输入。
   - 同一轮状态不变时返回缓存数组，避免 body、Inspector、选择逻辑重复 filter/sort。
   - 先在 `LibraryStore` 内完成，不改变调用方契约。

2. 减少 ContentView 中重复可见资产访问
   - `libraryBody` 已经拿到 `visibleAssets` 后，尽量继续向子视图和派生逻辑传递同一份结果。
   - `selectedInspectorAssets` 和批量操作优先使用 id map 或当前 visible snapshot，避免再次读计算属性。

3. 让 `configureScrollView` 幂等
   - 只在值变化时写 scroller/clipView/display 相关属性。
   - 避免每次 SwiftUI update 都触发 AppKit 属性写入。

4. 约束导入 progress 刷新频率
   - 按文件计数和时间间隔合并进度，例如最多每 100ms 或每 N 个文件更新一次。
   - 最后一项和错误状态必须立即上报，不能吞掉失败。

5. 合并图片属性读取
   - `imageExifMetadata` 和 `imageDimensions` 共享一次 `CGImageSourceCopyPropertiesAtIndex` 结果。
   - 保留现有错误处理语义，不增加静默 fallback。

6. 补充缩略图 cache 指标
   - 记录 cache hit、in-flight hit、decode latency、decode cancellation。
   - 在有数据后再决定并发数从 2 调到 3，避免凭感觉加并发。

预期收益：

- 搜索、选择、侧栏切换、resize 期间减少重复数组构建。
- 批量导入时 UI 更新更平稳。
- 缩略图优化能从“感觉流畅”变成可度量。

### Phase 2：中等改动，围绕已验证瓶颈

这些改动需要更多测试，但仍应保持在现有架构内。

1. 建立资源内存索引
   - 维护 `assetsByID`、`assetIDsByFolderID`、`assetIDsByTagID`、favorite/trash 集合。
   - `visibleAssets` 先缩小候选集合，再执行搜索和排序。
   - 更新路径必须和导入、删除、移动、标签、收藏状态保持一致。

2. 改善 live resize
   - resize 期间避免触发无意义的 filter/sort。
   - masonry layout 在 live resize 中可使用更粗粒度 invalidation，resize 结束后再做完整 layout 校正。
   - 侧栏宽度拖拽时限制状态提交频率，同时保持拖拽视觉连续。

3. CollectionView batch update
   - 对同一批 asset 的排序/筛选变化，优先考虑 batch update 或 diffable snapshot，而不是盲目 `reloadData`。
   - 先覆盖搜索/筛选/排序三个高频路径，不扩散到所有更新。

4. 缩略图 decode cancellation
   - 当前 in-flight task 可以合并请求，但 cell 复用后旧 decode 仍可能继续跑。
   - 引入可取消的可见区域请求策略，优先保证当前可见 asset 的 decode。

5. 资源库打开分阶段
   - 先显示 shell/loading，再异步加载 assets/folders/tags。
   - 对大库打开显示明确状态，不让主线程长时间不可交互。
   - 不改变底层存储格式，只改变加载节奏。

### Phase 3：只有数据证明需要时再做

这些是大库长期方向，不应作为第一轮优化。

1. Core Data 热查询索引
   - 根据 Instruments 和 SQLite query profile 添加索引。
   - 候选字段包括 `libraryID/isTrashed/importedAt`、`libraryID/id`、folder membership、tag membership、color sortIndex、folder parent/sortIndex。
   - 每个索引都要有对应查询证据，避免写入变慢和模型复杂度上升。

2. DB-backed paging
   - 当 50k-100k asset 下内存过滤和全量 hydrate 成为明确瓶颈，再考虑分页查询。
   - 优先只用于资源列表，不直接改所有 store API。

3. SearchIndex / FTS
   - 当名字、标签、描述、未来 OCR/EXIF 搜索都进入高频场景时再启用。
   - 第一版可以继续用内存搜索，避免索引一致性 bug。

4. 多尺寸缩略图
   - 当前 512px thumbnail 简单稳定。
   - 只有在滚动 decode、内存、磁盘 IO 数据证明 512px 不合适时，再引入 small/medium/large 或 `ThumbnailRecord`。

## 风险控制

- 每轮只优化一个热点路径，避免滚动、导入、Core Data、Shell layout 同时变化。
- 所有性能 cache 必须有清晰 invalidation 输入，不用“看起来能用”的隐式状态。
- 并发只做 bounded concurrency，不用 unbounded task group。
- 导入和存储路径不吞错误，不做 silent success。
- 涉及数据生命周期的改动必须跑现有导入、去重、删除、资源库测试。
- UI 纯视觉微调不新增 source-structure 测试，性能路径可以加粗粒度 benchmark。

## 推荐实施顺序

1. 加 signpost 和 `visibleAssets` 性能测试。
2. 缓存 `visibleAssets`，并减少 ContentView/Inspector 重复访问。
3. 节流导入 progress，合并图片属性读取。
4. 让 `configureScrollView` 和 resize 相关更新更幂等。
5. 根据测量结果调整缩略图 decode cancellation 和并发。
6. 如果 10k-50k 数据仍卡，再做内存索引和 collection batch update。
7. 只有 50k-100k 数据仍无法达标时，再进入 DB paging、SearchIndex、Core Data 索引和多尺寸缩略图。

## 验证命令

每轮实际改代码后至少执行：

```sh
git diff --check
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination platform=macOS build
```

涉及导入、存储、资源库生命周期时执行：

```sh
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS
```

性能专项验证不应只看测试通过，还需要保留 Instruments trace 或手动记录：

- 数据规模。
- 操作路径。
- 主线程最长阻塞。
- 滚动/resize hitch 数量。
- 峰值内存。
- 优化前后对比。

## 成功标准

- 10k asset 下瀑布流连续滚动没有明显卡顿，主线程长阻塞显著减少。
- 窗口 resize 和左右侧栏展开收起期间没有可感知冻结。
- 搜索和筛选输入不再因为重复全量排序产生明显延迟。
- 批量导入期间 UI progress 平稳更新，导入不会让主窗口交互明显变慢。
- 大库打开路径有清楚的 loading/阶段化体验，不再长时间白屏或无响应。
- 每个性能优化都有测量数据支撑，并能被最小验证命令覆盖基础正确性。
