package ui

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func renameDialogFixture(count int) *App {
	app := &App{
		Library:   &library.Library{},
		Texts:     i18n.Catalog{Language: "en"},
		assetByID: make(map[string]model.Asset, count),
	}
	ids := make([]string, count)
	for index := range ids {
		id := strconv.Itoa(index)
		ids[index] = id
		app.assetByID[id] = model.Asset{ID: id, Name: "Photo " + id}
	}
	app.OpenRename(ids)
	return app
}

func renameDialogTester(app *App) (*ui.Tester, <-chan func()) {
	posted := make(chan func(), 2)
	tester := ui.NewTesterWith(func(c *ui.Context) {
		app.renderRenameDialog(c, func(update func()) { posted <- update })
	}, 900, 760, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
	return tester, posted
}

func nextRenameUpdate(t *testing.T, posted <-chan func()) func() {
	t.Helper()
	select {
	case update := <-posted:
		return update
	case <-time.After(5 * time.Second):
		t.Fatal("后台没有返回重命名计划")
		return nil
	}
}

func assertRenameApply(t *testing.T, app *App, tester *ui.Tester, enabled bool) ui.AccessInfo {
	t.Helper()
	button, found := tester.AccessByRole(ui.RoleButton, app.T("rename_apply"))
	if !found || button.Disabled == enabled {
		t.Fatalf("应用按钮应可用 %t，找到 %t，状态 %+v", enabled, found, button)
	}
	return button
}

func TestRenameDialogAppliesOnlyCurrentPlan(t *testing.T) {
	app := renameDialogFixture(101)
	app.Rename.Template, app.Rename.Start = "A-{n}", "900"
	tester, posted := renameDialogTester(app)
	old := nextRenameUpdate(t, posted)
	button := assertRenameApply(t, app, tester, false)
	if !tester.HasText("A-0900") || len(app.Rename.preview.names) != 100 {
		t.Fatal("首帧应只预览前 100 行，序号按整批补零")
	}
	tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	if !app.Rename.Open || app.Busy {
		t.Fatal("等待全量计划时点应用不应执行")
	}
	field, found := tester.AccessByRole(ui.RoleTextField, app.T("rename_template"))
	if !found {
		t.Fatal("找不到名称格式输入框")
	}
	tester.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Type("A-{n}x")
	if !tester.HasText("A-0900x") {
		t.Fatalf("输入后应立即看到新规则的预览，当前格式 %q", app.Rename.Template)
	}
	old()
	tester.Frame()
	assertRenameApply(t, app, tester, false)
	if app.Rename.plan != nil {
		t.Fatal("采用了旧规则的全量计划")
	}

	nextRenameUpdate(t, posted)()
	tester.Frame()
	assertRenameApply(t, app, tester, true)
	if plan := app.Rename.plan; len(plan.ids) != 101 || plan.updates["0"] != "A-0900x" || plan.updates["100"] != "A-1000x" {
		t.Fatalf("全量计划没有覆盖整批素材：%+v", plan)
	}
	if err := tester.Click(app.T("cancel")); err != nil {
		t.Fatal(err)
	}
	if state := app.Rename; state.Open || state.plan != nil || state.source != nil || state.preview != nil || state.pending != nil || state.IDs != nil {
		t.Fatal("关闭后仍保留了选区或计算结果")
	}
}

func TestRenameDialogValidatesNamesBeyondPreview(t *testing.T) {
	app := renameDialogFixture(101)
	app.Rename.Mode, app.Rename.Find = 1, "Photo 100"
	tester, posted := renameDialogTester(app)
	nextRenameUpdate(t, posted)()
	tester.Frame()
	assertRenameApply(t, app, tester, false)
	if !tester.HasText(app.T("rename_empty_names", 1)) {
		t.Fatal("没有提示预览范围以外的空名称")
	}
}

func TestRenameDialogDropsPlanAfterCloseAndReopen(t *testing.T) {
	app := renameDialogFixture(2)
	ids := app.Rename.IDs
	app.Rename.Template = "New {name}"
	tester, posted := renameDialogTester(app)
	closed := nextRenameUpdate(t, posted)
	tester.Key(0, ui.KeyEscape)
	if app.Rename.Open || app.Rename.pending != nil {
		t.Fatal("关闭时没有清理待计算的计划")
	}
	app.OpenRename(ids)
	tester.Frame()
	closed()
	tester.Frame()
	assertRenameApply(t, app, tester, false)
	nextRenameUpdate(t, posted)()
	tester.Frame()
	assertRenameApply(t, app, tester, true)
}

func TestRenameDialogDropsPlanAfterSnapshotChange(t *testing.T) {
	app := renameDialogFixture(2)
	app.Rename.Template = "New {name}"
	tester, posted := renameDialogTester(app)
	old := nextRenameUpdate(t, posted)
	app.assetByID["0"] = model.Asset{ID: "0", Name: "Updated"}
	app.snapshotVersion++
	old()
	tester.Frame()
	assertRenameApply(t, app, tester, false)
	if !tester.HasText("New Updated") {
		t.Fatal("图库变更后没有刷新预览")
	}
	nextRenameUpdate(t, posted)()
	tester.Frame()
	assertRenameApply(t, app, tester, true)
	if app.Rename.plan.updates["0"] != "New Updated" {
		t.Fatal("全量计划未使用最新版图库数据")
	}
}

func TestRenameDialogPendingTextFits(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		t.Run(language, func(t *testing.T) {
			app := renameDialogFixture(101)
			app.Preferences.Language = language
			app.Texts.Language = language
			app.Rename.Template = "New-{n}"
			checks := []struct {
				text    string
				role    ui.Role
				size    float32
				weight  int
				padding float32
			}{
				{app.T("rename_title", 101), ui.RoleText, fontLead, 600, 0},
				{app.T("rename_preview"), ui.RoleText, fontSecondary, 600, 0},
				{app.T("cancel"), ui.RoleButton, fontBody, 500, 2 * space16},
				{app.T("rename_apply"), ui.RoleButton, fontBody, 500, 2 * space16},
			}
			widths := make([]float32, len(checks))
			tester := ui.NewTesterWith(func(c *ui.Context) {
				app.applyTheme(c)
				app.renameDialog(c)
				for index, check := range checks {
					widths[index], _ = c.MeasureText(0, ui.Span{Text: check.text, Font: c.Theme().Font, Size: check.size, Weight: check.weight, Lang: language})
				}
			}, 1100, 640, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
			tester.SetLocale(language)
			assertRenameApply(t, app, tester, false)
			for index, check := range checks {
				element, found := tester.AccessByRole(check.role, check.text)
				bounds := element.Bounds
				if !found || bounds.W+1 < widths[index]+check.padding || bounds.H < check.size || bounds.Y+bounds.H > 640 {
					t.Errorf("%q 被截断或越界：可用 %+v，文字及留白需要 %.1f", check.text, bounds, widths[index]+check.padding)
				}
			}
		})
	}
}

func TestRenameDialogPendingScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_RENAME_PENDING_SHOTS") != "1" {
		t.Skip("设置 MOMENTO_RENAME_PENDING_SHOTS=1 生成等待全量计划的截图")
	}
	const dir = "/tmp/momento-batch2-shots"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			app := fixture.app(t, "small", "zh-CN", theme)
			ids := make([]string, 0, 5)
			for _, asset := range app.Visible[:5] {
				ids = append(ids, asset.ID)
			}
			app.OpenRename(ids)
			app.Rename.Template = "旅行-{n} {date}"
			tester := ui.NewTesterWith(app.View, 1100, 700, ui.TesterOptions{
				Dark: theme == "dark", Preferences: ui.Preferences{ReduceMotion: true},
			})
			tester.SetLocale(app.Texts.Locale())
			tester.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			posted := make(chan func(), 1)
			app.requestRenamePlan(func(update func()) { posted <- update })
			nextRenameUpdate(t, posted)
			tester.Frame()
			assertRenameApply(t, app, tester, false)
			writeScreenshot(t, filepath.Join(dir, "rename-pending-zh-CN-"+theme+".png"), tester)
		})
	}
}

func BenchmarkRenameDialog100KKeystroke(b *testing.B) {
	app := renameDialogFixture(100_000)
	const template = "{name}-{n}"
	app.Rename.Template = template
	tester := ui.NewTesterWith(app.renameDialog, 900, 760, ui.TesterOptions{
		Preferences: ui.Preferences{ReduceMotion: true},
	})
	tester.Key(0, ui.KeyEnd)
	if !tester.Focused(app.T("rename_template")) {
		b.Fatal("名称格式输入框没有焦点")
	}
	tester.Type("x")
	if app.Rename.Template != template+"x" {
		b.Fatalf("未在模板末尾打字：%q", app.Rename.Template)
	}
	tester.Key(0, ui.KeyBackspace)
	if app.Rename.Template != template {
		b.Fatalf("未删除模板末尾的字符：%q", app.Rename.Template)
	}
	for b.Loop() {
		if app.Rename.Template == template {
			tester.Type("x")
		} else {
			tester.Key(0, ui.KeyBackspace)
		}
	}
	b.ReportAllocs()
	if app.Rename.Template != template && app.Rename.Template != template+"x" {
		b.Fatalf("没有在两条规则间切换：%q", app.Rename.Template)
	}
}
