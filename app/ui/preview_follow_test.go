package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestPreviewFollowsImportSelectionOutsideItsOriginalItems(t *testing.T) {
	a := sidebarTestApp()
	a.applySnapshot(a.Snapshot)
	first, imported := a.Visible[0], a.Visible[1]
	a.Visible = a.Visible[:1]
	a.Selected.Add(first.ID)
	a.ShowPreview(first.ID, false)
	w := a.Preview.active
	a.selectImportResult([]string{imported.ID})
	if a.Preview.active != w || w.asset.ID != imported.ID || !a.Selected.Has(imported.ID) {
		t.Fatal("导入后自动选中关闭了预览或没有跟随新素材")
	}
	if !slices.ContainsFunc(w.items, func(item model.Asset) bool { return item.ID == imported.ID }) {
		t.Fatal("预览跟随后没有有效的翻页位置")
	}
}

func TestPreviewRecentRefreshDoesNotUndoPaging(t *testing.T) {
	a := sidebarTestApp()
	for i := range a.Snapshot.Assets {
		a.Snapshot.Assets[i].LastUsedAt = time.Now().Add(-time.Duration(i) * time.Hour)
	}
	a.Query.Scope = "recent"
	a.applySnapshot(a.Snapshot)
	if len(a.Visible) < 2 {
		t.Fatal("缺少翻页样本")
	}
	selected, next := a.Visible[0].ID, a.Visible[1].ID
	a.Selected.Add(selected)
	a.ShowPreview(selected, false)
	w := a.Preview.active
	w.move(a, 1)
	if w.asset.ID != next {
		t.Fatal("翻页没有显示下一张")
	}
	a.Index = catalog.New(a.Snapshot)
	a.refine(false)
	if w.asset.ID != next || !a.Selected.Has(selected) {
		t.Fatal("最近使用刷新把预览跳回主窗口原选中项")
	}
	a.Selected.Clear()
	a.Selected.Add(next)
	a.FollowPreviewSelection()
	a.Selected.Clear()
	a.Selected.Add(selected)
	a.FollowPreviewSelection()
	if w.asset.ID != selected {
		t.Fatal("实际选择变化没有更新预览")
	}
}
