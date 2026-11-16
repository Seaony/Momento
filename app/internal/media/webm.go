package media

import (
	"encoding/binary"
	"io"
	"math"
	"strings"
	"time"
)

// WebM / Matroska 用 EBML 编码：每个元素是「变长 ID + 变长大小 + 内容」。
// 只读文件头的 EBML（确认 DocType）和 Segment 里的 Info、Tracks；遇到 Cluster（真正的音视频数据）就停。
// Info、Tracks 不在 Cluster 前面时，按 SeekHead 里记的位置去读。

var ebmlMagic = []byte{0x1A, 0x45, 0xDF, 0xA3}

const (
	ebmlIDHeader        = 0x1A45DFA3
	ebmlIDDocType       = 0x4282
	ebmlIDSegment       = 0x18538067
	ebmlIDSeekHead      = 0x114D9B74
	ebmlIDSeek          = 0x4DBB
	ebmlIDSeekID        = 0x53AB
	ebmlIDSeekPosition  = 0x53AC
	ebmlIDInfo          = 0x1549A966
	ebmlIDTimecodeScale = 0x2AD7B1
	ebmlIDDuration      = 0x4489
	ebmlIDTracks        = 0x1654AE6B
	ebmlIDTrackEntry    = 0xAE
	ebmlIDTrackType     = 0x83
	ebmlIDCodecID       = 0x86
	ebmlIDDefaultDur    = 0x23E383
	ebmlIDVideo         = 0xE0
	ebmlIDPixelWidth    = 0xB0
	ebmlIDPixelHeight   = 0xBA
	ebmlIDCluster       = 0x1F43B675

	// EBML 头只有几个短字段，正常几十字节。
	maxEBMLHeaderSize = 4 << 10
	// SeekHead、Info 正常几百字节。
	maxSeekHeadSize = 1 << 20
	maxInfoSize     = 1 << 20
	// Tracks 里有编码私有数据（如 Opus 头、AV1 配置），一般几 KB。
	maxTracksSize = 8 << 20
	// Segment 下最多看多少个元素头，防止恶意文件塞满小元素。
	maxSegmentChildren = 1 << 16
)

// ebmlUnknownSize 表示元素大小未知（直播录制的 Segment、Cluster 常这样写）。
const ebmlUnknownSize = math.MaxUint64

// readEBMLVint 读一个变长整数。keepMarker 为 true 时保留长度标记位（元素 ID 的写法）。
// 返回值、占用字节数；大小字段的所有数据位全 1 时返回 ebmlUnknownSize。
func readEBMLVint(b []byte, maxLen int, keepMarker bool) (uint64, int, bool) {
	if len(b) == 0 || b[0] == 0 {
		return 0, 0, false
	}
	n := 1
	for mask := byte(0x80); b[0]&mask == 0; mask >>= 1 {
		n++
	}
	if n > maxLen || n > len(b) {
		return 0, 0, false
	}
	v := uint64(b[0])
	if !keepMarker {
		v &= uint64(0xFF >> n)
	}
	allOnes := v == uint64(0xFF>>n)
	for i := 1; i < n; i++ {
		v = v<<8 | uint64(b[i])
		allOnes = allOnes && b[i] == 0xFF
	}
	if !keepMarker && allOnes {
		return ebmlUnknownSize, n, true
	}
	return v, n, true
}

// parseEBMLHeader 解析元素头，返回 ID、内容大小、头长度。
func parseEBMLHeader(b []byte) (id uint32, size uint64, hdr int, err error) {
	idv, n, ok := readEBMLVint(b, 4, true)
	if !ok {
		return 0, 0, 0, unsupportedf("EBML 元素 ID 无效")
	}
	sz, m, ok := readEBMLVint(b[n:], 8, false)
	if !ok {
		return 0, 0, 0, unsupportedf("EBML 元素 %#x 的大小无效", idv)
	}
	return uint32(idv), sz, n + m, nil
}

type ebmlElement struct {
	id        uint32
	start     int64
	dataStart int64
	size      uint64 // 可能是 ebmlUnknownSize
}

// readEBMLElement 读文件 off 处的元素头（最多 12 字节：4 字节 ID + 8 字节大小）。
func readEBMLElement(r io.ReaderAt, off, end int64) (ebmlElement, error) {
	n := min(int64(12), end-off)
	if n < 2 {
		return ebmlElement{}, unsupportedf("EBML 元素头在 %d 处越界", off)
	}
	b, err := readFull(r, off, int(n))
	if err != nil {
		return ebmlElement{}, err
	}
	id, size, hdr, err := parseEBMLHeader(b)
	if err != nil {
		return ebmlElement{}, err
	}
	return ebmlElement{id: id, start: off, dataStart: off + int64(hdr), size: size}, nil
}

// eachEBMLBytes 遍历内存里的一串元素。内存里的元素必须有已知大小且不越界。
func eachEBMLBytes(d []byte, fn func(id uint32, payload []byte) error) error {
	for off := 0; off < len(d); {
		id, size, hdr, err := parseEBMLHeader(d[off:])
		if err != nil {
			return err
		}
		if size == ebmlUnknownSize || size > uint64(len(d)-off-hdr) {
			return unsupportedf("EBML 元素 %#x 大小越界", id)
		}
		if err := fn(id, d[off+hdr:off+hdr+int(size)]); err != nil {
			return err
		}
		off += hdr + int(size)
	}
	return nil
}

func ebmlUint(b []byte) uint64 {
	if len(b) > 8 {
		return 0
	}
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}

func ebmlFloat(b []byte) float64 {
	switch len(b) {
	case 4:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b)))
	case 8:
		return math.Float64frombits(binary.BigEndian.Uint64(b))
	}
	return 0
}

type webmTrack struct {
	typ             uint64
	codecID         string
	defaultDuration uint64
	width, height   uint64
}

type webmState struct {
	timecodeScale uint64
	duration      float64
	haveInfo      bool
	haveTracks    bool
	tracks        []webmTrack
	seekInfo      int64 // 相对 Segment 内容起点，-1 表示没有
	seekTracks    int64
}

func probeWebM(r io.ReaderAt, size int64) (Info, error) {
	head, err := readEBMLElement(r, 0, size)
	if err != nil {
		return Info{}, err
	}
	if head.id != ebmlIDHeader || head.size == ebmlUnknownSize || head.size > maxEBMLHeaderSize {
		return Info{}, unsupportedf("EBML 头无效")
	}
	headData, err := readFull(r, head.dataStart, int(head.size))
	if err != nil {
		return Info{}, err
	}
	docType := ""
	if err := eachEBMLBytes(headData, func(id uint32, p []byte) error {
		if id == ebmlIDDocType {
			docType = strings.TrimRight(string(p), "\x00")
		}
		return nil
	}); err != nil {
		return Info{}, err
	}
	if docType != "webm" && docType != "matroska" {
		return Info{}, unsupportedf("EBML 文档类型是 %q", docType)
	}

	// 找 Segment：EBML 头后面可能先有 Void 之类的元素。
	off := head.dataStart + int64(head.size)
	var seg ebmlElement
	for i := 0; ; i++ {
		if i > 16 || off >= size {
			return Info{}, unsupportedf("没有 Segment")
		}
		el, err := readEBMLElement(r, off, size)
		if err != nil {
			return Info{}, err
		}
		if el.id == ebmlIDSegment {
			seg = el
			break
		}
		if el.size == ebmlUnknownSize || el.size > uint64(size-el.dataStart) {
			return Info{}, unsupportedf("Segment 前的元素 %#x 大小无效", el.id)
		}
		off = el.dataStart + int64(el.size)
	}
	segEnd := size
	if seg.size != ebmlUnknownSize && seg.size <= uint64(size-seg.dataStart) {
		segEnd = seg.dataStart + int64(seg.size)
	}
	// 大小未知或超出文件尾（没录完、被截断）时读到文件尾为止。

	st := &webmState{timecodeScale: 1_000_000, seekInfo: -1, seekTracks: -1}
	off = seg.dataStart
	for i := 0; off < segEnd && !(st.haveInfo && st.haveTracks); i++ {
		if i >= maxSegmentChildren {
			return Info{}, unsupportedf("Segment 元素超过 %d 个", maxSegmentChildren)
		}
		if segEnd-off < 2 {
			break
		}
		el, err := readEBMLElement(r, off, segEnd)
		if err != nil {
			return Info{}, err
		}
		if el.id == ebmlIDCluster || el.size == ebmlUnknownSize || el.size > uint64(segEnd-el.dataStart) {
			// 到了音视频数据、或遇到没法跳过的元素：头信息该在前面已经读完，剩下的交给 SeekHead。
			break
		}
		if err := st.readTopElement(r, el); err != nil {
			return Info{}, err
		}
		off = el.dataStart + int64(el.size)
	}
	if !st.haveInfo && st.seekInfo >= 0 {
		if err := st.readSeekTarget(r, seg.dataStart, segEnd, st.seekInfo, ebmlIDInfo); err != nil {
			return Info{}, err
		}
	}
	if !st.haveTracks && st.seekTracks >= 0 {
		if err := st.readSeekTarget(r, seg.dataStart, segEnd, st.seekTracks, ebmlIDTracks); err != nil {
			return Info{}, err
		}
	}
	return st.info()
}

// readSeekTarget 按 SeekHead 记的相对位置读 Info 或 Tracks。位置上不是期望的元素就当文件坏了。
func (st *webmState) readSeekTarget(r io.ReaderAt, segStart, segEnd, rel int64, want uint32) error {
	if rel > segEnd-segStart {
		return unsupportedf("SeekHead 位置 %d 越界", rel)
	}
	el, err := readEBMLElement(r, segStart+rel, segEnd)
	if err != nil {
		return err
	}
	if el.id != want || el.size == ebmlUnknownSize || el.size > uint64(segEnd-el.dataStart) {
		return unsupportedf("SeekHead 指向的元素 %#x 不对", el.id)
	}
	return st.readTopElement(r, el)
}

// readTopElement 读 Segment 直接子元素里我们关心的三种：SeekHead、Info、Tracks。
func (st *webmState) readTopElement(r io.ReaderAt, el ebmlElement) error {
	var limit uint64
	switch el.id {
	case ebmlIDSeekHead:
		limit = maxSeekHeadSize
	case ebmlIDInfo:
		limit = maxInfoSize
	case ebmlIDTracks:
		limit = maxTracksSize
	default:
		return nil
	}
	if el.size > limit {
		if el.id == ebmlIDSeekHead {
			// SeekHead 只是索引，大到不正常就不用它，Info、Tracks 一般就在前面。
			return nil
		}
		return unsupportedf("元素 %#x 大小 %d 超过上限", el.id, el.size)
	}
	d, err := readFull(r, el.dataStart, int(el.size))
	if err != nil {
		return err
	}
	switch el.id {
	case ebmlIDSeekHead:
		return st.parseSeekHead(d)
	case ebmlIDInfo:
		st.haveInfo = true
		return eachEBMLBytes(d, func(id uint32, p []byte) error {
			switch id {
			case ebmlIDTimecodeScale:
				if v := ebmlUint(p); v > 0 {
					st.timecodeScale = v
				}
			case ebmlIDDuration:
				st.duration = ebmlFloat(p)
			}
			return nil
		})
	default:
		st.haveTracks = true
		return eachEBMLBytes(d, func(id uint32, p []byte) error {
			if id != ebmlIDTrackEntry {
				return nil
			}
			t, err := parseWebMTrack(p)
			if err != nil {
				return err
			}
			st.tracks = append(st.tracks, t)
			return nil
		})
	}
}

func (st *webmState) parseSeekHead(d []byte) error {
	return eachEBMLBytes(d, func(id uint32, p []byte) error {
		if id != ebmlIDSeek {
			return nil
		}
		var target uint32
		pos := int64(-1)
		if err := eachEBMLBytes(p, func(id uint32, v []byte) error {
			switch id {
			case ebmlIDSeekID:
				target = uint32(ebmlUint(v))
			case ebmlIDSeekPosition:
				if u := ebmlUint(v); u < math.MaxInt64 {
					pos = int64(u)
				}
			}
			return nil
		}); err != nil {
			return err
		}
		switch target {
		case ebmlIDInfo:
			if st.seekInfo < 0 {
				st.seekInfo = pos
			}
		case ebmlIDTracks:
			if st.seekTracks < 0 {
				st.seekTracks = pos
			}
		}
		return nil
	})
}

func parseWebMTrack(d []byte) (webmTrack, error) {
	var t webmTrack
	err := eachEBMLBytes(d, func(id uint32, p []byte) error {
		switch id {
		case ebmlIDTrackType:
			t.typ = ebmlUint(p)
		case ebmlIDCodecID:
			t.codecID = strings.TrimRight(string(p), "\x00")
		case ebmlIDDefaultDur:
			t.defaultDuration = ebmlUint(p)
		case ebmlIDVideo:
			return eachEBMLBytes(p, func(id uint32, v []byte) error {
				switch id {
				case ebmlIDPixelWidth:
					t.width = ebmlUint(v)
				case ebmlIDPixelHeight:
					t.height = ebmlUint(v)
				}
				return nil
			})
		}
		return nil
	})
	return t, err
}

// webmCodec 把 Matroska 的 CodecID 换成和 MP4 一致风格的短名。
func webmCodec(id string) string {
	switch id {
	case "V_VP8":
		return "vp8"
	case "V_VP9":
		return "vp9"
	case "V_AV1":
		return "av1"
	case "V_MPEG4/ISO/AVC":
		return "avc1"
	case "V_MPEGH/ISO/HEVC":
		return "hvc1"
	}
	return strings.ToLower(strings.TrimPrefix(id, "V_"))
}

const (
	webmTrackVideo = 1
	webmTrackAudio = 2
	// 宽高超过这个就当数据坏了（Matroska 规范没限制，实际视频远小于此）。
	maxVideoDimension = 1 << 16
)

func (st *webmState) info() (Info, error) {
	if !st.haveTracks {
		return Info{}, unsupportedf("没有 Tracks")
	}
	var video *webmTrack
	hasAudio := false
	for i := range st.tracks {
		switch st.tracks[i].typ {
		case webmTrackVideo:
			if video == nil {
				video = &st.tracks[i]
			}
		case webmTrackAudio:
			hasAudio = true
		}
	}
	if video == nil {
		return Info{}, unsupportedf("没有视频轨")
	}
	if video.width > maxVideoDimension || video.height > maxVideoDimension {
		return Info{}, unsupportedf("视频尺寸 %dx%d 不正常", video.width, video.height)
	}
	info := Info{
		Width:    int(video.width),
		Height:   int(video.height),
		Codec:    webmCodec(video.codecID),
		HasAudio: hasAudio,
	}
	// Duration 以 TimecodeScale 纳秒为单位。浏览器 MediaRecorder 录的文件常常没有 Duration，这时给 0（读不出）。
	if ns := st.duration * float64(st.timecodeScale); ns > 0 && ns < float64(math.MaxInt64) && !math.IsNaN(ns) {
		info.Duration = time.Duration(ns)
	}
	if video.defaultDuration > 0 {
		info.FrameRate = roundFrameRate(float64(time.Second) / float64(video.defaultDuration))
	}
	return info, nil
}
