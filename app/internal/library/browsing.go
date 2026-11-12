package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/google/uuid"
)

func (l *Library) MarkUsed(ctx context.Context, ids []string, at time.Time) error {
	if at.IsZero() {
		at = time.Now()
	}
	return l.transaction(ctx, func(tx *sql.Tx) error {
		for _, id := range ids {
			if err := changed(tx.ExecContext(ctx, "UPDATE assets SET last_used_at=? WHERE id=?", at.UTC().Format(time.RFC3339Nano), id)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *Library) SetRotation(ctx context.Context, ids []string, rotation int) error {
	if rotation%90 != 0 {
		return i18n.NewError("error_rotation")
	}
	rotation = (rotation%360 + 360) % 360
	return l.transaction(ctx, func(tx *sql.Tx) error {
		for _, id := range ids {
			if err := changed(tx.ExecContext(ctx, "UPDATE assets SET rotation=?,updated_at=? WHERE id=?", rotation, time.Now().UTC().Format(time.RFC3339Nano), id)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *Library) RotateAssets(ctx context.Context, ids []string, delta int) error {
	if delta%90 != 0 {
		return i18n.NewError("error_rotation")
	}
	delta = (delta%360 + 360) % 360
	return l.transaction(ctx, func(tx *sql.Tx) error {
		for _, id := range ids {
			if err := changed(tx.ExecContext(ctx, "UPDATE assets SET rotation=(rotation+?)%360,updated_at=? WHERE id=?", delta, time.Now().UTC().Format(time.RFC3339Nano), id)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *Library) CreateSmartFolder(ctx context.Context, name, match string, rules []model.SmartRule) (model.SmartFolder, error) {
	folder := model.SmartFolder{ID: uuid.NewString(), Name: name, Match: match, Rules: rules}
	name, err := cleanName(name)
	if err != nil {
		return folder, err
	}
	folder.Name = name
	if err = catalog.ValidateSmartFolder(folder); err != nil {
		return folder, err
	}
	data, err := json.Marshal(rules)
	if err != nil {
		return folder, err
	}
	err = l.transaction(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_index)+1,0) FROM smart_folders").Scan(&folder.SortIndex); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO smart_folders(id,name,match_mode,rules_json,sort_index) VALUES(?,?,?,?,?)", folder.ID, folder.Name, folder.Match, string(data), folder.SortIndex)
		return err
	})
	return folder, err
}

func (l *Library) UpdateSmartFolder(ctx context.Context, id, name, match string, rules []model.SmartRule) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	if err = catalog.ValidateSmartFolder(model.SmartFolder{Name: name, Match: match, Rules: rules}); err != nil {
		return err
	}
	data, err := json.Marshal(rules)
	if err != nil {
		return err
	}
	return changed(l.DB.ExecContext(ctx, "UPDATE smart_folders SET name=?,match_mode=?,rules_json=? WHERE id=?", name, match, string(data), id))
}

func (l *Library) RenameSmartFolder(ctx context.Context, id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return changed(l.DB.ExecContext(ctx, "UPDATE smart_folders SET name=? WHERE id=?", name, id))
}

func (l *Library) DeleteSmartFolder(ctx context.Context, id string) error {
	return changed(l.DB.ExecContext(ctx, "DELETE FROM smart_folders WHERE id=?", id))
}

func quickAccessTable(kind string) string {
	switch kind {
	case "folder":
		return "folders"
	case "tag":
		return "tags"
	case "smart":
		return "smart_folders"
	}
	return ""
}

func (l *Library) PinQuickAccess(ctx context.Context, kind, targetID string) (model.QuickAccess, error) {
	entry := model.QuickAccess{ID: uuid.NewString(), Kind: kind, TargetID: targetID}
	table := quickAccessTable(kind)
	if table == "" {
		return entry, i18n.NewError("error_quick_access_kind")
	}
	err := l.transaction(ctx, func(tx *sql.Tx) error {
		var exists string
		if err := tx.QueryRowContext(ctx, "SELECT id FROM "+table+" WHERE id=?", targetID).Scan(&exists); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_index)+1,0) FROM quick_access").Scan(&entry.SortIndex); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO quick_access(id,kind,target_id,sort_index) VALUES(?,?,?,?) ON CONFLICT(kind,target_id) DO NOTHING", entry.ID, kind, targetID, entry.SortIndex); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, "SELECT id,sort_index FROM quick_access WHERE kind=? AND target_id=?", kind, targetID).Scan(&entry.ID, &entry.SortIndex)
	})
	return entry, err
}

func (l *Library) UnpinQuickAccess(ctx context.Context, id string) error {
	return changed(l.DB.ExecContext(ctx, "DELETE FROM quick_access WHERE id=?", id))
}

func (l *Library) MoveQuickAccess(ctx context.Context, id string, index int) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		var exists string
		if err := tx.QueryRowContext(ctx, "SELECT id FROM quick_access WHERE id=?", id).Scan(&exists); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, "SELECT id FROM quick_access WHERE id<>? ORDER BY sort_index,id", id)
		if err != nil {
			return err
		}
		var ids []string
		for rows.Next() {
			var other string
			if err = rows.Scan(&other); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, other)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err = rows.Close(); err != nil {
			return err
		}
		index = max(0, min(index, len(ids)))
		ids = append(ids, "")
		copy(ids[index+1:], ids[index:])
		ids[index] = id
		for i, other := range ids {
			if _, err = tx.ExecContext(ctx, "UPDATE quick_access SET sort_index=? WHERE id=?", i, other); err != nil {
				return err
			}
		}
		return nil
	})
}
