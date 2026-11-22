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
			a.applyAssetsUsed(ids, at)
		})
	}()
}

// applyAssetsUsed 在界面线程把使用时间写进内存里的素材和索引。
func (a *App) applyAssetsUsed(ids []string, at time.Time) {
	used := make(map[string]bool, len(ids))
	for _, id := range ids {
		used[id] = true
	}
	for _, i := range a.Index.UpdateLastUsed(used, at) {
		asset := a.Snapshot.Assets[i]
		a.assetByID[asset.ID] = asset
	}
	// 「最近使用」的范围变了，筛选面板里的数量要重算。snapshotVersion 不动：它管的缓存（选中项、智能文件夹计数）都不看使用时间。
	a.indexVersion++
	// 正在看「最近使用」时，刚用过的要排到前面；选中项按素材找回位置。预览窗口有自己的翻页顺序，不受影响。
	if a.Query.Scope == "recent" {
		a.refine(false)
	}
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
