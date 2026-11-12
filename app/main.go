package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"

	"github.com/Seaony/Momento/app/internal/browser"
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
			if err := app.FlushPreferences(); err != nil && !e.SessionEnding {
				e.PreventDefault()
				app.InvalidateWindows()
				return
			}
			app.Quitting = true
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
		} else if len(app.Recent) > 0 {
			app.OpenLibrary(app.Recent[0].Path)
		}
		if pendingURL != "" {
			openURL(pendingURL)
		}
		app.Extension = browser.NewService(ctx, browser.New(app.ImportRemote), func(browser.Status) {
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
	makeMenu := func() *mygo.Menu {
		menu := mygo.NewMenu(menuItems(a, runtime.GOOS))
		for _, definition := range appui.ShortcutDefinitions {
			if item := menu.ItemByID(definition.Action); item != nil {
				accelerator := a.Shortcut(definition.Action)
				if a.Settings.RecordingShortcut != "" {
					accelerator = ""
				}
				if a.Preview.Open && definition.Group != "preview" && (definition.Action == "view_masonry" || definition.Action == "thumbnail_larger" || definition.Action == "thumbnail_smaller") {
					accelerator = ""
				}
				if !a.Preview.Open && definition.Group == "preview" {
					accelerator = ""
				}
				item.SetAccelerator(accelerator)
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
}

func menuItems(a *appui.App, platform string) []*mygo.MenuItem {
	item := func(key, shortcut string, fn func()) *mygo.MenuItem {
		action := key
		switch key {
		case "lock_now":
			action = "lock"
		case "paste_import":
			action = "paste"
		case "navigate_back":
			action = "back"
		case "navigate_forward":
			action = "forward"
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
		return []*mygo.MenuItem{{Label: "Momento", Submenu: []*mygo.MenuItem{item("show_momento", "", a.ShowMainWindow), item("quit_momento", "", mygo.App.Quit)}}}
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
	edit := []*mygo.MenuItem{role("undo", "shortcut_undo", mygo.RoleUndo), role("redo", "shortcut_redo", mygo.RoleRedo), mygo.Separator(), role("cut", "shortcut_cut", mygo.RoleCut), role("copy", "shortcut_copy", mygo.RoleCopy), paste, role("select_all", "select_all", mygo.RoleSelectAll)}
	back := item("navigate_back", "CmdOrCtrl+[", a.NavigateBack)
	back.Disabled = !a.CanNavigateBack()
	forward := item("navigate_forward", "CmdOrCtrl+]", a.NavigateForward)
	forward.Disabled = !a.CanNavigateForward()
	return []*mygo.MenuItem{appMenu, {Label: a.T("menu_file"), Submenu: fileItems}, {Label: a.T("menu_edit"), Submenu: edit}, {Label: a.T("menu_view"), Submenu: []*mygo.MenuItem{
		item("view_masonry", "CmdOrCtrl+1", func() { a.ViewMode = 0 }), item("view_grid", "CmdOrCtrl+2", func() { a.ViewMode = 1 }), item("view_list", "CmdOrCtrl+3", func() { a.ViewMode = 2 }), mygo.Separator(), back, forward,
		item("thumbnail_larger", "CmdOrCtrl+=", func() { a.ChangeThumbnailSize(.1) }), item("thumbnail_smaller", "CmdOrCtrl+-", func() { a.ChangeThumbnailSize(-.1) }), mygo.Separator(),
		item("search", "CmdOrCtrl+F", func() { a.Search.Focus() }), item("filter", "CmdOrCtrl+Alt+F", a.ToggleFilter), item("sort", "CmdOrCtrl+Alt+S", a.ToggleSort), item("inspector", "CmdOrCtrl+Alt+I", func() { a.InspectorVisible = !a.InspectorVisible }), item("sidebar", "", func() { a.SidebarVisible = !a.SidebarVisible }), mygo.Separator(), role("fullscreen", "shortcut_fullscreen", mygo.RoleToggleFullScreen),
	}}, {Label: a.T("preview"), Submenu: []*mygo.MenuItem{
		item("preview_zoom_in", "CmdOrCtrl+=", a.PreviewZoomIn), item("preview_zoom_out", "CmdOrCtrl+-", a.PreviewZoomOut), item("preview_fit_window", "CmdOrCtrl+0", a.PreviewFitToWindow), item("preview_actual_size", "CmdOrCtrl+1", a.PreviewActualSize),
	}}, {Role: mygo.RoleWindowMenu}}
}
