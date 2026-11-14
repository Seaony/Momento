package ui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

var motionSpring = ui.Animation{Spring: ui.SpringOptions{Duration: 250 * time.Millisecond, Bounce: .04}}
var motionColor = ui.Animation{Duration: 120 * time.Millisecond}

func elevated(e ui.Element, t *ui.Theme) ui.Element {
	p := paletteFor(t)
	return e.Border(1, p.BorderSubtle).Shadow(0, 1, 2, 0, p.Shadow)
}

type paneWidthMotion struct {
	target    float32
	changedAt time.Time
}

func animatePaneWidth(c *ui.Context, e ui.Element, key string, target float32) float32 {
	state := ui.Local(e, key+"-target", func() paneWidthMotion { return paneWidthMotion{target: target} })
	if state.target != target {
		state.target, state.changedAt = target, c.Now()
	}
	// 弹簧的感知时长不等于结束时间，栏宽在约定终点收准，避免继续重排。
	if state.changedAt.IsZero() || c.Now().Sub(state.changedAt) >= 250*time.Millisecond {
		return e.Jump(key, target)
	}
	value := e.AnimateTo(key, target, motionSpring)
	if !e.Settled(key) {
		c.After(250*time.Millisecond - c.Now().Sub(state.changedAt))
	}
	return value
}

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

func motionTarget(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

func (a *App) noticeViewport(c *ui.Context) {
	width, _ := c.Size()
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(space24).AlignItems(ui.Center).Gap(space8)
		for _, notice := range toasts {
			parts := ui.ToastBase(c, notice)
			parts.Root.Row().AlignItems(ui.Center).Gap(space12).Padding(space12, space16).Radius(radiusPanel).
				MaxWidth(width-48).Material(panelMaterial(c.Theme())).TextColor(c.Theme().Text).
				Border(1, paletteFor(c.Theme()).BorderSubtle).Shadow(0, 1, 2, 0, paletteFor(c.Theme()).Shadow).
				Transition(ui.ElementTransition{Animation: &motionSpring, Position: true, Enter: &ui.Motion{Y: 12}, Exit: &ui.Motion{Y: 12}})
			parts.Root.Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(space4).Children(func() {
					ui.Text(c, notice.Title)
					if notice.Description != "" {
						ui.Text(c, notice.Description).TextColor(c.Theme().TextMuted)
					}
				})
				if notice.Action != "" {
					parts.ActionButton().Padding(space4, space8).Children(func() { ui.Text(c, notice.Action) })
				}
				close := parts.CloseButton().Label(a.T("close")).Tooltip(a.T("close")).Size(space24, space24).Radius(radiusCard).Center()
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
