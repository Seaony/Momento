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
