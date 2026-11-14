package ui

import "github.com/egoist/mygo/ui"

var momentoSymbols = map[string]*ui.SVG{
	"chevron-left":     iconRegular["caret-left"],
	"chevron-right":    iconRegular["caret-right"],
	"chevron-down":     iconRegular["caret-down"],
	"chevron-up":       iconRegular["caret-up"],
	"chevrons-up-down": iconRegular["caret-up-down"],
	"plus":             iconRegular["plus"],
	"minus":            iconRegular["minus"],
	"close":            iconRegular["x"],
	"check":            iconRegular["check"],
	"rotate-left":      iconRegular["arrow-counter-clockwise"],
	"rotate-right":     iconRegular["arrow-clockwise"],
}

func appSymbol(c *ui.Context, name string) ui.Element {
	return svgIcon(c, iconName(name), false).Size(14, 14)
}
