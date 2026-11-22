package ui

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestHoverAnimationFormats(t *testing.T) {
	for ext, want := range map[string]string{"GIF": "gif", "png": "png", "apng": "png", "webp": "", "jpg": ""} {
		if got := hoverAnimation(model.Asset{Extension: ext}); got != want {
			t.Fatalf("%s: %q", ext, got)
		}
	}
}

func TestOpenHoverAnimation(t *testing.T) {
	dir := t.TempDir()
	frame := func(c color.Color) *image.Paletted {
		img := image.NewPaletted(image.Rect(0, 0, 4, 4), color.Palette{color.Black, color.White})
		for i := range img.Pix {
			if c == color.White {
				img.Pix[i] = 1
			}
		}
		return img
	}
	var animated bytes.Buffer
	if err := gif.EncodeAll(&animated, &gif.GIF{Image: []*image.Paletted{frame(color.Black), frame(color.White)}, Delay: []int{10, 10}}); err != nil {
		t.Fatal(err)
	}
	gifPath := filepath.Join(dir, "a.gif")
	if err := os.WriteFile(gifPath, animated.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	player, err := openHoverAnimation(context.Background(), gifPath, "gif")
	if err != nil || player == nil {
		t.Fatalf("GIF 打不开：%v", err)
	}
	player.Close()

	// 普通 PNG 不是动画，不播。
	var still bytes.Buffer
	if err := png.Encode(&still, frame(color.White)); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "a.png")
	if err := os.WriteFile(pngPath, still.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if player, err := openHoverAnimation(context.Background(), pngPath, "png"); err != nil || player != nil {
		t.Fatalf("普通 PNG 被当成动画：%v %v", player, err)
	}
	if animated, _ := assets.IsAnimatedImage(bytes.NewReader(still.Bytes())); animated {
		t.Fatal("普通 PNG 被认成 APNG")
	}
	// 在 IHDR 后面插一个 acTL 块就是 APNG 的标志。
	data := still.Bytes()
	ihdrEnd := 8 + 12 + int(binary.BigEndian.Uint32(data[8:]))
	chunk := make([]byte, 12+8)
	binary.BigEndian.PutUint32(chunk, 8)
	copy(chunk[4:], "acTL")
	binary.BigEndian.PutUint32(chunk[8:], 1)
	binary.BigEndian.PutUint32(chunk[16:], crc32.ChecksumIEEE(chunk[4:16]))
	apng := append(append(append([]byte{}, data[:ihdrEnd]...), chunk...), data[ihdrEnd:]...)
	if animated, _ := assets.IsAnimatedImage(bytes.NewReader(apng)); !animated {
		t.Fatal("没认出 APNG")
	}
}

func TestHoverHugeGIFIsNotDecoded(t *testing.T) {
	path := writeHugeGIF(t)
	player, err := openHoverAnimation(context.Background(), path, "gif")
	// 报「太大」，卡片据此给提示，不当成打不开。
	if !errors.Is(err, errHoverAnimationTooLarge) || player != nil {
		t.Fatalf("大 GIF 不该解码播放：%v %v", player, err)
	}
}

func TestHoverAnimationCancelledBeforeDecoding(t *testing.T) {
	path := writePreviewGIF(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if player, err := openHoverAnimation(ctx, path, "gif"); !errors.Is(err, context.Canceled) || player != nil {
		t.Fatalf("取消后不该继续解码：%v %v", player, err)
	}
}
