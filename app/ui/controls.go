package ui

import "github.com/egoist/mygo/ui"

func styleSegment(c *ui.Context, item ui.Element, selected bool) {
	theme := c.Theme()
	item.TextColor(theme.Text)
	if selected {
		background, hover, pressed := theme.Background, theme.SurfaceHover, theme.SurfacePressed
		if theme.Dark {
			background, hover, pressed = theme.SurfaceHover, theme.SurfacePressed, theme.Surface
		}
		if item.Pressed() {
			background = pressed
		} else if item.Hovered() {
			background = hover
		}
		item.Background(background).Border(1, theme.Border).TextColor(titleColor(theme)).
			Shadow(0, 1, 2, 0, ui.Hex("#211A16").Alpha(.14))
	} else if item.Pressed() {
		item.Background(warmOverlay(theme, .11))
	} else if item.Hovered() {
		item.Background(warmOverlay(theme, .08))
	}
}
