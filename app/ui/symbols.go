package ui

import "github.com/egoist/mygo/ui"

var momentoSymbols = map[string]*ui.SVG{
	"chevron-left":     phosphorRegular["caret-left"],
	"chevron-right":    phosphorRegular["caret-right"],
	"chevron-down":     phosphorRegular["caret-down"],
	"chevron-up":       phosphorRegular["caret-up"],
	"chevrons-up-down": phosphorRegular["caret-up-down"],
	"plus":             phosphorRegular["plus"],
	"minus":            phosphorRegular["minus"],
	"close":            phosphorRegular["x"],
	"check":            phosphorRegular["check"],
	"rotate-left":      phosphorRegular["arrow-counter-clockwise"],
	"rotate-right":     phosphorRegular["arrow-clockwise"],
}

func appSymbol(c *ui.Context, name string) ui.Element {
	return phosphorIcon(c, phosphorName(name), false).Size(14, 14)
}
