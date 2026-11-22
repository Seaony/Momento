package library

import (
	"context"
	"database/sql"
	"time"
)

// SaveAssetEdit 在一个事务里改名称（rename 为真时）和备注、来源链接（details 为真时），任何一步失败都不改。
func (l *Library) SaveAssetEdit(ctx context.Context, id, name string, rename bool, note, sourceURL string, details bool) error {
	if rename {
		var err error
		if name, err = cleanName(name); err != nil {
			return err
		}
	}
	return l.transaction(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		if rename {
			if err := changed(tx.ExecContext(ctx, "UPDATE assets SET name=?,updated_at=? WHERE id=?", name, now, id)); err != nil {
				return err
			}
		}
		if details {
			return changed(tx.ExecContext(ctx, "UPDATE assets SET note=?,source_url=?,updated_at=? WHERE id=?", note, sourceURL, now, id))
		}
		return nil
	})
}

// SetAssetsNote 把多个素材的备注改成同一段，任何一个失败整批都不改。
func (l *Library) SetAssetsNote(ctx context.Context, ids []string, note string) error {
	return l.setAssetsColumn(ctx, ids, "UPDATE assets SET note=?,updated_at=? WHERE id=?", note)
}

// SetAssetsSourceURL 把多个素材的来源链接改成同一个，任何一个失败整批都不改。
func (l *Library) SetAssetsSourceURL(ctx context.Context, ids []string, sourceURL string) error {
	return l.setAssetsColumn(ctx, ids, "UPDATE assets SET source_url=?,updated_at=? WHERE id=?", sourceURL)
}

func (l *Library) setAssetsColumn(ctx context.Context, ids []string, query, value string) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		for _, id := range ids {
			if err := changed(tx.ExecContext(ctx, query, value, now, id)); err != nil {
				return err
			}
		}
		return nil
	})
}
