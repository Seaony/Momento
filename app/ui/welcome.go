package ui

import (
	"github.com/Seaony/Momento/app/internal/library"
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
	RecentChecked   bool // 这次显示欢迎页时是否已在后台查过最近图库还在不在
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
	// 没有打开的图库时侧栏不出现，找不到的图库只能在这里处理，所以进来先查一遍。
	if !a.Welcome.RecentChecked && len(a.Recent) > 0 {
		a.Welcome.RecentChecked = true
		a.CheckRecentLibraries()
	}
	fadeMotion(ui.Column(c.Key("welcome")).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+space16), space24, space24, space24).Center().Gap(space16).Background(panelColor(c.Theme())).DragWindow()).Children(func() {
		a.welcomeContent(c, false)
	})
}

func (a *App) welcomeContent(c *ui.Context, onboarding bool) {
	p := paletteFor(c.Theme())
	ui.Column(c).Width(panelWide).MaxWidthPercent(100).AlignItems(ui.Center).Gap(space16).Children(func() {
		ui.Box(c).Size(space64, space64).Radius(space32).Background(p.Subtle).Center().Children(func() {
			appIcon(c, "library").Size(space32, space32).TextColor(p.TextSecondary)
		})
		title := "welcome"
		if onboarding {
			title = "onboarding_library"
		}
		ui.Text(c, a.T(title)).FontSize(fontTitle).FontWeight(600).TextColor(paletteFor(c.Theme()).Text).FillWidth().TextAlign(ui.Center)
		ui.Text(c, a.T("welcome_hint")).TextColor(paletteFor(c.Theme()).TextSecondary).FillWidth().TextAlign(ui.Center)
		ui.Row(c).Gap(space12).Margin(space8, 0).Children(func() {
			if dialogPrimaryButton(c, a.T("new_library"), false).Clicked() {
				a.ShowNewLibrary()
			}
			if dialogButton(c, a.T("open_library"), false).Clicked() {
				a.ChooseOpenLibrary()
			}
		})
		if len(a.Recent) > 0 {
			ui.Scroll(c).Scrollbars(false).Width(panelNarrow).MaxWidthPercent(100).MaxHeight(192).Gap(space8).Children(func() {
				for _, recent := range a.Recent {
					a.welcomeRecentRow(c, recent)
				}
			})
		}
	})
}

// welcomeRecentRow 是欢迎页里的一条最近图库。找不到的图库（外接盘没插、被移走）标出来，
// 行尾放和图库菜单同一个「…」菜单（重新定位、从列表中移除），和打开按钮并排，不嵌在按钮里。
func (a *App) welcomeRecentRow(c *ui.Context, recent library.Recent) {
	p := paletteFor(c.Theme())
	missing := a.RecentMissing[recent.Path]
	row := ui.Row(c.Key(recent.Path)).FillWidth().Height(space40).Padding(0, space8, 0, 0).Gap(space4).Radius(radiusControl).AlignItems(ui.Center)
	elevated(row, c.Theme())
	var open ui.Element
	moreOpen := false
	row.Children(func() {
		open = ui.ButtonBase(c.Key("open")).Grow(1).MinWidth(0).FillHeight().Padding(space8, space12).Gap(space8).AlignItems(ui.Center).
			Label(recent.DisplayName).Tooltip(libraryDisplayPath(recent.Path)).Cursor(ui.CursorPointer)
		open.Children(func() {
			appIcon(c, "library").Size(iconSize, iconSize)
			ui.Text(c, recent.DisplayName).Grow(1).MinWidth(0).SingleLine()
			if missing {
				ui.Text(c, a.T("library_not_found")).FontSize(fontSecondary).TextColor(p.TextSecondary).SingleLine().Shrink(0)
			}
		})
		if missing {
			if a.Sidebar.LibraryMore == nil {
				a.Sidebar.LibraryMore = make(map[string]*bool)
			}
			state := a.Sidebar.LibraryMore[recent.Path]
			if state == nil {
				state = new(bool)
				a.Sidebar.LibraryMore[recent.Path] = state
			}
			moreOpen = *state
			a.libraryMoreMenu(c, recent, false, true, state)
		}
	})
	background, text := p.Raised, p.TextSecondary
	if open.Pressed() {
		background, text = p.Pressed, p.Text
	} else if row.Hovered() || moreOpen {
		background, text = p.Hover, p.Text
	}
	animateColors(row, background, text)
	if open.Clicked() {
		a.OpenLibrary(recent.Path)
	}
}

func (a *App) onboarding(c *ui.Context) {
	a.Welcome.Step = max(0, min(2, a.Welcome.Step))
	step := a.Welcome.Step
	direction := float32(1)
	if a.Welcome.Direction < 0 {
		direction = -1
	}
	fadeMotion(ui.Column(c.Key("onboarding")).Fill().Padding(max(MainToolbarHeight, c.TitleBar().Bottom+space16), space24, space24, space24).Background(panelColor(c.Theme())).DragWindow()).Children(func() {
		ui.Box(c.Key("onboarding-stage")).Grow(1).FillWidth().Clip().ChildExit(ui.Motion{X: -32 * direction}).Children(func() {
			ui.Column(c.Key(step)).Absolute().Fill().Center().Gap(space16).
				Transition(ui.ElementTransition{Animation: &motionSpring, Enter: &ui.Motion{X: 32 * direction}, Exit: &ui.Motion{X: -32 * direction}}).Children(func() {
				switch step {
				case 0:
					ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
					ui.Text(c, a.T("onboarding_theme")).FontSize(fontTitle).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
					ui.Text(c, a.T("onboarding_theme_hint")).MaxWidth(panelWide).TextAlign(ui.Center).TextColor(paletteFor(c.Theme()).TextSecondary).Margin(0, 0, space8, 0)
					a.ThemeSwatchesCentered(c)
				case 1:
					ui.Image(c, appLogo).Size(96, 96).Fit(ui.Contain)
					ui.Text(c, a.T("onboarding_extension")).FontSize(fontTitle).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
					ui.Text(c, a.T("onboarding_extension_hint")).MaxWidth(panelWide).TextAlign(ui.Center).TextColor(paletteFor(c.Theme()).TextSecondary)
					if dialogPrimaryButton(c, a.T("onboarding_get_extension"), false).Margin(space12, 0, 0, 0).Clicked() {
						c.OpenURLThen(browserExtensionURL, a.ShowError)
					}
				case 2:
					a.welcomeContent(c, true)
				}
			})
		})
		elevated(ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(space12, space16).Gap(space12).Radius(radiusPanel).Background(c.Theme().Background), c.Theme()).Children(func() {
			if step < 2 && dialogButton(c, a.T("onboarding_skip"), false).Clicked() {
				a.finishOnboarding()
			}
			ui.Box(c).Grow(1)
			ui.Text(c, a.T("onboarding_progress", step+1, 3)).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
			ui.Box(c).Grow(1)
			if dialogButton(c, a.T("onboarding_previous"), step == 0).Clicked() {
				a.Welcome.Direction = -1
				a.Welcome.Step--
			}
			if step < 2 && dialogPrimaryButton(c, a.T("onboarding_next"), false).Clicked() {
				a.Welcome.Direction = 1
				a.Welcome.Step++
			}
		})
	})
}
