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
	"recent":             "clock-counter-clockwise",
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
	// 筛选里的文件类型：Solar 没有按格式区分的图标，按种类取 Camera Minimalistic（照片）、Gallery Minimalistic（图像）、Play Circle（动图）、Videocamera Record（视频）。
	"type-photo":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><circle cx="12" cy="13" r="3"/><path d="M14.0001 20H10.0001C7.19117 20 5.78671 20 4.77782 19.3259C4.34106 19.034 3.96605 18.659 3.67422 18.2223C3.0001 17.2134 3.0001 15.8089 3.0001 13L3.0001 12.9981C3.0001 10.1905 3.0001 8.78661 3.67407 7.77794C3.96591 7.34118 4.34091 6.96618 4.77767 6.67434C5.78656 6.00022 7.19103 6.00022 9.99995 6.00022H14C16.8089 6.00022 18.2133 6.00022 19.2222 6.67434C19.659 6.96618 20.034 7.34118 20.3258 7.77794C21 8.78683 21.0001 10.1911 21.0001 13C21.0001 15.8089 21.0001 17.2134 20.326 18.2223C20.0341 18.659 19.6591 19.034 19.2224 19.3259C18.2135 20 16.809 20 14.0001 20Z"/><path d="M18 10H17.5"/><path d="M15 3H9"/></g></svg>`,
	"type-image":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><path d="M12 22C7.28595 22 4.92893 22 3.46447 20.5355C2 19.0711 2 16.714 2 12C2 7.28595 2 4.92893 3.46447 3.46447C4.92893 2 7.28595 2 12 2C16.714 2 19.0711 2 20.5355 3.46447C22 4.92893 22 7.28595 22 12C22 16.714 22 19.0711 20.5355 20.5355C19.0711 22 16.714 22 12 22Z"/><circle cx="16" cy="8" r="2"/><path d="M5 13.307L5.81051 12.5542C6.73658 11.6941 8.18321 11.7424 9.04988 12.6623L11.6974 15.4727C12.2356 16.0439 13.1166 16.1209 13.7457 15.6516C14.6522 14.9753 15.9144 15.0522 16.7322 15.8334L19 18"/></g></svg>`,
	"type-video":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><path d="M17 9.49995L17.6584 9.17077C19.6042 8.19783 20.5772 7.71135 21.2886 8.15102C22 8.5907 22 9.67848 22 11.8541V12.1458C22 14.3214 22 15.4092 21.2886 15.8489C20.5772 16.2885 19.6042 15.8021 17.6584 14.8291L17 14.4999V9.49995Z"/><path d="M13.5607 7.43934C14.1464 8.02513 14.1464 8.97487 13.5607 9.56066C12.9749 10.1464 12.0251 10.1464 11.4393 9.56066C10.8536 8.97487 10.8536 8.02513 11.4393 7.43934C12.0251 6.85355 12.9749 6.85355 13.5607 7.43934Z"/><path d="M9.5 20C6.21252 20 4.56878 20 3.46243 19.092C3.25989 18.9258 3.07418 18.7401 2.90796 18.5376C2 17.4312 2 15.7875 2 12.5V11.5C2 8.21252 2 6.56878 2.90796 5.46243C3.07418 5.25989 3.25989 5.07418 3.46243 4.90796C4.56878 4 6.21252 4 9.5 4C12.7875 4 14.4312 4 15.5376 4.90796C15.7401 5.07418 15.9258 5.25989 16.092 5.46243C17 6.56878 17 8.21252 17 11.5V12.5C17 15.7875 17 17.4312 16.092 18.5376C15.9258 18.7401 15.7401 18.9258 15.5376 19.092C14.4312 20 12.7875 20 9.5 20Z"/></g></svg>`,
	"type-live":   `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"><path d="M11 12a1 1 0 1 0 2 0a1 1 0 1 0-2 0"/><path d="M7 12a5 5 0 1 0 10 0a5 5 0 1 0-10 0m8.9 8.11v.01m3.14-2.51v.01M20.77 14v.01m0-4.01v.01m-1.73-3.62v.01M15.9 3.89v.01M12 3v.01m-3.9.88v.01M4.96 6.39v.01M3.23 10v.01m0 3.99v.01m1.73 3.6v.01m3.14 2.49v.01M12 21v.01"/></g></svg>`, // 实况照片：Solar 没有，借用 Tabler（MIT）的 Live Photo
	"type-motion": `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M9 9.21316L9 14.7868C9 15.7105 9.93371 16.2944 10.6935 15.8458L15.4137 13.059C16.1954 12.5974 16.1954 11.4026 15.4137 10.941L10.6935 8.15419C9.93371 7.70561 9 8.28947 9 9.21316Z"/></g></svg>`,
	// 筛选里的形状：横图用 Solar Rectangle，竖图是它转 90°，方图用 Stop（圆角正方形），全景用 Panorama。
	"shape-landscape": `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M15 19H9C6.1911 19 4.7866 19 3.7777 18.326C3.341 18.034 2.966 17.659 2.6741 17.222C2 16.213 2 14.809 2 12C2 9.191 2 7.787 2.6741 6.778C2.966 6.341 3.341 5.966 3.7777 5.674C4.7866 5 6.1911 5 9 5H15C17.8089 5 19.2134 5 20.2223 5.674C20.659 5.966 21.034 6.341 21.3259 6.778C22 7.787 22 9.191 22 12C22 14.809 22 16.213 21.3259 17.222C21.034 17.659 20.659 18.034 20.2223 18.326C19.2134 19 17.8089 19 15 19Z" stroke="black" stroke-width="1.5"/></svg>`,
	"shape-portrait":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g transform="rotate(90 12 12)"><path d="M15 19H9C6.1911 19 4.7866 19 3.7777 18.326C3.341 18.034 2.966 17.659 2.6741 17.222C2 16.213 2 14.809 2 12C2 9.191 2 7.787 2.6741 6.778C2.966 6.341 3.341 5.966 3.7777 5.674C4.7866 5 6.1911 5 9 5H15C17.8089 5 19.2134 5 20.2223 5.674C20.659 5.966 21.034 6.341 21.3259 6.778C22 7.787 22 9.191 22 12C22 14.809 22 16.213 21.3259 17.222C21.034 17.659 20.659 18.034 20.2223 18.326C19.2134 19 17.8089 19 15 19Z" stroke="black" stroke-width="1.5"/></g></svg>`,
	"shape-square":    `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M12 22C7.28595 22 4.92893 22 3.46447 20.5355C2 19.0711 2 16.714 2 12C2 7.28595 2 4.92893 3.46447 3.46447C4.92893 2 7.28595 2 12 2C16.714 2 19.0711 2 20.5355 3.46447C22 4.92893 22 7.28595 22 12C22 16.714 22 19.0711 20.5355 20.5355C19.0711 22 16.714 22 12 22Z" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
	"shape-panorama":  `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><path d="M22 5.85976C22 5.06521 21.2094 4.33387 19.883 3.75291C18.7083 3.23847 17.5 4.21975 17.5 5.50211V8.71958M22 5.85976V18.5762C22 19.0994 21.6573 19.5951 21.0445 20.0386C19.4426 21.1979 15.9948 21.9999 12 21.9999C7.24272 21.9999 3.26132 20.8626 2.24895 19.3388C2.08603 19.0936 2 18.8383 2 18.5762V5.85976M2 5.85976C2 7.05499 3.78889 8.10717 6.5 8.71958C8.0779 9.07601 9.96818 9.28348 12 9.28348C14.0318 9.28348 15.9221 9.07601 17.5 8.71958C20.2111 8.10717 22 7.05499 22 5.85976M2 5.85976C2 5.06521 2.79055 4.33387 4.11705 3.75291C5.29169 3.23847 6.5 4.21975 6.5 5.50211V8.71958"/><path d="M19.5 13C19.5 13.8284 18.8284 14.5 18 14.5C17.1716 14.5 16.5 13.8284 16.5 13C16.5 12.1716 17.1716 11.5 18 11.5C18.8284 11.5 19.5 12.1716 19.5 13Z"/><path d="M2.24805 19.3387L6.28948 15.5566L7.64554 14.4214C8.35104 13.8309 9.41434 13.8648 10.0772 14.4989L13.3983 17.6758C13.9303 18.1848 14.7679 18.2542 15.3835 17.8403L15.6143 17.6851C16.5002 17.0896 17.6987 17.1586 18.5035 17.8514L21.0437 20.0384"/></g></svg>`,
	// 排序方向：把 Solar Sort Vertical 拆成右边朝上、左边朝下两支箭头，分别上色表示升序或降序。
	"sort-up":        `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M16 18L16 6M12 10.125L16 6L20 10.125" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"sort-down":      `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M8 6L8 18M4 13.875L8 18L12 13.875" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"thumbnail-size": `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M2 12C2 7.28595 2 4.92893 3.46447 3.46447C4.92893 2 7.28595 2 12 2C16.714 2 19.0711 2 20.5355 3.46447C22 4.92893 22 7.28595 22 12C22 16.714 22 19.0711 20.5355 20.5355C19.0711 22 16.714 22 12 22C7.28595 22 4.92893 22 3.46447 20.5355C2 19.0711 2 16.714 2 12Z" stroke="black" stroke-width="1.5"/><path d="M17 7H14M17 7V10M17 7L13.5 10.5M7 17H10M7 17V14M7 17L10.5 13.5" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"filter":         `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M22 7L2 7" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M19 12L5 12" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M16 17H8" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
	"sidebar":        `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M2 11C2 7.22876 2 5.34315 3.17157 4.17157C4.34315 3 6.22876 3 10 3H14C17.7712 3 19.6569 3 20.8284 4.17157C22 5.34315 22 7.22876 22 11V13C22 16.7712 22 18.6569 20.8284 19.8284C19.6569 21 17.7712 21 14 21H10C6.22876 21 4.34315 21 3.17157 19.8284C2 18.6569 2 16.7712 2 13V11Z" stroke="black" stroke-width="1.5"/><path d="M15 21L15 3" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
	"import":         `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M3 15C3 17.8284 3 19.2426 3.87868 20.1213C4.75736 21 6.17157 21 9 21H15C17.8284 21 19.2426 21 20.1213 20.1213C21 19.2426 21 17.8284 21 15" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M12 3V16M12 16L16 11.625M12 16L8 11.625" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"sort":           `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M16 18L16 6M16 6L20 10.125M16 6L12 10.125" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/><path d="M8 6L8 18M8 18L12 13.875M8 18L4 13.875" stroke="black" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"masonry":        `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M2.5 6.5C2.5 4.61438 2.5 3.67157 3.08579 3.08579C3.67157 2.5 4.61438 2.5 6.5 2.5C8.38562 2.5 9.32843 2.5 9.91421 3.08579C10.5 3.67157 10.5 4.61438 10.5 6.5V17.5C10.5 19.3856 10.5 20.3284 9.91421 20.9142C9.32843 21.5 8.38562 21.5 6.5 21.5C4.61438 21.5 3.67157 21.5 3.08579 20.9142C2.5 20.3284 2.5 19.3856 2.5 17.5V6.5Z" stroke="black" stroke-width="1.5"/><path d="M13.5 15.5C13.5 13.6144 13.5 12.6716 14.0858 12.0858C14.6716 11.5 15.6144 11.5 17.5 11.5C19.3856 11.5 20.3284 11.5 20.9142 12.0858C21.5 12.6716 21.5 13.6144 21.5 15.5V17.5C21.5 19.3856 21.5 20.3284 20.9142 20.9142C20.3284 21.5 19.3856 21.5 17.5 21.5C15.6144 21.5 14.6716 21.5 14.0858 20.9142C13.5 20.3284 13.5 19.3856 13.5 17.5V15.5Z" stroke="black" stroke-width="1.5"/><path d="M13.5 5.5C13.5 4.56812 13.5 4.10218 13.6522 3.73463C13.8552 3.24458 14.2446 2.85523 14.7346 2.65224C15.1022 2.5 15.5681 2.5 16.5 2.5H18.5C19.4319 2.5 19.8978 2.5 20.2654 2.65224C20.7554 2.85523 21.1448 3.24458 21.3478 3.73463C21.5 4.10218 21.5 4.56812 21.5 5.5C21.5 6.43188 21.5 6.89782 21.3478 7.26537C21.1448 7.75542 20.7554 8.14477 20.2654 8.34776C19.8978 8.5 19.4319 8.5 18.5 8.5H16.5C15.5681 8.5 15.1022 8.5 14.7346 8.34776C14.2446 8.14477 13.8552 7.75542 13.6522 7.26537C13.5 6.89782 13.5 6.43188 13.5 5.5Z" stroke="black" stroke-width="1.5"/></svg>`,
	// 等高行：Solar Widget 5，两行宽窄不一的块。
	"justified": `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><g fill="none" stroke="black" stroke-linecap="round" stroke-width="1.5"><path d="M17.5 21.5C15.6144 21.5 14.6716 21.5 14.0858 20.9142C13.5 20.3284 13.5 19.3856 13.5 17.5V15.5C13.5 13.6144 13.5 12.6716 14.0858 12.0858C14.6716 11.5 15.6144 11.5 17.5 11.5C19.3856 11.5 20.3284 11.5 20.9142 12.0858C21.5 12.6716 21.5 13.6144 21.5 15.5V17.5C21.5 19.3856 21.5 20.3284 20.9142 20.9142C20.3284 21.5 19.3856 21.5 17.5 21.5Z"/><path d="M6 2.5C4.11438 2.5 3.17157 2.5 2.58579 3.08579C2 3.67157 2 4.61438 2 6.5V8.5C2 10.3856 2 11.3284 2.58579 11.9142C3.17157 12.5 4.11438 12.5 6 12.5C7.88562 12.5 8.82843 12.5 9.41421 11.9142C10 11.3284 10 10.3856 10 8.5V6.5C10 4.61438 10 3.67157 9.41421 3.08579C8.82843 2.5 7.88562 2.5 6 2.5Z"/><path d="M16.5 8.5H18.5C19.4319 8.5 19.8978 8.5 20.2654 8.34776C20.7554 8.14477 21.1448 7.75542 21.3478 7.26537C21.5 6.89782 21.5 6.43188 21.5 5.5C21.5 4.56812 21.5 4.10218 21.3478 3.73463C21.1448 3.24458 20.7554 2.85523 20.2654 2.65224C19.8978 2.5 19.4319 2.5 18.5 2.5H16.5C15.5681 2.5 15.1022 2.5 14.7346 2.65224C14.2446 2.85523 13.8552 3.24458 13.6522 3.73463C13.5 4.10218 13.5 4.56812 13.5 5.5C13.5 6.43188 13.5 6.89782 13.6522 7.26537C13.8552 7.75542 14.2446 8.14477 14.7346 8.34776C15.1022 8.5 15.5681 8.5 16.5 8.5Z"/><path d="M5 15.5H7C7.93188 15.5 8.39782 15.5 8.76537 15.6522C9.25542 15.8552 9.64477 16.2446 9.84776 16.7346C10 17.1022 10 17.5681 10 18.5C10 19.4319 10 19.8978 9.84776 20.2654C9.64477 20.7554 9.25542 21.1448 8.76537 21.3478C8.39782 21.5 7.93188 21.5 7 21.5H5C4.06812 21.5 3.60218 21.5 3.23463 21.3478C2.74458 21.1448 2.35523 20.7554 2.15224 20.2654C2 19.8978 2 19.4319 2 18.5C2 17.5681 2 17.1022 2.15224 16.7346C2.35523 16.2446 2.74458 15.8552 3.23463 15.6522C3.60218 15.5 4.06812 15.5 5 15.5Z"/></g></svg>`,
	"list":      `<svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M20 7L4 7" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M20 12L4 12" stroke="black" stroke-width="1.5" stroke-linecap="round"/><path d="M20 17L4 17" stroke="black" stroke-width="1.5" stroke-linecap="round"/></svg>`,
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
