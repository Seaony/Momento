package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// 测试数据按字节手工拼出来，没有放真实照片（实况照片带拍摄者隐私，也很大）。
// 格式依据：ExifTool 的 Apple MakerNote 说明，并对照过一张 iPhone 拍的 HEIC：
// MakerNote 以 "Apple iOS\0" + 00 01 + "MM" 开头（14 字节），后接 IFD，
// tag 0x0011 是 ASCII，数据偏移从 MakerNote 起点算。

const testCID = "FD0B0708-C73F-4F6D-B57B-4169BB991D46"

// appleMakerNote 拼一个苹果 MakerNote。cid 为空时不写 0x0011。
func appleMakerNote(cid string) []byte {
	type ent struct {
		tag, typ uint16
		count    uint32
		data     []byte
	}
	ents := []ent{{0x0001, 9, 1, []byte{0, 0, 0, 14}}}
	if cid != "" {
		ents = append(ents, ent{0x0011, 2, uint32(len(cid) + 1), append([]byte(cid), 0)})
	}
	ents = append(ents, ent{0x0014, 9, 1, []byte{0, 0, 0, 12}})
	out := append([]byte("Apple iOS\x00"), 0, 1, 'M', 'M')
	out = binary.BigEndian.AppendUint16(out, uint16(len(ents)))
	dataOff := len(out) + len(ents)*12 + 4
	var data []byte
	for _, e := range ents {
		out = binary.BigEndian.AppendUint16(out, e.tag)
		out = binary.BigEndian.AppendUint16(out, e.typ)
		out = binary.BigEndian.AppendUint32(out, e.count)
		if len(e.data) <= 4 {
			out = append(out, append(e.data, make([]byte, 4-len(e.data))...)...)
			continue
		}
		out = binary.BigEndian.AppendUint32(out, uint32(dataOff+len(data)))
		data = append(data, e.data...)
	}
	out = append(out, 0, 0, 0, 0)
	return append(out, data...)
}

// tiffWithMakerNote 拼 TIFF：IFD0（Make、Exif 指针）→ Exif IFD（MakerNote）。makerNote 为 nil 时 Exif IFD 里没有 MakerNote。
func tiffWithMakerNote(order binary.AppendByteOrder, makerNote []byte) []byte {
	out := []byte("MM")
	if order.String() == binary.LittleEndian.String() {
		out = []byte("II")
	}
	out = order.AppendUint16(out, 42)
	out = order.AppendUint32(out, 8)
	// IFD0 在 8：2 个条目，结束于 38；"Apple\0" 在 38～44；Exif IFD 在 44。
	out = order.AppendUint16(out, 2)
	out = order.AppendUint16(out, 0x010F)
	out = order.AppendUint16(out, 2)
	out = order.AppendUint32(out, 6)
	out = order.AppendUint32(out, 38)
	out = order.AppendUint16(out, 0x8769)
	out = order.AppendUint16(out, 4)
	out = order.AppendUint32(out, 1)
	out = order.AppendUint32(out, 44)
	out = order.AppendUint32(out, 0)
	out = append(out, "Apple\x00"...)
	if makerNote == nil {
		out = order.AppendUint16(out, 1)
		out = order.AppendUint16(out, 0x9003) // DateTimeOriginal，随便一个别的 tag
		out = order.AppendUint16(out, 2)
		out = order.AppendUint32(out, 1)
		out = order.AppendUint32(out, 0)
		return order.AppendUint32(out, 0)
	}
	out = order.AppendUint16(out, 1)
	out = order.AppendUint16(out, 0x927C)
	out = order.AppendUint16(out, 7)
	out = order.AppendUint32(out, uint32(len(makerNote)))
	out = order.AppendUint32(out, 62)
	out = order.AppendUint32(out, 0)
	return append(out, makerNote...)
}

func jpegWithExif(tiff []byte) []byte {
	out := []byte{0xFF, 0xD8}
	// 先放一个 APP0 JFIF，确认会跳过别的段。
	out = append(out, 0xFF, 0xE0, 0, 16)
	out = append(out, "JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00"...)
	if tiff != nil {
		app1 := append([]byte("Exif\x00\x00"), tiff...)
		out = append(out, 0xFF, 0xE1)
		out = binary.BigEndian.AppendUint16(out, uint16(len(app1)+2))
		out = append(out, app1...)
	}
	out = append(out, 0xFF, 0xDA, 0, 2)
	out = append(out, 1, 2, 3, 4)
	return append(out, 0xFF, 0xD9)
}

// heicWithExif 拼最小的 HEIC：ftyp + meta(hdlr, iinf, iloc[, idat]) + mdat。
// useIdat 为 true 时 Exif 项放在 meta 的 idat 里（iloc 版本 1、存放方式 1），否则放在 mdat（iloc 版本 0）。
func heicWithExif(tiff []byte, useIdat bool) []byte {
	item := binary.BigEndian.AppendUint32(nil, 6)
	item = append(item, "Exif\x00\x00"...)
	item = append(item, tiff...)
	infe := func(id uint16, typ string) []byte {
		b := u32s(2 << 24)
		b = binary.BigEndian.AppendUint16(b, id)
		b = binary.BigEndian.AppendUint16(b, 0)
		b = append(b, typ...)
		return box("infe", append(b, 0))
	}
	iinf := box("iinf", u32s(0), []byte{0, 2}, infe(1, "hvc1"), infe(2, "Exif"))
	hdlr := box("hdlr", u32s(0, 0), []byte("pict"), make([]byte, 13))
	ftyp := box("ftyp", []byte("heic"), u32s(0), []byte("mif1heic"))
	build := func(offset uint32) []byte {
		var iloc, extra []byte
		if useIdat {
			iloc = append(u32s(1<<24), 0x44, 0x00, 0, 1, 0, 2, 0, 1, 0, 0, 0, 1)
			iloc = append(iloc, u32s(0, uint32(len(item)))...)
			extra = box("idat", item)
		} else {
			iloc = append(u32s(0), 0x44, 0x00, 0, 1, 0, 2, 0, 0, 0, 1)
			iloc = append(iloc, u32s(offset, uint32(len(item)))...)
		}
		meta := box("meta", u32s(0), hdlr, iinf, box("iloc", iloc), extra)
		out := append(bytes.Clone(ftyp), meta...)
		return append(out, box("mdat", item)...)
	}
	probe := build(0)
	return build(uint32(len(probe) - len(item)))
}

func contentIDBytes(d []byte) (string, error) {
	return imageContentID(bytes.NewReader(d), int64(len(d)))
}

func TestImageContentID(t *testing.T) {
	mn := appleMakerNote(testCID)
	cases := map[string]struct {
		data []byte
		want string
	}{
		"JPEG 大端":               {jpegWithExif(tiffWithMakerNote(binary.BigEndian, mn)), testCID},
		"JPEG 小端 TIFF":          {jpegWithExif(tiffWithMakerNote(binary.LittleEndian, mn)), testCID},
		"HEIC mdat 里的 Exif":     {heicWithExif(tiffWithMakerNote(binary.BigEndian, mn), false), testCID},
		"HEIC idat 里的 Exif":     {heicWithExif(tiffWithMakerNote(binary.BigEndian, mn), true), testCID},
		"JPEG 没有 EXIF":          {jpegWithExif(nil), ""},
		"没有 MakerNote":          {jpegWithExif(tiffWithMakerNote(binary.BigEndian, nil)), ""},
		"别家相机的 MakerNote":       {jpegWithExif(tiffWithMakerNote(binary.BigEndian, []byte("Nikon\x00\x02\x10\x00\x00MM\x00*\x00\x00\x00\x08\x00\x00"))), ""},
		"苹果 MakerNote 没有内容 ID":  {jpegWithExif(tiffWithMakerNote(binary.BigEndian, appleMakerNote(""))), ""},
		"PNG":                   {append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...), ""},
		"MP4 视频（有 ftyp 没 meta）": {readSample(t, "probe_h264.mp4"), ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := contentIDBytes(c.data)
			if err != nil || got != c.want {
				t.Fatalf("得到 %q, %v；want %q", got, err, c.want)
			}
		})
	}
}

// 结构损坏要返回错误，不能 panic，也不能读出半截 ID。
func TestImageContentIDCorrupt(t *testing.T) {
	mn := appleMakerNote(testCID)
	badCount := bytes.Clone(mn)
	binary.BigEndian.PutUint16(badCount[14:], 0xFFFF)
	badOffset := bytes.Clone(mn)
	// 第二个条目（0x0011）的偏移改到 MakerNote 外面。
	binary.BigEndian.PutUint32(badOffset[14+2+12+8:], 0xFFFFFF00)
	cases := map[string][]byte{
		"MakerNote 条目数越界": jpegWithExif(tiffWithMakerNote(binary.BigEndian, badCount)),
		"内容 ID 偏移越界":      jpegWithExif(tiffWithMakerNote(binary.BigEndian, badOffset)),
		"JPEG 段长度越界":      {0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xF0, 'E', 'x', 'i', 'f', 0, 0, 0, 0},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			if got, err := contentIDBytes(d); err == nil {
				t.Fatalf("应返回错误，得到 %q", got)
			}
		})
	}
	// 各种截断：只要求不 panic，读出来的要么是完整 ID 要么是空。
	for _, d := range [][]byte{
		jpegWithExif(tiffWithMakerNote(binary.BigEndian, mn)),
		heicWithExif(tiffWithMakerNote(binary.BigEndian, mn), false),
		heicWithExif(tiffWithMakerNote(binary.BigEndian, mn), true),
	} {
		for cut := 0; cut < len(d); cut++ {
			if got, _ := contentIDBytes(d[:cut]); got != "" && got != testCID {
				t.Fatalf("截到 %d 读出半截 ID %q", cut, got)
			}
		}
	}
}

func TestImageContentIDFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "live.jpg")
	if err := os.WriteFile(p, jpegWithExif(tiffWithMakerNote(binary.BigEndian, appleMakerNote(testCID))), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := ImageContentID(p); err != nil || got != testCID {
		t.Fatalf("得到 %q, %v", got, err)
	}
	if _, err := ImageContentID(filepath.Join(dir, "missing.heic")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("文件不存在应返回 IO 错误，得到 %v", err)
	}
	if got, err := ImageContentID(dir); err != nil || got != "" {
		t.Fatalf("文件夹应返回空，得到 %q, %v", got, err)
	}
}

// 用真实实况照片核对（不放进仓库）：
// MOMENTO_LIVE_PHOTO_IMAGE=照片路径 MOMENTO_LIVE_PHOTO_CID=期望的内容 ID go test ./internal/media -run ContentIDReal
func TestImageContentIDRealPhoto(t *testing.T) {
	path, want := os.Getenv("MOMENTO_LIVE_PHOTO_IMAGE"), os.Getenv("MOMENTO_LIVE_PHOTO_CID")
	if path == "" || want == "" {
		t.Skip("没设置 MOMENTO_LIVE_PHOTO_IMAGE / MOMENTO_LIVE_PHOTO_CID")
	}
	if got, err := ImageContentID(path); err != nil || got != want {
		t.Fatalf("得到 %q, %v；want %q", got, err, want)
	}
}

func FuzzImageContentID(f *testing.F) {
	mn := appleMakerNote(testCID)
	f.Add(jpegWithExif(tiffWithMakerNote(binary.BigEndian, mn)))
	f.Add(jpegWithExif(tiffWithMakerNote(binary.LittleEndian, mn)))
	f.Add(heicWithExif(tiffWithMakerNote(binary.BigEndian, mn), false))
	f.Add(heicWithExif(tiffWithMakerNote(binary.BigEndian, mn), true))
	f.Fuzz(func(t *testing.T, d []byte) {
		// 只要求不 panic、不卡住；损坏数据返回错误是正常的。
		_, _ = contentIDBytes(d)
	})
}
