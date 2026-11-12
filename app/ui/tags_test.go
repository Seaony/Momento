package ui

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func tagsTestApp(snapshot model.Snapshot) *App {
	return &App{
		Library: &library.Library{}, Snapshot: snapshot, Index: catalog.New(snapshot),
		Texts: i18n.Catalog{Language: "en"},
	}
}

func tagsTester(a *App) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { a.tags(c) })
	}, 720, 640)
}

func TestTagsSearchAndCounts(t *testing.T) {
	snapshot := model.Snapshot{
		Tags: []model.Tag{{ID: "nature", Name: "Landscape 山水"}, {ID: "icon", Name: "Icon"}},
	}
	for range 1234 {
		snapshot.Assets = append(snapshot.Assets, model.Asset{TagIDs: []string{"nature"}})
	}
	snapshot.Assets = append(snapshot.Assets, model.Asset{TagIDs: []string{"nature"}, Trashed: true})
	a := tagsTestApp(snapshot)
	tt := tagsTester(a)
	if !tt.HasText("Landscape 山水") || !tt.HasText("Icon") || !tt.HasText(a.T("tag_asset_count", "1,234")) {
		t.Fatalf("标签和关联数量未显示：%v", tt.Texts())
	}
	if err := tt.Click(a.T("search_tags")); err != nil {
		t.Fatal(err)
	}
	tt.Type("LaNdScApE")
	if !tt.HasText("Landscape 山水") || tt.HasText("Icon") {
		t.Fatalf("搜索未忽略大小写：%v", tt.Texts())
	}
	a.Tags.Search = "山水"
	tt.Frame()
	if !tt.HasText("Landscape 山水") || tt.HasText("Icon") {
		t.Fatalf("中文搜索错误：%v", tt.Texts())
	}
	a.Tags.Search = "missing"
	tt.Frame()
	if !tt.HasText(a.T("no_tag_results", "missing")) || tt.HasText("Landscape 山水") {
		t.Fatalf("没有显示搜索空状态：%v", tt.Texts())
	}
}

func TestTagsNewAndRenameOpenNamePrompt(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	tt := tagsTester(a)
	if err := tt.Click(a.T("new_tag")); err != nil {
		t.Fatal(err)
	}
	if !a.PromptOpen || a.PromptTitle != a.T("new_tag") || a.PromptValue != "" || a.PromptSubmit == nil {
		t.Fatal("新建标签没有打开空名称对话框")
	}
	a.PromptOpen = false
	tt.Frame()
	if err := tt.Click(a.T("rename_tag") + " Landscape"); err != nil {
		t.Fatal(err)
	}
	if !a.PromptOpen || a.PromptTitle != a.T("rename_tag") || a.PromptValue != "Landscape" || a.PromptSubmit == nil {
		t.Fatal("重命名没有预填当前名称")
	}
	if a.Busy {
		t.Fatal("提交名称前执行了数据库操作")
	}
}

func TestTagsDeletionRequiresConfirmation(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	tt := tagsTester(a)
	if err := tt.Click(a.T("delete_tag") + " Landscape"); err != nil {
		t.Fatal(err)
	}
	if !a.ConfirmOpen || a.ConfirmTitle != a.T("delete_tag") || a.ConfirmDetail != a.T("delete_tag_detail", "Landscape") || a.ConfirmAction == nil {
		t.Fatal("删除标签没有要求确认")
	}
	if a.Busy || len(a.Snapshot.Tags) != 1 {
		t.Fatal("确认前执行了删除")
	}
}

func TestTagsEmptyStateAndBusyActions(t *testing.T) {
	a := tagsTestApp(model.Snapshot{})
	tt := tagsTester(a)
	if !tt.HasText(a.T("no_tags")) || !tt.HasText(a.T("no_tags_hint")) {
		t.Fatalf("空标签页未提供说明：%v", tt.Texts())
	}
	a.Busy = true
	tt.Frame()
	button, ok := tt.AccessByRole(ui.RoleButton, a.T("new_tag"))
	if !ok || !button.Disabled {
		t.Fatal("忙碌时新建按钮仍可操作")
	}
}

func TestTagsFormatCounts(t *testing.T) {
	for n, want := range map[int]string{0: "0", 99: "99", 999: "999", 1000: "1,000", 12345: "12,345", 1234567: "1,234,567"} {
		if got := formatTagCount(n); got != want {
			t.Fatalf("%d 格式化为 %q，期望 %q", n, got, want)
		}
	}
}
