package assets_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestEightFormatsImportWithThumbnails(t *testing.T) {
	for _, extension := range []string{"jpg", "png", "gif", "webp", "tiff", "bmp", "heic", "avif"} {
		t.Run(extension, func(t *testing.T) {
			ctx := context.Background()
			lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Formats.momento"), "Formats")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := lib.Close(); err != nil {
					t.Error(err)
				}
			})
			path := filepath.Join("testdata", "sample."+extension)
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			result, err := assets.New(lib.Root, lib.DB).Import(ctx, model.ImportRequest{Paths: []string{path}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if result.Imported != 1 || result.Skipped != 0 || len(result.Errors) != 0 {
				t.Fatalf("导入结果不正确：%+v", result)
			}
			snapshot, err := lib.Snapshot(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Assets) != 1 {
				t.Fatalf("素材数 = %d，期望 1", len(snapshot.Assets))
			}
			asset := snapshot.Assets[0]
			if asset.Width != 96 || asset.Height != 64 {
				t.Fatalf("图片尺寸 = %d × %d，期望 96 × 64", asset.Width, asset.Height)
			}
			if asset.ID != fmt.Sprintf("%x", sha256.Sum256(original)) {
				t.Fatalf("素材 ID 没有使用原文件 SHA-256：%s", asset.ID)
			}
			if asset.Extension != extension {
				t.Fatalf("格式 = %q，期望 %q", asset.Extension, extension)
			}
			stored, err := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(stored, original) {
				t.Fatal("原文件内容在导入时改变")
			}
			thumbnail, err := os.Open(filepath.Join(lib.Root, "thumbnails", asset.ID+".png"))
			if err != nil {
				t.Fatal(err)
			}
			img, decodeErr := png.Decode(thumbnail)
			closeErr := thumbnail.Close()
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if closeErr != nil {
				t.Fatal(closeErr)
			}
			bounds := img.Bounds()
			if bounds.Dx() <= 0 || bounds.Dy() <= 0 || max(bounds.Dx(), bounds.Dy()) > 512 {
				t.Fatalf("缩略图尺寸不正确：%v", bounds)
			}
		})
	}
}
