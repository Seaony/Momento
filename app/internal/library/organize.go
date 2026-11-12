package library

import (
	"context"
	"database/sql"
	"errors"
	"github.com/Seaony/Momento/app/internal/i18n"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

func parentValue(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func (l *Library) transaction(ctx context.Context, run func(*sql.Tx) error) error {
	tx, err := l.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := run(tx); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	return tx.Commit()
}

func (l *Library) CreateFolder(ctx context.Context, name, parentID string) (model.Folder, error) {
	name, err := cleanName(name)
	if err != nil {
		return model.Folder{}, err
	}
	folder := model.Folder{ID: uuid.NewString(), Name: name, ParentID: parentID}
	err = l.transaction(ctx, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(sort_index)+1,0) FROM folders WHERE parent_id IS ?", parentValue(parentID)).Scan(&folder.SortIndex); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO folders(id,name,parent_id,sort_index) VALUES(?,?,?,?)", folder.ID, name, parentValue(parentID), folder.SortIndex)
		return err
	})
	return folder, err
}

func (l *Library) RenameFolder(ctx context.Context, id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return changed(l.DB.ExecContext(ctx, "UPDATE folders SET name=? WHERE id=?", name, id))
}

func (l *Library) DeleteFolder(ctx context.Context, id string) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		var parent sql.NullString
		if err := tx.QueryRowContext(ctx, "SELECT parent_id FROM folders WHERE id=?", id).Scan(&parent); err != nil {
			return err
		}
		if err := changed(tx.ExecContext(ctx, "DELETE FROM folders WHERE id=?", id)); err != nil {
			return err
		}
		return reorderSiblings(ctx, tx, parent.String, "", 0)
	})
}

// index 是移走当前项之后，在目标同级列表中的插入位置。
func (l *Library) MoveFolder(ctx context.Context, id, parentID string, index int) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		var oldParent sql.NullString
		if err := tx.QueryRowContext(ctx, "SELECT parent_id FROM folders WHERE id=?", id).Scan(&oldParent); err != nil {
			return err
		}
		if parentID != "" {
			var descendant bool
			if err := tx.QueryRowContext(ctx, `WITH RECURSIVE tree(id) AS (SELECT id FROM folders WHERE id=? UNION ALL SELECT f.id FROM folders f JOIN tree t ON f.parent_id=t.id) SELECT EXISTS(SELECT 1 FROM tree WHERE id=?)`, id, parentID).Scan(&descendant); err != nil {
				return err
			}
			if descendant {
				return i18n.NewError("error_folder_cycle")
			}
		}
		if _, err := tx.ExecContext(ctx, "UPDATE folders SET parent_id=? WHERE id=?", parentValue(parentID), id); err != nil {
			return err
		}
		if oldParent.String != parentID {
			if err := reorderSiblings(ctx, tx, oldParent.String, "", 0); err != nil {
				return err
			}
		}
		return reorderSiblings(ctx, tx, parentID, id, index)
	})
}

func reorderSiblings(ctx context.Context, tx *sql.Tx, parentID, movedID string, index int) error {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM folders WHERE parent_id IS ? AND id<>? ORDER BY sort_index,id", parentValue(parentID), movedID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	if movedID != "" {
		index = max(0, min(index, len(ids)))
		ids = append(ids, "")
		copy(ids[index+1:], ids[index:])
		ids[index] = movedID
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE folders SET sort_index=? WHERE id=?", i, id); err != nil {
			return err
		}
	}
	return nil
}

func NormalizeTag(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(norm.NFKC.String(name)), " "))
}

func (l *Library) CreateTag(ctx context.Context, name string) (model.Tag, error) {
	name, err := cleanName(name)
	if err != nil {
		return model.Tag{}, err
	}
	tag := model.Tag{ID: uuid.NewString(), Name: name, NormalizedName: NormalizeTag(name)}
	err = l.transaction(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO tags(id,name,normalized_name,color_hex) VALUES(?,?,?,?) ON CONFLICT(normalized_name) DO NOTHING", tag.ID, tag.Name, tag.NormalizedName, tag.ColorHex); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, "SELECT id,name,normalized_name,color_hex FROM tags WHERE normalized_name=?", tag.NormalizedName).Scan(&tag.ID, &tag.Name, &tag.NormalizedName, &tag.ColorHex)
	})
	return tag, err
}

func (l *Library) RenameTag(ctx context.Context, id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return changed(l.DB.ExecContext(ctx, "UPDATE tags SET name=?,normalized_name=? WHERE id=?", name, NormalizeTag(name), id))
}

func (l *Library) DeleteTag(ctx context.Context, id string) error {
	return changed(l.DB.ExecContext(ctx, "DELETE FROM tags WHERE id=?", id))
}

func (l *Library) SetAssetFolders(ctx context.Context, assetIDs, folderIDs []string, add bool) error {
	return l.setMemberships(ctx, "asset_folders", "folder_id", assetIDs, folderIDs, add)
}

func (l *Library) SetAssetTags(ctx context.Context, assetIDs, tagIDs []string, add bool) error {
	return l.setMemberships(ctx, "asset_tags", "tag_id", assetIDs, tagIDs, add)
}

func (l *Library) setMemberships(ctx context.Context, table, column string, assetIDs, memberIDs []string, add bool) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		query := "DELETE FROM " + table + " WHERE asset_id=? AND " + column + "=?"
		if add {
			query = "INSERT OR IGNORE INTO " + table + "(asset_id," + column + ") VALUES(?,?)"
		}
		stmt, err := tx.PrepareContext(ctx, query)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, assetID := range assetIDs {
			for _, memberID := range memberIDs {
				if _, err := stmt.ExecContext(ctx, assetID, memberID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (l *Library) SetFavorite(ctx context.Context, assetIDs []string, value bool) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		for _, id := range assetIDs {
			if err := changed(tx.ExecContext(ctx, "UPDATE assets SET favorite=?,updated_at=? WHERE id=?", value, time.Now().UTC().Format(time.RFC3339Nano), id)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (l *Library) RenameAsset(ctx context.Context, id, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return changed(l.DB.ExecContext(ctx, "UPDATE assets SET name=?,updated_at=? WHERE id=?", name, time.Now().UTC().Format(time.RFC3339Nano), id))
}

func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
