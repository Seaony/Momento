package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// 样本由 testdata/probe_generate.sh 用 ffmpeg 生成。
func TestProbeSamples(t *testing.T) {
	cases := []struct {
		file string
		want Info
	}{
		{"probe_h264.mp4", Info{Width: 160, Height: 120, Duration: 2 * time.Second, Codec: "avc1", FrameRate: 25}},
		{"probe_h264_audio.mp4", Info{Width: 160, Height: 120, Duration: 2 * time.Second, Codec: "avc1", FrameRate: 25, HasAudio: true}},
		{"probe_rot90.mov", Info{Width: 160, Height: 120, Rotation: 90, Duration: time.Second, Codec: "avc1", FrameRate: 25}},
		{"probe_hevc.m4v", Info{Width: 160, Height: 120, Duration: time.Second, Codec: "hvc1", FrameRate: 25}},
		{"probe_av1.mp4", Info{Width: 160, Height: 120, Duration: time.Second, Codec: "av01", FrameRate: 25}},
		{"probe_frag.mp4", Info{Width: 160, Height: 120, Duration: 2 * time.Second, Codec: "avc1", FrameRate: 25}},
		{"probe_livephoto.mov", Info{Width: 160, Height: 120, Duration: time.Second, Codec: "avc1", FrameRate: 25,
			ContentID: "FD0B0708-C73F-4F6D-B57B-4169BB991D46"}},
		{"probe_vp9.webm", Info{Width: 160, Height: 120, Duration: 1008 * time.Millisecond, Codec: "vp9", FrameRate: 25, HasAudio: true}},
		{"probe_vp8.webm", Info{Width: 160, Height: 120, Duration: time.Second, Codec: "vp8", FrameRate: 25}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			got, err := Probe(filepath.Join("testdata", c.file))
			if err != nil {
				t.Fatalf("Probe: %v", err)
			}
			if got != c.want {
				t.Fatalf("Probe =\n %+v\nwant\n %+v", got, c.want)
			}
		})
	}
	if w, h := (Info{Width: 160, Height: 120, Rotation: 90}).DisplaySize(); w != 120 || h != 160 {
		t.Fatalf("旋转 90 度后显示尺寸应为 120x160，得到 %dx%d", w, h)
	}
}

func TestProbeMatrixRotation(t *testing.T) {
	const one = 1 << 16
	cases := []struct {
		a, b int32
		want int
	}{
		{one, 0, 0}, {0, one, 90}, {-one, 0, 180}, {0, -one, 270}, {0, 0, 0},
	}
	for _, c := range cases {
		if got := matrixRotation(c.a, c.b); got != c.want {
			t.Errorf("matrixRotation(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func readSample(t testing.TB, name string) []byte {
	t.Helper()
	d, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// topBoxOffset 返回顶层第一个指定类型盒子的偏移。
func topBoxOffset(t testing.TB, d []byte, typ string) int {
	t.Helper()
	for off := 0; off+8 <= len(d); {
		size := int(binary.BigEndian.Uint32(d[off:]))
		if string(d[off+4:off+8]) == typ {
			return off
		}
		if size < 8 {
			break
		}
		off += size
	}
	t.Fatalf("没有顶层 %q", typ)
	return 0
}

func probeBytes(d []byte) (Info, error) {
	return probeReaderAt(bytes.NewReader(d), int64(len(d)))
}

// 64 位大小：把 free(8 字节) + mdat 头合并成一个 size=1 的 mdat 头，总长度不变。
func TestProbeLargeBoxSize(t *testing.T) {
	d := readSample(t, "probe_h264.mp4") // ftyp free mdat moov
	free := topBoxOffset(t, d, "free")
	mdat := topBoxOffset(t, d, "mdat")
	if mdat != free+8 {
		t.Fatalf("样本结构变了：free 后面不是 mdat")
	}
	mdatSize := binary.BigEndian.Uint32(d[mdat:])
	binary.BigEndian.PutUint32(d[free:], 1)
	copy(d[free+4:], "mdat")
	binary.BigEndian.PutUint64(d[free+8:], uint64(mdatSize)+8)
	got, err := probeBytes(d)
	if err != nil || got.Duration != 2*time.Second || got.Width != 160 {
		t.Fatalf("64 位大小：%+v, %v", got, err)
	}
}

// size=0 表示盒子一直到文件尾。
func TestProbeZeroSizeBox(t *testing.T) {
	d := readSample(t, "probe_h264_audio.mp4") // ftyp moov free mdat
	binary.BigEndian.PutUint32(d[topBoxOffset(t, d, "mdat"):], 0)
	got, err := probeBytes(d)
	if err != nil || got.Duration != 2*time.Second || !got.HasAudio {
		t.Fatalf("size=0：%+v, %v", got, err)
	}
}

// 没下载完的文件：moov 在前且完整时照常读出，moov 被截断时返回 ErrUnsupported。
func TestProbeTruncated(t *testing.T) {
	d := readSample(t, "probe_h264_audio.mp4")
	mdat := topBoxOffset(t, d, "mdat")
	if got, err := probeBytes(d[:mdat+100]); err != nil || got.Duration != 2*time.Second {
		t.Fatalf("mdat 截断：%+v, %v", got, err)
	}
	moov := topBoxOffset(t, d, "moov")
	if _, err := probeBytes(d[:moov+200]); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("moov 截断应返回 ErrUnsupported，得到 %v", err)
	}
	tail := readSample(t, "probe_h264.mp4") // moov 在尾部
	if _, err := probeBytes(tail[:len(tail)-300]); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("尾部 moov 截断应返回 ErrUnsupported，得到 %v", err)
	}
}

// ---- 合成 MP4，用来测碎片化的各种写法 ----

func box(typ string, parts ...[]byte) []byte {
	var body []byte
	for _, p := range parts {
		body = append(body, p...)
	}
	out := binary.BigEndian.AppendUint32(nil, uint32(8+len(body)))
	return append(append(out, typ...), body...)
}

func u32s(vs ...uint32) []byte {
	var out []byte
	for _, v := range vs {
		out = binary.BigEndian.AppendUint32(out, v)
	}
	return out
}

func u64(v uint64) []byte { return binary.BigEndian.AppendUint64(nil, v) }

// synthTrak 造一条 30 帧/秒、timescale 30000 的 640x480 视频轨，stts 里有 moovSamples 个样本。
func synthTrak(id uint32, handler string, matrix [2]int32, moovSamples uint32) []byte {
	tkhd := u32s(0, 0, 0, id, 0, 0, 0, 0, 0, 0)
	tkhd = append(tkhd, u32s(uint32(matrix[0]), uint32(matrix[1]), 0, uint32(-matrix[1]), uint32(matrix[0]), 0, 0, 0, 1<<30)...)
	tkhd = append(tkhd, u32s(640<<16, 480<<16)...)
	mdhd := u32s(0, 0, 0, 30000, moovSamples*1001, 0)
	hdlr := append(u32s(0, 0), []byte(handler)...)
	hdlr = append(hdlr, make([]byte, 13)...)
	entry := make([]byte, 78)
	binary.BigEndian.PutUint16(entry[24:], 640)
	binary.BigEndian.PutUint16(entry[26:], 480)
	stsd := append(u32s(0, 1), box("avc1", entry)...)
	stts := u32s(0, 0)
	if moovSamples > 0 {
		stts = u32s(0, 1, moovSamples, 1001)
	}
	return box("trak", box("tkhd", tkhd), box("mdia", box("mdhd", mdhd), box("hdlr", hdlr),
		box("minf", box("stbl", box("stsd", stsd), box("stts", stts)))))
}

func synthMoof(trackID uint32, tfdt *uint64, sampleDurations []uint32, defaultDur uint32, n uint32) []byte {
	tfhdFlags, tfhd := uint32(0x020000), u32s(0, trackID)
	if defaultDur > 0 {
		tfhdFlags |= 0x8
		tfhd = u32s(0, trackID, defaultDur)
	}
	binary.BigEndian.PutUint32(tfhd, tfhdFlags)
	var trun []byte
	if sampleDurations != nil {
		trun = u32s(0x100, uint32(len(sampleDurations)))
		trun = append(trun, u32s(sampleDurations...)...)
	} else {
		trun = u32s(0, n)
	}
	parts := [][]byte{box("tfhd", tfhd)}
	if tfdt != nil {
		parts = append(parts, box("tfdt", append(u32s(1<<24), u64(*tfdt)...)))
	}
	parts = append(parts, box("trun", trun))
	return box("moof", box("mfhd", u32s(0, 1)), box("traf", parts...))
}

func synthFile(moovParts [][]byte, tail ...[]byte) []byte {
	out := box("ftyp", []byte("isom"), u32s(0x200), []byte("isomiso6"))
	out = append(out, box("moov", moovParts...)...)
	for _, p := range tail {
		out = append(out, p...)
	}
	return out
}

func TestProbeFragmentedVariants(t *testing.T) {
	mvhd := func(dur uint32) []byte { return box("mvhd", u32s(0, 0, 0, 1000, dur), make([]byte, 80)) }
	trex := box("trex", u32s(0, 1, 1, 1001, 0, 0))
	ident := [2]int32{1 << 16, 0}

	t.Run("mehd", func(t *testing.T) {
		d := synthFile([][]byte{mvhd(0), synthTrak(1, "vide", ident, 0),
			box("mvex", box("mehd", u32s(0, 5000)), trex)},
			synthMoof(1, nil, nil, 0, 150))
		got, err := probeBytes(d)
		if err != nil || got.Duration != 5*time.Second {
			t.Fatalf("%+v, %v", got, err)
		}
	})
	t.Run("trun 每个样本单独写时长", func(t *testing.T) {
		durs := make([]uint32, 60)
		for i := range durs {
			durs[i] = 1001
		}
		d := synthFile([][]byte{mvhd(0), synthTrak(1, "vide", ident, 0), box("mvex", trex)},
			synthMoof(1, nil, durs, 0, 0), synthMoof(1, nil, durs, 0, 0))
		got, err := probeBytes(d)
		if err != nil || got.Duration != 4004*time.Millisecond || got.FrameRate != 29.97 {
			t.Fatalf("%+v, %v", got, err)
		}
	})
	t.Run("tfdt 定起点、tfhd 默认时长", func(t *testing.T) {
		base := uint64(30000 * 10) // 第二段从 10 秒开始（前面的碎片没给）
		d := synthFile([][]byte{mvhd(0), synthTrak(1, "vide", ident, 30), box("mvex", trex)},
			synthMoof(1, &base, nil, 1000, 30))
		got, err := probeBytes(d)
		if err != nil || got.Duration != 11*time.Second {
			t.Fatalf("%+v, %v", got, err)
		}
	})
	t.Run("旋转 180 和 270", func(t *testing.T) {
		for want, m := range map[int][2]int32{180: {-1 << 16, 0}, 270: {0, -1 << 16}} {
			d := synthFile([][]byte{mvhd(2000), synthTrak(1, "vide", m, 60)})
			got, err := probeBytes(d)
			if err != nil || got.Rotation != want || got.Duration != 2*time.Second {
				t.Fatalf("want %d: %+v, %v", want, got, err)
			}
		}
	})
	t.Run("只有音频轨", func(t *testing.T) {
		d := synthFile([][]byte{mvhd(2000), synthTrak(1, "soun", ident, 60)})
		if _, err := probeBytes(d); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("纯音频应返回 ErrUnsupported，得到 %v", err)
		}
	})
}

// 恶意或损坏的结构：都要返回 ErrUnsupported，不能 panic、不能卡住。
func TestProbeHostile(t *testing.T) {
	ident := [2]int32{1 << 16, 0}
	mvhd := box("mvhd", u32s(0, 0, 0, 1000, 1000), make([]byte, 80))
	manyBoxes := make([]byte, 0, 8*(maxBMFFBoxVisits+10))
	for i := 0; i < maxBMFFBoxVisits+10; i++ {
		manyBoxes = append(manyBoxes, 0, 0, 0, 8, 'f', 'r', 'e', 'e')
	}
	hugeStts := synthTrak(1, "vide", ident, 0)
	// 把 stts 的条目数改成 0xFFFFFFFF。
	if i := bytes.Index(hugeStts, []byte("stts")); i > 0 {
		binary.BigEndian.PutUint32(hugeStts[i+8:], math.MaxUint32)
	}
	cases := map[string][]byte{
		"空文件":              {},
		"文本":               []byte("hello, this is not a video at all"),
		"全零":               make([]byte, 4096),
		"HEIC 图片（没有 moov）": box("ftyp", []byte("heic"), u32s(0), []byte("mif1heic")),
		"moov 大小超出文件":      append(box("ftyp", []byte("isom"), u32s(0)), 0xFF, 0xFF, 0xFF, 0xF0, 'm', 'o', 'o', 'v', 1, 2, 3),
		"64 位大小比头还小":       append(box("ftyp", []byte("isom"), u32s(0)), append(u32s(1), append([]byte("moov"), u64(3)...)...)...),
		"子盒子越界":            synthFile([][]byte{mvhd, {0, 0, 0x10, 0, 't', 'r', 'a', 'k', 0, 0}}),
		"大量小盒子":            synthFile([][]byte{mvhd, synthTrak(1, "vide", ident, 1), manyBoxes}),
		"stts 条目数超大":       synthFile([][]byte{mvhd, hugeStts}),
		"EBML 未知 DocType":  append(append([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x87}, 0x42, 0x82, 0x84), "avi "...),
		"EBML 头大小超大":       {0x1A, 0x45, 0xDF, 0xA3, 0x08, 0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0},
		"EBML 无效 vint":     {0x1A, 0x45, 0xDF, 0xA3, 0x00, 0x00, 0x00, 0x00, 0, 0, 0, 0},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			if _, err := probeBytes(d); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("应返回 ErrUnsupported，得到 %v", err)
			}
			if time.Since(start) > 5*time.Second {
				t.Fatalf("耗时 %v 太长", time.Since(start))
			}
		})
	}
}

// WebM 的 Info、Tracks 大小越界、Segment 未知大小。
func TestProbeWebMVariants(t *testing.T) {
	d := readSample(t, "probe_vp9.webm")
	seg := bytes.Index(d, []byte{0x18, 0x53, 0x80, 0x67})
	if seg < 0 {
		t.Fatal("样本里没有 Segment")
	}
	// Segment 改成未知大小（直播录制的写法）。
	unknown := bytes.Clone(d)
	sizeLen := 1
	for mask := byte(0x80); unknown[seg+4]&mask == 0; mask >>= 1 {
		sizeLen++
	}
	unknown[seg+4] = 0xFF >> (sizeLen - 1)
	for i := 1; i < sizeLen; i++ {
		unknown[seg+4+i] = 0xFF
	}
	if got, err := probeBytes(unknown); err != nil || got.Width != 160 || got.Codec != "vp9" {
		t.Fatalf("Segment 未知大小：%+v, %v", got, err)
	}
	// 截掉 Tracks 之后的内容：Tracks 不完整要返回 ErrUnsupported。
	tracks := bytes.Index(d, []byte{0x16, 0x54, 0xAE, 0x6B})
	if _, err := probeBytes(d[:tracks+20]); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Tracks 截断应返回 ErrUnsupported，得到 %v", err)
	}
}

// 对每个样本做各种截断和随机改字节，只要求不 panic，错误只能是 ErrUnsupported。
func TestProbeCorruptSamplesNoPanic(t *testing.T) {
	files, err := filepath.Glob("testdata/probe_*")
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for _, f := range files {
		if filepath.Ext(f) == ".sh" {
			continue
		}
		d := readSample(t, filepath.Base(f))
		for cut := 0; cut < len(d); cut += max(1, len(d)/200) {
			if _, err := probeBytes(d[:cut]); err != nil && !errors.Is(err, ErrUnsupported) {
				t.Fatalf("%s 截到 %d：错误不是 ErrUnsupported：%v", f, cut, err)
			}
		}
		for i := 0; i < 300; i++ {
			m := bytes.Clone(d)
			for j := 0; j < 1+rng.IntN(8); j++ {
				// 改动集中在前 4KB 和尾部 2KB（头信息所在），这样才打得到结构字段。
				pos := rng.IntN(min(len(m), 4096))
				if rng.IntN(2) == 0 {
					pos = len(m) - 1 - rng.IntN(min(len(m), 2048))
				}
				m[pos] = byte(rng.Uint32())
			}
			if _, err := probeBytes(m); err != nil && !errors.Is(err, ErrUnsupported) {
				t.Fatalf("%s 随机改字节后错误不是 ErrUnsupported：%v", f, err)
			}
		}
	}
}

func TestProbeIOErrors(t *testing.T) {
	_, err := Probe(filepath.Join(t.TempDir(), "missing.mp4"))
	if !errors.Is(err, fs.ErrNotExist) || errors.Is(err, ErrUnsupported) {
		t.Fatalf("文件不存在应返回 IO 错误，得到 %v", err)
	}
	if _, err := Probe(t.TempDir()); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("文件夹应返回 ErrUnsupported，得到 %v", err)
	}
	// 读盘失败（不是 EOF）要原样返回，不能伪装成格式不支持。
	boom := errors.New("disk failure")
	_, err = probeReaderAt(failingReader{boom}, 1<<20)
	if !errors.Is(err, boom) || errors.Is(err, ErrUnsupported) {
		t.Fatalf("读盘错误应原样返回，得到 %v", err)
	}
}

type failingReader struct{ err error }

func (r failingReader) ReadAt([]byte, int64) (int, error) { return 0, r.err }

// sparseVideo 模拟一个 1GB 级的视频：ftyp + 巨大的 mdat（读到全是 0）+ 尾部 moov，统计实际读了多少字节。
type sparseVideo struct {
	head, tail []byte
	size       int64
	read       atomic.Int64
}

func newSparseVideo(t testing.TB, mdatSize int64) *sparseVideo {
	d := readSample(t, "probe_h264.mp4") // ftyp free mdat moov
	moov := d[topBoxOffset(t, d, "moov"):]
	ftyp := d[:topBoxOffset(t, d, "free")]
	head := append(bytes.Clone(ftyp), u32s(1)...)
	head = append(head, "mdat"...)
	head = append(head, u64(uint64(mdatSize))...)
	return &sparseVideo{head: head, tail: moov, size: int64(len(ftyp)) + mdatSize + int64(len(moov))}
}

func (s *sparseVideo) ReadAt(p []byte, off int64) (int, error) {
	if off >= s.size {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) && off+int64(n) < s.size {
		pos := off + int64(n)
		switch tailStart := s.size - int64(len(s.tail)); {
		case pos < int64(len(s.head)):
			n += copy(p[n:], s.head[pos:])
		case pos >= tailStart:
			n += copy(p[n:], s.tail[pos-tailStart:])
		default:
			k := min(int64(len(p)-n), tailStart-pos)
			clear(p[n : n+int(k)])
			n += int(k)
		}
	}
	s.read.Add(int64(n))
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func TestProbeLargeFileReadsOnlyHeaders(t *testing.T) {
	v := newSparseVideo(t, 1<<30)
	got, err := probeReaderAt(v, v.size)
	if err != nil || got.Duration != 2*time.Second || got.Width != 160 {
		t.Fatalf("%+v, %v", got, err)
	}
	if n := v.read.Load(); n > 16<<10 {
		t.Fatalf("1GB 视频读了 %d 字节，应该只读头尾几 KB", n)
	}
}

// 真实磁盘上的 1GB 稀疏文件（不占实际空间），moov 在尾部。
func BenchmarkProbeLargeSparseFile(b *testing.B) {
	v := newSparseVideo(b, 1<<30)
	path := filepath.Join(b.TempDir(), "big.mp4")
	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	if _, err := f.Write(v.head); err != nil {
		b.Fatal(err)
	}
	if _, err := f.WriteAt(v.tail, v.size-int64(len(v.tail))); err != nil {
		b.Fatal(err)
	}
	if err := f.Close(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if _, err := Probe(path); err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzProbe(f *testing.F) {
	files, err := filepath.Glob("testdata/probe_*")
	if err != nil {
		f.Fatal(err)
	}
	for _, p := range files {
		if filepath.Ext(p) == ".sh" {
			continue
		}
		d, err := os.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(d)
	}
	f.Fuzz(func(t *testing.T, d []byte) {
		if _, err := probeBytes(d); err != nil && !errors.Is(err, ErrUnsupported) {
			t.Fatalf("错误不是 ErrUnsupported：%v", err)
		}
	})
}
