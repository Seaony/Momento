package ui

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func sidebarTestApp() *App {
	snapshot := model.Snapshot{Assets: []model.Asset{{ID: "a", Name: "Landscape", Favorite: true}, {ID: "b", Name: "Portrait", FolderIDs: []string{"root"}}, {ID: "c", Name: "Deleted", Trashed: true}}, Folders: []model.Folder{{ID: "root", Name: "Design references"}, {ID: "child", Name: "UI details", ParentID: "root"}}}
	app := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, Library: &library.Library{Manifest: model.Manifest{LibraryID: "test", DisplayName: "Design Library"}}, Snapshot: snapshot, Index: catalog.New(snapshot), Query: catalog.Query{Sort: "added", Descending: true}}
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
	if !app.sidebarCanDrop(AssetsDrag{LibraryID: "test", assets: func() []string { return []string{"a"} }}) {
		t.Fatal("同库素材被拒绝")
	}
	if app.sidebarCanDrop(AssetsDrag{LibraryID: "other", assets: func() []string { return []string{"a"} }}) {
		t.Fatal("接收了其他图库素材")
	}
	// 忙时也接收：写入进队列。
	app.Busy = true
	if !app.sidebarCanDrop(AssetsDrag{LibraryID: "test", assets: func() []string { return []string{"a"} }}) {
		t.Fatal("忙碌时拒绝了拖入")
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
			tt.Advance(time.Second)
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

func TestRandomSidebarDoesNotRepeatAllCount(t *testing.T) {
	a := sidebarTestApp()
	a.Preferences.SidebarHidden = []string{"untagged"}
	tt := ui.NewTester(a.sidebar, 236, 800)
	if _, ok := tt.AccessByRole(ui.RoleButton, a.T("random_browse")); !ok {
		t.Fatal("缺少随机浏览入口")
	}
	count := 0
	for _, text := range tt.Texts() {
		if text == "2" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("全部的计数应只出现一次，实际%d次", count)
	}
}

// 拖素材进文件夹：说明加了几个；全都已经在里面时说「已在」。
func TestSidebarDropAssetsNotifiesResult(t *testing.T) {
	a := sidebarTestApp()
	a.applySnapshot(a.Snapshot)
	a.Busy = true // 写入进队列，测试里不碰数据库
	a.addAssetsToFolder([]string{"a", "b"}, "root", "Design references")
	if a.Notice != a.T("added_to_folder", 1, "Design references") || len(a.queuedRuns) != 1 {
		t.Fatalf("加入文件夹的提示或写入不对：%q %d", a.Notice, len(a.queuedRuns))
	}
	a.Notice = ""
	a.addAssetsToFolder([]string{"b"}, "root", "Design references")
	if a.Notice != a.T("already_in_folder", "Design references") || len(a.queuedRuns) != 1 {
		t.Fatalf("全都已在文件夹里时应只提示：%q %d", a.Notice, len(a.queuedRuns))
	}
}

// 删除文件夹：空的直接删；有内容的确认语写明子文件夹和素材数量。
func TestSidebarDeleteFolderConfirmWording(t *testing.T) {
	a := sidebarTestApp()
	a.Busy = true
	a.deleteFolder(a.Snapshot.Folders[1])
	if a.ConfirmOpen || len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "delete_folder" {
		t.Fatalf("空文件夹应直接删除：confirm=%v queued=%d", a.ConfirmOpen, len(a.queuedRuns))
	}
	a.deleteFolder(a.Snapshot.Folders[0])
	if !a.ConfirmOpen || a.ConfirmDetail != a.T("delete_folder_tree_detail", "Design references", 1, 1) || a.ConfirmLabel != a.T("delete") {
		t.Fatalf("有子文件夹的确认语不对：%q", a.ConfirmDetail)
	}
	a.ConfirmOpen = false
	a.Snapshot.Folders = a.Snapshot.Folders[:1]
	a.deleteFolder(a.Snapshot.Folders[0])
	if a.ConfirmDetail != a.T("delete_folder_assets_detail", "Design references", 1) {
		t.Fatalf("没有子文件夹的确认语不对：%q", a.ConfirmDetail)
	}
}

func TestSidebarEmptyTrashConfirmShowsCount(t *testing.T) {
	a := sidebarTestApp()
	a.ConfirmEmptyTrash()
	if a.ConfirmDetail != a.T("empty_trash_count_detail", 1) || a.ConfirmLabel != a.T("empty_trash") {
		t.Fatalf("清空回收站确认没写数量或按钮不对：%q %q", a.ConfirmDetail, a.ConfirmLabel)
	}
}

// 只有子文件夹的右键菜单有「移到最外层」；回收站为空时右键不弹菜单。
func TestSidebarFolderMenuAndEmptyTrashMenu(t *testing.T) {
	a := sidebarTestApp()
	*a.sidebarOpenFlag("root") = true
	tt := ui.NewTester(a.sidebar, 236, 800)
	if err := tt.RightClick("UI details"); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(tt.Menu(), a.T("move_to_root")) {
		t.Fatalf("子文件夹右键没有移到最外层：%v", tt.Menu())
	}
	tt.CloseMenu()
	if err := tt.RightClick("Design references"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(tt.Menu(), a.T("move_to_root")) || !slices.Contains(tt.Menu(), a.T("new_subfolder")) {
		t.Fatalf("顶层文件夹右键菜单不对：%v", tt.Menu())
	}
	tt.CloseMenu()
	// 回收站里有一个素材（sidebarTestApp 里的 c）时有菜单；清空后不弹。
	if err := tt.RightClick(a.T("trash")); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(tt.Menu(), a.T("empty_trash")) {
		t.Fatalf("回收站有素材时右键没有清空：%v", tt.Menu())
	}
	tt.CloseMenu()
	a.Snapshot.Assets[2].Trashed = false
	a.applySnapshot(a.Snapshot)
	tt.Frame()
	if err := tt.RightClick(a.T("trash")); err != nil {
		t.Fatal(err)
	}
	if len(tt.Menu()) != 0 {
		t.Fatalf("回收站为空时不该弹出菜单：%v", tt.Menu())
	}
}

// 标签管理行的数字原来是标签个数，和其它行的素材数混在一起；现在不显示。
func TestSidebarTagManagementRowHasNoCount(t *testing.T) {
	a := sidebarTestApp()
	a.Preferences.SidebarCounts = true
	for i := range 7 {
		a.Snapshot.Tags = append(a.Snapshot.Tags, model.Tag{ID: fmt.Sprint("tag", i), Name: fmt.Sprint("T", i)})
	}
	tt := ui.NewTester(a.sidebar, 236, 800)
	if !tt.HasText(a.T("tag_management")) || tt.HasText("7") {
		t.Fatalf("标签管理行不该显示标签个数：%v", tt.Texts())
	}
}
