package ui

import "math"

// 规则网格和列表只计算可见窗口，卡片与瀑布流共用同一个画布和身份。
func (a *App) assetRows(width, top, bottom float32) ([]MasonryItem, float32) {
	columns, gap, rowHeight := 1, float32(0), float32(64)
	if a.ViewMode == 1 {
		gap = masonryGap
		columns = max(1, int((width+gap)/(160*a.thumbnailScale()+gap)))
		rowHeight = (width - float32(columns-1)*gap) / float32(columns)
	}
	rows := (len(a.Visible) + columns - 1) / columns
	first := max(0, int(math.Floor(float64(top/(rowHeight+gap)))))
	last := min(rows, int(math.Ceil(float64(bottom/(rowHeight+gap)))))
	items := make([]MasonryItem, 0, max(0, last-first)*columns)
	for row := first; row < last; row++ {
		for column := range columns {
			index := row*columns + column
			if index >= len(a.Visible) {
				break
			}
			itemWidth := width
			if columns > 1 {
				itemWidth = rowHeight
			}
			items = append(items, MasonryItem{Index: index, X: float32(column) * (itemWidth + gap), Y: float32(row) * (rowHeight + gap), Width: itemWidth, Height: rowHeight})
		}
	}
	return items, max(0, float32(rows)*(rowHeight+gap)-gap)
}
