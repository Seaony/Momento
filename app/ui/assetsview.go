package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

const (
	importProgressHeight    float32 = 120
	thumbnailProgressHeight float32 = 54 // compactProgress 的高度
)

// 重建缩略图、查找相似图片只有标题、张数和进度条，没有导入的计数和当前文件。
func compactProgress(p *model.ImportProgress) bool {
	return p != nil && (p.Stage == "thumbnails" || p.Stage == "similar")
}

func (a *App) progressHeight() float32 {
	if compactProgress(a.Progress) {
		return thumbnailProgressHeight
	}
	return importProgressHeight
}

func (a *App) contentTopInset() float32 {
	if a.Progress != nil {
		return MainToolbarHeight + a.progressHeight()
	}
	return MainToolbarHeight
}

func (a *App) assetArea(c *ui.Context) {
	// 上一帧鼠标已经离开那张视频卡片就关掉悬停播放器（上一帧的 hovered 这时已经算完整）。
	a.finishHoverVideo()
	a.hovered = ""
	// 和标签管理页之间来回切时整页淡入淡出。
	scroll := fadeMotion(ui.Scroll(c.Key("assets")).Scrollbars(false).Grow(1).FillWidth().TrackScroll(&a.scroll))
	inset := scroll.AnimateTo("header-inset", a.contentTopInset(), motionSpring)
	scroll.Padding(inset+space16, space16, space16)
	box := scroll.Bounds()
	width, height := box.W-2*space16, box.H
	if width <= 0 {
		width, height = c.Size()
		width -= 2*space16 + a.SidebarShownWidth() + a.InspectorShownWidth()
		if a.chromeLayoutReady {
			width--
		}
	}
	if height <= 0 {
		height = 640
	}
	width = a.settledAssetWidth(c, width)
	// 文件夹页先排子文件夹卡片，素材整体往下挪 lead；视口换算和键盘滚动都要算上它。
	subfolders := a.subfolderCards()
	lead := subfolderStripHeight(len(subfolders), width)
	a.assetLead = lead
	viewportTop := a.scroll.Y - inset - space16 - lead
	state := ui.Local(scroll, "asset-layout", func() assetLayoutState { return assetLayoutState{} })
	ratios := func() []float32 {
		if state.ratios == nil || state.ratiosVersion != a.visibleVersion {
			state.ratios = make([]float32, len(a.Visible))
			for i, asset := range a.Visible {
				w, h := assetDisplaySize(asset)
				if h > 0 {
					state.ratios[i] = float32(w) / float32(h)
				}
			}
			state.ratiosVersion = a.visibleVersion
		}
		return state.ratios
	}
	// visible 取一段高度里的卡片；itemAt 是第几张在新布局里的位置，切视图时找锚点和原来屏幕上那几张用。
	var visible func(top, bottom float32) []MasonryItem
	var itemAt func(index int) MasonryItem
	var contentHeight float32
	switch a.ViewMode {
	case 2:
		_, contentHeight = a.assetRows(width, 0, 0)
		visible = func(top, bottom float32) []MasonryItem {
			items, _ := a.assetRows(width, top, bottom)
			return items
		}
		itemAt = func(index int) MasonryItem {
			tops := a.list.tops
			return MasonryItem{Index: index, Y: tops[index], Width: width, Height: tops[index+1] - tops[index]}
		}
	case 1:
		layout := a.justifiedLayout(ratios, width)
		visible, contentHeight = layout.Visible, layout.Height()
		itemAt = func(index int) MasonryItem { return layout.items[index] }
	default:
		if a.layout == nil || a.layoutVersion != a.visibleVersion || a.layoutWidth != width {
			a.layout = NewMasonryScaled(ratios(), width, a.thumbnailScale())
			a.layoutWidth, a.layoutVersion = width, a.visibleVersion
		}
		layout := a.layout
		visible, contentHeight = layout.Visible, layout.Height()
		itemAt = func(index int) MasonryItem {
			column, row, _ := layout.locate(index)
			return layout.columns[column][row]
		}
	}
	now := c.Now()
	offset := inset + space16 + lead
	viewChanged := state.ready && state.mode != a.ViewMode
	if viewChanged {
		// 卡片变成列表（或反过来）：屏幕上的锚点卡片留在原来的高度，其余卡片围着它变形。
		// 滚动和卡片走同一条曲线、同样的时长，锚点在屏幕上全程不动；旧内容高度先留着，滚动位置不会被截断跳一下。
		state.morphUntil, state.morphHeight, state.morphTop, state.scrollTarget = now.Add(assetReflowDuration), state.contentHeight, viewportTop, -1
		// 原来屏幕上的那几张在新布局里的序号：变形期间一直排出来，它们才能从原位变过去。
		state.sources = state.sources[:0]
		anchor := -1
		for i, asset := range a.Visible {
			if state.onScreen[asset.ID] {
				state.sources = append(state.sources, i)
			}
			if asset.ID == state.anchorID {
				anchor = i
			}
		}
		if anchor >= 0 {
			item := itemAt(anchor)
			// 锚点那张在新布局里要完整露在工具栏下方：原来只露下半截的高图，变成矮的一行后不能整行跑到屏幕外。
			screen := max(inset, min(state.anchorOffset, height-item.Height))
			target := max(0, min(item.Y-screen+offset, offset+contentHeight+space16-height))
			state.morphTop = target - offset
			a.scroll.ScrollTo(0, target, ui.Animation{Duration: assetReflowDuration, Ease: assetReflowEase})
			// mygo 这一帧还按旧内容的滚动上限截断目标；新内容更长、目标超出旧上限时，下一帧按剩下的时长再滚一次。
			if target > a.scroll.MaxY {
				state.scrollTarget = target
			}
		}
	} else if state.scrollTarget >= 0 && now.Before(state.morphUntil) {
		a.scroll.ScrollTo(0, state.scrollTarget, ui.Animation{Duration: state.morphUntil.Sub(now), Ease: assetReflowEase})
		state.scrollTarget = -1
	}
	morphing := now.Before(state.morphUntil)
	boxHeight := contentHeight
	var items []MasonryItem
	if morphing {
		// 变形途中只排三处：当前视口、滚动终点的视口、原来屏幕上的那几张。中间跨过的卡片不排，图库再大、滚得再深，每帧也只有几屏的量。
		boxHeight = max(contentHeight, state.morphHeight)
		c.After(state.morphUntil.Sub(now))
		seen := map[int]bool{}
		add := func(item MasonryItem) {
			if !seen[item.Index] {
				seen[item.Index] = true
				items = append(items, item)
			}
		}
		for _, item := range visible(max(0, viewportTop-height), viewportTop+2*height) {
			add(item)
		}
		for _, item := range visible(max(0, state.morphTop-height), state.morphTop+2*height) {
			add(item)
		}
		for _, index := range state.sources {
			add(itemAt(index))
		}
	} else {
		items = visible(max(0, viewportTop-height), viewportTop+2*height)
	}
	state.contentHeight = contentHeight
	if !morphing {
		state.trackAnchor(items, viewportTop, inset, height, a.Selected.Has, a.Visible)
	}
	reflow := state.prepare(a.ViewMode, width, a.thumbnailScale())
	var openFolder string
	var subfolderToggled bool
	scroll.Children(func() {
		if len(subfolders) > 0 {
			openFolder, subfolderToggled = a.subfolderStrip(c, subfolders, width)
		}
		// 换浏览位置、滚动中途换了筛选（滚动回到顶部）时整块换掉：旧的淡出、新的淡入，卡片不从旧坐标飞过来。
		// 换视图和调缩略图大小一样，容器不换，每张卡片从原来的位置和大小变到新的；同一位置里的筛选、排序、增删按身份移动卡片。
		mode := ui.Box(c.Key(assetModeID{a.browseKey(), a.contentEpoch})).Height(boxHeight).FillWidth().Shrink(0).
			Transition(ui.ElementTransition{Duration: motionEnter, ExitDuration: motionExit, Colors: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
		mode.Children(func() {
			for _, item := range items {
				asset := a.Visible[item.Index]
				card := ui.Column(c.Key(asset.ID)).Absolute().Left(item.X).Top(item.Y).Size(item.Width, item.Height)
				switch {
				case reflow || viewChanged:
					// 缩略图大小、栏宽、窗口宽度、视图变了：每张卡片从原来的位置和大小弹到新的，图片跟着变宽变窄。
					card.Transition(ui.ElementTransition{Duration: assetReflowDuration, Ease: assetReflowEase, ExitDuration: motionExit, Position: true, Size: true,
						Enter: &ui.Motion{}, Exit: &ui.Motion{}})
				case item.Y+item.Height > viewportTop+inset && item.Y < viewportTop+height:
					// 筛选、排序、增删：屏幕里的卡片按身份移过去。
					card.Transition(ui.ElementTransition{Duration: motionEnter, ExitDuration: motionExit, Position: true,
						Enter: &ui.Motion{}, Exit: &ui.Motion{}})
				default:
					card.Transition(ui.ElementTransition{Duration: -1, Position: true, Size: true})
				}
				card.Children(func() { a.assetCard(c, asset, item.Index, a.ViewMode == 2, item.Width, viewChanged) })
			}
		})
		if mode.Clicked() {
			a.clearAssetSelection()
		}
	})
	if scroll.Clicked() {
		a.clearAssetSelection()
	}
	scroll.ContextMenu(a.blankContextMenu)
	if subfolderToggled {
		a.queuePreferences()
		a.Refine()
	}
	if openFolder != "" {
		a.SetScope("folder", openFolder)
	}
	a.assetPreviewShortcut(c, scroll)
	a.assetArrowKeys(c, inset, height)
	if len(a.Visible) == 0 {
		a.emptyState(c, inset+lead)
	}
}

// 没有素材可显示时的空状态：按「为什么空」写标题，只给此刻用得上的按钮。
func (a *App) emptyState(c *ui.Context, top float32) {
	p := paletteFor(c.Theme())
	search, filters := a.Query.Search != "", a.filtering()
	importing := a.Progress != nil && !compactProgress(a.Progress)
	art, title, hint := "scope-empty", a.T("scope_empty"), ""
	var actions func()
	switch {
	case search && filters:
		art, title, hint = "search-empty", a.T("search_empty", a.Query.Search), a.T("search_empty_hint")
		actions = func() {
			if dialogPrimaryButton(c, a.T("clear_search_and_filters"), false).Clicked() {
				a.Query.Search = ""
				a.clearFilters()
				a.Refine()
			}
		}
	case search:
		art, title, hint = "search-empty", a.T("search_empty", a.Query.Search), a.T("search_empty_keyword_hint")
		actions = func() {
			if dialogPrimaryButton(c, a.T("clear_search"), false).Clicked() {
				a.Query.Search = ""
				a.Refine()
			}
			// 搜索只在当前位置里找；不在「全部」时给一条去全部里找的路。
			if a.Query.Scope != "all" && a.Query.Scope != "" && dialogButton(c, a.T("search_in_all"), false).Clicked() {
				a.SetScope("all", "")
			}
		}
	case filters:
		art, title = "search-empty", a.T("filter_empty_title")
		actions = func() {
			if dialogPrimaryButton(c, a.T("clear_filters"), false).Clicked() {
				a.clearFilters()
				a.Refine()
			}
		}
	case a.Query.Scope == "trash":
		art, title, hint = "trash-empty", a.T("trash_empty_title"), a.T("trash_empty_hint")
	case a.Query.Scope == "recent":
		// 说明什么算「用过」，不然看不出为什么是空的。
		hint = a.T("recent_empty_hint")
	case len(a.Snapshot.Assets) == 0:
		art = "empty-library"
		if importing {
			title = a.T("importing_empty")
			break
		}
		title, hint = a.T("drop_images"), a.T("drop_images_hint")
		actions = func() {
			if dialogPrimaryButton(c, a.T("import_images_button"), false).Clicked() {
				a.ChooseImport(false)
			}
			if dialogButton(c, a.T("import_folder_button"), false).Clicked() {
				a.ChooseImport(true)
			}
		}
	case a.Query.Scope == "folder":
		if importing {
			title = a.T("importing_empty")
			break
		}
		title, hint = a.T("folder_empty_title"), a.T("folder_empty_hint")
		actions = func() {
			if dialogPrimaryButton(c, a.T("import"), false).Clicked() {
				a.ChooseImport(false)
			}
		}
	case a.Query.Scope == "favorites":
		title, hint = a.T("favorites_empty_title"), a.T("favorites_empty_hint")
	case a.Query.Scope == "uncategorized" || a.Query.Scope == "untagged":
		title = a.T("all_organized")
	}
	fadeMotion(ui.Column(c.Key("asset-empty")).Absolute().Top(top).Left(space24).Right(space24).Bottom(space24).
		Center().Gap(space12).PassThrough()).Children(func() {
		emptyArt(c.Key("empty-art"), art, emptyArtLarge).Margin(0, 0, space4, 0)
		ui.Text(c.Key("empty-title"), title).FontSize(fontTitle).FontWeight(600).TextColor(p.Text).TextAlign(ui.Center)
		if hint != "" {
			ui.Text(c.Key("empty-hint"), hint).FontSize(fontBody).TextColor(p.TextSecondary).TextAlign(ui.Center)
		}
		if actions != nil {
			ui.Row(c.Key("empty-actions")).Gap(space8).AlignItems(ui.Center).Children(actions)
		}
	})
}

const assetReflowDuration = motionLayout

// 起步就能看到在动，中段稳稳走过去，尾段慢慢收住：40ms 走一成，一半时间走八成，七成时间走到九成五。
// 原来那条前段太猛，三分之一的时间就走完八成多，看起来像一下跳过去。和栏宽同时在 motionLayout 准时落定。
var assetReflowEase = ui.CubicBezier(.3, .2, .2, 1)

// 列表行：缩略图宽固定 112，行高跟着缩略图的实际高度走，上下各留 12；名称和尺寸两行至少要 40 高。
const (
	listRowRadius     float32 = radiusPanel // 悬停底色和选中框的圆角
	listThumbnail     float32 = 72
	listRowPadding    float32 = space12
	listRowMinContent float32 = 40
	listMetaGap               = 7 // 标题和下面尺寸、大小一行的间距
	listFavoriteInset         = 4 // 收藏按钮离缩略图左上角的距离（缩略图比卡片小，比卡片的 8 收一点）
)

// 列表第二行的一项：小图标加文字，和预览底栏同一套图标，字号 11、三级文字色。
func listInfo(c *ui.Context, icon, text string) {
	color := paletteFor(c.Theme()).Text.Alpha(themeValue(c, .42, .48))
	ui.Row(c).Gap(previewInfoGap).AlignItems(ui.Center).Children(func() {
		appIcon(c, icon).Size(fontGroup, fontGroup).Shrink(0).TextColor(color)
		ui.Text(c, text).FontSize(fontGroup).FontFeatures("tnum").TextColor(color).SingleLine()
	})
}

func listRowHeight(asset model.Asset) float32 {
	_, h := listThumbnailSize(assetDisplaySize(asset))
	return max(h, listRowMinContent) + 2*listRowPadding
}

// 长边贴满方框，短边按原图比例；读不到尺寸时按正方形。
func listThumbnailSize(width, height int) (float32, float32) {
	if width <= 0 || height <= 0 {
		return listThumbnail, listThumbnail
	}
	if width >= height {
		return listThumbnail, max(space4, listThumbnail*float32(height)/float32(width))
	}
	return max(space4, listThumbnail*float32(width)/float32(height)), listThumbnail
}

// 卡片左上角的心和右上角的尺寸角标照旧版：高 16，红心用系统红。
const assetBadgeHeight = 16

// 心形按钮至少 24×24 才好点；位置按原来 22×16 的中心对齐，心的中心不动。
const (
	favoriteButtonSize float32 = 24
	favoriteHitExtra   float32 = favoriteButtonSize - assetBadgeHeight
)

// 尺寸、时长、实况角标共用的底色，压在亮色照片上也要看得清白字。
var assetBadgeFill = ui.RGBA(0, 0, 0, .62)

var favoriteRed = ui.Hex("#FF3B30")

// 素材区整块内容的身份：视图、浏览位置、内容批次任一变了就整块换掉。
// morphDuration：切视图那一帧和卡片变形一样长，其他时候立即生效（只为让 mygo 记住上一帧）。
func morphDuration(morph bool) time.Duration {
	if morph {
		return assetReflowDuration
	}
	return -1
}

// arriveMotion：出现时淡入，消失时直接撤掉。切视图时目标视图才有的内容（列表文字、网格角标）用它：
// 等卡片变形走过四成再淡入，和变形同时落定；不压在还没缩小的大图上，也不挂在还在变宽的卡片角上。
func arriveMotion(e ui.Element) ui.Element {
	delay := assetReflowDuration * 2 / 5
	return e.Transition(ui.ElementTransition{Animation: &ui.Animation{Duration: assetReflowDuration - delay, Delay: delay}, Shadows: true, Enter: &ui.Motion{}})
}

type assetModeID struct {
	location string
	epoch    uint64
}

type assetLayoutState struct {
	ready         bool
	mode          int
	width, scale  float32
	ratiosVersion uint64
	ratios        []float32
	contentHeight float32
	// 切视图时以哪张为锚：选中的那张在屏幕上就用它，否则用屏幕上最靠上的一张；offset 是它离视口顶部多远。
	anchorID     string
	anchorOffset float32
	// onScreen 是上一帧屏幕上的卡片；切视图时记成 sources（新布局里的序号），变形期间一直排出来。
	onScreen map[string]bool
	sources  []int
	// 变形进行到什么时候、变形前的内容高度、滚动要去的视口顶部；scrollTarget 是下一帧要补滚的位置（没有时为 -1）。
	morphUntil   time.Time
	morphHeight  float32
	morphTop     float32
	scrollTarget float32
}

// trackAnchor 每帧记下屏幕上的卡片和锚点卡片，切视图那一帧用它们对齐。只看屏幕里的（工具栏下方到窗口底部）。
func (s *assetLayoutState) trackAnchor(items []MasonryItem, viewportTop, inset, height float32, selected func(string) bool, assets []model.Asset) {
	s.anchorID = ""
	if s.onScreen == nil {
		s.onScreen = map[string]bool{}
	}
	clear(s.onScreen)
	best, chosen := -1, false
	for i, item := range items {
		if item.Y+item.Height <= viewportTop+inset || item.Y >= viewportTop+height {
			continue
		}
		id := assets[item.Index].ID
		s.onScreen[id] = true
		switch {
		case chosen:
		case selected(id):
			best, chosen = i, true
		case best < 0 || item.Y < items[best].Y || item.Y == items[best].Y && item.X < items[best].X:
			best = i
		}
	}
	if best >= 0 {
		s.anchorID, s.anchorOffset = assets[items[best].Index].ID, items[best].Y-viewportTop
	}
}

// 同一视图里布局宽度或缩略图大小变了，返回 true，这一帧起卡片按弹簧变到新布局。
func (s *assetLayoutState) prepare(mode int, width, scale float32) bool {
	reflow := s.ready && s.mode == mode && (s.width != width || s.scale != scale)
	s.ready, s.mode, s.width, s.scale = true, mode, width, scale
	return reflow
}

// 侧栏、检查器展开收起时，直接按它们落定后的宽度排版：卡片和栏宽走同一条弹簧一次到位，不在半路反复换列。
// 拖分隔条时栏宽直接跟手，显示宽度就是目标宽度，这里不改变什么。
func (a *App) settledAssetWidth(c *ui.Context, width float32) float32 {
	sidebar := float32(0)
	if a.SidebarVisible {
		sidebar = a.SidebarWidth
	}
	if !a.chromeLayoutReady || a.SidebarShownWidth() == sidebar && a.InspectorShownWidth() == a.InspectorTargetWidth() {
		return width
	}
	window, _ := c.Size()
	return window - 2*space16 - sidebar - a.InspectorTargetWidth() - 1
}

func (a *App) clearAssetSelection() {
	a.Selected.Clear()
	a.selectedIndex, a.anchor = -1, -1
	a.CollapseInspector()
	a.FollowPreviewSelection()
}

func (a *App) selectAsset(id string, index int, mods ui.Modifiers) {
	switch {
	case mods&ui.Shift != 0 && a.anchor >= 0 && a.anchor < len(a.Visible):
		if mods&ui.Cmd == 0 {
			a.Selected.Clear()
		}
		for i := min(a.anchor, index); i <= max(a.anchor, index); i++ {
			a.Selected.Add(a.Visible[i].ID)
		}
	case mods&ui.Cmd != 0:
		if a.Selected.Has(id) {
			a.Selected.Remove(id)
		} else {
			a.Selected.Add(id)
		}
		a.anchor = index
	default:
		a.Selected.Clear()
		a.Selected.Add(id)
		a.anchor = index
	}
	a.selectedIndex = index
	if a.Selected.Len() == 0 {
		a.CollapseInspector()
	} else {
		a.RevealInspector()
	}
	a.FollowPreviewSelection()
}

// morph 是切换视图的那一帧：缩略图、收藏按钮从这种视图的位置和大小变到另一种，阴影跟着淡入淡出。
func (a *App) assetCard(c *ui.Context, asset model.Asset, index int, list bool, frameWidth float32, morph bool) {
	p := paletteFor(c.Theme())
	width, height := assetDisplaySize(asset)
	selected := a.Selected.Has(asset.ID)
	// 两种视图是同一个卡片元素（同一种容器、同一个 key），切换时里面的缩略图才能接着变形，而不是换成另一张。
	card := ui.Column(c.Key(asset.ID)).Fill()
	if list {
		card.Radius(listRowRadius).BorderWidth(0, 0, 1, 0).BorderColor(p.hairline())
	} else {
		card.Radius(radiusCard).FocusRing(false).Shadow(0, 1, 2, 0, p.Shadow)
	}
	// 过渡每帧都要挂上（平时立即生效），mygo 才记得上一帧的样子；切视图那一帧换成和卡片变形同样的时长。
	card.Transition(ui.ElementTransition{Duration: morphDuration(morph), Ease: assetReflowEase, Shadows: true})
	card.Label(asset.Name).Focusable().AccessSelected(selected).Cursor(ui.CursorPointer)
	if a.focusAssetID == asset.ID {
		a.focusAssetID = ""
		card.Focus()
	}
	focused := card.FocusVisible()
	wasFocused := ui.Local(card, "keyboard-focus", func() bool { return false })
	revealFocus := focused && !*wasFocused
	*wasFocused = focused
	hovered := card.Hovered()
	if hovered {
		a.hovered = asset.ID
	}
	background := ui.Transparent
	if list {
		switch {
		case selected && hovered:
			background = p.Hover
		case selected:
			background = p.Selected
		case card.Pressed():
			background = p.Pressed
		case hovered:
			background = p.Hover
		}
	}
	animateColors(card, background, p.Text)
	// 图片卡悬停轻轻浮起；列表是整行，浮起会和行间分隔线错开，只换底色。
	pose := ui.Pose{}
	if hovered && !list {
		pose.Y = -2
	}
	card.AnimatePose(pose, motionSpring)
	card.Children(func() {
		if revealFocus {
			// 把页头遮住的高度算进定位目标，键盘聚焦时图片仍露在页头下方。
			ui.Box(c.Key("focus-reveal")).Absolute().Top(-a.contentTopInset() - space8).Left(0).Right(0).Bottom(0).
				PassThrough().ScrollIntoView()
		}
		thumb := a.Bitmap(asset)
		radius := float32(radiusCard)
		// 缩略图：网格里铺满卡片，列表里缩进行左边 72 宽的方框（按原图比例，长边贴满，居中）。
		// 两种视图是同一个框，切换时从一种的位置和大小变到另一种：用户看到的是卡片缩成了列表里的缩略图。
		slot := ui.Box(c.Key("thumbnail-slot")).Absolute()
		if list {
			w, h := listThumbnailSize(width, height)
			slot.Left(listRowPadding+(listThumbnail-w)/2).Top(listRowPadding+(max(h, listRowMinContent)-h)/2).Size(w, h)
		} else {
			slot.Left(0).Top(0).Right(0).Bottom(0)
		}
		slot.Transition(ui.ElementTransition{Duration: morphDuration(morph), Ease: assetReflowEase, Position: true, Size: true})
		fade := arrival(card, "thumbnail-arrival", thumb != nil)
		arriving := fade.track(c, thumb != nil)
		slot.Children(func() {
			if thumb == nil || arriving {
				placeholder := ui.Box(c.Key("thumbnail-placeholder")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Radius(radius).Background(p.Subtle)
				if list {
					placeholder.Shadow(0, 1, 2, 0, p.Shadow)
				}
			}
			if thumb != nil {
				image := ui.Image(c.Key("thumbnail"), thumb).Fill().Radius(radius)
				// 列表里图片按原图比例完整显示，带一层浅阴影；网格里裁满卡片。
				if list {
					image.Fit(ui.Contain).Shadow(0, 1, 2, 0, p.Shadow)
				} else {
					image.Fit(ui.Cover)
				}
				fade.pose(c, image)
			} else if w, h := listThumbnailSize(width, height); !list || w >= thumbnailFailureMin && h >= thumbnailFailureMin {
				a.thumbnailFailure(c, asset)
			}
		})
		if list {
			// 名称、尺寸、时间单独一层，切到列表时淡入；切回网格时直接撤掉，不留残影跟着卡片飘。
			arriveMotion(ui.Row(c.Key("list-row"))).Absolute().Left(0).Top(0).Right(0).Bottom(0).Gap(space16).Padding(listRowPadding, space12).Children(func() {
				ui.Box(c.Key("thumbnail-space")).Width(listThumbnail).Shrink(0)
				// 照旧版 macOS 列表行的字色层次：标题 13 号正文色约 72%，下面尺寸、大小一行 11 号约 42%（各带一个小图标），右边时间 12 号约 56%；深色各提亮一点。
				ui.Column(c.Key("list-meta")).Grow(1).MinWidth(0).Gap(listMetaGap).Children(func() {
					ui.Text(c, previewFileName(asset)).FontSize(fontBody).FontWeight(500).TextColor(p.Text.Alpha(themeValue(c, .72, .8))).SingleLine()
					dimensions, size := previewDetails(asset)
					ui.Row(c.Key("list-values")).Gap(space12).AlignItems(ui.Center).Children(func() {
						if dimensions != "" {
							listInfo(c, "maximize-square", dimensions)
						}
						if size != "" {
							listInfo(c, "diskette", size)
						}
						if d := assetDuration(asset); isVideoAsset(asset) && d > 0 {
							listInfo(c, "play", videoDuration(d))
						}
						if isLiveAsset(asset) {
							listInfo(c, "live-photo", a.T("live_photo"))
						}
					})
				})
				// 最右边是导入时间：一个月内写「3 天前」，更早写日期，悬停看完整时间。
				added := asset.CreatedAt.Local()
				ui.Text(c.Key("list-added"), a.Texts.Relative(added, time.Now())).FontSize(fontSecondary).FontWeight(500).FontFeatures("tnum").
					TextColor(p.Text.Alpha(themeValue(c, .56, .64))).SingleLine().Shrink(0).Tooltip(a.Texts.DateTime(added))
			})
		} else {
			a.hoverVideoOverlay(c, card, asset, hovered, radius)
			// 视频右下角：播放标记加时长，和尺寸角标同一套样式。
			if isVideoAsset(asset) {
				arriveMotion(ui.Row(c.Key("duration"))).Absolute().Bottom(space8).Right(space8).Height(assetBadgeHeight).Padding(0, 5).Gap(3).AlignItems(ui.Center).
					Radius(5).Background(assetBadgeFill).PassThrough().Children(func() {
					appIconSelected(c, "play", true).Size(8, 8).TextColor(ui.RGBA(255, 255, 255, .82))
					// 读不出时长的视频（有些录屏 WebM 不写时长）只显示播放标记。
					if d := assetDuration(asset); d > 0 {
						ui.Text(c, videoDuration(d)).FontFeatures("tnum").FontSize(10).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, .82))
					}
				})
			}

			// 实况照片右下角：实况标记加「实况」，和视频时长同一个位置、同一套样式。
			if isLiveAsset(asset) {
				arriveMotion(ui.Row(c.Key("live"))).Absolute().Bottom(space8).Right(space8).Height(assetBadgeHeight).Padding(0, 5).Gap(3).AlignItems(ui.Center).
					Radius(5).Background(assetBadgeFill).PassThrough().Children(func() {
					appIcon(c, "live-photo").Size(12, 12).TextColor(ui.RGBA(255, 255, 255, .82))
					ui.Text(c, a.T("live_photo")).FontSize(10).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, .82))
				})
			}

			// 尺寸角标照旧版：常驻右上角，黑色半透明底、白字。读不出尺寸的（Windows、Linux 上的 RAW 等）不显示。
			// 角标和上面的时长、实况标记切回网格时淡入；切到列表时直接撤掉。
			if width > 0 && height > 0 {
				badge := arriveMotion(ui.Text(c.Key("dimensions"), fmt.Sprintf("%d × %d", width, height))).Absolute().Top(space8).Right(space8).
					Radius(5).Background(assetBadgeFill).
					FontFeatures("tnum").FontSize(10).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, .82)).PassThrough()
				centeredTextHeight(badge, assetBadgeHeight, 3)
			}
		}
		if selected || focused {
			// 图片外圈留 2px；列表描边放在行内，避免盖住相邻行。
			ring := fadeMotion(ui.Box(c.Key("selection-ring")).Absolute().Left(-space4).Top(-space4).Right(-space4).Bottom(-space4).
				Radius(radiusPanel).Border(2, p.Accent).PassThrough())
			if list {
				ring.Left(2).Top(2).Right(2).Bottom(2).Radius(listRowRadius - 2)
			}
		}
		if !asset.Trashed && (hovered || asset.Favorite) {
			// 这张在多选里时，心形对全部选中的生效；已收藏时文字和提示写「取消收藏」。
			favoriteIDs := []string{asset.ID}
			if hovered && a.Selected.Len() > 1 && a.Selected.Has(asset.ID) {
				favoriteIDs = a.selectedIDs()
			}
			favoriteLabel := a.T("favorite")
			if !a.FavoriteValue(favoriteIDs) {
				favoriteLabel = a.T("unfavorite")
			}
			button := ui.ButtonBase(c.Key("favorite")).Label(favoriteLabel).Tooltip(favoriteLabel).Absolute().Cursor(ui.CursorPointer).
				Justify(ui.Center).AlignItems(ui.Center).Padding(0)
			// 旧版：图片左上角一颗白色心，不带底；鼠标移到心上才出黑色半透明底，收藏后变成红色实心。悬停才出现，收藏了常驻。
			if list {
				// 列表里贴着缩略图（在 72 宽的框里居中）的左上角；很扁的全景图放不下就在图里上下居中。
				w, h := listThumbnailSize(width, height)
				left := listRowPadding + (listThumbnail-w)/2
				top := listRowPadding + (max(h, listRowMinContent)-h)/2
				button.Top(top+min(listFavoriteInset, (h-assetBadgeHeight)/2)-favoriteHitExtra/2).Left(left+listFavoriteInset-2).Size(favoriteButtonSize, favoriteButtonSize).Radius(favoriteButtonSize / 2)
			} else {
				button.Top(space8-favoriteHitExtra/2).Left(6).Size(favoriteButtonSize, favoriteButtonSize).Radius(favoriteButtonSize / 2)
			}
			favoriteFocused := button.FocusVisible()
			wasFavoriteFocused := ui.Local(button, "keyboard-focus", func() bool { return false })
			if favoriteFocused && !*wasFavoriteFocused {
				ui.Box(c.Key("favorite-focus-reveal")).Absolute().Top(-a.contentTopInset() - space8).Left(0).Right(0).Bottom(0).
					PassThrough().ScrollIntoView()
			}
			*wasFavoriteFocused = favoriteFocused
			if morph {
				// 切视图时心形跟着缩略图挪到新位置，不是原地消失再出现。
				button.Transition(ui.ElementTransition{Duration: assetReflowDuration, Ease: assetReflowEase, ExitDuration: motionExit, Position: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
			} else {
				fadeMotion(button)
			}
			ink, background := ui.RGBA(255, 255, 255, .92), ui.Transparent
			if asset.Favorite {
				ink = favoriteRed
			}
			if button.Hovered() {
				background = ui.RGBA(0, 0, 0, .3)
			}
			animateColors(button, background, ink)
			// 收藏和取消时心形换成实心或空心，和颜色一起过渡。
			button.Children(func() { appIconSelected(c, "heart", asset.Favorite).Size(14, 14).ContentTransition(replaceMotion) })
			if button.Clicked() {
				a.ToggleFavorite(favoriteIDs)
			}
		}
	})
	if card.Shortcut(0, ui.KeyEnter) {
		a.selectAsset(asset.ID, index, c.Modifiers())
		a.ActivateAsset(asset.ID, false)
	} else if card.Clicked() {
		if !a.InspectorVisible && c.Modifiers() == 0 {
			a.assetClickID, a.assetClickBounds = asset.ID, card.Bounds()
		}
		a.selectAsset(asset.ID, index, c.Modifiers())
	}
	if card.DoubleClicked() {
		if !a.Selected.Has(asset.ID) {
			a.selectAsset(asset.ID, index, 0)
		}
		a.ActivateAsset(asset.ID, false)
	}
	if a.Library != nil {
		card.Drag(AssetsDrag{LibraryID: a.libraryID(), assets: func() []string {
			if a.Selected.Has(asset.ID) {
				return a.selectedIDs()
			}
			return []string{asset.ID}
		}})
		root := a.Library.Root
		AttachFileDrag(card, root, func() []model.Asset {
			if a.Selected.Has(asset.ID) {
				return a.SelectedAssets()
			}
			return []model.Asset{asset}
		}, a.ShowError, a.MarkAssetsUsed, a.reportDragCleanup)
	}
	// 右键菜单分五组：查看打开、整理、复制导出、重新处理、删除；回收站里的素材没有整理和重新处理两组。
	card.ContextMenu(func(m *ui.Menu) { a.assetContextMenu(m, asset, index) })
}

// 素材区空白处的右键菜单：导入、粘贴、全选；回收站里只有全选。
func (a *App) blankContextMenu(m *ui.Menu) {
	if a.Query.Scope != "trash" {
		if menuItem(m, a.T("import_menu"), "download-simple").Disabled(a.Busy).Chosen() {
			a.ChooseImport(false)
		}
		if menuItem(m, a.T("shortcut_paste"), "clipboard-add").Chosen() {
			a.PasteImport()
		}
	}
	if len(a.Visible) > 0 && menuItem(m, a.T("select_all"), "squares-four").Chosen() {
		a.SelectAllVisible()
	}
}

// 右键一张没选中的素材，先把选择换成它（和 Finder 一样），菜单里的操作都作用于当前选中的全部素材。
// 只有「用默认应用打开」「复制图片」这类一次只能对一张做的项，多选时不显示；「在文件管理器中显示」显示被右键的那张。
func (a *App) assetContextMenu(m *ui.Menu, asset model.Asset, index int) {
	if !a.Selected.Has(asset.ID) {
		a.selectAsset(asset.ID, index, 0)
	}
	ids := a.selectedIDs()
	multi := len(ids) > 1
	if menuItem(m, a.T("preview"), "eye").Chosen() {
		a.ShowPreview(asset.ID, false)
	}
	if !isVideoAsset(asset) && menuItem(m, a.T("start_slideshow_here"), "play").Chosen() {
		a.StartSlideshow(asset.ID)
	}
	if !multi && menuItem(m, a.T("open_default_app"), "arrow-square-out").Chosen() {
		a.OpenAssetExternally(asset)
	}
	if menuItem(m, a.T("reveal"), "folder-open").Chosen() {
		a.Reveal(asset)
	}
	if !asset.Trashed {
		m.Separator()
		favorite := a.T("unfavorite")
		if a.FavoriteValue(ids) {
			favorite = a.T("favorite")
		}
		if menuItem(m, favorite, "heart").Chosen() {
			a.ToggleFavorite(ids)
		}
		if menuItem(m, a.T("rename_assets"), "text-field-focus").Chosen() {
			// 一张就是直接改名；多张才用批量重命名的格式和替换。
			if len(ids) == 1 {
				a.RenameAsset(asset)
			} else {
				a.OpenRename(a.orderedIDs(ids))
			}
		}
		if menuItem(m, a.T("add_to_folder_menu"), "folder").Chosen() {
			a.OpenFolderPicker()
		}
		if menuItem(m, a.T("new_folder_with_selection"), "folder-plus").Chosen() {
			// 在文件夹里时新文件夹建在它下面。
			parent := ""
			if a.Query.Scope == "folder" {
				parent = a.Query.FolderID
			}
			a.ShowNewFolder(parent, "", ids)
		}
		if menuItem(m, a.T("copy_tags"), "tag").Chosen() {
			a.CopyTags()
		}
		if len(a.tagClipboard) > 0 && menuItem(m, a.T("paste_tags"), "clipboard-add").Chosen() {
			a.PasteTags()
		}
	}
	m.Separator()
	if !multi && !isVideoAsset(asset) && menuItem(m, a.T("copy_image"), "copy").Chosen() {
		a.CopyAssetImage(asset)
	}
	if menuItem(m, a.T("copy_file_path"), "clipboard-text").Chosen() {
		a.CopyAssetPaths(a.SelectedAssets())
	}
	if menuItem(m, a.T("export"), "upload-simple").Chosen() {
		a.ExportSelected()
	}
	if !asset.Trashed {
		m.Separator()
		if menuItem(m, a.T("refresh_thumbnail"), "arrow-clockwise").Chosen() {
			service, items := a.Assets, slices.Clone(a.SelectedAssets())
			a.Run("refresh_thumbnail", func(ctx context.Context) error {
				err := service.RefreshThumbnails(ctx, items)
				a.Window.Update(func() {
					for _, item := range items {
						a.Thumbnails.Invalidate(item.ID)
					}
				})
				return err
			})
		}
		if menuItem(m, a.T("reanalyze_colors"), "palette").Chosen() {
			service, items := a.Assets, slices.Clone(a.SelectedAssets())
			a.Run("reanalyze_colors", func(ctx context.Context) error { return service.ReanalyzeColors(ctx, items) })
		}
	}
	m.Separator()
	if asset.Trashed {
		if menuItem(m, a.T("restore"), "arrow-counter-clockwise").Chosen() {
			a.Restore(ids)
		}
		if menuItem(m, a.T("delete_permanently"), "trash").Chosen() {
			a.RequestDelete(ids)
		}
	} else if menuItem(m, a.T("move_to_trash"), "trash").Chosen() {
		a.RequestDelete(ids)
	}
}

// 缩略图读失败的卡片，灰块太小放不下图标就不画。
const thumbnailFailureMin float32 = 28

// 缩略图读失败（不是还在加载）的卡片：灰块中间画一个提示图标，悬停看原因。不自动重试，右键「刷新缩略图」可以重做。
func (a *App) thumbnailFailure(c *ui.Context, asset model.Asset) {
	if a.Library == nil || a.Thumbnails == nil {
		return
	}
	err := a.Thumbnails.Error(a.Library.Root, asset)
	if err == nil {
		return
	}
	p := paletteFor(c.Theme())
	ui.Column(c.Key("thumbnail-failed")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().PassThrough().Children(func() {
		ui.Box(c.Key("thumbnail-failed-icon")).Size(24, 24).Center().Tooltip(a.Texts.Error(err)).Children(func() {
			appIcon(c, "warning").Size(18, 18).TextColor(p.TextTertiary)
		})
	})
}

// orderedIDs 把 ID 按当前列表顺序排好，批量重命名的序号跟着看到的顺序走。
func (a *App) orderedIDs(ids []string) []string {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for _, asset := range a.Visible {
		if wanted[asset.ID] {
			ordered = append(ordered, asset.ID)
			delete(wanted, asset.ID)
		}
	}
	for _, id := range ids {
		if wanted[id] {
			ordered = append(ordered, id)
		}
	}
	return ordered
}

func (a *App) importProgress(c *ui.Context) {
	p := a.Progress
	colors := paletteFor(c.Theme())
	if p == nil {
		return
	}
	ui.Column(c.Key("import-progress")).Transition(ui.ElementTransition{Animation: &motionSpring, Height: true, Enter: &ui.Motion{Y: -10, Collapse: true}, Exit: &ui.Motion{Y: -10, Collapse: true}}).Height(a.progressHeight()).Shrink(0).Padding(space12, space16).Gap(space4).Background(toolbarColor(c.Theme())).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.T("import_stage_"+p.Stage)).FontSize(fontBody).FontWeight(600).TextColor(colors.Text)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d / %d", p.Processed, p.Total)).FontFeatures("tnum").FontSize(fontNumber).TextColor(colors.TextSecondary)
			if a.CanCancelImport() {
				if progressStopButton(c, a.T("stop")).Clicked() {
					a.CancelImport()
				}
			}
		})
		if !compactProgress(p) {
			ui.Text(c, a.T("import_counts", p.Imported, p.Skipped)).FontSize(fontBody).TextColor(colors.TextSecondary)
			// 扫描刚开始还没有当前文件，空着，不显示「.」。
			file := ""
			if p.CurrentFile != "" {
				file = filepath.Base(p.CurrentFile)
			}
			ui.Text(c, file).SingleLine().FontSize(fontSecondary).TextColor(colors.TextSecondary)
		}
		// 还在扫描、不知道一共多少张时用来回流动的进度条，不画一条看着像卡住的空条。
		progress := float64(-1)
		if p.Total > 0 {
			progress = float64(p.Processed) / float64(p.Total)
		}
		smoothProgress(c, progress).FillWidth()
	})
}

// 进度条右侧的文字按钮：平时是次要文字色，悬停才有浅底。
func progressStopButton(c *ui.Context, label string) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Height(24).Padding(0, space8).Margin(0, 0, 0, space8).Shrink(0).Radius(radiusControl).
		Justify(ui.Center).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	fill := ui.Transparent
	switch {
	case button.Pressed():
		fill = p.Pressed
	case button.Hovered():
		fill = p.Hover
	}
	button.Background(button.AnimateColor("stop-fill", fill, motionColor))
	button.Children(func() {
		ui.Text(c, label).FontSize(fontNumber).FontWeight(500).TextColor(p.Text).SingleLine()
	})
	return button
}
