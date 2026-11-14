package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func settingsMotionFrame(tt *ui.Tester) *image.RGBA {
	frame := tt.Image()
	copy := image.NewRGBA(frame.Bounds())
	draw.Draw(copy, copy.Bounds(), frame, frame.Bounds().Min, draw.Src)
	return copy
}

func settingsMotionFrames(tt *ui.Tester) []*image.RGBA {
	frames := []*image.RGBA{settingsMotionFrame(tt)}
	for _, delta := range []time.Duration{87500 * time.Microsecond, 87500 * time.Microsecond, 75 * time.Millisecond} {
		tt.Advance(delta)
		frames = append(frames, settingsMotionFrame(tt))
	}
	return frames
}

func writeSettingsMotionFrames(t *testing.T, name string, frames []*image.RGBA) {
	t.Helper()
	dir := "/tmp/momento-polish/shots2/motion"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	bounds := frames[0].Bounds()
	contact := image.NewRGBA(image.Rect(0, 0, bounds.Dx()*2, bounds.Dy()*2))
	for i, frame := range frames {
		draw.Draw(contact, image.Rect((i%2)*bounds.Dx(), (i/2)*bounds.Dy(), (i%2+1)*bounds.Dx(), (i/2+1)*bounds.Dy()), frame, bounds.Min, draw.Src)
	}
	file, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, contact); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSettingsSegmentSlidesAndKeepsBinding(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	a.Settings.Tab = SettingsPreview
	tt := polishedSettingsTester(a, UpdateCheckStatus{}, nil)
	if err := tt.Click(a.T("preview_fit_window")); err != nil {
		t.Fatal(err)
	}
	frames := settingsMotionFrames(tt)
	if a.Preferences.PreviewFit != "fit" {
		t.Fatal("分段切换没有保存值")
	}
	if bytes.Equal(frames[0].Pix, frames[1].Pix) || bytes.Equal(frames[1].Pix, frames[3].Pix) {
		t.Fatal("分段底块没有经过中间位置")
	}
	tt.Key(0, ui.KeyLeft)
	if a.Preferences.PreviewFit != "smart" {
		t.Fatal("分段动画打断了方向键切换")
	}
	if os.Getenv("MOMENTO_SETTINGS_MOTION") == "1" {
		writeSettingsMotionFrames(t, "settings-segment-0-35-70-100", frames)
	}
}

func TestSettingsPageAndDialogTransitions(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	tt := polishedSettingsTester(a, UpdateCheckStatus{}, nil)
	if err := tt.Click(a.T("settings_controls")); err != nil {
		t.Fatal(err)
	}
	pages := settingsMotionFrames(tt)
	if bytes.Equal(pages[0].Pix, pages[1].Pix) || !tt.HasText(a.T("double_click")) {
		t.Fatal("设置切页没有交叉淡入或内容丢失")
	}
	b := settingsTestApp("zh-CN", "light")
	dialog := ui.NewTester(func(c *ui.Context) {
		b.applyTheme(c)
		ui.Box(c).Fill().Background(c.Theme().Background)
		b.dialogs(c)
	}, 760, 560)
	b.Ask("重命名", "参考图片", func(string) {})
	dialog.Frame()
	opening := settingsMotionFrames(dialog)
	if bytes.Equal(opening[0].Pix, opening[1].Pix) {
		t.Fatal("对话框打开没有中间帧")
	}
	if err := dialog.Click(b.T("cancel")); err != nil {
		t.Fatal(err)
	}
	closing := settingsMotionFrames(dialog)
	if b.PromptOpen || bytes.Equal(closing[0].Pix, closing[1].Pix) {
		t.Fatal("对话框关闭没有退出动画")
	}
	if os.Getenv("MOMENTO_SETTINGS_MOTION") == "1" {
		writeSettingsMotionFrames(t, "settings-page-0-35-70-100", pages)
		writeSettingsMotionFrames(t, "dialog-open-0-35-70-100", opening)
		writeSettingsMotionFrames(t, "dialog-close-0-35-70-100", closing)
	}
}

func TestOnboardingStepSlides(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.welcome(c) }, 900, 640)
	if err := tt.Click(a.T("onboarding_next")); err != nil {
		t.Fatal(err)
	}
	frames := settingsMotionFrames(tt)
	if a.Welcome.Step != 1 || bytes.Equal(frames[0].Pix, frames[1].Pix) || !tt.HasText(a.T("onboarding_extension")) {
		t.Fatal("引导切换没有水平过渡或步骤丢失")
	}
	if os.Getenv("MOMENTO_SETTINGS_MOTION") == "1" {
		writeSettingsMotionFrames(t, "onboarding-step-0-35-70-100", frames)
	}
}

func TestSettingsSecondRoundScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SETTINGS_SECOND_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SETTINGS_SECOND_SCREENSHOTS=1 生成设置动画后的界面截图")
	}
	dir := "/tmp/momento-polish/shots2/settings"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"general", "sidebar", "controls", "preview", "notifications", "shortcuts"} {
			a, status := finalSettingsScreenshotState("zh-CN", theme, state)
			tt := polishedSettingsTester(a, status, nil)
			writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("settings-%s-%s.png", state, theme)), tt)
		}
	}
}

func TestDialogsSecondRoundScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_DIALOGS_SECOND_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_DIALOGS_SECOND_SCREENSHOTS=1 生成对话框和引导截图")
	}
	dir := "/tmp/momento-polish/shots2/dialogs"
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"new-library", "export", "prompt", "confirm", "extension-picker", "lock", "password", "onboarding-extension", "welcome"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				a := settingsTestApp("zh-CN", theme)
				a.Preferences.Onboarded = true
				view := a.dialogs
				switch state {
				case "new-library":
					a.NewLibraryOpen = true
				case "export":
					a.ExportOpen = true
					a.Dialog.Format = model.ExportJPEG
				case "prompt":
					a.Ask("重命名", "山间晨雾", func(string) {})
				case "confirm":
					a.Confirm("移到回收站", "这张图片会移到回收站，你可以稍后恢复。", "移到回收站", func() {})
				case "extension-picker":
					a.Import = importState{PickerOpen: true, PickerReply: make(chan importChoice, 1), PickerTags: map[string]bool{}}
					a.Snapshot.Tags = []model.Tag{{ID: "reference", Name: "设计参考"}, {ID: "photo", Name: "摄影"}}
					view = a.extensionImportPicker
				case "lock":
					a.Lock.Locked = true
					view = a.lockScreen
				case "password":
					a.Lock.DialogOpen, a.Lock.Mode = true, "enable"
					view = a.passwordSettingsDialog
				case "onboarding-extension":
					a.Preferences.Onboarded, a.Welcome.Step = false, 1
					view = a.welcome
				case "welcome":
					a.Recent = []library.Recent{{Path: "sample.momento", DisplayName: "设计参考"}}
					view = a.welcome
				}
				tt := ui.NewTester(func(c *ui.Context) {
					a.applyTheme(c)
					c.Root().Background(c.Theme().Background)
					view(c)
				}, 900, 640)
				tt.SetWindowActive(true)
				tt.SetLocale(a.Texts.Locale())
				tt.Frame()
				writeScreenshot(t, filepath.Join(dir, fmt.Sprintf("%s-%s.png", state, theme)), tt)
			})
		}
	}
}
