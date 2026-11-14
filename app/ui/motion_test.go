package ui

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"time"
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

func TestPhosphorFoundationHasBothWeights(t *testing.T) {
	for name, regular := range phosphorRegular {
		if regular == nil || phosphorFill[name] == nil {
			t.Fatalf("%s 缺少Regular或Fill", name)
		}
	}
}
