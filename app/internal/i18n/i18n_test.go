package i18n

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

var formatVerbs = regexp.MustCompile(`%\[[0-9]+\][sdv]|%[sdv]`)

func TestCatalogLanguage(t *testing.T) {
	for _, tt := range []struct {
		name, language, system, want string
	}{
		{"explicit Chinese", "zh-CN", "en-US", "新建图库"},
		{"explicit English", "en", "zh-CN", "Create Library"},
		{"explicit Japanese", "ja", "zh-CN", "ライブラリを作成"},
		{"Japanese region", "JA-jp", "en-US", "ライブラリを作成"},
		{"system Chinese", "system", "zh_CN.UTF-8", "新建图库"},
		{"system English", "system", "en-US", "Create Library"},
		{"system Japanese", "system", "ja_JP.UTF-8", "ライブラリを作成"},
		{"implicit system", "", "zh-Hans", "新建图库"},
		{"other language", "system", "fr-FR", "Create Library"},
		{"unsupported explicit", "de-DE", "ja-JP", "Create Library"},
		{"not a Chinese tag", "zhfake", "ja-JP", "Create Library"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Catalog{Language: tt.language, SystemLanguage: tt.system}).T("new_library"); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCatalogEnvironmentFallback(t *testing.T) {
	for _, variable := range []string{"LC_ALL", "LC_MESSAGES", "LANGUAGE", "LANG"} {
		t.Setenv(variable, "")
	}
	t.Setenv("LANG", "zh_CN.UTF-8")
	if got := (Catalog{Language: "system"}).T("open_library"); got != "打开图库" {
		t.Fatalf("environment fallback: %q", got)
	}
	t.Setenv("LC_ALL", "en_US.UTF-8")
	if got := (Catalog{Language: "system"}).T("open_library"); got != "Open Library" {
		t.Fatalf("LC_ALL precedence: %q", got)
	}
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	if got := (Catalog{Language: "system"}).T("open_library"); got != "ライブラリを開く" {
		t.Fatalf("Japanese environment fallback: %q", got)
	}
}

func TestExportQuantityCopy(t *testing.T) {
	for _, tc := range []struct {
		language string
		count    int
		want     string
	}{
		{"en", 1, "Export images (1)"},
		{"en", 3, "Export images (3)"},
		{"zh-CN", 1, "导出 1 个素材"},
		{"zh-CN", 3, "导出 3 个素材"},
		{"ja", 1, "1枚の画像を書き出す"},
		{"ja", 3, "3枚の画像を書き出す"},
	} {
		if got := (Catalog{Language: tc.language}).T("export_count", tc.count); got != tc.want {
			t.Errorf("%s, count %d: got %q, want %q", tc.language, tc.count, got, tc.want)
		}
	}
}

func TestSearchResultQuantityCopy(t *testing.T) {
	for _, tc := range []struct {
		language, count, want string
	}{
		{"en", "1", "Results: 1"},
		{"en", "3", "Results: 3"},
		{"en", "12,000", "Results: 12,000"},
		{"zh-CN", "1", "1 个结果"},
		{"zh-CN", "3", "3 个结果"},
		{"zh-CN", "12,000", "12,000 个结果"},
		{"ja", "1", "検索結果：1件"},
		{"ja", "3", "検索結果：3件"},
		{"ja", "12,000", "検索結果：12,000件"},
	} {
		if got := (Catalog{Language: tc.language}).T("result_count", tc.count); got != tc.want {
			t.Errorf("%s, count %s: got %q, want %q", tc.language, tc.count, got, tc.want)
		}
	}
}

func TestMessagesHaveThreeLanguagesAndMatchingArguments(t *testing.T) {
	for key, pair := range messages {
		for i, value := range pair {
			if strings.TrimSpace(value) == "" {
				t.Errorf("%s has no translation for language %d", key, i)
			}
		}
		want := formatVerbs.FindAllString(pair[0], -1)
		for i, value := range pair[1:] {
			if got := formatVerbs.FindAllString(value, -1); !slices.Equal(want, got) {
				t.Errorf("%s language %d argument mismatch: %v / %v", key, i+1, want, got)
			}
		}
	}
}

func TestCatalogLocaleMatchesNativeControls(t *testing.T) {
	for _, tc := range []struct{ language, system, want string }{
		{"zh-CN", "en-US", "zh-Hans"},
		{"zh-Hans", "en-US", "zh-Hans"},
		{"ja-JP", "en-US", "ja"},
		{"system", "ja_JP.UTF-8", "ja"},
		{"system", "en-US", "en"},
		{"system", "fr-FR", "en"},
		{"en", "ja-JP", "en"},
	} {
		if got := (Catalog{Language: tc.language, SystemLanguage: tc.system}).Locale(); got != tc.want {
			t.Errorf("language %q, system %q: locale %q, want %q", tc.language, tc.system, got, tc.want)
		}
	}
}

func TestFormatAndMissingKeyAreVisible(t *testing.T) {
	c := Catalog{Language: "zh-CN"}
	if got := c.T("copied_color", "#A1B2C3"); got != "已复制 #A1B2C3" {
		t.Fatalf("format color: %q", got)
	}
	if got := c.T("import_complete", 32, 5); got != "已导入 32 个素材，跳过 5 个重复" {
		t.Fatalf("format progress: %q", got)
	}
	if got := c.T("missing_test_key"); got != "⟦missing_test_key⟧" {
		t.Fatalf("missing key should be visible, got %q", got)
	}
}

func TestMessageKeysExist(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	appDir := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	paths := []string{filepath.Join(appDir, "main.go")}
	for _, dir := range []string{"ui", "internal"} {
		if err := filepath.WalkDir(filepath.Join(appDir, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				paths = append(paths, path)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range paths {
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			t.Error(err)
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			keyAt := 0
			forwardedKey := false
			selector, translated := call.Fun.(*ast.SelectorExpr)
			if translated {
				switch selector.Sel.Name {
				case "T", "NewError":
				case "WrapError":
					keyAt = 1
				case "sidebarLibraryAction", "inspectorDetail":
					keyAt, forwardedKey = 1, true
				case "sidebarScope":
					keyAt, forwardedKey = 2, true
				default:
					translated = false
				}
			}
			helper, menuItem := call.Fun.(*ast.Ident)
			menuItem = menuItem && helper.Name == "item" && filepath.Base(path) == "main.go"
			if !translated && !menuItem {
				return true
			}
			if len(call.Args) <= keyAt {
				return true
			}
			literal, ok := call.Args[keyAt].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			key, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Error(err)
				return true
			}
			pair, ok := messages[key]
			if !ok || pair[0] == "" || pair[1] == "" || pair[2] == "" {
				t.Errorf("%s: message %q must have Chinese, English and Japanese translations", set.Position(literal.Pos()), key)
				return true
			}
			if menuItem || forwardedKey {
				return true
			}
			verbs := formatVerbs.FindAllString(pair[0], -1)
			argumentAt := keyAt + 1
			if len(call.Args)-argumentAt != len(verbs) {
				t.Errorf("%s: message %q needs %d arguments, got %d", set.Position(literal.Pos()), key, len(verbs), len(call.Args)-argumentAt)
			}
			for i, verb := range verbs {
				if i+argumentAt >= len(call.Args) {
					break
				}
				argument, ok := call.Args[i+argumentAt].(*ast.BasicLit)
				if !ok {
					continue
				}
				if verb == "%s" && argument.Kind != token.STRING || verb == "%d" && argument.Kind != token.INT {
					t.Errorf("%s: message %q has %s argument with incompatible literal", set.Position(argument.Pos()), key, verb)
				}
			}
			return true
		})
	}
}

func TestMetadataAndColorKeysExist(t *testing.T) {
	for _, family := range strings.Fields("black white gray beige brown olive red coral orange amber yellow lime green mint teal cyan sky blue indigo violet purple magenta pink rose") {
		if _, ok := messages["color_"+family]; !ok {
			t.Errorf("missing color family %s", family)
		}
	}
	for _, field := range strings.Fields("cameraMake cameraModel creator lensModel exposureTime focalLength fNumber isoSpeedRatings flashFired exposureProgram meteringMode whiteBalance dpiWidth dpiHeight pixelWidth pixelHeight colorModel profileName contentCreatedAt fileCreatedAt fileModifiedAt") {
		if _, ok := messages["exif_"+field]; !ok {
			t.Errorf("missing EXIF label %s", field)
		}
	}
}

func TestImportStageKeysMatchProducers(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	appDir := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	stages := map[string]bool{}
	for _, dir := range []string{"internal/assets", "ui"} {
		if err := filepath.WalkDir(filepath.Join(appDir, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				field, ok := node.(*ast.KeyValueExpr)
				if !ok {
					return true
				}
				name, ok := field.Key.(*ast.Ident)
				if !ok || name.Name != "Stage" {
					return true
				}
				value, ok := field.Value.(*ast.BasicLit)
				if !ok || value.Kind != token.STRING {
					return true
				}
				stage, err := strconv.Unquote(value.Value)
				if err != nil {
					t.Error(err)
					return true
				}
				stages[stage] = true
				return true
			})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if len(stages) == 0 {
		t.Fatal("no import progress stages found")
	}
	for stage := range stages {
		if pair, ok := messages["import_stage_"+stage]; !ok || pair[0] == "" || pair[1] == "" || pair[2] == "" {
			t.Errorf("import stage %q needs Chinese, English and Japanese translations", stage)
		}
	}
}
