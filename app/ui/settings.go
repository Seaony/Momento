package ui

import (
	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/egoist/mygo/ui"
)

type settingsState struct {
	Tab           int
	LoginEnabled  bool
	LoginError    string
	ExtensionBusy bool
}

var settingsTabs = []struct{ key, icon string }{
	{"settings_general", "general"},
	{"settings_extension", "browser"},
	{"settings_about", "about"},
}

func (a *App) SettingsHeight() int {
	return a.settingsHeight(UpdateStatus())
}

func (a *App) settingsHeight(update UpdateCheckStatus) int {
	switch a.Settings.Tab {
	case 1:
		if a.ExtensionStatus.State == browser.Failed {
			return 336
		}
		return 292
	case 2:
		if update.State != UpdateIdle && update.State != "" {
			return 396
		}
		return 344
	default:
		height := 512
		if a.Settings.LoginError != "" {
			height += 44
		}
		if update.State != UpdateIdle && update.State != "" {
			height += 52
		}
		return height
	}
}

func (a *App) SettingsView(c *ui.Context) {
	a.settingsView(c, UpdateStatus())
}

func (a *App) settingsView(c *ui.Context, update UpdateCheckStatus) {
	a.applyTheme(c)
	a.Settings.Tab = max(0, min(a.Settings.Tab, len(settingsTabs)-1))
	background := settingsBackground(c)
	c.Root().Background(background)
	ui.Column(c).Fill().Background(background).Children(func() {
		bar := c.TitleBar()
		ui.Row(c.Key("settings-titlebar")).Height(36).Shrink(0).
			Padding(0, max(bar.Left, bar.Right)+12).Center().DragWindow().Children(func() {
			ui.Text(c, a.T(settingsTabs[a.Settings.Tab].key)).FontSize(13).Bold().SingleLine()
		})
		tabs := ui.TabsBase(c.Key("settings-tabs"), &a.Settings.Tab, len(settingsTabs))
		tabs.List.Height(72).Shrink(0).FillWidth().Padding(2, 20, 10, 20).Gap(6).Justify(ui.Center).DragWindow().Children(func() {
			for i, tab := range settingsTabs {
				button := tabs.Tab(i).MinWidth(88).Height(58).Padding(7, 12).Radius(9).
					Label(a.T(tab.key)).AccessSelected(a.Settings.Tab == i)
				if a.Settings.Tab == i {
					button.Background(c.Theme().SurfacePressed.Mix(background, 0.5)).Border(1, c.Theme().Border)
				} else if button.Hovered() {
					button.Background(c.Theme().SurfaceHover)
				}
				button.Children(func() {
					ui.Column(c).Center().Gap(4).Children(func() {
						ui.Image(c, settingsIcons[tab.icon]).Size(25, 25)
						ui.Text(c, a.T(tab.key)).FontSize(12).SingleLine()
					})
				})
			}
		})
		ui.Divider(c)
		ui.Scroll(c.Key("settings-content")).Grow(1).Padding(20).Children(func() {
			ui.Column(c).FillWidth().Gap(20).Children(func() {
				switch a.Settings.Tab {
				case 1:
					a.extensionSettings(c)
				case 2:
					a.aboutSettings(c, update)
				default:
					a.generalSettings(c, update)
				}
			})
		})
	})
	if a.SettingsWindow != nil {
		width, height := a.SettingsWindow.ContentSize()
		if wanted := a.settingsHeight(update); width != 560 || height != wanted {
			a.SettingsWindow.SetContentSize(560, wanted)
		}
	}
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
			card = card.Mix(c.Theme().SurfacePressed, 0.25)
		}
		ui.Column(c).FillWidth().Radius(12).Background(card).Border(1, c.Theme().Border).Children(content)
		if footer != "" {
			ui.Text(c, footer).FontSize(12).TextColor(c.Theme().TextMuted).Padding(0, 4)
		}
	})
}

func settingsRow(c *ui.Context, content func()) {
	ui.Row(c).Height(44).FillWidth().Padding(0, 14).Gap(12).AlignItems(ui.Center).Children(content)
}

func settingsSeparator(c *ui.Context) {
	ui.Divider(c).Margin(0, 14)
}

func (a *App) generalSettings(c *ui.Context, update UpdateCheckStatus) {
	settingsSection(c, "", a.T("launch_at_login_hint"), func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("launch_at_login")).SingleLine().Grow(1)
			enabled := a.Settings.LoginEnabled
			if ui.Switch(c.Key("launch-at-login"), &enabled).Label(a.T("launch_at_login")).Changed() {
				a.SetLaunchAtLogin(enabled)
			}
		})
	})
	if a.Settings.LoginError != "" {
		ui.Text(c, a.T("error_launch_at_login", a.Settings.LoginError)).FontSize(12).TextColor(c.Theme().Danger).Padding(0, 4).Role(ui.RoleAlert)
	}
	settingsSection(c, a.T("appearance_language"), "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("appearance")).Grow(1).SingleLine()
			a.appearanceChoices(c.Key("appearance"))
		})
		settingsSeparator(c)
		settingsRow(c, func() {
			ui.Text(c, a.T("language")).Grow(1).SingleLine()
			a.languageMenu(c.Key("language"))
		})
	})
	settingsSection(c, a.T("update"), "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("automatic_updates")).Grow(1).SingleLine()
			if ui.Switch(c.Key("automatic-updates"), &a.Preferences.AutoCheckUpdates).Label(a.T("automatic_updates")).Changed() {
				a.SavePreferences()
				StartAutomaticUpdateCheck(a.Preferences.AutoCheckUpdates)
			}
		})
		settingsSeparator(c)
		settingsRow(c, func() {
			ui.Text(c, a.T("current_version", "2.0.0")).Grow(1).SingleLine()
			a.updateButton(c, update)
		})
	})
	a.updateMessage(c, update)
}

func (a *App) appearanceChoices(c *ui.Context) {
	codes := []string{"system", "light", "dark"}
	labels := []string{a.T("follow_system"), a.T("appearance_light"), a.T("appearance_dark")}
	selected := 0
	for i, code := range codes {
		if code == a.Preferences.Appearance {
			selected = i
		}
	}
	segments := ui.SegmentedBase(c, &selected, len(codes))
	segments.Track.Shrink(0).Height(34).Padding(3).Gap(2).Radius(10).Background(c.Theme().SurfacePressed).Label(a.T("appearance")).Children(func() {
		for i, title := range labels {
			segment := segments.Segment(i).Height(28).Padding(0, 14).Radius(7).Label(title).Tooltip(title).Center()
			if selected == i {
				segment.Background(ui.Hex("#2563EB")).TextColor(ui.RGB(255, 255, 255))
			} else if segment.Hovered() {
				segment.Background(c.Theme().SurfacePressed.Mix(c.Theme().Text, 0.06))
			}
			segment.Children(func() { ui.Icon(c, settingsSymbols[codes[i]]).Size(16, 16) })
		}
	})
	if segments.Track.Changed() {
		a.Preferences.Appearance = codes[selected]
		a.SavePreferences()
	}
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
				message, color = a.T("error"), c.Theme().Danger
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
				ui.Text(c, message).FontSize(12).SingleLine().Role(ui.RoleStatus)
			})
		})
	})
	if a.ExtensionStatus.State == browser.Failed && a.ExtensionStatus.Err != nil {
		ui.Text(c, a.T("error_extension_listen", a.Texts.Error(a.ExtensionStatus.Err))).FontSize(12).TextColor(c.Theme().Danger).Padding(0, 4).Role(ui.RoleAlert)
	}
}

func (a *App) aboutSettings(c *ui.Context, update UpdateCheckStatus) {
	ui.Column(c).FillWidth().Center().Gap(8).Padding(8, 0).Children(func() {
		ui.Image(c, appLogo).Size(64, 64)
		ui.Text(c, "Momento").FontSize(20).Bold()
		ui.Text(c, a.T("current_version", "2.0.0")).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(8).Margin(12, 0, 0, 0).Children(func() {
			a.updateButton(c, update)
			if ui.Button(c, a.T("open_logs")).Height(28).Clicked() {
				a.OpenLogDirectory()
			}
		})
	})
	a.updateMessage(c, update)
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

func (a *App) updateMessage(c *ui.Context, update UpdateCheckStatus) {
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
		ui.Text(c, text).FontSize(12).TextColor(color).Padding(0, 4).Role(ui.RoleStatus)
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
