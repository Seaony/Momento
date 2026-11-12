package ui

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func sidebarTestApp() *App {
	snapshot := model.Snapshot{Assets: []model.Asset{{ID: "a", Name: "Landscape", Favorite: true}, {ID: "b", Name: "Portrait", FolderIDs: []string{"root"}}, {ID: "c", Name: "Deleted", Trashed: true}}, Folders: []model.Folder{{ID: "root", Name: "Design references"}, {ID: "child", Name: "UI details", ParentID: "root"}}}
	app := &App{Texts: i18n.Catalog{Language: "en"}, Library: &library.Library{Manifest: model.Manifest{LibraryID: "test", DisplayName: "Design Library"}}, Snapshot: snapshot, Index: catalog.New(snapshot), Query: catalog.Query{Sort: "added", Descending: true}}
	app.Refine()
	return app
}

func TestSidebarScopeAndFolders(t *testing.T) {
	app := sidebarTestApp()
	tt := ui.NewTester(app.sidebar, 236, 800)
	if err := tt.Click(app.T("favorites")); err != nil {
		t.Fatal(err)
	}
	if app.Query.Scope != "favorites" || len(app.Visible) != 1 || app.Visible[0].ID != "a" {
		t.Fatalf("收藏范围错误: %+v", app.Visible)
	}
	if err := tt.Click("Design references"); err != nil {
		t.Fatal(err)
	}
	if app.Query.Scope != "folder" || app.Query.FolderID != "root" || len(app.Visible) != 1 || app.Visible[0].ID != "b" {
		t.Fatalf("文件夹范围错误: %+v", app.Visible)
	}
	if tt.HasText("UI details") {
		t.Fatal("折叠文件夹显示了子项")
	}
	if err := tt.Click(app.T("toggle_folder", "Design references")); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("UI details") {
		t.Fatal("展开文件夹后未显示子项")
	}
	if err := tt.Click(app.T("tag_management")); err != nil {
		t.Fatal(err)
	}
	if !app.TagPage {
		t.Fatal("标签管理入口未打开")
	}
	if err := tt.Click(app.T("trash")); err != nil {
		t.Fatal(err)
	}
	if app.TagPage || app.Query.Scope != "trash" || len(app.Visible) != 1 || app.Visible[0].ID != "c" {
		t.Fatal("回收站范围错误")
	}
	if err := tt.Click(app.T("settings")); err != nil {
		t.Fatal(err)
	}
	if !app.SettingsOpen {
		t.Fatal("设置入口未打开")
	}
}

func TestSidebarLibraryMenuAndNewLibrary(t *testing.T) {
	app := sidebarTestApp()
	app.Recent = []library.Recent{{Path: "/tmp/design.momento", DisplayName: "Recent Design"}}
	tt := ui.NewTester(app.sidebar, 236, 800)
	if err := tt.Click(app.T("library_menu")); err != nil {
		t.Fatal(err)
	}
	if !app.Sidebar.LibraryOpen || !tt.HasText("Recent Design") {
		t.Fatal("图库切换器未显示最近图库")
	}
	if err := tt.Click(app.T("new_library")); err != nil {
		t.Fatal(err)
	}
	if !app.NewLibraryOpen || app.Sidebar.LibraryOpen {
		t.Fatal("新建图库未打开或切换器未关闭")
	}
}

func TestSidebarRejectsOtherLibraryDrag(t *testing.T) {
	app := sidebarTestApp()
	if !app.sidebarCanDrop(AssetsDrag{LibraryID: "test", IDs: []string{"a"}}) {
		t.Fatal("同库素材被拒绝")
	}
	if app.sidebarCanDrop(AssetsDrag{LibraryID: "other", IDs: []string{"a"}}) {
		t.Fatal("接收了其他图库素材")
	}
	app.Busy = true
	if app.sidebarCanDrop(AssetsDrag{LibraryID: "test", IDs: []string{"a"}}) {
		t.Fatal("忙碌时仍接收写入")
	}
}
