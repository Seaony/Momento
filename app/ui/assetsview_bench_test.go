package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type assetsViewPerformanceFixture struct {
	app     *App
	tester  *ui.Tester
	sources []*ui.Bitmap
	height  int
}

func newAssetsViewPerformanceFixture(tb testing.TB, height, selected int) assetsViewPerformanceFixture {
	tb.Helper()
	const count = 100_000
	snapshot := model.Snapshot{Assets: make([]model.Asset, count)}
	for i := 0; i < 30; i++ {
		snapshot.Folders = append(snapshot.Folders, model.Folder{ID: fmt.Sprintf("folder-%02d", i), Name: fmt.Sprintf("Folder %02d", i+1), SortIndex: i})
	}
	for i := 0; i < 80; i++ {
		snapshot.Tags = append(snapshot.Tags, model.Tag{ID: fmt.Sprintf("tag-%02d", i), Name: fmt.Sprintf("Tag %02d", i+1)})
	}
	for i := range snapshot.Assets {
		width, height := 128, 96
		if i%3 == 1 {
			width, height = 96, 128
		}
		snapshot.Assets[i] = model.Asset{
			ID: fmt.Sprintf("%064x", i+1), Name: fmt.Sprintf("Perf asset %06d", i+1),
			Extension: "jpg", MediaType: "image/jpeg", OriginalFileName: fmt.Sprintf("reference-%06d.jpg", i+1),
			Width: width, Height: height, Size: 4096,
			CreatedAt: time.Date(2026, 1, 1, 12, 0, i, 0, time.UTC),
			FolderIDs: []string{snapshot.Folders[i%30].ID}, TagIDs: []string{snapshot.Tags[i%80].ID},
		}
	}
	a := &App{
		Library:        &library.Library{Root: filepath.Clean("/tmp/momento-performance.momento")},
		manifest:       model.Manifest{LibraryID: "performance", DisplayName: "100,000 references"},
		Texts:          i18n.Catalog{Language: "en", SystemLanguage: "en"},
		Preferences:    Preferences{Language: "en", Theme: "light"},
		SidebarVisible: true, InspectorVisible: true, SidebarWidth: 236,
		Query: catalog.Query{Sort: "added", Descending: true}, selectedIndex: -1, anchor: -1,
		ctx: context.Background(), cancel: func() {},
	}
	var missing atomic.Bool
	a.Thumbnails = newThumbnailCache(func(func()) {})
	// 正式有界缓存只装当前视口和前后预读区域，不按资产总数预装。
	a.Thumbnails.decode = func(path string) (*ui.Bitmap, int64, error) {
		if missing.CompareAndSwap(false, true) {
			tb.Errorf("性能样例缩略图未预装：%s", path)
		}
		return nil, 0, fmt.Errorf("性能样例缩略图未预装")
	}
	a.applySnapshot(snapshot)
	for _, asset := range a.Visible[:selected] {
		a.Selected.Add(asset.ID)
	}
	f := assetsViewPerformanceFixture{app: a, height: height}
	for _, ink := range []color.RGBA{{R: 180, G: 198, B: 189, A: 255}, {R: 196, G: 174, B: 153, A: 255}, {R: 152, G: 178, B: 196, A: 255}, {R: 195, G: 188, B: 153, A: 255}} {
		img := image.NewRGBA(image.Rect(0, 0, 128, 96))
		draw.Draw(img, img.Bounds(), image.NewUniform(ink), image.Point{}, draw.Src)
		f.sources = append(f.sources, ui.NewBitmap(img))
	}
	ratios := make([]float32, len(a.Visible))
	for i, asset := range a.Visible {
		ratios[i] = float32(asset.Width) / float32(asset.Height)
	}
	a.layout = NewMasonry(ratios, 1100-236-281-32)
	f.preload(0)
	f.tester = ui.NewTester(a.View, 1100, height)
	// 首次测量使用回退宽度，先让真实分栏尺寸稳定再定位到库中间。
	f.tester.Frame()
	f.tester.Frame()
	f.assertBounded(tb)
	return f
}

func (f *assetsViewPerformanceFixture) preload(y float32) {
	a := f.app
	h := float32(f.height)
	top := y - a.contentTopInset() - 16
	items := a.layout.Visible(max(0, top-h), top+2*h)
	a.Thumbnails.mu.Lock()
	defer a.Thumbnails.mu.Unlock()
	for _, item := range items {
		asset := a.Visible[item.Index]
		key := thumbnailKey{root: a.Library.Root, id: asset.ID}
		if element := a.Thumbnails.entries[key]; element != nil {
			a.Thumbnails.order.MoveToFront(element)
			continue
		}
		bitmap := f.sources[item.Index%len(f.sources)]
		w, h := bitmap.Size()
		cost := int64(w*h*4) * 4 / 3
		a.Thumbnails.entries[key] = a.Thumbnails.order.PushFront(thumbnailEntry{key: key, bitmap: bitmap, bytes: cost})
		a.Thumbnails.bytes += cost
		for len(a.Thumbnails.entries) > thumbnailItemLimit || a.Thumbnails.bytes > thumbnailByteLimit {
			a.Thumbnails.remove(a.Thumbnails.order.Back())
		}
	}
}

func (f *assetsViewPerformanceFixture) builtCards() int {
	seen := make(map[string]bool)
	for _, element := range f.tester.AccessAll() {
		if strings.HasPrefix(element.Name, "Perf asset ") {
			seen[element.Name] = true
		}
	}
	return len(seen)
}

func (f *assetsViewPerformanceFixture) assertBounded(tb testing.TB) {
	tb.Helper()
	if len(f.app.Snapshot.Assets) != 100_000 || len(f.app.Visible) != 100_000 || len(f.app.assetByID) != 100_000 {
		tb.Fatal("性能样例并非10万条真实主状态索引")
	}
	cards := f.builtCards()
	if cards == 0 || cards > 200 {
		tb.Fatalf("10万条视图构建 %d 张卡片，应只构建视口及预读区", cards)
	}
	f.app.Thumbnails.mu.Lock()
	defer f.app.Thumbnails.mu.Unlock()
	if len(f.app.Thumbnails.pending) != 0 {
		tb.Fatal("性能测试受后台缩略图任务干扰")
	}
	if len(f.app.Thumbnails.entries) > thumbnailItemLimit || f.app.Thumbnails.bytes > thumbnailByteLimit {
		tb.Fatal("性能样例绕过正式缓存上限")
	}
}

func TestAssetsView100KBuildsOnlyViewport(t *testing.T) {
	for _, height := range []int{800, 640} {
		for _, selected := range []int{0, 3} {
			t.Run(fmt.Sprintf("1100x%d/selected-%d", height, selected), func(t *testing.T) {
				f := newAssetsViewPerformanceFixture(t, height, selected)
				first := f.builtCards()
				middle := f.app.layout.Height() / 2
				f.preload(middle)
				f.preload(middle + float32(height)/2)
				f.app.scroll.Y = middle
				start := time.Now()
				f.tester.Frame()
				frameTime := time.Since(start)
				f.assertBounded(t)
				before := f.app.scroll.Y
				start = time.Now()
				f.tester.Scroll(400, float32(height)/2, 0, float32(height)/2)
				scrollTime := time.Since(start)
				f.assertBounded(t)
				if f.app.scroll.Y <= before {
					t.Fatal("Tester滚动没有推进素材区")
				}
				if len(f.app.SelectedAssets()) != selected {
					t.Fatal("选中项与主状态索引不一致")
				}
				t.Logf("assets=%d first-built=%d middle-built=%d cache=%d/%dB Frame=%s Scroll=%s scrollY=%.0f->%.0f",
					len(f.app.Visible), first, f.builtCards(), len(f.app.Thumbnails.entries), f.app.Thumbnails.bytes, frameTime, scrollTime, before, f.app.scroll.Y)
			})
		}
	}
}

func BenchmarkAssetsView100KFrame(b *testing.B) {
	benchmarkAssetsView100K(b, false)
}

func BenchmarkAssetsView100KScroll(b *testing.B) {
	benchmarkAssetsView100K(b, true)
}

func BenchmarkAssetsView100KSelectedAssets(b *testing.B) {
	for _, count := range []int{0, 3} {
		b.Run(fmt.Sprintf("selected-%d", count), func(b *testing.B) {
			f := newAssetsViewPerformanceFixture(b, 800, count)
			var selected []model.Asset
			b.ReportAllocs()
			for b.Loop() {
				selected = f.app.SelectedAssets()
			}
			if len(selected) != count {
				b.Fatal("选中项读取错误")
			}
		})
	}
}

func benchmarkAssetsView100K(b *testing.B, scroll bool) {
	for _, height := range []int{800, 640} {
		for _, selected := range []int{0, 3} {
			b.Run(fmt.Sprintf("1100x%d/selected-%d", height, selected), func(b *testing.B) {
				f := newAssetsViewPerformanceFixture(b, height, selected)
				middle := f.app.layout.Height() / 2
				f.preload(middle)
				f.preload(middle + float32(height)/2)
				f.app.scroll.Y = middle
				f.tester.Frame()
				f.assertBounded(b)
				b.ReportAllocs()
				delta := float32(height) / 2
				for b.Loop() {
					if scroll {
						f.tester.Scroll(400, float32(height)/2, 0, delta)
						delta = -delta
					} else {
						f.tester.Frame()
					}
				}
				f.assertBounded(b)
				b.ReportMetric(float64(f.builtCards()), "built-cards")
				b.ReportMetric(float64(len(f.app.Thumbnails.entries)), "cached-thumbs")
			})
		}
	}
}
