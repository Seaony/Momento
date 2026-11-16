package media

// rawExtensions 是相机 RAW 格式（不带点、小写）。
var rawExtensions = map[string]bool{
	"dng": true, "cr2": true, "cr3": true, "crw": true, "nef": true, "nrw": true, "arw": true, "srf": true, "sr2": true,
	"raf": true, "orf": true, "rw2": true, "rwl": true, "pef": true, "srw": true, "x3f": true, "3fr": true, "iiq": true,
	"erf": true, "mrw": true, "raw": true,
}

// otherSystemImages 是除 RAW 外纯 Go 解不了、交给系统解码的图片格式。
var otherSystemImages = map[string]bool{
	"psd": true, "ico": true, "icns": true, "jp2": true, "j2k": true, "jpf": true, "jpx": true, "tga": true, "exr": true, "hdr": true,
}

// IsRawExtension 判断扩展名（带不带点都行）是不是相机 RAW。
func IsRawExtension(ext string) bool { return rawExtensions[lowerTrimDot(ext)] }

// IsSystemImageExtension 判断这种图片是否只能交给系统解码：相机 RAW、PSD、ICO、ICNS、JPEG 2000、TGA、EXR、HDR。
func IsSystemImageExtension(ext string) bool {
	ext = lowerTrimDot(ext)
	return rawExtensions[ext] || otherSystemImages[ext]
}
