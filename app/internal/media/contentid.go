package media

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// 实况照片的图片那半（HEIC 或 JPEG）把内容 ID 存在 EXIF 的苹果 MakerNote 里（tag 0x0011），
// 视频那半存在 moov/meta 的 com.apple.quicktime.content.identifier；两边相同就是同一张实况照片。
//
// 没用 imagemeta 读：它 v1.1.0 把苹果 MakerNote 头当成 26 字节、偏移基准也跟着错了 26 字节，
// 实测 iPhone 照片的内容 ID 只能读出后 10 个字符。这里按 ExifTool 记录的格式自己解析：
// MakerNote 以 "Apple iOS\0" 开头，共 14 字节头（含 2 字节版本和 "MM" 字节序），
// 后面是一个 IFD，数据偏移从 MakerNote 起点算。

const (
	// HEIC 的 meta 盒子一般几 KB，分块很多的大图也就几十 KB。
	maxHEICMetaSize = 16 << 20
	// EXIF 项本身通常几 KB；JPEG 的 APP1 最多 64 KB。
	maxExifSize = 4 << 20
	// JPEG 在图像数据前最多看这么多个段。
	maxJPEGSegments = 1024
	// IFD 条目数上限，防止乱码数据让循环跑很久。
	maxIFDEntries = 4096
)

var appleMakerNotePrefix = []byte("Apple iOS\x00")

// ImageContentID 读实况照片图片那半的内容 ID。
// 不是 JPEG/HEIC、没有 EXIF、没有苹果 MakerNote 或 MakerNote 里没有内容 ID 时返回 "" 和 nil；
// EXIF 结构损坏返回错误（调用方记日志后可以当普通图片继续导入）。
func ImageContentID(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", nil
	}
	return imageContentID(f, st.Size())
}

// imageContentID 是 ImageContentID 的实际实现，单独拿出来方便测试直接喂字节。
func imageContentID(r io.ReaderAt, size int64) (string, error) {
	if size < 12 {
		return "", nil
	}
	tiff, err := findExifTIFF(r, size)
	if err != nil || tiff == nil {
		return "", err
	}
	return appleContentIDFromTIFF(tiff)
}

// findExifTIFF 按文件头分辨 JPEG 和 HEIC，返回 EXIF 的 TIFF 数据；没有 EXIF 时返回 nil。
func findExifTIFF(r io.ReaderAt, size int64) ([]byte, error) {
	head, err := readExact(r, 0, 12)
	if err != nil {
		return nil, err
	}
	switch {
	case head[0] == 0xFF && head[1] == 0xD8:
		return jpegExifTIFF(r, size)
	case string(head[4:8]) == "ftyp":
		return heicExifTIFF(r, size)
	}
	// 其他格式（PNG、WebP 等）苹果相机不会用来存实况照片。
	return nil, nil
}

// readExact 和 readFull 一样读满 n 字节，但错误不包 ErrUnsupported（那个是视频专用的）。
func readExact(r io.ReaderAt, off int64, n int) ([]byte, error) {
	buf := make([]byte, n)
	got, err := r.ReadAt(buf, off)
	if got == n {
		return buf, nil
	}
	if err == nil || err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("media: 图片在 %d 处提前结束", off+int64(got))
	}
	return nil, err
}

func exifErrorf(format string, args ...any) error {
	return fmt.Errorf("media: 图片 EXIF 损坏: "+format, args...)
}

// jpegExifTIFF 在 JPEG 的段里找 APP1 "Exif\0\0"，返回其后的 TIFF 数据。
func jpegExifTIFF(r io.ReaderAt, size int64) ([]byte, error) {
	off := int64(2)
	for i := 0; i < maxJPEGSegments; i++ {
		if size-off < 4 {
			return nil, nil
		}
		h, err := readExact(r, off, 4)
		if err != nil {
			return nil, err
		}
		if h[0] != 0xFF {
			return nil, exifErrorf("JPEG 段标记在 %d 处无效", off)
		}
		marker := h[1]
		switch {
		case marker == 0xFF:
			// 段之间的填充字节，往后挪一个。
			off++
			continue
		case marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7):
			// 没有长度字段的独立标记。
			off += 2
			continue
		case marker == 0xDA || marker == 0xD9:
			// 到了图像数据或文件尾，EXIF 只会出现在这之前。
			return nil, nil
		}
		segLen := int64(binary.BigEndian.Uint16(h[2:4]))
		if segLen < 2 || off+2+segLen > size {
			return nil, exifErrorf("JPEG 段长度 %d 越界", segLen)
		}
		if marker == 0xE1 && segLen >= 8 {
			d, err := readExact(r, off+4, int(segLen-2))
			if err != nil {
				return nil, err
			}
			if bytes.HasPrefix(d, []byte("Exif\x00\x00")) {
				return d[6:], nil
			}
		}
		off += 2 + segLen
	}
	return nil, nil
}

// heicExifTIFF 在 HEIC 的 meta 盒子里找类型为 Exif 的项，按 iloc 读出来。
// Exif 项开头 4 字节是到 TIFF 头的偏移（通常跳过 "Exif\0\0" 这 6 字节）。
func heicExifTIFF(r io.ReaderAt, size int64) ([]byte, error) {
	var meta []byte
	off := int64(0)
	for i := 0; meta == nil && off+8 <= size && i < 64; i++ {
		h, err := readExact(r, off, int(min(16, size-off)))
		if err != nil {
			return nil, err
		}
		boxSize := uint64(binary.BigEndian.Uint32(h[0:4]))
		hdr := int64(8)
		switch boxSize {
		case 1:
			if len(h) < 16 {
				return nil, exifErrorf("HEIC 盒子大小越界")
			}
			boxSize, hdr = binary.BigEndian.Uint64(h[8:16]), 16
		case 0:
			boxSize = uint64(size - off)
		}
		if boxSize < uint64(hdr) || boxSize > uint64(size-off) {
			return nil, exifErrorf("HEIC 盒子 %q 大小 %d 越界", h[4:8], boxSize)
		}
		if string(h[4:8]) == "meta" {
			if boxSize-uint64(hdr) > maxHEICMetaSize {
				return nil, exifErrorf("HEIC meta 大小 %d 超过上限", boxSize)
			}
			if meta, err = readExact(r, off+hdr, int(boxSize)-int(hdr)); err != nil {
				return nil, err
			}
		}
		off += int64(boxSize)
	}
	if len(meta) < 4 {
		// 没有 meta：不是 HEIF 图片（可能是 MP4 之类），没有 EXIF。
		return nil, nil
	}

	var exifID uint32
	var iloc, idat []byte
	// meta 是带 4 字节版本标志的完整盒子。
	err := eachBoxBytes(meta[4:], func(typ string, p []byte) error {
		switch typ {
		case "iinf":
			id, err := findExifItemID(p)
			if err != nil {
				return err
			}
			exifID = id
		case "iloc":
			iloc = p
		case "idat":
			idat = p
		}
		return nil
	})
	if err != nil {
		return nil, exifErrorf("%v", err)
	}
	if exifID == 0 || iloc == nil {
		return nil, nil
	}
	item, err := readILocItem(r, size, iloc, idat, exifID)
	if err != nil || item == nil {
		return nil, err
	}
	if len(item) < 4 {
		return nil, exifErrorf("HEIC Exif 项太短")
	}
	skip := int64(binary.BigEndian.Uint32(item[0:4]))
	if skip > int64(len(item)-4) {
		return nil, exifErrorf("HEIC Exif 头偏移 %d 越界", skip)
	}
	return item[4+skip:], nil
}

// findExifItemID 在 iinf 里找类型为 Exif 的项编号，找不到返回 0。
func findExifItemID(p []byte) (uint32, error) {
	if len(p) < 6 {
		return 0, exifErrorf("iinf 太短")
	}
	start := 6
	if p[0] != 0 {
		start = 8
	}
	if len(p) < start {
		return 0, exifErrorf("iinf 太短")
	}
	var id uint32
	err := eachBoxBytes(p[start:], func(typ string, e []byte) error {
		if typ != "infe" || id != 0 || len(e) < 4 {
			return nil
		}
		// 只有版本 2、3 的 infe 带四字符项类型；更老的版本不会用来存 Exif。
		switch e[0] {
		case 2:
			if len(e) >= 12 && string(e[8:12]) == "Exif" {
				id = uint32(binary.BigEndian.Uint16(e[4:6]))
			}
		case 3:
			if len(e) >= 14 && string(e[10:14]) == "Exif" {
				id = binary.BigEndian.Uint32(e[4:8])
			}
		}
		return nil
	})
	return id, err
}

// readILocItem 按 iloc 记录读出指定项的全部数据（多个片段拼起来）。
func readILocItem(r io.ReaderAt, size int64, p, idat []byte, want uint32) ([]byte, error) {
	if len(p) < 8 {
		return nil, exifErrorf("iloc 太短")
	}
	version := p[0]
	offSize, lenSize := int(p[4]>>4), int(p[4]&0xF)
	baseSize, idxSize := int(p[5]>>4), 0
	if version == 1 || version == 2 {
		idxSize = int(p[5] & 0xF)
	}
	for _, s := range []int{offSize, lenSize, baseSize, idxSize} {
		if s != 0 && s != 4 && s != 8 {
			return nil, exifErrorf("iloc 字段宽度 %d 无效", s)
		}
	}
	pos := 6
	var count uint32
	if version < 2 {
		if len(p) < pos+2 {
			return nil, exifErrorf("iloc 太短")
		}
		count = uint32(binary.BigEndian.Uint16(p[pos:]))
		pos += 2
	} else {
		if len(p) < pos+4 {
			return nil, exifErrorf("iloc 太短")
		}
		count = binary.BigEndian.Uint32(p[pos:])
		pos += 4
	}
	readN := func(n int) (uint64, error) {
		if len(p)-pos < n {
			return 0, exifErrorf("iloc 太短")
		}
		var v uint64
		switch n {
		case 2:
			v = uint64(binary.BigEndian.Uint16(p[pos:]))
		case 4:
			v = uint64(binary.BigEndian.Uint32(p[pos:]))
		case 8:
			v = binary.BigEndian.Uint64(p[pos:])
		}
		pos += n
		return v, nil
	}
	for i := uint32(0); i < count; i++ {
		idWidth := 2
		if version == 2 {
			idWidth = 4
		}
		id, err := readN(idWidth)
		if err != nil {
			return nil, err
		}
		method := uint64(0)
		if version == 1 || version == 2 {
			v, err := readN(2)
			if err != nil {
				return nil, err
			}
			method = v & 0xF
		}
		if _, err := readN(2); err != nil { // data_reference_index
			return nil, err
		}
		base, err := readN(baseSize)
		if err != nil {
			return nil, err
		}
		extents, err := readN(2)
		if err != nil {
			return nil, err
		}
		var out []byte
		for e := uint64(0); e < extents; e++ {
			if _, err := readN(idxSize); err != nil {
				return nil, err
			}
			eoff, err := readN(offSize)
			if err != nil {
				return nil, err
			}
			elen, err := readN(lenSize)
			if err != nil {
				return nil, err
			}
			if uint32(id) != want {
				continue
			}
			start := base + eoff
			if elen == 0 {
				// 长度 0 表示一直到数据源结尾。
				switch method {
				case 0:
					if start > uint64(size) {
						return nil, exifErrorf("Exif 项位置越界")
					}
					elen = uint64(size) - start
				case 1:
					if start > uint64(len(idat)) {
						return nil, exifErrorf("Exif 项位置越界")
					}
					elen = uint64(len(idat)) - start
				}
			}
			if elen > maxExifSize || uint64(len(out))+elen > maxExifSize {
				return nil, exifErrorf("Exif 项大小超过上限")
			}
			switch method {
			case 0:
				if start > uint64(size) || elen > uint64(size)-start {
					return nil, exifErrorf("Exif 项位置越界")
				}
				d, err := readExact(r, int64(start), int(elen))
				if err != nil {
					return nil, err
				}
				out = append(out, d...)
			case 1:
				if start > uint64(len(idat)) || elen > uint64(len(idat))-start {
					return nil, exifErrorf("Exif 项在 idat 里越界")
				}
				out = append(out, idat[start:start+elen]...)
			default:
				return nil, exifErrorf("Exif 项的存放方式 %d 不支持", method)
			}
		}
		if uint32(id) == want {
			return out, nil
		}
	}
	return nil, nil
}

// tiffEntry 是 IFD 里的一个条目。
type tiffEntry struct {
	tag, typ uint16
	count    uint32
	value    []byte // 原始 4 字节值或偏移
}

// readIFD 读 d 里 off 处的 IFD 条目表。
func readIFD(d []byte, off uint32, order binary.ByteOrder) ([]tiffEntry, error) {
	if uint64(off)+2 > uint64(len(d)) {
		return nil, exifErrorf("IFD 偏移 %d 越界", off)
	}
	n := int(order.Uint16(d[off:]))
	if n > maxIFDEntries || int(off)+2+n*12 > len(d) {
		return nil, exifErrorf("IFD 条目数 %d 越界", n)
	}
	out := make([]tiffEntry, n)
	for i := range out {
		e := d[int(off)+2+i*12:]
		out[i] = tiffEntry{tag: order.Uint16(e[0:2]), typ: order.Uint16(e[2:4]), count: order.Uint32(e[4:8]), value: e[8:12]}
	}
	return out, nil
}

// entryBytes 取条目的数据：不超过 4 字节时就在条目里，否则按偏移从 base 里取。
func entryBytes(base []byte, e tiffEntry, order binary.ByteOrder, unit uint32) ([]byte, error) {
	n := uint64(e.count) * uint64(unit)
	if n <= 4 {
		return e.value[:n], nil
	}
	off := uint64(order.Uint32(e.value))
	if off > uint64(len(base)) || n > uint64(len(base))-off {
		return nil, exifErrorf("tag %#04x 的数据越界", e.tag)
	}
	return base[off : off+n], nil
}

func tiffByteOrder(b []byte) (binary.ByteOrder, bool) {
	switch string(b) {
	case "MM":
		return binary.BigEndian, true
	case "II":
		return binary.LittleEndian, true
	}
	return nil, false
}

// appleContentIDFromTIFF 沿 IFD0 → Exif IFD → MakerNote 找苹果内容 ID。
func appleContentIDFromTIFF(tiff []byte) (string, error) {
	if len(tiff) < 8 {
		return "", exifErrorf("TIFF 头太短")
	}
	order, ok := tiffByteOrder(tiff[0:2])
	if !ok || order.Uint16(tiff[2:4]) != 42 {
		return "", exifErrorf("TIFF 头无效")
	}
	ifd0, err := readIFD(tiff, order.Uint32(tiff[4:8]), order)
	if err != nil {
		return "", err
	}
	var exifIFD uint32
	for _, e := range ifd0 {
		if e.tag == 0x8769 {
			exifIFD = order.Uint32(e.value)
		}
	}
	if exifIFD == 0 {
		return "", nil
	}
	exif, err := readIFD(tiff, exifIFD, order)
	if err != nil {
		return "", err
	}
	for _, e := range exif {
		if e.tag != 0x927C {
			continue
		}
		mn, err := entryBytes(tiff, e, order, 1)
		if err != nil {
			return "", err
		}
		return appleMakerNoteContentID(mn)
	}
	return "", nil
}

// appleMakerNoteContentID 读苹果 MakerNote 里 tag 0x0011 的字符串。不是苹果的 MakerNote 返回 ""。
func appleMakerNoteContentID(mn []byte) (string, error) {
	if !bytes.HasPrefix(mn, appleMakerNotePrefix) {
		return "", nil
	}
	if len(mn) < 16 {
		return "", exifErrorf("苹果 MakerNote 太短")
	}
	order, ok := tiffByteOrder(mn[12:14])
	if !ok {
		return "", exifErrorf("苹果 MakerNote 字节序无效")
	}
	entries, err := readIFD(mn, 14, order)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.tag != 0x0011 {
			continue
		}
		// 苹果写成 ASCII（类型 2），个别工具改写后可能变成 UNDEFINED（类型 7），都按字节读。
		if e.typ != 2 && e.typ != 7 {
			return "", exifErrorf("内容 ID 的类型 %d 不对", e.typ)
		}
		v, err := entryBytes(mn, e, order, 1)
		if err != nil {
			return "", err
		}
		if i := bytes.IndexByte(v, 0); i >= 0 {
			v = v[:i]
		}
		return strings.TrimSpace(string(v)), nil
	}
	return "", nil
}
