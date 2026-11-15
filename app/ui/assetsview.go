package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

const importProgressHeight float32 = 120

func (a *App) contentTopInset() float32 {
	if a.Progress != nil {
		return MainToolbarHeight + importProgressHeight
	}
	return MainToolbarHeight
}

func (a *App) assetArea(c *ui.Context) {
	a.hovered = ""
	scroll := ui.Scroll(c.Key("assets")).Grow(1).FillWidth().TrackScroll(&a.scroll)
	inset := scroll.AnimateTo("header-inset", a.contentTopInset(), motionSpring)
	scroll.Padding(inset+space16, space16, space16).ScrollbarInsets(inset, 0, 0)
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
	viewportTop := a.scroll.Y - inset - space16
	top, bottom := max(0, viewportTop-height), viewportTop+2*height
	state := ui.Local(scroll, "asset-layout", func() assetLayoutState { return assetLayoutState{} })
	var items []MasonryItem
	var contentHeight float32
	var columns int
	switch a.ViewMode {
	case 1, 2:
		items, contentHeight, columns = a.assetRows(width, top, bottom)
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
		items, contentHeight, columns = a.layout.Visible(top, bottom), a.layout.Height(), len(a.layout.columns)
	}
	moveCards := state.prepare(a.ViewMode, columns, width, a.thumbnailScale())
	scroll.Children(func() {
		mode := ui.Box(c.Key(assetModeID{a.ViewMode})).Height(contentHeight).FillWidth().Shrink(0).
			Transition(ui.ElementTransition{Duration: assetEnterDuration, ExitDuration: assetExitDuration, Colors: true,
				Enter: &ui.Motion{Scale: .98}, Exit: &ui.Motion{}})
		mode.Children(func() {
			canvas := ui.Box(c.Key(assetCanvasID{state.generation})).Height(contentHeight).FillWidth().Shrink(0).
				Transition(ui.ElementTransition{Duration: assetColumnsEnterDuration, ExitDuration: assetColumnsExitDuration, Colors: true,
					Enter: &ui.Motion{}, Exit: &ui.Motion{}})
			canvas.Children(func() {
				for _, item := range items {
					asset := a.Visible[item.Index]
					card := ui.Column(c.Key(asset.ID)).Absolute().Left(item.X).Top(item.Y).Size(item.Width, item.Height)
					// 形状与列数不变时才移动卡片；变宽、缩放和换视图都直接布局。
					if moveCards && item.Y+item.Height > viewportTop+inset && item.Y < viewportTop+height {
						card.Transition(ui.ElementTransition{Duration: assetEnterDuration, ExitDuration: assetExitDuration, Position: true,
							Enter: &ui.Motion{}, Exit: &ui.Motion{}})
					} else {
						card.Transition(ui.ElementTransition{Duration: -1, Position: true, Size: true})
					}
					card.Children(func() { a.assetCard(c, asset, item.Index, a.ViewMode == 2, item.Width) })
				}
			})
			if canvas.Clicked() {
				a.clearAssetSelection()
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
	if len(a.Visible) == 0 {
		p := paletteFor(c.Theme())
		icon, title, hint := "photos", a.T("scope_empty"), a.T("scope_empty_hint")
		hasFilters := len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) > 0
		emptyLibrary := len(a.Snapshot.Assets) == 0 && a.Query.Scope != "trash" && a.Query.Search == "" && !hasFilters
		switch {
		case a.Query.Search != "":
			icon, title, hint = "search", a.T("search_empty", a.Query.Search), a.T("search_empty_hint")
		case a.Query.Scope == "trash":
			icon, title, hint = "trash", a.T("trash_empty_title"), a.T("trash_empty_hint")
		case emptyLibrary:
			title, hint = a.T("drop_images"), a.T("drop_images_hint")
		}
		fadeMotion(ui.Column(c.Key("asset-empty")).Absolute().Top(inset).Left(space24).Right(space24).Bottom(space24).
			Center().Gap(space12).PassThrough()).Children(func() {
			ui.Box(c.Key("empty-icon")).Size(space48, space48).Radius(space24).Background(p.Subtle).TextColor(p.TextSecondary).
				Center().Children(func() { appIcon(c, icon).Size(space24, space24) })
			ui.Text(c.Key("empty-title"), title).FontSize(fontTitle).FontWeight(600).TextColor(p.Text).TextAlign(ui.Center)
			ui.Text(c.Key("empty-hint"), hint).FontSize(fontBody).TextColor(p.TextSecondary).TextAlign(ui.Center)
			switch {
			case a.Query.Search != "":
				if primaryButton(c, a.T("clear_search"), false).Clicked() {
					a.Query.Search = ""
					a.Refine()
				}
			case hasFilters:
				if primaryButton(c, a.T("clear_filters"), false).Clicked() {
					a.Query.ColorFamilies, a.Query.TagIDs, a.Query.Extensions = nil, nil, nil
					a.Refine()
				}
			case emptyLibrary:
				if primaryButton(c, a.T("import"), false).Clicked() {
					a.ChooseImport(false)
				}
			}
		})
	}
}

const (
	assetEnterDuration        = 180 * time.Millisecond
	assetExitDuration         = 120 * time.Millisecond
	assetColumnsEnterDuration = 90 * time.Millisecond
	assetColumnsExitDuration  = 60 * time.Millisecond
)

// 卡片左上角的心和右上角的尺寸角标照旧版：高 16，红心用系统红。
const assetBadgeHeight = 16

var favoriteRed = ui.Hex("#FF3B30")

type assetModeID struct{ mode int }
type assetCanvasID struct{ generation uint64 }

type assetLayoutState struct {
	ready         bool
	mode, columns int
	width, scale  float32
	generation    uint64
	ratiosVersion uint64
	ratios        []float32
}

func (s *assetLayoutState) prepare(mode, columns int, width, scale float32) bool {
	sameShape := s.ready && s.mode == mode && s.columns == columns
	if s.ready && s.mode != mode {
		s.generation = 0
	} else if s.ready && s.columns != columns {
		// 变列数单独使用短过渡，不沿用切换视图的时长。
		s.generation++
	}
	moveCards := !s.ready || sameShape && s.width == width && s.scale == scale
	s.ready, s.mode, s.columns, s.width, s.scale = true, mode, columns, width, scale
	return moveCards
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
		card = ui.Row(c.Key(asset.ID)).Fill().Gap(space16).Padding(space8, space12).Radius(radiusCard).
			BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSubtle)
	} else {
		card = ui.Column(c.Key(asset.ID)).Fill().Radius(radiusCard).FocusRing(false).Shadow(0, 1, 2, 0, p.Shadow)
	}
	card.Label(asset.Name).Focusable().AccessSelected(selected).Cursor(ui.CursorPointer)
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
			if thumb != nil {
				ui.Image(c.Key("thumbnail"), thumb).Size(space48, space48).Fit(ui.Cover).Radius(radiusCard).Shadow(0, 1, 2, 0, p.Shadow)
			} else {
				ui.Box(c).Size(space48, space48).Radius(radiusCard).Background(p.Subtle).Shadow(0, 1, 2, 0, p.Shadow)
			}
			ui.Column(c.Key("list-meta")).Grow(1).MinWidth(0).Gap(space4).Children(func() {
				ui.Text(c, asset.Name).FontSize(fontBody).FontWeight(600).SingleLine()
				ui.Text(c.Key("list-values"), fmt.Sprintf("%d × %d · %s", width, height, fileSize(asset.Size))).FontFeatures("tnum").FontSize(fontNumber).TextColor(p.TextSecondary).SingleLine()
			})
			ui.Text(c.Key("list-extension"), strings.ToUpper(asset.Extension)).FontSize(fontSecondary).TextColor(p.TextSecondary).Margin(0, space40, 0, 0)
		} else {
			radius := float32(radiusCard)
			if thumb != nil {
				ui.Image(c.Key("thumbnail"), thumb).Fill().Fit(ui.Cover).Radius(radius)
			} else {
				ui.Box(c).Fill().Radius(radius).Background(p.Subtle)
			}

			// 尺寸角标照旧版：常驻右上角，黑色半透明底、白字。
			badge := ui.Text(c.Key("dimensions"), fmt.Sprintf("%d × %d", width, height)).Absolute().Top(space8).Right(space8).
				Radius(5).Background(ui.RGBA(0, 0, 0, .34)).
				FontFeatures("tnum").FontSize(10).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, .82)).PassThrough()
			centeredTextHeight(badge, assetBadgeHeight, 3)
		}
		if selected || focused {
			// 图片外圈留 2px；列表描边放在行内，避免盖住相邻行。
			ring := fadeMotion(ui.Box(c.Key("selection-ring")).Absolute().Left(-space4).Top(-space4).Right(-space4).Bottom(-space4).
				Radius(radiusPanel).Border(2, p.Accent).PassThrough())
			if list {
				ring.Left(2).Top(2).Right(2).Bottom(2).Radius(radiusCard)
			}
		}
		if hovered || asset.Favorite {
			button := ui.ButtonBase(c.Key("favorite")).Label(a.T("favorite")).Tooltip(a.T("favorite")).Absolute().Cursor(ui.CursorPointer).
				Justify(ui.Center).AlignItems(ui.Center).Padding(0)
			if list {
				button.Top(space16).Right(space8).Size(controlSmall, controlSmall).Radius(radiusControl).Background(panelColor(c.Theme()))
			} else {
				// 旧版：左上角一颗白色心，不带底；鼠标移到心上才出黑色半透明底，收藏后变成红色实心。
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
			if list {
				ink := p.TextSecondary
				if button.Hovered() || asset.Favorite {
					ink = p.Text
				}
				button.TextColor(button.AnimateColor("favorite-ink", ink, motionColor))
				button.Children(func() { appIconSelected(c, "heart", asset.Favorite).Size(iconSize, iconSize) })
			} else {
				ink, background := ui.RGBA(255, 255, 255, .92), ui.Transparent
				if asset.Favorite {
					ink = favoriteRed
				}
				if button.Hovered() {
					background = ui.RGBA(0, 0, 0, .3)
				}
				animateColors(button, background, ink)
				button.Children(func() { appIconSelected(c, "heart", asset.Favorite).Size(14, 14) })
			}
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
		ids := a.selectedIDs()
		if !a.Selected.Has(asset.ID) {
			ids = []string{asset.ID}
		}
		card.Drag(AssetsDrag{LibraryID: a.libraryID(), IDs: ids})
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
	ui.Column(c.Key("import-progress")).Transition(ui.ElementTransition{Animation: &motionSpring, Height: true, Enter: &ui.Motion{Y: -10, Collapse: true}, Exit: &ui.Motion{Y: -10, Collapse: true}}).Height(importProgressHeight).Shrink(0).Padding(space12, space16).Gap(space4).Background(toolbarColor(c.Theme())).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("import_stage_"+p.Stage)).FontSize(fontBody).FontWeight(600).TextColor(colors.Text)
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d / %d", p.Processed, p.Total)).FontFeatures("tnum").FontSize(fontNumber).TextColor(colors.TextSecondary)
		})
		ui.Text(c, a.T("import_counts", p.Imported, p.Skipped)).FontSize(fontBody).TextColor(colors.TextSecondary)
		ui.Text(c, filepath.Base(p.CurrentFile)).SingleLine().FontSize(fontSecondary).TextColor(colors.TextSecondary)
		progress := float64(0)
		if p.Total > 0 {
			progress = float64(p.Processed) / float64(p.Total)
		}
		ui.Progress(c, progress).FillWidth()
	})
}
