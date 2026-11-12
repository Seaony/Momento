package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
)

func TestPreferencesMissingFieldsKeepDefaults(t *testing.T) {
	p := DefaultPreferences()
	if err := json.Unmarshal([]byte(`{"language":"ja","appearance":"dark"}`), &p); err != nil {
		t.Fatal(err)
	}
	p = NormalizePreferences(p)
	if p.Language != "ja" || p.Theme != "dark" || !p.AutoCheckUpdates || !p.BrowserExtensionEnabled || p.LaunchAtLogin {
		t.Fatalf("缺字段的偏好没有保留开关默认值：%+v", p)
	}
	if p.ViewMode != 0 || p.Sort != "added" || !p.SortDescending || !p.SidebarVisible || !p.InspectorVisible || p.SidebarWidth != 236 {
		t.Fatalf("缺字段的偏好没有保留界面默认状态：%+v", p)
	}
}

func TestPreferencesExplicitFalseSurvivesLoading(t *testing.T) {
	p := DefaultPreferences()
	if err := json.Unmarshal([]byte(`{"autoCheckUpdates":false,"browserExtensionEnabled":false,"sortDescending":false,"sidebarVisible":false,"inspectorVisible":false}`), &p); err != nil {
		t.Fatal(err)
	}
	p = NormalizePreferences(p)
	if p.AutoCheckUpdates || p.BrowserExtensionEnabled || p.SortDescending || p.SidebarVisible || p.InspectorVisible {
		t.Fatalf("明确关闭的偏好被默认值覆盖：%+v", p)
	}
}

func TestNormalizePreferencesRestoresSupportedState(t *testing.T) {
	p := DefaultPreferences()
	p.ViewMode, p.Sort, p.Theme, p.SidebarWidth = 9, "unknown", "unknown", 900
	p.SortDescending = false
	p = NormalizePreferences(p)
	if p.ViewMode != 0 || p.Sort != "added" || !p.SortDescending || p.Theme != "system" || p.SidebarWidth != 340 {
		t.Fatalf("非法界面状态没有规范化：%+v", p)
	}
	p.SidebarWidth = 50
	p.Sort, p.SortDescending = "name", false
	p = NormalizePreferences(p)
	if p.SidebarWidth != 196 || p.Sort != "name" || p.SortDescending {
		t.Fatalf("合法排序被改动或侧栏超出下限：%+v", p)
	}
}

func TestPreferencesRestoreUIStateAfterFlush(t *testing.T) {
	a := preferenceTestApp(t)
	a.ViewMode, a.Query.Sort, a.Query.Descending = 2, "size", false
	a.SidebarVisible, a.InspectorVisible, a.SidebarWidth = false, false, 307.5
	a.Preferences.Language, a.Preferences.Theme = "ja", "dark"
	a.Preferences.AutoCheckUpdates, a.Preferences.BrowserExtensionEnabled = false, false
	a.RememberUIState()
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	reopened := &App{Preferences: readTestPreferences(t, a.configDir)}
	reopened.RestoreUIState()
	if reopened.ViewMode != 2 || reopened.Query.Sort != "size" || reopened.Query.Descending || reopened.SidebarVisible || reopened.InspectorVisible || reopened.SidebarWidth != 307.5 {
		t.Fatalf("重启后没有恢复界面状态：%+v，%+v", reopened.Preferences, reopened.Query)
	}
	if reopened.Preferences.Language != "ja" || reopened.Preferences.Theme != "dark" || reopened.Preferences.AutoCheckUpdates || reopened.Preferences.BrowserExtensionEnabled {
		t.Fatalf("保存界面状态覆盖了设置项：%+v", reopened.Preferences)
	}
}

func TestRememberUIStateOnlySavesChanges(t *testing.T) {
	a := preferenceTestApp(t)
	menuChanges := 0
	a.MenuChanged = func() { menuChanges++ }
	a.RememberUIState()
	if a.prefVersion.Load() != 0 {
		t.Fatal("界面没有变化仍写入偏好")
	}
	a.SidebarWidth = 288
	a.RememberUIState()
	version := a.prefVersion.Load()
	if version != 1 {
		t.Fatalf("一次变化应只发起一次保存，实际版本%d", version)
	}
	a.RememberUIState()
	if a.prefVersion.Load() != version || menuChanges != 0 {
		t.Fatal("重复布局重新保存，或调整侧栏时重建了菜单")
	}
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	if got := readTestPreferences(t, a.configDir).SidebarWidth; got != 288 {
		t.Fatalf("侧栏宽度没有保存：%v", got)
	}
}

func TestPreferencesFlushPreventsOlderWrites(t *testing.T) {
	a := preferenceTestApp(t)
	old := a.Preferences
	oldVersion := a.prefVersion.Add(1)
	for i := range 40 {
		a.ViewMode = i % 3
		a.SidebarWidth = float32(196 + i)
		a.RememberUIState()
	}
	a.ViewMode, a.Query.Sort, a.Query.Descending = 2, "name", false
	a.SidebarVisible, a.InspectorVisible, a.SidebarWidth = false, true, 312
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	// 模拟退出时才取得锁的旧写入，不能覆盖最后的同步保存。
	if err := a.writePreferences(old, oldVersion); err != nil {
		t.Fatal(err)
	}
	p := readTestPreferences(t, a.configDir)
	if p.ViewMode != 2 || p.Sort != "name" || p.SortDescending || p.SidebarVisible || !p.InspectorVisible || p.SidebarWidth != 312 {
		t.Fatalf("立即退出后的文件被旧状态覆盖：%+v", p)
	}
}

func TestPreferencesWriteFailureKeepsPreviousFile(t *testing.T) {
	a := preferenceTestApp(t)
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.configDir, "preferences.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path+".tmp", 0700); err != nil {
		t.Fatal(err)
	}
	a.ViewMode = 2
	if err := a.FlushPreferences(); err == nil || a.Error == "" {
		t.Fatalf("保存失败没有向调用方和界面报告：%v，%q", err, a.Error)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("失败的写入改坏了已有偏好")
	}
}

func TestPreferencesWithoutStorageDoNotQueueWrites(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	a.RestoreUIState()
	a.ViewMode = 1
	a.RememberUIState()
	if err := a.FlushPreferences(); err != nil || a.prefVersion.Load() != 0 {
		t.Fatalf("离线界面夹具不应向当前目录写文件：%v", err)
	}
}

func preferenceTestApp(t *testing.T) *App {
	t.Helper()
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, configDir: t.TempDir()}
	a.RestoreUIState()
	return a
}

func readTestPreferences(t *testing.T, dir string) Preferences {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := DefaultPreferences()
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPreferencesMigrateThemeAndPreserveNewChoices(t *testing.T) {
	for _, old := range []string{"system", "light", "dark"} {
		var p Preferences
		if err := json.Unmarshal([]byte(`{"appearance":"`+old+`"}`), &p); err != nil {
			t.Fatal(err)
		}
		if p.Theme != old || p.UIZoom != 1 || !p.TrayIcon || !p.PreviewChecker || !p.SelectImported || !p.Notify || !p.NotifySound {
			t.Fatalf("迁移丢失默认值: %+v", p)
		}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "appearance") {
			t.Fatal("仍在写旧主题字段")
		}
	}
	var p Preferences
	if err := json.Unmarshal([]byte(`{"appearance":"light","theme":"purple","trayIcon":false,"dockWhenHidden":false,"previewChecker":false,"selectImported":false,"notify":false,"notifySound":false,"uiZoom":1.5,"shortcuts":{"search":"Alt+F"}}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Theme != "purple" || p.TrayIcon || !p.DockWhenHidden || p.PreviewChecker || p.SelectImported || p.Notify || p.NotifySound || p.UIZoom != 1.5 || p.Shortcuts["search"] != "Alt+F" {
		t.Fatalf("明确设置未保留: %+v", p)
	}
}

func TestPreferenceSnapshotDoesNotShareMutableFields(t *testing.T) {
	p := DefaultPreferences()
	p.AutoImport["library"] = []AutoImportRule{{Path: "before", Enabled: true}}
	p.Shortcuts["search"] = "Alt+F"
	copy := clonePreferences(p)
	p.AutoImport["library"][0].Path = "after"
	p.Shortcuts["search"] = "Alt+G"
	p.NotifyEvents[0] = "duplicate"
	if copy.AutoImport["library"][0].Path != "before" || copy.Shortcuts["search"] != "Alt+F" || copy.NotifyEvents[0] != "extension" {
		t.Fatal("后台保存共享了可变偏好")
	}
}

func TestChinesePreferenceSurvivesFlushAndLocale(t *testing.T) {
	a := preferenceTestApp(t)
	a.Preferences.Language = "zh-CN"
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	p := readTestPreferences(t, a.configDir)
	if p.Language != "zh-CN" || (i18n.Catalog{Language: p.Language}).Locale() != "zh-Hans" {
		t.Fatalf("中文选择在保存读回后丢失: %s", p.Language)
	}
}
