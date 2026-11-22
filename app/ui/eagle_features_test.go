package ui

import (
	"errors"
	"slices"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestRenameDialogQueuesOnlyChangedNames(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = false
	ids := []string{a.Visible[0].ID, a.Visible[1].ID, a.Visible[2].ID}
	a.OpenRename(ids)
	source := &renameSource{assets: a.renameAssets(), tagNames: a.renameTagNames()}
	a.Rename.Mode, a.Rename.Find, a.Rename.With = 1, "Photo 1", "Cover"
	plan := (&renameRequest{inputs: a.renameInputs(), source: source}).calculate()
	if plan.problem != "" || len(plan.ids) != 1 || plan.ids[0] != ids[1] || plan.updates[ids[1]] != "Cover" {
		t.Fatalf("只应改第二张的名字：%+v", plan)
	}
	a.Rename.Mode, a.Rename.Template = 0, "  "
	if plan := (&renameRequest{inputs: a.renameInputs(), source: source}).calculate(); plan.problem == "" {
		t.Fatal("新名称为空时没有拦下")
	}
	a.Rename.Template, a.Rename.Start = "{n}", "-1"
	if plan := (&renameRequest{inputs: a.renameInputs(), source: source}).calculate(); plan.problem == "" {
		t.Fatal("起始序号是负数时没有拦下")
	}
}

// 右键重命名：只选一张时是预填原名的改名框，选多张才是批量重命名。
func TestRenameMenuSingleVersusMultiple(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	tt := ui.NewTester(a.assetArea, 700, 600)
	if err := tt.RightClick("Photo 0"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(a.T("rename_assets")); err != nil {
		t.Fatal(err)
	}
	if !a.PromptOpen || a.PromptValue != "Photo 0" || a.Rename.Open {
		t.Fatalf("单张应打开改名框：prompt %v %q，批量 %v", a.PromptOpen, a.PromptValue, a.Rename.Open)
	}
	a.PromptOpen, a.PromptSubmit = false, nil
	a.Selected.Add(a.Visible[0].ID)
	a.Selected.Add(a.Visible[1].ID)
	tt.Frame()
	if err := tt.RightClick("Photo 1"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(a.T("rename_assets")); err != nil {
		t.Fatal(err)
	}
	if a.PromptOpen || !a.Rename.Open || len(a.Rename.IDs) != 2 {
		t.Fatalf("多张应打开批量重命名：prompt %v，批量 %v %v", a.PromptOpen, a.Rename.Open, a.Rename.IDs)
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
	if len(a.Inspector.MultiPending) != 0 {
		t.Fatal("备注不一样时留空结束编辑，不应清空所有备注")
	}
	a.Inspector.MultiDraft = "统一备注"
	a.Selected.Clear()
	a.Selected.Add(a.Visible[3].ID)
	tt := ui.NewTester(a.inspector, 320, 800)
	tt.Frame()
	pending := a.Inspector.MultiPending
	if len(pending) != 1 || len(pending[0].IDs) != 2 || pending[0].IDs[0] != ids[0] || a.Inspector.Editing != "" {
		t.Fatalf("换成单选后没有按开始时那批存下：%+v", pending)
	}
	if pending[0].Field != "note" || pending[0].Value != "统一备注" {
		t.Fatalf("备注内容不对：%+v", pending[0])
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
	if len(a.Inspector.MultiPending) != 1 || a.Inspector.MultiPending[0].Value != "退出前写的" || a.Inspector.Editing != "" {
		t.Fatalf("退出前没有把多选编辑存下：%+v", a.Inspector.MultiPending)
	}
}

// 多选一起改备注写入失败：修改留在队列里标成失败，界面照旧显示用户写的内容，重新提交时再写一次。
func TestMultiDetailsKeepDraftAfterSaveFailure(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	ids := []string{a.Visible[0].ID, a.Visible[1].ID}
	for _, id := range ids {
		a.Selected.Add(id)
	}
	a.Inspector.Editing, a.Inspector.MultiIDs, a.Inspector.MultiDraft = "multi-note", ids, "写不进去的备注"
	a.FlushInspectorEdits()
	if len(a.Inspector.MultiPending) != 1 {
		t.Fatalf("多选修改没有进待保存队列：%+v", a.Inspector.MultiPending)
	}
	a.inspectorMultiEditFinished(a.Inspector.MultiPending[0], errors.New("disk full"))
	if !a.InspectorEditsFailed() || !a.InspectorEditsPending() || a.Inspector.SaveError == "" {
		t.Fatal("写入失败后没有留下可重试的修改")
	}
	if edit, ok := a.multiPendingFor("note", a.SelectedAssets()); !ok || edit.Value != "写不进去的备注" {
		t.Fatalf("界面没按未写入的内容显示：%+v", edit)
	}
	a.FlushInspectorEdits()
	if a.InspectorEditsFailed() || len(a.Inspector.MultiPending) != 1 {
		t.Fatal("再次提交时没有重试失败的多选修改")
	}
	a.inspectorMultiEditFinished(a.Inspector.MultiPending[0], nil)
	if a.InspectorEditsPending() {
		t.Fatal("写入成功后修改还留在队列里")
	}
}

// 相似图对话框：点别的缩略图换成保留它，合并这一组排进写入队列，保留的那张拿到其余的标签，其余进回收站；
// 这一组马上从对话框里拿掉，剩下的组还在。
func TestSimilarDialogMerge(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	snapshot := a.Snapshot
	snapshot.Assets[1].TagIDs = []string{"t1"}
	a.applySnapshot(snapshot)
	a.Similar = similarState{Open: true, Groups: []similarGroup{
		{IDs: []string{"0", "1", "2"}, Keep: "0"},
		{IDs: []string{"3", "4"}, Keep: "3"},
	}}
	tt := ui.NewTester(a.similarDialog, 800, 760)
	if !tt.HasText(a.T("similar_title", 2)) {
		t.Fatal("没有显示组数")
	}
	if err := tt.Click("Photo 1"); err != nil {
		t.Fatal(err)
	}
	if a.Similar.Groups[0].Keep != "1" {
		t.Fatalf("点缩略图没有换成保留它：%q", a.Similar.Groups[0].Keep)
	}
	// 点第一组的「合并」：合并要等这一帧画完再做，不能边画边删组（原来在这里越界）。
	if err := tt.Click(a.T("similar_merge")); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "merge_similar" {
		t.Fatalf("合并没有排进写入队列：%+v", a.queuedRuns)
	}
	if len(a.Similar.Groups) != 1 || a.Similar.Groups[0].Keep != "3" {
		t.Fatalf("合并的组应从对话框拿掉：%+v", a.Similar.Groups)
	}
	run := a.queuedRuns[0]
	a.patchAssets(run.patch.ids, run.patch.apply)
	keep, other := a.assetByID["1"], a.assetByID["2"]
	if keep.Trashed || len(keep.TagIDs) != 1 || !other.Trashed || !a.assetByID["0"].Trashed || a.assetByID["3"].Trashed {
		t.Fatalf("合并结果不对：保留 %+v，其余 %+v", keep, other)
	}
	// 剩下一组里有一张进了回收站，只剩一张，就没有可合并的了，对话框自己关掉。
	a.patchAssets([]string{"4"}, func(asset *model.Asset) { asset.Trashed = true })
	tt.Frame()
	if a.Similar.Open {
		t.Fatal("没有可合并的组时对话框应关掉")
	}
}

// 多选改备注写入失败后，又单独给其中一张改了备注：重试旧的那批时不能盖掉这张的新备注。
// 失败后原样再提交一次也要重试。
func TestMultiDetailsRetryDoesNotOverwriteLaterEdit(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	ids := []string{a.Visible[0].ID, a.Visible[1].ID}
	for _, id := range ids {
		a.Selected.Add(id)
	}
	a.Inspector.Editing, a.Inspector.MultiIDs, a.Inspector.MultiDraft = "multi-note", ids, "一起写的"
	a.FlushInspectorEdits()
	a.inspectorMultiEditFinished(a.Inspector.MultiPending[0], errors.New("disk full"))
	// 失败后原样再提交：值和显示的一样，也要重新排队重试。
	a.Inspector.Editing, a.Inspector.MultiIDs, a.Inspector.MultiDraft, a.Inspector.MultiOriginal = "multi-note", ids, "一起写的", "一起写的"
	a.applyMultiDetail("note")
	if len(a.Inspector.MultiPending) != 1 || a.Inspector.MultiPending[0].Failed {
		t.Fatalf("原样提交没有重试：%+v", a.Inspector.MultiPending)
	}
	// 单独改第二张的备注。
	second := a.assetByID[ids[1]]
	a.Inspector = inspectorState{AssetID: second.ID, Title: second.Name, SourceTitle: second.Name, Note: "单独写的", MultiPending: a.Inspector.MultiPending}
	a.saveInspectorEdits()
	if len(a.Inspector.MultiPending) != 1 || !slices.Equal(a.Inspector.MultiPending[0].IDs, ids[:1]) {
		t.Fatalf("旧的那批还会写第二张：%+v", a.Inspector.MultiPending)
	}
}

// 多选备注写入失败后单选其中一张：编辑框显示用户写的备注，但没动过就不算新修改，换选中时不会单独提交。
func TestMultiPendingShownWithoutAutoSubmit(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	ids := []string{a.Visible[0].ID, a.Visible[1].ID}
	slices.Sort(ids)
	a.Inspector.MultiPending = []inspectorMultiEdit{{Field: "note", Value: "一起写的", IDs: ids, Failed: true}}
	a.syncInspectorAsset(a.assetByID[ids[0]])
	if a.Inspector.Note != "一起写的" {
		t.Fatalf("编辑框没显示未写入的备注：%q", a.Inspector.Note)
	}
	a.saveInspectorEdits()
	if len(a.Inspector.Pending) != 0 || !slices.Equal(a.Inspector.MultiPending[0].IDs, ids) {
		t.Fatalf("没改动也单独提交了：%+v %+v", a.Inspector.Pending, a.Inspector.MultiPending)
	}
}
