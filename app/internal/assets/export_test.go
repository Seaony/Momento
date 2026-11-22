package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestExportBatchNumbersEachNameAndPreservesOccupiedPaths(t *testing.T) {
	root := t.TempDir()
	const relative = "source.jpg"
	if err := os.WriteFile(filepath.Join(root, relative), []byte("photo"), 0600); err != nil {
		t.Fatal(err)
	}
	output := t.TempDir()
	occupied := []string{"photo.jpg", "photo 3.jpg"}
	for _, name := range occupied {
		if err := os.WriteFile(filepath.Join(output, name), []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var items []model.Asset
	for _, name := range []string{"photo", "other", "photo", "other", "photo"} {
		items = append(items, model.Asset{Name: name, RelativePath: relative, Extension: "jpg"})
	}
	paths, err := New(root, nil).Export(t.Context(), items, output, model.ExportOriginal, 0)
	if err != nil {
		t.Fatal(err)
	}
	for index, path := range paths {
		paths[index] = filepath.Base(path)
	}
	if !slices.Equal(paths, []string{"photo 2.jpg", "other.jpg", "photo 4.jpg", "other 2.jpg", "photo 5.jpg"}) {
		t.Fatalf("导出编号不对：%v", paths)
	}
	for _, name := range occupied {
		if data, err := os.ReadFile(filepath.Join(output, name)); err != nil || string(data) != "keep" {
			t.Fatalf("已有文件 %s 被改动：%q %v", name, data, err)
		}
	}
}

func TestPublishLiveExportRetriesAfterMotionNameTaken(t *testing.T) {
	for _, mode := range []struct {
		name    string
		publish func(string, string) error
	}{{"hardlink", os.Link}, {"rename", os.Rename}} {
		t.Run(mode.name, func(t *testing.T) {
			dir := t.TempDir()
			imageStage := filepath.Join(dir, ".still")
			motionStage := filepath.Join(dir, ".motion")
			requested := filepath.Join(dir, "photo.jpg")
			blockedMotion := motionExportPath(requested)
			for path, content := range map[string]string{imageStage: "still", motionStage: "motion"} {
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			publish := func(source, destination string) error {
				if destination == blockedMotion {
					if err := os.WriteFile(destination, []byte("external"), 0600); err != nil {
						t.Fatal(err)
					}
					return &os.PathError{Op: "open", Path: destination, Err: os.ErrExist}
				}
				return mode.publish(source, destination)
			}
			final, number, err := publishExportToDirectory(t.Context(), requested, imageStage, motionStage, 1, publish)
			if err != nil || number != 2 || final != filepath.Join(dir, "photo 2.jpg") {
				t.Fatalf("没有换名重试：%q, %d, %v", final, number, err)
			}
			if _, err := os.Lstat(requested); !os.IsNotExist(err) {
				t.Fatalf("第一次发布留下了未配对图片：%v", err)
			}
			for path, want := range map[string]string{final: "still", motionExportPath(final): "motion", blockedMotion: "external"} {
				if data, err := os.ReadFile(path); err != nil || string(data) != want {
					t.Fatalf("%s: 实际 %q，预期 %q，错误 %v", path, data, want, err)
				}
			}
		})
	}
}

func BenchmarkBatchExportSameName(b *testing.B) {
	root := b.TempDir()
	const relative = "photo.jpg"
	if err := os.WriteFile(filepath.Join(root, relative), []byte("photo"), 0600); err != nil {
		b.Fatal(err)
	}
	service := New(root, nil)
	asset := model.Asset{Name: "photo", RelativePath: relative, Extension: "jpg"}
	for _, count := range []int{200, 100000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			items := make([]model.Asset, count)
			for index := range items {
				items[index] = asset
			}
			for b.Loop() {
				b.StopTimer()
				destination := b.TempDir()
				b.StartTimer()
				paths, err := service.Export(b.Context(), items, destination, model.ExportOriginal, 0)
				b.StopTimer()
				if err != nil || len(paths) != count {
					b.Fatalf("导出 %d 个文件，错误 %v", len(paths), err)
				}
				b.StartTimer()
			}
		})
	}
}
