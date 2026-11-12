package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Catalog struct{ Language string }

var messages = map[string][2]string{
	"welcome":      {"没有打开的图库", "No library open"},
	"welcome_hint": {"把灵感收进自己的素材库", "A home for your visual references"},
	"new_library":  {"新建图库", "Create Library"},
	"open_library": {"打开图库", "Open Library"},
}

func (c Catalog) T(key string, args ...any) string {
	pair, ok := messages[key]
	if !ok {
		return key
	}
	i := 0
	lang := c.Language
	if lang == "" || lang == "system" {
		lang = os.Getenv("LANG")
	}
	if !strings.HasPrefix(strings.ToLower(lang), "zh") {
		i = 1
	}
	if len(args) == 0 {
		return pair[i]
	}
	return fmt.Sprintf(pair[i], args...)
}
