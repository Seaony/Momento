package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

func autoImportTestLibrary(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Lib.momento")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAutoImportRulesSkipFinderCopyOfLibrary(t *testing.T) {
	original, copied := autoImportTestLibrary(t), autoImportTestLibrary(t)
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.AutoImport["same-id"] = []AutoImportRule{{Path: "/incoming", Enabled: true, Library: original}}
	a.Library = &library.Library{Root: copied, Manifest: model.Manifest{LibraryID: "same-id"}}
	if rules := a.bindAutoImportRules("same-id"); len(rules) != 0 {
		t.Fatalf("Finder 复制出来的副本也在监视原库的文件夹：%+v", rules)
	}
	if a.Settings.AutoImportErrors["/incoming"] == "" {
		t.Fatal("副本不监视时，设置里没有说明原因")
	}
	// 回到原库：规则照常生效，副本留下的说明清掉。
	a.Library = &library.Library{Root: original, Manifest: model.Manifest{LibraryID: "same-id"}}
	if rules := a.bindAutoImportRules("same-id"); len(rules) != 1 {
		t.Fatalf("原库的规则失效：%+v", rules)
	}
	if a.Settings.AutoImportErrors["/incoming"] != "" {
		t.Fatal("回到原库后还显示副本说明")
	}
}

func TestAutoImportRulesFollowMovedLibrary(t *testing.T) {
	moved := autoImportTestLibrary(t)
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.AutoImport["id"] = []AutoImportRule{{Path: "/old-rule", Enabled: true, Library: filepath.Join(t.TempDir(), "gone.momento")}, {Path: "/legacy", Enabled: true}}
	a.Library = &library.Library{Root: moved, Manifest: model.Manifest{LibraryID: "id"}}
	rules := a.bindAutoImportRules("id")
	if len(rules) != 2 {
		t.Fatalf("图库移动后规则失效：%+v", rules)
	}
	for _, rule := range a.Preferences.AutoImport["id"] {
		if rule.Library != moved {
			t.Fatalf("规则没有改记到图库的新位置：%+v", rule)
		}
	}
}

func TestDeletedFolderTargetFallsBackToLibraryRoot(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Library = &library.Library{Root: t.TempDir(), Manifest: model.Manifest{LibraryID: "id"}}
	a.Preferences.AutoImport["id"] = []AutoImportRule{{Path: "/a", FolderID: "kept"}, {Path: "/b", FolderID: "deleted"}}
	a.applySnapshot(model.Snapshot{Folders: []model.Folder{{ID: "kept", Name: "Kept"}}})
	rules := a.Preferences.AutoImport["id"]
	if rules[0].FolderID != "kept" || rules[1].FolderID != "" {
		t.Fatalf("目标文件夹被删后规则没有改回根目录：%+v", rules)
	}
}

// 原库还在、只是读不到（没权限）时不能当成移动，副本不能接管监视。
func TestAutoImportRulesUnreadableOwnerIsNotTreatedAsMoved(t *testing.T) {
	parent := t.TempDir()
	original := filepath.Join(parent, "locked", "Lib.momento")
	if err := os.MkdirAll(original, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(parent, "locked"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(filepath.Join(parent, "locked"), 0755) })
	if _, err := os.Stat(filepath.Join(original, "manifest.json")); err == nil || os.IsNotExist(err) {
		t.Skip("当前环境无法模拟读不到的目录")
	}
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.AutoImport["id"] = []AutoImportRule{{Path: "/incoming", Enabled: true, Library: original}}
	a.Library = &library.Library{Root: autoImportTestLibrary(t), Manifest: model.Manifest{LibraryID: "id"}}
	if rules := a.bindAutoImportRules("id"); len(rules) != 0 || a.Preferences.AutoImport["id"][0].Library != original {
		t.Fatalf("读不到原库时副本接管了规则：%+v", a.Preferences.AutoImport["id"])
	}
}

// 在副本里删了文件夹，不能把原库规则的目标改掉。
func TestDeletedFolderInCopyKeepsOriginalRule(t *testing.T) {
	original := autoImportTestLibrary(t)
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Library = &library.Library{Root: autoImportTestLibrary(t), Manifest: model.Manifest{LibraryID: "id"}}
	a.Preferences.AutoImport["id"] = []AutoImportRule{{Path: "/a", FolderID: "gone", Library: original}}
	a.applySnapshot(model.Snapshot{})
	if a.Preferences.AutoImport["id"][0].FolderID != "gone" {
		t.Fatal("副本里删文件夹改掉了原库的规则")
	}
}
