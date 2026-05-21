# 全项目实现质量 Review 报告

日期：2026-05-21

## 范围

本次 review 覆盖仓库中所有已跟踪文件，包括应用源码、测试、资源目录配置、Xcode 工程配置和文档。SwiftUI、AppKit、Core Data、文件系统持久化、窗口行为和 Liquid Glass 相关实现已先查阅 Apple 官方文档，再按当前项目架构判断。

## 逐文件结果摘要

| 文件 | 结论 | 说明 |
| --- | --- | --- |
| `CLAUDE.md` | 正常 | 当前仓库协作规则清晰；其中“不自动 commit”已被用户后续明确要求覆盖。 |
| `FEATURE.md` | 正常 | 产品方向文档，无代码风险。 |
| `Momento.xcodeproj/project.pbxproj` | 正常 | 工程结构和测试目标配置可构建；未做无关重排。 |
| `Momento.xcodeproj/project.xcworkspace/contents.xcworkspacedata` | 正常 | 工作区配置有效。 |
| `Momento.xcodeproj/xcshareddata/xcschemes/Momento.xcscheme` | 正常 | Scheme XML 有效。 |
| `Momento.xcodeproj/xcuserdata/seaony.xcuserdatad/xcschemes/xcschememanagement.plist` | 正常 | 用户级 Xcode 状态文件，保留现状。 |
| `Momento/AppKitBridge/AssetCollectionGridView.swift` | 需后续优化 | 当前职责清楚，但 cell 同步加载图片缩略图，不适合未来大资源库规模。 |
| `Momento/AppKitBridge/QuickLookPreviewController.swift` | 正常 | AppKit preview bridge 边界清晰。 |
| `Momento/AppKitBridge/SidebarTitlebarToggleConfigurator.swift` | 已补注释 | 使用 `NSTitlebarAccessoryViewController` 是为了进入系统标题栏坐标和命中区域，已补充中文约束说明。 |
| `Momento/AppKitBridge/WindowTransparencyConfigurator.swift` | 已补注释 | 窗口透明度集中在 app shell 管理，已补充中文不变量说明。 |
| `Momento/AppOpenHandler.swift` | 正常 | 文件打开入口简单，错误继续交给 store 暴露。 |
| `Momento/Assets.xcassets/AccentColor.colorset/Contents.json` | 正常 | JSON 有效。 |
| `Momento/Assets.xcassets/AppIcon.appiconset/Contents.json` | 正常 | JSON 有效，图标槽位完整。 |
| `Momento/Assets.xcassets/AppIcon.appiconset/*.png` | 正常 | 应用图标资源被 asset catalog 引用。 |
| `Momento/Assets.xcassets/Contents.json` | 正常 | JSON 有效。 |
| `Momento/ContentView.swift` | 正常 | 顶层状态注入和窗口校验入口清晰。 |
| `Momento/Core/AppLanguage.swift` | 正常 | 语言枚举职责单一。 |
| `Momento/Core/AppLocalization.swift` | 已修改 | 补齐新增存储错误的本地化入口。 |
| `Momento/Core/AssetModels.swift` | 正常 | 值类型模型边界清晰。 |
| `Momento/Core/LibraryStore.swift` | 正常 | 当前状态流集中；资源库缺失检测已有行为测试覆盖。 |
| `Momento/Core/MomentoLibraryType.swift` | 正常 | 类型判断职责单一。 |
| `Momento/DesignSystem/MomentoGlass.swift` | 正常 | Liquid Glass token 集中，符合当前视觉迭代模式。 |
| `Momento/Features/CommandPalette/MomentoCommandPalette.swift` | 正常 | UI 状态局部化，无明显数据副作用。 |
| `Momento/Features/Inspector/MomentoInspectorView.swift` | 需产品确认 | notes/tags 当前偏展示和局部交互，持久化模型需要后续产品确认。 |
| `Momento/Features/Library/MomentoCreateLibraryDialog.swift` | 正常 | 弹窗状态和创建动作边界清晰。 |
| `Momento/Features/Library/MomentoLibraryWelcomeView.swift` | 正常 | 欢迎页职责明确。 |
| `Momento/Features/Settings/MomentoSettingsView.swift` | 正常 | 设置项状态来源清楚。 |
| `Momento/Features/Shell/MomentoShellView.swift` | 正常 | 左侧栏、内容区、检查器布局状态集中。 |
| `Momento/Features/Sidebar/MomentoSidebarView.swift` | 正常 | 侧边栏 UI 与 store 数据绑定清楚。 |
| `Momento/Info.plist` | 正常 | XML 有效。 |
| `Momento/Localizable.xcstrings` | 已修改 | 补齐缺失键，删除空的过期键，新增覆盖测试。 |
| `Momento/MomentoApp.swift` | 正常 | App 入口、commands 和 scene 配置职责清楚。 |
| `Momento/Services/AssetImportService.swift` | 已补注释 | 补充 security-scoped access 和 hash 去重的中文说明。 |
| `Momento/Storage/LibraryAccessScope.swift` | 已补注释 | 补充资源库安全作用域生命周期说明。 |
| `Momento/Storage/LibraryManifest.swift` | 正常 | manifest schema 边界清晰。 |
| `Momento/Storage/LibraryMetadataStore.swift` | 已补注释 | 补充 Core Data 后台上下文与相对路径持久化说明。 |
| `Momento/Storage/LibraryStorage.swift` | 已修改 | 修复创建资源库可能覆盖已有 `.momento` 包的问题。 |
| `Momento/Storage/MomentoCoreDataStack.swift` | 需后续评估 | 当前同步加载对现阶段可接受；大型库打开路径需要后续评估异步化。 |
| `Momento/Storage/MomentoModel.xcdatamodeld/MomentoModel.xcdatamodel/contents` | 正常 | XML 有效；当前实体覆盖已实现的素材元数据。 |
| `MomentoTests/ImportServiceSmokeTests.swift` | 已修改 | 新增资源库重复创建回归测试。 |
| `MomentoTests/LiquidGlassSourceTests.swift` | 需后续优化 | 能保护当前架构约束，但源码字符串测试偏脆，需要逐步补行为测试。 |
| `MomentoTests/LocalizationCatalogTests.swift` | 新增 | 覆盖源码中使用的本地化键，防止漏翻译或空翻译。 |
| `MomentoTests/WelcomeViewSourceTests.swift` | 需后续优化 | 源码约束测试有效但脆弱。 |
| `MomentoTests/WindowChromeSourceTests.swift` | 需后续优化 | 能锁定窗口 chrome 约束，但仍属于源码字符串测试。 |
| `docs/full-project-review-prompt.md` | 正常 | 本次 review 的执行依据。 |
| `docs/superpowers/plans/2026-05-21-library-preview-settings-localization.md` | 需后续清理 | 计划文档存在历史状态和旧扩展名描述，建议单独归档或更新。 |

## 发现的问题

### P1 已修复：创建资源库可能覆盖已有包

文件：`Momento/Storage/LibraryStorage.swift`

`FileManager.createDirectory(at:withIntermediateDirectories:)` 在目录已存在时不会失败。原实现会继续生成新的 `AssetLibrary` 并写入 manifest，导致用户选择已有 `.momento` 包时覆盖 manifest，同时保留旧 database/assets，形成元数据和实际文件混杂的持久化风险。

修复：创建前显式检查目标路径是否存在，存在时抛出 `LibraryStorageError.libraryPackageAlreadyExists`，并通过现有本地化链路展示错误。

### P2 已修复：本地化键缺失且没有自动检测

文件：`Momento/Localizable.xcstrings`、`MomentoTests/LocalizationCatalogTests.swift`

源码中使用了 `Settings` 和 `Help Center`，但字符串目录缺失对应翻译；同时字符串目录里残留空的 `Grid bridge placeholder` 和 `Momento`。缺少自动覆盖会让新增文案静默回退。

修复：补齐缺失翻译、删除空的过期键，并新增测试扫描 `localization.string(...)` / `localization.format(...)` 使用的 key，要求 `en` 和 `zh-Hans` 都存在且非空。

### P2 需后续处理：缩略图加载仍是同步路径

文件：`Momento/AppKitBridge/AssetCollectionGridView.swift`

当前 cell 配置中同步读取图片，现阶段小规模可用，但与产品文档里“大型资源库”目标不匹配。稳定方案应是独立缩略图生成与缓存管线，避免滚动时主线程 I/O 和解码压力。本次未改，因为这会扩大到导入、缓存、失效和 UI 占位策略。

### P3 需产品确认：检查器 notes/tags 的持久化边界

文件：`Momento/Features/Inspector/MomentoInspectorView.swift`、`Momento/Storage/MomentoModel.xcdatamodeld/MomentoModel.xcdatamodel/contents`

检查器里 notes/tags 的最终数据模型需要产品确认。直接在当前 JSON 或 Core Data 上临时塞字段会破坏后续标签管理、搜索、迁移和多资源一致性。本次只记录风险，不做猜测性 schema 变更。

### P3 需后续优化：源码字符串测试偏脆

文件：`MomentoTests/LiquidGlassSourceTests.swift`、`MomentoTests/WelcomeViewSourceTests.swift`、`MomentoTests/WindowChromeSourceTests.swift`

这些测试能保护窗口和 Liquid Glass 约束，但依赖源码字符串，重构时容易误报。建议随着关键行为稳定后补充更接近真实运行的单元测试或 UI 层验证。

## 已完成修改

- `LibraryStorage.createLibraryPackage(at:name:)` 拒绝已有 `.momento` 包，避免覆盖已有资源库 manifest。
- 新增 `LibraryStorageError.libraryPackageAlreadyExists` 及对应本地化映射。
- 补齐 `A library already exists at the selected location.`、`Settings`、`Help Center` 的英中翻译。
- 删除字符串目录里空的过期键 `Grid bridge placeholder` 和 `Momento`。
- 新增 `LocalizationCatalogTests`，自动检查源码使用的本地化键和英中翻译完整性。
- 新增资源库重复创建回归测试，并验证原 manifest 不被覆盖。
- 为 AppKit 窗口桥接、透明 backing、security-scoped access、Core Data 持久化边界补充中文注释。

## 删除的无用代码或资源

- 删除 `Localizable.xcstrings` 中两个空的过期字符串键：`Grid bridge placeholder`、`Momento`。
- 未删除任何仍可能被运行时引用的 Swift 文件、测试文件、工程配置或图片资源。

## 中文注释范围和原则

本次没有给所有显而易见的属性和简单 SwiftUI body 逐行加注释，避免注释噪音。补充集中在需要解释“为什么”的位置：

- AppKit titlebar accessory 为什么必须用系统标题栏坐标。
- 窗口透明 backing 为什么集中在 shell 层控制。
- security-scoped resource access 为什么需要覆盖异步导入任务生命周期。
- Core Data 存储层为什么以值类型跨出后台 context。
- 相对路径为什么是资源库迁移和重新打开的持久化边界。

## 仍需产品确认或人工视觉确认

- 检查器 tags/notes 是否进入当前版本的持久化范围，以及标签是否需要独立实体、颜色、层级和搜索索引。
- 左侧栏、检查器、欢迎弹窗等 Liquid Glass 视觉细节仍需要人工视觉确认；本次 review 未启动 app 做截图验收。
- 大资源库缩略图策略需要单独设计：生成时机、缓存层级、失败占位、滚动性能和迁移策略。
- 历史计划文档是否需要归档或重写，避免旧 `.momentolibrary` 描述误导后续开发。

## 验证

已运行：

```bash
git diff --check
ruby -rjson -e 'JSON.parse(File.read("Momento/Localizable.xcstrings")); puts "Localizable.xcstrings: OK"'
xmllint --noout Momento.xcodeproj/xcshareddata/xcschemes/Momento.xcscheme Momento/Storage/MomentoModel.xcdatamodeld/MomentoModel.xcdatamodel/contents Momento/Info.plist
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-full-review -only-testing:MomentoTests/LocalizationCatalogTests test
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-full-review -only-testing:MomentoTests/LibraryPackagePersistenceTests/testLibraryPackageCreationRefusesExistingPackage test
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-full-review test
```

结果：以上命令均通过。

未运行：

```text
not run: 人工视觉截图验收。原因是本次 review 未主动启动 app，只做代码、配置和测试验证。
```
