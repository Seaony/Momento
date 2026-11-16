package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
	EXIFCollapsed           bool                        `json:"exifCollapsed"`
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
	SettingsOpen, NewLibraryOpen, EditLibraryOpen, ExportOpen bool
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
	assetClickID, pendingAssetActivation                      string
	assetClickBounds                                          ui.Rect
	layout                                                    *MasonryLayout
	layoutWidth                                               float32
	layoutVersion                                             uint64
	list                                                      listLayout
	focusAssetID                                              string // 方向键移动选择后，下一帧把键盘焦点交给这张卡片
	visibleVersion                                            uint64
	snapshotVersion                                           uint64
	selection                                                 selectionCache
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
		workspace := ui.Row(c.Key("workspace")).Fill().ObserveInput(a.observeWorkspaceInput).HandleInput(a.activateAssetDoubleClick)
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
	a.passwordSettingsDialog(c)
	if a.Notice != "" {
		c.Toast(a.Notice)
		a.Notice = ""
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

func (a *App) Run(label string, work func(context.Context) error) {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	gen := a.generation
	a.Busy = true
	go func() {
		a.workMu.Lock()
		defer a.workMu.Unlock()
		slog.Info("操作开始", "operation", label)
		err := work(a.ctx)
		var snapshot model.Snapshot
		if err == nil {
			snapshot, err = lib.Snapshot(a.ctx)
		}
		a.Window.Update(func() {
			if gen != a.generation {
				return
			}
			a.Busy = false
			if err != nil {
				a.ShowError(err)
				a.drainPending()
				return
			}
			a.manifest = lib.Manifest
			a.applySnapshot(snapshot)
			if label == "edit_library" {
				a.Window.SetTitle(a.manifest.DisplayName + " – Momento")
				a.remember()
			}
			a.drainPending()
			slog.Info("操作完成", "operation", label)
		})
	}()
}

func (a *App) applySnapshot(snapshot model.Snapshot) {
	a.Snapshot = snapshot
	a.snapshotVersion++
	a.Index = catalog.New(snapshot)
	a.assetByID = make(map[string]model.Asset, len(snapshot.Assets))
	for _, asset := range snapshot.Assets {
		a.assetByID[asset.ID] = asset
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
	a.Visible = a.browseAssets(a.Query)
	a.visibleVersion++
	if resetScroll {
		a.scroll = ui.ScrollState{}
	}
	a.anchor = -1
	a.selectedIndex = -1
	valid := make(map[string]bool, len(a.Visible))
	for _, v := range a.Visible {
		valid[v.ID] = true
	}
	for id := range a.Selected.All() {
		if !valid[id] {
			a.Selected.Remove(id)
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
	a.Busy = true
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

		var recents []library.Recent
		if err != nil && os.IsNotExist(err) && a.recents != nil {
			if removeErr := a.recents.Remove(path); removeErr != nil {
				err = errors.Join(err, removeErr)
			}
			recents, _ = a.recents.List()
		}
		a.Window.Update(func() {
			a.Busy = false
			if err != nil {
				a.ShowError(err)
				if recents != nil {
					a.Recent = recents
				}
				a.pendingImports = nil
				a.drainPending()
				return
			}
			if a.Library != nil {
				if err := a.Library.Close(); err != nil {
					a.ShowError(err)
				}
			}
			a.Library = lib
			if iconErr != nil {
				a.ShowError(iconErr)
			}
			a.manifest = lib.Manifest
			a.Assets = assets.New(lib.Root, lib.DB)
			a.Assets.AppVersion = mygo.App.Version()
			a.generation++
			a.Browse = browseState{}
			a.GBrowsing = gBrowsingState{}
			a.Inspector = inspectorState{}
			a.Query = catalog.Query{Sort: a.Preferences.Sort, Descending: a.Preferences.SortDescending}
			a.Selected.Clear()
			a.TagPage = false
			a.ClosePreview()
			a.Thumbnails.Clear()
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
	a.Confirm(a.T("delete_library"), a.T("delete_library_detail"), a.T("delete"), func() {
		a.Busy = true
		go func() {
			a.workMu.Lock()
			defer a.workMu.Unlock()
			err := lib.Delete(a.ctx, mygo.Shell.TrashItem)
			a.Window.Update(func() {
				a.Busy = false
				if err != nil {
					a.ShowError(err)
					return
				}
				a.DiscardInspectorEdits("library_deleted")
				a.Library = nil
				a.Assets = nil
				a.Index = nil
				a.Snapshot = model.Snapshot{}
				a.Visible = nil
				a.generation++
				a.Thumbnails.Clear()
				if err := a.recents.Remove(lib.Root); err != nil {
					a.ShowError(err)
				}
				a.Recent, _ = a.recents.List()
			})
		}()
	})
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
	a.startPathImport(paths, a.Query.FolderID, "", nil)
}

func (a *App) SetFavorite(ids []string, value bool) {
	lib := a.Library
	a.Run("favorite", func(ctx context.Context) error { return lib.SetFavorite(ctx, ids, value) })
}
func (a *App) SetTags(ids, tagIDs []string, add bool) {
	lib := a.Library
	a.Run("tags", func(ctx context.Context) error { return lib.SetAssetTags(ctx, ids, tagIDs, add) })
}
func (a *App) SetFolders(ids, folderIDs []string, add bool) {
	lib := a.Library
	a.Run("folders", func(ctx context.Context) error { return lib.SetAssetFolders(ctx, ids, folderIDs, add) })
}
func (a *App) RequestDelete(ids []string) {
	service := a.Assets
	if service == nil || len(ids) == 0 {
		return
	}
	if a.Query.Scope == "trash" {
		a.Confirm(a.T("delete_permanently"), a.T("delete_permanently_detail", len(ids)), a.T("delete"), func() {
			a.Run("delete_permanently", func(ctx context.Context) error { return service.DeletePermanently(ctx, ids) })
		})
	} else {
		a.Run("trash", func(ctx context.Context) error { return service.Trash(ctx, ids) })
	}
}
func (a *App) Restore(ids []string) {
	service := a.Assets
	a.Run("restore", func(ctx context.Context) error { return service.Restore(ctx, ids) })
}
func (a *App) ExportSelected() {
	if a.Selected.Len() > 0 {
		a.ExportOpen = true
	}
}
func (a *App) ExportTo(destination string, format model.ExportFormat, quality float64) {
	service := a.Assets
	selected := a.SelectedAssets()
	a.Run("export", func(ctx context.Context) error {
		_, err := service.Export(ctx, selected, destination, format, quality)
		return err
	})
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
			a.Window.Update(func() {
				if a.Library != nil && !a.Busy && a.Library.Missing() {
					lib := a.Library
					a.Library = nil
					a.Assets = nil
					a.generation++
					a.Thumbnails.Clear()
					go func() {
						a.workMu.Lock()
						defer a.workMu.Unlock()
						if err := lib.Close(); err != nil {
							slog.Error("关闭失踪图库", "error", err)
						}
					}()
					if err := a.recents.Remove(lib.Root); err != nil {
						a.ShowError(err)
					}
					a.Recent, _ = a.recents.List()
					a.Error = a.T("library_missing")
					// 图库没了，没保存的修改也写不回去了：内容记进日志后清掉，不然打开别的图库和退出都会一直等它。
					if a.DiscardInspectorEdits("library_missing") {
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
	for _, action := range []string{"new_library", "open_library", "import", "import_folder", "export", "settings", "search", "filter", "sort", "inspector", "sidebar", "thumbnail_larger", "thumbnail_smaller", "view_masonry", "view_list"} {
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
		case "view_list":
			a.ViewMode = 2
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
	if a.ActionShortcut(c, "delete") {
		a.RequestDelete(a.selectedIDs())
	}
	if c.Shortcut(0, ui.KeyEscape) {
		a.Selected.Clear()
		a.CollapseInspector()
		a.FollowPreviewSelection()
	}
	if a.ActionShortcut(c, "select_all") {
		for _, v := range a.Visible {
			a.Selected.Add(v.ID)
		}
		a.RevealInspector()
		a.FollowPreviewSelection()
	}
	if c.Shortcut(0, ui.KeyF) && a.hovered != "" {
		if asset, ok := a.assetByID[a.hovered]; ok {
			a.SetFavorite([]string{asset.ID}, !asset.Favorite)
		}
	}
	if a.ActionShortcut(c, "preview") && !a.PreviewOpen() {
		a.activatePreviewShortcut(c)
	}
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
	if available := a.InspectorAvailable(); a.menuInspectorAvailable != available {
		a.menuInspectorAvailable = available
		if a.InspectorAvailabilityChanged != nil {
			a.InspectorAvailabilityChanged(available)
		}
	}
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

// 扩展请求等待真实导入结果，不能在排队后就返回成功。
func (a *App) ImportRemote(ctx context.Context, imageURL, pageURL string) error {
	choice := importChoice{}
	picker := false
	mygo.RunOnMain(func() { picker = a.Preferences.ExtensionPicker && !a.IsLocked() })
	if picker {
		var err error
		choice, err = a.remoteImportChoice(ctx)
		if err != nil {
			return err
		}
	}
	var service *assets.Service
	var lib *library.Library
	var gen uint64
	var request model.ImportRequest
	readyErr := error(context.Canceled)
	mygo.RunOnMain(func() {
		if a.Library == nil {
			readyErr = i18n.NewError("library_required")
			return
		}
		if a.Busy {
			readyErr = i18n.NewError("busy")
			return
		}
		readyErr = nil
		a.Busy = true
		service = a.Assets
		lib = a.Library
		gen = a.generation
		request = model.ImportRequest{FolderID: choice.FolderID, TagIDs: choice.TagIDs, ImportKeywords: a.Preferences.ImportKeywords}
	})
	if readyErr != nil {
		return readyErr
	}
	return a.performRemote(ctx, service, lib, gen, imageURL, pageURL, request)
}
func (a *App) performRemote(ctx context.Context, service *assets.Service, lib *library.Library, gen uint64, imageURL, pageURL string, request model.ImportRequest) error {
	a.workMu.Lock()
	defer a.workMu.Unlock()
	result, err := service.ImportURLWithRequest(ctx, imageURL, pageURL, request)
	var snapshot model.Snapshot
	if err == nil {
		snapshot, err = lib.Snapshot(ctx)
	}
	a.Window.Update(func() {
		if gen != a.generation {
			return
		}
		a.Busy = false
		if err != nil {
			a.ShowError(err)
		} else {
			a.applySnapshot(snapshot)
			a.importFeedback("extension", result)
		}
		a.drainPending()
	})
	return err
}

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
	if len(a.pendingImports) == 0 || a.Library == nil {
		return
	}
	next := a.pendingImports[0]
	a.pendingImports = a.pendingImports[1:]
	go func() {
		if err := a.ImportRemote(a.ctx, next[0], next[1]); err != nil {
			slog.Error("深度链接导入失败", "error", err)
		}
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
