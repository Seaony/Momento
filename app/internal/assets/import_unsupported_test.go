package assets

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestImportOnlyUnsupportedFilesReportsLocalizedError(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	notes := filepath.Join(input, "notes.txt")
	// 访达替身是没有扩展名的普通文件，纯 Go 解析不了，按不支持的文件算。
	alias := filepath.Join(input, "Photos 替身")
	folder := filepath.Join(input, "Docs")
	for _, path := range []string{notes, alias, filepath.Join(folder, "a.psd"), filepath.Join(folder, "b.pdf")} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := s.Import(ctx, model.ImportRequest{Paths: []string{notes, alias, folder}}, nil)
	var localized *i18n.LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_import_unsupported" {
		t.Fatalf("全是不支持的文件时没报格式错误：%v", err)
	}
	if localized.Args[0] != 4 || localized.Args[1] != "notes.txt" {
		t.Fatalf("数量或示例文件名不对：%v", localized.Args)
	}
	if len(snapshot(t, ctx, lib).Assets) != 0 {
		t.Fatal("不支持的文件被导入")
	}
	// 空文件夹不是格式问题，照旧算导入 0 个。
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{t.TempDir()}}, nil)
	if err != nil || result.Imported != 0 || len(result.Errors) != 0 {
		t.Fatalf("空文件夹：%+v %v", result, err)
	}
}

func TestImportListsDirectlyDroppedUnsupportedFilesWithImages(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	image := fixtureCopy(t, "png", filepath.Join(input, "a.png"))
	notes := filepath.Join(input, "notes.txt")
	if err := os.WriteFile(notes, []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	// 文件夹里混着的非图片文件不逐个报，只有直接拖进来的才报。
	folder := filepath.Join(input, "Mixed")
	fixtureCopy(t, "jpg", filepath.Join(folder, "b.jpg"))
	if err := os.WriteFile(filepath.Join(folder, "readme.md"), []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{image, notes, folder}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 2 || len(result.Errors) != 1 || len(result.Failed) != 1 || result.Failed[0] != notes {
		t.Fatalf("导入结果不对：%+v", result)
	}
	var localized *i18n.LocalizedError
	if !errors.As(result.Errors[0], &localized) || localized.Key != "error_import_unsupported_file" || localized.Args[0] != "notes.txt" {
		t.Fatalf("不支持文件的错误不对：%v", result.Errors[0])
	}
	if len(snapshot(t, ctx, lib).Assets) != 2 {
		t.Fatal("图片没有导入")
	}
}

func TestImportFollowsDirectorySymlinksWithoutLooping(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 建符号链接需要额外权限")
	}
	ctx, lib, s := testLibrary(t)
	base := t.TempDir()
	target := filepath.Join(base, "Real")
	fixtureCopy(t, "png", filepath.Join(target, "a.png"))
	fixtureCopy(t, "jpg", filepath.Join(target, "Sub", "b.jpg"))
	// 子文件夹里放一个指回上级的链接，扫描不能绕圈。
	if err := os.Symlink(target, filepath.Join(target, "Sub", "loop")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "Outside")
	fixtureCopy(t, "gif", filepath.Join(outside, "c.gif"))
	if err := os.Symlink(outside, filepath.Join(target, "Linked")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "Link to Real")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{link}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 3 || len(result.Errors) != 0 {
		t.Fatalf("链接指向的文件夹没有按文件夹导入：%+v", result)
	}
	names := map[string]bool{}
	for _, folder := range snapshot(t, ctx, lib).Folders {
		names[folder.Name] = true
	}
	if names["Link to Real"] || !names["Sub"] || !names["Linked"] || names["loop"] {
		t.Fatalf("文件夹结构不对：%v", names)
	}
}
