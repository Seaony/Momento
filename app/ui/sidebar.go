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
	LibraryOpen bool
	RecentList  ui.ListState
	LibraryMore map[string]*bool // 每个最近图库行尾「…」菜单的开关，按路径区分
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
	width := a.SidebarWidth
	if width == 0 {
		width = 236
	}
	// 结构照旧版 macOS：图库 → 全部 → 分隔线 → 其余入口 → 分隔线 → 文件夹；底部只有三个图标按钮。
	ui.Box(c.Key("sidebar-clip")).Fill().ClipX().Children(func() {
		side := ui.Column(c.Key("sidebar-content")).Width(width).FillHeight().Shrink(0).Gap(0).FontSize(fontBody).FontWeight(sidebarWeight)
		sidebarMaterial(c, side)
		side.Children(func() {
			ui.Column(c).FillWidth().Padding(0, sidebarPadding).Shrink(0).Children(func() { a.sidebarTopBar(c) })
			ui.Scroll(c.Key("sidebar_scopes")).Grow(1).Padding(0, sidebarPadding, sidebarSectionGap, sidebarPadding).Gap(sidebarRowGap).Children(func() {
				a.sidebarScope(c, "all", "all", "照片", counts.All)
				sidebarDivider(c, sidebarDividerSpace-sidebarRowGap)
				if !a.sidebarHidden("recent") && a.Index != nil {
					a.sidebarScope(c, "recent", "recent", "recent", len(a.Index.Query(catalog.Query{Scope: "recent"})))
				}
				if !a.sidebarHidden("favorites") {
					a.sidebarScope(c, "favorites", "favorites", "喜欢", counts.Favorites)
				}
				if !a.sidebarHidden("random") {
					a.sidebarScope(c, "random", "random_browse", "随机浏览", -1)
				}
				if !a.sidebarHidden("unfiled") {
					a.sidebarScope(c, "uncategorized", "uncategorized", "unfiled", counts.Uncategorized)
				}
				if !a.sidebarHidden("untagged") {
					a.sidebarScope(c, "untagged", "untagged", "untagged", counts.Untagged)
				}
				if a.sidebarRow(c.Key("tags_page"), a.T("tag_management"), "标签组", len(a.Snapshot.Tags), a.TagPage).Clicked() {
					a.ShowTagsPage()
				}
				a.sidebarQuickAccess(c)
				if !a.sidebarHidden("folders") {
					sidebarDivider(c, sidebarGroupGap+sidebarDividerSpace-sidebarRowGap)
					// 文件夹不再单独分组，直接平铺；最下面一行新建，也是拖到根目录的落点。
					if len(a.Snapshot.Folders) > 0 {
						a.sidebarFolders(c, counts)
					}
					add := sidebarAddRow(c.Key("folders-add"), a.T("new_folder"), "create-library").Disabled(a.Busy || a.Library == nil)
					a.sidebarRootDrop(c, add)
					if add.Clicked() {
						a.sidebarCreateFolder("")
					}
				}
				a.sidebarSmartFolders(c)
			})
			ui.Divider(c).Margin(0, sidebarPadding).Background(paletteFor(c.Theme()).BorderSubtle)
			a.sidebarFooter(c, counts.Trash)
		})
	})
}

// 旧版的分隔线：很淡的细线，左右各缩进一点。
func sidebarDivider(c *ui.Context, space float32) {
	ui.Divider(c).Margin(space, sidebarDividerInset).Background(paletteFor(c.Theme()).BorderSubtle)
}

func (a *App) sidebarFooter(c *ui.Context, trashCount int) {
	ui.Row(c.Key("sidebar-footer")).FillWidth().Padding(sidebarFooterPadding, space12).Gap(sidebarFooterGap).Shrink(0).Children(func() {
		selected := !a.TagPage && a.Query.Scope == "trash"
		trash := sidebarFooterButton(c.Key("scope_trash"), a.T("trash"), "trash", selected)
		trash.AccessSelected(selected)
		if trashCount > 0 && a.Preferences.SidebarCounts {
			// 旧版：回收站数量是右上角的小角标。
			trash.Children(func() {
				ui.Text(c, sidebarCount(trashCount)).FontSize(sidebarBadgeSize).FontWeight(700).FontFeatures("tnum").TextColor(paletteFor(c.Theme()).Text).
					Absolute().Top(-space4).Right(-space4).MinWidth(11).Height(11).Padding(0, 3).Radius(radiusPanel).Background(paletteFor(c.Theme()).Hover).TextAlign(ui.Center).IgnoreInput(true)
			})
		}
		if trash.Clicked() {
			a.SetScope("trash", "")
		}
		trash.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.T("empty_trash")).Disabled(trashCount == 0 || a.Busy).Chosen() {
				a.Confirm(a.T("empty_trash"), a.T("empty_trash_detail"), a.T("clear"), func() {
					service := a.Assets
					a.Run("empty_trash", func(ctx context.Context) error { return service.EmptyTrash(ctx) })
				})
			}
		})
		if sidebarFooterButton(c.Key("settings"), a.T("settings"), "settings", false).Clicked() {
			a.ShowSettings()
		}
		if sidebarFooterButton(c.Key("help"), a.T("help"), "help", false).Clicked() {
			a.OpenHelp()
		}
	})
}

func sidebarFooterButton(c *ui.Context, label, icon string, selected bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Size(sidebarFooterButtonSize, sidebarFooterButtonSize).Shrink(0).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer).Label(label).Tooltip(label)
	background, ink := ui.Transparent, p.TextSecondary
	if button.Pressed() {
		background, ink = p.Pressed, p.Text
	} else if selected || button.Hovered() {
		background, ink = p.Hover, p.Text
	}
	animateColors(button, background, ink)
	button.Children(func() { appIcon(c, icon).Size(sidebarIconSize, sidebarIconSize) })
	return button
}

func (a *App) sidebarTopBar(c *ui.Context) {
	bar := c.TitleBar()
	height := float32(MainToolbarHeight)
	if bar.Left > 0 {
		height = max(height, bar.Bottom+8)
	}
	ui.Row(c.Key("sidebar-titlebar")).Height(height).FillWidth().Shrink(0).Gap(space8).DragWindow().Children(func() {
		if bar.Left > 0 {
			ui.Box(c).Width(max(0, bar.Left-12)).FillHeight()
			ui.Box(c).Grow(1)
		} else {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.sidebarLibrary(c) })
		}
		button := ui.ButtonBase(c.Key("sidebar-toggle")).Size(controlSmall, controlSmall).Padding(0).Radius(radiusCard).Cursor(ui.CursorPointer).Label(a.T("toggle_sidebar")).Tooltip(a.T("toggle_sidebar")).Children(func() {
			appIcon(c, "sidebar").Size(iconSize, iconSize)
		})
		sidebarControlColors(c, button, true)
		if button.Clicked() {
			a.SidebarVisible = !a.SidebarVisible
		}
	})
	if bar.Left > 0 {
		a.sidebarLibrary(c)
	}
	ui.Box(c).Height(sidebarLibraryGap).FillWidth().Shrink(0)
}

// 列表最后的「新建」行：表单同款浅底加虚线框，文字和图标与其他行同色，尺寸和图标位置也一样。
func sidebarAddRow(c *ui.Context, label, icon string) ui.Element {
	p := paletteFor(c.Theme())
	row := ui.ButtonBase(c).FillWidth().Height(sidebarRowHeight).Shrink(0).Padding(0, sidebarRowPadding).Gap(sidebarIconGap).Radius(sidebarRowRadius).
		Border(1, p.BorderSubtle).BorderStyle(ui.BorderDashed).Cursor(ui.CursorPointer).Label(label).Justify(ui.Start)
	background, ink := p.Field, p.Text.Alpha(.72)
	if row.Pressed() {
		background, ink = p.Pressed, p.Text
	} else if row.Hovered() {
		background, ink = p.FieldHover, p.Text
	}
	animateColors(row, background, ink)
	row.Children(func() {
		ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			appIcon(c, icon).Size(sidebarIconSize, sidebarIconSize)
		})
		ui.Text(c, label).SingleLine().Grow(1).MinWidth(0)
	})
	return row
}

func (a *App) sidebarRow(c *ui.Context, label, icon string, count int, selected bool) ui.Element {
	row := ui.ButtonBase(c).FillWidth().Height(sidebarRowHeight).Padding(0, sidebarRowPadding).Gap(sidebarIconGap).Radius(sidebarRowRadius).Cursor(ui.CursorPointer).Label(label).AccessSelected(selected).Justify(ui.Start)
	ink := sidebarRowStyle(c, row, selected)
	row.Children(func() {
		if icon != "" {
			ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				appIcon(c, icon).Size(sidebarIconSize, sidebarIconSize)
			})
		}
		ui.Text(c, label).SingleLine().Grow(1).MinWidth(0)
		if a.Preferences.SidebarCounts {
			sidebarCountText(c, count, ink)
		}
	})
	return row
}

// 旧版的数量：11 号半粗、等宽数字，颜色是本行文字色的 58%；0 不显示。
func sidebarCountText(c *ui.Context, count int, ink ui.Color) {
	if count <= 0 {
		return
	}
	ui.Text(c, sidebarCount(count)).FontSize(sidebarCountSize).FontWeight(600).FontFeatures("tnum").TextColor(ink.Alpha(.58)).MinWidth(space16).TextAlign(ui.End).Shrink(0)
}

// 旧版的行：平时文字和图标是正文色的 72%，悬停或选中时用正文色并铺一层浅底。返回本行文字色。
func sidebarRowStyle(c *ui.Context, row ui.Element, selected bool) ui.Color {
	p := paletteFor(c.Theme())
	background, ink := ui.Transparent, p.Text.Alpha(.72)
	if row.Pressed() {
		background, ink = p.Pressed, p.Text
	} else if selected || row.Hovered() {
		background, ink = p.Hover, p.Text
	}
	animateColors(row, background, ink)
	return ink
}

func sidebarControlColors(c *ui.Context, control ui.Element, muted bool) {
	background := ui.Color{}
	text := paletteFor(c.Theme()).Text
	if muted {
		text = paletteFor(c.Theme()).TextSecondary
	}
	if control.Pressed() {
		background, text = paletteFor(c.Theme()).Pressed, paletteFor(c.Theme()).Text
	} else if control.Hovered() {
		background, text = paletteFor(c.Theme()).Hover, paletteFor(c.Theme()).Text
	}
	animateColors(control, background, text)
}

func sidebarRowIcon(c *ui.Context, name string, selected, hovered bool) ui.Element {
	icon := appIconSelected(c, name, selected)
	color := paletteFor(c.Theme()).TextSecondary
	if selected {
		color = paletteFor(c.Theme()).Accent
	} else if hovered {
		color = paletteFor(c.Theme()).Text
	}
	return icon.TextColor(icon.AnimateColor("ink", color, motionColor))
}

func (a *App) sidebarScope(c *ui.Context, scope, key, icon string, count int) ui.Element {
	selected := !a.TagPage && (a.Query.Scope == scope || (scope == "all" && a.Query.Scope == ""))
	row := a.sidebarRow(c.Key("scope_"+scope), a.T(key), icon, count, selected)
	if row.Clicked() {
		a.SetScope(scope, "")
	}
	return row
}

func (a *App) sidebarFolders(c *ui.Context, counts catalog.Counts) {
	children := sidebarFolderChildren(a.Snapshot.Folders)
	tree := ui.TreeBase(c.Key("folder_tree"))
	tree.Root.Gap(sidebarRowGap).ClipY().Transition(ui.ElementTransition{Animation: &motionSpring, Height: true}).Children(func() {
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
			row := item.Row.Height(sidebarRowHeight).Padding(0, sidebarRowPadding, 0, float32(item.Depth())*sidebarFolderIndent).Gap(0).Radius(sidebarRowRadius).Cursor(ui.CursorPointer).Label(folder.Name).AccessSelected(!a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID)
			row.Transition(ui.ElementTransition{Animation: &motionSpring, Position: true})
			selected := !a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID
			ink := sidebarRowStyle(c, row, selected)
			row.Drag(AssetsDrag{LibraryID: a.libraryID(), FolderID: folder.ID})
			row.Children(func() {
				if opened != nil {
					arrow := ui.ButtonBase(c.Key("disclose")).Size(sidebarRowPadding, sidebarRowHeight).Shrink(0).Padding(0).Cursor(ui.CursorPointer).Justify(ui.Center).AlignItems(ui.Center).Label(a.T("toggle_folder", folder.Name))
					arrow.Children(func() { sidebarChevron(c, *opened).Size(iconTiny, iconTiny).Shrink(0) })
					if arrow.Clicked() {
						*opened = !*opened
					}
				} else {
					ui.Box(c).Size(sidebarRowPadding, sidebarRowHeight).Shrink(0)
				}
				ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
					appIcon(c, "文件夹").Size(sidebarIconSize, sidebarIconSize)
				})
				ui.Text(c, folder.Name).SingleLine().Grow(1).MinWidth(0).Margin(0, 0, 0, sidebarIconGap)
				if a.Preferences.SidebarCounts {
					ui.Box(c).Width(sidebarCountGap).Shrink(0)
					sidebarCountText(c, counts.Folders[folder.ID], ink)
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
					a.Confirm(a.T("delete_folder"), a.T("delete_folder_detail", folder.Name), a.T("delete"), func() {
						lib := a.Library
						if a.Query.Scope == "folder" && sidebarFolderContains(a.Snapshot.Folders, folder.ID, a.Query.FolderID) {
							a.SetScope("all", "")
						}
						a.Run("delete_folder", func(ctx context.Context) error { return lib.DeleteFolder(ctx, folder.ID) })
					})
				}
			})
			group := item.Group(func() {
				for _, child := range children[folder.ID] {
					build(child)
				}
			})
			if group.Valid() {
				group.ClipY().Transition(ui.ElementTransition{Animation: &motionSpring, Height: true, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}})
			}
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
		row.TextColor(paletteFor(c.Theme()).Text)
	}
	if drag, ok := ui.Drop[AssetsDrag](row); ok {
		a.sidebarMoveFolder(drag, "", folderInside)
	}
}
func (a *App) sidebarFolderDrop(c *ui.Context, row ui.Element, folder model.Folder) {
	_, y, _ := row.PointerPosition()
	position := sidebarDropPosition(y, sidebarRowHeight)
	if drag, over := ui.DragOver[AssetsDrag](row); over && a.sidebarCanDrop(drag) {
		if drag.FolderID != "" {
			if _, _, valid := sidebarFolderMove(a.Snapshot.Folders, drag.FolderID, folder.ID, position); valid {
				switch position {
				case folderBefore:
					row.BorderWidth(2, 0, 0, 0).BorderColor(paletteFor(c.Theme()).Border)
				case folderAfter:
					row.BorderWidth(0, 0, 2, 0).BorderColor(paletteFor(c.Theme()).Border)
				default:
					row.Background(paletteFor(c.Theme()).Selected).TextColor(paletteFor(c.Theme()).Text)
				}
			}
		} else if len(drag.IDs) > 0 {
			row.Background(paletteFor(c.Theme()).Selected).TextColor(paletteFor(c.Theme()).Text)
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
	arrow := appSymbol(c, "chevron-right").Size(iconSmall, iconSmall)
	angle := float32(0)
	if open {
		angle = 90
	}
	arrow.Rotate(arrow.AnimateTo("rotation", angle, motionSpring))
	return arrow
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
