package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	passwordlock "github.com/Seaony/Momento/app/internal/lock"
	"github.com/egoist/mygo/ui"
)

func passwordTestApp(t *testing.T) (*App, chan func()) {
	t.Helper()
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, configDir: t.TempDir()}
	a.RestoreUIState()
	a.InitPasswordLock()
	posted := make(chan func(), 4)
	a.Lock.post = func(fn func()) { posted <- fn }
	return a, posted
}

func finishPasswordWork(t *testing.T, a *App, posted <-chan func()) {
	t.Helper()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for a.PasswordOperationPending() {
		select {
		case fn := <-posted:
			fn()
		case <-timeout.C:
			t.Fatal("密码后台任务没有完成")
		}
	}
}

func enableTestPassword(t *testing.T, a *App, posted <-chan func()) string {
	t.Helper()
	a.ShowPasswordSettings("enable")
	a.Lock.NewPassword, a.Lock.ConfirmPassword = "secret-4826", "secret-4826"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash == "" || a.Lock.DialogOpen {
		t.Fatalf("启用密码锁失败：%s", a.Lock.ErrorKey)
	}
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	return a.Preferences.PasswordHash
}

func TestPasswordSettingsValidateAndPersistOnlyHash(t *testing.T) {
	a, posted := passwordTestApp(t)
	a.ShowPasswordSettings("enable")
	a.Lock.NewPassword, a.Lock.ConfirmPassword = "123", "123"
	a.submitPasswordSettings()
	if a.Lock.ErrorKey != "password_minimum" || a.PasswordOperationPending() || a.Preferences.PasswordHash != "" {
		t.Fatal("短密码被接受")
	}
	a.Lock.NewPassword, a.Lock.ConfirmPassword = "1234", "5678"
	a.submitPasswordSettings()
	if a.Lock.ErrorKey != "password_mismatch" || a.PasswordOperationPending() {
		t.Fatal("两次密码不一致时仍开始保存")
	}
	hash := enableTestPassword(t, a, posted)
	if a.IsLocked() {
		t.Fatal("启用密码锁不应打断当前会话")
	}
	data, err := os.ReadFile(filepath.Join(a.configDir, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-4826") || !strings.Contains(string(data), "argon2id") {
		t.Fatal("偏好文件没有只保存密码哈希")
	}
	var reopened Preferences
	if err := json.Unmarshal(data, &reopened); err != nil {
		t.Fatal(err)
	}
	if reopened.PasswordHash != hash {
		t.Fatal("读回的密码哈希不同")
	}
	a.Preferences = reopened
	a.InitPasswordLock()
	if !a.IsLocked() {
		t.Fatal("有密码的启动没有进入锁屏")
	}
	a.Lock.Password = "secret-4826"
	a.unlockPassword()
	finishPasswordWork(t, a, posted)
	if a.IsLocked() || a.Lock.Password != "" {
		t.Fatal("正确密码没有解锁或仍残留输入")
	}
}

func TestPasswordLockRejectsWrongPasswordAcrossSurfaces(t *testing.T) {
	a, posted := passwordTestApp(t)
	hash := enableTestPassword(t, a, posted)
	for range 4 {
		a.ShowPasswordSettings("disable")
		a.Lock.CurrentPassword = "wrong"
		a.submitPasswordSettings()
		finishPasswordWork(t, a, posted)
		if a.Preferences.PasswordHash != hash || a.Lock.ErrorKey != "password_wrong" {
			t.Fatal("错误的当前密码关闭了密码锁")
		}
		a.closePasswordDialog()
	}
	a.PromptOpen, a.ConfirmOpen = true, true
	a.SettingsOpen = true
	a.LockNow()
	if !a.IsLocked() || a.PreviewOpen() || a.PromptOpen || a.ConfirmOpen || a.SettingsOpen || a.Lock.CurrentPassword != "" {
		t.Fatal("立即锁定没有清理原界面与密码输入")
	}
	a.Lock.Password = "wrong"
	a.unlockPassword()
	finishPasswordWork(t, a, posted)
	if !a.IsLocked() || a.passwordCooldown() < 29*time.Second {
		t.Fatal("切换到锁屏清除了已有错误次数")
	}
	a.Lock.Password = "secret-4826"
	a.unlockPassword()
	if a.PasswordOperationPending() || !a.IsLocked() {
		t.Fatal("冷却期仍允许验证")
	}
}

func TestPasswordChangeAndDisableRequireCurrentPassword(t *testing.T) {
	a, posted := passwordTestApp(t)
	oldHash := enableTestPassword(t, a, posted)
	a.ShowPasswordSettings("change")
	a.Lock.CurrentPassword, a.Lock.NewPassword, a.Lock.ConfirmPassword = "secret-4826", "新密码4826", "新密码4826"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash == oldHash || a.Lock.DialogOpen {
		t.Fatal("修改密码未完成")
	}
	valid, err := passwordlock.Verify(a.Preferences.PasswordHash, "新密码4826")
	if err != nil || !valid {
		t.Fatal("新密码无法验证")
	}
	a.ShowPasswordSettings("disable")
	a.Lock.CurrentPassword = "secret-4826"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash == "" {
		t.Fatal("旧密码仍能关闭密码锁")
	}
	a.Lock.CurrentPassword = "新密码4826"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash != "" || a.Lock.DialogOpen {
		t.Fatal("当前密码未能关闭密码锁")
	}
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordSaveFailurePreservesOriginalPassword(t *testing.T) {
	a, posted := passwordTestApp(t)
	hash := enableTestPassword(t, a, posted)
	if err := os.Mkdir(filepath.Join(a.configDir, "preferences.json.tmp"), 0700); err != nil {
		t.Fatal(err)
	}
	a.ShowPasswordSettings("disable")
	a.Lock.CurrentPassword = "secret-4826"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash != hash || !a.Lock.DialogOpen || a.Lock.ErrorKey != "password_save_failed" {
		t.Fatal("保存失败仍修改了密码或显示成功")
	}
	// 等最后一次偏好写入完成后再读，避免把排队写入当成已持久化。
	if err := a.FlushPreferences(); err == nil {
		t.Fatal("写入故障未向调用方报告")
	}
	data, err := os.ReadFile(filepath.Join(a.configDir, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved Preferences
	if err := json.Unmarshal(data, &saved); err != nil || saved.PasswordHash != hash {
		t.Fatal("写失败改坏了已有密码")
	}
}

func TestPasswordDialogCancelAndCooldown(t *testing.T) {
	a, _ := passwordTestApp(t)
	a.ShowPasswordSettings("enable")
	view := func(c *ui.Context) {
		a.applyTheme(c)
		ui.Column(c).Fill()
		a.passwordSettingsDialog(c)
	}
	tt := ui.NewTester(view, 760, 560)
	if err := tt.Click(a.T("password_new")); err != nil {
		t.Fatal(err)
	}
	tt.Type("do-not-keep-this")
	tt.Key(0, ui.KeyEscape)
	if a.Lock.DialogOpen || a.Lock.NewPassword != "" {
		t.Fatal("取消没有清掉密码输入")
	}
	a.ShowPasswordSettings("enable")
	tt.Frame()
	if err := tt.Click(a.T("password_new")); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if a.Lock.NewPassword != "" {
		t.Fatal("重新打开浮层后能撤销找回密码")
	}
	clock := time.Now()
	a.Lock.now = func() time.Time { return clock }
	a.Lock.CooldownUntil = clock.Add(30 * time.Second)
	a.Lock.NewPassword, a.Lock.ConfirmPassword = "1234", "1234"
	tt.Frame()
	if button, ok := tt.AccessByRole(ui.RoleButton, a.T("password_action_enable")); !ok || !button.Disabled {
		t.Fatal("冷却期间按钮未禁用")
	}
	clock = clock.Add(31 * time.Second)
	tt.Frame()
	if button, ok := tt.AccessByRole(ui.RoleButton, a.T("password_action_enable")); !ok || button.Disabled {
		t.Fatal("冷却结束后按钮仍禁用")
	}
}

func TestPasswordSavingDefersOtherPreferenceWrites(t *testing.T) {
	a, _ := passwordTestApp(t)
	a.Lock.Saving = true
	a.SidebarWidth = 280
	a.RememberUIState()
	if a.prefVersion.Load() != 0 {
		t.Fatal("保存密码期间仍排入了旧密码的偏好快照")
	}
	a.Lock.Saving = false
	if err := a.FlushPreferences(); err != nil {
		t.Fatal(err)
	}
	if got := readTestPreferences(t, a.configDir).SidebarWidth; got != 280 {
		t.Fatal("等待密码保存时丢失了界面状态")
	}
}

func TestPasswordLockScreenHidesApplicationActions(t *testing.T) {
	a, _ := passwordTestApp(t)
	a.Preferences.PasswordHash = "invalid-hash-still-locks"
	a.InitPasswordLock()
	a.ShowPasswordSettings("disable")
	if a.Lock.DialogOpen {
		t.Fatal("锁定时还能打开密码设置")
	}
	tt := ui.NewTester(a.View, 1100, 640)
	if !tt.HasText(a.T("password_locked")) || tt.HasText(a.T("new_library")) || tt.HasText(a.T("open_library")) {
		t.Fatal("锁屏仍显示了应用内容或操作")
	}
}
