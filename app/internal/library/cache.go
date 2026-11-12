package library

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// 这里只移除可再生缓存；调用方随后用 assets 服务重建缩略图并重新加载。
func (l *Library) ClearCache(ctx context.Context) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	conn, err := l.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	for _, name := range []string{"thumbnails", ".drag-out"} {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(l.Root, name)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if err := os.Mkdir(path, 0755); err != nil {
			return err
		}
	}
	slog.Info("清除图库缓存", "path", l.Root)
	return nil
}
