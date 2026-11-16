package library

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	if err := src.copyPackage(ctx, destination, true, nil); err != nil {
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
// 导出的副本换一个新的图库 ID：沿用原 ID 的话，打开副本时原库的自动导入规则会在副本上生效，新图导进副本、源文件却被移到废纸篓。
func (l *Library) ExportLibrary(ctx context.Context, destination string) error {
	err := l.copyPackage(ctx, destination, true, nil)
	if err != nil {
		slog.Error("导出图库失败", "destination", destination, "error", err)
		return err
	}
	slog.Info("导出图库", "source", l.Root, "destination", destination)
	return nil
}

// ExportLibraryReplacing 和 ExportLibrary 一样，但 destination 已存在时（用户在保存对话框里确认了替换）
// 先把新包完整写好，再把旧目标挪开、放入新包，最后用 trash 把旧目标移到系统回收站。
// 放入新包失败时旧目标挪回原处；只有移到回收站失败时旧目标会留在报错里给出的位置，不会被删掉。
// 目标不能是当前图库本身，也不能包含当前图库。
func (l *Library) ExportLibraryReplacing(ctx context.Context, destination string, trash func(string) error) error {
	err := l.copyPackage(ctx, destination, true, trash)
	if err != nil {
		slog.Error("导出图库失败", "destination", destination, "replace", true, "error", err)
		return err
	}
	slog.Info("导出图库", "source", l.Root, "destination", destination, "replace", true)
	return nil
}

// trash 不为空时允许替换已存在的 destination，旧目标交给 trash 处理。
func (l *Library) copyPackage(ctx context.Context, destination string, newIdentity bool, trash func(string) error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if trash != nil {
		// 只解开上级目录的链接：目标本身是链接时替换的是这个链接，不是它指向的地方。
		parent, err := resolvePath(filepath.Dir(destination))
		if err != nil {
			return err
		}
		destination = filepath.Join(parent, filepath.Base(destination))
	} else if destination, err = resolvePath(destination); err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(l.Root)
	if err != nil {
		return err
	}
	if trash != nil && containsFile(destination, root) {
		return i18n.NewError("error_replace_current_library")
	}
	if containsFile(root, destination) {
		return i18n.NewError("error_library_destination")
	}
	if _, err := os.Lstat(destination); trash == nil && !os.IsNotExist(err) {
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
		// 缩略图可能是新的 .jpg 或旧的 .png，有哪个复制哪个，都没有就跳过。
		for _, ext := range []string{".jpg", ".png"} {
			thumb := "thumbnails/" + asset.ID + ext
			if err := copyLibraryFile(ctx, root, stage, thumb, "", -1); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := conn.Close(); err != nil {
		return err
	}
	conn = nil
	if _, err := os.Lstat(destination); err == nil && trash != nil {
		if err := replacePackage(stage, destination, trash); err != nil {
			return err
		}
		stage = ""
		return nil
	} else if !os.IsNotExist(err) {
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

// replacePackage 用写好的 stage 替换已存在的 destination：旧目标先挪进同目录的暂存文件夹（名字不变，
// 移到回收站后还是原名），新包改名到位，再把旧目标移到回收站。新包没放进去时旧目标挪回原处。
func replacePackage(stage, destination string, trash func(string) error) error {
	aside, err := os.MkdirTemp(filepath.Dir(destination), ".momento-replaced-")
	if err != nil {
		return err
	}
	old := filepath.Join(aside, filepath.Base(destination))
	if err := os.Rename(destination, old); err != nil {
		return errors.Join(err, os.Remove(aside))
	}
	if err := os.Rename(stage, destination); err != nil {
		if restoreErr := os.Rename(old, destination); restoreErr != nil {
			slog.Error("替换导出失败且旧目标没能挪回", "destination", destination, "kept", old, "error", restoreErr)
			return errors.Join(err, i18n.WrapError(restoreErr, "error_replace_restore_failed", old))
		}
		return errors.Join(err, os.Remove(aside))
	}
	if err := trash(old); err != nil {
		slog.Error("被替换的旧目标没能移到回收站", "kept", old, "error", err)
		return i18n.WrapError(err, "error_replace_kept_old", old)
	}
	slog.Info("替换导出目标，旧目标已移到回收站", "destination", destination)
	return os.Remove(aside)
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
	manifest, err := readManifest(root)
	if err != nil {
		return nil, err
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
	if version != manifest.SchemaVersion {
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
