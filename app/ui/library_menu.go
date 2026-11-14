package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	libraryMenuWidth   = 300
	libraryRowHeight   = space40 + space4
	libraryLogoSize    = space24
	libraryVisibleRows = 7 // 超过这么多个图库时列表自己滚动，操作组始终可见
	// 操作组的图标列与图库行的图标列对齐：手柄宽 + 间距之后才是图标。
	libraryActionInset = space4 + iconSmall + space8
)

// sidebarLibrary 是侧栏左上角的图库切换器：平时只有图标、名称和上下箭头，悬停才有底。
func (a *App) sidebarLibrary(c *ui.Context) {
	p := paletteFor(c.Theme())
	name := a.T("open_library")
	if a.Library != nil {
		name = a.libraryName()
	}
	trigger := ui.ButtonBase(c.Key("library_switcher")).Height(sidebarLibraryHeight).MinWidth(0).AlignSelf(ui.Start).Padding(0, sidebarLibraryEnd, 0, sidebarLibraryInset).Gap(space8).Radius(sidebarRowRadius).Cursor(ui.CursorPointer).Label(a.T("library_menu")).Expanded(a.Sidebar.LibraryOpen)
	background := ui.Transparent
	if trigger.Pressed() {
		background = p.Pressed
	} else if trigger.Hovered() || a.Sidebar.LibraryOpen {
		background = p.Hover
	}
	animateColors(trigger, background, p.Text)
	trigger.Children(func() {
		ui.Image(c, appLogo).Size(space20, space20).Radius(sidebarLogoRadius).Shrink(0)
		ui.Text(c, name).FontSize(fontBody).FontWeight(600).SingleLine().MinWidth(0)
		caret := p.TextSecondary
		if trigger.Hovered() || a.Sidebar.LibraryOpen {
			caret = p.Text
		}
		mark := appSymbol(c, "chevrons-up-down").Size(iconSmall, iconSmall).Shrink(0)
		mark.TextColor(mark.AnimateColor("caret", caret, motionColor))
	})
	if trigger.Clicked() {
		a.Sidebar.LibraryOpen = !a.Sidebar.LibraryOpen
	}
	frostedPopover(c, trigger, &a.Sidebar.LibraryOpen, func() { a.libraryMenu(c) })
}

func (a *App) libraryMenu(c *ui.Context) {
	ui.Column(c.Key("library_menu_panel")).Width(libraryMenuWidth).Gap(0).Children(func() {
		if len(a.Recent) > 0 {
			a.Sidebar.RecentList.Key = func(i int) any { return a.Recent[i].Path }
			a.Sidebar.RecentList.Label = func(i int) string { return a.Recent[i].DisplayName }
			a.Sidebar.RecentList.Reorder = func(rows []int, to int) {
				paths := make([]string, len(a.Recent))
				for i, r := range a.Recent {
					paths[i] = r.Path
				}
				a.ReorderRecent(sidebarReorderPaths(paths, rows, to))
			}
			ui.List(c.Key("recent_libraries"), &a.Sidebar.RecentList, len(a.Recent), func(i int) {
				a.libraryRecentRow(c, a.Recent[i])
			}).FillWidth().Height(float32(min(libraryVisibleRows, len(a.Recent)) * libraryRowHeight))
			ui.Divider(c).Margin(space4, 0)
		}
		a.libraryMenuAction(c, "new_library", "create-library", func() { a.ShowNewLibrary() })
		a.libraryMenuAction(c, "open_library", "open-library", func() { a.ChooseOpenLibrary() })
		a.libraryMenuAction(c, "import_library", "import-library", a.ChooseImportLibrary)
		a.libraryMenuAction(c, "export_library", "export-library", a.ChooseExportLibrary)
	})
}

// libraryRecentRow 一行一个图库：拖动手柄、图标、名称和路径、当前库打勾、更多操作。
func (a *App) libraryRecentRow(c *ui.Context, recent library.Recent) {
	p := paletteFor(c.Theme())
	current := a.Library != nil && a.Library.Root == recent.Path
	if a.Sidebar.LibraryMore == nil {
		a.Sidebar.LibraryMore = make(map[string]*bool)
	}
	moreOpen := a.Sidebar.LibraryMore[recent.Path]
	if moreOpen == nil {
		moreOpen = new(bool)
		a.Sidebar.LibraryMore[recent.Path] = moreOpen
	}
	// 外层只管悬停底色；主体按钮负责切换，「…」是并列的菜单按钮，不嵌在按钮里。
	row := ui.Row(c.Key("library-row")).FillWidth().Height(libraryRowHeight).Padding(0, space8, 0, space4).Gap(space4).Radius(radiusControl).AlignItems(ui.Center)
	var open ui.Element
	row.Children(func() {
		open = ui.ButtonBase(c.Key("library-open")).Grow(1).MinWidth(0).FillHeight().Padding(0).Gap(space8).Cursor(ui.CursorPointer).Label(recent.DisplayName).Role(ui.RoleRadio).AccessSelected(current).Justify(ui.Start)
		open.Children(func() {
			// List 只支持整行拖动排序；手柄负责提示可以拖。
			ui.Box(c.Key("handle")).Width(iconSmall).FillHeight().Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorGrab).Tooltip(a.T("library_drag_reorder")).Children(func() {
				appIcon(c, "drag-handle").Size(iconSmall, iconSmall).TextColor(p.TextTertiary)
			})
			ui.Image(c, appLogo).Size(libraryLogoSize, libraryLogoSize).Radius(radiusCard).Shrink(0)
			ui.Column(c).Grow(1).MinWidth(0).Gap(0).Children(func() {
				weight := 400
				if current {
					weight = 600
				}
				ui.Text(c, recent.DisplayName).FontSize(fontBody).FontWeight(weight).SingleLine().FillWidth().MinWidth(0)
				ui.Text(c, libraryDisplayPath(recent.Path)).FontSize(fontGroup).TextColor(p.TextTertiary).SingleLine().TruncateMiddle().FillWidth().MinWidth(0)
			})
			if current {
				appSymbol(c, "check").Size(iconSmall, iconSmall).TextColor(p.Accent).Shrink(0)
			}
		})
		a.libraryMoreMenu(c, recent, current, moreOpen)
	})
	background := ui.Transparent
	if open.Pressed() {
		background = p.Pressed
	} else if row.Hovered() || *moreOpen {
		background = p.Hover
	}
	animateColors(row, background, p.Text)
	if open.Clicked() {
		a.Sidebar.LibraryOpen = false
		if !current {
			a.OpenLibrary(recent.Path)
		}
	}
}

// libraryMoreMenu 是每个图库行尾的「…」菜单；当前图库能改名、清缓存和删除，其他图库只能显示或移出列表。
func (a *App) libraryMoreMenu(c *ui.Context, recent library.Recent, current bool, open *bool) {
	p := paletteFor(c.Theme())
	menu := ui.MenuBase(c.Key("more"), open)
	more := menu.Trigger.Size(controlSmall, controlSmall).Shrink(0).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer).Label(a.T("library_more")).Tooltip(a.T("library_more")).Justify(ui.Center).AlignItems(ui.Center)
	// 按钮压在已经是 Hover 底的行上，悬停要再深一档才看得出来。
	background, text := ui.Transparent, p.TextSecondary
	if more.Pressed() || more.Hovered() || *open {
		background, text = p.Pressed, p.Text
	}
	animateColors(more, background, text)
	more.Children(func() { appIcon(c, "more").Size(iconSmall, iconSmall) })
	menu.Popup(func(panel ui.Element) {
		elevated(popupMotion(panel), c.Theme()).Material(panelMaterial(c.Theme())).Radius(radiusPanel).Padding(space4).Gap(0)
		choose := func(chosen bool, action func()) {
			if chosen {
				a.Sidebar.LibraryOpen = false
				action()
			}
		}
		if current {
			name := a.libraryName()
			choose(libraryMenuItem(c, menu, a.T("rename_library"), "rename", false).Disabled(a.Busy).Clicked(), func() { a.Ask(a.T("rename_library"), name, a.RenameLibrary) })
			choose(libraryMenuItem(c, menu, a.T("reveal_library"), "reveal", false).Clicked(), a.RevealLibrary)
			choose(libraryMenuItem(c, menu, a.T("clear_library_cache"), "reload", false).Disabled(a.Busy).Clicked(), a.ClearLibraryCache)
			ui.Divider(c).Margin(space4, 0)
			choose(libraryMenuItem(c, menu, a.T("delete_library"), "delete", true).Disabled(a.Busy).Clicked(), a.DeleteLibrary)
			return
		}
		path := recent.Path
		choose(libraryMenuItem(c, menu, a.T("reveal_library"), "reveal", false).Clicked(), func() { mygo.Shell.ShowItemInFolder(path) })
		ui.Divider(c).Margin(space4, 0)
		choose(libraryMenuItem(c, menu, a.T("remove_recent_library"), "remove", false).Clicked(), func() { a.RemoveRecent(path) })
	})
}

func libraryMenuItem(c *ui.Context, menu ui.MenuParts, label, icon string, danger bool) ui.Element {
	p := paletteFor(c.Theme())
	item := menu.Item().Height(controlHeight).Padding(0, space12, 0, space8).Gap(space8).Radius(radiusControl).AlignItems(ui.Center).Label(label).Cursor(ui.CursorPointer)
	// 图标平时用次要色，高亮时跟文字一起变深；删除类整行用危险色。
	background, text, ink := ui.Transparent, p.Text, p.TextSecondary
	if item.Highlighted() {
		background, ink = p.Hover, p.Text
	}
	if danger {
		text, ink = p.Danger, p.Danger
	}
	animateColors(item, background, text)
	item.Children(func() {
		mark := appIcon(c, icon).Size(iconMedium, iconMedium).Shrink(0)
		mark.TextColor(mark.AnimateColor("ink", ink, motionColor))
		ui.Text(c, label).FontSize(fontBody).SingleLine()
	})
	return item
}

// libraryMenuAction 是列表下面的图库操作，图标列与上面图库行的图标列对齐。
func (a *App) libraryMenuAction(c *ui.Context, key, icon string, action func()) {
	p := paletteFor(c.Theme())
	row := ui.ButtonBase(c.Key(key)).FillWidth().Height(controlHeight).Padding(0, space8, 0, libraryActionInset).Gap(space8).Radius(radiusControl).Cursor(ui.CursorPointer).Label(a.T(key)).Justify(ui.Start).Disabled(a.Busy)
	background, ink := ui.Transparent, p.TextSecondary
	if row.Pressed() {
		background, ink = p.Pressed, p.Text
	} else if row.Hovered() {
		background, ink = p.Hover, p.Text
	}
	animateColors(row, background, p.Text)
	row.Children(func() {
		ui.Box(c).Size(libraryLogoSize, libraryLogoSize).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			mark := appIcon(c, icon).Size(iconSize, iconSize)
			mark.TextColor(mark.AnimateColor("ink", ink, motionColor))
		})
		ui.Text(c, a.T(key)).FontSize(fontBody).SingleLine().Grow(1).MinWidth(0)
	})
	if row.Clicked() {
		a.Sidebar.LibraryOpen = false
		action()
	}
}

// libraryDisplayPath 把用户目录缩写成 ~，路径短一些更容易认出图库。
func libraryDisplayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rest
	}
	return path
}

// RemoveRecent 只把图库从最近列表里拿掉，不动磁盘上的图库；写入在后台进行。
func (a *App) RemoveRecent(path string) {
	if a.recents == nil {
		return
	}
	store := a.recents
	go func() {
		recent, err := removeRecentLibrary(store, path)
		a.Window.Update(func() { a.recentRemoved(recent, err) })
	}()
}

func removeRecentLibrary(store *library.RecentStore, path string) ([]library.Recent, error) {
	if err := store.Remove(path); err != nil {
		return nil, err
	}
	return store.List()
}

func (a *App) recentRemoved(recent []library.Recent, err error) {
	if err != nil {
		a.ShowError(err)
		return
	}
	a.Recent = recent
}
