package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func addBackupAsset(t *testing.T, l *Library) model.Asset {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	img.Set(2, 3, color.NRGBA{R: 210, G: 90, B: 40, A: 255})
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data.Bytes())
	id := hex.EncodeToString(digest[:])
	rel := filepath.ToSlash(filepath.Join("assets", id[:2], id+".png"))
	if err := os.MkdirAll(filepath.Dir(filepath.Join(l.Root, filepath.FromSlash(rel))), 0755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(l.Root, filepath.FromSlash(rel)), filepath.Join(l.Root, "thumbnails", id+".png")} {
		if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := l.DB.Exec(`INSERT INTO assets(id,name,original_file_name,extension,media_type,size,width,height,exif_json,source_url,relative_path,favorite,trashed,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, "改过的标题", "original.png", "png", "image/png", data.Len(), 8, 6, `{"camera":"camera"}`, "https://example.com/source", rel, true, true, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := l.DB.Exec("INSERT INTO asset_colors(asset_id,hex,proportion,family,sort_index) VALUES(?,?,?,?,?)", id, "#D25A28", 1, "orange", 0); err != nil {
		t.Fatal(err)
	}
	folder, err := l.CreateFolder(t.Context(), "参考", "")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := l.CreateTag(t.Context(), "标签")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetFolders(t.Context(), []string{id}, []string{folder.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetTags(t.Context(), []string{id}, []string{tag.ID}, true); err != nil {
		t.Fatal(err)
	}
	return snapshot(t, l).Assets[0]
}

func assertNoCopyStage(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".momento-copy-") {
			t.Fatalf("失败后残留暂存目录：%s", entry.Name())
		}
	}
}

func TestExportLibraryCopiesLiveWALAndRemainsPortable(t *testing.T) {
	l := newTestLibrary(t)
	if _, err := l.DB.Exec("PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		t.Fatal(err)
	}
	asset := addBackupAsset(t, l)
	before := snapshot(t, l)
	wal, err := os.Stat(filepath.Join(l.Root, "database", "library.sqlite-wal"))
	if err != nil || wal.Size() == 0 {
		t.Fatalf("测试没有覆盖活跃 WAL：%v", err)
	}
	if err := os.WriteFile(filepath.Join(l.Root, ".drag-out", "temporary.png"), []byte("temp"), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "导出.momento")
	if err := l.ExportLibrary(t.Context(), destination); err != nil {
		t.Fatal(err)
	}
	backup, err := Open(t.Context(), destination)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, backup); !reflect.DeepEqual(got, before) {
		t.Fatalf("导出数据不一致：%+v / %+v", got, before)
	}
	if backup.Manifest != l.Manifest {
		t.Fatal("导出备份改变了清单")
	}
	original, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(asset.RelativePath)))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(original)
	if hex.EncodeToString(digest[:]) != asset.ID {
		t.Fatal("原图内容改变")
	}
	entries, err := os.ReadDir(filepath.Join(destination, ".drag-out"))
	if err != nil || len(entries) != 0 {
		t.Fatal("导出了临时拖动文件", entries, err)
	}
	if err := backup.Close(); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), "移动后.momento")
	if err := os.Rename(destination, moved); err != nil {
		t.Fatal(err)
	}
	backup, err = Open(t.Context(), moved)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if got := snapshot(t, backup); !reflect.DeepEqual(got, before) {
		t.Fatal("移动后数据改变")
	}
	if _, err := os.Stat(filepath.Join(moved, filepath.FromSlash(asset.RelativePath))); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l); !reflect.DeepEqual(got, before) {
		t.Fatal("导出改变了来源图库")
	}
}

func TestImportLibraryUsesUniqueNamesAndNewIdentity(t *testing.T) {
	l := newTestLibrary(t)
	addBackupAsset(t, l)
	before, err := os.ReadFile(filepath.Join(l.Root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	destinationDir := filepath.Join(t.TempDir(), "Libraries")
	first, err := ImportLibrary(t.Context(), l.Root, destinationDir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := ImportLibrary(t.Context(), l.Root, destinationDir)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if filepath.Base(first.Root) != "素材.momento" || filepath.Base(second.Root) != "素材 2.momento" {
		t.Fatal(first.Root, second.Root)
	}
	if first.Manifest.LibraryID == l.Manifest.LibraryID || second.Manifest.LibraryID == first.Manifest.LibraryID {
		t.Fatal("导入副本没有独立身份")
	}
	if got, want := snapshot(t, first), snapshot(t, l); !reflect.DeepEqual(got, want) {
		t.Fatal("导入丢失了数据", got, want)
	}
	after, err := os.ReadFile(filepath.Join(l.Root, "manifest.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("导入改变了来源清单")
	}
	assertNoCopyStage(t, destinationDir)
}

func TestLibraryCopyFailurePreservesSourceAndExistingTarget(t *testing.T) {
	l := newTestLibrary(t)
	asset := addBackupAsset(t, l)
	dir := t.TempDir()
	existing := filepath.Join(dir, "已有.momento")
	if err := os.Mkdir(existing, 0755); err != nil {
		t.Fatal(err)
	}
	important := filepath.Join(existing, "important.txt")
	if err := os.WriteFile(important, []byte("必须保留"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.ExportLibrary(t.Context(), existing); err == nil {
		t.Fatal("覆盖了已有目标")
	}
	if got, err := os.ReadFile(important); err != nil || string(got) != "必须保留" {
		t.Fatal("目标内容改变", err)
	}
	if err := l.ExportLibrary(t.Context(), filepath.Join(l.Root, "inside.momento")); err == nil {
		t.Fatal("允许导出到来源内部")
	}
	if _, err := ImportLibrary(t.Context(), l.Root, filepath.Join(l.Root, "Libraries")); err == nil {
		t.Fatal("允许导入到来源内部")
	}
	path := filepath.Join(l.Root, filepath.FromSlash(asset.RelativePath))
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	failed := filepath.Join(dir, "失败.momento")
	if err := l.ExportLibrary(t.Context(), failed); err == nil {
		t.Fatal("没有检测损坏原图")
	}
	if _, err := os.Lstat(failed); !os.IsNotExist(err) {
		t.Fatal("失败后发布了目标图库", err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "corrupt" {
		t.Fatal("导出失败改变了来源文件", err)
	}
	if len(snapshot(t, l).Assets) != 1 {
		t.Fatal("导出失败改变了来源数据库")
	}
	assertNoCopyStage(t, dir)
}

func TestCopyCancellationMissingFilesAndInvalidDatabase(t *testing.T) {
	l := newTestLibrary(t)
	asset := addBackupAsset(t, l)
	dir := t.TempDir()
	destination := filepath.Join(dir, "取消.momento")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := l.ExportLibrary(ctx, destination); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("取消后创建了目标")
	}
	if err := os.Remove(filepath.Join(l.Root, filepath.FromSlash(asset.RelativePath))); err != nil {
		t.Fatal(err)
	}
	if err := l.ExportLibrary(t.Context(), destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("缺失原图错误被吞掉：%v", err)
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("缺失原图仍发布了图库")
	}
	if _, err := l.DB.Exec("DROP TABLE asset_tags"); err != nil {
		t.Fatal(err)
	}
	destinationDir := filepath.Join(dir, "Libraries")
	if _, err := ImportLibrary(t.Context(), l.Root, destinationDir); err == nil {
		t.Fatal("导入了缺表图库")
	}
	entries, err := os.ReadDir(destinationDir)
	if err != nil || len(entries) != 0 {
		t.Fatal("导入失败残留目标", entries, err)
	}
	assertNoCopyStage(t, dir)
}

func TestLibraryCopyRejectsExternalSymlink(t *testing.T) {
	l := newTestLibrary(t)
	asset := addBackupAsset(t, l)
	original := filepath.Join(l.Root, filepath.FromSlash(asset.RelativePath))
	outside := filepath.Join(t.TempDir(), "outside.png")
	if err := os.Rename(original, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, original); err != nil {
		t.Skipf("本机无法建立符号链接：%v", err)
	}
	dir := t.TempDir()
	if err := l.ExportLibrary(t.Context(), filepath.Join(dir, "逃逸.momento")); err == nil {
		t.Fatal("导出了库外文件")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("改变了库外文件", err)
	}
	assertNoCopyStage(t, dir)
}

func TestClearCachePreservesOriginalsAndMetadata(t *testing.T) {
	l := newTestLibrary(t)
	asset := addBackupAsset(t, l)
	before := snapshot(t, l)
	if err := os.Mkdir(filepath.Join(l.Root, ".drag-out", "session"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.Root, ".drag-out", "session", "image.png"), []byte("temp"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := l.ClearCache(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(l.Root, "thumbnails", asset.ID+".png")); err != nil {
		t.Fatal("取消清理仍删除了缓存", err)
	}
	if err := l.ClearCache(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"thumbnails", ".drag-out"} {
		entries, err := os.ReadDir(filepath.Join(l.Root, dir))
		if err != nil || len(entries) != 0 {
			t.Fatal("未清理缓存", dir, entries, err)
		}
	}
	if _, err := os.Stat(filepath.Join(l.Root, filepath.FromSlash(asset.RelativePath))); err != nil {
		t.Fatal("清缓存删除了原图", err)
	}
	if got := snapshot(t, l); !reflect.DeepEqual(got, before) {
		t.Fatal("清缓存改变了元数据")
	}
	outside := t.TempDir()
	protected := filepath.Join(outside, "protected.txt")
	if err := os.WriteFile(protected, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(l.Root, "thumbnails")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(l.Root, "thumbnails")); err != nil {
		t.Skipf("本机无法建立符号链接：%v", err)
	}
	if err := l.ClearCache(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(protected); err != nil || string(got) != "keep" {
		t.Fatal("清缓存删到了库外文件", err)
	}
}

func TestWorkingPathRejectsICloudIncludingSymlinks(t *testing.T) {
	home := t.TempDir()
	cloud := filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs")
	if err := os.MkdirAll(cloud, 0755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(cloud, "new.momento"), filepath.Join(cloud, "nested", "new.momento"), strings.ToUpper(filepath.Join(cloud, "new.momento"))} {
		if err := validateWorkingPath(path, home); !errors.Is(err, ErrCloudLibrary) {
			t.Fatalf("未拒绝 iCloud 路径：%s：%v", path, err)
		}
	}
	if err := validateWorkingPath(filepath.Join(home, "Pictures", "new.momento"), home); err != nil {
		t.Fatal("拒绝了本地目录", err)
	}
	link := filepath.Join(home, "Cloud Link")
	if err := os.Symlink(cloud, link); err != nil {
		t.Skipf("本机无法建立符号链接：%v", err)
	}
	if err := validateWorkingPath(filepath.Join(link, "new.momento"), home); !errors.Is(err, ErrCloudLibrary) {
		t.Fatal("符号链接绕过了工作路径限制", err)
	}
}

func TestCloudLocationsCanBeCopySourcesAndExportDestinations(t *testing.T) {
	l := newTestLibrary(t)
	addBackupAsset(t, l)
	home := t.TempDir()
	t.Setenv("HOME", home)
	cloud := filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs")
	if err := os.MkdirAll(cloud, 0755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(cloud, "备份.momento")
	if err := l.ExportLibrary(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	imported, err := ImportLibrary(t.Context(), source, filepath.Join(t.TempDir(), "Libraries"))
	if err != nil {
		t.Fatal(err)
	}
	defer imported.Close()
	if got, want := snapshot(t, imported), snapshot(t, l); !reflect.DeepEqual(got, want) {
		t.Fatal("从备份导入丢失数据")
	}
}

func TestImportLibraryAllowsMissingRegenerableThumbnails(t *testing.T) {
	l := newTestLibrary(t)
	asset := addBackupAsset(t, l)
	if err := os.Remove(filepath.Join(l.Root, "thumbnails", asset.ID+".png")); err != nil {
		t.Fatal(err)
	}
	imported, err := ImportLibrary(t.Context(), l.Root, filepath.Join(t.TempDir(), "Libraries"))
	if err != nil {
		t.Fatal(err)
	}
	defer imported.Close()
	if _, err := os.Stat(filepath.Join(imported.Root, filepath.FromSlash(asset.RelativePath))); err != nil {
		t.Fatal("缺缓存时丢失了原图", err)
	}
	if got, want := snapshot(t, imported), snapshot(t, l); !reflect.DeepEqual(got, want) {
		t.Fatal("缺缓存时改变了素材记录")
	}
}
