package media

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/bits"
	"strings"
	"time"
)

// MP4 / MOV / M4V 都是 ISO BMFF（QuickTime）盒子结构：每个盒子是「4 字节大小 + 4 字节类型 + 内容」。
// 这里只沿着 moov → trak → mdia → minf → stbl 这条路读需要的小盒子，mdat（真正的音视频数据）整个跳过。

const (
	// 一次 Probe 最多看多少个盒子头。正常文件远远用不到，防止恶意文件塞几百万个小盒子拖慢导入。
	maxBMFFBoxVisits = 1 << 20
	// 碎片化 MP4 每个 moof 一般几百字节到几百 KB；单个超过这个大小就当文件坏了。
	maxMoofSize = 16 << 20
	// 所有 moof 加起来最多读这么多，超过就不再从碎片累加时长（时长给 0，表示读不出）。
	maxMoofTotal = 64 << 20
	// keys 盒子只存几十个元数据键名，正常几百字节。
	maxKeysSize = 1 << 20
	// ilst 里单个元数据项（我们只读内容 ID 那一项）的上限。
	maxIlstItemSize = 64 << 10
	// stts 一次读一块，避免长视频的大表一次性分配。
	sttsChunk = 64 << 10
)

// isBMFFTopLevel 判断文件开头第一个盒子的类型像不像 MP4/MOV。
// 新文件都以 ftyp 开头；老 QuickTime 文件可能直接是 moov、mdat、wide、free、skip 或 pnot。
func isBMFFTopLevel(typ string) bool {
	switch typ {
	case "ftyp", "moov", "mdat", "wide", "free", "skip", "pnot":
		return true
	}
	return false
}

// bmffBox 是一个盒子在文件里的位置。
type bmffBox struct {
	typ       string
	start     int64 // 盒子起点（含头）
	dataStart int64 // 内容起点
	end       int64 // 盒子终点（不含）
}

func (b bmffBox) dataSize() int64 { return b.end - b.dataStart }

// errBoxOverflow 表示盒子声称的大小超出了它的父盒子（或文件尾）。顶层遇到时可能只是文件被截断。
var errBoxOverflow = errors.New("box overflows parent")

type bmffReader struct {
	r      io.ReaderAt
	size   int64
	visits int
}

// readBoxHeader 读 off 处的盒子头。size=1 表示后面跟 64 位大小；size=0 表示一直到父盒子结尾。
// 盒子超出父盒子时返回同时包着 ErrUnsupported 和 errBoxOverflow 的错误。
func (p *bmffReader) readBoxHeader(off, parentEnd int64) (bmffBox, error) {
	p.visits++
	if p.visits > maxBMFFBoxVisits {
		return bmffBox{}, unsupportedf("盒子数量超过 %d 个", maxBMFFBoxVisits)
	}
	if parentEnd-off < 8 {
		return bmffBox{}, unsupportedf("盒子头在 %d 处越界", off)
	}
	h, err := readFull(p.r, off, 8)
	if err != nil {
		return bmffBox{}, err
	}
	size := uint64(binary.BigEndian.Uint32(h[0:4]))
	b := bmffBox{typ: string(h[4:8]), start: off, dataStart: off + 8}
	switch size {
	case 1:
		if parentEnd-off < 16 {
			return bmffBox{}, unsupportedf("%q 的 64 位大小越界", b.typ)
		}
		ext, err := readFull(p.r, off+8, 8)
		if err != nil {
			return bmffBox{}, err
		}
		size = binary.BigEndian.Uint64(ext)
		b.dataStart = off + 16
	case 0:
		size = uint64(parentEnd - off)
	}
	if size < uint64(b.dataStart-off) {
		return bmffBox{}, unsupportedf("%q 的大小 %d 比盒子头还小", b.typ, size)
	}
	if b.typ == "uuid" {
		// uuid 盒子头后面还有 16 字节扩展类型，内容从那之后开始。
		b.dataStart += 16
		if size < uint64(b.dataStart-off) {
			return bmffBox{}, unsupportedf("uuid 盒子太小")
		}
	}
	if size > uint64(parentEnd-off) {
		return bmffBox{}, unsupportedf("%q 大小 %d: %w", b.typ, size, errBoxOverflow)
	}
	b.end = off + int64(size)
	return b, nil
}

// eachChild 依次读 [start, end) 里的盒子头交给 fn。fn 返回 errStopIter 时正常结束。
func (p *bmffReader) eachChild(start, end int64, fn func(b bmffBox) error) error {
	for off := start; off < end; {
		if end-off < 8 {
			// 末尾不足一个盒子头的几字节：有些编码器会留零填充，当作结束。
			return nil
		}
		b, err := p.readBoxHeader(off, end)
		if err != nil {
			return err
		}
		if err := fn(b); err != nil {
			if errors.Is(err, errStopIter) {
				return nil
			}
			return err
		}
		off = b.end
	}
	return nil
}

var errStopIter = errors.New("stop iteration")

// readPayload 读盒子内容的前 n 字节（盒子更小时读整个盒子）。
func (p *bmffReader) readPayload(b bmffBox, n int64) ([]byte, error) {
	if ds := b.dataSize(); ds < n {
		n = ds
	}
	return readFull(p.r, b.dataStart, int(n))
}

// mp4Track 是一条轨道里读到的信息。
type mp4Track struct {
	id               uint32
	handler          string
	tkhdW, tkhdH     int
	rotation         int
	timescale        uint32
	mdhdDuration     uint64
	codec            string
	sampleW, sampleH int
	sttsSamples      uint64
	sttsDuration     uint64

	// 碎片化 MP4：从各个 moof 累加的样本数、时长和最后时间点。
	fragSamples  uint64
	fragDuration uint64
	fragCursor   uint64
	fragEnd      uint64
}

type mp4Movie struct {
	timescale        uint32
	duration         uint64
	durationUnknown  bool
	fragmented       bool
	fragmentDuration uint64
	trexDuration     map[uint32]uint32
	tracks           []*mp4Track
	contentID        string

	moofBytes     int64
	fragOverLimit bool
}

func probeBMFF(r io.ReaderAt, size int64) (Info, error) {
	p := &bmffReader{r: r, size: size}
	var mov *mp4Movie
	first := true
	err := p.eachChild(0, size, func(b bmffBox) error {
		if first {
			first = false
			if !isBMFFTopLevel(b.typ) {
				return unsupportedf("第一个盒子是 %q", b.typ)
			}
		}
		switch b.typ {
		case "moov":
			if mov != nil {
				return unsupportedf("有多个 moov")
			}
			m, err := p.parseMoov(b)
			if err != nil {
				return err
			}
			mov = m
			if !mov.fragmented {
				// 普通 MP4 的信息全在 moov 里，后面的盒子不用再看。
				return errStopIter
			}
		case "moof":
			if mov == nil || !mov.fragmented || mov.fragOverLimit {
				return nil
			}
			if b.dataSize() > maxMoofSize || mov.moofBytes+b.dataSize() > maxMoofTotal {
				mov.fragOverLimit = true
				return nil
			}
			mov.moofBytes += b.dataSize()
			data, err := p.readPayload(b, b.dataSize())
			if err != nil {
				return err
			}
			if err := mov.parseMoof(data); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		// 顶层盒子超出文件尾：多半是没下载完或被截断的文件。
		// moov 已经完整读到时信息是可靠的，照常返回；否则当作读不出。
		if !(errors.Is(err, errBoxOverflow) && mov != nil) {
			return Info{}, err
		}
	}
	if mov == nil {
		return Info{}, unsupportedf("没有 moov 盒子")
	}
	return mov.info()
}

// parseMoov 读 moov 里的 mvhd、trak、mvex 和元数据。
func (p *bmffReader) parseMoov(moov bmffBox) (*mp4Movie, error) {
	mov := &mp4Movie{trexDuration: map[uint32]uint32{}}
	err := p.eachChild(moov.dataStart, moov.end, func(b bmffBox) error {
		switch b.typ {
		case "mvhd":
			return p.parseMvhd(b, mov)
		case "trak":
			t, err := p.parseTrak(b)
			if err != nil {
				return err
			}
			mov.tracks = append(mov.tracks, t)
		case "mvex":
			mov.fragmented = true
			return p.parseMvex(b, mov)
		case "meta":
			return p.parseQuickTimeMeta(b, mov)
		case "udta":
			// 有些写法把 meta 放在 udta 里。
			return p.eachChild(b.dataStart, b.end, func(c bmffBox) error {
				if c.typ == "meta" {
					return p.parseQuickTimeMeta(c, mov)
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mov, nil
}

func (p *bmffReader) parseMvhd(b bmffBox, mov *mp4Movie) error {
	d, err := p.readPayload(b, 32)
	if err != nil {
		return err
	}
	if len(d) < 4 {
		return unsupportedf("mvhd 太短")
	}
	if d[0] == 1 {
		if len(d) < 32 {
			return unsupportedf("mvhd 太短")
		}
		mov.timescale = binary.BigEndian.Uint32(d[20:24])
		mov.duration = binary.BigEndian.Uint64(d[24:32])
		mov.durationUnknown = mov.duration == math.MaxUint64
	} else {
		if len(d) < 20 {
			return unsupportedf("mvhd 太短")
		}
		mov.timescale = binary.BigEndian.Uint32(d[12:16])
		mov.duration = uint64(binary.BigEndian.Uint32(d[16:20]))
		mov.durationUnknown = mov.duration == math.MaxUint32
	}
	return nil
}

func (p *bmffReader) parseMvex(b bmffBox, mov *mp4Movie) error {
	return p.eachChild(b.dataStart, b.end, func(c bmffBox) error {
		switch c.typ {
		case "mehd":
			d, err := p.readPayload(c, 12)
			if err != nil {
				return err
			}
			if len(d) >= 12 && d[0] == 1 {
				mov.fragmentDuration = binary.BigEndian.Uint64(d[4:12])
			} else if len(d) >= 8 {
				mov.fragmentDuration = uint64(binary.BigEndian.Uint32(d[4:8]))
			}
		case "trex":
			d, err := p.readPayload(c, 16)
			if err != nil {
				return err
			}
			if len(d) >= 16 {
				mov.trexDuration[binary.BigEndian.Uint32(d[4:8])] = binary.BigEndian.Uint32(d[12:16])
			}
		}
		return nil
	})
}

func (p *bmffReader) parseTrak(trak bmffBox) (*mp4Track, error) {
	t := &mp4Track{}
	err := p.eachChild(trak.dataStart, trak.end, func(b bmffBox) error {
		switch b.typ {
		case "tkhd":
			return p.parseTkhd(b, t)
		case "mdia":
			return p.eachChild(b.dataStart, b.end, func(c bmffBox) error {
				switch c.typ {
				case "mdhd":
					return p.parseMdhd(c, t)
				case "hdlr":
					d, err := p.readPayload(c, 12)
					if err != nil {
						return err
					}
					if len(d) >= 12 {
						t.handler = string(d[8:12])
					}
				case "minf":
					return p.eachChild(c.dataStart, c.end, func(m bmffBox) error {
						if m.typ != "stbl" {
							return nil
						}
						return p.eachChild(m.dataStart, m.end, func(s bmffBox) error {
							switch s.typ {
							case "stsd":
								return p.parseStsd(s, t)
							case "stts":
								return p.parseStts(s, t)
							}
							return nil
						})
					})
				}
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return t, nil
}

// parseTkhd 读轨道号、宽高（16.16 定点）和变换矩阵。
func (p *bmffReader) parseTkhd(b bmffBox, t *mp4Track) error {
	d, err := p.readPayload(b, 96)
	if err != nil {
		return err
	}
	if len(d) < 4 {
		return unsupportedf("tkhd 太短")
	}
	idOff, matrixOff := 12, 40
	if d[0] == 1 {
		idOff, matrixOff = 20, 52
	}
	if len(d) < matrixOff+36+8 {
		return unsupportedf("tkhd 太短")
	}
	t.id = binary.BigEndian.Uint32(d[idOff : idOff+4])
	m := d[matrixOff : matrixOff+36]
	a := int32(binary.BigEndian.Uint32(m[0:4]))
	bb := int32(binary.BigEndian.Uint32(m[4:8]))
	t.rotation = matrixRotation(a, bb)
	t.tkhdW = int(binary.BigEndian.Uint32(d[matrixOff+36:matrixOff+40]) >> 16)
	t.tkhdH = int(binary.BigEndian.Uint32(d[matrixOff+40:matrixOff+44]) >> 16)
	return nil
}

// matrixRotation 从变换矩阵前两项算出播放时顺时针要转的角度。
// QuickTime 矩阵把点 (x, y) 变成 (a·x + c·y, b·x + d·y)，屏幕 y 轴朝下，
// 所以 (a, b) 是原图向右方向转过去的方向：(0, 1) 朝下，即顺时针 90 度（iPhone 竖拍就是这样）。
// 镜像之类的矩阵只取最接近的 90 度倍数。
func matrixRotation(a, b int32) int {
	if a == 0 && b == 0 {
		return 0
	}
	deg := math.Atan2(float64(b), float64(a)) * 180 / math.Pi
	r := int(math.Round(deg/90)) * 90
	return ((r % 360) + 360) % 360
}

func (p *bmffReader) parseMdhd(b bmffBox, t *mp4Track) error {
	d, err := p.readPayload(b, 32)
	if err != nil {
		return err
	}
	if len(d) >= 32 && d[0] == 1 {
		t.timescale = binary.BigEndian.Uint32(d[20:24])
		t.mdhdDuration = binary.BigEndian.Uint64(d[24:32])
		if t.mdhdDuration == math.MaxUint64 {
			t.mdhdDuration = 0
		}
	} else if len(d) >= 20 && d[0] == 0 {
		t.timescale = binary.BigEndian.Uint32(d[12:16])
		t.mdhdDuration = uint64(binary.BigEndian.Uint32(d[16:20]))
		if t.mdhdDuration == math.MaxUint32 {
			t.mdhdDuration = 0
		}
	} else {
		return unsupportedf("mdhd 太短或版本不认识")
	}
	return nil
}

// parseStsd 读第一个样本条目的类型（编码）和视觉条目里的宽高。
func (p *bmffReader) parseStsd(b bmffBox, t *mp4Track) error {
	d, err := p.readPayload(b, 8+36)
	if err != nil {
		return err
	}
	if len(d) < 16 || binary.BigEndian.Uint32(d[4:8]) == 0 {
		// 没有样本条目：轨道不带编码描述，编码和宽高留空。
		return nil
	}
	entry := d[8:]
	if entrySize := binary.BigEndian.Uint32(entry[0:4]); int64(entrySize) > b.dataSize()-8 || entrySize < 8 {
		return unsupportedf("stsd 样本条目大小 %d 越界", entrySize)
	}
	t.codec = cleanFourCC(entry[4:8])
	if len(entry) >= 36 {
		t.sampleW = int(binary.BigEndian.Uint16(entry[32:34]))
		t.sampleH = int(binary.BigEndian.Uint16(entry[34:36]))
	}
	return nil
}

// cleanFourCC 去掉四字符码尾部的空格和不可见字符。
func cleanFourCC(b []byte) string {
	s := strings.TrimRight(string(b), " \x00")
	for _, c := range []byte(s) {
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return s
}

// parseStts 累加「样本数 × 每个样本时长」，得到总样本数和总时长，用来估算帧率。
// 表可能很大（长视频几十万条），分块读，不一次分配。
func (p *bmffReader) parseStts(b bmffBox, t *mp4Track) error {
	d, err := p.readPayload(b, 8)
	if err != nil {
		return err
	}
	if len(d) < 8 {
		return unsupportedf("stts 太短")
	}
	count := int64(binary.BigEndian.Uint32(d[4:8]))
	if count*8 > b.dataSize()-8 {
		return unsupportedf("stts 条目数 %d 越界", count)
	}
	off := b.dataStart + 8
	for remaining := count * 8; remaining > 0; {
		n := min(remaining, sttsChunk)
		chunk, err := readFull(p.r, off, int(n))
		if err != nil {
			return err
		}
		for i := 0; i+8 <= len(chunk); i += 8 {
			c := uint64(binary.BigEndian.Uint32(chunk[i : i+4]))
			delta := uint64(binary.BigEndian.Uint32(chunk[i+4 : i+8]))
			t.sttsSamples += c
			hi, lo := bits.Mul64(c, delta)
			sum, carry := bits.Add64(t.sttsDuration, lo, 0)
			if hi != 0 || carry != 0 {
				return unsupportedf("stts 时长溢出")
			}
			t.sttsDuration = sum
		}
		off += n
		remaining -= n
	}
	return nil
}

// parseQuickTimeMeta 在 meta（hdlr=mdta）里找 com.apple.quicktime.content.identifier。
// QuickTime 的 meta 没有 4 字节版本标志，ISO 的有；看内容开头是不是子盒子类型来区分。
func (p *bmffReader) parseQuickTimeMeta(b bmffBox, mov *mp4Movie) error {
	if mov.contentID != "" {
		return nil
	}
	start := b.dataStart
	peek, err := p.readPayload(b, 12)
	if err != nil {
		return err
	}
	if len(peek) >= 12 && !isMetaChild(string(peek[4:8])) && isMetaChild(string(peek[8:12])) {
		start += 4
	}
	var keyIndex uint32
	var ilst *bmffBox
	err = p.eachChild(start, b.end, func(c bmffBox) error {
		switch c.typ {
		case "keys":
			if c.dataSize() > maxKeysSize {
				// 键表大到不正常，不像苹果写的元数据；不找内容 ID。
				return nil
			}
			d, err := p.readPayload(c, c.dataSize())
			if err != nil {
				return err
			}
			keyIndex = findContentIDKey(d)
		case "ilst":
			cc := c
			ilst = &cc
		}
		return nil
	})
	if err != nil {
		return err
	}
	if keyIndex == 0 || ilst == nil {
		return nil
	}
	return p.eachChild(ilst.dataStart, ilst.end, func(item bmffBox) error {
		if binary.BigEndian.Uint32([]byte(item.typ)) != keyIndex {
			return nil
		}
		if item.dataSize() > maxIlstItemSize {
			return unsupportedf("内容 ID 元数据项大小 %d 不正常", item.dataSize())
		}
		d, err := p.readPayload(item, item.dataSize())
		if err != nil {
			return err
		}
		mov.contentID = parseIlstDataString(d)
		return errStopIter
	})
}

func isMetaChild(typ string) bool {
	switch typ {
	case "hdlr", "keys", "ilst", "free", "dinf", "iinf", "iloc", "pitm", "xml ", "bxml", "uuid", "mhdr":
		return true
	}
	return false
}

const contentIDKey = "com.apple.quicktime.content.identifier"

// findContentIDKey 在 keys 盒子内容里找内容 ID 键，返回 1 开始的序号；找不到返回 0。
func findContentIDKey(d []byte) uint32 {
	if len(d) < 8 {
		return 0
	}
	count := binary.BigEndian.Uint32(d[4:8])
	off := 8
	for i := uint32(1); i <= count && off+8 <= len(d); i++ {
		size := int(binary.BigEndian.Uint32(d[off : off+4]))
		if size < 8 || size > len(d)-off {
			return 0
		}
		if string(d[off+4:off+8]) == "mdta" && string(d[off+8:off+size]) == contentIDKey {
			return i
		}
		off += size
	}
	return 0
}

// parseIlstDataString 从 ilst 项内容里取第一个 data 盒子的文本值。
func parseIlstDataString(d []byte) string {
	for off := 0; off+8 <= len(d); {
		size := int(binary.BigEndian.Uint32(d[off : off+4]))
		if size < 8 || size > len(d)-off {
			return ""
		}
		if string(d[off+4:off+8]) == "data" && size >= 16 {
			// data 内容：4 字节类型（1 = UTF-8）+ 4 字节语言，然后是值。
			return strings.TrimSpace(strings.TrimRight(string(d[off+16:off+size]), "\x00"))
		}
		off += size
	}
	return ""
}

// parseMoof 累加碎片里每条轨道的样本数和时长。有 tfdt 时以它为这段的起点。
func (mov *mp4Movie) parseMoof(d []byte) error {
	return eachBoxBytes(d, func(typ string, traf []byte) error {
		if typ != "traf" {
			return nil
		}
		var (
			track      *mp4Track
			defaultDur uint32
			haveTfhd   bool
			baseTime   uint64
			haveTfdt   bool
			samples    uint64
			total      uint64
		)
		var runs [][]byte
		err := eachBoxBytes(traf, func(typ string, c []byte) error {
			switch typ {
			case "tfhd":
				if len(c) < 8 {
					return unsupportedf("tfhd 太短")
				}
				haveTfhd = true
				flags := binary.BigEndian.Uint32(c[0:4]) & 0xffffff
				id := binary.BigEndian.Uint32(c[4:8])
				for _, tr := range mov.tracks {
					if tr.id == id {
						track = tr
					}
				}
				defaultDur = mov.trexDuration[id]
				off := 8
				if flags&0x1 != 0 {
					off += 8
				}
				if flags&0x2 != 0 {
					off += 4
				}
				if flags&0x8 != 0 {
					if len(c) < off+4 {
						return unsupportedf("tfhd 太短")
					}
					defaultDur = binary.BigEndian.Uint32(c[off : off+4])
				}
			case "tfdt":
				if len(c) >= 12 && c[0] == 1 {
					baseTime, haveTfdt = binary.BigEndian.Uint64(c[4:12]), true
				} else if len(c) >= 8 {
					baseTime, haveTfdt = uint64(binary.BigEndian.Uint32(c[4:8])), true
				}
			case "trun":
				runs = append(runs, c)
			}
			return nil
		})
		if err != nil {
			return err
		}
		if !haveTfhd || track == nil {
			// 没有 tfhd 或轨道号对不上 moov 里的任何轨道：这段碎片不属于已知轨道，不计入。
			return nil
		}
		for _, r := range runs {
			n, dur, err := parseTrun(r, defaultDur)
			if err != nil {
				return err
			}
			samples += n
			var carry uint64
			total, carry = bits.Add64(total, dur, 0)
			if carry != 0 {
				return unsupportedf("trun 时长溢出")
			}
		}
		if haveTfdt {
			track.fragCursor = baseTime
		}
		track.fragCursor += total
		track.fragEnd = max(track.fragEnd, track.fragCursor)
		track.fragSamples += samples
		track.fragDuration += total
		return nil
	})
}

// parseTrun 返回一个 trun 的样本数和总时长。每个样本没单独写时长时用默认时长。
func parseTrun(c []byte, defaultDur uint32) (samples, total uint64, err error) {
	if len(c) < 8 {
		return 0, 0, unsupportedf("trun 太短")
	}
	flags := binary.BigEndian.Uint32(c[0:4]) & 0xffffff
	count := uint64(binary.BigEndian.Uint32(c[4:8]))
	off := 8
	if flags&0x1 != 0 {
		off += 4
	}
	if flags&0x4 != 0 {
		off += 4
	}
	per := 4 * bits.OnesCount32(flags&0xf00)
	if off > len(c) || count*uint64(per) > uint64(len(c)-off) {
		return 0, 0, unsupportedf("trun 样本数 %d 越界", count)
	}
	if flags&0x100 == 0 {
		hi, lo := bits.Mul64(count, uint64(defaultDur))
		if hi != 0 {
			return 0, 0, unsupportedf("trun 时长溢出")
		}
		return count, lo, nil
	}
	for i := uint64(0); i < count; i++ {
		total += uint64(binary.BigEndian.Uint32(c[off : off+4]))
		off += per
	}
	return count, total, nil
}

// eachBoxBytes 遍历内存里的一串盒子，fn 拿到类型和内容。大小越界算格式坏了。
func eachBoxBytes(d []byte, fn func(typ string, payload []byte) error) error {
	for off := 0; off < len(d); {
		if len(d)-off < 8 {
			return nil
		}
		size := uint64(binary.BigEndian.Uint32(d[off : off+4]))
		typ := string(d[off+4 : off+8])
		hdr := 8
		switch size {
		case 1:
			if len(d)-off < 16 {
				return unsupportedf("%q 的 64 位大小越界", typ)
			}
			size = binary.BigEndian.Uint64(d[off+8 : off+16])
			hdr = 16
		case 0:
			size = uint64(len(d) - off)
		}
		if size < uint64(hdr) || size > uint64(len(d)-off) {
			return unsupportedf("%q 的大小 %d 越界", typ, size)
		}
		if err := fn(typ, d[off+hdr:off+int(size)]); err != nil {
			return err
		}
		off += int(size)
	}
	return nil
}

// ticksToDuration 把「刻度数 / 每秒刻度」换成 time.Duration，超过 Duration 能表示的范围时给 0。
func ticksToDuration(ticks uint64, timescale uint32) time.Duration {
	if timescale == 0 || ticks == 0 {
		return 0
	}
	ts := uint64(timescale)
	whole, rem := ticks/ts, ticks%ts
	if whole > uint64(math.MaxInt64/int64(time.Second))-1 {
		return 0
	}
	return time.Duration(whole)*time.Second + time.Duration(rem*uint64(time.Second)/ts)
}

func (mov *mp4Movie) info() (Info, error) {
	var video *mp4Track
	hasAudio := false
	for _, t := range mov.tracks {
		switch t.handler {
		case "vide":
			if video == nil {
				video = t
			}
		case "soun":
			hasAudio = true
		}
	}
	if video == nil {
		return Info{}, unsupportedf("没有视频轨")
	}
	info := Info{
		Width:     video.sampleW,
		Height:    video.sampleH,
		Rotation:  video.rotation,
		Codec:     video.codec,
		HasAudio:  hasAudio,
		ContentID: mov.contentID,
	}
	if info.Width == 0 || info.Height == 0 {
		info.Width, info.Height = video.tkhdW, video.tkhdH
	}
	info.Duration = mov.pickDuration(video)

	samples := video.sttsSamples
	ticks := video.sttsDuration
	if mov.fragmented && !mov.fragOverLimit {
		samples += video.fragSamples
		ticks += video.fragDuration
	}
	if d := ticksToDuration(ticks, video.timescale); d > 0 && samples > 0 {
		info.FrameRate = roundFrameRate(float64(samples) / d.Seconds())
	}
	return info, nil
}

// pickDuration 按可靠程度依次取：mvhd 总时长、mehd 碎片总时长、碎片累加、视频轨 mdhd、各轨最长 mdhd。
func (mov *mp4Movie) pickDuration(video *mp4Track) time.Duration {
	var d time.Duration
	if !mov.durationUnknown {
		d = ticksToDuration(mov.duration, mov.timescale)
	}
	if mov.fragmented {
		d = max(d, ticksToDuration(mov.fragmentDuration, mov.timescale))
		if d == 0 && !mov.fragOverLimit {
			// 没有 mehd 时用碎片里视频轨走到的最后时间点（起点含 moov 里的样本）。
			d = ticksToDuration(max(video.fragEnd, video.sttsDuration+video.fragDuration), video.timescale)
		}
		return d
	}
	if d > 0 {
		return d
	}
	if d = ticksToDuration(video.mdhdDuration, video.timescale); d > 0 {
		return d
	}
	for _, t := range mov.tracks {
		d = max(d, ticksToDuration(t.mdhdDuration, t.timescale))
	}
	return d
}
