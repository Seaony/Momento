package ui

import (
	"runtime"
	"strings"
)

// 中日文字形：系统界面字体没有汉字时按这里的顺序补。
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

// 和旧版 macOS 一样用系统界面字体（macOS 上是 SF Pro），数字用 tnum 等宽数字，不另带字体。
func uiFont(locale string) string { return "system-ui, " + cjkFont(locale) }
