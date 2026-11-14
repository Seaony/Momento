package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestSidebarOrganizeGroupKeepsBrowsingState(t *testing.T) {
	a := sidebarTestApp()
	tt := ui.NewTester(a.sidebar, 236, 640)
	if err := tt.Click(a.T("untagged")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(a.T("organize")); err != nil {
		t.Fatal(err)
	}
	if !a.Sidebar.OrganizeCollapsed || a.Query.Scope != "untagged" {
		t.Fatal("收起整理分组改变了当前浏览范围")
	}
	for _, key := range []string{"untagged", "uncategorized", "tag_management"} {
		if tt.HasText(a.T(key)) {
			t.Fatalf("收起后仍显示整理条目 %s", key)
		}
	}
	if err := tt.Click(a.T("organize")); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(a.T("tag_management")) {
		t.Fatal("重新展开后缺少标签管理")
	}
}

func TestSidebarEmptyFolderActions(t *testing.T) {
	a := sidebarTestApp()
	a.Snapshot.Folders = nil
	tt := ui.NewTester(a.sidebar, 236, 640)
	if !tt.HasText(a.T("new_folder")) || !tt.HasText(a.T("smart_empty_action")) {
		t.Fatal("空文件夹分组缺少创建入口")
	}
	if err := tt.Click(a.T("smart_empty_action")); err != nil {
		t.Fatal(err)
	}
	if !a.GBrowsing.SmartOpen {
		t.Fatal("空智能文件夹入口未打开编辑器")
	}
	a.GBrowsing.SmartOpen = false
	a.Busy = true
	tt.Frame()
	for _, key := range []string{"new_folder", "smart_empty_action"} {
		info, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
		if !ok || !info.Disabled {
			t.Fatalf("忙碌时创建入口仍可操作：%s", key)
		}
	}
}

func TestSidebarLocalizedLabelsFit(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := sidebarTestApp()
			a.Texts.Language = language
			a.Snapshot.Folders = nil
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				a.sidebar(c)
				for _, key := range []string{"all", "recent", "favorites", "random_browse", "uncategorized", "untagged", "tag_management", "trash", "settings", "new_folder", "smart_empty_action"} {
					label := a.T(key)
					width, _ := c.MeasureText(0, ui.Span{Text: label, Size: 13})
					// 236宽侧栏扣除外内边距、图标、间距与三位数计数。
					available := float32(236 - 24 - 22 - 20 - 10 - 25 - 10)
					if key == "new_folder" || key == "smart_empty_action" {
						available += 35
					}
					if width > available {
						t.Errorf("%s 文案宽 %.1f，超过可用宽 %.1f", key, width, available)
					}
				}
			}, 236, 640)
			tt.SetLocale(a.Texts.Locale())
			tt.Frame()
			for _, text := range tt.Texts() {
				if strings.ContainsAny(text, "⟦⟧") || strings.Contains(text, "%!") {
					t.Fatalf("出现缺失文案：%s", text)
				}
			}
		})
	}
}

func TestSidebarPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SIDEBAR_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SIDEBAR_SCREENSHOTS=1 生成侧栏精修截图")
	}
	dir := "/tmp/momento-polish/shots/sidebar"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, appearance := range []string{"light", "dark"} {
		for _, state := range []string{"empty-groups", "populated", "collapsed", "library-menu"} {
			t.Run(appearance+"/"+state, func(t *testing.T) {
				a := fixture.app(t, "small", "zh-CN", appearance)
				a.Library.Manifest.DisplayName = "灵感素材库"
				if state == "populated" || state == "collapsed" {
					a.Snapshot.Folders = []model.Folder{{ID: "design", Name: "设计参考"}, {ID: "ui", Name: "界面细节", ParentID: "design"}}
					a.Snapshot.SmartFolders = []model.SmartFolder{{ID: "large", Name: "高清图片", Match: "all", Rules: []model.SmartRule{{Field: "width", Operator: "gte", Value: "1920"}}}}
					a.Snapshot.QuickAccess = []model.QuickAccess{{ID: "pin", Kind: "folder", TargetID: "design"}}
					a.applySnapshot(a.Snapshot)
				}
				a.Sidebar.OrganizeCollapsed = state == "collapsed"
				if state == "library-menu" {
					a.Recent = []library.Recent{{Path: "/tmp/reference.momento", DisplayName: "参考图库"}}
					a.Sidebar.LibraryOpen = true
				}
				tt := ui.NewTester(a.View, 1440, 900)
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetLocale(a.Texts.Locale())
				tt.SetWindowActive(true)
				tt.Frame()
				writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+appearance+"-1440x900.png"), tt)
			})
		}
	}
}
