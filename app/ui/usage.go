package ui

import (
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

func assetDisplaySize(asset model.Asset) (width, height int) {
	width, height = asset.Width, asset.Height
	if asset.Rotation == 90 || asset.Rotation == 270 {
		width, height = height, width
	}
	return
}
