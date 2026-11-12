package i18n

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
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
