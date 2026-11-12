package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/egoist/mygo/ui"
)

func TestPasswordSaveErrorDetailsAreLocalized(t *testing.T) {
	for _, locale := range []struct{ language, full, permission string }{
		{"zh-CN", "磁盘空间不足，请腾出空间后重试。", "无法写入配置目录，请检查文件夹权限。"},
		{"en", "The disk is full. Free up space and try again.", "Cannot write to the configuration folder. Check its permissions."},
		{"ja", "ディスクの空き容量が不足しています。空き容量を増やして再試行してください。", "設定フォルダに書き込めません。フォルダのアクセス権を確認してください。"},
	} {
		for _, failure := range []struct {
			name string
			err  error
			want string
		}{
			{"full", syscall.ENOSPC, locale.full},
			{"wrapped-full", &os.PathError{Op: "write", Path: "preferences.json", Err: syscall.ENOSPC}, locale.full},
			{"permission", os.ErrPermission, locale.permission},
			{"wrapped-permission", &os.PathError{Op: "write", Path: "preferences.json", Err: syscall.EACCES}, locale.permission},
			{"operation-not-permitted", fmt.Errorf("write preferences: %w", syscall.EPERM), locale.permission},
			{"unknown", errors.New("device disconnected"), "device disconnected"},
			{"none", nil, ""},
		} {
			t.Run(locale.language+"/"+failure.name, func(t *testing.T) {
				a := &App{Texts: i18n.Catalog{Language: locale.language}}
				if got := a.passwordErrorDetail(failure.err); got != failure.want {
					t.Fatalf("密码错误详情：%q，应为 %q", got, failure.want)
				}
				if failure.err != nil {
					a.Lock.ErrorKey, a.Lock.Detail = "password_save_failed", a.passwordErrorDetail(failure.err)
					tt := ui.NewTester(a.passwordFeedback, 420, 180)
					if !tt.HasText(a.T("password_save_failed", failure.want)) {
						t.Fatal("密码错误没有使用对应语言的详情")
					}
				}
			})
		}
	}
}

func TestPasswordPermissionFailureKeepsOldPasswordAndShowsLocalizedDetail(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("此检查使用 Unix 文件夹写权限，需要普通用户运行")
	}
	a, posted := passwordTestApp(t)
	a.Texts.Language = "ja"
	hash := enableTestPassword(t, a, posted)
	before, err := os.ReadFile(filepath.Join(a.configDir, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(a.configDir, 0500); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chmod(a.configDir, 0700); err != nil {
			t.Error(err)
		}
	}()
	a.ShowPasswordSettings("disable")
	a.Lock.CurrentPassword = "secret-4826"
	a.submitPasswordSettings()
	finishPasswordWork(t, a, posted)
	if a.Preferences.PasswordHash != hash || !a.Lock.DialogOpen || a.Lock.ErrorKey != "password_save_failed" || a.Lock.Detail != a.T("password_permission_denied") {
		t.Fatalf("保存权限错误丢失原密码或没有本地化：%q", a.Lock.Detail)
	}
	after, err := os.ReadFile(filepath.Join(a.configDir, "preferences.json"))
	if err != nil || string(after) != string(before) {
		t.Fatal("保存权限失败改写了原来的偏好")
	}
}
