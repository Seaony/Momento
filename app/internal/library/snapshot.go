package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Seaony/Momento/app/internal/i18n"
	"path/filepath"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func (l *Library) Snapshot(ctx context.Context) (snapshot model.Snapshot, err error) {
	tx, err := l.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return snapshot, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, tx.Rollback())
		}
	}()
	rows, err := tx.QueryContext(ctx, "SELECT id,name,original_file_name,extension,media_type,size,width,height,orientation,color_profile,exif_json,note,source_url,relative_path,favorite,trashed,created_at,updated_at FROM assets ORDER BY id")
	if err != nil {
		return snapshot, err
	}
	byID := map[string]int{}
	for rows.Next() {
		var asset model.Asset
		var exif, created, updated string
		if err = rows.Scan(&asset.ID, &asset.Name, &asset.OriginalFileName, &asset.Extension, &asset.MediaType, &asset.Size, &asset.Width, &asset.Height, &asset.Orientation, &asset.ColorProfile, &exif, &asset.Note, &asset.SourceURL, &asset.RelativePath, &asset.Favorite, &asset.Trashed, &created, &updated); err != nil {
			rows.Close()
			return snapshot, err
		}
		if !filepath.IsLocal(asset.RelativePath) || strings.ContainsAny(asset.RelativePath, "\\:") {
			rows.Close()
			return snapshot, i18n.NewError("error_asset_path", asset.ID)
		}
		if err = json.Unmarshal([]byte(exif), &asset.EXIF); err != nil {
			rows.Close()
			return snapshot, i18n.WrapError(err, "error_asset_exif", asset.ID)
		}
		if asset.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			rows.Close()
			return snapshot, err
		}
		if asset.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
			rows.Close()
			return snapshot, err
		}
		byID[asset.ID] = len(snapshot.Assets)
		snapshot.Assets = append(snapshot.Assets, asset)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return snapshot, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT asset_id,hex,proportion,family FROM asset_colors ORDER BY asset_id,sort_index")
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var id string
		var color model.Color
		if err = rows.Scan(&id, &color.Hex, &color.Proportion, &color.Family); err != nil {
			rows.Close()
			return snapshot, err
		}
		if i, ok := byID[id]; ok {
			snapshot.Assets[i].Colors = append(snapshot.Assets[i].Colors, color)
		}
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return snapshot, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT id,name,COALESCE(parent_id,''),sort_index FROM folders ORDER BY sort_index,id")
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var folder model.Folder
		if err = rows.Scan(&folder.ID, &folder.Name, &folder.ParentID, &folder.SortIndex); err != nil {
			rows.Close()
			return snapshot, err
		}
		snapshot.Folders = append(snapshot.Folders, folder)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return snapshot, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT id,name,normalized_name,color_hex FROM tags ORDER BY normalized_name,id")
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var tag model.Tag
		if err = rows.Scan(&tag.ID, &tag.Name, &tag.NormalizedName, &tag.ColorHex); err != nil {
			rows.Close()
			return snapshot, err
		}
		snapshot.Tags = append(snapshot.Tags, tag)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return snapshot, err
	}
	for _, relationship := range []struct{ table, column string }{{"asset_folders", "folder_id"}, {"asset_tags", "tag_id"}} {
		rows, err = tx.QueryContext(ctx, "SELECT asset_id,"+relationship.column+" FROM "+relationship.table+" ORDER BY asset_id,"+relationship.column)
		if err != nil {
			return snapshot, err
		}
		for rows.Next() {
			var assetID, memberID string
			if err = rows.Scan(&assetID, &memberID); err != nil {
				rows.Close()
				return snapshot, err
			}
			if i, ok := byID[assetID]; ok {
				if relationship.table == "asset_folders" {
					snapshot.Assets[i].FolderIDs = append(snapshot.Assets[i].FolderIDs, memberID)
				} else {
					snapshot.Assets[i].TagIDs = append(snapshot.Assets[i].TagIDs, memberID)
				}
			}
		}
		if err = errors.Join(rows.Err(), rows.Close()); err != nil {
			return snapshot, err
		}
	}
	err = tx.Commit()
	return snapshot, err
}
