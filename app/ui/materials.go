package ui

import (
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// 磨砂层先模糊背后的内容，再铺暖色；文字不直接压在图片上。
type frostedSurface struct {
	tint ui.Color
	blur float32
}

func (m frostedSurface) PaintMaterial(p *ui.Painter, r ui.Rect, radii [4]float32) {
	if p.Preferences().ReduceTransparency {
		p.Fill(r, m.tint.Alpha(1), radii[0])
		return
	}
	glass.Blur{Radius: m.blur}.PaintMaterial(p, r, radii)
	p.Fill(r, m.tint, radii[0])
}

func panelMaterial(t *ui.Theme) ui.Material {
	return frostedSurface{tint: t.Background.Alpha(.94), blur: 22}
}

func toolbarMaterial(t *ui.Theme) ui.Material {
	return frostedSurface{tint: t.Background.Alpha(.86), blur: 22}
}

func sidebarMaterial(c *ui.Context, e ui.Element) {
	if !c.Vibrancy() || c.Preferences().ReduceTransparency {
		e.Background(c.Theme().Surface)
		return
	}
	tint := ui.RGBA(248, 230, 210, .5)
	if c.Theme().Dark {
		tint = ui.RGBA(44, 35, 30, .78)
	}
	e.Background(tint)
}

func frostedPopover(c *ui.Context, anchor ui.Element, open *bool, content func()) ui.Element {
	panel := ui.Popover(c, anchor, open, content)
	if panel.Valid() {
		panel.Material(panelMaterial(c.Theme())).Radius(12)
	}
	return panel
}
