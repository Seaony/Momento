package assets

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"golang.org/x/image/draw"
)

// 快速转正要和逐像素转换的结果一模一样，包括子图、预乘透明和各种色度采样。
func TestOrientFastPathMatchesSlowPath(t *testing.T) {
	sources := map[string]image.Image{}
	nrgba := image.NewNRGBA(image.Rect(0, 0, 13, 9))
	rgba := image.NewRGBA(image.Rect(0, 0, 13, 9))
	gray := image.NewGray(image.Rect(0, 0, 13, 9))
	for y := range 9 {
		for x := range 13 {
			c := color.NRGBA{uint8(x * 19), uint8(y * 27), uint8(x*y + 7), uint8(255 - (x+y)%4*60)}
			nrgba.SetNRGBA(x, y, c)
			rgba.Set(x, y, c)
			gray.SetGray(x, y, color.Gray{uint8(x*y*3 + 1)})
		}
	}
	sources["nrgba"] = nrgba.SubImage(image.Rect(2, 1, 12, 8))
	sources["rgba"] = rgba.SubImage(image.Rect(1, 2, 11, 9))
	sources["gray"] = gray.SubImage(image.Rect(3, 0, 13, 7))
	for _, ratio := range []image.YCbCrSubsampleRatio{image.YCbCrSubsampleRatio444, image.YCbCrSubsampleRatio422, image.YCbCrSubsampleRatio420, image.YCbCrSubsampleRatio440, image.YCbCrSubsampleRatio411, image.YCbCrSubsampleRatio410} {
		ycc := image.NewYCbCr(image.Rect(0, 0, 13, 9), ratio)
		for i := range ycc.Y {
			ycc.Y[i] = uint8(i * 7)
		}
		for i := range ycc.Cb {
			ycc.Cb[i], ycc.Cr[i] = uint8(i*31), uint8(255-i*13)
		}
		sources[fmt.Sprintf("ycbcr-%v", ratio)] = ycc.SubImage(image.Rect(1, 1, 12, 8))
	}
	for name, src := range sources {
		for orientation := 2; orientation <= 8; orientation++ {
			fast := transformFast(src, orientation)
			if fast == nil {
				t.Fatalf("%s 没走快速路径", name)
			}
			slow := transformSlow(src, orientation)
			if fast.Bounds() != slow.Bounds() {
				t.Fatalf("%s 方向 %d 尺寸不同：%v %v", name, orientation, fast.Bounds(), slow.Bounds())
			}
			for y := range fast.Bounds().Dy() {
				for x := range fast.Bounds().Dx() {
					if got, want := fast.At(x, y), slow.At(x, y); got != want {
						t.Fatalf("%s 方向 %d 像素 %d,%d：%v != %v", name, orientation, x, y, got, want)
					}
				}
			}
		}
		if got := orient(src, 1); got != src {
			t.Fatalf("%s 方向 1 不该复制", name)
		}
	}
}

// 两段缩放要和原来的一次性 CatmullRom 缩放看起来一样。
func TestScaleToMatchesDirectCatmullRom(t *testing.T) {
	src := image.NewYCbCr(image.Rect(0, 0, 3000, 2000), image.YCbCrSubsampleRatio420)
	for y := range 2000 {
		for x := range 3000 {
			src.Y[y*src.YStride+x] = uint8(128 + 100*math.Sin(float64(x)/90)*math.Cos(float64(y)/70))
		}
	}
	for y := range 1000 {
		for x := range 1500 {
			src.Cb[y*src.CStride+x], src.Cr[y*src.CStride+x] = uint8(x/6), uint8(y/4)
		}
	}
	w, h := fitWithin(3000, 2000, thumbnailSize)
	got := scaleTo(src, w, h)
	want := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(want, want.Bounds(), src, src.Bounds(), draw.Src, nil)
	if got.Bounds() != want.Bounds() {
		t.Fatalf("尺寸 %v != %v", got.Bounds(), want.Bounds())
	}
	var sum, worst float64
	for y := range h {
		for x := range w {
			a, b := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA), want.NRGBAAt(x, y)
			for _, d := range []float64{float64(a.R) - float64(b.R), float64(a.G) - float64(b.G), float64(a.B) - float64(b.B)} {
				sum += d * d
				worst = max(worst, math.Abs(d))
			}
		}
	}
	psnr := 10 * math.Log10(255*255/(sum/float64(w*h*3)))
	if psnr < 40 || worst > 24 {
		t.Fatalf("两段缩放和原来差得太多：PSNR %.1f dB，最大差 %.0f", psnr, worst)
	}
}

func TestShrinkAveragesBlocksAndKeepsTransparentEdgesClean(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 5, 2))
	// 第一块一半红色不透明、一半全透明的黑：平均后颜色仍是纯红，只是半透明。
	src.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 255})
	src.SetNRGBA(1, 0, color.NRGBA{255, 0, 0, 255})
	src.SetNRGBA(4, 0, color.NRGBA{0, 0, 200, 255})
	got := shrink(src, 2).(*image.NRGBA)
	if got.Bounds().Dx() != 3 || got.Bounds().Dy() != 1 {
		t.Fatalf("尺寸 %v", got.Bounds())
	}
	if c := got.NRGBAAt(0, 0); c != (color.NRGBA{255, 0, 0, 128}) {
		t.Fatalf("透明边缘被染色：%v", c)
	}
	// 最右边不足一块，只按实际的两个像素平均。
	if c := got.NRGBAAt(2, 0); c != (color.NRGBA{0, 0, 200, 128}) {
		t.Fatalf("边上不足一块的平均不对：%v", c)
	}
}

// 带方向的 JPEG：缩略图尺寸、方向和原来的全尺寸转正再缩一致，记录的是转正后的尺寸。
func TestPrepareThumbnailOrientsAfterScaling(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1200, 900))
	for y := range 900 {
		for x := range 1200 {
			src.SetRGBA(x, y, color.RGBA{uint8(x / 5), uint8(y / 4), 90, 255})
		}
	}
	// 左上角涂一块白，转正后应在右上角。
	draw.Draw(src, image.Rect(0, 0, 200, 200), image.White, image.Point{}, draw.Src)
	path := writeOrientedJPEG(t, src)
	prepared, err := prepareThumbnail(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.width != 900 || prepared.height != 1200 || prepared.orientation != 6 {
		t.Fatalf("尺寸或方向不对：%dx%d %d", prepared.width, prepared.height, prepared.orientation)
	}
	full, err := Decode(path)
	if err != nil {
		t.Fatal(err)
	}
	old := resized(full, thumbnailSize)
	if prepared.thumbnail.Bounds() != old.Bounds() {
		t.Fatalf("缩略图尺寸 %v，原来是 %v", prepared.thumbnail.Bounds(), old.Bounds())
	}
	b := prepared.thumbnail.Bounds()
	if r, g, bl, _ := prepared.thumbnail.At(b.Max.X-10, b.Min.Y+10).RGBA(); r>>8 < 240 || g>>8 < 240 || bl>>8 < 240 {
		t.Fatal("缩略图没有转正")
	}
}

func writeOrientedJPEG(t *testing.T, img image.Image) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "oriented.jpg")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	payload := append([]byte("Exif\x00\x00"), imageTestTIFF()...)
	if _, err = f.Write(append([]byte{0xff, 0xd8, 0xff, 0xe1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}, payload...)); err != nil {
		t.Fatal(err)
	}
	var encoded jpegBuffer
	if err = jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write(encoded[2:]); err != nil {
		t.Fatal(err)
	}
	return path
}

type jpegBuffer []byte

func (b *jpegBuffer) Write(p []byte) (int, error) { *b = append(*b, p...); return len(p), nil }

func TestThumbnailFormatFollowsTransparency(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	clear := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	for y := range 30 {
		for x := range 40 {
			clear.SetNRGBA(x, y, color.NRGBA{200, 50, 50, uint8(x * 6)})
		}
	}
	f, err := os.Create(filepath.Join(input, "clear.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, clear); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	fixtureCopy(t, "png", filepath.Join(input, "solid.png"))
	if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil); err != nil {
		t.Fatal(err)
	}
	for _, a := range snapshot(t, ctx, lib).Assets {
		want := map[string]string{"clear": ".png", "solid": ".jpg"}[a.Name]
		path := thumbnailFile(t, lib.Root, a.ID)
		if filepath.Ext(path) != want {
			t.Fatalf("%s 的缩略图是 %s，应为 %s", a.Name, path, want)
		}
		if _, err := Decode(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestThumbnailPathAcceptsBothFormatsAndPrefersJPEG(t *testing.T) {
	root := t.TempDir()
	id := fmt.Sprintf("%064x", 1)
	if _, err := ThumbnailPath(root, id); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("没有缩略图时应返回不存在：%v", err)
	}
	if _, err := ThumbnailPath(root, "../escape"); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("非法 ID 应报错：%v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "thumbnails"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, ext := range []string{".png", ".jpg"} {
		if err := os.WriteFile(filepath.Join(root, "thumbnails", id+ext), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		path, err := ThumbnailPath(root, id)
		if err != nil || path != filepath.Join(root, "thumbnails", id+ext) {
			t.Fatalf("放了 %s 后读到 %s，%v", ext, path, err)
		}
	}
}

// 旧图库只有 PNG 缩略图：刷新后换成 JPEG 且不留旧文件，进度最后一次一定是 total/total。
func TestRefreshReplacesLegacyPNGAndReportsProgress(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	for _, ext := range []string{"jpg", "png", "gif", "webp", "tiff", "bmp", "heic", "avif"} {
		fixtureCopy(t, ext, filepath.Join(input, "sample."+ext))
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil); err != nil {
		t.Fatal(err)
	}
	items := snapshot(t, ctx, lib).Assets
	for _, a := range items {
		makeLegacyThumbnail(t, lib.Root, a.ID)
	}
	var calls [][2]int
	if err := s.RefreshThumbnailsProgress(ctx, items, func(done, total int) { calls = append(calls, [2]int{done, total}) }); err != nil {
		t.Fatal(err)
	}
	if len(calls) == 0 || calls[len(calls)-1] != [2]int{len(items), len(items)} {
		t.Fatalf("最后一次进度不对：%v", calls)
	}
	for i := 1; i < len(calls); i++ {
		if calls[i][0] < calls[i-1][0] {
			t.Fatalf("进度倒退：%v", calls)
		}
	}
	for _, a := range items {
		if filepath.Ext(thumbnailFile(t, lib.Root, a.ID)) != ".jpg" {
			t.Fatalf("%s 没有换成 JPEG", a.Name)
		}
		if _, err := os.Stat(filepath.Join(lib.Root, "thumbnails", a.ID+".png")); !os.IsNotExist(err) {
			t.Fatalf("%s 的旧 PNG 缩略图还在：%v", a.Name, err)
		}
	}
	var none []int
	if err := s.RefreshThumbnailsProgress(ctx, nil, func(done, total int) { none = append(none, done, total) }); err != nil || len(none) != 2 || none[0] != 0 || none[1] != 0 {
		t.Fatalf("空列表也要报一次 0/0：%v %v", none, err)
	}
}

// makeLegacyThumbnail 把素材的缩略图换成旧版本的 PNG。
func makeLegacyThumbnail(t *testing.T, root, id string) {
	t.Helper()
	current := thumbnailFile(t, root, id)
	img, err := Decode(current)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(current); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(root, "thumbnails", id+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

// 旧图库只有 PNG 缩略图时，永久删除、导出图库和崩溃后挪回都照常。
func TestLegacyPNGThumbnailsWorkEverywhere(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	for _, ext := range []string{"png", "gif", "jpg"} {
		importFixture(t, ctx, s, ext)
	}
	items := snapshot(t, ctx, lib).Assets
	for _, a := range items {
		makeLegacyThumbnail(t, lib.Root, a.ID)
	}
	// 导出图库带上 PNG 缩略图。
	exported := filepath.Join(t.TempDir(), "Exported.momento")
	if err := lib.ExportLibrary(ctx, exported); err != nil {
		t.Fatal(err)
	}
	for _, a := range items {
		if path := thumbnailFile(t, exported, a.ID); filepath.Ext(path) != ".png" {
			t.Fatalf("导出的缩略图不对：%s", path)
		}
	}
	// 永久删除把 PNG 缩略图一起删掉。
	if err := s.Trash(ctx, []string{items[0].ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePermanently(ctx, []string{items[0].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := ThumbnailPath(lib.Root, items[0].ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("永久删除后缩略图还在：%v", err)
	}
	// 模拟永久删除挪走文件后崩溃：上一版的「ID.thumbnail」和这一版的「ID.thumbnail.png」都要挪回。
	deletion := filepath.Join(lib.Root, ".delete-legacy")
	if err := os.Mkdir(deletion, 0700); err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{".thumbnail", ".thumbnail.png"} {
		a := items[i+1]
		if err := os.Rename(filepath.Join(lib.Root, "thumbnails", a.ID+".png"), filepath.Join(deletion, a.ID+name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for _, a := range items[1:] {
		if path := thumbnailFile(t, lib.Root, a.ID); filepath.Ext(path) != ".png" {
			t.Fatalf("没有挪回 PNG 缩略图：%s", path)
		}
	}
	if _, err := os.Stat(deletion); !os.IsNotExist(err) {
		t.Fatalf("暂存目录没清掉：%v", err)
	}
}

// 新版本永久删除中途崩溃，暂存里的 JPEG 缩略图也要挪回。
func TestInterruptedDeletionRestoresJPEGThumbnail(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "jpg")
	a := snapshot(t, ctx, lib).Assets[0]
	thumb := thumbnailFile(t, lib.Root, a.ID)
	deletion := filepath.Join(lib.Root, ".delete-jpeg")
	if err := os.Mkdir(deletion, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(thumb, filepath.Join(deletion, a.ID+".thumbnail.jpg")); err != nil {
		t.Fatal(err)
	}
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("JPEG 缩略图没有挪回：%v", err)
	}
}

func TestMemoryBudgetLimitsConcurrencyAndRunsOversizedAlone(t *testing.T) {
	ctx := context.Background()
	budget := newMemoryBudget(100)
	var running, peak, used atomic.Int64
	var wg sync.WaitGroup
	sizes := []int64{40, 40, 40, 250, 10, 10, 60, 30, 30}
	for _, size := range sizes {
		wg.Go(func() {
			granted, err := budget.acquire(ctx, size)
			if err != nil {
				t.Error(err)
				return
			}
			now := used.Add(granted)
			n := running.Add(1)
			if now > 100 {
				t.Errorf("超出预算：%d", now)
			}
			if size > 100 && n != 1 {
				t.Errorf("超大任务没有独自运行：同时 %d 个", n)
			}
			for {
				old := peak.Load()
				if n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			running.Add(-1)
			used.Add(-granted)
			budget.release(granted)
		})
	}
	wg.Wait()
	if peak.Load() < 2 {
		t.Fatalf("小任务应该能并行：峰值 %d", peak.Load())
	}
	if budget.used != 0 || len(budget.waiting) != 0 {
		t.Fatalf("额度没还清：%d %d", budget.used, len(budget.waiting))
	}
	// 排队时取消要立刻返回，并且不占额度。
	granted, err := budget.acquire(ctx, 100)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	go func() { time.Sleep(10 * time.Millisecond); cancel() }()
	if _, err = budget.acquire(cancelled, 10); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消没有生效：%v", err)
	}
	budget.release(granted)
	if budget.used != 0 || len(budget.waiting) != 0 {
		t.Fatalf("取消后额度没还清：%d %d", budget.used, len(budget.waiting))
	}
}

// 坏图和它的重复副本各占一条失败，计数加起来正好等于传入的文件数。
func TestImportFailedListsEveryBrokenSourceFile(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	broken := filepath.Join(input, "broken.jpg")
	copied := filepath.Join(input, "broken copy.jpg")
	for _, path := range []string{broken, copied} {
		if err := os.WriteFile(path, []byte("not an image"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	fixtureCopy(t, "png", filepath.Join(input, "good.png"))
	fixtureCopy(t, "gif", filepath.Join(input, "good.gif"))
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 2 || result.Skipped != 0 || len(result.Errors) != 2 || len(result.Failed) != 2 {
		t.Fatalf("计数不对：%+v", result)
	}
	if result.Imported+result.Skipped+len(result.Failed) != 4 {
		t.Fatalf("计数和文件数对不上：%+v", result)
	}
	for i, want := range []string{copied, broken} {
		if result.Failed[i] != want {
			t.Fatalf("第 %d 条失败是 %s，应为 %s", i, result.Failed[i], want)
		}
		if !strings.Contains(result.Errors[i].Error(), filepath.Base(want)) {
			t.Fatalf("第 %d 条错误和文件对不上：%v", i, result.Errors[i])
		}
	}
	if len(snapshot(t, ctx, lib).Assets) != 2 {
		t.Fatal("好图没有导入")
	}
}
