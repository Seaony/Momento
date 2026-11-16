package ui

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 数字键给选中的素材打星，Shift+数字打完跳到下一张；输入框有焦点时不抢键。
// Busy 时写入排队，不起后台任务，正好拿来检查排进去的是什么。
func TestRatingKeys(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	tt := ui.NewTester(a.assetArea, 700, 600)
	if err := tt.Click("Photo 0"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.Key3)
	tt.Frame()
	if len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "rating" || len(a.queuedRuns[0].patch.ids) != 1 || a.queuedRuns[0].patch.ids[0] != "0" {
		t.Fatalf("按 3 没有给选中的素材打星：%+v", a.queuedRuns)
	}
	asset := model.Asset{}
	a.queuedRuns[0].patch.apply(&asset)
	if asset.Rating != 3 {
		t.Fatalf("打的星数不对：%d", asset.Rating)
	}
	tt.Key(ui.Shift, ui.Key5)
	tt.Frame()
	if len(a.queuedRuns) != 2 || a.queuedRuns[1].patch.ids[0] != "0" || !a.Selected.Has(a.Visible[1].ID) || a.Selected.Len() != 1 {
		t.Fatalf("Shift+5 没有打完跳到下一张：选中 %v", a.selectedIDs())
	}
	tt.Key(0, ui.Key0)
	tt.Frame()
	a.queuedRuns[2].patch.apply(&asset)
	if asset.Rating != 0 || a.queuedRuns[2].patch.ids[0] != a.Visible[1].ID {
		t.Fatal("按 0 没有清除下一张的评分")
	}
}

func TestRenameDialogQueuesOnlyChangedNames(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = false
	ids := []string{a.Visible[0].ID, a.Visible[1].ID, a.Visible[2].ID}
	a.OpenRename(ids)
	a.Rename.Mode, a.Rename.Find, a.Rename.With = 1, "Photo 1", "Cover"
	assets := a.renameAssets()
	names, changed, problem := a.renamePlan(assets)
	if problem != "" || changed != 1 || names[1] != "Cover" || names[0] != "Photo 0" {
		t.Fatalf("替换结果不对：%v，改了 %d 个，问题 %q", names, changed, problem)
	}
	a.Rename.Mode, a.Rename.Template = 0, "  "
	if _, _, problem := a.renamePlan(assets); problem == "" {
		t.Fatal("新名称为空时没有拦下")
	}
	a.Rename.Template, a.Rename.Start = "{n}", "-1"
	if _, _, problem := a.renamePlan(assets); problem == "" {
		t.Fatal("起始序号是负数时没有拦下")
	}
}

// 父文件夹顶部列出子文件夹，点进去；勾上「显示子文件夹内容」后父文件夹也列出子文件夹里的素材。
func TestSubfolderStrip(t *testing.T) {
	a := sidebarTestApp()
	a.Snapshot.Assets = append(a.Snapshot.Assets, model.Asset{ID: "d", Name: "Nested", FolderIDs: []string{"child"}})
	a.applySnapshot(a.Snapshot)
	a.SetScope("folder", "root")
	tt := ui.NewTester(a.assetArea, 800, 600)
	if len(a.Visible) != 1 || !tt.HasText("UI details") {
		t.Fatalf("父文件夹应只列直接放进去的素材并显示子文件夹：%d", len(a.Visible))
	}
	if err := tt.Click(a.T("show_subfolder_contents")); err != nil {
		t.Fatal(err)
	}
	if !a.Preferences.SubfolderContents || len(a.Visible) != 2 {
		t.Fatalf("勾选后应包括子文件夹素材：%d", len(a.Visible))
	}
	if err := tt.Click("UI details"); err != nil {
		t.Fatal(err)
	}
	if a.Query.FolderID != "child" || len(a.Visible) != 1 || a.Visible[0].ID != "d" {
		t.Fatalf("点子文件夹卡片没有进去：%s %d", a.Query.FolderID, len(a.Visible))
	}
}

// 多选一起改备注：值不一样时留空不算清空；中途选择变成单张也按开始时那批存。
func TestMultiDetailsEdit(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	snapshot := a.Snapshot
	snapshot.Assets[0].Note, snapshot.Assets[1].Note = "旧备注", "另一条"
	a.applySnapshot(snapshot)
	ids := []string{a.Visible[0].ID, a.Visible[1].ID}
	for _, id := range ids {
		a.Selected.Add(id)
	}
	a.Inspector.Editing, a.Inspector.MultiIDs, a.Inspector.MultiMixed, a.Inspector.MultiDraft = "multi-note", ids, true, ""
	a.applyMultiDetail("note")
	if len(a.queuedRuns) != 0 {
		t.Fatal("备注不一样时留空结束编辑，不应清空所有备注")
	}
	a.Inspector.MultiDraft = "统一备注"
	a.Selected.Clear()
	a.Selected.Add(a.Visible[3].ID)
	tt := ui.NewTester(a.inspector, 320, 800)
	tt.Frame()
	if len(a.queuedRuns) != 1 || len(a.queuedRuns[0].patch.ids) != 2 || a.queuedRuns[0].patch.ids[0] != ids[0] || a.Inspector.Editing != "" {
		t.Fatalf("换成单选后没有按开始时那批存下：%+v", a.queuedRuns)
	}
	var asset model.Asset
	a.queuedRuns[0].patch.apply(&asset)
	if asset.Note != "统一备注" {
		t.Fatalf("备注内容不对：%q", asset.Note)
	}
}

// ⌘⇧C 复制选中素材的标签并集，⌘⇧V 加到别的素材上，已删除的标签跳过。
func TestCopyPasteTags(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	snapshot := a.Snapshot
	snapshot.Tags = []model.Tag{{ID: "t1", Name: "山"}, {ID: "t2", Name: "夜景"}}
	snapshot.Assets[0].TagIDs = []string{"t1"}
	snapshot.Assets[1].TagIDs = []string{"t2", "t1"}
	a.applySnapshot(snapshot)
	a.Selected.Add(a.Visible[0].ID)
	a.Selected.Add(a.Visible[1].ID)
	a.CopyTags()
	if len(a.tagClipboard) != 2 {
		t.Fatalf("复制的标签不对：%v", a.tagClipboard)
	}
	a.Snapshot.Tags = a.Snapshot.Tags[:1]
	a.Selected.Clear()
	a.Selected.Add(a.Visible[4].ID)
	a.PasteTags()
	if len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "tags" {
		t.Fatalf("粘贴标签没有排进写入：%+v", a.queuedRuns)
	}
	var asset model.Asset
	a.queuedRuns[0].patch.apply(&asset)
	if len(asset.TagIDs) != 1 || asset.TagIDs[0] != "t1" {
		t.Fatalf("应只加上还存在的标签：%v", asset.TagIDs)
	}
}

// 多选一起改备注还没失焦就退出：先按开始时那批存，存完之前不放行退出。
func TestMultiDetailsFlushOnQuit(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	ids := []string{a.Visible[0].ID, a.Visible[1].ID}
	a.Inspector.Editing, a.Inspector.MultiIDs, a.Inspector.MultiDraft = "multi-note", ids, "退出前写的"
	if a.PrepareInspectorQuit() {
		t.Fatal("多选编辑还没存就放行了退出")
	}
	if len(a.queuedRuns) != 1 || !a.Inspector.MultiSaving || a.Inspector.Editing != "" {
		t.Fatalf("退出前没有把多选编辑存下：%+v", a.queuedRuns)
	}
}
