package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func legacyLibrary(t *testing.T) *Library {
	t.Helper()
	root := filepath.Join(t.TempDir(), "旧图库.momento")
	for _, dir := range []string{"database", "assets", "thumbnails", ".drag-out"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	db, err := connect(t.Context(), filepath.Join(root, "database", "library.sqlite"), true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(t.Context(), schemaV1); err != nil {
		db.Close()
		t.Fatal(err)
	}
	now := time.Now().UTC()
	manifest := model.Manifest{SchemaVersion: 1, LibraryID: "old-library", DisplayName: "旧图库", CreatedAt: now, UpdatedAt: now}
	if err := writeJSON(filepath.Join(root, "manifest.json"), manifest); err != nil {
		db.Close()
		t.Fatal(err)
	}
	l := &Library{Root: root, DB: db, Manifest: manifest}
	t.Cleanup(func() { l.Close() })
	return l
}

func databaseVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	return version
}

func TestUpgradePreservesLegacyAssetsAndRelationships(t *testing.T) {
	old := legacyLibrary(t)
	seedAsset(t, old, "a")
	folder, err := old.CreateFolder(t.Context(), "旧文件夹", "")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := old.CreateTag(t.Context(), "旧标签")
	if err != nil {
		t.Fatal(err)
	}
	if err := old.SetAssetFolders(t.Context(), []string{"a"}, []string{folder.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := old.SetAssetTags(t.Context(), []string{"a"}, []string{tag.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := old.UpdateAssetDetails(t.Context(), "a", "旧备注", "https://example.com/original"); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, old)
	identity := old.Manifest.LibraryID
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(t.Context(), old.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	if upgraded.Manifest.SchemaVersion != 2 || upgraded.Manifest.LibraryID != identity || databaseVersion(t, upgraded.DB) != 2 {
		t.Fatal("版本或身份错误", upgraded.Manifest)
	}
	if got := snapshot(t, upgraded); !reflect.DeepEqual(got, before) {
		t.Fatal("升级改变了旧数据", got, before)
	}
	if _, err := upgraded.CreateSmartFolder(t.Context(), "蓝图", "all", []model.SmartRule{{Field: "name", Operator: "contains", Value: "素材"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved model.Manifest
	if err = json.Unmarshal(data, &saved); err != nil || saved.SchemaVersion != 2 {
		t.Fatal("未同步清单", err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := Open(t.Context(), old.Root)
	if err != nil {
		t.Fatal("重复打开不能升级两次", err)
	}
	defer again.Close()
	if len(snapshot(t, again).SmartFolders) != 1 {
		t.Fatal("再次打开丢失智能文件夹")
	}
}

func TestUpgradeFailureRollsBackDDLAndManifest(t *testing.T) {
	old := legacyLibrary(t)
	seedAsset(t, old, "a")
	if _, err := old.DB.Exec("CREATE TABLE quick_access(block_upgrade TEXT)"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), old.Root); err == nil {
		t.Fatal("冲突的旧库升级成功")
	}
	db, err := connect(t.Context(), filepath.Join(old.Root, "database", "library.sqlite"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if databaseVersion(t, db) != 1 {
		t.Fatal("失败升级改变了版本")
	}
	if _, err := db.Query("SELECT last_used_at FROM assets"); err == nil {
		t.Fatal("失败升级没有回滚新增列")
	}
	var name string
	if err := db.QueryRow("SELECT name FROM assets WHERE id='a'").Scan(&name); err != nil || name != "素材 a" {
		t.Fatal("旧数据受损", err)
	}
	after, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil || string(after) != string(before) {
		t.Fatal("失败升级修改了清单", err)
	}
}

func TestUpgradeManifestFailureRollsBackDatabase(t *testing.T) {
	old := legacyLibrary(t)
	seedAsset(t, old, "a")
	before := old.Manifest
	manifestPath := filepath.Join(old.Root, "manifest.json")
	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(manifestPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := upgrade(t.Context(), old.DB, old.Root, &old.Manifest); err == nil {
		t.Fatal("清单写入失败被忽略")
	}
	if old.Manifest != before || databaseVersion(t, old.DB) != 1 {
		t.Fatal("清单失败仍推进版本")
	}
	if _, err := old.DB.Query("SELECT rotation FROM assets"); err == nil {
		t.Fatal("清单失败没有回滚DDL")
	}
	if len(snapshot(t, old).Assets) != 1 {
		t.Fatal("清单失败丢失素材")
	}
	entries, err := os.ReadDir(old.Root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Fatal("失败升级残留清单临时文件")
		}
	}
}

func TestUpgradeRejectsBrokenOldTablesAndNewerVersions(t *testing.T) {
	for _, kind := range []string{"missing-table", "newer", "zero"} {
		t.Run(kind, func(t *testing.T) {
			old := legacyLibrary(t)
			switch kind {
			case "missing-table":
				_, _ = old.DB.Exec("DROP TABLE asset_tags")
			case "newer":
				_, _ = old.DB.Exec("PRAGMA user_version=99")
			case "zero":
				_, _ = old.DB.Exec("PRAGMA user_version=0")
			}
			old.Close()
			if _, err := Open(t.Context(), old.Root); err == nil {
				t.Fatal("损坏或未知结构被接受")
			}
		})
	}
}

func TestUpgradeRecoversManifestAheadOfDatabase(t *testing.T) {
	old := legacyLibrary(t)
	seedAsset(t, old, "a")
	old.Manifest.SchemaVersion = 2
	if err := writeJSON(filepath.Join(old.Root, "manifest.json"), old.Manifest); err != nil {
		t.Fatal(err)
	}
	old.Close()
	opened, err := Open(t.Context(), old.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if databaseVersion(t, opened.DB) != 2 || len(snapshot(t, opened).Assets) != 1 {
		t.Fatal("无法恢复中断的升级")
	}
}

func TestUpgradeCancellationLeavesLegacyData(t *testing.T) {
	old := legacyLibrary(t)
	seedAsset(t, old, "a")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := upgrade(ctx, old.DB, old.Root, &old.Manifest); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if databaseVersion(t, old.DB) != 1 || old.Manifest.SchemaVersion != 1 || len(snapshot(t, old).Assets) != 1 {
		t.Fatal("取消升级改动了旧数据")
	}
}

func TestImportLegacyLibraryUpgradesOnlyCopy(t *testing.T) {
	old := legacyLibrary(t)
	asset := addBackupAsset(t, old)
	before, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	imported, err := ImportLibrary(t.Context(), old.Root, filepath.Join(t.TempDir(), "Libraries"))
	if err != nil {
		t.Fatal(err)
	}
	defer imported.Close()
	if imported.Manifest.SchemaVersion != 2 || databaseVersion(t, imported.DB) != 2 || snapshot(t, imported).Assets[0].ID != asset.ID {
		t.Fatal("导入副本未升级")
	}
	after, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil || string(after) != string(before) || databaseVersion(t, old.DB) != 1 {
		t.Fatal("导入修改了旧来源", err)
	}
}

func TestUpgradeCommitFailureRestoresManifestAndRollsBack(t *testing.T) {
	old := legacyLibrary(t)
	seedAsset(t, old, "a")
	if _, err := old.DB.Exec("PRAGMA journal_mode=DELETE; PRAGMA busy_timeout=0"); err != nil {
		t.Fatal(err)
	}
	reader, err := sql.Open("sqlite", databaseURI(filepath.Join(old.Root, "database", "library.sqlite"), "ro"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	readTx, err := reader.BeginTx(t.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	var id string
	if err := readTx.QueryRow("SELECT id FROM assets").Scan(&id); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest := old.Manifest
	if err := upgrade(t.Context(), old.DB, old.Root, &old.Manifest); err == nil {
		t.Fatal("读锁没有阻止提交，未覆盖提交失败")
	}
	if old.Manifest != manifest || databaseVersion(t, old.DB) != 1 {
		t.Fatal("提交失败仍改变了版本")
	}
	if _, err := old.DB.Query("SELECT last_used_at FROM assets"); err == nil {
		t.Fatal("提交失败没有回滚新增列")
	}
	after, err := os.ReadFile(filepath.Join(old.Root, "manifest.json"))
	if err != nil || string(after) != string(before) {
		t.Fatal("提交失败没有恢复清单", err)
	}
	if len(snapshot(t, old).Assets) != 1 {
		t.Fatal("提交失败丢失旧素材")
	}
}
