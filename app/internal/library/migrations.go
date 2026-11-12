package library

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

var baseValidation = []string{
	"SELECT id,name,original_file_name,extension,media_type,size,width,height,orientation,color_profile,exif_json,note,source_url,relative_path,favorite,trashed,created_at,updated_at FROM assets LIMIT 0",
	"SELECT asset_id,hex,proportion,family,sort_index FROM asset_colors LIMIT 0",
	"SELECT id,name,parent_id,sort_index FROM folders LIMIT 0",
	"SELECT asset_id,folder_id FROM asset_folders LIMIT 0",
	"SELECT id,name,normalized_name,color_hex FROM tags LIMIT 0",
	"SELECT asset_id,tag_id FROM asset_tags LIMIT 0",
}

var v2Validation = []string{
	"SELECT last_used_at,rotation FROM assets LIMIT 0",
	"SELECT id,name,match_mode,rules_json,sort_index FROM smart_folders LIMIT 0",
	"SELECT id,kind,target_id,sort_index FROM quick_access LIMIT 0",
}

type schemaReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func validateSchema(ctx context.Context, db schemaReader, version int) error {
	queries := baseValidation
	if version >= 2 {
		queries = append(append([]string{}, baseValidation...), v2Validation...)
	}
	for _, query := range queries {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return i18n.WrapError(err, "error_database_invalid")
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func upgrade(ctx context.Context, db *sql.DB, root string, manifest *model.Manifest) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, rollbackErr := conn.ExecContext(context.Background(), "ROLLBACK")
			err = errors.Join(err, rollbackErr)
		}
	}()
	var version int
	if err = conn.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version < 1 || version > model.SchemaVersion || version > manifest.SchemaVersion {
		return i18n.NewError("error_database_invalid")
	}
	// 先确认老表齐全，不能把缺表损坏误当作待升级。
	if err = validateSchema(ctx, conn, version); err != nil {
		return err
	}
	if version == 1 {
		if _, err = conn.ExecContext(ctx, migrationV2); err != nil {
			return err
		}
	}
	if err = validateSchema(ctx, conn, model.SchemaVersion); err != nil {
		return err
	}
	before := *manifest
	next := before
	next.SchemaVersion = model.SchemaVersion
	if next.SchemaVersion != before.SchemaVersion {
		next.UpdatedAt = time.Now().UTC()
		if err = writeJSON(filepath.Join(root, "manifest.json"), next); err != nil {
			return err
		}
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		if next != before {
			err = errors.Join(err, writeJSON(filepath.Join(root, "manifest.json"), before))
		}
		return err
	}
	committed = true
	*manifest = next
	if version != model.SchemaVersion {
		slog.Info("升级图库结构", "path", root, "from", version, "to", model.SchemaVersion)
	}
	return nil
}
