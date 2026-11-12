package ui

import (
	"github.com/Seaony/Momento/app/resources"
	"github.com/egoist/mygo/ui"
)

var appLogo = func() *ui.Bitmap {
	bitmap, err := ui.DecodeBitmap(resources.WelcomeIcon)
	if err != nil {
		panic(err)
	}
	return bitmap
}()

const browserExtensionURL = "https://github.com/Seaony/Momento-Chomre-Extension/releases/latest"

type welcomeState struct {
	Step int
}

func (a *App) needsOnboarding() bool {
	return !a.Preferences.Onboarded && len(a.Recent) == 0
}

func (a *App) finishOnboarding() {
	a.Preferences.Onboarded = true
	a.SavePreferences()
}

func (a *App) welcome(c *ui.Context) {
	if a.needsOnboarding() {
		a.onboarding(c)
		return
	}
	ui.Column(c).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+16), 24, 24, 24).Center().Gap(16).DragWindow().Children(func() {
		a.welcomeContent(c, false)
	})
}

func (a *App) welcomeContent(c *ui.Context, onboarding bool) {
	ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
	ui.Text(c, a.T("welcome")).FontSize(24).Bold()
	ui.Text(c, a.T("welcome_hint")).TextColor(c.Theme().TextMuted)
	ui.Row(c).Gap(12).Margin(12, 0).Children(func() {
		if ui.PrimaryButton(c, a.T("new_library")).Clicked() {
			if onboarding {
				a.finishOnboarding()
			}
			a.ShowNewLibrary()
		}
		if ui.Button(c, a.T("open_library")).Clicked() {
			if onboarding {
				a.finishOnboarding()
			}
			a.ChooseOpenLibrary()
		}
	})
	for _, recent := range a.Recent {
		if ui.Button(c.Key(recent.Path), recent.DisplayName).Width(320).Clicked() {
			a.OpenLibrary(recent.Path)
		}
	}
}

func (a *App) onboarding(c *ui.Context) {
	a.Welcome.Step = max(0, min(2, a.Welcome.Step))
	step := a.Welcome.Step
	ui.Column(c).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+16), 24, 24, 24).DragWindow().Children(func() {
		ui.Column(c).Grow(1).FillWidth().Center().Gap(16).Children(func() {
			switch step {
			case 0:
				ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
				ui.Text(c, a.T("onboarding_theme")).FontSize(24).Bold()
				ui.Text(c, a.T("onboarding_theme_hint")).TextColor(c.Theme().TextMuted).Margin(0, 0, 12, 0)
				a.ThemeSwatches(c)
			case 1:
				ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
				ui.Text(c, a.T("onboarding_extension")).FontSize(24).Bold()
				ui.Text(c, a.T("onboarding_extension_hint")).TextColor(c.Theme().TextMuted)
				if ui.PrimaryButton(c, a.T("onboarding_get_extension")).Margin(12, 0, 0, 0).Clicked() {
					c.OpenURLThen(browserExtensionURL, a.ShowError)
				}
			case 2:
				a.welcomeContent(c, true)
			}
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			if ui.Button(c, a.T("onboarding_skip")).Clicked() {
				a.finishOnboarding()
			}
			ui.Box(c).Grow(1)
			ui.Text(c, a.T("onboarding_progress", step+1, 3)).FontSize(12).TextColor(c.Theme().TextMuted)
			ui.Box(c).Grow(1)
			if ui.Button(c, a.T("onboarding_previous")).Disabled(step == 0).Clicked() {
				a.Welcome.Step--
			}
			key := "onboarding_next"
			if step == 2 {
				key = "onboarding_finish"
			}
			if ui.PrimaryButton(c, a.T(key)).Clicked() {
				if step == 2 {
					a.finishOnboarding()
				} else {
					a.Welcome.Step++
				}
			}
		})
	})
}
