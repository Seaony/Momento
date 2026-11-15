package ui

import (
	"bytes"
	"image"
	"image/png"
	"log/slog"

	"github.com/Seaony/Momento/app/resources"
	"github.com/egoist/mygo/ui"
)

// 空状态插画（resources/empty）：单色蒙版，按当前主题的颜色着色后缓存，浅色、深色共用一套图。
type emptyArtKey struct {
	name string
	ink  ui.Color
}

var (
	emptyArtMasks   = map[string]*image.NRGBA{}
	emptyArtBitmaps = map[emptyArtKey]*ui.Bitmap{}
)

func emptyArtBitmap(name string, ink ui.Color) *ui.Bitmap {
	key := emptyArtKey{name, ink}
	if bitmap, ok := emptyArtBitmaps[key]; ok {
		return bitmap
	}
	mask, ok := emptyArtMasks[name]
	if !ok {
		data, err := resources.EmptyArt.ReadFile("empty/" + name + ".png")
		var img image.Image
		if err == nil {
			img, err = png.Decode(bytes.NewReader(data))
		}
		if err != nil {
			slog.Error("读取空状态插画失败", "name", name, "error", err)
			emptyArtMasks[name] = nil
			return nil
		}
		bounds := img.Bounds()
		mask = image.NewNRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				mask.Set(x, y, img.At(x, y))
			}
		}
		emptyArtMasks[name] = mask
	}
	if mask == nil {
		return nil
	}
	tinted := image.NewNRGBA(mask.Bounds())
	for i := 0; i < len(mask.Pix); i += 4 {
		tinted.Pix[i], tinted.Pix[i+1], tinted.Pix[i+2] = ink.R, ink.G, ink.B
		tinted.Pix[i+3] = uint8(uint16(mask.Pix[i+3]) * uint16(ink.A) / 255)
	}
	bitmap := ui.NewBitmap(tinted)
	emptyArtBitmaps[key] = bitmap
	return bitmap
}

// 空状态插画：三级文字色的浅线稿，读屏跳过（旁边的文字已经说清楚了）。
func emptyArt(c *ui.Context, name string, size float32) ui.Element {
	if bitmap := emptyArtBitmap(name, paletteFor(c.Theme()).TextTertiary.Alpha(emptyArtInk)); bitmap != nil {
		return ui.Image(c, bitmap).Size(size, size).AccessHidden(true).Shrink(0)
	}
	return ui.Box(c).Size(size, size).Shrink(0)
}

// 大的放在内容区和标签管理的空状态，小的放在选择器和设置里。
const (
	emptyArtLarge = 112
	emptyArtSmall = 56
)

// 插画线条的浓淡：三级文字色再淡一些，只做点缀。
const emptyArtInk = .7
