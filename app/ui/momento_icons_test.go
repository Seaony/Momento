package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestMomentoIconsRenderAtApplicationSizes(t *testing.T) {
	for name, regular := range phosphorRegular {
		fill := phosphorFill[name]
		if regular == nil || fill == nil {
			t.Fatalf("图标 %s 缺少 Regular 或 Fill", name)
		}
		for _, size := range []int{18, 16, 14} {
			t.Run(fmt.Sprintf("%s/%d", name, size), func(t *testing.T) {
				render := func(filled bool, ink ui.Color) []byte {
					tt := ui.NewTester(func(c *ui.Context) {
						ui.Box(c).Fill().TextColor(ink).Children(func() {
							phosphorIcon(c, name, filled).Size(float32(size), float32(size))
						})
					}, 24, 24)
					tt.Frame()
					return slices.Clone(tt.Image().Pix)
				}
				red, blue := render(false, ui.Hex("#D9622C")), render(false, ui.Hex("#7E5E4C"))
				if bytes.Equal(red, blue) {
					t.Fatal("Regular 未显示，或不随父控件颜色变化")
				}
				filled := render(true, ui.Hex("#D9622C"))
				if bytes.Equal(red, filled) {
					t.Fatal("Regular 与 Fill 没有切换")
				}
				if bytes.Equal(filled, render(true, ui.Hex("#7E5E4C"))) {
					t.Fatal("Fill 不随父控件颜色变化")
				}
			})
		}
	}
	for name, canonical := range appIconAliases {
		if phosphorRegular[canonical] == nil || phosphorFill[canonical] == nil {
			t.Errorf("图标语义 %s 指向不存在的 %s", name, canonical)
		}
	}
}

func TestMomentoIconScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_ICON_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_ICON_SCREENSHOTS=1 生成 Phosphor 图标总览")
	}
	const dir = "/tmp/momento-polish/shots2/icons"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(phosphorRegular))
	for name := range phosphorRegular {
		names = append(names, name)
	}
	slices.Sort(names)
	selected := map[string]bool{"images": true, "clock-counter-clockwise": true, "heart": true, "shuffle": true, "tray": true, "tag-simple": true, "tag": true, "folder": true, "folder-simple-star": true, "trash": true}
	for _, appearance := range []string{"light", "dark"} {
		t.Run(appearance, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(momentoTheme(appearance, false))
				ui.Column(c).Fill().Background(c.Theme().Background).Padding(24).Gap(20).Children(func() {
					ui.Text(c, "Momento · Phosphor 图标").FontSize(20).Bold().TextColor(titleColor(c.Theme()))
					ui.Text(c, "每项 Regular 18 / 16 / 14px · 侧栏当前行额外显示主题色 Fill 18px").FontSize(12).TextColor(c.Theme().TextMuted)
					for start := 0; start < len(names); start += 6 {
						ui.Row(c).Gap(16).TextColor(c.Theme().TextMuted).Children(func() {
							for _, name := range names[start:min(start+6, len(names))] {
								ui.Column(c).Width(170).Gap(8).Children(func() {
									ui.Row(c).Height(24).Gap(12).AlignItems(ui.Center).Children(func() {
										for _, size := range []float32{18, 16, 14} {
											phosphorIcon(c, name, false).Size(size, size)
										}
										if selected[name] {
											phosphorIcon(c, name, true).Size(18, 18).TextColor(c.Theme().Accent)
										}
									})
									ui.Text(c, name).FontSize(11).SingleLine()
								})
							}
						})
					}
				})
			}, 1160, 800)
			tt.SetLocale("zh-Hans")
			tt.Frame()
			writeScreenshot(t, filepath.Join(dir, "native-"+appearance+".png"), tt)
		})
	}
}
