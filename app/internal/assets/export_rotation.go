package assets

import (
	"context"
	"encoding/binary"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/gen2brain/avif"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

func originalRotationSupported(asset model.Asset) error {
	if _, err := normalizedRotation(asset.Rotation); err != nil {
		return err
	}
	if asset.Rotation%360 == 0 {
		return nil
	}
	switch strings.ToLower(asset.Extension) {
	case "jpg", "jpeg", "png", "gif", "tif", "tiff", "bmp", "avif":
		return nil
	default:
		return i18n.NewError("error_export_rotation_format", strings.ToUpper(asset.Extension))
	}
}

func writeRotatedOriginal(ctx context.Context, w io.Writer, source string, asset model.Asset, quality float64) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	rotation, err := normalizedRotation(asset.Rotation)
	if err != nil {
		return err
	}
	extension := strings.ToLower(asset.Extension)
	switch extension {
	case "gif":
		animation, err := gif.DecodeAll(contextReader{ctx: ctx, r: f})
		if err != nil {
			return err
		}
		width, height := animation.Config.Width, animation.Config.Height
		for i, frame := range animation.Image {
			if err := ctx.Err(); err != nil {
				return err
			}
			bounds := frame.Bounds()
			first := rotatedPoint(bounds.Min.X, bounds.Min.Y, width, height, rotation)
			last := rotatedPoint(bounds.Max.X-1, bounds.Max.Y-1, width, height, rotation)
			rotated := image.NewPaletted(image.Rect(min(first.X, last.X), min(first.Y, last.Y), max(first.X, last.X)+1, max(first.Y, last.Y)+1), frame.Palette)
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					point := rotatedPoint(x, y, width, height, rotation)
					rotated.SetColorIndex(point.X, point.Y, frame.ColorIndexAt(x, y))
				}
			}
			animation.Image[i] = rotated
		}
		size := rotatedSize(width, height, rotation)
		animation.Config.Width, animation.Config.Height = size.X, size.Y
		return gif.EncodeAll(w, animation)
	case "avif":
		animation, err := avif.DecodeAll(contextReader{ctx: ctx, r: f}, avif.Options{AutoRotate: true})
		if err != nil {
			return err
		}
		// 当前解码器没有读出循环次数，不能把有限动画悄悄变成无限动画。
		if len(animation.Image) > 1 {
			return i18n.NewError("error_export_rotation_animation", "AVIF")
		}
		for i, frame := range animation.Image {
			if err := ctx.Err(); err != nil {
				return err
			}
			animation.Image[i], err = RotateImage(frame, rotation)
			if err != nil {
				return err
			}
		}
		return avif.EncodeAll(w, animation, avif.Options{Lossless: true, Speed: 10})
	case "tif", "tiff":
		if err := singlePageTIFF(f); err != nil {
			return err
		}
	}
	img, _, _, _, err := PrepareImage(source)
	if err != nil {
		return err
	}
	img, err = RotateImage(img, rotation)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	switch extension {
	case "jpg", "jpeg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: max(1, int(quality*100))})
	case "png":
		return png.Encode(w, img)
	case "tif", "tiff":
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
	case "bmp":
		return bmp.Encode(w, img)
	default:
		return i18n.NewError("error_export_rotation_format", strings.ToUpper(extension))
	}
}

func singlePageTIFF(f *os.File) error {
	var header [8]byte
	if _, err := f.ReadAt(header[:], 0); err != nil {
		return err
	}
	var order binary.ByteOrder
	switch string(header[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return i18n.NewError("error_export_rotation_tiff")
	}
	if order.Uint16(header[2:4]) != 42 {
		return i18n.NewError("error_export_rotation_tiff")
	}
	root := int64(order.Uint32(header[4:8]))
	var count [2]byte
	if _, err := f.ReadAt(count[:], root); err != nil {
		return err
	}
	var next [4]byte
	if _, err := f.ReadAt(next[:], root+2+int64(order.Uint16(count[:]))*12); err != nil {
		return err
	}
	if order.Uint32(next[:]) != 0 {
		return i18n.NewError("error_export_rotation_tiff")
	}
	return nil
}
