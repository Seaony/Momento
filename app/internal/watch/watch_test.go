package watch

import (
	"context"
	"errors"
	"image"
	"io"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

func newTestWatcher(root string, importFile ImportFunc, trash func(string) error, event func(Event)) *Watcher {
	return New(root, importFile, trash, func(_ context.Context, events []Event) {
		if event != nil {
			for _, item := range events {
				event(item)
			}
		}
	}, func(_ context.Context, paths []string, _ map[string]string) (map[string]AssetStatus, error) {
		statuses := make(map[string]AssetStatus, len(paths))
		for _, path := range paths {
			statuses[path] = AssetStatus{Active: true}
		}
		return statuses, nil
	})
}

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
	w := newTestWatcher(root, func(ctx context.Context, sources []string, destination string) (model.ImportResult, error) {
		calls++
		expected, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(sources) != 1 || sources[0] != expected || destination != "folder" || trashed != 0 {
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
	for _, failure := range []string{"import", "result", "mismatch", "empty", "trash", "changed"} {
		t.Run(failure, func(t *testing.T) {
			root, folder, path := fixture(t)
			var calls, trashed int
			var events []Event
			problem := errors.New("blocked")
			w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
				calls++
				switch failure {
				case "import":
					return model.ImportResult{}, problem
				case "result":
					return model.ImportResult{Errors: []error{problem}, Failed: sources}, nil
				case "mismatch":
					// 有错误却没说是哪个文件：分不清，整批保留。
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
			expected := 2
			if failure == "trash" {
				expected = 1
			}
			if calls != expected {
				t.Fatal("重新启用后没有重试")
			}
		})
	}
}

func TestWatcherRetriesBusyAndAllowsConfirmedDuplicate(t *testing.T) {
	root, folder, path := fixture(t)
	calls, trashed := 0, 0
	w := newTestWatcher(root, func(context.Context, []string, string) (model.ImportResult, error) {
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
	w := newTestWatcher(root, func(context.Context, []string, string) (model.ImportResult, error) {
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
	w := newTestWatcher(root, func(_ context.Context, paths []string, _ string) (model.ImportResult, error) {
		imported = append(imported, paths...)
		return model.ImportResult{Imported: len(paths)}, nil
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

func TestWatcherFindsNewFilesUnderCachedParent(t *testing.T) {
	root, folder, _ := fixture(t)
	nested := filepath.Join(folder, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	var imported []string
	watcher := New(root, func(_ context.Context, paths []string, _ string) (model.ImportResult, error) {
		imported = append(imported, paths...)
		return model.ImportResult{Imported: len(paths)}, nil
	}, os.Remove, nil, nil)
	clk := &clock{t: time.Now()}
	watcher.now = clk.now
	watcher.Update([]Rule{{Path: folder, Enabled: true}})
	ctx := context.Background()
	watcher.Scan(ctx)
	subdir, err := os.Stat(nested)
	if err != nil {
		t.Fatal(err)
	}
	incoming := writeFiles(t, nested, "new.png")
	changed := subdir.ModTime().Add(time.Second)
	if err := os.Chtimes(nested, changed, changed); err != nil {
		t.Fatal(err)
	}
	watcher.Scan(ctx)
	watcher.Scan(ctx)
	if !slices.Contains(imported, incoming[0]) {
		t.Fatalf("父目录没有变化时漏扫了子目录的新文件：%v", imported)
	}

	subdir, err = os.Stat(nested)
	if err != nil {
		t.Fatal(err)
	}
	watcher.Scan(ctx)
	incoming = writeFiles(t, nested, "same-time.png")
	if err := os.Chtimes(nested, subdir.ModTime(), subdir.ModTime()); err != nil {
		t.Fatal(err)
	}
	clk.t = clk.t.Add(time.Minute)
	watcher.Scan(ctx)
	watcher.Scan(ctx)
	if !slices.Contains(imported, incoming[0]) {
		t.Fatalf("重读目录后仍漏掉修改时间不变时的新文件：%v", imported)
	}
}

func TestWatcherKeepsStatesForUnchangedRules(t *testing.T) {
	root, folder, _ := fixture(t)
	watcher := New(root, nil, os.Remove, nil, nil)
	clk := &clock{t: time.Now()}
	watcher.now = clk.now
	watcher.Update([]Rule{{Path: folder, FolderID: "kept", Enabled: true}})
	watcher.Scan(context.Background())
	realFolder, err := filepath.EvalSymlinks(folder)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(realFolder, "image.png")
	state := watcher.files[path]
	state.failures, state.retryAt, state.attempted = 3, clk.now().Add(time.Hour), true
	watcher.files[path] = state
	watcher.Update([]Rule{{Path: folder, FolderID: "kept", Enabled: true}, {Path: filepath.Join(filepath.Dir(folder), "changed"), Enabled: true}})
	watcher.Scan(context.Background())
	kept, exists := watcher.files[path]
	if !exists || kept.failures != 3 || !kept.retryAt.Equal(state.retryAt) || !kept.attempted {
		t.Fatalf("未变化规则的状态没有保留：%+v %v", kept, exists)
	}
	watcher.Update([]Rule{{Path: filepath.Join(filepath.Dir(folder), "changed"), Enabled: true}})
	watcher.Scan(context.Background())
	if _, exists := watcher.files[path]; exists {
		t.Fatal("被删规则的状态没有清掉")
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
	w := newTestWatcher(lib.Root, func(ctx context.Context, paths []string, folder string) (model.ImportResult, error) {
		return service.Import(ctx, model.ImportRequest{Paths: paths, FolderID: folder}, nil)
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
	w := New(root, nil, nil, nil, nil)
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
	w := newTestWatcher(lib.Root, func(ctx context.Context, paths []string, folder string) (model.ImportResult, error) {
		return service.Import(ctx, model.ImportRequest{Paths: paths, FolderID: folder}, nil)
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
	w := newTestWatcher(root, func(context.Context, []string, string) (model.ImportResult, error) {
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

// 在 folder 里写几个文件，返回解析过符号链接的绝对路径（和监视器看到的一致）。
func writeFiles(t *testing.T, folder string, names ...string) []string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(folder)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(folder, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, filepath.Join(resolved, name))
	}
	return paths
}

func TestWatcherBatchKeepsOnlyFailedFiles(t *testing.T) {
	root, folder, _ := fixture(t)
	if err := os.Remove(filepath.Join(folder, "image.png")); err != nil {
		t.Fatal(err)
	}
	paths := writeFiles(t, folder, "a.png", "b.png", "c.png")
	problem := errors.New("坏图")
	var calls [][]string
	var trashed []string
	var events []Event
	var batches int
	w := New(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		calls = append(calls, slices.Clone(sources))
		// a 是新图，c 是图库里已有的重复图，b 解不开。
		return model.ImportResult{Imported: 1, Skipped: 1, Errors: []error{problem}, Failed: []string{paths[1]}}, nil
	}, func(source string) error { trashed = append(trashed, source); return os.Remove(source) }, func(_ context.Context, batch []Event) {
		batches++
		events = append(events, batch...)
	}, nil)
	w.Update([]Rule{{Path: folder, FolderID: "f", Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if len(calls) != 1 || !slices.Equal(calls[0], paths) {
		t.Fatalf("没有合成一次导入：%v", calls)
	}
	if !slices.Equal(trashed, []string{paths[0], paths[2]}) {
		t.Fatalf("移走的文件不对：%v", trashed)
	}
	if _, err := os.Stat(paths[1]); err != nil {
		t.Fatal("失败的文件被移走了")
	}
	// 成功的先报、失败的最后报，按规则记错误的界面最后留下的是错误。
	if batches != 1 || len(events) != 3 || events[0].Err != nil || events[1].Err != nil || events[2].Path != paths[1] || !errors.Is(events[2].Err, problem) {
		t.Fatalf("事件不对：%+v", events)
	}
	for _, event := range events {
		if event.RulePath != folder {
			t.Fatalf("没带上规则原始路径：%+v", event)
		}
	}
	w.Scan(context.Background())
	if len(calls) != 1 {
		t.Fatal("失败文件没有变化却重复导入")
	}
}

func TestWatcherBatchErrorKeepsEveryFile(t *testing.T) {
	root, folder, _ := fixture(t)
	paths := writeFiles(t, folder, "a.png", "b.png")
	problem := errors.New("数据库写不进去")
	calls := 0
	var events []Event
	w := newTestWatcher(root, func(context.Context, []string, string) (model.ImportResult, error) {
		calls++
		return model.ImportResult{}, problem
	}, func(string) error { t.Fatal("整批失败时不能移走源文件"); return nil }, func(e Event) { events = append(events, e) })
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if calls != 1 || len(events) != 3 {
		t.Fatalf("导入 %d 次，事件 %+v", calls, events)
	}
	for _, event := range events {
		if !errors.Is(event.Err, problem) {
			t.Fatalf("每个文件都应报整批的错误：%+v", event)
		}
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("整批失败丢了源文件")
		}
	}
}

func TestWatcherGroupsBatchesByTargetFolder(t *testing.T) {
	root, _, _ := fixture(t)
	base := t.TempDir()
	var folders []string
	for _, name := range []string{"One", "Two", "Three"} {
		folder := filepath.Join(base, name)
		if err := os.Mkdir(folder, 0700); err != nil {
			t.Fatal(err)
		}
		folders = append(folders, folder)
	}
	one := writeFiles(t, folders[0], "a.png", "b.png")
	two := writeFiles(t, folders[1], "c.png")
	three := writeFiles(t, folders[2], "d.png")
	calls := map[string][]string{}
	var order []string
	w := newTestWatcher(root, func(_ context.Context, sources []string, folderID string) (model.ImportResult, error) {
		if _, ok := calls[folderID]; ok {
			t.Fatalf("同一目标文件夹导入了两次：%s", folderID)
		}
		order = append(order, folderID)
		calls[folderID] = slices.Clone(sources)
		return model.ImportResult{Imported: len(sources)}, nil
	}, os.Remove, nil)
	// 一、三两个规则导入到同一个文件夹，应合成一批。
	w.Update([]Rule{{Path: folders[0], FolderID: "x", Enabled: true}, {Path: folders[1], FolderID: "y", Enabled: true}, {Path: folders[2], FolderID: "x", Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if !slices.Equal(order, []string{"x", "y"}) || !slices.Equal(calls["x"], append(one, three...)) || !slices.Equal(calls["y"], two) {
		t.Fatalf("分组不对：%v %v", order, calls)
	}
	for _, path := range append(append(one, two...), three...) {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("导入成功的源文件没有移走：%s", path)
		}
	}
}

// 实况照片：图片和同名视频导成一个素材，视频记在 Motion 里，两个源文件都移走。
func TestWatcherTrashesLivePhotoMotion(t *testing.T) {
	root, folder, _ := fixture(t)
	if err := os.Remove(filepath.Join(folder, "image.png")); err != nil {
		t.Fatal(err)
	}
	paths := writeFiles(t, folder, "IMG_1.HEIC", "IMG_1.MOV")
	var trashed []string
	w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		return model.ImportResult{Imported: 1, Motion: []string{paths[1]}}, nil
	}, func(source string) error { trashed = append(trashed, source); return os.Remove(source) }, func(Event) {})
	w.Update([]Rule{{Path: folder, FolderID: "f", Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if !slices.Equal(trashed, paths) {
		t.Fatalf("实况照片的两个源文件应都移走：%v", trashed)
	}
}

// 给测试用的可调时钟。
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func TestWatcherRetriesTemporaryImportFailureWithBackoff(t *testing.T) {
	root, folder, path := fixture(t)
	clk := &clock{t: time.Now()}
	calls, trashed := 0, 0
	problem := errors.New("磁盘已满")
	w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		calls++
		if calls < 3 {
			return model.ImportResult{}, problem
		}
		return model.ImportResult{Imported: len(sources)}, nil
	}, func(source string) error { trashed++; return os.Remove(source) }, nil)
	w.now = clk.now
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	if calls != 1 {
		t.Fatalf("第一次尝试次数 %d", calls)
	}
	w.Scan(context.Background())
	if calls != 1 {
		t.Fatal("退避时间内重试了")
	}
	clk.t = clk.t.Add(retryBackoff)
	w.Scan(context.Background())
	if calls != 2 {
		t.Fatalf("退避结束后没有重试：%d", calls)
	}
	clk.t = clk.t.Add(retryBackoff + time.Second)
	w.Scan(context.Background())
	if calls != 2 {
		t.Fatal("第二次失败后退避没有翻倍")
	}
	clk.t = clk.t.Add(retryBackoff)
	w.Scan(context.Background())
	if calls != 3 || trashed != 1 {
		t.Fatalf("临时障碍解除后没有导入并移源：%d/%d", calls, trashed)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("源文件还在")
	}
}

// 临时失败一直按封顶间隔重试（磁盘清出空间后能自己导进来），格式无法识别只试一次。
func TestWatcherKeepsRetryingTemporaryFailureAndStopsOnPermanentError(t *testing.T) {
	root, folder, _ := fixture(t)
	clk := &clock{t: time.Now()}
	calls := 0
	w := newTestWatcher(root, func(context.Context, []string, string) (model.ImportResult, error) {
		calls++
		return model.ImportResult{}, errors.New("没权限")
	}, func(string) error { return nil }, nil)
	w.now = clk.now
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	for i := 0; i < 10; i++ {
		w.Scan(context.Background())
		// 每轮都超过封顶间隔，每轮都该再试一次。
		clk.t = clk.t.Add(retryCeiling + time.Second)
	}
	if calls != 10 {
		t.Fatalf("临时失败应每轮重试：%d 次", calls)
	}

	root, folder, path := fixture(t)
	calls = 0
	w = newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		calls++
		return model.ImportResult{Failed: sources, Errors: []error{i18n.NewError("error_import_unsupported_file", "image.png")}}, nil
	}, func(string) error { t.Fatal("不应移源"); return nil }, nil)
	w.now = clk.now
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	for i := 0; i < 4; i++ {
		w.Scan(context.Background())
		clk.t = clk.t.Add(time.Hour)
	}
	if calls != 1 {
		t.Fatalf("格式无法识别应只尝试一次：%d", calls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("源文件丢了")
	}
}

// 入库了但移源失败：重试时只移源，不再导入。
func TestWatcherRetriesOnlyTrashAfterImportSucceeded(t *testing.T) {
	root, folder, path := fixture(t)
	clk := &clock{t: time.Now()}
	imports, trashCalls := 0, 0
	w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		imports++
		return model.ImportResult{Imported: len(sources)}, nil
	}, func(source string) error {
		trashCalls++
		if trashCalls == 1 {
			return errors.New("废纸篓不可用")
		}
		return os.Remove(source)
	}, nil)
	w.now = clk.now
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	clk.t = clk.t.Add(retryBackoff)
	w.Scan(context.Background())
	if imports != 1 || trashCalls != 2 {
		t.Fatalf("应只重试移源：导入 %d 次，移源 %d 次", imports, trashCalls)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("源文件还在")
	}
}

func TestWatcherTrashRetryChecksFileIdentity(t *testing.T) {
	root, folder, path := fixture(t)
	clk := &clock{t: time.Now()}
	imports, trashCalls := 0, 0
	w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		imports++
		return model.ImportResult{Imported: len(sources)}, nil
	}, func(string) error { trashCalls++; return errors.New("废纸篓不可用") }, nil)
	w.now = clk.now
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(context.Background())
	// 文件被换了内容：不能当作已入库的那份去移走，要重新走完整流程。
	if err := os.WriteFile(path, []byte("another content"), 0600); err != nil {
		t.Fatal(err)
	}
	clk.t = clk.t.Add(retryBackoff)
	w.Scan(context.Background())
	if trashCalls != 1 {
		t.Fatalf("文件变了还去移源：%d", trashCalls)
	}
	w.Scan(context.Background())
	if imports != 2 {
		t.Fatalf("文件变了应重新导入：%d", imports)
	}
}

func TestWatcherRechecksAssetBeforeTrashAfterRulesRestart(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  AssetStatus
		err     error
		imports int
		trashes int
	}{
		{name: "active", status: AssetStatus{Active: true}, imports: 1, trashes: 3},
		{name: "trashed", status: AssetStatus{Active: true, Trashed: true}, imports: 2, trashes: 3},
		{name: "missing", imports: 2, trashes: 3},
		{name: "lookup-failed", err: syscall.EIO, imports: 1, trashes: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, folder, original := fixture(t)
			if err := os.Remove(original); err != nil {
				t.Fatal(err)
			}
			paths := writeFiles(t, folder, "photo.HEIC", "photo.MOV")
			clk := &clock{t: time.Now()}
			imports, trashes, checks := 0, 0, 0
			var last Event
			watcher := New(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
				imports++
				if imports == 1 {
					return model.ImportResult{
						Imported: 1, Motion: []string{paths[1]},
						MotionOwners: []model.MotionOwner{{SourcePath: paths[1], AssetID: "owner"}},
					}, nil
				}
				if !slices.Equal(sources, paths[1:]) {
					t.Fatalf("应只重新导入剩下的源文件：%v", sources)
				}
				return model.ImportResult{Imported: 1}, nil
			}, func(source string) error {
				trashes++
				if trashes == 2 {
					return syscall.EPERM
				}
				return os.Remove(source)
			}, func(_ context.Context, events []Event) {
				last = events[len(events)-1]
			}, func(_ context.Context, sources []string, motionOwners map[string]string) (map[string]AssetStatus, error) {
				checks++
				if !slices.Equal(sources, paths[1:]) || motionOwners[paths[1]] != "owner" {
					t.Fatalf("规则重启后丢失实况视频归属：%v %v", sources, motionOwners)
				}
				return map[string]AssetStatus{paths[1]: test.status}, test.err
			})
			watcher.now = clk.now
			ctx := context.Background()
			rules := []Rule{{Path: folder, Enabled: true}}
			watcher.Update(rules)
			watcher.Scan(ctx)
			watcher.Scan(ctx)
			watcher.Update(nil)
			watcher.Scan(ctx)
			watcher.Update(rules)
			clk.t = clk.t.Add(retryBackoff)
			watcher.Scan(ctx)
			if checks != 1 || imports != test.imports || trashes != test.trashes || !errors.Is(last.Err, test.err) {
				t.Fatalf("状态检查 / 导入 / 移源 = %d / %d / %d，最后错误 %v", checks, imports, trashes, last.Err)
			}
		})
	}
}

func TestImageDecodeFailureRetriesOnlyForIO(t *testing.T) {
	for _, key := range []string{"error_image_unreadable", "error_decode_image"} {
		t.Run(key, func(t *testing.T) {
			params := []any(nil)
			if key == "error_decode_image" {
				params = []any{"image.png"}
			}
			for _, test := range []struct {
				cause     error
				permanent bool
			}{
				{cause: &os.PathError{Op: "read", Path: "image.png", Err: syscall.EIO}},
				{cause: syscall.EIO},
				{cause: image.ErrFormat, permanent: true},
				{cause: io.ErrUnexpectedEOF, permanent: true},
			} {
				err := i18n.WrapError(i18n.WrapError(test.cause, key, params...), "error_import_read", "image.png")
				if permanent(err) != test.permanent {
					t.Fatalf("错误 %v 的永久失败判定与预期 %v 不符", err, test.permanent)
				}
			}
		})
	}
}

func TestWatcherCancelStopsTrashingRemainingFiles(t *testing.T) {
	root, folder, _ := fixture(t)
	paths := writeFiles(t, folder, "a.png", "b.png", "c.png")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var trashed []string
	imports := 0
	w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		imports++
		return model.ImportResult{Imported: len(sources)}, nil
	}, func(source string) error {
		trashed = append(trashed, source)
		cancel() // 移第一个文件时停用规则
		return os.Remove(source)
	}, nil)
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	w.Scan(ctx)
	if len(trashed) != 1 {
		t.Fatalf("取消后还在移源：%v", trashed)
	}
	for _, path := range paths[1:] {
		if _, err := os.Stat(path); err != nil {
			t.Fatal("取消后未处理的源文件应保持原地")
		}
	}
	// 之后重新启用：已入库的不再导入，只补做移源。
	w.Scan(context.Background())
	if len(trashed) != 4 {
		t.Fatalf("重新扫描后没有补做移源：%v", trashed)
	}
	if imports != 1 {
		t.Fatalf("导入 %d 次", imports)
	}
}

// 实况视频和库里配的那段对不上：不重试，免得视频下次没有图片可配、被当成单独素材导进去再移走。
func TestWatcherKeepsLiveMotionConflict(t *testing.T) {
	root, folder, path := fixture(t)
	clk := &clock{t: time.Now()}
	calls := 0
	w := newTestWatcher(root, func(_ context.Context, sources []string, _ string) (model.ImportResult, error) {
		calls++
		return model.ImportResult{Failed: sources, Errors: []error{i18n.NewError("error_live_motion_conflict", "image.mov")}}, nil
	}, func(string) error { t.Fatal("不应移源"); return nil }, nil)
	w.now = clk.now
	w.Update([]Rule{{Path: folder, Enabled: true}})
	w.Scan(context.Background())
	for i := 0; i < 4; i++ {
		w.Scan(context.Background())
		clk.t = clk.t.Add(retryCeiling + time.Second)
	}
	if calls != 1 {
		t.Fatalf("配对冲突应只尝试一次：%d", calls)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("源文件丢了")
	}
}
