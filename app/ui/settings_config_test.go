package ui

import (
	"errors"
	"strings"
	"testing"
)

func TestLaunchAtLoginUsesActualStateAfterFailure(t *testing.T) {
	previousSet, previousRead := setLaunchAtLogin, launchAtLogin
	t.Cleanup(func() { setLaunchAtLogin, launchAtLogin = previousSet, previousRead })
	for _, test := range []struct {
		name    string
		wanted  bool
		actual  bool
		failure bool
	}{
		{"enable", true, true, false},
		{"enable-failed", true, false, true},
		{"disable-failed", false, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := settingsTestApp("en", "light")
			a.Settings.LoginError = "previous failure"
			setLaunchAtLogin = func(value bool) error {
				if value != test.wanted {
					t.Fatalf("设置值错误：%v", value)
				}
				if test.failure {
					return errors.New("permission denied")
				}
				return nil
			}
			launchAtLogin = func() bool { return test.actual }
			a.SetLaunchAtLogin(test.wanted)
			if a.Settings.LoginEnabled != test.actual || a.Preferences.LaunchAtLogin != test.actual {
				t.Fatal("开关或偏好没有回到系统真实状态")
			}
			if test.failure != strings.Contains(a.Settings.LoginError, "permission denied") || !test.failure && a.Settings.LoginError != "" {
				t.Fatalf("失败原因没有正确更新：%q", a.Settings.LoginError)
			}
		})
	}
}
