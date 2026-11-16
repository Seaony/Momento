package library

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
)

// Snapshot 读出整个图库供界面浏览。为了打开快、占内存少：
//   - 素材按表里的存放顺序读，不按 ID 排（界面自己排序）；
//   - 不读 EXIF（Asset.EXIF 为空），选中时用 AssetEXIF 单独读；
//   - 每个素材的 FolderIDs、TagIDs 仍按 ID 升序，界面的内存补丁依赖这个顺序；Colors 按 sort_index 排。
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
	// 扩展名、色彩配置、颜色分类、文件夹和标签 ID 大量重复，同样的字符串只留一份。
	strs := interner{}
	var total int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM assets").Scan(&total); err != nil {
		return snapshot, err
	}
	snapshot.Assets = make([]model.Asset, 0, total)
	extra := ",last_used_at,rotation"
	if l.Manifest.SchemaVersion < 2 {
		extra = ",'',0"
	}
	// 只读打开的旧图库（备份、导入图库的来源）没升级过，没有视频那几列，按图片算。
	if l.Manifest.SchemaVersion >= 4 {
		extra += ",kind,duration_ms,motion_path"
	} else {
		extra += ",'image',0,''"
	}
	if l.Manifest.SchemaVersion >= 5 {
		extra += ",rating"
	} else {
		extra += ",0"
	}
	// 不加 ORDER BY：按 id 排要走索引逐行回表，10 万张时比顺序扫表慢得多。
	rows, err := tx.QueryContext(ctx, "SELECT id,name,original_file_name,extension,media_type,size,width,height,orientation,color_profile,note,source_url,relative_path,favorite,trashed,created_at,updated_at"+extra+" FROM assets")
	if err != nil {
		return snapshot, err
	}
	byID := make(map[string]int32, total)
	for rows.Next() {
		var asset model.Asset
		var extension, mediaType, colorProfile, created, updated, lastUsed, kind string
		if err = rows.Scan(&asset.ID, &asset.Name, &asset.OriginalFileName, &extension, &mediaType, &asset.Size, &asset.Width, &asset.Height, &asset.Orientation, &colorProfile, &asset.Note, &asset.SourceURL, &asset.RelativePath, &asset.Favorite, &asset.Trashed, &created, &updated, &lastUsed, &asset.Rotation, &kind, &asset.DurationMS, &asset.MotionPath, &asset.Rating); err != nil {
			rows.Close()
			return snapshot, err
		}
		asset.Extension, asset.MediaType, asset.ColorProfile, asset.Kind = strs.get(extension), strs.get(mediaType), strs.get(colorProfile), strs.get(kind)
		if !filepath.IsLocal(asset.RelativePath) || strings.ContainsAny(asset.RelativePath, "\\:") ||
			asset.MotionPath != "" && (!filepath.IsLocal(asset.MotionPath) || strings.ContainsAny(asset.MotionPath, "\\:")) {
			rows.Close()
			return snapshot, i18n.NewError("error_asset_path", asset.ID)
		}
		if asset.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			rows.Close()
			return snapshot, err
		}
		if asset.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
			rows.Close()
			return snapshot, err
		}
		if lastUsed != "" {
			if asset.LastUsedAt, err = time.Parse(time.RFC3339Nano, lastUsed); err != nil {
				rows.Close()
				return snapshot, err
			}
		}
		if asset.Rotation != 0 && asset.Rotation != 90 && asset.Rotation != 180 && asset.Rotation != 270 {
			rows.Close()
			return snapshot, i18n.NewError("error_rotation")
		}
		byID[asset.ID] = int32(len(snapshot.Assets))
		snapshot.Assets = append(snapshot.Assets, asset)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return snapshot, err
	}
	if err = loadColors(ctx, tx, snapshot.Assets, byID, strs); err != nil {
		return snapshot, err
	}
	// 升级前的旧库（只在升级流程里读）还没有 icon 列。
	folderIcon := "icon"
	if l.Manifest.SchemaVersion < 3 {
		folderIcon = "''"
	}
	rows, err = tx.QueryContext(ctx, "SELECT id,name,COALESCE(parent_id,''),sort_index,"+folderIcon+" FROM folders ORDER BY sort_index,id")
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var folder model.Folder
		if err = rows.Scan(&folder.ID, &folder.Name, &folder.ParentID, &folder.SortIndex, &folder.Icon); err != nil {
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
		// 按 (asset_id, 成员 ID) 排序走的是主键覆盖索引，顺序读、不回表，成员 ID 因此按升序排好。
		var assetID, memberID string
		for rows.Next() {
			if err = rows.Scan(&assetID, &memberID); err != nil {
				rows.Close()
				return snapshot, err
			}
			if i, ok := byID[assetID]; ok {
				if relationship.table == "asset_folders" {
					snapshot.Assets[i].FolderIDs = append(snapshot.Assets[i].FolderIDs, strs.get(memberID))
				} else {
					snapshot.Assets[i].TagIDs = append(snapshot.Assets[i].TagIDs, strs.get(memberID))
				}
			}
		}
		if err = errors.Join(rows.Err(), rows.Close()); err != nil {
			return snapshot, err
		}
	}
	if l.Manifest.SchemaVersion >= 6 {
		if err = loadAnnotations(ctx, tx, snapshot.Assets, byID); err != nil {
			return snapshot, err
		}
	}
	if l.Manifest.SchemaVersion >= 2 {
		rows, err = tx.QueryContext(ctx, "SELECT id,name,match_mode,rules_json,sort_index FROM smart_folders ORDER BY sort_index,id")
		if err != nil {
			return snapshot, err
		}
		for rows.Next() {
			var folder model.SmartFolder
			var rules string
			if err = rows.Scan(&folder.ID, &folder.Name, &folder.Match, &rules, &folder.SortIndex); err != nil {
				rows.Close()
				return snapshot, err
			}
			if err = json.Unmarshal([]byte(rules), &folder.Rules); err != nil {
				rows.Close()
				return snapshot, err
			}
			if err = catalog.ValidateSmartFolder(folder); err != nil {
				rows.Close()
				return snapshot, err
			}
			snapshot.SmartFolders = append(snapshot.SmartFolders, folder)
		}
		if err = errors.Join(rows.Err(), rows.Close()); err != nil {
			return snapshot, err
		}
		rows, err = tx.QueryContext(ctx, "SELECT id,kind,target_id,sort_index FROM quick_access ORDER BY sort_index,id")
		if err != nil {
			return snapshot, err
		}
		for rows.Next() {
			var entry model.QuickAccess
			if err = rows.Scan(&entry.ID, &entry.Kind, &entry.TargetID, &entry.SortIndex); err != nil {
				rows.Close()
				return snapshot, err
			}
			snapshot.QuickAccess = append(snapshot.QuickAccess, entry)
		}
		if err = errors.Join(rows.Err(), rows.Close()); err != nil {
			return snapshot, err
		}
	}
	err = tx.Commit()
	return snapshot, err
}

// loadColors 顺序扫颜色表，再按 (素材, sort_index) 归位；所有素材的颜色放在同一个底层数组里，省掉几十万次小分配。
func loadColors(ctx context.Context, tx *sql.Tx, assets []model.Asset, byID map[string]int32, strs interner) error {
	var total int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM asset_colors").Scan(&total); err != nil {
		return err
	}
	colors := make([]model.Color, 0, total)
	// 每个颜色属于哪个素材、排第几，用来判断顺序和重排。
	type colorKey struct {
		asset int32
		sort  int64
	}
	compare := func(a, b colorKey) int { return cmp.Or(cmp.Compare(a.asset, b.asset), cmp.Compare(a.sort, b.sort)) }
	keys := make([]colorKey, 0, total)
	rows, err := tx.QueryContext(ctx, "SELECT asset_id,hex,proportion,family,sort_index FROM asset_colors")
	if err != nil {
		return err
	}
	var assetID, family string
	for rows.Next() {
		var color model.Color
		var sortIndex int64
		if err = rows.Scan(&assetID, &color.Hex, &color.Proportion, &family, &sortIndex); err != nil {
			return errors.Join(err, rows.Close())
		}
		i, ok := byID[assetID]
		if !ok {
			continue
		}
		color.Family = strs.get(family)
		colors = append(colors, color)
		keys = append(keys, colorKey{i, sortIndex})
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	// 颜色通常和素材按同样顺序写入，已经排好；只有不是时才重排。
	if !slices.IsSortedFunc(keys, compare) {
		order := make([]int, len(keys))
		for i := range order {
			order[i] = i
		}
		slices.SortFunc(order, func(a, b int) int { return compare(keys[a], keys[b]) })
		sortedColors, sortedKeys := make([]model.Color, len(colors)), make([]colorKey, len(keys))
		for to, from := range order {
			sortedColors[to], sortedKeys[to] = colors[from], keys[from]
		}
		colors, keys = sortedColors, sortedKeys
	}
	for start := 0; start < len(keys); {
		end := start + 1
		for end < len(keys) && keys[end].asset == keys[start].asset {
			end++
		}
		// 容量卡在自己这段，之后谁 append 都会另起数组，不会改到别的素材。
		assets[keys[start].asset].Colors = colors[start:end:end]
		start = end
	}
	return nil
}

// interner 让重复出现的短字符串共用一份内存。
type interner map[string]string

func (m interner) get(s string) string {
	if canonical, ok := m[s]; ok {
		return canonical
	}
	m[s] = s
	return s
}

// AssetEXIF 读一张素材的 EXIF；快照不带 EXIF，检查器选中素材时再读。素材不存在返回 sql.ErrNoRows。
func (l *Library) AssetEXIF(ctx context.Context, id string) (map[string]string, error) {
	var raw string
	if err := l.DB.QueryRowContext(ctx, "SELECT exif_json FROM assets WHERE id=?", id).Scan(&raw); err != nil {
		return nil, err
	}
	var exif map[string]string
	if err := json.Unmarshal([]byte(raw), &exif); err != nil {
		return nil, i18n.WrapError(err, "error_asset_exif", id)
	}
	return exif, nil
}
