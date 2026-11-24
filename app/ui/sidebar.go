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
	FolderID  string
	// 拖素材时才有。放下时才取 ID：全选十万张时，不能每帧给每张卡片都复制一遍选中项。
	assets func() []string
}

// 拖的是不是素材；悬停时只用它判断，不取 ID。
func (d AssetsDrag) HasAssets() bool { return d.assets != nil }

func (d AssetsDrag) IDs() []string {
	if d.assets == nil {
		return nil
	}
	return d.assets()
}

type sidebarState struct {
	LibraryOpen bool
	RecentList  ui.ListState
	LibraryMore map[string]*bool // 每个最近图库行尾「…」菜单的开关，按路径区分
	Open        map[string]*bool
	smartExt    smartExtensionCache
}

func (a *App) sidebar(c *ui.Context) {
	a.gBrowsingCounts()
	counts := a.GBrowsing.Counts
	width := a.SidebarWidth
	if width == 0 {
		width = 236
	}
	// 结构照旧版 macOS：图库 → 全部 → 分隔线 → 其余入口 → 分隔线 → 文件夹；底部只有三个图标按钮。
	ui.Box(c.Key("sidebar-clip")).Fill().ClipX().Children(func() {
		side := ui.Column(c.Key("sidebar-content")).Width(width).FillHeight().Shrink(0).Gap(0).FontSize(fontBody).FontWeight(sidebarWeight)
		side.Background(paletteFor(c.Theme()).Sidebar)
		side.Children(func() {
			ui.Column(c).FillWidth().Padding(0, sidebarPadding).Shrink(0).Children(func() { a.sidebarTopBar(c) })
			ui.Scroll(c.Key("sidebar_scopes")).Scrollbars(false).Grow(1).Padding(0, sidebarPadding, sidebarSectionGap, sidebarPadding).Gap(sidebarRowGap).Children(func() {
				a.sidebarScope(c, "all", "all", "照片", counts.All)
				if !a.sidebarHidden("random") {
					a.sidebarScope(c, "random", "random_browse", "随机浏览", -1)
				}
				sidebarDivider(c, sidebarDividerSpace-sidebarRowGap)
				if !a.sidebarHidden("favorites") {
					a.sidebarScope(c, "favorites", "favorites", "喜欢", counts.Favorites)
				}
				// 最近 30 天预览、打开、复制过的素材，按使用时间排。
				if !a.sidebarHidden("recent") {
					a.sidebarScope(c, "recent", "recently_used", "recent", -1)
				}
				if !a.sidebarHidden("unfiled") {
					a.sidebarScope(c, "uncategorized", "uncategorized", "unfiled", counts.Uncategorized)
				}
				if !a.sidebarHidden("untagged") {
					a.sidebarScope(c, "untagged", "untagged", "untagged", counts.Untagged)
				}
				if a.sidebarRow(c.Key("tags_page"), a.T("tag_management"), "标签组", -1, a.TagPage).Clicked() {
					a.ShowTagsPage()
				}
				a.sidebarQuickAccess(c)
				if !a.sidebarHidden("folders") {
					sidebarDivider(c, sidebarGroupGap+sidebarDividerSpace-sidebarRowGap)
					// 文件夹不再单独分组，直接平铺；最下面一行新建，也是拖到根目录的落点。
					if len(a.Snapshot.Folders) > 0 {
						a.sidebarFolders(c, counts)
					}
					add := sidebarAddRow(c.Key("folders-add"), a.T("new_folder"), "create-library").Disabled(a.Library == nil)
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
				badge := ui.Text(c.Key("trash-count"), sidebarCount(trashCount)).FontSize(sidebarBadgeSize).FontWeight(700).FontFeatures("tnum").TextColor(paletteFor(c.Theme()).Text).
					Absolute().Top(-space4).Right(-space4).MinWidth(11).Radius(radiusPanel).Background(paletteFor(c.Theme()).Hover).TextAlign(ui.Center).IgnoreInput(true)
				// 删进回收站时数字往上滚，清空时角标淡出。
				fadeMotion(countMotion(centeredTextHeight(badge, 11, 3), trashCount))
			})
		}
		if trash.Clicked() {
			a.SetScope("trash", "")
		}
		// 回收站是空的，右键菜单里没有可用项，就不弹。
		if trashCount > 0 {
			trash.ContextMenu(func(m *ui.Menu) {
				if menuItem(m, a.T("empty_trash"), "trash").Chosen() {
					a.ConfirmEmptyTrash()
				}
			})
		}
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
	background, ink := ui.Transparent, p.Text.Alpha(sidebarIdleInk)
	if button.Pressed() {
		background, ink = p.Pressed, p.Text
	} else if selected || button.Hovered() {
		background, ink = p.Hover, p.Text
	}
	animateColors(button, background, ink)
	button.Children(func() { sidebarAppIcon(c, icon) })
	return button
}

func (a *App) sidebarTopBar(c *ui.Context) {
	bar := c.TitleBar()
	// 和工具栏同高：红绿灯、侧栏收起按钮、工具栏控件的中心在同一条线上。
	height := float32(MainToolbarHeight)
	ui.Row(c.Key("sidebar-titlebar")).Height(height).FillWidth().Shrink(0).Gap(space8).DragWindow().Children(func() {
		if bar.Left > 0 {
			ui.Box(c).Width(max(0, bar.Left-12)).FillHeight()
			ui.Box(c).Grow(1)
		} else {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.sidebarLibrary(c) })
		}
		// 收起按钮比侧栏图标小一号，平时更淡，悬停才变成正文色。
		button := ui.ButtonBase(c.Key("sidebar-toggle")).Size(sidebarToggleSize, sidebarToggleSize).Padding(0).Radius(radiusCard).Cursor(ui.CursorPointer).Label(a.T("collapse_sidebar")).Tooltip(a.T("collapse_sidebar")).Children(func() {
			sidebarAppIcon(c, "sidebar").Size(iconMedium, iconMedium)
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
	background, ink := p.Field, p.Text.Alpha(sidebarIdleInk)
	if row.Pressed() {
		background, ink = p.Pressed, p.Text
	} else if row.Hovered() {
		background, ink = p.FieldHover, p.Text
	}
	animateColors(row, background, ink)
	row.Children(func() {
		ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			sidebarAppIcon(c, icon)
		})
		ui.Text(c, label).SingleLine().Grow(1).MinWidth(0)
	})
	return row
}

func (a *App) sidebarRow(c *ui.Context, label, icon string, count int, selected bool) ui.Element {
	return a.sidebarRowDrop(c, label, icon, count, selected, nil)
}

// dropping 不为空时，素材拖到这一行上方（放下就进这一行）它返回 true，这一行用和选中一样的底色提示。
func (a *App) sidebarRowDrop(c *ui.Context, label, icon string, count int, selected bool, dropping func(ui.Element) bool) ui.Element {
	row := ui.ButtonBase(c).FillWidth().Height(sidebarRowHeight).Padding(0, sidebarRowPadding).Gap(sidebarIconGap).Radius(sidebarRowRadius).Cursor(ui.CursorPointer).Label(label).AccessSelected(selected).Justify(ui.Start)
	ink := sidebarRowStyle(c, row, selected, dropping != nil && dropping(row))
	row.Children(func() {
		if icon != "" {
			sidebarNavIcon(c, icon, selected, ink)
		}
		ui.Text(c, label).SingleLine().Grow(1).MinWidth(0)
		if a.Preferences.SidebarCounts {
			sidebarCountText(c, count, ink)
		}
	})
	return row
}

// 旧版的数量：11 号半粗、等宽数字，颜色是本行文字色的 58%；0 不显示。
// 数量变了按增减方向滚动，从 0 变成有或变回 0 时淡入淡出。
func sidebarCountText(c *ui.Context, count int, ink ui.Color) {
	if count <= 0 {
		return
	}
	text := ui.Text(c.Key("count"), sidebarCount(count)).FontSize(sidebarCountSize).FontWeight(600).FontFeatures("tnum").TextColor(ink.Alpha(sidebarCountInk)).MinWidth(space16).TextAlign(ui.End).Shrink(0)
	fadeMotion(countMotion(text, count))
}

// 旧版的行：平时文字和图标是正文色的 72%，悬停或选中时用正文色并铺一层浅底。返回本行文字色。
// dropping 表示素材或文件夹正拖到这一行上方、放下就进这个文件夹，和选中同一个底色。
func sidebarRowStyle(c *ui.Context, row ui.Element, selected, dropping bool) ui.Color {
	p := paletteFor(c.Theme())
	background, ink := ui.Transparent, p.Text.Alpha(sidebarIdleInk)
	// 每行都留一圈边框，选中才显出来，文字不会被挤动；边框和浮起的轻影跟底色一起渐变，不在换选中时突然冒出来。
	border, shadow := ui.Transparent, ui.Transparent
	if selected {
		// 选中项保留白色浮起底，比旧版的浅灰底更醒目。
		background, ink, border, shadow = p.Selected, p.Text, p.BorderSubtle, p.Shadow
		row.FontWeight(sidebarSelectedWeight)
	}
	switch {
	case dropping:
		background, ink = p.Selected, p.Text
	case row.Pressed():
		background, ink = p.Pressed, p.Text
	case row.Hovered() && !selected:
		background, ink = p.Hover, p.Text
	}
	row.Border(1, row.AnimateColor("selection-border", border, motionColor)).
		Shadow(0, 1, 2, 0, row.AnimateColor("selection-shadow", shadow, motionColor))
	animateColors(row, background, ink)
	return ink
}

// 侧栏行图标：选中时用品牌色，其余跟随本行文字色；图形和粗细不随选中变化，颜色渐变。
func sidebarNavIcon(c *ui.Context, name string, selected bool, ink ui.Color) {
	ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		sidebarIconInk(c, sidebarAppIcon(c, name), selected, ink)
	})
}

// 文件夹行的图标：没选图标时用默认文件夹图标；选了图标的用它的加粗线性版；选中时都上品牌色。
func sidebarFolderIcon(c *ui.Context, icon string, selected bool, ink ui.Color) {
	if folderIconID(icon) == "folder" {
		sidebarNavIcon(c, "文件夹", selected, ink)
		return
	}
	ui.Box(c).Size(sidebarIconBox, sidebarIconBox).Shrink(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		sidebarIconInk(c, folderSidebarIcon(c, icon).Size(sidebarIconSize, sidebarIconSize), selected, ink)
	})
}

func sidebarIconInk(c *ui.Context, icon ui.Element, selected bool, ink ui.Color) {
	if selected {
		ink = paletteFor(c.Theme()).Accent
	}
	icon.TextColor(icon.AnimateColor("ink", ink, motionColor))
}

func sidebarControlColors(c *ui.Context, control ui.Element, muted bool) {
	background := ui.Color{}
	text := paletteFor(c.Theme()).Text
	if muted {
		text = paletteFor(c.Theme()).Text.Alpha(sidebarToggleInk)
	}
	if control.Pressed() {
		background, text = paletteFor(c.Theme()).Pressed, paletteFor(c.Theme()).Text
	} else if control.Hovered() {
		background, text = paletteFor(c.Theme()).Hover, paletteFor(c.Theme()).Text
	}
	animateColors(control, background, text)
}

// 给行里的图标上色：选中品牌色，悬停正文色，平时次要文字色。
func sidebarRowIconColor(c *ui.Context, icon ui.Element, selected, hovered bool) ui.Element {
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
			open := a.sidebarOpenFlag(folder.ID)
			var opened *bool
			if len(children[folder.ID]) > 0 {
				opened = open
			}
			c.Key(folder.ID)
			item := tree.Item(opened)
			row := item.Row.Height(sidebarRowHeight).Padding(0, sidebarRowPadding, 0, float32(item.Depth())*sidebarFolderIndent).Gap(0).Radius(sidebarRowRadius).Cursor(ui.CursorPointer).Label(folder.Name).AccessSelected(!a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID)
			// 新建的文件夹在它的位置展开出来、删掉的收起，其他行滑开让位。
			rowMotion(row)
			selected := !a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == folder.ID
			ink := sidebarRowStyle(c, row, selected, a.sidebarDropInside(row, folder))
			row.Drag(AssetsDrag{LibraryID: a.libraryID(), FolderID: folder.ID})
			row.Children(func() {
				ui.Box(c).Size(sidebarRowPadding, sidebarRowHeight).Shrink(0)
				sidebarFolderIcon(c, folder.Icon, selected, ink)
				ui.Text(c, folder.Name).SingleLine().Shrink(1).MinWidth(0).Margin(0, 0, 0, sidebarIconGap)
				// 有子文件夹时，展开箭头紧跟在名字后面，不挤占图标那一列，所有行的图标保持对齐。
				if opened != nil {
					arrow := ui.ButtonBase(c.Key("disclose")).Size(sidebarDiscloseHit, sidebarDiscloseHit).Shrink(0).Padding(0).Margin(0, 0, 0, 2).Radius(radiusSmall).
						Cursor(ui.CursorPointer).Justify(ui.Center).AlignItems(ui.Center).Label(a.T("toggle_folder", folder.Name))
					arrow.Children(func() {
						chevron := sidebarChevron(c, *opened).Size(iconTiny, iconTiny).Shrink(0)
						chevron.TextColor(chevron.AnimateColor("ink", ink.Alpha(themeValue(c, .6, .65)), motionColor))
					})
					if arrow.Clicked() {
						*opened = !*opened
					}
				}
				ui.Box(c).Grow(1).MinWidth(0)
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
			row.ContextMenu(func(m *ui.Menu) { a.folderMenu(m, folder) })
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
	a.ShowNewFolder(parentID, "", nil)
}

// 文件夹在侧栏里的展开开关。
func (a *App) sidebarOpenFlag(id string) *bool {
	if a.Sidebar.Open == nil {
		a.Sidebar.Open = make(map[string]*bool)
	}
	open := a.Sidebar.Open[id]
	if open == nil {
		open = new(bool)
		a.Sidebar.Open[id] = open
	}
	return open
}

// 文件夹的右键菜单：侧栏文件夹行和文件夹页顶部的子文件夹卡片共用。
func (a *App) folderMenu(m *ui.Menu, folder model.Folder) {
	if menuItem(m, a.T("new_subfolder"), "folder-plus").Chosen() {
		*a.sidebarOpenFlag(folder.ID) = true
		a.sidebarCreateFolder(folder.ID)
	}
	if menuItem(m, a.T("edit_folder"), "pencil-simple").Chosen() {
		a.renameSidebarFolder(folder)
	}
	if folder.ParentID != "" && menuItem(m, a.T("move_to_root"), "arrow-up").Chosen() {
		a.sidebarMoveFolder(AssetsDrag{LibraryID: a.libraryID(), FolderID: folder.ID}, "", folderInside)
	}
	m.Separator()
	a.QuickAccessMenu(m, "folder", folder.ID)
	m.Separator()
	if menuItem(m, a.T("delete_folder"), "trash").Chosen() {
		a.deleteFolder(folder)
	}
}

// 删除文件夹：里面没有素材也没有子文件夹就直接删；否则先确认，并写明素材会保留。
// 回收站里的素材也算：删了文件夹，它们恢复后就回不到原来的文件夹了。
func (a *App) deleteFolder(folder model.Folder) {
	tree := catalog.FolderTree(a.Snapshot.Folders, folder.ID)
	subfolders := len(tree) - 1
	assets := 0
	for i := range a.Snapshot.Assets {
		asset := &a.Snapshot.Assets[i]
		if slices.ContainsFunc(asset.FolderIDs, func(id string) bool { return tree[id] }) {
			assets++
		}
	}
	remove := func() {
		lib := a.Library
		if a.Query.Scope == "folder" && sidebarFolderContains(a.Snapshot.Folders, folder.ID, a.Query.FolderID) {
			a.SetScope("all", "")
		}
		a.runThenNotify("delete_folder", func(ctx context.Context) error { return lib.DeleteFolder(ctx, folder.ID) }, a.T("folder_deleted", folder.Name))
	}
	if assets == 0 && subfolders == 0 {
		remove()
		return
	}
	var detail string
	switch {
	case subfolders == 0:
		detail = a.T("delete_folder_assets_detail", folder.Name, assets)
	case assets == 0:
		detail = a.T("delete_folder_tree_empty_detail", folder.Name, subfolders)
	default:
		detail = a.T("delete_folder_tree_detail", folder.Name, subfolders, assets)
	}
	a.Confirm(a.T("delete_folder"), detail, a.T("delete"), remove)
}

// 写成功以后再提示；失败只弹错误，不会先说「已完成」又报错。
func (a *App) runThenNotify(label string, work func(context.Context) error, message string) {
	gen := a.generation
	a.Run(label, func(ctx context.Context) error {
		if err := work(ctx); err != nil {
			return err
		}
		a.Window.Update(func() {
			if a.generation == gen {
				a.Notify(message)
			}
		})
		return nil
	})
}

func (a *App) sidebarCanDrop(drag AssetsDrag) bool {
	return a.Library != nil && drag.LibraryID == a.libraryID()
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
			// 放进这一行的高亮在 sidebarRowStyle 里随底色渐变；插到上下方的细线要跟着指针立刻出现。
			if _, _, valid := sidebarFolderMove(a.Snapshot.Folders, drag.FolderID, folder.ID, position); valid {
				switch position {
				case folderBefore:
					row.BorderWidth(2, 0, 0, 0).BorderColor(paletteFor(c.Theme()).Border)
				case folderAfter:
					row.BorderWidth(0, 0, 2, 0).BorderColor(paletteFor(c.Theme()).Border)
				}
			}
		}
	}
	if drag, ok := ui.Drop[AssetsDrag](row); ok && a.sidebarCanDrop(drag) {
		if drag.FolderID != "" {
			a.sidebarMoveFolder(drag, folder.ID, position)
		} else if ids := drag.IDs(); len(ids) > 0 {
			a.addAssetsToFolder(ids, folder.ID, folder.Name)
		}
	}
}

// sidebarDropInside 报告正拖着的素材或文件夹放下时会进这个文件夹（不是插到它上下方）。
func (a *App) sidebarDropInside(row ui.Element, folder model.Folder) bool {
	drag, over := ui.DragOver[AssetsDrag](row)
	if !over || !a.sidebarCanDrop(drag) {
		return false
	}
	if drag.FolderID == "" {
		return drag.HasAssets()
	}
	_, y, _ := row.PointerPosition()
	position := sidebarDropPosition(y, sidebarRowHeight)
	_, _, valid := sidebarFolderMove(a.Snapshot.Folders, drag.FolderID, folder.ID, position)
	return valid && position == folderInside
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
		*a.sidebarOpenFlag(parent) = true
	}
	lib := a.Library
	a.Run("move_folder", func(ctx context.Context) error { return lib.MoveFolder(ctx, drag.FolderID, parent, index) })
}

func sidebarChevron(c *ui.Context, open bool) ui.Element {
	arrow := sidebarAppIcon(c, "chevron-right").Size(iconSmall, iconSmall)
	angle := float32(0)
	if open {
		angle = 90
	}
	turnPose(arrow, angle)
	return arrow
}
func (a *App) sidebarHidden(key string) bool {
	return slices.Contains(a.Preferences.SidebarHidden, key)
}
func (a *App) renameSidebarFolder(folder model.Folder) {
	a.ShowEditFolder(folder)
}

// 把拖进来的素材加入文件夹，并说明加了几个；全都已经在里面时也说一声。
func (a *App) addAssetsToFolder(ids []string, folderID, name string) {
	added := 0
	for _, id := range ids {
		if asset, ok := a.assetByID[id]; ok && !slices.Contains(asset.FolderIDs, folderID) {
			added++
		}
	}
	if added == 0 {
		a.Notify(a.T("already_in_folder", name))
		return
	}
	a.SetFolders(ids, []string{folderID}, true)
	a.Notify(a.T("added_to_folder", added, name))
}

// 把拖进来的素材加上标签，提示和文件夹一样。
func (a *App) addAssetsToTag(ids []string, tagID, name string) {
	added := 0
	for _, id := range ids {
		if asset, ok := a.assetByID[id]; ok && !slices.Contains(asset.TagIDs, tagID) {
			added++
		}
	}
	if added == 0 {
		a.Notify(a.T("already_tagged", name))
		return
	}
	a.SetTags(ids, []string{tagID}, true)
	a.Notify(a.T("tagged_assets", added, name))
}

// 素材正拖到这一行上方，放下就能进去。
func (a *App) sidebarDropsAssets(row ui.Element) bool {
	drag, over := ui.DragOver[AssetsDrag](row)
	return over && a.sidebarCanDrop(drag) && drag.HasAssets()
}

// 清空回收站前先确认；侧栏回收站右键和回收站页的工具栏按钮共用。
// 打开确认框时就记下回收站里有哪些素材：确认的是这几个，之后才移进回收站的（排队中的删除）不跟着一起永久删掉。
func (a *App) ConfirmEmptyTrash() {
	var ids []string
	for _, asset := range a.Snapshot.Assets {
		if asset.Trashed {
			ids = append(ids, asset.ID)
		}
	}
	// 确认框是对这个图库说的：确认前换了图库就不删（两个图库里同一张图的 ID 相同，不能删到新图库里去）。
	service, generation := a.Assets, a.generation
	a.Confirm(a.T("empty_trash"), a.T("empty_trash_count_detail", len(ids)), a.T("empty_trash"), func() {
		if a.generation != generation {
			a.Notify(a.T("delete_library_changed"))
			return
		}
		a.Run("empty_trash", func(ctx context.Context) error { return service.EmptyTrash(ctx, ids) })
	})
}
