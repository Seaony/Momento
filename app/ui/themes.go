package ui

import (
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var themeKeys = []string{"system", "light", "light-gray", "dark-gray", "dark", "navy", "purple"}

// 系统主题只决定明暗，所有窗口的颜色仍从同一套色板读取。
func momentoTheme(key string, systemDark bool) *ui.Theme {
	if key == "system" || key == "" {
		if systemDark {
			key = "dark"
		} else {
			key = "light"
		}
	}
	dark := key == "dark-gray" || key == "dark" || key == "navy" || key == "purple"
	theme := *ui.LightTheme()
	if dark {
		theme = *ui.DarkTheme()
	}
	theme.Dark = dark
	switch key {
	case "light-gray":
		theme.Background, theme.Surface = ui.Hex("#E9E9ED"), ui.Hex("#DEDEE5")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#D4D4DC"), ui.Hex("#C8C8D2")
		theme.Border, theme.Text, theme.TextMuted = ui.Hex("#B8B8C4"), ui.Hex("#202028"), ui.Hex("#50505C")
	case "dark-gray":
		theme.Background, theme.Surface = ui.Hex("#303034"), ui.Hex("#3A3A40")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#414148"), ui.Hex("#48484F")
		theme.Border, theme.Text, theme.TextMuted = ui.Hex("#64646E"), ui.Hex("#FAFAFC"), ui.Hex("#C8C8D2")
	case "dark":
		theme.Background, theme.Surface = ui.Hex("#18181B"), ui.Hex("#27272A")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#323236"), ui.Hex("#3C3C41")
		theme.Border, theme.Text, theme.TextMuted = ui.Hex("#3F3F46"), ui.Hex("#F4F4F5"), ui.Hex("#B4B4BF")
	case "navy":
		theme.Background, theme.Surface = ui.Hex("#161E30"), ui.Hex("#222E45")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#2B3953"), ui.Hex("#34435E")
		theme.Border, theme.Text, theme.TextMuted = ui.Hex("#465978"), ui.Hex("#F0F5FF"), ui.Hex("#BECCE2")
	case "purple":
		theme.Background, theme.Surface = ui.Hex("#251D32"), ui.Hex("#332842")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#3D314F"), ui.Hex("#473A59")
		theme.Border, theme.Text, theme.TextMuted = ui.Hex("#615173"), ui.Hex("#F8F2FF"), ui.Hex("#D0BFDF")
	default:
		theme.Background, theme.Surface = ui.Hex("#FFFFFF"), ui.Hex("#F4F4F5")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#E9E9EC"), ui.Hex("#DDDDE1")
		theme.Border, theme.Text, theme.TextMuted = ui.Hex("#D9D9DE"), ui.Hex("#18181B"), ui.Hex("#5E5E68")
	}
	if dark {
		theme.Accent, theme.AccentHover = ui.Hex("#93C5FD"), ui.Hex("#BFDBFE")
		theme.AccentPressed, theme.AccentText = ui.Hex("#60A5FA"), ui.Hex("#111827")
		theme.InactiveAccent = ui.Hex("#B9BECA")
		theme.Danger, theme.Warning, theme.Success = ui.Hex("#FF9B9B"), ui.Hex("#FCD478"), ui.Hex("#88DDA9")
		theme.Selection, theme.Focus = ui.RGBA(147, 197, 253, .30), ui.Hex("#93C5FD")
		theme.Inverse, theme.InverseText = theme.Text, theme.Background
		theme.Scrollbar = ui.RGBA(240, 240, 255, .4)
	} else {
		theme.Accent, theme.AccentHover = ui.Hex("#1D4ED8"), ui.Hex("#1E40AF")
		theme.AccentPressed, theme.AccentText = ui.Hex("#1E3A8A"), ui.Hex("#FFFFFF")
		theme.InactiveAccent = ui.Hex("#5B6170")
		theme.Danger, theme.Warning, theme.Success = ui.Hex("#B91C1C"), ui.Hex("#8F4500"), ui.Hex("#166534")
		theme.Selection, theme.Focus = ui.RGBA(29, 78, 216, .22), ui.Hex("#1D4ED8")
		theme.Inverse, theme.InverseText = theme.Text, theme.Background
		theme.Scrollbar = ui.RGBA(24, 24, 35, .4)
	}
	return &theme
}

func (a *App) applyTheme(c *ui.Context) {
	c.SetZoom(preferenceZoom(a.Preferences.UIZoom))
	c.SetTheme(momentoTheme(a.Preferences.Theme, c.Theme().Dark))
	if a.Preferences.Language != "system" && a.Preferences.Language != "" {
		a.Texts.Language = a.Preferences.Language
	}
}

func themeSource(key string) mygo.ThemeSource {
	switch key {
	case "light", "light-gray":
		return mygo.ThemeLight
	case "dark-gray", "dark", "navy", "purple":
		return mygo.ThemeDark
	default:
		return mygo.ThemeSystem
	}
}

func (a *App) applyAppearance() { mygo.Theme.SetSource(themeSource(a.Preferences.Theme)) }

func preferenceZoom(zoom float32) float32 {
	switch zoom {
	case .8, .9, 1, 1.1, 1.25, 1.5:
		return zoom
	}
	return 1
}

func zoomedWindowSize(width, height int, zoom float32, workArea mygo.Rectangle) (int, int) {
	width = int(math.Ceil(float64(float32(width) * zoom)))
	height = int(math.Ceil(float64(float32(height) * zoom)))
	if workArea.Width > 0 {
		width = min(width, workArea.Width)
	}
	if workArea.Height > 0 {
		height = min(height, workArea.Height)
	}
	return width, height
}

// 窗口尺寸用系统单位；内容缩放后同步放大，放不下时止于屏幕可用范围。
func (a *App) ApplyWindowZoom() {
	zoom := preferenceZoom(a.Preferences.UIZoom)
	if a.Window != nil {
		area := mygo.Screen.DisplayMatching(a.Window.Bounds()).WorkArea
		minW, minH := zoomedWindowSize(1100, 640, zoom, area)
		a.Window.SetMinimumSize(minW, minH)
		width, height := a.Window.ContentSize()
		width, height = max(width, minW), max(height, minH)
		if area.Width > 0 {
			width = min(width, area.Width)
		}
		if area.Height > 0 {
			height = min(height, area.Height)
		}
		a.Window.SetContentSize(width, height)
	}
	if a.SettingsWindow != nil {
		area := mygo.Screen.DisplayMatching(a.SettingsWindow.Bounds()).WorkArea
		width, height := zoomedWindowSize(760, 560, zoom, area)
		a.SettingsWindow.SetMinimumSize(0, 0)
		a.SettingsWindow.SetMaximumSize(0, 0)
		a.SettingsWindow.SetContentSize(width, height)
		a.SettingsWindow.SetMinimumSize(width, height)
		a.SettingsWindow.SetMaximumSize(width, height)
	}
	a.InvalidateWindows()
}

func themeNameKey(key string) string {
	switch key {
	case "light":
		return "appearance_light"
	case "light-gray":
		return "theme_light_gray"
	case "dark-gray":
		return "theme_dark_gray"
	case "dark":
		return "appearance_dark"
	case "navy":
		return "theme_navy"
	case "purple":
		return "theme_purple"
	default:
		return "follow_system"
	}
}

// 设置和引导共用色块，名称也用于读屏标签和悬停提示。
func (a *App) ThemeSwatches(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Gap(8).Children(func() {
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			for _, key := range themeKeys {
				selected := a.Preferences.Theme == key || (a.Preferences.Theme == "" && key == "system")
				name := a.T(themeNameKey(key))
				button := ui.ButtonBase(c.Key(key)).Size(28, 28).Padding(0).Radius(14).Label(name).Tooltip(name).AccessSelected(selected)
				hovered := button.Hovered()
				color := momentoTheme(key, c.Theme().Dark).Background
				button.Draw(func(p *ui.Painter, r ui.Rect) {
					p.Clip(r, 14, func() {
						p.Fill(r, color, 14)
						if key == "system" {
							p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W / 2, H: r.H}, momentoTheme("light", false).Background, 0)
							p.Fill(ui.Rect{X: r.X + r.W/2, Y: r.Y, W: r.W / 2, H: r.H}, momentoTheme("dark", true).Background, 0)
						}
					})
					border, width := theme.Border, float32(1)
					if selected {
						border, width = theme.Accent, 2
					} else if hovered {
						border = theme.AccentHover
					}
					p.Stroke(r, border, 14, width)
				})
				if button.Clicked() {
					a.Preferences.Theme = key
					a.SavePreferences()
				}
			}
		})
		ui.Text(c, a.T(themeNameKey(a.Preferences.Theme))).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine()
	})
}
