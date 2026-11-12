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

func TestSidebarScreenshotFolderCounts(t *testing.T) {
	fixture := newScreenshotFixture(t)
	for _, state := range []string{"filter", "filter-selected", "filter-color-hover"} {
		t.Run(state, func(t *testing.T) {
			app := fixture.app(t, state, "en", "light")
			counts := app.Index.Counts()
			if counts.All != 14 || counts.Uncategorized != 0 || counts.Folders["design"] != 7 || counts.Folders["ui"] != 7 {
				t.Fatalf("筛选截图的父子文件夹计数错误：%+v", counts)
			}
			for i, asset := range app.Snapshot.Assets {
				want := "design"
				if i%2 != 0 {
					want = "ui"
				}
				if len(asset.FolderIDs) != 1 || asset.FolderIDs[0] != want {
					t.Fatalf("第%d张截图素材的直接文件夹为%v，期望%s", i+1, asset.FolderIDs, want)
				}
			}
			tt := ui.NewTester(app.sidebar, 236, 800)
			if err := tt.Click(app.T("toggle_folder", "Design references")); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText("UI layouts") {
				t.Fatal("展开后没有显示存放其余7张素材的子文件夹")
			}
			sevens := 0
			for _, text := range tt.Texts() {
				if text == "7" {
					sevens++
				}
			}
			if sevens != 2 {
				t.Fatalf("父子文件夹应各显示7，实际出现%d个7", sevens)
			}
			for _, folder := range app.Snapshot.Folders {
				if err := tt.Click(folder.Name); err != nil {
					t.Fatal(err)
				}
				if app.Query.Scope != "folder" || app.Query.FolderID != folder.ID || len(app.Visible) != 7 {
					t.Fatalf("点击%s后应只显示7张直接成员，实际范围%+v、数量%d", folder.Name, app.Query, len(app.Visible))
				}
				for _, asset := range app.Visible {
					if len(asset.FolderIDs) != 1 || asset.FolderIDs[0] != folder.ID {
						t.Fatalf("%s范围混入了其他文件夹素材%s", folder.Name, asset.ID)
					}
				}
			}
		})
	}
}
