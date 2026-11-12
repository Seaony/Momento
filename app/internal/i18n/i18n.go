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
	i := 0
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
	if !strings.HasPrefix(strings.ToLower(lang), "zh") {
		i = 1
	}
	if len(args) == 0 {
		return pair[i]
	}
	return fmt.Sprintf(pair[i], args...)
}
