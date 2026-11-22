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
	xdraw "golang.org/x/image/draw"
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

// Export 导出素材，返回实际写入的路径（和 items 同序）。destination 是已有文件夹时逐个写进去，重名自动改名；
// 否则 destination 就是保存对话框选的文件路径，已存在时直接替换。
func (s *Service) Export(ctx context.Context, items []model.Asset, destination string, format model.ExportFormat, quality float64) (paths []string, err error) {
	return s.ExportWithMotion(ctx, items, destination, format, quality, false)
}

// ExportWithMotion 同 Export；motion 为真时实况照片的视频也导出，和图片同名、放在一起（照片 App 认这种配对）。
// 返回的仍只有图片路径。
func (s *Service) ExportWithMotion(ctx context.Context, items []model.Asset, destination string, format model.ExportFormat, quality float64, motion bool) (paths []string, err error) {
	return s.ExportWithOptions(ctx, items, destination, model.ExportOptions{Format: format, Quality: quality, Motion: motion})
}

// ExportWithOptions 按 options 导出；转成 JPEG/PNG 时可以限制最长边。
func (s *Service) ExportWithOptions(ctx context.Context, items []model.Asset, destination string, options model.ExportOptions) (paths []string, err error) {
	format, quality, motion := options.Format, options.Quality, options.Motion
	if format != model.ExportOriginal && format != model.ExportJPEG && format != model.ExportPNG {
		return nil, i18n.NewError("error_export_format", string(format))
	}
	// 原格式是原样复制文件，改不了尺寸；不能悄悄导出原尺寸。
	if options.MaxEdge < 0 || options.MaxEdge > 0 && format == model.ExportOriginal {
		return nil, i18n.NewError("error_export_max_edge")
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
	// 不能导出进图库包里面：单张确认替换时会覆盖图库自己的原图，导出到包内文件夹也会留下图库不认识的文件。
	if s.insideLibrary(destination) {
		return nil, i18n.NewError("error_export_into_library")
	}
	slog.Info("开始导出", "count", len(items), "format", format, "maxEdge", options.MaxEdge)
	defer func() {
		if err != nil {
			slog.Error("导出失败", "written", len(paths), "error", err)
		} else {
			slog.Info("导出完成", "count", len(paths))
		}
	}()
	for _, asset := range items {
		if _, e := normalizedRotation(asset.Rotation); e != nil {
			return nil, e
		}
		if format == model.ExportOriginal && !isVideo(asset.Extension) {
			if e := originalRotationSupported(asset); e != nil {
				return nil, e
			}
		}
	}
	nextNumber := map[string]int{}
	for _, a := range items {
		if err = ctx.Err(); err != nil {
			return paths, err
		}
		// 视频没有「转成 JPEG/PNG」这回事，一律原样导出原文件。
		format := format
		if isVideo(a.Extension) {
			format, a.Rotation = model.ExportOriginal, 0
		}
		source, e := s.assetPath(a.RelativePath)
		if e != nil {
			return paths, e
		}
		requested := destination
		if directory {
			requested = filepath.Join(destination, exportName(a, format))
		} else if isVideo(a.Extension) && !strings.EqualFold(strings.TrimPrefix(filepath.Ext(requested), "."), a.Extension) {
			// 保存框里的名字带着别的后缀（比如选了 JPEG 格式）：视频内容不会变，换成视频自己的后缀，免得打不开。
			switch strings.ToLower(filepath.Ext(requested)) {
			case ".jpg", ".jpeg", ".png":
				requested = strings.TrimSuffix(requested, filepath.Ext(requested))
			}
			requested += "." + strings.ToLower(a.Extension)
		}
		temp, e := os.CreateTemp(filepath.Dir(requested), ".momento-export-*")
		if e != nil {
			return paths, e
		}
		staged := temp.Name()
		e = func() (err error) {
			defer func() { err = errors.Join(err, temp.Close()) }()
			if format == model.ExportOriginal && a.Rotation%360 != 0 {
				if err = writeRotatedOriginal(ctx, temp, source, a, quality); err != nil {
					return err
				}
			} else if format == model.ExportOriginal {
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
				img, err = RotateImage(img, a.Rotation)
				if err != nil {
					return err
				}
				img = fitLongestEdge(img, options.MaxEdge)
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
		withMotion := motion && a.MotionPath != ""
		// 实况照片的视频也先完整暂存好，两个文件都齐了才发布，免得只放上一张静态图或换掉了旧图却没有视频。
		var stagedMotion string
		if withMotion {
			if stagedMotion, e = s.stageMotionExport(ctx, a, filepath.Dir(requested)); e != nil {
				return paths, errors.Join(e, removeIfExists(staged))
			}
		}
		final := requested
		if directory {
			// 导出到文件夹：重名时自动改成「名字 2.ext」，不覆盖已有文件；带视频时两个名字都要空着，配对才不会乱。
			number := max(1, nextNumber[requested])
			for ; ; number++ {
				final = numberedPath(requested, number)
				if withMotion {
					if _, statErr := os.Lstat(motionExportPath(final)); statErr == nil {
						continue
					}
				}
				// 找空名可能要一会儿（同名很多、外置盘慢）：发布前再看一次是否已取消。
				if e = ctx.Err(); e != nil {
					break
				}
				if e = publishFile(staged, final); os.IsExist(e) {
					continue
				}
				if e != nil || !withMotion {
					break
				}
				if e = publishFile(stagedMotion, motionExportPath(final)); e == nil {
					break
				}
				// 视频没放上：撤回刚放的图片，不留单独的静态图；视频名刚被别人占了就换下一个编号。
				if removeErr := os.Remove(final); removeErr != nil {
					e = errors.Join(e, removeErr)
					break
				}
				if !os.IsExist(e) {
					break
				}
			}
			if e == nil {
				nextNumber[requested] = number + 1
			}
		} else if _, statErr := os.Lstat(motionExportPath(final)); withMotion && statErr == nil {
			// 保存框只确认了替换图片，同名视频不能悄悄覆盖。
			e = i18n.NewError("error_export_motion_exists", filepath.Base(motionExportPath(final)))
		} else {
			// 保存对话框选的路径，重名时用户已确认替换。先放视频（目标不存在，不会覆盖谁），再原子替换图片；
			// 图片没换成就撤回视频，旧图片从头到尾不动。
			if withMotion {
				e = publishFile(stagedMotion, motionExportPath(final))
			}
			if e == nil {
				if e = os.Rename(staged, final); e == nil {
					slog.Info("导出覆盖目标文件", "path", final)
				} else if withMotion {
					e = errors.Join(e, os.Remove(motionExportPath(final)))
				}
			}
		}
		cleanupErr := removeIfExists(staged)
		if withMotion {
			cleanupErr = errors.Join(cleanupErr, removeIfExists(stagedMotion))
		}
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

// fitLongestEdge 把最长边缩到 maxEdge 以内，保持比例；maxEdge 为 0 或图本来就不大时原样返回。
func fitLongestEdge(img image.Image, maxEdge int) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if maxEdge <= 0 || max(w, h) <= maxEdge {
		return img
	}
	scale := float64(maxEdge) / float64(max(w, h))
	bounds := image.Rect(0, 0, max(1, int(float64(w)*scale+.5)), max(1, int(float64(h)*scale+.5)))
	// 用不预乘透明度的格式缩：预乘的 8 位会把半透明像素的颜色压糙；16 位的图缩完仍是 16 位。
	var out draw.Image = image.NewNRGBA(bounds)
	switch img.ColorModel() {
	case color.RGBA64Model, color.NRGBA64Model, color.Gray16Model:
		out = image.NewNRGBA64(bounds)
	}
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	return out
}

// motionExportPath 是实况照片视频导出后的位置：和图片同名，后缀 .mov。
func motionExportPath(image string) string {
	return strings.TrimSuffix(image, filepath.Ext(image)) + ".mov"
}

// stageMotionExport 把实况照片的视频完整复制到 dir 里的暂存文件，返回暂存路径；由调用方和图片一起发布。
func (s *Service) stageMotionExport(ctx context.Context, a model.Asset, dir string) (staged string, err error) {
	source, err := s.assetPath(a.MotionPath)
	if err != nil {
		return "", err
	}
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	temp, err := os.CreateTemp(dir, ".momento-export-*")
	if err != nil {
		return "", err
	}
	staged = temp.Name()
	_, err = io.Copy(temp, contextReader{ctx: ctx, r: in})
	if err == nil {
		err = temp.Sync()
	}
	if err = errors.Join(err, temp.Close()); err == nil {
		// 复制完到发布之间被取消，也不能接着发布。
		err = ctx.Err()
	}
	if err != nil {
		return "", errors.Join(err, removeIfExists(staged))
	}
	return staged, nil
}

// insideLibrary 按文件身份判断 path 是否在图库包里（不分大小写的文件系统、链接都算上）。
func (s *Service) insideLibrary(path string) bool {
	root, err := os.Stat(s.root)
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	for p := abs; ; {
		if info, err := os.Stat(p); err == nil && os.SameFile(info, root) {
			return true
		}
		next := filepath.Dir(p)
		if next == p {
			return false
		}
		p = next
	}
}
