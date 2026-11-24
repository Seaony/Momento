package assets

import (
	"context"
	"errors"
	"github.com/Seaony/Momento/app/internal/i18n"
	"log/slog"
	"os"
	"path/filepath"
)

type deletionFile struct{ from, to string }

func (s *Service) DeletePermanently(ctx context.Context, ids []string) (err error) {
	s.mutation.Lock()
	defer s.mutation.Unlock()
	return s.deletePermanently(ctx, ids)
}

func (s *Service) deletePermanently(ctx context.Context, ids []string) (err error) {
	if len(ids) == 0 {
		return nil
	}
	slog.Info("开始永久删除", "count", len(ids))
	defer func() {
		if err != nil {
			slog.Error("永久删除失败", "error", err)
		} else {
			slog.Info("永久删除完成", "count", len(ids))
		}
	}()
	stage, err := os.MkdirTemp(s.root, ".delete-")
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Join(err, os.RemoveAll(stage))
	}
	committed := false
	var moved []deletionFile
	defer func() {
		if !committed {
			_ = tx.Rollback()
			for i := len(moved) - 1; i >= 0; i-- {
				file := moved[i]
				err = errors.Join(err, os.Rename(file.to, file.from))
			}
		}
		// 恢复失败时保留暂存目录，错误里给出位置，避免进一步丢失原图。
		if !committed && err != nil {
			entries, readErr := os.ReadDir(stage)
			if readErr != nil {
				err = errors.Join(err, readErr)
				return
			}
			if len(entries) > 0 {
				err = errors.Join(err, i18n.NewError("error_delete_retained_files", stage))
				return
			}
		}
		err = errors.Join(err, os.RemoveAll(stage))
	}()
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		var relative, motion string
		var trashed bool
		if err = tx.QueryRowContext(ctx, "SELECT relative_path,trashed,motion_path FROM assets WHERE id=?", id).Scan(&relative, &trashed, &motion); err != nil {
			return err
		}
		if !trashed {
			return i18n.NewError("error_asset_not_trashed", id)
		}
		thumb, e := thumbnailBase(id)
		if e != nil {
			return e
		}
		// 暂存名带上素材 ID 和种类（缩略图还带格式），中途崩溃后打开图库时能按名字挪回原处。
		// 缩略图 .jpg、.png 两种都挪，通常只有一种存在。
		files := []struct{ rel, kind string }{{relative, "original"}}
		if motion != "" {
			// 实况照片的视频一起删。
			files = append(files, struct{ rel, kind string }{motion, "motion"})
		}
		for _, ext := range thumbnailExtensions {
			files = append(files, struct{ rel, kind string }{thumb + ext, "thumbnail" + ext})
		}
		thumbnails := 0
		for _, file := range files {
			rel := file.rel
			from, e := s.assetPath(rel)
			if os.IsNotExist(e) {
				if file.kind == "original" || file.kind == "motion" {
					slog.Warn("永久删除时文件已不存在", "path", rel)
				}
				continue
			}
			if e != nil {
				return e
			}
			to := filepath.Join(stage, id+"."+file.kind)
			if err = os.Rename(from, to); err != nil {
				return err
			}
			moved = append(moved, deletionFile{from: from, to: to})
			if file.kind != "original" && file.kind != "motion" {
				thumbnails++
			}
		}
		if thumbnails == 0 {
			slog.Warn("永久删除时缩略图已不存在", "id", id)
		}
		if _, err = tx.ExecContext(ctx, "DELETE FROM assets WHERE id=?", id); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// EmptyTrash 清空确认时回收站里的那几个：其中已经被恢复或已经永久删掉的跳过，不让整批失败；
// 确认之后才移进回收站的也不删，用户没看到它们就不算确认过。
func (s *Service) EmptyTrash(ctx context.Context, ids []string) error {
	s.mutation.Lock()
	defer s.mutation.Unlock()
	confirmed := make(map[string]bool, len(ids))
	for _, id := range ids {
		confirmed[id] = true
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM assets WHERE trashed=1 ORDER BY id")
	if err != nil {
		return err
	}
	var trashed []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if confirmed[id] {
			trashed = append(trashed, id)
		}
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	if skipped := len(ids) - len(trashed); skipped > 0 {
		slog.Info("清空回收站时跳过已不在回收站的素材", "skipped", skipped)
	}
	return s.deletePermanently(ctx, trashed)
}
