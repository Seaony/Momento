package media

import (
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// 样本由 testdata/poster_generate.sh 生成：320×180，左红右蓝，1 秒后左半变绿。
func posterSample(name string) string { return filepath.Join("testdata", name) }

// skipIfUnavailable 在这台机器没能力截帧时跳过（比如 Linux 没装 ffmpeg）。
func skipIfUnavailable(t testing.TB, err error) {
	t.Helper()
	if errors.Is(err, ErrPosterUnavailable) {
		t.Skipf("poster unavailable on this system: %v", err)
	}
}

// colorName 粗分像素主色，视频有压缩误差，不比精确值。
func colorName(img image.Image, x, y int) string {
	r, g, b, _ := img.At(x, y).RGBA()
	r, g, b = r>>8, g>>8, b>>8
	switch {
	case r > 150 && g < 100 && b < 100:
		return "red"
	case g > 90 && r < 100 && b < 100:
		return "green"
	case b > 150 && r < 100 && g < 100:
		return "blue"
	}
	return "other"
}

func mustPoster(t testing.TB, name string, at time.Duration, maxSize int) image.Image {
	t.Helper()
	img, err := Poster(context.Background(), posterSample(name), at, maxSize)
	skipIfUnavailable(t, err)
	if err != nil {
		t.Fatalf("Poster(%s): %v", name, err)
	}
	return img
}

func checkColor(t *testing.T, img image.Image, x, y int, want string) {
	t.Helper()
	if got := colorName(img, x, y); got != want {
		t.Errorf("pixel (%d,%d) = %s %v, want %s", x, y, got, img.At(x, y), want)
	}
}

func TestPosterH264(t *testing.T) {
	img := mustPoster(t, "poster_h264.mp4", 0, 400)
	b := img.Bounds()
	// 原图 320×180 比 maxSize 小，不放大
	if b.Dx() != 320 || b.Dy() != 180 {
		t.Fatalf("size = %v, want 320x180", b.Size())
	}
	checkColor(t, img, 40, 90, "red")
	checkColor(t, img, 280, 90, "blue")
}

func TestPosterAt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 系统缩略图不能指定时刻")
	}
	img := mustPoster(t, "poster_h264.mp4", 1500*time.Millisecond, 400)
	checkColor(t, img, 40, 90, "green")
	checkColor(t, img, 280, 90, "blue")
}

func TestPosterDownscale(t *testing.T) {
	img := mustPoster(t, "poster_h264.mp4", 0, 100)
	b := img.Bounds()
	if b.Dx() != 100 || b.Dy() < 55 || b.Dy() > 57 {
		t.Fatalf("size = %v, want 100x56 (±1)", b.Size())
	}
	checkColor(t, img, 10, 28, "red")
	checkColor(t, img, 90, 28, "blue")
}

func TestPosterRotate90(t *testing.T) {
	img := mustPoster(t, "poster_rotate90.mov", 0, 400)
	b := img.Bounds()
	// 编码画面 320×180，带顺时针 90° 旋转，转正后应该是竖的 180×320，原来左边的红色到了上面
	if b.Dx() != 180 || b.Dy() != 320 {
		t.Fatalf("size = %v, want 180x320", b.Size())
	}
	checkColor(t, img, 90, 40, "red")
	checkColor(t, img, 90, 280, "blue")

	small := mustPoster(t, "poster_rotate90.mov", 0, 160)
	if s := small.Bounds().Size(); s.Y != 160 || s.X < 89 || s.X > 91 {
		t.Fatalf("rotated downscale size = %v, want 90x160 (±1)", s)
	}
}

func TestPosterShortVideo(t *testing.T) {
	// 视频只有 0.5 秒，at=1s 超出时长，应取第一帧（左红），而不是最后一帧（左绿）
	img := mustPoster(t, "poster_short.mp4", time.Second, 400)
	if runtime.GOOS != "windows" {
		checkColor(t, img, 40, 90, "red")
	}
	checkColor(t, img, 280, 90, "blue")
}

func TestPosterWebM(t *testing.T) {
	// at=0 直接截帧，at=1s 会先读时长，两条路径的报错都要能识别成“系统不支持”
	for _, at := range []time.Duration{0, time.Second} {
		testPosterWebM(t, at)
	}
}

func testPosterWebM(t *testing.T, at time.Duration) {
	img, err := Poster(context.Background(), posterSample("poster_vp9.webm"), at, 400)
	if err != nil {
		// macOS 的 AVFoundation 不认 WebM 容器；只要能被识别成“这台机器不支持”，导入就能回退到通用封面
		if !errors.Is(err, ErrPosterUnavailable) {
			t.Fatalf("WebM error should be ErrPosterUnavailable, got %v", err)
		}
		t.Logf("WebM unsupported here (falls back to generic poster): %v", err)
		return
	}
	if s := img.Bounds().Size(); s.X != 320 || s.Y != 180 {
		t.Fatalf("size = %v, want 320x180", s)
	}
	checkColor(t, img, 280, 90, "blue")
}

func TestPosterGarbage(t *testing.T) {
	for _, at := range []time.Duration{0, time.Second} {
		testPosterGarbage(t, at)
	}
}

func testPosterGarbage(t *testing.T, at time.Duration) {
	start := time.Now()
	_, err := Poster(context.Background(), posterSample("poster_garbage.mp4"), at, 400)
	if err == nil {
		t.Fatal("garbage file should fail")
	}
	t.Logf("garbage error after %v: %v", time.Since(start), err)
	if runtime.GOOS == "windows" {
		// 系统缩略图分不清坏文件和不支持，统一按取不到处理
		return
	}
	// 这台机器本来就截不了帧（如 Linux 没装 ffmpeg）时，坏文件也只能报取不到
	if _, okErr := Poster(context.Background(), posterSample("poster_h264.mp4"), 0, 64); errors.Is(okErr, ErrPosterUnavailable) {
		t.Skipf("poster unavailable on this system: %v", okErr)
	}
	if errors.Is(err, ErrPosterUnavailable) {
		t.Fatalf("broken file must not look like ErrPosterUnavailable: %v", err)
	}
}

func TestPosterMissingFile(t *testing.T) {
	_, err := Poster(context.Background(), posterSample("poster_missing.mp4"), 0, 400)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want os.ErrNotExist", err)
	}
}

func TestPosterBadMaxSize(t *testing.T) {
	if _, err := Poster(context.Background(), posterSample("poster_h264.mp4"), 0, 0); err == nil {
		t.Fatal("maxSize 0 should fail")
	}
}

func TestPosterCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	_, err := Poster(ctx, posterSample("poster_h264.mp4"), 0, 400)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Fatalf("canceled call took %v", d)
	}

	// 截帧进行中超时：要么已经截完，要么尽快返回超时错误
	for range 20 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		start := time.Now()
		img, err := Poster(ctx, posterSample("poster_h264.mp4"), 1500*time.Millisecond, 400)
		cancel()
		skipIfUnavailable(t, err)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want nil or DeadlineExceeded", err)
		}
		if err == nil && img == nil {
			t.Fatal("nil image without error")
		}
		if d := time.Since(start); d > time.Second {
			t.Fatalf("timed-out call took %v", d)
		}
	}
}

// TestPosterConcurrent 让 8 个 goroutine 同时截帧，看会不会崩、内存会不会一直涨。
func TestPosterConcurrent(t *testing.T) {
	if _, err := Poster(context.Background(), posterSample("poster_h264.mp4"), 0, 64); err != nil {
		skipIfUnavailable(t, err)
		t.Fatal(err)
	}
	names := []string{"poster_h264.mp4", "poster_rotate90.mov", "poster_short.mp4", "poster_garbage.mp4"}
	round := func() {
		var wg sync.WaitGroup
		errs := make(chan error, 8)
		for g := range 8 {
			wg.Go(func() {
				for i := range 12 {
					name := names[(g+i)%len(names)]
					img, err := Poster(context.Background(), posterSample(name), 500*time.Millisecond, 128)
					if name == "poster_garbage.mp4" {
						if err == nil {
							errs <- errors.New("garbage file should fail")
							return
						}
						continue
					}
					if err != nil {
						errs <- err
						return
					}
					if s := img.Bounds().Size(); s.X > 128 || s.Y > 128 {
						errs <- errors.New("poster exceeds maxSize")
						return
					}
				}
			})
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatal(err)
		}
	}

	rounds := 6
	if testing.Short() {
		rounds = 2
	}
	round()
	base := maxRSS()
	for range rounds - 1 {
		round()
	}
	last := maxRSS()
	t.Logf("peak RSS after first round %.1f MB, after %d rounds %.1f MB (%d calls)",
		float64(base)/1e6, rounds, float64(last)/1e6, rounds*8*12)
	// 每轮 96 次；真有泄漏（每次漏一帧位图或一个生成器）几轮下来会明显上涨
	if base > 0 && last-base > 64<<20 {
		t.Fatalf("peak RSS grew %.1f MB over %d rounds", float64(last-base)/1e6, rounds-1)
	}
}

func BenchmarkPosterH264(b *testing.B) {
	benchPoster(b, "poster_h264.mp4", 0)
}

func BenchmarkPosterH264At(b *testing.B) {
	benchPoster(b, "poster_h264.mp4", 1500*time.Millisecond)
}

func BenchmarkPosterRotate90(b *testing.B) {
	benchPoster(b, "poster_rotate90.mov", 0)
}

func benchPoster(b *testing.B, name string, at time.Duration) {
	for b.Loop() {
		if _, err := Poster(context.Background(), posterSample(name), at, 256); err != nil {
			skipIfUnavailable(b, err)
			b.Fatal(err)
		}
	}
}
