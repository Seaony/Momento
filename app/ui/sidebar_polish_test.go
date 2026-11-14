package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestSidebarFlatEntriesAndAddRows(t *testing.T) {
	a := sidebarTestApp()
	a.Preferences.SmartFolders = true
	tt := ui.NewTester(a.sidebar, 236, 800)
	for _, text := range []string{a.T("untagged"), a.T("uncategorized"), a.T("tag_management"), "Design references", a.T("new_folder"), a.T("new_smart_folder")} {
		if !tt.HasText(text) {
			t.Fatalf("侧栏应该直接平铺显示：%s", text)
		}
	}
	for _, text := range []string{a.T("folders"), a.T("smart_folders")} {
		if tt.HasText(text) {
			t.Fatalf("文件夹不再分组，不应出现分组标题：%s", text)
		}
	}
	folder, _ := tt.Find("Design references")
	add, _ := tt.Find(a.T("new_folder"))
	if add.Y <= folder.Y {
		t.Fatal("新建文件夹应在文件夹列表最下面")
	}
	if err := tt.Click(a.T("new_smart_folder")); err != nil {
		t.Fatal(err)
	}
	if !a.GBrowsing.SmartOpen {
		t.Fatal("新建智能文件夹没有打开编辑器")
	}
	a.GBrowsing.SmartOpen = false
	a.Busy = true
	tt.Frame()
	for _, key := range []string{"new_folder", "new_smart_folder"} {
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
				for _, key := range []string{"all", "recent", "favorites", "random_browse", "uncategorized", "untagged", "tag_management", "new_folder", "new_smart_folder"} {
					label := a.T(key)
					width, _ := c.MeasureText(0, ui.Span{Text: label, Size: 13})
					// 236 宽侧栏扣除左右留白、行内留白、图标、间距与三位数计数。
					available := float32(236 - 2*sidebarPadding - 2*sidebarRowPadding - sidebarIconBox - sidebarIconGap - 25)
					if key == "new_folder" || key == "new_smart_folder" {
						available = 236 - 2*sidebarPadding - 2*sidebarRowPadding - sidebarIconBox - sidebarIconGap
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

func TestSidebarFooterHasCompactActionsAndOptionalCount(t *testing.T) {
	for _, count := range []int{0, 3} {
		a := sidebarTestApp()
		tt := ui.NewTester(func(c *ui.Context) { a.sidebarFooter(c, count) }, 236, 46)
		for _, key := range []string{"trash", "settings", "help"} {
			info, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
			if !ok || info.Bounds.W != sidebarFooterButtonSize || info.Bounds.H != sidebarFooterButtonSize || info.Description != a.T(key) {
				t.Fatalf("底部图标按钮缺少点击区域或提示：%s %+v", key, info)
			}
		}
		if tt.HasText("0") || (count > 0 && !tt.HasText(sidebarCount(count))) {
			t.Fatal("回收站角标没有按空与非空显示")
		}
		if err := tt.Click(a.T("settings")); err != nil {
			t.Fatal(err)
		}
		if !a.SettingsOpen {
			t.Fatal("底部设置图标没有打开设置")
		}
	}
}

func TestSidebarPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SIDEBAR_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SIDEBAR_SCREENSHOTS=1 生成侧栏精修截图")
	}
	dir := "/tmp/momento-polish/shots2/sidebar"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, appearance := range []string{"light", "dark"} {
		for _, state := range []string{"empty-groups", "populated", "library-menu", "trash", "smart-editor", "smart-dropdown"} {
			t.Run(appearance+"/"+state, func(t *testing.T) {
				a := fixture.app(t, "small", "zh-CN", appearance)
				a.Library.Manifest.DisplayName = "灵感素材库"
				// 智能文件夹默认关闭，截图要覆盖它的分组和表单。
				a.Preferences.SmartFolders = true
				if state == "populated" {
					a.Snapshot.Folders = []model.Folder{{ID: "design", Name: "设计参考"}, {ID: "ui", Name: "界面细节", ParentID: "design"}}
					a.Snapshot.SmartFolders = []model.SmartFolder{{ID: "large", Name: "高清图片", Match: "all", Rules: []model.SmartRule{{Field: "width", Operator: "gte", Value: "1920"}}}}
					a.Snapshot.QuickAccess = []model.QuickAccess{{ID: "pin", Kind: "folder", TargetID: "design"}}
					a.applySnapshot(a.Snapshot)
				}
				if state == "populated" {
					open := true
					a.Sidebar.Open = map[string]*bool{"design": &open}
				}
				if state == "trash" {
					a.Snapshot.Assets[0].Trashed = true
					a.applySnapshot(a.Snapshot)
					a.SetScope("trash", "")
				}
				if strings.HasPrefix(state, "smart-") {
					a.EditSmartFolder("")
					a.GBrowsing.SmartName = "灵感参考"
				}
				if state == "library-menu" {
					a.Recent = []library.Recent{{Path: "/tmp/reference.momento", DisplayName: "参考图库"}}
					a.Sidebar.LibraryOpen = true
				}
				tt := ui.NewTester(a.View, 1440, 900)
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetLocale(a.Texts.Locale())
				tt.SetWindowActive(true)
				tt.Frame()
				tt.Advance(time.Second)
				if state == "smart-dropdown" {
					if err := tt.Click(a.T("smart_rule_field", 1)); err != nil {
						t.Fatal(err)
					}
					tt.Advance(time.Second)
				}
				writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+appearance+"-1440x900.png"), tt)
			})
		}
	}
}
