package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
	scroll.Padding(inset+16, 16, 16).ScrollbarInsets(inset, 0, 0)
	box := scroll.Bounds()
	width, height := box.W-32, box.H
	if width <= 0 {
		width, height = c.Size()
		width -= 32 + a.SidebarShownWidth() + a.InspectorShownWidth()
		if a.chromeLayoutReady {
			width--
		}
	}
	width = a.assetLayoutWidth(c, scroll, width)
	if height <= 0 {
		height = 640
	}
	viewportTop := a.scroll.Y - inset - 16
	top, bottom := max(0, viewportTop-height), viewportTop+2*height
	var items []MasonryItem
	var contentHeight float32
	switch a.ViewMode {
	case 1, 2:
		items, contentHeight = a.assetRows(width, top, bottom)
	default:
		if a.layout == nil || a.layoutVersion != a.visibleVersion || a.layoutWidth != width {
			ratios := make([]float32, len(a.Visible))
			for i, asset := range a.Visible {
				w, h := assetDisplaySize(asset)
				if h > 0 {
					ratios[i] = float32(w) / float32(h)
				}
			}
			a.layout = NewMasonryScaled(ratios, width, a.thumbnailScale())
			a.layoutWidth, a.layoutVersion = width, a.visibleVersion
		}
		items, contentHeight = a.layout.Visible(top, bottom), a.layout.Height()
	}
	scroll.Children(func() {
		canvas := ui.Box(c.Key("asset-canvas")).Height(contentHeight).FillWidth().Shrink(0)
		canvas.Children(func() {
			for _, item := range items {
				asset := a.Visible[item.Index]
				card := ui.Column(c.Key(asset.ID)).Absolute().Left(item.X).Top(item.Y).Size(item.Width, item.Height)
				// 只给屏幕中的卡片过渡，离屏位置直接更新，滚动本身不做位置动画。
				if item.Y+item.Height > viewportTop+inset && item.Y < viewportTop+height {
					layoutMotion(card)
				}
				card.Children(func() { a.assetCard(c, asset, item.Index, a.ViewMode == 2, item.Width) })
			}
		})
		if canvas.Clicked() {
			a.clearAssetSelection()
		}
	})
	if scroll.Clicked() {
		a.clearAssetSelection()
	}
	a.assetPreviewShortcut(c, scroll)
	if len(a.Visible) == 0 {
		fadeMotion(ui.Column(c.Key("asset-empty")).Absolute().Top(inset).Left(24).Right(24).Bottom(24).
			Center().Gap(12).PassThrough()).Children(func() {
			if a.Query.Search != "" {
				ui.Text(c, a.T("search_empty", a.Query.Search)).FontSize(20).Bold()
				ui.Text(c, a.T("result_count", "0")).TextColor(c.Theme().TextMuted)
			} else if len(a.Snapshot.Assets) > 0 {
				ui.Text(c, a.T("scope_empty")).FontSize(20).Bold()
			} else {
				ui.Text(c, a.T("drop_images")).FontSize(22).Bold()
				ui.Text(c, a.T("drop_images_hint")).TextColor(c.Theme().TextMuted)
				ui.Row(c).Gap(8).Children(func() {
					if primaryButton(c, a.T("import"), false).Clicked() {
						a.ChooseImport(false)
					}
					if secondaryButton(c, a.T("install_extension")).Clicked() {
						c.OpenURL("https://github.com/Seaony/Momento-Chomre-Extension/releases/latest")
					}
				})
			}
		})
	}
}

type assetChromeLayout struct {
	sidebar, inspector bool
	moving             bool
}

func (a *App) assetLayoutWidth(c *ui.Context, scroll ui.Element, actual float32) float32 {
	if !a.chromeLayoutReady {
		return actual
	}
	inspector := a.InspectorTargetWidth() > 0
	state := ui.Local(scroll, "chrome-layout", func() assetChromeLayout {
		return assetChromeLayout{sidebar: a.SidebarVisible, inspector: inspector}
	})
	if state.sidebar != a.SidebarVisible || state.inspector != inspector {
		state.sidebar, state.inspector, state.moving = a.SidebarVisible, inspector, true
	}
	sidebarWidth := float32(0)
	if a.SidebarVisible {
		sidebarWidth = a.SidebarWidth
	}
	if state.moving && a.SidebarShownWidth() == sidebarWidth && a.InspectorShownWidth() == a.InspectorTargetWidth() {
		state.moving = false
	}
	if state.moving {
		// 栏宽正在变化时只朝最终卡片位置移动一次，避免每帧重排造成拖尾。
		width, _ := c.Size()
		return width - 32 - sidebarWidth - a.InspectorTargetWidth() - 1
	}
	return actual
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
	width, height := assetDisplaySize(asset)
	selected := a.Selected.Has(asset.ID)
	var card ui.Element
	if list {
		card = ui.Row(c.Key(asset.ID)).Fill().Gap(16).Padding(8, 10).Radius(8).
			BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border)
	} else {
		card = ui.Column(c.Key(asset.ID)).Fill().Radius(8).FocusRing(false)
		if a.ViewMode == 1 {
			card.Radius(10).Background(c.Theme().Surface)
		}
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
	if a.ViewMode == 1 && !list {
		background = c.Theme().Surface
	}
	if list {
		switch {
		case selected && hovered:
			background = c.Theme().Accent.Alpha(.18)
		case selected:
			background = c.Theme().Accent.Alpha(.12)
		case card.Pressed():
			background = warmOverlay(c.Theme(), .11)
		case hovered:
			background = warmOverlay(c.Theme(), .06)
		}
	}
	animateColors(card, background, c.Theme().Text)
	card.Children(func() {
		if revealFocus {
			// 把页头遮住的高度算进定位目标，键盘聚焦时图片仍露在页头下方。
			ui.Box(c.Key("focus-reveal")).Absolute().Top(-a.contentTopInset() - 8).Left(0).Right(0).Bottom(0).
				PassThrough().ScrollIntoView()
		}
		thumb := a.Bitmap(asset)
		if list {
			if thumb != nil {
				ui.Image(c.Key("thumbnail"), thumb).Size(48, 48).Fit(ui.Cover).Radius(6)
			} else {
				ui.Box(c).Size(48, 48).Radius(6).Background(c.Theme().Surface)
			}
			fadeMotion(ui.Column(c.Key("list-meta")).Grow(1).MinWidth(0).Gap(4)).Children(func() {
				ui.Text(c, asset.Name).FontSize(13).Bold().SingleLine()
				ui.Text(c, fmt.Sprintf("%d × %d · %s", width, height, fileSize(asset.Size))).FontSize(12).TextColor(c.Theme().TextMuted)
			})
			fadeMotion(ui.Text(c.Key("list-extension"), strings.ToUpper(asset.Extension)).FontSize(12).TextColor(c.Theme().TextMuted).Margin(0, 36, 0, 0))
		} else {
			radius := float32(8)
			if a.ViewMode == 1 {
				radius = 10
				ui.Box(c.Key("grid-image")).Fill().Padding(16).Center().Children(func() {
					if thumb != nil {
						w, h := thumb.Size()
						scale := max(0, frameWidth-32) / max(float32(w), float32(h))
						ui.Image(c.Key("thumbnail"), thumb).Size(float32(w)*scale, float32(h)*scale).Fit(ui.Contain).Radius(4)
					}
				})
			} else if thumb != nil {
				ui.Image(c.Key("thumbnail"), thumb).Fill().Fit(ui.Cover).Radius(radius)
			} else {
				ui.Box(c).Fill().Radius(radius).Background(c.Theme().Surface)
			}
			shade := ui.Box(c.Key("hover-shade")).Absolute().Left(0).Top(0).Right(0).Bottom(0).Radius(radius).PassThrough()
			darkness := ui.Transparent
			if hovered {
				darkness = ui.RGBA(58, 35, 24, .07)
			}
			if card.Pressed() {
				darkness = ui.RGBA(58, 35, 24, .11)
			}
			shade.Background(shade.AnimateColor("shade", darkness, motionColor))
			if hovered {
				fadeMotion(ui.Text(c.Key("dimensions"), fmt.Sprintf("%d × %d", width, height)).Absolute().Bottom(6).Right(6).
					Padding(3, 6).Radius(8).FontSize(12).Material(panelMaterial(c.Theme())).
					TextColor(c.Theme().Text).PassThrough())
			}
			if selected || focused {
				// 外圈不占图片空间，描边与图片之间留 2px。
				fadeMotion(ui.Box(c.Key("selection-ring")).Absolute().Left(-4).Top(-4).Right(-4).Bottom(-4).Radius(radius+4).
					Border(2, c.Theme().Accent).PassThrough())
			}
		}
		if hovered || asset.Favorite {
			button := ui.Button(c.Key("favorite"), "").Label(a.T("favorite")).Absolute().Top(6).Right(6).Size(28, 28).Padding(0).Radius(8).BorderWidth(0).Material(panelMaterial(c.Theme())).Cursor(ui.CursorPointer).Children(func() { appIcon(c, "heart").Size(18, 18) })
			if list {
				button.Top(18)
			}
			favoriteFocused := button.FocusVisible()
			wasFavoriteFocused := ui.Local(button, "keyboard-focus", func() bool { return false })
			if favoriteFocused && !*wasFavoriteFocused {
				ui.Box(c.Key("favorite-focus-reveal")).Absolute().Top(-a.contentTopInset() - 8).Left(0).Right(0).Bottom(0).
					PassThrough().ScrollIntoView()
			}
			*wasFavoriteFocused = favoriteFocused
			fadeMotion(button)
			ink, tint, border := c.Theme().TextMuted, c.Theme().Background.Alpha(.94), ui.Transparent
			if button.Hovered() {
				ink, tint = c.Theme().Text, c.Theme().SurfaceHover.Alpha(.94)
			}
			if button.Pressed() {
				tint = c.Theme().SurfacePressed.Alpha(.94)
			}
			if asset.Favorite {
				border = c.Theme().Accent
			}
			button.Material(frostedSurface{tint: button.AnimateColor("favorite-tint", tint, motionColor), blur: 22}).
				TextColor(button.AnimateColor("favorite-ink", ink, motionColor)).
				Border(1, button.AnimateColor("favorite-border", border, motionColor))
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
	if p == nil {
		return
	}
	ui.Column(c.Key("import-progress")).Transition(ui.ElementTransition{Animation: &motionSpring, Height: true, Enter: &ui.Motion{Y: -10, Collapse: true}, Exit: &ui.Motion{Y: -10, Collapse: true}}).Height(importProgressHeight).Shrink(0).Padding(12, 16).Gap(6).Material(toolbarMaterial(c.Theme())).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("import_stage_"+p.Stage)).Bold()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d / %d", p.Processed, p.Total))
		})
		ui.Text(c, a.T("import_counts", p.Imported, p.Skipped)).FontSize(12)
		ui.Text(c, filepath.Base(p.CurrentFile)).SingleLine().FontSize(12).TextColor(c.Theme().TextMuted)
		progress := float64(0)
		if p.Total > 0 {
			progress = float64(p.Processed) / float64(p.Total)
		}
		ui.Progress(c, progress).FillWidth()
	})
}
