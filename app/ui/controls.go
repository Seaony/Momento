package ui

import "github.com/egoist/mygo/ui"

func styleSegment(c *ui.Context, item ui.Element, selected bool) {
	theme := c.Theme()
	background, text := ui.Transparent, theme.TextMuted
	if selected {
		text = titleColor(theme)
	} else if item.Pressed() {
		background, text = warmOverlay(theme, .11), theme.Text
	} else if item.Hovered() {
		background, text = warmOverlay(theme, .08), theme.Text
	}
	animateColors(item, background, text)
}

// 底块画在整组文字后面，移动途中也不会盖住相邻选项。
func slidingSegmentTrack(c *ui.Context, track ui.Element, items []ui.Element, selected int, padding, gap float32) {
	theme := c.Theme()
	background, hover, pressed := theme.Background, theme.SurfaceHover, theme.SurfacePressed
	if theme.Dark {
		background, hover, pressed = theme.SurfaceHover, theme.SurfacePressed, theme.Surface
	}
	if items[selected].Pressed() {
		background = pressed
	} else if items[selected].Hovered() {
		background = hover
	}
	background = track.AnimateColor("indicator-color", background, motionColor)
	position := track.AnimateTo("indicator-position", float32(selected), motionSpring)
	track.Draw(func(p *ui.Painter, r ui.Rect) {
		inset := padding + 1
		width := (r.W - inset*2 - gap*float32(len(items)-1)) / float32(len(items))
		box := ui.Rect{X: r.X + inset + position*(width+gap), Y: r.Y + inset, W: width, H: r.H - inset*2}
		p.Shadow(box, 4, 0, 1, 2, 0, ui.Hex("#211A16").Alpha(.14))
		p.Fill(box, background, 4)
		p.Stroke(box, theme.Border, 4, 1)
	})
}

func settingsCheckbox(c *ui.Context, checked *bool, label string) ui.Element {
	theme := c.Theme()
	row := ui.CheckboxBase(c, checked).Label(label).Gap(8).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Children(func() {
		background, border := theme.Background, theme.Border.Mix(theme.Text, .25)
		if *checked {
			background, border = theme.Accent, theme.Accent
		} else if row.Hovered() {
			border = theme.Focus
		}
		box := ui.Box(c.Key("checkbox-face")).Size(16, 16).Radius(4).Shrink(0).Center()
		box.Border(1, box.AnimateColor("border", border, motionColor))
		animateColors(box, background, theme.AccentText)
		box.Children(func() {
			mark := appSymbol(c, "check").Size(14, 14)
			fade := motionColor
			fade.NonMotion = true
			mark.Opacity(mark.AnimateTo("checked", motionTarget(*checked), fade))
		})
		ui.Text(c, label).FontSize(13)
	})
	return row
}

func animatedRadio[T comparable](c *ui.Context, selected *T, value T, label string) ui.Element {
	theme := c.Theme()
	row := ui.RadioBase(c, selected, value).Label(label).Gap(8).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Children(func() {
		on := *selected == value
		background, border := theme.Background, theme.Border.Mix(theme.Text, .25)
		if on {
			background, border = theme.Accent, theme.Accent
		} else if row.Hovered() {
			border = theme.Focus
		}
		face := ui.Box(c.Key("radio-face")).Size(16, 16).Radius(8).Center().Shrink(0)
		face.Border(1, face.AnimateColor("border", border, motionColor))
		animateColors(face, background, theme.Text)
		face.Children(func() {
			dot := ui.Box(c).Size(6, 6).Radius(3).Background(theme.AccentText)
			fade := motionColor
			fade.NonMotion = true
			dot.Opacity(dot.AnimateTo("checked", motionTarget(on), fade))
		})
		ui.Text(c, label).FontSize(13)
	})
	return row
}

func animatedToggle(c *ui.Context, on *bool, label string) ui.Element {
	theme := c.Theme()
	button := ui.ToggleBase(c, on).Label(label).Padding(6, 14).Gap(6).Radius(theme.Radius).Border(1, theme.Border)
	background := theme.Surface
	if *on || button.Pressed() {
		background = theme.SurfacePressed
	} else if button.Hovered() {
		background = theme.SurfaceHover
	}
	animateColors(button, background, theme.Text)
	return button.Children(func() { ui.Text(c, label).SingleLine() })
}
