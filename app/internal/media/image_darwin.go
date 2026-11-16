//go:build darwin

package media

// macOS 用系统 ImageIO（CGImageSource）读尺寸、出缩略图和解全尺寸图：RAW、PSD 等格式纯 Go 解不了，
// HEIC/AVIF 走系统也比纯 Go 解码快得多。通过 purego 直接调系统框架，不用 cgo。

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// SystemImageDecoder 表示这个平台能用系统完整解码图片（预览和导出也能用）。
const SystemImageDecoder = true

const (
	cfNumberSInt64Type = 4
)

type imageIOAPI struct {
	cfRelease             func(cf uintptr)
	cfURLFromPath         func(alloc uintptr, buf *byte, length int, isDir bool) uintptr
	cfNumberCreate        func(alloc uintptr, kind int, value unsafe.Pointer) uintptr
	cfNumberGetValue      func(number uintptr, kind int, value unsafe.Pointer) bool
	cfDictionaryCreate    func(alloc uintptr, keys, values unsafe.Pointer, count int, keyCallbacks, valueCallbacks uintptr) uintptr
	cfDictionaryGetValue  func(dict, key uintptr) uintptr
	sourceCreateWithURL   func(url, options uintptr) uintptr
	sourceGetCount        func(src uintptr) uintptr
	sourceCopyProperties  func(src, index, options uintptr) uintptr
	sourceCreateThumbnail func(src, index, options uintptr) uintptr

	keyCallbacks, valueCallbacks, booleanTrue                    uintptr
	thumbnailAlways, thumbnailMaxSize, thumbnailTransform, cache uintptr
	pixelWidth, pixelHeight, orientation                         uintptr
}

var loadImageIO = sync.OnceValues(func() (*imageIOAPI, error) {
	open := func(path string) (uintptr, error) {
		h, err := purego.Dlopen(path, purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			return 0, fmt.Errorf("load %s: %w", path, err)
		}
		return h, nil
	}
	cf, err := open("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation")
	if err != nil {
		return nil, err
	}
	io, err := open("/System/Library/Frameworks/ImageIO.framework/ImageIO")
	if err != nil {
		return nil, err
	}
	a := &imageIOAPI{}
	for _, f := range []struct {
		fn   any
		lib  uintptr
		name string
	}{
		{&a.cfRelease, cf, "CFRelease"},
		{&a.cfURLFromPath, cf, "CFURLCreateFromFileSystemRepresentation"},
		{&a.cfNumberCreate, cf, "CFNumberCreate"},
		{&a.cfNumberGetValue, cf, "CFNumberGetValue"},
		{&a.cfDictionaryCreate, cf, "CFDictionaryCreate"},
		{&a.cfDictionaryGetValue, cf, "CFDictionaryGetValue"},
		{&a.sourceCreateWithURL, io, "CGImageSourceCreateWithURL"},
		{&a.sourceGetCount, io, "CGImageSourceGetCount"},
		{&a.sourceCopyProperties, io, "CGImageSourceCopyPropertiesAtIndex"},
		{&a.sourceCreateThumbnail, io, "CGImageSourceCreateThumbnailAtIndex"},
	} {
		p, err := purego.Dlsym(f.lib, f.name)
		if err != nil {
			return nil, fmt.Errorf("dlsym %s: %w", f.name, err)
		}
		purego.RegisterFunc(f.fn, p)
	}
	// 回调表是结构体本身，要传它的地址；其余全局常量是指针变量，要再读一次。
	for _, v := range []struct {
		dst   *uintptr
		lib   uintptr
		name  string
		deref bool
	}{
		{&a.keyCallbacks, cf, "kCFTypeDictionaryKeyCallBacks", false},
		{&a.valueCallbacks, cf, "kCFTypeDictionaryValueCallBacks", false},
		{&a.booleanTrue, cf, "kCFBooleanTrue", true},
		{&a.thumbnailAlways, io, "kCGImageSourceCreateThumbnailFromImageAlways", true},
		{&a.thumbnailMaxSize, io, "kCGImageSourceThumbnailMaxPixelSize", true},
		{&a.thumbnailTransform, io, "kCGImageSourceCreateThumbnailWithTransform", true},
		{&a.cache, io, "kCGImageSourceShouldCache", true},
		{&a.pixelWidth, io, "kCGImagePropertyPixelWidth", true},
		{&a.pixelHeight, io, "kCGImagePropertyPixelHeight", true},
		{&a.orientation, io, "kCGImagePropertyOrientation", true},
	} {
		p, err := purego.Dlsym(v.lib, v.name)
		if err != nil {
			return nil, fmt.Errorf("dlsym %s: %w", v.name, err)
		}
		if v.deref {
			p = **(**uintptr)(unsafe.Pointer(&p))
		}
		*v.dst = p
	}
	return a, nil
})

// openSource 打开图片，读出转正后的尺寸。调用方负责 CFRelease 返回的 source。
func (a *imageIOAPI) openSource(path string) (src uintptr, w, h int, err error) {
	buf := append([]byte(path), 0)
	url := a.cfURLFromPath(0, &buf[0], len(path), false)
	if url == 0 {
		return 0, 0, 0, fmt.Errorf("media: image %s: bad path", path)
	}
	defer a.cfRelease(url)
	src = a.sourceCreateWithURL(url, 0)
	if src == 0 || a.sourceGetCount(src) == 0 {
		if src != 0 {
			a.cfRelease(src)
		}
		return 0, 0, 0, fmt.Errorf("%w: %s", ErrUnsupported, path)
	}
	props := a.sourceCopyProperties(src, 0, 0)
	if props == 0 {
		a.cfRelease(src)
		return 0, 0, 0, fmt.Errorf("%w: %s: no image properties", ErrUnsupported, path)
	}
	defer a.cfRelease(props)
	number := func(key uintptr) int {
		var v int64
		if n := a.cfDictionaryGetValue(props, key); n != 0 && a.cfNumberGetValue(n, cfNumberSInt64Type, unsafe.Pointer(&v)) {
			return int(v)
		}
		return 0
	}
	w, h = number(a.pixelWidth), number(a.pixelHeight)
	if w <= 0 || h <= 0 {
		a.cfRelease(src)
		return 0, 0, 0, fmt.Errorf("%w: %s: no pixel size", ErrUnsupported, path)
	}
	// 方向 5～8 要转 90°，宽高对调。
	if o := number(a.orientation); o >= 5 && o <= 8 {
		w, h = h, w
	}
	return src, w, h, nil
}

// thumbnail 让系统按方向转正并缩到最长边不超过 maxSize（小图不放大），再画成 RGBA。
func (a *imageIOAPI) thumbnail(path string, src uintptr, maxSize int) (image.Image, error) {
	size := int64(maxSize)
	number := a.cfNumberCreate(0, cfNumberSInt64Type, unsafe.Pointer(&size))
	if number == 0 {
		return nil, errors.New("media: image: CFNumberCreate failed")
	}
	defer a.cfRelease(number)
	// 不用文件里自带的小缩略图（常常只有 160～320 像素），一律从原图生成。
	keys := [4]uintptr{a.thumbnailAlways, a.thumbnailMaxSize, a.thumbnailTransform, a.cache}
	values := [4]uintptr{a.booleanTrue, number, a.booleanTrue, a.booleanTrue}
	options := a.cfDictionaryCreate(0, unsafe.Pointer(&keys[0]), unsafe.Pointer(&values[0]), len(keys), a.keyCallbacks, a.valueCallbacks)
	if options == 0 {
		return nil, errors.New("media: image: CFDictionaryCreate failed")
	}
	defer a.cfRelease(options)
	cg := a.sourceCreateThumbnail(src, 0, options)
	if cg == 0 {
		return nil, fmt.Errorf("media: image %s: system decoder failed", path)
	}
	defer a.cfRelease(cg)
	av, err := loadAV()
	if err != nil {
		return nil, err
	}
	return av.toRGBA(cg, maxSize)
}

// ImageSize 读图片转正后的宽高，不解码像素。
func ImageSize(path string) (int, int, error) {
	a, err := loadImageIO()
	if err != nil {
		return 0, 0, err
	}
	src, w, h, err := a.openSource(path)
	if err != nil {
		return 0, 0, err
	}
	a.cfRelease(src)
	return w, h, nil
}

// ImageThumbnail 出转正后的缩略图，最长边不超过 maxSize。
func ImageThumbnail(path string, maxSize int) (image.Image, error) {
	a, err := loadImageIO()
	if err != nil {
		return nil, err
	}
	src, _, _, err := a.openSource(path)
	if err != nil {
		return nil, err
	}
	defer a.cfRelease(src)
	return a.thumbnail(path, src, maxSize)
}

// PreviewImage 给预览用，macOS 上就是全尺寸图。
func PreviewImage(path string) (image.Image, error) { return DecodeImage(path) }

// DecodeImage 解出转正后的全尺寸图（色彩转成 sRGB），给预览和导出用。
func DecodeImage(path string) (image.Image, error) {
	a, err := loadImageIO()
	if err != nil {
		return nil, err
	}
	src, w, h, err := a.openSource(path)
	if err != nil {
		return nil, err
	}
	defer a.cfRelease(src)
	return a.thumbnail(path, src, max(w, h))
}
