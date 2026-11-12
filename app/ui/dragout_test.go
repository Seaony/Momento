package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

func dragAsset(t *testing.T, root, relative, original, name string) model.Asset {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("original:"+relative), 0600); err != nil {
		t.Fatal(err)
	}
	return model.Asset{RelativePath: filepath.ToSlash(relative), OriginalFileName: original, Name: name, Extension: "png"}
}

func TestPrepareFileDragPreservesNamesAndOriginals(t *testing.T) {
	root := t.TempDir()
	items := []model.Asset{
		dragAsset(t, root, "assets/aa/one.png", "原图.png", "修改过的标题"),
		dragAsset(t, root, "assets/bb/two.png", "原图.png", "另一个标题"),
		dragAsset(t, root, "assets/cc/three.png", "", "没有原始文件名"),
	}
	session, err := prepareFileDrag(root, items, os.Link)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(session.directory) != filepath.Join(canonical, ".drag-out") {
		t.Fatalf("临时目录在图库之外：%s", session.directory)
	}
	want := []string{"原图.png", "原图 2.png", "没有原始文件名.png"}
	for i, path := range session.paths {
		if filepath.Base(path) != want[i] {
			t.Fatalf("文件名 %q，想要 %q", filepath.Base(path), want[i])
		}
		original, err := os.Stat(filepath.Join(root, filepath.FromSlash(items[i].RelativePath)))
		if err != nil {
			t.Fatal(err)
		}
		dragged, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(original, dragged) {
			t.Fatal("没有使用硬链接")
		}
	}
	data, err := transfer.FileData(session.paths...)
	if err != nil {
		t.Fatal(err)
	}
	files, err := data.Files()
	if err != nil || len(files) != len(items) {
		t.Fatalf("拖出路径: %v %v", files, err)
	}
	if err := session.cleanup(); err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(item.RelativePath)))
		if err != nil || !strings.HasPrefix(string(contents), "original:") {
			t.Fatalf("原图受到影响：%q %v", contents, err)
		}
	}
	if _, err := os.Stat(session.directory); !os.IsNotExist(err) {
		t.Fatalf("会话目录没有清理：%v", err)
	}
}

func TestPrepareFileDragCopiesWhenHardLinkFails(t *testing.T) {
	root := t.TempDir()
	item := dragAsset(t, root, "assets/image.png", "image.png", "image")
	links := 0
	session, err := prepareFileDrag(root, []model.Asset{item}, func(string, string) error { links++; return os.ErrPermission })
	if err != nil {
		t.Fatal(err)
	}
	defer session.cleanup()
	source := filepath.Join(root, filepath.FromSlash(item.RelativePath))
	a, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(session.paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(a, b) || links != 1 {
		t.Fatal("没有走复制回退")
	}
	got, err := os.ReadFile(session.paths[0])
	if err != nil || string(got) != "original:assets/image.png" {
		t.Fatalf("复制内容 %q %v", got, err)
	}
}

func TestFailedFileDragRemovesOnlyItsSession(t *testing.T) {
	root := t.TempDir()
	first := dragAsset(t, root, "assets/image.png", "image.png", "image")
	other, err := prepareFileDrag(root, []model.Asset{first}, os.Link)
	if err != nil {
		t.Fatal(err)
	}
	defer other.cleanup()
	missing := model.Asset{RelativePath: "assets/missing.png", Name: "missing", Extension: "png"}
	failed, err := prepareFileDrag(root, []model.Asset{first, missing}, os.Link)
	if err == nil || failed != nil {
		t.Fatal("缺失原图没有使整次拖出失败")
	}
	entries, err := os.ReadDir(filepath.Join(root, ".drag-out"))
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(other.directory) {
		t.Fatalf("失败清理误删其他会话：%v %v", entries, err)
	}
	if _, err := os.Stat(other.paths[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "assets/image.png")); err != nil {
		t.Fatal(err)
	}
}

func TestFileDragRejectsPathsOutsideLibrary(t *testing.T) {
	root := t.TempDir()
	outside := dragAsset(t, t.TempDir(), "outside.png", "outside.png", "outside")
	for _, relative := range []string{"../outside.png", filepath.Join(t.TempDir(), "absolute.png")} {
		if session, err := prepareFileDrag(root, []model.Asset{{RelativePath: relative}}, os.Link); err == nil || session != nil {
			t.Fatalf("接受了越界路径 %q", relative)
		}
	}
	outsidePath := filepath.Join(t.TempDir(), "external.png")
	if err := os.WriteFile(outsidePath, []byte(outside.Name), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, filepath.Join(root, "symlink.png")); err != nil {
		t.Fatal(err)
	}
	if session, err := prepareFileDrag(root, []model.Asset{{RelativePath: "symlink.png"}}, os.Link); err == nil || session != nil {
		t.Fatal("接受了指向库外的符号链接")
	}
}

func TestFileDragRejectsSymlinkTemporaryDirectory(t *testing.T) {
	root := t.TempDir()
	item := dragAsset(t, root, "assets/image.png", "image.png", "image")
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".drag-out")); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareFileDrag(root, []model.Asset{item}, os.Link); err == nil {
		t.Fatal("接受了指向库外的拖出目录")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatal("写入了库外目录")
	}
}

func TestFileDragCaseInsensitiveCollisionAndSafeName(t *testing.T) {
	root := t.TempDir()
	items := []model.Asset{
		dragAsset(t, root, "assets/a.png", "Photo.png", ""),
		dragAsset(t, root, "assets/b.png", "photo.png", ""),
		dragAsset(t, root, "assets/c.png", "../../escape.png", ""),
	}
	session, err := prepareFileDrag(root, items, os.Link)
	if err != nil {
		t.Fatal(err)
	}
	defer session.cleanup()
	if filepath.Base(session.paths[1]) != "photo 2.png" {
		t.Fatal("没有处理大小写重名")
	}
	for _, path := range session.paths {
		if filepath.Dir(path) != session.directory {
			t.Fatalf("文件名逃逸会话目录：%s", path)
		}
	}
	if _, err := prepareFileDrag(root, nil, os.Link); err == nil {
		t.Fatal("接受了空选择")
	}
}

func TestCopyDragFileNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "source"), filepath.Join(dir, "existing")
	if err := os.WriteFile(source, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := copyDragFile(source, destination); !errors.Is(err, os.ErrExist) {
		t.Fatalf("覆盖已有文件：%v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "keep" {
		t.Fatal("已有文件被修改")
	}
}

func TestAttachFileDragPreparesAtGestureAndCleansOnEscape(t *testing.T) {
	root := t.TempDir()
	item := dragAsset(t, root, "assets/image.png", "image.png", "image")
	var reported error
	selections := 0
	tt := ui.NewTester(func(c *ui.Context) {
		element := ui.Box(c).Size(100, 60).Label("source").Drag("internal-asset")
		AttachFileDrag(element, root, func() []model.Asset { selections++; return []model.Asset{item} }, func(err error) { reported = err })
	}, 200, 100)
	if selections != 0 {
		t.Fatal("渲染界面时就准备了文件")
	}
	tt.Press(10, 10)
	tt.Move(50, 20)
	if reported != nil {
		t.Fatal(reported)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".drag-out"))
	if err != nil || len(entries) != 1 || selections != 1 {
		t.Fatalf("拖动没有准备文件：%v %v，读取选择 %d 次", entries, err, selections)
	}
	tt.Key(0, ui.KeyEscape)
	entries, err = os.ReadDir(filepath.Join(root, ".drag-out"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("取消拖动没有清理：%v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(root, "assets/image.png")); err != nil {
		t.Fatal(err)
	}
}
