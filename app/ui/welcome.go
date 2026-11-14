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
	Step, Direction int
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
	fadeMotion(ui.Column(c.Key("welcome")).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+space16), space24, space24, space24).Center().Gap(space16).Material(panelMaterial(c.Theme())).DragWindow()).Children(func() {
		a.welcomeContent(c, false)
	})
}

func (a *App) welcomeContent(c *ui.Context, onboarding bool) {
	p := paletteFor(c.Theme())
	ui.Column(c).Width(560).MaxWidthPercent(100).AlignItems(ui.Center).Gap(space16).Children(func() {
		ui.Box(c).Size(space64, space64).Radius(space32).Background(p.Subtle).Center().Children(func() {
			appIcon(c, "library").Size(space32, space32).TextColor(p.TextSecondary)
		})
		title := "welcome"
		if onboarding {
			title = "onboarding_library"
		}
		ui.Text(c, a.T(title)).FontSize(16).FontWeight(600).TextColor(paletteFor(c.Theme()).Text).FillWidth().TextAlign(ui.Center)
		ui.Text(c, a.T("welcome_hint")).TextColor(paletteFor(c.Theme()).TextSecondary).FillWidth().TextAlign(ui.Center)
		ui.Row(c).Gap(space12).Margin(space8, 0).Children(func() {
			if primaryButton(c, a.T("new_library"), false).Clicked() {
				a.ShowNewLibrary()
			}
			if settingsButton(c, a.T("open_library")).Clicked() {
				a.ChooseOpenLibrary()
			}
		})
		if len(a.Recent) > 0 {
			ui.Scroll(c).Width(360).MaxWidthPercent(100).MaxHeight(192).Gap(space8).Children(func() {
				for _, recent := range a.Recent {
					button := ui.ButtonBase(c.Key(recent.Path)).FillWidth().Height(40).Padding(space8, space12).Gap(space8).
						Radius(radiusControl).Label(recent.DisplayName).Tooltip(recent.DisplayName).Cursor(ui.CursorPointer)
					elevated(button, c.Theme())
					background, text := p.Raised, p.TextSecondary
					if button.Pressed() {
						background, text = p.Pressed, p.Text
					} else if button.Hovered() {
						background, text = p.Hover, p.Text
					}
					animateColors(button, background, text)
					button.Children(func() {
						appIcon(c, "library").Size(18, 18)
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
	direction := float32(1)
	if a.Welcome.Direction < 0 {
		direction = -1
	}
	fadeMotion(ui.Column(c.Key("onboarding")).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+space16), space24, space24, space24).Material(panelMaterial(c.Theme())).DragWindow()).Children(func() {
		ui.Box(c.Key("onboarding-stage")).Grow(1).FillWidth().Clip().ChildExit(ui.Motion{X: -32 * direction}).Children(func() {
			ui.Column(c.Key(step)).Absolute().Fill().Center().Gap(space16).
				Transition(ui.ElementTransition{Animation: &motionSpring, Enter: &ui.Motion{X: 32 * direction}, Exit: &ui.Motion{X: -32 * direction}}).Children(func() {
				switch step {
				case 0:
					ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
					ui.Text(c, a.T("onboarding_theme")).FontSize(16).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
					ui.Text(c, a.T("onboarding_theme_hint")).MaxWidth(560).TextAlign(ui.Center).TextColor(paletteFor(c.Theme()).TextSecondary).Margin(0, 0, space8, 0)
					a.ThemeSwatchesCentered(c)
				case 1:
					ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
					ui.Text(c, a.T("onboarding_extension")).FontSize(16).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
					ui.Text(c, a.T("onboarding_extension_hint")).MaxWidth(560).TextAlign(ui.Center).TextColor(paletteFor(c.Theme()).TextSecondary)
					if primaryButton(c, a.T("onboarding_get_extension"), false).Margin(space12, 0, 0, 0).Clicked() {
						c.OpenURLThen(browserExtensionURL, a.ShowError)
					}
				case 2:
					a.welcomeContent(c, true)
				}
			})
		})
		elevated(ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(space12, space16).Gap(space12).Radius(radiusPanel).Material(toolbarMaterial(c.Theme())), c.Theme()).Children(func() {
			if step < 2 && settingsButton(c, a.T("onboarding_skip")).Clicked() {
				a.finishOnboarding()
			}
			ui.Box(c).Grow(1)
			ui.Text(c, a.T("onboarding_progress", step+1, 3)).Font(monoFont(c)).FontSize(fontNumber).TextColor(paletteFor(c.Theme()).TextSecondary)
			ui.Box(c).Grow(1)
			if settingsButton(c, a.T("onboarding_previous")).Disabled(step == 0).Clicked() {
				a.Welcome.Direction = -1
				a.Welcome.Step--
			}
			if step < 2 && primaryButton(c, a.T("onboarding_next"), false).Clicked() {
				a.Welcome.Direction = 1
				a.Welcome.Step++
			}
		})
	})
}
