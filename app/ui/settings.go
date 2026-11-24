package ui

import (
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	SettingsGeneral = iota
	SettingsAppearance
	SettingsSidebar
	SettingsControls
	SettingsPreview
	SettingsImport
	SettingsShortcuts
	SettingsNotifications
	SettingsPassword
	SettingsExtension
	SettingsAbout
)

type settingsState struct {
	Tab                int
	LoginEnabled       bool
	LoginError         string
	ExtensionBusy      bool
	NotificationDenied bool
	AutoImportErrors   map[string]string
	// 出错的自动导入文件夹里，哪些是文件夹不在了：收到监视事件时判断好，界面渲染时不去碰磁盘。
	AutoImportMissing  map[string]bool
	AutoImportAddError string
	ShortcutErrors     map[string]string
	ShortcutSearch     string
	RecordingShortcut  string
	// 录快捷键没录成时的待处理项（冲突、不合规）：取消录制或换页就清掉。
	ShortcutPending map[string]shortcutClaim
	// 每组「恢复默认」后的提示：已恢复，或因冲突没恢复。
	ShortcutGroupNotes map[string]shortcutGroupNote
	// 选好要监视的文件夹后弹出的确认，确认前不加规则。
	AutoImportConfirm *autoImportConfirm
	// 设置窗口没有提示条，密码锁的操作结果显示在密码锁页面里几秒。
	PasswordNotice   string
	PasswordNoticeAt time.Time
	ExtensionCopied  bool
	Scroll           ui.ScrollState
}

var settingsTabs = []struct{ key, icon string }{
	{"settings_general", "general"},
	{"settings_appearance", "appearance"},
	{"settings_sidebar", "sidebar"},
	{"settings_controls", "controls"},
	{"settings_preview", "preview"},
	{"settings_import", "import"},
	{"settings_shortcuts", "shortcuts"},
	{"settings_notifications", "notifications"},
	{"settings_password", "password"},
	{"settings_extension", "browser"},
	{"settings_about", "about"},
}

func (a *App) SettingsHeight() int                  { return 560 }
func (a *App) settingsHeight(UpdateCheckStatus) int { return 560 }
func (a *App) SettingsView(c *ui.Context)           { a.settingsView(c, UpdateStatus()) }

func (a *App) settingsView(c *ui.Context, update UpdateCheckStatus) {
	a.applyTheme(c)
	p := paletteFor(c.Theme())
	a.SyncMenuFocus(c)
	a.Settings.Tab = max(0, min(a.Settings.Tab, len(settingsTabs)-1))
	background := settingsBackground(c)
	c.Root().Background(background)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		side := ui.Column(c.Key("settings-sidebar")).Width(200).Shrink(0).BorderWidth(0, 1, 0, 0).BorderColor(p.BorderSubtle)
		side.Background(p.Sidebar)
		side.Children(func() {
			bar := c.TitleBar()
			if bar.Left > 0 {
				ui.Box(c.Key("settings-sidebar-titlebar")).Height(max(titleBarHeight, bar.Bottom+space8)).FillWidth().Shrink(0).DragWindow()
			} else {
				ui.Row(c.Key("settings-sidebar-titlebar")).Height(rowHeight).FillWidth().Padding(0, space16).DragWindow().Children(func() { ui.Text(c, a.T("settings")).Bold() })
			}
			ui.Scroll(c.Key("settings-navigation")).Scrollbars(false).Grow(1).Padding(space8).Gap(space4).Children(func() {
				for i, tab := range settingsTabs {
					if i == 3 || i == 7 || i == 9 {
						ui.Divider(c).Margin(space4, space8)
					}
					row := ui.ButtonBase(c.Key(tab.key)).FillWidth().Height(controlHeight).Padding(space4, space8).Gap(space8).Radius(radiusControl).Cursor(ui.CursorPointer).Label(a.T(tab.key)).Role(ui.RoleTab).AccessSelected(a.Settings.Tab == i).Justify(ui.Start)
					background, text := ui.Transparent, p.TextSecondary
					if i == a.Settings.Tab {
						background, text = p.Raised, p.Text
						elevated(row, c.Theme())
					} else if row.Pressed() {
						background, text = p.Pressed, p.Text
					} else if row.Hovered() {
						background, text = p.Hover, p.Text
					}
					border := ui.Transparent
					if i == a.Settings.Tab {
						border = p.BorderSubtle
					}
					row.Border(1, row.AnimateColor("selection-border", border, motionColor)).
						Transition(ui.ElementTransition{Animation: &motionColor, Shadows: true})
					animateColors(row, background, text)
					row.Children(func() {
						// 和主侧栏同一套加粗图标、同样大小，选中只换颜色。
						ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
							sidebarRowIconColor(c, sidebarAppIcon(c, tab.icon), a.Settings.Tab == i, row.Hovered())
						})
						label := ui.Text(c, a.T(tab.key)).FontSize(fontBody).SingleLine().Grow(1).MinWidth(0)
						if i == a.Settings.Tab {
							label.FontWeight(600)
						}
					})
					if row.Clicked() {
						a.Settings.Tab = i
						a.Settings.Scroll = ui.ScrollState{}
						a.CancelShortcutRecording()
						a.Settings.ShortcutPending, a.Settings.ShortcutGroupNotes = nil, nil
						a.Settings.PasswordNotice, a.Settings.ExtensionCopied = "", false
					}
				}
			})
		})
		ui.Column(c).Grow(1).MinWidth(0).Background(background).Children(func() {
			bar := c.TitleBar()
			ui.Row(c.Key("settings-titlebar")).Height(rowHeight).Shrink(0).FillWidth().Padding(0, bar.Right+space24, 0, space24).Center().DragWindow().Children(func() {
				ui.Text(c.Key("settings-title"), a.T(settingsTabs[a.Settings.Tab].key)).FontSize(fontBody).FontWeight(600).TextColor(p.Text).SingleLine().ContentTransition(replaceMotion)
			})
			ui.Scroll(c.Key("settings-content")).Scrollbars(false).TrackScroll(&a.Settings.Scroll).Grow(1).Padding(space16, space24, space24, space24).Children(func() {
				fadeMotion(ui.Column(c.Key(a.Settings.Tab)).FillWidth().Gap(space24)).Children(func() {
					switch a.Settings.Tab {
					case SettingsAppearance:
						a.appearanceSettings(c)
					case SettingsSidebar:
						a.sidebarSettings(c)
					case SettingsControls:
						a.controlsSettings(c)
					case SettingsPreview:
						a.previewSettings(c)
					case SettingsImport:
						a.importSettings(c)
					case SettingsShortcuts:
						a.shortcutsSettings(c)
					case SettingsNotifications:
						a.notificationsSettings(c)
					case SettingsPassword:
						a.passwordSettings(c)
					case SettingsExtension:
						a.extensionSettings(c)
					case SettingsAbout:
						a.aboutSettings(c, update)
					default:
						a.generalSettings(c)
					}
				})
			})
		})
	})
	a.passwordSettingsDialog(c, true)
	a.autoImportConfirmDialog(c)
}

func settingsBackground(c *ui.Context) ui.Color {
	p := paletteFor(c.Theme())
	return p.Background
}

func settingsSection(c *ui.Context, title, footer string, content func()) {
	p := paletteFor(c.Theme())
	ui.Column(c).FillWidth().Gap(space8).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(fontGroup).FontWeight(600).LetterSpacing(.6).TextColor(p.TextSecondary).Padding(0, space4)
		}
		elevated(ui.Column(c).FillWidth().Radius(radiusPanel).Background(p.Raised), c.Theme()).Children(content)
		if footer != "" {
			ui.Text(c, footer).FontSize(fontSecondary).TextColor(p.TextSecondary).Padding(0, space4)
		}
	})
}
func settingsRow(c *ui.Context, content func()) {
	ui.Row(c).MinHeight(space48).FillWidth().Padding(space8, space16).Gap(space12).AlignItems(ui.Center).Children(content)
}
func settingsSeparator(c *ui.Context) { ui.Divider(c).Margin(0, space16) }

func settingsMenuButton(c *ui.Context, label string) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Height(controlSmall).Padding(0, space8).Gap(space8).Shrink(0).
		TextColor(p.Text).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	return menuFieldStyle(c, button)
}

func settingsButton(c *ui.Context, label string) ui.Element {
	return settingsMenuButton(c, label).Padding(0, space12).Justify(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(fontBody).SingleLine()
	})
}

// 保留译文语序，标签用正文，独立版本值用等宽字体。
func (a *App) settingsVersionText(c *ui.Context) ui.Element {
	before, after, _ := strings.Cut(a.T("current_version", "\uFFFC"), "\uFFFC")
	return ui.Row(c).Gap(space8).AlignItems(ui.Center).Children(func() {
		if label := strings.TrimSpace(before); label != "" {
			ui.Text(c, label)
		}
		ui.Text(c, mygo.App.Version()).FontFeatures("tnum").FontSize(fontNumber)
		if label := strings.TrimSpace(after); label != "" {
			ui.Text(c, label)
		}
	})
}

func (a *App) generalSettings(c *ui.Context) {
	settingsSection(c, "", "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("launch_at_login")).Grow(1)
			enabled := a.Settings.LoginEnabled
			if ui.Switch(c.Key("launch-at-login"), &enabled).Label(a.T("launch_at_login")).Changed() {
				a.SetLaunchAtLogin(enabled)
			}
		})
		if a.Settings.LoginError != "" {
			settingsErrorDetail(c, a.T("launch_at_login_failed"), a.Settings.LoginError)
		}
	})
	// 选「退出」时窗口一关程序就结束，托盘、Dock 和后台运行都不相关，整块不显示。
	background := a.Preferences.CloseAction != "quit"
	footer := ""
	if background {
		footer = a.T("background_hint")
	}
	settingsSection(c, "", footer, func() {
		a.stringSetting(c, "close_action", &a.Preferences.CloseAction, []string{"background", "quit"}, []string{"close_background", "close_quit"})
		if !background {
			return
		}
		settingsSeparator(c)
		if settingsPlatform() == "darwin" {
			a.switchSetting(c, "tray_icon", &a.Preferences.TrayIcon, false)
			settingsSeparator(c)
			note := ""
			if !a.Preferences.TrayIcon {
				a.Preferences.DockWhenHidden = true
				note = a.T("dock_required_hint")
			}
			a.switchSettingNote(c, "dock_when_hidden", note, &a.Preferences.DockWhenHidden, !a.Preferences.TrayIcon)
			return
		}
		// 后台运行时没有托盘图标就找不回窗口，托盘图标不能关。
		if !a.Preferences.TrayIcon {
			a.Preferences.TrayIcon = true
			a.SavePreferences()
		}
		a.switchSettingNote(c, "system_tray_icon", a.T("tray_required_hint"), &a.Preferences.TrayIcon, true)
	})
	settingsSection(c, a.T("update"), "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("automatic_updates")).Grow(1)
			if ui.Switch(c.Key("automatic-updates"), &a.Preferences.AutoCheckUpdates).Label(a.T("automatic_updates")).Changed() {
				a.SavePreferences()
				StartAutomaticUpdateCheck(a.Preferences.AutoCheckUpdates)
			}
		})
	})
}
func (a *App) languageMenu(c *ui.Context) {
	codes := []string{"system", "en", "zh-CN", "ja"}
	labels := []string{a.T("follow_system"), "English", "简体中文", "日本語"}
	choiceMenu(c, a.T("language"), max(0, slices.Index(codes, a.Preferences.Language)), labels, func(i int) {
		a.Preferences.Language = codes[i]
		a.SavePreferences()
	})
}

func (a *App) extensionSettings(c *ui.Context) {
	p := paletteFor(c.Theme())
	status := a.ExtensionStatus
	failed := status.State == browser.Failed || status.State == browser.PortOccupied
	settingsSection(c, "", a.T("extension_local_only"), func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("extension_allow")).Grow(1).SingleLine()
			enabled := a.Preferences.BrowserExtensionEnabled
			if ui.Switch(c.Key("browser-extension"), &enabled).Disabled(a.Settings.ExtensionBusy).Label(a.T("extension_allow")).Changed() {
				a.Settings.ExtensionCopied = false
				a.SetExtensionEnabled(enabled)
			}
		})
		settingsSeparator(c)
		settingsRow(c, func() {
			ui.Text(c, a.T("extension_status")).Grow(1).Shrink(0).SingleLine()
			message, color := a.T("extension_disabled"), p.TextSecondary
			switch {
			case status.State == browser.Running:
				message, color = a.T("extension_running"), p.Success
			case failed:
				message, color = a.T("extension_failed"), p.Danger
			}
			ui.Row(c).Gap(space8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
				ui.Text(c, message).FontSize(fontSecondary).SingleLine().Role(ui.RoleStatus)
			})
		})
		if failed {
			settingsSeparator(c)
			// 端口被占用时直接说怎么办；其他原因只有原始错误可看。原始错误链放进「复制详细信息」。
			detail := a.T("error_extension_listen", a.Texts.Error(status.Err))
			message := detail
			if status.State == browser.PortOccupied {
				message = a.T("extension_port_in_use_retry")
			}
			settingsRow(c, func() {
				ui.Text(c, message).FontSize(fontSecondary).TextColor(p.Danger).Grow(1).MinWidth(0).Role(ui.RoleAlert)
				if settingsButton(c.Key("extension-retry"), a.T("extension_retry")).Height(controlSmall).Disabled(a.Settings.ExtensionBusy).Clicked() {
					a.Settings.ExtensionCopied = false
					a.SetExtensionEnabled(true)
				}
				copyLabel := a.T("extension_copy_details")
				if a.Settings.ExtensionCopied {
					copyLabel = a.T("copied")
				}
				if settingsButton(c.Key("extension-copy"), copyLabel).Height(controlSmall).Clicked() {
					c.WriteClipboardThen(detail, func(err error) {
						if err != nil {
							a.ShowError(err)
						} else {
							a.Settings.ExtensionCopied = true
						}
					})
				}
			})
		}
		settingsSeparator(c)
		settingsRow(c, func() {
			if settingsButton(c.Key("extension-get"), a.T("extension_get")).Height(controlSmall).Clicked() {
				c.OpenURLThen(browserExtensionURL, a.ShowError)
			}
		})
	})
	// 扩展关着时这一项没用，不显示。
	if a.Preferences.BrowserExtensionEnabled {
		settingsSection(c, "", "", func() {
			a.switchSetting(c, "extension_picker", &a.Preferences.ExtensionPicker, false)
		})
	}
}

func (a *App) aboutSettings(c *ui.Context, update UpdateCheckStatus) {
	p := paletteFor(c.Theme())
	ui.Column(c).FillWidth().Center().Gap(space8).Padding(space8, 0, 0, 0).Children(func() {
		ui.Image(c, appLogo).Size(space64, space64)
		ui.Text(c, "Momento").FontSize(fontDisplay).FontWeight(600).TextColor(p.Text)
		a.settingsVersionText(c).FontSize(fontSecondary).TextColor(p.TextSecondary)
		a.updateMessage(c, update, true)
		ui.Row(c).Gap(space8).Margin(space12, 0, 0, 0).Children(func() {
			if update.State == UpdateAvailable {
				if primaryButton(c, a.T("update_to_version", update.Version), false).Height(controlSmall).Radius(radiusControl).Clicked() {
					ShowAvailableUpdate()
				}
			} else {
				a.updateButton(c, update)
			}
			if settingsButton(c, a.T("open_logs")).Clicked() {
				a.OpenLogDirectory()
			}
		})
	})
}

func (a *App) updateButton(c *ui.Context, update UpdateCheckStatus) {
	label := a.T("check_updates")
	if update.State == UpdateChecking {
		label = a.T("updates_checking")
	}
	if settingsButton(c.Key("check-updates"), label).Disabled(update.State == UpdateChecking).Clicked() {
		CheckForUpdates()
	}
	if update.State == UpdateAvailable {
		if primaryButton(c, a.T("update"), false).Height(controlSmall).Radius(radiusControl).Clicked() {
			ShowAvailableUpdate()
		}
	}
}

func (a *App) updateMessage(c *ui.Context, update UpdateCheckStatus, centered bool) {
	p := paletteFor(c.Theme())
	text := ""
	color := p.TextSecondary
	switch update.State {
	case UpdateChecking:
		text = a.T("updates_checking")
	case UpdateLatest:
		text = a.T("updates_latest")
	case UpdateAvailable:
		text = a.T("updates_available", update.Version)
	case UpdateUnconfigured:
		text = a.T("updates_unconfigured")
	case UpdateFailed:
		text = a.T("updates_failed", a.Texts.Error(update.Err))
		color = p.Danger
	}
	if text != "" {
		message := revealMotion(ui.Text(c.Key("update-message"), text).FontSize(fontSecondary).TextColor(color).Padding(0, space4).ContentTransition(replaceMotion)).Role(ui.RoleStatus)
		if centered {
			message.FillWidth().TextAlign(ui.Center)
		}
	}
}

// dialogPanel 是所有对话框的外壳：遮罩、动效、浮起面板，里面按 dialogLayout 排。
func dialogPanel(c *ui.Context, open *bool, width float32, icon, title string, body, footer func()) ui.Element {
	p := paletteFor(c.Theme())
	return ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(p.Scrim)
		dialogSurface(c, panel.Width(width).Label(title))
		dialogMotion(backdrop, panel)
		dialogLayout(c, icon, title, body, footer)
	})
}

// 对话框面板本身：没有内边距，标题栏、分隔线、内容、按钮行各自留白。
func dialogSurface(c *ui.Context, panel ui.Element) ui.Element {
	panel.MaxWidthPercent(92).MaxHeightPercent(92).Padding(0).Gap(0).Radius(radiusPanel).Background(panelColor(c.Theme()))
	return elevated(panel, c.Theme())
}

// 所有对话框同一个骨架：标题栏（次要色小图标 + 14 号标题）、通栏分隔线、内容、右下角的操作按钮。body 为空时只有标题和按钮。
func dialogLayout(c *ui.Context, icon, title string, body, footer func()) {
	p := paletteFor(c.Theme())
	ui.Row(c.Key("dialog-header")).FillWidth().Height(dialogHeaderHeight).Padding(0, space20).Gap(space8).AlignItems(ui.Center).Shrink(0).Children(func() {
		if icon != "" {
			appIcon(c, icon).Size(iconMedium, iconMedium).Shrink(0).TextColor(p.TextSecondary)
		}
		ui.Text(c, title).FontSize(fontLead).FontWeight(600).TextColor(p.Text).SingleLine().Grow(1).MinWidth(0)
	})
	ui.Box(c.Key("dialog-divider")).FillWidth().Height(1).Shrink(0).Background(p.BorderSubtle.Mix(p.Border, .5))
	if body != nil {
		ui.Column(c.Key("dialog-body")).FillWidth().MinHeight(0).Shrink(1).Padding(space20, space20, 0, space20).Gap(space16).Children(body)
	}
	ui.Row(c.Key("dialog-footer")).FillWidth().Wrap().Padding(space20).Justify(ui.End).AlignItems(ui.Center).Gap(space8).Shrink(0).Children(footer)
}

// 对话框底部和页面中间（空状态、欢迎页）的次要按钮：高 32、左右留 16、最窄 72，浅底无边框，13 号 500。
func dialogButton(c *ui.Context, label string, disabled bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Height(controlHeight).MinWidth(dialogButtonMinWidth).Padding(0, space16).Shrink(0).Radius(radiusControl).
		Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label).Disabled(disabled)
	fill := themeValue(c, .055, .075)
	switch {
	case button.Pressed():
		fill += .06
	case button.Hovered():
		fill += .03
	}
	ink := p.Text
	if disabled {
		fill, ink = themeValue(c, .035, .05), p.TextTertiary
	}
	button.Background(button.AnimateColor("dialog-button", p.Text.Alpha(fill), motionColor))
	button.Children(func() {
		ui.Text(c, label).FontSize(fontBody).FontWeight(500).TextColor(button.AnimateColor("dialog-button-ink", ink, motionColor)).SingleLine().ContentTransition(replaceMotion)
	})
	return button
}

// 对话框底部和页面中间的主按钮：和次要按钮同尺寸，品牌色底；禁用时退成次要按钮的浅底加三级色字。
func dialogPrimaryButton(c *ui.Context, label string, disabled bool) ui.Element {
	return dialogActionButton(c, label, disabled, false)
}

// 删除确认用的危险色主按钮。深色下危险色是浅红，文字用背景色才看得清（白字对比度只有 2:1）。
func dialogDangerButton(c *ui.Context, label string) ui.Element {
	return dialogActionButton(c, label, false, true)
}

func dialogActionButton(c *ui.Context, label string, disabled, danger bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Height(controlHeight).MinWidth(dialogButtonMinWidth).Padding(0, space16).Shrink(0).Radius(radiusControl).
		Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label).Disabled(disabled)
	theme := c.Theme()
	fill, ink := theme.Accent, theme.AccentText
	switch {
	case disabled:
		fill, ink = p.Text.Alpha(themeValue(c, .035, .05)), p.TextTertiary
	case danger:
		fill, ink = p.Danger, p.Background
		if button.Pressed() {
			fill = fill.Mix(theme.Text, .15)
		} else if button.Hovered() {
			fill = fill.Mix(theme.Text, .08)
		}
	case button.Pressed():
		fill = theme.AccentPressed
	case button.Hovered():
		fill = theme.AccentHover
	}
	button.Background(button.AnimateColor("dialog-primary", fill, motionColor))
	button.Children(func() {
		ui.Text(c, label).FontSize(fontBody).FontWeight(500).TextColor(button.AnimateColor("dialog-primary-ink", ink, motionColor)).SingleLine().ContentTransition(replaceMotion)
	})
	return button
}
