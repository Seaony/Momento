package ui

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func assetViewFixture() *App {
	a := &App{Texts: i18n.Catalog{Language: "en"}, selectedIndex: -1, anchor: -1, Query: catalog.Query{Sort: "name"}}
	var snapshot model.Snapshot
	for i := 0; i < 6; i++ {
		snapshot.Assets = append(snapshot.Assets, model.Asset{ID: fmt.Sprint(i), Name: fmt.Sprintf("Photo %d", i), Width: 600, Height: 400, Extension: "jpg", CreatedAt: time.Now()})
	}
	a.applySnapshot(snapshot)
	return a
}
func TestAssetSelectionInThreeViews(t *testing.T) {
	for _, mode := range viewModes {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			a := assetViewFixture()
			a.ViewMode = mode
			tt := ui.NewTester(a.assetArea, 700, 600)
			if err := tt.Click("Photo 0"); err != nil {
				t.Fatal(err)
			}
			if !a.Selected.Has("0") || a.Selected.Len() != 1 {
				t.Fatalf("单选: %v", a.Selected.Len())
			}
			if err := tt.ClickWith(ui.Cmd, "Photo 1"); err != nil {
				t.Fatal(err)
			}
			if a.Selected.Len() != 2 {
				t.Fatalf("加选: %v", a.Selected.Len())
			}
			if err := tt.ClickWith(ui.Shift, "Photo 3"); err != nil {
				t.Fatal(err)
			}
			if a.Selected.Len() != 3 || !a.Selected.Has("1") || !a.Selected.Has("3") {
				t.Fatalf("连选: %v", a.Selected.Len())
			}
		})
	}
}
func TestFilterResetsSelectionAnchor(t *testing.T) {
	a := assetViewFixture()
	a.selectAsset("5", 5, 0)
	a.Query.Search = "Photo 0"
	a.Refine()
	a.selectAsset("0", 0, ui.Shift)
	if a.Selected.Len() != 1 || !a.Selected.Has("0") {
		t.Fatal("筛选后连选不正确")
	}
	a.anchor = 100
	a.selectAsset("0", 0, ui.Shift)
	if a.Selected.Len() != 1 {
		t.Fatal("过期锚点没有安全重置")
	}
}
func TestSnapshotRefreshKeepsScrollPosition(t *testing.T) {
	a := assetViewFixture()
	a.scroll.Y = 240
	a.applySnapshot(a.Snapshot)
	if a.scroll.Y != 240 {
		t.Fatal("后台刷新让滚动跳回顶部")
	}
	a.Refine()
	if a.scroll.Y != 0 {
		t.Fatal("新筛选未回到顶部")
	}
}

func TestAppearanceReturnsToSystem(t *testing.T) {
	a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: Preferences{Theme: "dark"}}
	dark := false
	tt := ui.NewTester(func(c *ui.Context) { a.View(c); dark = c.Theme().Dark }, 1100, 640)
	if !dark {
		t.Fatal("没有使用深色设置")
	}
	a.Preferences.Theme = "system"
	tt.SetDark(false)
	tt.Frame()
	if dark {
		t.Fatal("跟随系统仍残留深色设置")
	}
}

func TestAssetListSelectionAfterScrolling(t *testing.T) {
	a := assetViewFixture()
	a.ViewMode = 2
	snapshot := model.Snapshot{}
	for i := 0; i < 500; i++ {
		snapshot.Assets = append(snapshot.Assets, model.Asset{ID: fmt.Sprint(i), Name: fmt.Sprintf("Photo %03d", i), Width: 600, Height: 400, Extension: "jpg"})
	}
	a.applySnapshot(snapshot)
	tt := ui.NewTester(a.assetArea, 700, 600)
	if err := tt.Click("Photo 000"); err != nil {
		t.Fatal(err)
	}
	a.scroll.Y = math.MaxFloat32
	tt.Frame()
	if err := tt.ClickWith(ui.Cmd, "Photo 499"); err != nil {
		t.Fatal(err)
	}
	if a.Selected.Len() != 2 || !a.Selected.Has("0") || !a.Selected.Has("499") {
		t.Fatal("滚动后列表没有保留加选")
	}
}

// 方向键移动选择：列表上下；瀑布流上下在同一列，左右到相邻列；没选中时第一下选第一张。
func TestAssetArrowKeysMoveSelection(t *testing.T) {
	press := func(tt *ui.Tester, key ui.Key) { tt.Key(0, key); tt.Frame() }
	selected := func(a *App) string {
		if a.Selected.Len() != 1 {
			t.Fatalf("应只选中一张，实际 %d 张", a.Selected.Len())
		}
		return a.Visible[a.selectedIndex].ID
	}

	a := assetViewFixture()
	tt := ui.NewTester(a.assetArea, 700, 600)
	press(tt, ui.KeyDown)
	if selected(a) != a.Visible[0].ID {
		t.Fatal("没选中时第一下没有选中第一张")
	}
	if len(a.layout.columns) != 3 {
		t.Fatalf("样本应排成 3 列，实际 %d 列", len(a.layout.columns))
	}
	for _, step := range []struct {
		key  ui.Key
		want int
	}{{ui.KeyRight, 1}, {ui.KeyRight, 2}, {ui.KeyRight, 2}, {ui.KeyDown, 5}, {ui.KeyLeft, 4}, {ui.KeyUp, 1}, {ui.KeyUp, 1}} {
		press(tt, step.key)
		if got := selected(a); got != a.Visible[step.want].ID {
			t.Fatalf("瀑布流按 %v 后选中 %s，应为 %s", step.key, got, a.Visible[step.want].ID)
		}
	}

	a = assetViewFixture()
	a.ViewMode = 2
	tt = ui.NewTester(a.assetArea, 700, 300)
	press(tt, ui.KeyDown)
	for _, step := range []struct {
		key  ui.Key
		want int
	}{{ui.KeyDown, 1}, {ui.KeyDown, 2}, {ui.KeyRight, 2}, {ui.KeyUp, 1}, {ui.KeyUp, 0}, {ui.KeyUp, 0}} {
		press(tt, step.key)
		if got := selected(a); got != a.Visible[step.want].ID {
			t.Fatalf("列表按 %v 后选中 %s，应为 %s", step.key, got, a.Visible[step.want].ID)
		}
	}
	// 走到最后一行时滚过去，选中的那行完整露在视口里。
	for range 5 {
		press(tt, ui.KeyDown)
	}
	tt.Advance(time.Second)
	if a.scroll.Y <= 0 {
		t.Fatal("选中项走出视口时没有滚动")
	}
	box, ok := tt.Find(a.Visible[5].Name)
	if !ok || box.Y < 0 || box.Y+box.H > 300 {
		t.Fatalf("最后一行没有滚进视口：%+v", box)
	}
}

// 重新筛选会清掉 selectedIndex，方向键仍从选中的那张接着走；点过的卡片不抢回车。
func TestAssetArrowKeysAfterRefineAndEnter(t *testing.T) {
	a := assetViewFixture()
	tt := ui.NewTester(a.assetArea, 700, 600)
	if err := tt.Click(a.Visible[0].Name); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	a.Refine()
	tt.Frame()
	tt.Key(0, ui.KeyRight)
	tt.Frame()
	if a.Selected.Len() != 1 || !a.Selected.Has(a.Visible[2].ID) {
		t.Fatalf("重新筛选后方向键没有从原来的选中项接着走：%v", a.Selected)
	}
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.Selected.Len() != 1 || !a.Selected.Has(a.Visible[2].ID) {
		t.Fatalf("回车被之前点过的卡片拿走了：%v", a.Selected)
	}
}
