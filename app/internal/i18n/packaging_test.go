package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestMacBundleDeclaresCatalogLanguages(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "mygo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MacOS struct {
			InfoPlist struct {
				DevelopmentRegion string   `json:"CFBundleDevelopmentRegion"`
				Localizations     []string `json:"CFBundleLocalizations"`
			} `json:"infoPlist"`
		} `json:"macos"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	plist := config.MacOS.InfoPlist
	for _, language := range []string{"en", "zh-CN", "ja"} {
		locale := (Catalog{Language: language}).Locale()
		if !slices.Contains(plist.Localizations, locale) {
			t.Errorf("macOS包未声明界面语言%s（%s），系统对话框无法使用该语言", language, locale)
		}
	}
	fallback := (Catalog{Language: "fr-FR"}).Locale()
	if fallback != "en" || plist.DevelopmentRegion != fallback {
		t.Errorf("macOS包的默认语言%q与界面回落语言%q不一致，均应为en", plist.DevelopmentRegion, fallback)
	}
}
