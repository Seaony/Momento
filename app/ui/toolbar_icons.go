package ui

import "github.com/egoist/mygo/ui"

var toolbarIcons = map[string]*ui.SVG{
	"thumbnail-size": momentoIcons["thumbnail-size"],
	"filter":         momentoIcons["filter"],
	"sidebar":        momentoIcons["sidebar"],
	"inspector":      momentoIcons["inspector"],
	"import":         momentoIcons["import"],
	"sort":           momentoIcons["sort"],
	"masonry":        momentoIcons["masonry"],
	"grid":           momentoIcons["grid"],
	"list":           momentoIcons["list"],
}

var appIconAliases = map[string]string{
	"heart":         "favorite",
	"photos":        "photos",
	"preview":       "photos",
	"settings":      "settings",
	"general":       "settings",
	"appearance":    "appearance",
	"sidebar":       "sidebar",
	"controls":      "controls",
	"shortcuts":     "keyboard",
	"notifications": "notifications",
	"password":      "lock",
	"browser":       "browser",
	"about":         "info",
	"thumbnail":     "thumbnail-size",
	"slideshow":     "play",
	"menu":          "list",
}

func appIcon(c *ui.Context, name string) ui.Element {
	if canonical, ok := appIconAliases[name]; ok {
		name = canonical
	}
	return ui.Image(c, momentoIcons[name]).Size(20, 20)
}
