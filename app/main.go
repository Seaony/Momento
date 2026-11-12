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
	slog.Info("启动", "version", "2.0.0", "logDirectory", logging.Directory())
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
			app.Window.Restore()
			app.Window.Focus()
		}
	})
	mygo.App.OnQuit(func() { cancel() })
	mygo.App.WhenReady(func() {
		app = appui.New()
		app.OpenWindow()
		appui.OnUpdateChange(func() { app.Window.Invalidate() })
		app.MenuChanged = func() { installMenu(app) }
		installMenu(app)
		if pendingFile != "" {
			app.OpenLibrary(pendingFile)
		} else if len(app.Recent) > 0 {
			app.OpenLibrary(app.Recent[0].Path)
		}
		if pendingURL != "" {
			openURL(pendingURL)
		}
		go func() {
			if err := browser.Listen(ctx, browser.New(app.ImportRemote)); err != nil {
				app.Window.Update(func() { app.ShowError(err) })
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		slog.Error("应用退出失败", "error", err)
		os.Exit(1)
	}
	slog.Info("退出")
}

func installMenu(a *appui.App) {
	item := func(key, shortcut string, fn func()) *mygo.MenuItem {
		return &mygo.MenuItem{Label: a.T(key), Accelerator: shortcut, Click: func(_ *mygo.MenuItem, _ *mygo.Window) { fn(); a.Window.Invalidate() }}
	}
	menu := mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: a.T("menu_file"), Submenu: []*mygo.MenuItem{
			item("new_library", "CmdOrCtrl+N", a.ShowNewLibrary), item("open_library", "CmdOrCtrl+O", a.ChooseOpenLibrary), mygo.Separator(),
			item("import", "CmdOrCtrl+I", func() { a.ChooseImport(false) }), item("import_folder", "CmdOrCtrl+Shift+I", func() { a.ChooseImport(true) }), item("export", "CmdOrCtrl+E", a.ExportSelected), mygo.Separator(), {Role: mygo.RoleClose},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: a.T("menu_view"), Submenu: []*mygo.MenuItem{
			item("view_masonry", "CmdOrCtrl+1", func() { a.ViewMode = 0 }), item("view_grid", "CmdOrCtrl+2", func() { a.ViewMode = 1 }), item("view_list", "CmdOrCtrl+3", func() { a.ViewMode = 2 }), mygo.Separator(),
			item("search", "CmdOrCtrl+F", func() { a.Search.Focus() }), item("filter", "CmdOrCtrl+Alt+F", a.ToggleFilter), item("sort", "CmdOrCtrl+Alt+S", a.ToggleSort), item("inspector", "CmdOrCtrl+Alt+I", func() { a.InspectorVisible = !a.InspectorVisible }), item("sidebar", "", func() { a.SidebarVisible = !a.SidebarVisible }), mygo.Separator(), {Role: mygo.RoleToggleFullScreen},
		}},
		{Label: a.T("settings"), Submenu: []*mygo.MenuItem{item("settings", "CmdOrCtrl+,", func() { a.SettingsOpen = true })}},
		{Role: mygo.RoleWindowMenu},
	})
	if runtime.GOOS == "darwin" {
		mygo.App.SetMenu(menu)
	} else {
		a.Window.SetMenu(menu)
	}
}
