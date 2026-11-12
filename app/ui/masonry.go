package ui

import (
	"math"
	"sort"
)

const (
	masonryMinWidth       float32 = 164
	masonryGap            float32 = 8
	masonryMinHeight      float32 = 132
	masonryMaxHeight      float32 = 278
	masonryFallbackHeight float32 = 214
)

type MasonryItem struct {
	Index               int
	X, Y, Width, Height float32
}

type MasonryLayout struct {
	columns [][]MasonryItem
	height  float32
}

// ratios 为图片宽高比，外侧留白由调用方负责。
func NewMasonry(ratios []float32, width float32) *MasonryLayout {
	return NewMasonryScaled(ratios, width, 1)
}

func NewMasonryScaled(ratios []float32, width, scale float32) *MasonryLayout {
	scale = max(.5, min(2, scale))
	minWidth := masonryMinWidth * scale
	width = max(width, minWidth)
	count := max(1, int((width+masonryGap)/(minWidth+masonryGap)))
	itemWidth := (width - float32(count-1)*masonryGap) / float32(count)
	layout := &MasonryLayout{columns: make([][]MasonryItem, count)}
	heights := make([]float32, count)
	for column := range layout.columns {
		layout.columns[column] = make([]MasonryItem, 0, (len(ratios)+count-1)/count)
	}
	for index, ratio := range ratios {
		column := 0
		for candidate := 1; candidate < count; candidate++ {
			if heights[candidate] < heights[column] {
				column = candidate
			}
		}
		height := masonryFallbackHeight * scale
		if ratio > 0 && !math.IsInf(float64(ratio), 0) && !math.IsNaN(float64(ratio)) {
			height = min(max(itemWidth/ratio, masonryMinHeight*scale), masonryMaxHeight*scale)
		}
		item := MasonryItem{
			Index: index,
			X:     float32(column) * (itemWidth + masonryGap), Y: heights[column],
			Width: itemWidth, Height: height,
		}
		layout.columns[column] = append(layout.columns[column], item)
		heights[column] = item.Y + item.Height + masonryGap
		layout.height = max(layout.height, item.Y+item.Height)
	}
	return layout
}

// 每列按底边二分，只访问与视口相交的卡片。
func (m *MasonryLayout) Visible(top, bottom float32) []MasonryItem {
	if bottom <= top {
		return nil
	}
	var visible []MasonryItem
	for _, column := range m.columns {
		first := sort.Search(len(column), func(i int) bool {
			return column[i].Y+column[i].Height > top
		})
		for i := first; i < len(column) && column[i].Y < bottom; i++ {
			visible = append(visible, column[i])
		}
	}
	sort.Slice(visible, func(i, j int) bool { return visible[i].Index < visible[j].Index })
	return visible
}

func (m *MasonryLayout) Height() float32 { return m.height }
