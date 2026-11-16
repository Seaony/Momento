package library

import (
	"context"
	"database/sql"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

// MergedAsset 是把 others 合并进 keep 之后 keep 的样子：标签、文件夹取并集，收藏有一个就算，
// 评分取最高，备注把不同的几段接起来，来源链接 keep 没有时用第一个有的。界面改内存和写库都用它，两边一致。
func MergedAsset(keep model.Asset, others []model.Asset) model.Asset {
	merged := keep
	merged.TagIDs = slices.Clone(keep.TagIDs)
	merged.FolderIDs = slices.Clone(keep.FolderIDs)
	notes := []string{}
	if note := strings.TrimSpace(keep.Note); note != "" {
		notes = append(notes, keep.Note)
	}
	for _, other := range others {
		for _, id := range other.TagIDs {
			if !slices.Contains(merged.TagIDs, id) {
				merged.TagIDs = append(merged.TagIDs, id)
			}
		}
		for _, id := range other.FolderIDs {
			if !slices.Contains(merged.FolderIDs, id) {
				merged.FolderIDs = append(merged.FolderIDs, id)
			}
		}
		merged.Favorite = merged.Favorite || other.Favorite
		merged.Rating = max(merged.Rating, other.Rating)
		if note := strings.TrimSpace(other.Note); note != "" && !slices.ContainsFunc(notes, func(n string) bool { return strings.TrimSpace(n) == note }) {
			notes = append(notes, other.Note)
		}
		if strings.TrimSpace(merged.SourceURL) == "" {
			merged.SourceURL = other.SourceURL
		}
	}
	merged.Note = strings.Join(notes, "\n\n")
	// 和从库里读出来的一样按升序，界面直接改内存后不会和重读的顺序不同。
	slices.Sort(merged.TagIDs)
	slices.Sort(merged.FolderIDs)
	return merged
}

// MergeGroup 是一组相似素材：Others 并进 Keep。
type MergeGroup struct {
	Keep   model.Asset
	Others []model.Asset
}

// MergeAssets 把每组 Others 的标签、文件夹、备注等并进 Keep，再把 Others 移到回收站（可以恢复）。
// 所有组在一个事务里做完，有一组失败就全部不动。
func (l *Library) MergeAssets(ctx context.Context, groups ...MergeGroup) error {
	merged := 0
	err := l.transaction(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		for _, group := range groups {
			if err := mergeGroup(ctx, tx, group, now); err != nil {
				return err
			}
			merged += len(group.Others)
		}
		return nil
	})
	if err == nil {
		slog.Info("合并相似素材", "groups", len(groups), "merged", merged)
	}
	return err
}

func mergeGroup(ctx context.Context, tx *sql.Tx, group MergeGroup, now string) error {
	keep := group.Keep
	merged := MergedAsset(keep, group.Others)
	if err := changed(tx.ExecContext(ctx, "UPDATE assets SET note=?,source_url=?,favorite=?,rating=?,updated_at=? WHERE id=? AND trashed=0",
		merged.Note, merged.SourceURL, merged.Favorite, merged.Rating, now, keep.ID)); err != nil {
		return err
	}
	for _, id := range merged.TagIDs {
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_tags(asset_id,tag_id) VALUES(?,?)", keep.ID, id); err != nil {
			return err
		}
	}
	for _, id := range merged.FolderIDs {
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO asset_folders(asset_id,folder_id) VALUES(?,?)", keep.ID, id); err != nil {
			return err
		}
	}
	for _, other := range group.Others {
		if other.ID == keep.ID {
			continue
		}
		if err := changed(tx.ExecContext(ctx, "UPDATE assets SET trashed=1,updated_at=? WHERE id=? AND trashed=0", now, other.ID)); err != nil {
			return err
		}
	}
	return nil
}
