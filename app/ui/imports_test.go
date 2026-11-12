package ui

import (
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestAutoImportWatcherRestartsForReopenedLibrary(t *testing.T) {
	rules := []AutoImportRule{{Path: "incoming", Enabled: true}}
	state := importState{WatchCancel: func() {}, WatchLibrary: "same-id", WatchGeneration: 2, WatchRules: rules}
	if !state.watching("same-id", 2, rules) {
		t.Fatal("同一已运行配置被误判为要重启")
	}
	if state.watching("same-id", 3, rules) {
		t.Fatal("重开同ID图库后仍复用捕获旧代际的监视器")
	}
	if state.watching("other", 2, rules) {
		t.Fatal("换库后仍复用旧监视器")
	}
}
func TestExtensionPickerKeepsFolderIDsForDuplicateNames(t *testing.T) {
	for _, selected := range []string{"", "second"} {
		a := &App{Texts: i18n.Catalog{Language: "en"}, Preferences: DefaultPreferences()}
		a.Snapshot.Folders = []model.Folder{{ID: "root", Name: a.T("all_assets")}, {ID: "parent-one", Name: "One"}, {ID: "parent-two", Name: "Two"}, {ID: "first", Name: "Photos", ParentID: "parent-one"}, {ID: "second", Name: "Photos", ParentID: "parent-two"}}
		reply := make(chan importChoice, 1)
		a.Import = importState{PickerOpen: true, PickerReply: reply, PickerFolder: selected, PickerTags: map[string]bool{}}
		tt := ui.NewTester(a.extensionImportPicker, 600, 600)
		if selected != "" {
			if err := tt.Click(a.T("folders")); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Two / Photos"); err != nil {
				t.Fatal(err)
			}
		}
		if err := tt.Click(a.T("import")); err != nil {
			t.Fatal(err)
		}
		choice := <-reply
		if choice.FolderID != selected {
			t.Fatalf("同名目录或根选项被匹配到其他目录: %q / %q", choice.FolderID, selected)
		}
	}
}

func TestLockingCancelsAnAlreadyVisibleExtensionPicker(t *testing.T) {
	a := &App{Preferences: DefaultPreferences()}
	a.Preferences.PasswordHash = "configured"
	reply := make(chan importChoice, 1)
	a.Import = importState{PickerOpen: true, PickerReply: reply}
	a.LockNow()
	if !a.IsLocked() || a.Import.PickerOpen || a.Import.PickerReply != nil {
		t.Fatal("锁定后仍保留分类浮层")
	}
	select {
	case choice := <-reply:
		if !choice.Canceled {
			t.Fatal("已显示的浮层未经确认就导入")
		}
	default:
		t.Fatal("扩展请求仍挂起等待选择")
	}
}
