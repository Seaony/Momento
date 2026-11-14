package ui

import "github.com/egoist/mygo/ui"

var settingsIcons = map[string]*ui.SVG{
	"general":       momentoIcons["settings"],
	"appearance":    momentoIcons["appearance"],
	"sidebar":       momentoIcons["sidebar"],
	"controls":      momentoIcons["controls"],
	"preview":       momentoIcons["photos"],
	"import":        momentoIcons["import"],
	"shortcuts":     momentoIcons["keyboard"],
	"notifications": momentoIcons["notifications"],
	"password":      momentoIcons["lock"],
	"browser":       momentoIcons["browser"],
	"about":         momentoIcons["info"],
}
