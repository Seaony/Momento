package ui

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"path/filepath"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
)

func (a *App) MarkAssetsUsed(ids []string) {
	if a.Library == nil || a.Window == nil || len(ids) == 0 {
		return
	}
	ids = append([]string(nil), ids...)
	lib, gen, at := a.Library, a.generation, time.Now()
	go func() {
		a.workMu.Lock()
		err := lib.MarkUsed(a.ctx, ids, at)
		a.workMu.Unlock()
		a.Window.Update(func() {
			if gen != a.generation {
				return
			}
			if err != nil {
				a.ShowError(err)
				return
			}
			used := make(map[string]bool, len(ids))
			for _, id := range ids {
				used[id] = true
			}
			for i := range a.Snapshot.Assets {
				asset := &a.Snapshot.Assets[i]
				if used[asset.ID] && at.After(asset.LastUsedAt) {
					asset.LastUsedAt = at
					a.assetByID[asset.ID] = *asset
				}
			}
			a.Index.UpdateLastUsed(used, at)
			if a.Query.Scope == "recent" {
				a.refine(false)
			}
		})
	}()
}

func (a *App) CopySelected() {
	if a.IsLocked() || a.Library == nil {
		return
	}
	items := a.SelectedAssets()
	if len(items) == 0 {
		return
	}
	paths := make([]string, 0, len(items))
	ids := make([]string, 0, len(items))
	for _, asset := range items {
		paths = append(paths, filepath.Join(a.Library.Root, filepath.FromSlash(asset.RelativePath)))
		ids = append(ids, asset.ID)
	}
	if err := mygo.Clipboard.WriteFiles(paths...); err != nil {
		a.ShowError(err)
		return
	}
	a.MarkAssetsUsed(ids)
	a.Notify(a.T("copied"))
}

func (a *App) RotatePreview(delta int) {
	if w := a.previewTarget(); w != nil {
		a.rotatePreviewWindow(w, delta)
	}
}

func (a *App) rotatePreviewWindow(w *previewWindow, delta int) {
	if a.IsLocked() || a.Busy || w.rotating || w.closing || w.disposed {
		return
	}
	w.rotating = true
	w.operationError = ""
	asset, root := w.asset, w.root
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		a.workMu.Lock()
		updated, err := rotatePreviewAsset(ctx, root, asset, delta)
		var snapshot model.Snapshot
		if err == nil {
			lib, openErr := library.Open(ctx, root)
			if openErr == nil {
				snapshot, err = lib.Snapshot(ctx)
				err = errors.Join(err, lib.Close())
			} else {
				err = openErr
			}
		}
		a.workMu.Unlock()
		if err != nil {
			slog.Error("旋转预览失败", "library", root, "asset", asset.ID, "error", err)
		}
		mygo.RunOnMain(func() {
			if a.Thumbnails != nil && err == nil {
				a.Thumbnails.Invalidate(asset.ID)
			}
			if err == nil && a.Library != nil && a.Library.Root == root {
				a.applySnapshot(snapshot)
				a.layout = nil
			}
			targets := append([]*previewWindow(nil), a.Preview.pinned...)
			if a.Preview.active != nil {
				targets = append(targets, a.Preview.active)
			}
			for _, target := range targets {
				if target.root != root {
					continue
				}
				for i := range target.items {
					if target.items[i].ID == asset.ID && err == nil {
						target.items[i] = updated
					}
				}
				if target.asset.ID == asset.ID && err == nil {
					target.setAsset(a, updated)
				}
			}
			if !w.disposed {
				w.rotating = false
				if err != nil {
					w.operationError = a.Texts.Error(err)
				}
				w.invalidate()
			}
			if a.Window != nil {
				a.Window.Update(func() {})
			}
		})
	}()
	w.invalidate()
}

// 固定窗口可能属于已切走的图库，写入始终使用它自己保存的路径。
func rotatePreviewAsset(ctx context.Context, root string, asset model.Asset, delta int) (updated model.Asset, err error) {
	lib, err := library.Open(ctx, root)
	if err != nil {
		return asset, err
	}
	defer func() { err = errors.Join(err, lib.Close()) }()
	service := assets.New(root, lib.DB)
	updated = asset
	updated.Rotation = ((asset.Rotation+delta)%360 + 360) % 360
	if err = lib.SetRotation(ctx, []string{asset.ID}, updated.Rotation); err != nil {
		return asset, err
	}
	if err = service.RefreshThumbnails(ctx, []model.Asset{updated}); err != nil {
		rollback := lib.SetRotation(ctx, []string{asset.ID}, asset.Rotation)
		if rollback == nil {
			rollback = service.RefreshThumbnails(ctx, []model.Asset{asset})
		}
		return asset, errors.Join(err, rollback)
	}
	return updated, nil
}

func assetDisplaySize(asset model.Asset) (width, height int) {
	width, height = asset.Width, asset.Height
	if asset.Rotation == 90 || asset.Rotation == 270 {
		width, height = height, width
	}
	return
}
