package ui

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type toolbarState struct {
	FilterOpen    bool
	ThumbnailOpen bool
	SortOpen      bool
	ViewOpen      bool
	SearchOpen    bool
	ImportOpen    bool
	TagSearch     string
	ScrubStart    float32  // 按下刻度尺时指针在刻度区里的横坐标
	ScrubBase     float32  // 按下时拖动层的位移（可能还在弹回途中）
	ScrubOffset   ui.Point // 透明拖动层跟着指针的位移，松手弹回 0
	shapeKey      string
	shapeCount    map[string]int
	typeKey       string
	typeExtCount  map[string]int
	typeKindCount map[string]int
	tagKey        string
	tagCount      map[string]int
	colorKey      string
	colorCount    map[string]int
	blurSearch    bool // 搜索框没内容时按了 Esc，下一帧把焦点从输入框移走
	formatVersion uint64
	extensions    []string
}

var toolbarFamilies = []struct{ ID, Hex string }{
	{"red", "#E34848"}, {"coral", "#EE7667"}, {"orange", "#E48A39"}, {"amber", "#D5A234"},
	{"yellow", "#D4BC38"}, {"lime", "#A3B64D"}, {"green", "#539B68"}, {"mint", "#6CB69C"},
	{"teal", "#458C8F"}, {"cyan", "#50A5B9"}, {"sky", "#6BA6D4"}, {"blue", "#527ABD"},
	{"indigo", "#6565B0"}, {"violet", "#8D69B8"}, {"purple", "#A56CB4"}, {"magenta", "#BB6B9D"},
	{"pink", "#D587AA"}, {"rose", "#CB6A86"}, {"brown", "#886747"}, {"olive", "#85834E"},
	{"beige", "#C8BA96"}, {"white", "#FFFFFF"}, {"gray", "#939393"}, {"black", "#252525"},
}

// 所有搜索入口都先展开窄栏里的输入框，再请求聚焦。
func (a *App) FocusSearch() {
	a.Toolbar.SearchOpen = true
	a.Search.Focus()
	a.InvalidateWindows()
}

func (a *App) toolbar(c *ui.Context) {
	t := c.Theme()
	p := paletteFor(t)
	bar := c.TitleBar()
	// 没有红绿灯时左边留 20；红绿灯压在工具栏上时，从红绿灯右边再留 12。
	left := float32(space20)
	if lights := bar.Left - a.SidebarShownWidth(); lights > 0 {
		left = lights + space12
	}
	right := space16 + max(float32(0), bar.Right-a.InspectorShownWidth())
	width := a.toolbarWidth(c) - left - right
	if a.pruneFilterTags() {
		a.refine(false)
	}
	hint := displayShortcut(a.Shortcut("search"))
	if !a.PreviewOpen() && a.ActionShortcut(c, "search") {
		a.FocusSearch()
	}
	row := ui.Row(c.Key("toolbar")).Height(MainToolbarHeight).Shrink(0).FillWidth().Padding(0, right, 0, left).AlignItems(ui.Center).Gap(space12).DragWindow().
		Background(toolbarColor(t)).BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSubtle).Role(ui.RoleToolbar).FocusGroup(ui.Horizontal).Label(a.T("browse_tools")).TabStop(false).FocusRing(false)
	if bounds := row.LayoutBounds(); bounds.W > 0 {
		width = bounds.W - left - right
	}
	bare := a.toolbarBare()
	inlineRuler, labels, title, compactSearch := a.toolbarFit(c, width, bare)
	if a.Toolbar.blurSearch {
		a.Toolbar.blurSearch = false
		row.Focus()
	}
	row.Children(func() {
		if !a.SidebarVisible {
			// 和侧栏收起同时横向展开，标题和搜索框跟着滑开，不被一下挤走。
			button := toolbarIconButton(c.Key("toggle-sidebar"), a.T("toggle_sidebar"), "sidebar").
				Transition(ui.ElementTransition{Animation: &motionSpring, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}})
			if button.Clicked() {
				a.SidebarVisible = true
			}
		}
		if title {
			a.toolbarTitle(c)
		}
		if compactSearch {
			// 窄栏里搜索收成放大镜；有搜索词时用强调色加小圆点，悬停提示写搜索词。
			term := strings.TrimSpace(a.Query.Search)
			if a.TagPage {
				term = strings.TrimSpace(a.Tags.Search)
			}
			tip := a.toolbarTip(a.T("search_assets"), "search")
			if term != "" {
				tip = a.T("search_with_term", term)
			}
			search := toolbarChip(c.Key("compact-search"), a.T("search_assets"), tip, a.Toolbar.SearchOpen, accentTone(c, term != "")...)
			search.Children(func() {
				toolbarIcon(c, "search")
				if term != "" {
					toolbarDot(c)
				}
			})
			if search.Clicked() {
				a.Toolbar.SearchOpen = !a.Toolbar.SearchOpen
				a.Search.Focus()
			}
			floatingPopover(c.Key("search-popover"), search, &a.Toolbar.SearchOpen, func() { a.toolbarSearch(c, hint).Width(panelNarrow - 2*space12) }).Width(panelNarrow)
		} else {
			a.Toolbar.SearchOpen = false
			search := a.toolbarSearch(c, hint)
			if title {
				search.Margin(0, 0, 0, space8)
			}
		}
		ui.Spacer(c)
		// 标签管理页只管标签：不放缩略图、筛选、排序、视图这些照片用的按钮，右边换成「新建标签」。
		if a.TagPage {
			a.Toolbar.ThumbnailOpen, a.Toolbar.FilterOpen, a.Toolbar.SortOpen, a.Toolbar.ViewOpen = false, false, false, false
			a.toolbarUpdate(c, labels)
			a.toolbarAction(c, "new-tag", a.T("new_tag"), labels, a.NewTag)
			return
		}
		// 一个素材都没有：缩略图、筛选、排序、视图切换都没用，只留标题、搜索和导入。
		if bare {
			a.Toolbar.ThumbnailOpen, a.Toolbar.FilterOpen, a.Toolbar.SortOpen, a.Toolbar.ViewOpen = false, false, false, false
			a.toolbarUpdate(c, labels)
			if a.Query.Scope != "trash" {
				a.importButton(c, labels)
			}
			return
		}
		// 列表视图用不上缩略图大小：刻度尺（或它的按钮）连同后面的竖线一起收起淡出，回到瀑布流再展开。
		if a.ViewMode == 2 {
			a.Toolbar.ThumbnailOpen = false
		} else {
			// 只动宽度：换档时整组贴着右边缩放，不会从旧位置滑过搜索框。
			ui.Row(c.Key("thumbnail-group")).Shrink(0).Gap(space12).AlignItems(ui.Center).
				Transition(ui.ElementTransition{Animation: &motionSpring, Width: true, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}).
				Children(func() {
					if inlineRuler {
						a.Toolbar.ThumbnailOpen = false
						a.thumbnailRuler(c.Key("inline-thumbnail-size"))
					} else {
						thumbnail := toolbarChip(c.Key("thumbnail-size-button"), a.T("thumbnail_size"), a.T("thumbnail_size"), a.Toolbar.ThumbnailOpen)
						thumbnail.Children(func() { toolbarIcon(c, "thumbnail-size") })
						if thumbnail.Clicked() {
							a.Toolbar.ThumbnailOpen = !a.Toolbar.ThumbnailOpen
							a.Toolbar.FilterOpen = false
							a.Toolbar.SortOpen = false
							a.Toolbar.ViewOpen = false
						}
						floatingPopover(c.Key("thumbnail-size-popover"), thumbnail, &a.Toolbar.ThumbnailOpen, func() {
							ui.Column(c).Gap(space12).Children(func() {
								ui.Text(c, a.T("thumbnail_size")).FontSize(fontSecondary).Bold()
								a.thumbnailRuler(c.Key("popover-thumbnail-size"))
							})
						})
					}
					toolbarDivider(c)
				})
		}
		a.toolbarFilter(c, labels)
		// 最近使用永远按最后使用时间排，随机浏览没有顺序：这两处排序没有作用，不放按钮。
		if a.toolbarSortable() {
			sortTip := ""
			if !labels {
				sortTip = a.toolbarTip(a.T("sort"), "sort")
			}
			sort := toolbarChip(c.Key("sort"), a.T("sort"), sortTip, a.Toolbar.SortOpen, accentTone(c, a.sortChanged())...)
			sort.Children(func() {
				toolbarIcon(c, "sort")
				if labels {
					ui.Text(c, a.T("sort")).SingleLine()
					toolbarCaret(c)
				}
			})
			if labels {
				sort.Padding(0, toolbarChipPadding-1, 0, toolbarChipPadding)
			}
			if sort.Clicked() {
				a.Toolbar.SortOpen = !a.Toolbar.SortOpen
				a.Toolbar.FilterOpen = false
				a.Toolbar.ThumbnailOpen = false
				a.Toolbar.ViewOpen = false
			}
			floatingPopover(c.Key("sorting"), sort, &a.Toolbar.SortOpen, func() { a.sortPanel(c) }).Width(popSortWidth).Padding(popSortPadding)
		} else {
			a.Toolbar.SortOpen = false
		}
		if width < 360 {
			icon := []string{"masonry", "justified", "list"}[a.ViewMode]
			view := toolbarChip(c.Key("view-menu"), a.T("view_mode"), a.T("view_mode"), a.Toolbar.ViewOpen)
			view.Children(func() { toolbarIcon(c, icon) })
			if view.Clicked() {
				a.Toolbar.ViewOpen = !a.Toolbar.ViewOpen
				a.Toolbar.FilterOpen = false
				a.Toolbar.SortOpen = false
				a.Toolbar.ThumbnailOpen = false
			}
			floatingPopover(c.Key("view-popover"), view, &a.Toolbar.ViewOpen, func() { a.viewSwitcher(c) })
		} else {
			a.Toolbar.ViewOpen = false
			a.viewSwitcher(c)
		}
		a.toolbarUpdate(c, labels)
		toolbarDivider(c)
		// 回收站里导入没有意义：右上角换成「清空回收站」，选中素材时旁边多一个「恢复」。
		if a.Query.Scope == "trash" {
			a.emptyTrashButton(c, labels)
			if len(a.SelectedAssets()) > 0 {
				a.restoreButton(c, labels)
			}
		} else {
			a.importButton(c, labels)
		}
	})
}

// toolbarBare 表示当前位置里没有素材可看：整库为空（回收站里的不算），回收站页则看回收站是否为空。
func (a *App) toolbarBare() bool {
	if a.Index == nil || a.TagPage {
		return false
	}
	counts := a.Index.Counts()
	if a.Query.Scope == "trash" {
		return counts.Trash == 0
	}
	return counts.All == 0
}

// toolbarSortable 表示当前位置能排序：最近使用按最后使用时间、随机浏览没有顺序，排序对它们没有作用。
func (a *App) toolbarSortable() bool {
	return a.Query.Scope != "random" && a.Query.Scope != "recent"
}

// sortChanged 表示排序不是默认的「添加时间、新的在前」。
func (a *App) sortChanged() bool {
	return a.Query.Sort != "" && a.Query.Sort != "added" || !a.Query.Descending
}

// accentTone 在 on 为真时返回品牌强调色，给「有条件生效」的工具栏按钮用；否则返回空，按钮保持默认颜色。
func accentTone(c *ui.Context, on bool) []ui.Color {
	if !on {
		return nil
	}
	return []ui.Color{paletteFor(c.Theme()).Accent}
}

// 工具栏按钮悬停提示：只有图标的按钮才需要，带上用户查不到的快捷键。
func (a *App) toolbarTip(label, action string) string {
	if key := displayShortcut(a.Shortcut(action)); key != "" {
		return label + "  " + key
	}
	return label
}

// filterCount 是生效的筛选条件个数。
func (a *App) filterCount() int {
	return len(a.Query.ColorFamilies) + len(a.Query.TagIDs) + len(a.Query.Extensions) + len(a.Query.Kinds) + len(a.Query.Shapes)
}

// filterLabel 是筛选按钮上的字：有条件时带条件数。
func (a *App) filterLabel() string {
	if count := a.filterCount(); count > 0 {
		return a.T("filter_count", count)
	}
	return a.T("filter")
}

// toolbarFilter 是工具栏的筛选按钮：有条件时文字和图标用强调色并带条件数，只有图标时在图标角上放小圆点；
// 悬停在带文字的按钮上，右端出现小 ×，一键清空全部筛选。
func (a *App) toolbarFilter(c *ui.Context, labels bool) {
	p := paletteFor(c.Theme())
	filtering := a.filterCount() > 0
	clearable := filtering && labels
	group := ui.Row(c.Key("filter-group")).Height(toolbarControlHeight).Shrink(0).AlignItems(ui.Center)
	group.Children(func() {
		tip := ""
		if !labels {
			tip = a.toolbarTip(a.filterLabel(), "filter")
		}
		filter := toolbarChip(c.Key("filter"), a.filterLabel(), tip, a.Toolbar.FilterOpen, accentTone(c, filtering)...)
		filter.Children(func() {
			toolbarIcon(c, "filter")
			if labels {
				ui.Text(c, a.filterLabel()).SingleLine()
			}
			if filtering && !labels {
				toolbarDot(c)
			}
		})
		if labels {
			right := float32(space12)
			if clearable {
				right += toolbarClearSlot
			}
			filter.Padding(0, right, 0, toolbarChipPadding)
		}
		if filter.Clicked() {
			a.Toolbar.FilterOpen = !a.Toolbar.FilterOpen
			a.Toolbar.SortOpen = false
			a.Toolbar.ThumbnailOpen = false
			a.Toolbar.ViewOpen = false
		}
		floatingPopover(c.Key("filters"), filter, &a.Toolbar.FilterOpen, func() { a.filterPanel(c) }).Width(popFilterWidth).Padding(0)
		if clearable {
			shown := group.Hovered()
			clear := ui.ButtonBase(c.Key("clear-filters-chip")).Label(a.T("clear_filters")).Tooltip(a.T("clear_filters")).Size(toolbarClearSize, toolbarClearSize).
				Absolute().Right(toolbarClearInset).Top((toolbarControlHeight - toolbarClearSize) / 2).Padding(0).Radius(5).Cursor(ui.CursorPointer).Center()
			background, ink := ui.Transparent, toolbarInk(p, .6)
			if clear.Pressed() {
				background, ink = toolbarFill(p, .16), p.Text
			} else if clear.Hovered() {
				background, ink = toolbarFill(p, .1), p.Text
			}
			animateColors(clear, background, ink)
			clear.Children(func() { toolbarIcon(c, "close").Size(10, 10) })
			showPose(clear, shown).IgnoreInput(!shown)
			if clear.Clicked() {
				a.clearAllFilters()
			}
		}
	})
}

func (a *App) toolbarUpdate(c *ui.Context, labels bool) {
	if version := AvailableUpdateVersion(); version != "" {
		tip := a.T("update_to_version", version)
		update := toolbarChip(c.Key("available-update"), a.T("update"), tip, false, accentTone(c, true)...)
		update.Children(func() {
			toolbarIcon(c, "update")
			if labels {
				ui.Text(c, a.T("update")).SingleLine()
			}
		})
		if labels {
			update.Padding(0, space12, 0, toolbarChipPadding)
		}
		if update.Clicked() {
			ShowAvailableUpdate()
		}
	}
}

// 按真实文字宽度决定放得下多少：先把刻度尺收进弹出层，再去掉按钮文字，再藏标题，最后把搜索框收成按钮。
// 搜索框一直按最小宽度算（真正显示时有多少空位就长到多宽），所以搜索框总是先缩，标题最后才让位。
func (a *App) toolbarFit(c *ui.Context, width float32, bare bool) (inlineRuler, labels, title, compactSearch bool) {
	measure := func(text string, size float32, weight int) float32 {
		return toolbarTextWidth(c, a.Texts.Locale(), text, size, weight)
	}
	// 标题里的项数至少按两位数预留，「1 项」「14 项」之间不会因为差几个像素来回换档；更多时按实际宽度算。
	name, count := a.browseTitle()
	countWidth := max(measure(a.T("toolbar_item_count", "00"), toolbarCountSize, 400), measure(a.T("toolbar_item_count", formatTagCount(count)), toolbarCountSize, 400))
	titleWidth := min(measure(name, toolbarTitleSize, 650), toolbarTitleWidth) + space8 + countWidth
	// 搜索框能缩到最小宽度，提示文字放不下时截断，所以按最小宽度算。
	search := float32(toolbarSearchMin)
	ruler := rulerWidth + 1 + 21 + measure(a.thumbnailLabel(), toolbarFont, 400)
	chip := func(text string, extra float32) float32 {
		return toolbarChipPadding + 16 + toolbarChipGap + measure(text, toolbarFont, 400) + space12 + extra
	}
	filter, sort := chip(a.filterLabel(), 0), chip(a.T("sort"), toolbarChipGap+12-3)
	if a.filterCount() > 0 {
		filter += toolbarClearSlot
	}
	primary := func(text string) float32 {
		return 11 + toolbarAddIcon + toolbarChipGap + measure(text, toolbarFont, 500) + 14
	}
	// 右边的动作按钮：带文字时各自的宽度；只剩图标时每个都是方块。
	var actions []float32
	switch {
	case a.TagPage:
		actions = []float32{primary(a.T("new_tag"))}
	case a.Query.Scope == "trash":
		if a.Index != nil && a.Index.Counts().Trash > 0 {
			actions = append(actions, chip(a.T("empty_trash_short"), 0))
		}
		if len(a.SelectedAssets()) > 0 {
			actions = append(actions, primary(a.T("restore")))
		}
	default:
		actions = []float32{primary(a.T("import"))}
	}
	update := float32(0)
	if AvailableUpdateVersion() != "" {
		update = chip(a.T("update"), 0)
	}
	fits := func(inlineRuler, labels, title bool) bool {
		// 固定会有：搜索、弹簧。
		total, items := search, 2
		if !a.SidebarVisible {
			total, items = total+controlHeight, items+1
		}
		if update > 0 {
			if labels {
				total += update
			} else {
				total += toolbarControlHeight
			}
			items++
		}
		if title {
			total, items = total+titleWidth+space8, items+1
		}
		if !a.TagPage && !bare {
			// 缩略图刻度尺（或它的按钮）加后面一条竖线；列表视图没有。
			if a.ViewMode != 2 {
				total, items = total+1+4, items+2
				if inlineRuler {
					total += ruler
				} else {
					total += toolbarControlHeight
				}
			}
			// 筛选、排序（有的位置没有）、视图切换、再一条竖线。
			tools := 1
			if labels {
				total += filter
			} else {
				total += toolbarControlHeight
			}
			if a.toolbarSortable() {
				tools++
				if labels {
					total += sort
				} else {
					total += toolbarControlHeight
				}
			}
			total, items = total+2*toolbarSegmentInset+3*toolbarSegmentWidth+2*2+1+4, items+tools+2
		}
		for _, w := range actions {
			if labels {
				total += w
			} else {
				total += toolbarControlHeight
			}
			items++
		}
		return total+float32(items-1)*space12 <= width
	}
	switch {
	case fits(true, true, true):
		return true, true, true, false
	case fits(false, true, true):
		return false, true, true, false
	case fits(false, false, true):
		return false, false, true, false
	case fits(false, false, false):
		return false, false, false, false
	}
	return false, false, false, true
}

func toolbarTextWidth(c *ui.Context, lang, text string, size float32, weight int) float32 {
	w, _ := c.MeasureText(0, ui.Span{Text: text, Font: c.Theme().Font, Size: size, Weight: weight, Lang: lang})
	return w
}

// 标题：当前浏览的位置和里面有多少项，按文字基线对齐。
func (a *App) toolbarTitle(c *ui.Context) {
	p := paletteFor(c.Theme())
	name, count := a.browseTitle()
	ui.Row(c.Key("toolbar-title")).AlignItems(ui.Baseline).Gap(space8).Shrink(0).Children(func() {
		ui.Text(c.Key("title"), name).FontSize(toolbarTitleSize).FontWeight(650).TextColor(p.Text).SingleLine().MaxWidth(toolbarTitleWidth).
			ContentTransition(replaceMotion)
		countMotion(ui.Text(c.Key("count"), a.T("toolbar_item_count", formatTagCount(count))).FontSize(toolbarCountSize).FontFeatures("tnum").TextColor(toolbarInk(p, .5)).SingleLine(), count)
	})
}

// 当前浏览位置的名字和项数：侧栏入口用它的名字，文件夹、标签、智能文件夹用自己的名字。
func (a *App) browseTitle() (string, int) {
	if a.TagPage {
		return a.T("tag_management"), len(a.visibleTags())
	}
	count := len(a.Visible)
	switch a.Query.Scope {
	case "folder":
		for _, folder := range a.Snapshot.Folders {
			if folder.ID == a.Query.FolderID {
				return folder.Name, count
			}
		}
	case "tag":
		for _, tag := range a.Snapshot.Tags {
			if tag.ID == a.Browse.TagID {
				return tag.Name, count
			}
		}
	case "smart":
		for _, folder := range a.Snapshot.SmartFolders {
			if folder.ID == a.Query.SmartFolderID {
				return folder.Name, count
			}
		}
	case "favorites", "uncategorized", "untagged", "trash":
		return a.T(a.Query.Scope), count
	case "random":
		return a.T("random_browse"), count
	case "recent":
		return a.T("recently_used"), count
	}
	return a.T("all"), count
}

// 导入：品牌色按钮，带圈加号加文字；窄的时候只剩加号。点开一个和图库菜单同款的小菜单，选图片或文件夹。
func (a *App) importButton(c *ui.Context, label bool) {
	// 灰掉的按钮不显示悬停提示，所以忙的时候把原因挂在外面这一层。
	wrap := ui.Row(c.Key("import-wrap")).Shrink(0)
	if a.Busy {
		wrap.Tooltip(a.T("import_busy"))
	}
	wrap.Children(func() {
		menu := ui.MenuBase(c.Key("import"), &a.Toolbar.ImportOpen)
		if a.Busy {
			a.Toolbar.ImportOpen = false
		}
		tip := ""
		if !label {
			tip = a.toolbarTip(a.T("import"), "import")
		}
		a.toolbarActionStyle(c, fadeMotion(menu.Trigger.Disabled(a.Busy)), "add", a.T("import"), tip, label, a.Toolbar.ImportOpen)
		menu.Popup(func(panel ui.Element) {
			// 菜单右边和按钮右边对齐，往下空 6，不压住按钮。
			elevated(popupMotion(panel), c.Theme()).Background(panelColor(c.Theme())).Radius(radiusPanel).Padding(space4).Gap(0).
				AttachTo(menu.Trigger, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(6, 0, 6, 0)
			if libraryMenuItem(c, menu, a.T("import_files"), "photos", false).Clicked() {
				a.ChooseImport(false)
			}
			if libraryMenuItem(c, menu, a.T("import_folder"), "folder-open", false).Clicked() {
				a.ChooseImport(true)
			}
		})
	})
}

// 工具栏最右边的品牌色主按钮（新建标签）：带圈加号加文字，窄的时候只剩加号，点击执行 action。
func (a *App) toolbarAction(c *ui.Context, key, text string, label bool, action func()) {
	button := ui.ButtonBase(c.Key(key))
	tip := ""
	if !label {
		tip = text
	}
	a.toolbarActionStyle(c, button, "add", text, tip, label, false)
	if button.Clicked() {
		action()
	}
}

// 回收站页的「清空回收站」：和筛选、排序同类的普通按钮，只有文字用危险色；回收站空了就不显示，点了先确认。
func (a *App) emptyTrashButton(c *ui.Context, label bool) {
	if a.Index == nil || a.Index.Counts().Trash == 0 {
		return
	}
	// 已经在回收站里，按钮只写「清空」；只剩图标时提示写全。
	text := a.T("empty_trash_short")
	tip := ""
	if !label {
		tip = a.T("empty_trash")
	}
	button := toolbarChip(c.Key("empty-trash"), text, tip, false, paletteFor(c.Theme()).Danger).Disabled(a.Busy)
	button.Children(func() {
		toolbarIcon(c, "trash")
		if label {
			ui.Text(c, text).SingleLine()
		}
	})
	if label {
		button.Padding(0, space12, 0, toolbarChipPadding)
	}
	// 进出回收站时和「导入」交叉淡化，交代这里换了一个按钮。
	fadeMotion(button)
	if button.Clicked() {
		a.ConfirmEmptyTrash()
	}
}

// 回收站里选中素材时，原来「导入」的位置换成品牌色的「恢复」。
func (a *App) restoreButton(c *ui.Context, label bool) {
	button := fadeMotion(ui.ButtonBase(c.Key("restore")))
	tip := ""
	if !label {
		tip = a.T("restore")
	}
	a.toolbarActionStyle(c, button, "restore", a.T("restore"), tip, label, false)
	if button.Clicked() {
		a.Restore(a.selectedIDs())
	}
}

// 工具栏最右边主按钮的外观，导入（菜单按钮）、新建标签和恢复共用。
// 带文字时不设悬停提示（文字已经写着）；tip 是只有图标时、或按钮灰掉时要说的话。
func (a *App) toolbarActionStyle(c *ui.Context, button ui.Element, icon, text, tip string, label, open bool) {
	p := paletteFor(c.Theme())
	button.Label(text).Height(toolbarControlHeight).MinWidth(toolbarControlHeight).Shrink(0).
		Radius(toolbarRadius).Gap(toolbarChipGap).AlignItems(ui.Center).Justify(ui.Center).Cursor(ui.CursorPointer).DimDisabled(false)
	if tip != "" {
		button.Tooltip(tip)
	}
	background, ink := p.Accent, p.OnAccent
	switch {
	case a.Busy || button.IsDisabled():
		background, ink = toolbarFill(p, toolbarChipFill), toolbarInk(p, .5)
	case button.Pressed():
		background = p.AccentPressed
	case button.Hovered() || open:
		background = p.AccentHover
	}
	animateColors(button, background, ink)
	button.Children(func() {
		// 白字在实色底上显粗、1.5 的线显细：文字用 500，图标用描线加粗到 2 的那套，两边分量相当。
		ui.Icon(c, iconSidebar[iconName(icon)]).Size(toolbarAddIcon, toolbarAddIcon)
		if label {
			ui.Text(c, text).FontSize(toolbarFont).FontWeight(500).SingleLine()
		}
	})
	if label {
		button.Padding(0, space12+2, 0, toolbarChipPadding+1)
	} else {
		button.Width(toolbarControlHeight).Padding(0)
	}
}

// 工具栏按钮：浅浅一层底色、圆角 10，图标在前、文字在后；没文字时是 34 的方块。
// tip 为空时不设悬停提示（带文字的按钮不必重复自己的字）；tone 给出时文字和图标一直用这个颜色（强调色、危险色）。
func toolbarChip(c *ui.Context, label, tip string, active bool, tone ...ui.Color) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Height(toolbarControlHeight).MinWidth(toolbarControlHeight).Shrink(0).Padding(0).
		Radius(toolbarRadius).Gap(toolbarChipGap).AlignItems(ui.Center).Justify(ui.Center).FontSize(toolbarFont).Cursor(ui.CursorPointer)
	if tip != "" {
		button.Tooltip(tip)
	}
	background, text := toolbarFill(p, toolbarChipFill), toolbarInk(p, .86)
	switch {
	case button.Pressed():
		background, text = toolbarFill(p, .12), p.Text
	case active:
		background, text = toolbarFill(p, .1), p.Text
	case button.Hovered():
		background, text = toolbarFill(p, .075), p.Text
	}
	if len(tone) > 0 {
		text = tone[0]
	}
	animateColors(button, background, text)
	return button
}

// 图标右上角的小圆点：这个按钮背后有条件在生效。
func toolbarDot(c *ui.Context) {
	ui.Box(c.Key("dot")).Size(toolbarDotSize, toolbarDotSize).Radius(toolbarDotSize / 2).Background(paletteFor(c.Theme()).Accent).Absolute().Top(5).Right(5).PassThrough()
}

// 组和组之间的细竖线。
func toolbarDivider(c *ui.Context) {
	ui.Box(c).Size(1, 16).Shrink(0).Margin(0, 2).Background(toolbarFill(paletteFor(c.Theme()), .1))
}

// 排序文字后面的小下拉箭头，提示点开是选项。
func toolbarCaret(c *ui.Context) {
	ui.Icon(c, toolbarIcons["caret-down"]).Size(12, 12).Shrink(0).TextColor(toolbarInk(paletteFor(c.Theme()), .5))
}

// 工具栏用的颜色都按「正文色叠在工具栏底色（侧栏色）上多少」来取，浅色深色同一套比例，结果是不透明色。
func toolbarFill(p *designPalette, amount float32) ui.Color { return p.Sidebar.Mix(p.Text, amount) }
func toolbarInk(p *designPalette, amount float32) ui.Color  { return p.Sidebar.Mix(p.Text, amount) }

func toolbarIconButton(c *ui.Context, label, name string, selected ...bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(controlHeight, controlHeight).Shrink(0).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer)
	background, text := ui.Transparent, p.TextSecondary
	if len(selected) > 0 && selected[0] {
		background, text = p.Subtle, p.Text
	}
	if button.Pressed() {
		background, text = p.Pressed, p.Text
	} else if button.Hovered() {
		background, text = p.Hover, p.Text
	}
	animateColors(button, background, text)
	return button.Children(func() {
		if momentoSymbols[name] != nil {
			appSymbol(c, name)
		} else {
			toolbarIcon(c, name)
		}
	})
}

func (a *App) toolbarSearch(c *ui.Context, hint string) ui.Element {
	p := paletteFor(c.Theme())
	// 平时 300 宽；提示文字更长的语言加宽到能完整显示，空间不够时再往下缩。
	// 标签管理页里搜的是标签。
	query, label, key := &a.Query.Search, a.T("search_assets"), "asset-search"
	if a.TagPage {
		query, label, key = &a.Tags.Search, a.T("search_tags"), "tag-search"
	}
	need := 11 + 13 + space8 + toolbarTextWidth(c, a.Texts.Locale(), label, toolbarFont, 400) + 6
	if hint != "" {
		need += space8 + toolbarTextWidth(c, a.Texts.Locale(), hint, toolbarHintFont, 400) + 12
	}
	field := ui.Row(c.Key(key+"-field")).Height(toolbarControlHeight).Width(max(toolbarSearchWidth, need)).MinWidth(toolbarSearchMin).Shrink(1).
		Gap(space8).Padding(0, 6, 0, 11).Radius(toolbarRadius).Cursor(ui.CursorText)
	// 被挤窄放不下完整提示时换成「搜索」，不让字被切掉半个。
	placeholder := label
	if bounds := field.LayoutBounds(); bounds.W > 0 && bounds.W+.5 < need {
		placeholder = a.T("search")
	}
	field.Children(func() {
		ui.Icon(c, toolbarIcons["search"]).Size(13, 13).Shrink(0).TextColor(toolbarInk(p, .55))
		shown := *query
		if shown == "" {
			shown = placeholder
		}
		changed := func() {
			if a.TagPage {
				a.Tags.Scroll.ScrollTo(0, 0, motionSpring)
			} else {
				a.Refine()
			}
		}
		input := ui.TextInputBase(c.Key(key), query).Bind(&a.Search).Search().Placeholder(placeholder).PlaceholderColor(placeholderColor(c)).Label(label).Grow(1).MinWidth(0).Padding(0, 0, searchTextLift(shown), 0).FontSize(toolbarFont).FocusRing(false)
		if input.Changed() {
			changed()
		}
		// Esc：有内容先清空并留在输入框，没内容才让输入框失焦（窄栏里同时收起弹出的搜索框）。
		if input.Shortcut(0, ui.KeyEscape) {
			if *query != "" {
				*query = ""
				changed()
			} else {
				a.Toolbar.SearchOpen = false
				a.Toolbar.blurSearch = true
				a.InvalidateWindows()
			}
		}
		// 开始输入时键帽淡出、清除按钮淡入，清空时反过来。
		if *query != "" {
			if fadeMotion(toolbarIconButton(c.Key("clear-search"), a.T("clear_search"), "close").Size(space20, space24)).Clicked() {
				*query = ""
				changed()
				a.Search.Focus()
			}
		} else if hint != "" {
			// 快捷键提示是一枚小键帽：比搜索框再深一层的底，字更淡。
			fadeMotion(centeredTextHeight(ui.Text(c.Key("search-shortcut-hint"), hint).FontSize(toolbarHintFont).TextColor(toolbarInk(p, .55)).SingleLine().Shrink(0).
				Radius(5).Background(toolbarFill(p, toolbarChipFill+.06)).IgnoreInput(true), 16, 6))
		}
	})
	// 点框里任意空白处也能输入。
	if field.Clicked() {
		a.Search.Focus()
	}
	background, border := toolbarFill(p, toolbarChipFill), ui.Transparent
	if field.FocusWithin() {
		background, border = toolbarFill(p, toolbarChipFill), p.Focus
	} else if field.Hovered() {
		background = toolbarFill(p, .075)
	}
	field.Background(field.AnimateColor("field-background", background, motionColor))
	return field.Border(1, field.AnimateColor("field-border", border, motionColor))
}

func (a *App) viewSwitcher(c *ui.Context) {
	p := paletteFor(c.Theme())
	// 瀑布流、等高行、列表；分段序号就是视图编号。
	modes := []struct {
		ID, Label string
		Mode      int
	}{{"masonry", "view_masonry", 0}, {"justified", "view_justified", 1}, {"list", "view_list", 2}}
	index := 0
	for i, mode := range modes {
		if mode.Mode == a.ViewMode {
			index = i
		}
	}
	seg := ui.SegmentedBase(c.Key("view-mode"), &index, len(modes))
	seg.Track.Height(toolbarControlHeight).Padding(toolbarSegmentInset).Gap(2).Radius(toolbarRadius).Background(toolbarFill(p, toolbarChipFill)).Shrink(0).AlignItems(ui.Center).Label(a.T("view_mode")).Children(func() {
		for i, item := range modes {
			part := seg.Segment(i).Size(toolbarSegmentWidth, toolbarControlHeight-2*toolbarSegmentInset).Padding(0).Center().Radius(toolbarRadius - toolbarSegmentInset).Cursor(ui.CursorPointer).Label(a.T(item.Label)).Tooltip(a.toolbarTip(a.T(item.Label), item.Label))
			// 选中的底由下面整条轨道画，这里只管图标色和未选中格的悬停底。
			background, text := ui.Transparent, toolbarInk(p, .6)
			if index == i {
				text = p.Text
			} else if part.Hovered() || part.Pressed() {
				background, text = toolbarFill(p, .075), toolbarInk(p, .86)
			}
			animateColors(part, background, text)
			part.Children(func() { toolbarIcon(c, item.ID) })
		}
	})
	a.ViewMode = modes[index].Mode
	// 选中底块切换时滑过去。
	position := seg.Track.AnimateTo("indicator-position", float32(index), motionSpring)
	seg.Track.Draw(func(pt *ui.Painter, r ui.Rect) {
		h := float32(toolbarControlHeight - 2*toolbarSegmentInset)
		pt.Fill(ui.Rect{X: r.X + toolbarSegmentInset + position*(toolbarSegmentWidth+2), Y: r.Y + (r.H-h)/2, W: toolbarSegmentWidth, H: h}, toolbarFill(p, .13), toolbarRadius-toolbarSegmentInset)
	})
}

// 筛选弹层：标题栏（标题 + 清除）一条通栏分隔线，下面颜色、标签、文件类型三段只靠留白分开。
func (a *App) filterPanel(c *ui.Context) {
	p := paletteFor(c.Theme())
	active := a.filtering()
	ui.Column(c).FillWidth().Gap(0).Children(func() {
		ui.Row(c).FillWidth().Height(popHeaderHeight).Padding(0, popInset-popClearPad, 0, popInset).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.T("filter")).FontSize(fontBody).FontWeight(600).Grow(1).MinWidth(0).SingleLine()
			// 有条件才显示「清除筛选」，没条件时不占位。
			if active {
				clear := ui.ButtonBase(c.Key("clear-filters")).Height(popChipHeight).Padding(0, popClearPad).Radius(radiusSmall).Cursor(ui.CursorPointer).Label(a.T("clear_filters"))
				// 次要操作：字比标题小两号、颜色用三级文字色。
				background, text := ui.Transparent, p.TextTertiary
				switch {
				case clear.Pressed():
					background, text = p.Pressed, p.Text
				case clear.Hovered():
					background, text = p.Hover, p.Text
				}
				animateColors(clear, background, text)
				clear.Children(func() { ui.Text(c, a.T("clear_filters")).FontSize(fontGroup).SingleLine() })
				if clear.Clicked() {
					a.clearAllFilters()
				}
			}
		})
		ui.Divider(c).Background(p.BorderSubtle)
		ui.Scroll(c.Key("filter-scroll")).Scrollbars(false).FillWidth().MaxHeight(popBodyMax).Padding(popInset-2, popInset, popInset, popInset).Gap(popSectionGap).Children(func() {
			colors := a.colorCounts()
			popSection(c, a.T("filter_colors"), func() {
				for start := 0; start < len(toolbarFamilies); start += popSwatchColumns {
					ui.Row(c.Key(start)).FillWidth().Justify(ui.SpaceBetween).Children(func() {
						for _, family := range toolbarFamilies[start:min(start+popSwatchColumns, len(toolbarFamilies))] {
							chosen := slices.Contains(a.Query.ColorFamilies, family.ID)
							if a.colorSwatch(c.Key(family.ID), family.ID, family.Hex, chosen, colors[family.ID]).Clicked() {
								a.Query.ColorFamilies = toolbarToggle(a.Query.ColorFamilies, family.ID, !chosen)
								a.Refine()
							}
						}
					})
				}
			})
			a.filterTagSection(c)
			a.filterTypeSection(c)
			shapes := a.shapeCounts()
			// 一个能选的形状都没有时，整段不显示。
			if slices.ContainsFunc(catalog.Shapes, func(shape string) bool { return shapes[shape] > 0 || slices.Contains(a.Query.Shapes, shape) }) {
				popSection(c, a.T("filter_shapes"), func() {
					ui.Row(c.Key("shape-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
						for _, shape := range catalog.Shapes {
							chosen := slices.Contains(a.Query.Shapes, shape)
							// 没有素材的形状不列出来，已经选上的照常显示，免得取消不了。
							if shapes[shape] == 0 && !chosen {
								continue
							}
							// 横图、竖图、方图、全景带形状图标，16:9 这类比例只写数字。
							label, icon := shape, ""
							if !strings.Contains(shape, ":") {
								label, icon = a.T("shape_"+shape), "shape-"+shape
							}
							if popCountChip(c.Key("shape-"+shape), &chosen, icon, label, shapes[shape]).Changed() {
								a.Query.Shapes = toolbarToggle(a.Query.Shapes, shape, chosen)
								a.Refine()
							}
						}
					})
				})
			}
		})
	})
}

// filterTagSection 是筛选里的标签段：只列当前位置、其他条件下有素材的标签，已勾选的照常显示；
// 一个都没有时整段不显示，图库里根本没有标签才写「还没有标签」。
func (a *App) filterTagSection(c *ui.Context) {
	p := paletteFor(c.Theme())
	counts := a.tagCounts()
	tags := slices.Clone(a.Snapshot.Tags)
	slices.SortFunc(tags, func(x, y model.Tag) int { return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)) })
	if len(tags) > 0 {
		tags = slices.DeleteFunc(tags, func(tag model.Tag) bool { return counts[tag.ID] == 0 && !slices.Contains(a.Query.TagIDs, tag.ID) })
		if len(tags) == 0 {
			return
		}
	}
	popSection(c, a.T("filter_tags"), func() {
		// 搜索框只在标签多时出现；不显示时也不按旧搜索词过滤，免得标签莫名消失又没处清。
		term := strings.TrimSpace(a.Toolbar.TagSearch)
		query := ""
		if len(a.Snapshot.Tags) > popTagSearchFrom {
			query = strings.ToLower(term)
			field, _ := searchField(c.Key("tag-search"), &a.Toolbar.TagSearch, a.T("search_tags"), a.T("clear_search"))
			// 底色换成和胶囊同一套浅底，边框和聚焦边框照旧。
			field.FillWidth().Margin(0, 0, 2, 0).Background(popFill(p, .05))
		}
		tags = slices.DeleteFunc(tags, func(tag model.Tag) bool {
			return query != "" && !strings.Contains(strings.ToLower(tag.Name), query)
		})
		// 没有标签、或搜不到时只留一行浅色说明，不画空的标签行，免得多出一截间距。
		if len(tags) == 0 {
			if len(a.Snapshot.Tags) == 0 {
				popEmpty(c, a.T("no_tags"))
			} else {
				popEmpty(c, a.T("no_tag_results", term))
			}
			return
		}
		ui.Row(c.Key("tag-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
			for _, tag := range tags {
				chosen := slices.Contains(a.Query.TagIDs, tag.ID)
				if popCountChip(c.Key("filter-tag-"+tag.ID), &chosen, "", tag.Name, counts[tag.ID]).Changed() {
					a.Query.TagIDs = toolbarToggle(a.Query.TagIDs, tag.ID, chosen)
					a.Refine()
				}
			}
		})
	})
}

// filterTypeSection 是筛选里的文件类型段：只列当前位置、其他条件下有素材的类型，已勾选的照常显示。
func (a *App) filterTypeSection(c *ui.Context) {
	extensions, kinds := a.typeCounts()
	shows := func(count int, chosen bool) bool { return count > 0 || chosen }
	var types []string
	for _, ext := range a.toolbarExtensions() {
		if shows(extensions[ext], slices.Contains(a.Query.Extensions, ext)) {
			types = append(types, ext)
		}
	}
	video := shows(kinds[model.KindVideo], slices.Contains(a.Query.Kinds, model.KindVideo))
	live := shows(kinds[model.KindLive], slices.Contains(a.Query.Kinds, model.KindLive))
	if len(types) == 0 && !video && !live {
		return
	}
	popSection(c, a.T("filter_types"), func() {
		ui.Row(c.Key("type-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
			// 有视频时，最前面放一个「视频」，一次选中所有视频格式。
			if video {
				chosen := slices.Contains(a.Query.Kinds, model.KindVideo)
				if popCountChip(c.Key("kind-video"), &chosen, "type-video", a.T("filter_kind_video"), kinds[model.KindVideo]).Changed() {
					a.Query.Kinds = toolbarToggle(a.Query.Kinds, model.KindVideo, chosen)
					a.Refine()
				}
			}
			if live {
				chosen := slices.Contains(a.Query.Kinds, model.KindLive)
				if popCountChip(c.Key("kind-live"), &chosen, "type-live", a.T("filter_kind_live"), kinds[model.KindLive]).Changed() {
					a.Query.Kinds = toolbarToggle(a.Query.Kinds, model.KindLive, chosen)
					a.Refine()
				}
			}
			for _, ext := range types {
				chosen := slices.Contains(a.Query.Extensions, ext)
				if popCountChip(c.Key("extension-"+ext), &chosen, extensionIcon(ext), strings.ToUpper(ext), extensions[ext]).Changed() {
					a.Query.Extensions = toolbarToggle(a.Query.Extensions, ext, chosen)
					a.Refine()
				}
			}
		})
	})
}

// 筛选里勾着的标签被删掉后，从筛选条件里去掉；不然列表一直是空的，筛选面板里也找不到它来取消。
func (a *App) pruneFilterTags() bool {
	if len(a.Query.TagIDs) == 0 || a.Index == nil {
		return false
	}
	kept := slices.DeleteFunc(slices.Clone(a.Query.TagIDs), func(id string) bool {
		return !slices.ContainsFunc(a.Snapshot.Tags, func(tag model.Tag) bool { return tag.ID == id })
	})
	if len(kept) == len(a.Query.TagIDs) {
		return false
	}
	if len(kept) == 0 {
		kept = nil
	}
	a.Query.TagIDs = kept
	return true
}

// popSection 是筛选里的一段：小标题，下面是内容。
func popSection(c *ui.Context, title string, content func()) {
	ui.Column(c.Key(title)).FillWidth().Gap(space8 + 2).Children(func() {
		ui.Text(c, title).FontSize(fontGroup).FontWeight(600).TextColor(paletteFor(c.Theme()).TextSecondary).SingleLine()
		content()
	})
}

// colorSwatch 是一个圆角方块色块；选中时外面多一圈同心的正文色细框，色块本身不变。
// count 是当前位置里这个颜色的素材数：悬停提示写「红色 · 12」，没有素材又没选中的色块淡下去。
func (a *App) colorSwatch(c *ui.Context, id, hex string, chosen bool, count int) ui.Element {
	p := paletteFor(c.Theme())
	name := a.T("color_" + id)
	tip := a.T("color_count", name, count)
	button := ui.ButtonBase(c).Label(tip).Tooltip(tip).Size(popSwatchHit, popSwatchHit).Shrink(0).Padding(0).Radius(popSwatchRadius + (popSwatchHit-popSwatch)/2).Cursor(ui.CursorPointer).Center()
	button.Role(ui.RoleCheckBox).Checked(chosen)
	if count == 0 && !chosen {
		button.Opacity(popSwatchEmptyOpacity)
	}
	ring := ui.Transparent
	if chosen {
		ring = p.Text
	} else if button.Hovered() {
		ring = p.Border
	}
	button.Border(2, button.AnimateColor("ring", ring, motionColor))
	button.Children(func() {
		ui.Box(c).Size(popSwatch, popSwatch).Radius(popSwatchRadius).Background(ui.Hex(hex)).Border(1, p.Text.Alpha(.12)).PassThrough()
	})
	return button
}

// popChip 是筛选弹层的圆角开关：平时浅底，选中反色；icon 为空时只有文字，不带数量。
func popChip(c *ui.Context, on *bool, icon, label string) ui.Element {
	return popCountChip(c, on, icon, label, -1)
}

// popCountChip 是带数量的筛选胶囊：可带种类图标，数量用淡一级的颜色跟在名字后面；count 小于 0 时不显示数量。
func popCountChip(c *ui.Context, on *bool, icon, label string, count int) ui.Element {
	p := paletteFor(c.Theme())
	accessible := label
	if count >= 0 {
		accessible = fmt.Sprintf("%s %d", label, count)
	}
	chip := ui.ToggleBase(c, on).Label(accessible).Height(popChipHeight).Padding(0, space8+1).Gap(popChipIconGap).Radius(popChipRadius).Cursor(ui.CursorPointer).AlignItems(ui.Center).MaxWidth(popChipMaxWidth)
	background, text := popFill(p, .05), p.Text
	switch {
	case *on:
		background, text = p.Text, p.Raised
	case chip.Pressed():
		background = popFill(p, .12)
	case chip.Hovered():
		background = popFill(p, .085)
	}
	animateColors(chip, background, text)
	if icon != "" {
		chip.Padding(0, space8+1, 0, space8-1)
	}
	return chip.Children(func() {
		if icon != "" {
			toolbarIcon(c, icon).Size(iconSmall, iconSmall).Shrink(0)
		}
		ui.Row(c).Gap(space4).AlignItems(ui.Center).MinWidth(0).Children(func() {
			ui.Text(c, label).FontSize(fontSecondary).SingleLine().MinWidth(0)
			if count >= 0 {
				countMotion(ui.Text(c.Key("count"), fmt.Sprint(count)).FontSize(fontSecondary).FontFeatures("tnum").TextColor(text.Alpha(.55)).SingleLine().Shrink(0), count)
			}
		})
	})
}

// extensionIcon 按种类给文件类型配图标：照片、图像、动图、视频。
func extensionIcon(ext string) string {
	switch ext {
	case "jpg", "jpeg", "heic", "heif", "avif":
		return "type-photo"
	case "gif":
		return "type-motion"
	}
	if media.IsVideoExtension(ext) {
		return "type-video"
	}
	if media.IsRawExtension(ext) {
		return "type-photo"
	}
	return "type-image"
}

// popEmpty 是弹层里一段没有内容时的说明：和小标题同字号、更淡，贴近小标题。
func popEmpty(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(fontGroup).TextColor(paletteFor(c.Theme()).TextTertiary.Alpha(.75)).Margin(-space4, 0, 0, 0)
}

// 弹层里的底色按「正文色叠在弹层底色上多少」来取，和工具栏一个思路。
func popFill(p *designPalette, amount float32) ui.Color { return p.Raised.Mix(p.Text, amount) }

// 排序弹层：像系统菜单，当前字段前面打勾、右边箭头表示升序或降序；
// 点别的字段就换过去，再点当前字段就反转方向。弹层不收起，方便连着点。
func (a *App) sortPanel(c *ui.Context) {
	ui.Column(c).FillWidth().Gap(0).Children(func() {
		for _, item := range []struct{ ID, Label string }{{"added", "sort_added"}, {"name", "sort_name"}, {"size", "sort_size"}} {
			chosen := a.Query.Sort == item.ID || a.Query.Sort == "" && item.ID == "added"
			direction := ""
			if chosen {
				direction = "sort_ascending"
				if a.Query.Descending {
					direction = "sort_descending"
				}
			}
			if a.popSortRow(c.Key(item.ID), a.T(item.Label), direction).Clicked() {
				a.changeSort(item.ID)
			}
		}
	})
}

// popSortRow 是排序弹层的一行：左边勾选位，中间文字，当前字段右边是方向箭头；悬停铺浅底。
// direction 为空表示不是当前字段。
func (a *App) popSortRow(c *ui.Context, label, direction string) ui.Element {
	p := paletteFor(c.Theme())
	chosen := direction != ""
	row := ui.ButtonBase(c).Label(label).Role(ui.RoleRadio).Checked(chosen).FillWidth().Height(popRowHeight).Padding(0, popRowPadRight, 0, popRowPadLeft).Gap(popRowGap).Radius(radiusSmall + 1).Justify(ui.Start).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if chosen {
		row.Description(a.T(direction)).Tooltip(a.T(direction))
	}
	background := ui.Transparent
	if row.Pressed() {
		background = p.Pressed
	} else if row.Hovered() {
		background = p.Hover
	}
	animateColors(row, background, p.Text)
	// 换字段时勾和箭头从旧的一行淡出、在新的一行淡入；反转方向时两支箭头的深浅交换过渡，看得出方向变了。
	row.Children(func() {
		mark := ui.Box(c.Key("mark")).Size(iconSmall, iconSmall).Shrink(0).Center()
		showPose(mark, chosen).Children(func() {
			appSymbol(c, "check").Size(iconSmall, iconSmall).TextColor(p.Accent)
		})
		ui.Text(c, label).FontSize(fontBody).SingleLine().Grow(1).MinWidth(0)
		// 双向箭头：当前方向那支用正文色，另一支淡下去。
		up, down := p.Text, p.TextTertiary.Alpha(.5)
		if direction == "sort_descending" {
			up, down = down, up
		}
		arrows := ui.Box(c.Key("direction")).Size(iconSize, iconSize).Shrink(0)
		showPose(arrows, chosen).Children(func() {
			upIcon := toolbarIcon(c, "sort-up").Absolute().Left(0).Top(0).Size(iconSize, iconSize)
			upIcon.TextColor(upIcon.AnimateColor("ink", up, motionColor))
			downIcon := toolbarIcon(c, "sort-down").Absolute().Left(0).Top(0).Size(iconSize, iconSize)
			downIcon.TextColor(downIcon.AnimateColor("ink", down, motionColor))
		})
	})
	return row
}

func (a *App) changeSort(field string) {
	if a.Query.Scope == "random" {
		return
	}
	if a.Query.Sort == field || a.Query.Sort == "" && field == "added" {
		a.Query.Descending = !a.Query.Descending
	} else {
		a.Query.Sort = field
		// 时间默认新的在前。
		a.Query.Descending = field == "added"
	}
	a.Refine()
}

func (a *App) toolbarExtensions() []string {
	a.refreshToolbarFormats()
	return a.Toolbar.extensions
}

func (a *App) refreshToolbarFormats() {
	state := &a.Toolbar
	if state.extensions != nil && state.formatVersion == a.snapshotVersion {
		return
	}
	unique := map[string]bool{}
	for i := range a.Snapshot.Assets {
		asset := &a.Snapshot.Assets[i]
		ext := strings.TrimPrefix(strings.ToLower(asset.Extension), ".")
		if ext != "" {
			unique[ext] = true
		}
	}
	extensions := make([]string, 0, len(unique))
	for ext := range unique {
		extensions = append(extensions, ext)
	}
	slices.Sort(extensions)
	state.formatVersion, state.extensions = a.snapshotVersion, extensions
}

// filtering 表示有没有颜色、标签、类型、形状筛选条件。
func (a *App) filtering() bool {
	return a.filterCount() > 0
}

func (a *App) clearFilters() {
	a.Query.ColorFamilies, a.Query.TagIDs, a.Query.Extensions, a.Query.Kinds, a.Query.Shapes = nil, nil, nil, nil, nil
}

// clearAllFilters 清空全部筛选并刷新列表；弹层里的「清除筛选」和工具栏按钮上的 × 共用。
func (a *App) clearAllFilters() {
	a.clearFilters()
	a.Refine()
}

// countKey 是筛选面板数量缓存的键：图库或任何条件变了就重算。
func (a *App) countKey(q catalog.Query) string {
	version := a.snapshotVersion
	if q.Scope == "recent" {
		version = a.indexVersion
	}
	return fmt.Sprintf("%d|%s|%s|%s|%s|%v|%v|%v|%v|%v|%v|%s", version, q.Scope, q.FolderID, q.ScopeTagID, q.SmartFolderID, q.ColorFamilies, q.TagIDs, q.Extensions, q.Kinds, q.Shapes, q.IncludeSubfolders, q.Search)
}

// shapeCounts 是筛选面板里各形状的数量，条件和图库没变就用上次算的。
func (a *App) shapeCounts() map[string]int {
	if a.Index == nil {
		return nil
	}
	a.refreshFilterCounts()
	return a.Toolbar.shapeCount
}

// typeCounts 是筛选面板里各扩展名、各种类的数量，缓存方式同 shapeCounts。
func (a *App) typeCounts() (extensions, kinds map[string]int) {
	if a.Index == nil {
		return nil, nil
	}
	a.refreshFilterCounts()
	return a.Toolbar.typeExtCount, a.Toolbar.typeKindCount
}

// tagCounts 是筛选面板里各标签的数量，缓存方式同 shapeCounts。
func (a *App) tagCounts() map[string]int {
	if a.Index == nil {
		return nil
	}
	a.refreshFilterCounts()
	return a.Toolbar.tagCount
}

// colorCounts 是筛选面板里各颜色的数量，缓存方式同 shapeCounts。
func (a *App) colorCounts() map[string]int {
	if a.Index == nil {
		return nil
	}
	a.refreshFilterCounts()
	return a.Toolbar.colorCount
}

func (a *App) refreshFilterCounts() {
	state := &a.Toolbar
	query := a.browseQuery(a.Query)
	shapeQuery, typeQuery, tagQuery, colorQuery := query, query, query, query
	shapeQuery.Shapes = nil
	typeQuery.Extensions, typeQuery.Kinds = nil, nil
	tagQuery.TagIDs = nil
	colorQuery.ColorFamilies = nil
	shapeKey, typeKey, tagKey, colorKey := a.countKey(shapeQuery), a.countKey(typeQuery), a.countKey(tagQuery), a.countKey(colorQuery)
	staleShapes, staleTypes, staleTags, staleColors := state.shapeKey != shapeKey, state.typeKey != typeKey, state.tagKey != tagKey, state.colorKey != colorKey
	if !staleShapes && !staleTypes && !staleTags && !staleColors {
		return
	}

	visibleShapes := staleShapes && len(query.Shapes) == 0
	visibleTypes := staleTypes && len(query.Extensions)+len(query.Kinds) == 0
	visibleTags := staleTags && len(query.TagIDs) == 0
	visibleColors := staleColors && len(query.ColorFamilies) == 0
	if visibleShapes || visibleTypes || visibleTags || visibleColors {
		a.countVisibleFilters(visibleShapes, visibleTypes, visibleTags, visibleColors)
	}
	if staleShapes && !visibleShapes {
		state.shapeCount = a.Index.ShapeCounts(shapeQuery)
	}
	if staleTypes && !visibleTypes {
		state.typeExtCount, state.typeKindCount = a.Index.TypeCounts(typeQuery)
	}
	if staleTags && !visibleTags {
		state.tagCount = a.Index.TagCounts(tagQuery)
	}
	if staleColors && !visibleColors {
		// 当前选了颜色时，别的颜色要按「不算颜色条件」的结果数，所以重新查一遍。
		state.colorCount = map[string]int{}
		for _, asset := range a.Index.Query(colorQuery) {
			addColorCounts(state.colorCount, &asset)
		}
	}
	state.shapeKey, state.typeKey, state.tagKey, state.colorKey = shapeKey, typeKey, tagKey, colorKey
}

// addColorCounts 给素材里出现的每个颜色族各加一（同一族在一张图里出现几次也只算一张）。
func addColorCounts(counts map[string]int, asset *model.Asset) {
	for i, color := range asset.Colors {
		if !slices.ContainsFunc(asset.Colors[:i], func(earlier model.Color) bool { return earlier.Family == color.Family }) {
			counts[color.Family]++
		}
	}
}

func (a *App) countVisibleFilters(shapes, types, tags, colors bool) {
	state := &a.Toolbar
	if shapes {
		state.shapeCount = make(map[string]int, len(catalog.Shapes))
	}
	if types {
		state.typeExtCount, state.typeKindCount = map[string]int{}, map[string]int{}
	}
	if tags {
		state.tagCount = map[string]int{}
	}
	if colors {
		state.colorCount = map[string]int{}
	}
	for i := range a.Visible {
		asset := &a.Visible[i]
		if shapes {
			catalog.AddShapeCounts(state.shapeCount, asset)
		}
		if types {
			ext := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(asset.Extension)), ".")
			state.typeExtCount[ext]++
			state.typeKindCount[asset.Kind]++
		}
		if tags {
			for _, id := range asset.TagIDs {
				state.tagCount[id]++
			}
		}
		if colors {
			addColorCounts(state.colorCount, asset)
		}
	}
}

func toolbarToggle(values []string, value string, on bool) []string {
	at := slices.Index(values, value)
	if on {
		if at < 0 {
			return append(values, value)
		}
		return values
	}
	if at >= 0 {
		return slices.Delete(values, at, at+1)
	}
	return values
}

// 工具栏规格照设计稿，整体比稿子矮一截：控件高 28、圆角 8，底色和文字色都是正文色叠在背景上的比例。
const (
	toolbarControlHeight = 28
	toolbarRadius        = 8
	toolbarChipPadding   = 10 // 带文字按钮左边距（右边 12）
	toolbarChipGap       = 6
	toolbarChipFill      = .043 // 搜索框、按钮、刻度尺、视图切换共用的底色
	toolbarSearchWidth   = 300
	toolbarSearchMin     = 160 // 设计稿是 180；1100 窗口开着侧栏时差 10 点放不下刻度尺，缩到 160
	toolbarSegmentInset  = 2
	toolbarSegmentWidth  = 30
	toolbarTitleSize     = 15
	toolbarFont          = fontSecondary // 按钮、搜索框、「N 列」、导入：比正文小一号
	toolbarHintFont      = fontCaption   // ⌘K 键帽
	toolbarTitleWidth    = 220           // 文件夹名太长时截断
	toolbarCountSize     = 11.5
	toolbarAddIcon       = 16
	toolbarDotSize       = 6
	toolbarClearSlot     = 10 // 筛选按钮在悬停才出现的 × 前预留的宽度
	toolbarClearSize     = 16
	toolbarClearInset    = 3

	// 缩略图刻度尺：刻度固定，左端最小、右端最大，指针在上面左右拖；两档之间 4 格。
	rulerWidth    = 112
	rulerPad      = 8 // 两端第一档离刻度区边缘的距离
	rulerTickTall = 10
	rulerTickLow  = 6
	rulerNeedle   = 14

	// 筛选和排序弹层。
	popFilterWidth        = 300
	popSortWidth          = 180
	popInset              = 16 // 筛选弹层左右留白
	popHeaderHeight       = 44
	popBodyMax            = 420 // 正文超出这个高度就滚动
	popSectionGap         = 18
	popSwatchColumns      = 8
	popSwatchHit          = 28 // 颜色圆点的点击区，也是选中细环的外径
	popSwatch             = 20
	popSwatchRadius       = 6
	popSwatchEmptyOpacity = .3 // 当前位置里没有这个颜色时色块的不透明度
	popChipHeight         = 26
	popChipGap            = 6
	popChipRadius         = 7
	popChipIconGap        = 5
	popChipMaxWidth       = 200 // 标签名太长时截断
	popRowHeight          = 28
	popRowPadLeft         = 6
	popRowPadRight        = 8
	popRowGap             = 6
	popSortPadding        = 4
	popClearPad           = 6  // 「清除筛选」按钮左右内边距
	popTagSearchFrom      = 12 // 标签超过这么多才显示搜索框
)

func (a *App) toolbarWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	return width - a.SidebarShownWidth() - a.InspectorShownWidth()
}

// 缩略图大小是一把刻度尺：刻度固定，左端最小、右端最大；按下指针跳到那里，拖着跟手，松手停到最近一档；右边显示现在是几列。
func (a *App) thumbnailRuler(c *ui.Context) {
	p := paletteFor(c.Theme())
	last := len(thumbnailSteps) - 1
	gap := float32(rulerWidth-2*rulerPad) / float32(last) // 相邻两档的距离
	index := float64(thumbnailStep(a.thumbnailScale()))
	box := ui.Row(c).Height(toolbarControlHeight).Shrink(0).Radius(toolbarRadius).Background(toolbarFill(p, toolbarChipFill)).AlignItems(ui.Center).Clip()
	box.Children(func() {
		// SliderBase 管键盘和读屏；指针交给上面盖着的刻度区。
		slider := ui.SliderBase(c.Key("thumbnail-size"), &index, 0, float64(last)).Step(1).Size(rulerWidth, toolbarControlHeight).Shrink(0).
			Label(a.T("thumbnail_size")).Tooltip(a.T("thumbnail_size"))
		if slider.Changed() {
			a.SetThumbnailSize(thumbnailSteps[int(math.Round(index))])
		}
		at := int(math.Round(index))
		target := float32(at)
		// 指针横坐标换成连续的档位，夹在两端之间。
		toPosition := func(x float32) float32 { return max(0, min(float32(last), (x-rulerPad)/gap)) }
		// 透明的拖动层盖住刻度区。用 Draggable 而不是 Dragged：指针拖出刻度区也一直跟着，拖过的按压也不会被当成点击；
		// 它松手立刻弹回原位，下一次按下从它当时的位移算起。
		scrub := ui.Box(c.Key("scrub")).Absolute().Top(0).Left(0).Bottom(0).Width(rulerWidth).Cursor(ui.CursorResizeEW)
		scrub.Draggable(&a.Toolbar.ScrubOffset, ui.DraggableOptions{Axis: ui.Horizontal, Snap: func(ui.Point) ui.Point { return ui.Point{} },
			Animation: ui.Animation{Duration: time.Millisecond, NonMotion: true}})
		if scrub.PressedInside() {
			a.Toolbar.ScrubBase = a.Toolbar.ScrubOffset.X
			if x, _, ok := scrub.PointerPosition(); ok {
				a.Toolbar.ScrubStart = x
			}
			// 点按刻度区时把焦点交给滑块，方向键、Home、End 接着能用。
			slider.Focus()
		}
		switch {
		case scrub.Dragging():
			// 拖着时指针跟手，缩略图按最近的一档实时变。
			target = toPosition(a.Toolbar.ScrubStart + a.Toolbar.ScrubOffset.X - a.Toolbar.ScrubBase)
			at = int(math.Round(float64(target)))
		case scrub.Clicked():
			if x, _, ok := scrub.PointerPosition(); ok {
				at = int(math.Round(float64(toPosition(x))))
				target = float32(at)
			}
		}
		if _, _, ended := scrub.DragEnded(); ended {
			target = float32(at)
			slider.Focus()
		}
		if at != int(math.Round(index)) {
			a.SetThumbnailSize(thumbnailSteps[at])
		}
		if scrub.Dragging() {
			slider.Jump("ruler-position", target)
		}
		position := slider.AnimateTo("ruler-position", target, motionSpring)
		// 刻度固定；指针左边已经生效的那段从最左很浅的正文色渐变到指针处的品牌色，右边的不上色。
		// 每档一根高刻度，两档之间 3 根矮的。
		faint, idle := toolbarInk(p, .2), toolbarInk(p, .3)
		slider.Draw(func(pt *ui.Painter, r ui.Rect) {
			start := r.X + rulerPad
			needle := start + position*gap
			for j := 0; j <= last*4; j++ {
				x := start + float32(j)*gap/4
				h, color := float32(rulerTickLow), idle
				if j%4 == 0 {
					h = rulerTickTall
				}
				if x <= needle && needle > start {
					color = faint.Mix(p.Accent, (x-start)/(needle-start))
				}
				pt.Fill(ui.Rect{X: x - .5, Y: r.Y + (r.H-h)/2, W: 1, H: h}, color, 0)
			}
			pt.Fill(ui.Rect{X: needle - 1, Y: r.Y + (r.H-rulerNeedle)/2, W: 2, H: rulerNeedle}, p.Accent, 1)
		})
		ui.Box(c).Size(1, rulerNeedle).Shrink(0).Background(toolbarFill(p, .08))
		// 右边写现在是几列，比百分比直观。
		// 列数（或行高）跟着刻度滚动，变多往上滚、变少往下滚。
		label, shown := a.thumbnailLabelValue()
		countMotion(ui.Text(c.Key("columns"), label).FontSize(toolbarFont).FontFeatures("tnum").
			TextColor(toolbarInk(p, .86)).SingleLine().Shrink(0).Padding(0, 11, 0, 10), shown)
	})
}

// thumbnailLabel 是刻度尺右边的字：瀑布流写几列，等高行写行高。
func (a *App) thumbnailLabel() string {
	label, _ := a.thumbnailLabelValue()
	return label
}

// thumbnailLabelValue 同时返回刻度尺右边的字和里面的数。
func (a *App) thumbnailLabelValue() (string, int) {
	if a.ViewMode == 1 {
		height := int(justifiedRowHeight * a.thumbnailScale())
		return a.T("toolbar_row_height", strconv.Itoa(height)), height
	}
	columns := masonryColumns(a.layoutWidth, a.thumbnailScale())
	return a.T("toolbar_columns", strconv.Itoa(columns)), columns
}
