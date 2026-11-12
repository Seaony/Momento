package ui

import (
	"github.com/egoist/mygo/ui"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestPreviewSessionKeepsOrderWhileRecentItemsMoveToFront(t *testing.T) {
	a := &App{Visible: []model.Asset{{ID: "a"}, {ID: "b"}, {ID: "c"}}, Preview: previewState{ID: "a", Open: true}}
	a.movePreview(1)
	if a.Preview.ID != "b" {
		t.Fatalf("第一次下一张=%q，预期b", a.Preview.ID)
	}
	a.Visible = []model.Asset{{ID: "b"}, {ID: "a"}, {ID: "c"}}
	a.movePreview(1)
	if a.Preview.ID != "c" {
		t.Fatalf("最近使用重排后下一张=%q，预期c", a.Preview.ID)
	}
	a.Visible = []model.Asset{{ID: "c"}, {ID: "b"}, {ID: "a"}}
	a.movePreview(1)
	if a.Preview.ID != "c" {
		t.Fatalf("末尾越界=%q，预期c", a.Preview.ID)
	}
	a.movePreview(-1)
	if a.Preview.ID != "b" {
		t.Fatalf("上一张=%q，预期b", a.Preview.ID)
	}
	a.Visible = []model.Asset{{ID: "b"}, {ID: "c"}, {ID: "a"}}
	a.movePreview(-1)
	if a.Preview.ID != "a" {
		t.Fatalf("再次重排后上一张=%q，预期a", a.Preview.ID)
	}
}

func TestPreviewRecentReorderKeepsPositionAndButtons(t *testing.T) {
	a := previewFixture(288, 192)
	a.Texts.Language = "en"
	a.Preferences.Theme = "light"
	seed := a.Visible[0]
	items := make([]model.Asset, 3)
	for i, id := range []string{"a", "b", "c"} {
		items[i] = seed
		items[i].ID = id
		items[i].Name = []string{"Alpha.png", "Bravo.png", "Charlie.png"}[i]
	}
	a.Visible = items
	a.assetByID = map[string]model.Asset{"a": items[0], "b": items[1], "c": items[2]}
	bitmap, pixelBitmap := a.Preview.Bitmap, a.Preview.pixelBitmap
	ready := func() {
		asset, ok := a.previewAsset(a.Preview.ID)
		if !ok {
			t.Fatal("预览素材丢失")
		}
		a.Preview.target(a.Library.Root, asset.ID, asset.Rotation)
		a.Preview.Bitmap, a.Preview.pixelBitmap = bitmap, pixelBitmap
	}
	a.Preview.ID = "a"
	ready()
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.preview(c) }, 1100, 640)
	tt.Advance(5 * time.Millisecond)
	check := func(name string, position int, previousDisabled, nextDisabled bool) {
		t.Helper()
		if !tt.HasText(name) || !tt.HasText(a.T("preview_position", position, 3)) {
			t.Fatalf("标题或计数跟随重排改变：%v", tt.Texts())
		}
		for _, button := range []struct {
			key      string
			disabled bool
		}{{"previous_asset", previousDisabled}, {"next_asset", nextDisabled}} {
			info, ok := tt.AccessByRole(ui.RoleButton, a.T(button.key))
			if !ok || info.Disabled != button.disabled {
				t.Fatalf("%s禁用状态%+v，预期%v", button.key, info, button.disabled)
			}
		}
		capturePreviewNavigation(t, tt, position)
	}
	check("Alpha.png", 1, true, false)
	a.movePreview(1)
	a.Visible = []model.Asset{items[1], items[0], items[2]}
	ready()
	tt.Frame()
	tt.Advance(5 * time.Millisecond)
	check("Bravo.png", 2, false, false)
	// 旋转重载和素材离开当前筛选都不能改变会话顺序。
	changed := a.assetByID["b"]
	changed.Rotation = 90
	a.assetByID["b"] = changed
	a.Visible = []model.Asset{items[2], items[0]}
	ready()
	tt.Frame()
	tt.Advance(5 * time.Millisecond)
	if !tt.HasText("Bravo.png") || !slices.Equal(a.Preview.navigation, []string{"a", "b", "c"}) {
		t.Fatal("旋转或范围变化重置了预览顺序")
	}
	a.movePreview(1)
	a.Visible = []model.Asset{items[2], items[1], items[0]}
	ready()
	tt.Frame()
	tt.Advance(5 * time.Millisecond)
	check("Charlie.png", 3, false, true)
}

func TestPreviewNavigationStartsFreshAfterClosing(t *testing.T) {
	a := &App{Visible: []model.Asset{{ID: "a"}, {ID: "b"}, {ID: "c"}}, Preview: previewState{ID: "a", Open: true}}
	a.InitPreviewNavigation()
	a.Visible = []model.Asset{{ID: "c"}, {ID: "b"}, {ID: "a"}}
	a.Preview.target("library", "a", 90)
	if !slices.Equal(a.Preview.navigation, []string{"a", "b", "c"}) {
		t.Fatal("旋转重载改掉了导航顺序")
	}
	a.Preview.Open = false
	a.Preview.target("", "", 0)
	if a.Preview.navigation != nil {
		t.Fatal("关闭后保留了旧导航顺序")
	}
	a.Preview.Open = true
	a.Preview.ID = "c"
	a.movePreview(1)
	if a.Preview.ID != "b" || !slices.Equal(a.Preview.navigation, []string{"c", "b", "a"}) {
		t.Fatal("重开没有采用新的范围顺序")
	}
}

func capturePreviewNavigation(t *testing.T, tt *ui.Tester, position int) {
	t.Helper()
	if os.Getenv("MOMENTO_PREVIEW_SCREENSHOTS") != "1" {
		return
	}
	dir := os.Getenv("MOMENTO_PREVIEW_SCREENSHOT_DIR")
	if dir == "" {
		dir = "/tmp/momento-eagle/preview/screenshots"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, []string{"recent-navigation-1.png", "recent-navigation-2.png", "recent-navigation-3.png"}[position-1]))
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(file, tt.Image())
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func TestPreviewEntryCapturesNewScopeOrder(t *testing.T) {
	a := &App{Visible: []model.Asset{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	a.ShowPreview("a", false)
	if !slices.Equal(a.Preview.navigation, []string{"a", "b", "c"}) {
		t.Fatal("打开预览时没有固定导航顺序")
	}
	a.Visible = []model.Asset{{ID: "c"}, {ID: "b"}, {ID: "a"}}
	a.ClosePreview()
	a.ShowPreview("c", false)
	if !slices.Equal(a.Preview.navigation, []string{"c", "b", "a"}) {
		t.Fatal("关闭重开后没有固定新的导航顺序")
	}
	a.movePreview(1)
	a.movePreview(1)
	if a.Preview.ID != "a" {
		t.Fatal("重开的预览没有沿新顺序翻页")
	}
}
