package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestGBrowsingScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_G_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_G_SCREENSHOTS=1 生成智能文件夹和快速访问截图")
	}
	renderGBrowsingScreenshots(t, "/tmp/momento-eagle/g-ui/screenshots")
}
func renderGBrowsingScreenshots(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	type record struct{ File, State, Language, Theme string }
	records := []record{}
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, state := range []string{"sidebar", "empty", "editor-new", "editor-tags-color", "editor-dimensions-date-note", "editor-error", "editor-saving"} {
				t.Run(language+"/"+theme+"/"+state, func(t *testing.T) {
					a := smartTestApp()
					a.Preferences.Language = language
					a.Preferences.Theme = theme
					a.Texts.Language = language
					a.Library.Manifest.DisplayName = "Momento Samples"
					switch state {
					case "sidebar":
						a.BrowseSmartFolder("smart-a")
					case "empty":
						a.Snapshot.QuickAccess = nil
						a.Snapshot.SmartFolders = nil
						a.applySnapshot(a.Snapshot)
					default:
						a.EditSmartFolder("")
						a.GBrowsing.SmartName = "Design reference images"
						a.GBrowsing.SmartRules = []model.SmartRule{{Field: "name", Operator: "contains", Value: "design"}}
						switch state {
						case "editor-tags-color":
							a.GBrowsing.SmartMatch = 1
							a.GBrowsing.SmartRules = []model.SmartRule{{Field: "extension", Operator: "equals", Value: "png"}, {Field: "tag", Operator: "equals", Value: "tag-a"}, {Field: "color", Operator: "equals", Value: "blue"}}
						case "editor-dimensions-date-note":
							a.GBrowsing.SmartRules = []model.SmartRule{{Field: "width", Operator: "gte", Value: "1920"}, {Field: "height", Operator: "gte", Value: "1080"}, {Field: "size", Operator: "lte", Value: "20971520"}, {Field: "added", Operator: "after", Value: "2026-11-01"}, {Field: "note", Operator: "contains", Value: "design reference"}}
						case "editor-error":
							a.GBrowsing.SmartError = a.T("error_smart_rule")
						case "editor-saving":
							a.GBrowsing.Saving = true
							a.Busy = true
						}
					}
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Row(c).Fill().Children(func() {
							ui.Column(c).Width(236).FillHeight().Padding(8).Gap(4).Background(c.Theme().Surface).Children(func() {
								ui.Box(c).Height(48).FillWidth().DragWindow()
								ui.Text(c, a.libraryName()).Bold().Padding(8, 10)
								a.sidebarQuickAccess(c)
								a.sidebarSmartFolders(c)
							})
							ui.Column(c).Grow(1).FillHeight().Children(func() { a.toolbar(c); a.assetArea(c) })
						})
						a.smartFolderDialog(c)
					}, 1100, 640)
					tt.SetDark(theme == "dark")
					tt.SetLocale(a.Texts.Locale())
					tt.SetWindowActive(true)
					tt.Frame()
					for _, text := range tt.Texts() {
						if strings.ContainsAny(text, "⟦⟧") || strings.Contains(text, "%!") {
							t.Fatalf("截图存在缺失或错误文案：%s", text)
						}
					}
					if state != "empty" && !tt.HasText(a.T("quick_access")) {
						t.Fatal("截图缺少快速访问分组")
					}
					if strings.HasPrefix(state, "editor") && !tt.HasText(a.T("smart_all")) {
						t.Fatal("截图缺少智能条件编辑器")
					}
					name := fmt.Sprintf("%s-%s-%s.png", state, language, theme)
					writeScreenshot(t, filepath.Join(dir, name), tt)
					records = append(records, record{name, state, language, theme})
				})
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
