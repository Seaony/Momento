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

func TestMomentoSymbolsFollowParentColor(t *testing.T) {
	for name, symbol := range momentoSymbols {
		t.Run(name, func(t *testing.T) {
			red := symbol.Image(14, 14, color.NRGBA{R: 255, A: 255})
			blue := symbol.Image(14, 14, color.NRGBA{B: 255, A: 255})
			if bytes.Equal(red.Pix, blue.Pix) {
				t.Fatal("符号没有采用 currentColor")
			}
			coverage := 0
			for i := 3; i < len(red.Pix); i += 4 {
				coverage += int(red.Pix[i])
			}
			if coverage < 4*255 || coverage > 60*255 {
				t.Fatalf("14px符号笔画缺失或包含大面积填充：覆盖量%d", coverage)
			}
			images := make([][]byte, 0, 2)
			for _, ink := range []ui.Color{ui.Hex("#D9622C"), ui.Hex("#7E5E4C")} {
				tt := ui.NewTester(func(c *ui.Context) {
					ui.Box(c).Fill().TextColor(ink).Children(func() { appSymbol(c, name) })
				}, 24, 24)
				tt.Frame()
				images = append(images, slices.Clone(tt.Image().Pix))
			}
			if bytes.Equal(images[0], images[1]) {
				t.Fatal("appSymbol 阻止了父控件修改文字颜色")
			}
		})
	}
	for name, canonical := range appSymbolAliases {
		if momentoSymbols[canonical] == nil {
			t.Errorf("符号别名 %s 指向不存在的 %s", name, canonical)
		}
	}
}

func TestMomentoSymbolScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_SYMBOL_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_SYMBOL_SCREENSHOTS=1 生成小操作符号总览")
	}
	const dir = "/tmp/momento-polish/shots/symbols"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	names := []string{"chevron-left", "chevron-right", "chevron-down", "chevron-up", "chevrons-up-down", "plus", "minus", "close", "check", "rotate-left", "rotate-right"}
	for _, appearance := range []string{"light", "dark"} {
		t.Run(appearance, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(momentoTheme(appearance, false))
				ui.Column(c).Fill().Padding(24).Gap(10).Background(c.Theme().Background).Children(func() {
					ui.Text(c, "Momento · 小操作符号").FontSize(20).Bold().TextColor(titleColor(c.Theme()))
					ui.Row(c).Gap(24).TextColor(c.Theme().TextMuted).Children(func() {
						ui.Text(c, "符号").Width(180)
						ui.Text(c, "14px").Width(72)
						ui.Text(c, "16px").Width(72)
						ui.Text(c, "18px").Width(72)
						ui.Text(c, "父控件悬停颜色").Width(136)
					})
					for _, name := range names {
						ui.Row(c).Height(32).Gap(24).AlignItems(ui.Center).TextColor(c.Theme().TextMuted).Children(func() {
							ui.Text(c, name).Width(180).FontSize(13)
							for _, size := range []float32{14, 16, 18} {
								ui.Box(c).Width(72).Children(func() { appSymbol(c, name).Size(size, size) })
							}
							ui.Box(c).Width(136).TextColor(c.Theme().Text).Children(func() { appSymbol(c, name) })
						})
					}
				})
			}, 740, 570)
			tt.SetLocale("zh-Hans")
			tt.Frame()
			writeScreenshot(t, filepath.Join(dir, "native-"+appearance+".png"), tt)
		})
	}
}
