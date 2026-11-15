package ui

import (
	"unicode"

	"github.com/egoist/mygo/ui"
)

func styleSegment(c *ui.Context, item ui.Element, selected bool) {
	theme := c.Theme()
	palette := paletteFor(theme)
	background, text := ui.Transparent, theme.TextMuted
	if selected {
		text = titleColor(theme)
	} else if item.Pressed() {
		background, text = palette.Pressed, theme.Text
	} else if item.Hovered() {
		background, text = palette.Hover, theme.Text
	}
	animateColors(item, background, text)
}

// 底块画在整组文字后面，移动途中也不会盖住相邻选项。
func slidingSegmentTrack(c *ui.Context, track ui.Element, items []ui.Element, selected int, padding, gap float32) {
	theme := c.Theme()
	palette := paletteFor(theme)
	background := palette.SegmentSelected
	hover, pressed := palette.SegmentSelected.Mix(palette.Text, .04), palette.SegmentSelected.Mix(palette.Text, .08)
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
		p.Shadow(box, radiusSmall, 0, 1, 2, 0, palette.Shadow)
		p.Fill(box, background, radiusSmall)
		p.Stroke(box, palette.BorderSubtle, radiusSmall, 1)
	})
}

func settingsCheckbox(c *ui.Context, checked *bool, label string) ui.Element {
	theme := c.Theme()
	palette := paletteFor(theme)
	row := ui.CheckboxBase(c, checked).Label(label).Gap(space8).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Children(func() {
		background, border := palette.Raised, palette.Border
		if *checked {
			background, border = theme.Accent, theme.Accent
		} else if row.Hovered() {
			border = theme.Focus
		}
		box := ui.Box(c.Key("checkbox-face")).Size(checkSize, checkSize).Radius(radiusSmall).Shrink(0).Center()
		box.Border(1, box.AnimateColor("border", border, motionColor))
		animateColors(box, background, theme.AccentText)
		box.Children(func() {
			mark := appSymbol(c, "check").Size(iconSmall, iconSmall)
			fade := motionColor
			fade.NonMotion = true
			mark.Opacity(mark.AnimateTo("checked", motionTarget(*checked), fade))
		})
		ui.Text(c, label).FontSize(fontBody)
	})
	return row
}

func animatedRadio[T comparable](c *ui.Context, selected *T, value T, label string) ui.Element {
	theme := c.Theme()
	palette := paletteFor(theme)
	row := ui.RadioBase(c, selected, value).Label(label).Gap(space8).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row.Children(func() {
		on := *selected == value
		background, border := palette.Raised, palette.Border
		if on {
			background, border = theme.Accent, theme.Accent
		} else if row.Hovered() {
			border = theme.Focus
		}
		face := ui.Box(c.Key("radio-face")).Size(checkSize, checkSize).Radius(checkSize / 2).Center().Shrink(0)
		face.Border(1, face.AnimateColor("border", border, motionColor))
		animateColors(face, background, theme.Text)
		face.Children(func() {
			dot := ui.Box(c).Size(6, 6).Radius(3).Background(theme.AccentText)
			fade := motionColor
			fade.NonMotion = true
			dot.Opacity(dot.AnimateTo("checked", motionTarget(on), fade))
		})
		ui.Text(c, label).FontSize(fontBody)
	})
	return row
}

func animatedToggle(c *ui.Context, on *bool, label string) ui.Element {
	theme := c.Theme()
	palette := paletteFor(theme)
	button := ui.ToggleBase(c, on).Label(label).Padding(space8, space12).Gap(space8).Radius(theme.Radius).Border(1, theme.Border)
	background := palette.Raised
	if *on || button.Pressed() {
		background = palette.Pressed
	} else if button.Hovered() {
		background = palette.Hover
	}
	animateColors(button, background, theme.Text)
	return button.Children(func() { ui.Text(c, label).SingleLine() })
}

// 表单控件统一外观：浅底 + 浅边框；悬停底色加深一点，聚焦变浮起面并换焦点色边框。
// 输入框底下多留一点，这一行居中时把字抬上去，让字形中心对准框中心。
// 真窗口 2 倍截图实测（离线测试器量不出来）：中文、日文比框中心低约 1.5 点，英文大写约低 0.25 点，所以按显示的文字分开抬。
func searchTextLift(text string) float32 {
	for _, r := range text {
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana) {
			return 3.25
		}
	}
	return 1
}

// 应用里的搜索框：放大镜、输入框、有内容时的清除按钮。不用 ui.SearchField，因为它的占位颜色改不了、中文也偏下；
// 输入框撑满整行，点框里任意处都能输入；有内容时按 Esc 先清空，和系统搜索框一样。
func searchField(c *ui.Context, query *string, label, clearLabel string) (field ui.Element, changed bool) {
	p := paletteFor(c.Theme())
	shown := *query
	if shown == "" {
		shown = label
	}
	field = ui.Row(c).Height(controlHeight).Gap(space8).Padding(0, space8).Cursor(ui.CursorText)
	field.Children(func() {
		appIcon(c, "search").Size(iconMedium, iconMedium).TextColor(p.TextSecondary).Shrink(0)
		input := ui.TextInputBase(c.Key("input"), query).Search().Label(label).Placeholder(label).PlaceholderColor(placeholderColor(c)).
			Grow(1).MinWidth(0).FillHeight().Padding(0, 0, searchTextLift(shown), 0).FontSize(fontBody).FocusRing(false)
		changed = input.Changed()
		if *query != "" {
			// 挂在输入框上：弹出层里也是先清空，不会直接关掉弹出层。
			if input.Shortcut(0, ui.KeyEscape) {
				*query, changed = "", true
			}
			// 点清除会把焦点带到按钮上，清完还给输入框，方便接着输入。
			if toolbarIconButton(c.Key("clear"), clearLabel, "close").Size(space20, space24).Clicked() {
				*query, changed = "", true
				input.Focus()
			}
		}
	})
	return fieldStyle(c, field), changed
}

// 占位文字比三级文字再浅：一眼看出是提示，不像已经输入的内容。
func placeholderColor(c *ui.Context) ui.Color { return paletteFor(c.Theme()).TextTertiary.Alpha(.6) }

func fieldStyle(c *ui.Context, field ui.Element) ui.Element {
	p := paletteFor(c.Theme())
	background, border := p.Field, p.BorderSubtle
	if field.FocusWithin() {
		background, border = p.Raised, p.Focus
	} else if field.Hovered() {
		background = p.FieldHover
	}
	field.Radius(radiusControl).FocusRing(false)
	field.Background(field.AnimateColor("field-background", background, motionColor))
	return field.Border(1, field.AnimateColor("field-border", border, motionColor))
}

// 菜单按钮（下拉选择）和输入框同一套浅底浅边框。
func menuFieldStyle(c *ui.Context, button ui.Element) ui.Element {
	p := paletteFor(c.Theme())
	background := p.Field
	if button.Pressed() {
		background = p.Pressed
	} else if button.Hovered() {
		background = p.FieldHover
	}
	button.Radius(radiusControl).Border(1, p.BorderSubtle)
	animateColors(button, background, p.Text)
	return button
}
