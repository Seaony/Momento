package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func toolbarMotionFrames(tt *ui.Tester, crop image.Rectangle) []*image.RGBA {
	var frames []*image.RGBA
	previous := time.Duration(0)
	for _, elapsed := range []time.Duration{0, 87500 * time.Microsecond, 175 * time.Millisecond, 250 * time.Millisecond} {
		if elapsed > previous {
			tt.Advance(elapsed - previous)
		}
		frame := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
		draw.Draw(frame, frame.Bounds(), tt.Image(), crop.Min, draw.Src)
		frames = append(frames, frame)
		previous = elapsed
	}
	return frames
}

func writeToolbarMotionSheet(t *testing.T, name string, rows ...[]*image.RGBA) {
	t.Helper()
	dir := "/tmp/momento-polish/shots2/motion"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	width, height := rows[0][0].Bounds().Dx(), rows[0][0].Bounds().Dy()
	out := image.NewRGBA(image.Rect(0, 0, width*4, (height+24)*len(rows)))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.RGBA{255, 250, 245, 255}), image.Point{}, draw.Src)
	for row, frames := range rows {
		for i, frame := range frames {
			x, y := width*i, (height+24)*row
			drawer := font.Drawer{Dst: out, Src: image.NewUniform(color.RGBA{58, 35, 24, 255}), Face: basicfont.Face7x13, Dot: fixed.P(x+8, y+16)}
			drawer.DrawString([]string{"0%", "35%", "70%", "100%"}[i])
			draw.Draw(out, image.Rect(x, y+24, x+width, y+24+height), frame, image.Point{}, draw.Src)
		}
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, out); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestToolbarMotionScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_TOOLBAR_MOTION_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_TOOLBAR_MOTION_SCREENSHOTS=1 输出动效联系图")
	}
	newTester := func() (*App, *ui.Tester) {
		a := toolbarTestApp()
		a.Texts.Language = "zh-CN"
		a.Preferences.Theme = "light"
		a.InspectorVisible = false
		tt := ui.NewTester(func(c *ui.Context) {
			a.applyTheme(c)
			ui.Column(c).Fill().Background(c.Theme().Background).Children(func() { a.toolbar(c); ui.Spacer(c) })
		}, 900, 640)
		tt.Frame()
		tt.Advance(time.Second)
		return a, tt
	}
	t.Run("popover", func(t *testing.T) {
		a, tt := newTester()
		toolbarClick(t, tt, a.T("sort"))
		crop := image.Rect(600, 50, 900, 300)
		opening := toolbarMotionFrames(tt, crop)
		if bytes.Equal(opening[1].Pix, opening[3].Pix) {
			t.Fatal("浮层中间帧与终点相同")
		}
		tt.Advance(time.Second)
		tt.Key(0, ui.KeyEscape)
		closing := toolbarMotionFrames(tt, crop)
		if a.Toolbar.SortOpen {
			t.Fatal("浮层关闭状态没有生效")
		}
		if bytes.Equal(closing[0].Pix, closing[3].Pix) {
			t.Fatal("浮层关闭没有淡出帧")
		}
		writeToolbarMotionSheet(t, "toolbar-popover", opening, closing)
	})
	t.Run("segmented", func(t *testing.T) {
		a, tt := newTester()
		toolbarClick(t, tt, a.T("view_list"))
		if a.ViewMode != 2 {
			t.Fatal("视图没有切换")
		}
		frames := toolbarMotionFrames(tt, image.Rect(670, 0, 800, 64))
		if bytes.Equal(frames[1].Pix, frames[3].Pix) {
			t.Fatal("分段中间帧与终点相同")
		}
		writeToolbarMotionSheet(t, "toolbar-segmented", frames)
	})
}
