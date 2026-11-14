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
	ui.Column(c).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+16), 24, 24, 24).Center().Gap(16).Material(panelMaterial(c.Theme())).DragWindow().Children(func() {
		a.welcomeContent(c, false)
	})
}

func (a *App) welcomeContent(c *ui.Context, onboarding bool) {
	ui.Column(c).Width(560).MaxWidthPercent(100).AlignItems(ui.Center).Gap(16).Children(func() {
		ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
		title := "welcome"
		if onboarding {
			title = "onboarding_library"
		}
		ui.Text(c, a.T(title)).FontSize(24).Bold().TextColor(titleColor(c.Theme())).FillWidth().TextAlign(ui.Center)
		ui.Text(c, a.T("welcome_hint")).TextColor(c.Theme().TextMuted).FillWidth().TextAlign(ui.Center)
		ui.Row(c).Gap(12).Margin(8, 0).Children(func() {
			if primaryButton(c, a.T("new_library"), false).Clicked() {
				a.ShowNewLibrary()
			}
			if ui.Button(c, a.T("open_library")).Clicked() {
				a.ChooseOpenLibrary()
			}
		})
		if len(a.Recent) > 0 {
			ui.Scroll(c).Width(360).MaxWidthPercent(100).MaxHeight(192).Gap(8).Children(func() {
				for _, recent := range a.Recent {
					button := ui.ButtonBase(c.Key(recent.Path)).FillWidth().Height(40).Padding(8, 12).Gap(8).
						Radius(8).Label(recent.DisplayName).Tooltip(recent.DisplayName).Cursor(ui.CursorPointer)
					if button.Pressed() {
						button.Background(warmOverlay(c.Theme(), .11))
					} else if button.Hovered() {
						button.Background(warmOverlay(c.Theme(), .06))
					} else {
						button.Background(c.Theme().Surface)
					}
					button.Children(func() {
						appIcon(c, "library").Size(20, 20)
						ui.Text(c, recent.DisplayName).Grow(1).SingleLine()
					})
					if button.Clicked() {
						a.OpenLibrary(recent.Path)
					}
				}
			})
		}
	})
}

func (a *App) onboarding(c *ui.Context) {
	a.Welcome.Step = max(0, min(2, a.Welcome.Step))
	step := a.Welcome.Step
	ui.Column(c).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+16), 24, 24, 24).Material(panelMaterial(c.Theme())).DragWindow().Children(func() {
		ui.Column(c).Grow(1).FillWidth().Center().Gap(16).Children(func() {
			switch step {
			case 0:
				ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
				ui.Text(c, a.T("onboarding_theme")).FontSize(24).Bold().TextColor(titleColor(c.Theme()))
				ui.Text(c, a.T("onboarding_theme_hint")).MaxWidth(560).TextAlign(ui.Center).TextColor(c.Theme().TextMuted).Margin(0, 0, 8, 0)
				a.ThemeSwatchesCentered(c)
			case 1:
				ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
				ui.Text(c, a.T("onboarding_extension")).FontSize(24).Bold().TextColor(titleColor(c.Theme()))
				ui.Text(c, a.T("onboarding_extension_hint")).MaxWidth(560).TextAlign(ui.Center).TextColor(c.Theme().TextMuted)
				if primaryButton(c, a.T("onboarding_get_extension"), false).Margin(12, 0, 0, 0).Clicked() {
					c.OpenURLThen(browserExtensionURL, a.ShowError)
				}
			case 2:
				a.welcomeContent(c, true)
			}
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(12, 16).Gap(12).Radius(12).Material(toolbarMaterial(c.Theme())).Border(1, c.Theme().Border).Children(func() {
			if step < 2 && ui.Button(c, a.T("onboarding_skip")).Clicked() {
				a.finishOnboarding()
			}
			ui.Box(c).Grow(1)
			ui.Text(c, a.T("onboarding_progress", step+1, 3)).FontSize(12).TextColor(c.Theme().TextMuted)
			ui.Box(c).Grow(1)
			if ui.Button(c, a.T("onboarding_previous")).Disabled(step == 0).Clicked() {
				a.Welcome.Step--
			}
			if step < 2 && primaryButton(c, a.T("onboarding_next"), false).Clicked() {
				a.Welcome.Step++
			}
		})
	})
}
