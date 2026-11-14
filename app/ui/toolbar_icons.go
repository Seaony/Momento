package ui

import "github.com/egoist/mygo/ui"

var appIconAliases = map[string]string{
	"照片":                 "images",
	"喜欢":                 "heart",
	"随机浏览":               "shuffle",
	"文件夹":                "folder",
	"标签组":                "tag",
	"标签":                 "tag-simple",
	"回收站":                "trash",
	"设置":                 "gear-six",
	"最近使用":               "clock-counter-clockwise",
	"未分类":                "tray",
	"未打标签":               "tag-simple",
	"智能文件夹":              "folder-simple-star",
	"photos":             "images",
	"preview":            "images",
	"recent":             "clock-counter-clockwise",
	"favorite":           "heart",
	"random":             "shuffle",
	"unfiled":            "tray",
	"untagged":           "tag-simple",
	"tags":               "tag",
	"smart-folder":       "folder-simple-star",
	"settings":           "gear-six",
	"general":            "gear-six",
	"help":               "question",
	"import":             "download-simple",
	"export":             "upload-simple",
	"filter":             "funnel-simple",
	"sort":               "sort-ascending",
	"masonry":            "squares-four",
	"grid":               "grid-four",
	"search":             "magnifying-glass",
	"thumbnail-size":     "frame-corners",
	"thumbnail":          "frame-corners",
	"sidebar":            "sidebar-simple",
	"inspector":          "sidebar-simple",
	"collapse-inspector": "caret-double-right",
	"link":               "link-simple",
	"external-link":      "arrow-square-out",
	"library":            "images-square",
	"create-library":     "folder-plus",
	"lock":               "lock-simple",
	"password":           "lock-simple",
	"controls":           "mouse-simple",
	"shortcuts":          "keyboard",
	"notifications":      "bell",
	"appearance":         "palette",
	"browser":            "globe",
	"about":              "info",
	"eye-off":            "eye-slash",
	"edit":               "pencil-simple",
	"extension":          "puzzle-piece",
	"menu":               "list",
	"tag":                "tag-simple",
	"chevron-left":       "caret-left",
	"chevron-right":      "caret-right",
	"chevron-down":       "caret-down",
	"chevron-up":         "caret-up",
	"chevrons-up-down":   "caret-up-down",
	"chevrons":           "caret-up-down",
	"arrow-left":         "caret-left",
	"arrow-right":        "caret-right",
	"add":                "plus",
	"close":              "x",
	"rotate-left":        "arrow-counter-clockwise",
	"rotate-right":       "arrow-clockwise",
	"restore":            "arrow-counter-clockwise",
}

func phosphorName(name string) string {
	if canonical, ok := appIconAliases[name]; ok {
		return canonical
	}
	return name
}

func appIcon(c *ui.Context, name string) ui.Element {
	return phosphorIcon(c, phosphorName(name), false)
}

func appIconSelected(c *ui.Context, name string, selected bool) ui.Element {
	return phosphorIcon(c, phosphorName(name), selected)
}

var toolbarIcons = map[string]*ui.SVG{
	"thumbnail-size": phosphorRegular["frame-corners"],
	"filter":         phosphorRegular["funnel-simple"],
	"sidebar":        phosphorRegular["sidebar-simple"],
	"inspector":      phosphorRegular["sidebar-simple"],
	"import":         phosphorRegular["download-simple"],
	"sort":           phosphorRegular["sort-ascending"],
	"masonry":        phosphorRegular["squares-four"],
	"grid":           phosphorRegular["grid-four"],
	"list":           phosphorRegular["list"],
}
