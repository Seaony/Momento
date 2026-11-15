package ui

import "sort"

// 列表每行高度不同（跟着缩略图走），按筛选结果算一次每行的顶部位置，之后每帧只二分查找可见的几行。
type listLayout struct {
	version uint64
	tops    []float32 // 第 i 行的顶部；最后多一项是总高度
}

func (a *App) assetRows(width, top, bottom float32) ([]MasonryItem, float32) {
	if a.list.tops == nil || a.list.version != a.visibleVersion {
		tops := make([]float32, len(a.Visible)+1)
		for i, asset := range a.Visible {
			tops[i+1] = tops[i] + listRowHeight(asset)
		}
		a.list = listLayout{version: a.visibleVersion, tops: tops}
	}
	tops := a.list.tops
	first := sort.Search(len(a.Visible), func(i int) bool { return tops[i+1] > top })
	var items []MasonryItem
	for i := first; i < len(a.Visible) && tops[i] < bottom; i++ {
		items = append(items, MasonryItem{Index: i, Y: tops[i], Width: width, Height: tops[i+1] - tops[i]})
	}
	return items, tops[len(tops)-1]
}
