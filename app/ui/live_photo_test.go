package ui

import (
	"image/color"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func livePhotoAsset() model.Asset {
	return model.Asset{ID: "live", Name: "院子", RelativePath: "assets/li/live.heic", MotionPath: "assets/li/live.live.mov", Extension: "heic",
		Width: 400, Height: 300, Kind: model.KindLive, DurationMS: 1600, Size: 2 << 20}
}

// 悬停实况照片的卡片，播的是配对的那段视频。
func TestHoverLivePhotoPlaysMotion(t *testing.T) {
	var opened []string
	fakeHoverPlayer(t, &opened)
	a := &App{Preferences: DefaultPreferences(), Library: &library.Library{Root: "/lib"}}
	asset := livePhotoAsset()
	tt := ui.NewTester(func(c *ui.Context) {
		card := ui.Box(c.Key("card")).Size(200, 150)
		card.Children(func() { a.hoverVideoOverlay(c, card, asset, true, 8) })
	}, 300, 200)
	tt.Frame()
	tt.Advance(hoverVideoDelay + 50*time.Millisecond)
	if len(opened) != 1 || opened[0] != "/lib/assets/li/live.live.mov" {
		t.Fatalf("悬停实况照片没有播配对视频：%v", opened)
	}
}

// 预览实况照片：不碰不播；移到「实况」上播一遍（带声音），播完回到照片；翻走就关。
func TestPreviewLivePhotoPlaysOnBadge(t *testing.T) {
	opened, _ := useFakeVideos(t, func() *fakeVideo {
		return newFakeVideo(solidFrame(40, 30, color.RGBA{90, 120, 200, 255}), 10, 50*time.Millisecond)
	})
	items := []model.Asset{livePhotoAsset(), {ID: "next", Name: "下一张", RelativePath: "assets/next.png", Extension: "png", Width: 240, Height: 160, Kind: model.KindImage}}
	a := &App{Library: &library.Library{Root: "library.momento"}, Visible: items, Preferences: DefaultPreferences()}
	a.ShowPreview("live", false)
	w := a.Preview.active
	w.picture.Bitmap = ui.NewBitmap(solidFrame(400, 300, color.RGBA{200, 180, 120, 255}))
	tt := ui.NewTester(previewTestView(a), 900, 640)
	tt.Advance(50 * time.Millisecond)
	if len(*opened) != 0 {
		t.Fatal("没碰「实况」就开始播了")
	}
	badge, ok := tt.Access(a.T("live_photo_play"))
	if !ok {
		t.Fatal("实况照片预览没有「实况」")
	}
	tt.Move(badge.Bounds.X+badge.Bounds.W/2, badge.Bounds.Y+badge.Bounds.H/2)
	tt.Frame()
	want := filepath.Join("library.momento", "assets", "li", "live.live.mov")
	if len(*opened) != 1 || (*opened)[0] != want || w.live.player == nil || w.live.player.Muted() || w.live.player.Loop() {
		t.Fatalf("移到「实况」上应带声音播一遍配对视频：%v", *opened)
	}
	tt.Advance(time.Second)
	if w.live.player != nil {
		t.Fatal("播完没有回到照片")
	}
	// 一直停在上面不会反复播；点一下再播一遍。
	tt.Frame()
	if len(*opened) != 1 {
		t.Fatal("鼠标没动却又播了一遍")
	}
	if err := tt.Click(a.T("live_photo_play")); err != nil || w.live.player == nil || len(*opened) != 2 {
		t.Fatalf("点「实况」没有重播：%v", err)
	}
	w.move(a, 1)
	tt.Frame()
	if w.live.player != nil {
		t.Fatal("翻到下一张后实况视频没关")
	}
}
