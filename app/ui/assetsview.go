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
	if len(a.Visible) == 0 {
		ui.Column(c).Grow(1).FillWidth().Padding(a.contentTopInset(), 24, 24).Center().Gap(12).Children(func() {
			if a.Query.Search != "" {
				ui.Text(c, a.T("search_empty", a.Query.Search)).FontSize(20).Bold()
				ui.Text(c, a.T("result_count", "0")).TextColor(c.Theme().TextMuted)
			} else if len(a.Snapshot.Assets) > 0 {
				ui.Text(c, a.T("scope_empty")).FontSize(20).Bold()
			} else {
				ui.Text(c, a.T("drop_images")).FontSize(22).Bold()
				ui.Text(c, a.T("drop_images_hint")).TextColor(c.Theme().TextMuted)
				ui.Row(c).Gap(8).Children(func() {
					if ui.PrimaryButton(c, a.T("import")).Clicked() {
						a.ChooseImport(false)
					}
					if ui.Button(c, a.T("install_extension")).Clicked() {
						c.OpenURL("https://github.com/Seaony/Momento-Chomre-Extension/releases/latest")
					}
				})
			}
		})
		return
	}
	a.hovered = ""
	switch a.ViewMode {
	case 1:
		a.assetGrid(c)
	case 2:
		a.list.Key = func(i int) any { return a.Visible[i].ID }
		a.list.Selection = nil
		a.list.Selected = nil
		a.list.Label = func(i int) string { return a.Visible[i].Name }
		list := ui.List(c.Key("list"), &a.list, len(a.Visible), func(i int) { a.assetCard(c, a.Visible[i], i, true) }).Grow(1).Gap(0).Padding(a.contentTopInset()+8, 16, 16).ScrollbarInsets(a.contentTopInset(), 0, 0)
		if list.Submitted() && a.selectedIndex >= 0 && a.selectedIndex < len(a.Visible) {
			a.ActivateAsset(a.Visible[a.selectedIndex].ID, false)
		}
		a.assetPreviewShortcut(c, list)
	default:
		scroll := ui.Scroll(c.Key("masonry")).Grow(1).FillWidth().TrackScroll(&a.scroll).Padding(a.contentTopInset()+16, 16, 16).ScrollbarInsets(a.contentTopInset(), 0, 0)
		box := scroll.Bounds()
		w, h := box.W-32, box.H
		if w <= 0 {
			cw, ch := c.Size()
			w = cw - 32
			if a.SidebarVisible {
				w -= a.SidebarWidth + 1
			}
			if a.InspectorVisible {
				w -= 280
			}
			h = ch
		}
		if a.layout == nil || a.layoutVersion != a.visibleVersion || a.layoutWidth != w {
			ratios := make([]float32, len(a.Visible))
			for i, v := range a.Visible {
				vw, vh := assetDisplaySize(v)
				if vh > 0 {
					ratios[i] = float32(vw) / float32(vh)
				}
			}
			a.layout = NewMasonryScaled(ratios, w, a.thumbnailScale())
			a.layoutWidth = w
			a.layoutVersion = a.visibleVersion
		}
		if h <= 0 {
			h = 640
		}
		top, bottom := max(0, a.scroll.Y-a.contentTopInset()-16-h), a.scroll.Y-a.contentTopInset()-16+2*h
		scroll.Children(func() {
			ui.Box(c).Height(a.layout.Height()).FillWidth().Shrink(0).Children(func() {
				for _, item := range a.layout.Visible(top, bottom) {
					ui.Column(c.Key(a.Visible[item.Index].ID)).Absolute().Left(item.X).Top(item.Y).Size(item.Width, item.Height).Children(func() { a.assetCard(c, a.Visible[item.Index], item.Index, false) })
				}
			})
		})
		a.assetPreviewShortcut(c, scroll)
	}
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
}

func (a *App) assetCard(c *ui.Context, asset model.Asset, index int, list bool) {
	width, height := assetDisplaySize(asset)
	selected := a.Selected.Has(asset.ID)
	var card ui.Element
	if list {
		card = ui.Row(c.Key(asset.ID)).Height(64).FillWidth().Gap(16).Padding(8, 10).Radius(8).
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
	if list {
		switch {
		case selected && hovered:
			card.Background(c.Theme().Accent.Alpha(0.18))
		case selected:
			card.Background(c.Theme().Accent.Alpha(0.12))
		case hovered:
			card.Background(warmOverlay(c.Theme(), .06))
		}
	}
	card.Children(func() {
		if revealFocus {
			// 把页头遮住的高度算进定位目标，键盘聚焦时图片仍露在页头下方。
			ui.Box(c.Key("focus-reveal")).Absolute().Top(-a.contentTopInset() - 8).Left(0).Right(0).Bottom(0).
				PassThrough().ScrollIntoView()
		}
		thumb := a.Bitmap(asset)
		if list {
			if thumb != nil {
				ui.Image(c, thumb).Size(48, 48).Fit(ui.Cover).Radius(6)
			} else {
				ui.Box(c).Size(48, 48).Radius(6).Background(c.Theme().Surface)
			}
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, asset.Name).FontSize(13).Bold().SingleLine()
				ui.Text(c, fmt.Sprintf("%d × %d · %s", width, height, fileSize(asset.Size))).FontSize(12).TextColor(c.Theme().TextMuted)
			})
			ui.Text(c, strings.ToUpper(asset.Extension)).FontSize(12).TextColor(c.Theme().TextMuted).Margin(0, 36, 0, 0)
		} else {
			radius := float32(8)
			if a.ViewMode == 1 {
				radius = 10
				ui.Box(c).Fill().Padding(16).Children(func() {
					if thumb != nil {
						ui.Image(c, thumb).Fill().Fit(ui.Contain).Radius(4)
					}
				})
			} else if thumb != nil {
				ui.Image(c, thumb).Fill().Fit(ui.Cover).Radius(radius)
			} else {
				ui.Box(c).Fill().Radius(radius).Background(c.Theme().Surface)
			}
			if hovered {
				ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Radius(radius).
					Background(ui.RGBA(58, 35, 24, .07)).PassThrough()
				ui.Text(c, fmt.Sprintf("%d × %d", width, height)).Absolute().Bottom(6).Right(6).
					Padding(3, 6).Radius(8).FontSize(12).Material(panelMaterial(c.Theme())).
					TextColor(c.Theme().Text).PassThrough()
			}
			if selected || focused {
				// 外圈不占图片空间，描边与图片之间留 2px。
				ui.Box(c).Absolute().Left(-4).Top(-4).Right(-4).Bottom(-4).Radius(radius+4).
					Border(2, c.Theme().Accent).PassThrough()
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
			if asset.Favorite {
				button.Border(1, c.Theme().Accent)
			}
			if button.Clicked() {
				a.SetFavorite([]string{asset.ID}, !asset.Favorite)
			}
		}
	})
	if card.Clicked() {
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
	ui.Column(c).Height(importProgressHeight).Shrink(0).Padding(12, 16).Gap(6).Material(toolbarMaterial(c.Theme())).Children(func() {
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
