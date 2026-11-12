package ui

import (
	"errors"
	"image"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestPreviewFollowsVisibleOrderWithoutWrapping(t *testing.T) {
	a := &App{Visible: []model.Asset{{ID: "c"}, {ID: "a"}, {ID: "b"}}, Preview: previewState{ID: "a", Open: true}}
	a.movePreview(1)
	if a.Preview.ID != "b" {
		t.Fatalf("下一张 = %q", a.Preview.ID)
	}
	a.movePreview(1)
	if a.Preview.ID != "b" {
		t.Fatal("在列表末尾回到了开头")
	}
	a.movePreview(-1)
	a.movePreview(-1)
	a.movePreview(-1)
	if a.Preview.ID != "c" {
		t.Fatalf("上一张 = %q", a.Preview.ID)
	}
	a.Preview.ID = "filtered-out"
	a.movePreview(1)
	if a.Preview.ID != "filtered-out" {
		t.Fatal("跳到了当前结果以外的素材")
	}
}

func TestPreviewDropsResultsAfterSwitchingAssetOrLibrary(t *testing.T) {
	p := previewState{Open: true}
	bitmap := ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 8)))
	p.target("first.momento", "one")
	first := p.Generation
	p.Loading = true
	p.target("first.momento", "two")
	p.finish(first, bitmap, nil)
	if p.Bitmap != nil || p.Loading {
		t.Fatal("旧素材的解码结果污染了新素材")
	}
	second := p.Generation
	p.target("second.momento", "two")
	p.finish(second, bitmap, nil)
	if p.Bitmap != nil {
		t.Fatal("另一个图库的图片污染了当前图库")
	}
	p.finish(p.Generation, bitmap, nil)
	if p.Bitmap != bitmap {
		t.Fatal("当前原图没有显示")
	}
}

func TestPreviewFailureVisibleAndCloseDropsPendingResult(t *testing.T) {
	p := previewState{Open: true}
	p.target("library.momento", "one")
	p.Loading = true
	p.finish(p.Generation, nil, errors.New("文件已移走"))
	if p.errorText != "文件已移走" || p.Loading {
		t.Fatal("原图加载失败没有留下可见错误")
	}
	p.Open = false
	p.finish(p.Generation, ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 4, 4))), nil)
	if p.Bitmap != nil {
		t.Fatal("关闭预览后接受了待处理图片")
	}
	p.target("", "")
	if p.errorText != "" {
		t.Fatal("预览关闭后没有释放错误状态")
	}
}

func TestPreviewEscapeClosesDialog(t *testing.T) {
	a := readyPreviewApp()
	tt := ui.NewTester(a.preview, 1100, 640)
	if !tt.HasText("Preview original") {
		if _, ok := tt.Find(a.T("close_preview")); !ok {
			t.Fatal("没有渲染预览浮层")
		}
	}
	tt.Key(0, ui.KeyEscape)
	if a.Preview.Open || a.Preview.Bitmap != nil {
		t.Fatal("Escape 没有关闭预览并释放原图")
	}
}

func TestPreviewSpaceReleaseClosesHeldPreview(t *testing.T) {
	a := readyPreviewApp()
	tt := ui.NewTester(a.preview, 1100, 640)
	tt.KeyDown(ui.KeySpace)
	a.Preview.Held = true
	tt.Frame()
	if !a.Preview.Open {
		t.Fatal("空格仍按住时预览已关闭")
	}
	tt.KeyRepeat(0, ui.KeySpace)
	if !a.Preview.Open {
		t.Fatal("空格自动重复触发了关闭按钮")
	}
	tt.KeyUp(ui.KeySpace)
	if a.Preview.Open || a.Preview.Bitmap != nil {
		t.Fatal("松开空格没有关闭预览并释放原图")
	}
}

func readyPreviewApp() *App {
	a := &App{
		Library: &library.Library{Root: "library.momento"},
		Visible: []model.Asset{{ID: "one", Name: "Preview original", RelativePath: "assets/one.png"}},
		Preview: previewState{ID: "one", Open: true},
	}
	a.Preview.target(a.Library.Root, a.Preview.ID)
	a.Preview.Bitmap = ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 240, 160)))
	return a
}
