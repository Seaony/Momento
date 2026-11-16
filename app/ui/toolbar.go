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
	ratingKey     string
	ratingCount   [6]int
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
		Background(toolbarColor(t)).BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSubtle).Role(ui.RoleToolbar).FocusGroup(ui.Horizontal).Label(a.T("browse_tools"))
	if bounds := row.LayoutBounds(); bounds.W > 0 {
		width = bounds.W - left - right
	}
	inlineRuler, labels, title, compactSearch := a.toolbarFit(c, width)
	row.Children(func() {
		if !a.SidebarVisible {
			button := toolbarIconButton(c.Key("toggle-sidebar"), a.T("toggle_sidebar"), "sidebar").
				Transition(ui.ElementTransition{Animation: &motionColor, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
			if button.Clicked() {
				a.SidebarVisible = true
			}
		}
		if title {
			a.toolbarTitle(c)
		}
		if compactSearch {
			search := toolbarChip(c.Key("compact-search"), a.T("search_assets"), a.Toolbar.SearchOpen)
			search.Children(func() { toolbarIcon(c, "search") })
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
			a.toolbarUpdate(c)
			a.toolbarAction(c, "new-tag", a.T("new_tag"), labels, a.NewTag)
			return
		}
		// 列表视图用不上缩略图大小：刻度尺（或它的按钮）连同后面的竖线一起收起淡出，回到瀑布流再展开。
		if a.ViewMode == 2 {
			a.Toolbar.ThumbnailOpen = false
		} else {
			ui.Row(c.Key("thumbnail-group")).Shrink(0).Gap(space12).AlignItems(ui.Center).
				Transition(ui.ElementTransition{Animation: &motionSpring, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}).
				Children(func() {
					if inlineRuler {
						a.Toolbar.ThumbnailOpen = false
						a.thumbnailRuler(c.Key("inline-thumbnail-size"))
					} else {
						thumbnail := toolbarChip(c.Key("thumbnail-size-button"), a.T("thumbnail_size"), a.Toolbar.ThumbnailOpen)
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
		filtering := a.filtering()
		filter := toolbarChip(c.Key("filter"), a.T("filter"), filtering || a.Toolbar.FilterOpen)
		filter.Children(func() {
			toolbarIcon(c, "filter")
			if labels {
				ui.Text(c, a.T("filter")).SingleLine()
			}
		})
		if labels {
			filter.Padding(0, space12, 0, toolbarChipPadding)
		}
		if filter.Clicked() {
			a.Toolbar.FilterOpen = !a.Toolbar.FilterOpen
			a.Toolbar.SortOpen = false
			a.Toolbar.ThumbnailOpen = false
			a.Toolbar.ViewOpen = false
		}
		floatingPopover(c.Key("filters"), filter, &a.Toolbar.FilterOpen, func() { a.filterPanel(c) }).Width(popFilterWidth).Padding(0)
		sort := toolbarChip(c.Key("sort"), a.T("sort"), a.Toolbar.SortOpen).Disabled(a.Query.Scope == "random")
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
		if a.Query.Scope == "random" {
			sort.Tooltip(a.T("random_no_sort"))
			a.Toolbar.SortOpen = false
		}
		if sort.Clicked() {
			a.Toolbar.SortOpen = !a.Toolbar.SortOpen
			a.Toolbar.FilterOpen = false
			a.Toolbar.ThumbnailOpen = false
			a.Toolbar.ViewOpen = false
		}
		floatingPopover(c.Key("sorting"), sort, &a.Toolbar.SortOpen, func() { a.sortPanel(c) }).Width(popSortWidth).Padding(popSortPadding)
		if width < 360 {
			icon := []string{"masonry", "justified", "list"}[a.ViewMode]
			view := toolbarChip(c.Key("view-menu"), a.T("view_mode"), a.Toolbar.ViewOpen)
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
		a.toolbarUpdate(c)
		toolbarDivider(c)
		// 回收站里导入没有意义，右上角换成「清空回收站」。
		if a.Query.Scope == "trash" {
			a.emptyTrashButton(c, labels)
		} else {
			a.importButton(c, labels)
		}
	})
}

func (a *App) toolbarUpdate(c *ui.Context) {
	if version := AvailableUpdateVersion(); version != "" {
		update := toolbarChip(c.Key("available-update"), a.T("update"), true).Tooltip(a.T("update_to_version", version))
		update.Children(func() { toolbarIcon(c, "import") })
		if update.Clicked() {
			ShowAvailableUpdate()
		}
	}
}

// 按真实文字宽度决定放得下多少：先把刻度尺收进弹出层，再去掉按钮文字，再藏标题，最后把搜索框收成按钮。
func (a *App) toolbarFit(c *ui.Context, width float32) (inlineRuler, labels, title, compactSearch bool) {
	measure := func(text string, size float32, weight int) float32 {
		return toolbarTextWidth(c, a.Texts.Locale(), text, size, weight)
	}
	name, count := a.browseTitle()
	titleWidth := min(measure(name, toolbarTitleSize, 650), toolbarTitleWidth) + space8 + measure(a.T("toolbar_item_count", formatTagCount(count)), toolbarCountSize, 400)
	// 搜索框能缩到最小宽度，提示文字放不下时截断，所以按最小宽度算。
	search := float32(toolbarSearchMin)
	ruler := rulerWidth + 1 + 21 + measure(a.thumbnailLabel(), toolbarFont, 400)
	chip := func(text string, extra float32) float32 {
		return toolbarChipPadding + 16 + toolbarChipGap + measure(text, toolbarFont, 400) + space12 + extra
	}
	filter, sort := chip(a.T("filter"), 0), chip(a.T("sort"), toolbarChipGap+12-3)
	actionText := a.T("import")
	if a.Query.Scope == "trash" {
		actionText = a.T("empty_trash")
	}
	importLabel := 11 + toolbarAddIcon + toolbarChipGap + measure(actionText, toolbarFont, 600) + 14
	fits := func(inlineRuler, labels, title bool) bool {
		if a.TagPage {
			// 标签管理页只有：（侧栏按钮）、标题、搜索、弹簧、（更新）、新建标签。
			total, items := search, 3
			if !a.SidebarVisible {
				total, items = total+controlHeight, items+1
			}
			if AvailableUpdateVersion() != "" {
				total, items = total+toolbarControlHeight, items+1
			}
			if title {
				total, items = total+titleWidth+space8, items+1
			}
			if labels {
				total += 11 + toolbarAddIcon + toolbarChipGap + measure(a.T("new_tag"), toolbarFont, 600) + 14
			} else {
				total += toolbarControlHeight
			}
			return total+float32(items-1)*space12 <= width
		}
		// 固定会有：搜索、弹簧、刻度尺或它的按钮、两条竖线、筛选、排序、视图切换、导入；列表视图没有刻度尺和它后面那条竖线。
		total, items := search+2*(1+4)+2*toolbarSegmentInset+3*toolbarSegmentWidth+2*2, 9
		list := a.ViewMode == 2
		if list {
			total, items = total-(1+4), items-2
		}
		if !a.SidebarVisible {
			total, items = total+controlHeight, items+1
		}
		if AvailableUpdateVersion() != "" {
			total, items = total+toolbarControlHeight, items+1
		}
		if title {
			total, items = total+titleWidth+space8, items+1
		}
		switch {
		case list:
		case inlineRuler:
			total += ruler
		default:
			total += toolbarControlHeight
		}
		if labels {
			total += filter + sort + importLabel
		} else {
			total += 3 * toolbarControlHeight
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
			ContentTransition(ui.ContentTransition{Animation: &motionSpring})
		ui.Text(c.Key("count"), a.T("toolbar_item_count", formatTagCount(count))).FontSize(toolbarCountSize).FontFeatures("tnum").TextColor(toolbarInk(p, .5)).SingleLine().
			ContentTransition(ui.ContentTransition{Numeric: true, Animation: &motionSpring})
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
	menu := ui.MenuBase(c.Key("import"), &a.Toolbar.ImportOpen)
	if a.Busy {
		a.Toolbar.ImportOpen = false
	}
	a.toolbarActionStyle(c, menu.Trigger.Disabled(a.Busy), "add", a.T("import"), label, a.Toolbar.ImportOpen, false)
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
}

// 工具栏最右边的品牌色主按钮（新建标签）：带圈加号加文字，窄的时候只剩加号，点击执行 action。
func (a *App) toolbarAction(c *ui.Context, key, text string, label bool, action func()) {
	button := ui.ButtonBase(c.Key(key)).Disabled(a.Busy)
	a.toolbarActionStyle(c, button, "add", text, label, false, false)
	if button.Clicked() {
		action()
	}
}

// 回收站页右上角的「清空回收站」：危险色，回收站空了或正在忙时变灰；点了先确认。
func (a *App) emptyTrashButton(c *ui.Context, label bool) {
	empty := a.Index == nil || a.Index.Counts().Trash == 0
	button := ui.ButtonBase(c.Key("empty-trash")).Disabled(a.Busy || empty)
	a.toolbarActionStyle(c, button, "trash", a.T("empty_trash"), label, false, true)
	if button.Clicked() {
		a.ConfirmEmptyTrash()
	}
}

// 工具栏最右边主按钮的外观，导入（菜单按钮）、新建标签和清空回收站共用；danger 用危险色底。
func (a *App) toolbarActionStyle(c *ui.Context, button ui.Element, icon, text string, label, open, danger bool) {
	p := paletteFor(c.Theme())
	button.Label(text).Tooltip(text).Height(toolbarControlHeight).MinWidth(toolbarControlHeight).Shrink(0).
		Radius(toolbarRadius).Gap(toolbarChipGap).AlignItems(ui.Center).Justify(ui.Center).Cursor(ui.CursorPointer).DimDisabled(false)
	background, ink := p.Accent, p.OnAccent
	switch {
	case a.Busy || button.IsDisabled():
		background, ink = toolbarFill(p, toolbarChipFill), toolbarInk(p, .5)
	case danger:
		// 深色下危险色是浅红，字用背景色才看得清。
		background, ink = p.Danger, p.Background
		if button.Pressed() {
			background = background.Mix(p.Text, .15)
		} else if button.Hovered() {
			background = background.Mix(p.Text, .08)
		}
	case button.Pressed():
		background = p.AccentPressed
	case button.Hovered() || open:
		background = p.AccentHover
	}
	animateColors(button, background, ink)
	button.Children(func() {
		toolbarIcon(c, icon).Size(toolbarAddIcon, toolbarAddIcon)
		if label {
			ui.Text(c, text).FontSize(toolbarFont).FontWeight(600).SingleLine()
		}
	})
	if label {
		button.Padding(0, space12+2, 0, toolbarChipPadding+1)
	} else {
		button.Width(toolbarControlHeight).Padding(0)
	}
}

// 工具栏按钮：浅浅一层底色、圆角 10，图标在前、文字在后；没文字时是 34 的方块。
func toolbarChip(c *ui.Context, label string, active bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Height(toolbarControlHeight).MinWidth(toolbarControlHeight).Shrink(0).Padding(0).
		Radius(toolbarRadius).Gap(toolbarChipGap).AlignItems(ui.Center).Justify(ui.Center).FontSize(toolbarFont).Cursor(ui.CursorPointer)
	background, text := toolbarFill(p, toolbarChipFill), toolbarInk(p, .86)
	switch {
	case button.Pressed():
		background, text = toolbarFill(p, .12), p.Text
	case active:
		background, text = toolbarFill(p, .1), p.Text
	case button.Hovered():
		background, text = toolbarFill(p, .075), p.Text
	}
	animateColors(button, background, text)
	return button
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
		if *query != "" {
			if toolbarIconButton(c.Key("clear-search"), a.T("clear_search"), "close").Size(space20, space24).Clicked() {
				*query = ""
				changed()
				a.Search.Focus()
			}
		} else if hint != "" {
			// 快捷键提示是一枚小键帽：比搜索框再深一层的底，字更淡。
			centeredTextHeight(ui.Text(c.Key("search-shortcut-hint"), hint).FontSize(toolbarHintFont).TextColor(toolbarInk(p, .55)).SingleLine().Shrink(0).
				Radius(5).Background(toolbarFill(p, toolbarChipFill+.06)).IgnoreInput(true), 16, 6)
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
			part := seg.Segment(i).Size(toolbarSegmentWidth, toolbarControlHeight-2*toolbarSegmentInset).Padding(0).Center().Radius(toolbarRadius - toolbarSegmentInset).Cursor(ui.CursorPointer).Label(a.T(item.Label)).Tooltip(a.T(item.Label))
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
			clear := ui.ButtonBase(c.Key("clear-filters")).Height(popChipHeight).Padding(0, popClearPad).Radius(radiusSmall).Cursor(ui.CursorPointer).Label(a.T("clear_filters")).Disabled(!active).DimDisabled(false)
			// 次要操作：字比标题小两号、颜色用三级文字色，没有条件时再淡一半。
			background, text := ui.Transparent, p.TextTertiary
			switch {
			case !active:
				text = p.TextTertiary.Alpha(.5)
			case clear.Pressed():
				background, text = p.Pressed, p.Text
			case clear.Hovered():
				background, text = p.Hover, p.Text
			}
			animateColors(clear, background, text)
			clear.Children(func() { ui.Text(c, a.T("clear_filters")).FontSize(fontGroup).SingleLine() })
			if clear.Clicked() {
				a.clearFilters()
				a.Refine()
			}
		})
		ui.Divider(c).Background(p.BorderSubtle)
		ui.Scroll(c.Key("filter-scroll")).Scrollbars(false).FillWidth().MaxHeight(popBodyMax).Padding(popInset-2, popInset, popInset, popInset).Gap(popSectionGap).Children(func() {
			popSection(c, a.T("filter_colors"), func() {
				for start := 0; start < len(toolbarFamilies); start += popSwatchColumns {
					ui.Row(c.Key(start)).FillWidth().Justify(ui.SpaceBetween).Children(func() {
						for _, family := range toolbarFamilies[start:min(start+popSwatchColumns, len(toolbarFamilies))] {
							chosen := slices.Contains(a.Query.ColorFamilies, family.ID)
							if a.colorSwatch(c.Key(family.ID), family.ID, family.Hex, chosen).Clicked() {
								a.Query.ColorFamilies = toolbarToggle(a.Query.ColorFamilies, family.ID, !chosen)
								a.Refine()
							}
						}
					})
				}
			})
			popSection(c, a.T("filter_tags"), func() {
				// 搜索框只在标签多时出现；不显示时也不按旧搜索词过滤，免得标签莫名消失又没处清。
				query := ""
				if len(a.Snapshot.Tags) > popTagSearchFrom {
					query = strings.ToLower(strings.TrimSpace(a.Toolbar.TagSearch))
					field, _ := searchField(c.Key("tag-search"), &a.Toolbar.TagSearch, a.T("search_tags"), a.T("clear_search"))
					// 底色换成和胶囊同一套浅底，边框和聚焦边框照旧。
					field.FillWidth().Margin(0, 0, 2, 0).Background(popFill(p, .05))
				}
				tags := slices.Clone(a.Snapshot.Tags)
				slices.SortFunc(tags, func(x, y model.Tag) int { return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)) })
				tags = slices.DeleteFunc(tags, func(tag model.Tag) bool {
					return query != "" && !strings.Contains(strings.ToLower(tag.Name), query)
				})
				// 没有标签时只留一行浅色说明，不画空的标签行，免得多出一截间距。
				if len(tags) == 0 {
					popEmpty(c, a.T("no_tags"))
					return
				}
				ui.Row(c.Key("tag-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
					for _, tag := range tags {
						chosen := slices.Contains(a.Query.TagIDs, tag.ID)
						if popChip(c.Key("filter-tag-"+tag.ID), &chosen, "", tag.Name).Changed() {
							a.Query.TagIDs = toolbarToggle(a.Query.TagIDs, tag.ID, chosen)
							a.Refine()
						}
					}
				})
			})
			popSection(c, a.T("filter_types"), func() {
				types := a.toolbarExtensions()
				if len(types) == 0 {
					popEmpty(c, a.T("no_file_types"))
					return
				}
				ui.Row(c.Key("type-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
					// 图库里有视频时，最前面放一个「视频」，一次选中所有视频格式。
					if a.libraryHasKind(model.KindVideo) {
						chosen := slices.Contains(a.Query.Kinds, model.KindVideo)
						if popChip(c.Key("kind-video"), &chosen, "type-video", a.T("filter_kind_video")).Changed() {
							a.Query.Kinds = toolbarToggle(a.Query.Kinds, model.KindVideo, chosen)
							a.Refine()
						}
					}
					if a.libraryHasKind(model.KindLive) {
						chosen := slices.Contains(a.Query.Kinds, model.KindLive)
						if popChip(c.Key("kind-live"), &chosen, "type-live", a.T("filter_kind_live")).Changed() {
							a.Query.Kinds = toolbarToggle(a.Query.Kinds, model.KindLive, chosen)
							a.Refine()
						}
					}
					for _, ext := range types {
						chosen := slices.Contains(a.Query.Extensions, ext)
						if popChip(c.Key("extension-"+ext), &chosen, extensionIcon(ext), strings.ToUpper(ext)).Changed() {
							a.Query.Extensions = toolbarToggle(a.Query.Extensions, ext, chosen)
							a.Refine()
						}
					}
				})
			})
			popSection(c, a.T("filter_shapes"), func() {
				counts := a.shapeCounts()
				ui.Row(c.Key("shape-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
					for _, shape := range catalog.Shapes {
						chosen := slices.Contains(a.Query.Shapes, shape)
						// 没有素材的形状不列出来，已经选上的照常显示，免得取消不了。
						if counts[shape] == 0 && !chosen {
							continue
						}
						label := shape
						if !strings.Contains(shape, ":") {
							label = a.T("shape_" + shape)
						}
						if popCountChip(c.Key("shape-"+shape), &chosen, label, counts[shape]).Changed() {
							a.Query.Shapes = toolbarToggle(a.Query.Shapes, shape, chosen)
							a.Refine()
						}
					}
				})
			})
			// 一张都没打过星时不显示评分这一段。
			if counts := a.ratingCounts(); counts != [6]int{counts[0]} || len(a.Query.Ratings) > 0 {
				popSection(c, a.T("filter_rating"), func() {
					ui.Row(c.Key("rating-chips")).FillWidth().Wrap().Gap(popChipGap).Children(func() {
						for rating := 5; rating >= 0; rating-- {
							chosen := slices.Contains(a.Query.Ratings, rating)
							if counts[rating] == 0 && !chosen {
								continue
							}
							if popCountChip(c.Key(fmt.Sprint("rating-", rating)), &chosen, a.ratingLabel(rating), counts[rating]).Changed() {
								if chosen {
									a.Query.Ratings = append(a.Query.Ratings, rating)
								} else {
									a.Query.Ratings = slices.DeleteFunc(slices.Clone(a.Query.Ratings), func(r int) bool { return r == rating })
								}
								a.Refine()
							}
						}
					})
				})
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
func (a *App) colorSwatch(c *ui.Context, id, hex string, chosen bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(a.T("color_"+id)).Tooltip(a.T("color_"+id)).Size(popSwatchHit, popSwatchHit).Shrink(0).Padding(0).Radius(popSwatchRadius + (popSwatchHit-popSwatch)/2).Cursor(ui.CursorPointer).Center()
	button.Role(ui.RoleCheckBox).Checked(chosen)
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

// popChip 是标签和文件类型共用的圆角开关：平时浅底，选中反色；icon 为空时只有文字。
func popChip(c *ui.Context, on *bool, icon, label string) ui.Element {
	p := paletteFor(c.Theme())
	chip := ui.ToggleBase(c, on).Label(label).Height(popChipHeight).Padding(0, space8+1).Gap(popChipIconGap).Radius(popChipRadius).Cursor(ui.CursorPointer).AlignItems(ui.Center).MaxWidth(popChipMaxWidth)
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
		ui.Text(c, label).FontSize(fontSecondary).SingleLine().MinWidth(0)
	})
}

// popCountChip 是带数量的筛选胶囊，数量用淡一级的颜色跟在名字后面。
func popCountChip(c *ui.Context, on *bool, label string, count int) ui.Element {
	p := paletteFor(c.Theme())
	chip := ui.ToggleBase(c, on).Label(fmt.Sprintf("%s %d", label, count)).Height(popChipHeight).Padding(0, space8+1).Gap(space4).Radius(popChipRadius).Cursor(ui.CursorPointer).AlignItems(ui.Center).MaxWidth(popChipMaxWidth)
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
	return chip.Children(func() {
		ui.Text(c, label).FontSize(fontSecondary).SingleLine().MinWidth(0)
		ui.Text(c, fmt.Sprint(count)).FontSize(fontSecondary).FontFeatures("tnum").TextColor(text.Alpha(.55)).SingleLine().Shrink(0)
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
		for _, item := range []struct{ ID, Label string }{{"added", "sort_added"}, {"name", "sort_name"}, {"size", "sort_size"}, {"rating", "sort_rating"}} {
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
	row.Children(func() {
		ui.Box(c).Size(iconSmall, iconSmall).Shrink(0).Center().Children(func() {
			if chosen {
				appSymbol(c, "check").Size(iconSmall, iconSmall).TextColor(p.Accent)
			}
		})
		ui.Text(c, label).FontSize(fontBody).SingleLine().Grow(1).MinWidth(0)
		if chosen {
			// 双向箭头：当前方向那支用正文色，另一支淡下去。
			up, down := p.Text, p.TextTertiary.Alpha(.5)
			if direction == "sort_descending" {
				up, down = down, up
			}
			ui.Box(c).Size(iconSize, iconSize).Shrink(0).Children(func() {
				toolbarIcon(c, "sort-up").Absolute().Left(0).Top(0).Size(iconSize, iconSize).TextColor(up)
				toolbarIcon(c, "sort-down").Absolute().Left(0).Top(0).Size(iconSize, iconSize).TextColor(down)
			})
		}
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
		// 时间和评分默认新的、高的在前。
		a.Query.Descending = field == "added" || field == "rating"
	}
	a.Refine()
}

func (a *App) toolbarExtensions() []string {
	unique := map[string]bool{}
	for _, asset := range a.Snapshot.Assets {
		ext := strings.TrimPrefix(strings.ToLower(asset.Extension), ".")
		if ext != "" {
			unique[ext] = true
		}
	}
	out := make([]string, 0, len(unique))
	for ext := range unique {
		out = append(out, ext)
	}
	slices.Sort(out)
	return out
}

// filtering 表示有没有颜色、标签、类型、形状筛选条件。
func (a *App) filtering() bool {
	return len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions)+len(a.Query.Kinds)+len(a.Query.Shapes)+len(a.Query.Ratings) > 0
}

func (a *App) clearFilters() {
	a.Query.ColorFamilies, a.Query.TagIDs, a.Query.Extensions, a.Query.Kinds, a.Query.Shapes, a.Query.Ratings = nil, nil, nil, nil, nil, nil
}

// ratingLabel 是评分筛选的文字：0 是「未评分」，其余是几星。
func (a *App) ratingLabel(rating int) string {
	switch rating {
	case 0:
		return a.T("rating_none")
	case 1:
		return a.T("rating_one_star")
	}
	return a.T("rating_stars", rating)
}

// countKey 是筛选面板数量缓存的键：图库或任何条件变了就重算。
func (a *App) countKey(q catalog.Query) string {
	return fmt.Sprintf("%d|%s|%s|%s|%v|%v|%v|%v|%v|%v|%v|%s", a.indexVersion, q.Scope, q.FolderID, q.SmartFolderID, q.ColorFamilies, q.TagIDs, q.Extensions, q.Kinds, q.Shapes, q.Ratings, q.IncludeSubfolders, q.Search)
}

// shapeCounts 是筛选面板里各形状的数量，条件和图库没变就用上次算的。
func (a *App) shapeCounts() map[string]int {
	if a.Index == nil {
		return nil
	}
	q := a.browseQuery(a.Query)
	q.Shapes = nil
	if key := a.countKey(q); a.Toolbar.shapeKey != key {
		a.Toolbar.shapeKey, a.Toolbar.shapeCount = key, a.Index.ShapeCounts(q)
	}
	return a.Toolbar.shapeCount
}

// ratingCounts 是筛选面板里 0～5 星各自的数量，缓存方式同 shapeCounts。
func (a *App) ratingCounts() [6]int {
	if a.Index == nil {
		return [6]int{}
	}
	q := a.browseQuery(a.Query)
	q.Ratings = nil
	if key := a.countKey(q); a.Toolbar.ratingKey != key {
		a.Toolbar.ratingKey, a.Toolbar.ratingCount = key, a.Index.RatingCounts(q)
	}
	return a.Toolbar.ratingCount
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

	// 缩略图刻度尺：刻度固定，左端最小、右端最大，指针在上面左右拖；两档之间 4 格。
	rulerWidth    = 112
	rulerPad      = 8 // 两端第一档离刻度区边缘的距离
	rulerTickTall = 10
	rulerTickLow  = 6
	rulerNeedle   = 14

	// 筛选和排序弹层。
	popFilterWidth   = 300
	popSortWidth     = 180
	popInset         = 16 // 筛选弹层左右留白
	popHeaderHeight  = 44
	popBodyMax       = 420 // 正文超出这个高度就滚动
	popSectionGap    = 18
	popSwatchColumns = 8
	popSwatchHit     = 28 // 颜色圆点的点击区，也是选中细环的外径
	popSwatch        = 20
	popSwatchRadius  = 6
	popChipHeight    = 26
	popChipGap       = 6
	popChipRadius    = 7
	popChipIconGap   = 5
	popChipMaxWidth  = 200 // 标签名太长时截断
	popRowHeight     = 28
	popRowPadLeft    = 6
	popRowPadRight   = 8
	popRowGap        = 6
	popSortPadding   = 4
	popClearPad      = 6  // 「清除筛选」按钮左右内边距
	popTagSearchFrom = 12 // 标签超过这么多才显示搜索框
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
		ui.Text(c.Key("columns"), a.thumbnailLabel()).FontSize(toolbarFont).FontFeatures("tnum").
			TextColor(toolbarInk(p, .86)).SingleLine().Shrink(0).Padding(0, 11, 0, 10)
	})
}

// libraryHasKind 判断当前图库里有没有这种素材（不看回收站外的筛选范围，和文件类型列表一致）。
func (a *App) libraryHasKind(kind string) bool {
	for i := range a.Snapshot.Assets {
		if a.Snapshot.Assets[i].Kind == kind {
			return true
		}
	}
	return false
}

// thumbnailLabel 是刻度尺右边的字：瀑布流写几列，等高行写行高。
func (a *App) thumbnailLabel() string {
	if a.ViewMode == 1 {
		return a.T("toolbar_row_height", strconv.Itoa(int(justifiedRowHeight*a.thumbnailScale())))
	}
	return a.T("toolbar_columns", strconv.Itoa(masonryColumns(a.layoutWidth, a.thumbnailScale())))
}
