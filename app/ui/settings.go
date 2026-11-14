package ui

import (
	"strings"

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
	AutoImportAddError string
	ShortcutErrors     map[string]string
	ShortcutSearch     string
	RecordingShortcut  string
	Scroll             ui.ScrollState
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
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	} else {
		c.Root().Background(background)
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		side := ui.Column(c.Key("settings-sidebar")).Width(200).Shrink(0).BorderWidth(0, 1, 0, 0).BorderColor(p.BorderSubtle)
		sidebarMaterial(c, side)
		side.Children(func() {
			bar := c.TitleBar()
			if bar.Left > 0 {
				ui.Box(c.Key("settings-sidebar-titlebar")).Height(max(44, bar.Bottom+space8)).FillWidth().Shrink(0).DragWindow()
			} else {
				ui.Row(c.Key("settings-sidebar-titlebar")).Height(rowHeight).FillWidth().Padding(0, space16).DragWindow().Children(func() { ui.Text(c, a.T("settings")).Bold() })
			}
			ui.Scroll(c.Key("settings-navigation")).Grow(1).Padding(space8).Gap(space4).Children(func() {
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
						sidebarRowIcon(c, tab.icon, a.Settings.Tab == i, row.Hovered()).Size(iconSize, iconSize).Shrink(0)
						label := ui.Text(c, a.T(tab.key)).FontSize(fontBody).SingleLine().Grow(1).MinWidth(0)
						if i == a.Settings.Tab {
							label.FontWeight(600)
						}
					})
					if row.Clicked() {
						a.Settings.Tab = i
						a.Settings.Scroll = ui.ScrollState{}
						a.CancelShortcutRecording()
					}
				}
			})
		})
		ui.Column(c).Grow(1).MinWidth(0).Background(background).Children(func() {
			bar := c.TitleBar()
			ui.Row(c.Key("settings-titlebar")).Height(rowHeight).Shrink(0).FillWidth().Padding(0, bar.Right+space24, 0, space24).Center().DragWindow().Children(func() {
				ui.Text(c, a.T(settingsTabs[a.Settings.Tab].key)).FontSize(fontBody).FontWeight(600).TextColor(p.Text).SingleLine()
			})
			ui.Scroll(c.Key("settings-content")).TrackScroll(&a.Settings.Scroll).Grow(1).Padding(space16, space24, space24, space24).Children(func() {
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
						a.generalSettings(c, update)
					}
				})
			})
		})
	})
	a.passwordSettingsDialog(c)
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
		ui.Text(c, mygo.App.Version()).Font(monoFont(c)).FontSize(fontNumber)
		if label := strings.TrimSpace(after); label != "" {
			ui.Text(c, label)
		}
	})
}

func (a *App) generalSettings(c *ui.Context, update UpdateCheckStatus) {
	settingsSection(c, "", "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("launch_at_login")).Grow(1)
			enabled := a.Settings.LoginEnabled
			if ui.Switch(c.Key("launch-at-login"), &enabled).Label(a.T("launch_at_login")).Changed() {
				a.SetLaunchAtLogin(enabled)
			}
		})
		if a.Settings.LoginError != "" {
			settingsError(c, a.T("error_launch_at_login", a.Settings.LoginError))
		}
	})
	settingsSection(c, "", a.T("background_hint"), func() {
		a.stringSetting(c, "close_action", &a.Preferences.CloseAction, []string{"background", "quit"}, []string{"close_background", "close_quit"})
		settingsSeparator(c)
		key := "tray_icon"
		if settingsPlatform() != "darwin" {
			key = "system_tray_icon"
		}
		a.switchSetting(c, key, &a.Preferences.TrayIcon, false)
		if settingsPlatform() == "darwin" {
			settingsSeparator(c)
			if !a.Preferences.TrayIcon {
				a.Preferences.DockWhenHidden = true
			}
			a.switchSetting(c, "dock_when_hidden", &a.Preferences.DockWhenHidden, !a.Preferences.TrayIcon)
		}
	})
	settingsSection(c, a.T("update"), "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("automatic_updates")).Grow(1)
			if ui.Switch(c.Key("automatic-updates"), &a.Preferences.AutoCheckUpdates).Label(a.T("automatic_updates")).Changed() {
				a.SavePreferences()
				StartAutomaticUpdateCheck(a.Preferences.AutoCheckUpdates)
			}
		})
		settingsSeparator(c)
		settingsRow(c, func() {
			a.settingsVersionText(c).Grow(1)
			a.updateButton(c, update)
		})
	})
	a.updateMessage(c, update, false)
}
func (a *App) languageMenu(c *ui.Context) {
	codes := []string{"system", "en", "zh-CN", "ja"}
	labels := []string{a.T("follow_system"), "English", "简体中文", "日本語"}
	selected := 0
	for i, code := range codes {
		if code == a.Preferences.Language {
			selected = i
		}
	}
	trigger := settingsMenuButton(c, a.T("language"))
	trigger.Children(func() {
		ui.Text(c, labels[selected]).SingleLine()
		symbol := appSymbol(c, "chevrons-up-down").Size(iconSmall, iconSmall).Shrink(0)
		symbol.TextColor(symbol.AnimateColor("symbol-color", symbolColor(c.Theme(), trigger.Hovered()), motionColor))
	}).Menu(func(m *ui.Menu) {
		for i, label := range labels {
			if m.Item(label).Radio(a.Preferences.Language == codes[i]).Chosen() {
				a.Preferences.Language = codes[i]
				a.SavePreferences()
			}
		}
	})
}

func (a *App) extensionSettings(c *ui.Context) {
	p := paletteFor(c.Theme())
	settingsSection(c, "", a.T("extension_local_only"), func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("extension_allow")).Grow(1).SingleLine()
			enabled := a.Preferences.BrowserExtensionEnabled
			if ui.Switch(c.Key("browser-extension"), &enabled).Disabled(a.Settings.ExtensionBusy).Label(a.T("extension_allow")).Changed() {
				a.SetExtensionEnabled(enabled)
			}
		})
		settingsSeparator(c)
		settingsRow(c, func() {
			ui.Text(c, a.T("extension_status")).Grow(1).Shrink(0).SingleLine()
			message, color := a.T("extension_disabled"), p.TextSecondary
			switch a.ExtensionStatus.State {
			case browser.Running:
				message, color = a.T("extension_running"), p.Success
			case browser.PortOccupied:
				message, color = a.T("extension_port_in_use"), p.Danger
			case browser.Failed:
				message, color = a.T("extension_failed"), p.Danger
			}
			ui.Row(c).Gap(space8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
				status := message
				if a.ExtensionStatus.State == browser.Running {
					status += "  " + a.ExtensionStatus.Address
				}
				ui.Row(c).Gap(space8).AlignItems(ui.Center).Role(ui.RoleStatus).Label(status).Children(func() {
					ui.Text(c, message).FontSize(fontSecondary).SingleLine()
					if a.ExtensionStatus.State == browser.Running {
						ui.Text(c, a.ExtensionStatus.Address).Font(monoFont(c)).FontSize(fontNumber).SingleLine()
					}
				})
			})
		})
	})
	if (a.ExtensionStatus.State == browser.Failed || a.ExtensionStatus.State == browser.PortOccupied) && a.ExtensionStatus.Err != nil {
		ui.Text(c, a.T("error_extension_listen", a.Texts.Error(a.ExtensionStatus.Err))).FontSize(fontSecondary).TextColor(p.Danger).Padding(0, space4).Role(ui.RoleAlert)
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
		message := fadeMotion(ui.Text(c, text).FontSize(fontSecondary).TextColor(color).Padding(0, space4)).Role(ui.RoleStatus)
		if centered {
			message.FillWidth().TextAlign(ui.Center)
		}
	}
}

func settingsPanel(c *ui.Context, open *bool, width float32, body func()) ui.Element {
	p := paletteFor(c.Theme())
	return ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Material(frostedSurface{tint: p.Scrim, blur: 12})
		panel.Width(width).MaxWidthPercent(92).MaxHeightPercent(92).Padding(space24).Gap(space12).
			Radius(radiusPanel).Material(panelMaterial(c.Theme()))
		elevated(panel, c.Theme())
		dialogMotion(backdrop, panel)
		body()
	})
}
