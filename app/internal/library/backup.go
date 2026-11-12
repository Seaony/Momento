package library

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Seaony/Momento/app/internal/i18n"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/google/uuid"
)

// destinationDir 是应用数据目录中的 Libraries 目录。
func ImportLibrary(ctx context.Context, source, destinationDir string) (_ *Library, err error) {
	slog.Info("开始导入图库", "source", source, "destinationDirectory", destinationDir)
	defer func() {
		if err != nil {
			slog.Error("导入图库失败", "source", source, "error", err)
		}
	}()
	if err := ValidateWorkingPath(destinationDir); err != nil {
		return nil, err
	}
	src, err := openCopySource(ctx, source)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, src.Close()) }()
	destinationDir, err = resolvePath(destinationDir)
	if err != nil {
		return nil, err
	}
	if withinPath(src.Root, destinationDir) {
		return nil, i18n.NewError("error_library_destination")
	}
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return nil, err
	}
	base := libraryFileName(src.Manifest.DisplayName)
	destination := filepath.Join(destinationDir, base+".momento")
	for i := 2; ; i++ {
		if _, err := os.Lstat(destination); os.IsNotExist(err) {
			break
		} else if err != nil {
			return nil, err
		}
		destination = filepath.Join(destinationDir, fmt.Sprintf("%s %d.momento", base, i))
	}
	if err := src.copyPackage(ctx, destination, true); err != nil {
		return nil, err
	}
	result, err := Open(ctx, destination)
	if err != nil {
		return nil, errors.Join(err, os.RemoveAll(destination))
	}
	slog.Info("导入图库", "source", src.Root, "destination", destination)
	return result, nil
}

// destination 是新目录包的完整路径，已有位置不会被覆盖。
func (l *Library) ExportLibrary(ctx context.Context, destination string) error {
	err := l.copyPackage(ctx, destination, false)
	if err != nil {
		slog.Error("导出图库失败", "destination", destination, "error", err)
		return err
	}
	slog.Info("导出图库", "source", l.Root, "destination", destination)
	return nil
}

func (l *Library) copyPackage(ctx context.Context, destination string, newIdentity bool) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	destination, err = resolvePath(destination)
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(l.Root)
	if err != nil {
		return err
	}
	if withinPath(root, destination) {
		return i18n.NewError("error_library_destination")
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		if err == nil {
			return i18n.NewError("error_destination_exists", destination)
		}
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(destination), ".momento-copy-")
	if err != nil {
		return err
	}
	defer func() {
		if stage != "" {
			err = errors.Join(err, os.RemoveAll(stage))
		}
	}()
	for _, dir := range []string{"database", "assets", "thumbnails", ".drag-out"} {
		if err := os.Mkdir(filepath.Join(stage, dir), 0755); err != nil {
			return err
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	conn, err := l.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if conn != nil {
			err = errors.Join(err, conn.Close())
		}
	}()
	// SQLite 生成包含 WAL 已提交内容的一致快照；连接保留到原图复制完成。
	databasePath := filepath.Join(stage, "database", "library.sqlite")
	if _, err := conn.ExecContext(ctx, "VACUUM INTO ?", databasePath); err != nil {
		return err
	}
	manifest := l.Manifest
	if newIdentity {
		manifest.LibraryID = uuid.NewString()
		manifest.CreatedAt, manifest.UpdatedAt = time.Now().UTC(), time.Now().UTC()
	}
	if err := writeJSON(filepath.Join(stage, "manifest.json"), manifest); err != nil {
		return err
	}
	snapshot, err := validateCopy(ctx, stage)
	if err != nil {
		return err
	}
	for _, asset := range snapshot.Assets {
		hash, err := hex.DecodeString(asset.ID)
		if err != nil || len(hash) != sha256.Size || hex.EncodeToString(hash) != asset.ID {
			return i18n.NewError("error_asset_id", asset.ID)
		}
		if !strings.HasPrefix(asset.RelativePath, "assets/") {
			return i18n.NewError("error_asset_path", asset.RelativePath)
		}
		if err := copyLibraryFile(ctx, root, stage, asset.RelativePath, asset.ID, asset.Size); err != nil {
			return err
		}
		thumb := filepath.ToSlash(filepath.Join("thumbnails", asset.ID+".png"))
		if err := copyLibraryFile(ctx, root, stage, thumb, "", -1); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := conn.Close(); err != nil {
		return err
	}
	conn = nil
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		if err == nil {
			return i18n.NewError("error_destination_exists", destination)
		}
		return err
	}
	if err := os.Rename(stage, destination); err != nil {
		return err
	}
	stage = ""
	return nil
}

func openCopySource(ctx context.Context, path string) (*Library, error) {
	root, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var manifest model.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, i18n.WrapError(err, "error_manifest_invalid")
	}
	if manifest.SchemaVersion != model.SchemaVersion || manifest.LibraryID == "" || strings.TrimSpace(manifest.DisplayName) == "" || manifest.CreatedAt.IsZero() || manifest.UpdatedAt.IsZero() {
		return nil, i18n.NewError("error_manifest_invalid")
	}
	db, err := sql.Open("sqlite", databaseURI(filepath.Join(root, "database", "library.sqlite"), "ro"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	if version != model.SchemaVersion {
		return nil, errors.Join(i18n.NewError("error_database_invalid"), db.Close())
	}
	return &Library{Root: root, DB: db, Manifest: manifest}, nil
}

func validateCopy(ctx context.Context, root string) (snapshot model.Snapshot, err error) {
	l, err := openCopySource(ctx, root)
	if err != nil {
		return model.Snapshot{}, err
	}
	defer func() { err = errors.Join(err, l.Close()) }()
	var check string
	if err := l.DB.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&check); err != nil {
		return model.Snapshot{}, err
	}
	if check != "ok" {
		return model.Snapshot{}, i18n.NewError("error_database_check", check)
	}
	rows, err := l.DB.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return model.Snapshot{}, err
	}
	broken := rows.Next()
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		return model.Snapshot{}, err
	}
	if broken {
		return model.Snapshot{}, i18n.NewError("error_database_relations")
	}
	return l.Snapshot(ctx)
}

func copyLibraryFile(ctx context.Context, sourceRoot, destinationRoot, relative, hash string, size int64) (err error) {
	if !filepath.IsLocal(filepath.FromSlash(relative)) || strings.ContainsAny(relative, "\\:") {
		return i18n.NewError("error_asset_path", relative)
	}
	source, err := filepath.EvalSymlinks(filepath.Join(sourceRoot, filepath.FromSlash(relative)))
	if err != nil {
		return err
	}
	if !withinPath(sourceRoot, source) {
		return i18n.NewError("error_asset_path", relative)
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, in.Close()) }()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return i18n.NewError("error_asset_file", relative)
	}
	destination := filepath.Join(destinationRoot, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, out.Close()) }()
	digest := sha256.New()
	n, err := io.Copy(io.MultiWriter(out, digest), libraryCopyReader{ctx: ctx, r: in})
	if err != nil {
		return err
	}
	if (hash != "" && hex.EncodeToString(digest.Sum(nil)) != hash) || (size >= 0 && n != size) {
		return i18n.NewError("error_asset_checksum", relative)
	}
	return out.Sync()
}

type libraryCopyReader struct {
	ctx context.Context
	r   io.Reader
}

func (r libraryCopyReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

func libraryFileName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return ' '
		}
		return r
	}, name)
	name = strings.Trim(strings.Join(strings.Fields(name), " "), ". ")
	if name == "" {
		return "Library"
	}
	upper := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL" || (len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9') {
		name = "_" + name
	}
	return name
}
