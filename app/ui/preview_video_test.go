package ui

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// fakeVideo 是离线测试用的解码器：同一张图重复 count 帧，每帧 step 长；Close 时记下来，用来确认播放器被关掉。
type fakeVideo struct {
	frame  image.Image
	count  int
	step   time.Duration
	next   int
	failAt int // 解到第几帧时报错，0 表示不报错
	once   sync.Once
	closed chan struct{}
}

func newFakeVideo(frame image.Image, count int, step time.Duration) *fakeVideo {
	return &fakeVideo{frame: frame, count: count, step: step, closed: make(chan struct{})}
}
func (f *fakeVideo) Next() (ui.VideoFrame, error) {
	if f.failAt > 0 && f.next+1 >= f.failAt {
		return ui.VideoFrame{}, errors.New("解码坏帧")
	}
	if f.next >= f.count {
		return ui.VideoFrame{}, io.EOF
	}
	f.next++
	return ui.VideoFrame{Image: f.frame, Duration: f.step}, nil
}
func (f *fakeVideo) Rewind() error           { f.next = 0; return nil }
func (f *fakeVideo) Duration() time.Duration { return time.Duration(f.count) * f.step }
func (f *fakeVideo) Close() error            { f.once.Do(func() { close(f.closed) }); return nil }
func (f *fakeVideo) waitClosed(t *testing.T, why string) {
	t.Helper()
	select {
	case <-f.closed:
	case <-time.After(2 * time.Second):
		t.Fatal(why)
	}
}

func solidFrame(width, height int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	return img
}

// useFakeVideos 把打开播放器换成假的：每次打开都新建一个解码器并记下路径；make 返回 nil 时按打不开处理。
func useFakeVideos(t *testing.T, newDecoder func() *fakeVideo) (opened *[]string, decoders *[]*fakeVideo) {
	t.Helper()
	paths, list := []string{}, []*fakeVideo{}
	previous := openPreviewVideo
	openPreviewVideo = func(path string) (*ui.VideoPlayer, error) {
		paths = append(paths, path)
		d := newDecoder()
		if d == nil {
			return nil, errors.New("系统不支持这种视频")
		}
		list = append(list, d)
		return ui.NewVideoPlayer(d)
	}
	t.Cleanup(func() { openPreviewVideo = previous })
	return &paths, &list
}

// 一个视频夹在两张图片中间，预览从视频开始。
func videoPreviewApp() *App { return videoPreviewAppAt("library.momento") }

func videoPreviewAppAt(root string) *App {
	items := []model.Asset{
		{ID: "before", Name: "前一张", RelativePath: "assets/before.png", Extension: "png", Width: 240, Height: 160, Kind: model.KindImage},
		{ID: "clip", Name: "海边", RelativePath: "assets/clip.mp4", Extension: "mp4", Width: 320, Height: 180, Kind: model.KindVideo, DurationMS: 30000, Size: 4 << 20},
		{ID: "after", Name: "后一张", RelativePath: "assets/after.png", Extension: "png", Width: 240, Height: 160, Kind: model.KindImage},
	}
	a := &App{Library: &library.Library{Root: root}, Visible: items, Preferences: DefaultPreferences()}
	a.ShowPreview("clip", false)
	return a
}

func TestPreviewVideoDoesNotDecodeAsImage(t *testing.T) {
	opened, _ := useFakeVideos(t, func() *fakeVideo {
		return newFakeVideo(solidFrame(32, 18, color.RGBA{200, 80, 40, 255}), 900, time.Second/30)
	})
	a := videoPreviewApp()
	w := a.Preview.active
	if w.needsPicture(a) {
		t.Fatal("视频还要走图片解码")
	}
	tt := ui.NewTester(previewTestView(a), 900, 640)
	tt.Advance(50 * time.Millisecond)
	if w.picture.Loading || w.decoding || w.picture.errorText != "" || w.picture.Bitmap != nil {
		t.Fatal("视频预览触发了图片解码或报了解码错误")
	}
	want := filepath.Join("library.momento", "assets", "clip.mp4")
	if len(*opened) != 1 || (*opened)[0] != want || w.video.player == nil || !w.video.player.Playing() {
		t.Fatalf("没有自动打开并播放视频：opened=%v", *opened)
	}
	if w.video.player.Loop() || w.video.player.Muted() {
		t.Fatal("预览视频应该带声音、只播一遍")
	}
	if !tt.HasText(videoDuration(30 * time.Second)) {
		t.Fatal("底栏和控制条没有显示时长")
	}
	// 缩放快捷键和触控板缩放对视频无效也不报错。
	a.PreviewZoomIn()
	a.PreviewActualSize()
	tt.Pinch(450, 300, 2)
	tt.Frame()
	if w.picture.errorText != "" || w.picture.Zoom != 1 {
		t.Fatal("缩放作用到了视频上")
	}
	w.move(a, 1)
	if !w.needsPicture(a) {
		t.Fatal("翻到图片后不再解码原图")
	}
}

func TestPreviewVideoPlayerClosesWhenLeaving(t *testing.T) {
	_, decoders := useFakeVideos(t, func() *fakeVideo {
		return newFakeVideo(solidFrame(32, 18, color.RGBA{40, 90, 200, 255}), 900, time.Second/30)
	})
	a := videoPreviewApp()
	w := a.Preview.active
	tt := ui.NewTester(previewTestView(a), 900, 640)
	tt.Advance(50 * time.Millisecond)
	if len(*decoders) != 1 {
		t.Fatal("没有打开播放器")
	}
	tt.Key(0, ui.KeyRight)
	if a.PreviewAssetID() != "after" {
		t.Fatal("右键没有翻到下一张")
	}
	(*decoders)[0].waitClosed(t, "翻到图片后旧播放器没关")
	if w.video.player != nil {
		t.Fatal("翻到图片后还留着播放器")
	}
	tt.Key(0, ui.KeyLeft)
	tt.Advance(50 * time.Millisecond)
	if a.PreviewAssetID() != "clip" || len(*decoders) != 2 || w.video.player == nil {
		t.Fatal("翻回视频没有重新打开播放器")
	}
	a.ClosePreview()
	(*decoders)[1].waitClosed(t, "关闭预览后播放器没关")
	if w.video.player != nil {
		t.Fatal("关闭预览后还留着播放器")
	}
}

func TestPreviewVideoClosingStopsSoundBeforeFadeEnds(t *testing.T) {
	_, decoders := useFakeVideos(t, func() *fakeVideo {
		return newFakeVideo(solidFrame(32, 18, color.RGBA{40, 90, 200, 255}), 900, time.Second/30)
	})
	a := videoPreviewApp()
	w := a.Preview.active
	tt := ui.NewTester(previewTestView(a), 900, 640)
	tt.Advance(50 * time.Millisecond)
	// 有窗口时关闭要先播 420ms 动画，播放器在动画开始时就关，最后一帧留着陪窗口淡出。
	w.stopVideo()
	(*decoders)[0].waitClosed(t, "开始关闭动画时没有停掉播放器")
	if w.video.player == nil || w.video.active() != nil {
		t.Fatal("关闭动画期间应只留最后一帧、不能再操作")
	}
	if w.videoShortcut(func(ui.Modifiers, ui.Key) bool { return true }, time.Now()) {
		t.Fatal("关闭动画期间视频快捷键仍然生效")
	}
	w.dispose()
	if w.video.player != nil {
		t.Fatal("销毁窗口后还留着播放器")
	}
}

func TestPreviewVideoShortcuts(t *testing.T) {
	useFakeVideos(t, func() *fakeVideo {
		return newFakeVideo(solidFrame(32, 18, color.RGBA{60, 160, 90, 255}), 900, time.Second/30)
	})
	a := videoPreviewApp()
	w := a.Preview.active
	tt := ui.NewTester(previewTestView(a), 900, 640)
	tt.Advance(50 * time.Millisecond)
	player := w.video.player
	tt.Key(0, ui.KeyK)
	if player.Playing() {
		t.Fatal("K 没有暂停")
	}
	tt.Key(ui.Shift, ui.KeyRight)
	tt.Frame()
	at := player.Position()
	if at < 5*time.Second || at > 5*time.Second+200*time.Millisecond {
		t.Fatalf("Shift+→ 后位置=%v", at)
	}
	tt.Key(ui.Shift, ui.KeyLeft)
	tt.Frame()
	if player.Position() > 200*time.Millisecond {
		t.Fatalf("Shift+← 后位置=%v", player.Position())
	}
	player.Seek(2 * time.Second)
	tt.Frame()
	before := player.Position()
	tt.Key(0, ui.KeyBracketRight)
	tt.Frame()
	if player.Playing() || player.Position() <= before || player.Position()-before > 100*time.Millisecond {
		t.Fatalf("] 没有暂停后前进一帧：%v → %v", before, player.Position())
	}
	tt.Key(0, ui.KeyBracketLeft)
	tt.Frame()
	if player.Position() != before {
		t.Fatalf("[ 没有后退一帧：%v", player.Position())
	}
	tt.Key(0, ui.KeyM)
	if !player.Muted() || !w.video.muted {
		t.Fatal("M 没有静音")
	}
	tt.Key(0, ui.KeyK)
	if !player.Playing() {
		t.Fatal("K 没有接着播放")
	}
	// 左右键仍然翻页；翻到图片后视频快捷键不再生效、不影响图片。
	tt.Key(0, ui.KeyRight)
	if a.PreviewAssetID() != "after" {
		t.Fatal("视频上右键没有翻页")
	}
	for _, key := range []ui.Key{ui.KeyK, ui.KeyM, ui.KeyBracketLeft} {
		tt.Key(0, key)
	}
	tt.Key(ui.Shift, ui.KeyLeft)
	if a.PreviewAssetID() != "after" || w.video.player != nil || w.closing {
		t.Fatal("视频快捷键影响了图片预览")
	}
	tt.Key(0, ui.KeyLeft)
	tt.Advance(50 * time.Millisecond)
	if a.PreviewAssetID() != "clip" || !w.video.player.Muted() {
		t.Fatal("翻回视频后没有记住静音")
	}
	// 空格仍然关闭预览。
	tt.KeyDown(ui.KeySpace)
	tt.KeyUp(ui.KeySpace)
	if a.PreviewOpen() || w.video.player != nil {
		t.Fatal("视频上空格没有关闭预览")
	}
}

func TestPreviewVideoControlsAndReplay(t *testing.T) {
	useFakeVideos(t, func() *fakeVideo {
		return newFakeVideo(solidFrame(32, 18, color.RGBA{60, 160, 90, 255}), 30, 50*time.Millisecond)
	})
	a := videoPreviewApp()
	w := a.Preview.active
	tt := ui.NewTester(previewTestView(a), 900, 640)
	tt.Advance(50 * time.Millisecond)
	for _, label := range []string{a.T("video_pause"), a.T("video_mute"), a.T("video_position")} {
		if _, ok := tt.Access(label); !ok {
			t.Fatalf("控制条缺少 %q", label)
		}
	}
	tt.Advance(2 * time.Second)
	player := w.video.player
	if !player.Ended() {
		t.Fatal("视频没有播完")
	}
	if err := tt.Click(a.T("video_play")); err != nil {
		t.Fatal("播完后没有可重新播放的按钮：", err)
	}
	if !player.Playing() || player.Position() > 100*time.Millisecond {
		t.Fatalf("重新播放没有从头开始：%v", player.Position())
	}
	if err := tt.Click(a.T("video_mute")); err != nil || !player.Muted() {
		t.Fatal("静音按钮无效")
	}
	if _, ok := tt.Access(a.T("video_unmute")); !ok {
		t.Fatal("静音后按钮没有换成取消静音")
	}
	// 点画面暂停；点过按钮后空格仍然关闭预览，不被按钮抢走。
	video, ok := tt.Access("海边")
	if !ok {
		t.Fatal("视频画面没有无障碍名称")
	}
	tt.ClickAt(video.Bounds.X+video.Bounds.W/2, video.Bounds.Y+video.Bounds.H/2)
	if player.Playing() {
		t.Fatal("点视频画面没有暂停")
	}
	tt.KeyDown(ui.KeySpace)
	tt.KeyUp(ui.KeySpace)
	if a.PreviewOpen() {
		t.Fatal("点过控制条按钮后空格不再关闭预览")
	}
}

func TestPreviewVideoUnplayable(t *testing.T) {
	for _, tc := range []struct {
		name       string
		missing    bool // 原文件已经不在
		newDecoder func() *fakeVideo
	}{
		{"打不开", false, func() *fakeVideo { return nil }},
		{"播放中出错", false, func() *fakeVideo {
			d := newFakeVideo(solidFrame(32, 18, color.RGBA{200, 80, 40, 255}), 90, time.Second/30)
			d.failAt = 3
			return d
		}},
		{"原文件不在", true, func() *fakeVideo { return nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			useFakeVideos(t, tc.newDecoder)
			var external []string
			previous := openPreviewVideoExternally
			openPreviewVideoExternally = func(path string) error { external = append(external, path); return nil }
			t.Cleanup(func() { openPreviewVideoExternally = previous })
			root := t.TempDir()
			if !tc.missing {
				if err := os.MkdirAll(filepath.Join(root, "assets"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "assets", "clip.mp4"), []byte("video"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			a := videoPreviewAppAt(root)
			w := a.Preview.active
			tt := ui.NewTester(previewTestView(a), 900, 640)
			tt.Advance(500 * time.Millisecond)
			message := a.T("video_unplayable")
			if tc.missing {
				message = a.T("video_missing")
			}
			if w.video.failed == nil || !tt.HasText(message) {
				t.Fatal("播放不了时没有提示")
			}
			if tc.missing {
				// 原文件不在，用默认应用打开也打不开，不放这个按钮。
				if tt.HasText(a.T("open_with_default_app")) {
					t.Fatal("原文件不在时不该有「用默认应用打开」")
				}
				return
			}
			if _, ok := tt.Access(a.T("video_pause")); ok {
				t.Fatal("播放不了时还显示控制条")
			}
			if err := tt.Click(a.T("open_with_default_app")); err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(root, "assets", "clip.mp4")
			if len(external) != 1 || external[0] != want {
				t.Fatalf("没有用默认应用打开原文件：%v", external)
			}
			if w.videoShortcut(func(ui.Modifiers, ui.Key) bool { return true }, time.Now()) {
				t.Fatal("播放不了时视频快捷键仍然生效")
			}
		})
	}
}

// 视频预览离线截图：正在播放（控制条显示）、播放不了，中文浅色、深色各一张。
func TestPreviewVideoScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_PREVIEW_VIDEO_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_PREVIEW_VIDEO_SCREENSHOTS=1 输出视频预览截图")
	}
	dir := "/tmp/momento-stage4/preview"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, state := range []string{"playing", "unplayable"} {
		for _, theme := range []string{"light", "dark"} {
			a := fixture.app(t, "small", "zh-CN", theme)
			index := -1
			for i, item := range a.Visible {
				if item.Extension == "jpg" && item.Width > item.Height {
					index = i
					break
				}
			}
			if index < 0 {
				t.Fatal("样例库里没有横图")
			}
			asset := a.Visible[index]
			frame, _, _, _, err := assets.PrepareImage(filepath.Join(fixture.root, filepath.FromSlash(asset.RelativePath)))
			if err != nil {
				t.Fatal(err)
			}
			// 按常见的 1080p 视频摆窗口，样例小图放大当画面。
			asset.Name, asset.Extension, asset.Kind, asset.DurationMS = "海边日落延时", "mp4", model.KindVideo, 45000
			asset.Width, asset.Height, asset.Size, asset.Rotation = 1920, 1920*frame.Bounds().Dy()/frame.Bounds().Dx(), 38<<20, 0
			a.Visible[index] = asset
			if a.assetByID != nil {
				a.assetByID[asset.ID] = asset
			}
			previous := openPreviewVideo
			openPreviewVideo = func(string) (*ui.VideoPlayer, error) {
				if state == "unplayable" {
					return nil, errors.New("系统不支持 WebM")
				}
				return ui.NewVideoPlayer(newFakeVideo(frame, 45, time.Second))
			}
			a.ShowPreview(asset.ID, false)
			w := a.Preview.active
			w.placeholder = fixture.bitmaps[asset.ID]
			canvas := mygo.Rectangle{Width: 900, Height: 640}
			frameRect := previewLayout(canvas, mygo.Rectangle{}, asset, a.Preferences.PreviewFit == "fit", a.Texts.Locale(), 1)
			tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, frameRect.Width, frameRect.Height)
			tt.SetLocale(a.Texts.Locale())
			w.openedAt = tt.Now().Add(-time.Second)
			tt.Frame()
			if state == "playing" {
				w.video.player.Seek(12 * time.Second)
				tt.Move(float32(frameRect.Width)/2, float32(frameRect.Height)/3)
				tt.Advance(300 * time.Millisecond)
			} else {
				tt.Advance(300 * time.Millisecond)
			}
			out := image.NewRGBA(image.Rect(0, 0, canvas.Width, canvas.Height))
			paintDesktop(out, theme == "dark")
			draw.Draw(out, image.Rect(frameRect.X, frameRect.Y, frameRect.X+frameRect.Width, frameRect.Y+frameRect.Height), tt.Image(), image.Point{}, draw.Over)
			writePreviewWindowContact(t, filepath.Join(dir, state+"-"+theme+".png"), out)
			a.CloseAllPreviewWindows()
			openPreviewVideo = previous
		}
	}
}
