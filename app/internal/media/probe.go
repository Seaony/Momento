package media

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

// Probe 读视频文件头，返回时长、尺寸、旋转、编码等信息。
// 只按容器结构跳着读需要的那几块（MP4/MOV 的 moov、WebM 的 Info 和 Tracks），不把整个文件读进内存。
// 不是能识别的视频、或者头信息坏了，返回能被 errors.Is(err, ErrUnsupported) 认出的错误；
// 打不开、读盘失败这类 IO 错误原样返回。
func Probe(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Info{}, err
	}
	if st.IsDir() {
		return Info{}, unsupportedf("%s 是文件夹", path)
	}
	return probeReaderAt(f, st.Size())
}

// probeReaderAt 是 Probe 的实际实现，单独拿出来方便测试时统计读了多少字节。
func probeReaderAt(r io.ReaderAt, size int64) (Info, error) {
	if size < 8 {
		return Info{}, unsupportedf("文件太小（%d 字节）", size)
	}
	head, err := readFull(r, 0, 8)
	if err != nil {
		return Info{}, err
	}
	switch {
	case bytes.Equal(head[:4], ebmlMagic):
		return probeWebM(r, size)
	case isBMFFTopLevel(string(head[4:8])):
		return probeBMFF(r, size)
	default:
		return Info{}, unsupportedf("不认识的文件头 % x", head)
	}
}

// unsupportedf 生成包着 ErrUnsupported 的错误，附上具体原因方便查日志。
func unsupportedf(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrUnsupported}, args...)...)
}

// readFull 从 off 处读满 n 字节。
// 文件比结构里声称的短（截断、大小字段乱写）算格式坏了，返回 ErrUnsupported；其他 IO 错误原样返回。
// n 的上限由调用方按各自的结构限制好，这里不会被恶意大小骗去分配大内存。
func readFull(r io.ReaderAt, off int64, n int) ([]byte, error) {
	if off < 0 || n < 0 {
		return nil, unsupportedf("读取位置无效（%d，%d 字节）", off, n)
	}
	buf := make([]byte, n)
	got, err := r.ReadAt(buf, off)
	if got == n {
		// ReadAt 读满时也可能顺带返回 io.EOF，数据是完整的。
		return buf, nil
	}
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, unsupportedf("文件在 %d 处提前结束", off+int64(got))
	}
	return nil, err
}

// roundFrameRate 把帧率保留三位小数，避免 29.97002997 这种长尾数。
func roundFrameRate(fps float64) float64 {
	if fps <= 0 || math.IsNaN(fps) || math.IsInf(fps, 0) || fps > 10000 {
		return 0
	}
	return math.Round(fps*1000) / 1000
}
