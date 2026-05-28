# Momento 性能深度 Review

日期：2026-05-28
状态：current（A1/A2/B1/B2/B3 已落地；B3 真实 App 人工验证未执行；C 类保留为后续执行建议）

本文件是一次对实际源码的深度性能走查，不是路线规划。`performance-optimization-plan.md` 提供方法论和测量门槛，本文件提供具体卡点和对应改法。两份配合使用：本文给出「改哪里、为什么慢、怎么改」，路线文档给出「改之前先测什么、不要做什么」。

走查范围：`LibraryStore`、`ContentView`、`AssetCollectionGridView`、`LibraryMetadataStore`、`AssetImportService`、`AssetThumbnailService`、`MomentoSidebarView`。

## 核心结论：mutation → 全量重算级联

整个 App 的性能特征由一个事实决定：`LibraryStore.assets: [AssetItem]`（`LibraryStore.swift:10`）是唯一可观察 source of truth，且大量派生数据是「每次访问即重算」的 computed property。

只要任意一个 asset 字段发生改动（收藏、改标题、改标签、移动文件夹、移入/移出废纸篓、导入），都会写 `assets`。写入走两类路径：经 `mergeAssets`（例如收藏 `toggleFavorite` 在 `LibraryStore.swift:768` 调 `mergeAssets`），或直接下标写（如缩略图刷新在 `:831` 的 `assets[index] = asset`、改标签在 `:504`/`:1264` 的 `assets[index].tags = ...`）。任一次写都触发 SwiftUI 重渲染 `ContentView.body`，连锁导致：

1. `store.visibleAssets` 整段重算：scope filter + applyFilters + search filter + 一次 locale-aware 排序（`LibraryStore.swift:77-116`）。
2. `sidebarAssetCounts` 整段重算：对全部 assets 做一次 O(n) 扫描（`ContentView.swift:1228-1269`）。
3. `AssetCollectionGridView.updateNSView` 对 `[AssetItem]` 做一次全量 `!=` 比较（`AssetCollectionGridView.swift:305`），命中变化后再做一次 O(n) 的 `itemUpdateIndexPaths` 扫描（`:412-438`）。

也就是说：在 1 万素材的库里点一次收藏，会触发约 4-5 次对全量 assets 的遍历，外加一次 locale-aware 全量排序。这是当前最主要的性能成本来源，单个 issue 拆开看都不致命，叠加在同一次 UI 更新里才是问题。

因此优化的第一目标不是「让某个函数更快」，而是「切断不必要的全量重算」。

## 已经做对的，不要回头改

避免把下面这些当成待优化项重写：

- `configureScrollView` 已是幂等写入，仅在值变化时改 AppKit 属性（`AssetCollectionGridView.swift:342-382`）。
- 轻量字段变化走原地 cell 刷新，不做 `reloadData`（`applyAssetChanges` / `itemUpdateIndexPaths` / `canUpdateItemInPlace`，`:384-456`）。
- 瀑布流布局缓存全部 frame，可见区用二分查找，滚动 origin 变化不失效（`AssetMasonryCollectionViewLayout`，`:1157-1335`）。
- 缩略图解码已有 bounded 并发（可见 3 / 预取 1）、in-flight 合并、可取消、prefetch（`AssetPreviewDecodeLimiter` + `AssetPreviewImageProvider`，`:1349-1791`）。
- 缩略图缓存有上限（512 张 / 96MB）且 key 随文件 hash/路径失效（`:1449-1461`）。
- 缩略图是预生成的 512px 文件，滚动时不再 downsample 原图（`AssetThumbnailService.swift`）。
- 导入进度已节流到约 10 次/秒，最后一项与跳过项强制上报（`AssetImportProgressReporter`，`AssetImportService.swift:266+`）。
- 图片尺寸与 EXIF 已共享一次 ImageIO 读取（`imageImportProperties`，`AssetImportService.swift:204`）。
- Core Data 关系加载是批量 `IN` 查询，没有 N+1（`folderIDsByAssetID` / `colorsByAssetID` / `tagsByAssetID`，`LibraryMetadataStore.swift:1044-1124`），且 `fetchBatchSize = 200`。
- cell 复用时取消旧的解码 Task 并校验 asset id/mode 一致（`configure` / `prepareForReuse`，`:1997-2080`）。

## 发现的问题（按影响排序）

### P0-1：`visibleAssets` 每次访问全量重算，无 memoization

文件：`LibraryStore.swift:77-116`

`visibleAssets` 是 computed property，每次读取都执行完整的「scope → filter → search → sort」。`libraryBody` 在 `ContentView.swift:203` 已经把它取出来缓存到局部变量再向下传，这一步是对的；但只要 body 因为任何被观察状态变化而重渲染（搜索框输入、hover、inspector 开合、收藏、导入……），这条链就会整段重跑。

10k 素材的现成 benchmark 已存在（`MomentoTests/LibraryStorePerformanceTests.swift`），可直接用来量化。

证据：computed property 没有任何缓存字段；任何触发 body 重算的状态变化都会重新执行整条链。

优化方案：

- 给 `LibraryStore` 增加 `@ObservationIgnored` 的私有缓存字段（缓存 `visibleAssets` 结果 + cache key）。getter 命中 key 直接返回，未命中才重算并写入缓存。
- cache key 至少覆盖：assets 版本号、`sidebarSelection`、`filterState`、normalized `searchQuery`、`sortOption`、`sortDirection`。
- 关键约束：缓存字段必须对 Observation 隐藏（`@ObservationIgnored`），否则 getter 写缓存会再次触发观察刷新，形成循环。getter 不得修改任何可观察业务状态。
- assets 版本号必须由所有 mutation path 显式推进（见 P0-2 的版本号方案），不能依赖数组 identity。
- 必须补 invalidation 测试：搜索、筛选、排序、文件夹/标签/收藏/废纸篓变化、导入后结果都要正确刷新。

这是单点收益最高的改动。

### P0-2：每次 mutation 缺少廉价「版本号」，下游只能靠全量比较感知变化

文件：`LibraryStore.swift:10`（`assets` 定义）、`AssetCollectionGridView.swift:305`

`updateNSView` 通过 `context.coordinator.currentAssets != assets` 判断 assets 是否变化（`:305`）。`[AssetItem]` 的 `!=` 会逐元素比较，每个 `AssetItem` 还含 `tags`、`paletteColors`、`exifMetadata` 等嵌套数组（`AssetModels.swift:207-228`）。这次比较在**每一次** `updateNSView` 都跑，包括只是搜索框聚焦、hover、inspector 开合这种与 assets 无关的重渲染。命中变化后还要再做一次 `itemUpdateIndexPaths` 的 O(n) 扫描（`:412-438`），其中 `canUpdateItemInPlace` 为每个变化项构造一个 mutated 副本再比较（`:440-456`）。

证据：`!=` 是全量逐元素比较；它处在每次 SwiftUI 更新都会走的 `updateNSView` 路径上。

优化方案：

- 在 `LibraryStore` 维护一个单调递增的 `assetsVersion: UInt64`，所有改动 `assets` 的路径统一 +1（写入点完整清单见下方 B1，含收藏经 `mergeAssets`、标签 `:504`/`:1264`、删除 `:907`/`:923`、reload `:1201`/`:1446`、缩略图刷新 `:831` 等）。
- ⚠️ **`assetsVersion` 只用于 P0-1 的 `visibleAssets` cache key**，不能直接拿去给 grid 做跳过判断——grid 收的是 `visibleAssets`，filter/search/sort/sidebar 变化会改 `visibleAssets` 但不改 `assetsVersion`。给 grid 用的是另一个随 `visibleAssets` 重算而 bump 的 `visibleAssetsRevision`（详见 B3）。
- 注意：版本号必须覆盖所有写入路径，漏掉任何一条都会导致 UI 不刷新。**推荐保留现有 `assets` 存储方式**，在现有写入点旁用很小的 version bump helper 明确推进（见 B1）；不要把 `assets` 重新包装成自定义 computed property。

### P0-3（已处理 A2）：`currentLibraryAssets` 曾有双重全量 filter

文件：`LibraryStore.swift:1053-1063`

原问题：

```
currentLibraryAssets   = allCurrentLibraryAssets.filter { !$0.isTrashed }   // O(n) 拷贝
allCurrentLibraryAssets = assets.filter { $0.libraryID == currentLibrary.id } // O(n) 拷贝
```

已处理：`currentLibraryAssets` 已改为一次遍历完成 `libraryID == currentLibrary.id && !isTrashed`，少一次中间数组分配；`allCurrentLibraryAssets` 保留给 trash 分支单独使用。

后续不要继续把 `libraryID` 过滤降级为 debug 断言，除非先确认 import/切库路径不会临时混入他库 asset；这属于行为相关改动，不在 A2 范围内。

### P0-4：排序大量使用 `localizedStandardCompare`

文件：`LibraryStore.swift:1132-1157`

`sortAssets` 对 `.name` 排序用 `displayName.localizedStandardCompare`（locale-aware，单次比较成本远高于普通字符串比较）。更隐蔽的是 tie-breaker `lhs.id.localizedStandardCompare(rhs.id)`（`:1147`）在**所有**排序选项下、只要主比较相等就会触发——`.addedTime`（默认）下时间戳相等的项会落到这条昂贵比较上。O(n log n) 次比较，10k 素材约 13 万次，每次都是 locale 比较。

证据：排序闭包里 `localizedStandardCompare` 同时出现在主比较和 tie-breaker；tie-breaker 与排序选项无关。

优化方案（按真实成本排序）：

- **主成本是 `.name` 排序的主比较器**（每次比较都调 `displayName.localizedStandardCompare`），不是 tie-breaker。`.addedTime`/`.fileSize` 的主比较是 `Date.compare` / 整数比较，本身不贵。所以排序优化只对 `.name` 有意义，且只有 profile 证明 `.name` 排序是热点时才做：预计算每个 asset 的折叠排序键（一次性算好缓存进派生结构），排序时比较预计算键。
- **tie-breaker 的成本被高估了**：`lhs.id.localizedStandardCompare(rhs.id)`（`:1147`）只在主比较 `.orderedSame` 时触发，即主键完全相等的项之间。`.name` 下是同名、`.addedTime` 下是同一时间戳（亚秒精度，罕见），实际触发次数通常很少，不是热点。
- **⚠️ tie-breaker 改 `<` 不是零行为风险**：`id` 是 sha256 hash 字符串，`localizedStandardCompare` 是数字感知比较（"2" < "10"），普通 `<` 是纯字典序。两者对含数字段的 hash 排序结果**可能不同**，会改变并列项的显示先后。这是可观察的行为变化。除非有 profile 证据且接受这点，否则不要为了微小收益改它。

结论：本项**不列入可直接执行清单**，归入需 profiling 判断的 C 类。

### P1-5（已处理 A1）：`mergeAssets` 导入合并曾是 O(n×m)

文件：`LibraryStore.swift:1223-1240`

原问题：循环内对每个 `updatedAssets` 元素执行 `assets.firstIndex(where:)`，导入 m 个素材到 n 个的库时最坏 O(n×m)。

已处理：`mergeAssets` 现在先从当前 `assets` 构建一次 `[AssetItem.ID: Int]` 映射，再 O(m) 更新/追加。映射只记录首次出现的 index，保留原 `firstIndex` 的「命中第一个」语义；追加新 asset 后同步更新映射，保持同一批 `updatedAssets` 内重复 id 的结果一致。

### P1-6：`sidebarAssetCounts` 每次渲染全量扫描

文件：`ContentView.swift:1228-1269`

每次 body 渲染都对 `store.assets` 做一次 O(n) 扫描统计各分类计数。与 P0-2 叠加：一次无关重渲染就可能触发这次全量扫描。

证据：computed property，无缓存，处在 `libraryBody` 渲染路径上（`:215`）。

优化方案：

- 随 P0-1/P0-2 的缓存一起处理：把计数下沉到 `LibraryStore`，按 `assetsVersion` 失效缓存；或与 `visibleAssets` 缓存复用同一次遍历。
- 不要单独再建一套独立缓存，避免多套失效逻辑。

### P1-7：打开资源库同步阻塞主线程，且 `loadFolders()` 重复执行

文件：`LibraryStore.swift:1191-1209`、`LibraryMetadataStore.swift:26-51`、`:1049`

`activateLibrary` 同步调用 `loadAssets()` / `loadFolders()` / `loadTags()`（`performAndWait`），其中 `loadAssets` 要 fetch 全部记录 + 3 次关系批量 fetch + 构造完整 `AssetItem` 数组，全在调用线程（主线程）完成。大库打开期间首屏不可交互/白屏。

另外 `loadFolders()` 被执行两次：`activateLibrary:1194` 一次，`loadAssets` 内部 `folderIDsByAssetID:1049` 又一次。

证据：`activateLibrary` 串行调用三个同步加载；`folderIDsByAssetID` 自己又 `try loadFolders()`。

优化方案：

- 低风险先做：把已加载的 folders 传进 `folderIDsByAssetID`（或缓存本次 load 的 folders），去掉重复的 `loadFolders()`。
- 中等风险（改调用契约，需单独评估）：分阶段打开——先显示 shell/loading，再异步 hydrate assets/folders/tags。这一步会动公开调用时序，按 `performance-optimization-plan.md` 的门槛先有大库 baseline 再做，不改底层存储格式。

### P1-8：导入管线全串行，整批在内存累积后一次性落库

文件：`AssetImportService.swift:123-262`

每个文件顺序执行 hash → copy → 缩略图（ImageIO）→ 调色板（CPU）→ EXIF/尺寸。缩略图和调色板都是 CPU/IO 密集，但彼此不重叠。同时整批结果累积进 `imported` 数组，最后 `saveImportedBatch` 一次性写 Core Data（`LibraryStore.swift:961`）——超大批量导入会有内存峰值和一次很长的 save。

证据：单层 `for candidate in candidates` 串行；`imported.append` 累积；落库在循环外一次完成。

优化方案（证据门槛后再做）：

- 用 bounded `TaskGroup`（如并发 = 核心数的一半，避免抢占滚动解码）并行化每文件的 decode/palette/缩略图工作；hash 去重和最终顺序保持确定性。
- 超大批量考虑分块 save，降低单次 save 时长与内存峰值。
- 保留现有真实错误暴露与进度节流语义，不引入 silent fallback。
- 先用 1000 张导入 profile 确认瓶颈在 per-file CPU 而非磁盘 IO，再决定并发度。

### P2-9：关系 fetch 在「整库加载」时携带冗余 `IN <全部 id>`

文件：`LibraryMetadataStore.swift:1052`、`:1082`、`:1101`

关系批量 fetch 用 `libraryID == %@ AND assetID IN %@`，传入的是 `loadAssets` 取到的**全部** asset id。整库加载时这个 `IN` 覆盖了库里所有行，对 SQLite 是一条很长的 `IN` 查询，相对 `libraryID == %@` 没有筛选收益。

证据：`loadAssets` 把所有 assetID 集合传给三个关系 helper；整库场景下 `IN` 等价于「全部」。

优化方案：当调用方是整库加载时，走 `libraryID == %@` 不带 `IN`；针对子集的 fetch 保留 `IN`。可用一个参数区分「整库」与「子集」两种入口，避免误删子集路径的过滤。

### P2-10：缩略图统一存为 PNG

文件：`AssetThumbnailService.swift:46`

缩略图统一编码为 PNG。PNG 适合带透明的图形，但对照片类内容磁盘占用更大、解码比 JPEG/HEIC 慢，滚动时每个 cell 的解码（`decodedThumbnailImage`，`AssetCollectionGridView.swift:1761-1784`）会更吃 CPU。

证据：`CGImageDestinationCreateWithURL(..., UTType.png.identifier, ...)` 固定 PNG。

优化方案（证据门槛后再做）：对不含 alpha 的图片用 JPEG（或 HEIC）编码缩略图，保留 PNG 给透明素材。先量化滚动解码延迟，确认 PNG 解码确实是热点再改；改动涉及已存缩略图的兼容（key 不变，新导入用新格式，旧文件自然失效重建即可）。

### P2-11：缺测量基建（无 signpost）

文件：全局（`grep os_signpost` 无结果）

目前没有任何 `os_signpost`，只有一个 `visibleAssets` 的 XCTest benchmark。无法把一次滚动/resize 卡顿归因到 layout / decode / filter-sort / Core Data / SwiftUI 更新中的哪一段。

优化方案：按 `performance-optimization-plan.md` 的 Phase 0，在热点路径加 Debug-only `OSSignposter`（`visibleAssets`、`activateLibrary`、`updateNSView`、masonry `prepare`、`importBatch`、`AssetPreviewImageProvider.imageAsync`）。release 行为不变。这是其它优化的前置：先能测，再改。

## 给自动执行 agent 的执行清单

把上面的发现分成三类。**只有 A 类可以盲执行**；B 类可执行但正确性敏感，必须先实现配套测试并全绿，且建议人工复核 Observation 交互；C 类不要自动执行（需要 profiling 或会改公共行为）。所有 file:line 以代码锚点 `old` 片段为准做精确匹配，不按行号盲改。

### A 类：已执行（确定性、行为保持、有真实收益）

下面两项已在本轮落地，保留 old/new 作为变更记录。

**A1 — `mergeAssets` 用 id→index 映射消除 O(n×m)。** 文件 `LibraryStore.swift`（约 :1219-1227）。

old：

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

new：

```swift
    private func mergeAssets(_ updatedAssets: [AssetItem]) {
        guard !updatedAssets.isEmpty else { return }
        var indexByID: [AssetItem.ID: Int] = Dictionary(minimumCapacity: assets.count)
        for (index, asset) in assets.enumerated() where indexByID[asset.id] == nil {
            indexByID[asset.id] = index
        }
        for asset in updatedAssets {
            if let index = indexByID[asset.id] {
                assets[index] = asset
            } else {
                indexByID[asset.id] = assets.count
                assets.append(asset)
            }
        }
    }
```

对唯一 id 数据结果与原实现一致（`id` 是内容 hash + Core Data 唯一约束，正常无重复）。`where indexByID[asset.id] == nil` 让映射只记录首次出现的 index，与原 `firstIndex` 的「命中第一个」语义严格对齐——即使内存里偶发重复 id 也不会出现「原实现更新第一个、新实现更新最后一个」的偏差。查找从 O(n) 降到 O(1)，`assets` 写入次数不变，Observation 行为不变。

**A2 — `currentLibraryAssets` 合并双层 filter，去掉一次中间数组分配。** 文件 `LibraryStore.swift`（约 :1053-1063）。只改 `currentLibraryAssets`，**保留** `allCurrentLibraryAssets`（它被 `visibleAssets` 的 trash 分支 `:105` 单独使用，不能删）。

old：

```swift
    private var currentLibraryAssets: [AssetItem] {
        allCurrentLibraryAssets.filter { !$0.isTrashed }
    }
```

new：

```swift
    private var currentLibraryAssets: [AssetItem] {
        guard let currentLibrary else {
            return []
        }
        return assets.filter { $0.libraryID == currentLibrary.id && !$0.isTrashed }
    }
```

结果一致（同一集合），少分配一个中间数组。注意：这只是微优化，真正的重复计算要靠 B 类的 memoization 解决，不要把 A2 当成主优化。

A 类验证：`xcodebuild test`（跑全套，确认 import/删除/标签/资源库测试全绿）+ `git diff --check`。

### B 类：已执行但正确性敏感（已配测试，仍建议人工复核）

这三项相互关联，是消除「mutation → 全量重算级联」的核心，但都触及 SwiftUI Observation 失效正确性，**实现错会导致 UI 不刷新（脏缓存）**。实现前必须用 Context7 查 Observation 框架（`@ObservationIgnored`、`access(keyPath:)`、`withMutation(keyPath:)`）的当前官方用法，不要凭记忆写。

执行状态：B1/B2/B3 已落地，并补了 `LibraryStoreAssetsVersionTests`、`LibraryStoreVisibleAssetsCacheTests`、`AssetCollectionGridUpdateDecisionTests`。B3 的真实 App 人工验证项（搜索、筛选、排序、切侧栏、收藏/改名/改标签、切库后 grid 立即刷新）仍需用户运行 App 复核。

**B1 — `assetsVersion` 单调版本号（P0-2，B2/B3 的基础）。**

`assets` 的写入点共 11 处（已逐一核实），漏掉任何一处都会让缓存变脏：

```
LibraryStore.swift:
  431  assets = []
  504  assets[index].tags = updatedTags
  731  assets = try metadataStore.loadAssets()
  831  assets[index] = asset
  907  assets.removeAll { $0.id == assetID }
  923  assets.removeAll { deletedIDs.contains($0.id) }
  1201 assets = loadedAssets
  1235 assets[index] = asset        （mergeAssets 内）
  1238 assets.append(asset)         （mergeAssets 内）
  1264 assets[index].tags = transform(...)
  1446 assets = try metadataStore.loadAssets()
```

注意 `:504`、`:1264` 是 `assets[index].tags = ...` 这种**嵌套字段写**，手动在 11 处 +1 极易漏。但**不要**把 `assets` 改成自定义 computed property 包私有数组来统一 bump：

```swift
@ObservationIgnored private var _assets: [AssetItem]
@ObservationIgnored private(set) var assetsVersion: UInt64 = 0

var assets: [AssetItem] {
    get {
        access(keyPath: \.assets)
        return _assets
    }
    set {
        withMutation(keyPath: \.assets) {
            _assets = newValue
            assetsVersion &+= 1
        }
    }
}
```

这个看似集中，但会让 `assets[index] = ...` / `assets[index].tags = ...` / `assets.append(...)` / `assets.removeAll(...)` 走「getter 取整段数组 → mutation → setter 写回整段数组」路径，可能触发大数组 CoW 拷贝，正好把性能热点搬到 mutation 上。

推荐做法：

- 保留 `var assets: [AssetItem]` 作为当前 `@Observable` 管理的可观察状态，不另包 `_assets`。
- 新增 `@ObservationIgnored private(set) var assetsVersion: UInt64 = 0`。
- 新增一个很小的私有 helper：

```swift
private func bumpAssetsVersion() {
    assetsVersion &+= 1
}
```

- 在 11 个写入点旁边显式 bump，但把写入点尽量收敛到现有 helper：`mergeAssets` 内部一批更新只在函数末尾 bump 一次；`assets = ...` reload/close 路径写完后 bump；`removeAll`、缩略图刷新、无 metadataStore 的 tag 写入路径写完后 bump。
- bump 必须只在确实写入后发生；空 `updatedAssets`、空删除集合、guard return 的 no-op 路径不要 bump，避免让 cache 无意义失效。
- 若后续继续改 B 类，优先把这些写入点收敛成更少的语义 helper（例如 `replaceAssets(_:)`、`removeAssets(where:)`），但 helper 只能封装现有写法和 bump，不要重新包装整个数组存储。

这样保留当前数组 mutation 行为，只增加 cache invalidation 信号；漏写风险靠下面的 invalidation 测试兜住，而不是用会引入额外拷贝的 computed property 兜住。

**B2 — `visibleAssets` memoization（P0-1）。** 文件 `LibraryStore.swift:77-116`。加 `@ObservationIgnored` 缓存，key 覆盖所有输入：

```swift
@ObservationIgnored private var visibleAssetsCache: [AssetItem]?
@ObservationIgnored private var visibleAssetsCacheKey: VisibleAssetsCacheKey?

private struct VisibleAssetsCacheKey: Equatable {
    let assetsVersion: UInt64
    let libraryID: AssetLibrary.ID?
    let sidebarSelection: SidebarSelection
    let filterState: AssetFilterState
    let normalizedQuery: String
    let sortOption: AssetSortOption
    let sortDirection: AssetSortDirection
}
```

`visibleAssets` getter 流程：

1. **第一行无条件读一次 `assets` 以注册 Observation 依赖**：`_ = assets`（或 `access(keyPath: \.assets)`）。这一步 O(1)（CoW 引用，不拷贝）。
2. 再组 key（`SidebarSelection`、`AssetFilterState` 均已是 `Equatable`，已确认；key 里的 `assetsVersion` 来自 B1，是 `@ObservationIgnored`，只用于判断缓存是否有效，不承担 Observation 订阅职责）。
3. key 命中则返回 `visibleAssetsCache!`，未命中才跑现有「scope→filter→search→sort」、写回缓存、并 bump `visibleAssetsRevision`（见 B3）。

⚠️ **第 1 步不能省**（Codex review 指出的真实风险）：`assetsVersion` 是 `@ObservationIgnored`，cache-hit 路径若不读 `assets`，这次 body 渲染的 Observation 跟踪集就不含 `assets`；当某个无关可观察状态触发渲染、恰好 cache 命中、且本次渲染没有其它地方读 `assets`（例如无选中项时 `selectedAsset` 提前返回不读 `assets`）时，后续 `assets` mutation 不会触发重渲染 → UI 脏掉。无条件 `access` 消除这个风险。

**getter 不得修改任何被观察状态**，只写 `@ObservationIgnored` 缓存字段。

必过 invalidation 测试（新增到 `LibraryStorePerformanceTests` 或新文件）——每条都要断言"mutation 后 `visibleAssets` 结果正确变化"：导入、删除、移入/移出废纸篓、收藏切换、改标签、移动文件夹、改 `searchQuery`、改 `filterState`、改 `sortOption`/`sortDirection`、切 `sidebarSelection`、切库。任意一条不刷新即说明 B1 漏了写入点或 key 漏了字段。

**B3 — grid 用「visibleAssets 修订号」跳过全量数组比较（P0-2 的消费端） + `sidebarAssetCounts` 缓存（P1-6）。**

⚠️ **不要用 `assetsVersion` 给 grid 做跳过判断**（Codex review 指出的真实 bug）：grid 接收的是 `visibleAssets`（已过滤/排序），而搜索、筛选、排序、切侧栏都会改变 `visibleAssets` 但**不会**改变 `assetsVersion`（后者只在 asset 本身 mutation 时 bump）。若按 `assetsVersion` 跳过，搜索/筛选/排序/切侧栏后 grid 不刷新。必须用一个随 `visibleAssets` 一起变化的修订号。

- 新增 `@ObservationIgnored private(set) var visibleAssetsRevision: UInt64 = 0`，在 **B2 的 `visibleAssets` getter cache-miss（重算）分支里** `visibleAssetsRevision &+= 1`。它在「assets 变了」或「filter/search/sort/sidebar 变了」时都会前进，正好对应 `visibleAssets` 内容可能变化的全部情况。
- grid：`AssetCollectionGridView.swift:305` 现在是 `context.coordinator.currentAssets != assets`（全量逐元素比较）。新增 `visibleAssetsRevision: UInt64` 属性传入 `AssetCollectionGridView`，coordinator 存上次修订号；修订号未变直接跳过（O(1)），变化时再走现有 `applyAssetChanges`（它内部仍会决定原地更新 vs reload，且对「重算但可见集未变」的情况是 no-op，正确但略有一次 O(n) diff，可接受）。在 `ContentView.libraryBody` 先 `let visibleAssets = store.visibleAssets`（已在 :203，会触发重算并 bump 修订号），再把 `store.visibleAssetsRevision` 透传给 grid，保证传下去的是本次最新值。这是新增输入参数（非破坏现有公共契约）。
- `sidebarAssetCounts`（`ContentView.swift:1228-1269`）：只依赖 `assets` + 当前库，把它下沉到 `LibraryStore` 做计算属性，按 `(assetsVersion, libraryID)` 缓存（`@ObservationIgnored`），`ContentView` 改为读 store 的缓存结果。**注意这里用 `assetsVersion` 是对的**（计数只随 asset 集合变化，与 filter/search/sort/sidebar 无关），不要和 grid 的修订号搞混。读取入口同样要确保注册了对 `assets` 的 Observation 依赖（同 B2 第 1 步）。

B 类验证：build + 全套 `xcodebuild test` + 新增 invalidation 测试全绿 + `LibraryStorePerformanceTests` 记录 before/after。**任一 invalidation 测试失败禁止合入。**

⚠️ **store invalidation 单测不足以覆盖 B3**（Codex review 指出）：B3 的刷新 bug 发生在 SwiftUI/AppKit bridge 更新路径——store 的 `visibleAssets` 返回值可能完全正确（单测全绿），但 grid 因修订号判断错误而不应用新数据。因此 B3 必须额外覆盖：

- 可自动化（推荐）：对 grid 的更新决策写逻辑测试——给定「修订号变化 / 未变化」两种输入，断言是否走 `applyAssetChanges`；把「修订号比较 + 是否触发更新」这段决策逻辑抽成可单测的纯函数再测。
- 人工必测项（无法自动化的部分）：搜索、筛选、排序、切侧栏后 grid 列表**立即刷新**；收藏/改名/改标签后对应 cell **立即更新**；切库后 grid 重载。这几项必须由人工在真实 App 验证，agent 不得声称已验证。

### C 类：不要自动执行（需 profiling 或会改公共行为）

这些项保留为「有证据再做」，自动执行 agent 默认**跳过**，仅在用户明确要求时处理：

- **P0-4 `.name` 排序键缓存**：仅 `.name` 排序有意义，且 tie-breaker 改 `<` 会改变并列项显示顺序（见 P0-4 节）。需 profile 证明 `.name` 排序是热点 + 接受行为变化。
- **P1-7 资源库分阶段异步打开**：改公开调用时序，需大库 baseline。（其中「去掉重复 `loadFolders()`」属低风险，但需改 `loadAssets` 内部 folder 传递路径并验证 folder 排序，收益很小，默认也不做。）
- **P1-8 导入并发**：需 1000 张导入 profile 证明瓶颈在 per-file CPU。
- **P2-9 整库关系 fetch 去掉冗余 `IN`**：需区分整库/子集入口，避免误删子集过滤。
- **P2-10 缩略图 JPEG/HEIC**：需滚动解码延迟 profile + 兼容已存缩略图。
- **P2-11 signpost**：测量基建；属于"先能测"，但不改业务逻辑，可在用户要做 C 类其它项前先加。

### 建议顺序

A1、A2、B1、B2、B3 已完成。下一步如继续优化，C 类一律等用户指令 + profiling。B 类是消除主级联的关键，但因为正确性敏感，仍建议人工运行 App 复核搜索、筛选、排序、切侧栏、收藏/改名/改标签与切库后的 grid 刷新。

## 验证基线

每个改动至少：

```sh
git diff --check
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination platform=macOS build
```

涉及 assets/导入/资源库生命周期：

```sh
xcodebuild test -project Momento.xcodeproj -scheme Momento -destination platform=macOS
```

性能项还要用 `LibraryStorePerformanceTests`（10k 数据）记录改动前后耗时，并对缓存类改动补 invalidation 一致性测试。没有 before/after 数据的「看起来更快」不合入。

## 不做项

延续仓库既有约束，本轮不做：

- 不引入持久化 SearchIndex / FTS / 智能文件夹。
- 不引入 `ThumbnailRecord`、多尺寸缩略图队列（除非滚动 decode profile 证明 512px 单尺寸不够）。
- 不做 Core Data 热查询索引 / DB-backed paging（除非 50k-100k profile 证明现架构达不到）。
- 不为派生缓存引入 Repository/Coordinator 分层；缓存只作为 `LibraryStore` 内部 `@ObservationIgnored` 派生结构存在。
- 不用空 catch / silent fallback 掩盖导入或加载失败。
