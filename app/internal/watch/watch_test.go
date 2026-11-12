package watch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

func fixture(t *testing.T) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "Library.momento")
	folder := filepath.Join(base, "Incoming")
	for _, path := range []string{root, folder} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(folder, "image.png")
	if err := os.WriteFile(path, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	return root, folder, path
}

func TestWatcherWaitsForStableImageAndTrashesAfterImport(t *testing.T) {
	root, folder, path := fixture(t)
	var calls, trashed int
	var events []Event
	w := New(root, func(ctx context.Context, source, destination string) (model.ImportResult, error) {
		calls++
		expected, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		if source != expected || destination != "folder" || trashed != 0 {
			t.Fatal("导入目标或执行顺序不正确")
		}
		return model.ImportResult{Imported: 1}, nil
	}, func(source string) error { trashed++; return os.Remove(source) }, func(e Event) { events = append(events, e) })
	w.Update([]Rule{{Path: folder, FolderID: "folder", Enabled: true}})
	w.Scan(context.Background())
	if calls != 0 || trashed != 0 {
		t.Fatal("未稳定就导入")
	}
	if err := os.WriteFile(path, []byte("still writing"), 0600); err != nil {
		t.Fatal(err)
	}
	w.Scan(context.Background())
	if calls != 0 {
		t.Fatal("仍在写入就导入")
	}
	w.Scan(context.Background())
	w.Scan(context.Background())
	if calls != 1 || trashed != 1 || len(events) != 1 || events[0].Err != nil {
		t.Fatalf("导入结果错误: %d/%d %+v", calls, trashed, events)
	}
}

func TestWatcherRetainsSourcesOnFailureAndRetriesWhenReenabled(t *testing.T) {
	for _, failure := range []string{"import", "result", "empty", "trash", "changed"} {
		t.Run(failure, func(t *testing.T) {
			root, folder, path := fixture(t)
			var calls, trashed int
			var events []Event
			problem := errors.New("blocked")
			w := New(root, func(context.Context, string, string) (model.ImportResult, error) {
				calls++
				switch failure {
				case "import":
					return model.ImportResult{}, problem
				case "result":
					return model.ImportResult{Imported: 1, Errors: []error{problem}}, nil
				case "empty":
					return model.ImportResult{}, nil
				case "changed":
					if err := os.WriteFile(path, []byte("different content"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				return model.ImportResult{Imported: 1}, nil
			}, func(string) error { trashed++; return problem }, func(e Event) { events = append(events, e) })
			w.Update([]Rule{{Path: folder, Enabled: true}})
			w.Scan(context.Background())
			w.Scan(context.Background())
			if _, err := os.Stat(path); err != nil {
				t.Fatal("失败后源文件丢失")
			}
			if calls != 1 || len(events) != 1 || events[0].Err == nil {
				t.Fatalf("没有报告失败: %d %+v", calls, events)
			}
			if failure != "trash" && trashed != 0 {
				t.Fatal("失败后仍调用了废纸篓")
			}
			if failure == "changed" {
				return
			}
			w.Scan(context.Background())
			if calls != 1 {
				t.Fatal("失败文件没有变化却重复处理")
			}
			w.Update(nil)
			w.Scan(context.Background())
			w.Update([]Rule{{Path: folder, Enabled: true}})
			w.Scan(context.Background())
			w.Scan(context.Background())
			if calls != 2 {
				t.Fatal("重新启用后没有重试")
			}
		})
	}
}

func TestWatcherRetriesBusyAndAllowsConfirmedDuplicate(t *testing.T) {
	root, folder, path := fixture(t)
	calls, trashed := 0, 0
	w := New(root, func(context.Context, string, string) (model.ImportResult, error) {
		calls++
		if calls == 1 {
			return model.ImportResult{}, assets.ErrImportBusy
		}
		return model.ImportResult{Skipped: 1}, nil
	}, func(source string) error { trashed++; return os.Remove(source) }, nil)
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	w.Scan(context.Background())
	if calls != 2 || trashed != 1 {
		t.Fatalf("繁忙重试或重复项处理错误: %d/%d", calls, trashed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestWatcherRejectsLibraryPathsAndReportsMissingFolder(t *testing.T) {
	root, folder, _ := fixture(t)
	child := filepath.Join(root, "assets")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{root, child, filepath.Dir(root), alias} {
		if err := ValidatePath(root, path); !errors.Is(err, ErrLibraryPath) {
			t.Fatalf("允许监视图库自身或祖先: %s %v", path, err)
		}
	}
	other := filepath.Join(t.TempDir(), "Other.momento", "assets")
	if err := os.MkdirAll(other, 0700); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePath(root, other); !errors.Is(err, ErrLibraryPath) {
		t.Fatal("允许监视其他图库内部")
	}
	if err := ValidatePath(root, folder); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(folder, "missing")
	var events []Event
	w := New(root, func(context.Context, string, string) (model.ImportResult, error) {
		t.Fatal("不应导入")
		return model.ImportResult{}, nil
	}, func(string) error { t.Fatal("不应删除"); return nil }, func(e Event) { events = append(events, e) })
	w.Update([]Rule{{Path: missing, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if len(events) != 1 || !os.IsNotExist(events[0].Err) {
		t.Fatalf("不存在目录没有显示一次错误: %+v", events)
	}
}

func TestWatcherSkipsHiddenLinksLibrariesAndOverlappingRules(t *testing.T) {
	root, folder, _ := fixture(t)
	for _, name := range []string{".hidden.png", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	nested := filepath.Join(folder, "sub")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(folder, "image.png"), filepath.Join(folder, "link.png")); err != nil {
		t.Fatal(err)
	}
	own := filepath.Join(folder, "Other.momento")
	if err := os.Mkdir(own, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(own, "inside.png"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	var imported []string
	var events []Event
	w := New(root, func(_ context.Context, path, _ string) (model.ImportResult, error) {
		imported = append(imported, path)
		return model.ImportResult{Imported: 1}, nil
	}, os.Remove, func(e Event) { events = append(events, e) })
	w.Update([]Rule{{Path: folder, Enabled: true}, {Path: nested, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if len(imported) != 1 || filepath.Base(imported[0]) != "image.png" {
		t.Fatalf("导入了应跳过的文件: %v", imported)
	}
	if len(events) != 2 || !errors.Is(events[0].Err, ErrOverlappingFolders) {
		t.Fatalf("重叠目录没有报错: %+v", events)
	}
}

func TestWatcherRealImportCommitsBeforeRemovingSource(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Library.momento"), "Library")
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	service := assets.New(lib.Root, lib.DB)
	folder := t.TempDir()
	path := filepath.Join(folder, "sample.png")
	data, err := os.ReadFile("../assets/testdata/sample.png")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var count int
	w := New(lib.Root, func(ctx context.Context, path, folder string) (model.ImportResult, error) {
		return service.Import(ctx, model.ImportRequest{Paths: []string{path}, FolderID: folder}, nil)
	}, func(source string) error {
		snapshot, err := lib.Snapshot(ctx)
		if err != nil {
			return err
		}
		if len(snapshot.Assets) != 1 {
			t.Fatal("提交之前删除了源文件")
		}
		stored, err := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(snapshot.Assets[0].RelativePath)))
		if err != nil {
			return err
		}
		if string(stored) != string(data) {
			t.Fatal("图库副本不完整")
		}
		count++
		return os.Remove(source)
	}, nil)
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(ctx)
	w.Scan(ctx)
	if count != 1 {
		t.Fatal("没有完成真实导入")
	}
}

func TestWatcherRunStopsOnCancellation(t *testing.T) {
	root, folder, _ := fixture(t)
	w := New(root, nil, nil, nil)
	w.Update([]Rule{{Path: folder, Enabled: true}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("监视器取消后没有退出")
	}
}

func TestWatcherRetainsSourceWhenRealTransactionCannotCommit(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Library.momento"), "Library")
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	service := assets.New(lib.Root, lib.DB)
	if _, err := lib.DB.ExecContext(ctx, `CREATE TRIGGER refuse_import BEFORE INSERT ON assets BEGIN SELECT RAISE(ABORT,'test rejected import'); END`); err != nil {
		t.Fatal(err)
	}
	folder := t.TempDir()
	path := filepath.Join(folder, "sample.png")
	data, err := os.ReadFile("../assets/testdata/sample.png")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	var events []Event
	w := New(lib.Root, func(ctx context.Context, path, folder string) (model.ImportResult, error) {
		return service.Import(ctx, model.ImportRequest{Paths: []string{path}, FolderID: folder}, nil)
	}, func(string) error { t.Fatal("失败事务调用了废纸篓"); return nil }, func(e Event) { events = append(events, e) })
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(ctx)
	w.Scan(ctx)
	if len(events) != 1 || events[0].Err == nil {
		t.Fatal("失败没有上报")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("失败事务丢失了源文件")
	}
	snapshot, err := lib.Snapshot(ctx)
	if err != nil || len(snapshot.Assets) != 0 {
		t.Fatalf("失败事务留下了记录: %+v %v", snapshot, err)
	}
}

func TestWatcherConfigurationCanChangeWhileScanning(t *testing.T) {
	root, folder, _ := fixture(t)
	w := New(root, func(context.Context, string, string) (model.ImportResult, error) {
		return model.ImportResult{}, assets.ErrImportBusy
	}, func(string) error { t.Fatal("繁忙时不能删除"); return nil }, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 50 {
			w.Update([]Rule{{Path: folder, Enabled: true}})
			w.Update(nil)
		}
	}()
	for range 50 {
		w.Scan(context.Background())
	}
	<-done
}
