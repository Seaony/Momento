package main

import (
	"image"
	"image/color"
	"image/draw"
)

func sampleImage(index int, large bool) *image.RGBA {
	w, h := 64, 48
	if large {
		sizes := [][2]int{{288, 192}, {192, 256}, {256, 256}, {288, 160}, {192, 288}, {288, 216}}
		w, h = sizes[index%len(sizes)][0], sizes[index%len(sizes)][1]
	} else if index%3 == 1 {
		w, h = 48, 64
	}
	palettes := [][3]color.RGBA{
		{{R: 204, G: 218, B: 204, A: 255}, {R: 92, G: 126, B: 99, A: 255}, {R: 36, G: 68, B: 54, A: 255}},
		{{R: 236, G: 211, B: 184, A: 255}, {R: 187, G: 132, B: 99, A: 255}, {R: 110, G: 73, B: 64, A: 255}},
		{{R: 196, G: 218, B: 232, A: 255}, {R: 113, G: 151, B: 173, A: 255}, {R: 48, G: 83, B: 110, A: 255}},
		{{R: 237, G: 218, B: 157, A: 255}, {R: 191, G: 159, B: 74, A: 255}, {R: 119, G: 105, B: 51, A: 255}},
		{{R: 219, G: 209, B: 222, A: 255}, {R: 162, G: 134, B: 166, A: 255}, {R: 103, G: 77, B: 115, A: 255}},
		{{R: 226, G: 224, B: 216, A: 255}, {R: 164, G: 165, B: 157, A: 255}, {R: 86, G: 96, B: 91, A: 255}},
	}
	palette := palettes[index%len(palettes)]
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(palette[0]), image.Point{}, draw.Src)
	for x := 0; x < w; x++ {
		horizon := h/2 + (x*(index%7+1)/7)%(max(1, h/8))
		draw.Draw(img, image.Rect(x, horizon, x+1, h), image.NewUniform(palette[1]), image.Point{}, draw.Src)
		front := h*3/4 + (x*(index%5+1)/4)%(max(1, h/8))
		draw.Draw(img, image.Rect(x, front, x+1, h), image.NewUniform(palette[2]), image.Point{}, draw.Src)
	}
	return img
}
