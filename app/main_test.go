package main

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	appui "github.com/Seaony/Momento/app/ui"
	"github.com/egoist/mygo"
)

func TestSettingsMenuLocationAndShortcut(t *testing.T) {
	for _, platform := range []string{"darwin", "windows", "linux"} {
		for _, language := range []string{"zh-CN", "en", "ja"} {
			t.Run(platform+"/"+language, func(t *testing.T) {
				a := &appui.App{Texts: i18n.Catalog{Language: language}}
				opened := 0
				a.SettingsRequested = func() { opened++ }
				var settings *mygo.MenuItem
				matches := 0
				for i, menu := range menuItems(a, platform) {
					for _, entry := range menu.Submenu {
						if entry.Label != a.T("settings") {
							continue
						}
						matches++
						settings = entry
						want := 1
						if platform == "darwin" {
							want = 0
						}
						if i != want {
							t.Fatalf("设置入口位于菜单%d，期望%d", i, want)
						}
					}
				}
				if matches != 1 || settings.Accelerator != "CmdOrCtrl+," {
					t.Fatalf("设置入口或快捷键错误：%d，%+v", matches, settings)
				}
				settings.Click(settings, nil)
				if opened != 1 || !a.SettingsOpen {
					t.Fatal("菜单没有打开设置窗口")
				}
			})
		}
	}
}
