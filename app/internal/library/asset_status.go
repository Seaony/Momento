package library

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// AssetStatus 是自动导入移源前的轻量核对结果；map 里有这项才表示素材行还在。
type AssetStatus struct {
	Trashed        bool
	OriginalExists bool
	MotionExists   bool
}

// AssetStatuses 按 ID 查回收站状态和包内文件是否存在。素材 ID 就是内容哈希，不需要再读原图或视频算哈希。
func (l *Library) AssetStatuses(ctx context.Context, ids []string) (map[string]AssetStatus, error) {
	statuses := make(map[string]AssetStatus, len(ids))
	for _, id := range ids {
		var relative, motion string
		var trashed bool
		err := l.DB.QueryRowContext(ctx, "SELECT trashed,relative_path,motion_path FROM assets WHERE id=?", id).
			Scan(&trashed, &relative, &motion)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		status := AssetStatus{Trashed: trashed}
		exists, err := assetFileExists(l.Root, relative)
		if err != nil {
			return nil, err
		}
		status.OriginalExists = exists
		if motion != "" {
			exists, err = assetFileExists(l.Root, motion)
			if err != nil {
				return nil, err
			}
			status.MotionExists = exists
		}
		statuses[id] = status
	}
	return statuses, nil
}

func assetFileExists(root, relative string) (bool, error) {
	if !filepath.IsLocal(filepath.FromSlash(relative)) {
		return false, i18n.NewError("error_asset_relative_path", relative)
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative)))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
