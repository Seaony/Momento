package ui

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDialogDesignScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_DIALOG_DESIGN_SCREENSHOTS") != "1" {
		t.Skip("显式生成第三轮对话框与空图库联系图")
	}
	dir := "/tmp/momento-polish/shots3/dialogs-fix"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	states := []string{"welcome", "onboarding-theme", "locked", "new-library", "export", "confirmation", "password-change", "extension-picker", "error"}
	const width, height = 1100, 640
	for _, theme := range []string{"light", "dark"} {
		contact := image.NewRGBA(image.Rect(0, 0, width*3, height*3))
		for i, state := range states {
			a, view := polishedOverlayApp("zh-CN", theme, state)
			tt := ui.NewTester(view, width, height)
			tt.SetLocale(a.Texts.Locale())
			tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
			tt.Advance(time.Second)
			at := image.Pt((i%3)*width, (i/3)*height)
			draw.Draw(contact, image.Rectangle{Min: at, Max: at.Add(image.Pt(width, height))}, tt.Image(), image.Point{}, draw.Src)
		}
		path := filepath.Join(dir, theme+".png")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(f, contact)
		closeErr := f.Close()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
}
