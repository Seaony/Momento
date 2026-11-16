package ui

import (
	"image"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 放映：跳过视频，从选中的那张开始；到间隔自动下一张，空格暂停后不翻；左右翻页到头接着另一头；改间隔记进偏好；Esc 退出。
func TestSlideshow(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{Root: t.TempDir()}
	snapshot := a.Snapshot
	snapshot.Assets[1].Kind, snapshot.Assets[1].Extension = model.KindVideo, "mp4"
	a.applySnapshot(snapshot)
	a.Selected.Add(a.Visible[2].ID)
	a.StartSlideshowFromSelection()
	show := a.Slideshow
	if show == nil || len(show.items) != len(a.Visible)-1 || show.items[show.index].ID != a.Visible[2].ID {
		t.Fatalf("放映没有跳过视频或没从选中的开始：%+v", show)
	}
	for _, item := range show.items {
		show.bitmaps[item.ID] = ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 40, 30)))
	}
	start := show.index
	tt := ui.NewTester(func(c *ui.Context) {
		if s := a.Slideshow; s != nil {
			s.view(c, a)
		}
	}, 800, 600)
	show.shownAt = tt.Now()
	tt.Advance(a.slideshowInterval() + 100*time.Millisecond)
	tt.Frame()
	if show.index != start+1 {
		t.Fatalf("到间隔没有自动下一张：%d", show.index)
	}
	tt.Key(0, ui.KeySpace)
	// 按住空格的重复按键不能来回切换暂停。
	tt.KeyRepeat(0, ui.KeySpace)
	tt.KeyRepeat(0, ui.KeySpace)
	tt.KeyRepeat(0, ui.KeySpace)
	tt.Advance(3 * a.slideshowInterval())
	tt.Frame()
	if !show.paused || show.index != start+1 {
		t.Fatal("暂停后不该翻页")
	}
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyLeft)
	if show.index != (start+1-3+len(show.items))%len(show.items) {
		t.Fatalf("往前翻到头应接着最后一张：%d", show.index)
	}
	tt.Move(400, 300)
	tt.Frame()
	if err := tt.Click(a.T("slideshow_seconds", 10)); err != nil {
		t.Fatal(err)
	}
	if a.Preferences.SlideshowSeconds != 10 || a.slideshowInterval() != 10*time.Second {
		t.Fatal("改间隔没有记下")
	}
	tt.Key(0, ui.KeyEscape)
	if a.Slideshow != nil {
		t.Fatal("Esc 没有退出放映")
	}
	// 只有视频时不放映，给出提示。
	only := assetViewFixture()
	only.Library = &library.Library{Root: t.TempDir()}
	snap := only.Snapshot
	for i := range snap.Assets {
		snap.Assets[i].Kind = model.KindVideo
	}
	only.applySnapshot(snap)
	only.StartSlideshow("")
	if only.Slideshow != nil {
		t.Fatal("没有图片时不该开始放映")
	}
}

// 放映解图：一次只解一张；翻出前后范围或放映结束后才解好的图不建位图。
func TestSlideshowDecodeBounded(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{Root: t.TempDir()}
	a.StartSlideshow(a.Visible[0].ID)
	show := a.Slideshow
	show.decoding = "busy"
	show.load(a)
	if len(show.bitmaps) != 0 || len(show.failed) != 0 || show.decoding != "busy" {
		t.Fatal("已有一张在解时不该再开新的解码")
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	far := show.items[len(show.items)/2].ID
	show.loaded(a, far, img, nil)
	if show.bitmaps[far] != nil {
		t.Fatal("翻出前后范围的图不该留")
	}
	show.loaded(a, "busy", img, nil)
	if show.decoding != "" {
		t.Fatal("解完没有让出位置")
	}
	current := show.items[show.index].ID
	a.StopSlideshow()
	show.loaded(a, current, img, nil)
	if show.bitmaps[current] != nil {
		t.Fatal("放映结束后解好的图不该再建位图")
	}
}

// 等高行：工具栏切到等高行；左右是前后一张，上下到上一行、下一行。
func TestJustifiedViewKeys(t *testing.T) {
	a := assetViewFixture()
	tt := ui.NewTester(a.assetArea, 700, 600)
	a.ViewMode = 1
	tt.Frame()
	if a.justified.layout == nil || len(a.justified.layout.items) != len(a.Visible) {
		t.Fatal("等高行没有排出来")
	}
	if err := tt.Click("Photo 0"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	if !a.Selected.Has(a.Visible[1].ID) {
		t.Fatal("右键应选下一张")
	}
	tt.Key(0, ui.KeyDown)
	l := a.justified.layout
	if got := a.selectedIDs(); len(got) != 1 || l.row(a.selectedIndex) != l.row(1)+1 {
		t.Fatalf("下键应到下一行：%v 行 %d", got, l.row(a.selectedIndex))
	}
}
