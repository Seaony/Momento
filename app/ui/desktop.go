package ui

import (
	"errors"
	"log/slog"
	"runtime"
	"slices"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
)

func (a *App) ShowMainWindow() {
	if a.Window == nil {
		return
	}
	if runtime.GOOS == "darwin" {
		mygo.App.SetActivationPolicy(mygo.ActivationPolicyRegular)
	}
	a.Window.Show()
	a.Window.Restore()
	a.Window.Focus()
	if a.WindowShown != nil {
		a.WindowShown()
	}
}

func (a *App) HideMainWindow() {
	if a.Window == nil {
		return
	}
	a.Window.Hide()
	if a.SettingsWindow != nil {
		a.SettingsWindow.Hide()
	}
	if a.WindowShown != nil {
		a.WindowShown()
	}
}

func (a *App) importFeedback(event string, result model.ImportResult) {
	if a.Preferences.SelectImported && (result.Imported > 0 || result.Restored > 0) && len(result.AssetIDs) > 0 && !a.IsLocked() {
		a.selectImportResult(result.AssetIDs)
	}
	message := a.T("import_complete", result.Imported, result.Skipped)
	a.Notify(message)
	if !a.Preferences.Notify || a.Window == nil || a.Window.IsFocused() || a.SettingsWindow != nil && a.SettingsWindow.IsFocused() {
		return
	}
	if !slices.Contains(a.Preferences.NotifyEvents, event) && !(result.Skipped > 0 && slices.Contains(a.Preferences.NotifyEvents, "duplicate")) {
		return
	}
	ids := slices.Clone(result.AssetIDs)
	root := ""
	if a.Library != nil {
		root = a.Library.Root
	}
	note := mygo.NewNotification(mygo.NotificationOptions{Title: "Momento", Body: message, Silent: !a.Preferences.NotifySound, Group: a.libraryID()})
	note.OnClick(func() {
		a.ShowMainWindow()
		if a.Library != nil && a.Library.Root == root && !a.IsLocked() {
			a.selectImportResult(ids)
		}
		note.Close()
	})
	ctx := a.ctx
	go func() {
		if err := note.ShowContext(ctx); err != nil {
			slog.Error("显示系统通知失败", "error", err)
			a.Window.Update(func() {
				if errors.Is(err, mygo.ErrNotificationsDenied) {
					a.Settings.NotificationDenied = true
				}
				a.InvalidateWindows()
			})
		}
	}()
}

func (a *App) selectImportResult(ids []string) {
	if len(ids) == 0 {
		return
	}
	a.Query.Search = ""
	a.SetScope("all", "")
	for _, id := range ids {
		if _, ok := a.assetByID[id]; ok {
			a.Selected.Add(id)
		}
	}
	a.RevealInspector()
}

func (a *App) closeMainWindow(e *mygo.CloseEvent) {
	if a.Quitting {
		if a.cancel != nil {
			a.cancel()
		}
		return
	}
	e.PreventDefault()
	a.FlushInspectorEdits()
	if a.Preferences.CloseAction == "background" {
		a.HideMainWindow()
		return
	}
	mygo.App.Quit()
}

func (a *App) PrepareInspectorQuit() bool {
	a.FlushInspectorEdits()
	if a.InspectorEditsPending() {
		a.quitAfterInspector = true
		return false
	}
	return true
}
func (a *App) completeInspectorTransitions() {
	if a.InspectorEditsFailed() {
		a.quitAfterInspector = false
		a.pendingLibraryAfterEdits = ""
		return
	}
	if a.Busy || a.InspectorEditsPending() {
		return
	}
	if a.quitAfterInspector {
		a.quitAfterInspector = false
		mygo.App.Quit()
		return
	}
	if path := a.pendingLibraryAfterEdits; path != "" {
		a.pendingLibraryAfterEdits = ""
		a.OpenLibrary(path)
	}
}
