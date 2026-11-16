package assets

import (
	"context"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/gen2brain/avif"
	"github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// Decode 返回已经转正的第一帧，原文件保持不动。
func Decode(path string) (image.Image, error) {
	img, _, _, _, err := PrepareImage(path)
	return img, err
}

// maxImagePixels 是能解码的最大像素数（2 亿），测试里可以调小。
var maxImagePixels int64 = 200_000_000

// thumbnailSize 是缩略图最长边。
const thumbnailSize = 512

// decodeImage 解出第一帧，还没转正。解码前先只读尺寸：像素太多直接拒绝；
// admit 不为空时先拿尺寸去排队申请内存，返回的 release 由调用方在用完大图后调用。
func decodeImage(ctx context.Context, path string, admit func(context.Context, image.Config, bool) (func(), error)) (image.Image, string, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, "", nil, err
	}
	defer f.Close()
	var img image.Image
	ext := strings.ToLower(filepath.Ext(path))
	// 读不出尺寸时照常解码，由解码器给出真正的错误。
	var config image.Config
	switch ext {
	case ".heic", ".heif":
		config, err = heic.DecodeConfig(f)
	case ".avif":
		config, err = avif.DecodeConfig(f)
	default:
		config, _, err = image.DecodeConfig(f)
	}
	if err == nil && int64(config.Width)*int64(config.Height) > maxImagePixels {
		return nil, "", nil, i18n.WrapError(i18n.NewError("error_image_too_large", config.Width, config.Height), "error_decode_image", filepath.Base(path))
	}
	release := func() {}
	if admit != nil {
		if release, err = admit(ctx, config, err == nil); err != nil {
			return nil, "", nil, err
		}
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		release()
		return nil, "", nil, err
	}
	switch ext {
	case ".heic", ".heif":
		img, err = heic.Decode(f)
	case ".avif":
		img, err = avif.Decode(f, avif.Options{AutoRotate: true})
	default:
		img, _, err = image.Decode(f)
	}
	if err != nil {
		release()
		return nil, "", nil, i18n.WrapError(err, "error_decode_image", filepath.Base(path))
	}
	return img, ext, release, nil
}

// decoderOriented 表示解码器已经按容器方向转正（HEIC、AVIF），不能再转一次。
func decoderOriented(ext string) bool {
	return ext == ".heic" || ext == ".heif" || ext == ".avif"
}

// PrepareImage 返回全尺寸、已转正的第一帧和元数据，给预览和导出用。
func PrepareImage(path string) (image.Image, map[string]string, int, string, error) {
	img, ext, _, err := decodeImage(context.Background(), path, nil)
	if err != nil {
		return nil, nil, 0, "", err
	}
	metadata, orientation, profile := readMetadata(path, img)
	if !decoderOriented(ext) {
		img = orient(img, orientation)
	}
	return img, metadata, orientation, profile, nil
}

// preparedImage 是导入和重建缩略图用的结果：只留已转正的缩略图，全尺寸大图用完就丢。
type preparedImage struct {
	thumbnail     image.Image
	width, height int // 原图转正后的尺寸
	metadata      map[string]string
	orientation   int
	profile       string
	// kind 和 durationMS 只有视频会填；图片的 kind 留空，按图片处理。
	kind       string
	durationMS int64
}

// prepareThumbnail 解码后先缩小再转正，避免对全尺寸大图做转正和多次高质量缩放。
// 大图解码要按内存预算排队，缩好后马上归还额度。
func prepareThumbnail(ctx context.Context, path string) (preparedImage, error) {
	img, ext, release, err := decodeImage(ctx, path, decodeBudget.admit)
	if err != nil {
		return preparedImage{}, err
	}
	defer release()
	metadata, orientation, profile := readMetadata(path, img)
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// 转正前后最长边不变，缩略图尺寸可以先按原方向算。
	dw, dh := fitWithin(w, h, thumbnailSize)
	thumb := scaleTo(img, dw, dh)
	// 缩好后大图就没用了，先还额度让下一张开始解码。
	release()
	if !decoderOriented(ext) {
		thumb = orient(thumb, orientation)
		if orientation >= 5 && orientation <= 8 {
			w, h = h, w
		}
	}
	return preparedImage{thumbnail: thumb, width: w, height: h, metadata: metadata, orientation: orientation, profile: profile}, nil
}

// fitWithin 按比例缩到最长边不超过 longest，本来就小的不放大。
func fitWithin(w, h, longest int) (int, int) {
	if max(w, h) <= longest {
		return w, h
	}
	if w >= h {
		return longest, max(1, h*longest/w)
	}
	return max(1, w*longest/h), longest
}

func resized(src image.Image, longest int) image.Image {
	b := src.Bounds()
	w, h := fitWithin(b.Dx(), b.Dy(), longest)
	return scaleTo(src, w, h)
}

// scaleTo 缩到指定尺寸，尺寸没变直接返回原图。
// 原图比目标大两倍以上时，先按整数倍取块平均快速缩到目标的 2～4 倍，再用 CatmullRom 缩到最终尺寸；
// 块平均每个像素都参与计算，不会像跳点采样那样出摩尔纹。
func scaleTo(src image.Image, w, h int) image.Image {
	b := src.Bounds()
	if b.Dx() == w && b.Dy() == h {
		return src
	}
	if k := max(b.Dx(), b.Dy()) / (2 * max(w, h)); k >= 2 {
		src = shrink(src, k)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}

// orient 按 EXIF 方向转正，方向 1 或无效时原样返回。常见像素格式直接搬字节，其他格式逐像素转换。
func orient(src image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return src
	}
	if dst := transformFast(src, orientation); dst != nil {
		return dst
	}
	return transformSlow(src, orientation)
}

// sourceMap 给出转正后 (dx,dy) 在原图里的位置：x = x0 + dx*xdx + dy*xdy，y = y0 + dx*ydx + dy*ydy。
type sourceMap struct{ x0, y0, xdx, xdy, ydx, ydy int }

func orientationMap(orientation, w, h int) sourceMap {
	switch orientation {
	case 2:
		return sourceMap{w - 1, 0, -1, 0, 0, 1}
	case 3:
		return sourceMap{w - 1, h - 1, -1, 0, 0, -1}
	case 4:
		return sourceMap{0, h - 1, 1, 0, 0, -1}
	case 5:
		return sourceMap{0, 0, 0, 1, 1, 0}
	case 6:
		return sourceMap{0, h - 1, 0, 1, -1, 0}
	case 7:
		return sourceMap{w - 1, h - 1, 0, -1, -1, 0}
	case 8:
		return sourceMap{w - 1, 0, 0, -1, 1, 0}
	}
	return sourceMap{0, 0, 1, 0, 0, 1}
}

func orientedSize(orientation, w, h int) (int, int) {
	if orientation >= 5 {
		return h, w
	}
	return w, h
}

// transformFast 处理 NRGBA、RGBA、YCbCr、Gray，结果统一是 NRGBA（和逐像素转换的结果一致）；其他格式返回 nil。
func transformFast(src image.Image, orientation int) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	m := orientationMap(orientation, w, h)
	dw, dh := orientedSize(orientation, w, h)
	switch s := src.(type) {
	case *image.NRGBA, *image.RGBA, *image.Gray:
		var pix []uint8
		var stride, size int
		switch s := s.(type) {
		case *image.NRGBA:
			pix, stride, size = s.Pix, s.Stride, 4
		case *image.RGBA:
			pix, stride, size = s.Pix, s.Stride, 4
		case *image.Gray:
			pix, stride, size = s.Pix, s.Stride, 1
		}
		dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
		stepX := m.xdx*size + m.ydx*stride
		stepY := m.xdy*size + m.ydy*stride
		base := m.y0*stride + m.x0*size
		_, premultiplied := s.(*image.RGBA)
		for dy := range dh {
			off := base + dy*stepY
			row := dst.Pix[dy*dst.Stride : dy*dst.Stride+dw*4]
			for i := 0; i < len(row); i += 4 {
				if size == 1 {
					v := pix[off]
					row[i], row[i+1], row[i+2], row[i+3] = v, v, v, 0xff
				} else {
					p := pix[off : off+4 : off+4]
					row[i], row[i+1], row[i+2], row[i+3] = p[0], p[1], p[2], p[3]
					if premultiplied && p[3] != 0xff {
						unpremultiply(row[i : i+4 : i+4])
					}
				}
				off += stepX
			}
		}
		return dst
	case *image.YCbCr:
		dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
		for dy := range dh {
			x, y := b.Min.X+m.x0+dy*m.xdy, b.Min.Y+m.y0+dy*m.ydy
			row := dst.Pix[dy*dst.Stride : dy*dst.Stride+dw*4]
			for i := 0; i < len(row); i += 4 {
				yi, ci := s.YOffset(x, y), s.COffset(x, y)
				row[i], row[i+1], row[i+2] = color.YCbCrToRGB(s.Y[yi], s.Cb[ci], s.Cr[ci])
				row[i+3] = 0xff
				x += m.xdx
				y += m.ydx
			}
		}
		return dst
	}
	return nil
}

// unpremultiply 把预乘的 RGBA 换成非预乘，算法与 color.NRGBAModel 相同。
func unpremultiply(p []uint8) {
	a := uint32(p[3]) * 0x101
	if a == 0 {
		p[0], p[1], p[2] = 0, 0, 0
		return
	}
	for i := range 3 {
		p[i] = uint8((uint32(p[i]) * 0x101 * 0xffff / a) >> 8)
	}
}

// transformSlow 逐像素转换，16 位通道保留精度，其他格式转成 NRGBA。
func transformSlow(src image.Image, orientation int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	m := orientationMap(orientation, w, h)
	dw, dh := orientedSize(orientation, w, h)
	var dst interface {
		image.Image
		Set(int, int, color.Color)
	}
	switch src.ColorModel() {
	case color.Gray16Model, color.RGBA64Model, color.NRGBA64Model:
		dst = image.NewNRGBA64(image.Rect(0, 0, dw, dh))
	default:
		dst = image.NewNRGBA(image.Rect(0, 0, dw, dh))
	}
	for dy := range dh {
		for dx := range dw {
			dst.Set(dx, dy, src.At(b.Min.X+m.x0+dx*m.xdx+dy*m.xdy, b.Min.Y+m.y0+dx*m.ydx+dy*m.ydy))
		}
	}
	return dst
}

// shrink 按 k×k 的块取平均缩小，边上不足一块的按实际像素平均。透明像素按预乘平均，不会把边缘染黑。
func shrink(src image.Image, k int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := (w+k-1)/k, (h+k-1)/k
	// sums 按目标列累加 k 行，满一块就写出一行。
	sums := make([]uint64, dw*4)
	var out image.Image
	var flush func(dy, rows int)
	var add func(sy int)
	switch s := src.(type) {
	case *image.YCbCr:
		dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
		out = dst
		// COffset 行列可分离，先算好每列的色度偏移。
		columns := make([]int, w)
		origin := s.COffset(b.Min.X, b.Min.Y)
		for x := range w {
			columns[x] = s.COffset(b.Min.X+x, b.Min.Y) - origin
		}
		add = func(sy int) {
			yRow := s.Y[s.YOffset(b.Min.X, b.Min.Y+sy):]
			cRow := s.COffset(b.Min.X, b.Min.Y+sy)
			for dx := range dw {
				var ys, cbs, crs uint64
				for x := dx * k; x < min(w, dx*k+k); x++ {
					ys += uint64(yRow[x])
					ci := cRow + columns[x]
					cbs += uint64(s.Cb[ci])
					crs += uint64(s.Cr[ci])
				}
				sums[dx*4] += ys
				sums[dx*4+1] += cbs
				sums[dx*4+2] += crs
			}
		}
		flush = func(dy, rows int) {
			row := dst.Pix[dy*dst.Stride:]
			for dx := range dw {
				n := uint64(rows * (min(w, dx*k+k) - dx*k))
				yy, cb, cr := (sums[dx*4]+n/2)/n, (sums[dx*4+1]+n/2)/n, (sums[dx*4+2]+n/2)/n
				row[dx*4], row[dx*4+1], row[dx*4+2] = color.YCbCrToRGB(uint8(yy), uint8(cb), uint8(cr))
				row[dx*4+3] = 0xff
			}
		}
	case *image.NRGBA:
		dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
		out = dst
		add = func(sy int) {
			p := s.Pix[s.PixOffset(b.Min.X, b.Min.Y+sy):]
			for dx := range dw {
				var r, g, bl, a uint64
				for x := dx * k; x < min(w, dx*k+k); x++ {
					q := p[x*4 : x*4+4 : x*4+4]
					alpha := uint64(q[3])
					r += uint64(q[0]) * alpha
					g += uint64(q[1]) * alpha
					bl += uint64(q[2]) * alpha
					a += alpha
				}
				sums[dx*4] += r
				sums[dx*4+1] += g
				sums[dx*4+2] += bl
				sums[dx*4+3] += a
			}
		}
		flush = func(dy, rows int) {
			row := dst.Pix[dy*dst.Stride:]
			for dx := range dw {
				n := uint64(rows * (min(w, dx*k+k) - dx*k))
				a := sums[dx*4+3]
				if a == 0 {
					continue
				}
				for c := range 3 {
					row[dx*4+c] = uint8((sums[dx*4+c] + a/2) / a)
				}
				row[dx*4+3] = uint8((a + n/2) / n)
			}
		}
	case *image.RGBA, *image.Gray:
		size, stride, pix := 4, 0, []uint8(nil)
		if g, ok := s.(*image.Gray); ok {
			size, stride, pix = 1, g.Stride, g.Pix[g.PixOffset(b.Min.X, b.Min.Y):]
		} else {
			r := s.(*image.RGBA)
			stride, pix = r.Stride, r.Pix[r.PixOffset(b.Min.X, b.Min.Y):]
		}
		dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
		out = dst
		add = func(sy int) {
			p := pix[sy*stride:]
			for dx := range dw {
				var c [4]uint64
				for x := dx * k; x < min(w, dx*k+k); x++ {
					if size == 1 {
						c[0] += uint64(p[x])
						continue
					}
					q := p[x*4 : x*4+4 : x*4+4]
					c[0] += uint64(q[0])
					c[1] += uint64(q[1])
					c[2] += uint64(q[2])
					c[3] += uint64(q[3])
				}
				for i := range 4 {
					sums[dx*4+i] += c[i]
				}
			}
		}
		flush = func(dy, rows int) {
			row := dst.Pix[dy*dst.Stride:]
			for dx := range dw {
				n := uint64(rows * (min(w, dx*k+k) - dx*k))
				if size == 1 {
					v := uint8((sums[dx*4] + n/2) / n)
					row[dx*4], row[dx*4+1], row[dx*4+2], row[dx*4+3] = v, v, v, 0xff
					continue
				}
				for c := range 4 {
					row[dx*4+c] = uint8((sums[dx*4+c] + n/2) / n)
				}
			}
		}
	default:
		// 其他格式（调色板、CMYK、16 位等）逐像素读，按 16 位预乘平均。
		dst := image.NewRGBA64(image.Rect(0, 0, dw, dh))
		out = dst
		add = func(sy int) {
			for x := range w {
				r, g, bl, a := src.At(b.Min.X+x, b.Min.Y+sy).RGBA()
				dx := x / k
				sums[dx*4] += uint64(r)
				sums[dx*4+1] += uint64(g)
				sums[dx*4+2] += uint64(bl)
				sums[dx*4+3] += uint64(a)
			}
		}
		flush = func(dy, rows int) {
			for dx := range dw {
				n := uint64(rows * (min(w, dx*k+k) - dx*k))
				dst.SetRGBA64(dx, dy, color.RGBA64{uint16((sums[dx*4] + n/2) / n), uint16((sums[dx*4+1] + n/2) / n), uint16((sums[dx*4+2] + n/2) / n), uint16((sums[dx*4+3] + n/2) / n)})
			}
		}
	}
	for dy := range dh {
		clear(sums)
		rows := min(h, dy*k+k) - dy*k
		for sy := dy * k; sy < dy*k+rows; sy++ {
			add(sy)
		}
		flush(dy, rows)
	}
	return out
}

func imageColorModel(img image.Image) string {
	switch img.ColorModel() {
	case color.GrayModel, color.Gray16Model:
		return "Gray"
	case color.CMYKModel:
		return "CMYK"
	default:
		return "RGB"
	}
}
