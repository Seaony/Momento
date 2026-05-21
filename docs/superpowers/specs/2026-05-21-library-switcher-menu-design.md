# Library Switcher Menu Design

## 背景

主页面左侧栏顶部现在使用 SwiftUI 原生 `Menu` 作为资源库入口。这个入口的视觉由系统菜单控制，菜单项包含创建资源库、打开资源库、关闭资源库和最近资源库切换。新的需求是：点击左上角资源库区域后，在其下方展开一个参考截图结构的自定义菜单；菜单项只保留“创建资源库”；视觉必须符合当前 Momento 的 Liquid Glass 风格。

本设计只描述方案，不直接执行实现。实现需在用户 review 并确认后再开始。

## 官方文档依据

- SwiftUI `popover(isPresented:)` 是官方的状态驱动浮层入口，适合需要系统级 dismissal、焦点和窗口边界处理的浮层。
- SwiftUI `Menu` 适合系统菜单语义，但菜单外观和布局主要由系统控制，不适合做截图里的自定义 Liquid Glass 列表结构。
- SwiftUI Liquid Glass 应优先使用原生 `glassEffect`、`GlassEffectContainer` 和 `.buttonStyle(.glass)` / `.glassProminent`，避免用半透明色块模拟玻璃。
- 自定义可点击行需要保留明确的 hover、pointer、accessibility label 和 keyboard dismissal 行为。

## 当前代码落点

主要文件：

- `Momento/Features/Sidebar/MomentoSidebarView.swift`
  - 当前 `libraryMenu` 是原生 `Menu`。
  - 这里最适合替换为自定义资源库触发区和下拉菜单，因为它已经持有 `libraryName`、`recentLibraries`、`onCreateLibrary`、`onOpenLibrary`、`onSwitchLibrary`、`onCloseLibrary`。
- `Momento/DesignSystem/MomentoGlass.swift`
  - 复用 `MomentoGlassBackground`、`MomentoTheme.sidebarIconHoverBackground`、`MomentoTheme.rowRadius`、`MomentoTheme.panelRadius` 等既有 token。
- `Momento/Features/Library/MomentoCreateLibraryDialog.swift`
  - 点击菜单项后继续调用现有 `onCreateLibrary`，打开现有创建资源库弹窗，不改创建流程。
- `MomentoTests/LiquidGlassSourceTests.swift` 或 `MomentoTests/WelcomeViewSourceTests.swift`
  - 当前项目已有源码约束测试。实现阶段可补一个小范围测试，防止资源库入口继续使用原生 `Menu` 或重新带回打开/关闭/最近资源库项。

## 目标行为

1. 点击左侧栏顶部资源库区域，菜单在该区域下方展开。
2. 再次点击资源库区域，菜单收起。
3. 点击菜单里的“创建资源库...”后：
   - 先收起菜单。
   - 调用现有 `onCreateLibrary`。
   - 继续打开现有创建资源库弹窗。
4. 菜单项只保留“创建资源库...”。
5. 不在菜单中显示：
   - 搜索框。
   - 当前资源库路径行。
   - 打开其它资源库。
   - 清除缓存并重新加载。
   - 合并其它资源库。
   - 关闭资源库。
   - 最近资源库列表。
6. 点击菜单外区域或按 Escape 时收起菜单。
7. 左侧栏收起、窗口 resize 或资源库切换时，不留下悬空菜单。

## 视觉设计

菜单参考截图的“结构”，但不照搬 UI。

拟定视觉：

- 菜单宽度：跟随当前左侧栏内容宽度，左右和资源库触发区对齐。
- 菜单位置：资源库触发区下方 8px 左右。
- 菜单背景：`MomentoGlassBackground(glass: .regular, cornerRadius: 14)`，必要时轻微 tint，但不叠不透明底。
- 边框：使用当前 `MomentoTheme.subtleStroke.opacity(0.42)` 的细线，和侧边栏一致。
- 阴影：只用非常轻的深色阴影增强层级，不做厚重弹窗阴影。
- 菜单项：
  - 左侧 SF Symbol：`archivebox` 或 `folder.badge.plus`。
  - 文案：`Create Library...` / `创建资源库...`。
  - 高度约 34-38px，和左侧栏行高接近但稍强一点。
  - Hover 时使用原生 `glassEffect(.regular, in:)` 或现有低透明白色 hover token。
  - Pointer 使用 `.pointerStyle(.link)`。
- 动画：
  - 展开：opacity + 从上方轻微位移，`smooth(duration: 0.16-0.18)`。
  - 收起：同方向反向。
  - 遵守 reduce motion，必要时只做 opacity。

## 方案比较

### 方案 A：保留 SwiftUI `Menu`

做法：删掉多余菜单项，只留下创建资源库，并调整 label。

优点：改动最小；系统自动处理键盘和关闭。

缺点：菜单外观由系统控制，无法稳定做 Liquid Glass 结构；和用户“必须使用 Liquid Glass 风格”的要求冲突。

结论：不推荐。

### 方案 B：SwiftUI `popover`

做法：资源库 header button 控制 `popover(isPresented:)`，popover 内容用自定义 Liquid Glass 视图。

优点：系统负责窗口边界、焦点和外部点击关闭；比手写 overlay 稳定。

缺点：macOS popover 可能带系统箭头、默认外框和额外 margin，视觉不一定像截图中贴在侧边栏下方的菜单；也可能和当前浮动侧边栏的圆角/玻璃层级产生割裂。

结论：可作为备选。如果实现自定义 overlay 的 dismissal 出现明显问题，再退到这个方案。

### 方案 C：侧边栏内自定义 Liquid Glass overlay

做法：替换 `libraryMenu`，使用普通 `Button` 作为资源库触发区；在 `MomentoSidebarView` 内维护 `isLibrarySwitcherMenuPresented`，展开时在 header 下方用 `overlay` / `ZStack` 渲染自定义 `MomentoLibrarySwitcherMenu`。

优点：视觉控制最好；菜单和左侧栏属于同一块 Liquid Glass 体系；能精准控制宽度、间距、圆角、hover、动画和菜单项数量。

缺点：需要自己处理菜单外点击、Escape、窗口 resize/侧边栏收起时的关闭逻辑。

结论：推荐。它最符合“参考结构、UI 自己写、必须 Liquid Glass”的要求，并且当前菜单只有一个动作，手写交互复杂度可控。

## 推荐方案

采用方案 C：在侧边栏内部实现自定义 Liquid Glass 菜单。

核心原则：

- 不引入新依赖。
- 不改创建资源库业务流程，只复用 `onCreateLibrary`。
- 不把菜单状态放进 `LibraryStore`，因为这是纯 UI 展开状态。
- 不改变左侧栏整体布局、拖拽宽度、收起展开按钮和底部按钮行为。
- 不使用系统 `Menu`，避免视觉无法控制。

## 组件结构

### `MomentoSidebarView`

新增局部状态：

- `@State private var isLibrarySwitcherMenuPresented = false`
- `@State private var isLibraryHeaderHovered = false`

调整：

- 将 `libraryMenu` 改名或重写为 `librarySwitcher`.
- Header 使用 `Button`。
- Header 点击时 `withAnimation(.smooth(duration: 0.16))` 切换菜单状态。
- 菜单项点击时先关闭菜单，再调用 `onCreateLibrary()`。
- 在 `onExitCommand` 中关闭菜单。

### `MomentoLibrarySwitcherMenu`

建议新增为 `MomentoSidebarView.swift` 内的 `private struct`，避免为了一个单项菜单过早拆文件。

职责：

- 渲染 Liquid Glass 菜单容器。
- 渲染唯一菜单项“创建资源库...”。
- 管理菜单项 hover。
- 暴露 `onCreateLibrary` 回调。

不负责：

- 创建资源库弹窗状态。
- 打开 Finder。
- 最近资源库数据。
- 资源库切换。

## 关闭行为

实现阶段需要谨慎处理关闭行为，避免引入全局状态：

- 点击 header：toggle。
- 点击菜单项：关闭后执行动作。
- Escape：关闭。
- 侧边栏收起：由于整个 `MomentoSidebarView` 被移除，菜单自然消失。
- 点击侧边栏滚动区：可以在外层内容区添加轻量 tap handler 关闭菜单，但不应拦截原本的侧边栏行点击。
- 点击 app 其它区域：第一版可不做全窗口透明遮罩，避免影响主内容点击。若用户要求全局外点关闭，再把关闭状态提升到 `MomentoShellView` 统一处理。

## 本地化

需要新增或复用文案：

- 推荐新增：`Create Library...` / `创建资源库...`

原因：菜单项后续会打开创建弹窗和路径选择流程，macOS 习惯用省略号表示动作还需要后续输入。现有 `Create Library` 可继续保留给欢迎页按钮和弹窗标题。

## 测试计划

实现阶段建议最小验证：

1. 新增源码约束测试：
   - `MomentoSidebarView` 不再使用原生 `Menu` 包裹资源库入口。
   - 资源库菜单只暴露 `onCreateLibrary` 一个动作。
   - 菜单使用 `MomentoGlassBackground` 或原生 `glassEffect`。
   - 菜单项使用 hover 反馈和 `.pointerStyle(.link)`。
2. 运行：

```bash
git diff --check
ruby -rjson -e 'JSON.parse(File.read("Momento/Localizable.xcstrings"))'
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' -derivedDataPath /tmp/MomentoDerivedData-library-menu test
```

3. 人工视觉确认：
   - 菜单是否出现在资源库 header 下方。
   - 菜单是否看起来属于同一套 Liquid Glass。
   - 点击“创建资源库...”是否打开现有创建弹窗。
   - 左侧栏拖拽宽度和收起/展开是否未回退。

## 风险

- 自定义 overlay 的外部点击关闭不如系统 popover 自动。如果第一版交互不够稳，需要把 menu state 提升到 shell 层或改用 popover。
- 如果菜单宽度完全跟随侧边栏，在最小宽度下文案可能需要截断；实现时需要 `lineLimit(1)`。
- 源码字符串测试只能保护结构约束，不能替代真实 UI 截图验收。

## 待确认点

1. 菜单项文案是否使用带省略号的 `创建资源库...`。
2. 菜单展开时是否覆盖下方侧边栏内容，而不是把内容往下挤。
3. 点击主内容区是否必须立即关闭菜单；如果必须，状态需要提升到 `MomentoShellView`，改动范围会稍大。
