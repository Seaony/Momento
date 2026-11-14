package ui

import (
	"github.com/egoist/mygo/ui"
	"time"
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
type assetMotionID struct{ ID string }

func motionTarget(on bool) float32 {
	if on {
		return 1
	}
	return 0
}
