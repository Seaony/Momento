package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/libraryicon"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type AutoImportRule struct {
	Path     string `json:"path"`
	FolderID string `json:"folderID"`
	Enabled  bool   `json:"enabled"`
	// 规则所属图库包的位置。用 Finder 直接复制的图库包和原库 ID 相同，靠位置区分，免得副本也去监视同一个文件夹。
	Library string `json:"library,omitempty"`
}

type Preferences struct {
	Language                string                      `json:"language"`
	Theme                   string                      `json:"theme"`
	UIZoom                  float32                     `json:"uiZoom"`
	LaunchAtLogin           bool                        `json:"launchAtLogin"`
	AutoCheckUpdates        bool                        `json:"autoCheckUpdates"`
	BrowserExtensionEnabled bool                        `json:"browserExtensionEnabled"`
	CloseAction             string                      `json:"closeAction"`
	TrayIcon                bool                        `json:"trayIcon"`
	DockWhenHidden          bool                        `json:"dockWhenHidden"`
	SidebarCounts           bool                        `json:"sidebarCounts"`
	FolderDoubleClick       string                      `json:"folderDoubleClick"`
	SidebarHidden           []string                    `json:"sidebarHidden"`
	DoubleClick             string                      `json:"doubleClick"`
	SpaceKey                string                      `json:"spaceKey"`
	PreviewWheel            string                      `json:"previewWheel"`
	PreviewFit              string                      `json:"previewFit"`
	PreviewPixelated        bool                        `json:"previewPixelated"`
	PreviewChecker          bool                        `json:"previewChecker"`
	HoverPlayVideos         bool                        `json:"hoverPlayVideos"` // 鼠标停在视频卡片上时静音播放
	SelectImported          bool                        `json:"selectImported"`
	ImportKeywords          bool                        `json:"importKeywords"`
	ExtensionPicker         bool                        `json:"extensionPicker"`
	AutoImport              map[string][]AutoImportRule `json:"autoImport"`
	Shortcuts               map[string]string           `json:"shortcuts"`
	Notify                  bool                        `json:"notify"`
	NotifyEvents            []string                    `json:"notifyEvents"`
	NotifySound             bool                        `json:"notifySound"`
	PasswordHash            string                      `json:"passwordHash,omitempty"`
	ThumbnailSize           float32                     `json:"thumbnailSize"`
	Onboarded               bool                        `json:"onboarded"`
	ViewMode                int                         `json:"viewMode"`
	Sort                    string                      `json:"sort"`
	SortDescending          bool                        `json:"sortDescending"`
	SidebarVisible          bool                        `json:"sidebarVisible"`
	SidebarWidth            float32                     `json:"sidebarWidth"`
	SmartFolders            bool                        `json:"smartFolders"`
	SubfolderContents       bool                        `json:"subfolderContents"` // 父文件夹里也显示子文件夹的素材
	EXIFCollapsed           bool                        `json:"exifCollapsed"`
	ExportPresets           []ExportPreset              `json:"exportPresets,omitempty"`
	SlideshowSeconds        int                         `json:"slideshowSeconds,omitempty"` // 放映间隔，0 是默认 5 秒
}

type App struct {
	Window                                                    *mygo.Window
	SettingsWindow                                            *mygo.Window
	SettingsRequested                                         func()
	Extension                                                 *browser.Service
	ExtensionStatus                                           browser.Status
	Texts                                                     i18n.Catalog
	Library                                                   *library.Library
	Assets                                                    *assets.Service
	Snapshot                                                  model.Snapshot
	Index                                                     *catalog.Index
	Query                                                     catalog.Query
	Visible                                                   []model.Asset
	Selected                                                  assetSelection
	ViewMode                                                  int
	TagPage                                                   bool
	SidebarVisible, InspectorVisible                          bool
	SidebarWidth                                              float32
	sidebarShownWidth, inspectorShownWidth                    float32
	sidebarSplitWidth                                         float32
	chromeLayoutReady                                         bool
	Search                                                    ui.Handle
	Busy                                                      bool
	Progress                                                  *model.ImportProgress
	Error, Notice                                             string
	tagClipboard                                              []string // ⌘⇧C 复制下来的标签，⌘⇧V 加到选中的素材上
	SettingsOpen, NewLibraryOpen, EditLibraryOpen, ExportOpen bool
	Rename                                                    renameState
	Similar                                                   similarState
	libraryOpening                                            bool       // 正在后台打开另一个图库
	Slideshow                                                 *slideshow // 正在全屏放映
	justified                                                 justifiedCache
	Preferences                                               Preferences
	Browse                                                    browseState
	GBrowsing                                                 gBrowsingState
	Welcome                                                   welcomeState
	Lock                                                      lockState
	Import                                                    importState
	Quitting                                                  bool
	quitAfterInspector                                        bool
	pendingLibraryAfterEdits                                  string
	TextInputFocused                                          bool
	menuInspectorAvailable                                    bool
	menuPreviewOpen                                           bool
	PreferencesChanged                                        func()
	WindowShown                                               func()
	Recent                                                    []library.Recent
	hoverVideoState                                           hoverVideo
	RecentMissing                                             map[string]bool // 打不开的图库（外接盘没插、被移走），菜单里显示成灰色「找不到」
	Sidebar                                                   sidebarState
	Toolbar                                                   toolbarState
	Inspector                                                 inspectorState
	Preview                                                   previewWindows
	Settings                                                  settingsState
	Dialog                                                    dialogsState
	Tags                                                      tagsState
	MenuChanged                                               func()
	InspectorAvailabilityChanged                              func(bool)
	PromptOpen, ConfirmOpen                                   bool
	PromptIcon, PromptTitle, PromptValue                      string
	PromptSubmit                                              func(string)
	ConfirmTitle, ConfirmDetail                               string
	ConfirmLabel                                              string
	ConfirmAction                                             func()
	Thumbnails                                                *thumbnailCache
	manifest                                                  model.Manifest
	pendingOpen                                               string
	pendingImports                                            [][2]string
	queuedRuns                                                []queuedRun
	noticeAction                                              *noticeAction
	remoteImporting                                           bool // 正在处理一条深度链接导入
	remoteBatches                                             map[string]*remoteBatch
	assetByID                                                 map[string]model.Asset
	configDir                                                 string
	preferencesBlocked                                        error // 偏好文件读不出来又没法挪走时，本次运行不写回，免得覆盖用户原来的设置
	recents                                                   *library.RecentStore
	ctx                                                       context.Context
	cancel                                                    context.CancelFunc
	prefMu                                                    sync.Mutex
	prefVersion                                               atomic.Uint64
	workMu                                                    sync.Mutex
	generation                                                uint64
	scroll                                                    ui.ScrollState
	selectedIndex                                             int
	anchor                                                    int
	hovered                                                   string
	subfolders                                                subfolderCache
	assetLead                                                 float32 // 素材上面子文件夹卡片区占的高度
	indexVersion                                              uint64  // 图库数据每变一次加一，筛选数量、子文件夹卡片的缓存按它失效
	assetClickID, pendingAssetActivation                      string
	assetClickBounds                                          ui.Rect
	layout                                                    *MasonryLayout
	layoutWidth                                               float32
	layoutVersion                                             uint64
	list                                                      listLayout
	focusAssetID                                              string // 方向键移动选择后，下一帧把键盘焦点交给这张卡片
	visibleVersion                                            uint64
	// contentEpoch 在滚动中途换筛选、搜索或排序（列表回到顶部）时加一：素材区整块淡入新内容，卡片不从旧坐标飞过来。
	contentEpoch uint64
	// shownLibrary 是上一帧显示的图库：换了图库那一帧整窗直接换成新的，不让侧栏每一行、每张卡片各自进出。
	shownLibrary    string
	snapshotVersion uint64
	selection       selectionCache

	// MenuStateChanged 在菜单项该不该能点变了时调用，主程序据此灰掉此刻点了没用的菜单项。
	MenuStateChanged func(MenuState)
	// TrayFailed：系统没有托盘（比如部分 Linux 桌面）没建出图标，窗口藏起来就找不回了。
	TrayFailed bool
	menuState  MenuState
	// runningPatch 是正在写库、排队时已先画到界面上的那一笔：写库途中整库重读时也要叠回去。
	runningPatch *assetPatch
}

func New() *App {
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{ctx: ctx, cancel: cancel, SidebarVisible: true, SidebarWidth: 236, selectedIndex: -1, anchor: -1, Preferences: DefaultPreferences(), Query: catalog.Query{Sort: "added", Descending: true}}
	var preferencesProblem func() string
	dir, err := library.ConfigDir()
	if err != nil {
		a.Error = a.Texts.Error(err)
	} else {
		a.configDir = dir
		a.recents = library.NewRecentStore(dir)
		preferencesProblem = a.loadPreferences()
		a.Recent, err = a.recents.List()
		if err != nil {
			a.Error = a.Texts.Error(err)
		}
	}
	a.RestoreUIState()
	a.Texts.Language = a.Preferences.Language
	if preferred := mygo.App.PreferredLanguages(); len(preferred) > 0 {
		a.Texts.SystemLanguage = preferred[0]
	}
	a.applyLocale()
	if preferencesProblem != nil {
		a.Error = preferencesProblem()
	}
	a.applyAppearance()
	a.InitPasswordLock()
	return a
}

func (a *App) T(key string, args ...any) string { return a.Texts.T(key, args...) }

func (a *App) ShowSettings() {
	if a.IsLocked() {
		a.ShowMainWindow()
		return
	}
	a.SettingsOpen = true
	if a.SettingsRequested != nil {
		a.SettingsRequested()
	}
}

// 设置窗口的红绿灯：顶栏 44 高，按钮高 14，上下各留 15 正好居中。
func TrafficLights() *mygo.Point { return &mygo.Point{X: 14, Y: 15} }

// 主窗口的红绿灯：中心对齐 52 高工具栏的中线，和搜索框、导入按钮在一条线上。
func mainTrafficLights() *mygo.Point {
	return &mygo.Point{X: 14, Y: (MainToolbarHeight - 14) / 2}
}

func (a *App) OpenWindow() {
	a.Window = mygo.NewWindow(mygo.WindowOptions{
		Title: "Momento", Width: 1100, Height: 800, MinWidth: 1100, MinHeight: 640, StateKey: "main",
		TitleBarStyle: mygo.TitleBarHiddenInset, TitleBarHeight: MainToolbarHeight,
		TrafficLightPosition: mainTrafficLights(),
		BackgroundColor:      WindowBackgroundColor(), Content: ui.View(a.View),
	})
	a.Thumbnails = newThumbnailCache(func(fn func()) { a.Window.Update(fn) })
	a.Window.OnFileDrop(func(e *mygo.FileDropEvent) { a.ImportPaths(e.Paths) })
	a.ApplyWindowZoom()
	a.Window.OnClose(a.closeMainWindow)
	a.Window.OnBlur(a.CancelPreviewGesture)
	a.Window.OnHide(a.CancelPreviewGesture)

	go a.watchLibrary()
}

func (a *App) View(c *ui.Context) {
	if a.Thumbnails != nil {
		a.Thumbnails.NextFrame()
		defer a.Thumbnails.EndFrame()
	}
	if id := a.libraryID(); id != a.shownLibrary {
		if a.shownLibrary != "" && id != "" {
			c.DisableAnimations()
		}
		a.shownLibrary = id
	}
	a.applyTheme(c)
	a.SyncMenuFocus(c)
	a.ContinueInspectorEdits()
	a.completeInspectorTransitions()
	if a.IsLocked() {
		a.lockScreen(c)
		return
	}
	defer a.RememberUIState()
	if a.Library == nil {
		a.welcome(c)
	} else {
		if !a.InspectorAvailable() && a.InspectorVisible {
			a.CollapseInspector()
		}
		// 从欢迎页打开图库时整个工作区淡入，关掉图库时淡出回欢迎页。
		workspace := fadeMotion(ui.Row(c.Key("workspace")).Fill().ObserveInput(a.observeWorkspaceInput).HandleInput(a.activateAssetDoubleClick))
		// 分隔条被拖动（或用方向键移动）时，Split 改的宽度和上一帧显示的不同；这时栏宽直接跟手，不走弹簧。
		dragging := a.chromeLayoutReady && a.SidebarVisible && a.sidebarSplitWidth != a.sidebarShownWidth
		if dragging {
			a.SidebarWidth = max(196, min(340, a.sidebarSplitWidth))
		}
		a.SidebarWidth = max(196, min(340, a.SidebarWidth))
		sidebarTarget := float32(0)
		if a.SidebarVisible {
			sidebarTarget = a.SidebarWidth
		}
		a.sidebarShownWidth = animatePaneWidth(c, workspace, "sidebar-width", sidebarTarget, dragging)
		a.inspectorShownWidth = animatePaneWidth(c, workspace, "inspector-width", a.InspectorTargetWidth(), false)
		a.chromeLayoutReady = true
		workspace.Children(func() {
			body := func() {
				ui.Row(c.Key("library-body")).Fill().AlignItems(ui.Stretch).Children(func() {
					ui.Column(c.Key("content")).Grow(1).MinWidth(0).Background(c.Theme().Background).Children(func() {
						if a.TagPage {
							a.tags(c)
						} else {
							a.assetArea(c)
						}
						ui.Column(c.Key("main-header")).Absolute().Top(0).Left(0).Right(0).Children(func() {
							a.toolbar(c)
							a.importProgress(c)
						})
					})
					ui.Column(c.Key("inspector-slot")).Width(a.inspectorShownWidth).Shrink(0).Clip().Children(func() {
						if a.InspectorVisible && a.InspectorAvailable() {
							fadeMotion(ui.Column(c.Key("inspector-content")).Width(280).FillHeight().Background(c.Theme().Background).BorderWidth(0, 0, 0, 1).BorderColor(c.Theme().Border)).Children(func() { a.inspector(c) })
						}
					})
				})
			}
			a.sidebarSplitWidth = a.sidebarShownWidth
			ui.Split(c.Key("library-split"), &a.sidebarSplitWidth, func() { a.sidebar(c) }, body).SplitReset(236).Fill()
		})
	}
	a.ReconcilePreviewKeys(c)
	a.shortcuts(c)
	a.dialogs(c)
	a.extensionImportPicker(c)
	a.smartFolderDialog(c)
	a.passwordSettingsDialog(c, false)
	if a.Notice != "" {
		c.Toast(a.Notice)
		a.Notice = ""
	}
	if n := a.noticeAction; n != nil {
		a.noticeAction = nil
		c.ToastAction(n.message, n.label, n.action)
	}
	a.noticeViewport(c)
}

func (a *App) ShowError(err error) {
	if err != nil {
		slog.Error("操作失败", "error", err)
		a.Error = a.Texts.Error(err)
	}
}
func (a *App) Notify(s string) { a.Notice = s }

// 带一个按钮的提示，比如移到回收站后的「撤销」、自动导入完成后的「查看」；按钮点了提示就关掉。
type noticeAction struct {
	message, label string
	action         func()
}

func (a *App) NotifyAction(message, label string, action func()) {
	a.noticeAction = &noticeAction{message: message, label: label, action: action}
}
func (a *App) Ask(icon, title, initial string, submit func(string)) {
	a.PromptIcon = icon
	a.PromptTitle = title
	a.PromptValue = initial
	a.PromptSubmit = submit
	a.PromptOpen = true
}
func (a *App) Confirm(title, detail, label string, action func()) {
	a.ConfirmTitle = title
	a.ConfirmDetail = detail
	a.ConfirmLabel = label
	a.ConfirmAction = action
	a.ConfirmOpen = true
}
func (a *App) ShowNewLibrary() { a.NewLibraryOpen = true }
func (a *App) ShowEditLibrary() {
	if a.Library != nil {
		a.EditLibraryOpen = true
	}
}

func (a *App) Run(label string, work func(context.Context) error) { a.run(label, work, nil) }

// 只改了几张素材的字段时用：写库成功后按同样的规则直接改内存里的数据，不再整库重读（十万张要一秒多）。
// patch 为 nil 表示数据没变（比如导出），什么都不用刷新。
func (a *App) RunPatch(label string, ids []string, work func(context.Context) error, patch func(*model.Asset)) {
	a.run(label, work, &assetPatch{ids: ids, apply: patch})
}

type assetPatch struct {
	ids   []string
	apply func(*model.Asset)
	// failed 在写入失败、或排队时换了图库被作废时，在界面线程上调用（可以为空）。
	failed func(error)
	// optimistic 表示排队时已经先把改动画在界面上了，写入失败要整库重读退回真实状态。
	optimistic bool
}

type queuedRun struct {
	generation uint64
	label      string
	work       func(context.Context) error
	patch      *assetPatch
}

func (a *App) run(label string, work func(context.Context) error, patch *assetPatch) {
	if a.Library == nil {
		return
	}
	// 导入或别的写入还没结束时，不能把用户的操作悄悄丢掉：排队，空闲后按顺序执行。
	if a.Busy {
		// 改素材字段的操作先在界面上改好，按下去马上有回应（导入几分钟时心形不会一直不变红）；写库仍按顺序排队。
		if patch != nil && patch.apply != nil && len(patch.ids) > 0 {
			a.patchAssets(patch.ids, patch.apply)
			patch.optimistic = true
		}
		a.queuedRuns = append(a.queuedRuns, queuedRun{generation: a.generation, label: label, work: work, patch: patch})
		slog.Info("操作排队", "operation", label, "queued", len(a.queuedRuns))
		return
	}
	lib := a.Library
	gen := a.generation
	a.Busy = true
	if patch != nil && patch.optimistic {
		a.runningPatch = patch
	}
	go func() {
		a.workMu.Lock()
		defer a.workMu.Unlock()
		slog.Info("操作开始", "operation", label)
		err := work(a.ctx)
		var snapshot model.Snapshot
		if err == nil && patch == nil {
			snapshot, err = lib.Snapshot(a.ctx)
		}
		a.Window.Update(func() {
			if gen != a.generation {
				return
			}
			a.Busy = false
			a.runningPatch = nil
			if err != nil {
				if patch != nil && patch.failed != nil {
					patch.failed(err)
				}
				a.ShowError(err)
				if patch != nil && patch.optimistic {
					// 界面上已经提前改了，写库失败就整库重读，不能留着一个没存进去的状态。
					a.Refresh()
				}
				a.drainPending()
				return
			}
			a.manifest = lib.Manifest
			if patch == nil {
				a.applySnapshot(snapshot)
			} else if !patch.optimistic {
				// 排队时已经先画到界面上的不再贴一次：后面排着的改动也已经叠在上面了，再贴会把较新的状态盖回去（快速收藏再取消时心形变回红色），中间还会清掉选择。
				a.patchAssets(patch.ids, patch.apply)
			}
			if label == "edit_library" {
				a.Window.SetTitle(a.manifest.DisplayName + " – Momento")
				a.remember()
			}
			a.drainPending()
			slog.Info("操作完成", "operation", label)
		})
	}()
}

// 按 ID 原地改内存里的素材，索引只重算这几张。快照的素材切片只有界面线程在读，交给后台的都是副本，所以可以原地改；
// apply 里换新切片，不原地改成员切片，选中缓存等副本里的旧值才不会跟着变。
func (a *App) patchAssets(ids []string, apply func(*model.Asset)) {
	if apply == nil || len(ids) == 0 || a.Index == nil {
		return
	}
	targets := make(map[string]bool, len(ids))
	for _, id := range ids {
		targets[id] = true
	}
	before := make(map[string]model.Asset, len(ids))
	for _, id := range ids {
		if asset, ok := a.assetByID[id]; ok {
			before[id] = asset
		}
	}
	now := time.Now().UTC()
	changed := a.Index.Patch(targets, func(asset *model.Asset) {
		apply(asset)
		asset.UpdatedAt = now
	})
	a.snapshotVersion++
	a.indexVersion++
	a.applyVisibleChanges(before, changed)
}

func (a *App) mergeAssets(partial model.Snapshot) {
	before := make(map[string]model.Asset, len(partial.Assets))
	for _, asset := range partial.Assets {
		if old, exists := a.assetByID[asset.ID]; exists {
			before[asset.ID] = old
		}
	}
	merged, changed, metadataChanged := a.Index.Merge(partial)
	a.Snapshot = merged
	a.snapshotVersion++
	a.indexVersion++
	if metadataChanged {
		for _, index := range changed {
			asset := merged.Assets[index]
			a.assetByID[asset.ID] = asset
		}
		a.refine(false)
		a.pruneAutoImportTargets()
		return
	}
	a.applyVisibleChanges(before, changed)
}

func (a *App) applyVisibleChanges(before map[string]model.Asset, changed []int) {
	for _, index := range changed {
		asset := a.Snapshot.Assets[index]
		a.assetByID[asset.ID] = asset
	}
	if len(changed) == 0 {
		return
	}
	if len(changed) > 64 || a.Query.Scope == "random" {
		a.refine(false)
		return
	}
	query := a.browseQuery(a.Query)
	if query.Now.IsZero() {
		query.Now = time.Now()
	}
	compare := catalog.AssetOrder(query)
	layoutChanged := false
	currentID, anchorID := a.visibleIDAt(a.selectedIndex), a.visibleIDAt(a.anchor)
	visibleIDs := a.Preview.visibleIDs
	if a.Preview.visibleVersion != a.visibleVersion {
		visibleIDs = nil
	}
	for _, index := range changed {
		asset := a.Snapshot.Assets[index]
		old, existed := before[asset.ID]
		position, visible := 0, false
		if existed {
			position, visible = slices.BinarySearchFunc(a.Visible, &old, compare)
		}
		matches := a.Index.Matches(query, asset.ID)
		if visible && matches && compare(old, &asset) == 0 {
			a.Visible[position] = asset
			layoutChanged = layoutChanged || old.Width != asset.Width || old.Height != asset.Height || old.Rotation != asset.Rotation
			continue
		}
		if visible {
			a.Visible = slices.Delete(a.Visible, position, position+1)
			delete(visibleIDs, asset.ID)
		}
		if matches {
			position, _ = slices.BinarySearchFunc(a.Visible, &asset, compare)
			a.Visible = slices.Insert(a.Visible, position, asset)
			if visibleIDs != nil {
				visibleIDs[asset.ID] = true
			}
		} else if visible {
			a.Selected.Remove(asset.ID)
		}
		layoutChanged = layoutChanged || visible || matches
	}
	if query.Scope == "recent" {
		cutoff := query.Now.AddDate(0, 0, -30)
		last := len(a.Visible)
		for last > 0 && a.Visible[last-1].LastUsedAt.Before(cutoff) {
			last--
			id := a.Visible[last].ID
			a.Selected.Remove(id)
			delete(visibleIDs, id)
		}
		if last != len(a.Visible) {
			a.Visible = slices.Delete(a.Visible, last, len(a.Visible))
			layoutChanged = true
		}
	}
	if layoutChanged {
		find := func(id string) int {
			if asset, exists := a.assetByID[id]; exists {
				if position, ok := slices.BinarySearchFunc(a.Visible, &asset, compare); ok {
					return position
				}
			}
			return -1
		}
		a.anchor, a.selectedIndex = find(anchorID), find(currentID)
		a.visibleVersion++
		a.Preview.visibleIDs, a.Preview.visibleVersion = visibleIDs, a.visibleVersion
	}
	a.FollowPreviewSelection()
}

func (a *App) visibleIDAt(index int) string {
	if index >= 0 && index < len(a.Visible) {
		return a.Visible[index].ID
	}
	return ""
}

// 和数据库里一样：成员 ID 去重并按升序排好。
func setMembership(members, ids []string, add bool) []string {
	out := slices.Clone(members)
	for _, id := range ids {
		if add {
			if !slices.Contains(out, id) {
				out = append(out, id)
			}
		} else {
			out = slices.DeleteFunc(out, func(member string) bool { return member == id })
		}
	}
	slices.Sort(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *App) applySnapshot(snapshot model.Snapshot) {
	a.Snapshot = snapshot
	a.snapshotVersion++
	a.Index = catalog.New(snapshot)
	a.indexVersion++
	a.assetByID = make(map[string]model.Asset, len(snapshot.Assets))
	for _, asset := range snapshot.Assets {
		a.assetByID[asset.ID] = asset
	}
	// 正在写和还在排队的操作已经提前画在界面上了，重读的快照里还没有它们，按顺序再叠一次，免得心形、回收站状态闪回去。
	pending := make([]*assetPatch, 0, len(a.queuedRuns)+1)
	if a.runningPatch != nil {
		pending = append(pending, a.runningPatch)
	}
	for _, queued := range a.queuedRuns {
		if queued.patch != nil && queued.patch.optimistic && queued.generation == a.generation {
			pending = append(pending, queued.patch)
		}
	}
	for _, patch := range pending {
		targets := make(map[string]bool, len(patch.ids))
		for _, id := range patch.ids {
			targets[id] = true
		}
		for _, index := range a.Index.Patch(targets, patch.apply) {
			asset := a.Snapshot.Assets[index]
			a.assetByID[asset.ID] = asset
		}
	}
	a.refine(false)
	a.pruneAutoImportTargets()
}

// 自动导入规则指向的文件夹被删掉（连同父文件夹一起删也算）后，改回导入到图库根目录；不然每次自动导入都失败，设置页却显示成根目录。
func (a *App) pruneAutoImportTargets() {
	libraryID := a.libraryID()
	rules := a.Preferences.AutoImport[libraryID]
	if len(rules) == 0 {
		return
	}
	exists := make(map[string]bool, len(a.Snapshot.Folders))
	for _, folder := range a.Snapshot.Folders {
		exists[folder.ID] = true
	}
	var next []AutoImportRule
	for i, rule := range rules {
		// 只改属于当前这个图库包的规则；在副本里删了文件夹，不能改到原库的规则。
		if rule.FolderID == "" || exists[rule.FolderID] || rule.Library != "" && a.Library != nil && !samePath(rule.Library, a.Library.Root) {
			continue
		}
		if next == nil {
			next = slices.Clone(rules)
		}
		slog.Warn("自动导入的目标文件夹已不存在，改为导入到图库根目录", "path", rule.Path, "folder", rule.FolderID)
		next[i].FolderID = ""
	}
	if next != nil {
		a.Preferences.AutoImport[libraryID] = next
		a.SavePreferences()
	}
}
func (a *App) Refine() { a.refine(true) }
func (a *App) refine(resetScroll bool) {
	if a.Index == nil {
		return
	}
	// 后台刷新（收藏、加标签、自动导入）后，Shift 范围选择的起点和当前项按素材找回新位置，不能直接丢掉。
	idAt := func(index int) string {
		if index >= 0 && index < len(a.Visible) {
			return a.Visible[index].ID
		}
		return ""
	}
	anchorID, currentID := idAt(a.anchor), idAt(a.selectedIndex)
	a.Visible = a.browseAssets(a.Query)
	a.visibleVersion++
	if resetScroll {
		if a.scroll.Y > 0 {
			a.contentEpoch++
		}
		a.scroll = ui.ScrollState{}
	}
	a.anchor, a.selectedIndex = -1, -1
	if anchorID != "" || currentID != "" {
		for i, asset := range a.Visible {
			if asset.ID == anchorID {
				a.anchor = i
			}
			if asset.ID == currentID {
				a.selectedIndex = i
			}
		}
	}
	if a.Selected.Len() > 0 {
		valid := make(map[string]bool, len(a.Visible))
		for _, v := range a.Visible {
			valid[v.ID] = true
		}
		for id := range a.Selected.All() {
			if !valid[id] {
				a.Selected.Remove(id)
			}
		}
	}
	a.FollowPreviewSelection()
}
func (a *App) Refresh() { a.Run("reload", func(ctx context.Context) error { return nil }) }
func (a *App) SetScope(scope, folderID string) {
	a.navigateScope(scope, folderID, false)
}
func (a *App) ChooseOpenLibrary() {
	go func() {
		paths, err := mygo.Dialog.Open(libraryOpenOptions(a.Window, a.T("open_library")))
		a.Window.Update(func() {
			if err != nil {
				a.ShowError(err)
			} else if len(paths) > 0 {
				a.OpenLibrary(paths[0])
			}
		})
	}()
}
func (a *App) OpenLibrary(path string) {
	a.FlushInspectorEdits()
	if a.InspectorEditsPending() {
		a.pendingLibraryAfterEdits = path
		return
	}

	if a.Busy {
		if a.Library == nil {
			a.pendingOpen = path
		} else {
			a.Notify(a.T("busy"))
		}
		return
	}
	a.Busy, a.libraryOpening = true, true
	go func() {
		a.workMu.Lock()
		defer a.workMu.Unlock()
		lib, err := library.Open(a.ctx, path)
		var iconErr error
		if err == nil {
			iconErr = libraryicon.Install(lib.Root)
		}
		var snapshot model.Snapshot
		if err == nil {
			snapshot, err = lib.Snapshot(a.ctx)
			if err != nil {
				lib.Close()
			}
		}

		a.Window.Update(func() {
			a.Busy, a.libraryOpening = false, false
			if err != nil {
				a.ShowError(err)
				// 外接盘没插这类情况保留在列表里，标成「找不到」，插上后还能直接打开。
				if errors.Is(err, os.ErrNotExist) {
					a.markRecentMissing(path, true)
				}
				// 图库没打开，排队的链接也导不进去了：告诉用户有几条没导入，不悄悄丢掉。
				if n := len(a.pendingImports); n > 0 {
					slog.Error("图库打开失败，排队的链接导入取消", "count", n)
					a.Error += "\n" + a.T("pending_imports_dropped", n)
					a.pendingImports = nil
				}
				a.drainPending()
				return
			}
			if a.Library != nil {
				a.freezePreviews()
				if a.Library.Root != lib.Root {
					a.ClearDragSessionsAsync(a.Library.Root, nil)
				}
				if err := a.Library.Close(); err != nil {
					a.ShowError(err)
				}
			}
			a.Library = lib
			a.markRecentMissing(lib.Root, false)
			a.Welcome.RecentChecked = false
			if iconErr != nil {
				a.ShowError(iconErr)
			}
			a.manifest = lib.Manifest
			a.Assets = assets.New(lib.Root, lib.DB)
			a.Assets.AppVersion = mygo.App.Version()
			a.generation++
			// 确认框是对旧图库说的（清空回收站、删除文件夹等），换了图库就关掉。
			a.ConfirmOpen, a.ConfirmAction = false, nil
			a.Browse = browseState{}
			a.GBrowsing = gBrowsingState{}
			a.Inspector = inspectorState{}
			a.Query = catalog.Query{Sort: a.Preferences.Sort, Descending: a.Preferences.SortDescending}
			a.Selected.Clear()
			a.TagPage = false
			a.StopSlideshow()
			a.ClosePreview()
			a.closeHoverVideo()
			a.Thumbnails.Clear()
			// 新图库从顶部开始看，不沿用上一个图库的滚动位置。
			a.scroll = ui.ScrollState{}
			a.applySnapshot(snapshot)
			a.Window.SetTitle(lib.Manifest.DisplayName + " – Momento")
			if !a.Preferences.Onboarded {
				a.finishOnboarding()
			}
			a.remember()
			a.ConfigureAutoImport()
			a.drainPending()
			slog.Info("打开图库", "path", lib.Root, "assets", len(snapshot.Assets))
		})
	}()
}
func (a *App) remember() {
	if a.recents == nil || a.Library == nil {
		return
	}
	m := a.manifest
	root := a.Library.Root
	gen := a.generation
	go func() {
		err := a.recents.Add(library.Recent{Path: root, LibraryID: m.LibraryID, DisplayName: m.DisplayName, Icon: m.Icon, LastOpened: time.Now()})
		var recent []library.Recent
		if err == nil {
			recent, err = a.recents.List()
		}
		a.Window.Update(func() {
			if gen != a.generation {
				return
			}
			if err != nil {
				a.ShowError(err)
			} else {
				a.Recent = recent
			}
		})
	}()
}
func (a *App) CreateLibraryAt(path, name, icon string) {
	if a.Busy {
		return
	}
	a.Busy = true
	go func() {
		lib, err := library.Create(a.ctx, path, name)
		if err == nil && icon != "" {
			err = lib.Edit(name, icon)
			if err != nil {
				err = errors.Join(err, lib.Close())
			}
		}
		if err == nil {
			err = lib.Close()
		}
		a.Window.Update(func() {
			a.Busy = false
			if err != nil {
				a.ShowError(err)
				a.drainPending()
			} else {
				a.NewLibraryOpen = false
				a.OpenLibrary(path)
			}
		})
	}()
}
func (a *App) EditLibrary(name, icon string) {
	lib := a.Library
	a.Run("edit_library", func(ctx context.Context) error { return lib.Edit(name, icon) })
}
func (a *App) ReorderRecent(paths []string) {
	if a.recents == nil {
		return
	}
	go func() {
		err := a.recents.Reorder(paths)
		var recent []library.Recent
		if err == nil {
			recent, err = a.recents.List()
		}
		a.Window.Update(func() {
			if err != nil {
				a.ShowError(err)
			} else {
				a.Recent = recent
			}
		})
	}()
}
func (a *App) DeleteLibrary() {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	// 写明删的是哪个图库、里面有多少素材，同时开着几个图库时不会删错。
	a.Confirm(a.T("delete_library"), a.T("delete_library_named", a.manifest.DisplayName, len(a.Snapshot.Assets)), a.T("delete_library"), func() {
		// 确认框开着时换了图库：确认的是旧图库，不能删掉或关掉现在这个。
		if a.Library != lib || a.Busy {
			a.Notify(a.T("delete_library_changed"))
			return
		}
		a.Busy = true
		go func() {
			a.workMu.Lock()
			defer a.workMu.Unlock()
			err := lib.Delete(a.ctx, mygo.Shell.TrashItem)
			a.Window.Update(func() {
				a.Busy = false
				if err != nil {
					a.ShowError(err)
					a.drainPending()
					return
				}
				if a.Library == lib {
					a.closeLibrary("library_deleted")
				}
				if err := a.recents.Remove(lib.Root); err != nil {
					a.ShowError(err)
				}
				a.Recent, _ = a.recents.List()
			})
		}()
	})
}

// 关掉当前图库时一起清掉跟它有关的东西：没保存的修改、预览窗口、自动导入监视、选择、缓存和滚动位置。
// 返回是否放弃了没保存的修改。图库连接本身由调用方关闭。
func (a *App) closeLibrary(reason string) bool {
	discarded := a.DiscardInspectorEdits(reason)
	a.StopSlideshow()
	a.ClosePreview()
	a.freezePreviews()
	a.closeHoverVideo()
	a.cancelExtensionPicker()
	if a.Import.WatchCancel != nil {
		a.Import.WatchCancel()
	}
	a.Import.WatchCancel, a.Import.WatchLibrary, a.Import.WatchRules = nil, "", nil
	for _, queued := range a.Import.queued {
		if queued.cleanup != nil {
			queued.cleanup()
		}
	}
	a.Import.queued, a.Import.manualCancel, a.Import.manualCanceled = nil, nil, false
	a.Library, a.Assets, a.Index = nil, nil, nil
	a.Snapshot, a.manifest = model.Snapshot{}, model.Manifest{}
	a.Visible, a.assetByID = nil, nil
	a.visibleVersion++
	a.Selected.Clear()
	a.queuedRuns, a.runningPatch = nil, nil
	// 确认框是对旧图库说的（清空回收站、删除文件夹等），换了图库就关掉，不能确认后作用到新图库上。
	a.ConfirmOpen, a.ConfirmAction = false, nil
	a.scroll = ui.ScrollState{}
	a.anchor, a.selectedIndex = -1, -1
	a.generation++
	if a.Thumbnails != nil {
		a.Thumbnails.Clear()
	}
	return discarded
}
func (a *App) RevealLibrary() {
	if a.Library != nil {
		mygo.Shell.ShowItemInFolder(a.Library.Root)
	}
}

func (a *App) ChooseImport(folder bool) {
	if a.Library == nil || a.Busy {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.Window, Title: a.T("import"), Directory: folder, Multiple: true})
		a.Window.Update(func() {
			if err != nil {
				a.ShowError(err)
			} else if len(paths) > 0 {
				a.ImportPaths(paths)
			}
		})
	}()
}
func (a *App) ImportPaths(paths []string) {
	// 拖进来的是 .momento 图库包：当作打开图库，不能把包里的原图和缩略图当新素材导进来。
	var libraries []string
	for _, path := range paths {
		if strings.EqualFold(filepath.Ext(filepath.Clean(path)), ".momento") {
			libraries = append(libraries, path)
		}
	}
	switch {
	case len(libraries) == 0:
		a.startPathImport(paths, a.Query.FolderID, "", nil)
	case len(libraries) == 1 && len(paths) == 1:
		a.OpenLibrary(libraries[0])
	default:
		a.ShowError(i18n.NewError("error_drop_library_mixed"))
	}
}

func (a *App) SetFavorite(ids []string, value bool) {
	lib := a.Library
	a.RunPatch("favorite", ids, func(ctx context.Context) error { return lib.SetFavorite(ctx, ids, value) }, func(asset *model.Asset) { asset.Favorite = value })
}
func (a *App) SetTags(ids, tagIDs []string, add bool) {
	lib := a.Library
	a.RunPatch("tags", ids, func(ctx context.Context) error { return lib.SetAssetTags(ctx, ids, tagIDs, add) }, func(asset *model.Asset) {
		asset.TagIDs = setMembership(asset.TagIDs, tagIDs, add)
	})
}
func (a *App) SetFolders(ids, folderIDs []string, add bool) {
	lib := a.Library
	a.RunPatch("folders", ids, func(ctx context.Context) error { return lib.SetAssetFolders(ctx, ids, folderIDs, add) }, func(asset *model.Asset) {
		asset.FolderIDs = setMembership(asset.FolderIDs, folderIDs, add)
	})
}
func (a *App) RequestDelete(ids []string) {
	service := a.Assets
	if service == nil || len(ids) == 0 {
		return
	}
	if a.Query.Scope == "trash" {
		a.Confirm(a.T("delete_permanently"), a.T("delete_permanently_detail", len(ids)), a.T("delete_permanently"), func() {
			a.Run("delete_permanently", func(ctx context.Context) error { return service.DeletePermanently(ctx, ids) })
		})
	} else {
		ids = slices.Clone(ids)
		a.RunPatch("trash", ids, func(ctx context.Context) error { return service.Trash(ctx, ids) }, func(asset *model.Asset) { asset.Trashed = true })
		gen := a.generation
		a.NotifyAction(a.T("moved_to_trash", len(ids)), a.T("undo"), func() {
			// 换了图库后这些 ID 不属于当前图库，撤销按钮就不做事。
			if gen == a.generation {
				a.restore(ids, false)
			}
		})
	}
}
func (a *App) Restore(ids []string) { a.restore(ids, true) }
func (a *App) restore(ids []string, notify bool) {
	service := a.Assets
	if service == nil || len(ids) == 0 {
		return
	}
	a.RunPatch("restore", ids, func(ctx context.Context) error { return service.Restore(ctx, ids) }, func(asset *model.Asset) { asset.Trashed = false })
	if notify {
		a.Notify(a.T("restored_assets", len(ids)))
	}
}

// 有一张没收藏就全部收藏，全都收藏了才取消；右键菜单的文字和 F 键都按这个判断。
func (a *App) FavoriteValue(ids []string) bool {
	for _, id := range ids {
		if !a.assetByID[id].Favorite {
			return true
		}
	}
	return false
}
func (a *App) ToggleFavorite(ids []string) {
	if len(ids) > 0 {
		a.SetFavorite(ids, a.FavoriteValue(ids))
	}
}
func (a *App) ExportSelected() {
	if a.Selected.Len() > 0 {
		a.ExportOpen = true
	}
}
func (a *App) ExportTo(destination string, options model.ExportOptions, selected []model.Asset) {
	service := a.Assets
	a.RunPatch("export", nil, func(ctx context.Context) error {
		written, err := service.ExportWithOptions(ctx, selected, destination, options)
		if err != nil {
			return err
		}
		// 写明导到了哪个文件夹：多张时可能自动改了名，单张就是保存框里选的位置。
		folder := destination
		if len(written) > 0 {
			folder = filepath.Dir(written[0])
		}
		a.Window.Update(func() {
			a.NotifyAction(a.T("assets_exported", len(written), filepath.Base(folder)), a.T("reveal_short"), func() {
				if len(written) == 1 {
					mygo.Shell.ShowItemInFolder(written[0])
				} else {
					mygo.Shell.ShowItemInFolder(folder)
				}
			})
		})
		return nil
	}, nil)
}
func (a *App) Bitmap(asset model.Asset) *ui.Bitmap {
	if a.Thumbnails == nil || a.Library == nil {
		return nil
	}
	return a.Thumbnails.Get(a.Library.Root, asset)
}
func (a *App) Reveal(asset model.Asset) {
	if a.Library != nil {
		mygo.Shell.ShowItemInFolder(filepath.Join(a.Library.Root, filepath.FromSlash(asset.RelativePath)))
	}
}

func (a *App) watchLibrary() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			// 检查图库还在不在要读文件，图库在 NAS 上断网时会卡几十秒，不能放在主线程。
			var watched *library.Library
			mygo.RunOnMain(func() {
				if !a.Busy {
					watched = a.Library
				}
			})
			if watched == nil || !watched.Missing() {
				continue
			}
			a.Window.Update(func() {
				if a.Library == watched && !a.Busy {
					lib := a.Library
					// 图库没了，没保存的修改也写不回去了：内容记进日志后清掉，不然打开别的图库和退出都会一直等它。
					discarded := a.closeLibrary("library_missing")
					go func() {
						a.workMu.Lock()
						defer a.workMu.Unlock()
						if err := lib.Close(); err != nil {
							slog.Error("关闭失踪图库", "error", err)
						}
					}()
					// 外接盘拔掉、图库被挪走时留在图库菜单里（显示找不到），插回来还能直接打开。
					a.markRecentMissing(lib.Root, true)
					a.Error = a.T("library_missing")
					if discarded {
						a.Error += "\n" + a.T("inspector_edits_discarded")
					}
				}
			})
		}
	}
}

func (a *App) shortcuts(c *ui.Context) {
	if a.TakePreviewInput() {
		return
	}
	if a.IsLocked() || a.Settings.RecordingShortcut != "" {
		return
	}
	if a.ActionShortcut(c, "lock") {
		a.LockNow()
		return
	}
	if a.PreviewOpen() {
		return
	}
	for _, action := range []string{"new_library", "open_library", "import", "import_folder", "export", "settings", "search", "filter", "sort", "inspector", "sidebar", "thumbnail_larger", "thumbnail_smaller", "view_masonry", "view_justified", "view_list", "slideshow"} {
		if !a.ActionShortcut(c, action) {
			continue
		}
		switch action {
		case "new_library":
			a.ShowNewLibrary()
		case "open_library":
			a.ChooseOpenLibrary()
		case "import":
			a.ChooseImport(false)
		case "import_folder":
			a.ChooseImport(true)
		case "export":
			a.ExportSelected()
		case "settings":
			a.ShowSettings()
		case "search":
			a.FocusSearch()
		case "filter":
			a.ToggleFilter()
		case "sort":
			a.ToggleSort()
		case "inspector":
			a.ToggleInspector()
		case "sidebar":
			a.SidebarVisible = !a.SidebarVisible
		case "thumbnail_larger":
			a.StepThumbnailSize(1)
		case "thumbnail_smaller":
			a.StepThumbnailSize(-1)
		case "view_masonry":
			a.ViewMode = 0
		case "view_justified":
			a.ViewMode = 1
		case "view_list":
			a.ViewMode = 2
		case "slideshow":
			a.StartSlideshowFromSelection()
		}
	}
	if c.FocusTakesText() {
		return
	}
	if a.ActionShortcut(c, "copy") {
		a.CopySelected()
	}
	if a.ActionShortcut(c, "paste") {
		a.PasteImport()
	}
	// 单按 F 是收藏，Shift+F 打开检查器里的文件夹选择器，直接搜着加。
	if a.ActionShortcut(c, "add_to_folder") && a.Selected.Len() > 0 && !a.TagPage && a.Query.Scope != "trash" {
		a.RevealInspector()
		a.Inspector.TagsOpen, a.Inspector.FoldersOpen = false, true
	}
	if a.ActionShortcut(c, "copy_tags") {
		a.CopyTags()
	}
	if a.ActionShortcut(c, "paste_tags") {
		a.PasteTags()
	}
	if a.ActionShortcut(c, "delete") || a.alternateDeleteKey(c) {
		a.RequestDelete(a.selectedIDs())
	}
	if c.Shortcut(0, ui.KeyEscape) {
		a.Selected.Clear()
		a.CollapseInspector()
		a.FollowPreviewSelection()
	}
	// 标签管理页看不到素材，全选会选中背后列表，接着删除就会误删。
	if a.ActionShortcut(c, "select_all") && !a.TagPage {
		for _, v := range a.Visible {
			a.Selected.Add(v.ID)
		}
		a.RevealInspector()
		a.FollowPreviewSelection()
	}
	if c.Shortcut(0, ui.KeyF) && a.Query.Scope != "trash" && !a.TagPage {
		a.ToggleFavorite(a.favoriteKeyTargets())
	}
	if a.ActionShortcut(c, "preview") && !a.PreviewOpen() {
		a.activatePreviewShortcut(c)
	}
}

// F 收藏：鼠标停在选区外的那张就只收藏它（和空格预览一样悬停优先），否则收藏全部选中的。
func (a *App) favoriteKeyTargets() []string {
	if a.hovered != "" && !a.Selected.Has(a.hovered) {
		return []string{a.hovered}
	}
	if a.Selected.Len() > 0 {
		return a.selectedIDs()
	}
	return nil
}

// 习惯按 Delete 键（Windows、Linux）或 ⌘⌫（Finder）删除的也能删；用户改过删除快捷键就只认改后的。
func (a *App) alternateDeleteKey(c *ui.Context) bool {
	if definition, ok := shortcutDefinition("delete"); !ok || a.Shortcut("delete") != definition.Default {
		return false
	}
	return c.Shortcut(0, ui.KeyDelete) || runtime.GOOS == "darwin" && c.Shortcut(ui.Super, ui.KeyBackspace)
}

func (a *App) SyncMenuFocus(c *ui.Context) {
	if !c.WindowActive() {
		return
	}
	focused := c.FocusTakesText()
	if focused != a.TextInputFocused || a.menuPreviewOpen != a.PreviewOpen() {
		a.TextInputFocused = focused
		a.menuPreviewOpen = a.PreviewOpen()
		if a.MenuChanged != nil {
			a.MenuChanged()
		}
	}
	if state := a.CurrentMenuState(); state != a.menuState {
		a.menuState = state
		if a.MenuStateChanged != nil {
			a.MenuStateChanged(state)
		}
	}
	if available := a.InspectorAvailable(); a.menuInspectorAvailable != available {
		a.menuInspectorAvailable = available
		if a.InspectorAvailabilityChanged != nil {
			a.InspectorAvailabilityChanged(available)
		}
	}
}

// MenuState 是菜单项能不能点所依赖的状态：没图库时导入、视图类没用，没选中时导出、复制没用，列表视图里缩略图大小没用。
type MenuState struct {
	Library, Selection, Busy, Grid, Preview bool
	// TextFocus 时复制、粘贴是给输入框用的系统角色，不能跟着素材状态灰掉。
	TextFocus bool
	// Browse：工具栏此刻有筛选、排序、视图按钮（不在标签管理页、也不是空图库）；Sortable：此处排序有作用。
	// 和工具栏用同一套条件，菜单里不留点了没反应的项。
	Browse, Sortable bool
}

func (a *App) CurrentMenuState() MenuState {
	return MenuState{
		Library:   a.Library != nil,
		Selection: a.Selected.Len() > 0 && !a.TagPage,
		Busy:      a.Busy,
		Grid:      a.ViewMode != 2 && !a.TagPage,
		Preview:   a.PreviewOpen(),
		TextFocus: a.TextInputFocused,
		Browse:    a.Library != nil && !a.TagPage && !a.toolbarBare(),
		Sortable:  a.toolbarSortable(),
	}
}

// MenuItemEnabled 按状态判断某个菜单项此刻能不能点；不在表里的菜单项一直可点。
func (s MenuState) MenuItemEnabled(id string) bool {
	switch id {
	case "import", "import_folder":
		return s.Library && !s.Busy
	case "export", "copy_tags", "paste_tags":
		return s.Library && s.Selection
	case "copy":
		return s.TextFocus || s.Library && s.Selection
	case "paste":
		return s.TextFocus || s.Library
	case "thumbnail_larger", "thumbnail_smaller":
		return s.Library && s.Browse && s.Grid
	case "view_masonry", "view_justified", "view_list", "filter":
		return s.Library && s.Browse
	case "sort":
		return s.Library && s.Browse && s.Sortable
	case "slideshow", "search":
		return s.Library
	case "preview_zoom_in", "preview_zoom_out", "preview_fit", "preview_actual":
		return s.Preview
	}
	return true
}

func fileSize(n int64) string {
	if n >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	}
	if n >= 1<<10 {
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

func (a *App) ToggleFilter() { a.Toolbar.FilterOpen = !a.Toolbar.FilterOpen }
func (a *App) ToggleSort()   { a.Toolbar.SortOpen = !a.Toolbar.SortOpen }

func (a *App) libraryID() string {
	if a.manifest.LibraryID != "" {
		return a.manifest.LibraryID
	}
	if a.Library != nil {
		return a.Library.Manifest.LibraryID
	}
	return ""
}
func (a *App) libraryName() string {
	if a.manifest.DisplayName != "" {
		return a.manifest.DisplayName
	}
	if a.Library != nil {
		return a.Library.Manifest.DisplayName
	}
	return ""
}
func (a *App) QueueImport(imageURL, pageURL string) {
	if a.Library == nil && !a.Busy {
		a.ShowError(i18n.NewError("library_required"))
		return
	}
	a.pendingImports = append(a.pendingImports, [2]string{imageURL, pageURL})
	a.drainPending()
}
func (a *App) drainPending() {
	a.ContinueInspectorEdits()
	a.completeInspectorTransitions()
	if a.Busy {
		return
	}
	if a.pendingOpen != "" {
		path := a.pendingOpen
		a.pendingOpen = ""
		if a.Library == nil || a.Library.Root != path {
			a.OpenLibrary(path)
			return
		}
	}
	// 排队的操作属于打开它们时的那个图库；换了图库就作废，不能写进新图库。
	for len(a.queuedRuns) > 0 && !a.Busy {
		next := a.queuedRuns[0]
		a.queuedRuns = a.queuedRuns[1:]
		if next.generation != a.generation || a.Library == nil {
			slog.Warn("图库已切换，丢弃排队的操作", "operation", next.label)
			if next.patch != nil && next.patch.failed != nil {
				next.patch.failed(errors.New("library changed before the write ran"))
			}
			continue
		}
		a.run(next.label, next.work, next.patch)
		return
	}
	if a.startQueuedPathImport() {
		return
	}
	if len(a.pendingImports) == 0 || a.Library == nil || a.remoteImporting {
		return
	}
	// 先占位：上一条还没开始写库时又来一条，不能两条同时去抢，抢输的那条以前会被丢掉。
	next := a.pendingImports[0]
	a.pendingImports = a.pendingImports[1:]
	a.remoteImporting = true
	go func() {
		err := a.ImportRemote(a.ctx, browser.Request{URL: next[0], PageURL: next[1]})
		var localized *i18n.LocalizedError
		busy := errors.As(err, &localized) && localized.Key == "busy"
		if err != nil {
			slog.Error("深度链接导入失败", "error", err, "requeued", busy)
		}
		a.Window.Update(func() {
			a.remoteImporting = false
			switch {
			case busy:
				// 正好碰上别的写入：放回队首，空闲后再来。
				a.pendingImports = append([][2]string{next}, a.pendingImports...)
			case errors.As(err, &localized) && localized.Key == "library_required":
				a.ShowError(err)
			}
			a.drainPending()
		})
	}()
}

func (a *App) applyLocale() {
	if a.Preferences.Language == "" || a.Preferences.Language == "system" {
		mygo.App.SetLocale("")
		a.Texts.SystemLanguage = mygo.App.Locale()
		return
	}
	mygo.App.SetLocale(a.Texts.Locale())
}

func (a *App) InvalidateWindows() {
	if a.Window != nil {
		a.Window.Invalidate()
	}
	if a.SettingsWindow != nil {
		a.SettingsWindow.Invalidate()
	}
	a.InvalidatePreviewWindows()
}

func (a *App) InspectorAvailable() bool { return a.Selected.Len() > 0 && !a.TagPage }

func (a *App) RevealInspector() {
	if a.InspectorAvailable() {
		a.InspectorVisible = true
	}
}

func (a *App) CollapseInspector() {
	if !a.InspectorVisible {
		return
	}
	a.saveInspectorEdits()
	a.InspectorVisible = false
}

func (a *App) ToggleInspector() {
	if a.InspectorVisible {
		a.CollapseInspector()
	} else {
		a.RevealInspector()
	}
}

func (a *App) HelpURL() string {
	if strings.HasPrefix(a.Texts.Locale(), "zh") {
		return "https://github.com/Seaony/Momento/blob/master/README.zh-CN.md"
	}
	return "https://github.com/Seaony/Momento#readme"
}

func (a *App) OpenHelp() {
	if err := mygo.Shell.OpenExternal(a.HelpURL()); err != nil {
		a.ShowError(err)
	}
}

func (a *App) SidebarShownWidth() float32 {
	if !a.chromeLayoutReady && a.SidebarVisible {
		return a.SidebarWidth
	}
	return a.sidebarShownWidth
}

func (a *App) InspectorShownWidth() float32 {
	if !a.chromeLayoutReady && a.InspectorVisible && a.InspectorAvailable() {
		return 280
	}
	return a.inspectorShownWidth
}

func (a *App) InspectorTargetWidth() float32 {
	if a.InspectorVisible && a.InspectorAvailable() {
		return 280
	}
	return 0
}

// 检查器展开会移动卡片，第二击仍归第一次点击的素材。
func (a *App) observeAssetDoubleClick(ev ui.InputEvent) {
	if ev.Kind != ui.InputPointerDown || ev.Button != 0 {
		return
	}
	a.pendingAssetActivation = ""
	if ev.Clicks == 2 && ev.Mods == 0 && !a.PreviewOpen() && a.Selected.Has(a.assetClickID) && a.assetClickBounds.Contains(ev.X, ev.Y) {
		a.pendingAssetActivation = a.assetClickID
	}
	a.assetClickID = ""
}

func (a *App) activateAssetDoubleClick(ev ui.InputEvent) bool {
	if ev.Kind != ui.InputPointerDown || a.pendingAssetActivation == "" {
		return false
	}
	id := a.pendingAssetActivation
	a.pendingAssetActivation = ""
	a.ActivateAsset(id, false)
	return true
}

func (a *App) observeWorkspaceInput(ev ui.InputEvent) {
	a.observeAssetDoubleClick(ev)
	a.ObservePreviewInput(ev)
}
