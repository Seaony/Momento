package ui

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func waitUpdateState(t *testing.T, states <-chan UpdateCheckStatus, want UpdateCheckState) UpdateCheckStatus {
	t.Helper()
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case status := <-states:
			if status.State == want {
				return status
			}
		case <-deadline.C:
			t.Fatalf("更新检查没有到达%s", want)
		}
	}
}

func TestUpdateCheckerResults(t *testing.T) {
	failure := errors.New("network unavailable")
	for _, tc := range []struct {
		name    string
		enabled bool
		update  *mygo.Update
		err     error
		want    UpdateCheckState
	}{
		{"latest", true, nil, nil, UpdateLatest},
		{"available", true, &mygo.Update{Version: "2.1.0"}, nil, UpdateAvailable},
		{"failed", true, nil, failure, UpdateFailed},
		{"disabled", false, nil, nil, UpdateUnconfigured},
		{"disabled_during_check", true, nil, mygo.ErrUpdatesDisabled, UpdateUnconfigured},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			u := newUpdateChecker(func() bool { return tc.enabled }, func(context.Context) (*mygo.Update, error) {
				calls.Add(1)
				return tc.update, tc.err
			}, func(fn func()) { fn() })
			if got := u.current(); got.State != UpdateIdle || got.Version != "" || got.Err != nil {
				t.Fatalf("初始状态错误：%+v", got)
			}
			states := make(chan UpdateCheckStatus, 3)
			u.onChange(func() { states <- u.current() })
			u.start()
			if got := <-states; got.State != UpdateChecking {
				t.Fatalf("未先通知检查中：%+v", got)
			}
			got := waitUpdateState(t, states, tc.want)
			if tc.update != nil && got.Version != tc.update.Version {
				t.Fatalf("版本丢失：%+v", got)
			}
			if tc.want == UpdateFailed && !errors.Is(got.Err, failure) {
				t.Fatalf("原始错误丢失：%v", got.Err)
			}
			if !tc.enabled && calls.Load() != 0 {
				t.Fatal("未配置更新时仍然发起了检查")
			}
		})
	}
}

func TestUpdateCheckerCoalescesAndAllowsManualRecheck(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var calls atomic.Int32
	u := newUpdateChecker(func() bool { return true }, func(context.Context) (*mygo.Update, error) {
		calls.Add(1)
		started <- struct{}{}
		<-release
		return nil, nil
	}, func(fn func()) { fn() })
	states := make(chan UpdateCheckStatus, 8)
	u.onChange(func() { states <- u.current() })
	u.start()
	<-started
	u.start()
	u.start()
	if calls.Load() != 1 {
		t.Fatal("检查中重复发起请求")
	}
	close(release)
	waitUpdateState(t, states, UpdateLatest)
	u.start()
	<-started
	waitUpdateState(t, states, UpdateLatest)
	if calls.Load() != 2 {
		t.Fatal("完成检查后不能手动重查")
	}
}

func TestAutomaticUpdatePreferenceDelegatesBothDirections(t *testing.T) {
	previous := setAutomaticUpdateChecks
	t.Cleanup(func() { setAutomaticUpdateChecks = previous })
	var forwarded []bool
	setAutomaticUpdateChecks = func(enabled bool) { forwarded = append(forwarded, enabled) }
	StartAutomaticUpdateCheck(false)
	StartAutomaticUpdateCheck(true)
	StartAutomaticUpdateCheck(false)
	if !slices.Equal(forwarded, []bool{false, true, false}) {
		t.Fatalf("自动检查开关未交给原生更新器：%v", forwarded)
	}
}

func TestUpdateCheckerClearsOldVersionAndRemovesListener(t *testing.T) {
	u := newUpdateChecker(nil, nil, func(fn func()) { fn() })
	var notifications int
	off := u.onChange(func() { notifications++ })
	checked := time.Now()
	u.nativeVersion("2.1.0", checked)
	if got := u.current(); got.State != UpdateAvailable || got.Version != "2.1.0" {
		t.Fatalf("原生更新窗口的可用版本未同步：%+v", got)
	}
	u.nativeVersion("", checked)
	if got := u.current(); got.State != UpdateIdle || got.Version != "" {
		t.Fatalf("跳过版本后仍显示更新：%+v", got)
	}
	u.finish(&mygo.Update{Version: "2.2.0"}, nil)
	u.finish(nil, errors.New("check failed"))
	if got := u.current(); got.State != UpdateFailed || got.Version != "" {
		t.Fatalf("检查失败后保留过期版本：%+v", got)
	}
	before := notifications
	off()
	u.finish(nil, nil)
	if notifications != before {
		t.Fatal("已移除的监听仍收到通知")
	}
}

func TestNativeUpdatePreferencesDoNotChangeCheckResult(t *testing.T) {
	u := newUpdateChecker(nil, nil, func(fn func()) { fn() })
	checked := time.Now()
	u.nativeVersion("", checked)
	if u.current().State != UpdateIdle {
		t.Fatal("原生返回空版本也可能是跳过版本，不能声称已是最新")
	}
	u.finish(&mygo.Update{Version: "2.1.0"}, nil)
	u.nativeVersion("", checked)
	if got := u.current(); got.State != UpdateAvailable || got.Version != "2.1.0" {
		t.Fatalf("只改原生偏好清掉了手动检查结果：%+v", got)
	}
	u.mu.Lock()
	u.status = UpdateCheckStatus{State: UpdateChecking}
	u.mu.Unlock()
	u.nativeVersion("2.2.0", checked.Add(time.Minute))
	if u.current().State != UpdateChecking {
		t.Fatal("原生事件提前结束了仍在进行的手动检查")
	}
}
