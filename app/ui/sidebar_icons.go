package ui

import "github.com/egoist/mygo/ui"

func sidebarIcon(name string) *ui.SVG {
	return iconRegular[iconName(name)]
}

var browseRandomIcon = iconRegular["shuffle"]
var browseLinkIcon = iconRegular["link-simple"]

// 图库菜单用到的语义名，并进统一的图标别名表；以后整体换图标库只改映射。
var libraryMenuIconAliases = map[string]string{
	"drag-handle":    "dots-six-vertical",
	"more":           "dots-three",
	"reveal":         "folder-open",
	"open-library":   "folder-open",
	"import-library": "download-simple",
	"export-library": "upload-simple",
	"rename":         "pencil-simple",
	"reload":         "arrow-clockwise",
	"delete":         "trash",
	"remove":         "x",
}

func init() {
	for name, canonical := range libraryMenuIconAliases {
		appIconAliases[name] = canonical
	}
}
