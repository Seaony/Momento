package ui

import (
	"errors"
	"log/slog"
	"runtime"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
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
	a.CancelPreviewGesture()
	// 设置窗口跟着藏起来，正在录的快捷键和输了一半的密码一起作废。
	a.ResetSettingsSession()
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

// 设置窗口关掉或跟着主窗口藏起来时调用：正在录的快捷键作废（不然回来后所有快捷键都失效），
// 输了一半的密码清掉；密码正在校验或保存时不动，等结果出来自己关。
func (a *App) ResetSettingsSession() {
	a.CancelShortcutRecording()
	if a.Lock.DialogOpen && !a.PasswordOperationPending() {
		a.closePasswordDialog()
	}
}

// 每次打开设置窗口都重新读系统里「登录时启动」的真实状态，用户可能在系统设置里改过。
func (a *App) RefreshLaunchAtLogin() {
	a.Settings.LoginEnabled = launchAtLogin()
}

func (a *App) importFeedback(event string, result model.ImportResult) {
	a.importFeedbackFor(event, result, a.importDestination(), false)
}

type importDestination struct {
	root, id, name string
}

func (a *App) importDestination() importDestination {
	if a.Library == nil {
		return importDestination{}
	}
	return importDestination{root: a.Library.Root, id: a.libraryID(), name: a.manifest.DisplayName}
}

func (a *App) importFeedbackFor(event string, result model.ImportResult, destination importDestination, showLibraryName bool) {
	if a.Preferences.SelectImported && (result.Imported > 0 || result.Restored > 0) && len(result.AssetIDs) > 0 && !a.IsLocked() && a.importDestination().root == destination.root {
		a.selectImportResult(result.AssetIDs)
	}
	message := a.T("import_complete", result.Imported, result.Skipped)
	if len(result.Errors) > 0 {
		message = a.T("import_complete_failed", result.Imported, result.Skipped, len(result.Errors))
		a.showImportFailures(result.Errors)
	}
	if showLibraryName && destination.name != "" {
		message = a.T("import_complete_library", destination.name, message)
	}
	a.Notify(message)
	if !a.Preferences.Notify || a.Window == nil || a.Window.IsFocused() || a.SettingsWindow != nil && a.SettingsWindow.IsFocused() {
		return
	}
	if !slices.Contains(a.Preferences.NotifyEvents, event) && !(result.Skipped > 0 && slices.Contains(a.Preferences.NotifyEvents, "duplicate")) {
		return
	}
	ids := slices.Clone(result.AssetIDs)
	note := mygo.NewNotification(mygo.NotificationOptions{Title: "Momento", Body: message, Silent: !a.Preferences.NotifySound, Group: destination.id})
	note.OnClick(func() {
		a.ShowMainWindow()
		if a.Library != nil && a.Library.Root == destination.root && !a.IsLocked() {
			a.selectImportResult(ids)
		}
		note.Close()
	})
	ctx := a.ctx
	go func() {
		err := note.ShowContext(ctx)
		if err != nil {
			slog.Error("显示系统通知失败", "error", err)
		}
		a.Window.Update(func() { a.notificationShown(err) })
	}()
}

// 通知被系统拒绝时在设置里提示；用户重新允许后，下一条通知发得出去，提示就收起来。
func (a *App) notificationShown(err error) {
	denied := errors.Is(err, mygo.ErrNotificationsDenied)
	if err != nil && !denied {
		return
	}
	if a.Settings.NotificationDenied != denied {
		a.Settings.NotificationDenied = denied
		a.InvalidateWindows()
	}
}

// 坏图被跳过时，导入结束后列出失败的文件和原因；太多时只列前 20 条，全部写进日志。
func (a *App) showImportFailures(errs []error) {
	const shown = 20
	lines := []string{a.T("import_failed_files", len(errs))}
	for i, err := range errs {
		slog.Error("导入时跳过文件", "error", err)
		if i < shown {
			lines = append(lines, a.Texts.Error(err))
		}
	}
	if len(errs) > shown {
		lines = append(lines, a.T("import_failed_more", len(errs)-shown))
	}
	a.Error = strings.Join(lines, "\n")
}

func (a *App) selectImportResult(ids []string) {
	if len(ids) == 0 {
		return
	}
	if (a.Query.Scope == "all" || a.Query.Scope == "") && !a.TagPage && a.Query.Search == "" {
		a.Selected.Clear()
		compare := catalog.AssetOrder(a.Query)
		missing := false
		for _, id := range ids {
			if asset, exists := a.assetByID[id]; exists {
				if _, visible := slices.BinarySearchFunc(a.Visible, &asset, compare); visible {
					a.Selected.Add(id)
				} else if a.Index.Matches(a.Query, id) {
					missing = true
					a.Selected.Add(id)
				}
			}
		}
		a.scroll = ui.ScrollState{}
		a.Toolbar.SortOpen = false
		if missing {
			a.refine(true)
		}
	} else {
		a.Query.Search = ""
		a.applyBrowseSelection(browseLocation{Scope: "all"}, ids)
	}
	a.RevealInspector()
	a.FollowPreviewSelection()
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
		// 写不进去（磁盘满、只读卷）时，给一个放弃修改的出口，不然既打不开别的图库也退不出去。
		if quit, path := a.quitAfterInspector, a.pendingLibraryAfterEdits; quit || path != "" {
			a.Error = "" // 原因放进确认框里，不叠两个弹窗
			a.Confirm(a.T("inspector_save_failed"), a.T("inspector_save_failed_detail", a.Inspector.SaveError), a.T("discard_changes"), func() {
				a.DiscardInspectorEdits("save_failed")
				if quit {
					mygo.App.Quit()
				} else {
					a.OpenLibrary(path)
				}
			})
		}
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
