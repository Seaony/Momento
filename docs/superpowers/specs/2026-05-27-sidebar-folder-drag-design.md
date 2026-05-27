# 侧边栏文件夹拖拽：嵌套与排序顺畅化

修订日期：2026-05-27
范围：`Momento/Features/Sidebar/MomentoSidebarView.swift`

## 1. 背景

侧边栏的「文件夹」分组已经能拖动，但有两个用户报告的问题：

1. **无法把文件夹嵌套到另一个文件夹内**。即使把行拖到目标的正中央，松手后基本只会变成同级排序，不会变成子文件夹。
2. **同级排序时视觉抖动**。在两行之间挪动鼠标，落点指示在 `before / into / after` 之间反复跳变，并伴随每次 120ms 动画，给人"卡顿、犹豫"的观感。

## 2. 当前实现的根因

代码现状（行号基于 master `fa5bbc7`）：

- 每行 30pt 高。`MomentoSidebarFolderDropPlacementResolver.placement` 把 y ∈ [0,8] 归为 `.before`、[22,30] 归为 `.after`、中间 14pt 归为 `.into`。
- 但 `MomentoSidebarFolderDropPlacementResolver.effectivePlacement`（1076–1100 行）在 `rawPlacement == .into` 时加了一条额外规则：**如果被拖文件夹和目标是同一个 parent 的兄弟，且 `prefersNesting` 为 false，就把 `.into` 改写成 `.before` 或 `.after`**。
- `prefersNesting` 只在 `springLoadedFolderDropID == row.folder.id` 时为 true。这个状态由 `scheduleFolderExpansionAfterDropHover` 在悬停 ≈0.4s 后设置——但它原本是用来"自动展开子项"的。结果两件事被绑死：用户必须先悬停足够久让目标展开，才能解锁嵌套。这个隐藏 gating 是嵌套 bug 的全部原因。
- `MomentoSidebarFolderDropDelegate.updateTarget`（1280 行）每次 `dropUpdated` 都用 `withAnimation(.smooth(duration: 0.12)) { targetedFolderDrop = ... }`。鼠标在边界附近移动时，每次 placement 改变都重启动画，造成视觉抖动。
- `folderInsertionDropTarget`（463–481 行，提交 fa5bbc7 引入）在每行上方叠了一条 4pt 的 `.before` 专用 drop strip。它是为绕开"中段贪婪吞掉 `.before`"加的补丁；当中段恢复诚实后，这条带变成第三层重叠 drop target，没有功能性贡献。

## 3. 设计

采用 Eagle / Notion 风格：**中段 = 嵌套，上下两条窄边 = 排序，无 spring-load 解锁要求**。

### 3.1 嵌套规则解耦

简化 `effectivePlacement` 为恒等函数（或删除并让调用方直接用 `placement`），仅依赖 `MomentoSidebarFolderDropResolver.moveCommand` 内已有的循环引用保护（不允许把祖先拖入自身或其后代）。

- 同步删除 `prefersNesting` 参数、`springLoadedFolderDropID` 在 placement 决策路径上的所有引用。
- `springLoadedFolderDropID` 状态本身**保留**，继续驱动 `expandFolderAfterDrop` 在悬停 ≈0.4s 后自动展开目标的子项，让多层下钻仍然顺畅。它不再影响"能否嵌套"。

### 3.2 拖拽过程不动画 placement 切换

`MomentoSidebarFolderDropDelegate.updateTarget` 中把 `withAnimation(.smooth(duration: 0.12)) { targetedFolderDrop = … }` 改为直接赋值。

- 实现做法：在 `folderDropIndicator` 自身用 `.animation(.smooth(duration: 0.12), value: isVisible)` 或 `.transition(.opacity)`，给指示器的出现/消失留淡入淡出；状态量 `targetedFolderDrop` 的赋值不再包 `withAnimation`，所以同一行内 `.before ↔ .into ↔ .after` 的切换是瞬时换指示器位置。
- `clearTargetIfNeeded` 中的相同动画包裹也一并去掉。`MomentoSidebarRootFolderDropDelegate` 里同类动画包裹同步去掉以保持一致。

### 3.3 移除冗余的 insertion strip

删除：

- `folderInsertionDropTarget(before:)` 函数（463–481 行）。
- `MomentoSidebarFolderDropSurface` 枚举的 `.insertionBefore` case 与 id 拼装（1023–1038 行）。
- `MomentoSidebarFolderDropSurfaceResolver.surfaces` 整个 helper（1040–1049 行）。

简化 `if isFolderSectionExpanded` 分支（398–413 行）：

```swift
VStack(alignment: .leading, spacing: 0) {
    ForEach(visibleFolderRows) { row in
        folderRow(row)
    }
    folderRootDropTarget
}
```

依赖每行自身 8pt 的 `.before` 边缘热区承担"插入到行前"的命中——和 `.after` / `.into` 对称、单一 drop delegate。

### 3.4 不动的部分

- 视觉指示器（`folderDropIndicator` 细线、`sidebarAssetDropRowBackground` 的高亮）样式、厚度、配色都不动。
- `MomentoSidebarFolderDropResolver.moveCommand` 的循环引用防护、`rootEnd` 处理逻辑不动。
- 拖入资源（`MomentoSidebarAssetDropDelegate`）路径不动。
- `MomentoSidebarRootFolderDropDelegate`（拖到列表末尾变顶级）保留。

## 4. 验证

- **类型/构建**：`xcodebuild ... build`。
- **手动验证清单**（用户在 app 内操作）：
  1. 顶级有两个文件夹 A、B，拖 A 到 B 的中段 → 松手后 A 成为 B 的子文件夹，B 自动展开。
  2. A 是 B 的兄弟，拖 A 到 B 的最上 8pt → 松手后 A 排到 B 前面；拖到最下 8pt → A 排到 B 后面。
  3. 在 A 行中部短按悬停（< 0.4s）后挪走 → 不会触发嵌套，也不会触发展开。
  4. 拖 A 到 B 中段并停住 ≈0.4s → B 自动展开，再拖到 B 的某个子项 → 可以嵌套到孙级。
  5. 拖 A 到自身或后代上 → 不出现高亮、松手 noop。
  6. 拖到文件夹列表最末尾 `folderRootDropTarget` → 变顶级。
  7. 同级两行间快速来回挪动 → 指示器只有出现/消失淡入淡出，placement 切换瞬时无抖动。
- **回归**：执行 `MomentoTests` 全量。如果原有 `MomentoSidebarFolderDropPlacementResolver` / `MomentoSidebarFolderDropSurfaceResolver` 单测因签名变化失败，按本设计的新行为更新断言（不引入新的视觉细节测试）。

## 5. 风险与遗留

- **风险：8pt 边缘热区在快速拖拽时可能略难命中**。如果上线后反馈"很难只排序、总是嵌套"，下一步可以把 `folderDropEdgeZoneHeight` 调到 10–12pt。本次先保持 8pt，避免预先调参。
- **遗留：sidebar 文件 2130 行**，已超出项目 1000 行约束。本次只精减相关函数，不做大规模拆分。后续单独立项把 folder 拖拽相关结构体（Delegate / Resolver / Surface 枚举）抽到 `MomentoSidebarFolderDragDrop.swift`。
