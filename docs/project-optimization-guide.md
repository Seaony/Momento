# Momento 项目优化 Review 指南

日期：2026-05-28

## 目标

这份文档用于 review 当前 Momento 项目的实现质量，重点回答：

- 哪些实现是符合当前产品阶段的合理选择。
- 哪些地方已经出现过度设计、维护成本偏高或“最佳实践”被误用的风险。
- 后续优化应该按什么顺序做，避免把稳定项目改成不可控的大重构。

本轮只做文档审计，不直接修改业务代码。

## 当前结论

Momento 当前主体方向是健康的：本地 `.momento` package、Core Data + SQLite、SwiftUI shell + AppKit `NSCollectionView`、security-scoped bookmark、String Catalog、Sparkle 更新和数据生命周期测试都符合这个项目的目标。

主要问题不是“架构错误”，而是功能快速推进后留下的几个维护压力点：

1. 大文件继续变大，尤其 `LibraryStore`、`ContentView`、`AssetCollectionGridView` 和 `LibraryMetadataStore`。
2. 部分架构护栏测试依赖源码字符串，能防回归但偏脆。
3. 文档里仍有一些旧阶段规划，容易和当前实现状态混在一起。
4. 项目里 tracked 了用户级 Xcode metadata，且当前工作区还有一个未提交的 `project.pbxproj` section-order diff。
5. 性能路线已经开始落地，但还缺真正的 profile / signpost 基线。

## 应保留的设计

### SwiftUI + AppKit 混合边界

`Momento/AppKitBridge/AssetCollectionGridView.swift` 使用 `NSCollectionView` 是合理的。这个路径承担高数量素材渲染、拖拽、file promise、hover/selection 和异步缩略图解码，直接替换成 `LazyVGrid` 会丢失已有 AppKit 能力，也不符合 `AGENTS.md` 的架构约束。

优化方向不是重写，而是继续收窄职责：把纯逻辑、pasteboard payload、layout resolver、preview provider 这类可以独立测试的部分逐步拆出去。

### 本地资源库安全 guard

`LibraryStorage.validateLiveLocalLibraryLocation` 拒绝把 iCloud Drive 中的 `.momento` package 当实时库使用，这不是 iCloud 同步功能，而是数据安全边界。当前仍建议保留，避免用户把 live SQLite package 放到会被系统同步协调的目录中。

### Core Data 作为资源库元数据源

当前 `LibraryMetadataStore` 把 assets、folders、tags、颜色和关系从 Core Data hydrate 成 app state，符合本地素材库的阶段目标。现在不应该为了“最佳实践”提前引入 repository 层、CQRS、SearchIndex、FTS 或多级持久缓存。

### 粗粒度架构护栏测试

`ArchitectureGuardTests` 里有源码字符串测试。它不完美，但目前保护的是 Liquid Glass、窗口 chrome、Sparkle、拖拽 UTI、删除确认等易回归规则。应保留少量高价值护栏，不要扩张成细碎 UI 结构测试。

## 发现的问题

### P1：核心状态对象职责过宽

文件：

- `Momento/Core/LibraryStore.swift`
- `Momento/ContentView.swift`

问题：

`LibraryStore` 同时负责资源库生命周期、recent library、筛选排序、标签/文件夹、导入导出协调、删除、缓存清理、错误消息和选择状态。`ContentView` 又承接大量命令、dialog、toast、toolbar、import/export 和外部 URL 处理。两者都还可维护，但已经接近“单文件吸收所有业务”的边界。

风险：

- 新功能容易继续塞进同一个文件。
- 同一业务规则可能在 View 和 Store 间重复判断。
- 修改选择、筛选、删除、导入时需要读很长上下文，回归成本高。

建议：

短期不要拆大架构。后续遇到真实功能改动时，按这些自然边界小步提取：

- `LibraryStore+Selection` 或独立 selection resolver：只放选择集合、primary selection、prune 规则。
- `LibraryStore+RecentLibraries` 或 `RecentLibraryCoordinator`：只处理 recent library open/rename/delete/move。
- `AssetCommandHandler` 或局部 helper：收敛 `ContentView` 中批量删除、导出、刷新缩略图等命令包装。

禁止：

- 不要一次性把 `LibraryStore` 拆成协议森林。
- 不要为了“单一职责”把简单同步调用拆成多层 service。

### P1：文档状态和代码状态混杂

文件：

- `FEATURE.md`
- `docs/superpowers/plans/2026-05-21-library-preview-settings-localization.md`
- `docs/superpowers/plans/2026-05-23-p0-p1-core-gap-implementation.md`
- `docs/performance-optimization-plan.md`

问题：

文档里同时存在目标规格、历史执行计划、已经实现的决策和未来不做项。例如 `FEATURE.md` 还描述 `ThumbnailRecord`、`SearchIndex`、SVG/PDF/video 等长线能力；旧计划文档中也保留 `cloud sync` 作为非目标。这些不是当前代码问题，但会影响后续 agent 或人工 review 的判断。

建议：

新增一个文档索引，给每份文档标状态：

- `current`: 当前约束或运行手册。
- `historical`: 历史计划，只能作背景。
- `future`: 产品愿景，不代表当前实现。

同时在历史计划顶部加一句“此文档为历史执行计划，当前实现以 README、AGENTS 和最新 review 文档为准”。

### P2：源码字符串测试偏脆

文件：

- `MomentoTests/ArchitectureGuardTests.swift`

问题：

很多断言是 `source.contains(...)`。这类测试能保护架构约定，但也会因等价重构、换行、命名调整而失败。

建议：

保留这些测试的范围，但建立准入规则：

- 只保护平台能力和高风险行为：entitlements、document type、Sparkle、UTI、窗口 chrome、删除确认、拖拽协议。
- 不新增纯视觉结构测试，例如某个 View 名、某个 modifier 顺序、某段源码字符串。
- 对可以行为化的规则，优先改成行为测试或 plist/model 解析测试。

可先处理的项：

- Sparkle 和 UTI 已经用 plist 解析，方向正确。
- 删除确认、command-delete、拖拽音效这类仍大量依赖源码字符串，后续改相关功能时再补行为测试。

### P2：性能优化已有路线，但缺 profile 闭环

文件：

- `docs/performance-optimization-plan.md`
- `MomentoTests/LibraryStorePerformanceTests.swift`
- `Momento/Core/LibraryStore.swift`
- `Momento/AppKitBridge/AssetCollectionGridView.swift`

问题：

已经有 `visibleAssets` benchmark，也有缩略图 decode limiter、bounded cache、masonry layout cache。但还缺实际用户操作路径的 signpost / Instruments 数据。

风险：

后续优化容易回到“看起来会快”的缓存堆叠。

建议：

下一轮性能工作只做 Phase 0：

- 给 `visibleAssets`、`activateLibrary`、`AssetCollectionGridView.updateNSView`、masonry `prepare`、import pipeline 加 Debug-only signpost。
- 建一个固定 10k asset 样本库，记录打开、滚动、搜索、resize、导入的 baseline。
- 更新 `docs/performance-optimization-plan.md`，删除已过期项：导入时图片尺寸和 EXIF 已经共享一次 ImageIO properties 读取。

### P2：Xcode 用户级 metadata 被 tracked

文件：

- `Momento.xcodeproj/xcuserdata/seaony.xcuserdatad/xcschemes/xcschememanagement.plist`
- `.gitignore`

问题：

用户级 Xcode scheme management 文件已经被 tracked，`.gitignore` 目前只忽略 `/build/`、`/dist/`、`/.worktrees/`。这类文件通常不应该进入仓库，除非团队明确依赖它。

建议：

单独做一个小清理：

- 从 git 中移除 `Momento.xcodeproj/xcuserdata/...`。
- `.gitignore` 增加 `*.xcuserstate`、`xcuserdata/`、`.DS_Store`。
- 保留 shared scheme：`Momento.xcodeproj/xcshareddata/xcschemes/Momento.xcscheme`。

### P2：当前工作区存在无关 `project.pbxproj` 重排 diff

文件：

- `Momento.xcodeproj/project.pbxproj`

问题：

当前未提交 diff 只是 PBX section 顺序重排，未改变构建配置或文件引用。它会干扰后续 review 和 commit。

建议：

下次开始功能改动前先处理：

- 如果确认不是用户有意修改，回退这个 diff。
- 如果 Xcode 会持续重排，单独提交一次“normalize project file ordering”，不要混进业务提交。

### P2：导入 pipeline 进度刷新需要继续约束

文件：

- `Momento/Services/AssetImportService.swift`

现状：

导入已在 `Task.detached` 中执行，并且重复导入、hash、缩略图、颜色、EXIF、尺寸等路径基本清楚。当前 `ImageImportProperties` 已避免重复读取 ImageIO properties，这是一个好的改进。

风险：

大量导入时，每个文件都可能触发 progress report、缩略图生成和颜色分析。现在可以接受，但需要 profile 确认 UI 刷新频率和主线程压力。

建议：

- 保留现在的真实错误暴露，不引入 silent fallback。
- 后续只在 profile 证明 progress 更新过频时做节流。
- 节流必须保证最后一次、错误和取消状态立即上报。

### P3：`RecentLibraryStore.load()` 对坏数据直接清空

文件：

- `Momento/Storage/LibraryAccessScope.swift`

问题：

`RecentLibraryStore.load()` 如果 JSON decode 失败，会直接返回空数组。对 UserDefaults 中 recent library registry 来说，这是可接受的简单策略，但用户体验上会表现为最近库全部消失。

建议：

不要马上加复杂迁移层。只有遇到真实坏数据或要继续演进 registry schema 时，再引入 lossy decode 和 cleanup。引入时必须有测试，不能吞掉错误后假装成功。

## 过度设计风险清单

这些事情现在不要做：

- 不要引入通用 Repository/UseCase/Coordinator 分层，只为拆文件而拆。
- 不要做持久 `SearchIndex`、FTS、智能文件夹，除非已有规模和搜索 profile 证明需要。
- 不要引入 `ThumbnailRecord` 或缩略图生成队列，除非现有 deterministic thumbnail path 无法满足 profile。
- 不要把 SwiftUI drop 默认迁移到 AppKit receiver；只有证据证明 SwiftUI payload 桥接不可靠时才加 AppKit fallback。
- 不要为了替代源码字符串测试而引入 UI automation。先用 plist/model/resolver 行为测试替换可替换的部分。
- 不要删除本地 iCloud Drive live-library guard，除非产品明确接受由文件同步目录导致的 SQLite/package 数据风险。

## 最佳实践对照

### 已符合

- SwiftUI shell + AppKit collection bridge 用在正确位置。
- 本地 package 内保存相对路径，避免依赖原始导入路径。
- security-scoped bookmark 生命周期有专门封装。
- Core Data model 保留版本，已有迁移测试覆盖旧库打开。
- 用户可见文案走 `AppLocalization` / String Catalog。
- 删除和导入这类数据生命周期有端到端测试。
- 浏览器导入使用 local-only listener，边界比开放端口更合理。

### 需要持续约束

- `try?` 只应用在清理、测试 teardown、可缺省 metadata 读取等低风险路径；数据写入路径不能扩大使用。
- `Task.detached` 只用于明确后台 IO/CPU 工作，必须在回到 MainActor 前暴露真实错误。
- 缓存必须有 owner、key、容量和 invalidation；不能新增全局永久 cache。
- 文档必须标注状态，避免历史计划被当成当前要求。

## 推荐优化顺序

### 第一阶段：低风险整理

1. 清理或隔离 `project.pbxproj` section-order diff。
2. 移除 tracked Xcode `xcuserdata`，补 `.gitignore`。
3. 给 docs 加状态索引，标记历史计划和当前约束。
4. 更新 `docs/performance-optimization-plan.md` 中已过期的导入性能建议。

### 第二阶段：测量优先

1. 加 Debug-only signpost。
2. 固化 10k asset benchmark 数据。
3. 用 profile 决定是否继续优化 `visibleAssets`、masonry layout、thumbnail decode 或 import progress。

### 第三阶段：按真实改动拆边界

1. 做 recent library 功能时，提取 recent library 边界。
2. 做选择/批量操作时，提取 selection resolver。
3. 做 sidebar 拖拽时，只提取 drop resolver，不改整个 Sidebar 架构。
4. 做 asset grid 性能时，只提取 preview provider / layout / pasteboard 的纯逻辑，不重写 AppKit bridge。

### 第四阶段：大规模能力只在有证据时进入

只有 50k-100k asset profile 证明现有架构无法达标时，才讨论：

- Core Data 热查询索引。
- DB-backed paging。
- FTS / SearchIndex。
- 多尺寸缩略图。
- 分阶段资源库打开。

## Review 决策模板

后续每个优化 PR 先回答：

```text
Problem:
- 具体卡在哪里或维护痛点是什么。

Evidence:
- 代码证据、测试失败、profile 数据或用户复现路径。

Change:
- 本次只改哪些文件，哪些不改。

Non-goals:
- 明确不做哪些看似相关的扩展。

Validation:
- tests:
- build:
- diff check:
- manual QA:

Rollback:
- 回退后是否影响数据格式或用户库。
```

## 本轮未做

- 未运行 App 做人工视觉 QA。
- 未做 Instruments profile。
- 未修改业务代码。
- 未跑完整测试套件；本文档本身只需要 diff/Markdown 层面的验证。
