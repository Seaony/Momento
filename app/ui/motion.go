package ui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

var motionSpring = ui.Animation{Spring: ui.SpringOptions{Duration: 250 * time.Millisecond, Bounce: .04}}
var motionColor = ui.Animation{Duration: 120 * time.Millisecond}

func animateColors(e ui.Element, background, text ui.Color) {
	e.Background(e.AnimateColor("surface-color", background, motionColor)).TextColor(e.AnimateColor("text-color", text, motionColor))
}

func popupMotion(e ui.Element) ui.Element {
	return e.AttachedOrigin().Transition(ui.ElementTransition{Animation: &motionSpring, Position: true,
		Enter: &ui.Motion{Scale: .96, Y: -4, Flip: true}, Exit: &ui.Motion{Scale: .96, Y: -4, Flip: true}})
}

func dialogMotion(backdrop, panel ui.Element) {
	backdrop.Transition(ui.ElementTransition{Animation: &motionSpring, Colors: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
	panel.Transition(ui.ElementTransition{Animation: &motionSpring, Position: true,
		Enter: &ui.Motion{Scale: .96, Y: 10}, Exit: &ui.Motion{Scale: .96, Y: 10}})
}

func fadeMotion(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Animation: &motionSpring, Colors: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
}

func layoutMotion(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Animation: &motionSpring, Position: true, Size: true,
		Enter: &ui.Motion{Scale: .98}, Exit: &ui.Motion{Scale: .98}})
}

// 同一素材的缩略图与预览共用身份，位置变化只影响屏幕内的元素。

func motionTarget(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

func (a *App) noticeViewport(c *ui.Context) {
	width, _ := c.Size()
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(24).AlignItems(ui.Center).Gap(8)
		for _, notice := range toasts {
			parts := ui.ToastBase(c, notice)
			parts.Root.Row().AlignItems(ui.Center).Gap(12).Padding(10, 14).Radius(12).
				MaxWidth(width-48).Material(panelMaterial(c.Theme())).TextColor(c.Theme().Text).
				Border(1, c.Theme().Border).Shadow(0, 4, 16, 0, ui.RGBA(40, 24, 16, .16)).
				Transition(ui.ElementTransition{Animation: &motionSpring, Position: true, Enter: &ui.Motion{Y: 12}, Exit: &ui.Motion{Y: 12}})
			parts.Root.Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, notice.Title)
					if notice.Description != "" {
						ui.Text(c, notice.Description).TextColor(c.Theme().TextMuted)
					}
				})
				if notice.Action != "" {
					parts.ActionButton().Padding(4, 8).Children(func() { ui.Text(c, notice.Action) })
				}
				close := parts.CloseButton().Label(a.T("close")).Tooltip(a.T("close")).Size(24, 24).Radius(6).Center()
				background, text := ui.Transparent, c.Theme().TextMuted
				if close.Hovered() {
					background, text = c.Theme().SurfaceHover, c.Theme().Text
				}
				animateColors(close, background, text)
				close.Children(func() { appSymbol(c, "x") })
			})
		}
	})
}
