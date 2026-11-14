package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMomentoIconsRenderAtApplicationSizes(t *testing.T) {
	for name, icon := range momentoIcons {
		for _, size := range []int{20, 22} {
			t.Run(fmt.Sprintf("%s/%d", name, size), func(t *testing.T) {
				img := icon.Image(size, size, color.NRGBA{R: 255, A: 255})
				other := icon.Image(size, size, color.NRGBA{B: 255, A: 255})
				if !bytes.Equal(img.Pix, other.Pix) {
					t.Fatal("图标受文字颜色影响")
				}
				opaque, warm, cream := 0, 0, 0
				for y := 0; y < size; y++ {
					for x := 0; x < size; x++ {
						c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
						if c.A < 220 {
							continue
						}
						opaque++
						if int(c.R)-int(c.B) > 25 {
							warm++
						}
						if c.R > 240 && c.G > 225 && c.B > 210 {
							cream++
						}
					}
				}
				if opaque < 30 || warm < 15 || cream < 2 {
					t.Fatalf("原生图标缺少完整主体或奶油内容：主体%d，暖色%d，奶油%d", opaque, warm, cream)
				}
			})
		}
	}
	for name, canonical := range appIconAliases {
		if momentoIcons[canonical] == nil {
			t.Errorf("图标别名 %s 指向不存在的 %s", name, canonical)
		}
	}
}

func TestMomentoIconScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_ICON_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ICON_SCREENSHOTS=1 生成实际尺寸图标总览")
	}
	const dir = "/tmp/momento-polish/shots/icons"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(momentoIcons))
	for name := range momentoIcons {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, appearance := range []string{"light", "dark"} {
		t.Run(appearance, func(t *testing.T) {
			view := func(c *ui.Context) {
				c.SetTheme(momentoTheme(appearance, false))
				ui.Column(c).Fill().Background(c.Theme().Background).Padding(24).Gap(20).Children(func() {
					ui.Text(c, "Momento · 暖橘物件图标").FontSize(20).Bold().TextColor(titleColor(c.Theme()))
					ui.Text(c, "每项左侧20px · 右侧22px · 原生渲染").FontSize(12).TextColor(c.Theme().TextMuted)
					for start := 0; start < len(names); start += 6 {
						ui.Row(c).Gap(12).Children(func() {
							for _, name := range names[start:min(start+6, len(names))] {
								ui.Column(c).Width(136).Gap(8).Children(func() {
									ui.Row(c).Height(26).Gap(12).AlignItems(ui.Center).Children(func() {
										ui.Image(c, momentoIcons[name]).Size(20, 20)
										ui.Image(c, momentoIcons[name]).Size(22, 22)
									})
									ui.Text(c, name).FontSize(11).TextColor(c.Theme().TextMuted).SingleLine()
								})
							}
						})
					}
				})
			}
			tt := ui.NewTester(view, 944, 820)
			tt.SetLocale("zh-Hans")
			tt.Frame()
			writeScreenshot(t, filepath.Join(dir, "native-"+appearance+".png"), tt)
		})
	}
}
