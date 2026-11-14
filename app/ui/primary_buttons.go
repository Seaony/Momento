package ui

import "github.com/egoist/mygo/ui"

func primaryButton(c *ui.Context, label string, disabled bool) ui.Element {
	return primaryButtonState(c, ui.PrimaryButton(c, label), disabled)
}

// 禁用时保留文字对比，不再把整个橙色按钮混成半透明。
func primaryButtonState(c *ui.Context, button ui.Element, disabled bool) ui.Element {
	button.Disabled(disabled).DimDisabled(false).Padding(space4, space12).Radius(radiusControl).FontSize(fontBody)
	theme := c.Theme()
	background, text := theme.Accent, theme.AccentText
	if disabled {
		background, text = theme.Surface, theme.TextMuted
		button.Border(1, theme.Border)
	} else if button.Pressed() {
		background = theme.AccentPressed
	} else if button.Hovered() {
		background = theme.AccentHover
	}
	animateColors(button, background, text)
	return button
}

func secondaryButton(c *ui.Context, label string) ui.Element {
	button := ui.Button(c, label).Padding(space4, space12).Radius(radiusControl).FontSize(fontBody)
	theme := c.Theme()
	background := theme.Surface
	if button.Pressed() {
		background = theme.SurfacePressed
	} else if button.Hovered() {
		background = theme.SurfaceHover
	}
	animateColors(button, background, theme.Text)
	return button
}
