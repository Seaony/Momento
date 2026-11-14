package ui

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestDisabledPrimaryKeepsContrastAndBlocksInput(t *testing.T) {
	for _, key := range []string{"light", "dark"} {
		t.Run(key, func(t *testing.T) {
			hit := false
			theme := momentoTheme(key, false)
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(theme)
				ui.Column(c).Fill().Padding(16).Children(func() {
					if primaryButton(c, "Unlock", true).Clicked() {
						hit = true
					}
				})
			}, 220, 100)
			tt.SetWindowActive(true)
			tt.Frame()
			button, ok := tt.AccessByRole(ui.RoleButton, "Unlock")
			if !ok || !button.Disabled {
				t.Fatal("禁用主按钮没有保留禁用语义")
			}
			r := button.Bounds
			pixel := tt.Image().RGBAAt(int(r.X+5), int(r.Y+r.H/2))
			if pixel.R != theme.Surface.R || pixel.G != theme.Surface.G || pixel.B != theme.Surface.B {
				t.Fatalf("禁用底色仍被染色或变透明：%v", pixel)
			}
			tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
			tt.Key(0, ui.KeyEnter)
			if hit {
				t.Fatal("禁用主按钮仍能触发操作")
			}
			if ratio := themeContrast(theme.TextMuted, theme.Surface); ratio < 4.5 {
				t.Fatalf("禁用文字对比度 %.2f", ratio)
			}
		})
	}
}
