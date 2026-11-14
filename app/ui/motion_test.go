package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestMotionColorUsesIntermediateFrames(t *testing.T) {
	target := ui.Hex("#3A2318")
	var shown ui.Color
	tt := ui.NewTester(func(c *ui.Context) {
		e := ui.Box(c.Key("motion")).Fill()
		shown = e.AnimateColor("test", target, motionColor)
		e.Background(shown)
	}, 80, 80)
	target = ui.Hex("#FFFAF5")
	tt.Frame()
	tt.Advance(40 * time.Millisecond)
	if shown == target || shown == ui.Hex("#3A2318") {
		t.Fatal("颜色直接跳到了端点")
	}
	tt.Advance(time.Second)
	if shown != target {
		t.Fatal("颜色动画没有抵达目标")
	}
}

// 侧栏、设置导航会显示选中态，它们用到的图标都要有填充版。
func TestIconFoundationHasBothWeights(t *testing.T) {
	for name, regular := range iconRegular {
		if regular == nil {
			t.Fatalf("%s 缺少线性版", name)
		}
	}
	for _, name := range []string{"images", "clock-counter-clockwise", "heart", "shuffle", "tray", "tag-simple", "tag", "folder", "folder-simple-star", "trash", "gear-six", "palette", "mouse-simple", "keyboard", "bell", "lock-simple", "globe", "info"} {
		if iconFill[name] == nil {
			t.Fatalf("%s 缺少填充版", name)
		}
	}
}

func TestNoticeUsesAnimatedEntryAndExit(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	add := false
	tt := ui.NewTester(func(c *ui.Context) {
		a.applyTheme(c)
		ui.Box(c).Fill().Background(c.Theme().Background)
		if add {
			c.Toast("保存成功")
			add = false
		}
		a.noticeViewport(c)
	}, 640, 480)
	before := append([]byte(nil), tt.Image().Pix...)
	add = true
	tt.Frame()
	tt.Advance(87 * time.Millisecond)
	middle := append([]byte(nil), tt.Image().Pix...)
	tt.Advance(time.Second)
	if bytes.Equal(before, middle) || bytes.Equal(middle, tt.Image().Pix) {
		t.Fatal("提示条没有淡入滑动中间帧")
	}
	if err := tt.Click(a.T("close")); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if tt.HasText("保存成功") {
		t.Fatal("提示条关闭后仍存在")
	}
}
