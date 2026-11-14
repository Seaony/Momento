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
	return frostedSurface{tint: paletteFor(t).Raised.Alpha(materialPanelAlpha), blur: space24}
}

func toolbarMaterial(t *ui.Theme) ui.Material {
	return frostedSurface{tint: t.Background.Alpha(materialPanelAlpha), blur: space24}
}

func sidebarMaterial(c *ui.Context, e ui.Element) {
	p := paletteFor(c.Theme())
	if !c.Vibrancy() || c.Preferences().ReduceTransparency {
		e.Background(p.Sidebar)
		return
	}
	tint := p.Sidebar.Alpha(materialSidebarAlpha)
	if c.Theme().Dark {
		tint = p.Sidebar.Alpha(materialSidebarDark)
	}
	e.Background(tint)
}

func frostedPopover(c *ui.Context, anchor ui.Element, open *bool, content func()) ui.Element {
	panel := ui.Popover(c, anchor, open, content)
	if panel.Valid() {
		elevated(popupMotion(panel), c.Theme()).Material(panelMaterial(c.Theme())).Radius(radiusPanel)
	}
	return panel
}
