package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func newTestLibrary(t *testing.T) *Library {
	t.Helper()
	l, err := Create(t.Context(), filepath.Join(t.TempDir(), "图库.momento"), "素材")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	return l
}

func seedAsset(t *testing.T, l *Library, id string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := l.DB.Exec("INSERT INTO assets(id,name,relative_path,created_at,updated_at) VALUES(?,?,?,?,?)", id, "素材 "+id, "assets/ab/"+id+".png", now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func snapshot(t *testing.T, l *Library) model.Snapshot {
	t.Helper()
	s, err := l.Snapshot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLibraryLifecycleAndPortablePaths(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "abc")
	originalRoot, originalID := l.Root, l.Manifest.LibraryID
	var journal string
	var foreignKeys int
	if err := l.DB.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil || journal != "wal" {
		t.Fatalf("WAL: %s %v", journal, err)
	}
	if err := l.DB.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("外键: %d %v", foreignKeys, err)
	}
	for _, dir := range []string{"database", "assets", "thumbnails", ".drag-out"} {
		if info, err := os.Stat(filepath.Join(l.Root, dir)); err != nil || !info.IsDir() {
			t.Fatalf("目录 %s: %v", dir, err)
		}
	}
	if err := l.Rename("  新名字  "); err != nil {
		t.Fatal(err)
	}
	if l.Root != originalRoot || l.Manifest.LibraryID != originalID {
		t.Fatal("重命名改变了图库位置或身份")
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), "已搬动.momento")
	if err := os.Rename(l.Root, moved); err != nil {
		t.Fatal(err)
	}
	opened, err := Open(t.Context(), moved)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if opened.Manifest.DisplayName != "新名字" {
		t.Fatal(opened.Manifest)
	}
	s := snapshot(t, opened)
	if len(s.Assets) != 1 || s.Assets[0].RelativePath != "assets/ab/abc.png" {
		t.Fatal(s.Assets)
	}
}

func TestCreateNeverOverwritesExistingData(t *testing.T) {
	root := filepath.Join(t.TempDir(), "已有.momento")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "important.txt")
	if err := os.WriteFile(file, []byte("保留"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(t.Context(), root, "替换"); err == nil {
		t.Fatal("覆盖了现有位置")
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "保留" {
		t.Fatalf("已有数据改变: %q %v", data, err)
	}
	if _, err := Create(t.Context(), filepath.Join(t.TempDir(), "空名.momento"), "  "); err == nil {
		t.Fatal("允许空名称")
	}
}

func TestCreateFailureRemovesOnlyNewPackage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "失败.momento")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Create(ctx, root, "失败"); err == nil {
		t.Fatal("取消操作成功了")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("失败后残留图库: %v", err)
	}
}

func TestOpenNeverCreatesDatabaseOrTables(t *testing.T) {
	l := newTestLibrary(t)
	if _, err := l.DB.Exec("DROP TABLE asset_tags"); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), l.Root); err == nil {
		t.Fatal("缺表仍成功打开")
	}
	db, err := sql.Open("sqlite", filepath.Join(l.Root, "database", "library.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='asset_tags'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("打开补建了表: %d %v", n, err)
	}
	missing := newTestLibrary(t)
	if err := missing.Close(); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(missing.Root, "database", "library.sqlite")
	if err := os.Remove(dbPath); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), missing.Root); err == nil {
		t.Fatal("没有数据库仍成功打开")
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("打开创建了数据库: %v", err)
	}
}

func TestManifestFailuresAndMissingDetection(t *testing.T) {
	for _, contents := range []string{`{"schemaVersion":1}`, `not json`, `{}`} {
		l := newTestLibrary(t)
		if err := os.WriteFile(filepath.Join(l.Root, "manifest.json"), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(t.Context(), l.Root); err == nil {
			t.Fatalf("无效清单 %q 被接受", contents)
		}
	}
	l := newTestLibrary(t)
	if l.Missing() {
		t.Fatal("有效图库被判丢失")
	}
	if err := os.Remove(filepath.Join(l.Root, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if !l.Missing() {
		t.Fatal("未检测缺失清单")
	}
	if err := os.RemoveAll(l.Root); err != nil {
		t.Fatal(err)
	}
	if !l.Missing() {
		t.Fatal("未检测缺失包")
	}
	trashLibrary := newTestLibrary(t)
	if err := trashLibrary.Close(); err != nil {
		t.Fatal(err)
	}
	trashDir := filepath.Join(t.TempDir(), ".Trash")
	if err := os.Mkdir(trashDir, 0700); err != nil {
		t.Fatal(err)
	}
	trashPath := filepath.Join(trashDir, "已删除.momento")
	if err := os.Rename(trashLibrary.Root, trashPath); err != nil {
		t.Fatal(err)
	}
	trashLibrary.Root = trashPath
	if !trashLibrary.Missing() {
		t.Fatal("未检测系统回收站")
	}
	if _, err := Open(t.Context(), trashPath); err == nil {
		t.Fatal("打开了系统回收站内图库")
	}
}

func TestRenameFailurePreservesManifest(t *testing.T) {
	l := newTestLibrary(t)
	before := l.Manifest
	if err := l.Rename(""); err == nil {
		t.Fatal("允许空名称")
	}
	if l.Manifest != before {
		t.Fatal("失败改变了内存清单")
	}
	path := filepath.Join(l.Root, "manifest.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := l.Rename("不能写"); err == nil {
		t.Fatal("写入失败被吞掉")
	}
	if l.Manifest != before {
		t.Fatal("写入失败改变了内存清单")
	}
	entries, err := os.ReadDir(l.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatal("残留临时清单")
		}
	}
}

func TestDeleteUsesTrashCallback(t *testing.T) {
	l := newTestLibrary(t)
	called := false
	trashPath := filepath.Join(t.TempDir(), "回收站.momento")
	if err := l.Delete(t.Context(), func(path string) error { called = true; return os.Rename(path, trashPath) }); err != nil {
		t.Fatal(err)
	}
	if !called || !l.Missing() {
		t.Fatal("未通过回收站回调移动图库")
	}
	if _, err := os.Stat(filepath.Join(trashPath, "database", "library.sqlite")); err != nil {
		t.Fatal("图库被永久删除", err)
	}
}

func TestDeleteFailureLeavesCurrentLibraryUsable(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	want := errors.New("系统拒绝移动")
	if err := l.Delete(t.Context(), func(string) error { return want }); !errors.Is(err, want) {
		t.Fatal(err)
	}
	if l.Missing() || len(snapshot(t, l).Assets) != 1 {
		t.Fatal("删除失败后当前图库不能继续使用")
	}
	if err := l.RenameAsset(t.Context(), "a", "删除失败后继续编辑"); err != nil {
		t.Fatal(err)
	}
}

func TestFolderTreeMoveDeleteAndMemberships(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	a, err := l.CreateFolder(t.Context(), "A", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := l.CreateFolder(t.Context(), "B", "")
	if err != nil {
		t.Fatal(err)
	}
	c, err := l.CreateFolder(t.Context(), "C", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	d, err := l.CreateFolder(t.Context(), "D", c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := l.MoveFolder(t.Context(), a.ID, d.ID, 0); err == nil {
		t.Fatal("允许移入子孙")
	}
	if err := l.MoveFolder(t.Context(), a.ID, a.ID, 0); err == nil {
		t.Fatal("允许移入自己")
	}
	if err := l.MoveFolder(t.Context(), b.ID, "", 0); err != nil {
		t.Fatal(err)
	}
	s := snapshot(t, l)
	roots := []model.Folder{}
	for _, f := range s.Folders {
		if f.ParentID == "" {
			roots = append(roots, f)
		}
	}
	if len(roots) != 2 || roots[0].ID != b.ID || roots[0].SortIndex != 0 || roots[1].SortIndex != 1 {
		t.Fatal(roots)
	}
	if err := l.MoveFolder(t.Context(), c.ID, b.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := l.RenameFolder(t.Context(), c.ID, "改名"); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetFolders(t.Context(), []string{"a"}, []string{c.ID, d.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetFolders(t.Context(), []string{"a"}, []string{c.ID}, true); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l).Assets[0].FolderIDs; len(got) != 2 {
		t.Fatal(got)
	}
	if err := l.SetAssetFolders(t.Context(), []string{"a"}, []string{c.ID}, false); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l).Assets[0].FolderIDs; !slices.Equal(got, []string{d.ID}) {
		t.Fatal(got)
	}
	if err := l.DeleteFolder(t.Context(), b.ID); err != nil {
		t.Fatal(err)
	}
	s = snapshot(t, l)
	if len(s.Folders) != 1 || s.Folders[0].ID != a.ID || len(s.Assets) != 1 || len(s.Assets[0].FolderIDs) != 0 {
		t.Fatal(s)
	}
}

func TestInvalidFolderParentAndBatchRelationsRollback(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	f, err := l.CreateFolder(t.Context(), "F", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.CreateFolder(t.Context(), "无效", "missing"); err == nil {
		t.Fatal("允许不存在的父级")
	}
	if err := l.MoveFolder(t.Context(), f.ID, "missing", 0); err == nil {
		t.Fatal("移动到不存在父级")
	}
	if err := l.SetAssetFolders(t.Context(), []string{"a", "missing"}, []string{f.ID}, true); err == nil {
		t.Fatal("无效批量加入成功")
	}
	s := snapshot(t, l)
	if s.Folders[0].ParentID != "" || len(s.Assets[0].FolderIDs) != 0 {
		t.Fatal("失败未回滚", s)
	}
	if err := l.SetFavorite(t.Context(), []string{"a", "missing"}, true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if snapshot(t, l).Assets[0].Favorite {
		t.Fatal("批量收藏失败未回滚")
	}
}

func TestTagDedupeRenameDeleteAndBatchEdit(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	seedAsset(t, l, "b")
	first, err := l.CreateTag(t.Context(), "  ＵＩ   Design  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := l.CreateTag(t.Context(), "ui design")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.NormalizedName != "ui design" {
		t.Fatal(first, second)
	}
	other, err := l.CreateTag(t.Context(), "Other")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.RenameTag(t.Context(), other.ID, "UI Design"); err == nil {
		t.Fatal("重名覆盖了已有标签")
	}
	if err := l.RenameTag(t.Context(), first.ID, "界面"); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetTags(t.Context(), []string{"a", "b"}, []string{first.ID, other.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetTags(t.Context(), []string{"a"}, []string{other.ID}, false); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetTags(t.Context(), []string{"a"}, []string{other.ID, "missing"}, true); err == nil {
		t.Fatal("不存在标签未报错")
	}
	if got := snapshot(t, l).Assets[0].TagIDs; !slices.Equal(got, []string{first.ID}) {
		t.Fatal("关系失败未回滚", got)
	}
	if err := l.SetFavorite(t.Context(), []string{"a", "b"}, true); err != nil {
		t.Fatal(err)
	}
	if err := l.RenameAsset(t.Context(), "a", "新标题"); err != nil {
		t.Fatal(err)
	}
	if err := l.DeleteTag(t.Context(), first.ID); err != nil {
		t.Fatal(err)
	}
	s := snapshot(t, l)
	if len(s.Tags) != 1 || s.Tags[0].ID != other.ID || len(s.Assets[0].TagIDs) != 0 || !s.Assets[0].Favorite || s.Assets[0].Name != "新标题" {
		t.Fatal(s)
	}
	if len(s.Assets[1].TagIDs) != 1 {
		t.Fatal(s.Assets[1])
	}
}

func TestRelativePathConstraintAndSnapshotMetadata(t *testing.T) {
	l := newTestLibrary(t)
	for i, path := range []string{"/tmp/private.png", "../private.png", "assets/../../private.png", `C:\private.png`, `\\server\private.png`} {
		_, err := l.DB.Exec("INSERT INTO assets(id,relative_path,created_at,updated_at) VALUES(?,?,?,?)", fmt.Sprint(i), path, time.Now().Format(time.RFC3339Nano), time.Now().Format(time.RFC3339Nano))
		if err == nil {
			t.Fatalf("绝对或逃逸路径被接受: %s", path)
		}
	}
	seedAsset(t, l, "a")
	exif, err := json.Marshal(map[string]string{"camera": "Camera"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.DB.Exec("UPDATE assets SET exif_json=?,width=2400,height=1600,source_url='https://example.com',trashed=1 WHERE id='a'", string(exif)); err != nil {
		t.Fatal(err)
	}
	if _, err := l.DB.Exec("INSERT INTO asset_colors(asset_id,hex,proportion,family,sort_index) VALUES('a','#AABBCC',1,'blue',0)"); err != nil {
		t.Fatal(err)
	}
	a := snapshot(t, l).Assets[0]
	if a.EXIF["camera"] != "Camera" || a.Width != 2400 || a.Height != 1600 || a.SourceURL != "https://example.com" || !a.Trashed || len(a.Colors) != 1 || a.Colors[0].Family != "blue" {
		t.Fatal(a)
	}
}

func TestTrashLocations(t *testing.T) {
	for _, path := range []string{"/Users/u/.Trash/a.momento", "/Volumes/Disk/.Trashes/501/a.momento", `C:\$RECYCLE.BIN\user\a.momento`, "/home/u/.local/share/Trash/files/a.momento", "/media/disk/.Trash-1000/files/a.momento"} {
		if !inTrash(path) {
			t.Errorf("没有识别系统回收站：%s", path)
		}
	}
	if inTrash("/home/u/.Trash-design/a.momento") {
		t.Fatal("普通目录被当作系统回收站")
	}
}

func TestDatabaseURIHasNoWindowsAuthority(t *testing.T) {
	for _, test := range []struct{ path, wantPath string }{
		{`C:\Users\A B\素材.momento\database\library.sqlite`, "/C:/Users/A B/素材.momento/database/library.sqlite"},
		{"C:/Users/A B/素材.momento/database/library.sqlite", "/C:/Users/A B/素材.momento/database/library.sqlite"},
		{"/tmp/图库 #1?.momento/database/library.sqlite", "/tmp/图库 #1?.momento/database/library.sqlite"},
	} {
		u, err := url.Parse(databaseURI(test.path, "rw"))
		if err != nil {
			t.Fatal(err)
		}
		if u.Scheme != "file" || u.Host != "" || u.Path != test.wantPath {
			t.Fatalf("%q -> %s", test.path, u)
		}
		if u.Query().Get("mode") != "rw" || !slices.Contains(u.Query()["_pragma"], "foreign_keys(1)") {
			t.Fatal(u.RawQuery)
		}
	}
}
