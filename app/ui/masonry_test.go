package ui

import (
	"fmt"
	"math"
	"slices"
	"testing"
)

func TestMasonryDimensionsAndShortestColumn(t *testing.T) {
	layout := NewMasonry([]float32{1, 2, 0.5, 1, 1}, 516)
	want := []MasonryItem{
		{Index: 0, X: 0, Y: 0, Width: 164, Height: 164},
		{Index: 1, X: 176, Y: 0, Width: 164, Height: 132},
		{Index: 2, X: 352, Y: 0, Width: 164, Height: 278},
		{Index: 3, X: 176, Y: 144, Width: 164, Height: 164},
		{Index: 4, X: 0, Y: 176, Width: 164, Height: 164},
	}
	if got := layout.Visible(0, layout.Height()); !slices.Equal(got, want) {
		t.Fatalf("布局为 %+v，期望 %+v", got, want)
	}
	if layout.Height() != 340 {
		t.Fatalf("内容高度 %v，期望 340", layout.Height())
	}
}

func TestMasonryColumnWidthFillsContainer(t *testing.T) {
	for _, width := range []float32{100, 164, 335, 336, 507, 508, 1100, 1800} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			ratios := make([]float32, 40)
			for i := range ratios {
				ratios[i] = 1
			}
			layout := NewMasonry(ratios, width)
			var right float32
			for _, item := range layout.Visible(0, 1) {
				if item.Width < 164 {
					t.Fatalf("卡片宽度 %v 小于 164", item.Width)
				}
				right = max(right, item.X+item.Width)
			}
			if math.Abs(float64(right-max(width, 164))) > 0.001 {
				t.Fatalf("首行右边缘 %v，容器宽度 %v", right, width)
			}
		})
	}
}

func TestMasonryMissingDimensionsAndEmptyLayout(t *testing.T) {
	layout := NewMasonry([]float32{0, -1, float32(math.NaN()), float32(math.Inf(1))}, 164)
	for _, item := range layout.Visible(0, layout.Height()) {
		if item.Height != 214 {
			t.Fatalf("缺失尺寸高度 %v，期望 214", item.Height)
		}
	}
	empty := NewMasonry(nil, 1100)
	if empty.Height() != 0 || len(empty.Visible(0, 800)) != 0 {
		t.Fatal("空布局含卡片或非零高度")
	}
}

func TestMasonryVisibleIntersectionBoundaries(t *testing.T) {
	layout := NewMasonry([]float32{1, 1, 1}, 164)
	for _, tc := range []struct {
		top, bottom float32
		ids         []int
	}{
		{0, 164, []int{0}},
		{164, 176, nil},
		{163, 177, []int{0, 1}},
		{176, 340, []int{1}},
		{-100, 1, []int{0}},
		{0, 0, nil},
		{300, 100, nil},
		{600, 800, nil},
	} {
		t.Run(fmt.Sprintf("%v-%v", tc.top, tc.bottom), func(t *testing.T) {
			var ids []int
			for _, item := range layout.Visible(tc.top, tc.bottom) {
				ids = append(ids, item.Index)
			}
			if !slices.Equal(ids, tc.ids) {
				t.Fatalf("可见 %v，期望 %v", ids, tc.ids)
			}
		})
	}
}

func TestMasonryVisibleMatchesFullScanAt100K(t *testing.T) {
	ratios := masonryTestRatios(100_000)
	layout := NewMasonry(ratios, 1100)
	all := layout.Visible(0, layout.Height())
	if len(all) != len(ratios) {
		t.Fatalf("布局数量 %d，期望 %d", len(all), len(ratios))
	}
	for i, item := range all {
		if item.Index != i {
			t.Fatalf("卡片顺序 %d: %d", i, item.Index)
		}
		if item.Height < 132 || item.Height > 278 {
			t.Fatalf("高度超出范围 %+v", item)
		}
	}
	for _, top := range []float32{0, 200, 100_000, layout.Height() / 2, layout.Height() - 800, layout.Height()} {
		var want []MasonryItem
		for _, item := range all {
			if item.Y < top+800 && item.Y+item.Height > top {
				want = append(want, item)
			}
		}
		if got := layout.Visible(top, top+800); !slices.Equal(got, want) {
			t.Fatalf("视口 %v: 得到 %d 项，期望 %d 项", top, len(got), len(want))
		}
	}
	for _, column := range layout.columns {
		for i := 1; i < len(column); i++ {
			if column[i].Y < column[i-1].Y+column[i-1].Height+masonryGap {
				t.Fatal("卡片重叠或间距不足")
			}
		}
	}
}

func masonryTestRatios(count int) []float32 {
	ratios := make([]float32, count)
	for i := range ratios {
		ratios[i] = 0.25 + float32((i*37)%300)/100
	}
	return ratios
}

func BenchmarkMasonryBuild100K(b *testing.B) {
	ratios := masonryTestRatios(100_000)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		NewMasonry(ratios, 1100)
	}
}

func BenchmarkMasonryVisible100K(b *testing.B) {
	layout := NewMasonry(masonryTestRatios(100_000), 1100)
	top := layout.Height() / 2
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		layout.Visible(top, top+800)
	}
}
