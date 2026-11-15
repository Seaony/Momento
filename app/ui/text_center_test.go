package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 角标、小计数这类固定高度的文字要上下居中：放大 4 倍量白字上下的留白，相差不能超过 1 点（4 像素）。
func TestCenteredTextHeightCentersGlyphs(t *testing.T) {
	for _, tc := range []struct {
		text         string
		size, height float32
		weight       int
	}{{"5640 × 2400", 10, assetBadgeHeight, 500}, {"12", sidebarBadgeSize, 11, 700}, {"快速访问", fontSecondary, sidebarHeader, 600}} {
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetZoom(4)
			c.Root().Background(ui.RGB(255, 0, 0))
			centeredTextHeight(ui.Text(c, tc.text).Absolute().Top(4).Left(4).Background(ui.RGB(0, 0, 0)).
				FontFeatures("tnum").FontSize(tc.size).FontWeight(tc.weight).TextColor(ui.RGB(255, 255, 255)), tc.height, 3)
		}, 400, 200)
		tt.Frame()
		img := tt.Image()
		top, bottom, textTop, textBottom := -1, -1, -1, -1
		for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
			for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				if r < 0x2000 && g < 0x2000 && b < 0x2000 {
					if top < 0 {
						top = y
					}
					bottom = y
				}
				if r > 0xc000 && g > 0xc000 && b > 0xc000 {
					if textTop < 0 {
						textTop = y
					}
					textBottom = y
				}
			}
		}
		above, below := textTop-top, bottom-textBottom
		if top < 0 || textTop < 0 || above-below > 4 || below-above > 4 {
			t.Fatalf("%q 没有上下居中：上留白 %d，下留白 %d（4 倍像素）", tc.text, above, below)
		}
	}
}
