package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func thumbnailTestCache() (*thumbnailCache, <-chan struct{}) {
	completed := make(chan struct{}, 1024)
	c := newThumbnailCache(func(fn func()) { fn(); completed <- struct{}{} })
	return c, completed
}

func waitThumbnail(t *testing.T, completed <-chan struct{}) {
	t.Helper()
	select {
	case <-completed:
	case <-time.After(5 * time.Second):
		t.Fatal("缩略图任务未结束")
	}
}

func thumbnailTestBitmap() *ui.Bitmap {
	return ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 16, 16)))
}

func writeThumbnailTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 200, G: 100, B: 20, A: 255})
	encodeErr := png.Encode(f, img)
	closeErr := f.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func TestThumbnailLoadsPNGAndAccountsPixels(t *testing.T) {
	root := t.TempDir()
	writeThumbnailTestPNG(t, filepath.Join(root, "thumbnails", "asset.png"), 64, 32)
	c, completed := thumbnailTestCache()
	if got := c.Get(root, model.Asset{ID: "asset"}); got != nil {
		t.Fatal("首次请求不应同步解码")
	}
	waitThumbnail(t, completed)
	bitmap := c.Get(root, model.Asset{ID: "asset"})
	if bitmap == nil {
		t.Fatal("后台解码后未缓存")
	}
	if w, h := bitmap.Size(); w != 64 || h != 32 {
		t.Fatalf("尺寸 %d×%d", w, h)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bytes != 10923 {
		t.Fatalf("缓存像素与mipmap预留 %d，期望10923", c.bytes)
	}
	if len(c.pending) != 0 {
		t.Fatal("已完成任务仍在等待")
	}
}

func TestThumbnailConcurrencyAndRequestsAreDeduplicated(t *testing.T) {
	c, completed := thumbnailTestCache()
	started := make(chan struct{}, thumbnailConcurrency)
	release := make(chan struct{})
	var calls, active, peak atomic.Int32
	c.decode = func(string) (*ui.Bitmap, int64, error) {
		calls.Add(1)
		n := active.Add(1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		started <- struct{}{}
		<-release
		active.Add(-1)
		return thumbnailTestBitmap(), 1024, nil
	}
	for i := 0; i < thumbnailConcurrency; i++ {
		asset := model.Asset{ID: fmt.Sprint(i)}
		c.Get("library", asset)
		c.Get("library", asset)
	}
	for i := 0; i < thumbnailConcurrency; i++ {
		waitThumbnail(t, started)
	}
	c.Get("library", model.Asset{ID: "seventh"})
	if calls.Load() != thumbnailConcurrency {
		t.Fatalf("启动了 %d 路任务", calls.Load())
	}
	close(release)
	for i := 0; i < thumbnailConcurrency; i++ {
		waitThumbnail(t, completed)
	}
	c.Get("library", model.Asset{ID: "seventh"})
	waitThumbnail(t, completed)
	if peak.Load() > thumbnailConcurrency || calls.Load() != 7 {
		t.Fatalf("峰值 %d，总调用 %d", peak.Load(), calls.Load())
	}
}

func TestThumbnailLRUEvictsAt512Items(t *testing.T) {
	c, completed := thumbnailTestCache()
	bitmap := thumbnailTestBitmap()
	c.decode = func(string) (*ui.Bitmap, int64, error) { return bitmap, 1024, nil }
	for i := 0; i < thumbnailItemLimit; i++ {
		c.Get("library", model.Asset{ID: fmt.Sprint(i)})
		waitThumbnail(t, completed)
	}
	c.Get("library", model.Asset{ID: "0"})
	c.Get("library", model.Asset{ID: "new"})
	waitThumbnail(t, completed)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) != 512 {
		t.Fatalf("条目数 %d", len(c.entries))
	}
	if _, ok := c.entries[thumbnailKey{"library", "0"}]; !ok {
		t.Fatal("刚使用的条目被淘汰")
	}
	if _, ok := c.entries[thumbnailKey{"library", "1"}]; ok {
		t.Fatal("最久未使用的条目未淘汰")
	}
}

func TestThumbnailLRUEvictsAt96MB(t *testing.T) {
	c, completed := thumbnailTestCache()
	bitmap := thumbnailTestBitmap()
	c.decode = func(string) (*ui.Bitmap, int64, error) { return bitmap, 32 << 20, nil }
	for _, id := range []string{"a", "b", "c"} {
		c.Get("library", model.Asset{ID: id})
		waitThumbnail(t, completed)
	}
	c.Get("library", model.Asset{ID: "a"})
	c.Get("library", model.Asset{ID: "d"})
	waitThumbnail(t, completed)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bytes != 96<<20 || len(c.entries) != 3 {
		t.Fatalf("缓存 %d 字节，%d 项", c.bytes, len(c.entries))
	}
	if _, ok := c.entries[thumbnailKey{"library", "b"}]; ok {
		t.Fatal("超过预算后未淘汰最旧条目")
	}
}

func TestThumbnailFailureDoesNotRetryUntilInvalidated(t *testing.T) {
	c, completed := thumbnailTestCache()
	var calls atomic.Int32
	c.decode = func(string) (*ui.Bitmap, int64, error) { calls.Add(1); return nil, 0, errors.New("图片损坏") }
	asset := model.Asset{ID: "broken"}
	c.Get("library", asset)
	waitThumbnail(t, completed)
	for i := 0; i < 20; i++ {
		if c.Get("library", asset) != nil {
			t.Fatal("损坏图片得到位图")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("失败后重试 %d 次", calls.Load())
	}
	if err := c.Error("library", asset); err == nil || err.Error() != "图片损坏" {
		t.Fatalf("失败原因不可读取：%v", err)
	}
	if err := c.Error("other", asset); err != nil {
		t.Fatal("其他图库读取到错误记录")
	}
	c.Invalidate(asset.ID)
	if err := c.Error("library", asset); err != nil {
		t.Fatal("刷新后残留失败原因")
	}
	c.Get("library", asset)
	waitThumbnail(t, completed)
	if calls.Load() != 2 {
		t.Fatal("显式刷新没有重新读取")
	}
}

func TestThumbnailClearDiscardsOldLibraryResult(t *testing.T) {
	updates := make(chan func(), 4)
	c := newThumbnailCache(func(fn func()) { updates <- fn })
	c.decode = func(string) (*ui.Bitmap, int64, error) { return thumbnailTestBitmap(), 1024, nil }
	asset := model.Asset{ID: "same-hash"}
	c.Get("first", asset)
	old := waitThumbnailUpdate(t, updates)
	c.Clear()
	c.Get("second", asset)
	current := waitThumbnailUpdate(t, updates)
	old()
	c.mu.Lock()
	if len(c.entries) != 0 || len(c.pending) != 1 {
		t.Fatalf("旧结果污染新库：%d 项，%d 待处理", len(c.entries), len(c.pending))
	}
	c.mu.Unlock()
	current()
	if c.Get("second", asset) == nil {
		t.Fatal("新库缩略图未就绪")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.entries[thumbnailKey{"first", asset.ID}]; ok {
		t.Fatal("缓存残留旧库")
	}
}

func TestThumbnailInvalidateDiscardsOldTicketAndSupportsRoot(t *testing.T) {
	updates := make(chan func(), 4)
	c := newThumbnailCache(func(fn func()) { updates <- fn })
	c.decode = func(string) (*ui.Bitmap, int64, error) { return thumbnailTestBitmap(), 1024, nil }
	asset := model.Asset{ID: "same-hash"}
	c.Get("library", asset)
	old := waitThumbnailUpdate(t, updates)
	c.Invalidate(asset.ID)
	c.Get("library", asset)
	current := waitThumbnailUpdate(t, updates)
	old()
	c.mu.Lock()
	if len(c.entries) != 0 || len(c.pending) != 1 {
		t.Fatalf("旧任务覆盖刷新任务")
	}
	c.mu.Unlock()
	current()
	c.Get("other", asset)
	waitThumbnailUpdate(t, updates)()
	c.Invalidate("library")
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) != 1 {
		t.Fatalf("根目录刷新影响了其他图库，剩余 %d 项", len(c.entries))
	}
	if _, ok := c.entries[thumbnailKey{"other", asset.ID}]; !ok {
		t.Fatal("其他图库被清除")
	}
}

func waitThumbnailUpdate(t *testing.T, updates <-chan func()) func() {
	t.Helper()
	select {
	case fn := <-updates:
		return fn
	case <-time.After(5 * time.Second):
		t.Fatal("没有收到主线程更新")
		return nil
	}
}

func TestThumbnailRejectsCorruptOrOversizedPNG(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.png")
	if err := os.WriteFile(broken, []byte("bad png"), 0600); err != nil {
		t.Fatal(err)
	}
	oversized := filepath.Join(dir, "oversized.png")
	writeThumbnailTestPNG(t, oversized, 513, 32)
	for _, path := range []string{broken, oversized, filepath.Join(dir, "missing.png")} {
		if _, _, err := decodeThumbnail(path); err == nil {
			t.Fatalf("%s 没有报错", filepath.Base(path))
		}
	}
}
