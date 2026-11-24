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
	// 关掉设置窗口就放下没确认的自动导入弹窗，下次打开不会冒出上次的。
	a.Settings.AutoImportConfirm = nil
}

// 每次打开设置窗口都重新读系统里「登录时启动」的真实状态，用户可能在系统设置里改过。
func (a *App) RefreshLaunchAtLogin() {
	a.Settings.LoginEnabled = launchAtLogin()
	// 上次失败的原因只对上次那次操作有意义，重新读回真实状态后清掉。
	a.Settings.LoginError = ""
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
	here := !a.IsLocked() && a.importDestination().root == destination.root
	ids := slices.Clone(result.AssetIDs)
	// 只有用户自己发起的导入（拖入、粘贴、菜单）才选中新素材；自动导入和浏览器扩展在后台进来，不能打断手上的事，提示里给「查看」再跳过去。
	manual := event == ""
	if manual && a.Preferences.SelectImported && (result.Imported > 0 || result.Restored > 0) && len(ids) > 0 && here {
		a.selectImportResult(ids)
	}
	message := a.T("import_complete", result.Imported, result.Skipped)
	if result.Skipped == 0 {
		message = a.T("import_complete_new", result.Imported)
	}
	if len(result.Errors) > 0 {
		message = a.T("import_complete_failed", result.Imported, result.Skipped, len(result.Errors))
	}
	if showLibraryName && destination.name != "" {
		message = a.T("import_complete_library", destination.name, message)
	}
	switch {
	case len(result.Errors) > 0:
		// 部分失败不弹模态框，失败清单点「查看」再看。
		errs := slices.Clone(result.Errors)
		for _, err := range errs {
			slog.Error("导入时跳过文件", "error", err)
		}
		a.NotifyAction(message, a.T("view"), func() { a.showImportFailures(errs) })
	case !manual && len(ids) > 0 && here:
		a.NotifyAction(message, a.T("view"), func() {
			if a.Library != nil && a.Library.Root == destination.root && !a.IsLocked() {
				a.selectImportResult(ids)
			}
		})
	default:
		a.Notify(message)
	}
	if !a.Preferences.Notify || a.Window == nil || a.Window.IsFocused() || a.SettingsWindow != nil && a.SettingsWindow.IsFocused() {
		return
	}
	if !slices.Contains(a.Preferences.NotifyEvents, event) && !(result.Skipped > 0 && slices.Contains(a.Preferences.NotifyEvents, "duplicate")) {
		return
	}
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
	// 新素材落在当前位置里（在文件夹里导入就进这个文件夹）就留在原地选中；一张都不在才回到「全部」。
	matches := false
	if !a.TagPage {
		for _, id := range ids {
			if a.Index.Matches(a.browseQuery(a.Query), id) {
				matches = true
				break
			}
		}
	}
	if matches {
		a.Selected.Clear()
		compare := catalog.AssetOrder(a.browseQuery(a.Query))
		missing := false
		for _, id := range ids {
			if asset, exists := a.assetByID[id]; exists {
				if _, visible := slices.BinarySearchFunc(a.Visible, &asset, compare); visible {
					a.Selected.Add(id)
				} else if a.Index.Matches(a.browseQuery(a.Query), id) {
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
	if a.Preferences.CloseAction == "background" && a.canFindHiddenWindow() {
		a.HideMainWindow()
		return
	}
	mygo.App.Quit()
}

// 关窗后还能找回窗口吗：macOS 一直有 Dock 图标；Windows、Linux 要靠托盘图标，没开或没建出来就只能直接退出。
func (a *App) canFindHiddenWindow() bool {
	return runtime.GOOS == "darwin" || a.Preferences.TrayIcon && !a.TrayFailed
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
