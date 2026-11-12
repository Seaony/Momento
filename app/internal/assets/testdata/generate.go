//go:build ignore

package main

import (
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func main() {
	out := "internal/assets/testdata"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		log.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 96; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 255 / 95), uint8(y * 255 / 63), uint8((x + y) * 255 / 158), 255})
		}
	}
	encoders := map[string]func(io.Writer) error{
		"sample.jpg":  func(w io.Writer) error { return jpeg.Encode(w, img, &jpeg.Options{Quality: 90}) },
		"sample.png":  func(w io.Writer) error { return png.Encode(w, img) },
		"sample.gif":  func(w io.Writer) error { return gif.Encode(w, img, nil) },
		"sample.bmp":  func(w io.Writer) error { return bmp.Encode(w, img) },
		"sample.tiff": func(w io.Writer) error { return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate}) },
	}
	for name, encode := range encoders {
		f, err := os.Create(filepath.Join(out, name))
		if err != nil {
			log.Fatal(err)
		}
		if err := encode(f); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
