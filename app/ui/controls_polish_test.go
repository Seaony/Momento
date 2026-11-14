package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSegmentPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SEGMENT_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SEGMENT_SCREENSHOTS=1 生成分段控件截图")
	}
	dir := "/tmp/momento-polish/shots/segments"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"general", "sidebar", "controls", "preview", "smart-match"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				var tt *ui.Tester
				if state == "smart-match" {
					a := smartTestApp()
					a.Preferences.Language, a.Preferences.Theme = "zh-CN", theme
					a.Texts.Language = "zh-CN"
					a.EditSmartFolder("")
					a.GBrowsing.SmartName = "设计参考"
					a.GBrowsing.SmartMatch = 1
					a.GBrowsing.SmartRules[0].Value = "design"
					tt = ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						ui.Box(c).Fill().Background(c.Theme().Background)
						a.smartFolderDialog(c)
					}, 900, 640)
					tt.SetDark(theme == "dark")
					tt.SetLocale(a.Texts.Locale())
					tt.SetWindowActive(true)
					tt.Frame()
					if item, ok := tt.AccessByRole(ui.RoleRadio, a.T("smart_any")); !ok || !item.Checked {
						t.Fatal("智能匹配选中态丢失")
					}
				} else {
					a, status := finalSettingsScreenshotState("zh-CN", theme, state)
					tt = polishedSettingsTester(a, status, nil)
					checkFinalSettingsScreenshot(t, a, tt, status)
				}
				writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("segments-%s-zh-CN-%s.png", state, theme)), tt)
			})
		}
	}
}
