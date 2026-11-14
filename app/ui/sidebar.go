package ui

import (
	"context"
	"slices"

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
	LibraryOpen       bool
	OrganizeCollapsed bool
	RecentList        ui.ListState
	Open              map[string]*bool
}

func (a *App) sidebar(c *ui.Context) {
	if a.Sidebar.Open == nil {
		a.Sidebar.Open = make(map[string]*bool)
	}
	counts := catalog.Counts{}
	if a.Index != nil {
		counts = a.Index.Counts()
	}
	side := ui.Column(c).Fill().Padding(0, 12, 12, 12).Gap(0).FontSize(13)
	sidebarMaterial(c, side)
	side.Children(func() {
		a.sidebarTopBar(c)
		ui.Scroll(c.Key("sidebar_scopes")).Grow(1).Gap(1).Children(func() {
			a.sidebarScope(c, "all", "all", "照片", counts.All)
			if !a.sidebarHidden("recent") && a.Index != nil {
				a.sidebarScope(c, "recent", "recent", "recent", len(a.Index.Query(catalog.Query{Scope: "recent"})))
			}
			if !a.sidebarHidden("favorites") {
				a.sidebarScope(c, "favorites", "favorites", "喜欢", counts.Favorites)
			}
			if !a.sidebarHidden("random") {
				a.sidebarScope(c, "random", "random_browse", "随机浏览", -1)
			}
			a.sidebarQuickAccess(c)
			organize := ui.ButtonBase(c.Key("organize")).FillWidth().Height(28).Margin(19, 0, 0, 0).Padding(0, 10).Gap(8).Radius(8).Cursor(ui.CursorPointer).Label(a.T("organize")).TextColor(c.Theme().TextMuted).Expanded(!a.Sidebar.OrganizeCollapsed)
			if organize.Hovered() {
				organize.Background(warmOverlay(c.Theme(), .06)).TextColor(c.Theme().Text)
			}
			organize.Children(func() {
				ui.Text(c, a.T("organize")).FontSize(11).Bold().LetterSpacing(.88).TextColor(c.Theme().TextMuted).Grow(1)
				sidebarChevron(c, !a.Sidebar.OrganizeCollapsed).Size(13, 13)
			})
			if organize.Clicked() {
				a.Sidebar.OrganizeCollapsed = !a.Sidebar.OrganizeCollapsed
			}
			if !a.Sidebar.OrganizeCollapsed {
				if !a.sidebarHidden("unfiled") {
					a.sidebarScope(c, "uncategorized", "uncategorized", "unfiled", counts.Uncategorized)
				}
				if !a.sidebarHidden("untagged") {
					a.sidebarScope(c, "untagged", "untagged", "untagged", counts.Untagged)
				}
				if a.sidebarRow(c.Key("tags_page"), a.T("tag_management"), "标签组", len(a.Snapshot.Tags), a.TagPage).Clicked() {
					a.ShowTagsPage()
				}
			}
			if !a.sidebarHidden("folders") {
				a.sidebarSectionHeader(c, a.T("folders"), a.T("new_folder"), 20, func() { a.sidebarCreateFolder("") })
				if len(a.Snapshot.Folders) == 0 {
					if sidebarEmptyAction(c.Key("folders-empty"), a.T("new_folder"), "文件夹").Disabled(a.Busy || a.Library == nil).Clicked() {
						a.sidebarCreateFolder("")
					}
				} else {
					a.sidebarFolders(c, counts)
				}
			}
			a.sidebarSmartFolders(c)
		})
		ui.Divider(c).Margin(10, 0)
		trash := a.sidebarScope(c, "trash", "trash", "回收站", counts.Trash)
		trash.ContextMenu(func(m *ui.Menu) {
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

func (a *App) sidebarTopBar(c *ui.Context) {
	bar := c.TitleBar()
	height := float32(MainToolbarHeight)
	if bar.Left > 0 {
		height = max(height, bar.Bottom+8)
	}
	ui.Row(c.Key("sidebar-titlebar")).Height(height).FillWidth().Shrink(0).Gap(8).DragWindow().Children(func() {
		if bar.Left > 0 {
			ui.Box(c).Width(max(0, bar.Left-12)).FillHeight()
			ui.Box(c).Grow(1)
		} else {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.sidebarLibrary(c) })
		}
		button := ui.ButtonBase(c.Key("sidebar-toggle")).Size(28, 28).Padding(0).Radius(7).Cursor(ui.CursorPointer).Label(a.T("toggle_sidebar")).Tooltip(a.T("toggle_sidebar")).Children(func() {
			ui.Image(c, sidebarIcon("sidebar")).Size(20, 20)
		})
		if button.Hovered() {
			button.Background(warmOverlay(c.Theme(), .08))
		}
		if button.Clicked() {
			a.SidebarVisible = !a.SidebarVisible
		}
	})
	if bar.Left > 0 {
		a.sidebarLibrary(c)
	}
	ui.Box(c).Height(14).FillWidth().Shrink(0)
}

func (a *App) sidebarSectionHeader(c *ui.Context, title, action string, top float32, onAdd func()) {
	row := ui.Row(c.Key(title+"-section")).Height(28).Margin(top, 0, 1, 0).Padding(0, 2, 0, 10).FillWidth().AlignItems(ui.Center)
	row.Children(func() {
		label := ui.Text(c, title).FontSize(11).Bold().LetterSpacing(.88).TextColor(c.Theme().TextMuted).Grow(1)
		if title == a.T("folders") {
			a.sidebarRootDrop(c, row)
			label.Label(a.T("move_to_root"))
		}
		if sidebarSymbolButton(c, action).Disabled(a.Busy || a.Library == nil).Clicked() {
			onAdd()
		}
	})
}

func sidebarEmptyAction(c *ui.Context, label, icon string) ui.Element {
	row := ui.ButtonBase(c).FillWidth().Height(32).Padding(0, 10).Gap(8).Radius(8).Border(1, warmOverlay(c.Theme(), .24)).BorderStyle(ui.BorderDashed).Cursor(ui.CursorPointer).Label(label).Justify(ui.Start).TextColor(c.Theme().TextMuted)
	if row.Hovered() {
		row.Background(warmOverlay(c.Theme(), .06))
	}
	row.Children(func() {
		ui.Image(c, sidebarIcon(icon)).Size(20, 20).Shrink(0)
		ui.Text(c, label).FontSize(13).SingleLine().Grow(1).MinWidth(0)
	})
	return row
}

func (a *App) sidebarRow(c *ui.Context, label, icon string, count int, selected bool) ui.Element {
	row := ui.ButtonBase(c).FillWidth().Height(32).Padding(0, 10).Gap(10).Radius(8).Cursor(ui.CursorPointer).Label(label).AccessSelected(selected).Justify(ui.Start).Border(1, ui.Color{})
	if selected {
		sidebarSelectedRow(c, row)
	} else if row.Hovered() {
		row.Background(warmOverlay(c.Theme(), .06))
	}
	row.Children(func() {
		if icon != "" {
			ui.Image(c, sidebarIcon(icon)).Size(20, 20).Shrink(0)
		}
		ui.Text(c, label).SingleLine().Grow(1).MinWidth(0)
		if count >= 0 && a.Preferences.SidebarCounts {
			color := c.Theme().TextMuted
			ui.Text(c, sidebarCount(count)).FontSize(11.5).Font("monospace").FontFeatures("tnum").FontWeight(400).TextColor(color).Shrink(0)
		}
	})
	return row
}

func sidebarSelectedRow(c *ui.Context, row ui.Element) {
	color := ui.Hex("#FFFDFA")
	if c.Theme().Dark {
		color = c.Theme().SurfaceHover
	}
	row.Background(color).Border(1, warmOverlay(c.Theme(), .13)).TextColor(titleColor(c.Theme())).Bold()
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
	trigger := ui.ButtonBase(c.Key("library_switcher")).FillWidth().Height(38).Padding(0, 10, 0, 6).Gap(10).Radius(10).Border(1, warmOverlay(c.Theme(), .14)).Background(c.Theme().Background.Alpha(.55)).Cursor(ui.CursorPointer).Label(a.T("library_menu")).Justify(ui.Start).TextColor(c.Theme().TextMuted)
	if trigger.Hovered() {
		trigger.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	}
	trigger.Children(func() {
		ui.Image(c, appLogo).Size(26, 26).Radius(7).Shrink(0)
		ui.Text(c, name).TextColor(titleColor(c.Theme())).FontSize(13.5).Bold().Grow(1).MinWidth(0).SingleLine()
		appSymbol(c, "chevrons-up-down").Size(15, 15).Shrink(0)
	})
	if trigger.Clicked() {
		a.Sidebar.LibraryOpen = !a.Sidebar.LibraryOpen
	}
	frostedPopover(c, trigger, &a.Sidebar.LibraryOpen, func() {
		_, windowHeight := c.Size()
		menuHeight := float32(310)
		if len(a.Recent) > 0 {
			menuHeight += float32(min(10, len(a.Recent))*32 + 32)
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
				}).Height(float32(min(10, len(a.Recent)) * 32)).MaxHeight(320)
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
			row := item.Row.Height(32).Padding(0, 10).PaddingStart(6+float32(item.Depth())*16).Gap(6).Radius(8).Cursor(ui.CursorPointer).Border(1, ui.Color{}).Label(folder.Name).AccessSelected(!a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID)
			if !a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID {
				sidebarSelectedRow(c, row)
			} else if row.Hovered() {
				row.Background(warmOverlay(c.Theme(), .06))
			}
			row.Drag(AssetsDrag{LibraryID: a.libraryID(), FolderID: folder.ID})
			row.Children(func() {
				if opened != nil {
					arrow := ui.ButtonBase(c.Key("disclose")).Size(16, 20).Padding(0).Cursor(ui.CursorPointer).TextColor(c.Theme().TextMuted).Label(a.T("toggle_folder", folder.Name))
					if arrow.Hovered() {
						arrow.TextColor(c.Theme().Text)
					}
					arrow.Children(func() { sidebarChevron(c, *opened).Size(13, 13) })
					if arrow.Clicked() {
						*opened = !*opened
					}
				} else {
					ui.Box(c).Size(16, 20)
				}
				ui.Image(c, sidebarIcon("文件夹")).Size(20, 20).Shrink(0)
				ui.Text(c, folder.Name).SingleLine().Grow(1).MinWidth(0)
				color := c.Theme().TextMuted
				if a.Preferences.SidebarCounts {
					ui.Text(c, sidebarCount(counts.Folders[folder.ID])).FontSize(11.5).Font("monospace").FontFeatures("tnum").FontWeight(400).TextColor(color).Shrink(0)
				}
			})
			if row.Clicked() {
				a.SetScope("folder", folder.ID)
			}
			if row.DoubleClicked() {
				if a.Preferences.FolderDoubleClick == "rename" {
					a.renameSidebarFolder(folder)
				} else if opened != nil {
					*opened = !*opened
				}
			}
			a.sidebarFolderDrop(c, row, folder)
			row.ContextMenu(func(m *ui.Menu) {
				a.QuickAccessMenu(m, "folder", folder.ID)
				m.Separator()
				if m.Item(a.T("new_subfolder")).Disabled(a.Busy).Chosen() {
					*open = true
					a.sidebarCreateFolder(folder.ID)
				}
				if m.Item(a.T("rename_folder")).Disabled(a.Busy).Chosen() {
					a.renameSidebarFolder(folder)
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
	position := sidebarDropPosition(y, 32)
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

func sidebarChevron(c *ui.Context, open bool) ui.Element {
	arrow := appSymbol(c, "chevron-right").Size(13, 13)
	if open {
		arrow.Rotate(90)
	}
	return arrow
}
func sidebarSymbolButton(c *ui.Context, label string) ui.Element {
	button := ui.ButtonBase(c).Size(24, 24).Padding(0).Radius(6).Cursor(ui.CursorPointer).Label(label).Tooltip(label).TextColor(c.Theme().TextMuted).Children(func() { appSymbol(c, "plus").Size(14, 14) })
	if button.Hovered() {
		button.Background(warmOverlay(c.Theme(), .08)).TextColor(c.Theme().Text)
	}
	return button
}

func (a *App) sidebarHidden(key string) bool {
	return slices.Contains(a.Preferences.SidebarHidden, key)
}
func (a *App) renameSidebarFolder(folder model.Folder) {
	if a.Busy || a.Library == nil {
		return
	}
	lib := a.Library
	a.Ask(a.T("rename_folder"), folder.Name, func(name string) {
		a.Run("rename_folder", func(ctx context.Context) error { return lib.RenameFolder(ctx, folder.ID, name) })
	})
}
