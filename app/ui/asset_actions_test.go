package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

// 复制图片：PNG 原样放进剪贴板；JPEG 和旋转过的转成转好方向的 PNG。
func TestAssetPNG(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "a.png")
	if err := os.WriteFile(pngPath, encoded.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := assetPNG(pngPath, model.Asset{Extension: "png"})
	if err != nil || !bytes.Equal(data, encoded.Bytes()) {
		t.Fatalf("PNG 应原样复制：%v", err)
	}
	data, err = assetPNG(pngPath, model.Asset{Extension: "png", Rotation: 90})
	if err != nil {
		t.Fatal(err)
	}
	if rotated, err := png.Decode(bytes.NewReader(data)); err != nil || rotated.Bounds().Dx() != 2 || rotated.Bounds().Dy() != 4 {
		t.Fatalf("旋转 90° 后应是 2×4：%v", err)
	}
	jpegPath := filepath.Join(dir, "b.jpg")
	f, err := os.Create(jpegPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatal(err)
	}
	f.Close()
	data, err = assetPNG(jpegPath, model.Asset{Extension: "jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("JPEG 没有转成 PNG：%v", err)
	}
}

func TestClipboardImageURLs(t *testing.T) {
	if got := clipboardImageURLs(" https://a.com/1.jpg\n\nhttp://b.com/2.png \n"); len(got) != 2 || got[1] != "http://b.com/2.png" {
		t.Fatalf("链接没取对：%v", got)
	}
	for _, text := range []string{"", "随便一段文字", "https://a.com/1.jpg\n这一行不是链接", "file:///Users/x.png", "https://a.com/x y.jpg"} {
		if got := clipboardImageURLs(text); got != nil {
			t.Fatalf("%q 不该当成链接导入：%v", text, got)
		}
	}
}
