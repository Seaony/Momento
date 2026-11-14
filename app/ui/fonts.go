package ui

import (
	"embed"
	"fmt"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

//go:embed fonts/*.ttf fonts/*-OFL.txt
var bundledFonts embed.FS

func init() {
	for _, face := range []struct{ file, family string }{
		{"Inter-Regular.ttf", "Inter"}, {"Inter-Medium.ttf", "Inter"},
		{"Inter-SemiBold.ttf", "Inter"}, {"Inter-Bold.ttf", "Inter"},
		{"GeistMono-Regular.ttf", "Geist Mono"},
	} {
		data, err := bundledFonts.ReadFile("fonts/" + face.file)
		if err == nil {
			err = ui.RegisterFont(data, face.family)
		}
		if err != nil {
			panic(fmt.Errorf("载入内置字体 %s：%w", face.file, err))
		}
	}
}

func cjkFont(locale string) string {
	if strings.HasPrefix(locale, "ja") {
		switch runtime.GOOS {
		case "darwin":
			return "Hiragino Sans, Yu Gothic, Noto Sans CJK JP, Source Han Sans JP"
		case "windows":
			return "Yu Gothic UI, Yu Gothic, Meiryo, Noto Sans CJK JP, Source Han Sans JP"
		default:
			return "Noto Sans CJK JP, Source Han Sans JP, Noto Sans JP"
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return "PingFang SC, Noto Sans CJK SC, Source Han Sans SC"
	case "windows":
		return "Microsoft YaHei UI, Microsoft YaHei, Noto Sans CJK SC, Source Han Sans SC"
	default:
		return "Noto Sans CJK SC, Source Han Sans SC, Noto Sans SC"
	}
}

func uiFont(locale string) string   { return "Inter, " + cjkFont(locale) }
func monoFont(c *ui.Context) string { return "Geist Mono, " + cjkFont(c.Locale()) }
