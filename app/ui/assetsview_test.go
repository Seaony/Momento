package ui

import (
	"fmt"
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
	for mode := 0; mode < 3; mode++ {
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
	a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: Preferences{Appearance: "dark"}}
	dark := false
	tt := ui.NewTester(func(c *ui.Context) { a.View(c); dark = c.Theme().Dark }, 1100, 640)
	if !dark {
		t.Fatal("没有使用深色设置")
	}
	a.Preferences.Appearance = "system"
	tt.SetDark(false)
	tt.Frame()
	if dark {
		t.Fatal("跟随系统仍残留深色设置")
	}
}
