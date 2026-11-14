package ui

import (
	"fmt"
	"math"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var themeKeys = []string{"system", "light", "dark"}

type designPalette struct {
	Background, Sidebar, Raised, Subtle, Hover, Pressed, Selected ui.Color
	BorderSubtle, Border, Text, TextSecondary, TextTertiary       ui.Color
	SegmentSelected                                               ui.Color
	Accent, AccentHover, AccentPressed, OnAccent, Focus, Link     ui.Color
	Danger, Warning, Success, Shadow, Scrim                       ui.Color
}

var lightPalette = designPalette{
	Background: ui.Hex("#FFFCFA"), Sidebar: ui.Hex("#FFF7F1"), Raised: ui.Hex("#FFFFFF"),
	Subtle: ui.Hex("#FDF1E8"), Hover: ui.Hex("#FBECE0"), Pressed: ui.Hex("#F7E2D2"), Selected: ui.Hex("#FFFFFF"),
	BorderSubtle: ui.Hex("#F3E3D6"), Border: ui.Hex("#EAD6C6"),
	Text: ui.Hex("#2B221D"), TextSecondary: ui.Hex("#6F5F55"), TextTertiary: ui.Hex("#7D6B60"),
	SegmentSelected: ui.Hex("#FFFFFF"),
	Accent:          ui.Hex("#F47B45"), AccentHover: ui.Hex("#FA8B59"), AccentPressed: ui.Hex("#E8733D"),
	OnAccent: ui.Hex("#2B221D"), Focus: ui.Hex("#D9622C"), Link: ui.Hex("#A74320"),
	Danger: ui.Hex("#B91C1C"), Warning: ui.Hex("#8F4500"), Success: ui.Hex("#166534"),
	Shadow: ui.RGBA(92, 56, 32, .07), Scrim: ui.RGBA(43, 34, 29, .25),
}

var darkPalette = designPalette{
	Background: ui.Hex("#1C1714"), Sidebar: ui.Hex("#231C17"), Raised: ui.Hex("#31271F"),
	Subtle: ui.Hex("#2A211B"), Hover: ui.Hex("#382B22"), Pressed: ui.Hex("#433126"), Selected: ui.Hex("#31271F"),
	BorderSubtle: ui.Hex("#3A2D24"), Border: ui.Hex("#4B3A2E"),
	Text: ui.Hex("#F6EEE7"), TextSecondary: ui.Hex("#CFBCAE"), TextTertiary: ui.Hex("#B29886"),
	SegmentSelected: ui.Hex("#4D3D31"),
	Accent:          ui.Hex("#FF9A66"), AccentHover: ui.Hex("#FFAB7E"), AccentPressed: ui.Hex("#F48A54"),
	OnAccent: ui.Hex("#2B221D"), Focus: ui.Hex("#FF9A66"), Link: ui.Hex("#FFB48F"),
	Danger: ui.Hex("#FF9B9B"), Warning: ui.Hex("#FCD478"), Success: ui.Hex("#88DDA9"),
	Shadow: ui.RGBA(0, 0, 0, .24), Scrim: ui.RGBA(28, 23, 20, .60),
}

func paletteFor(t *ui.Theme) *designPalette {
	if t.Dark {
		return &darkPalette
	}
	return &lightPalette
}

const (
	space4        = 4
	space8        = 8
	space12       = 12
	space16       = 16
	space20       = 20
	space24       = 24
	space32       = 32
	space40       = 40
	space48       = 48
	space64       = 64
	radiusSmall   = 4
	radiusCard    = 6
	radiusControl = 8
	radiusPanel   = 12
	fontBody      = 13
	fontSecondary = 12
	fontGroup     = 11
	fontNumber    = 12
)

func nativeColorPair(light, dark ui.Color) string {
	return fmt.Sprintf("light-dark(rgba(%d,%d,%d,%.3f), rgba(%d,%d,%d,%.3f))", light.R, light.G, light.B, float64(light.A)/255, dark.R, dark.G, dark.B, float64(dark.A)/255)
}

func WindowBackgroundColor() string {
	return nativeColorPair(lightPalette.Background, darkPalette.Background)
}
func previewGlassTint() string {
	return nativeColorPair(lightPalette.Sidebar.Alpha(.28), darkPalette.Sidebar.Alpha(.86))
}

// 系统主题只决定明暗，所有窗口共用暖橘品牌色。
func momentoTheme(key string, systemDark bool) *ui.Theme {
	dark := key == "dark" || ((key == "system" || key == "") && systemDark)
	theme := *ui.LightTheme()
	if dark {
		theme = *ui.DarkTheme()
	}
	theme.Dark = dark
	p := paletteFor(&theme)
	theme.Radius, theme.FontSize, theme.Spacing, theme.Font = radiusControl, fontBody, space4, uiFont("en")
	theme.Background, theme.Surface = p.Background, p.Raised
	theme.SurfaceHover, theme.SurfacePressed = p.Hover, p.Pressed
	theme.Text, theme.TextMuted = p.Text, p.TextSecondary
	theme.Border = p.BorderSubtle
	theme.Accent, theme.AccentHover, theme.AccentPressed, theme.AccentText = p.Accent, p.AccentHover, p.AccentPressed, p.OnAccent
	theme.InactiveAccent = p.Accent
	theme.Danger, theme.Warning, theme.Success = p.Danger, p.Warning, p.Success
	theme.Selection, theme.Focus, theme.Scrollbar = p.Accent.Alpha(.2), p.Focus, p.TextSecondary.Alpha(.4)
	theme.Inverse, theme.InverseText = theme.Text, theme.Background
	return &theme
}

func titleColor(t *ui.Theme) ui.Color {
	return paletteFor(t).Text
}

func linkColor(t *ui.Theme) ui.Color {
	return paletteFor(t).Link
}

func (a *App) applyTheme(c *ui.Context) {
	c.SetZoom(preferenceZoom(a.Preferences.UIZoom))
	if a.Preferences.Language != "system" && a.Preferences.Language != "" {
		a.Texts.Language = a.Preferences.Language
	}
	theme := momentoTheme(a.Preferences.Theme, c.Theme().Dark)
	theme.Font = uiFont(a.Texts.Locale())
	c.SetTheme(theme)
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	} else {
		c.Root().Background(theme.Background)
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
						p.Fill(ui.Rect{X: box.X + 20, Y: box.Y + 8, W: max(0, box.W-28), H: 4}, colors.Border, 2)
						p.Fill(ui.Rect{X: box.X + 20, Y: box.Y + 20, W: max(0, box.W-28), H: 20}, colors.Surface, radiusSmall)
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
					p.Stroke(ui.Rect{X: r.X - 4, Y: r.Y - 4, W: r.W + 8, H: r.H + 8}, outline, radiusPanel, 2)
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
