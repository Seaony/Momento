package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestGeneratePortableLibraryWith14UniqueImages(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "Sample.momento")
	var output bytes.Buffer
	if err := run(ctx, []string{"-path", path}, &output); err != nil {
		t.Fatal(err)
	}
	lib, err := library.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Assets) != 14 || len(snapshot.Folders) != 0 || len(snapshot.Tags) != 0 {
		t.Fatalf("样本数量错误：%d素材，%d文件夹，%d标签", len(snapshot.Assets), len(snapshot.Folders), len(snapshot.Tags))
	}
	formats := make(map[string]int)
	paths := make([]string, 0, len(snapshot.Assets))
	hashes := make(map[string]bool)
	for _, asset := range snapshot.Assets {
		formats[asset.Extension]++
		if hashes[asset.ID] {
			t.Fatal("生成了重复哈希")
		}
		hashes[asset.ID] = true
		if !filepath.IsLocal(filepath.FromSlash(asset.RelativePath)) {
			t.Fatalf("路径不是包内相对路径：%s", asset.RelativePath)
		}
		originalPath := filepath.Join(path, filepath.FromSlash(asset.RelativePath))
		paths = append(paths, originalPath)
		data, err := os.ReadFile(originalPath)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != asset.ID || int64(len(data)) != asset.Size {
			t.Fatal("数据库哈希或大小和真实原文件不同")
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != asset.Width || img.Bounds().Dy() != asset.Height {
			t.Fatal("原图尺寸记录错误")
		}
		thumbnail, err := os.ReadFile(filepath.Join(path, "thumbnails", asset.ID+".png"))
		if err != nil {
			t.Fatal(err)
		}
		thumb, err := png.Decode(bytes.NewReader(thumbnail))
		if err != nil {
			t.Fatal(err)
		}
		if max(thumb.Bounds().Dx(), thumb.Bounds().Dy()) > 512 {
			t.Fatal("缩略图超出512")
		}
		if len(asset.Colors) == 0 {
			t.Fatal("未生成颜色索引")
		}
	}
	if formats["jpg"] != 10 || formats["png"] != 4 {
		t.Fatalf("格式分布 %v", formats)
	}
	result, err := assets.New(lib.Root, lib.DB).Import(ctx, model.ImportRequest{Paths: paths}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 0 || result.Skipped != 14 {
		t.Fatalf("重新导入不能正确去重：%+v", result)
	}
}

func TestGenerateNeverOverwritesExistingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Existing.momento")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(path, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := generate(context.Background(), path, 14, &bytes.Buffer{}); err == nil {
		t.Fatal("覆盖了已有路径")
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep" {
		t.Fatal("已有数据被修改或删除")
	}
	if _, err := os.Stat(filepath.Join(path, "manifest.json")); !os.IsNotExist(err) {
		t.Fatal("改动了已有目录")
	}
}

func TestGenerateLargeModeUsesUniquePortableJPEGs(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "Large.momento")
	const count = 211
	if err := generate(ctx, path, count, io.Discard); err != nil {
		t.Fatal(err)
	}
	lib, err := library.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Assets) != count {
		t.Fatalf("素材数量 %d，期望 %d", len(snapshot.Assets), count)
	}
	paths := make([]string, 0, count)
	seen := make(map[string]bool)
	for _, asset := range snapshot.Assets {
		if asset.Extension != "jpg" || asset.MediaType != "image/jpeg" || seen[asset.ID] {
			t.Fatalf("大库格式或唯一性错误：%+v", asset)
		}
		seen[asset.ID] = true
		if !filepath.IsLocal(filepath.FromSlash(asset.RelativePath)) {
			t.Fatalf("原图路径不在包内：%s", asset.RelativePath)
		}
		original := filepath.Join(path, filepath.FromSlash(asset.RelativePath))
		paths = append(paths, original)
		data, err := os.ReadFile(original)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != asset.ID || int64(len(data)) != asset.Size {
			t.Fatal("大库原图的哈希或大小不真实")
		}
		img, format, err := image.Decode(bytes.NewReader(data))
		if err != nil || format != "jpeg" {
			t.Fatalf("样本不是可解码 JPEG：%s，%v", format, err)
		}
		if img.Bounds().Dx() != asset.Width || img.Bounds().Dy() != asset.Height || max(asset.Width, asset.Height) > 64 {
			t.Fatal("大库样本尺寸错误")
		}
		thumbnail, err := os.ReadFile(filepath.Join(path, "thumbnails", asset.ID+".png"))
		if err != nil {
			t.Fatal(err)
		}
		thumb, err := png.Decode(bytes.NewReader(thumbnail))
		if err != nil || thumb.Bounds() != img.Bounds() {
			t.Fatalf("大库缩略图错误：%v", err)
		}
	}
	result, err := assets.New(lib.Root, lib.DB).Import(ctx, model.ImportRequest{Paths: paths}, nil)
	if err != nil || result.Imported != 0 || result.Skipped != count {
		t.Fatalf("大库重新导入不能完整去重：%+v，%v", result, err)
	}
}

func TestGenerateCountAndArgumentValidation(t *testing.T) {
	for _, args := range [][]string{
		{}, {"-path", "wrong"}, {"-path", "new.momento", "-count", "-1"}, {"-path", "new.momento", "extra"},
	} {
		if err := run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Fatalf("错误参数被接受：%v", args)
		}
	}
	path := filepath.Join(t.TempDir(), "Empty.momento")
	if err := run(context.Background(), []string{"-path", path, "-count", "0"}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	lib, err := library.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	snapshot, err := lib.Snapshot(context.Background())
	if err != nil || len(snapshot.Assets) != 0 {
		t.Fatal("没有生成合法空库")
	}
}

func TestGenerateCanceledWorkRemovesOnlyNewPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Canceled.momento")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := generate(ctx, path, 14, &bytes.Buffer{}); err == nil {
		t.Fatal("取消没有中止生成")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("取消后留下不完整图库")
	}
}

func TestGenerateCancellationAfterWritingRemovesNewPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Partial.momento")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- generate(ctx, path, 10000, io.Discard) }()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	written := false
	for !written {
		select {
		case err := <-done:
			t.Fatalf("写入前生成已经结束：%v", err)
		case <-ticker.C:
			files, err := filepath.Glob(filepath.Join(path, "assets", "*", "*"))
			if err != nil {
				t.Fatal(err)
			}
			written = len(files) > 0
		case <-timeout.C:
			cancel()
			<-done
			t.Fatal("没有观察到素材写入")
		}
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("中途取消错误：%v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("中途取消后没有结束")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("中途取消后留下不完整图库")
	}
}
