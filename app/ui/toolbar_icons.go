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
	"未分类":                "tray",
	"未打标签":               "tag-simple",
	"智能文件夹":              "folder-simple-star",
	"photos":             "images",
	"preview":            "images",
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
	"search":             "magnifying-glass",
	"thumbnail-size":     "frame-corners",
	"thumbnail":          "frame-corners",
	"sidebar":            "sidebar-simple",
	"inspector":          "sidebar-simple",
	"collapse-inspector": "sidebar-simple",
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
	"restore":            "arrow-counter-clockwise",
}

func iconName(name string) string {
	if canonical, ok := appIconAliases[name]; ok {
		return canonical
	}
	return name
}

func appIcon(c *ui.Context, name string) ui.Element {
	return svgIcon(c, iconName(name), false)
}

func appIconSelected(c *ui.Context, name string, selected bool) ui.Element {
	return svgIcon(c, iconName(name), selected)
}

// 工具栏图标用 Solar 线性版，线条按设计稿加粗到 2（和侧栏同一套加粗规则）；检查器开关和侧栏开关同一个图。
var toolbarIconSource = map[string]string{
	// 筛选里的文件类型：Solar 没有按格式区分的图标，按种类取 Camera Minimalistic（照片）、Gallery Minimalistic（图像）、Play Circle（动图）。
	"type-photo":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><circle cx="12" cy="13" r="3"/><path d="M14.0001 20H10.0001C7.19117 20 5.78671 20 4.77782 19.3259C4.34106 19.034 3.96605 18.659 3.67422 18.2223C3.0001 17.2134 3.0001 15.8089 3.0001 13L3.0001 12.9981C3.0001 10.1905 3.0001 8.78661 3.67407 7.77794C3.96591 7.34118 4.34091 6.96618 4.77767 6.67434C5.78656 6.00022 7.19103 6.00022 9.99995 6.00022H14C16.8089 6.00022 18.2133 6.00022 19.2222 6.67434C19.659 6.96618 20.034 7.34118 20.3258 7.77794C21 8.78683 21.0001 10.1911 21.0001 13C21.0001 15.8089 21.0001 17.2134 20.326 18.2223C20.0341 18.659 19.6591 19.034 19.2224 19.3259C18.2135 20 16.809 20 14.0001 20Z"/><path d="M18 10H17.5"/><path d="M15 3H9"/></g></svg>`,
	"type-image":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><path d="M12 22C7.28595 22 4.92893 22 3.46447 20.5355C2 19.0711 2 16.714 2 12C2 7.28595 2 4.92893 3.46447 3.46447C4.92893 2 7.28595 2 12 2C16.714 2 19.0711 2 20.5355 3.46447C22 4.92893 22 7.28595 22 12C22 16.714 22 19.0711 20.5355 20.5355C19.0711 22 16.714 22 12 22Z"/><circle cx="16" cy="8" r="2"/><path d="M5 13.307L5.81051 12.5542C6.73658 11.6941 8.18321 11.7424 9.04988 12.6623L11.6974 15.4727C12.2356 16.0439 13.1166 16.1209 13.7457 15.6516C14.6522 14.9753 15.9144 15.0522 16.7322 15.8334L19 18"/></g></svg>`,
	"type-motion": `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M9 9.21316L9 14.7868C9 15.7105 9.93371 16.2944 10.6935 15.8458L15.4137 13.059C16.1954 12.5974 16.1954 11.4026 15.4137 10.941L10.6935 8.15419C9.93371 7.70561 9 8.28947 9 9.21316Z"/></g></svg>`,
	// 排序方向：把 Solar Sort Vertical 拆成右边朝上、左边朝下两支箭头，分别上色表示升序或降序。
	"sort-up":        `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M16 18L16 6M12 10.125L16 6L20 10.125" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"sort-down":      `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M8 6L8 18M4 13.875L8 18L12 13.875" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"thumbnail-size": `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M2 12C2 7.28595 2 4.92893 3.46447 3.46447C4.92893 2 7.28595 2 12 2C16.714 2 19.0711 2 20.5355 3.46447C22 4.92893 22 7.28595 22 12C22 16.714 22 19.0711 20.5355 20.5355C19.0711 22 16.714 22 12 22C7.28595 22 4.92893 22 3.46447 20.5355C2 19.0711 2 16.714 2 12Z" stroke="black" stroke-width="1.5"/><path d="M17 7H14M17 7V10M17 7L13.5 10.5M7 17H10M7 17V14M7 17L10.5 13.5" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"filter":         `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M22 7L2 7" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M19 12L5 12" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M16 17H8" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
	"sidebar":        `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M2 11C2 7.22876 2 5.34315 3.17157 4.17157C4.34315 3 6.22876 3 10 3H14C17.7712 3 19.6569 3 20.8284 4.17157C22 5.34315 22 7.22876 22 11V13C22 16.7712 22 18.6569 20.8284 19.8284C19.6569 21 17.7712 21 14 21H10C6.22876 21 4.34315 21 3.17157 19.8284C2 18.6569 2 16.7712 2 13V11Z" stroke="black" stroke-width="1.5"/><path d="M15 21L15 3" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
	"import":         `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 15C3 17.8284 3 19.2426 3.87868 20.1213C4.75736 21 6.17157 21 9 21H15C17.8284 21 19.2426 21 20.1213 20.1213C21 19.2426 21 17.8284 21 15" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M12 3V16M12 16L16 11.625M12 16L8 11.625" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"sort":           `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M16 18L16 6M16 6L20 10.125M16 6L12 10.125" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M8 6L8 18M8 18L12 13.875M8 18L4 13.875" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"masonry":        `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M2.5 6.5C2.5 4.61438 2.5 3.67157 3.08579 3.08579C3.67157 2.5 4.61438 2.5 6.5 2.5C8.38562 2.5 9.32843 2.5 9.91421 3.08579C10.5 3.67157 10.5 4.61438 10.5 6.5V17.5C10.5 19.3856 10.5 20.3284 9.91421 20.9142C9.32843 21.5 8.38562 21.5 6.5 21.5C4.61438 21.5 3.67157 21.5 3.08579 20.9142C2.5 20.3284 2.5 19.3856 2.5 17.5V6.5Z" stroke="black" stroke-width="1.5"/><path d="M13.5 15.5C13.5 13.6144 13.5 12.6716 14.0858 12.0858C14.6716 11.5 15.6144 11.5 17.5 11.5C19.3856 11.5 20.3284 11.5 20.9142 12.0858C21.5 12.6716 21.5 13.6144 21.5 15.5V17.5C21.5 19.3856 21.5 20.3284 20.9142 20.9142C20.3284 21.5 19.3856 21.5 17.5 21.5C15.6144 21.5 14.6716 21.5 14.0858 20.9142C13.5 20.3284 13.5 19.3856 13.5 17.5V15.5Z" stroke="black" stroke-width="1.5"/><path d="M13.5 5.5C13.5 4.56812 13.5 4.10218 13.6522 3.73463C13.8552 3.24458 14.2446 2.85523 14.7346 2.65224C15.1022 2.5 15.5681 2.5 16.5 2.5H18.5C19.4319 2.5 19.8978 2.5 20.2654 2.65224C20.7554 2.85523 21.1448 3.24458 21.3478 3.73463C21.5 4.10218 21.5 4.56812 21.5 5.5C21.5 6.43188 21.5 6.89782 21.3478 7.26537C21.1448 7.75542 20.7554 8.14477 20.2654 8.34776C19.8978 8.5 19.4319 8.5 18.5 8.5H16.5C15.5681 8.5 15.1022 8.5 14.7346 8.34776C14.2446 8.14477 13.8552 7.75542 13.6522 7.26537C13.5 6.89782 13.5 6.43188 13.5 5.5Z" stroke="black" stroke-width="1.5"/></svg>`,
	"list":           `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M20 7L4 7" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M20 12L4 12" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M20 17L4 17" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
}

var toolbarIcons = func() map[string]*ui.SVG {
	icons := parseIcons(toolbarIconSource, true)
	icons["inspector"] = icons["sidebar"]
	// 搜索、导入的加号、排序后的小箭头直接用通用图标的加粗版。
	for name, source := range map[string]string{"search": "magnifying-glass", "caret-down": "caret-down"} {
		icons[name] = iconSidebar[source]
	}
	return icons
}()

// 工具栏里的图标：工具栏专用的图形优先，没有就回落到普通图标。
func toolbarIcon(c *ui.Context, name string) ui.Element {
	if svg := toolbarIcons[name]; svg != nil {
		return ui.Icon(c, svg).Size(iconSize, iconSize)
	}
	return appIcon(c, name)
}
