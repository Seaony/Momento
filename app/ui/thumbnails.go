package ui

import (
	"container/list"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

const (
	thumbnailItemLimit         = 512
	thumbnailByteLimit   int64 = 96 << 20
	thumbnailConcurrency       = 6
)

type thumbnailKey struct{ root, id string }

type thumbnailEntry struct {
	key    thumbnailKey
	bitmap *ui.Bitmap
	bytes  int64
	err    error
}

type thumbnailCache struct {
	mu         sync.Mutex
	update     func(func())
	entries    map[thumbnailKey]*list.Element
	order      list.List
	pending    map[thumbnailKey]uint64
	nextTicket uint64
	slots      chan struct{}
	bytes      int64
	decode     func(string) (*ui.Bitmap, int64, error)
}

func newThumbnailCache(update func(func())) *thumbnailCache {
	return &thumbnailCache{
		update:  update,
		entries: make(map[thumbnailKey]*list.Element),
		pending: make(map[thumbnailKey]uint64),
		slots:   make(chan struct{}, thumbnailConcurrency),
		decode:  decodeThumbnail,
	}
}

func (c *thumbnailCache) Get(root string, asset model.Asset) *ui.Bitmap {
	key := thumbnailKey{root: filepath.Clean(root), id: asset.ID}
	c.mu.Lock()
	if element, ok := c.entries[key]; ok {
		c.order.MoveToFront(element)
		bitmap := element.Value.(thumbnailEntry).bitmap
		c.mu.Unlock()
		return bitmap
	}
	if _, waiting := c.pending[key]; waiting {
		c.mu.Unlock()
		return nil
	}
	select {
	case c.slots <- struct{}{}:
	default:
		c.mu.Unlock()
		return nil
	}
	c.nextTicket++
	ticket := c.nextTicket
	c.pending[key] = ticket
	c.mu.Unlock()

	go func() {
		bitmap, size, err := c.decode(filepath.Join(key.root, "thumbnails", key.id+".png"))
		<-c.slots
		c.update(func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			if c.pending[key] != ticket {
				return
			}
			delete(c.pending, key)
			if err != nil {
				slog.Error("读取缩略图失败", "library", key.root, "asset", key.id, "error", err)
				bitmap, size = nil, 0
			}
			entry := thumbnailEntry{key: key, bitmap: bitmap, bytes: size, err: err}
			c.entries[key] = c.order.PushFront(entry)
			c.bytes += size
			for len(c.entries) > thumbnailItemLimit || c.bytes > thumbnailByteLimit {
				c.remove(c.order.Back())
			}
		})
	}()
	return nil
}

func (c *thumbnailCache) Error(root string, asset model.Asset) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element := c.entries[thumbnailKey{root: filepath.Clean(root), id: asset.ID}]; element != nil {
		return element.Value.(thumbnailEntry).err
	}
	return nil
}

func (c *thumbnailCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[thumbnailKey]*list.Element)
	c.pending = make(map[thumbnailKey]uint64)
	c.order.Init()
	c.bytes = 0
}

// 刷新单张图片传 ID，刷新整个图库传根目录。
func (c *thumbnailCache) Invalidate(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	root := filepath.Clean(value)
	for key, element := range c.entries {
		if key.id == value || key.root == root {
			c.remove(element)
		}
	}
	for key := range c.pending {
		if key.id == value || key.root == root {
			delete(c.pending, key)
		}
	}
}

func (c *thumbnailCache) remove(element *list.Element) {
	entry := element.Value.(thumbnailEntry)
	c.bytes -= entry.bytes
	delete(c.entries, entry.key)
	c.order.Remove(element)
}

func decodeThumbnail(path string) (*ui.Bitmap, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	config, err := png.DecodeConfig(f)
	if err != nil {
		return nil, 0, err
	}
	if config.Width > 512 || config.Height > 512 {
		return nil, 0, i18n.NewError("error_thumbnail_dimensions", config.Width, config.Height)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, 0, err
	}
	img, err := png.Decode(f)
	if err != nil {
		return nil, 0, err
	}
	bitmap := ui.NewBitmap(img)
	w, h := bitmap.Size()
	pixels := int64(w) * int64(h) * 4
	// 缩小绘制时框架会创建 mipmap，给这些副本预留三分之一。
	return bitmap, pixels + (pixels+2)/3, nil
}
