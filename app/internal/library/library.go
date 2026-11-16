package library

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Seaony/Momento/app/internal/i18n"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type Library struct {
	Root     string
	DB       *sql.DB
	Manifest model.Manifest
	mu       sync.Mutex
}

func Create(ctx context.Context, path, displayName string) (_ *Library, err error) {
	if err := ValidateWorkingPath(path); err != nil {
		return nil, err
	}
	displayName, err = cleanName(displayName)
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if inTrash(root) {
		return nil, i18n.NewError("error_library_in_trash")
	}
	if _, err := os.Lstat(root); !os.IsNotExist(err) {
		if err == nil {
			return nil, i18n.NewError("error_library_exists", root)
		}
		return nil, err
	}
	if err := os.Mkdir(root, 0755); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if cleanupErr := os.RemoveAll(root); cleanupErr != nil {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	for _, dir := range []string{"database", "assets", "thumbnails", ".drag-out"} {
		if err = os.Mkdir(filepath.Join(root, dir), 0755); err != nil {
			return nil, err
		}
	}
	db, err := connect(ctx, filepath.Join(root, "database", "library.sqlite"), true)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, schema); err != nil {
		return nil, errors.Join(err, tx.Rollback())
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	manifest := model.Manifest{SchemaVersion: model.SchemaVersion, LibraryID: uuid.NewString(), DisplayName: displayName, CreatedAt: now, UpdatedAt: now}
	if err = writeJSON(filepath.Join(root, "manifest.json"), manifest); err != nil {
		return nil, err
	}
	slog.Info("创建图库", "path", root, "libraryID", manifest.LibraryID)
	return &Library{Root: root, DB: db, Manifest: manifest}, nil
}

func Open(ctx context.Context, path string) (_ *Library, err error) {
	if err := ValidateWorkingPath(path); err != nil {
		return nil, err
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	if inTrash(root) {
		return nil, i18n.NewError("error_library_in_trash")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, i18n.NewError("error_library_directory")
	}
	var manifest model.Manifest
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, i18n.WrapError(err, "error_manifest_invalid")
	}
	if (manifest.SchemaVersion < 1 || manifest.SchemaVersion > model.SchemaVersion) || manifest.LibraryID == "" || strings.TrimSpace(manifest.DisplayName) == "" || manifest.CreatedAt.IsZero() || manifest.UpdatedAt.IsZero() {
		return nil, i18n.NewError("error_manifest_invalid")
	}
	dbPath := filepath.Join(root, "database", "library.sqlite")
	if _, err = os.Stat(dbPath); err != nil {
		return nil, err
	}
	db, err := connect(ctx, dbPath, false)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, db.Close())
		}
	}()
	if err = upgrade(ctx, db, root, &manifest); err != nil {
		return nil, err
	}
	cleanStaging(ctx, root, db)

	slog.Info("打开图库", "path", root, "libraryID", manifest.LibraryID)
	return &Library{Root: root, DB: db, Manifest: manifest}, nil
}

// cleanStaging 清掉上次导入或永久删除中途退出留下的 .import-*、.delete-* 暂存目录。
// 永久删除没提交时数据库里还有记录，先把暂存里的原图和缩略图挪回原处，避免丢图。
// 清理出错只记日志，不影响打开图库。
func cleanStaging(ctx context.Context, root string, db *sql.DB) {
	entries, err := os.ReadDir(root)
	if err != nil {
		slog.Error("读取图库目录失败，没能清理暂存", "path", root, "error", err)
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		deletion := strings.HasPrefix(name, ".delete-")
		if !entry.IsDir() || !deletion && !strings.HasPrefix(name, ".import-") {
			continue
		}
		dir := filepath.Join(root, name)
		if deletion {
			if err := restoreDeletion(ctx, root, dir, db); err != nil {
				slog.Error("挪回中断删除的文件失败，保留暂存目录", "path", dir, "error", err)
				continue
			}
		}
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("清理残留暂存目录失败", "path", dir, "error", err)
			continue
		}
		slog.Info("清理残留暂存目录", "path", dir)
	}
}

// restoreDeletion 把暂存里仍登记在库的文件挪回原处。文件名是「素材ID.original」、「素材ID.thumbnail.jpg」或「素材ID.thumbnail.png」，
// 稍早的版本缩略图只有 PNG，名字是「素材ID.thumbnail」；更老的版本用序号命名，只能按内容哈希认出原图，认不出的缩略图交给之后重建。
func restoreDeletion(ctx context.Context, root, dir string, db *sql.DB) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		id, kind, _ := strings.Cut(entry.Name(), ".")
		thumbnailExt := map[string]string{"thumbnail": ".png", "thumbnail.png": ".png", "thumbnail.jpg": ".jpg"}[kind]
		if kind != "original" && thumbnailExt == "" {
			if id, err = hashFile(path); err != nil {
				return err
			}
			kind = "original"
		}
		var relative string
		err := db.QueryRowContext(ctx, "SELECT relative_path FROM assets WHERE id=?", id).Scan(&relative)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if thumbnailExt != "" {
			relative = "thumbnails/" + id + thumbnailExt
		}
		target, err := packageDestination(root, relative)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(path, target); err != nil {
			return err
		}
		slog.Warn("挪回中断删除的文件", "id", id, "path", relative)
	}
	return nil
}

// 包内相对路径对应的位置。逐级检查上层目录，缺的就建，碰到符号链接或普通文件就拒绝，不会把文件写到图库包外面。
func packageDestination(root, relative string) (string, error) {
	if relative == "" || !filepath.IsLocal(filepath.FromSlash(relative)) {
		return "", i18n.NewError("error_asset_relative_path", relative)
	}
	current := root
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
	return filepath.Join(root, filepath.FromSlash(relative)), nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func connect(ctx context.Context, path string, create bool) (*sql.DB, error) {
	if create {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", databaseURI(path, "rw"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err = db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}

func databaseURI(path, mode string) string {
	path = filepath.ToSlash(path)
	if len(path) >= 2 && path[1] == ':' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) {
		// Windows 盘符属于路径，前面加 /，避免被 file URI 当成主机名。
		path = "/" + strings.ReplaceAll(path, "\\", "/")
	}
	u := url.URL{Scheme: "file", Path: path}
	q := u.Query()
	q.Set("mode", mode)
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "busy_timeout(5000)")
	u.RawQuery = q.Encode()
	return u.String()
}

func (l *Library) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.DB.Close()
}

// Edit 改图库的名称和图标，一起写回 manifest。
func (l *Library) Edit(name, icon string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	next := l.Manifest
	next.DisplayName, next.Icon, next.UpdatedAt = name, icon, time.Now().UTC()
	if err := writeJSON(filepath.Join(l.Root, "manifest.json"), next); err != nil {
		return err
	}
	l.Manifest = next
	slog.Info("编辑图库", "path", l.Root, "name", name, "icon", icon)
	return nil
}

func (l *Library) Missing() bool {
	if inTrash(l.Root) {
		return true
	}
	root, err := os.Stat(l.Root)
	if err != nil || !root.IsDir() {
		return true
	}
	manifest, err := os.Stat(filepath.Join(l.Root, "manifest.json"))
	return err != nil || manifest.IsDir()
}

func (l *Library) Delete(ctx context.Context, trash func(string) error) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	conn, err := l.DB.Conn(ctx)
	if err != nil {
		return err
	}
	// 先释放文件句柄；移动失败时仍保留可用的 DB，让当前图库继续工作。
	l.DB.SetMaxIdleConns(0)
	defer l.DB.SetMaxIdleConns(1)
	if err := conn.Close(); err != nil {
		return err
	}
	if err := trash(l.Root); err != nil {
		return err
	}
	if err := l.DB.Close(); err != nil {
		return err
	}
	slog.Info("图库移到系统回收站", "path", l.Root)
	return nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", i18n.NewError("error_name_required")
	}
	return name, nil
}

func inTrash(path string) bool {
	parts := strings.Split(strings.ReplaceAll(filepath.Clean(path), "\\", "/"), "/")
	for i, p := range parts {
		if p == ".Trash" || p == ".Trashes" || strings.EqualFold(p, "$Recycle.Bin") {
			return true
		}
		if strings.HasPrefix(p, ".Trash-") {
			if _, err := strconv.ParseUint(strings.TrimPrefix(p, ".Trash-"), 10, 32); err == nil {
				return true
			}
		}
		if p == "Trash" && i+1 < len(parts) && parts[i+1] == "files" {
			return true
		}
	}
	return false
}

func writeJSON(path string, value any) (err error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".momento-*.tmp")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer func() {
		if err != nil {
			if removeErr := os.Remove(temp); removeErr != nil && !os.IsNotExist(removeErr) {
				err = errors.Join(err, removeErr)
			}
		}
	}()
	if _, err = f.Write(append(data, '\n')); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
