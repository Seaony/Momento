package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/egoist/mygo"
)

func TestSettingsSemanticScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_SEMANTIC_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_SEMANTIC_SCREENSHOTS=1 生成语义色设置截图")
	}
	dir := "/tmp/momento-polish/shots3/settings-fix"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		a := settingsTestApp("zh-CN", theme)
		tt := polishedSettingsTester(a, UpdateCheckStatus{}, nil)
		checkFinalSettingsScreenshot(t, a, tt, UpdateCheckStatus{})
		writeScreenshot(t, filepath.Join(dir, "settings-"+theme+".png"), tt)
	}
}

func TestSettingsValueLabelsKeepLocalizedOrder(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, tab := range []int{SettingsGeneral, SettingsAbout} {
				a := settingsTestApp(language, theme)
				a.Settings.Tab = tab
				tt := polishedSettingsTester(a, UpdateCheckStatus{}, nil)
				before, after, _ := strings.Cut(a.T("current_version", "\uFFFC"), "\uFFFC")
				value, ok := tt.Find(mygo.App.Version())
				if !ok || value.W == 0 || value.X+value.W > 736 || value.Y+value.H > 560 {
					t.Fatalf("%s/%s 版本值没有完整显示", language, theme)
				}
				if label := strings.TrimSpace(before); label != "" {
					bounds, ok := tt.Find(label)
					if !ok || bounds.X+bounds.W > value.X || bounds.H == 0 {
						t.Fatalf("%s 版本前标签被截断或顺序错误", language)
					}
				}
				if label := strings.TrimSpace(after); label != "" {
					bounds, ok := tt.Find(label)
					if !ok || bounds.X < value.X+value.W || bounds.H == 0 {
						t.Fatalf("%s 版本后标签被截断或顺序错误", language)
					}
				}
			}
			a := settingsTestApp(language, theme)
			a.Settings.Tab = SettingsExtension
			a.ExtensionStatus.State = browser.Running
			tt := polishedSettingsTester(a, UpdateCheckStatus{}, nil)
			label, labelOK := tt.Find(a.T("extension_running"))
			address, addressOK := tt.Find(a.ExtensionStatus.Address)
			if !labelOK || !addressOK || label.X+label.W > address.X || address.X+address.W > 736 || label.H == 0 || address.H == 0 {
				t.Fatalf("%s/%s 扩展状态与独立地址未完整排列", language, theme)
			}
		}
	}
}
