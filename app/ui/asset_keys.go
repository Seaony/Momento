package ui

import (
	"math"
	"slices"
	"sort"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 方向键在素材之间移动选择。列表视图上下走一行；瀑布流上下在同一列里走，左右跳到相邻一列里和当前卡片高度最接近的那张。
// 还没选中时第一下选中第一张。空格预览开着时左右键归预览翻页，这里只管上下，预览跟着选中项换图。
func (a *App) assetArrowKeys(c *ui.Context, inset, height float32) {
	if len(a.Visible) == 0 || c.FocusTakesText() || a.IsLocked() || a.Settings.RecordingShortcut != "" {
		return
	}
	previewPaging := a.Preview.active != nil && !a.Preview.active.closing
	for _, key := range []ui.Key{ui.KeyUp, ui.KeyDown, ui.KeyLeft, ui.KeyRight} {
		if (key == ui.KeyLeft || key == ui.KeyRight) && previewPaging {
			continue
		}
		if !c.Shortcut(0, key) {
			continue
		}
		index, ok := a.arrowTarget(key)
		if !ok {
			return
		}
		a.selectAsset(a.Visible[index].ID, index, 0)
		// 键盘焦点跟着走，回车打开的是新选中的这张，不是之前点过的那张。
		a.focusAssetID = a.Visible[index].ID
		if y, h, ok := a.assetRect(index); ok {
			a.revealAssetRect(y, h, inset, height)
		}
		return
	}
}

var ratingKeys = []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5}

// 数字键给选中的素材打星（没选中时给鼠标下那张），0 清除；按住 Shift 打完跳到下一张，方便一张张过。
func (a *App) assetRatingKeys(c *ui.Context, inset, height float32) {
	if a.Library == nil || c.FocusTakesText() || a.IsLocked() || a.Settings.RecordingShortcut != "" || a.TagPage {
		return
	}
	for rating, key := range ratingKeys {
		next := c.Shortcut(ui.Shift, key)
		if !next && !c.Shortcut(0, key) {
			continue
		}
		ids := a.selectedIDs()
		if len(ids) == 0 && a.hovered != "" {
			ids = []string{a.hovered}
		}
		if len(ids) == 0 {
			return
		}
		a.SetRating(ids, rating)
		if next && len(ids) == 1 {
			if index := slices.IndexFunc(a.Visible, func(asset model.Asset) bool { return asset.ID == ids[0] }); index >= 0 && index+1 < len(a.Visible) {
				a.selectAsset(a.Visible[index+1].ID, index+1, 0)
				a.focusAssetID = a.Visible[index+1].ID
				if y, h, ok := a.assetRect(index + 1); ok {
					a.revealAssetRect(y, h, inset, height)
				}
			}
		}
		return
	}
}

func (a *App) arrowTarget(key ui.Key) (int, bool) {
	current := a.selectedIndex
	if current < 0 || current >= len(a.Visible) || !a.Selected.Has(a.Visible[current].ID) {
		// 收藏、改标签等会重新筛选并清掉 selectedIndex，选中的素材还在；从选中的里找回位置，找不到才从第一张开始。
		current = slices.IndexFunc(a.Visible, func(asset model.Asset) bool { return a.Selected.Has(asset.ID) })
		if current < 0 {
			return 0, true
		}
	}
	if a.ViewMode == 1 && a.justified.layout != nil && len(a.justified.layout.items) == len(a.Visible) {
		switch key {
		case ui.KeyLeft:
			return current - 1, current > 0
		case ui.KeyRight:
			return current + 1, current+1 < len(a.Visible)
		case ui.KeyUp, ui.KeyDown:
			return a.justified.layout.neighbor(current, key == ui.KeyUp)
		}
		return 0, false
	}
	if a.ViewMode != 0 || a.layout == nil {
		switch key {
		case ui.KeyUp:
			return current - 1, current > 0
		case ui.KeyDown:
			return current + 1, current+1 < len(a.Visible)
		}
		return 0, false
	}
	column, row, ok := a.layout.locate(current)
	if !ok {
		return 0, false
	}
	items := a.layout.columns[column]
	switch key {
	case ui.KeyUp:
		if row > 0 {
			return items[row-1].Index, true
		}
	case ui.KeyDown:
		if row+1 < len(items) {
			return items[row+1].Index, true
		}
	case ui.KeyLeft, ui.KeyRight:
		next := column - 1
		if key == ui.KeyRight {
			next = column + 1
		}
		if next < 0 || next >= len(a.layout.columns) || len(a.layout.columns[next]) == 0 {
			return 0, false
		}
		return nearestByCenter(a.layout.columns[next], items[row].Y+items[row].Height/2).Index, true
	}
	return 0, false
}

// 一列里中线最接近 y 的卡片；列内按 Y 递增，二分后比较前后两张。
func nearestByCenter(column []MasonryItem, y float32) MasonryItem {
	i := sort.Search(len(column), func(i int) bool { return column[i].Y+column[i].Height/2 >= y })
	if i == len(column) {
		return column[i-1]
	}
	if i > 0 && math.Abs(float64(column[i-1].Y+column[i-1].Height/2-y)) < math.Abs(float64(column[i].Y+column[i].Height/2-y)) {
		return column[i-1]
	}
	return column[i]
}

// 第 index 张素材在画布里的上沿和高度。
func (a *App) assetRect(index int) (float32, float32, bool) {
	if a.ViewMode == 2 {
		if index+1 < len(a.list.tops) {
			return a.list.tops[index], a.list.tops[index+1] - a.list.tops[index], true
		}
		return 0, 0, false
	}
	if a.ViewMode == 1 {
		if l := a.justified.layout; l != nil && index >= 0 && index < len(l.items) {
			return l.items[index].Y, l.items[index].Height, true
		}
		return 0, 0, false
	}
	if a.layout == nil {
		return 0, 0, false
	}
	column, row, ok := a.layout.locate(index)
	if !ok {
		return 0, 0, false
	}
	item := a.layout.columns[column][row]
	return item.Y, item.Height, true
}

// 选中的素材被页头挡住或在视口外时滚过去，上下各留 16。
func (a *App) revealAssetRect(y, h, inset, height float32) {
	// 素材上面有子文件夹卡片时，画布坐标整体往下挪。
	y += a.assetLead
	// 画布坐标里，页头下沿是 scroll.Y - 16，视口下沿是 scroll.Y - inset - 16 + height。
	if y < a.scroll.Y {
		a.scroll.Y = max(0, y)
	} else if bottom := y + h + inset + 2*space16; bottom > a.scroll.Y+height {
		a.scroll.Y = bottom - height
	}
}
