package ui

import "github.com/egoist/mygo/ui"

func (a *App) assetGrid(c *ui.Context) {
	width := a.grid.Bounds(c).W - 32
	if width <= 0 {
		width, _ = c.Size()
		width -= 32
		if a.SidebarVisible {
			width -= a.SidebarWidth + 1
		}
		if a.InspectorVisible {
			width -= 281
		}
	}
	scale := a.thumbnailScale()
	columns := max(1, int((width+8)/(160*scale+8)))
	rows := (len(a.Visible) + columns - 1) / columns
	ui.List(c.Key("grid"), &a.grid, rows, func(row int) {
		ui.Row(c).FillWidth().Height(190 * scale).Gap(8).AlignItems(ui.Stretch).Children(func() {
			for column := range columns {
				index := row*columns + column
				cell := ui.Column(c).Grow(1).Basis(0).MinWidth(0)
				if index < len(a.Visible) {
					cell.Children(func() { a.assetCard(c, a.Visible[index], index, false) })
				}
			}
		})
	}).Grow(1).Gap(8).Padding(16)
}
