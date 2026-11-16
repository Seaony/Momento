//go:build darwin

package media

// macOS 用 AVFoundation 的 AVAssetImageGenerator 截帧，通过 purego 直接调系统框架，不用 cgo。
// 用异步接口 generateCGImagesAsynchronouslyForTimes:completionHandler:，
// 这样 ctx 取消时能调 cancelAllCGImageGeneration 真正停掉解码，不会被坏文件拖住。

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"runtime"
	"structs"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// systemPoster：这个平台的 poster 用系统接口，截不了的格式再交给 ffmpeg 试。
const systemPoster = true

// cmTime 对应 CoreMedia 的 CMTime（24 字节）。
type cmTime struct {
	_         structs.HostLayout
	Value     int64
	Timescale int32
	Flags     uint32
	Epoch     int64
}

const cmTimeFlagsValid = 1

type cgSize struct {
	_    structs.HostLayout
	W, H float64
}

type cgRect struct {
	_          structs.HostLayout
	X, Y, W, H float64
}

// AVFoundation 的错误码（AVError.h）。
const (
	avErrorFileFormatNotRecognized = -11828
	avErrorDecoderNotFound         = -11833
)

// AVKeyValueStatus：时长加载完成。
const avKeyValueStatusLoaded = 2

// AVAssetImageGeneratorResult。
const (
	avGenSucceeded = 0
	avGenCancelled = 2
)

// CGBitmapInfo：RGBA 字节顺序、预乘 alpha，正好和 image.RGBA 的内存布局一致。
const (
	cgImageAlphaPremultipliedLast = 1
	cgBitmapByteOrder32Big        = 4 << 12
	cgInterpolationHigh           = 3
)

type avAPI struct {
	poolPush func() uintptr
	poolPop  func(pool uintptr)

	// 带结构体参数或返回值的 objc_msgSend 要单独声明类型
	msgGetTime   func(obj objc.ID, sel objc.SEL) cmTime
	msgSetTime   func(obj objc.ID, sel objc.SEL, t cmTime)
	msgSetSize   func(obj objc.ID, sel objc.SEL, s cgSize)
	msgValueTime func(cls objc.ID, sel objc.SEL, t cmTime) objc.ID
	msgString    func(obj objc.ID, sel objc.SEL) string
	msgFromUTF8  func(cls objc.ID, sel objc.SEL, s string) objc.ID
	msgInt       func(obj objc.ID, sel objc.SEL) int
	msgStatus    func(obj objc.ID, sel objc.SEL, key objc.ID, errOut *objc.ID) int

	cgImageRetain          func(img uintptr) uintptr
	cgImageRelease         func(img uintptr)
	cgImageGetWidth        func(img uintptr) uintptr
	cgImageGetHeight       func(img uintptr) uintptr
	cgColorSpaceCreateName func(name uintptr) uintptr
	cgColorSpaceRelease    func(cs uintptr)
	cgBitmapContextCreate  func(data unsafe.Pointer, w, h, bitsPerComponent, bytesPerRow uintptr, cs uintptr, info uint32) uintptr
	cgBitmapContextData    func(ctx uintptr) unsafe.Pointer
	cgBitmapContextStride  func(ctx uintptr) uintptr
	cgContextSetInterp     func(ctx uintptr, quality int32)
	cgContextDrawImage     func(ctx uintptr, r cgRect, img uintptr)
	cgContextRelease       func(ctx uintptr)
	kCGColorSpaceSRGB      uintptr

	clsNSString, clsNSURL, clsNSArray, clsNSValue, clsAsset, clsGenerator objc.ID

	selStringWithUTF8, selFileURL, selAssetWithURL, selDuration, selGenWithAsset                  objc.SEL
	selSetApplies, selSetMaxSize, selSetTolBefore, selSetTolAfter, selValueWithTime, selArrayWith objc.SEL
	selGenerate, selCancelAll, selRetain, selRelease, selLocalizedDesc, selDomain, selCode        objc.SEL
	selUserInfo, selObjectForKey, selUTF8, selLoadValues, selCancelLoading, selStatusOfValue      objc.SEL
}

var loadAV = sync.OnceValues(func() (*avAPI, error) {
	open := func(path string) (uintptr, error) {
		h, err := purego.Dlopen(path, purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			return 0, fmt.Errorf("load %s: %w", path, err)
		}
		return h, nil
	}
	libObjC, err := open("/usr/lib/libobjc.A.dylib")
	if err != nil {
		return nil, err
	}
	if _, err := open("/System/Library/Frameworks/Foundation.framework/Foundation"); err != nil {
		return nil, err
	}
	if _, err := open("/System/Library/Frameworks/CoreMedia.framework/CoreMedia"); err != nil {
		return nil, err
	}
	if _, err := open("/System/Library/Frameworks/AVFoundation.framework/AVFoundation"); err != nil {
		return nil, err
	}
	libCG, err := open("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics")
	if err != nil {
		return nil, err
	}
	sym := func(lib uintptr, name string) (uintptr, error) {
		p, err := purego.Dlsym(lib, name)
		if err != nil {
			return 0, fmt.Errorf("dlsym %s: %w", name, err)
		}
		return p, nil
	}

	a := &avAPI{}
	msgSend, err := sym(libObjC, "objc_msgSend")
	if err != nil {
		return nil, err
	}
	// amd64 上返回大于 16 字节结构体的消息要走 objc_msgSend_stret；arm64 统一用 objc_msgSend
	msgSendStret := msgSend
	if runtime.GOARCH == "amd64" {
		if msgSendStret, err = sym(libObjC, "objc_msgSend_stret"); err != nil {
			return nil, err
		}
	}
	purego.RegisterFunc(&a.msgGetTime, msgSendStret)
	purego.RegisterFunc(&a.msgSetTime, msgSend)
	purego.RegisterFunc(&a.msgSetSize, msgSend)
	purego.RegisterFunc(&a.msgValueTime, msgSend)
	purego.RegisterFunc(&a.msgString, msgSend)
	purego.RegisterFunc(&a.msgFromUTF8, msgSend)
	purego.RegisterFunc(&a.msgInt, msgSend)
	purego.RegisterFunc(&a.msgStatus, msgSend)

	for _, f := range []struct {
		fn   any
		lib  uintptr
		name string
	}{
		{&a.poolPush, libObjC, "objc_autoreleasePoolPush"},
		{&a.poolPop, libObjC, "objc_autoreleasePoolPop"},
		{&a.cgImageRetain, libCG, "CGImageRetain"},
		{&a.cgImageRelease, libCG, "CGImageRelease"},
		{&a.cgImageGetWidth, libCG, "CGImageGetWidth"},
		{&a.cgImageGetHeight, libCG, "CGImageGetHeight"},
		{&a.cgColorSpaceCreateName, libCG, "CGColorSpaceCreateWithName"},
		{&a.cgColorSpaceRelease, libCG, "CGColorSpaceRelease"},
		{&a.cgBitmapContextCreate, libCG, "CGBitmapContextCreate"},
		{&a.cgBitmapContextData, libCG, "CGBitmapContextGetData"},
		{&a.cgBitmapContextStride, libCG, "CGBitmapContextGetBytesPerRow"},
		{&a.cgContextSetInterp, libCG, "CGContextSetInterpolationQuality"},
		{&a.cgContextDrawImage, libCG, "CGContextDrawImage"},
		{&a.cgContextRelease, libCG, "CGContextRelease"},
	} {
		p, err := sym(f.lib, f.name)
		if err != nil {
			return nil, err
		}
		purego.RegisterFunc(f.fn, p)
	}
	// kCGColorSpaceSRGB 是全局变量，dlsym 拿到的是变量地址，要再读一次
	p, err := sym(libCG, "kCGColorSpaceSRGB")
	if err != nil {
		return nil, err
	}
	a.kCGColorSpaceSRGB = **(**uintptr)(unsafe.Pointer(&p))

	cls := func(name string) (objc.ID, error) {
		c := objc.GetClass(name)
		if c == 0 {
			return 0, fmt.Errorf("objc class %s not found", name)
		}
		return objc.ID(c), nil
	}
	for _, c := range []struct {
		dst  *objc.ID
		name string
	}{
		{&a.clsNSString, "NSString"},
		{&a.clsNSURL, "NSURL"},
		{&a.clsNSArray, "NSArray"},
		{&a.clsNSValue, "NSValue"},
		{&a.clsAsset, "AVURLAsset"},
		{&a.clsGenerator, "AVAssetImageGenerator"},
	} {
		if *c.dst, err = cls(c.name); err != nil {
			return nil, err
		}
	}
	sel := objc.RegisterName
	a.selStringWithUTF8 = sel("stringWithUTF8String:")
	a.selFileURL = sel("fileURLWithPath:")
	a.selAssetWithURL = sel("URLAssetWithURL:options:")
	a.selDuration = sel("duration")
	a.selGenWithAsset = sel("assetImageGeneratorWithAsset:")
	a.selSetApplies = sel("setAppliesPreferredTrackTransform:")
	a.selSetMaxSize = sel("setMaximumSize:")
	a.selSetTolBefore = sel("setRequestedTimeToleranceBefore:")
	a.selSetTolAfter = sel("setRequestedTimeToleranceAfter:")
	a.selValueWithTime = sel("valueWithCMTime:")
	a.selArrayWith = sel("arrayWithObject:")
	a.selGenerate = sel("generateCGImagesAsynchronouslyForTimes:completionHandler:")
	a.selCancelAll = sel("cancelAllCGImageGeneration")
	a.selRetain = sel("retain")
	a.selRelease = sel("release")
	a.selLocalizedDesc = sel("localizedDescription")
	a.selDomain = sel("domain")
	a.selCode = sel("code")
	a.selUserInfo = sel("userInfo")
	a.selObjectForKey = sel("objectForKey:")
	a.selUTF8 = sel("UTF8String")
	a.selLoadValues = sel("loadValuesAsynchronouslyForKeys:completionHandler:")
	a.selCancelLoading = sel("cancelLoading")
	a.selStatusOfValue = sel("statusOfValueForKey:error:")
	return a, nil
})

// genResult 是完成回调交回来的结果；回调跑在 AVFoundation 自己的队列上。
type genResult struct {
	img    uintptr // 已 CGImageRetain，用完要释放
	result int
	nsErr  *nsError
}

type nsError struct {
	domain string
	code   int
	desc   string
	under  string // NSUnderlyingError 的描述，常带着更底层的 OSStatus
}

func (e *nsError) String() string {
	s := fmt.Sprintf("%s (%s %d)", e.desc, e.domain, e.code)
	if e.under != "" {
		s += "; " + e.under
	}
	return s
}

// withPool 在固定线程上开一个 autorelease pool 跑 fn；push/pop 必须在同一个系统线程上。
func (a *avAPI) withPool(fn func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	pool := a.poolPush()
	defer a.poolPop(pool)
	fn()
}

func poster(ctx context.Context, path string, at time.Duration, maxSize int) (image.Image, error) {
	a, err := loadAV()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPosterUnavailable, err)
	}

	var asset objc.ID
	a.withPool(func() {
		nsPath := a.msgFromUTF8(a.clsNSString, a.selStringWithUTF8, path)
		url := a.clsNSURL.Send(a.selFileURL, nsPath)
		asset = a.clsAsset.Send(a.selAssetWithURL, url, objc.ID(0))
		asset.Send(a.selRetain)
	})
	releaseAsset := func() { a.withPool(func() { asset.Send(a.selRelease) }) }

	// 视频比 at 短（或时长读不出）就取第一帧
	t := cmTime{Timescale: 1, Flags: cmTimeFlagsValid}
	if at > 0 {
		d, err := a.loadDuration(ctx, path, asset)
		if err != nil {
			releaseAsset()
			return nil, err
		}
		if d.Flags&cmTimeFlagsValid != 0 && d.Timescale > 0 &&
			at < time.Duration(float64(d.Value)/float64(d.Timescale)*float64(time.Second)) {
			t = cmTime{Value: at.Microseconds(), Timescale: 1_000_000, Flags: cmTimeFlagsValid}
		}
	}

	done := make(chan genResult, 1)
	// 只请求一个时刻，回调只会来一次（取消时也会来，result 为 cancelled）
	block := objc.NewBlock(func(_ objc.Block, _ cmTime, img uintptr, _ cmTime, result int, errObj objc.ID) {
		r := genResult{result: result}
		if result == avGenSucceeded && img != 0 {
			r.img = a.cgImageRetain(img)
		}
		if errObj != 0 {
			r.nsErr = a.describeError(errObj)
		}
		done <- r
	})

	var gen objc.ID
	a.withPool(func() {
		gen = a.clsGenerator.Send(a.selGenWithAsset, asset)
		gen.Send(a.selRetain)
		gen.Send(a.selSetApplies, true)
		// maximumSize 按转正后的画面算，只缩不放；下面画进位图时还会按 maxSize 再兜一次
		a.msgSetSize(gen, a.selSetMaxSize, cgSize{W: float64(maxSize), H: float64(maxSize)})
		// 容差为 0：精确取 at 那一帧，测试也能据此确认时刻对不对；封面只截一帧，多解几帧的代价可以接受
		zero := cmTime{Timescale: 1, Flags: cmTimeFlagsValid}
		a.msgSetTime(gen, a.selSetTolBefore, zero)
		a.msgSetTime(gen, a.selSetTolAfter, zero)

		times := a.clsNSArray.Send(a.selArrayWith, a.msgValueTime(a.clsNSValue, a.selValueWithTime, t))
		gen.Send(a.selGenerate, times, block)
		// 生成器自己持有 asset，这里放掉我们那份
		asset.Send(a.selRelease)
	})
	// AVFoundation 已经复制了 block，这里释放自己那份
	block.Release()

	select {
	case r := <-done:
		a.withPool(func() { gen.Send(a.selRelease) })
		return a.finish(path, r, maxSize)
	case <-ctx.Done():
		a.withPool(func() { gen.Send(a.selCancelAll) })
		// 取消后回调仍会来一次，在后台等它把图和生成器释放掉，调用方不用等
		go func() {
			r := <-done
			if r.img != 0 {
				a.cgImageRelease(r.img)
			}
			a.withPool(func() { gen.Send(a.selRelease) })
		}()
		return nil, ctx.Err()
	}
}

// finish 把回调结果变成 Go 图像或错误。
func (a *avAPI) finish(path string, r genResult, maxSize int) (image.Image, error) {
	if r.img != 0 {
		defer a.cgImageRelease(r.img)
		return a.toRGBA(r.img, maxSize)
	}
	if r.result == avGenCancelled {
		return nil, fmt.Errorf("media: poster %s: generation cancelled", path)
	}
	if r.nsErr == nil {
		return nil, fmt.Errorf("media: poster %s: AVFoundation returned no image (result %d)", path, r.result)
	}
	return nil, posterError(path, r.nsErr)
}

// posterError 区分“系统解不了”和“文件坏了”：
// 缺解码器、或者是 AVFoundation 不认的 WebM/Matroska 容器，算这台机器没能力，返回 ErrPosterUnavailable；
// 其余（如 -11829 解析失败）当作文件本身有问题。
func posterError(path string, e *nsError) error {
	if e.code == avErrorDecoderNotFound ||
		(e.code == avErrorFileFormatNotRecognized && isMatroska(path)) {
		return fmt.Errorf("%w: %s: %s", ErrPosterUnavailable, path, e)
	}
	return fmt.Errorf("media: poster %s: %s", path, e)
}

// loadDuration 异步加载 asset 的时长。同步读 duration 会阻塞且没法取消（实测并发时能卡近 1 秒），
// 所以用 loadValuesAsynchronouslyForKeys，ctx 取消时调 cancelLoading。
func (a *avAPI) loadDuration(ctx context.Context, path string, asset objc.ID) (cmTime, error) {
	loaded := make(chan struct{}, 1)
	block := objc.NewBlock(func(_ objc.Block) { loaded <- struct{}{} })
	var key objc.ID
	a.withPool(func() {
		// 加载回调到来之前多持有一份 asset，取消后由后台等待的 goroutine 释放
		asset.Send(a.selRetain)
		key = a.msgFromUTF8(a.clsNSString, a.selStringWithUTF8, "duration")
		key.Send(a.selRetain)
		asset.Send(a.selLoadValues, a.clsNSArray.Send(a.selArrayWith, key), block)
	})
	block.Release()
	releasePending := func() { a.withPool(func() { asset.Send(a.selRelease) }) }

	select {
	case <-loaded:
		releasePending()
	case <-ctx.Done():
		a.withPool(func() {
			asset.Send(a.selCancelLoading)
			key.Send(a.selRelease)
		})
		go func() {
			<-loaded
			releasePending()
		}()
		return cmTime{}, ctx.Err()
	}

	var d cmTime
	var status int
	var nerr *nsError
	a.withPool(func() {
		var errObj objc.ID
		status = a.msgStatus(asset, a.selStatusOfValue, key, &errObj)
		key.Send(a.selRelease)
		switch {
		case status == avKeyValueStatusLoaded:
			d = a.msgGetTime(asset, a.selDuration)
		case errObj != 0:
			nerr = a.describeError(errObj)
		}
	})
	switch {
	case status == avKeyValueStatusLoaded:
		return d, nil
	case nerr != nil:
		return cmTime{}, posterError(path, nerr)
	}
	return cmTime{}, fmt.Errorf("media: poster %s: loading duration ended with status %d", path, status)
}

// describeError 读出 NSError 的描述；在回调线程上调用，GCD 队列自带 autorelease pool。
func (a *avAPI) describeError(e objc.ID) *nsError {
	str := func(s objc.ID) string {
		if s == 0 {
			return ""
		}
		return a.msgString(s, a.selUTF8)
	}
	out := &nsError{
		domain: str(e.Send(a.selDomain)),
		code:   a.msgInt(e, a.selCode),
		desc:   str(e.Send(a.selLocalizedDesc)),
	}
	if info := e.Send(a.selUserInfo); info != 0 {
		key := a.msgFromUTF8(a.clsNSString, a.selStringWithUTF8, "NSUnderlyingError")
		if u := info.Send(a.selObjectForKey, key); u != 0 {
			out.under = fmt.Sprintf("%s (%s %d)", str(u.Send(a.selLocalizedDesc)), str(u.Send(a.selDomain)), a.msgInt(u, a.selCode))
		}
	}
	return out
}

// toRGBA 把 CGImage 缩放画进 sRGB、RGBA 预乘的位图，再拷成 image.RGBA。
// 位图内存由 CoreGraphics 自己分配，不把 Go 内存交给 C 长期持有。
func (a *avAPI) toRGBA(img uintptr, maxSize int) (image.Image, error) {
	sw, sh := int(a.cgImageGetWidth(img)), int(a.cgImageGetHeight(img))
	if sw <= 0 || sh <= 0 {
		return nil, fmt.Errorf("media: poster: empty frame %dx%d", sw, sh)
	}
	// generator 的 maximumSize 通常已经缩好；这里再按 maxSize 画一次，保证结果不超限
	w, h := fitSize(sw, sh, maxSize)
	cs := a.cgColorSpaceCreateName(a.kCGColorSpaceSRGB)
	if cs == 0 {
		return nil, errors.New("media: poster: cannot create sRGB color space")
	}
	defer a.cgColorSpaceRelease(cs)
	ctx := a.cgBitmapContextCreate(nil, uintptr(w), uintptr(h), 8, 0, cs, cgImageAlphaPremultipliedLast|cgBitmapByteOrder32Big)
	if ctx == 0 {
		return nil, fmt.Errorf("media: poster: cannot create %dx%d bitmap context", w, h)
	}
	defer a.cgContextRelease(ctx)
	a.cgContextSetInterp(ctx, cgInterpolationHigh)
	a.cgContextDrawImage(ctx, cgRect{W: float64(w), H: float64(h)}, img)

	data := a.cgBitmapContextData(ctx)
	stride := int(a.cgBitmapContextStride(ctx))
	if data == nil || stride < w*4 {
		return nil, fmt.Errorf("media: poster: bad bitmap context (stride %d)", stride)
	}
	src := unsafe.Slice((*byte)(data), stride*h)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	// 位图内存第一行就是画面最上面一行
	for y := range h {
		copy(out.Pix[y*out.Stride:y*out.Stride+w*4], src[y*stride:y*stride+w*4])
	}
	return out, nil
}

// isMatroska 看文件头是不是 EBML（WebM/MKV 共用的容器头）。
func isMatroska(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var b [4]byte
	if _, err := io.ReadFull(f, b[:]); err != nil {
		return false
	}
	return b == [4]byte{0x1A, 0x45, 0xDF, 0xA3}
}
