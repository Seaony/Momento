package ui

import (
	"errors"
	"log/slog"
	"os"
	"time"
	"unicode/utf8"

	passwordlock "github.com/Seaony/Momento/app/internal/lock"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type lockState struct {
	Locked, DialogOpen bool
	// DialogInSettings 为真表示密码对话框是从设置窗口打开的，只在设置窗口里显示。
	DialogInSettings bool
	// ForgotOpen 是锁屏上「忘记密码？」展开后的恢复办法。
	ForgotOpen       bool
	Busy, Saving     bool
	Mode             string
	Password         string
	CurrentPassword  string
	NewPassword      string
	ConfirmPassword  string
	ErrorKey, Detail string
	CooldownUntil    time.Time
	Guard            *passwordlock.Guard
	Generation       uint64
	EditorSession    uint64
	// Rejections 每被拒一次（输错、太短、两次不一致、冷却中）加一，面板跟着轻抖一下。
	Rejections uint64
	post       func(func())
	now        func() time.Time
}

func (a *App) InitPasswordLock() {
	a.Lock.Guard = passwordlock.NewGuard()
	a.Lock.Locked = a.Preferences.PasswordHash != ""
}

func (a *App) IsLocked() bool { return a.Lock.Locked }

func (a *App) PasswordOperationPending() bool { return a.Lock.Busy || a.Lock.Saving }
func (a *App) PasswordSavePending() bool      { return a.Lock.Saving }

func (a *App) LockNow() {
	if a.Preferences.PasswordHash == "" {
		return
	}
	a.FlushInspectorEdits()
	a.cancelExtensionPicker()
	a.Lock.Locked = true
	a.Lock.DialogOpen = false
	if !a.Lock.Saving {
		a.Lock.Generation++
		a.Lock.Busy = false
	}
	a.clearPasswordSecrets()
	a.Lock.ErrorKey, a.Lock.Detail = "", ""
	a.Lock.Rejections = 0 // 新出现的面板不带着上一次的抖动
	a.Lock.ForgotOpen = false
	a.Settings.AutoImportConfirm = nil
	a.CloseAllPreviewWindows()
	a.StopSlideshow()
	a.PromptOpen, a.ConfirmOpen, a.NewLibraryOpen, a.EditLibraryOpen, a.ExportOpen = false, false, false, false, false
	a.SettingsOpen = false
	if a.SettingsWindow != nil {
		a.SettingsWindow.Close()
	}
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
	a.InvalidateWindows()
	slog.Info("界面已锁定")
}

func (a *App) ShowPasswordSettings(mode string) {
	if a.IsLocked() || a.PasswordOperationPending() {
		return
	}
	if mode != "enable" && mode != "disable" && mode != "change" {
		return
	}
	if a.Preferences.PasswordHash == "" {
		mode = "enable"
	} else if mode == "enable" {
		mode = "change"
	}
	a.clearPasswordSecrets()
	a.Lock.Mode, a.Lock.DialogOpen, a.Lock.DialogInSettings = mode, true, true
	a.Lock.ErrorKey, a.Lock.Detail = "", ""
	a.Lock.Rejections = 0 // 新出现的面板不带着上一次的抖动
	a.InvalidateWindows()
}

func (a *App) lockScreen(c *ui.Context) {
	// 压在图库上面一层：解锁时锁屏在图库之上淡出，露出下面的内容。
	backdrop := ui.Column(c.Key("lock-screen")).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+space16), space24, space24, space24).
		Center().Background(c.Theme().Background).DragWindow().ZIndex(10)
	backdrop.Children(func() {
		panel := ui.Column(c.Key("unlock-panel")).Width(panelDialog).MaxWidthPercent(100).Padding(space24).Center().Gap(space16).Radius(radiusPanel).
			Background(panelColor(c.Theme()))
		elevated(panel, c.Theme())
		// 锁定那一帧整窗不播动画（见 View），背景立刻整块盖住图库；解锁时才淡出露出图库。
		backdrop.Transition(ui.ElementTransition{ExitDuration: motionEnter, Colors: true, Exit: stillMotion})
		panel.Transition(dialogPanelTransition)
		a.rejectionShake(panel)
		panel.Children(func() {
			ui.Image(c, appLogo).Size(80, 80).Fit(ui.Contain)
			ui.Text(c, a.T("password_locked")).FontSize(fontTitle).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
			ui.Column(c).FillWidth().Gap(space12).Children(func() {
				input := fieldStyle(c, ui.TextInput(c.Key([2]any{"unlock-password", a.Lock.EditorSession}), &a.Lock.Password).
					Password().AutoFocus().Label(a.T("password_enter")).Placeholder(a.T("password_enter")).PlaceholderColor(placeholderColor(c)).FillWidth().
					Disabled(a.PasswordOperationPending() || a.passwordCooldown() > 0))
				a.passwordFeedback(c, false)
				label := a.T("password_unlock")
				if a.Lock.Busy {
					label = a.T("password_verifying")
				}
				ready := !a.PasswordOperationPending() && a.passwordCooldown() == 0 && a.Lock.Password != ""
				button := primaryButton(c.Key("unlock"), label, !ready).FillWidth()
				if ready && (button.Clicked() || input.Submitted()) {
					a.unlockPassword()
				}
				a.forgotPassword(c)
			})
		})
	})
}

// forgotPassword 是锁屏上的出路：输错一次后出现「忘记密码？」，配置损坏时直接展开办法。
func (a *App) forgotPassword(c *ui.Context) {
	invalid := a.Lock.ErrorKey == "password_invalid"
	if a.Lock.Rejections == 0 && !invalid {
		return
	}
	p := paletteFor(c.Theme())
	ui.Column(c.Key("forgot-password")).FillWidth().Gap(space8).AlignItems(ui.Center).Children(func() {
		if !a.Lock.ForgotOpen && !invalid {
			link := ui.ButtonBase(c.Key("forgot")).Height(24).Padding(0, popClearPad).Radius(radiusSmall).Cursor(ui.CursorPointer).Label(a.T("password_forgot"))
			background, text := ui.Transparent, p.TextTertiary
			if link.Pressed() {
				background, text = p.Pressed, p.Text
			} else if link.Hovered() {
				background, text = p.Hover, p.Text
			}
			animateColors(link, background, text)
			link.Children(func() { ui.Text(c, a.T("password_forgot")).FontSize(fontSecondary).SingleLine() })
			if link.Clicked() {
				a.Lock.ForgotOpen = true
			}
			return
		}
		ui.Text(c, a.T("password_forgot_help")).FontSize(fontSecondary).TextColor(p.TextSecondary).FillWidth().Selectable()
		if settingsButton(c.Key("show-config"), a.T("password_show_config")).Clicked() {
			a.showConfigDirectory()
		}
	})
}

func (a *App) showConfigDirectory() {
	if a.configDir == "" {
		a.ShowError(errors.New("application configuration directory is unavailable"))
		return
	}
	if err := mygo.Shell.OpenPath(a.configDir); err != nil {
		slog.Error("打开配置文件夹失败", "path", a.configDir, "error", err)
		a.ShowError(err)
	}
}

// 密码对话框只在发起它的窗口里显示：inSettings 为真是设置窗口，否则是主窗口。
func (a *App) passwordSettingsDialog(c *ui.Context, inSettings bool) {
	if !a.Lock.DialogOpen || a.Lock.DialogInSettings != inSettings {
		return
	}
	ui.Overlay(c, func() {
		back := ui.Column(c.Key("password-settings-backdrop")).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Center().Background(paletteFor(c.Theme()).Scrim).Modal()
		back.Children(func() {
			title := a.T("password_dialog_" + a.Lock.Mode)
			panel := dialogSurface(c, ui.Column(c.Key("password-settings")).Width(panelDialog)).Role(ui.RoleDialog).Label(title)
			dialogMotion(back, panel)
			a.rejectionShake(panel)
			panel.Children(func() {
				var submit bool
				dialogLayout(c, "lock", title, func() {
					if a.Lock.Mode != "enable" {
						dialogSection(c, a.T("password_current"), func() {
							input := fieldStyle(c, ui.TextInput(c.Key([2]any{"current-password", a.Lock.EditorSession}), &a.Lock.CurrentPassword).
								Password().AutoFocus().Label(a.T("password_current")).FillWidth().Disabled(a.PasswordOperationPending()))
							submit = input.Submitted()
						})
					}
					if a.Lock.Mode != "disable" {
						dialogSection(c, a.T("password_new"), func() {
							input := ui.TextInput(c.Key([2]any{"new-password", a.Lock.EditorSession}), &a.Lock.NewPassword).
								Password().Label(a.T("password_new")).FillWidth().Disabled(a.PasswordOperationPending())
							if a.Lock.Mode == "enable" {
								input.AutoFocus()
							}
							fieldStyle(c, input)
						})
						dialogSection(c, a.T("password_confirm"), func() {
							input := fieldStyle(c, ui.TextInput(c.Key([2]any{"confirm-password", a.Lock.EditorSession}), &a.Lock.ConfirmPassword).
								Password().Label(a.T("password_confirm")).FillWidth().Disabled(a.PasswordOperationPending()))
							submit = submit || input.Submitted()
							// 太短时这一行自己变红，不再另外弹同一句。
							hint := paletteFor(c.Theme()).TextSecondary
							if a.Lock.ErrorKey == "password_minimum" {
								hint = paletteFor(c.Theme()).Danger
							}
							ui.Text(c, a.T("password_minimum")).FontSize(fontSecondary).TextColor(hint)
						})
					}
					if a.Lock.Mode == "enable" {
						ui.Text(c, a.T("password_dialog_warning")).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
					}
					a.passwordFeedback(c, a.Lock.Mode != "disable")
				}, func() {
					if dialogButton(c, a.T("cancel"), a.PasswordOperationPending()).Clicked() {
						a.closePasswordDialog()
					}
					ready := !a.PasswordOperationPending() && a.passwordCooldown() == 0
					if a.Lock.Mode != "enable" {
						ready = ready && a.Lock.CurrentPassword != ""
					}
					if a.Lock.Mode != "disable" {
						ready = ready && a.Lock.NewPassword != "" && a.Lock.ConfirmPassword != ""
					}
					label := a.T("password_action_" + a.Lock.Mode)
					if a.Lock.Busy {
						label = a.T("password_verifying")
					}
					button := dialogPrimaryButton(c.Key("password-save"), label, !ready)
					if ready && (button.Clicked() || submit) {
						a.submitPasswordSettings()
					}
				})
			})
			if !a.PasswordOperationPending() && (panel.PressedOutside() || back.OverlayShortcut(0, ui.KeyEscape)) {
				a.closePasswordDialog()
			}
		})
	})
}

// 提示在原地展开，按钮跟着往下滑；冷却的秒数用等宽数字一秒一秒往下滚，整行不左右抖。
func (a *App) passwordFeedback(c *ui.Context, minimumShown bool) {
	if remaining := a.passwordCooldown(); remaining > 0 {
		seconds := int((remaining + time.Second - 1) / time.Second)
		text := ui.Text(c.Key("password-cooldown"), a.T("password_cooldown", seconds)).FontSize(fontSecondary).FontFeatures("tnum").TextColor(paletteFor(c.Theme()).Danger)
		revealMotion(countMotion(text, seconds))
		c.After(time.Second)
	} else if a.Lock.ErrorKey != "" && a.Lock.ErrorKey != "password_cooldown" && !(minimumShown && a.Lock.ErrorKey == "password_minimum") {
		text := a.T(a.Lock.ErrorKey)
		if a.Lock.ErrorKey == "password_save_failed" {
			text = a.T(a.Lock.ErrorKey, a.Lock.Detail)
		}
		revealMotion(ui.Text(c.Key("password-error"), text).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).Danger).ContentTransition(replaceMotion))
	}
}

// rejectionShake 在密码被拒后让面板轻抖一下；还没被拒过不调用，免得面板第一次出现就抖。
func (a *App) rejectionShake(panel ui.Element) {
	if a.Lock.Rejections > 0 {
		shakeMotion(panel, a.Lock.Rejections)
	}
}

func (a *App) passwordCooldown() time.Duration {
	now := time.Now()
	if a.Lock.now != nil {
		now = a.Lock.now()
	}
	return max(0, a.Lock.CooldownUntil.Sub(now))
}

func (a *App) passwordGuard() *passwordlock.Guard {
	if a.Lock.Guard == nil {
		a.Lock.Guard = passwordlock.NewGuard()
	}
	return a.Lock.Guard
}

func (a *App) clearPasswordSecrets() {
	a.Lock.Password, a.Lock.CurrentPassword, a.Lock.NewPassword, a.Lock.ConfirmPassword = "", "", "", ""
	// 新建编辑会话，不能用撤销找回上次输入的密码。
	a.Lock.EditorSession++
}

func (a *App) closePasswordDialog() {
	a.Lock.DialogOpen = false
	a.clearPasswordSecrets()
}

func (a *App) postPasswordResult(fn func()) {
	if a.Lock.post != nil {
		a.Lock.post(fn)
	} else if a.Window != nil {
		a.Window.Update(fn)
	} else {
		fn()
	}
}

func (a *App) unlockPassword() {
	if !a.IsLocked() || a.PasswordOperationPending() || a.passwordCooldown() > 0 {
		return
	}
	password, hash, guard := a.Lock.Password, a.Preferences.PasswordHash, a.passwordGuard()
	a.clearPasswordSecrets()
	a.Lock.Busy = true
	a.Lock.Generation++
	generation := a.Lock.Generation
	go func() {
		err := guard.Check(hash, password)
		remaining := guard.Remaining()
		a.postPasswordResult(func() {
			if generation != a.Lock.Generation {
				return
			}
			a.Lock.Busy = false
			if err != nil {
				a.passwordFailure(err, remaining)
			} else {
				a.Lock.Locked, a.Lock.ErrorKey, a.Lock.Detail = false, "", ""
				a.Lock.ForgotOpen = false
				if a.MenuChanged != nil {
					a.MenuChanged()
				}
				slog.Info("界面已解锁")
			}
			a.InvalidateWindows()
		})
	}()
}

func (a *App) submitPasswordSettings() {
	if !a.Lock.DialogOpen || a.PasswordOperationPending() || a.passwordCooldown() > 0 {
		return
	}
	mode := a.Lock.Mode
	current, next, confirm := a.Lock.CurrentPassword, a.Lock.NewPassword, a.Lock.ConfirmPassword
	if mode != "disable" {
		if !utf8.ValidString(next) || utf8.RuneCountInString(next) < 4 {
			a.Lock.ErrorKey = "password_minimum"
			a.Lock.Rejections++
			return
		}
		if next != confirm {
			a.Lock.ErrorKey = "password_mismatch"
			a.Lock.Rejections++
			return
		}
	}
	hash, guard := a.Preferences.PasswordHash, a.passwordGuard()
	// 输入留在框里：当前密码错了只清当前密码，保存失败时全部保留。
	a.Lock.Busy = true
	a.Lock.Generation++
	generation := a.Lock.Generation
	go func() {
		var err error
		currentRejected := false
		if mode != "enable" {
			err = guard.Check(hash, current)
			currentRejected = err != nil
		}
		newHash := ""
		if err == nil && mode != "disable" {
			newHash, err = passwordlock.Hash(next)
		}
		remaining := guard.Remaining()
		a.postPasswordResult(func() {
			if generation != a.Lock.Generation {
				return
			}
			if err != nil {
				a.Lock.Busy = false
				if currentRejected {
					a.Lock.CurrentPassword = ""
					a.Lock.EditorSession++ // 重建输入框，撤销不能找回刚输的密码
				}
				a.passwordFailure(err, remaining)
				a.InvalidateWindows()
				return
			}
			a.persistPasswordHash(newHash, mode)
		})
	}()
}

func (a *App) persistPasswordHash(hash, mode string) {
	a.Lock.Saving = true
	p := clonePreferences(a.Preferences)
	p.PasswordHash = hash
	version := a.prefVersion.Add(1)
	go func() {
		var err error
		if a.configDir == "" {
			err = errors.New("application configuration directory is unavailable")
		} else {
			err = a.writePreferences(p, version)
		}
		a.postPasswordResult(func() {
			a.Lock.Saving, a.Lock.Busy = false, false
			if err != nil {
				slog.Error("保存密码设置失败", "error", err)
				a.Lock.ErrorKey, a.Lock.Detail = "password_save_failed", a.passwordErrorDetail(err)
				a.InvalidateWindows()
				return
			} else {
				a.Preferences.PasswordHash = hash
				if hash == "" {
					a.Lock.Locked = false
				}
				a.passwordGuard().Reset()
				a.Lock.CooldownUntil = time.Time{}
				a.Lock.ErrorKey, a.Lock.Detail = "", ""
				a.Lock.Rejections = 0 // 新出现的面板不带着上一次的抖动
				inSettings := a.Lock.DialogInSettings
				a.closePasswordDialog()
				// 设置窗口没有提示条，反馈显示在密码锁页面里。
				if message := a.T("password_saved_" + mode); inSettings {
					a.Settings.PasswordNotice, a.Settings.PasswordNoticeAt = message, time.Now()
				} else {
					a.Notify(message)
				}
				slog.Info("密码设置已保存", "operation", mode)
			}
			a.SavePreferences()
		})
	}()
}

func (a *App) passwordErrorDetail(err error) string {
	switch {
	case passwordDiskFull(err):
		return a.T("password_disk_full")
	case errors.Is(err, os.ErrPermission):
		return a.T("password_permission_denied")
	default:
		return a.Texts.Error(err)
	}
}

func (a *App) passwordFailure(err error, remaining time.Duration) {
	switch {
	case errors.Is(err, passwordlock.ErrCooldown):
		a.Lock.ErrorKey = "password_cooldown"
		a.Lock.CooldownUntil = time.Now().Add(remaining)
	case errors.Is(err, passwordlock.ErrWrongPassword):
		a.Lock.ErrorKey = "password_wrong"
	case errors.Is(err, passwordlock.ErrPasswordTooShort):
		a.Lock.ErrorKey = "password_minimum"
	default:
		a.Lock.ErrorKey = "password_invalid"
	}
	a.Lock.Rejections++
	slog.Warn("密码验证失败", "reason", a.Lock.ErrorKey)
}
