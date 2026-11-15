package ui

import "github.com/egoist/mygo/ui"

// 弹出层、菜单、对话框用实色浮起底。
func panelColor(t *ui.Theme) ui.Color { return paletteFor(t).Raised }

// 工具栏和侧栏同色，连成一块；盖在滚动内容上，所以是实色。
func toolbarColor(t *ui.Theme) ui.Color { return paletteFor(t).Sidebar }

func floatingPopover(c *ui.Context, anchor ui.Element, open *bool, content func()) ui.Element {
	panel := ui.Popover(c, anchor, open, content)
	if panel.Valid() {
		elevated(popupMotion(panel), c.Theme()).Background(panelColor(c.Theme())).Radius(radiusPanel)
	}
	return panel
}
