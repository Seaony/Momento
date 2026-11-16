package ui

import (
	"math"
	"testing"
)

// 铺满的行每张一样高、右边对齐到宽度；最后一行不拉伸；奇怪的比例不出 NaN；方向键上下找中线最近的那张。
func TestJustifiedLayout(t *testing.T) {
	ratios := []float32{1.5, 1, .75, 2, 1, 1, 0, float32(math.NaN()), 10, .1, 1.5}
	width := float32(900)
	l := newJustified(ratios, width, 1)
	if len(l.items) != len(ratios) || l.rows[len(l.rows)-1] != len(ratios) {
		t.Fatalf("行数据不对：%v", l.rows)
	}
	for r := 0; r < len(l.rows)-1; r++ {
		first, end := l.rows[r], l.rows[r+1]
		h := l.items[first].Height
		for i := first; i < end; i++ {
			it := l.items[i]
			if it.Height != h || it.Index != i || math.IsNaN(float64(it.Width)) || it.Width <= 0 {
				t.Fatalf("第 %d 行第 %d 张不对：%+v", r, i, it)
			}
			if i > first && math.Abs(float64(it.X-(l.items[i-1].X+l.items[i-1].Width+masonryGap))) > .01 {
				t.Fatalf("间距不对：%+v", it)
			}
		}
		last := l.items[end-1]
		if r < len(l.rows)-2 && math.Abs(float64(last.X+last.Width-width)) > .01 {
			t.Fatalf("第 %d 行没铺满：%v", r, last.X+last.Width)
		}
		if r == len(l.rows)-2 && (h != justifiedRowHeight || last.X+last.Width > width) {
			t.Fatalf("最后一行应保持目标高度且不超宽：%+v", last)
		}
		if r > 0 && l.items[first].Y <= l.items[l.rows[r-1]].Y {
			t.Fatal("行没有往下排")
		}
	}
	if got := l.Visible(0, 1); len(got) == 0 || got[0].Index != 0 {
		t.Fatal("顶部可见的卡片不对")
	}
	if _, ok := l.neighbor(0, true); ok {
		t.Fatal("第一行往上不该有")
	}
	down, ok := l.neighbor(0, false)
	if !ok || l.row(down) != 1 {
		t.Fatalf("往下应到第二行：%d", down)
	}
	if up, ok := l.neighbor(down, true); !ok || l.row(up) != 0 {
		t.Fatal("往上应回到第一行")
	}
	if empty := newJustified(nil, width, 1); empty.Height() != 0 || empty.Visible(0, 100) != nil {
		t.Fatal("没有素材时应是空的")
	}
}
