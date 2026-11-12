package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLibraryIconIsDeclaredForDocumentAndExportedType(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "mygo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		MacOS struct {
			InfoPlist struct {
				Documents []struct {
					Types []string `json:"LSItemContentTypes"`
					Icon  string   `json:"CFBundleTypeIconFile"`
				} `json:"CFBundleDocumentTypes"`
				Types []struct {
					ID   string `json:"UTTypeIdentifier"`
					Icon string `json:"UTTypeIconFile"`
				} `json:"UTExportedTypeDeclarations"`
			} `json:"infoPlist"`
		} `json:"macos"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	icons := map[string]string{}
	for _, document := range config.MacOS.InfoPlist.Documents {
		for _, id := range document.Types {
			icons[id] = document.Icon
		}
	}
	for _, declaration := range config.MacOS.InfoPlist.Types {
		if declaration.ID != "com.seaony.momento.library" {
			continue
		}
		if declaration.Icon == "" || declaration.Icon != icons[declaration.ID] {
			t.Fatal("图库导出类型与文档类型没有指向同一份图标")
		}
		icon, err := os.ReadFile(filepath.Join("darwin", declaration.Icon))
		if err != nil {
			t.Fatal(err)
		}
		if len(icon) < 8 || string(icon[:4]) != "icns" {
			t.Fatal("声明的图库图标不是有效 ICNS 资源")
		}
		return
	}
	t.Fatal("没有导出图库类型")
}
