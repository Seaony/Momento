package ui

import (
	"log/slog"
	"path/filepath"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type previewState struct {
	ID         string
	Open       bool
	Held       bool
	Bitmap     *ui.Bitmap
	Loading    bool
	Generation uint64

	targetRoot string
	targetID   string
	errorText  string
}

func (p *previewState) target(root, id string) {
	if p.targetRoot == root && p.targetID == id {
		return
	}
	p.Generation++
	p.targetRoot, p.targetID = root, id
	p.Bitmap, p.errorText = nil, ""
}

func (p *previewState) finish(generation uint64, bitmap *ui.Bitmap, err error) {
	p.Loading = false
	if generation != p.Generation || !p.Open {
		return
	}
	if err != nil {
		p.errorText = err.Error()
		return
	}
	p.Bitmap = bitmap
}

func (a *App) preview(c *ui.Context) {
	p := &a.Preview
	if !p.Open {
		p.target("", "")
		return
	}
	if a.Library == nil || p.Held && !c.KeyDown(ui.KeySpace) {
		a.ClosePreview()
		p.target("", "")
		return
	}
	index := previewIndex(a.Visible, p.ID)
	if index < 0 {
		a.ClosePreview()
		p.target("", "")
		return
	}
	asset := a.Visible[index]
	p.target(a.Library.Root, asset.ID)
	if p.Bitmap == nil && p.errorText == "" && !p.Loading {
		a.loadPreview(asset, a.Library.Root)
	}
	width, height := c.Size()
	ui.DialogBase(c.Key("original-preview"), &p.Open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.78))
		panel.Size(width*0.86, height*0.86).Padding(12).Gap(12).
			Background(c.Theme().Background).Radius(12).Label(a.T("preview_original")).TabStop(false).AutoFocus()
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, asset.Name).Bold().Grow(1).SingleLine()
			ui.Text(c, a.T("preview_position", index+1, len(a.Visible))).FontSize(12).TextColor(c.Theme().TextMuted)
			if ui.Button(c.Key("close-preview"), "×").Label(a.T("close_preview")).Size(40, 40).Clicked() {
				a.ClosePreview()
			}
		})
		ui.Row(c).Grow(1).MinHeight(0).Gap(8).Children(func() {
			if ui.Button(c.Key("previous-preview"), "‹").Label(a.T("previous_asset")).Width(40).Height(48).Disabled(index == 0).Clicked() {
				a.movePreview(-1)
			}
			ui.Column(c).Grow(1).FillHeight().MinWidth(0).Center().Children(func() {
				switch {
				case p.Bitmap != nil:
					ui.Image(c, p.Bitmap).Fill().Fit(ui.Contain).Label(asset.Name)
				case p.errorText != "":
					ui.Text(c, a.T("preview_failed")).Bold()
					ui.Text(c, p.errorText).TextColor(c.Theme().TextMuted).MaxWidth(560)
				default:
					ui.Text(c, a.T("preview_loading")).TextColor(c.Theme().TextMuted)
				}
			})
			if ui.Button(c.Key("next-preview"), "›").Label(a.T("next_asset")).Width(40).Height(48).Disabled(index == len(a.Visible)-1).Clicked() {
				a.movePreview(1)
			}
		})
		if panel.OverlayShortcut(0, ui.KeyLeft) {
			a.movePreview(-1)
		}
		if panel.OverlayShortcut(0, ui.KeyRight) {
			a.movePreview(1)
		}
	})
	if !p.Open {
		p.target("", "")
	}
}

func previewIndex(visible []model.Asset, id string) int {
	for i := range visible {
		if visible[i].ID == id {
			return i
		}
	}
	return -1
}

func (a *App) movePreview(delta int) {
	index := previewIndex(a.Visible, a.Preview.ID)
	if index < 0 || index+delta < 0 || index+delta >= len(a.Visible) {
		return
	}
	a.Preview.ID = a.Visible[index+delta].ID
}

func (a *App) loadPreview(asset model.Asset, root string) {
	p := &a.Preview
	p.Loading = true
	generation := p.Generation
	win := a.Window
	path := filepath.Join(root, filepath.FromSlash(asset.RelativePath))
	go func() {
		img, _, _, _, err := assets.PrepareImage(path)
		var bitmap *ui.Bitmap
		if err == nil {
			bitmap = ui.NewBitmap(img)
		} else {
			slog.Error("原图预览加载失败", "asset", asset.ID, "error", err)
		}
		win.Update(func() {
			p.finish(generation, bitmap, err)
			if generation == p.Generation && p.Open && err != nil {
				p.errorText = a.Texts.Error(err)
			}
		})
	}()
}
