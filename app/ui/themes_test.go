package ui

import (
	"math"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func themeContrast(a, b ui.Color) float64 {
	luminance := func(color ui.Color) float64 {
		linear := func(value uint8) float64 {
			v := float64(value) / 255
			if v <= .04045 {
				return v / 12.92
			}
			return math.Pow((v+.055)/1.055, 2.4)
		}
		return .2126*linear(color.R) + .7152*linear(color.G) + .0722*linear(color.B)
	}
	x, y := luminance(a), luminance(b)
	return (max(x, y) + .05) / (min(x, y) + .05)
}

func TestThemeTextContrast(t *testing.T) {
	for _, key := range themeKeys {
		for _, dark := range []bool{false, true} {
			theme := momentoTheme(key, dark)
			for name, color := range map[string]ui.Color{"text": theme.Text, "muted": theme.TextMuted} {
				for surface, background := range map[string]ui.Color{"background": theme.Background, "surface": theme.Surface, "hover": theme.SurfaceHover, "pressed": theme.SurfacePressed} {
					if contrast := themeContrast(color, background); contrast < 4.5 {
						t.Errorf("%s/%t %s 在 %s 上的对比度 %.2f < 4.5", key, dark, name, surface, contrast)
					}
				}
			}
			for name, background := range map[string]ui.Color{"accent": theme.Accent, "accent-hover": theme.AccentHover, "accent-pressed": theme.AccentPressed, "inactive": theme.InactiveAccent} {
				if contrast := themeContrast(theme.AccentText, background); contrast < 4.5 {
					t.Errorf("%s %s 文字对比度 %.2f < 4.5", key, name, contrast)
				}
			}
			if contrast := themeContrast(theme.InverseText, theme.Inverse); contrast < 4.5 {
				t.Errorf("%s 提示条对比度 %.2f < 4.5", key, contrast)
			}
			for name, color := range map[string]ui.Color{"border": theme.Border, "danger": theme.Danger, "warning": theme.Warning, "success": theme.Success, "selection": theme.Selection, "focus": theme.Focus, "inverse": theme.Inverse, "inverse-text": theme.InverseText} {
				if color.A == 0 {
					t.Errorf("%s 未定义 %s", key, name)
				}
			}
		}
	}
}

func TestThemeFollowsSystemAndNativeAppearance(t *testing.T) {
	for _, tc := range []struct {
		key    string
		source mygo.ThemeSource
	}{{"system", mygo.ThemeSystem}, {"light", mygo.ThemeLight}, {"light-gray", mygo.ThemeLight}, {"dark-gray", mygo.ThemeDark}, {"dark", mygo.ThemeDark}, {"navy", mygo.ThemeDark}, {"purple", mygo.ThemeDark}} {
		if themeSource(tc.key) != tc.source {
			t.Errorf("%s 原生外观不匹配", tc.key)
		}
	}
	if momentoTheme("system", false).Dark || !momentoTheme("system", true).Dark {
		t.Fatal("系统明暗没有跟随")
	}
	a := &App{Preferences: DefaultPreferences()}
	var theme *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); theme = c.Theme() }, 100, 100)
	if theme.Dark {
		t.Fatal("离线浅色系统变成深色")
	}
	a.Preferences.Theme = "purple"
	tt.Frame()
	if !theme.Dark || theme.Background != momentoTheme("purple", false).Background {
		t.Fatal("深紫没有即时生效")
	}
	a.Preferences.Theme = "system"
	tt.SetDark(false)
	tt.Frame()
	if theme.Dark {
		t.Fatal("返回系统后残留深色主题")
	}
	tt.SetDark(true)
	tt.Frame()
	if !theme.Dark {
		t.Fatal("系统外观改变后主题没有跟随")
	}
}

func TestThemeSwatchesSelectAndExposeNames(t *testing.T) {
	for _, lang := range []string{"zh-CN", "en", "ja"} {
		a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: lang}}
		a.Preferences.Language = lang
		tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.ThemeSwatches(c) }, 300, 100)
		for _, key := range themeKeys {
			name := a.T(themeNameKey(key))
			if strings.HasPrefix(name, "⟦") {
				t.Fatalf("%s/%s 缺少翻译", lang, key)
			}
			if control, ok := tt.AccessByRole(ui.RoleButton, name); !ok || control.Disabled {
				t.Fatalf("%s/%s 缺少可读名称", lang, key)
			}
			if err := tt.Click(name); err != nil {
				t.Fatal(err)
			}
			if a.Preferences.Theme != key || !tt.HasText(name) {
				t.Fatalf("%s/%s 没有即时选中", lang, key)
			}
		}
	}
}

func TestZoomUsesLogicalMinimumAndScreenWorkArea(t *testing.T) {
	for _, tc := range []struct {
		zoom          float32
		area          mygo.Rectangle
		width, height int
	}{
		{.8, mygo.Rectangle{}, 880, 512}, {1, mygo.Rectangle{}, 1100, 640},
		{1.5, mygo.Rectangle{}, 1650, 960}, {1.5, mygo.Rectangle{Width: 1440, Height: 850}, 1440, 850},
	} {
		width, height := zoomedWindowSize(1100, 640, tc.zoom, tc.area)
		if width != tc.width || height != tc.height {
			t.Fatalf("%v 缩放：得到 %dx%d，预期 %dx%d", tc.zoom, width, height, tc.width, tc.height)
		}
	}
	a := &App{Preferences: DefaultPreferences()}
	a.Preferences.UIZoom = 1.5
	var zoom float32
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); zoom = c.Zoom() }, 300, 200)
	if zoom != 1.5 {
		t.Fatal("主窗口没有应用缩放")
	}
	a.Preferences.UIZoom = .8
	tt.Frame()
	if zoom != .8 {
		t.Fatal("缩放修改没有即时生效")
	}
}

func TestThemeSwatchesShowHueBorderAndSelection(t *testing.T) {
	for _, background := range []string{"light", "purple"} {
		a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
		a.Preferences.Language, a.Preferences.Theme = "en", background
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Column(c).Fill().Padding(20).Children(func() { a.ThemeSwatches(c) })
		}, 320, 96)
		tt.SetScale(4)
		tt.SetWindowActive(true)
		theme := momentoTheme(background, false)
		pixel := func(x, y float32, want ui.Color, what string) {
			t.Helper()
			got := tt.Image().RGBAAt(int(x*4), int(y*4))
			if got.R != want.R || got.G != want.G || got.B != want.B {
				t.Fatalf("%s/%s 颜色不符：得到%v，预期%v", background, what, got, want)
			}
		}
		for _, key := range themeKeys {
			node, ok := tt.AccessByRole(ui.RoleButton, a.T(themeNameKey(key)))
			if !ok {
				t.Fatalf("缺少 %s 色块", key)
			}
			r := node.Bounds
			pixel(r.X+r.W-.5, r.Y+r.H/2, theme.Border, key+"边框")
			if key == background {
				pixel(r.X+r.W/2, r.Y-2, theme.Accent, key+"选中外圈")
			}
			if key == "navy" || key == "purple" {
				fill := themeSwatchColor(key, false)
				pixel(r.X+r.W/2, r.Y+r.H/2, fill, key+"代表色")
				if fill == momentoTheme(key, false).Background {
					t.Fatalf("%s 色块仍与深色背景难以区分", key)
				}
				if key == "navy" && (int(fill.B) < int(fill.R)+60 || fill.B < fill.G) {
					t.Fatal("深蓝色块没有足够蓝色辨识度")
				}
				if key == "purple" && (int(fill.R) < int(fill.G)+35 || int(fill.B) < int(fill.G)+60) {
					t.Fatal("深紫色块没有足够紫色辨识度")
				}
			}
		}
	}
}
