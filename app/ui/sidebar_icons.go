package ui

import "github.com/egoist/mygo/ui"

func sidebarIcon(name string) *ui.SVG {
	return phosphorRegular[phosphorName(name)]
}

var browseRandomIcon = phosphorRegular["shuffle"]
var browseLinkIcon = phosphorRegular["link-simple"]
