package ui

import (
	"errors"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func smartTestApp() *App {
	a := sidebarTestApp()
	a.Preferences.Language = "en"
	a.Snapshot.Tags = []model.Tag{{ID: "tag-a", Name: "Reference"}}
	a.Snapshot.Assets[0].TagIDs = []string{"tag-a"}
	a.Snapshot.SmartFolders = []model.SmartFolder{
		{ID: "smart-a", Name: "Landscape Rules", Match: "all", Rules: []model.SmartRule{{Field: "name", Operator: "contains", Value: "Landscape"}}},
		{ID: "smart-b", Name: "Portrait Rules", Match: "any", Rules: []model.SmartRule{{Field: "name", Operator: "contains", Value: "Portrait"}}},
	}
	a.Snapshot.QuickAccess = []model.QuickAccess{
		{ID: "quick-folder", Kind: "folder", TargetID: "root", SortIndex: 0},
		{ID: "quick-tag", Kind: "tag", TargetID: "tag-a", SortIndex: 1},
		{ID: "quick-smart", Kind: "smart", TargetID: "smart-a", SortIndex: 2},
	}
	a.applySnapshot(a.Snapshot)
	return a
}

func TestSmartEditorChangesFieldsAndPreservesInvalidDraft(t *testing.T) {
	a := smartTestApp()
	a.EditSmartFolder("")
	tt := ui.NewTester(a.smartFolderDialog, 900, 640)
	if err := tt.Click(a.T("save")); err != nil {
		t.Fatal(err)
	}
	if !a.GBrowsing.SmartOpen || a.GBrowsing.SmartError != a.T("smart_name_required") {
		t.Fatal("无名称的智能文件夹没有保留表单并显示错误")
	}
	if err := tt.Click(a.T("smart_folder_name")); err != nil {
		t.Fatal(err)
	}
	tt.Type("UI References")
	if err := tt.Click(a.T("smart_rule_value", 1)); err != nil {
		t.Fatal(err)
	}
	tt.Type("design")
	if err := tt.Click(a.T("smart_any")); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(a.T("add_smart_rule")); err != nil {
		t.Fatal(err)
	}
	if len(a.GBrowsing.SmartRules) != 2 || a.GBrowsing.SmartMatch != 1 {
		t.Fatal("添加条件或任一匹配没有生效")
	}
	if err := tt.Click(a.T("smart_rule_field", 2)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(a.T("smart_field_added")); err != nil {
		t.Fatal(err)
	}
	if rule := a.GBrowsing.SmartRules[1]; rule.Field != "added" || rule.Operator != "after" || rule.Value != "" {
		t.Fatalf("切换字段没有更换比较方式：%+v", rule)
	}
	if err := tt.Click(a.T("remove_smart_rule_number", 2)); err != nil {
		t.Fatal(err)
	}
	folder, err := a.smartDraft()
	if err != nil || folder.Name != "UI References" || folder.Match != "any" || len(folder.Rules) != 1 || folder.Rules[0].Value != "design" {
		t.Fatalf("有效输入没有进入待保存数据：%+v，%v", folder, err)
	}
}

func TestSmartEditorCopiesRulesAndKeepsSaveFailureVisible(t *testing.T) {
	a := smartTestApp()
	a.EditSmartFolder("smart-a")
	a.GBrowsing.SmartRules[0].Value = "changed"
	if a.Snapshot.SmartFolders[0].Rules[0].Value != "Landscape" {
		t.Fatal("尚未保存的条件改动了图库快照")
	}
	a.Busy = true
	a.GBrowsing.Saving = true
	a.finishSmartFolderSave(a.generation, "smart-a", model.Snapshot{}, errors.New("disk full"))
	if !a.GBrowsing.SmartOpen || a.GBrowsing.Saving || a.Busy || a.GBrowsing.SmartError == "" || a.GBrowsing.SmartRules[0].Value != "changed" {
		t.Fatal("保存失败后表单被关闭、条件丢失或没有错误")
	}
	a.finishSmartFolderSave(a.generation, "smart-a", a.Snapshot, nil)
	if a.GBrowsing.SmartOpen || a.Query.Scope != "smart" || a.Query.SmartFolderID != "smart-a" || len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatal("成功保存后没有进入对应智能范围")
	}
}

func TestSmartHistoryRestoresFolderIdentity(t *testing.T) {
	a := smartTestApp()
	a.BrowseSmartFolder("smart-a")
	a.SetScope("folder", "root")
	a.BrowseSmartFolder("smart-b")
	a.NavigateBack()
	if a.Query.Scope != "folder" || a.Query.FolderID != "root" || a.Query.SmartFolderID != "" {
		t.Fatal("从智能范围后退没有恢复文件夹")
	}
	a.NavigateBack()
	if a.Query.SmartFolderID != "smart-a" || len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatal("历史丢失了前一个智能文件夹")
	}
	a.NavigateForward()
	a.NavigateForward()
	if a.Query.SmartFolderID != "smart-b" || len(a.Visible) != 1 || a.Visible[0].ID != "b" {
		t.Fatal("前进没有恢复智能文件夹")
	}
}

func TestQuickAccessUsesTargetsAndSharesSidebarCounts(t *testing.T) {
	a := smartTestApp()
	for _, entry := range a.Snapshot.QuickAccess {
		name, _, count, selected := a.quickAccessTarget(entry)
		if name == "" || count != 1 || selected {
			t.Fatalf("固定目标名称、数量或选中态错误：%+v，%s，%d", entry, name, count)
		}
	}
	tt := ui.NewTester(a.sidebarQuickAccess, 236, 320)
	if err := tt.Click("Reference"); err != nil {
		t.Fatal(err)
	}
	if a.Query.Scope != "tag" || a.Browse.TagID != "tag-a" || len(a.Visible) != 1 {
		t.Fatal("固定标签没有进入标签范围")
	}
	if _, _, _, selected := a.quickAccessTarget(a.Snapshot.QuickAccess[1]); !selected {
		t.Fatal("固定标签没有显示选中")
	}
	if err := tt.Click("Landscape Rules"); err != nil {
		t.Fatal(err)
	}
	if a.Query.Scope != "smart" || a.Query.SmartFolderID != "smart-a" {
		t.Fatal("固定智能文件夹没有进入智能范围")
	}
	if err := tt.RightClick("Landscape Rules"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"quick_move_up", "quick_move_down", "quick_unpin"} {
		found := false
		for _, item := range tt.Menu() {
			if item == a.T(key) {
				found = true
			}
		}
		if !found {
			t.Fatalf("快速访问菜单缺少%s：%v", key, tt.Menu())
		}
	}
}

func TestSmartSidebarOpensEditorAndHiddenGroupsDisappear(t *testing.T) {
	a := smartTestApp()
	tt := ui.NewTester(a.sidebarSmartFolders, 236, 320)
	if err := tt.Click(a.T("new_smart_folder")); err != nil {
		t.Fatal(err)
	}
	if !a.GBrowsing.SmartOpen || a.GBrowsing.SmartID != "" {
		t.Fatal("智能分组没有打开新建表单")
	}
	a.Preferences.SidebarHidden = []string{"smart", "quick"}
	tt.Frame()
	if tt.HasText(a.T("smart_folders")) || tt.HasText("Landscape Rules") {
		t.Fatal("隐藏智能分组后仍显示")
	}
	quick := ui.NewTester(a.sidebarQuickAccess, 236, 320)
	if quick.HasText(a.T("quick_access")) {
		t.Fatal("隐藏快速访问后仍显示")
	}
}

func TestGBrowsingCountsRefreshAfterSnapshotChange(t *testing.T) {
	a := smartTestApp()
	if a.smartFolderCount("smart-a") != 1 {
		t.Fatal("初始智能范围计数错误")
	}
	snapshot := a.Snapshot
	snapshot.Assets = append(snapshot.Assets, model.Asset{ID: "new", Name: "Landscape Detail", FolderIDs: []string{"root"}, TagIDs: []string{"tag-a"}})
	a.applySnapshot(snapshot)
	if a.smartFolderCount("smart-a") != 2 {
		t.Fatal("图库变化后沿用了旧智能计数")
	}
	for _, entry := range a.Snapshot.QuickAccess {
		_, _, count, _ := a.quickAccessTarget(entry)
		if count != 2 {
			t.Fatalf("固定目标仍使用旧计数：%+v，%d", entry, count)
		}
	}
}
