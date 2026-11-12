package assets

import (
	"bytes"
	"errors"
	"image"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/evanoberholster/imagemeta"
	"github.com/evanoberholster/imagemeta/meta"
	"github.com/evanoberholster/imagemeta/meta/isobmff"
	metajpeg "github.com/evanoberholster/imagemeta/meta/jpeg"
	metapng "github.com/evanoberholster/imagemeta/meta/png"
	"github.com/evanoberholster/imagemeta/meta/utils"
)

func readMetadata(path string, img image.Image) (map[string]string, int, string) {
	values := map[string]string{
		"pixelWidth":  strconv.Itoa(img.Bounds().Dx()),
		"pixelHeight": strconv.Itoa(img.Bounds().Dy()),
		"colorModel":  imageColorModel(img),
	}
	orientation, profile := 1, ""
	f, err := os.Open(path)
	if err != nil {
		slog.Warn("读取图片元数据失败", "path", path, "error", err)
		return values, orientation, profile
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil {
		values["fileModifiedAt"] = info.ModTime().UTC().Format(time.RFC3339Nano)
		if created, ok := sourceCreatedAt(info); ok {
			values["fileCreatedAt"] = created.UTC().Format(time.RFC3339Nano)
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".tif", ".tiff", ".png", ".heic", ".heif", ".avif":
	default:
		return values, orientation, profile
	}
	e, err := imagemeta.Decode(f)
	if err != nil && !errors.Is(err, imagemeta.ErrNoExif) && !errors.Is(err, imagemeta.ErrMetadataNotSupported) {
		slog.Warn("读取 EXIF 失败", "path", path, "error", err)
	}
	present, presenceErr := presentExifTags(f, ext)
	if presenceErr != nil && !errors.Is(presenceErr, meta.ErrNoExif) {
		slog.Warn("读取 EXIF 字段失败", "path", path, "error", presenceErr)
	}
	put := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			values[key] = value
		}
	}
	putNumber := func(key string, value float64) {
		if value > 0 {
			put(key, strconv.FormatFloat(value, 'f', -1, 64))
		}
	}
	if e.IFD0.Orientation >= 1 && e.IFD0.Orientation <= 8 {
		orientation = int(e.IFD0.Orientation)
	}
	date := e.OriginalDate()
	if date.IsZero() {
		date = e.DigitizedDate()
	}
	if date.IsZero() {
		date = e.ModifyDate()
	}
	if !date.IsZero() {
		values["contentCreatedAt"] = date.Format(time.RFC3339Nano)
	}
	put("cameraMake", e.IFD0.Make)
	put("cameraModel", e.IFD0.Model)
	put("creator", e.IFD0.Artist)
	put("lensModel", e.ExifIFD.LensModel)
	if e.ExifIFD.ExposureTime > 0 {
		put("exposureTime", strconv.FormatFloat(float64(e.ExifIFD.ExposureTime), 'f', -1, 32))
	}
	if e.ExifIFD.FocalLength > 0 {
		put("focalLength", strconv.FormatFloat(float64(e.ExifIFD.FocalLength), 'f', -1, 32))
	}
	if e.ExifIFD.FNumber > 0 {
		put("fNumber", strconv.FormatFloat(float64(e.ExifIFD.FNumber), 'f', -1, 32))
	}
	if e.ExifIFD.ISOSpeedRatings > 0 {
		put("isoSpeedRatings", strconv.FormatUint(uint64(e.ExifIFD.ISOSpeedRatings), 10))
	}
	if present[0x9209] {
		put("flashFired", strconv.FormatBool(uint16(e.ExifIFD.Flash)&1 != 0))
	}
	for _, field := range []struct {
		key   string
		tag   uint16
		value uint64
	}{{"exposureProgram", 0x8822, uint64(e.ExifIFD.ExposureProgram)}, {"meteringMode", 0x9207, uint64(e.ExifIFD.MeteringMode)}, {"whiteBalance", 0xa403, uint64(e.ExifIFD.WhiteBalance)}} {
		if present[field.tag] {
			put(field.key, strconv.FormatUint(field.value, 10))
		}
	}
	multiplier := 1.0
	if e.IFD0.ResolutionUnit == 3 {
		multiplier = 2.54
	}
	if e.IFD0.ResolutionUnit == 2 || e.IFD0.ResolutionUnit == 3 {
		putNumber("dpiWidth", e.IFD0.XResolution*multiplier)
		putNumber("dpiHeight", e.IFD0.YResolution*multiplier)
	}
	if e.ExifIFD.ColorSpace == 1 {
		profile = "sRGB"
	}
	if ext == ".jpg" || ext == ".jpeg" {
		if _, err := f.Seek(0, io.SeekStart); err == nil {
			markerMetadata, err := metajpeg.ScanMetadata(f)
			if err != nil {
				slog.Warn("读取图片色彩配置失败", "path", path, "error", err)
			} else if markerMetadata.ICC != nil && markerMetadata.ICC.ProfileDescription != "" {
				profile = markerMetadata.ICC.ProfileDescription
			}
		}
	}
	put("profileName", profile)
	return values, orientation, profile
}

// 库把缺失字段和零值放在同一个字段里，只检查 tag 是否存在，数值仍由库解析。
func presentExifTags(f *os.File, ext string) (map[uint16]bool, error) {
	found := map[uint16]bool{}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return found, err
	}
	collect := func(r io.Reader, h meta.ExifHeader, prefix int) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if prefix > 0 {
			data = append(make([]byte, prefix), data...)
		}
		return collectTIFFTags(bytes.NewReader(data), int64(h.FirstIfdOffset), h.ByteOrder, found)
	}
	switch ext {
	case ".jpg", ".jpeg":
		return found, metajpeg.ScanJPEG(f, func(r io.Reader, h meta.ExifHeader) error { return collect(r, h, 0) }, nil)
	case ".png":
		h, err := metapng.ScanPngHeader(f)
		if err != nil {
			return found, err
		}
		return found, collectTIFFTags(io.NewSectionReader(f, int64(h.TiffHeaderOffset), int64(h.ExifLength)), int64(h.FirstIfdOffset), h.ByteOrder, found)
	case ".tif", ".tiff":
		var header [8]byte
		if _, err := f.ReadAt(header[:], 0); err != nil {
			return found, err
		}
		order := utils.BinaryOrder(header[:])
		if order == utils.UnknownEndian {
			return found, meta.ErrInvalidHeader
		}
		return found, collectTIFFTags(f, int64(order.Uint32(header[4:])), order, found)
	case ".heic", ".heif", ".avif":
		r := isobmff.NewReader(f, func(r io.Reader, h meta.ExifHeader) error { return collect(r, h, 8) }, nil, nil)
		defer r.Close()
		if err := r.ReadFTYP(); err != nil {
			return found, err
		}
		return found, r.ReadMetadataUntilEOF()
	}
	return found, nil
}

func collectTIFFTags(r io.ReaderAt, root int64, order utils.ByteOrder, found map[uint16]bool) error {
	pending := []int64{root}
	visited := map[int64]bool{}
	for len(pending) > 0 {
		offset := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[offset] {
			continue
		}
		visited[offset] = true
		var count [2]byte
		if _, err := r.ReadAt(count[:], offset); err != nil {
			return err
		}
		entries := make([]byte, int(order.Uint16(count[:]))*12)
		if _, err := r.ReadAt(entries, offset+2); err != nil {
			return err
		}
		for i := 0; i < len(entries); i += 12 {
			tag := order.Uint16(entries[i:])
			found[tag] = true
			if tag == 0x8769 {
				pending = append(pending, int64(order.Uint32(entries[i+8:])))
			}
		}
	}
	return nil
}
