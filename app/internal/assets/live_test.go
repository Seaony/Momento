package assets

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

// liveContentID 是样本实况视频里的内容 ID；样本图片里没有，测试里让图片读出同一个。
const liveContentID = "FD0B0708-C73F-4F6D-B57B-4169BB991D46"

// livePair 在临时文件夹里放一对同名的「图片 + MOV」，返回两个路径；图片的内容 ID 和视频一致，是一对真正的实况照片。
func livePair(t *testing.T, dir, name string) (still, motion string) {
	t.Helper()
	fakeContentID(t, liveContentID)
	still = fixtureCopy(t, "heic", filepath.Join(dir, name+".HEIC"))
	data, err := os.ReadFile(mediaFixture("probe_livephoto.mov"))
	if err != nil {
		t.Fatal(err)
	}
	motion = filepath.Join(dir, name+".MOV")
	if err = os.WriteFile(motion, data, 0600); err != nil {
		t.Fatal(err)
	}
	return still, motion
}

func fakeContentID(t *testing.T, id string) {
	t.Helper()
	previous := readImageContentID
	readImageContentID = func(string) (string, error) { return id, nil }
	t.Cleanup(func() { readImageContentID = previous })
}

// 同名的图片和 MOV 配成一个实况素材：视频存进包里，时长取视频的；导入数按素材算，视频记在 Motion 里。
func TestImportPairsLivePhoto(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	still, motion := livePair(t, dir, "IMG_0001")
	info, err := media.Probe(motion)
	if err != nil {
		t.Fatal(err)
	}
	if info.ContentID != liveContentID {
		t.Fatalf("样本视频的内容 ID 变了：%q", info.ContentID)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{dir}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Skipped != 0 || len(result.Errors) != 0 || len(result.Motion) != 1 || result.Motion[0] != motion {
		t.Fatalf("导入结果不对：%+v", result)
	}
	assets := snapshot(t, ctx, lib).Assets
	if len(assets) != 1 {
		t.Fatalf("应只有一个素材：%d", len(assets))
	}
	asset := assets[0]
	if asset.Kind != model.KindLive || asset.Extension != "heic" || asset.DurationMS != info.Duration.Milliseconds() || asset.MotionPath == "" {
		t.Fatalf("实况素材不对：%+v", asset)
	}
	stored, err := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.MotionPath)))
	want, _ := os.ReadFile(motion)
	if err != nil || !bytes.Equal(stored, want) {
		t.Fatal("包里的实况视频不是源视频", err)
	}
	if original, _ := os.ReadFile(still); len(original) == 0 {
		t.Fatal("源图片不该被动")
	}

	// 再导一次：整对算一个重复，视频仍记在 Motion 里，自动导入据此把两个源文件都移走。
	again, err := s.Import(ctx, model.ImportRequest{Paths: []string{still, motion}}, nil)
	if err != nil || again.Imported != 0 || again.Skipped != 1 || len(again.Motion) != 1 {
		t.Fatalf("重复导入实况照片的结果不对：%+v %v", again, err)
	}
}

// 内容 ID 对不上的同名文件不配对，各导各的。
func TestImportLivePhotoContentIDMismatchImportsSeparately(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	livePair(t, dir, "IMG_0002")
	info, err := media.Probe(filepath.Join(dir, "IMG_0002.MOV"))
	if err != nil {
		t.Fatal(err)
	}
	if info.ContentID == "" {
		t.Skip("样本视频没有内容 ID")
	}
	fakeContentID(t, "SOMETHING-ELSE")
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{dir}}, nil)
	if err != nil || result.Imported != 2 || len(result.Motion) != 0 {
		t.Fatalf("内容 ID 不同应导成两个素材：%+v %v", result, err)
	}
	kinds := map[string]int{}
	for _, asset := range snapshot(t, ctx, lib).Assets {
		kinds[asset.Kind]++
	}
	if kinds[model.KindImage] != 1 || kinds[model.KindVideo] != 1 {
		t.Fatalf("种类不对：%v", kinds)
	}
}

// 图片坏了：这对实况照片整对失败，图片和视频各记一条，自动导入就不会把视频移走。
func TestImportBrokenLiveStillFailsBothFiles(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	still, motion := livePair(t, dir, "IMG_0003")
	if err := os.WriteFile(still, []byte("not an image"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{dir}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 0 || len(result.Failed) != 2 || len(result.Errors) != 2 || result.Failed[0] != still || result.Failed[1] != motion {
		t.Fatalf("坏的实况照片应两个文件都记失败：%+v", result)
	}
	if n := len(snapshot(t, ctx, lib).Assets); n != 0 {
		t.Fatalf("不该入库：%d", n)
	}
}

func importLive(t *testing.T, s *Service, lib *library.Library) model.Asset {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	livePair(t, dir, "IMG_0004")
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{dir}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	if asset.Kind != model.KindLive {
		t.Fatalf("没配成实况照片：%+v", asset)
	}
	return asset
}

// 永久删除连视频一起删；中途崩溃时视频也能在打开图库时挪回原处。
func TestDeleteLivePhotoRemovesAndRestoresMotion(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	asset := importLive(t, s, lib)
	motion := filepath.Join(lib.Root, filepath.FromSlash(asset.MotionPath))
	if err := s.Trash(ctx, []string{asset.ID}); err != nil {
		t.Fatal(err)
	}
	// 模拟挪进暂存后、提交前崩溃。
	deletion := filepath.Join(lib.Root, ".delete-live")
	if err := os.Mkdir(deletion, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(motion, filepath.Join(deletion, asset.ID+".motion")); err != nil {
		t.Fatal(err)
	}
	root := lib.Root
	if err := lib.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := library.Open(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, err = os.Stat(motion); err != nil {
		t.Fatalf("没有挪回实况视频：%v", err)
	}
	s = New(reopened.Root, reopened.DB)
	if err = s.DeletePermanently(ctx, []string{asset.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(motion); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("永久删除后实况视频还在：%v", err)
	}
}

// 导出时勾了视频：图片和视频同名放在一起；没勾只导图片。保存框单个导出时，同名视频已存在就报错，不悄悄覆盖。
func TestExportLivePhotoWithMotion(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	asset := importLive(t, s, lib)
	plain := t.TempDir()
	if paths, err := s.Export(ctx, []model.Asset{asset}, plain, model.ExportOriginal, .9); err != nil || len(paths) != 1 {
		t.Fatalf("只导图片失败：%v %v", paths, err)
	}
	if entries, _ := os.ReadDir(plain); len(entries) != 1 {
		t.Fatalf("没勾视频却导出了 %d 个文件", len(entries))
	}
	dir := t.TempDir()
	for range 2 {
		if _, err := s.ExportWithMotion(ctx, []model.Asset{asset}, dir, model.ExportOriginal, .9, true); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"IMG_0004.heic", "IMG_0004.mov", "IMG_0004 2.heic", "IMG_0004 2.mov"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("缺少 %s：%v", name, err)
		}
	}
	want, _ := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.MotionPath)))
	if got, _ := os.ReadFile(filepath.Join(dir, "IMG_0004.mov")); !bytes.Equal(got, want) {
		t.Fatal("导出的视频内容不对")
	}
	single := filepath.Join(t.TempDir(), "photo.heic")
	if err := os.WriteFile(motionExportPath(single), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExportWithMotion(ctx, []model.Asset{asset}, single, model.ExportOriginal, .9, true); err == nil {
		t.Fatal("同名视频已存在应报错")
	}
	if kept, _ := os.ReadFile(motionExportPath(single)); string(kept) != "keep" {
		t.Fatal("已有的同名视频被覆盖了")
	}
}

// 复制（导出）图库时实况视频跟着走。
func TestExportLibraryCopiesLiveMotion(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	asset := importLive(t, s, lib)
	destination := filepath.Join(t.TempDir(), "Copy.momento")
	if err := lib.ExportLibrary(ctx, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(destination, filepath.FromSlash(asset.MotionPath))); err != nil {
		t.Fatalf("复制的图库里没有实况视频：%v", err)
	}
}

// 图库里已有这张图（以前单独导入过），再带着同名视频导入：补存视频、升级成实况，视频才算处理过。
func TestImportUpgradesExistingStillToLive(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	still, motion := livePair(t, dir, "IMG_0005")
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{still}}, nil); err != nil {
		t.Fatal(err)
	}
	if kind := snapshot(t, ctx, lib).Assets[0].Kind; kind != model.KindImage {
		t.Fatalf("先导入的应是普通图片：%s", kind)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{still, motion}}, nil)
	if err != nil || result.Skipped != 1 || len(result.Motion) != 1 || len(result.Failed) != 0 {
		t.Fatalf("结果不对：%+v %v", result, err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	if asset.Kind != model.KindLive || asset.MotionPath == "" || asset.DurationMS <= 0 {
		t.Fatalf("没有升级成实况照片：%+v", asset)
	}
	want, _ := os.ReadFile(motion)
	if got, err := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.MotionPath))); err != nil || !bytes.Equal(got, want) {
		t.Fatal("补存的视频不对", err)
	}

	// 同一张图配了另一段视频：这段视频记失败，源文件留着，图片照常算重复。
	other := filepath.Join(t.TempDir(), "IMG_0005")
	if err = os.MkdirAll(other, 0700); err != nil {
		t.Fatal(err)
	}
	otherStill := fixtureCopy(t, "heic", filepath.Join(other, "IMG_0005.HEIC"))
	otherMotion := filepath.Join(other, "IMG_0005.MOV")
	if err = os.WriteFile(otherMotion, append(want, 0), 0600); err != nil {
		t.Fatal(err)
	}
	result, err = s.Import(ctx, model.ImportRequest{Paths: []string{otherStill, otherMotion}}, nil)
	if err != nil || result.Skipped != 1 || len(result.Motion) != 0 || len(result.Failed) != 1 || result.Failed[0] != otherMotion {
		t.Fatalf("另一段视频应记失败：%+v %v", result, err)
	}
}

// 区分大小写的磁盘上，A/IMG 和 a/IMG 在两个文件夹，不能配对。
func TestPairLivePhotosKeepsFolderCase(t *testing.T) {
	previous := probeVideo
	probeVideo = func(string) (media.Info, error) { return media.Info{Duration: time.Second, ContentID: "ID"}, nil }
	t.Cleanup(func() { probeVideo = previous })
	fakeContentID(t, "ID")
	got := pairLivePhotos([]candidate{{path: "/x/A/IMG.JPG", extension: "jpg"}, {path: "/x/a/IMG.MOV", extension: "mov"}, {path: "/x/B/img.heic", extension: "heic"}, {path: "/x/B/IMG.MOV", extension: "mov"}})
	if len(got) != 3 || got[0].motion != nil || got[2].motion == nil || got[2].motion.path != "/x/B/IMG.MOV" {
		t.Fatalf("配对不对：%+v", got)
	}
}

// 只是同名、没有一致实况内容 ID 的照片和视频不配对：各成一个素材，视频不挂到照片上。
func TestPairLivePhotosRequiresMatchingContentID(t *testing.T) {
	previous := probeVideo
	t.Cleanup(func() { probeVideo = previous })
	previousImage := readImageContentID
	t.Cleanup(func() { readImageContentID = previousImage })
	for name, ids := range map[string]struct {
		image, video string
		imageErr     error
	}{
		"图片没有内容 ID":   {video: "ID"},
		"视频没有内容 ID":   {image: "ID"},
		"两边都没有":       {},
		"图片内容 ID 读不出": {video: "ID", imageErr: errors.New("bad exif")},
	} {
		probeVideo = func(string) (media.Info, error) { return media.Info{Duration: time.Second, ContentID: ids.video}, nil }
		readImageContentID = func(string) (string, error) { return ids.image, ids.imageErr }
		got := pairLivePhotos([]candidate{{path: "/x/IMG.JPG", extension: "jpg"}, {path: "/x/IMG.MOV", extension: "mov"}})
		if len(got) != 2 || got[0].motion != nil {
			t.Fatalf("%s：不该配成实况照片：%+v", name, got)
		}
	}
}

// 导入同名的普通照片和视频：两个都入库、都算导入，没有视频记进 Motion；自动导入据此把两个源文件当各自入库的素材处理。
func TestImportSameNameWithoutContentIDImportsSeparately(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	still, motion := livePair(t, dir, "IMG_0006")
	fakeContentID(t, "")
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{still, motion}}, nil)
	if err != nil || result.Imported != 2 || result.Skipped != 0 || len(result.Motion) != 0 || len(result.Failed) != 0 {
		t.Fatalf("只是同名应导成两个素材：%+v %v", result, err)
	}
	kinds := map[string]int{}
	for _, asset := range snapshot(t, ctx, lib).Assets {
		kinds[asset.Kind]++
		if asset.MotionPath != "" {
			t.Fatalf("普通照片不该挂视频：%+v", asset)
		}
	}
	if kinds[model.KindImage] != 1 || kinds[model.KindVideo] != 1 {
		t.Fatalf("种类不对：%v", kinds)
	}
}

// 图库里实况照片的视频文件丢了，重新导入原来那对：用这段视频补回，整对算重复，源视频记进 Motion。
// 视频位置被别的东西占着时仍然拒绝。
func TestImportRepairsMissingLiveMotion(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	dir := t.TempDir()
	still, motion := livePair(t, dir, "IMG_0007")
	if _, err := s.Import(ctx, model.ImportRequest{Paths: []string{dir}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := snapshot(t, ctx, lib).Assets[0]
	stored := filepath.Join(lib.Root, filepath.FromSlash(asset.MotionPath))
	if err := os.Remove(stored); err != nil {
		t.Fatal(err)
	}
	result, err := s.Import(ctx, model.ImportRequest{Paths: []string{still, motion}}, nil)
	if err != nil || result.Skipped != 1 || len(result.Motion) != 1 || result.Motion[0] != motion || len(result.Failed) != 0 {
		t.Fatalf("重导应补回视频：%+v %v", result, err)
	}
	want, _ := os.ReadFile(motion)
	if got, err := os.ReadFile(stored); err != nil || !bytes.Equal(got, want) {
		t.Fatal("补回的视频不对", err)
	}
	if after := snapshot(t, ctx, lib).Assets[0]; after.Kind != model.KindLive || after.MotionPath != asset.MotionPath {
		t.Fatalf("补回后记录不对：%+v", after)
	}

	// 视频位置上是一个指向别处的失效链接：不算缺失，不往链接里写，整批拒绝。
	if err = os.Remove(stored); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.mov")
	if err = os.Symlink(outside, stored); err != nil {
		t.Fatal(err)
	}
	_, err = s.Import(ctx, model.ImportRequest{Paths: []string{still, motion}}, nil)
	var localized *i18n.LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_import_unregistered_file" {
		t.Fatalf("视频位置被占时应拒绝：%v", err)
	}
	if _, statErr := os.Stat(outside); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("不该顺着链接写到图库外：%v", statErr)
	}
}

// 带视频导出时库里的视频丢了：单张替换已有图片，旧图片原样保留；导出到文件夹不留孤立的静态图；两处都不留暂存文件。
func TestExportLivePhotoMotionFailureLeavesNothing(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	asset := importLive(t, s, lib)
	names := func(dir string) []string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, e := range entries {
			out = append(out, e.Name())
		}
		return out
	}

	// 先确认正常替换：图片换成新的，视频放在旁边。
	replaced := filepath.Join(t.TempDir(), "photo.heic")
	if err := os.WriteFile(replaced, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExportWithMotion(ctx, []model.Asset{asset}, replaced, model.ExportOriginal, .9, true); err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(filepath.Join(lib.Root, filepath.FromSlash(asset.RelativePath)))
	if got, _ := os.ReadFile(replaced); !bytes.Equal(got, want) {
		t.Fatal("确认替换后图片没换成新的")
	}
	if got := names(filepath.Dir(replaced)); len(got) != 2 {
		t.Fatalf("替换导出后应只有图片和视频：%v", got)
	}

	if err := os.Remove(filepath.Join(lib.Root, filepath.FromSlash(asset.MotionPath))); err != nil {
		t.Fatal(err)
	}
	single := filepath.Join(t.TempDir(), "photo.heic")
	if err := os.WriteFile(single, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExportWithMotion(ctx, []model.Asset{asset}, single, model.ExportOriginal, .9, true); err == nil {
		t.Fatal("视频丢了应报错")
	}
	if got, _ := os.ReadFile(single); string(got) != "old" {
		t.Fatal("导出失败却换掉了旧图片")
	}
	if got := names(filepath.Dir(single)); len(got) != 1 {
		t.Fatalf("失败后留下了别的文件：%v", got)
	}

	folder := t.TempDir()
	if _, err := s.ExportWithMotion(ctx, []model.Asset{asset}, folder, model.ExportOriginal, .9, true); err == nil {
		t.Fatal("视频丢了应报错")
	}
	if got := names(folder); len(got) != 0 {
		t.Fatalf("导出到文件夹失败后留下了文件：%v", got)
	}
}
