package assets

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func safeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, ". ")
	if name == "" {
		return "image"
	}
	return name
}

func exportName(a model.Asset, format model.ExportFormat) string {
	name := a.Name
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(a.OriginalFileName, filepath.Ext(a.OriginalFileName))
	}
	name = safeName(name)
	extension := a.Extension
	switch format {
	case model.ExportJPEG:
		extension = "jpg"
	case model.ExportPNG:
		extension = "png"
	}
	return name + "." + strings.ToLower(extension)
}

func numberedPath(path string, index int) string {
	if index == 1 {
		return path
	}
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + fmt.Sprintf(" %d", index) + ext
}

func (s *Service) Export(ctx context.Context, items []model.Asset, destination string, format model.ExportFormat, quality float64) (paths []string, err error) {
	if format != model.ExportOriginal && format != model.ExportJPEG && format != model.ExportPNG {
		return nil, i18n.NewError("error_export_format", string(format))
	}
	if len(items) == 0 {
		return nil, nil
	}
	if quality == 0 {
		quality = .9
	}
	if quality < 0 || quality > 1 {
		return nil, i18n.NewError("error_jpeg_quality")
	}
	directory := false
	info, statErr := os.Stat(destination)
	if statErr == nil {
		directory = info.IsDir()
	} else if !os.IsNotExist(statErr) {
		return nil, statErr
	}
	if len(items) > 1 && !directory {
		return nil, i18n.NewError("error_export_directory")
	}
	slog.Info("开始导出", "count", len(items), "format", format)
	defer func() {
		if err != nil {
			slog.Error("导出失败", "written", len(paths), "error", err)
		} else {
			slog.Info("导出完成", "count", len(paths))
		}
	}()
	for _, a := range items {
		if err = ctx.Err(); err != nil {
			return paths, err
		}
		source, e := s.assetPath(a.RelativePath)
		if e != nil {
			return paths, e
		}
		requested := destination
		if directory {
			requested = filepath.Join(destination, exportName(a, format))
		}
		temp, e := os.CreateTemp(filepath.Dir(requested), ".momento-export-*")
		if e != nil {
			return paths, e
		}
		staged := temp.Name()
		e = func() (err error) {
			defer func() { err = errors.Join(err, temp.Close()) }()
			if format == model.ExportOriginal {
				f, err := os.Open(source)
				if err != nil {
					return err
				}
				defer f.Close()
				_, err = io.Copy(temp, contextReader{ctx: ctx, r: f})
				if err != nil {
					return err
				}
			} else {
				img, _, _, _, err := PrepareImage(source)
				if err != nil {
					return err
				}
				if format == model.ExportJPEG {
					opaque := image.NewRGBA(img.Bounds())
					draw.Draw(opaque, opaque.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
					draw.Draw(opaque, opaque.Bounds(), img, img.Bounds().Min, draw.Over)
					if err = jpeg.Encode(temp, opaque, &jpeg.Options{Quality: max(1, int(quality*100))}); err != nil {
						return err
					}
				} else if err = png.Encode(temp, img); err != nil {
					return err
				}
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			return temp.Sync()
		}()
		if e != nil {
			return paths, errors.Join(e, removeIfExists(staged))
		}
		var final string
		for index := 1; ; index++ {
			final = numberedPath(requested, index)
			e = publishFile(staged, final)
			if !os.IsExist(e) {
				break
			}
		}
		cleanupErr := removeIfExists(staged)
		if e != nil {
			return paths, errors.Join(e, cleanupErr)
		}
		paths = append(paths, final)
		if cleanupErr != nil {
			return paths, cleanupErr
		}
	}
	return paths, nil
}
