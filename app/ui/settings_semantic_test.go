package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsSemanticScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_SEMANTIC_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_SEMANTIC_SCREENSHOTS=1 生成语义色设置截图")
	}
	dir := "/tmp/momento-polish/shots3/settings"
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
