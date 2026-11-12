package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestPasswordLockScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_LOCK_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_LOCK_SCREENSHOTS=1 生成密码锁截图")
	}
	dir := "/tmp/momento-eagle/lockui"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	type record struct{ File, Language, Theme, State string }
	records := []record{}
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, state := range []string{"locked", "wrong", "cooldown", "enable", "change", "save-error", "save-permission-error"} {
				t.Run(language+"/"+theme+"/"+state, func(t *testing.T) {
					a := settingsTestApp(language, theme)
					a.Preferences.PasswordHash = "screenshot-only"
					a.InitPasswordLock()
					width, height := 1100, 640
					view := a.View
					switch state {
					case "wrong":
						a.Lock.ErrorKey = "password_wrong"
					case "cooldown":
						a.Lock.ErrorKey = "password_cooldown"
						a.Lock.CooldownUntil = time.Now().Add(30 * time.Second)
					case "enable", "change", "save-error", "save-permission-error":
						a.Lock.Locked = false
						a.Settings.Tab = SettingsPassword
						mode := state
						if state == "enable" {
							a.Preferences.PasswordHash = ""
						} else if state == "save-error" || state == "save-permission-error" {
							mode = "change"
						}
						a.ShowPasswordSettings(mode)
						if state == "save-error" {
							a.Lock.ErrorKey, a.Lock.Detail = "password_save_failed", a.passwordErrorDetail(&os.PathError{Op: "write", Path: "preferences.json", Err: syscall.ENOSPC})
						} else if state == "save-permission-error" {
							a.Lock.ErrorKey, a.Lock.Detail = "password_save_failed", a.passwordErrorDetail(&os.PathError{Op: "write", Path: "preferences.json", Err: os.ErrPermission})
						}
						width, height, view = 760, 560, a.SettingsView
					}
					tt := ui.NewTester(view, width, height)
					tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 36})
					tt.SetWindowActive(true)
					tt.SetLocale(a.Texts.Locale())
					tt.Frame()
					for _, text := range tt.Texts() {
						if strings.Contains(text, "⟦") {
							t.Fatalf("密码截图有缺失文案：%s", text)
						}
					}
					file := fmt.Sprintf("%s-%s-%s.png", language, theme, state)
					writeScreenshot(t, filepath.Join(dir, file), tt)
					records = append(records, record{file, language, theme, state})
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
