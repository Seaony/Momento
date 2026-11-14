package ui

import "github.com/egoist/mygo/ui"

var settingsIcons = map[string]*ui.SVG{
	"general":       phosphorRegular["gear-six"],
	"appearance":    phosphorRegular["palette"],
	"sidebar":       phosphorRegular["sidebar-simple"],
	"controls":      phosphorRegular["mouse-simple"],
	"preview":       phosphorRegular["images"],
	"import":        phosphorRegular["download-simple"],
	"shortcuts":     phosphorRegular["keyboard"],
	"notifications": phosphorRegular["bell"],
	"password":      phosphorRegular["lock-simple"],
	"browser":       phosphorRegular["globe"],
	"about":         phosphorRegular["info"],
}
