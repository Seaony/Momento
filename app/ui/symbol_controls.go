package ui

import "github.com/egoist/mygo/ui"

func symbolColor(theme *ui.Theme, hovered bool) ui.Color {
	if hovered {
		return theme.Text
	}
	return theme.TextMuted
}
