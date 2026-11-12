package assets

import (
	"context"
	"errors"
	"fmt"
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
		var relative string
		var trashed bool
		if err = tx.QueryRowContext(ctx, "SELECT relative_path,trashed FROM assets WHERE id=?", id).Scan(&relative, &trashed); err != nil {
			return err
		}
		if !trashed {
			return i18n.NewError("error_asset_not_trashed", id)
		}
		thumb, e := thumbnailRelative(id)
		if e != nil {
			return e
		}
		for _, rel := range []string{relative, thumb} {
			from, e := s.assetPath(rel)
			if os.IsNotExist(e) {
				slog.Warn("永久删除时文件已不存在", "path", rel)
				continue
			}
			if e != nil {
				return e
			}
			to := filepath.Join(stage, fmt.Sprint(len(moved)))
			if err = os.Rename(from, to); err != nil {
				return err
			}
			moved = append(moved, deletionFile{from: from, to: to})
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

func (s *Service) EmptyTrash(ctx context.Context) error {
	s.mutation.Lock()
	defer s.mutation.Unlock()
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM assets WHERE trashed=1 ORDER BY id")
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return err
	}
	return s.deletePermanently(ctx, ids)
}
