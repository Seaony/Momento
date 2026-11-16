package ui

import (
	"math"
	"sort"
)

const (
	justifiedRowHeight float32 = 200 // 缩略图大小 1 时每行的目标高度
	justifiedMinRatio  float32 = .33 // 太窄太长的图按这个比例排，图片在卡片里裁切显示
	justifiedMaxRatio  float32 = 4
)

// justifiedLayout 是等高行：每行的图一样高，宽度按比例，整行刚好铺满；最后一行不拉伸，保持目标高度。
// items 按素材顺序，也按从上到下、从左到右排。
type justifiedLayout struct {
	items  []MasonryItem
	rows   []int // 每行第一张在 items 里的下标；最后多一项 len(items)
	height float32
}

func newJustified(ratios []float32, width, scale float32) *justifiedLayout {
	scale = max(.5, min(2, scale))
	target := justifiedRowHeight * scale
	width = max(width, masonryMinWidth*scale)
	layout := &justifiedLayout{items: make([]MasonryItem, len(ratios))}
	ratio := func(i int) float32 {
		r := ratios[i]
		if r <= 0 || math.IsInf(float64(r), 0) || math.IsNaN(float64(r)) {
			return 1
		}
		return max(justifiedMinRatio, min(justifiedMaxRatio, r))
	}
	var y float32
	start, sum := 0, float32(0)
	place := func(end int, height float32) {
		layout.rows = append(layout.rows, start)
		x := float32(0)
		for i := start; i < end; i++ {
			w := ratio(i) * height
			// 铺满的行最后一张补齐到右边，免得浮点误差留一条缝。
			if i == end-1 && height != target {
				w = width - x
			}
			layout.items[i] = MasonryItem{Index: i, X: x, Y: y, Width: w, Height: height}
			x += w + masonryGap
		}
		y += height + masonryGap
		start, sum = end, 0
	}
	for i := range ratios {
		sum += ratio(i)
		gaps := float32(i-start) * masonryGap
		if sum*target+gaps >= width {
			place(i+1, (width-gaps)/sum)
		}
	}
	if start < len(ratios) {
		place(len(ratios), target)
	}
	layout.rows = append(layout.rows, len(ratios))
	if len(ratios) > 0 {
		layout.height = y - masonryGap
	}
	return layout
}

func (l *justifiedLayout) Height() float32 { return l.height }

// Visible 按行二分，只返回和视口相交的卡片。
func (l *justifiedLayout) Visible(top, bottom float32) []MasonryItem {
	if bottom <= top || len(l.items) == 0 {
		return nil
	}
	first := sort.Search(len(l.items), func(i int) bool { return l.items[i].Y+l.items[i].Height > top })
	end := first
	for end < len(l.items) && l.items[end].Y < bottom {
		end++
	}
	return l.items[first:end]
}

// row 是第 index 张在第几行。
func (l *justifiedLayout) row(index int) int {
	return sort.Search(len(l.rows)-1, func(r int) bool { return l.rows[r+1] > index })
}

// neighbor 是按方向键从第 index 张走到的那张：左右是前后一张，上下是上一行、下一行里中线最近的那张。
func (l *justifiedLayout) neighbor(index int, up bool) (int, bool) {
	if index < 0 || index >= len(l.items) {
		return 0, false
	}
	r := l.row(index)
	next := r + 1
	if up {
		next = r - 1
	}
	if next < 0 || next >= len(l.rows)-1 {
		return 0, false
	}
	center := l.items[index].X + l.items[index].Width/2
	best := l.rows[next]
	for i := l.rows[next]; i < l.rows[next+1]; i++ {
		if math.Abs(float64(l.items[i].X+l.items[i].Width/2-center)) < math.Abs(float64(l.items[best].X+l.items[best].Width/2-center)) {
			best = i
		}
	}
	return best, true
}

// justifiedCache 记住按哪个筛选结果、宽度、缩略图大小排的等高行，这三样都没变就不重排。
type justifiedCache struct {
	layout       *justifiedLayout
	version      uint64
	width, scale float32
}

func (a *App) justifiedLayout(ratios func() []float32, width float32) *justifiedLayout {
	j := &a.justified
	if j.layout == nil || j.version != a.visibleVersion || j.width != width || j.scale != a.thumbnailScale() {
		*j = justifiedCache{layout: newJustified(ratios(), width, a.thumbnailScale()), version: a.visibleVersion, width: width, scale: a.thumbnailScale()}
	}
	return j.layout
}
