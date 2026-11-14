package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
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
	tt.Advance(400 * time.Millisecond)
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

func TestSmartBrowsingPreservesCurrentFolderIdentity(t *testing.T) {
	a := smartTestApp()
	a.BrowseSmartFolder("smart-a")
	if a.Query.Scope != "smart" || a.Query.SmartFolderID != "smart-a" || len(a.Visible) != 1 || a.Visible[0].ID != "a" {
		t.Fatal("没有切到第一个智能文件夹")
	}
	a.SetScope("folder", "root")
	if a.Query.Scope != "folder" || a.Query.FolderID != "root" || a.Query.SmartFolderID != "" {
		t.Fatal("切到普通文件夹后仍保留智能范围")
	}
	a.BrowseSmartFolder("smart-b")
	if a.Query.Scope != "smart" || a.Query.SmartFolderID != "smart-b" || a.Query.FolderID != "" || len(a.Visible) != 1 || a.Visible[0].ID != "b" {
		t.Fatal("没有切到第二个智能文件夹")
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

func TestSmartFileSizeInputUsesMB(t *testing.T) {
	a := smartTestApp()
	a.EditSmartFolder("")
	a.GBrowsing.SmartName = "Large images"
	a.GBrowsing.SmartRules = []model.SmartRule{{Field: "size", Operator: "gte"}}
	tt := ui.NewTester(a.smartFolderDialog, 900, 640)
	if err := tt.Click(a.T("smart_rule_value", 1)); err != nil {
		t.Fatal(err)
	}
	tt.Type("1.5")
	folder, err := a.smartDraft()
	if err != nil || folder.Rules[0].Value != "1572864" {
		t.Fatalf("1.5 MB 没有换算成字节：%+v，%v", folder, err)
	}
	if a.GBrowsing.SmartRules[0].Value != "1.5" {
		t.Fatal("准备保存时改坏了仍在编辑的 MB 输入")
	}
	if !catalog.MatchesSmart(model.Asset{Size: 1572864}, folder) || catalog.MatchesSmart(model.Asset{Size: 1572863}, folder) {
		t.Fatal("MB 条件没有按换算后的大小筛选")
	}
}

func TestSmartFileSizeReopensByteRulesWithoutChangingThresholds(t *testing.T) {
	for _, test := range []struct{ bytes, mb string }{
		{"0", "0"}, {"2097152", "2"}, {"1", "0.00000095367431640625"},
		{"2147483647", "2047.99999904632568359375"},
	} {
		t.Run(test.bytes, func(t *testing.T) {
			a := smartTestApp()
			a.Snapshot.SmartFolders[0].Rules = []model.SmartRule{{Field: "size", Operator: "lte", Value: test.bytes}}
			a.EditSmartFolder("smart-a")
			if got := a.GBrowsing.SmartRules[0].Value; got != test.mb {
				t.Fatalf("旧字节条件没有显示为 MB：%s，想要 %s", got, test.mb)
			}
			folder, err := a.smartDraft()
			if err != nil || folder.Rules[0].Value != test.bytes || a.Snapshot.SmartFolders[0].Rules[0].Value != test.bytes {
				t.Fatalf("重新保存改变了旧条件：%+v，%v", folder, err)
			}
		})
	}
}

func TestSmartFileSizeRejectsInvalidMBAndKeepsDraft(t *testing.T) {
	for _, value := range []string{"", "-1", "NaN", "Inf", "1.2.3", "1/2", "9000000000000"} {
		t.Run(value, func(t *testing.T) {
			a := smartTestApp()
			a.EditSmartFolder("")
			a.GBrowsing.SmartName = "Size rule"
			a.GBrowsing.SmartRules = []model.SmartRule{{Field: "size", Operator: "gte", Value: value}}
			if _, err := a.smartDraft(); err == nil || a.GBrowsing.SmartRules[0].Value != value {
				t.Fatalf("非法 MB 条件未被拒绝或输入丢失：%q，%v", value, err)
			}
		})
	}
}

func TestQuickAccessGroupOnlyAppearsWithPinnedItems(t *testing.T) {
	a := smartTestApp()
	entries := a.Snapshot.QuickAccess
	a.Snapshot.QuickAccess = nil
	tt := ui.NewTester(a.sidebarQuickAccess, 236, 320)
	if tt.HasText(a.T("quick_access")) || len(tt.AccessAll()) != 0 {
		t.Fatal("没有固定项时仍渲染快速访问分组")
	}
	a.Snapshot.QuickAccess = entries
	tt.Frame()
	if !tt.HasText(a.T("quick_access")) || !tt.HasText("Landscape Rules") {
		t.Fatal("固定项存在时没有显示快速访问")
	}
	a.Snapshot.QuickAccess = nil
	tt.Frame()
	if tt.HasText(a.T("quick_access")) {
		t.Fatal("最后一个固定项移除后分组仍显示")
	}
}

func TestSmartEditorLocalizedTextFits(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, match := range []int{0, 1} {
				t.Run(language+"/"+theme+"/"+string(rune('0'+match)), func(t *testing.T) {
					a := smartTestApp()
					a.Preferences.Language, a.Preferences.Theme = language, theme
					a.Texts.Language = language
					a.EditSmartFolder("")
					a.GBrowsing.SmartMatch = match
					a.GBrowsing.SmartRules = []model.SmartRule{{Field: "width", Operator: "gte", Value: "1920"}, {Field: "height", Operator: "gte", Value: "1080"}, {Field: "size", Operator: "lte", Value: "1.5"}, {Field: "added", Operator: "after", Value: "2026-11-01"}, {Field: "note", Operator: "contains", Value: "design"}}
					a.GBrowsing.SmartError = a.T("error_smart_rule")
					type textCheck struct {
						label, text           string
						size, weight, padding float32
					}
					checks := []textCheck{
						{a.T("smart_folder_create_title"), a.T("smart_folder_create_title"), 20, 700, 0},
						{a.T("smart_all"), a.T("smart_all"), 13, 400, 24},
						{a.T("smart_any"), a.T("smart_any"), 13, 400, 24},
						{a.T("add_smart_rule"), a.T("add_smart_rule"), 13, 400, 0},
						{a.T("cancel"), a.T("cancel"), 13, 400, 0},
						{a.T("save"), a.T("save"), 13, 400, 0},
						{a.T("smart_pixels"), a.T("smart_pixels"), 12, 400, 0},
						{a.T("smart_megabytes"), a.T("smart_megabytes"), 12, 400, 0},
						{a.T("error_smart_rule"), a.T("error_smart_rule"), 13, 400, 0},
					}
					for i, rule := range a.GBrowsing.SmartRules {
						checks = append(checks, textCheck{a.T("smart_rule_field", i+1), a.T("smart_field_" + rule.Field), 13, 400, 48}, textCheck{a.T("smart_rule_operator", i+1), a.T("smart_operator_" + rule.Operator), 13, 400, 48})
					}
					measured := make([]ui.Rect, len(checks))
					tt := ui.NewTester(func(c *ui.Context) {
						a.applyTheme(c)
						a.smartFolderDialog(c)
						for i, check := range checks {
							w, h := c.MeasureText(0, ui.Span{Text: check.text, Font: c.Theme().Font, Size: check.size, Weight: int(check.weight), Lang: language})
							measured[i] = ui.Rect{W: w + check.padding, H: h * check.size / max(c.Theme().FontSize, check.size)}
						}
					}, 1100, 640)
					tt.SetLocale(a.Texts.Locale())
					tt.Frame()
					for i, check := range checks {
						bounds, ok := tt.Find(check.label)
						if !ok || bounds.X < 0 || bounds.Y < 0 || bounds.X+bounds.W > 1100 || bounds.Y+bounds.H > 640 || bounds.W+1 < measured[i].W || bounds.H+1 < measured[i].H {
							t.Errorf("文案被裁切或没有完整显示：%s，可用%+v，需要%+v", check.text, bounds, measured[i])
						}
					}
				})
			}
		}
	}
}

func TestQuickAccessLocalizedHeadingFits(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			t.Run(language+"/"+theme, func(t *testing.T) {
				a := smartTestApp()
				a.Preferences.Language, a.Preferences.Theme = language, theme
				a.Texts.Language = language
				var textWidth float32
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					a.sidebar(c)
					textWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("quick_access"), Size: 12, Weight: 700, Lang: language})
				}, 236, 640)
				bounds, ok := tt.Find(a.T("quick_access"))
				if !ok || bounds.W+1 < textWidth || bounds.X+bounds.W > 236 || bounds.Y+bounds.H > 640 {
					t.Fatalf("快速访问标题被裁切：%+v，文字宽度%f", bounds, textWidth)
				}
				a.Snapshot.QuickAccess = nil
				tt.Frame()
				if tt.HasText(a.T("quick_access")) {
					t.Fatal("没有固定项时仍显示本地化分组标题")
				}
			})
		}
	}
}
