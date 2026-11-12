package assets

import (
	"bytes"
	"context"
	"encoding/binary"
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
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
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
	folder, err := lib.CreateFolder(ctx, "Folder", "")
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

func TestImportDecodeFailureLeavesNoFilesOrRows(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	input := t.TempDir()
	fixtureCopy(t, "png", filepath.Join(input, "a.png"))
	if err := os.WriteFile(filepath.Join(input, "b.jpg"), []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{input}}, nil)
	if err == nil {
		t.Fatal("损坏图片导入未失败")
	}
	if result.Imported != 0 || len(snapshot(t, ctx, lib).Assets) != 0 || len(storedFiles(t, lib.Root)) != 0 {
		t.Fatal("失败导入留下部分数据")
	}
	stages, err := filepath.Glob(filepath.Join(lib.Root, ".import-*"))
	if err != nil || len(stages) != 0 {
		t.Fatalf("暂存未清理：%v %v", stages, err)
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
	if err := s.EmptyTrash(ctx); err != nil {
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
	thumb := filepath.Join(lib.Root, "thumbnails", a.ID+".png")
	if err := os.Remove(thumb); err != nil {
		t.Fatal(err)
	}
	if err := s.RefreshThumbnails(ctx, []model.Asset{a}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(thumb)
	if err != nil {
		t.Fatal(err)
	}
	_, err = png.Decode(f)
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
		if _, err := thumbnailRelative(id); err == nil {
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
	if got.Width != 10 || got.Height != 20 || got.Orientation != 6 || got.EXIF["fileModifiedAt"] != modified.Format(time.RFC3339Nano) || got.EXIF["cameraMake"] != "Canon" || got.EXIF["flashFired"] != "false" {
		t.Fatalf("元数据不完整：%+v", got)
	}
	reopened, err := library.Open(ctx, lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if !reflect.DeepEqual(snapshot(t, ctx, reopened).Assets[0].EXIF, got.EXIF) {
		t.Fatal("EXIF没有持久化")
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
