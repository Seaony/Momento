package ui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
)

// 存储对话框里确认了替换，导出覆盖同名图库包，旧的那份移到回收站；没确认时仍报已存在。
func TestExportLibraryReplacesConfirmedDestination(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Source.momento"), "Source")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lib.Close() })
	destination := filepath.Join(t.TempDir(), "Copy.momento")
	if err := os.MkdirAll(destination, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "old.txt"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := exportLibrary(ctx, lib, destination, false); err == nil {
		t.Fatal("没确认替换却覆盖了已有目标")
	}
	previous := trashItem
	t.Cleanup(func() { trashItem = previous })
	var trashed []string
	trashItem = func(path string) error {
		trashed = append(trashed, path)
		return os.RemoveAll(path)
	}
	if err := exportLibrary(ctx, lib, destination, true); err != nil {
		t.Fatal(err)
	}
	if len(trashed) != 1 {
		t.Fatalf("旧目标没有交给回收站：%v", trashed)
	}
	if _, err := os.Stat(filepath.Join(destination, "manifest.json")); err != nil {
		t.Fatal("替换后目标不是新导出的图库")
	}
	if _, err := os.Stat(filepath.Join(destination, "old.txt")); !os.IsNotExist(err) {
		t.Fatal("旧目标的内容还在")
	}
}
