package ui

import (
	"fmt"
	"testing"
	"time"

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

func TestTagsLocalizedControlsFit(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
			a.Texts.Language = language
			var searchWidth, actionWidth float32
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				ui.Column(c).Fill().Children(func() { a.tags(c) })
				searchWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("search_tags"), Size: 13, Lang: language})
				actionWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("new_tag"), Size: 13, Lang: language})
			}, 584, 640)
			tt.SetLocale(a.Texts.Locale())
			tt.Frame()
			for _, check := range []struct {
				label string
				width float32
			}{{a.T("search_tags"), searchWidth + 44}, {a.T("new_tag"), actionWidth}, {a.T("rename_tag") + " Landscape", 28}, {a.T("delete_tag") + " Landscape", 28}} {
				bounds, ok := tt.Find(check.label)
				if !ok || bounds.W+1 < check.width || bounds.X < 0 || bounds.X+bounds.W > 584 || bounds.Y < 0 || bounds.Y+bounds.H > 640 {
					t.Errorf("标签页控件未完整显示：%s，范围%+v，需要宽度%f", check.label, bounds, check.width)
				}
			}
		})
	}
}

func TestTagsOnlyBuildVisibleRows(t *testing.T) {
	snapshot := model.Snapshot{}
	for i := range 12000 {
		name := fmt.Sprintf("Tag %05d", i)
		snapshot.Tags = append(snapshot.Tags, model.Tag{ID: name, Name: name})
	}
	a := tagsTestApp(snapshot)
	tt := tagsTester(a)
	if !tt.HasText("Tag 00000") || tt.HasText("Tag 11999") || len(tt.AccessAll()) > 200 {
		t.Fatalf("标签页没有按可见窗口构建：%d 个元素", len(tt.AccessAll()))
	}
	tt.Scroll(400, 500, 0, 1000000)
	tt.Advance(time.Second)
	if !tt.HasText("Tag 11999") || tt.HasText("Tag 00000") || len(tt.AccessAll()) > 200 {
		t.Fatalf("标签页滚到底后没有更新可见窗口：%d 个元素", len(tt.AccessAll()))
	}
}

func TestTagsImportInsetMovesSmoothly(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "a", Name: "Reference"}}})
	tt := tagsTester(a)
	before, _ := tt.Find(a.T("new_tag"))
	a.Progress = &model.ImportProgress{Stage: "importing", Total: 10}
	tt.Frame()
	tt.Advance(90 * time.Millisecond)
	middle, _ := tt.Find(a.T("new_tag"))
	tt.Advance(time.Second)
	after, _ := tt.Find(a.T("new_tag"))
	if middle.Y <= before.Y || middle.Y >= after.Y || after.Y-before.Y != 120 {
		t.Fatalf("导入进度出现时标签内容没有平滑让位：%+v → %+v → %+v", before, middle, after)
	}
}
