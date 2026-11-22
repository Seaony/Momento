package ui

import (
	"unicode"

	"github.com/egoist/mygo/ui"
)

// choiceMenu 是「几选一」：按钮上写当前选项，点开是打勾的菜单。设置、重命名方式、智能文件夹的匹配方式都用它。
func choiceMenu(c *ui.Context, label string, selected int, labels []string, choose func(int)) ui.Element {
	// 读屏软件不读按钮里的文字，当前选项要单独作为值报出来。
	trigger := settingsMenuButton(c, label).Value(labels[selected])
	trigger.Children(func() {
		ui.Text(c, labels[selected]).FontSize(fontBody).SingleLine()
		symbol := appSymbol(c, "chevrons-up-down").Size(iconSmall, iconSmall).Shrink(0)
		symbol.TextColor(symbol.AnimateColor("symbol-color", symbolColor(c.Theme(), trigger.Hovered()), motionColor))
	}).Menu(func(m *ui.Menu) {
		for i, text := range labels {
			if m.Item(text).Radio(i == selected).Chosen() {
				choose(i)
			}
		}
	})
	return trigger
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
	field, changed, _ = searchFieldFocus(c, query, label, clearLabel, false)
	return field, changed
}

// autofocus 为真时，搜索框出现的那一帧就把光标放进输入框（弹出的选择器一打开就能打字）。
// submitted 报告这一帧在输入框里按了回车。
func searchFieldFocus(c *ui.Context, query *string, label, clearLabel string, autofocus bool) (field ui.Element, changed, submitted bool) {
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
		if autofocus {
			input.AutoFocus()
		}
		changed, submitted = input.Changed(), input.Submitted()
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
	return fieldStyle(c, field), changed, submitted
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
