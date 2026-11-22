package assets

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// MaxGIFDecodeBytes 是一个 GIF 全量解码后允许占用的帧像素字节数（每像素 1 字节的索引色）。
// 取 256 MiB：共享解码预算 decodeBudget 是 1 GiB，预览、悬停、导出可能同时各解一个，
// 再加上播放时每个动图还有一张 RGBA 画布和位图，单个动图只占预算的四分之一。
// 常见 GIF（几百像素宽、几百帧）远低于它；1920×1080 满画幅超过约 128 帧才会被拒绝。
const MaxGIFDecodeBytes int64 = 256 << 20

// gifFrameOverhead 是每帧除像素和颜色表以外的固定开销（image.Paletted、延时、处理方式等），按 256 字节估。
const gifFrameOverhead = 256

// GIFDecodeCost 只读 GIF 的头和块结构（不解 LZW 像素），返回全量解码并播放要占的字节数：
// 各帧按自己的矩形每像素 1 字节（和 image/gif 实际分配的一致），再加播放时合成用的逻辑画布
// （mygo 的 GIF 解码器一张 RGBA 画布，遇到「恢复到上一帧」再存一份同样大的副本，所以按 8 字节一像素算）。
// 逻辑画布可以远大于各帧，只看帧矩形会漏掉画布那一大块。
func GIFDecodeCost(r io.Reader) (frames int, bytes int64, err error) {
	br := bufio.NewReader(r)
	var head [13]byte
	if _, err = io.ReadFull(br, head[:]); err != nil {
		return 0, 0, err
	}
	if string(head[:3]) != "GIF" {
		return 0, 0, errors.New("not a GIF")
	}
	bytes = int64(binary.LittleEndian.Uint16(head[6:8])) * int64(binary.LittleEndian.Uint16(head[8:10])) * 8
	// 没有局部颜色表的帧用全局颜色表；带透明色时解码器给每帧复制一份，按每帧都复制算。
	globalColors := int64(0)
	if head[10]&0x80 != 0 {
		globalColors = int64(2 << (head[10] & 7))
	}
	if err = skipColorTable(br, head[10]); err != nil {
		return 0, 0, err
	}
	for {
		block, err := br.ReadByte()
		if err != nil {
			return frames, bytes, err
		}
		switch block {
		case 0x3B:
			return frames, bytes, nil
		case 0x21:
			if _, err = br.ReadByte(); err != nil {
				return frames, bytes, err
			}
			if err = skipSubBlocks(br); err != nil {
				return frames, bytes, err
			}
		case 0x2C:
			var desc [9]byte
			if _, err = io.ReadFull(br, desc[:]); err != nil {
				return frames, bytes, err
			}
			frames++
			bytes += int64(binary.LittleEndian.Uint16(desc[4:6]))*int64(binary.LittleEndian.Uint16(desc[6:8])) + gifFrameOverhead
			// 颜色表解码后每色是一个 color.Color 接口值（16 字节）加一个 RGBA 值。
			if desc[8]&0x80 != 0 {
				bytes += int64(2<<(desc[8]&7)) * 20
			} else {
				bytes += globalColors * 20
			}
			if err = skipColorTable(br, desc[8]); err != nil {
				return frames, bytes, err
			}
			if _, err = br.ReadByte(); err != nil { // LZW 最小码长
				return frames, bytes, err
			}
			if err = skipSubBlocks(br); err != nil {
				return frames, bytes, err
			}
		default:
			return frames, bytes, errors.New("bad GIF block")
		}
	}
}

// 带全局或局部颜色表的标志位时，跳过 3×2^(n+1) 字节。
func skipColorTable(br *bufio.Reader, packed byte) error {
	if packed&0x80 == 0 {
		return nil
	}
	_, err := br.Discard(3 << (packed&7 + 1))
	return err
}

func skipSubBlocks(br *bufio.Reader) error {
	for {
		size, err := br.ReadByte()
		if err != nil || size == 0 {
			return err
		}
		if _, err = br.Discard(int(size)); err != nil {
			return err
		}
	}
}

// CheckGIFDecodeSize 在全量解码前检查 GIF 的总帧像素，超过 MaxGIFDecodeBytes 返回 error_animation_too_large。
// 头信息读不出时不拦，交给后面的解码器给出真正的错误。
func CheckGIFDecodeSize(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	frames, size, _ := GIFDecodeCost(f) // 读到一半出错时已统计的部分照样比较
	if size > MaxGIFDecodeBytes {
		return i18n.NewError("error_animation_too_large", frames, size>>20, MaxGIFDecodeBytes>>20)
	}
	return nil
}

// animatedAVIF 读 ftyp 盒判断是不是动态 AVIF（品牌 avis）：解码库不提供只读帧数的接口，
// 不先判断就只能 DecodeAll 把所有帧解开。
func animatedAVIF(f *os.File) (bool, error) {
	var head [16]byte
	if _, err := f.ReadAt(head[:8], 0); err != nil {
		return false, err
	}
	size := int64(binary.BigEndian.Uint32(head[:4]))
	if string(head[4:8]) != "ftyp" || size < 16 || size > 4096 {
		return false, nil
	}
	box := make([]byte, size-8)
	if _, err := f.ReadAt(box, 8); err != nil {
		return false, err
	}
	if string(box[:4]) == "avis" {
		return true, nil
	}
	for i := 8; i+4 <= len(box); i += 4 {
		if string(box[i:i+4]) == "avis" {
			return true, nil
		}
	}
	return false, nil
}
