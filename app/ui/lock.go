package ui

import (
	"errors"
	"log/slog"
	"os"
	"time"
	"unicode/utf8"

	passwordlock "github.com/Seaony/Momento/app/internal/lock"
	"github.com/egoist/mygo/ui"
)

type lockState struct {
	Locked, DialogOpen bool
	Busy, Saving       bool
	Mode               string
	Password           string
	CurrentPassword    string
	NewPassword        string
	ConfirmPassword    string
	ErrorKey, Detail   string
	CooldownUntil      time.Time
	Guard              *passwordlock.Guard
	Generation         uint64
	EditorSession      uint64
	post               func(func())
	now                func() time.Time
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
	a.Preview.Open = false
	a.PromptOpen, a.ConfirmOpen, a.NewLibraryOpen, a.ExportOpen = false, false, false, false
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
	a.Lock.Mode, a.Lock.DialogOpen = mode, true
	a.Lock.ErrorKey, a.Lock.Detail = "", ""
	a.InvalidateWindows()
}

func (a *App) lockScreen(c *ui.Context) {
	ui.Column(c).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+16), 24, 24, 24).
		Center().Material(frostedSurface{tint: c.Theme().Background.Alpha(.78), blur: 28}).DragWindow().Children(func() {
		ui.Column(c).Width(400).MaxWidthPercent(100).Padding(24).Center().Gap(16).Radius(12).
			Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).Children(func() {
			ui.Image(c, appLogo).Size(80, 80).Fit(ui.Contain)
			ui.Text(c, a.T("password_locked")).FontSize(24).Bold().TextColor(titleColor(c.Theme()))
			ui.Column(c).FillWidth().Gap(12).Children(func() {
				input := ui.TextInput(c.Key([2]any{"unlock-password", a.Lock.EditorSession}), &a.Lock.Password).
					Password().AutoFocus().Label(a.T("password_enter")).Placeholder(a.T("password_enter")).FillWidth().
					Disabled(a.PasswordOperationPending() || a.passwordCooldown() > 0)
				a.passwordFeedback(c)
				label := a.T("password_unlock")
				if a.Lock.Busy {
					label = a.T("password_verifying")
				}
				ready := !a.PasswordOperationPending() && a.passwordCooldown() == 0 && a.Lock.Password != ""
				button := primaryButton(c.Key("unlock"), label, !ready).FillWidth()
				if ready && (button.Clicked() || input.Submitted()) {
					a.unlockPassword()
				}
			})
		})
	})
}

func (a *App) passwordSettingsDialog(c *ui.Context) {
	if !a.Lock.DialogOpen {
		return
	}
	ui.Overlay(c, func() {
		back := ui.Column(c.Key("password-settings-backdrop")).Absolute().Left(0).Top(0).Right(0).Bottom(0).
			Center().Material(frostedSurface{tint: ui.Hex("#211A16").Alpha(.42), blur: 12}).Modal()
		back.Children(func() {
			title := a.T("password_dialog_" + a.Lock.Mode)
			panel := ui.Column(c.Key("password-settings")).Width(420).MaxWidthPercent(92).MaxHeightPercent(92).
				Padding(24).Gap(12).Radius(12).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border).
				Role(ui.RoleDialog).Label(title)
			panel.Children(func() {
				dialogHeading(c, "lock", title)
				var submit bool
				if a.Lock.Mode != "enable" {
					ui.Text(c, a.T("password_current")).Bold()
					input := ui.TextInput(c.Key([2]any{"current-password", a.Lock.EditorSession}), &a.Lock.CurrentPassword).
						Password().Label(a.T("password_current")).FillWidth().Disabled(a.PasswordOperationPending())
					submit = input.Submitted()
				}
				if a.Lock.Mode != "disable" {
					ui.Text(c, a.T("password_new")).Bold()
					ui.TextInput(c.Key([2]any{"new-password", a.Lock.EditorSession}), &a.Lock.NewPassword).
						Password().Label(a.T("password_new")).FillWidth().Disabled(a.PasswordOperationPending())
					ui.Text(c, a.T("password_confirm")).Bold()
					input := ui.TextInput(c.Key([2]any{"confirm-password", a.Lock.EditorSession}), &a.Lock.ConfirmPassword).
						Password().Label(a.T("password_confirm")).FillWidth().Disabled(a.PasswordOperationPending())
					submit = submit || input.Submitted()
					ui.Text(c, a.T("password_minimum")).FontSize(12).TextColor(c.Theme().TextMuted)
				}
				a.passwordFeedback(c)
				ui.Row(c).Gap(8).Justify(ui.End).Margin(4, 0, 0, 0).Children(func() {
					if ui.Button(c, a.T("cancel")).Disabled(a.PasswordOperationPending()).Clicked() {
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
					button := primaryButton(c.Key("password-save"), label, !ready)
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

func (a *App) passwordFeedback(c *ui.Context) {
	if remaining := a.passwordCooldown(); remaining > 0 {
		seconds := int((remaining + time.Second - 1) / time.Second)
		ui.Text(c, a.T("password_cooldown", seconds)).TextColor(c.Theme().Danger)
		c.After(time.Second)
	} else if a.Lock.ErrorKey != "" && a.Lock.ErrorKey != "password_cooldown" {
		text := a.T(a.Lock.ErrorKey)
		if a.Lock.ErrorKey == "password_save_failed" {
			text = a.T(a.Lock.ErrorKey, a.Lock.Detail)
		}
		ui.Text(c, text).TextColor(c.Theme().Danger)
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
			return
		}
		if next != confirm {
			a.Lock.ErrorKey = "password_mismatch"
			return
		}
	}
	hash, guard := a.Preferences.PasswordHash, a.passwordGuard()
	a.clearPasswordSecrets()
	a.Lock.Busy = true
	a.Lock.Generation++
	generation := a.Lock.Generation
	go func() {
		var err error
		if mode != "enable" {
			err = guard.Check(hash, current)
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
				a.closePasswordDialog()
				a.Notify(a.T("password_saved_" + mode))
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
	slog.Warn("密码验证失败", "reason", a.Lock.ErrorKey)
}
