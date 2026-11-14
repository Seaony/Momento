package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func polishedOverlayApp(language, theme, state string) (*App, func(*ui.Context)) {
	a := settingsTestApp(language, theme)
	a.Preferences.Onboarded = true
	view := a.dialogs
	switch state {
	case "preview":
		a = previewFixture(960, 640)
		a.Texts.Language, a.Preferences.Theme = language, theme
		a.Visible[0].Name = "暖光下的山间晨雾.jpg"
		view = a.preview
	case "locked":
		a.Lock.Locked = true
		a.Lock.ErrorKey = "password_wrong"
		view = a.lockScreen
	case "password-change":
		a.Lock.DialogOpen, a.Lock.Mode = true, "change"
		view = a.passwordSettingsDialog
	case "welcome":
		a.Recent = []library.Recent{{Path: "/examples/design.momento", DisplayName: "设计灵感"}, {Path: "/examples/photos.momento", DisplayName: "旅行与摄影"}}
		view = a.welcome
	case "onboarding-theme", "onboarding-extension":
		a.Preferences.Onboarded = false
		if state == "onboarding-extension" {
			a.Welcome.Step = 1
		}
		view = a.welcome
	case "confirmation":
		a.Confirm(a.T("empty_trash"), a.T("empty_trash_detail"), func() {})
	case "new-library":
		a.NewLibraryOpen = true
	case "export":
		a.ExportOpen = true
		a.Selected.Add("sample")
		a.Snapshot.Assets = []model.Asset{{ID: "sample", Name: "暖光", Extension: "jpg"}}
		a.Dialog.ExportWasOpen, a.Dialog.Format, a.Dialog.Quality = true, model.ExportJPEG, .9
	case "error":
		a.Error = a.T("error_import_busy")
	case "extension-picker":
		a.Import = importState{PickerOpen: true, PickerTags: map[string]bool{}, PickerReply: make(chan importChoice, 1)}
		a.Snapshot.Tags = []model.Tag{{ID: "design", Name: "界面设计"}, {ID: "photo", Name: "摄影"}}
		view = a.extensionImportPicker
	}
	return a, func(c *ui.Context) {
		a.applyTheme(c)
		c.Root().Background(c.Theme().Background)
		view(c)
	}
}

func TestOverlayPolishLocalizedLayout(t *testing.T) {
	states := []string{"preview", "locked", "password-change", "welcome", "onboarding-theme", "onboarding-extension", "confirmation", "new-library", "export", "error", "extension-picker"}
	for _, language := range []string{"en", "ja"} {
		for _, theme := range []string{"light", "dark"} {
			for _, state := range states {
				t.Run(fmt.Sprintf("%s/%s/%s", language, theme, state), func(t *testing.T) {
					a, view := polishedOverlayApp(language, theme, state)
					measure := map[string]float32{}
					for _, key := range []string{"cancel", "confirm", "save", "new_library", "open_library", "choose_location", "create_library", "export", "copy_error", "close", "import", "password_unlock", "password_action_change", "onboarding_skip", "onboarding_next", "onboarding_previous", "onboarding_get_extension"} {
						measure[a.T(key)] = 0
					}
					tt := ui.NewTester(func(c *ui.Context) {
						view(c)
						for text := range measure {
							measure[text], _ = c.MeasureText(0, ui.Span{Text: text, Size: 13})
						}
					}, 1100, 640)
					tt.SetLocale(a.Texts.Locale())
					tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 36})
					tt.Advance(5 * time.Millisecond)
					for _, item := range tt.AccessAll() {
						if item.Bounds.W <= 0 || item.Bounds.H <= 0 {
							continue
						}
						if item.Bounds.X < 0 || item.Bounds.Y < 0 || item.Bounds.X+item.Bounds.W > 1100.1 || item.Bounds.Y+item.Bounds.H > 640.1 {
							t.Errorf("元素超出窗口：%s %+v", item.Name, item.Bounds)
						}
						if item.Role == ui.RoleButton {
							if width, ok := measure[item.Name]; ok && width > item.Bounds.W-24+1 {
								t.Errorf("按钮文字被截断：%s，文字 %.1f，可用 %.1f", item.Name, width, item.Bounds.W-24)
							}
						}
					}
					for _, text := range tt.Texts() {
						if strings.ContainsAny(text, "⟦⟧") || strings.Contains(text, "%!") {
							t.Errorf("文案错误：%s", text)
						}
					}
				})
			}
		}
	}
}

func TestOverlayPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_OVERLAY_POLISH_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_OVERLAY_POLISH_SCREENSHOTS=1 输出预览与对话框截图")
	}
	dir := "/tmp/momento-polish/shots/preview"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	states := []string{"preview", "locked", "password-change", "welcome", "onboarding-theme", "onboarding-extension", "confirmation", "new-library", "export", "extension-picker"}
	for _, theme := range []string{"light", "dark"} {
		for _, state := range states {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a, view := polishedOverlayApp("zh-CN", theme, state)
				tt := ui.NewTester(view, 1440, 900)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: 44, Left: 80, Bottom: 36})
				tt.Advance(5 * time.Millisecond)
				writeScreenshot(t, filepath.Join(dir, state+"-"+theme+".png"), tt)
			})
		}
	}
}
