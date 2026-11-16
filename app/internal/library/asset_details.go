package library

import (
	"context"
	"database/sql"
	"time"
)

func (l *Library) UpdateAssetDetails(ctx context.Context, id, note, sourceURL string) error {
	return changed(l.DB.ExecContext(ctx, "UPDATE assets SET note=?,source_url=?,updated_at=? WHERE id=?", note, sourceURL, time.Now().UTC().Format(time.RFC3339Nano), id))
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
