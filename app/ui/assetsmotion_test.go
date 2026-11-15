package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func assetsMotionFixture(t *testing.T) *App {
	t.Helper()
	a := assetViewFixture()
	a.Library = &library.Library{Root: t.TempDir()}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	t.Cleanup(a.cancel)
	a.Preferences = DefaultPreferences()
	a.Preferences.Onboarded = true
	a.Preferences.Theme = "light"
	a.Thumbnails = newThumbnailCache(func(func()) { t.Error("动画样本不应异步加载缩略图") })
	for i, asset := range a.Visible {
		img := image.NewRGBA(image.Rect(0, 0, 120, 80))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: uint8(95 + i*20), G: uint8(170 - i*12), B: uint8(140 - i*8), A: 255}), image.Point{}, draw.Src)
		key := thumbnailKey{root: a.Library.Root, id: asset.ID}
		bitmap := ui.NewBitmap(img)
		a.Thumbnails.entries[key] = a.Thumbnails.order.PushFront(thumbnailEntry{key: key, bitmap: bitmap, bytes: 120 * 80 * 4})
		a.Thumbnails.bytes += 120 * 80 * 4
	}
	return a
}

func assetMotionBounds(t *testing.T, tt *ui.Tester, name string) ui.Rect {
	t.Helper()
	box, ok := tt.Find(name)
	if !ok {
		t.Fatalf("没有找到卡片 %s", name)
	}
	return box
}

func rectDistance(a, b ui.Rect) float64 {
	return math.Abs(float64(a.X-b.X)) + math.Abs(float64(a.Y-b.Y)) + math.Abs(float64(a.W-b.W)) + math.Abs(float64(a.H-b.H))
}

func TestAssetReorderUsesStableIdentityAndIntermediatePosition(t *testing.T) {
	for _, mode := range viewModes {
		t.Run(string(rune('0'+mode)), func(t *testing.T) {
			a := assetsMotionFixture(t)
			a.ViewMode = mode
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 700, 960)
			tt.Advance(time.Second)
			before := assetMotionBounds(t, tt, "Photo 0")
			a.Query.Descending = true
			a.Refine()
			tt.Frame()
			start := assetMotionBounds(t, tt, "Photo 0")
			tt.Advance(87 * time.Millisecond)
			middle := assetMotionBounds(t, tt, "Photo 0")
			tt.Advance(163 * time.Millisecond)
			after := assetMotionBounds(t, tt, "Photo 0")
			endPixels := append([]byte(nil), tt.Image().Pix...)
			tt.Advance(time.Second)
			if !bytes.Equal(endPixels, tt.Image().Pix) {
				t.Fatal("排序100%帧仍有卡片残影或移动")
			}
			if rectDistance(before, start) > 1 || rectDistance(before, after) < 40 {
				t.Fatalf("重排没有从原位置开始：%+v -> %+v -> %+v", before, start, after)
			}
			if rectDistance(middle, before) < 1 || rectDistance(middle, after) < 1 {
				t.Fatalf("重排中间帧直接跳到端点：%+v -> %+v -> %+v", before, middle, after)
			}
		})
	}
}

func TestAssetViewSwitchCrossfadesWithoutCardMorphing(t *testing.T) {
	a := assetsMotionFixture(t)
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 700, 600)
	tt.Advance(time.Second)
	a.ViewMode = 2
	tt.Frame()
	start := assetMotionBounds(t, tt, "Photo 1")
	tt.Advance(87 * time.Millisecond)
	middle := assetMotionBounds(t, tt, "Photo 1")
	tt.Advance(163 * time.Millisecond)
	after := assetMotionBounds(t, tt, "Photo 1")
	endPixels := append([]byte(nil), tt.Image().Pix...)
	tt.Advance(time.Second)
	if !bytes.Equal(endPixels, tt.Image().Pix) {
		t.Fatal("100%帧仍有旧布局残影或未完成的变形")
	}
	for _, frame := range []ui.Rect{start, middle} {
		if math.Abs(float64(frame.W/frame.H-after.W/after.H)) > .02 || frame.W/after.W < .975 || frame.W/after.W > 1.005 {
			t.Fatalf("切换视图时卡片在逐张变形：%+v -> %+v -> %+v", start, middle, after)
		}
	}
	if math.Abs(float64(after.H-listRowHeight(a.Visible[1]))) > .01 {
		t.Fatalf("列表终点行高异常：%+v", after)
	}
}

// 调缩略图大小：卡片从原来的位置和大小弹到新的，中间帧在两者之间，之后完全落定。
// 700 宽时 1 和 1.5 两档列数不同；320 宽时列数相同、只改大小，两种都要有过渡。
func TestAssetSizeChangeGlidesCards(t *testing.T) {
	for _, width := range []int{700, 320} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := assetsMotionFixture(t)
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, width, 600)
			tt.Advance(time.Second)
			before := assetMotionBounds(t, tt, "Photo 1")
			columns := len(a.layout.columns)
			a.SetThumbnailSize(1.5)
			tt.Frame()
			start := assetMotionBounds(t, tt, "Photo 1")
			tt.Advance(87 * time.Millisecond)
			middle := assetMotionBounds(t, tt, "Photo 1")
			tt.Advance(time.Second)
			after := assetMotionBounds(t, tt, "Photo 1")
			endPixels := append([]byte(nil), tt.Image().Pix...)
			tt.Advance(time.Second)
			if !bytes.Equal(endPixels, tt.Image().Pix) {
				t.Fatal("调大小落定后仍有未完成的动画")
			}
			if (width == 700) == (len(a.layout.columns) == columns) {
				t.Fatalf("样本没有覆盖预期的列数变化：%d -> %d", columns, len(a.layout.columns))
			}
			if rectDistance(before, start) > 1 || rectDistance(before, after) < 5 {
				t.Fatalf("卡片没有从原布局出发或没有到新布局：%+v -> %+v -> %+v", before, start, after)
			}
			if rectDistance(middle, before) < 1 || rectDistance(middle, after) < 1 || rectDistance(middle, before)+rectDistance(middle, after) > rectDistance(before, after)+1 {
				t.Fatalf("中间帧不在新旧布局之间：%+v -> %+v -> %+v", before, middle, after)
			}
		})
	}
}

func TestAssetRemovalAndInsertionFadeOnlyVisibleCards(t *testing.T) {
	for _, mode := range viewModes {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			a := assetsMotionFixture(t)
			a.ViewMode = mode
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 700, 960)
			tt.Advance(time.Second)
			box := assetMotionBounds(t, tt, "Photo 5")
			x, y := int(box.X+box.W/2), int(box.Y+box.H/2)
			if mode == 2 {
				x, y = int(box.X+30), int(box.Y+30)
			}
			before := tt.Image().RGBAAt(x, y)
			a.Query.Search = "Photo 0"
			a.Refine()
			tt.Frame()
			start := tt.Image().RGBAAt(x, y)
			tt.Advance(87 * time.Millisecond)
			middle := tt.Image().RGBAAt(x, y)
			tt.Advance(time.Second)
			after := tt.Image().RGBAAt(x, y)
			if start != before || middle == before || middle == after {
				t.Fatalf("删除没有经过淡出：%v -> %v -> %v -> %v", before, start, middle, after)
			}
			if _, ok := tt.Find("Photo 5"); ok {
				t.Fatal("退场卡片仍接受交互")
			}
			a.Query.Search = ""
			a.Refine()
			tt.Frame()
			appearing := tt.Image().RGBAAt(x, y)
			tt.Advance(87 * time.Millisecond)
			appearingMiddle := tt.Image().RGBAAt(x, y)
			tt.Advance(time.Second)
			appeared := tt.Image().RGBAAt(x, y)
			if appearing != after || appearingMiddle == after || appearingMiddle == appeared {
				t.Fatalf("新增没有经过淡入：%v -> %v -> %v -> %v", after, appearing, appearingMiddle, appeared)
			}
		})
	}
}

func TestAssetImportProgressMovesContentWithIntermediateInset(t *testing.T) {
	a := assetsMotionFixture(t)
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill().Children(func() {
			a.assetArea(c)
			ui.Column(c.Key("header")).Absolute().Top(MainToolbarHeight).Left(0).Right(0).Children(func() { a.importProgress(c) })
		})
	}, 700, 600)
	tt.Advance(time.Second)
	before := assetMotionBounds(t, tt, "Photo 0")
	a.Progress = &model.ImportProgress{Stage: "importing", Total: 3}
	tt.Frame()
	start := assetMotionBounds(t, tt, "Photo 0")
	tt.Advance(87 * time.Millisecond)
	middle := assetMotionBounds(t, tt, "Photo 0")
	tt.Advance(time.Second)
	after := assetMotionBounds(t, tt, "Photo 0")
	if before.Y != start.Y || middle.Y <= before.Y || middle.Y >= after.Y || after.Y-before.Y != importProgressHeight {
		t.Fatalf("导入进度使内容直接跳动：%+v -> %+v -> %+v -> %+v", before, start, middle, after)
	}
	a.Progress = nil
	tt.Frame()
	tt.Advance(87 * time.Millisecond)
	closing := assetMotionBounds(t, tt, "Photo 0")
	if closing.Y <= before.Y || closing.Y >= after.Y {
		t.Fatalf("导入进度关闭没有中间位置：%+v", closing)
	}
}

func TestAssetHoverCapsuleAndSelectionRingFade(t *testing.T) {
	a := assetsMotionFixture(t)
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 700, 600)
	tt.Advance(time.Second)
	box := assetMotionBounds(t, tt, "Photo 0")
	x, y := int(box.X-3), int(box.Y+box.H/2)
	before := tt.Image().RGBAAt(x, y)
	a.selectAsset("0", 0, 0)
	tt.Frame()
	start := tt.Image().RGBAAt(x, y)
	tt.Advance(87 * time.Millisecond)
	middle := tt.Image().RGBAAt(x, y)
	tt.Advance(time.Second)
	after := tt.Image().RGBAAt(x, y)
	if start != before || middle == before || middle == after {
		t.Fatalf("选中描边没有淡入：%v -> %v -> %v -> %v", before, start, middle, after)
	}
	tt.Move(box.X+box.W/2, box.Y+box.H/2)
	dims := assetMotionBounds(t, tt, "600 × 400")
	x, y = int(dims.X+4), int(dims.Y+dims.H/2)
	capsuleStart := tt.Image().RGBAAt(x, y)
	tt.Advance(87 * time.Millisecond)
	capsuleMiddle := tt.Image().RGBAAt(x, y)
	tt.Advance(time.Second)
	capsuleEnd := tt.Image().RGBAAt(x, y)
	if capsuleMiddle == capsuleStart || capsuleMiddle == capsuleEnd {
		t.Fatalf("悬停尺寸胶囊没有淡入：%v -> %v -> %v", capsuleStart, capsuleMiddle, capsuleEnd)
	}
	tt.Move(0, 0)
	tt.Advance(87 * time.Millisecond)
	capsuleLeaving := tt.Image().RGBAAt(x, y)
	leavingImage := append([]byte(nil), tt.Image().Pix...)
	tt.Advance(time.Second)
	capsuleGone := tt.Image().RGBAAt(x, y)
	diff := 0
	goneImage := tt.Image()
	for yy := int(dims.Y) - 4; yy < int(dims.Y+dims.H)+4; yy++ {
		for xx := int(dims.X) - 2; xx < int(dims.X+dims.W)+2; xx++ {
			offset := goneImage.PixOffset(xx, yy)
			if !bytes.Equal(leavingImage[offset:offset+4], goneImage.Pix[offset:offset+4]) {
				diff++
			}
		}
	}
	if diff == 0 {
		t.Fatalf("悬停尺寸胶囊没有淡出：%v -> %v -> %v", capsuleEnd, capsuleLeaving, capsuleGone)
	}
}

func assertAssetsDoNotOverlap(t *testing.T, a *App, tt *ui.Tester) {
	t.Helper()
	for i, first := range a.Visible {
		one, ok := tt.Find(first.Name)
		if !ok {
			continue
		}
		for _, second := range a.Visible[i+1:] {
			two, ok := tt.Find(second.Name)
			if ok && min(one.X+one.W, two.X+two.W)-max(one.X, two.X) > .5 && min(one.Y+one.H, two.Y+two.H)-max(one.Y, two.Y) > .5 {
				t.Fatalf("落定后素材互相重叠：%s %+v / %s %+v", first.Name, one, second.Name, two)
			}
		}
	}
}

// 检查器展开时，第一帧就按最终宽度排版，卡片随栏宽一起弹过去。
func TestAssetInspectorLaysOutAtFinalWidthAndCardsGlide(t *testing.T) {
	a := assetsMotionFixture(t)
	a.SidebarVisible, a.SidebarWidth = true, 236
	tt := ui.NewTester(a.View, 1100, 640)
	tt.Advance(time.Second)
	before := assetMotionBounds(t, tt, "Photo 3")
	a.selectAsset(a.Visible[0].ID, 0, 0)
	tt.Frame()
	if math.Abs(float64(a.layoutWidth-551)) > .01 {
		t.Fatalf("检查器展开第一帧没有按最终宽度排版：%v", a.layoutWidth)
	}
	tt.Advance(87 * time.Millisecond)
	middle := assetMotionBounds(t, tt, "Photo 3")
	tt.Advance(time.Second)
	after := assetMotionBounds(t, tt, "Photo 3")
	if rectDistance(before, after) < 20 || rectDistance(middle, before) < 5 || rectDistance(middle, after) < 5 {
		t.Fatalf("卡片没有随检查器弹过去：%+v -> %+v -> %+v", before, middle, after)
	}
	endPixels := append([]byte(nil), tt.Image().Pix...)
	tt.Advance(time.Second)
	if !bytes.Equal(endPixels, tt.Image().Pix) {
		t.Fatal("检查器展开落定后仍有未完成的动画")
	}
	assertAssetsDoNotOverlap(t, a, tt)
}

// 侧栏收起和展开：栏宽一路变化，列数只换一次，不在半路来回跳。
func TestAssetSidebarToggleChangesColumnsOnce(t *testing.T) {
	a := assetsMotionFixture(t)
	a.SidebarVisible, a.SidebarWidth = true, 236
	tt := ui.NewTester(a.View, 1440, 900)
	tt.Advance(time.Second)
	for _, shown := range []bool{false, true} {
		a.SidebarVisible = shown
		tt.Frame()
		counts := map[int]bool{len(a.layout.columns): true}
		target := float32(0)
		if shown {
			target = 236
		}
		for i := 0; i < 125; i++ {
			tt.Advance(8 * time.Millisecond)
			counts[len(a.layout.columns)] = true
			if a.SidebarShownWidth() == target {
				break
			}
		}
		if len(counts) != 1 || a.SidebarShownWidth() != target {
			t.Fatalf("侧栏动画中列数变了不止一次或栏宽未落定：%v，%v", counts, a.SidebarShownWidth())
		}
		tt.Advance(time.Second)
		endPixels := append([]byte(nil), tt.Image().Pix...)
		tt.Advance(time.Second)
		if !bytes.Equal(endPixels, tt.Image().Pix) {
			t.Fatal("侧栏落定后仍有未完成的动画")
		}
		assertAssetsDoNotOverlap(t, a, tt)
	}
}

func TestAssetBlankClickClearsSelectionAndInspector(t *testing.T) {
	for _, mode := range viewModes {
		a := assetViewFixture()
		a.ViewMode = mode
		tt := ui.NewTester(a.assetArea, 700, 960)
		if err := tt.Click("Photo 0"); err != nil {
			t.Fatal(err)
		}
		if !a.InspectorVisible {
			t.Fatal("选择素材没有展开检查器")
		}
		a.CollapseInspector()
		if err := tt.Click("Photo 0"); err != nil {
			t.Fatal(err)
		}
		if !a.InspectorVisible {
			t.Fatal("点同一张素材没有重新展开检查器")
		}
		tt.ClickAt(690, 950)
		if a.Selected.Len() != 0 || a.InspectorVisible {
			t.Fatalf("视图%d点击空白没有清空选择并收起检查器", mode)
		}
	}
}

func TestAssetMotionContactSheets(t *testing.T) {
	if os.Getenv("MOMENTO_ASSET_MOTION_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ASSET_MOTION_SCREENSHOTS=1 生成卡片动画联系图")
	}
	dir := "/tmp/momento-polish/shots2/motion"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, state := range []string{"reorder", "resize", "selection", "view", "filter"} {
		t.Run(state, func(t *testing.T) {
			a := fixture.app(t, "small", "zh-CN", "light")
			a.SidebarVisible, a.InspectorVisible = false, false
			tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 920, 640)
			tt.Advance(time.Second)
			switch state {
			case "reorder":
				a.Query.Descending = !a.Query.Descending
				a.Refine()
			case "resize":
				a.Preferences.ThumbnailSize = 1.5
				a.layout = nil
			case "selection":
				a.selectAsset(a.Visible[1].ID, 1, 0)
			case "view":
				a.ViewMode = 2
			case "filter":
				a.Query.Search = a.Visible[0].Name
				a.Refine()
			}
			tt.Frame()
			sheet := image.NewRGBA(image.Rect(0, 0, 1840, 1328))
			elapsed := time.Duration(0)
			for i, at := range []time.Duration{0, 87 * time.Millisecond, 175 * time.Millisecond, 250 * time.Millisecond} {
				if at > elapsed {
					tt.Advance(at - elapsed)
				}
				elapsed = at
				frame := tt.Image()
				x, y := (i%2)*920, (i/2)*664
				draw.Draw(sheet, image.Rect(x, y, x+920, y+24), image.NewUniform(color.RGBA{R: 255, G: 250, B: 245, A: 255}), image.Point{}, draw.Src)
				d := font.Drawer{Dst: sheet, Src: image.NewUniform(color.RGBA{R: 58, G: 35, B: 24, A: 255}), Face: basicfont.Face7x13, Dot: fixed.P(x+12, y+17)}
				d.DrawString(fmt.Sprintf("%s  %d%%  (%dms)", state, []int{0, 35, 70, 100}[i], at.Milliseconds()))
				draw.Draw(sheet, image.Rect(x, y+24, x+920, y+664), frame, image.Point{}, draw.Src)
			}
			if state != "selection" {
				endPixels := append([]byte(nil), tt.Image().Pix...)
				tt.Advance(time.Second)
				if !bytes.Equal(endPixels, tt.Image().Pix) {
					t.Fatal("100%联系图仍有未完成动画或残影")
				}
			}
			f, err := os.Create(filepath.Join(dir, "assets-"+state+".png"))
			if err != nil {
				t.Fatal(err)
			}
			if err := png.Encode(f, sheet); err != nil {
				f.Close()
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
