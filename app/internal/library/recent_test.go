package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestRecentKeepsOrderNoLimitAndPersistence(t *testing.T) {
	r := NewRecentStore(t.TempDir())
	entries, err := r.List()
	if err != nil || len(entries) != 0 {
		t.Fatalf("初始: %v %v", entries, err)
	}
	if _, ok := MostRecent(entries); ok {
		t.Fatal("空列表返回了最近图库")
	}
	base := time.Now().UTC().Add(-time.Hour)
	for i := range 12 {
		entry := Recent{Path: filepath.Join(t.TempDir(), fmt.Sprintf("%d.momento", i)), LibraryID: fmt.Sprint(i), DisplayName: fmt.Sprint(i), LastOpened: base.Add(time.Duration(i) * time.Minute)}
		if err := r.Add(entry); err != nil {
			t.Fatal(err)
		}
	}
	entries, err = r.List()
	if err != nil {
		t.Fatal(err)
	}
	// 不设上限，新图库按加入顺序排在末尾，第 11、12 个不会挤掉前面的。
	if len(entries) != 12 || entries[0].LibraryID != "0" || entries[11].LibraryID != "11" {
		t.Fatal(entries)
	}
	paths := make([]string, len(entries))
	for i, entry := range entries {
		paths[i] = entry.Path
	}
	slices.Reverse(paths)
	if err := r.Reorder(paths); err != nil {
		t.Fatal(err)
	}
	// 再次打开已有图库只更新名称和时间，用户拖好的位置不变。
	chosen := Recent{Path: paths[5], LibraryID: "6", DisplayName: "改名", Icon: "tree", LastOpened: time.Now().UTC()}
	if err := r.Add(chosen); err != nil {
		t.Fatal(err)
	}
	reopened := NewRecentStore(filepath.Dir(r.Path))
	entries, err = reopened.List()
	if err != nil || len(entries) != 12 || entries[5] != chosen || entries[0].Path != paths[0] || entries[11].Path != paths[11] {
		t.Fatal(entries, err)
	}
	if latest, ok := MostRecent(entries); !ok || latest != chosen {
		t.Fatal("最近打开的不是刚打开的那个", latest)
	}
	if err := r.Remove(chosen.Path); err != nil {
		t.Fatal(err)
	}
	entries, err = r.List()
	if err != nil || len(entries) != 11 {
		t.Fatal(entries, err)
	}
	if latest, _ := MostRecent(entries); latest.LibraryID != "11" {
		t.Fatal("移除后最近图库不对", latest)
	}
}

func TestMissingLibraryStaysAndCanBeRelocated(t *testing.T) {
	dir := t.TempDir()
	r := NewRecentStore(t.TempDir())
	l, err := Create(t.Context(), filepath.Join(dir, "外接盘.momento"), "外接盘")
	if err != nil {
		t.Fatal(err)
	}
	m := l.Manifest
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	other := newTestLibrary(t)
	first := Recent{Path: filepath.Join(t.TempDir(), "a.momento"), LibraryID: "a", DisplayName: "a"}
	moved := Recent{Path: l.Root, LibraryID: m.LibraryID, DisplayName: m.DisplayName}
	for _, entry := range []Recent{first, moved, {Path: other.Root, LibraryID: other.Manifest.LibraryID, DisplayName: "other"}} {
		if err := r.Add(entry); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckLibraryPath(l.Root); err != nil {
		t.Fatal(err)
	}
	// 图库被挪走（外接盘没插）：检查报错，但列表里的记录还在。
	newRoot := filepath.Join(t.TempDir(), "新位置.momento")
	if err := os.Rename(l.Root, newRoot); err != nil {
		t.Fatal(err)
	}
	if err := CheckLibraryPath(l.Root); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatal("找不到的图库没报错", err)
	}
	if _, err := Open(t.Context(), l.Root); err == nil {
		t.Fatal("打开了不存在的图库")
	}
	before, err := os.ReadFile(r.Path)
	if err != nil {
		t.Fatal(err)
	}
	// 选了别的图库：报错，记录不变。
	if _, err := r.Relocate(l.Root, other.Root); err == nil {
		t.Fatal("接受了不同的图库")
	}
	if _, err := r.Relocate(filepath.Join(dir, "不在列表.momento"), newRoot); err == nil {
		t.Fatal("接受了列表里没有的记录")
	}
	if after, err := os.ReadFile(r.Path); err != nil || string(after) != string(before) {
		t.Fatal("失败的重新定位改了记录", err)
	}
	resolved, err := filepath.EvalSymlinks(newRoot)
	if err != nil {
		t.Fatal(err)
	}
	// 用户先双击打开过新位置，列表里多了一条；重新定位后合并成原位置那一条。
	if err := r.Add(Recent{Path: resolved, LibraryID: m.LibraryID, DisplayName: m.DisplayName}); err != nil {
		t.Fatal(err)
	}
	entry, err := r.Relocate(l.Root, newRoot)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := r.List()
	if err != nil || len(entries) != 3 || entries[1] != entry || entry.Path != resolved || entry.LibraryID != m.LibraryID {
		t.Fatal(entries, err)
	}
	if err := CheckLibraryPath(entry.Path); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidRecentDataAndReorderAreNotOverwritten(t *testing.T) {
	r := NewRecentStore(t.TempDir())
	if err := r.Add(Recent{Path: filepath.Join(t.TempDir(), "one.momento")}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(r.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, paths := range [][]string{nil, {"unknown"}, {"unknown", "unknown"}} {
		if err := r.Reorder(paths); err == nil {
			t.Fatal("接受无效排序")
		}
		after, err := os.ReadFile(r.Path)
		if err != nil || string(after) != string(before) {
			t.Fatal("无效排序改变记录")
		}
	}
	if err := os.WriteFile(r.Path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.List(); err == nil {
		t.Fatal("损坏配置未报错")
	}
	if err := r.Add(Recent{Path: filepath.Join(t.TempDir(), "two.momento")}); err == nil {
		t.Fatal("损坏配置被覆盖")
	}
	after, err := os.ReadFile(r.Path)
	if err != nil || string(after) != "broken" {
		t.Fatal("配置丢失")
	}
}

func TestRecentConcurrentUpdates(t *testing.T) {
	r := NewRecentStore(t.TempDir())
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			if err := r.Add(Recent{Path: filepath.Join(filepath.Dir(r.Path), fmt.Sprintf("%d.momento", i))}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	entries, err := r.List()
	if err != nil || len(entries) != 8 {
		t.Fatal(entries, err)
	}
}
