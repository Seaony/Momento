package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

func testLibrary(t *testing.T) (context.Context, *library.Library, *Service) {
	t.Helper()
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Test.momento"), "Test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := lib.Close(); err != nil {
			t.Error(err)
		}
	})
	return ctx, lib, New(lib.Root, lib.DB)
}
func snapshot(t *testing.T, ctx context.Context, lib *library.Library) model.Snapshot {
	t.Helper()
	v, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// thumbnailFile 返回素材当前的缩略图路径，没有就让测试失败。
func thumbnailFile(t *testing.T, root, id string) string {
	t.Helper()
	path, err := ThumbnailPath(root, id)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func fixtureCopy(t *testing.T, ext, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "sample."+ext))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
func importFixture(t *testing.T, ctx context.Context, s *Service, ext string) {
	t.Helper()
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample."+ext)}}, nil); err != nil {
		t.Fatal(err)
	}
}
func storedFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, dir := range []string{"assets", "thumbnails"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				out = append(out, path)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestImportDeduplicatesPreservesEditsRestoresAndBackfillsSource(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	a := fixtureCopy(t, "png", filepath.Join(input, "a.png"))
	b := fixtureCopy(t, "png", filepath.Join(input, "b.PNG"))
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{a, b}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Skipped != 1 {
		t.Fatalf("%+v", result)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	if asset.Name != "a" || asset.OriginalFileName != "a.png" || asset.Extension != "png" || asset.Width != 96 || asset.Height != 64 || len(asset.Colors) == 0 || len(asset.Colors) > 8 {
		t.Fatalf("%+v", asset)
	}
	folder, err := lib.CreateFolder(ctx, "Folder", "", "")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := lib.CreateTag(ctx, "Tag")
	if err != nil {
		t.Fatal(err)
	}
	if err = lib.SetAssetFolders(ctx, []string{asset.ID}, []string{folder.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err = lib.SetAssetTags(ctx, []string{asset.ID}, []string{tag.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err = lib.RenameAsset(ctx, asset.ID, "Edited title"); err != nil {
		t.Fatal(err)
	}
	if err = lib.SetFavorite(ctx, []string{asset.ID}, true); err != nil {
		t.Fatal(err)
	}
	if _, err = lib.DB.ExecContext(ctx, "UPDATE assets SET note='My note' WHERE id=?", asset.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Trash(ctx, []string{asset.ID}); err != nil {
		t.Fatal(err)
	}
	result, err = s.Import(ctx, model.ImportRequest{Paths: []string{b}, SourceURL: "https://example.com/first"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Restored != 1 || result.Skipped != 0 {
		t.Fatalf("%+v", result)
	}
	_, err = s.Import(ctx, model.ImportRequest{Paths: []string{a}, SourceURL: "https://example.com/second"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := snapshot(t, ctx, lib).Assets[0]
	if got.ID != asset.ID || got.Trashed || got.Name != "Edited title" || !got.Favorite || got.Note != "My note" || got.SourceURL != "https://example.com/first" || !reflect.DeepEqual(got.FolderIDs, []string{folder.ID}) || !reflect.DeepEqual(got.TagIDs, []string{tag.ID}) || !got.CreatedAt.Equal(asset.CreatedAt) {
		t.Fatalf("用户编辑或关系被覆盖：%+v", got)
	}
	if len(storedFiles(t, lib.Root)) != 2 {
		t.Fatal("重复写入了原图或缩略图")
	}
	if err = os.RemoveAll(input); err != nil {
		t.Fatal(err)
	}
	reopened, err := library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if !reflect.DeepEqual(snapshot(t, ctx, reopened).Assets[0], got) {
		t.Fatal("重开后数据未持久化")
	}
}

func TestImportFoldersPreservesHierarchySkipsHiddenAndRoot(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	root := filepath.Join(t.TempDir(), "Selected root")
	fixtureCopy(t, "png", filepath.Join(root, "root.png"))
	fixtureCopy(t, "jpg", filepath.Join(root, "A", "one.jpg"))
	fixtureCopy(t, "png", filepath.Join(root, "A", "Nested", "same.png"))
	fixtureCopy(t, "gif", filepath.Join(root, "B", "two.gif"))
	fixtureCopy(t, "bmp", filepath.Join(root, ".hidden", "skip.bmp"))
	fixtureCopy(t, "tiff", filepath.Join(root, ".skip.tiff"))
	if err := os.WriteFile(filepath.Join(root, "ignore.txt"), []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{root}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 3 || result.Skipped != 1 {
		t.Fatalf("%+v", result)
	}
	snap := snapshot(t, ctx, lib)
	if len(snap.Folders) != 3 {
		t.Fatalf("folders=%+v", snap.Folders)
	}
	byName := map[string]model.Folder{}
	for _, f := range snap.Folders {
		byName[f.Name] = f
	}
	if byName["A"].ParentID != "" || byName["B"].ParentID != "" || byName["Nested"].ParentID != byName["A"].ID || byName["Selected root"].ID != "" {
		t.Fatalf("folders=%+v", snap.Folders)
	}
	for _, a := range snap.Assets {
		if a.Extension == "png" && !reflect.DeepEqual(a.FolderIDs, []string{byName["Nested"].ID}) {
			t.Fatalf("去重后子文件夹成员关系缺失：%+v", a)
		}
	}
}

func TestImportSkipsBrokenImageAndImportsTheRest(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	fixtureCopy(t, "png", filepath.Join(input, "a.png"))
	fixtureCopy(t, "gif", filepath.Join(input, "c.gif"))
	if err := os.WriteFile(filepath.Join(input, "b.jpg"), []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil)
	if err != nil {
		t.Fatalf("单张坏图让整批失败：%v", err)
	}
	if result.Imported != 2 || result.Skipped != 0 || len(result.Errors) != 1 || len(result.AssetIDs) != 2 {
		t.Fatalf("导入结果不对：%+v", result)
	}
	if !strings.Contains(result.Errors[0].Error(), "b.jpg") {
		t.Fatalf("错误里没有文件名：%v", result.Errors[0])
	}
	if len(snapshot(t, ctx, lib).Assets) != 2 || len(storedFiles(t, lib.Root)) != 4 {
		t.Fatal("好图没有完整导入")
	}
	stages, err := filepath.Glob(filepath.Join(lib.Root, ".import-*"))
	if err != nil || len(stages) != 0 {
		t.Fatalf("暂存未清理：%v %v", stages, err)
	}
}

func TestImportSkipsUnreadableAndDuplicateBrokenFilesWithConsistentCounts(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	input := t.TempDir()
	fixtureCopy(t, "png", filepath.Join(input, "again.png"))
	fixtureCopy(t, "jpg", filepath.Join(input, "fresh.jpg"))
	fixtureCopy(t, "jpg", filepath.Join(input, "fresh-copy.jpg"))
	for _, name := range []string{"broken-1.jpg", "broken-2.jpg"} {
		if err := os.WriteFile(filepath.Join(input, name), []byte("not an image"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	candidates := 5
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		locked := fixtureCopy(t, "gif", filepath.Join(input, "locked.gif"))
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0600) })
		candidates++
	}
	var last model.ImportProgress
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, func(p model.ImportProgress) { last = p })
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Skipped != 2 || len(result.Errors) != candidates-3 {
		t.Fatalf("导入结果不对：%+v", result)
	}
	if result.Imported+result.Skipped+len(result.Errors) != candidates {
		t.Fatalf("计数和候选总数对不上：%+v", result)
	}
	if last.Processed != candidates || last.Total != candidates || last.Imported+last.Skipped+len(result.Errors) != candidates {
		t.Fatalf("最终进度不对：%+v", last)
	}
	if len(snapshot(t, ctx, lib).Assets) != 2 {
		t.Fatal("好图没有导入")
	}
}

func TestImportReusesSameOriginalLeftByInterruptedImport(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	source := filepath.Join("testdata", "sample.png")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	id := hex.EncodeToString(sum[:])
	original := filepath.Join(lib.Root, "assets", id[:2], id+".png")
	thumbnail := filepath.Join(lib.Root, "thumbnails", id+".png")
	// 模拟上次导入时文件已放好、事务没提交就断电。
	if err = os.MkdirAll(filepath.Dir(original), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(original, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(thumbnail, []byte("half written"), 0600); err != nil {
		t.Fatal(err)
	}
	// 这次也失败回滚时，不能删掉不是这次放进去的原图。
	if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{source}, TagIDs: []string{"missing"}}, nil); err == nil {
		t.Fatal("缺失标签应让事务失败")
	}
	if _, err = os.Stat(original); err != nil {
		t.Fatalf("回滚删掉了已有原图：%v", err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil)
	if err != nil || result.Imported != 1 {
		t.Fatalf("已有同内容原图时导入失败：%+v %v", result, err)
	}
	// 残缺的旧 PNG 缩略图要换成新的 JPEG，不留两份。
	if _, err = os.Stat(thumbnail); !os.IsNotExist(err) {
		t.Fatalf("残缺的旧缩略图还在：%v", err)
	}
	f, err := os.Open(thumbnailFile(t, lib.Root, id))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = image.Decode(f)
	f.Close()
	if err != nil {
		t.Fatalf("残缺缩略图没有被覆盖：%v", err)
	}
	if err = os.WriteFile(filepath.Join(lib.Root, "assets", id[:2], id+".gif"), []byte("other content"), 0600); err != nil {
		t.Fatal(err)
	}
	gif := fixtureCopy(t, "gif", filepath.Join(t.TempDir(), "other.gif"))
	gifData, err := os.ReadFile(gif)
	if err != nil {
		t.Fatal(err)
	}
	gifSum := sha256.Sum256(gifData)
	gifID := hex.EncodeToString(gifSum[:])
	conflict := filepath.Join(lib.Root, "assets", gifID[:2], gifID+".gif")
	if err = os.MkdirAll(filepath.Dir(conflict), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(conflict, []byte("different"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Import(ctx, model.ImportRequest{Paths: []string{gif}}, nil); err == nil {
		t.Fatal("同名但内容不同的文件被当作已装好")
	}
	if kept, _ := os.ReadFile(conflict); string(kept) != "different" {
		t.Fatal("覆盖了内容不同的已有文件")
	}
}

func TestOpenLibraryRestoresInterruptedDeletionAndCleansStaging(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Test.momento"), "Test")
	if err != nil {
		t.Fatal(err)
	}
	s := New(lib.Root, lib.DB)
	for _, ext := range []string{"png", "gif", "jpg"} {
		importFixture(t, ctx, s, ext)
	}
	assets := snapshot(t, ctx, lib).Assets
	if err = s.Trash(ctx, []string{assets[0].ID, assets[1].ID, assets[2].ID}); err != nil {
		t.Fatal(err)
	}
	// 第 3 张真的删掉了：记录没了，文件留在暂存里。
	gone := assets[2]
	if err = s.DeletePermanently(ctx, []string{gone.ID}); err != nil {
		t.Fatal(err)
	}
	// 模拟永久删除挪走文件后、提交前崩溃：新命名和老版本的序号命名各一个。
	deletion := filepath.Join(lib.Root, ".delete-123")
	legacy := filepath.Join(lib.Root, ".delete-456")
	for _, dir := range []string{deletion, legacy, filepath.Join(lib.Root, ".import-789")} {
		if err = os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	moves := []struct{ from, to string }{
		{filepath.Join(lib.Root, filepath.FromSlash(assets[0].RelativePath)), filepath.Join(deletion, assets[0].ID+".original")},
		{thumbnailFile(t, lib.Root, assets[0].ID), filepath.Join(deletion, assets[0].ID+".thumbnail.jpg")},
		{filepath.Join(lib.Root, filepath.FromSlash(assets[1].RelativePath)), filepath.Join(legacy, "0")},
		{thumbnailFile(t, lib.Root, assets[1].ID), filepath.Join(legacy, "1")},
	}
	for _, move := range moves {
		if err = os.Rename(move.from, move.to); err != nil {
			t.Fatal(err)
		}
	}
	fixtureCopy(t, "jpg", filepath.Join(deletion, gone.ID+".original"))
	if err = os.WriteFile(filepath.Join(lib.Root, ".import-789", "source-0.png"), []byte("staged"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = lib.Close(); err != nil {
		t.Fatal(err)
	}
	lib, err = library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	for _, path := range []string{moves[0].from, moves[1].from, moves[2].from} {
		if _, err = os.Stat(path); err != nil {
			t.Fatalf("没有挪回中断删除的文件：%v", err)
		}
	}
	if _, err = os.Stat(filepath.Join(lib.Root, filepath.FromSlash(gone.RelativePath))); !os.IsNotExist(err) {
		t.Fatalf("已提交删除的原图被挪了回来：%v", err)
	}
	left, err := filepath.Glob(filepath.Join(lib.Root, ".[di][em]*-*"))
	if err != nil || len(left) != 0 {
		t.Fatalf("残留暂存没有清掉：%v %v", left, err)
	}
	if got := snapshot(t, ctx, lib).Assets; len(got) != 2 {
		t.Fatalf("记录数不对：%d", len(got))
	}
}

func TestImportRepairsMissingOriginalOfExistingAsset(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	asset := snapshot(t, ctx, lib).Assets[0]
	if err := lib.SetRotation(ctx, []string{asset.ID}, 90); err != nil {
		t.Fatal(err)
	}
	if err := s.Trash(ctx, []string{asset.ID}); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath))
	for _, path := range []string{original, thumbnailFile(t, lib.Root, asset.ID)} {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample.png")}}, nil)
	if err != nil || result.Imported != 1 || result.Restored != 1 || len(result.Errors) != 0 {
		t.Fatalf("缺原图的素材没有补回：%+v %v", result, err)
	}
	if missing, err := s.storedCopyMissing(ctx, asset.RelativePath, asset.ID); err != nil || missing {
		t.Fatalf("补回的原图不对：%v %v", missing, err)
	}
	f, err := os.Open(thumbnailFile(t, lib.Root, asset.ID))
	if err != nil {
		t.Fatal(err)
	}
	thumb, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if b := thumb.Bounds(); b.Dx() >= b.Dy() {
		t.Fatalf("补回的缩略图没有带上旋转：%v", b)
	}
}

func TestImportDatabaseFailureRollsBackFilesAndVirtualFolders(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	if _, err := lib.DB.ExecContext(ctx, `CREATE TRIGGER reject_jpeg BEFORE INSERT ON assets WHEN NEW.extension='jpg' BEGIN SELECT RAISE(ABORT,'拒绝测试写入'); END`); err != nil {
		t.Fatal(err)
	}
	input := t.TempDir()
	fixtureCopy(t, "png", filepath.Join(input, "A", "a.png"))
	fixtureCopy(t, "jpg", filepath.Join(input, "B", "b.jpg"))
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil)
	if err == nil {
		t.Fatal("数据库失败没有返回错误")
	}
	snap := snapshot(t, ctx, lib)
	if result.Imported != 0 || len(snap.Assets) != 0 || len(snap.Folders) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("整批导入没有同时回滚文件与数据")
	}
}

func TestImportBusyAndProgress(t *testing.T) {
	ctx, _, s := testLibrary(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	var values []model.ImportProgress
	blocked := false
	go func() {
		_, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample.png")}}, func(p model.ImportProgress) {
			values = append(values, p)
			if p.Stage == "scanning" && !blocked {
				blocked = true
				close(entered)
				<-release
			}
		})
		done <- err
	}()
	<-entered
	if _, err := s.Import(ctx, model.ImportRequest{}, nil); !errors.Is(err, ErrImportBusy) {
		t.Fatalf("第二次导入未拒绝：%v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(values) < 3 || values[0].Stage != "scanning" || values[len(values)-1].Stage != "finishing" {
		t.Fatalf("阶段不全：%+v", values)
	}
	final := values[len(values)-1]
	if final.Total != 1 || final.Processed != 1 || final.Imported != 1 || final.Skipped != 0 {
		t.Fatalf("最终进度错误：%+v", final)
	}
}

func TestCancelledImportDoesNotModifyLibrary(t *testing.T) {
	_, lib, s := testLibrary(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample.png")}}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	if len(snapshot(t, context.Background(), lib).Assets) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("取消后仍写入数据")
	}
}

func TestTrashRestoreAndEmptyTrashPreserveLiveAssets(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	importFixture(t, ctx, s, "jpg")
	snap := snapshot(t, ctx, lib)
	ids := []string{snap.Assets[0].ID, snap.Assets[1].ID}
	if err := s.Trash(ctx, ids); err != nil {
		t.Fatal(err)
	}
	for _, a := range snapshot(t, ctx, lib).Assets {
		if !a.Trashed {
			t.Fatal("批量软删除未持久化")
		}
	}
	if len(storedFiles(t, lib.Root)) != 4 {
		t.Fatal("软删除动了文件")
	}
	if err := s.Restore(ctx, ids[:1]); err != nil {
		t.Fatal(err)
	}
	// 已恢复的 ids[0] 也在确认清单里：跳过它，只删还在回收站的。
	if err := s.EmptyTrash(ctx, ids); err != nil {
		t.Fatal(err)
	}
	snap = snapshot(t, ctx, lib)
	if len(snap.Assets) != 1 || snap.Assets[0].ID != ids[0] || snap.Assets[0].Trashed || len(storedFiles(t, lib.Root)) != 2 {
		t.Fatalf("%+v", snap.Assets)
	}
	if err := s.DeletePermanently(ctx, ids[:1]); err == nil {
		t.Fatal("允许直接永久删除未入回收站素材")
	}
}

func TestPermanentDeleteDatabaseFailureRestoresFilesAndRelations(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	a := snapshot(t, ctx, lib).Assets[0]
	tag, err := lib.CreateTag(ctx, "Keep")
	if err != nil {
		t.Fatal(err)
	}
	if err = lib.SetAssetTags(ctx, []string{a.ID}, []string{tag.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err = s.Trash(ctx, []string{a.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err = lib.DB.ExecContext(ctx, `CREATE TRIGGER reject_delete BEFORE DELETE ON assets BEGIN SELECT RAISE(ABORT,'拒绝测试删除'); END`); err != nil {
		t.Fatal(err)
	}
	if err = s.DeletePermanently(ctx, []string{a.ID}); err == nil {
		t.Fatal("删除失败未报错")
	}
	snap := snapshot(t, ctx, lib)
	if len(snap.Assets) != 1 || !snap.Assets[0].Trashed || len(snap.Assets[0].TagIDs) != 1 || len(storedFiles(t, lib.Root)) != 2 {
		t.Fatal("删除失败没有回滚")
	}
	if _, err = lib.DB.ExecContext(ctx, "DROP TRIGGER reject_delete"); err != nil {
		t.Fatal(err)
	}
	if err = s.DeletePermanently(ctx, []string{a.ID}); err != nil {
		t.Fatal(err)
	}
	if len(snapshot(t, ctx, lib).Assets) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("永久删除不完整")
	}
	var count int
	if err = lib.DB.QueryRowContext(ctx, "SELECT count(*) FROM asset_tags").Scan(&count); err != nil || count != 0 {
		t.Fatalf("关联未级联删除：%d %v", count, err)
	}
}

func TestExportNeverOverwritesAndConverts(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	a := snapshot(t, ctx, lib).Assets[0]
	dest := t.TempDir()
	existing := filepath.Join(dest, "sample.png")
	if err := os.WriteFile(existing, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	paths, err := s.Export(ctx, []model.Asset{a, a}, dest, model.ExportOriginal, 0)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(paths[0]) != "sample 2.png" || filepath.Base(paths[1]) != "sample 3.png" {
		t.Fatalf("%v", paths)
	}
	content, err := os.ReadFile(existing)
	if err != nil || string(content) != "keep" {
		t.Fatal("覆盖了原文件")
	}
	original, _ := os.ReadFile(filepath.Join("testdata", "sample.png"))
	copyData, _ := os.ReadFile(paths[0])
	if !bytes.Equal(original, copyData) {
		t.Fatal("原格式导出修改内容")
	}
	for _, format := range []model.ExportFormat{model.ExportJPEG, model.ExportPNG} {
		paths, err = s.Export(ctx, []model.Asset{a}, dest, format, .9)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(paths[0])
		if err != nil {
			t.Fatal(err)
		}
		img, kind, decodeErr := image.Decode(f)
		f.Close()
		if decodeErr != nil || img.Bounds().Dx() != 96 || img.Bounds().Dy() != 64 {
			t.Fatalf("转换失败：%v", decodeErr)
		}
		if string(format) != kind {
			t.Fatalf("格式 %s 期望 %s", kind, format)
		}
	}
	exact := filepath.Join(dest, "chosen.png")
	paths, err = s.Export(ctx, []model.Asset{a}, exact, model.ExportPNG, 0)
	if err != nil || len(paths) != 1 || paths[0] != exact {
		t.Fatalf("单文件保存：%v %v", paths, err)
	}
}

func TestRefreshThumbnailAndColors(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	a := snapshot(t, ctx, lib).Assets[0]
	if err := os.Remove(thumbnailFile(t, lib.Root, a.ID)); err != nil {
		t.Fatal(err)
	}
	if err := s.RefreshThumbnails(ctx, []model.Asset{a}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(thumbnailFile(t, lib.Root, a.ID))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = image.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lib.DB.ExecContext(ctx, "DELETE FROM asset_colors WHERE asset_id=?", a.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.ReanalyzeColors(ctx, []model.Asset{a}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot(t, ctx, lib).Assets[0].Colors, a.Colors) {
		t.Fatal("颜色重算不稳定")
	}
}

func TestImportURLUnversionedRequestUsesProductName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "Momento" {
			t.Errorf("未打包请求的产品名错误：%q", got)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if _, err := (&Service{}).ImportURL(context.Background(), server.URL+"/image.png", ""); err == nil {
		t.Fatal("服务不可用时下载没有报告错误")
	}
}

func TestImportURLDownloadsLocallyAndKeepsSource(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	configData, err := os.ReadFile("../../mygo.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		t.Fatal(err)
	}
	if config.Version == "" {
		t.Fatal("应用配置缺少版本号")
	}
	s.AppVersion = config.Version
	data, err := os.ReadFile(filepath.Join("testdata", "sample.png"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("User-Agent"), "Momento/"+config.Version; got != want {
			t.Errorf("下载请求版本不一致：%q，应为%q", got, want)
		}
		switch r.URL.Path {
		case "/image":
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Content-Disposition", `attachment; filename="reference.png"`)
			_, _ = w.Write(data)
		case "/broken":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("broken"))
		default:
			http.Error(w, "missing", http.StatusNotFound)
		}
	}))
	defer server.Close()
	result, err := s.ImportURL(ctx, server.URL+"/image", "https://example.com/article")
	if err != nil || result.Imported != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	a := snapshot(t, ctx, lib).Assets[0]
	if a.SourceURL != "https://example.com/article" || a.OriginalFileName != "reference.png" {
		t.Fatalf("%+v", a)
	}
	for _, raw := range []string{"file:///private/test.png", server.URL + "/missing", server.URL + "/broken"} {
		if _, err = s.ImportURL(ctx, raw, ""); err == nil {
			t.Fatalf("坏地址没返回错误：%s", raw)
		}
	}
	if len(snapshot(t, ctx, lib).Assets) != 1 {
		t.Fatal("下载失败污染图库")
	}
}

func TestRelativePathsRejectEscape(t *testing.T) {
	_, _, s := testLibrary(t)
	for _, path := range []string{"../outside.png", "/tmp/outside.png", ""} {
		if _, err := s.assetPath(path); err == nil {
			t.Fatalf("接受逃逸路径：%q", path)
		}
	}
	for _, id := range []string{"../outside", strings.Repeat("g", 64), ""} {
		if _, err := thumbnailBase(id); err == nil {
			t.Fatalf("接受非法标识：%q", id)
		}
	}
	outside := fixtureCopy(t, "png", filepath.Join(t.TempDir(), "outside.png"))
	if err := os.Symlink(outside, filepath.Join(s.root, "linked.png")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.assetPath("linked.png"); err == nil {
		t.Fatal("接受指向图库外的符号链接")
	}
}

func TestProgressThrottle(t *testing.T) {
	var calls int
	p := progressReporter{callback: func(model.ImportProgress) { calls++ }}
	p.send(model.ImportProgress{}, false)
	p.send(model.ImportProgress{}, false)
	if calls != 1 {
		t.Fatalf("未节流：%d", calls)
	}
	p.last = time.Now().Add(-101 * time.Millisecond)
	p.send(model.ImportProgress{}, false)
	if calls != 2 {
		t.Fatalf("节流窗口后未回调：%d", calls)
	}
	p.send(model.ImportProgress{}, true)
	if calls != 3 {
		t.Fatal("最终进度未强制发送")
	}
}

func TestImportRefusesStorageSymlinksWithoutWritingOutside(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	outside := t.TempDir()
	if err := os.Remove(filepath.Join(lib.Root, "assets")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(lib.Root, "assets")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{filepath.Join("testdata", "sample.png")}}, nil); err == nil {
		t.Fatal("导入跟随了包内符号链接")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("写到了包外：%v %v", entries, err)
	}
	if len(snapshot(t, ctx, lib).Assets) != 0 {
		t.Fatal("失败后保留记录")
	}
}

func TestImportEXIFPersistsAndKeepsSourceModificationTime(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 20, 10))); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	payload := imageTestTIFF()
	chunk := make([]byte, 8+len(payload)+4)
	binary.BigEndian.PutUint32(chunk, uint32(len(payload)))
	copy(chunk[4:], "eXIf")
	copy(chunk[8:], payload)
	binary.BigEndian.PutUint32(chunk[len(chunk)-4:], crc32.ChecksumIEEE(chunk[4:len(chunk)-4]))
	data = append(append(append([]byte{}, data[:33]...), chunk...), data[33:]...)
	path := filepath.Join(t.TempDir(), "metadata.png")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	modified := time.Date(2020, 2, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{path}}, nil); err != nil {
		t.Fatal(err)
	}
	got := snapshot(t, ctx, lib).Assets[0]
	// 快照不带 EXIF，单独读。
	exif, err := lib.AssetEXIF(ctx, got.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != 10 || got.Height != 20 || got.Orientation != 6 || exif["fileModifiedAt"] != modified.Format(time.RFC3339Nano) || exif["cameraMake"] != "Canon" || exif["flashFired"] != "false" {
		t.Fatalf("元数据不完整：%+v %v", got, exif)
	}
	reopened, err := library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if again, err := reopened.AssetEXIF(ctx, got.ID); err != nil || !reflect.DeepEqual(again, exif) {
		t.Fatalf("EXIF没有持久化：%v", err)
	}
}

func TestExportUsesEditedTitleAndOriginalExtension(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	a := snapshot(t, ctx, lib).Assets[0]
	a.Name = "Edited title"
	paths, err := s.Export(ctx, []model.Asset{a}, t.TempDir(), model.ExportOriginal, 0)
	if err != nil || len(paths) != 1 || filepath.Base(paths[0]) != "Edited title.png" {
		t.Fatalf("%v %v", paths, err)
	}
}

func TestImportURLStopsOversizedAndStalledDownloads(t *testing.T) {
	previousSize, previousIdle := maxDownloadBytes, downloadIdleTimeout
	maxDownloadBytes, downloadIdleTimeout = 1024, 200*time.Millisecond
	t.Cleanup(func() { maxDownloadBytes, downloadIdleTimeout = previousSize, previousIdle })
	ctx, lib, s := testLibrary(t)
	data, err := os.ReadFile(filepath.Join("testdata", "sample.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= 1024 {
		t.Fatal("样例图片太小，测不到大小上限")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		switch r.URL.Path {
		case "/declared":
			_, _ = w.Write(data)
		case "/chunked":
			// 不带长度，边下边数才能发现超限。
			for i := 0; i < len(data); i += 256 {
				_, _ = w.Write(data[i:min(i+256, len(data))])
				w.(http.Flusher).Flush()
			}
		case "/stalled":
			_, _ = w.Write(data[:100])
			w.(http.Flusher).Flush()
			select {
			case <-r.Context().Done():
			case <-time.After(5 * time.Second):
			}
		}
	}))
	defer server.Close()
	for path, key := range map[string]string{"/declared": "error_download_too_large", "/chunked": "error_download_too_large", "/stalled": "error_download_stalled"} {
		started := time.Now()
		_, err := s.ImportURL(ctx, server.URL+path, "")
		var localized *i18n.LocalizedError
		if !errors.As(err, &localized) || localized.Key != key {
			t.Fatalf("%s 应返回 %s：%v", path, key, err)
		}
		if time.Since(started) > 3*time.Second {
			t.Fatalf("%s 没有及时停止", path)
		}
	}
	if len(snapshot(t, ctx, lib).Assets) != 0 {
		t.Fatal("下载失败污染图库")
	}
}

func TestImportURLReportsSkippedFileAsError(t *testing.T) {
	ctx, _, s := testLibrary(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("broken"))
	}))
	defer server.Close()
	result, err := s.ImportURL(ctx, server.URL+"/broken.png", "")
	if err == nil || result.Imported != 0 || len(result.AssetIDs) != 0 {
		t.Fatalf("坏图下载被当作成功：%+v %v", result, err)
	}
}

func TestRefreshThumbnailsAndColorsContinuePastBrokenOriginals(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	for _, ext := range []string{"png", "gif", "jpg", "bmp", "tiff", "webp", "avif"} {
		fixtureCopy(t, ext, filepath.Join(input, "sample."+ext))
	}
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil); err != nil {
		t.Fatal(err)
	}
	items := snapshot(t, ctx, lib).Assets
	if len(items) != 7 {
		t.Fatalf("样例数量不对：%d", len(items))
	}
	// 前 6 张原图一张丢失、其余损坏，最后一张完好。
	for i, a := range items[:6] {
		path := filepath.Join(lib.Root, filepath.FromSlash(a.RelativePath))
		var err error
		if i == 0 {
			err = os.Remove(path)
		} else {
			err = os.WriteFile(path, []byte("broken"), 0600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	good := items[6]
	thumb := thumbnailFile(t, lib.Root, good.ID)
	if err := os.Remove(thumb); err != nil {
		t.Fatal(err)
	}
	err := s.RefreshThumbnails(ctx, items)
	var localized *i18n.LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_thumbnails_failed_more" || localized.Args[0] != 6 {
		t.Fatalf("缩略图汇总错误不对：%v", err)
	}
	if !strings.Contains(err.Error(), items[0].Name) {
		t.Fatalf("汇总错误里没有文件名：%v", err)
	}
	if _, err = os.Stat(thumb); err != nil {
		t.Fatalf("坏图后面的缩略图没有生成：%v", err)
	}
	err = s.RefreshThumbnails(ctx, items[:1])
	if !errors.As(err, &localized) || localized.Key != "error_thumbnail_refresh" {
		t.Fatalf("单张失败应给出具体原因：%v", err)
	}
	if _, err = lib.DB.ExecContext(ctx, "DELETE FROM asset_colors WHERE asset_id=?", good.ID); err != nil {
		t.Fatal(err)
	}
	err = s.ReanalyzeColors(ctx, items[4:])
	if !errors.As(err, &localized) || localized.Key != "error_colors_failed" || localized.Args[0] != 2 {
		t.Fatalf("颜色汇总错误不对：%v", err)
	}
	for _, a := range snapshot(t, ctx, lib).Assets {
		if a.ID == good.ID && !reflect.DeepEqual(a.Colors, good.Colors) {
			t.Fatal("坏图后面的颜色没有重算")
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err = s.RefreshThumbnails(cancelled, items); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消后应立即返回：%v", err)
	}
}

// 图库包里的上层目录被换成指向包外的符号链接时，挪回中断删除的文件不能写到包外。
func TestInterruptedDeletionRestoreRefusesSymlinkedDirectory(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Test.momento"), "Test")
	if err != nil {
		t.Fatal(err)
	}
	s := New(lib.Root, lib.DB)
	importFixture(t, ctx, s, "png")
	asset := snapshot(t, ctx, lib).Assets[0]
	original := filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath))
	deletion := filepath.Join(lib.Root, ".delete-1")
	if err = os.Mkdir(deletion, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(original, filepath.Join(deletion, asset.ID+".original")); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	parent := filepath.Dir(original)
	if err = os.RemoveAll(parent); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, parent); err != nil {
		t.Fatal(err)
	}
	if err = lib.Close(); err != nil {
		t.Fatal(err)
	}
	lib, err = library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("文件被写到了图库包外：%v", entries)
	}
	if _, err = os.Stat(filepath.Join(deletion, asset.ID+".original")); err != nil {
		t.Fatalf("拒绝挪回时暂存里的文件应原样保留：%v", err)
	}
}

func TestExportToChosenFileReplacesItAndFolderExportRenames(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	a := snapshot(t, ctx, lib).Assets[0]
	dest := t.TempDir()
	// 保存对话框选的已存在文件：用户已确认替换，要写到这个路径本身。
	chosen := filepath.Join(dest, "chosen.png")
	if err := os.WriteFile(chosen, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	paths, err := s.Export(ctx, []model.Asset{a}, chosen, model.ExportOriginal, 0)
	if err != nil || len(paths) != 1 || paths[0] != chosen {
		t.Fatalf("没有写到选定的路径：%v %v", paths, err)
	}
	original, _ := os.ReadFile(filepath.Join("testdata", "sample.png"))
	if got, err := os.ReadFile(chosen); err != nil || !bytes.Equal(got, original) {
		t.Fatal("没有替换选定的文件", err)
	}
	entries, err := os.ReadDir(dest)
	if err != nil || len(entries) != 1 {
		t.Fatalf("留下了多余文件：%v %v", entries, err)
	}
	// 导出到文件夹时重名照旧改名，返回值里是实际写入的路径。
	paths, err = s.Export(ctx, []model.Asset{a}, dest, model.ExportOriginal, 0)
	if err != nil || len(paths) != 1 || paths[0] != filepath.Join(dest, "sample.png") {
		t.Fatalf("%v %v", paths, err)
	}
	paths, err = s.Export(ctx, []model.Asset{a}, dest, model.ExportOriginal, 0)
	if err != nil || len(paths) != 1 || paths[0] != filepath.Join(dest, "sample 2.png") {
		t.Fatalf("文件夹导出没有改名：%v %v", paths, err)
	}
}

// 导出目标在图库包里（包括只差大小写的路径）时拒绝，不能覆盖图库自己的原图。
func TestExportRefusesDestinationInsideLibrary(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	importFixture(t, ctx, s, "png")
	a := snapshot(t, ctx, lib).Assets[0]
	original := filepath.Join(lib.Root, filepath.FromSlash(a.RelativePath))
	before, _ := os.ReadFile(original)
	targets := []string{original, filepath.Dir(original), lib.Root}
	if upper := strings.ToUpper(original); upper != original {
		if _, err := os.Stat(upper); err == nil {
			targets = append(targets, upper)
		}
	}
	for _, target := range targets {
		_, err := s.Export(ctx, []model.Asset{a}, target, model.ExportPNG, 0)
		var localized *i18n.LocalizedError
		if !errors.As(err, &localized) || localized.Key != "error_export_into_library" {
			t.Fatalf("%s：%v", target, err)
		}
	}
	if after, _ := os.ReadFile(original); !bytes.Equal(before, after) {
		t.Fatal("图库原图被改动")
	}
}

// 视频重新分析颜色用封面截帧来算；取不到封面时报出封面错误。实况照片用它的静态图。
func TestReanalyzeColorsUsesVideoPosterAndLiveStill(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{mediaFixture("probe_h264.mp4")}}, nil); err != nil {
		t.Fatal(err)
	}
	video := snapshot(t, ctx, lib).Assets[0]
	if _, err := lib.DB.ExecContext(ctx, "DELETE FROM asset_colors"); err != nil {
		t.Fatal(err)
	}
	cover := image.NewRGBA(image.Rect(0, 0, 64, 36))
	for i := range cover.Pix {
		cover.Pix[i] = 0xff
		if i%4 == 1 || i%4 == 2 {
			cover.Pix[i] = 0
		}
	}
	previous := videoPoster
	t.Cleanup(func() { videoPoster = previous })
	var gotAt time.Duration
	var gotSize int
	videoPoster = func(_ context.Context, _ string, at time.Duration, size int) (image.Image, error) {
		gotAt, gotSize = at, size
		return cover, nil
	}
	if err := s.ReanalyzeColors(ctx, []model.Asset{video}); err != nil {
		t.Fatal(err)
	}
	if gotAt != posterAt || gotSize != thumbnailSize {
		t.Fatalf("没按导入时的封面参数截帧：%v %d", gotAt, gotSize)
	}
	if colors := snapshot(t, ctx, lib).Assets[0].Colors; len(colors) == 0 || !reflect.DeepEqual(colors, Palette(cover)) {
		t.Fatalf("视频颜色没按封面算：%v", colors)
	}

	videoPoster = func(context.Context, string, time.Duration, int) (image.Image, error) {
		return nil, media.ErrPosterUnavailable
	}
	err := s.ReanalyzeColors(ctx, []model.Asset{video})
	var localized *i18n.LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_colors_refresh" || !errors.Is(err, media.ErrPosterUnavailable) {
		t.Fatalf("取不到封面应报出封面错误：%v", err)
	}
	if cause, ok := localized.Cause.(*i18n.LocalizedError); !ok || cause.Key != "error_video_poster" {
		t.Fatalf("错误里没有封面这一层：%v", err)
	}

	ctx, lib, s = testLibrary(t)
	live := importLive(t, s, lib)
	if _, err = lib.DB.ExecContext(ctx, "DELETE FROM asset_colors"); err != nil {
		t.Fatal(err)
	}
	if err = s.ReanalyzeColors(ctx, []model.Asset{live}); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, ctx, lib).Assets[0].Colors; len(got) == 0 || !reflect.DeepEqual(got, live.Colors) {
		t.Fatalf("实况照片的颜色没按静态图算：%v，期望 %v", got, live.Colors)
	}
}
