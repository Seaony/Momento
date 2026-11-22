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

var shapeRatios = [...]struct {
	name  string
	ratio float64
}{{"16:9", 16.0 / 9}, {"4:3", 4.0 / 3}, {"3:2", 3.0 / 2}}

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
	ratio := max(w, h) / min(w, h)
	switch shape {
	case ShapeSquare, ShapeLandscape, ShapePortrait:
		return shapeOrientation(w, h, ratio) == shape
	case ShapePanorama:
		return ratio >= 2
	}
	for _, preset := range shapeRatios {
		if preset.name == shape {
			return matchesRatio(ratio, preset.ratio)
		}
	}
	a, b, ok := strings.Cut(shape, ":")
	x, errX := strconv.ParseFloat(a, 64)
	y, errY := strconv.ParseFloat(b, 64)
	if !ok || errX != nil || errY != nil || x <= 0 || y <= 0 {
		return false
	}
	return matchesRatio(ratio, max(x, y)/min(x, y))
}

func shapeOrientation(width, height, ratio float64) string {
	if ratio <= 1+ratioTolerance {
		return ShapeSquare
	}
	if width > height {
		return ShapeLandscape
	}
	return ShapePortrait
}

func matchesRatio(ratio, target float64) bool {
	return abs(ratio-target) <= target*ratioTolerance
}

func AddShapeCounts(counts map[string]int, asset *model.Asset) {
	width, height := displaySize(asset)
	if width <= 0 || height <= 0 {
		return
	}
	ratio := max(width, height) / min(width, height)
	counts[shapeOrientation(width, height, ratio)]++
	if ratio >= 2 {
		counts[ShapePanorama]++
	}
	for _, preset := range shapeRatios {
		if matchesRatio(ratio, preset.ratio) {
			counts[preset.name]++
		}
	}
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
		AddShapeCounts(counts, &idx.snapshot.Assets[i])
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
