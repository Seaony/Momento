package ui

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// 检查器 EXIF 区按这个顺序显示；像素尺寸已在详细信息里，水平、垂直分辨率合成一行。
var inspectorEXIFKeys = []string{
	"fileCreatedAt", "fileModifiedAt", "contentCreatedAt", "resolution",
	"colorModel", "profileName", "cameraMake", "cameraModel", "lensModel", "exposureTime",
	"focalLength", "isoSpeedRatings", "flashFired", "fNumber", "exposureProgram", "meteringMode",
	"whiteBalance", "creator",
}

// 图库里存的是原始值（RFC3339 时间、秒数、毫米数、EXIF 枚举号），显示前在这里换成人看的格式。
func inspectorEXIFValue(texts i18n.Catalog, values map[string]string, key string) string {
	raw := strings.TrimSpace(values[key])
	switch key {
	case "resolution":
		width, height := exifNumber(values["dpiWidth"]), exifNumber(values["dpiHeight"])
		if width == "" {
			width = height
		}
		if width == "" {
			return ""
		}
		if height == "" || width == height {
			return width + " dpi"
		}
		return width + " × " + height + " dpi"
	case "fileCreatedAt", "fileModifiedAt":
		// 文件时间是真实时刻，换成本机时区。
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return texts.DateTime(t.Local())
		}
	case "contentCreatedAt":
		// 拍摄时间是相机记下的钟点，不换时区，否则会差出几个小时。
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return texts.DateTime(t)
		}
	case "exposureTime":
		return exposureText(raw)
	case "focalLength":
		if v := exifNumber(raw); v != "" {
			return v + " mm"
		}
	case "fNumber":
		if v := exifNumber(raw); v != "" {
			return "f/" + v
		}
	case "flashFired":
		if raw == "true" || raw == "false" {
			return texts.T("exif_flash_" + raw)
		}
	case "exposureProgram", "meteringMode", "whiteBalance":
		if n, err := strconv.Atoi(raw); err == nil {
			if label, ok := exifEnumKey(key, n); ok {
				return texts.T(label)
			}
		}
	}
	return raw
}

// 曝光时间：不到 1 秒写成 1/N s，和预览底栏一致。
func exposureText(raw string) string {
	t, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || t <= 0 {
		return strings.TrimSpace(raw)
	}
	if t < 1 {
		if n := int(math.Round(1 / t)); n > 1 {
			return "1/" + strconv.Itoa(n) + " s"
		}
	}
	return exifNumber(raw) + " s"
}

// 数字最多留一位小数，去掉末尾的 0；不是数字就原样返回。
func exifNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
}

func exifEnumKey(key string, n int) (string, bool) {
	limit := map[string]int{"exposureProgram": 8, "meteringMode": 6, "whiteBalance": 1}[key]
	if (n >= 0 && n <= limit) || (key == "meteringMode" && n == 255) {
		return "exif_" + key + "_" + strconv.Itoa(n), true
	}
	return "", false
}

func hasInspectorEXIF(texts i18n.Catalog, values map[string]string) bool {
	for _, key := range inspectorEXIFKeys {
		if inspectorEXIFValue(texts, values, key) != "" {
			return true
		}
	}
	return false
}

// 有值的 EXIF 行：[标签, 显示值]。
func (a *App) inspectorEXIFItems(values map[string]string) [][2]string {
	var items [][2]string
	for _, key := range inspectorEXIFKeys {
		if value := inspectorEXIFValue(a.Texts, values, key); value != "" {
			items = append(items, [2]string{a.T("exif_" + key), value})
		}
	}
	return items
}
