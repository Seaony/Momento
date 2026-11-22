package assets

import (
	"context"
	"errors"
	"image"
	"image/color"
	"log/slog"
	"runtime"
	"sync"
)

// decodeBudget 是预览、导出、导入和重建缩略图共用的解码内存预算。
var decodeBudget = newMemoryBudget(1 << 30)

// residentAnimationBudgetLimit 给常驻播放的动图留一半共享预算，保证缩略图和导入总有额度可用。
const residentAnimationBudgetLimit = 1 << 29

// ErrResidentBudgetBusy 表示常驻动图要的额度现在拿不到；调用方应改走静态首帧，不要排队等待。
var ErrResidentBudgetBusy = errors.New("resident animation budget is busy")

// decodeEstimate 预估解码一张图要的内存：每像素按 RGBA 4 字节（16 位通道按 8 字节），
// 再乘 1.5 留给解码器缓冲和缩小用的中间图。
func decodeEstimate(config image.Config) int64 {
	bytesPerPixel := int64(4)
	switch config.ColorModel {
	case color.RGBA64Model, color.NRGBA64Model:
		bytesPerPixel = 8
	}
	return int64(config.Width) * int64(config.Height) * bytesPerPixel * 3 / 2
}

// decodeWorkers 是解码类任务的并发数，实际同时解码几张还要看内存预算。
func decodeWorkers() int { return max(runtime.NumCPU(), 2) }

// memoryBudget 是按字节计数的先来先得排队器。单张超过总额时按总额算，等前面都做完后独自运行。
type memoryBudget struct {
	mu           sync.Mutex
	limit        int64
	residentUsed int64
	used         int64
	waiting      []*budgetWaiter
}

type budgetWaiter struct {
	bytes int64
	ready chan struct{}
}

func newMemoryBudget(limit int64) *memoryBudget { return &memoryBudget{limit: limit} }

// acquire 拿到额度或 ctx 取消才返回；后来的不会插到排队中的大图前面。
func (b *memoryBudget) acquire(ctx context.Context, bytes int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	bytes = min(max(bytes, 1), b.limit)
	b.mu.Lock()
	if len(b.waiting) == 0 && b.used+bytes <= b.limit {
		b.used += bytes
		b.mu.Unlock()
		return bytes, nil
	}
	w := &budgetWaiter{bytes: bytes, ready: make(chan struct{})}
	b.waiting = append(b.waiting, w)
	b.mu.Unlock()
	select {
	case <-w.ready:
		if err := ctx.Err(); err != nil {
			b.release(bytes)
			return 0, err
		}
		return bytes, nil
	case <-ctx.Done():
		b.mu.Lock()
		select {
		case <-w.ready:
			// 取消的同时刚好轮到，把额度还回去。
			b.used -= bytes
		default:
			for i, other := range b.waiting {
				if other == w {
					b.waiting = append(b.waiting[:i], b.waiting[i+1:]...)
					break
				}
			}
		}
		b.wake()
		b.mu.Unlock()
		return 0, ctx.Err()
	}
}

func (b *memoryBudget) release(bytes int64) {
	b.mu.Lock()
	b.used -= bytes
	b.wake()
	defer b.mu.Unlock()
}

// tryReserveResident 立即拿额度；共享预算已有排队者时也不插队，拿不到就让常驻播放降级。
func (b *memoryBudget) tryReserveResident(bytes int64) (func(), bool) {
	bytes = min(max(bytes, 1), b.limit)
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.waiting) > 0 || b.used+bytes > b.limit || b.residentUsed+bytes > residentAnimationBudgetLimit {
		return nil, false
	}
	b.used += bytes
	b.residentUsed += bytes
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			b.used, b.residentUsed = b.used-bytes, b.residentUsed-bytes
			b.wake()
			b.mu.Unlock()
		})
	}, true
}

// wake 按排队顺序放行装得下的，队首装不下就停，保证大图不会一直被小图插队。调用方必须已持锁。
func (b *memoryBudget) wake() {
	for len(b.waiting) > 0 {
		w := b.waiting[0]
		if b.used+w.bytes > b.limit {
			return
		}
		b.used += w.bytes
		b.waiting = b.waiting[1:]
		close(w.ready)
	}
}

// admit 给 decodeImage 用：按像素数预估内存后排队，返回只生效一次的归还函数。
// 读不出尺寸时不知道多大，按总额算，独自运行最稳妥。
func (b *memoryBudget) admit(ctx context.Context, config image.Config, known bool) (func(), error) {
	bytes := b.limit
	if known {
		bytes = decodeEstimate(config)
	}
	release, err := b.reserve(ctx, bytes)
	if err == nil && bytes > b.limit {
		slog.Info("图片较大，单独解码", "width", config.Width, "height", config.Height)
	}
	return release, err
}

func (b *memoryBudget) reserve(ctx context.Context, bytes int64) (func(), error) {
	granted, err := b.acquire(ctx, bytes)
	if err != nil {
		return nil, err
	}
	var once sync.Once
	return func() { once.Do(func() { b.release(granted) }) }, nil
}
