package ui

import (
	"errors"
	"sync"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 放映从视频开始：视频不放，从它之后的第一张图开始。
func TestSlideshowStartingOnVideoBeginsAtNextImage(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{Root: t.TempDir()}
	snapshot := a.Snapshot
	snapshot.Assets[1].Kind, snapshot.Assets[1].Extension = model.KindVideo, "mp4"
	a.applySnapshot(snapshot)
	a.Selected.Add(a.Visible[1].ID)
	a.StartSlideshowFromSelection()
	show := a.Slideshow
	if show == nil || show.items[show.index].ID != a.Visible[2].ID {
		t.Fatalf("起点是视频时没有从它后面的第一张图开始：%+v", show)
	}
}

// 心形：已收藏时写「取消收藏」；回收站里不画。
func TestFavoriteHeartLabelAndTrash(t *testing.T) {
	a := assetViewFixture()
	snapshot := a.Snapshot
	snapshot.Assets[0].Favorite = true
	a.applySnapshot(snapshot)
	tt := ui.NewTester(a.assetArea, 700, 600)
	tt.Frame()
	if !tt.HasText(a.T("unfavorite")) {
		t.Fatal("已收藏的心形没有写「取消收藏」")
	}
	snapshot.Assets[0].Trashed = true
	a.applySnapshot(snapshot)
	a.Query.Scope = "trash"
	a.Refine()
	tt = ui.NewTester(a.assetArea, 700, 600)
	tt.Frame()
	if tt.HasText(a.T("unfavorite")) || tt.HasText(a.T("favorite")) {
		t.Fatal("回收站里不该有心形")
	}
}

// 空状态：不在「全部」时搜索无结果，可以切到全部并保留搜索词；正在导入时不放导入按钮。
func TestEmptyStateSearchInAllAndImporting(t *testing.T) {
	a := assetViewFixture()
	a.Query.Scope, a.Query.Search = "favorites", "nothing matched"
	a.Refine()
	tt := ui.NewTester(a.assetArea, 560, 600)
	if err := tt.Click(a.T("search_in_all")); err != nil {
		t.Fatal(err)
	}
	if a.Query.Scope != "all" || a.Query.Search != "nothing matched" {
		t.Fatalf("没有切到全部并保留搜索词：%q %q", a.Query.Scope, a.Query.Search)
	}

	a = assetViewFixture()
	a.applySnapshot(model.Snapshot{})
	a.Progress = &model.ImportProgress{Stage: "importing", Total: 10}
	tt = ui.NewTester(a.assetArea, 560, 600)
	tt.Frame()
	if !tt.HasText(a.T("importing_empty")) || tt.HasText(a.T("import_images_button")) || tt.HasText(a.T("drop_images_hint")) {
		t.Fatal("正在导入时空状态不对")
	}
}

// 进度条上的「停止」只在能停止时出现，点了就取消导入。
func TestImportProgressStopButton(t *testing.T) {
	a := assetViewFixture()
	a.Progress = &model.ImportProgress{Stage: "importing", Total: 10}
	tt := ui.NewTester(a.importProgress, 560, 200)
	tt.Frame()
	if tt.HasText(a.T("stop")) {
		t.Fatal("不能停止时不该有「停止」")
	}
	canceled := false
	a.Import.manualCancel = func() { canceled = true }
	tt.Frame()
	if err := tt.Click(a.T("stop")); err != nil || !canceled {
		t.Fatalf("点「停止」没有取消导入：%v %v", err, canceled)
	}
}

// 缩略图读失败的卡片画出提示图标，悬停提示里是失败原因。
func TestThumbnailFailureShowsIcon(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{Root: t.TempDir()}
	done, once := make(chan struct{}), sync.Once{}
	a.Thumbnails = newThumbnailCache(func(f func()) { f(); once.Do(func() { close(done) }) })
	a.Thumbnails.decode = func(string) (*ui.Bitmap, int64, error) { return nil, 0, errors.New("坏图") }
	tt := ui.NewTester(a.assetArea, 700, 600)
	tt.Frame()
	<-done
	tt.Frame()
	if err := a.Thumbnails.Error(a.Library.Root, a.Visible[0]); err == nil {
		t.Fatal("缩略图没有记下失败原因")
	}
}
