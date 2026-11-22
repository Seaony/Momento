package ui

import (
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

// 关闭动画还没播完时再按空格，新预览立刻打开；旧窗口照样播完后被清掉。
func TestPreviewReopensWhilePreviousIsClosing(t *testing.T) {
	a := readyPreviewApp()
	old := a.Preview.active
	old.closing, old.closedAt = true, time.Now()
	a.ShowPreview("one", false)
	w := a.Preview.active
	if w == nil || w == old || !a.PreviewOpen() {
		t.Fatal("关闭动画期间再次预览没有打开新窗口")
	}
	if !slices.Contains(a.Preview.fading, old) {
		t.Fatal("正在关闭的旧窗口没被记下，关不掉")
	}
	a.removePreviewWindow(old)
	if len(a.Preview.fading) != 0 || a.Preview.active != w {
		t.Fatal("旧窗口播完后误动了新预览")
	}
	old2 := w
	old2.closing = true
	a.ShowPreview("one", false)
	a.CloseAllPreviewWindows()
	if !old2.disposed || len(a.Preview.fading) != 0 {
		t.Fatal("全部关闭时漏掉了还在播关闭动画的窗口")
	}
}

// 翻页跳过打开预览后删掉或被筛掉的图，顺序仍按打开时的。
func TestPreviewPagingSkipsRemovedAssets(t *testing.T) {
	a := readyPreviewApp()
	a.Visible = []model.Asset{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	a.Preview = previewWindows{}
	a.ShowPreview("a", false)
	w := a.Preview.active
	// b 被删掉或筛掉，c 移到最前（最近使用排序）。
	a.Visible = []model.Asset{{ID: "c"}, {ID: "a"}, {ID: "d"}}
	a.visibleVersion++
	w.move(a, 1)
	if w.asset.ID != "c" {
		t.Fatalf("翻到了已不在列表里的图：%s", w.asset.ID)
	}
	w.move(a, 1)
	if w.asset.ID != "d" {
		t.Fatalf("没有按打开时的顺序翻页：%s", w.asset.ID)
	}
	// 当前图已不在主窗口列表（换了范围）：只去掉已不存在或进了回收站的图。
	a.Visible = []model.Asset{{ID: "x"}}
	a.visibleVersion++
	a.assetByID = map[string]model.Asset{"a": {ID: "a"}, "c": {ID: "c", Trashed: true}, "d": {ID: "d"}}
	w.move(a, -1)
	if w.asset.ID != "a" {
		t.Fatalf("翻到了已移进回收站的图：%s", w.asset.ID)
	}
}
