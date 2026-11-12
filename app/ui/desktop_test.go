package ui

import (
	"github.com/egoist/mygo"
	"testing"
)

func TestExplicitQuitDoesNotHideAndCancelWindowClose(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	event := &mygo.CloseEvent{}
	a.closeMainWindow(event)
	if !event.DefaultPrevented() {
		t.Fatal("普通关窗不应退出进程")
	}
	a.Quitting = true
	canceled := false
	a.cancel = func() { canceled = true }
	event = &mygo.CloseEvent{}
	a.closeMainWindow(event)
	if event.DefaultPrevented() || !canceled {
		t.Fatal("显式退出被后台运行拦截，或未停止后台任务")
	}
}
