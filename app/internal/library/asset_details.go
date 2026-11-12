package library

import (
	"context"
	"time"
)

func (l *Library) UpdateAssetDetails(ctx context.Context, id, note, sourceURL string) error {
	return changed(l.DB.ExecContext(ctx, "UPDATE assets SET note=?,source_url=?,updated_at=? WHERE id=?", note, sourceURL, time.Now().UTC().Format(time.RFC3339Nano), id))
}
