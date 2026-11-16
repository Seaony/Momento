package assets

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

// benchJPEG 生成一张带 EXIF 方向 6 的合成 JPEG，seed 不同内容就不同，避免导入时被去重。
func benchJPEG(b *testing.B, dir string, width, height, seed int) string {
	b.Helper()
	img := image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio420)
	for y := range height {
		for x := range width {
			// 渐变加细纹理，压缩后的体积接近真实照片。
			v := uint8((x*255/width + y*255/height + (x*y+seed)%23) / 2)
			img.Y[y*img.YStride+x] = v
		}
	}
	for y := range (height + 1) / 2 {
		for x := range (width + 1) / 2 {
			img.Cb[y*img.CStride+x] = uint8(64 + (x+seed)%128)
			img.Cr[y*img.CStride+x] = uint8(64 + y%128)
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}); err != nil {
		b.Fatal(err)
	}
	payload := append([]byte("Exif\x00\x00"), imageTestTIFF()...)
	data := []byte{0xff, 0xd8, 0xff, 0xe1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}
	data = append(data, payload...)
	data = append(data, encoded.Bytes()[2:]...)
	path := filepath.Join(dir, fmt.Sprintf("bench-%dx%d-%d.jpg", width, height, seed))
	if err := os.WriteFile(path, data, 0600); err != nil {
		b.Fatal(err)
	}
	return path
}

// peakHeap 在后台每毫秒采一次堆上存活对象的字节数，返回期间的最大值。
func peakHeap() func() uint64 {
	runtime.GC()
	sample := []metrics.Sample{{Name: "/memory/classes/heap/objects:bytes"}}
	var peak atomic.Uint64
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			metrics.Read(sample)
			if v := sample[0].Value.Uint64(); v > peak.Load() {
				peak.Store(v)
			}
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		}
	})
	return func() uint64 { close(stop); wg.Wait(); return peak.Load() }
}

func benchImport(b *testing.B, paths []string) {
	ctx := context.Background()
	var peak uint64
	for i := range b.N {
		b.StopTimer()
		lib, err := library.Create(ctx, filepath.Join(b.TempDir(), fmt.Sprintf("Bench%d.momento", i)), "Bench")
		if err != nil {
			b.Fatal(err)
		}
		s := New(lib.Root, lib.DB)
		done := peakHeap()
		b.StartTimer()
		result, err := s.Import(ctx, model.ImportRequest{Paths: paths}, nil)
		b.StopTimer()
		peak = max(peak, done())
		if err != nil || result.Imported != len(paths) {
			b.Fatalf("导入结果 %+v，错误 %v", result, err)
		}
		if err = lib.Close(); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(peak)/(1<<20), "peak-MB")
}

// 运行：go test ./internal/assets -run '^$' -bench ImportJPEG -benchmem -benchtime 3x
func BenchmarkImportJPEG(b *testing.B) {
	for _, size := range []struct {
		name          string
		width, height int
		count         int
	}{{"12MP", 4000, 3000, 1}, {"50MP", 8660, 5774, 1}, {"6x50MP", 8660, 5774, 6}} {
		b.Run(size.name, func(b *testing.B) {
			dir := b.TempDir()
			paths := make([]string, size.count)
			for i := range paths {
				paths[i] = benchJPEG(b, dir, size.width, size.height, i)
			}
			benchImport(b, paths)
		})
	}
}

// 运行：go test ./internal/assets -run '^$' -bench PrepareImage -benchmem
// 预览大图走的全尺寸解码加转正（方向 6）。
func BenchmarkPrepareImage(b *testing.B) {
	for _, size := range []struct {
		name          string
		width, height int
	}{{"12MP", 4000, 3000}, {"50MP", 8660, 5774}} {
		b.Run(size.name, func(b *testing.B) {
			path := benchJPEG(b, b.TempDir(), size.width, size.height, 0)
			b.ResetTimer()
			for range b.N {
				if _, _, _, _, err := PrepareImage(path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
