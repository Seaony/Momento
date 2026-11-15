package i18n

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Catalog struct {
	Language       string
	SystemLanguage string
}

func (c Catalog) T(key string, args ...any) string {
	pair, ok := messages[key]
	if !ok {
		slog.Error("缺少界面文案", "key", key)
		return "⟦" + key + "⟧"
	}
	i := 1
	switch c.Locale() {
	case "zh-Hans":
		i = 0
	case "ja":
		i = 2
	}
	if len(args) == 0 {
		return pair[i]
	}
	return fmt.Sprintf(pair[i], args...)
}

// 和原生控件使用同一组语言标记；未支持的语言显示英文。
func (c Catalog) Locale() string {
	lang := c.Language
	if lang == "" || lang == "system" {
		lang = c.SystemLanguage
		if lang == "" {
			for _, variable := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
				if lang = os.Getenv(variable); lang != "" {
					break
				}
			}
		}
	}
	lang = strings.ToLower(strings.TrimSpace(lang))
	if end := strings.IndexAny(lang, "-_.@:"); end >= 0 {
		lang = lang[:end]
	}
	switch lang {
	case "zh":
		return "zh-Hans"
	case "ja":
		return "ja"
	}
	return "en"
}

// 日期加时间，和系统「缩写日期 + 短时间」一致：2026年10月7日 15:55、Oct 7, 2026 at 3:55 PM。
func (c Catalog) DateTime(t time.Time) string {
	if c.Locale() == "en" {
		return t.Format("Jan 2, 2006 at 3:04 PM")
	}
	return t.Format("2006年1月2日 15:04")
}

// 只有日期：2026年10月7日、Oct 7, 2026。
func (c Catalog) Date(t time.Time) string {
	if c.Locale() == "en" {
		return t.Format("Jan 2, 2006")
	}
	return t.Format("2006年1月2日")
}

// 一个月内写成「3 天前」这类相对时间，更早的写具体日期。
func (c Catalog) Relative(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return c.T("relative_now")
	case d < time.Hour:
		return c.T("relative_minutes", int(d/time.Minute))
	case d < 24*time.Hour:
		return c.T("relative_hours", int(d/time.Hour))
	case d < 48*time.Hour:
		return c.T("relative_yesterday")
	case d < 30*24*time.Hour:
		return c.T("relative_days", int(d/(24*time.Hour)))
	}
	return c.Date(t)
}
