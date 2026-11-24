package ui

import (
	"fmt"
	"slices"
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
		Texts: i18n.Catalog{Language: "en"}, TagPage: true, SidebarVisible: true,
	}
}

// 和主窗口一样：标签列表铺满，工具栏盖在上面。
func tagsView(a *App) func(c *ui.Context) {
	return func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			a.tags(c)
			ui.Column(c.Key("main-header")).Absolute().Top(0).Left(0).Right(0).Children(func() { a.toolbar(c) })
		})
	}
}

func tagsTester(a *App) *ui.Tester {
	return ui.NewTester(tagsView(a), 720, 640)
}

// 标签管理页的工具栏只有标题、搜索标签和新建标签，没有筛选、排序、视图和导入。
func TestTagsToolbarOnlyHasTagControls(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	tt := tagsTester(a)
	for _, key := range []string{"search_tags", "new_tag", "tag_management"} {
		if _, ok := tt.Find(a.T(key)); !ok {
			t.Errorf("工具栏缺少 %s", key)
		}
	}
	for _, key := range []string{"filter", "sort", "view_mode", "import", "thumbnail_size", "search_assets"} {
		if _, ok := tt.Find(a.T(key)); ok {
			t.Errorf("标签管理页不应出现 %s", key)
		}
	}
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
	hoverTagRow(t, tt, "Landscape")
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
	a := tagsTestApp(model.Snapshot{
		Tags:   []model.Tag{{ID: "nature", Name: "Landscape"}},
		Assets: []model.Asset{{ID: "a", TagIDs: []string{"nature"}}, {ID: "b", TagIDs: []string{"nature"}}},
	})
	tt := tagsTester(a)
	hoverTagRow(t, tt, "Landscape")
	if err := tt.Click(a.T("delete_tag") + " Landscape"); err != nil {
		t.Fatal(err)
	}
	if !a.ConfirmOpen || a.ConfirmTitle != a.T("delete_tag") || a.ConfirmDetail != a.T("delete_tag_assets_detail", "Landscape", 2) || a.ConfirmLabel != a.T("delete") || a.ConfirmAction == nil {
		t.Fatalf("删除有素材的标签没有写明数量并要求确认：%q", a.ConfirmDetail)
	}
	if a.Busy || len(a.Snapshot.Tags) != 1 {
		t.Fatal("确认前执行了删除")
	}
}

// 没有素材用的标签直接删，不弹确认；写入忙时排队。
func TestTagsDeleteUnusedTagWithoutConfirmation(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	a.Busy = true
	tt := tagsTester(a)
	hoverTagRow(t, tt, "Landscape")
	if err := tt.Click(a.T("delete_tag") + " Landscape"); err != nil {
		t.Fatal(err)
	}
	if a.ConfirmOpen || len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "delete_tag" {
		t.Fatalf("没有素材的标签应直接排队删除：confirm=%v queued=%+v", a.ConfirmOpen, a.queuedRuns)
	}
}

// 整行单击就打开标签，行尾的改名按钮不触发打开；右键菜单里有重命名和删除。
func TestTagsRowClickOpensAndMenuHasRenameDelete(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}, Assets: []model.Asset{{ID: "a", TagIDs: []string{"nature"}}}})
	tt := tagsTester(a)
	hoverTagRow(t, tt, "Landscape")
	if err := tt.Click(a.T("rename_tag") + " Landscape"); err != nil {
		t.Fatal(err)
	}
	if a.Query.Scope == "tag" || a.TagPage == false || !a.PromptOpen {
		t.Fatalf("点改名按钮不该打开标签：scope=%q tagPage=%v", a.Query.Scope, a.TagPage)
	}
	a.PromptOpen = false
	if err := tt.RightClick("Landscape"); err != nil {
		t.Fatal(err)
	}
	menu := tt.Menu()
	for _, key := range []string{"rename_tag", "delete_tag"} {
		if !slices.Contains(menu, a.T(key)) {
			t.Fatalf("右键菜单缺少 %s：%v", key, menu)
		}
	}
	tt.CloseMenu()
	if err := tt.Click("Landscape"); err != nil {
		t.Fatal(err)
	}
	if a.Query.Scope != "tag" || a.Browse.TagID != "nature" || a.TagPage {
		t.Fatalf("单击标签行没有打开它：scope=%q tag=%q", a.Query.Scope, a.Browse.TagID)
	}
}

// 搜不到时直接新建；已有同名（规范化后相同）的只提示，不新建；没有任何标签时空状态里有新建按钮。
func TestTagsCreateFromSearchAndEmptyState(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	a.Busy = true
	a.Tags.Search = "Macro"
	tt := tagsTester(a)
	if err := tt.Click(a.T("new_tag_named", "Macro")); err != nil {
		t.Fatal(err)
	}
	if len(a.queuedRuns) != 1 || a.queuedRuns[0].label != "create_tag" {
		t.Fatalf("搜索无结果的新建按钮没有创建：%+v", a.queuedRuns)
	}
	a.createTag(" ＬＡＮＤＳＣＡＰＥ ")
	if len(a.queuedRuns) != 1 || a.Notice != a.T("tag_exists") {
		t.Fatalf("重名标签不该新建，应提示：queued=%d notice=%q", len(a.queuedRuns), a.Notice)
	}

	empty := tagsTestApp(model.Snapshot{})
	tt = tagsTester(empty)
	buttons := 0
	for _, info := range tt.AccessAll() {
		if info.Role == ui.RoleButton && info.Name == empty.T("new_tag") {
			buttons++
		}
	}
	if buttons != 2 {
		t.Fatalf("空标签页除了工具栏，中间也要有新建标签按钮：%d 个", buttons)
	}
}

func TestTagsEmptyStateExplainsAndOffersNewTag(t *testing.T) {
	a := tagsTestApp(model.Snapshot{})
	tt := tagsTester(a)
	if !tt.HasText(a.T("no_tags")) || !tt.HasText(a.T("no_tags_hint")) {
		t.Fatalf("空标签页未提供说明：%v", tt.Texts())
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
			view := tagsView(a)
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				view(c)
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
	before, _ := tt.Find("Reference")
	a.Progress = &model.ImportProgress{Stage: "importing", Total: 10}
	tt.Frame()
	tt.Advance(90 * time.Millisecond)
	middle, _ := tt.Find("Reference")
	tt.Advance(time.Second)
	after, _ := tt.Find("Reference")
	if middle.Y <= before.Y || middle.Y >= after.Y || after.Y-before.Y != 120 {
		t.Fatalf("导入进度出现时标签内容没有平滑让位：%+v → %+v → %+v", before, middle, after)
	}
}

// 改名、删除按钮平时收起，先把鼠标移到这一行上它们才可点。
func hoverTagRow(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	box, ok := tt.Find(name)
	if !ok {
		t.Fatalf("找不到标签行 %s", name)
	}
	tt.Move(box.X+box.W/2, box.Y+box.H/2)
	tt.Advance(time.Second)
}

func TestTagRowToolsOnlyOnHover(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	a.Busy = true // 没素材用的标签点删除会直接删；忙时进队列，测试里不碰数据库
	tt := tagsTester(a)
	if err := tt.Click(a.T("delete_tag") + " Landscape"); err == nil {
		t.Fatal("没悬停时删除按钮不应该能点")
	}
	hoverTagRow(t, tt, "Landscape")
	if err := tt.Click(a.T("delete_tag") + " Landscape"); err != nil {
		t.Fatal(err)
	}
}

// 纯键盘：Tab 移到标签行后，再按 Tab 能走到改名、删除按钮。
func TestTagRowToolsReachableByKeyboard(t *testing.T) {
	a := tagsTestApp(model.Snapshot{Tags: []model.Tag{{ID: "nature", Name: "Landscape"}}})
	tt := tagsTester(a)
	for range 20 {
		if tt.Focused("Landscape") {
			break
		}
		tt.Key(0, ui.KeyTab)
		tt.Advance(100 * time.Millisecond)
	}
	if !tt.Focused("Landscape") {
		t.Fatal("Tab 走不到标签行")
	}
	tt.Key(0, ui.KeyTab)
	tt.Advance(time.Second)
	if !tt.Focused(a.T("rename_tag") + " Landscape") {
		t.Fatal("键盘移到标签行后，改名按钮没有接到焦点")
	}
	tt.Key(0, ui.KeyTab)
	tt.Advance(100 * time.Millisecond)
	if !tt.Focused(a.T("delete_tag") + " Landscape") {
		t.Fatal("键盘走不到删除按钮")
	}
}
