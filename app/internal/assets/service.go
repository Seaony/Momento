package assets

import (
	"context"
	"database/sql"
	"errors"
	"image"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

var ErrImportBusy = i18n.NewError("error_import_busy")

type Service struct {
	root         string
	db           *sql.DB
	importing    sync.Mutex
	mutation     sync.Mutex
	prepareImage func(string) (image.Image, map[string]string, int, string, error)
}

func New(root string, db *sql.DB) *Service {
	return &Service{root: root, db: db, prepareImage: PrepareImage}
}

func (s *Service) assetPath(relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || !filepath.IsLocal(filepath.FromSlash(relative)) {
		return "", i18n.NewError("error_asset_relative_path", relative)
	}
	path := filepath.Join(s.root, filepath.FromSlash(relative))
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(s.root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || !filepath.IsLocal(rel) {
		return "", i18n.NewError("error_asset_path_escape", relative)
	}
	return resolved, nil
}

// 图库里的写入目录由应用创建，不跟随包内被替换的符号链接。
func (s *Service) storageDestination(relative string) (string, error) {
	if relative == "" || !filepath.IsLocal(filepath.FromSlash(relative)) {
		return "", i18n.NewError("error_asset_relative_path", relative)
	}
	current := s.root
	for _, part := range strings.Split(filepath.Dir(filepath.FromSlash(relative)), string(filepath.Separator)) {
		if part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err = os.Mkdir(current, 0700); err != nil {
				return "", err
			}
			continue
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", i18n.NewError("error_storage_directory", current)
		}
	}
	return filepath.Join(s.root, filepath.FromSlash(relative)), nil
}

func thumbnailRelative(id string) (string, error) {
	if len(id) != 64 || strings.ContainsAny(id, "/\\") {
		return "", i18n.NewError("error_asset_id", id)
	}
	for _, r := range id {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return "", i18n.NewError("error_asset_id", id)
		}
	}
	return filepath.Join("thumbnails", id+".png"), nil
}

func copyFile(ctx context.Context, src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, out.Close())
		if err != nil {
			err = errors.Join(err, removeIfExists(dst))
		}
	}()
	_, err = io.Copy(out, contextReader{ctx: ctx, r: in})
	if err == nil {
		err = out.Sync()
	}
	return err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// 同目录暂存后才发布文件，排他的占位保证不会覆盖同名文件。
func publishFile(staged, dst string) error {
	if err := os.Link(staged, dst); err == nil {
		return nil
	} else if os.IsExist(err) {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return errors.Join(err, removeIfExists(dst))
	}
	if err = os.Rename(staged, dst); err != nil {
		return errors.Join(err, removeIfExists(dst))
	}
	return nil
}

func (s *Service) setTrashed(ctx context.Context, ids []string, value bool) error {
	s.mutation.Lock()
	defer s.mutation.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err = tx.ExecContext(ctx, "UPDATE assets SET trashed=?, updated_at=? WHERE id=?", value, nowText(), id); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err == nil {
		slog.Info("修改素材回收站状态", "count", len(ids), "trashed", value)
	}
	return err
}
func (s *Service) Trash(ctx context.Context, ids []string) error { return s.setTrashed(ctx, ids, true) }
func (s *Service) Restore(ctx context.Context, ids []string) error {
	return s.setTrashed(ctx, ids, false)
}

func (s *Service) RefreshThumbnails(ctx context.Context, items []model.Asset) error {
	s.mutation.Lock()
	defer s.mutation.Unlock()
	for _, a := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		path, err := s.assetPath(a.RelativePath)
		if err != nil {
			return err
		}
		img, _, _, _, err := PrepareImage(path)
		if err != nil {
			return i18n.WrapError(err, "error_thumbnail_refresh", a.Name)
		}
		rel, err := thumbnailRelative(a.ID)
		if err != nil {
			return err
		}
		destination, err := s.storageDestination(rel)
		if err != nil {
			return err
		}
		if err = WriteThumbnail(destination, img); err != nil {
			return err
		}
	}
	slog.Info("刷新缩略图", "count", len(items))
	return nil
}

func (s *Service) ReanalyzeColors(ctx context.Context, items []model.Asset) error {
	s.mutation.Lock()
	defer s.mutation.Unlock()
	palettes := make(map[string][]model.Color, len(items))
	for _, a := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		path, err := s.assetPath(a.RelativePath)
		if err != nil {
			return err
		}
		img, _, _, _, err := PrepareImage(path)
		if err != nil {
			return err
		}
		palettes[a.ID] = Palette(img)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, a := range items {
		if err = replaceColors(ctx, tx, a.ID, palettes[a.ID]); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	slog.Info("重新分析颜色", "count", len(items))
	return nil
}
