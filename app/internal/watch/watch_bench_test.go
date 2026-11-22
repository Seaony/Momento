package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

func BenchmarkWatchIdle100000(b *testing.B) {
	base := b.TempDir()
	root := filepath.Join(base, "Library.momento")
	incoming := filepath.Join(base, "Incoming")
	if err := os.Mkdir(root, 0700); err != nil {
		b.Fatal(err)
	}
	for dirIndex := range 100 {
		dir := filepath.Join(incoming, fmt.Sprint(dirIndex))
		if err := os.MkdirAll(dir, 0700); err != nil {
			b.Fatal(err)
		}
		for fileIndex := range 1000 {
			path := filepath.Join(dir, fmt.Sprintf("%04d.png", fileIndex))
			if err := os.WriteFile(path, nil, 0600); err != nil {
				b.Fatal(err)
			}
		}
	}
	watcher := New(root, nil, nil, nil, nil)
	watcher.Update([]Rule{{Path: incoming, Enabled: true}})
	ctx := context.Background()
	watcher.Scan(ctx)
	for key, state := range watcher.files {
		state.attempted = true
		watcher.files[key] = state
	}
	for b.Loop() {
		watcher.Scan(ctx)
	}
}

func BenchmarkWatchCheckResult100000(b *testing.B) {
	paths := make([]string, 100000)
	errs := make([]error, len(paths))
	for index := range paths {
		paths[index] = fmt.Sprintf("/incoming/%05d.png", index)
		errs[index] = ErrNotImported
	}
	result := model.ImportResult{Failed: paths, Errors: errs}
	failed := make(map[string]error, len(paths))
	for b.Loop() {
		if err := checkResult(result, paths, failed); err != nil {
			b.Fatal(err)
		}
	}
}
