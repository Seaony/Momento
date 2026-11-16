package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

const (
	importProgressHeight    float32 = 120
	thumbnailProgressHeight float32 = 54 // 重建缩略图只有标题、张数和进度条
)

func (a *App) progressHeight() float32 {
	if a.Progress != nil && a.Progress.Stage == "thumbnails" {
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
	if a.Thumbnails != nil {
		a.Thumbnails.NextFrame()
	}
	// 上一帧鼠标已经离开那张视频卡片就关掉悬停播放器（上一帧的 hovered 这时已经算完整）。
	a.finishHoverVideo()
	a.hovered = ""
	scroll := ui.Scroll(c.Key("assets")).Scrollbars(false).Grow(1).FillWidth().TrackScroll(&a.scroll)
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
	viewportTop := a.scroll.Y - inset - space16
	top, bottom := max(0, viewportTop-height), viewportTop+2*height
	state := ui.Local(scroll, "asset-layout", func() assetLayoutState { return assetLayoutState{} })
	var items []MasonryItem
	var contentHeight float32
	switch a.ViewMode {
	case 1, 2:
		items, contentHeight = a.assetRows(width, top, bottom)
	default:
		if a.layout == nil || a.layoutVersion != a.visibleVersion || a.layoutWidth != width {
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
			a.layout = NewMasonryScaled(state.ratios, width, a.thumbnailScale())
			a.layoutWidth, a.layoutVersion = width, a.visibleVersion
		}
		items, contentHeight = a.layout.Visible(top, bottom), a.layout.Height()
	}
	reflow := state.prepare(a.ViewMode, width, a.thumbnailScale())
	scroll.Children(func() {
		mode := ui.Box(c.Key(assetModeID{a.ViewMode})).Height(contentHeight).FillWidth().Shrink(0).
			Transition(ui.ElementTransition{Duration: assetEnterDuration, ExitDuration: assetExitDuration, Colors: true,
				Enter: &ui.Motion{Scale: .98}, Exit: &ui.Motion{}})
		mode.Children(func() {
			for _, item := range items {
				asset := a.Visible[item.Index]
				card := ui.Column(c.Key(asset.ID)).Absolute().Left(item.X).Top(item.Y).Size(item.Width, item.Height)
				switch {
				case reflow:
					// 缩略图大小、栏宽、窗口宽度变了：每张卡片从原来的位置和大小弹到新的，图片跟着变宽变窄。
					card.Transition(ui.ElementTransition{Duration: assetReflowDuration, Ease: assetReflowEase, ExitDuration: assetExitDuration, Position: true, Size: true,
						Enter: &ui.Motion{}, Exit: &ui.Motion{}})
				case item.Y+item.Height > viewportTop+inset && item.Y < viewportTop+height:
					// 筛选、排序、增删：屏幕里的卡片按身份移过去。
					card.Transition(ui.ElementTransition{Duration: assetEnterDuration, ExitDuration: assetExitDuration, Position: true,
						Enter: &ui.Motion{}, Exit: &ui.Motion{}})
				default:
					card.Transition(ui.ElementTransition{Duration: -1, Position: true, Size: true})
				}
				card.Children(func() { a.assetCard(c, asset, item.Index, a.ViewMode == 2, item.Width) })
			}
		})
		if mode.Clicked() {
			a.clearAssetSelection()
		}
	})
	if scroll.Clicked() {
		a.clearAssetSelection()
	}
	a.assetPreviewShortcut(c, scroll)
	a.assetArrowKeys(c, inset, height)
	if len(a.Visible) == 0 {
		p := paletteFor(c.Theme())
		art, title, hint := "scope-empty", a.T("scope_empty"), a.T("scope_empty_hint")
		hasFilters := len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions)+len(a.Query.Kinds) > 0
		emptyLibrary := len(a.Snapshot.Assets) == 0 && a.Query.Scope != "trash" && a.Query.Search == "" && !hasFilters
		switch {
		case a.Query.Search != "":
			art, title, hint = "search-empty", a.T("search_empty", a.Query.Search), a.T("search_empty_hint")
		case a.Query.Scope == "trash":
			art, title, hint = "trash-empty", a.T("trash_empty_title"), a.T("trash_empty_hint")
		case emptyLibrary:
			art, title, hint = "empty-library", a.T("drop_images"), a.T("drop_images_hint")
		}
		fadeMotion(ui.Column(c.Key("asset-empty")).Absolute().Top(inset).Left(space24).Right(space24).Bottom(space24).
			Center().Gap(space12).PassThrough()).Children(func() {
			emptyArt(c.Key("empty-art"), art, emptyArtLarge).Margin(0, 0, space4, 0)
			ui.Text(c.Key("empty-title"), title).FontSize(fontTitle).FontWeight(600).TextColor(p.Text).TextAlign(ui.Center)
			ui.Text(c.Key("empty-hint"), hint).FontSize(fontBody).TextColor(p.TextSecondary).TextAlign(ui.Center)
			switch {
			case a.Query.Search != "":
				if dialogPrimaryButton(c, a.T("clear_search"), false).Clicked() {
					a.Query.Search = ""
					a.Refine()
				}
			case hasFilters:
				if dialogPrimaryButton(c, a.T("clear_filters"), false).Clicked() {
					a.Query.ColorFamilies, a.Query.TagIDs, a.Query.Extensions, a.Query.Kinds = nil, nil, nil, nil
					a.Refine()
				}
			case emptyLibrary:
				if dialogPrimaryButton(c, a.T("import"), false).Clicked() {
					a.ChooseImport(false)
				}
			}
		})
	}
}

const (
	assetEnterDuration  = 180 * time.Millisecond
	assetExitDuration   = 120 * time.Millisecond
	assetReflowDuration = 250 * time.Millisecond
)

// 前段快、尾段柔和，和栏宽弹簧的手感接近，但在 250ms 准时落定，和栏宽同时到位。
var assetReflowEase = ui.CubicBezier(.25, 1, .5, 1)

// 列表行：缩略图宽固定 112，行高跟着缩略图的实际高度走，上下各留 12；名称和尺寸两行至少要 40 高。
const (
	listRowRadius     float32 = radiusPanel // 悬停底色和选中框的圆角
	listThumbnail     float32 = 112
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

var favoriteRed = ui.Hex("#FF3B30")

type assetModeID struct{ mode int }

type assetLayoutState struct {
	ready         bool
	mode          int
	width, scale  float32
	ratiosVersion uint64
	ratios        []float32
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

func (a *App) assetCard(c *ui.Context, asset model.Asset, index int, list bool, frameWidth float32) {
	p := paletteFor(c.Theme())
	width, height := assetDisplaySize(asset)
	selected := a.Selected.Has(asset.ID)
	var card ui.Element
	if list {
		card = ui.Row(c.Key(asset.ID)).Fill().Gap(space16).Padding(listRowPadding, space12).Radius(listRowRadius).
			BorderWidth(0, 0, 1, 0).BorderColor(p.hairline())
	} else {
		card = ui.Column(c.Key(asset.ID)).Fill().Radius(radiusCard).FocusRing(false).Shadow(0, 1, 2, 0, p.Shadow)
	}
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
	pose := ui.Pose{}
	if hovered {
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
		if list {
			// 缩略图按原图比例显示，长边贴满 112，放在 112 宽的框里居中，名称一列仍然对齐；极细长的图短边最少 4，图片用 Contain 不裁切。
			w, h := listThumbnailSize(width, height)
			ui.Box(c.Key("thumbnail-frame")).Width(listThumbnail).FillHeight().Shrink(0).Center().Children(func() {
				if thumb != nil {
					ui.Image(c.Key("thumbnail"), thumb).Size(w, h).Fit(ui.Contain).Radius(radiusCard).Shadow(0, 1, 2, 0, p.Shadow)
				} else {
					ui.Box(c).Size(w, h).Radius(radiusCard).Background(p.Subtle).Shadow(0, 1, 2, 0, p.Shadow)
				}
			})
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
		} else {
			radius := float32(radiusCard)
			if thumb != nil {
				ui.Image(c.Key("thumbnail"), thumb).Fill().Fit(ui.Cover).Radius(radius)
			} else {
				ui.Box(c).Fill().Radius(radius).Background(p.Subtle)
			}
			a.hoverVideoOverlay(c, card, asset, hovered, radius)
			// 视频右下角：播放标记加时长，和尺寸角标同一套样式。
			if isVideoAsset(asset) {
				ui.Row(c.Key("duration")).Absolute().Bottom(space8).Right(space8).Height(assetBadgeHeight).Padding(0, 5).Gap(3).AlignItems(ui.Center).
					Radius(5).Background(ui.RGBA(0, 0, 0, .34)).PassThrough().Children(func() {
					appIconSelected(c, "play", true).Size(8, 8).TextColor(ui.RGBA(255, 255, 255, .82))
					// 读不出时长的视频（有些录屏 WebM 不写时长）只显示播放标记。
					if d := assetDuration(asset); d > 0 {
						ui.Text(c, videoDuration(d)).FontFeatures("tnum").FontSize(10).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, .82))
					}
				})
			}

			// 实况照片右下角：实况标记加「实况」，和视频时长同一个位置、同一套样式。
			if isLiveAsset(asset) {
				ui.Row(c.Key("live")).Absolute().Bottom(space8).Right(space8).Height(assetBadgeHeight).Padding(0, 5).Gap(3).AlignItems(ui.Center).
					Radius(5).Background(ui.RGBA(0, 0, 0, .34)).PassThrough().Children(func() {
					appIcon(c, "live-photo").Size(12, 12).TextColor(ui.RGBA(255, 255, 255, .82))
					ui.Text(c, a.T("live_photo")).FontSize(10).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, .82))
				})
			}

			// 尺寸角标照旧版：常驻右上角，黑色半透明底、白字。读不出尺寸的（Windows、Linux 上的 RAW 等）不显示。
			if width > 0 && height > 0 {
				badge := ui.Text(c.Key("dimensions"), fmt.Sprintf("%d × %d", width, height)).Absolute().Top(space8).Right(space8).
					Radius(5).Background(ui.RGBA(0, 0, 0, .34)).
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
		if hovered || asset.Favorite {
			button := ui.ButtonBase(c.Key("favorite")).Label(a.T("favorite")).Tooltip(a.T("favorite")).Absolute().Cursor(ui.CursorPointer).
				Justify(ui.Center).AlignItems(ui.Center).Padding(0)
			// 旧版：图片左上角一颗白色心，不带底；鼠标移到心上才出黑色半透明底，收藏后变成红色实心。悬停才出现，收藏了常驻。
			if list {
				// 列表里贴着缩略图（在 112 宽的框里居中）的左上角。
				w, h := listThumbnailSize(width, height)
				left := listRowPadding + (listThumbnail-w)/2
				top := listRowPadding + (max(h, listRowMinContent)-h)/2
				button.Top(top+listFavoriteInset).Left(left+listFavoriteInset-1).Size(22, assetBadgeHeight).Radius(assetBadgeHeight / 2)
			} else {
				button.Top(space8).Left(7).Size(22, assetBadgeHeight).Radius(assetBadgeHeight / 2)
			}
			favoriteFocused := button.FocusVisible()
			wasFavoriteFocused := ui.Local(button, "keyboard-focus", func() bool { return false })
			if favoriteFocused && !*wasFavoriteFocused {
				ui.Box(c.Key("favorite-focus-reveal")).Absolute().Top(-a.contentTopInset() - space8).Left(0).Right(0).Bottom(0).
					PassThrough().ScrollIntoView()
			}
			*wasFavoriteFocused = favoriteFocused
			fadeMotion(button)
			ink, background := ui.RGBA(255, 255, 255, .92), ui.Transparent
			if asset.Favorite {
				ink = favoriteRed
			}
			if button.Hovered() {
				background = ui.RGBA(0, 0, 0, .3)
			}
			animateColors(button, background, ink)
			button.Children(func() { appIconSelected(c, "heart", asset.Favorite).Size(14, 14) })
			if button.Clicked() {
				a.SetFavorite([]string{asset.ID}, !asset.Favorite)
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
		}, a.ShowError, a.MarkAssetsUsed)
	}
	card.ContextMenu(func(m *ui.Menu) {
		ids := a.selectedIDs()
		if !a.Selected.Has(asset.ID) {
			ids = []string{asset.ID}
		}
		if m.Item(a.T("preview")).Chosen() {
			a.ShowPreview(asset.ID, false)
		}
		if m.Item(a.T("export")).Chosen() {
			if !a.Selected.Has(asset.ID) {
				a.selectAsset(asset.ID, index, 0)
			}
			a.ExportSelected()
		}
		if !asset.Trashed {
			if m.Item(a.T("refresh_thumbnail")).Chosen() {
				service := a.Assets
				a.Run("refresh_thumbnail", func(ctx context.Context) error {
					err := service.RefreshThumbnails(ctx, []model.Asset{asset})
					a.Window.Update(func() { a.Thumbnails.Invalidate(asset.ID) })
					return err
				})
			}
			if m.Item(a.T("reanalyze_colors")).Chosen() {
				service := a.Assets
				a.Run("reanalyze_colors", func(ctx context.Context) error { return service.ReanalyzeColors(ctx, []model.Asset{asset}) })
			}
		}
		if m.Item(a.T("reveal")).Chosen() {
			a.Reveal(asset)
		}
		m.Separator()
		if asset.Trashed {
			if m.Item(a.T("restore")).Chosen() {
				a.Restore(ids)
			}
			if m.Item(a.T("delete_permanently")).Chosen() {
				a.RequestDelete(ids)
			}
		} else {
			if m.Item(a.T("move_to_trash")).Chosen() {
				a.RequestDelete(ids)
			}
		}
	})
}

func (a *App) importProgress(c *ui.Context) {
	p := a.Progress
	colors := paletteFor(c.Theme())
	if p == nil {
		return
	}
	ui.Column(c.Key("import-progress")).Transition(ui.ElementTransition{Animation: &motionSpring, Height: true, Enter: &ui.Motion{Y: -10, Collapse: true}, Exit: &ui.Motion{Y: -10, Collapse: true}}).Height(a.progressHeight()).Shrink(0).Padding(space12, space16).Gap(space4).Background(toolbarColor(c.Theme())).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("import_stage_"+p.Stage)).FontSize(fontBody).FontWeight(600).TextColor(colors.Text)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d / %d", p.Processed, p.Total)).FontFeatures("tnum").FontSize(fontNumber).TextColor(colors.TextSecondary)
		})
		if p.Stage != "thumbnails" {
			ui.Text(c, a.T("import_counts", p.Imported, p.Skipped)).FontSize(fontBody).TextColor(colors.TextSecondary)
			// 扫描刚开始还没有当前文件，空着，不显示「.」。
			file := ""
			if p.CurrentFile != "" {
				file = filepath.Base(p.CurrentFile)
			}
			ui.Text(c, file).SingleLine().FontSize(fontSecondary).TextColor(colors.TextSecondary)
		}
		progress := float64(0)
		if p.Total > 0 {
			progress = float64(p.Processed) / float64(p.Total)
		}
		ui.Progress(c, progress).FillWidth()
	})
}
