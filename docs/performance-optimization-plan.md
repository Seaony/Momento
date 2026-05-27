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
- 不因为“可能会快”就新增缓存或索引。任何缓存都必须有明确 owner、key、invalidation、容量边界和验证方式。

## 执行门槛

后续每个性能优化任务都必须先写清楚这 5 件事：

- Hypothesis：当前卡顿具体怀疑发生在哪里，例如 repeated `visibleAssets` sort、masonry layout prepare、thumbnail decode、Core Data fetch。
- Baseline：改动前用同一数据规模和操作路径记录耗时、主线程阻塞、内存或 hit rate。
- Change：只改一个热点路径，不把滚动、导入、Core Data、Shell layout 混在同一个 PR。
- Validation：改动后用同一场景复测，并运行对应测试、build 或 `git diff --check`。
- Stop condition：如果复测没有收益，回退或停止扩展，不继续堆缓存、索引或并发。

Phase 的进入条件也必须明确：

- Phase 0 是必做前置，除非是纯文档或明显幂等的小改动。
- Phase 1 只处理已经有代码证据且风险低的重复计算、重复 IO、重复 UI 更新。
- Phase 2 只有在 Phase 1 后仍能稳定复现瓶颈时进入，并且需要覆盖一致性测试。
- Phase 3 只有 50k-100k asset 的测量数据证明现有架构无法达标时进入。

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

这些高频打点应只在 Debug、diagnostic build 或明确的采样开关下启用。滚动和缩略图路径优先记录 aggregate counter，避免打点本身影响 trace。

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

### 量化验收指标

第一轮 baseline 可以校准下面的目标，但后续优化不能只写“更流畅”，必须能落到同一套指标上。

| 路径 | 初始目标 | 不接受的结果 |
| --- | --- | --- |
| 瀑布流滚动 | 10k asset 下连续滚动没有 Momento 代码造成的重复 >50ms 主线程阻塞 | 优化后 hitch 数量不降反升，或 thumbnail decode 抢占当前可见图片 |
| 窗口和侧栏 resize | resize 期间不重复执行全量 filter/sort；layout prepare 耗时可解释 | 为了降频导致拖拽反馈明显滞后 |
| `visibleAssets` | 10k asset 常用筛选/排序一次计算 p95 目标 <16ms；如果 localized name sort 超出目标，至少必须消除同一输入下的重复计算 | 同一输入在一次 UI 更新中重复计算多次 |
| 搜索输入 | 输入期间可交互，必要时 debounce；结果更新不阻塞连续输入 | 每个字符都触发不可取消的全量重排且造成停顿 |
| 视图切换 | 10k asset 下 masonry/grid/list 切换目标 <200ms，50k asset 记录基线 | 引入 batch update 后选择状态或滚动位置错乱 |
| 批量导入 | progress UI 更新不超过约 10 次/秒，最终状态和错误立即上报 | 为了节流吞掉最后进度、错误或取消状态 |
| 缩略图缓存 | 维持现有 bounded cache 思路，记录 hit rate 和 decode latency | 新增无上限缓存，或缓存 key 无法随文件变化失效 |
| 资源库打开 | 大库打开有阶段化 loading，首屏 shell 先可见 | 为了异步化改变公开调用契约或隐藏加载失败 |

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

1. 收敛 `visibleAssets` 的调用边界
   - `libraryBody` 已经拿到 `visibleAssets` 后，尽量继续向子视图和派生逻辑传递同一份结果。
   - `selectedInspectorAssets` 和批量操作优先使用 id map 或当前 visible snapshot，避免再次读计算属性。
   - 这是缓存前的第一步：如果减少重复访问已经解决卡顿，不继续新增缓存。

2. 仅在测量后添加 `visibleAssets` memoization
   - 只有 baseline 证明 repeated `visibleAssets` filter/sort 是热点时才做。
   - 缓存字段必须对 SwiftUI Observation 隐藏，例如使用 `@ObservationIgnored` 的私有存储，避免 getter 写入触发新的观察刷新。
   - 缓存 key 至少覆盖 current library、assets 版本、sidebar selection、filter state、normalized search query、sort option、sort direction。
   - assets 版本必须由导入、删除、移入废纸篓、恢复、收藏、重命名、移动文件夹、标签更新等 mutation path 明确推进，不能依赖隐式数组 identity。
   - getter 不得改变任何可观察业务状态；缓存 miss 只能更新不可观察缓存。
   - 必须补一组 invalidation 测试，覆盖搜索、筛选、排序、文件夹、标签、收藏、废纸篓和导入后的结果变化。

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
   - 进入条件：Phase 1 后仍能证明候选集合筛选是热点，而不是 thumbnail decode、layout 或 Core Data fetch。
   - 维护 `assetsByID`、`assetIDsByFolderID`、`assetIDsByTagID`、favorite/trash 集合。
   - `assets` 仍是 source of truth，索引只能是派生结构，并且必须能从 `assets` 完整重建。
   - `visibleAssets` 先缩小候选集合，再执行搜索和排序。
   - 更新路径必须和导入、删除、移动、标签、收藏状态保持一致；每条 mutation path 需要测试或断言覆盖。

2. 改善 live resize
   - resize 期间避免触发无意义的 filter/sort。
   - masonry layout 在 live resize 中可使用更粗粒度 invalidation，resize 结束后再做完整 layout 校正。
   - 侧栏宽度拖拽时限制状态提交频率，同时保持拖拽视觉连续。

3. CollectionView batch update
   - 进入条件：trace 证明 `reloadData` 是视图切换、搜索或筛选的主要卡顿来源。
   - 对同一批 asset 的排序/筛选变化，优先考虑 batch update 或 diffable snapshot，而不是盲目 `reloadData`。
   - 先覆盖搜索/筛选/排序三个高频路径，不扩散到所有更新。
   - 必须验证 selection、hover、preview、scroll position 和右侧 Inspector 不出现错位。

4. 缩略图 decode cancellation
   - 当前 in-flight task 可以合并请求，但 cell 复用后旧 decode 仍可能继续跑。
   - 引入可取消的可见区域请求策略，优先保证当前可见 asset 的 decode。

5. 资源库打开分阶段
   - 先显示 shell/loading，再异步加载 assets/folders/tags。
   - 对大库打开显示明确状态，不让主线程长时间不可交互。
   - 不改变底层存储格式，只改变加载节奏。

### Phase 3：只有数据证明需要时再做

这些是大库长期方向，不应作为第一轮优化。

Phase 3 的任何任务都需要单独设计文档。没有 50k-100k asset 的可复现 profile，不进入这个阶段。

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
- 缓存必须有容量上限或生命周期边界；不能新增全局永久缓存。
- 索引必须是派生结构，source of truth 仍然是现有 store / metadata 数据。
- 不改变 `visibleAssets`、导入、资源库打开等现有同步/异步调用契约，除非单独设计迁移方案。
- 并发只做 bounded concurrency，不用 unbounded task group。
- 导入和存储路径不吞错误，不做 silent success。
- 涉及数据生命周期的改动必须跑现有导入、去重、删除、资源库测试。
- UI 纯视觉微调不新增 source-structure 测试，性能路径可以加粗粒度 benchmark。

## 推荐实施顺序

1. 加 signpost 和 `visibleAssets` 性能测试。
2. 减少 ContentView/Inspector 对 `visibleAssets` 的重复访问。
3. 只有 repeated filter/sort 仍是热点时，才加受约束的 `visibleAssets` memoization。
4. 节流导入 progress，合并图片属性读取。
5. 让 `configureScrollView` 和 resize 相关更新更幂等。
6. 根据测量结果调整缩略图 decode cancellation 和并发。
7. 如果 10k-50k 数据仍卡，再做内存索引和 collection batch update。
8. 只有 50k-100k 数据仍无法达标时，再进入 DB paging、SearchIndex、Core Data 索引和多尺寸缩略图。

## 单项执行模板

每个后续性能 PR 或任务说明建议使用这个模板：

```text
Hypothesis:
- <本次只验证一个性能假设>

Baseline:
- dataset: <asset 数量、文件类型、视图模式>
- action: <滚动/resize/搜索/导入/打开资源库>
- measurement: <耗时、主线程阻塞、内存、hit rate>

Change:
- <具体修改文件和逻辑>
- not doing: <明确不做的缓存/索引/重构>

Validation:
- before/after: <同一场景对比>
- tests/build: <实际运行项>
- regression checks: <选择、Inspector、滚动位置、错误处理等>

Rollback:
- <如果收益不足或出现回归，如何回退>
```

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

- 10k asset 下瀑布流连续滚动没有明显卡顿，且 trace 中没有 Momento 代码反复造成 >50ms 主线程阻塞。
- 窗口 resize 和左右侧栏展开收起期间没有可感知冻结，并且不会重复触发全量 filter/sort。
- 搜索和筛选输入不再因为重复全量排序产生明显延迟；如果需要 debounce，必须保证最终结果准确且取消逻辑清晰。
- 批量导入期间 UI progress 平稳更新，导入不会让主窗口交互明显变慢，错误、取消和最终完成状态不能被节流吞掉。
- 大库打开路径有清楚的 loading/阶段化体验，不再长时间白屏或无响应，也不隐藏 Core Data 或文件访问失败。
- 每个性能优化都有 before/after 测量数据支撑，并能被最小验证命令覆盖基础正确性。
- 新增缓存或索引必须通过 invalidation / consistency 检查；没有数据证明收益时不得合入。
