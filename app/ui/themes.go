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
	// 表单控件：Field 浅底 + BorderSubtle 浅边框一起表现层次，聚焦时换成浮起面 + 焦点色。
	Field, FieldHover                                         ui.Color
	Accent, AccentHover, AccentPressed, OnAccent, Focus, Link ui.Color
	Danger, Warning, Success, Shadow, Scrim                   ui.Color
}

var lightPalette = designPalette{
	Background: ui.Hex("#FFFCFA"), Sidebar: ui.Hex("#FFF7F1"), Raised: ui.Hex("#FFFFFF"),
	Subtle: ui.Hex("#FDF1E8"), Hover: ui.Hex("#FBECE0"), Pressed: ui.Hex("#F7E2D2"), Selected: ui.Hex("#FFFFFF"),
	BorderSubtle: ui.Hex("#F3E3D6"), Border: ui.Hex("#EAD6C6"),
	Text: ui.Hex("#2B221D"), TextSecondary: ui.Hex("#6F5F55"), TextTertiary: ui.Hex("#7D6B60"),
	SegmentSelected: ui.Hex("#FFFFFF"),
	Field:           ui.Hex("#F7EEE6"), FieldHover: ui.Hex("#F3E8DE"),
	Accent: ui.Hex("#F47B45"), AccentHover: ui.Hex("#FA8B59"), AccentPressed: ui.Hex("#E8733D"),
	OnAccent: ui.Hex("#2B221D"), Focus: ui.Hex("#D9622C"), Link: ui.Hex("#A74320"),
	Danger: ui.Hex("#B91C1C"), Warning: ui.Hex("#8F4500"), Success: ui.Hex("#166534"),
	Shadow: ui.RGBA(92, 56, 32, .07), Scrim: ui.RGBA(43, 34, 29, .25),
}

// 深色以近黑为主，只留一点暖意；橘色只在强调色上出现。
var darkPalette = designPalette{
	Background: ui.Hex("#111010"), Sidebar: ui.Hex("#171615"), Raised: ui.Hex("#201E1D"),
	Subtle: ui.Hex("#1A1918"), Hover: ui.Hex("#262422"), Pressed: ui.Hex("#2E2B29"), Selected: ui.Hex("#232120"),
	BorderSubtle: ui.Hex("#252321"), Border: ui.Hex("#322F2D"),
	Text: ui.Hex("#F4F0ED"), TextSecondary: ui.Hex("#B6AEA8"), TextTertiary: ui.Hex("#938A84"),
	SegmentSelected: ui.Hex("#2F2C2A"),
	Field:           ui.Hex("#222020"), FieldHover: ui.Hex("#282625"),
	Accent: ui.Hex("#FF9A66"), AccentHover: ui.Hex("#FFAB7E"), AccentPressed: ui.Hex("#F48A54"),
	OnAccent: ui.Hex("#2B221D"), Focus: ui.Hex("#FF9A66"), Link: ui.Hex("#FFB48F"),
	Danger: ui.Hex("#FF9B9B"), Warning: ui.Hex("#FCD478"), Success: ui.Hex("#88DDA9"),
	Shadow: ui.RGBA(0, 0, 0, .36), Scrim: ui.RGBA(0, 0, 0, .55),
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
	fontTitle     = 16 // 对话框、空状态标题
	fontHeading   = 20 // 独立窗口大标题
	fontDisplay   = 24 // 关于页产品名

	iconSize      = 18 // 侧栏、工具栏图标
	iconMedium    = 16 // 菜单、检查器里的图标
	iconSmall     = 14 // 箭头、行内小图标
	iconTiny      = 12 // 药丸里的移除按钮
	controlHeight = 32 // 按钮、输入框
	controlSmall  = 28 // 紧凑按钮、图标按钮
	controlLarge  = 36 // 主要入口按钮
	checkSize     = 16 // 复选框、单选框
	rowHeight     = 44 // 设置导航、菜单里的两行项

	// 侧栏照旧版 macOS（git 2a13147^ MomentoSidebarView.swift）抄的规格。
	sidebarPadding          = 14 // 侧栏左右留白
	sidebarLibraryGap       = 18 // 图库选择器和第一行之间
	sidebarLibraryInset     = 5  // 图库选择器左边留白
	sidebarLogoRadius       = 5
	sidebarLibraryHeight    = 26
	sidebarRowHeight        = 30
	sidebarRowGap           = 2
	sidebarRowPadding       = 8 // 行内左右留白
	sidebarRowRadius        = 9
	sidebarIconBox          = 18 // 图标占位宽
	sidebarIconSize         = iconMedium
	sidebarIconGap          = 10
	sidebarWeight           = 500
	sidebarCountSize        = fontGroup
	sidebarCountGap         = 6
	sidebarBadgeSize        = 8 // 回收站角标
	sidebarDividerInset     = 12
	sidebarDividerSpace     = 6 // 分隔线上下各留
	sidebarSectionGap       = 18
	sidebarGroupGap         = 6  // 入口组、分隔线、文件夹分区之间
	sidebarHeader           = 32 // 「快速访问」标题行
	sidebarLibraryEnd       = 7  // 图库选择器右边留白
	sidebarFolderIndent     = 18
	sidebarFooterButtonSize = 28
	sidebarFooterGap        = 6
	sidebarFooterPadding    = 9

	panelNarrow = 360 // 弹出层、引导里的窄面板
	panelDialog = 420 // 对话框
	panelWide   = 560 // 宽面板、引导页

	materialPanelAlpha   = .88 // 弹出层、菜单、对话框磨砂的着色；再低压在纯黑图片上时次要文字对比度不到 4.5
	materialSidebarAlpha = .86 // 侧栏系统材质上的着色（浅色）
	materialSidebarDark  = .70 // 侧栏系统材质上的着色（深色）
	materialOverlayAlpha = .92 // 压在图片上的小按钮，保证图标看得清
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
	theme.Background, theme.Surface = p.Background, p.Field
	theme.SurfaceHover, theme.SurfacePressed = p.FieldHover, p.Pressed
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
	row := ui.Row(c).Gap(space12).AlignItems(ui.Center)
	if centered {
		row.Justify(ui.Center)
	}
	row.Children(func() {
		for _, key := range themeKeys {
			selected := a.Preferences.Theme == key || (a.Preferences.Theme == "" && key == "system")
			name := a.T(themeNameKey(key))
			button := ui.ButtonBase(c.Key(key)).Width(92).Padding(space4).Column().Gap(space8).AlignItems(ui.Center).Radius(radiusControl).Cursor(ui.CursorPointer).Label(name).Tooltip(name).AccessSelected(selected)
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
				ui.Box(c).Size(76, 48).Radius(radiusControl).Draw(func(p *ui.Painter, r ui.Rect) {
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
				ui.Text(c, name).FontSize(fontSecondary).TextColor(text).SingleLine()
			})
			if button.Clicked() {
				a.Preferences.Theme = key
				a.SavePreferences()
			}
		}
	})
}
