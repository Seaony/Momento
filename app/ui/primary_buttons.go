package ui

import "github.com/egoist/mygo/ui"

func primaryButton(c *ui.Context, label string, disabled bool) ui.Element {
	return primaryButtonState(c, ui.PrimaryButton(c, label), disabled)
}

// 禁用时保留文字对比，不再把整个橙色按钮混成半透明。
func primaryButtonState(c *ui.Context, button ui.Element, disabled bool) ui.Element {
	button.Disabled(disabled).DimDisabled(false)
	if disabled {
		button.Background(c.Theme().Surface).TextColor(c.Theme().TextMuted).Border(1, c.Theme().Border)
	}
	return button
}
