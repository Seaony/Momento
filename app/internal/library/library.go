package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/Seaony/Momento/app/internal/i18n"
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

	slog.Info("打开图库", "path", root, "libraryID", manifest.LibraryID)
	return &Library{Root: root, DB: db, Manifest: manifest}, nil
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

func (l *Library) Rename(name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	next := l.Manifest
	next.DisplayName, next.UpdatedAt = name, time.Now().UTC()
	if err := writeJSON(filepath.Join(l.Root, "manifest.json"), next); err != nil {
		return err
	}
	l.Manifest = next
	slog.Info("重命名图库", "path", l.Root, "name", name)
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
