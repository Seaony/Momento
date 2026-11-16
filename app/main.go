package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/logging"
	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
)

func main() {
	logs, err := logging.Open()
	if err != nil {
		slog.Error("无法创建日志", "error", err)
		os.Exit(1)
	}
	defer logs.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	mygo.App.SetName("Momento")
	mygo.App.SetLanguages("en", "zh-Hans", "ja")
	appui.InitUpdater()
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	slog.Info("启动", "version", mygo.App.Version(), "logDirectory", logging.Directory())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var app *appui.App
	var pendingFile, pendingURL string
	openURL := func(raw string) {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "momento" || u.Host != "import" || u.Query().Get("url") == "" {
			if app != nil {
				app.ShowError(fmt.Errorf("%s", app.T("invalid_import_url")))
			}
			return
		}
		imageURL := u.Query().Get("url")
		app.QueueImport(imageURL, "")
	}
	mygo.App.OnOpenFile(func(path string) {
		if app == nil {
			pendingFile = path
		} else {
			app.OpenLibrary(path)
		}
	})
	mygo.App.OnOpenURL(func(raw string) {
		if app == nil {
			pendingURL = raw
		} else {
			openURL(raw)
		}
	})
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if app != nil {
			app.ShowMainWindow()
		}
	})
	mygo.App.OnQuit(func() { cancel() })
	preferencesQuitWarned := false
	mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
		if app != nil {
			if app.PasswordOperationPending() && !e.SessionEnding {
				e.PreventDefault()
				app.Notify(app.T("busy"))
				app.InvalidateWindows()
				return
			}
			if !e.SessionEnding && !app.PrepareInspectorQuit() {
				e.PreventDefault()
				app.InvalidateWindows()
				return
			}
			// 设置没存上：第一次拦下退出，把窗口叫出来让用户看到原因（窗口可能在后台，从托盘退出时以前就是「点了没反应」）；
			// 用户看过后再退出一次就照常退出，最近的设置改动不保存，原因已经记在日志里。
			if err := app.FlushPreferences(); err != nil && !e.SessionEnding {
				if !preferencesQuitWarned {
					preferencesQuitWarned = true
					e.PreventDefault()
					app.Error += "\n" + app.T("quit_without_saving_preferences")
					app.ShowMainWindow()
					app.InvalidateWindows()
					return
				}
				slog.Error("设置保存失败，按用户再次要求退出", "error", err)
			}
			app.CloseAllPreviewWindows()
			app.Quitting = true
			// 拖出给别的应用的临时副本，退出时清掉；清不掉只能记日志，窗口马上就关了。
			if app.Library != nil {
				if err := appui.ClearDragSessions(app.Library.Root); err != nil {
					slog.Error("退出时清理拖出临时文件失败", "error", err)
				}
			}
		}
	})
	mygo.App.OnQuitCanceled(func() {
		if app != nil {
			app.Quitting = false
		}
	})
	mygo.App.WhenReady(func() {
		app = appui.New()
		app.OpenWindow()
		if err := installLibraryFileAssociation(); err != nil {
			app.ShowError(err)
		}
		app.SettingsRequested = func() { showSettingsWindow(app) }
		appui.OnUpdateChange(app.InvalidateWindows)
		desktop := installDesktop(app)
		app.MenuChanged = func() { installMenu(app); desktop.refresh() }
		installMenu(app)
		mygo.App.OnLocaleChanged(func(locale string) {
			if app.Preferences.Language == "" || app.Preferences.Language == "system" {
				app.Texts.SystemLanguage = locale
				app.MenuChanged()
				app.InvalidateWindows()
			}
		})
		if pendingFile != "" {
			app.OpenLibrary(pendingFile)
		} else if recent, ok := library.MostRecent(app.Recent); ok {
			app.OpenLibrary(recent.Path)
		}
		if pendingURL != "" {
			openURL(pendingURL)
		}
		app.Extension = browser.NewService(ctx, browser.New(browser.Options{Version: mygo.App.Version(), Import: app.ImportRemote}), func(browser.Status) {
			app.Window.Update(func() {
				app.ExtensionStatus = app.Extension.Status()
				app.InvalidateWindows()
			})
		})
		app.ExtensionStatus = app.Extension.Status()
		app.SetExtensionEnabled(app.Preferences.BrowserExtensionEnabled)
		appui.StartAutomaticUpdateCheck(app.Preferences.AutoCheckUpdates)
	})
	if err := mygo.App.Run(); err != nil {
		slog.Error("应用退出失败", "error", err)
		os.Exit(1)
	}
	slog.Info("退出")
}

func installMenu(a *appui.App) {
	var menus []*mygo.Menu
	makeMenu := func() *mygo.Menu {
		menu := mygo.NewMenu(menuItems(a, runtime.GOOS))
		menus = append(menus, menu)
		for _, definition := range appui.ShortcutDefinitions {
			if item := menu.ItemByID(definition.Action); item != nil {
				item.SetAccelerator(a.MenuAccelerator(definition.Action))
			}
		}
		return menu
	}
	if runtime.GOOS == "darwin" {
		mygo.App.SetMenu(makeMenu())
	} else {
		a.Window.SetMenu(makeMenu())
		if a.SettingsWindow != nil {
			a.SettingsWindow.SetMenu(makeMenu())
		}
	}
	if a.SettingsWindow != nil {
		a.SettingsWindow.SetTitle(a.T("settings"))
	}
	a.InspectorAvailabilityChanged = func(available bool) {
		for _, menu := range menus {
			if item := menu.ItemByID("inspector"); item != nil {
				item.SetEnabled(available)
			}
		}
	}
}

func menuItems(a *appui.App, platform string) []*mygo.MenuItem {
	item := func(key, shortcut string, fn func()) *mygo.MenuItem {
		action := key
		switch key {
		case "lock_now":
			action = "lock"
		case "paste_import":
			action = "paste"
		case "preview_fit_window":
			action = "preview_fit"
		case "preview_actual_size":
			action = "preview_actual"
		}
		accelerator := a.Shortcut(action)
		if a.Settings.RecordingShortcut != "" {
			accelerator = ""
		}
		return &mygo.MenuItem{ID: action, Label: a.T(key), Accelerator: accelerator, Click: func(*mygo.MenuItem, *mygo.Window) { fn(); a.RememberUIState(); a.InvalidateWindows() }}
	}
	role := func(action, key string, value mygo.MenuRole) *mygo.MenuItem {
		return &mygo.MenuItem{ID: action, Label: a.T(key), Role: value, Accelerator: a.Shortcut(action)}
	}
	if a.Settings.RecordingShortcut != "" {
		return []*mygo.MenuItem{{Label: "Momento", Submenu: []*mygo.MenuItem{{Label: a.T("settings_shortcuts"), Disabled: true}}}}
	}
	if a.IsLocked() {
		// 锁屏时也要能用退出快捷键（和平时同一个），不然 ⌘Q 没反应。
		quit := item("quit_momento", "", mygo.App.Quit)
		quit.Accelerator = a.Shortcut("quit")
		return []*mygo.MenuItem{{Label: "Momento", Submenu: []*mygo.MenuItem{item("show_momento", "", a.ShowMainWindow), quit}}}
	}
	settings := item("settings", "CmdOrCtrl+,", a.ShowSettings)
	lock := item("lock_now", "CmdOrCtrl+L", a.LockNow)
	lock.Disabled = a.Preferences.PasswordHash == ""
	fileItems := []*mygo.MenuItem{
		item("new_library", "CmdOrCtrl+N", a.ShowNewLibrary), item("open_library", "CmdOrCtrl+O", a.ChooseOpenLibrary), mygo.Separator(),
		item("import", "CmdOrCtrl+I", func() { a.ChooseImport(false) }), item("import_folder", "CmdOrCtrl+Shift+I", func() { a.ChooseImport(true) }), item("export", "CmdOrCtrl+E", a.ExportSelected),
	}
	appMenu := &mygo.MenuItem{Role: mygo.RoleAppMenu}
	if platform == "darwin" {
		appMenu.Submenu = []*mygo.MenuItem{{Role: mygo.RoleAbout}, mygo.Separator(), settings, item("check_updates", "", func() { a.Settings.Tab = appui.SettingsGeneral; a.ShowSettings(); appui.CheckForUpdates() }), lock, mygo.Separator(), {Role: mygo.RoleServices}, mygo.Separator(), {Role: mygo.RoleHide}, {Role: mygo.RoleHideOthers}, {Role: mygo.RoleUnhide}, mygo.Separator(), role("quit", "quit_momento", mygo.RoleQuit)}
	} else {
		fileItems = append(fileItems, mygo.Separator(), settings, lock)
	}
	fileItems = append(fileItems, mygo.Separator(), role("close_window", "shortcut_close_window", mygo.RoleClose))
	paste := item("paste_import", "CmdOrCtrl+V", a.PasteImport)
	if a.TextInputFocused {
		paste = role("paste", "shortcut_paste", mygo.RolePaste)
	}
	copyItem := role("copy", "shortcut_copy", mygo.RoleCopy)
	if !a.TextInputFocused {
		copyItem = item("shortcut_copy", "CmdOrCtrl+C", a.CopySelected)
		copyItem.ID = "copy"
		copyItem.Accelerator = a.Shortcut("copy")
	}
	edit := []*mygo.MenuItem{role("undo", "shortcut_undo", mygo.RoleUndo), role("redo", "shortcut_redo", mygo.RoleRedo), mygo.Separator(), role("cut", "shortcut_cut", mygo.RoleCut), copyItem, paste, role("select_all", "select_all", mygo.RoleSelectAll)}
	inspector := item("inspector", "CmdOrCtrl+Alt+I", a.ToggleInspector)
	inspector.Disabled = !a.InspectorAvailable()
	return []*mygo.MenuItem{appMenu, {Label: a.T("menu_file"), Submenu: fileItems}, {Label: a.T("menu_edit"), Submenu: edit}, {Label: a.T("menu_view"), Submenu: []*mygo.MenuItem{
		item("view_masonry", "CmdOrCtrl+1", func() { a.ViewMode = 0 }), item("view_list", "CmdOrCtrl+2", func() { a.ViewMode = 2 }), mygo.Separator(),
		item("thumbnail_larger", "CmdOrCtrl+=", func() { a.StepThumbnailSize(1) }), item("thumbnail_smaller", "CmdOrCtrl+-", func() { a.StepThumbnailSize(-1) }), mygo.Separator(),
		item("search", "CmdOrCtrl+K", a.FocusSearch), item("filter", "CmdOrCtrl+Alt+F", a.ToggleFilter), item("sort", "CmdOrCtrl+Alt+S", a.ToggleSort), inspector, item("sidebar", "", func() { a.SidebarVisible = !a.SidebarVisible }), mygo.Separator(), role("fullscreen", "shortcut_fullscreen", mygo.RoleToggleFullScreen),
	}}, {Label: a.T("preview"), Submenu: []*mygo.MenuItem{
		item("preview_zoom_in", "CmdOrCtrl+=", a.PreviewZoomIn), item("preview_zoom_out", "CmdOrCtrl+-", a.PreviewZoomOut), item("preview_fit_window", "CmdOrCtrl+0", a.PreviewFitToWindow), item("preview_actual_size", "CmdOrCtrl+1", a.PreviewActualSize),
	}}, {Role: mygo.RoleWindowMenu}, {Label: a.T("help"), Submenu: []*mygo.MenuItem{item("user_guide", "", a.OpenHelp)}}}
}
