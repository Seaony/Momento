package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBrowseScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_BROWSE_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_BROWSE_SCREENSHOTS=1 生成浏览截图")
	}
	dir := "/tmp/momento-eagle/browse"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	type record struct {
		File, State, Language, Theme, Scope string
		Count, Visible                      int
		Bytes                               int64
	}
	records := []record{}
	for _, language := range []string{"zh-CN", "en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, state := range []string{"all", "folder", "trash-empty", "search-empty", "random", "thumbnail-small", "thumbnail-large", "list", "details", "details-bottom", "wide-toolbar", "thumbnail-popover", "icons"} {
				a := fixture.app(t, "large", language, theme)
				switch state {
				case "folder":
					a.SetScope("folder", "folder-00")
				case "trash-empty":
					a.SetScope("trash", "")
				case "search-empty":
					a.Query.Search = "no-matching-asset"
					a.Refine()
				case "random":
					a.SetScope("random", "")
				case "thumbnail-small":
					a.SetThumbnailSize(.5)
				case "thumbnail-large":
					a.ViewMode = 1
					a.SetThumbnailSize(2)
				case "list":
					a.ViewMode = 2
				case "details", "details-bottom":
					asset := a.Visible[0]
					asset.Note = "设计参考与拍摄说明\nNotes for this reference image.\n画像のメモ。"
					asset.SourceURL = "https://example.com/reference/landscape"
					a.Snapshot.Assets[11999] = asset
					a.applySnapshot(a.Snapshot)
					a.Selected.Add(asset.ID)
				}
				// 预装当前范围首屏，截图过程不启动异步解码。
				for _, asset := range a.Visible[:min(256, len(a.Visible))] {
					if _, exists := a.Thumbnails.entries[thumbnailKey{root: filepath.Clean(fixture.root), id: asset.ID}]; exists {
						continue
					}
					index, err := strconv.ParseUint(asset.ID, 16, 64)
					if err != nil {
						t.Fatal(err)
					}
					bitmap := fixture.bitmaps[fixture.snapshot.Assets[int(index)%len(fixture.snapshot.Assets)].ID]
					width, height := bitmap.Size()
					bytes := int64(width*height*4) * 4 / 3
					key := thumbnailKey{root: filepath.Clean(fixture.root), id: asset.ID}
					a.Thumbnails.entries[key] = a.Thumbnails.order.PushFront(thumbnailEntry{key: key, bitmap: bitmap, bytes: bytes})
					a.Thumbnails.bytes += bytes
				}
				if a.Thumbnails.bytes > thumbnailByteLimit || len(a.Thumbnails.entries) > thumbnailItemLimit {
					t.Fatalf("%s截图预装超过正式缓存上限: %d", state, a.Thumbnails.bytes)
				}
				a.Thumbnails.update = func(func()) { t.Errorf("%s/%s/%s截图仍有未预装缩略图", state, language, theme) }
				width := 1100
				if state == "wide-toolbar" || state == "list" {
					width = 1500
				}
				view := a.View
				if state == "icons" {
					view = func(c *ui.Context) {
						a.applyTheme(c)
						ui.Column(c).Fill().Padding(20).Gap(20).Children(func() {
							for _, size := range []float32{19, 24, 28, 48} {
								ui.Row(c).Gap(24).Children(func() {
									for _, icon := range []*ui.SVG{browseRandomIcon, browseLinkIcon, toolbarIcons["thumbnail-size"], sidebarIcon("文件夹"), sidebarIcon("照片")} {
										ui.Image(c, icon).Size(size, size)
									}
								})
							}
						})
					}
				}
				tt := ui.NewTester(view, width, 640)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetDark(theme == "dark")
				tt.Frame()
				if state == "thumbnail-popover" {
					if err := tt.Click(a.T("thumbnail_size")); err != nil {
						t.Fatal(err)
					}
					if _, ok := tt.AccessByRole(ui.RoleSlider, a.T("thumbnail_size")); !ok {
						t.Fatal("1100窗口没有打开缩略图滑块")
					}
				}
				if state == "details-bottom" {
					tt.Scroll(float32(width-100), 540, 0, 500)
				}
				count, size := 0, int64(0)
				for _, asset := range a.Visible {
					count++
					size += asset.Size
				}
				if state == "all" && (count != 12000 || !tt.HasText("12,000")) {
					t.Fatal("全部素材合计不符")
				}
				if state == "folder" && (count != 400 || !tt.HasText("400")) {
					t.Fatal("文件夹素材合计不符")
				}
				if state == "trash-empty" && count != 0 {
					t.Fatal("空回收站仍有素材")
				}
				if state == "search-empty" && len(a.Visible) != 0 {
					t.Fatal("空搜索仍显示素材")
				}
				file := fmt.Sprintf("%s-%s-%s.png", state, language, theme)
				writeScreenshot(t, filepath.Join(dir, file), tt)
				records = append(records, record{file, state, language, theme, a.Query.Scope, count, len(a.Visible), size})
			}
		}
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
