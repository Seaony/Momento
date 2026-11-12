package library

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestRecentLimitDedupeOrderAndPersistence(t *testing.T) {
	r := NewRecentStore(t.TempDir())
	entries, err := r.List()
	if err != nil || len(entries) != 0 {
		t.Fatalf("初始: %v %v", entries, err)
	}
	for i := range 12 {
		entry := Recent{Path: filepath.Join(t.TempDir(), fmt.Sprintf("%d.momento", i)), LibraryID: fmt.Sprint(i), DisplayName: fmt.Sprint(i)}
		if err := r.Add(entry); err != nil {
			t.Fatal(err)
		}
	}
	entries, err = r.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 10 || entries[0].LibraryID != "11" || entries[9].LibraryID != "2" || entries[0].LastOpened.IsZero() {
		t.Fatal(entries)
	}
	chosen := entries[5]
	chosen.DisplayName = "改名"
	chosen.LastOpened = time.Now().UTC()
	if err := r.Add(chosen); err != nil {
		t.Fatal(err)
	}
	entries, err = r.List()
	if err != nil || len(entries) != 10 || entries[0] != chosen {
		t.Fatal(entries, err)
	}
	paths := make([]string, len(entries))
	for i, entry := range entries {
		paths[i] = entry.Path
	}
	slices.Reverse(paths)
	if err := r.Reorder(paths); err != nil {
		t.Fatal(err)
	}
	reopened := NewRecentStore(filepath.Dir(r.Path))
	entries, err = reopened.List()
	if err != nil || entries[0].Path != paths[0] || entries[len(entries)-1].Path != chosen.Path {
		t.Fatal(entries, err)
	}
	if err := r.Remove(chosen.Path); err != nil {
		t.Fatal(err)
	}
	entries, err = r.List()
	if err != nil || len(entries) != 9 {
		t.Fatal(entries, err)
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
