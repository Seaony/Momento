package ui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// HasText 是子串匹配，「已恢复默认」会命中「恢复默认」；要整句相等时用这个。
func hasExactText(tt *ui.Tester, text string) bool {
	return slices.Contains(tt.Texts(), text)
}

func settingsTester(a *App) *ui.Tester {
	tt := ui.NewTester(func(c *ui.Context) { a.settingsView(c, UpdateCheckStatus{}) }, 760, 560)
	tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 30})
	tt.SetWindowActive(true)
	tt.Frame()
	return tt
}

func shortcutRowTester(a *App, action string) *ui.Tester {
	definition, _ := shortcutDefinition(action)
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			a.shortcutRow(c, definition)
			ui.Box(c.Key("blank")).Size(400, 80)
		})
	}, 600, 240)
}

func TestShortcutRecordingCancelsWhenWindowLosesFocus(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	tt := shortcutRowTester(a, "search")
	if err := tt.Click(a.T("shortcut_binding", a.T("search"))); err != nil {
		t.Fatal(err)
	}
	if a.Settings.RecordingShortcut != "search" {
		t.Fatal("点了录制按钮没有开始录制")
	}
	tt.SetWindowActive(false)
	tt.Frame()
	if a.Settings.RecordingShortcut != "" {
		t.Fatal("设置窗口失焦后还在录制")
	}
}

func TestShortcutRecordingCancelsWhenPressingElsewhere(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	tt := shortcutRowTester(a, "search")
	if err := tt.Click(a.T("shortcut_binding", a.T("search"))); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(100, 180)
	tt.Frame()
	if a.Settings.RecordingShortcut != "" {
		t.Fatal("点别处后还在录制")
	}
}

func TestShortcutConflictOffersUseAnyway(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	tt := shortcutRowTester(a, "show_app")
	if err := tt.Click(a.T("shortcut_binding", a.T("shortcut_show_app"))); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyN)
	tt.Advance(time.Second)
	message := a.T("shortcut_conflict_in_use", displayShortcut("Super+N"), a.T("new_library"))
	if a.Settings.RecordingShortcut != "show_app" || !tt.HasText(message) || !tt.HasText(a.T("shortcut_use_anyway")) {
		t.Fatalf("冲突后应保持录制并显示说明和按钮：%q", a.Settings.ShortcutErrors["show_app"])
	}
	if err := tt.Click(a.T("shortcut_use_anyway")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Shortcut("new_library") != "" || a.Shortcut("show_app") != "Super+N" || a.Settings.RecordingShortcut != "" || len(a.Settings.ShortcutErrors) != 0 {
		t.Fatalf("仍然使用没有把快捷键抢过来：new=%q show=%q recording=%q errors=%v", a.Shortcut("new_library"), a.Shortcut("show_app"), a.Settings.RecordingShortcut, a.Settings.ShortcutErrors)
	}
}

func TestShortcutEscapeClearsConflictText(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	tt := shortcutRowTester(a, "show_app")
	if err := tt.Click(a.T("shortcut_binding", a.T("shortcut_show_app"))); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyN)
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.Settings.RecordingShortcut != "" || a.Settings.ShortcutErrors["show_app"] != "" || tt.HasText(a.T("shortcut_use_anyway")) {
		t.Fatal("Esc 取消录制后冲突提示还在")
	}
}

func TestGlobalShortcutNeedsModifier(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	for _, accelerator := range []string{"A", "Shift+A", "Space"} {
		if a.SetShortcut("show_app", accelerator) {
			t.Fatalf("全局快捷键接受了没有修饰键的 %s", accelerator)
		}
	}
	if a.Settings.ShortcutErrors["show_app"] != a.T("shortcut_global_modifier") {
		t.Fatalf("提示不对：%q", a.Settings.ShortcutErrors["show_app"])
	}
	if !a.SetShortcut("show_app", "Alt+A") || !a.SetShortcut("search", "A") {
		t.Fatal("带修饰键的全局快捷键，或非全局的单键被拒绝")
	}
}

func TestShortcutGroupRestoreOnlyWhenCustomized(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	a.Settings.Tab = SettingsShortcuts
	tt := settingsTester(a)
	if hasExactText(tt, a.T("restore_defaults")) {
		t.Fatal("没有自定义时不该显示恢复默认")
	}
	a.SetShortcut("export", "Alt+G")
	tt.Frame()
	if !hasExactText(tt, a.T("restore_defaults")) {
		t.Fatal("有自定义的组没有恢复默认")
	}
	if err := tt.Click(a.T("restore_defaults")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Shortcut("export") != "CmdOrCtrl+E" || !hasExactText(tt, a.T("shortcut_restored")) || hasExactText(tt, a.T("restore_defaults")) {
		t.Fatalf("恢复后没有提示「已恢复默认」，或按钮还在：%q notes=%v texts=%v", a.Shortcut("export"), a.Settings.ShortcutGroupNotes, tt.Texts())
	}
}

func TestShortcutSearchWithoutMatch(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	a.Settings.Tab = SettingsShortcuts
	a.Settings.ShortcutSearch = "xyz-not-found"
	tt := settingsTester(a)
	if !tt.HasText(a.T("shortcut_no_match")) {
		t.Fatal("搜索没有结果时没有说明")
	}
}

func TestViewShortcutsFollowToolbarOrder(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	for action, want := range map[string]string{"view_masonry": "CmdOrCtrl+1", "view_justified": "CmdOrCtrl+2", "view_list": "CmdOrCtrl+3"} {
		if got := a.Shortcut(action); got != want {
			t.Fatalf("%s 默认是 %s，应为 %s", action, got, want)
		}
	}
	var order []string
	for _, definition := range ShortcutDefinitions {
		if definition.Group == "view" && (definition.Action == "view_masonry" || definition.Action == "view_justified" || definition.Action == "view_list") {
			order = append(order, definition.Action)
		}
	}
	if len(order) != 3 || order[0] != "view_masonry" || order[1] != "view_justified" || order[2] != "view_list" {
		t.Fatalf("设置页顺序不对：%v", order)
	}
}

func TestLockScreenForgotPasswordPath(t *testing.T) {
	a, posted := passwordTestApp(t)
	enableTestPassword(t, a, posted)
	a.LockNow()
	tt := ui.NewTester(a.lockScreen, 760, 560)
	if tt.HasText(a.T("password_forgot")) {
		t.Fatal("还没输错就出现了忘记密码")
	}
	a.Lock.Password = "wrong"
	a.unlockPassword()
	finishPasswordWork(t, a, posted)
	tt.Frame()
	if !tt.HasText(a.T("password_forgot")) {
		t.Fatal("输错一次后没有「忘记密码？」")
	}
	if err := tt.Click(a.T("password_forgot")); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(a.T("password_forgot_help")) || !tt.HasText(a.T("password_show_config")) {
		t.Fatal("点开后没有恢复办法和显示配置文件夹按钮")
	}
	a.Lock.ForgotOpen, a.Lock.Rejections = false, 0
	a.Lock.ErrorKey = "password_invalid"
	tt.Frame()
	if !tt.HasText(a.T("password_forgot_help")) {
		t.Fatal("配置损坏时没有直接给出路")
	}
}

func TestPasswordDialogOnlyShowsInRequestingWindow(t *testing.T) {
	a, _ := passwordTestApp(t)
	a.ShowPasswordSettings("enable")
	main := ui.NewTester(func(c *ui.Context) { a.passwordSettingsDialog(c, false) }, 760, 560)
	settings := ui.NewTester(func(c *ui.Context) { a.passwordSettingsDialog(c, true) }, 760, 560)
	if main.HasText(a.T("password_new")) {
		t.Fatal("从设置页打开的密码对话框也出现在主窗口")
	}
	if !settings.HasText(a.T("password_new")) || !settings.HasText(a.T("password_dialog_warning")) {
		t.Fatal("设置窗口里没有密码对话框，或缺少忘记密码的提醒")
	}
}

func TestPasswordWrongCurrentKeepsNewPassword(t *testing.T) {
	a, posted := passwordTestApp(t)
	hash := enableTestPassword(t, a, posted)
	a.ShowPasswordSettings("change")
	a.Lock.CurrentPassword, a.Lock.NewPassword, a.Lock.ConfirmPassword = "wrong", "new-pass-1", "new-pass-1"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash != hash || a.Lock.ErrorKey != "password_wrong" {
		t.Fatal("当前密码错误时不该改密码")
	}
	if a.Lock.CurrentPassword != "" || a.Lock.NewPassword != "new-pass-1" || a.Lock.ConfirmPassword != "new-pass-1" {
		t.Fatalf("当前密码错误只该清当前密码：%q %q %q", a.Lock.CurrentPassword, a.Lock.NewPassword, a.Lock.ConfirmPassword)
	}
}

func TestPasswordSavedNoticeShowsInSettings(t *testing.T) {
	a, posted := passwordTestApp(t)
	enableTestPassword(t, a, posted)
	if a.Settings.PasswordNotice != a.T("password_saved_enable") || a.Notice != "" {
		t.Fatalf("设置里完成后的反馈应显示在设置窗口：notice=%q main=%q", a.Settings.PasswordNotice, a.Notice)
	}
	a.Settings.Tab = SettingsPassword
	tt := settingsTester(a)
	if !tt.HasText(a.T("password_saved_enable")) || !tt.HasText(a.T("lock_now")) || !tt.HasText(a.T("password_lock_hint")) {
		t.Fatal("密码锁页缺少状态反馈或立即锁定按钮")
	}
	a.Settings.PasswordNoticeAt = time.Now().Add(-time.Minute)
	tt.Frame()
	if tt.HasText(a.T("password_saved_enable")) {
		t.Fatal("反馈几秒后没有消失")
	}
}

func TestCountAutoImportFilesFollowsWatcherRules(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"a.png": "x", "b.txt": "x", ".hidden.png": "x", "sub/c.jpg": "x", ".hiddendir/d.png": "x", "Lib.momento/e.png": "x",
	} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	count, err := countAutoImportFiles(root)
	if err != nil || count != 2 {
		t.Fatalf("数到 %d 个，想要 2 个（a.png、sub/c.jpg）：%v", count, err)
	}
}

func autoImportTestApp(language string) *App {
	a := settingsTestApp(language, "light")
	a.Settings.Tab = SettingsImport
	a.Library = &library.Library{Manifest: model.Manifest{LibraryID: "lib"}, Root: "/lib.momento"}
	a.Snapshot.Folders = []model.Folder{{ID: "p", Name: "Parent"}, {ID: "c", Name: "Child", ParentID: "p"}}
	return a
}

func TestAutoImportConfirmBeforeAdding(t *testing.T) {
	a := autoImportTestApp("zh-CN")
	a.Settings.AutoImportConfirm = &autoImportConfirm{Path: "/Users/x/下载", Count: 7, LibraryID: "lib"}
	tt := settingsTester(a)
	if !tt.HasText(a.T("auto_import_confirm", "下载", 7)) || len(a.Preferences.AutoImport["lib"]) != 0 {
		t.Fatal("确认前不该添加规则，且要说清现有文件数")
	}
	if err := tt.Click(a.T("cancel")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Settings.AutoImportConfirm != nil || len(a.Preferences.AutoImport["lib"]) != 0 {
		t.Fatal("取消后不该添加规则")
	}
	a.Settings.AutoImportConfirm = &autoImportConfirm{Path: "/Users/x/下载", Count: 7, LibraryID: "lib"}
	tt.Frame()
	if err := tt.Click(a.T("auto_import_confirm_add")); err != nil {
		t.Fatal(err)
	}
	rules := a.Preferences.AutoImport["lib"]
	if len(rules) != 1 || rules[0].Path != "/Users/x/下载" || !rules[0].Enabled {
		t.Fatalf("确认后没有添加规则：%+v", rules)
	}
}

func TestAutoImportConfirmEmptyFolderText(t *testing.T) {
	a := autoImportTestApp("zh-CN")
	a.Settings.AutoImportConfirm = &autoImportConfirm{Path: "/Users/x/下载", Count: 0, LibraryID: "lib"}
	tt := settingsTester(a)
	if !tt.HasText(a.T("auto_import_confirm_empty", "下载")) {
		t.Fatal("空文件夹的确认文案不对")
	}
}

func TestAutoImportMissingFolderCanBeReplacedKeepingSettings(t *testing.T) {
	a := autoImportTestApp("zh-CN")
	missing := filepath.Join(t.TempDir(), "gone")
	a.Preferences.AutoImport = map[string][]AutoImportRule{"lib": {{Path: missing, FolderID: "c", Enabled: false}, {Path: "/other", Enabled: true}}}
	a.Settings.AutoImportErrors = map[string]string{missing: "lstat: no such file or directory"}
	a.Settings.AutoImportMissing = map[string]bool{missing: true}
	tt := settingsTester(a)
	if !tt.HasText(a.T("auto_import_missing")) || !tt.HasText(a.T("auto_import_choose_again")) || tt.HasText("lstat: no such file or directory") {
		t.Fatal("文件夹不存在时应显示友好说明和「重新选择…」")
	}
	if !tt.HasText("Parent / Child") {
		t.Fatal("目标文件夹没有显示完整层级")
	}
	a.commitAutoImportFolder(&autoImportConfirm{Path: "/new", Replace: missing, LibraryID: "lib"})
	rules := a.Preferences.AutoImport["lib"]
	if len(rules) != 2 || rules[0].Path != "/new" || rules[0].FolderID != "c" || rules[0].Enabled || a.Settings.AutoImportErrors[missing] != "" || a.Settings.AutoImportMissing[missing] {
		t.Fatalf("重新选择应只换路径：%+v", rules)
	}
}

func TestGeneralSettingsHideBackgroundOptionsWhenQuitting(t *testing.T) {
	before := settingsPlatform
	t.Cleanup(func() { settingsPlatform = before })
	settingsPlatform = func() string { return "darwin" }
	a := settingsTestApp("zh-CN", "light")
	a.Preferences.CloseAction = "quit"
	tt := settingsTester(a)
	if tt.HasText(a.T("tray_icon")) || tt.HasText(a.T("dock_when_hidden")) || tt.HasText(a.T("background_hint")) {
		t.Fatal("选「退出」时仍显示托盘、Dock 和后台说明")
	}
	if hasExactText(tt, a.T("check_updates")) {
		t.Fatal("通用页不该再放版本和检查更新")
	}
	a.Preferences.CloseAction, a.Preferences.TrayIcon = "background", false
	tt.Frame()
	if !tt.HasText(a.T("dock_required_hint")) {
		t.Fatal("没有菜单栏图标时强制开启 Dock 的原因没说明")
	}
}

func TestTrayCannotBeOffWhileRunningInBackgroundOffMac(t *testing.T) {
	before := settingsPlatform
	t.Cleanup(func() { settingsPlatform = before })
	settingsPlatform = func() string { return "windows" }
	a := settingsTestApp("zh-CN", "light")
	a.Preferences.CloseAction, a.Preferences.TrayIcon = "background", false
	tt := settingsTester(a)
	toggle, ok := tt.AccessByRole(ui.RoleSwitch, a.T("system_tray_icon"))
	if !ok || !toggle.Disabled || !toggle.Checked || !a.Preferences.TrayIcon || !tt.HasText(a.T("tray_required_hint")) {
		t.Fatal("后台运行时托盘图标必须开启且不能关，并说明原因")
	}
}

func TestExtensionPageHasInstallLinkAndHidesAddress(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	a.Settings.Tab = SettingsExtension
	a.ExtensionStatus = browser.Status{State: browser.Running, Address: browser.Address}
	a.Preferences.BrowserExtensionEnabled = false
	tt := settingsTester(a)
	if !tt.HasText(a.T("extension_get")) || tt.HasText(browser.Address) || tt.HasText(a.T("extension_picker")) {
		t.Fatal("扩展页应有获取链接，不显示本机地址，扩展关闭时不显示先选文件夹和标签")
	}
	a.Preferences.BrowserExtensionEnabled = true
	tt.Frame()
	if !tt.HasText(a.T("extension_picker")) {
		t.Fatal("扩展开启后没有先选文件夹和标签这一项")
	}
	a.Settings.Tab = SettingsImport
	tt.Frame()
	if tt.HasText(a.T("extension_picker")) {
		t.Fatal("这一项应从导入页挪走")
	}
}

func TestAppearanceAndSidebarPagesAfterRegrouping(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	a.Settings.Tab = SettingsAppearance
	tt := settingsTester(a)
	if tt.HasText(a.T("sidebar_counts")) {
		t.Fatal("外观页不该再有「在侧栏显示数量」")
	}
	a.Settings.Tab = SettingsSidebar
	tt.Frame()
	if !tt.HasText(a.T("sidebar_counts")) {
		t.Fatal("侧栏页缺少「在侧栏显示数量」")
	}
}

// 录制中的提示比平时的组合键长，三种语言下都要完整显示，不挤掉左边的名称。
func TestShortcutRecordingLabelFitsInEveryLanguage(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, definition := range ShortcutDefinitions {
			a := settingsTestApp(language, "light")
			a.Settings.Tab = SettingsShortcuts
			a.Settings.RecordingShortcut = definition.Action
			tt := settingsTester(a)
			recording, ok := tt.Find(a.T("shortcut_recording"))
			if !ok || recording.X+recording.W > 736 {
				t.Fatalf("%s/%s 录制提示被截断：%+v", language, definition.Action, recording)
			}
			// 左边的名称（内容区里的那个，不是导航里的同名页签）最多折成两行，不能被截断。
			for _, item := range tt.AccessAll() {
				if item.Name == a.T(definition.Key) && item.Bounds.X > 200 && item.Bounds.H > 34 {
					t.Fatalf("%s/%s 名称被挤成三行以上：%+v", language, definition.Action, item.Bounds)
				}
			}
		}
	}
}

func TestExtensionFailureRowFitsInEveryLanguage(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		a := settingsTestApp(language, "light")
		a.Settings.Tab = SettingsExtension
		a.Preferences.BrowserExtensionEnabled = true
		a.ExtensionStatus = browser.Status{State: browser.PortOccupied, Err: os.ErrExist}
		tt := settingsTester(a)
		for _, label := range []string{a.T("extension_retry"), a.T("extension_copy_details"), a.T("extension_port_in_use_retry")} {
			bounds, ok := tt.Find(label)
			if !ok || bounds.W == 0 || bounds.X+bounds.W > 736 {
				t.Fatalf("%s 「%s」被截断：%+v", language, label, bounds)
			}
		}
	}
}

func TestPasswordPageNoticeFitsInEveryLanguage(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		a := settingsTestApp(language, "light")
		a.Settings.Tab = SettingsPassword
		a.Preferences.PasswordHash = "hash"
		a.Settings.PasswordNotice, a.Settings.PasswordNoticeAt = a.T("password_saved_disable"), time.Now()
		tt := settingsTester(a)
		for _, label := range []string{a.T("password_saved_disable"), a.T("lock_now"), a.T("password_lock_enabled")} {
			bounds, ok := tt.Find(label)
			if !ok || bounds.W == 0 || bounds.X+bounds.W > 736 {
				t.Fatalf("%s 「%s」被截断：%+v", language, label, bounds)
			}
		}
	}
}
