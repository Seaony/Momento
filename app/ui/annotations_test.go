package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 存下的框换到显示位置再换回来要一样；转 90° 时左上角的框跑到右上角。
func TestAnnotationViewRoundTrip(t *testing.T) {
	note := model.Annotation{X: .1, Y: .2, W: .3, H: .4}
	for _, rotation := range []int{0, 90, 180, 270} {
		for _, flipped := range []bool{false, true} {
			x, y, w, h := annotationView(note, rotation, flipped)
			back := annotationStored(x+w, y+h, x, y, rotation, flipped)
			if math.Abs(back.X-note.X)+math.Abs(back.Y-note.Y)+math.Abs(back.W-note.W)+math.Abs(back.H-note.H) > 1e-9 {
				t.Fatalf("旋转 %d 翻转 %v 换回来不一样：%+v", rotation, flipped, back)
			}
		}
	}
	if x, y, w, h := annotationView(model.Annotation{X: 0, Y: 0, W: .2, H: .1}, 90, false); math.Abs(x-.9) > 1e-9 || y != 0 || math.Abs(w-.1) > 1e-9 || math.Abs(h-.2) > 1e-9 {
		t.Fatalf("顺时针转 90° 后左上角的框应在右上角：%v %v %v %v", x, y, w, h)
	}
	if got := annotationStored(-.5, -.5, .5, .5, 0, false); got.X != 0 || got.Y != 0 || got.W != .5 || got.H != .5 {
		t.Fatalf("拖到图外的部分应收进图里：%+v", got)
	}
}

// annotationFixture 建一个真实图库，导入 count 张图，打开第一张的预览；写入都排队，runQueued 按顺序执行并核对内存和库里一致。
func annotationFixture(t *testing.T, count int) (*App, *previewWindow, func()) {
	t.Helper()
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Notes.momento"), "Notes")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { lib.Close() })
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 240, 160))
	var paths []string
	for n := range count {
		for i := range img.Pix {
			img.Pix[i] = 200
		}
		img.Set(n, 0, color.NRGBA{A: 255})
		path := filepath.Join(dir, fmt.Sprintf("board-%d.png", n))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
		paths = append(paths, path)
	}
	if _, err := assets.New(lib.Root, lib.DB).Import(ctx, model.ImportRequest{Paths: paths}, nil); err != nil {
		t.Fatal(err)
	}
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Library: lib, Texts: i18n.Catalog{Language: "zh-CN"}, Preferences: DefaultPreferences()}
	a.applySnapshot(snapshot)
	asset := a.Visible[0]
	w := &previewWindow{root: lib.Root, asset: asset, items: append([]model.Asset(nil), a.Visible...), hasFocus: true}
	w.picture.Open = true
	w.picture.target(w.root, asset.ID, 0)
	w.picture.Bitmap = ui.NewBitmap(img)
	a.Preview.active = w
	// 写入排队，测试自己按顺序执行，好检查每一步。
	a.Busy = true
	runQueued := func() {
		t.Helper()
		for len(a.queuedRuns) > 0 {
			run := a.queuedRuns[0]
			a.queuedRuns = a.queuedRuns[1:]
			if err := run.work(ctx); err != nil {
				run.patch.failed(err)
				continue
			}
			a.patchAssets(run.patch.ids, run.patch.apply)
		}
		fresh, err := lib.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range fresh.Assets {
			got := a.assetByID[want.ID].Annotations
			if len(got) != len(want.Annotations) {
				t.Fatalf("内存 %d 条批注，库里 %d 条", len(got), len(want.Annotations))
			}
			for i := range got {
				if got[i].ID != want.Annotations[i].ID || got[i].Text != want.Annotations[i].Text || math.Abs(got[i].X-want.Annotations[i].X) > 1e-9 {
					t.Fatalf("内存和库里的批注不同：%+v / %+v", got[i], want.Annotations[i])
				}
			}
		}
	}
	return a, w, runQueued
}

// 预览里按 A 进批注模式，在图上拖出框、写字、回车存下；点框改字；删除；Esc 一层层退出。每一步内存和数据库一致。
func TestPreviewAnnotationFlow(t *testing.T) {
	a, w, runQueued := annotationFixture(t, 1)
	asset := a.Visible[0]
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 400, 320)
	tt.Key(0, ui.KeyA)
	if !w.notes.On {
		t.Fatal("按 A 没有进入批注模式")
	}
	tt.Frame()
	photo := w.picture.imageRect(w.picture.viewport)
	at := func(u, v float32) (float32, float32) { return photo.X + u*photo.W, photo.Y + v*photo.H }
	tt.Press(at(.2, .2))
	tt.Move(at(.4, .5))
	tt.Move(at(.6, .7))
	tt.Release(at(.6, .7))
	tt.Frame()
	edit := w.notes.edit
	if edit == nil || edit.ID != "" || math.Abs(edit.Rect.X-.2) > .01 || math.Abs(edit.Rect.W-.4) > .01 || math.Abs(edit.Rect.H-.5) > .01 {
		t.Fatalf("拖出的框不对：%+v", edit)
	}
	tt.Type("logo 太小")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if w.notes.edit != nil || len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "annotation_add" {
		t.Fatalf("回车没有存下批注：%+v", a.queuedRuns)
	}
	runQueued()
	if notes := a.assetByID[asset.ID].Annotations; len(notes) != 1 || notes[0].Text != "logo 太小" {
		t.Fatalf("存下的批注不对：%+v", notes)
	}
	// 点一下框改字。
	tt.Frame()
	tt.ClickAt(at(.4, .4))
	tt.Frame()
	if w.notes.edit == nil || w.notes.edit.Text != "logo 太小" {
		t.Fatal("点框没有打开它的文字")
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("logo 再大一点")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	runQueued()
	if notes := a.assetByID[asset.ID].Annotations; notes[0].Text != "logo 再大一点" {
		t.Fatalf("改字没生效：%+v", notes)
	}
	// Esc 先收起输入框，再退出批注模式，窗口还开着。
	tt.ClickAt(at(.4, .4))
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if w.notes.edit != nil || !w.notes.On || w.closing {
		t.Fatal("第一次 Esc 应只收起输入框")
	}
	tt.Key(0, ui.KeyEscape)
	if w.notes.On || w.closing {
		t.Fatal("第二次 Esc 应只退出批注模式")
	}
	// 删除。
	tt.Key(0, ui.KeyA)
	tt.Frame()
	tt.ClickAt(at(.4, .4))
	tt.Frame()
	if err := tt.Click(a.T("delete")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	runQueued()
	if len(a.assetByID[asset.ID].Annotations) != 0 {
		t.Fatal("删除没生效")
	}
	// 写到一半退出：先存下，等写完再退。
	tt.Press(at(.1, .1))
	tt.Move(at(.3, .3))
	tt.Release(at(.3, .3))
	tt.Frame()
	tt.Type("退出前写的")
	if a.PrepareInspectorQuit() {
		t.Fatal("还有没存的批注时不该马上退出")
	}
	runQueued()
	if notes := a.assetByID[asset.ID].Annotations; len(notes) != 1 || notes[0].Text != "退出前写的" {
		t.Fatalf("退出前没把批注存下：%+v", notes)
	}
}

// 审查发现的几种丢字情况：改到一半再点同一个框、写失败、锁定、已开着别的图的预览、主窗口按 Esc。
func TestAnnotationDraftsAreNotLost(t *testing.T) {
	a, w, runQueued := annotationFixture(t, 2)
	ctx := context.Background()
	first, second := a.Visible[0], a.Visible[1]
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 400, 320)
	tt.Key(0, ui.KeyA)
	tt.Frame()
	photo := w.picture.imageRect(w.picture.viewport)
	at := func(u, v float32) (float32, float32) { return photo.X + u*photo.W, photo.Y + v*photo.H }
	tt.Press(at(.2, .2))
	tt.Move(at(.6, .6))
	tt.Release(at(.6, .6))
	tt.Frame()
	tt.Type("原来的字")
	tt.Key(0, ui.KeyEnter)
	runQueued()
	note := a.assetByID[first.ID].Annotations[0]

	// 改了一半又点同一个框：输入框里的字留着。
	tt.Frame()
	tt.ClickAt(at(.4, .4))
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("改了一半")
	tt.ClickAt(at(.5, .55)) // 框里、输入面板外
	tt.Frame()
	if w.notes.edit == nil || w.notes.edit.ID != note.ID || w.notes.edit.Text != "改了一半" {
		t.Fatalf("再点同一个框不该丢掉改了一半的字：%+v", w.notes.edit)
	}
	if err := tt.Click(a.T("save")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()

	// 写失败：留在待存列表里，退出被拦下并给出放弃的出口；放弃前不能退出。
	if _, err := a.Library.DB.Exec("DELETE FROM asset_annotations WHERE id=?", note.ID); err != nil {
		t.Fatal(err)
	}
	runQueued0 := a.queuedRuns[0]
	a.queuedRuns = a.queuedRuns[1:]
	err := runQueued0.work(ctx)
	if err == nil {
		t.Fatal("批注被删后改字应该写失败")
	}
	runQueued0.patch.failed(err)
	if !a.annotationSavesFailed() || !a.InspectorEditsFailed() || a.PrepareInspectorQuit() {
		t.Fatal("批注写失败后应拦下退出")
	}
	// 再次退出会重试一次（又排进队列），仍失败就还在列表里。
	if len(a.queuedRuns) != 1 {
		t.Fatalf("再次退出应重试写入：%d", len(a.queuedRuns))
	}
	retry := a.queuedRuns[0]
	a.queuedRuns = nil
	if err := retry.work(ctx); err == nil {
		t.Fatal("重试仍应失败")
	} else {
		retry.patch.failed(err)
	}
	if !a.DiscardInspectorEdits("test") || a.annotationSavesPending() {
		t.Fatal("放弃后待存列表应清空")
	}
	// 数据库里那条已经没了，内存也同步一下再继续。
	fresh, _ := a.Library.Snapshot(ctx)
	a.applySnapshot(fresh)

	// 锁定（关掉所有预览窗口）前先存下正在写的。
	tt.Press(at(.1, .1))
	tt.Move(at(.3, .3))
	tt.Release(at(.3, .3))
	tt.Frame()
	tt.Type("锁定前写的")
	a.CloseAllPreviewWindows()
	runQueued()
	if notes := a.assetByID[first.ID].Annotations; len(notes) != 1 || notes[0].Text != "锁定前写的" {
		t.Fatalf("关窗口前没存下批注：%+v", notes)
	}

	// 已开着第一张的预览，从第二张的入口加批注：预览换成第二张并进入批注模式。
	a.Preview.active = w
	w.closing, w.disposed = false, false
	a.AnnotateAsset(second.ID, "")
	if a.Preview.active != w || w.asset.ID != second.ID || !w.notes.On {
		t.Fatal("批注入口没有把已开的预览换到这张图")
	}

	// 主窗口有焦点时按 Esc 也一层层退。
	w.notes.edit = &annotationEdit{Rect: model.Annotation{X: .1, Y: .1, W: .2, H: .2}}
	main := ui.NewTester(func(c *ui.Context) { a.ReconcilePreviewKeys(c) }, 200, 200)
	main.Key(0, ui.KeyEscape)
	if w.notes.edit != nil || !w.notes.On || w.closing {
		t.Fatal("主窗口按 Esc 应先收起输入框")
	}
	main.Key(0, ui.KeyEscape)
	if w.notes.On || w.closing {
		t.Fatal("主窗口第二次 Esc 应退出批注模式")
	}
}

// 复核发现的：旧的失败写入不能在重试时覆盖后来存好的字；换图库期间不能改；图库没了时不再发起写入；
// 从检查器点批注不覆盖预览里正在写的字。
func TestAnnotationSavesAfterReview(t *testing.T) {
	a, w, runQueued := annotationFixture(t, 1)
	asset := a.Visible[0]
	a.addAnnotation(asset.ID, model.Annotation{X: .1, Y: .1, W: .3, H: .3, Text: "原来"})
	runQueued()
	note := a.assetByID[asset.ID].Annotations[0]

	// 先改成「甲」写失败，再改成「乙」写成功；之后重试不能把「乙」改回「甲」。
	a.updateAnnotation(asset.ID, note.ID, "甲")
	failed := a.queuedRuns[0]
	a.queuedRuns = nil
	failed.patch.failed(errors.New("disk full"))
	a.updateAnnotation(asset.ID, note.ID, "乙")
	runQueued()
	a.retryAnnotationSaves()
	runQueued()
	if got := a.assetByID[asset.ID].Annotations[0].Text; got != "乙" || a.annotationSavesPending() {
		t.Fatalf("重试把新写的字改回去了：%q，待存 %d", got, len(a.annotationSaves))
	}

	// 从检查器点同一条：保留预览里改了一半的字；点别的条目前先存下。
	w.notes.On = true
	w.notes.edit = &annotationEdit{ID: note.ID, Rect: note, Text: "改了一半"}
	a.AnnotateAsset(asset.ID, note.ID)
	if w.notes.edit == nil || w.notes.edit.Text != "改了一半" {
		t.Fatal("检查器入口覆盖了正在写的字")
	}
	w.notes.edit = &annotationEdit{Rect: model.Annotation{X: .5, Y: .5, W: .2, H: .2}, Text: "新框"}
	a.AnnotateAsset(asset.ID, note.ID)
	runQueued()
	if notes := a.assetByID[asset.ID].Annotations; len(notes) != 2 || notes[1].Text != "新框" {
		t.Fatalf("换条目前没有存下正在写的新框：%+v", notes)
	}

	// 后台正在打开另一个图库：不能画框、不能写。
	a.libraryOpening = true
	if w.annotatable(a) {
		t.Fatal("换图库期间不该能改批注")
	}
	a.libraryOpening = false

	// 图库没了：正在写的记日志丢掉，不发起写入，也不让 Busy 挂住。
	w.notes.edit = &annotationEdit{Rect: model.Annotation{X: .1, Y: .6, W: .2, H: .2}, Text: "拔盘前写的"}
	a.Busy = false
	if !a.closeLibrary("library_missing") {
		t.Fatal("丢掉的批注草稿应算进放弃的修改")
	}
	if a.Busy || len(a.queuedRuns) != 0 || a.annotationSavesPending() {
		t.Fatal("图库没了时不该再发起批注写入")
	}
}
