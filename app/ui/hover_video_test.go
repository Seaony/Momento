package ui

import (
	"image"
	"image/color"
	"io"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 假视频：两秒、10 帧纯色画面。
type fakeVideoDecoder struct{ frame int }

func (d *fakeVideoDecoder) Next() (ui.VideoFrame, error) {
	if d.frame >= 10 {
		return ui.VideoFrame{}, io.EOF
	}
	d.frame++
	img := image.NewRGBA(image.Rect(0, 0, 16, 9))
	for i := range img.Pix {
		img.Pix[i] = 0x80
	}
	img.Set(0, 0, color.White)
	return ui.VideoFrame{Image: img, Duration: 200 * time.Millisecond}, nil
}
func (d *fakeVideoDecoder) Rewind() error           { d.frame = 0; return nil }
func (d *fakeVideoDecoder) Duration() time.Duration { return 2 * time.Second }

func fakeHoverPlayer(t *testing.T, opened *[]string) {
	t.Helper()
	previous := openHoverVideo
	openHoverVideo = func(path string) (*ui.VideoPlayer, error) {
		*opened = append(*opened, path)
		return ui.NewVideoPlayer(&fakeVideoDecoder{})
	}
	t.Cleanup(func() { openHoverVideo = previous })
}

// 鼠标停在视频卡片上一会儿才打开播放器，静音循环；离开卡片就关掉；关掉设置后不再播。
func TestHoverVideoOpensAfterDelayAndClosesOnLeave(t *testing.T) {
	var opened []string
	fakeHoverPlayer(t, &opened)
	a := &App{Preferences: DefaultPreferences(), Library: &library.Library{Root: "/lib"}}
	asset := model.Asset{ID: "v", Kind: model.KindVideo, RelativePath: "assets/v/v.mp4", DurationMS: 2000}
	hovered := true
	tt := ui.NewTester(func(c *ui.Context) {
		a.finishHoverVideo()
		a.hovered = ""
		card := ui.Box(c.Key("card")).Size(200, 120)
		card.Children(func() {
			if hovered {
				a.hovered = asset.ID
			}
			a.hoverVideoOverlay(c, card, asset, hovered, 8)
		})
	}, 300, 200)
	tt.Frame()
	if len(opened) != 0 {
		t.Fatal("刚悬停就打开了播放器")
	}
	tt.Advance(hoverVideoDelay + 50*time.Millisecond)
	tt.Frame()
	if len(opened) != 1 || opened[0] != "/lib/assets/v/v.mp4" {
		t.Fatalf("悬停够久没有打开对应视频：%v", opened)
	}
	player := a.hoverVideoState.player
	if player == nil || !player.Muted() || !player.Loop() {
		t.Fatal("悬停播放应静音循环")
	}
	hovered = false
	tt.Frame()
	tt.Frame()
	if a.hoverVideoState.player != nil || a.hoverVideoState.id != "" {
		t.Fatal("鼠标离开后播放器没有关")
	}
	a.Preferences.HoverPlayVideos = false
	hovered = true
	tt.Advance(time.Second)
	tt.Frame()
	if len(opened) != 1 {
		t.Fatal("关掉悬停播放后仍打开了播放器")
	}
}

func TestVideoDurationText(t *testing.T) {
	for d, want := range map[time.Duration]string{
		300 * time.Millisecond: "0:01", 12 * time.Second: "0:12", 12*time.Minute + 5*time.Second: "12:05",
		time.Hour + 2*time.Minute + 3*time.Second: "1:02:03", 0: "0:00",
	} {
		if got := videoDuration(d); got != want {
			t.Fatalf("%v：%s，期望 %s", d, got, want)
		}
	}
}

// 保存框里视频的默认文件名用视频自己的扩展名，不跟着选的图片格式走。
func TestExportDialogFileNameForVideo(t *testing.T) {
	asset := model.Asset{Name: "clip", Extension: "mov", OriginalFileName: "IMG_0001.MOV", Kind: model.KindVideo}
	if got := exportDialogFileName(asset, model.ExportJPEG); got != "IMG_0001.MOV" {
		t.Fatalf("视频默认文件名：%s", got)
	}
}
