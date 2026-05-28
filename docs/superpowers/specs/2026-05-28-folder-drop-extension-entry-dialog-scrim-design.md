# 设计与实施方案：拖图到文件夹修复 + 浏览器扩展入口 + 弹窗遮罩层

日期：2026-05-28
状态：spec（可执行级别，供 Codex 直接实施）

本批包含三个相互独立的小功能/修复，无共享状态，分别实施、分别提交、分别验证。

## 执行总则（给自动执行 agent）

- 每个功能独立一个 commit，类型见各节末尾。
- 所有引用的 file:line 基于 2026-05-28 的代码快照；如果实际行号有偏移，以**代码锚点字符串**（本文给出的 `old` 片段）为准做精确匹配，不要按行号盲改。
- 不改任何公共契约（函数签名、返回结构）。除本文明确列出的改动外不顺手改其它代码。
- 验证命令见每节「验证」。**功能一无法用自动化 UI 测试覆盖拖拽行为**，实现后必须显式告知需要人工 QA，不要声称"已验证拖拽生效"。
- 实施顺序：先功能三（零风险）→ 功能二（小、独立）→ 功能一（需人工 QA，放最后）。

---

## 功能三：模态弹窗加深底部遮罩层（最简单，先做）

### 背景

弹窗已有遮罩组件 `MomentoDialogBackdrop`，但当前是 10% 黑，几乎看不见，导致弹窗与底部 Liquid Glass 界面层级不分。所有模态弹窗（创建/编辑库、文件夹命名、删除库、删除文件夹、危险确认、素材导出）都通过 `ZStack { MomentoDialogBackdrop(...) ; 内容 }` 复用它，因此**单点改动**即可全部生效。

### 可直接执行的改动

文件：`Momento/Features/Library/MomentoCreateLibraryDialog.swift`，`MomentoDialogBackdrop`（约 :616-626）。

old：

```swift
        Rectangle()
            .fill(Color.black.opacity(0.10))
            .ignoresSafeArea()
```

new：

```swift
        Rectangle()
            .fill(Color.black.opacity(0.35))
            .ignoresSafeArea()
```

只改这一个数值。不动 `.contentShape`、`.onTapGesture(perform: dismiss)` 等其它部分。

### 明确不做

- 不给 `importProgressOverlay`（`ContentView.swift` 约 :1107-1115）加遮罩：它是非模态进度提示（`.allowsHitTesting(false)`，导入时用户仍可操作主窗口），加遮罩会错误阻断交互。
- 不动欢迎页 `WelcomeGlassBackdrop`、系统 `contextMenu`/`Menu`/popover、toolbar 的 sort/filter。
- 遮罩不覆盖 macOS 标题栏（与当前行为一致）。

### 验证

```sh
git diff --check
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination 'platform=macOS' build
```

手动 QA（用户）：打开任一弹窗，底部界面明显变暗、层级清晰；点遮罩仍能关闭弹窗。

提交：`style: deepen modal dialog backdrop`

---

## 功能二：空库页新增「安装浏览器扩展」按钮

### 背景

`ContentView.emptyGridState`（库已打开但无素材时显示，约 :1271-1301）当前只有「Import Assets」按钮。浏览器扩展把网页图片导入「当前打开的库」，入口放这个空库页最合理（已与用户确认）。点击后用 `NSWorkspace.shared.open` 打开扩展仓库最新 release 页。

仓库：`https://github.com/Seaony/Momento-Chomre-Extension`（仓库名拼写即 `Chomre`，不是本文 typo）。按 release 已发布处理，入口直接打开 `/releases/latest`，让用户落到可下载/安装的版本页。

### 可直接执行的改动

**改动 1：新增 hover 状态。** 文件 `Momento/ContentView.swift`，在现有 `isEmptyImportButtonHovered`（约 :64）声明后加一行：

old：

```swift
    @State private var isEmptyImportButtonHovered = false
```

new：

```swift
    @State private var isEmptyImportButtonHovered = false
    @State private var isInstallExtensionButtonHovered = false
```

**改动 2：在 `emptyGridState` 的导入按钮后追加第二个按钮。** 在 `emptyGridState` 里，找到导入按钮整块（从 `Button { isImporterPresented = true }` 到它的 `.padding(.top, 30)`，约 :1283-1297），在该 `Button` 闭包之后、`VStack` 闭合 `}`（约 :1298）之前插入：

```swift
            Button {
                installBrowserExtension()
            } label: {
                Label(localization.string("Install Browser Extension"), systemImage: "puzzlepiece.extension.fill")
            }
            .buttonStyle(.glass)
            .controlSize(.large)
            .scaleEffect(isInstallExtensionButtonHovered && !reduceMotion ? 1.035 : 1)
            .brightness(isInstallExtensionButtonHovered ? 0.08 : 0)
            .animation(reduceMotion ? nil : .smooth(duration: 0.16), value: isInstallExtensionButtonHovered)
            .pointerStyle(.link)
            .onHover { isHovered in
                isInstallExtensionButtonHovered = isHovered
            }
            .padding(.top, 12)
```

（`.padding(.top, 12)` 让两个按钮纵向间距合理；如果观感更想要并排，可改为把两个 Button 放进 `HStack`，但默认按纵向叠放实现，最小改动。）

**改动 3：新增点击方法。** 在 `ContentView` 里靠近其它 import 相关私有方法处（例如 `assignDroppedAssetsToFolder` 附近）新增：

```swift
    private func installBrowserExtension() {
        let releaseURL = URL(string: "https://github.com/Seaony/Momento-Chomre-Extension/releases/latest")!
        NSWorkspace.shared.open(releaseURL)
    }
```

`ContentView.swift` 已 `import AppKit` 且已在用 `NSWorkspace`，无需新增 import。

**改动 4：新增本地化条目。** 文件 `Momento/Localizable.xcstrings`，在 `strings` 对象里新增一个 key（按字母序插入，与现有条目同结构；`LocalizationCatalogTests` 要求 en + zh-Hans 都有值）：

```json
    "Install Browser Extension" : {
      "localizations" : {
        "en" : {
          "stringUnit" : {
            "state" : "translated",
            "value" : "Install Browser Extension"
          }
        },
        "zh-Hans" : {
          "stringUnit" : {
            "state" : "translated",
            "value" : "安装浏览器扩展"
          }
        }
      }
    }
```

（不要写 `extractionState: stale`；Xcode 下次构建会自行管理。zh-Hans 用「安装浏览器扩展」，与扩展自身 README 及 Chrome 习惯一致。）

### 验证

```sh
git diff --check
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination 'platform=macOS' build
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test -only-testing:MomentoTests/LocalizationCatalogTests
```

手动 QA（用户）：空库页显示两个按钮、不挤；点「安装浏览器扩展」用默认浏览器打开最新 release 页；点「导入」仍正常。

提交：`feat: add browser extension install entry on empty library`

### 非目标

- 不做应用内自动下载/解压/安装扩展（Chrome MV3 无法被宿主直接安装）。
- 不在欢迎页加该按钮。不改浏览器导入服务本身。

---

## 功能一：修复「拖素材到侧边栏文件夹完成关联」（需人工 QA，放最后）

### 现状与最可能根因（已走查现有链路）

这条链路代码里已存在（提交 `2cc2f71`），但实测「拖了没反应」。静态走查确认：

- 拖拽源 OK：`AssetFilePromiseProvider.writableTypes`（`AssetFilePromiseProvider.swift:50-51`）暴露 `com.seaony.momento.asset-ids`；`pasteboardPropertyList` 返回正确编码的 payload，`DragPasteboardWriterTests.swift:19-40` 已验证。
- 拖拽源操作掩码 OK：`AssetCollectionGridView.swift:250-251` 对 local 与非 local 都设 `.copy`，应用内拖放被允许。
- UTI 导出 OK：`Momento/Info.plist` 的 `UTExportedTypeDeclarations` 含 `com.seaony.momento.asset-ids`。
- drop 逻辑 OK：`MomentoSidebarView.swift:521` 文件夹行有 `.onDrop(of: [assetIDsUTType], delegate: MomentoSidebarAssetDropDelegate)`；delegate（`:1306-1372`）校验类型、显示 hover、读 payload 后调 `onAssignDroppedAssetsToFolder` → `store.assignAssets`（`LibraryStore.swift:740`）落库。
- 层级不遮挡 OK：folder-reorder 的 `.onDrop`（`:548`）在 `folderDropOverlay`，`allowsHitTesting(draggingFolderID != nil ...)`，拖资产时 `draggingFolderID == nil` 不抢占。

**最可能根因**：唯一未被证明可靠的一环是「`NSCollectionView` + `NSFilePromiseProvider` 发起的 AppKit 拖拽」被「SwiftUI `.onDrop` / `DropInfo`」接收。这是全项目唯一的「AppKit 拖拽 → SwiftUI 落点」组合，无可工作的同类参考。现有证据足够支持把落点改成 AppKit 原生 `NSDraggingDestination`，但不能自动证明真实拖拽 pasteboard 在运行时一定包含 `assetIDsPasteboardType`；最终有效性必须以人工 QA 为准。

### 执行与验证限制（必读）

- 本机约定不主动启动 App，本修复的核心行为（真实拖拽）**无法被自动化测试或本 agent 直接验证**。
- 自动执行 agent 的职责：实现下述代码 + 编译通过 + 跑可自动化的单测 + **如实报告"拖拽行为待人工 QA"**，不得声称拖拽已生效。
- 下述方案覆盖 SwiftUI `.onDrop`「不识别类型」和「识别但取不到数据」两种常见失败路径，因此不需要先做运行时根因区分即可实施；若真实 pasteboard 没写入自定义类型，人工 QA 会失败，按 fallback 处理。
- 方案中有两个只能运行时验证的交互点（见「已知风险」）。若人工 QA 发现拖拽仍不生效，按「fallback」处理，不要堆补丁。

### 方案（推荐，唯一主方案）：给文件夹行加 AppKit 落点

用 AppKit 原生 `NSDraggingDestination` 直接读 `draggingPasteboard`，绕开 SwiftUI 桥接。与项目既有 AppKitBridge 模式一致（grid 本身就是因此用 AppKit）。

**新增文件** `Momento/AppKitBridge/SidebarFolderAssetDropView.swift`：

```swift
// 中文注释：文件夹行的素材 drop 用 AppKit 原生 drag destination 接收，
// 规避 SwiftUI .onDrop 收不到 NSFilePromiseProvider 自定义类型的问题。
import AppKit
import SwiftUI

struct SidebarFolderAssetDropView: NSViewRepresentable {
    var currentLibraryID: AssetLibrary.ID?
    var onTargetedChange: (Bool) -> Void
    var onDropAssetIDs: (Set<AssetItem.ID>) -> Void

    func makeNSView(context: Context) -> AssetFolderDropTargetView {
        let view = AssetFolderDropTargetView()
        view.registerForDraggedTypes([AssetDragPasteboardWriter.assetIDsPasteboardType])
        view.handlers = context.coordinator
        return view
    }

    func updateNSView(_ nsView: AssetFolderDropTargetView, context: Context) {
        context.coordinator.parent = self
        nsView.handlers = context.coordinator
    }

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    @MainActor
    final class Coordinator: AssetFolderDropTargetHandlers {
        var parent: SidebarFolderAssetDropView

        init(_ parent: SidebarFolderAssetDropView) {
            self.parent = parent
        }

        func canAcceptAssetDrop() -> Bool {
            parent.currentLibraryID != nil
        }

        func assetDropTargetedChanged(_ isTargeted: Bool) {
            parent.onTargetedChange(isTargeted)
        }

        func performAssetDrop(from pasteboard: NSPasteboard) -> Bool {
            guard let data = pasteboard.data(forType: AssetDragPasteboardWriter.assetIDsPasteboardType),
                  let payload = try? JSONDecoder.momento.decode(AssetDragPasteboardPayload.self, from: data),
                  payload.libraryID == parent.currentLibraryID else {
                return false
            }
            parent.onDropAssetIDs(Set(payload.assetIDs))
            return true
        }
    }
}

@MainActor
protocol AssetFolderDropTargetHandlers: AnyObject {
    func canAcceptAssetDrop() -> Bool
    func assetDropTargetedChanged(_ isTargeted: Bool)
    func performAssetDrop(from pasteboard: NSPasteboard) -> Bool
}

@MainActor
final class AssetFolderDropTargetView: NSView {
    weak var handlers: AssetFolderDropTargetHandlers?

    override func draggingEntered(_ sender: NSDraggingInfo) -> NSDragOperation {
        guard handlers?.canAcceptAssetDrop() == true,
              sender.draggingPasteboard.availableType(from: [AssetDragPasteboardWriter.assetIDsPasteboardType]) != nil else {
            return []
        }
        handlers?.assetDropTargetedChanged(true)
        return .copy
    }

    override func draggingUpdated(_ sender: NSDraggingInfo) -> NSDragOperation {
        guard handlers?.canAcceptAssetDrop() == true,
              sender.draggingPasteboard.availableType(from: [AssetDragPasteboardWriter.assetIDsPasteboardType]) != nil else {
            return []
        }
        return .copy
    }

    override func draggingExited(_ sender: NSDraggingInfo?) {
        handlers?.assetDropTargetedChanged(false)
    }

    override func draggingEnded(_ sender: NSDraggingInfo) {
        handlers?.assetDropTargetedChanged(false)
    }

    override func performDragOperation(_ sender: NSDraggingInfo) -> Bool {
        let accepted = handlers?.performAssetDrop(from: sender.draggingPasteboard) ?? false
        handlers?.assetDropTargetedChanged(false)
        return accepted
    }
}
```

**集成进 `MomentoSidebarView.folderRow`**：把现有的 SwiftUI 资产 `.onDrop`（仅这一条）换成上面的 AppKit 视图作为 `.background`。

old（`MomentoSidebarView.swift:521-527`）：

```swift
        .onDrop(of: [AssetDragPasteboardWriter.assetIDsUTType], delegate: MomentoSidebarAssetDropDelegate(
            currentLibraryID: currentLibraryID,
            targetID: rowID,
            targetedAssetDropID: $targetedAssetDropID
        ) { assetIDs in
            onAssignDroppedAssetsToFolder(assetIDs, folder.id)
        })
```

new：

```swift
        .background(
            SidebarFolderAssetDropView(
                currentLibraryID: currentLibraryID,
                onTargetedChange: { isTargeted in
                    withAnimation(.smooth(duration: 0.12)) {
                        if isTargeted {
                            targetedAssetDropID = rowID
                        } else if targetedAssetDropID == rowID {
                            targetedAssetDropID = nil
                        }
                    }
                },
                onDropAssetIDs: { assetIDs in
                    onAssignDroppedAssetsToFolder(assetIDs, folder.id)
                }
            )
        )
```

注意：`.background` 必须加在 row 既有 `.background { sidebarAssetDropRowBackground(...) }`（`:488-494`）**之外/之后**，避免被高亮背景遮挡；建议放在原 `.onDrop` 所在位置（高亮背景已经在更内层）。`targetedAssetDropID`、`rowID`、`currentLibraryID`、`onAssignDroppedAssetsToFolder`、`folder` 在 `folderRow` 作用域内均可用，无需新增参数。

**清理**：删除现在已无用的 `MomentoSidebarAssetDropDelegate`（`:1306-1372`），它只被这一处 `.onDrop` 使用（用 `rg -n "MomentoSidebarAssetDropDelegate" Momento MomentoTests` 确认引用后再删）。

**同步更新架构护栏测试。** 当前 `ArchitectureGuardTests.testSidebarAcceptsInternalAssetDropsForOrganization` 断言 sidebar 里包含 `MomentoSidebarAssetDropDelegate`。删除 delegate 后必须同步改这个测试，否则全量测试会失败。

old（`MomentoTests/ArchitectureGuardTests.swift:245-255`）：

```swift
    func testSidebarAcceptsInternalAssetDropsForOrganization() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarURL(), encoding: .utf8)

        XCTAssertTrue(sidebarSource.contains("MomentoSidebarAssetDropDelegate"))
        XCTAssertTrue(sidebarSource.contains("AssetDragPasteboardWriter.assetIDsUTType"))
        XCTAssertTrue(sidebarSource.contains("onAssignDroppedAssetsToFolder(assetIDs, folder.id)"))
        XCTAssertFalse(sidebarSource.contains("sidebarTagSection"))
        XCTAssertFalse(sidebarSource.contains("onAssignDroppedAssetsToTag"))
        XCTAssertTrue(contentSource.contains("try store.assignAssets(ids: assetIDs, to: folderID)"))
    }
```

new：

```swift
    func testSidebarAcceptsInternalAssetDropsForOrganization() throws {
        let contentSource = try String(contentsOf: contentViewURL(), encoding: .utf8)
        let sidebarSource = try String(contentsOf: sidebarURL(), encoding: .utf8)
        let bridgeSource = try appKitBridgeSource()

        XCTAssertTrue(sidebarSource.contains("SidebarFolderAssetDropView("))
        XCTAssertTrue(sidebarSource.contains("onAssignDroppedAssetsToFolder(assetIDs, folder.id)"))
        XCTAssertTrue(bridgeSource.contains("registerForDraggedTypes([AssetDragPasteboardWriter.assetIDsPasteboardType])"))
        XCTAssertTrue(bridgeSource.contains("performAssetDrop(from pasteboard: NSPasteboard) -> Bool"))
        XCTAssertFalse(sidebarSource.contains("MomentoSidebarAssetDropDelegate"))
        XCTAssertFalse(sidebarSource.contains("sidebarTagSection"))
        XCTAssertFalse(sidebarSource.contains("onAssignDroppedAssetsToTag"))
        XCTAssertTrue(contentSource.contains("try store.assignAssets(ids: assetIDs, to: folderID)"))
    }
```

### 单元测试（可自动化的部分）

新增 `MomentoTests/SidebarFolderAssetDropTests.swift`，验证新 AppKit drop coordinator 的可测路径：同库 payload 会触发关联，跨库 payload 会被拒绝（不依赖真实拖拽 UI）：

```swift
import AppKit
import XCTest
@testable import Momento

@MainActor
final class SidebarFolderAssetDropTests: XCTestCase {
    func testPerformsSameLibraryAssetDropFromPasteboard() throws {
        var capturedAssetIDs: Set<AssetItem.ID>?
        let view = SidebarFolderAssetDropView(
            currentLibraryID: "library",
            onTargetedChange: { _ in },
            onDropAssetIDs: { capturedAssetIDs = $0 }
        )
        let coordinator = view.makeCoordinator()
        let pasteboard = try makePasteboard(
            libraryID: "library",
            assetIDs: ["asset-a", "asset-b"],
            primaryAssetID: "asset-a"
        )

        XCTAssertTrue(coordinator.performAssetDrop(from: pasteboard))
        XCTAssertEqual(try XCTUnwrap(capturedAssetIDs), ["asset-a", "asset-b"])
    }

    func testRejectsCrossLibraryAssetDropFromPasteboard() throws {
        var capturedAssetIDs: Set<AssetItem.ID>?
        let view = SidebarFolderAssetDropView(
            currentLibraryID: "library-b",
            onTargetedChange: { _ in },
            onDropAssetIDs: { capturedAssetIDs = $0 }
        )
        let coordinator = view.makeCoordinator()
        let pasteboard = try makePasteboard(
            libraryID: "library-a",
            assetIDs: ["asset-a"],
            primaryAssetID: "asset-a"
        )

        XCTAssertFalse(coordinator.performAssetDrop(from: pasteboard))
        XCTAssertNil(capturedAssetIDs)
    }

    private func makePasteboard(
        libraryID: AssetLibrary.ID,
        assetIDs: [AssetItem.ID],
        primaryAssetID: AssetItem.ID
    ) throws -> NSPasteboard {
        let pasteboard = NSPasteboard(name: NSPasteboard.Name("SidebarFolderAssetDropTests.\(UUID().uuidString)"))
        pasteboard.clearContents()
        let data = try XCTUnwrap(AssetDragPasteboardWriter.encodedPayload(
            libraryID: libraryID,
            assetIDs: assetIDs,
            primaryAssetID: primaryAssetID
        ))
        pasteboard.setData(data, forType: AssetDragPasteboardWriter.assetIDsPasteboardType)
        return pasteboard
    }
}
```

`DragPasteboardWriterTests` 已覆盖拖拽源能写出 asset payload；`store.assignAssets` 的落库一致性已有现成覆盖，跑既有 drag/sidebar/library 测试即可。

### 已知风险（人工 QA 必查）

1. **`.background` 的 AppKit 视图能否收到拖拽**：SwiftUI 内容在前、drop 视图在后。AppKit drag destination 基于注册类型 + 视图几何，理论上前层 SwiftUI 视图未注册该类型时会落到后层 drop 视图。若 QA 发现拖到文件夹无高亮/无关联，说明前层遮挡了拖拽 → 走 fallback。
2. **不要重写 `hitTest` 返回 nil**：那样虽然能让鼠标点击穿透，但会同时让该视图退出拖拽目标判定，反而收不到拖拽。本方案不覆盖 `hitTest`，靠 `.background` 分层。鼠标点击仍由前层 SwiftUI 内容（`.onTapGesture` 选中）处理。

### fallback（仅当人工 QA 证明 `.background` 收不到拖拽时）

不要在 `.background` 上叠加更多 hack。改为：用**单个** AppKit drop host 覆盖整个文件夹列表区域（`VStack`，`:401`），在 `draggingUpdated` 里用光标 Y 坐标对照各行 frame 计算命中的 `folderID`（行高 `MomentoSidebarMenuMetrics.folderRowHeight` 已知），命中后高亮并在 `performDragOperation` 关联。此为更大改动，需单独设计，不在本次默认范围。

### 验证

```sh
git diff --check
xcodebuild -project Momento.xcodeproj -scheme Momento -configuration Debug -destination 'platform=macOS' build
xcodebuild -project Momento.xcodeproj -scheme Momento -destination 'platform=macOS' test
```

- 跑新增 `SidebarFolderAssetDropTests` + 既有 drag/sidebar/library 测试。
- **人工 QA（用户，必做）**：单选/多选拖到文件夹能关联；拖到文件夹时行高亮出现/消失；folder 之间重排仍正常；拖素材到 Finder 导出仍正常；跨库 payload 不误关联。

提交：`fix: receive asset folder drops via AppKit drag destination`

### 非目标

- 不改 grid 拖拽源、file promise 导出、`store.assignAssets` 契约。
- 不重写 sidebar 其它拖拽（folder/library 重排）。

---

## 执行前提与人工验收

- 功能二按浏览器扩展 release 已发布处理，入口固定打开 `/releases/latest`；zh-Hans 文案固定为「安装浏览器扩展」。
- 功能三 `0.35` 作为首版遮罩深度执行；若用户运行 App 后觉得过深或过浅，再按视觉反馈单独微调。
- 功能一按「实现 + 编译 + 单测 + 人工 QA」执行；agent 只报告自动化结果，拖拽是否真实生效由用户运行 App 验证。
