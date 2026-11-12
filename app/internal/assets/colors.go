package assets

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
)

type colorBucket struct{ red, green, blue, count int }

func (b colorBucket) hex() string {
	return fmt.Sprintf("#%02X%02X%02X", b.red/b.count, b.green/b.count, b.blue/b.count)
}
func (b colorBucket) distance(other colorBucket) float64 {
	r, g, bl := float64(b.red/b.count-other.red/other.count), float64(b.green/b.count-other.green/other.count), float64(b.blue/b.count-other.blue/other.count)
	return math.Sqrt(r*r + g*g + bl*bl)
}
func sortBuckets(b []colorBucket) {
	sort.Slice(b, func(i, j int) bool {
		if b[i].count == b[j].count {
			return b[i].hex() < b[j].hex()
		}
		return b[i].count > b[j].count
	})
}

func Palette(img image.Image) []model.Color {
	img = resized(img, 128)
	b := img.Bounds()
	buckets := map[int]colorBucket{}
	total := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a>>8 < 13 {
				continue
			}
			r, g, bl = r>>8, g>>8, bl>>8
			key := int((r/16)<<8 | (g/16)<<4 | bl/16)
			bucket := buckets[key]
			bucket.red += int(r)
			bucket.green += int(g)
			bucket.blue += int(bl)
			bucket.count++
			buckets[key] = bucket
			total++
		}
	}
	if total == 0 {
		return nil
	}
	ordered := make([]colorBucket, 0, len(buckets))
	for _, bucket := range buckets {
		ordered = append(ordered, bucket)
	}
	sortBuckets(ordered)
	merged := make([]colorBucket, 0, len(ordered))
	for _, bucket := range ordered {
		match := -1
		for i, other := range merged {
			if bucket.distance(other) <= 28 {
				match = i
				break
			}
		}
		if match < 0 {
			merged = append(merged, bucket)
		} else {
			merged[match].red += bucket.red
			merged[match].green += bucket.green
			merged[match].blue += bucket.blue
			merged[match].count += bucket.count
		}
	}
	sortBuckets(merged)
	colors := make([]model.Color, 0, min(8, len(merged)))
	for _, bucket := range merged[:min(8, len(merged))] {
		hex := bucket.hex()
		colors = append(colors, model.Color{Hex: hex, Proportion: float64(bucket.count) / float64(total), Family: ColorFamily(hex)})
	}
	return colors
}

func ColorFamily(hex string) string {
	cleaned := strings.Trim(strings.TrimSpace(hex), "#")
	if len(cleaned) != 6 {
		return ""
	}
	value, err := strconv.ParseUint(cleaned, 16, 24)
	if err != nil {
		return ""
	}
	r, g, b := float64(value>>16&255)/255, float64(value>>8&255)/255, float64(value&255)/255
	hi, lo := max(r, g, b), min(r, g, b)
	delta := hi - lo
	saturation := 0.0
	if hi != 0 {
		saturation = delta / hi
	}
	if hi <= 0.18 {
		return "black"
	}
	hue := 0.0
	if delta != 0 {
		switch hi {
		case r:
			hue = 60 * math.Mod((g-b)/delta, 6)
		case g:
			hue = 60 * ((b-r)/delta + 2)
		default:
			hue = 60 * ((r-g)/delta + 4)
		}
	}
	if hue < 0 {
		hue += 360
	}
	if saturation <= 0.14 {
		if hi >= 0.82 {
			return "white"
		}
		if hi <= 0.28 {
			return "black"
		}
		return "gray"
	}
	if saturation <= 0.34 && hi >= 0.62 && hue >= 35 && hue < 70 {
		return "beige"
	}
	if hue >= 15 && hue < 50 && hi < 0.58 && saturation > 0.22 {
		return "brown"
	}
	if hue >= 50 && hue < 90 && hi < 0.58 && saturation > 0.22 {
		return "olive"
	}
	for _, family := range []struct {
		limit float64
		name  string
	}{{12, "red"}, {25, "coral"}, {38, "orange"}, {50, "amber"}, {65, "yellow"}, {85, "lime"}, {145, "green"}, {165, "mint"}, {195, "teal"}, {205, "cyan"}, {220, "sky"}, {250, "blue"}, {265, "indigo"}, {285, "violet"}, {305, "purple"}, {325, "magenta"}, {340, "pink"}, {350, "rose"}, {360, "red"}} {
		if hue < family.limit {
			return family.name
		}
	}
	return "red"
}
