//go:build windows

package media

// Windows 借系统缩略图（IShellItemImageFactory，背后是 Media Foundation 的视频缩略图处理器）拿封面，
// 纯 syscall 调 COM，不用 cgo。
// 系统缩略图自己决定截哪一帧（通常是开头附近或视频里标记的封面），没法指定时刻，所以 at 在 Windows 上不起作用。
// 系统没装对应解码器（如 N 版缺媒体功能包、没装 VP9/HEVC 扩展）或者文件坏了都拿不到缩略图，
// 两者在这个接口上分不清，统一返回 ErrPosterUnavailable，调用方用通用视频封面。

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/image/draw"
	"golang.org/x/sys/windows"
)

// systemPoster：这个平台的 poster 用系统接口，截不了的格式再交给 ffmpeg 试。
const systemPoster = true

var (
	shell32                         = windows.NewLazySystemDLL("shell32.dll")
	gdi32                           = windows.NewLazySystemDLL("gdi32.dll")
	procSHCreateItemFromParsingName = shell32.NewProc("SHCreateItemFromParsingName")
	procCreateCompatibleDC          = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC                    = gdi32.NewProc("DeleteDC")
	procGetObjectW                  = gdi32.NewProc("GetObjectW")
	procGetDIBits                   = gdi32.NewProc("GetDIBits")
	procDeleteObject                = gdi32.NewProc("DeleteObject")

	// IID_IShellItemImageFactory {bcc18b79-ba16-442f-80c4-8a59c30c463b}
	iidShellItemImageFactory = windows.GUID{
		Data1: 0xbcc18b79, Data2: 0xba16, Data3: 0x442f,
		Data4: [8]byte{0x80, 0xc4, 0x8a, 0x59, 0xc3, 0x0c, 0x46, 0x3b},
	}
)

// SIIGBF 标志：允许返回比请求大的图（之后自己缩），只要真正的缩略图、不要文件类型图标。
const (
	siigbfBiggerSizeOK  = 0x1
	siigbfThumbnailOnly = 0x8
)

type shellItemImageFactory struct {
	vtbl *shellItemImageFactoryVtbl
}

type shellItemImageFactoryVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetImage       uintptr
}

// winBitmap 对应 GDI 的 BITMAP 结构。
type winBitmap struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       uintptr
}

// bitmapInfoHeader 对应 BITMAPINFOHEADER；GetDIBits 只要 32 位 BI_RGB，不需要调色板。
type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type posterOutcome struct {
	img image.Image
	err error
}

func poster(ctx context.Context, path string, _ time.Duration, maxSize int) (image.Image, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("media: poster %s: %w", path, err)
	}
	// COM 调用是同步的、没法中途取消；放到单独的 goroutine 里，ctx 到期就先返回，
	// 那边做完会自己释放 COM 对象和位图，结果没人收就随 GC 回收
	done := make(chan posterOutcome, 1)
	go func() {
		img, err := shellThumbnail(abs, maxSize)
		done <- posterOutcome{img, err}
	}()
	select {
	case r := <-done:
		return r.img, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// shellThumbnail 在固定的系统线程上初始化 COM，取系统缩略图并转成 Go 图像。
func shellThumbnail(path string, maxSize int) (image.Image, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Shell 对象是单线程套间的，按官方建议用 STA；线程已被别人以 MTA 初始化时直接沿用
	// HRESULT 只看低 32 位
	err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE)
	var hrInit uint32
	if e, ok := err.(syscall.Errno); ok {
		hrInit = uint32(e)
	}
	switch {
	case err == nil || hrInit == uint32(windows.S_FALSE):
		defer windows.CoUninitialize()
	case hrInit == uint32(windows.RPC_E_CHANGED_MODE):
	default:
		return nil, fmt.Errorf("%w: CoInitializeEx: %v", ErrPosterUnavailable, err)
	}

	p16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("media: poster %s: %w", path, err)
	}
	var factory *shellItemImageFactory
	hr, _, _ := syscall.SyscallN(procSHCreateItemFromParsingName.Addr(),
		uintptr(unsafe.Pointer(p16)), 0,
		uintptr(unsafe.Pointer(&iidShellItemImageFactory)),
		uintptr(unsafe.Pointer(&factory)))
	if failedHR(hr) || factory == nil {
		return nil, fmt.Errorf("%w: SHCreateItemFromParsingName %s: %v", ErrPosterUnavailable, path, hresultError(hr))
	}
	defer syscall.SyscallN(factory.vtbl.Release, uintptr(unsafe.Pointer(factory)))

	var hbm uintptr
	args := []uintptr{uintptr(unsafe.Pointer(factory))}
	// SIZE{cx, cy} 按值传：64 位下 8 字节结构体放进一个寄存器，32 位下按两个 LONG 压栈
	if unsafe.Sizeof(uintptr(0)) == 8 {
		args = append(args, uintptr(uint32(maxSize))|uintptr(uint32(maxSize))<<32)
	} else {
		args = append(args, uintptr(maxSize), uintptr(maxSize))
	}
	args = append(args, siigbfBiggerSizeOK|siigbfThumbnailOnly, uintptr(unsafe.Pointer(&hbm)))
	hr, _, _ = syscall.SyscallN(factory.vtbl.GetImage, args...)
	if failedHR(hr) || hbm == 0 {
		return nil, fmt.Errorf("%w: IShellItemImageFactory.GetImage %s: %v", ErrPosterUnavailable, path, hresultError(hr))
	}
	defer procDeleteObject.Call(hbm)

	img, err := hbitmapToRGBA(hbm)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrPosterUnavailable, path, err)
	}
	return scaleToFit(img, maxSize), nil
}

// hbitmapToRGBA 用 GetDIBits 把 HBITMAP 读成 32 位 BGRA，再转成 image.RGBA。
func hbitmapToRGBA(hbm uintptr) (*image.RGBA, error) {
	var bm winBitmap
	if r, _, err := procGetObjectW.Call(hbm, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm))); r == 0 {
		return nil, fmt.Errorf("GetObject: %v", err)
	}
	w, h := int(bm.Width), int(bm.Height)
	if h < 0 {
		h = -h
	}
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty thumbnail %dx%d", w, h)
	}

	hdc, _, err := procCreateCompatibleDC.Call(0)
	if hdc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %v", err)
	}
	defer procDeleteDC.Call(hdc)

	// 高度给负数表示要自上而下的行顺序，省得自己翻转
	bi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(w),
		Height:   -int32(h),
		Planes:   1,
		BitCount: 32,
	}
	buf := make([]byte, w*h*4)
	if r, _, err := procGetDIBits.Call(hdc, hbm, 0, uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), 0); r == 0 {
		return nil, fmt.Errorf("GetDIBits: %v", err)
	}

	// 系统缩略图常见两种：alpha 全 0（其实不透明），或者带预乘 alpha（给 AlphaBlend 用）
	opaque := true
	if bm.BitsPixel == 32 {
		for i := 3; i < len(buf); i += 4 {
			if buf[i] != 0 {
				opaque = false
				break
			}
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(buf); i += 4 {
		a := buf[i+3]
		if opaque {
			a = 0xff
		}
		// BGRA → RGBA；预乘 alpha 和 image.RGBA 的约定一致，直接搬
		out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = buf[i+2], buf[i+1], buf[i], a
	}
	return out, nil
}

// scaleToFit 系统可能给出比请求更大的图，这里缩到最长边不超过 maxSize。
func scaleToFit(img *image.RGBA, maxSize int) *image.RGBA {
	b := img.Bounds()
	w, h := fitSize(b.Dx(), b.Dy(), maxSize)
	if w == b.Dx() && h == b.Dy() {
		return img
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

func failedHR(hr uintptr) bool { return int32(uint32(hr)) < 0 }

func hresultError(hr uintptr) error {
	if !failedHR(hr) {
		return fmt.Errorf("no image (HRESULT 0x%08X)", uint32(hr))
	}
	return fmt.Errorf("%w (HRESULT 0x%08X)", syscall.Errno(uint32(hr)), uint32(hr))
}
