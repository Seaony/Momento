package ui

import (
	"context"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 同一图库内的文件夹和素材拖动共用一个类型，目标行才能同时接收两者。
type AssetsDrag struct {
	LibraryID string
	IDs       []string
	FolderID  string
}

type sidebarState struct {
	LibraryOpen bool
	RecentList  ui.ListState
	Open        map[string]*bool
}

func (a *App) sidebar(c *ui.Context) {
	if a.Sidebar.Open == nil {
		a.Sidebar.Open = make(map[string]*bool)
	}
	counts := catalog.Counts{}
	if a.Index != nil {
		counts = a.Index.Counts()
	}
	ui.Column(c).Fill().Padding(0, 8, 8, 8).Gap(4).Background(c.Theme().Surface).Children(func() {
		if c.TitleBar().Left > 0 {
			ui.Box(c.Key("sidebar-titlebar")).Height(max(MainToolbarHeight, c.TitleBar().Bottom+8)).FillWidth().Shrink(0).DragWindow()
			a.sidebarLibrary(c)
		} else {
			ui.Column(c.Key("sidebar-titlebar")).Height(MainToolbarHeight).FillWidth().Shrink(0).Center().DragWindow().Children(func() {
				a.sidebarLibrary(c)
			})
		}
		ui.Divider(c).Margin(8, 0)
		ui.Scroll(c.Key("sidebar_scopes")).Grow(1).Gap(3).Children(func() {
			a.sidebarScope(c, "all", "all", "照片", counts.All)
			a.sidebarScope(c, "favorites", "favorites", "喜欢", counts.Favorites)
			a.sidebarScope(c, "uncategorized", "uncategorized", "文件夹", counts.Uncategorized)
			a.sidebarScope(c, "untagged", "untagged", "标签", counts.Untagged)
			tags := a.sidebarRow(c.Key("tags_page"), a.T("tag_management"), "标签组", -1, a.TagPage)
			if tags.Clicked() {
				a.TagPage = true
				a.Selected.Clear()
			}
			ui.Row(c).Height(40).Padding(8, 8, 0, 10).Children(func() {
				title := ui.Text(c, a.T("folders")).FontSize(12).Bold().TextColor(c.Theme().TextMuted).Grow(1)
				a.sidebarRootDrop(c, title)
				if sidebarSymbolButton(c, a.T("new_folder")).Disabled(a.Busy).Clicked() {
					a.sidebarCreateFolder("")
				}
			})
			a.sidebarFolders(c, counts)
			root := ui.Box(c.Key("folders_root_drop")).Height(24).FillWidth().Label(a.T("move_to_root"))
			if drag, over := ui.DragOver[AssetsDrag](root); over && a.sidebarCanDrop(drag) && drag.FolderID != "" {
				root.Background(c.Theme().SurfaceHover).BorderWidth(0, 0, 2, 0).BorderColor(c.Theme().Accent)
			}
			if drag, ok := ui.Drop[AssetsDrag](root); ok {
				a.sidebarMoveFolder(drag, "", folderInside)
			}
		})
		ui.Divider(c)
		trash := a.sidebarScope(c, "trash", "trash", "回收站", counts.Trash)
		trash.PopoverMenu(func(m *ui.Menu) {
			if m.Item(a.T("empty_trash")).Disabled(counts.Trash == 0 || a.Busy).Chosen() {
				a.Confirm(a.T("empty_trash"), a.T("empty_trash_detail"), func() {
					service := a.Assets
					a.Run("empty_trash", func(ctx context.Context) error { return service.EmptyTrash(ctx) })
				})
			}
		})
		if a.sidebarRow(c.Key("settings"), a.T("settings"), "设置", -1, false).Clicked() {
			a.ShowSettings()
		}
	})
}

func (a *App) sidebarRow(c *ui.Context, label, icon string, count int, selected bool) ui.Element {
	row := ui.ButtonBase(c).FillWidth().Height(34).Padding(6, 10).Gap(8).Radius(6).Label(label).AccessSelected(selected).Justify(ui.Start)
	if selected {
		row.Background(c.Theme().SurfacePressed).TextColor(c.Theme().Text)
	} else if row.Hovered() {
		row.Background(c.Theme().SurfaceHover)
	}
	row.Children(func() {
		if icon != "" {
			ui.Image(c, sidebarIcon(icon)).Size(19, 19).Shrink(0)
		}
		ui.Text(c, label).SingleLine().Grow(1).MinWidth(0)
		if count >= 0 {
			color := c.Theme().TextMuted
			if selected {
				color = c.Theme().Text
			}
			ui.Text(c, sidebarCount(count)).FontSize(12).TextColor(color).Shrink(0)
		}
	})
	return row
}

func (a *App) sidebarScope(c *ui.Context, scope, key, icon string, count int) ui.Element {
	selected := !a.TagPage && (a.Query.Scope == scope || (scope == "all" && a.Query.Scope == ""))
	row := a.sidebarRow(c.Key("scope_"+scope), a.T(key), icon, count, selected)
	if row.Clicked() {
		a.SetScope(scope, "")
	}
	return row
}

func (a *App) sidebarLibrary(c *ui.Context) {
	name := a.T("open_library")
	if a.Library != nil {
		name = a.libraryName()
	}
	trigger := ui.ButtonBase(c.Key("library_switcher")).FillWidth().Height(40).Padding(8, 10).Gap(8).Radius(8).Label(a.T("library_menu")).Justify(ui.Start)
	if trigger.Hovered() {
		trigger.Background(c.Theme().SurfaceHover)
	}
	trigger.Children(func() {
		ui.Image(c, sidebarIcon("照片")).Size(22, 22)
		ui.Text(c, name).Bold().Grow(1).MinWidth(0).SingleLine()
		sidebarChevron(c, true).Size(12, 12)
	})
	if trigger.Clicked() {
		a.Sidebar.LibraryOpen = !a.Sidebar.LibraryOpen
	}
	ui.Popover(c, trigger, &a.Sidebar.LibraryOpen, func() {
		_, windowHeight := c.Size()
		menuHeight := float32(310)
		if len(a.Recent) > 0 {
			menuHeight += float32(min(10, len(a.Recent))*34 + 32)
		}
		ui.Scroll(c.Key("library_menu_scroll")).Width(284).Height(min(menuHeight, max(200, windowHeight-100))).Gap(4).Children(func() {
			if len(a.Recent) > 0 {
				ui.Text(c, a.T("recent_libraries")).FontSize(12).TextColor(c.Theme().TextMuted).Padding(4, 8)
				a.Sidebar.RecentList.Key = func(i int) any { return a.Recent[i].Path }
				a.Sidebar.RecentList.Reorder = func(rows []int, to int) {
					paths := make([]string, len(a.Recent))
					for i, r := range a.Recent {
						paths[i] = r.Path
					}
					a.ReorderRecent(sidebarReorderPaths(paths, rows, to))
				}
				ui.List(c.Key("recent_libraries"), &a.Sidebar.RecentList, len(a.Recent), func(i int) {
					recent := a.Recent[i]
					selected := a.Library != nil && a.Library.Root == recent.Path
					if a.sidebarRow(c, recent.DisplayName, "照片", -1, selected).Clicked() {
						a.Sidebar.LibraryOpen = false
						a.OpenLibrary(recent.Path)
					}
				}).Height(float32(min(10, len(a.Recent)) * 34)).MaxHeight(340)
				ui.Divider(c).Margin(4, 0)
			}
			a.sidebarLibraryAction(c, "new_library", func() { a.ShowNewLibrary() })
			a.sidebarLibraryAction(c, "open_library", func() { a.ChooseOpenLibrary() })
			a.sidebarLibraryAction(c, "import_library", a.ChooseImportLibrary)
			a.sidebarLibraryAction(c, "export_library", a.ChooseExportLibrary)
			ui.Divider(c).Margin(4, 0)
			a.sidebarLibraryAction(c, "rename_library", func() { a.Ask(a.T("rename_library"), name, a.RenameLibrary) })
			a.sidebarLibraryAction(c, "reveal_library", a.RevealLibrary)
			a.sidebarLibraryAction(c, "clear_library_cache", a.ClearLibraryCache)
			a.sidebarLibraryAction(c, "delete_library", a.DeleteLibrary)
		})
	})
}

func (a *App) sidebarLibraryAction(c *ui.Context, key string, action func()) {
	row := a.sidebarRow(c.Key(key), a.T(key), "", -1, false).Disabled(a.Busy)
	if key == "delete_library" {
		row.TextColor(c.Theme().Danger)
	}
	if row.Clicked() {
		a.Sidebar.LibraryOpen = false
		action()
	}
}

func (a *App) sidebarFolders(c *ui.Context, counts catalog.Counts) {
	children := sidebarFolderChildren(a.Snapshot.Folders)
	tree := ui.TreeBase(c.Key("folder_tree"))
	tree.Root.Gap(2).Children(func() {
		var build func(model.Folder)
		build = func(folder model.Folder) {
			open := a.Sidebar.Open[folder.ID]
			if open == nil {
				open = new(bool)
				a.Sidebar.Open[folder.ID] = open
			}
			var opened *bool
			if len(children[folder.ID]) > 0 {
				opened = open
			}
			c.Key(folder.ID)
			item := tree.Item(opened)
			row := item.Row.Height(34).Padding(6, 10).PaddingStart(6 + float32(item.Depth())*16).Gap(6).Radius(6).Label(folder.Name).AccessSelected(!a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID)
			if !a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID {
				row.Background(c.Theme().SurfacePressed).TextColor(c.Theme().Text)
			} else if row.Hovered() {
				row.Background(c.Theme().SurfaceHover)
			}
			row.Drag(AssetsDrag{LibraryID: a.libraryID(), FolderID: folder.ID})
			row.Children(func() {
				if opened != nil {
					arrow := ui.ButtonBase(c.Key("disclose")).Size(16, 20).Padding(0).Label(a.T("toggle_folder", folder.Name)).Children(func() { sidebarChevron(c, *opened).Size(10, 10) })
					if arrow.Clicked() {
						*opened = !*opened
					}
				} else {
					ui.Box(c).Size(16, 20)
				}
				ui.Image(c, sidebarIcon("文件夹")).Size(18, 18).Shrink(0)
				ui.Text(c, folder.Name).SingleLine().Grow(1).MinWidth(0)
				color := c.Theme().TextMuted
				if !a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID {
					color = c.Theme().Text
				}
				ui.Text(c, sidebarCount(counts.Folders[folder.ID])).FontSize(12).TextColor(color).Shrink(0)
			})
			if row.Clicked() {
				a.SetScope("folder", folder.ID)
			}
			a.sidebarFolderDrop(c, row, folder)
			row.PopoverMenu(func(m *ui.Menu) {
				if m.Item(a.T("new_subfolder")).Disabled(a.Busy).Chosen() {
					*open = true
					a.sidebarCreateFolder(folder.ID)
				}
				if m.Item(a.T("rename_folder")).Disabled(a.Busy).Chosen() {
					lib := a.Library
					a.Ask(a.T("rename_folder"), folder.Name, func(name string) {
						a.Run("rename_folder", func(ctx context.Context) error { return lib.RenameFolder(ctx, folder.ID, name) })
					})
				}
				m.Separator()
				if m.Item(a.T("delete_folder")).Disabled(a.Busy).Chosen() {
					a.Confirm(a.T("delete_folder"), a.T("delete_folder_detail", folder.Name), func() {
						lib := a.Library
						if a.Query.Scope == "folder" && sidebarFolderContains(a.Snapshot.Folders, folder.ID, a.Query.FolderID) {
							a.SetScope("all", "")
						}
						a.Run("delete_folder", func(ctx context.Context) error { return lib.DeleteFolder(ctx, folder.ID) })
					})
				}
			})
			item.Group(func() {
				for _, child := range children[folder.ID] {
					build(child)
				}
			})
		}
		for _, folder := range children[""] {
			build(folder)
		}
	})
}

func (a *App) sidebarCreateFolder(parentID string) {
	lib := a.Library
	a.Ask(a.T("new_folder"), "", func(name string) {
		a.Run("new_folder", func(ctx context.Context) error { _, err := lib.CreateFolder(ctx, name, parentID); return err })
	})
}

func (a *App) sidebarCanDrop(drag AssetsDrag) bool {
	return a.Library != nil && !a.Busy && drag.LibraryID == a.libraryID()
}
func (a *App) sidebarRootDrop(c *ui.Context, row ui.Element) {
	if drag, over := ui.DragOver[AssetsDrag](row); over && a.sidebarCanDrop(drag) && drag.FolderID != "" {
		row.TextColor(c.Theme().Accent)
	}
	if drag, ok := ui.Drop[AssetsDrag](row); ok {
		a.sidebarMoveFolder(drag, "", folderInside)
	}
}
func (a *App) sidebarFolderDrop(c *ui.Context, row ui.Element, folder model.Folder) {
	_, y, _ := row.PointerPosition()
	position := sidebarDropPosition(y, 34)
	if drag, over := ui.DragOver[AssetsDrag](row); over && a.sidebarCanDrop(drag) {
		if drag.FolderID != "" {
			if _, _, valid := sidebarFolderMove(a.Snapshot.Folders, drag.FolderID, folder.ID, position); valid {
				switch position {
				case folderBefore:
					row.BorderWidth(2, 0, 0, 0).BorderColor(c.Theme().Accent)
				case folderAfter:
					row.BorderWidth(0, 0, 2, 0).BorderColor(c.Theme().Accent)
				default:
					row.Background(c.Theme().Selection).TextColor(c.Theme().Text)
				}
			}
		} else if len(drag.IDs) > 0 {
			row.Background(c.Theme().Selection).TextColor(c.Theme().Text)
		}
	}
	if drag, ok := ui.Drop[AssetsDrag](row); ok && a.sidebarCanDrop(drag) {
		if drag.FolderID != "" {
			a.sidebarMoveFolder(drag, folder.ID, position)
		} else if len(drag.IDs) > 0 {
			a.SetFolders(drag.IDs, []string{folder.ID}, true)
		}
	}
}
func (a *App) sidebarMoveFolder(drag AssetsDrag, targetID string, position folderDropPosition) {
	if !a.sidebarCanDrop(drag) || drag.FolderID == "" {
		return
	}
	parent, index, ok := sidebarFolderMove(a.Snapshot.Folders, drag.FolderID, targetID, position)
	if !ok {
		return
	}
	if parent != "" {
		open := a.Sidebar.Open[parent]
		if open == nil {
			open = new(bool)
			a.Sidebar.Open[parent] = open
		}
		*open = true
	}
	lib := a.Library
	a.Run("move_folder", func(ctx context.Context) error { return lib.MoveFolder(ctx, drag.FolderID, parent, index) })
}

var sidebarArrow = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path d="M6 3.5 10.5 8 6 12.5" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
var sidebarPlus = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16"><path d="M8 3v10M3 8h10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>`))

func sidebarChevron(c *ui.Context, open bool) ui.Element {
	arrow := ui.Icon(c, sidebarArrow)
	if open {
		arrow.Rotate(90)
	}
	return arrow
}
func sidebarSymbolButton(c *ui.Context, label string) ui.Element {
	return ui.ButtonBase(c).Size(28, 28).Radius(6).Label(label).Children(func() { ui.Icon(c, sidebarPlus).Size(16, 16) })
}
