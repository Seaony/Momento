package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestAssetHeaderOverlayKeepsFirstRowVisible(t *testing.T) {
	for mode := 0; mode < 3; mode++ {
		for _, importing := range []bool{false, true} {
			t.Run(fmt.Sprintf("mode-%d/importing-%t", mode, importing), func(t *testing.T) {
				a := assetViewFixture()
				a.ViewMode = mode
				snapshot := a.Snapshot
				for i := 6; i < 30; i++ {
					snapshot.Assets = append(snapshot.Assets, model.Asset{ID: fmt.Sprint(i), Name: fmt.Sprintf("Photo %02d", i), Width: 600, Height: 400})
				}
				a.applySnapshot(snapshot)
				if importing {
					a.Progress = &model.ImportProgress{Stage: "importing", Total: 3}
				}
				tt := ui.NewTester(a.assetArea, 700, 600)
				box, ok := tt.Find(a.Visible[0].Name)
				if !ok || box.Y < a.contentTopInset() || box.Y+box.H > 600 {
					t.Fatalf("第一行被悬浮页头遮挡或超出视口：%+v", box)
				}
				before := box.Y
				tt.Scroll(350, 400, 0, 90)
				box, ok = tt.Find(a.Visible[0].Name)
				if ok && box.Y >= before {
					t.Fatalf("顶部留白没有随内容滚动：%+v", box)
				}
			})
		}
	}
}

func TestAssetKeyboardFocusStaysBelowOverlay(t *testing.T) {
	for mode := 0; mode < 3; mode++ {
		for _, scale := range []float32{1, 2} {
			t.Run(fmt.Sprintf("%d/%g", mode, scale), func(t *testing.T) {
				a := assetViewFixture()
				a.ViewMode, a.Preferences.ThumbnailSize = mode, scale
				a.Visible[0].Favorite = true
				a.Progress = &model.ImportProgress{Stage: "importing", Total: 3}
				tt := ui.NewTester(a.assetArea, 700, 600)
				if err := tt.Click(a.Visible[0].Name); err != nil {
					t.Fatal(err)
				}
				tt.Move(0, 0)
				checked := 0
				for i := 0; i < 14; i++ {
					tt.Key(0, ui.KeyTab)
					if tt.Focused(a.T("favorite")) {
						box, ok := tt.Find(a.T("favorite"))
						if !ok || box.Y < a.contentTopInset() {
							t.Fatalf("收藏按钮键盘焦点被页头遮挡：%+v", box)
						}
					}
					for _, asset := range a.Visible {
						if tt.Focused(asset.Name) {
							box, ok := tt.Find(asset.Name)
							if !ok || box.Y < a.contentTopInset() {
								t.Fatalf("键盘焦点被页头遮挡：%s %+v", asset.Name, box)
							}
							checked++
						}
					}
				}
				if checked == 0 {
					t.Fatal("没有验证到素材键盘焦点")
				}
			})
		}
	}
}

func TestAssetDimensionsOnlyAppearOnHover(t *testing.T) {
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for mode := 0; mode < 2; mode++ {
			t.Run(fmt.Sprintf("%s/%d", language, mode), func(t *testing.T) {
				a := assetViewFixture()
				a.Texts.Language, a.ViewMode = language, mode
				tt := ui.NewTester(a.assetArea, 700, 600)
				if tt.HasText("600 × 400") {
					t.Fatal("未悬停时仍有尺寸角标")
				}
				box, ok := tt.Find(a.Visible[0].Name)
				if !ok {
					t.Fatal("没有找到素材")
				}
				tt.Move(box.X+box.W/2, box.Y+box.H/2)
				if !tt.HasText("600 × 400") || tt.Cursor() != ui.CursorPointer {
					t.Fatal("悬停没有显示尺寸和手型指针")
				}
				dims, ok := tt.Find("600 × 400")
				if !ok || dims.X < box.X || dims.X+dims.W > box.X+box.W || dims.Y < box.Y || dims.Y+dims.H > box.Y+box.H {
					t.Fatalf("尺寸胶囊超出图片：%+v，图片%+v", dims, box)
				}
				tt.Move(0, 0)
				tt.Advance(motionColor.Duration)
				if tt.HasText("600 × 400") {
					t.Fatal("移开鼠标后尺寸角标没有隐藏")
				}
			})
		}
	}
}

func TestAssetPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_ASSET_POLISH_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ASSET_POLISH_SCREENSHOTS=1 生成素材区截图")
	}
	dir := "/tmp/momento-polish/shots2/assets"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"masonry-hover", "grid-hover", "list-hover", "empty", "search-empty", "importing", "masonry-scroll"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a := fixture.app(t, state, "zh-CN", theme)
				if state == "masonry-scroll" {
					a.Preferences.ThumbnailSize = 1.5
				}
				tt := ui.NewTester(a.View, 1440, 900)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				tt.Frame()
				prepareScreenshotState(t, a, tt, state)
				if state == "masonry-scroll" {
					tt.Scroll(650, 450, 0, 180)
				}
				tt.Advance(time.Second)
				if len(a.Visible) > 0 {
					box, ok := tt.Find(a.Visible[0].Name)
					if state != "masonry-scroll" && (!ok || box.Y < a.contentTopInset()) {
						t.Fatalf("首行落在页头后：%+v", box)
					}
				}
				writeScreenshot(t, filepath.Join(dir, state+"-"+theme+".png"), tt)
			})
		}
	}
}

func TestAssetThirdRoundLocalesFit(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		for _, state := range []string{"list", "search", "scope", "trash", "empty"} {
			t.Run(language+"/"+state, func(t *testing.T) {
				a := assetViewFixture()
				a.Texts.Language, a.Preferences.Language = language, language
				a.ViewMode = 2
				title, hint := "", ""
				switch state {
				case "search":
					a.Query.Search = "mountain landscape"
					a.Refine()
					title, hint = a.T("search_empty", a.Query.Search), a.T("search_empty_hint")
				case "scope":
					a.Query.Scope = "favorites"
					a.Refine()
					title, hint = a.T("scope_empty"), a.T("scope_empty_hint")
				case "trash":
					a.Query.Scope = "trash"
					a.Refine()
					title, hint = a.T("trash_empty_title"), a.T("trash_empty_hint")
				case "empty":
					a.applySnapshot(model.Snapshot{})
					title, hint = a.T("drop_images"), a.T("drop_images_hint")
				}
				tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 560, 600)
				tt.SetLocale(a.Texts.Locale())
				if state == "list" {
					values := fmt.Sprintf("600 × 400 · %s", fileSize(a.Visible[0].Size))
					box, ok := tt.Find(values)
					if !ok || box.H > 20 || box.X < 16 || box.X+box.W > 544 {
						t.Fatalf("列表等宽数值发生截断或换行：%+v", box)
					}
					return
				}
				var previousBottom float32
				for _, text := range []string{title, hint} {
					box, ok := tt.Find(text)
					if !ok || box.X < 24 || box.X+box.W > 536 || box.Y < previousBottom || box.Y+box.H > 600 {
						t.Fatalf("空态文字发生越界或重叠：%q %+v", text, box)
					}
					previousBottom = box.Y + box.H
				}
				if state == "empty" && !tt.HasText(a.T("import")) {
					t.Fatal("空图库缺少导入入口")
				}
			})
		}
	}
}

func TestAssetHoverLiftKeepsAdjacentLayout(t *testing.T) {
	a := assetsMotionFixture(t)
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.assetArea(c) }, 700, 600)
	tt.Advance(time.Second)
	box := assetMotionBounds(t, tt, "Photo 0")
	neighbor := assetMotionBounds(t, tt, "Photo 1")
	x, y := int(box.X+box.W/2), int(box.Y-1)
	before := tt.Image().RGBAAt(x, y)
	tt.Move(box.X+box.W/2, box.Y+box.H/2)
	tt.Advance(time.Second)
	if tt.Image().RGBAAt(x, y) == before {
		t.Fatal("悬停图片没有在原顶边上方浮起")
	}
	if rectDistance(neighbor, assetMotionBounds(t, tt, "Photo 1")) > .001 {
		t.Fatal("悬停上浮改变了相邻素材的布局")
	}
	tt.Move(0, 0)
	tt.Advance(time.Second)
	if tt.Image().RGBAAt(x, y) != before {
		t.Fatal("移出后素材没有回到原位置")
	}
}

func TestAssetThirdRoundScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_ASSET_THIRD_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ASSET_THIRD_SCREENSHOTS=1 生成第三轮素材截图")
	}
	dir := "/tmp/momento-polish/shots3/assets"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		t.Run(theme, func(t *testing.T) {
			a := fixture.app(t, "masonry-hover", "zh-CN", theme)
			tt := ui.NewTester(a.View, 1440, 900)
			tt.SetLocale(a.Texts.Locale())
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.SetWindowActive(true)
			tt.Frame()
			prepareScreenshotState(t, a, tt, "masonry-hover")
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(dir, theme+".png"), tt)
		})
	}
}
