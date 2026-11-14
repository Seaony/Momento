package ui

import (
	"github.com/Seaony/Momento/app/internal/model"
	"testing"
)

func TestPreviewSessionKeepsOrderWhileRecentItemsMoveToFront(t *testing.T) {
	a := readyPreviewApp()
	a.Visible = []model.Asset{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	a.Preview = previewWindows{}
	a.ShowPreview("a", false)
	w := a.Preview.active
	w.move(a, 1)
	if w.asset.ID != "b" {
		t.Fatal("第一次翻页没有到b")
	}
	a.Visible = []model.Asset{{ID: "b"}, {ID: "a"}, {ID: "c"}}
	w.move(a, 1)
	if w.asset.ID != "c" {
		t.Fatal("最近使用重排改变了会话顺序")
	}
	w.move(a, 1)
	if w.asset.ID != "c" {
		t.Fatal("末尾翻页回绕")
	}
	w.move(a, -1)
	w.move(a, -1)
	w.move(a, -1)
	if w.asset.ID != "a" {
		t.Fatal("开头翻页回绕")
	}
}

func TestPreviewReopenCapturesNewOrder(t *testing.T) {
	a := readyPreviewApp()
	a.Visible = []model.Asset{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	a.Preview = previewWindows{}
	a.ShowPreview("a", false)
	first := a.Preview.active
	a.CloseAllPreviewWindows()
	a.Visible = []model.Asset{{ID: "c"}, {ID: "b"}, {ID: "a"}}
	a.ShowPreview("c", false)
	if a.Preview.active == first || a.Preview.active.items[0].ID != "c" || a.Preview.active.items[2].ID != "a" {
		t.Fatal("新会话没有采用新顺序")
	}
}
