package ui

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/Seaony/Momento/app/internal/watch"
	"github.com/egoist/mygo/ui"
)

func TestAutoImportWatcherRestartsForReopenedLibrary(t *testing.T) {
	rules := []AutoImportRule{{Path: "incoming", Enabled: true}}
	state := importState{WatchCancel: func() {}, WatchLibrary: "same-id", WatchGeneration: 2, WatchRules: rules}
	if !state.watching("same-id", 2, rules) {
		t.Fatal("同一已运行配置被误判为要重启")
	}
	if state.watching("same-id", 3, rules) {
		t.Fatal("重开同ID图库后仍复用捕获旧代际的监视器")
	}
	if state.watching("other", 2, rules) {
		t.Fatal("换库后仍复用旧监视器")
	}
}

func TestAutoImportRechecksBothPartsOfLivePhotoBeforeMovingSource(t *testing.T) {
	for _, check := range []string{"active", "trashed", "missing-owner", "missing-still", "changed-video"} {
		t.Run(check, func(t *testing.T) {
			ctx := context.Background()
			lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Library.momento"), "Library")
			if err != nil {
				t.Fatal(err)
			}
			defer lib.Close()
			folder := t.TempDir()
			still, video := []byte("still data"), []byte("video data")
			ownerID := fmt.Sprintf("%x", sha256.Sum256(still))
			videoID := fmt.Sprintf("%x", sha256.Sum256(video))
			if _, err = lib.DB.ExecContext(ctx, `INSERT INTO assets(id,relative_path,motion_path,kind,created_at,updated_at) VALUES
				(?, 'assets/still.heic', 'assets/paired.mov', 'live', '', ''),
				(?, 'assets/video.mov', '', 'video', '', '')`, ownerID, videoID); err != nil {
				t.Fatal(err)
			}
			for path, data := range map[string][]byte{
				filepath.Join(folder, "file.HEIC"):              still,
				filepath.Join(folder, "file.MOV"):               video,
				filepath.Join(lib.Root, "assets", "still.heic"): still,
				filepath.Join(lib.Root, "assets", "paired.mov"): video,
				filepath.Join(lib.Root, "assets", "video.mov"):  video,
			} {
				if err = os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			imports, moves := 0, 0
			stopCtx, stop := context.WithCancel(ctx)
			defer stop()
			watcher := watch.New(lib.Root, func(_ context.Context, paths []string, _ string) (model.ImportResult, error) {
				imports++
				if imports == 1 {
					return model.ImportResult{Imported: 1, AssetIDs: []string{ownerID}, Motion: paths[1:]}, nil
				}
				return model.ImportResult{Skipped: 1}, nil
			}, func(path string) error {
				moves++
				stop()
				return os.Remove(path)
			}, nil, func(ctx context.Context, paths []string) (map[string]watch.AssetStatus, error) {
				return autoImportAssetStatus(ctx, lib, paths)
			})
			rules := []watch.Rule{{Path: folder, Enabled: true}}
			watcher.Update(rules)
			watcher.Scan(ctx)
			watcher.Scan(stopCtx)
			switch check {
			case "trashed":
				_, err = lib.DB.ExecContext(ctx, "UPDATE assets SET trashed=1 WHERE id=?", ownerID)
			case "missing-owner":
				_, err = lib.DB.ExecContext(ctx, "DELETE FROM assets WHERE id=?", ownerID)
			case "missing-still":
				err = os.Remove(filepath.Join(lib.Root, "assets", "still.heic"))
			case "changed-video":
				err = os.WriteFile(filepath.Join(lib.Root, "assets", "paired.mov"), []byte("other video"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			watcher.Update(nil)
			watcher.Scan(ctx)
			watcher.Update(rules)
			watcher.Scan(ctx)
			expected := 2
			if check == "active" {
				expected = 1
			}
			if imports != expected || moves != 2 {
				t.Fatalf("导入 %d 次，移源 %d 次，预期导入 %d 次、移源 2 次", imports, moves, expected)
			}
		})
	}
}
func TestExtensionPickerKeepsFolderIDsForDuplicateNames(t *testing.T) {
	for _, selected := range []string{"", "second"} {
		a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: DefaultPreferences()}
		a.Snapshot.Folders = []model.Folder{{ID: "root", Name: a.T("all_assets")}, {ID: "parent-one", Name: "One"}, {ID: "parent-two", Name: "Two"}, {ID: "first", Name: "Photos", ParentID: "parent-one"}, {ID: "second", Name: "Photos", ParentID: "parent-two"}}
		reply := make(chan importChoice, 1)
		a.Import = importState{PickerOpen: true, PickerReply: reply, PickerFolder: selected, PickerTags: map[string]bool{}}
		tt := ui.NewTester(a.extensionImportPicker, 600, 600)
		if selected != "" {
			if err := tt.Click(a.T("folders")); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Two / Photos"); err != nil {
				t.Fatal(err)
			}
		}
		if err := tt.Click(a.T("import")); err != nil {
			t.Fatal(err)
		}
		choice := <-reply
		if choice.FolderID != selected {
			t.Fatalf("同名目录或根选项被匹配到其他目录: %q / %q", choice.FolderID, selected)
		}
	}
}

func TestLockingCancelsAnAlreadyVisibleExtensionPicker(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	a.Preferences.PasswordHash = "configured"
	reply := make(chan importChoice, 1)
	a.Import = importState{PickerOpen: true, PickerReply: reply}
	a.LockNow()
	if !a.IsLocked() || a.Import.PickerOpen || a.Import.PickerReply != nil {
		t.Fatal("锁定后仍保留分类浮层")
	}
	select {
	case choice := <-reply:
		if !choice.Canceled {
			t.Fatal("已显示的浮层未经确认就导入")
		}
	default:
		t.Fatal("扩展请求仍挂起等待选择")
	}
}
