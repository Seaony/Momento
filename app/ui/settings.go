package ui

import "github.com/egoist/mygo/ui"

type settingsState struct{}

func (a *App) settings(c *ui.Context) {
	settingsPanel(c.Key("settings"), &a.SettingsOpen, 440, func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("settings")).FontSize(20).Bold().Grow(1)
			if ui.Button(c, a.T("close")).Clicked() {
				a.SettingsOpen = false
			}
		})
		ui.Divider(c)
		ui.Text(c, a.T("language")).Bold()
		languageLabels := []string{a.T("follow_system"), a.T("language_english"), a.T("language_chinese")}
		languageCodes := []string{"system", "en", "zh-CN"}
		language := languageLabels[0]
		for i, code := range languageCodes {
			if a.Preferences.Language == code {
				language = languageLabels[i]
			}
		}
		if ui.Select(c.Key("language"), &language, languageLabels).Label(a.T("language")).FillWidth().Changed() {
			for i, label := range languageLabels {
				if language == label {
					a.Preferences.Language = languageCodes[i]
					a.SavePreferences()
					break
				}
			}
		}
		ui.Text(c, a.T("appearance")).Bold().Margin(4, 0, 0, 0)
		appearanceLabels := []string{a.T("follow_system"), a.T("appearance_light"), a.T("appearance_dark")}
		appearanceCodes := []string{"system", "light", "dark"}
		appearance := appearanceLabels[0]
		for i, code := range appearanceCodes {
			if a.Preferences.Appearance == code {
				appearance = appearanceLabels[i]
			}
		}
		if ui.Select(c.Key("appearance"), &appearance, appearanceLabels).Label(a.T("appearance")).FillWidth().Changed() {
			for i, label := range appearanceLabels {
				if appearance == label {
					a.Preferences.Appearance = appearanceCodes[i]
					a.SavePreferences()
					break
				}
			}
		}
		ui.Divider(c).Margin(8, 0)
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("version")).Grow(1)
			ui.Text(c, "2.0.0").TextColor(c.Theme().TextMuted)
		})
		if ui.Button(c, a.T("check_updates")).FillWidth().Clicked() {
			CheckForUpdates()
		}
	})
}

func settingsPanel(c *ui.Context, open *bool, width float32, body func()) ui.Element {
	return ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.42))
		panel.Width(width).MaxWidthPercent(92).MaxHeightPercent(92).Padding(24).Gap(12).
			Radius(12).Background(c.Theme().Background).Border(1, c.Theme().Border)
		body()
	})
}
