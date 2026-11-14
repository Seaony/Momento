package ui

import (
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var themeKeys = []string{"system", "light", "dark"}

// 系统主题只决定明暗，所有窗口共用暖橘品牌色。
func momentoTheme(key string, systemDark bool) *ui.Theme {
	dark := key == "dark" || ((key == "system" || key == "") && systemDark)
	theme := *ui.LightTheme()
	if dark {
		theme = *ui.DarkTheme()
	}
	theme.Dark = dark
	theme.Radius = 8
	theme.FontSize = 13
	if dark {
		theme.Background, theme.Surface = ui.Hex("#211A16"), ui.Hex("#2C231E")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#382D26"), ui.Hex("#44362D")
		theme.Text, theme.TextMuted = ui.Hex("#F8EEE5"), ui.Hex("#D0B49F")
		theme.Border = ui.Hex("#483A30")
		theme.Accent, theme.AccentHover = ui.Hex("#FF9A66"), ui.Hex("#FFAC80")
		theme.AccentPressed, theme.AccentText = ui.Hex("#F48A54"), ui.Hex("#2E1B12")
		theme.InactiveAccent = ui.Hex("#D5A88D")
		theme.Danger, theme.Warning, theme.Success = ui.Hex("#FF9B9B"), ui.Hex("#FCD478"), ui.Hex("#88DDA9")
		theme.Selection, theme.Focus = ui.RGBA(255, 154, 102, .24), theme.Accent
		theme.Scrollbar = ui.RGBA(232, 190, 158, .4)
	} else {
		theme.Background, theme.Surface = ui.Hex("#FFFAF5"), ui.Hex("#FBF0E5")
		theme.SurfaceHover, theme.SurfacePressed = ui.Hex("#F5EBE0"), ui.Hex("#EEE2D6")
		theme.Text, theme.TextMuted = ui.Hex("#3A2318"), ui.Hex("#7E5E4C")
		theme.Border = ui.Hex("#F1E6DC")
		theme.Accent, theme.AccentHover = ui.Hex("#F47B45"), ui.Hex("#FA8B59")
		theme.AccentPressed, theme.AccentText = ui.Hex("#E8733D"), ui.Hex("#3A2318")
		theme.InactiveAccent = ui.Hex("#DCA180")
		theme.Danger, theme.Warning, theme.Success = ui.Hex("#B91C1C"), ui.Hex("#8F4500"), ui.Hex("#166534")
		theme.Selection, theme.Focus = ui.RGBA(244, 123, 69, .20), ui.Hex("#D9622C")
		theme.Scrollbar = ui.RGBA(120, 72, 40, .4)
	}
	theme.Inverse, theme.InverseText = theme.Text, theme.Background
	return &theme
}

func titleColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#FFF7EF")
	}
	return ui.Hex("#2E1B12")
}

func linkColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#FFB48F")
	}
	return ui.Hex("#9E3514")
}

func warmOverlay(t *ui.Theme, alpha float32) ui.Color {
	if t.Dark {
		return ui.RGBA(232, 190, 158, alpha)
	}
	return ui.RGBA(120, 72, 40, alpha)
}

func (a *App) applyTheme(c *ui.Context) {
	c.SetZoom(preferenceZoom(a.Preferences.UIZoom))
	c.SetTheme(momentoTheme(a.Preferences.Theme, c.Theme().Dark))
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	if a.Preferences.Language != "system" && a.Preferences.Language != "" {
		a.Texts.Language = a.Preferences.Language
	}
}

func themeSource(key string) mygo.ThemeSource {
	switch key {
	case "light":
		return mygo.ThemeLight
	case "dark":
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
	case "dark":
		return "appearance_dark"
	default:
		return "appearance_system"
	}
}

// 设置和引导共用色块，名称也用于读屏标签和悬停提示。
func (a *App) ThemeSwatches(c *ui.Context) {
	a.themeSwatches(c, false)
}

func (a *App) ThemeSwatchesCentered(c *ui.Context) {
	a.themeSwatches(c, true)
}

func (a *App) themeSwatches(c *ui.Context, centered bool) {
	theme := c.Theme()
	row := ui.Row(c).Gap(12).AlignItems(ui.Center)
	if centered {
		row.Justify(ui.Center)
	}
	row.Children(func() {
		for _, key := range themeKeys {
			selected := a.Preferences.Theme == key || (a.Preferences.Theme == "" && key == "system")
			name := a.T(themeNameKey(key))
			button := ui.ButtonBase(c.Key(key)).Width(92).Padding(4).Column().Gap(8).AlignItems(ui.Center).Radius(8).Cursor(ui.CursorPointer).Label(name).Tooltip(name).AccessSelected(selected)
			hovered := button.Hovered()
			text := theme.TextMuted
			outline := ui.Transparent
			if selected {
				text, outline = theme.Text, theme.Accent
			} else if hovered {
				text, outline = theme.Text, theme.Focus
			}
			text = button.AnimateColor("theme-label", text, motionColor)
			outline = button.AnimateColor("theme-outline", outline, motionColor)
			button.Children(func() {
				ui.Box(c).Size(76, 48).Radius(8).Draw(func(p *ui.Painter, r ui.Rect) {
					palette := momentoTheme(key, theme.Dark)
					draw := func(box ui.Rect, colors *ui.Theme) {
						p.Fill(box, colors.Background, 0)
						p.Fill(ui.Rect{X: box.X, Y: box.Y, W: 16, H: box.H}, colors.Surface, 0)
						p.Fill(ui.Rect{X: box.X + 21, Y: box.Y + 10, W: max(0, box.W-28), H: 4}, colors.Border, 2)
						p.Fill(ui.Rect{X: box.X + 21, Y: box.Y + 21, W: max(0, box.W-28), H: 18}, colors.Surface, 3)
					}
					p.Clip(r, 8, func() {
						if key == "system" {
							p.Clip(ui.Rect{X: r.X, Y: r.Y, W: r.W / 2, H: r.H}, 0, func() { draw(r, momentoTheme("light", false)) })
							p.Clip(ui.Rect{X: r.X + r.W/2, Y: r.Y, W: r.W / 2, H: r.H}, 0, func() { draw(r, momentoTheme("dark", true)) })
						} else {
							draw(r, palette)
						}
					})
					p.Stroke(r, theme.Border, 8, 1)
					p.Stroke(ui.Rect{X: r.X - 3, Y: r.Y - 3, W: r.W + 6, H: r.H + 6}, outline, 11, 2)
				})
				ui.Text(c, name).FontSize(12).TextColor(text).SingleLine()
			})
			if button.Clicked() {
				a.Preferences.Theme = key
				a.SavePreferences()
			}
		}
	})
}
