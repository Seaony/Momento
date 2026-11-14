package ui

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

// 图片可能全黑或全白，磨砂上的文字在这两个边界仍需可读。
func TestFrostedTextContrastOverImages(t *testing.T) {
	for _, key := range []string{"light", "dark"} {
		theme := momentoTheme(key, false)
		for _, material := range []ui.Material{panelMaterial(theme), toolbarMaterial(theme)} {
			tint := material.(frostedSurface).tint
			for _, under := range []uint8{0, 255} {
				alpha := float64(tint.A) / 255
				blend := func(v uint8) uint8 { return uint8(float64(v)*alpha + float64(under)*(1-alpha) + .5) }
				background := ui.RGB(blend(tint.R), blend(tint.G), blend(tint.B))
				for _, foreground := range []ui.Color{theme.Text, theme.TextMuted} {
					if ratio := themeContrast(foreground, background); ratio < 4.5 {
						t.Errorf("%s 磨砂文字对比度 %.2f < 4.5", key, ratio)
					}
				}
			}
		}
	}
}
