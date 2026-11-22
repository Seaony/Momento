package ui

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 宣传图用的主界面截图：把 MOMENTO_PROMO_IMAGES 里的图导入一个临时图库，按 Retina 2 倍渲染中英文、浅深色。
// MOMENTO_PROMO_IMAGES=<图片目录> MOMENTO_PROMO_OUT=<输出目录> go test ./ui -run '^TestPromoScreenshots$' -count=1
func TestPromoScreenshots(t *testing.T) {
	images, out := os.Getenv("MOMENTO_PROMO_IMAGES"), os.Getenv("MOMENTO_PROMO_OUT")
	if images == "" || out == "" {
		t.Skip("设置 MOMENTO_PROMO_IMAGES 和 MOMENTO_PROMO_OUT 生成宣传截图")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(images)
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, e := range entries {
		if ext := strings.ToLower(filepath.Ext(e.Name())); ext == ".jpg" || ext == ".png" {
			paths = append(paths, filepath.Join(images, e.Name()))
		}
	}
	slices.Sort(paths)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	// 检查器会显示路径，放在一个短而干净的位置。
	base := "/tmp/momento-promo-library"
	if err := os.RemoveAll(base); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	lib, err := library.Create(ctx, filepath.Join(base, "Moodboard.momento"), "Moodboard")
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	result, err := assets.New(lib.Root, lib.DB).Import(ctx, model.ImportRequest{Paths: paths}, nil)
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("导入宣传素材：%v %v", err, result.Errors)
	}
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 文件夹和标签让侧栏和检查器看起来像真在用的图库。
	snapshot.Folders = []model.Folder{{ID: "travel", Name: "Travel", Icon: "camera"}, {ID: "interior", Name: "Interior", Icon: "palette"}, {ID: "posters", Name: "Posters"}}
	snapshot.Tags = []model.Tag{{ID: "light", Name: "Light"}, {ID: "warm", Name: "Warm"}, {ID: "calm", Name: "Calm"}}
	bitmaps := map[string]*ui.Bitmap{}
	for i := range snapshot.Assets {
		asset := &snapshot.Assets[i]
		asset.FolderIDs = []string{snapshot.Folders[i%len(snapshot.Folders)].ID}
		asset.TagIDs = []string{snapshot.Tags[i%len(snapshot.Tags)].ID}
		asset.Favorite = i%4 == 0
		bitmap, _, err := loadThumbnail(filepath.Join(lib.Root, "thumbnails", asset.ID))
		if err != nil {
			t.Fatal(err)
		}
		bitmaps[asset.ID] = bitmap
	}
	for _, language := range []string{"zh-CN", "en"} {
		for _, appearance := range []string{"light", "dark"} {
			a := &App{
				Library: lib,
				Texts:   i18n.Catalog{Language: language, SystemLanguage: language},
				Preferences: func() Preferences {
					p := DefaultPreferences()
					p.Language, p.Theme, p.Onboarded, p.ThumbnailSize = language, appearance, true, 1.5
					return p
				}(),
				SidebarVisible: true, InspectorVisible: true, SidebarWidth: 236,
				Query: catalog.Query{Sort: "added", Descending: true}, selectedIndex: -1, anchor: -1,
				ctx: context.Background(), cancel: func() {},
			}
			a.Thumbnails = newThumbnailCache(func(func()) {})
			for id, bitmap := range bitmaps {
				width, height := bitmap.Size()
				bytes := int64(width*height*4) * 4 / 3
				key := thumbnailKey{root: filepath.Clean(lib.Root), id: id}
				a.Thumbnails.entries[key] = a.Thumbnails.order.PushFront(thumbnailEntry{key: key, bitmap: bitmap, bytes: bytes})
				a.Thumbnails.bytes += bytes
			}
			a.applySnapshot(snapshot)
			a.Selected.Add(a.Visible[1].ID)
			tt := ui.NewTester(a.View, 1440, 900)
			tt.SetScale(2)
			tt.SetLocale(a.Texts.Locale())
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.SetWindowActive(true)
			tt.Advance(time.Second)
			writeScreenshot(t, filepath.Join(out, "main-"+language+"-"+appearance+".png"), tt)
		}
	}
}
