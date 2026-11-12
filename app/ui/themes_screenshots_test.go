package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 主题矩阵单独生成，避免拿旧的浅深色截图充当新色板验收。
func TestThemeScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_THEME_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_THEME_SCREENSHOTS=1 生成七主题截图")
	}
	dir := "/tmp/momento-eagle/theme"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	type record struct {
		File   string  `json:"file"`
		Theme  string  `json:"theme"`
		State  string  `json:"state"`
		Width  int     `json:"width"`
		Height int     `json:"height"`
		Zoom   float32 `json:"zoom"`
	}
	records := []record{}
	render := func(t *testing.T, key, state string, zoom float32) {
		t.Helper()
		width, height := zoomedWindowSize(1100, 640, zoom, mygo.Rectangle{})
		a := fixture.app(t, "large", "zh-CN", key)
		a.Preferences = DefaultPreferences()
		a.Preferences.Theme, a.Preferences.Language, a.Preferences.UIZoom = key, "zh-CN", zoom
		view := a.View
		bar := ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37}
		switch state {
		case "single":
			for _, asset := range a.Visible {
				if asset.Extension == "jpg" && len(asset.Colors) == 8 {
					a.Selected.Add(asset.ID)
					break
				}
			}
			if a.Selected.Len() != 1 {
				t.Fatal("单选主题截图缺少真实素材")
			}
		case "filter":
			a.Toolbar.FilterOpen = true
		case "settings-appearance":
			a = settingsTestApp("zh-CN", key)
			a.Preferences.UIZoom = zoom
			a.Settings.Tab = SettingsAppearance
			width, height = zoomedWindowSize(760, 560, zoom, mygo.Rectangle{})
			view = a.SettingsView
			bar = ui.TitleBar{Height: 44, Left: 80, Bottom: 25}
		}
		tt := ui.NewTester(view, width, height)
		tt.SetLocale("zh-Hans")
		tt.SetTitleBar(bar)
		tt.SetWindowActive(true)
		tt.Frame()
		if state == "large" && len(a.Visible) != 12000 {
			t.Fatal("主题截图没有使用大图库")
		}
		if state == "settings-appearance" {
			for _, theme := range themeKeys {
				name := a.T(themeNameKey(theme))
				if strings.HasPrefix(name, "⟦") {
					t.Fatalf("主题截图缺少翻译 %s", theme)
				}
				if _, ok := tt.Find(name); !ok {
					t.Fatalf("主题选择器缺少 %s", theme)
				}
			}
		}
		if a.Thumbnails != nil && len(a.Thumbnails.pending) > 0 {
			t.Fatal("主题截图尚有缩略图未加载")
		}
		file := fmt.Sprintf("%s-%s-%d.png", key, state, int(zoom*100))
		writeScreenshot(t, filepath.Join(dir, file), tt)
		records = append(records, record{file, key, state, width, height, zoom})
	}
	for _, key := range themeKeys {
		for _, state := range []string{"large", "single", "filter", "settings-appearance"} {
			t.Run(key+"/"+state, func(t *testing.T) { render(t, key, state, 1) })
		}
	}
	render(t, "light", "large", 1.5)
	render(t, "light", "settings-appearance", 1.5)
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
