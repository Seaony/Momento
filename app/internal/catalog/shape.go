package catalog

import (
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
)

// 形状筛选：横、竖、方、全景，加上几种常见比例。
const (
	ShapeLandscape = "landscape"
	ShapePortrait  = "portrait"
	ShapeSquare    = "square"
	ShapePanorama  = "panorama"
)

// Shapes 是筛选面板里依次列出的形状和比例。比例不分横竖：16:9 也包括 9:16。
var Shapes = []string{ShapeLandscape, ShapePortrait, ShapeSquare, ShapePanorama, "16:9", "4:3", "3:2"}

// 宽高比相差 3% 以内算同一比例，方图同理（裁剪时差一两个像素很常见）。
const ratioTolerance = 0.03

// displaySize 是转过之后看到的宽高。
func displaySize(asset *model.Asset) (float64, float64) {
	w, h := float64(asset.Width), float64(asset.Height)
	if asset.Rotation == 90 || asset.Rotation == 270 {
		w, h = h, w
	}
	return w, h
}

// MatchesShape 判断素材是不是这种形状或比例；不知道尺寸的素材什么形状都不算。
func MatchesShape(asset *model.Asset, shape string) bool {
	w, h := displaySize(asset)
	if w <= 0 || h <= 0 {
		return false
	}
	long, short := max(w, h), min(w, h)
	square := long/short <= 1+ratioTolerance
	switch shape {
	case ShapeSquare:
		return square
	case ShapeLandscape:
		return w > h && !square
	case ShapePortrait:
		return h > w && !square
	case ShapePanorama:
		return long/short >= 2
	}
	a, b, ok := strings.Cut(shape, ":")
	x, errX := strconv.ParseFloat(a, 64)
	y, errY := strconv.ParseFloat(b, 64)
	if !ok || errX != nil || errY != nil || x <= 0 || y <= 0 {
		return false
	}
	want := max(x, y) / min(x, y)
	return abs(long/short-want) <= want*ratioTolerance
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func matchesShapes(asset *model.Asset, shapes []string) bool {
	if len(shapes) == 0 {
		return true
	}
	for _, shape := range shapes {
		if MatchesShape(asset, shape) {
			return true
		}
	}
	return false
}

// ShapeCounts 是按当前其他条件（不算形状条件本身）各形状有多少素材，给筛选面板标数量。
func (idx *Index) ShapeCounts(q Query) map[string]int {
	counts := make(map[string]int, len(Shapes))
	q.Shapes = nil
	idx.eachMatching(q, func(i int) {
		for _, shape := range Shapes {
			if MatchesShape(&idx.snapshot.Assets[i], shape) {
				counts[shape]++
			}
		}
	})
	return counts
}

// RatingCounts 是按当前其他条件（不算评分条件本身）0～5 星各有多少素材，下标就是星数，0 是没评分。
func (idx *Index) RatingCounts(q Query) [6]int {
	var counts [6]int
	q.Ratings = nil
	idx.eachMatching(q, func(i int) {
		if rating := idx.snapshot.Assets[i].Rating; rating >= 0 && rating <= 5 {
			counts[rating]++
		}
	})
	return counts
}

// TypeCounts 是按当前其他条件（不算文件类型条件本身）各扩展名、各种类有多少素材，给筛选面板标数量。
func (idx *Index) TypeCounts(q Query) (extensions, kinds map[string]int) {
	extensions, kinds = map[string]int{}, map[string]int{}
	q.Extensions, q.Kinds = nil, nil
	idx.eachMatching(q, func(i int) {
		extensions[idx.entries[i].extension]++
		kinds[idx.snapshot.Assets[i].Kind]++
	})
	return extensions, kinds
}

// TagCounts 是按查询条件各标签有多少素材。调用方先去掉筛选里的标签条件再传进来：
// 在某个标签的素材页里，范围本身就是用 TagIDs 限定的，这里不能一起清掉。
func (idx *Index) TagCounts(q Query) map[string]int {
	counts := map[string]int{}
	idx.eachMatching(q, func(i int) {
		for _, id := range idx.snapshot.Assets[i].TagIDs {
			counts[id]++
		}
	})
	return counts
}
