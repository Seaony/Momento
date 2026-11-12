package ui

import (
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
	a.SyncMenuFocus(c)
	a.Settings.Tab = max(0, min(a.Settings.Tab, len(settingsTabs)-1))
	background := settingsBackground(c)
	c.Root().Background(background)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		ui.Column(c.Key("settings-sidebar")).Width(200).Shrink(0).Background(c.Theme().Surface).BorderWidth(0, 1, 0, 0).BorderColor(c.Theme().Border).Children(func() {
			bar := c.TitleBar()
			if bar.Left > 0 {
				ui.Box(c.Key("settings-sidebar-titlebar")).Height(max(44, bar.Bottom+8)).FillWidth().Shrink(0).DragWindow()
			} else {
				ui.Row(c.Key("settings-sidebar-titlebar")).Height(44).FillWidth().Padding(0, 18).DragWindow().Children(func() { ui.Text(c, a.T("settings")).Bold() })
			}
			ui.Scroll(c.Key("settings-navigation")).Grow(1).Padding(8).Gap(3).Children(func() {
				for i, tab := range settingsTabs {
					if i == 3 || i == 7 || i == 9 {
						ui.Divider(c).Margin(7, 6)
					}
					row := ui.ButtonBase(c.Key(tab.key)).FillWidth().Height(34).Padding(6, 10).Gap(8).Radius(6).Label(a.T(tab.key)).Role(ui.RoleTab).AccessSelected(a.Settings.Tab == i).Justify(ui.Start)
					if i == a.Settings.Tab {
						row.Background(c.Theme().SurfacePressed).TextColor(c.Theme().Text)
					} else if row.Hovered() {
						row.Background(c.Theme().SurfaceHover)
					}
					row.Children(func() {
						ui.Icon(c, settingsIcons[tab.icon]).Size(19, 19).Shrink(0)
						ui.Text(c, a.T(tab.key)).SingleLine().Grow(1).MinWidth(0)
					})
					if row.Clicked() {
						a.Settings.Tab = i
						a.Settings.Scroll = ui.ScrollState{}
						a.CancelShortcutRecording()
					}
				}
			})
		})
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			bar := c.TitleBar()
			ui.Row(c.Key("settings-titlebar")).Height(44).Shrink(0).FillWidth().Padding(0, bar.Right+20, 0, 20).Center().DragWindow().Children(func() { ui.Text(c, a.T(settingsTabs[a.Settings.Tab].key)).FontSize(13).Bold().SingleLine() })
			ui.Scroll(c.Key("settings-content")).TrackScroll(&a.Settings.Scroll).Grow(1).Padding(16, 20, 24, 20).Children(func() {
				ui.Column(c.Key(a.Settings.Tab)).FillWidth().Gap(20).Children(func() {
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
	if c.Theme().Dark {
		return c.Theme().Background
	}
	return c.Theme().Surface
}

func settingsSection(c *ui.Context, title, footer string, content func()) {
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(12).Bold().TextColor(c.Theme().TextMuted).Padding(0, 4)
		}
		card := c.Theme().Surface
		if !c.Theme().Dark {
			card = card.Mix(c.Theme().SurfacePressed, .25)
		}
		ui.Column(c).FillWidth().Radius(12).Background(card).Border(1, c.Theme().Border).Children(content)
		if footer != "" {
			ui.Text(c, footer).FontSize(12).TextColor(c.Theme().TextMuted).Padding(0, 4)
		}
	})
}
func settingsRow(c *ui.Context, content func()) {
	ui.Row(c).MinHeight(44).FillWidth().Padding(8, 14).Gap(12).AlignItems(ui.Center).Children(content)
}
func settingsSeparator(c *ui.Context) { ui.Divider(c).Margin(0, 14) }

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
		settingsRow(c, func() { ui.Text(c, a.T("current_version", mygo.App.Version())).Grow(1); a.updateButton(c, update) })
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
	ui.ButtonBase(c).Height(28).Padding(0).Gap(5).Shrink(0).TextColor(c.Theme().Text).AlignItems(ui.Center).Label(a.T("language")).Children(func() {
		ui.Text(c, labels[selected]).SingleLine()
		ui.Icon(c, settingsSymbols["chevrons"]).Size(10, 14)
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
			message, color := a.T("extension_disabled"), c.Theme().TextMuted
			switch a.ExtensionStatus.State {
			case browser.Running:
				message, color = a.T("extension_running")+"  "+a.ExtensionStatus.Address, c.Theme().Success
			case browser.PortOccupied:
				message, color = a.T("extension_port_in_use"), c.Theme().Danger
			case browser.Failed:
				message, color = a.T("extension_failed"), c.Theme().Danger
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
				ui.Text(c, message).FontSize(12).SingleLine().Role(ui.RoleStatus)
			})
		})
	})
	if (a.ExtensionStatus.State == browser.Failed || a.ExtensionStatus.State == browser.PortOccupied) && a.ExtensionStatus.Err != nil {
		ui.Text(c, a.T("error_extension_listen", a.Texts.Error(a.ExtensionStatus.Err))).FontSize(12).TextColor(c.Theme().Danger).Padding(0, 4).Role(ui.RoleAlert)
	}
}

func (a *App) aboutSettings(c *ui.Context, update UpdateCheckStatus) {
	ui.Column(c).FillWidth().Center().Gap(8).Padding(8, 0, 0, 0).Children(func() {
		ui.Image(c, appLogo).Size(64, 64)
		ui.Text(c, "Momento").FontSize(20).Bold()
		ui.Text(c, a.T("current_version", mygo.App.Version())).TextColor(c.Theme().TextMuted)
		a.updateMessage(c, update, true)
		ui.Row(c).Gap(8).Margin(12, 0, 0, 0).Children(func() {
			if update.State == UpdateAvailable {
				if ui.PrimaryButton(c, a.T("update_to_version", update.Version)).Height(28).FontSize(12).Clicked() {
					ShowAvailableUpdate()
				}
			} else {
				a.updateButton(c, update)
			}
			if ui.Button(c, a.T("open_logs")).Height(28).Clicked() {
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
	if ui.Button(c.Key("check-updates"), label).Height(28).FontSize(12).Disabled(update.State == UpdateChecking).Clicked() {
		CheckForUpdates()
	}
	if update.State == UpdateAvailable {
		if ui.PrimaryButton(c, a.T("update")).Height(28).FontSize(12).Clicked() {
			ShowAvailableUpdate()
		}
	}
}

func (a *App) updateMessage(c *ui.Context, update UpdateCheckStatus, centered bool) {
	text := ""
	color := c.Theme().TextMuted
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
		color = c.Theme().Danger
	}
	if text != "" {
		message := ui.Text(c, text).FontSize(12).TextColor(color).Padding(0, 4).Role(ui.RoleStatus)
		if centered {
			message.FillWidth().TextAlign(ui.Center)
		}
	}
}

func settingsPanel(c *ui.Context, open *bool, width float32, body func()) ui.Element {
	return ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.42))
		panel.Width(width).MaxWidthPercent(92).MaxHeightPercent(92).Padding(24).Gap(12).
			Radius(12).Background(c.Theme().Background).Border(1, c.Theme().Border)
		body()
	})
}
