package ui

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/plugins/updater/native"
)

type UpdateCheckState string

const (
	UpdateIdle         UpdateCheckState = "idle"
	UpdateChecking     UpdateCheckState = "checking"
	UpdateLatest       UpdateCheckState = "latest"
	UpdateAvailable    UpdateCheckState = "available"
	UpdateFailed       UpdateCheckState = "failed"
	UpdateUnconfigured UpdateCheckState = "unconfigured"
)

type UpdateCheckStatus struct {
	State   UpdateCheckState
	Version string
	Err     error
}

type updateChecker struct {
	mu                sync.Mutex
	status            UpdateCheckStatus
	nativeVersionSeen string
	nativeLastCheck   time.Time
	listeners         map[uint64]func()
	nextListener      uint64
	enabled           func() bool
	check             func(context.Context) (*mygo.Update, error)
	dispatch          func(func())
}

func newUpdateChecker(enabled func() bool, check func(context.Context) (*mygo.Update, error), dispatch func(func())) *updateChecker {
	return &updateChecker{status: UpdateCheckStatus{State: UpdateIdle}, enabled: enabled, check: check, dispatch: dispatch, listeners: make(map[uint64]func())}
}

var updates = newUpdateChecker(mygo.Updater.Enabled, mygo.Updater.Check, mygo.RunOnMain)
var setAutomaticUpdateChecks = updater.SetAutomaticChecks

// 在 App.Run 前注册，更新窗口也使用原生界面。
func InitUpdater() {
	mygo.Use(native.New(updater.Options{DisableAutomaticChecks: true}))
	// 偏好加载前先关掉自动检查，加载后再按应用设置启用。
	updater.SetAutomaticChecks(false)
	updates.mu.Lock()
	updates.nativeVersionSeen, updates.nativeLastCheck = updater.AvailableVersion(), updater.LastCheck()
	updates.mu.Unlock()
	updater.OnChange(func() {
		version, checked := updater.AvailableVersion(), updater.LastCheck()
		slog.Info("更新状态已变化", "availableVersion", version, "lastCheck", checked)
		updates.nativeVersion(version, checked)
	})
	updater.OnInstalled(func() { slog.Info("更新已安装，等待重新启动") })
}

func CheckForUpdates() {
	slog.Info("手动检查更新")
	updates.start()
}

// 原生更新器会在启动后检查，并继续保留跳过版本和自动下载的选择。
func StartAutomaticUpdateCheck(enabled bool) {
	setAutomaticUpdateChecks(enabled)
}

// 检查与安装分开，只有用户选择更新时才打开安装窗口。
func ShowAvailableUpdate() {
	if UpdateStatus().State == UpdateChecking {
		return
	}
	updater.CheckForUpdates()
}

func UpdateStatus() UpdateCheckStatus { return updates.current() }

func UpdateMenuItem() *mygo.MenuItem {
	item := updater.MenuItem()
	item.Click = func(*mygo.MenuItem, *mygo.Window) { CheckForUpdates() }
	return item
}

func AvailableUpdateVersion() string { return UpdateStatus().Version }

func OnUpdateChange(changed func()) func() { return updates.onChange(changed) }

func (u *updateChecker) current() UpdateCheckStatus {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.status
}

func (u *updateChecker) start() {
	u.mu.Lock()
	if u.status.State == UpdateChecking {
		u.mu.Unlock()
		return
	}
	u.status = UpdateCheckStatus{State: UpdateChecking}
	u.mu.Unlock()
	u.changed()
	go func() {
		if !u.enabled() {
			u.finish(nil, mygo.ErrUpdatesDisabled)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		update, err := u.check(ctx)
		u.finish(update, err)
	}()
}

func (u *updateChecker) finish(update *mygo.Update, err error) {
	status := UpdateCheckStatus{State: UpdateLatest}
	switch {
	case errors.Is(err, mygo.ErrUpdatesDisabled):
		status.State = UpdateUnconfigured
	case err != nil:
		status.State, status.Err = UpdateFailed, err
		slog.Error("检查更新失败", "error", err)
	case update != nil:
		status.State, status.Version = UpdateAvailable, update.Version
	}
	u.mu.Lock()
	u.status = status
	u.mu.Unlock()
	u.changed()
}

func (u *updateChecker) nativeVersion(version string, checked time.Time) {
	u.mu.Lock()
	if version == u.nativeVersionSeen && checked.Equal(u.nativeLastCheck) {
		u.mu.Unlock()
		return
	}
	u.nativeVersionSeen, u.nativeLastCheck = version, checked
	if u.status.State == UpdateChecking {
		u.mu.Unlock()
		return
	}
	u.status = UpdateCheckStatus{State: UpdateIdle}
	if version != "" {
		u.status.State, u.status.Version = UpdateAvailable, version
	}
	u.mu.Unlock()
	u.changed()
}

func (u *updateChecker) onChange(fn func()) func() {
	u.mu.Lock()
	u.nextListener++
	id := u.nextListener
	u.listeners[id] = fn
	u.mu.Unlock()
	return func() {
		u.mu.Lock()
		delete(u.listeners, id)
		u.mu.Unlock()
	}
}

func (u *updateChecker) changed() {
	u.mu.Lock()
	listeners := make([]func(), 0, len(u.listeners))
	for _, listener := range u.listeners {
		listeners = append(listeners, listener)
	}
	u.mu.Unlock()
	if len(listeners) > 0 {
		u.dispatch(func() {
			for _, listener := range listeners {
				listener()
			}
		})
	}
}
