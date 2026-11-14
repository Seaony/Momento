package ui

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestLibraryOpenOptions(t *testing.T) {
	parent := new(mygo.Window)
	options := libraryOpenOptions(parent, "Open Library")
	if options.Parent != parent || options.Title != "Open Library" {
		t.Fatal("没有保留父窗口和对话框标题")
	}
	if options.Multiple || !options.Directory || len(options.Filters) != 0 {
		t.Fatal("图库按单个目录选择，不用普通文件过滤")
	}
	if options.TreatPackagesAsDirectories != (runtime.GOOS == "darwin") {
		t.Fatal("macOS需要允许选择目录包自身")
	}

}

func TestSidebarLibraryManagementMenu(t *testing.T) {
	app := libraryMenuTestApp(t)
	tt := ui.NewTester(app.sidebar, 400, 640)
	if err := tt.Click(app.T("library_menu")); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"new_library", "open_library", "import_library", "export_library"} {
		if !tt.HasText(app.T(key)) {
			t.Errorf("菜单缺少 %s", key)
		}
	}
	// 改名、清缓存、删除都在当前图库的「…」菜单里。
	open := true
	app.Sidebar.LibraryMore = map[string]*bool{app.Library.Root: &open}
	tt.Frame()
	for _, key := range []string{"rename_library", "reveal_library", "clear_library_cache", "delete_library"} {
		if !tt.HasText(app.T(key)) {
			t.Errorf("当前图库的更多菜单缺少 %s", key)
		}
	}
}

func TestSidebarLibraryManagementDisabledWhileBusy(t *testing.T) {
	app := libraryMenuTestApp(t)
	app.Busy = true
	tt := ui.NewTester(app.sidebar, 400, 640)
	if err := tt.Click(app.T("library_menu")); err != nil {
		t.Fatal(err)
	}
	open := true
	app.Sidebar.LibraryMore = map[string]*bool{app.Library.Root: &open}
	tt.Frame()
	for _, key := range []string{"import_library", "export_library", "rename_library", "clear_library_cache", "delete_library"} {
		access, ok := tt.Access(app.T(key))
		if !ok || !access.Disabled {
			t.Errorf("%s 未在忙碌时禁用", key)
		}
	}
}

func TestSidebarLibraryMenuScrollsWithTenRecentLibraries(t *testing.T) {
	app := sidebarTestApp()
	for i := range 10 {
		app.Recent = append(app.Recent, library.Recent{Path: fmt.Sprintf("/tmp/%d.momento", i), DisplayName: fmt.Sprintf("Library %d", i)})
	}
	tt := ui.NewTester(app.sidebar, 236, 640)
	if err := tt.Click(app.T("library_menu")); err != nil {
		t.Fatal(err)
	}
	tt.Scroll(100, 500, 0, 650)
	if err := tt.Click(app.T("new_library")); err != nil {
		t.Fatal(err)
	}
	if !app.NewLibraryOpen {
		t.Fatal("滚动后的图库操作未响应")
	}
}

func TestLibraryExportPath(t *testing.T) {
	for input, want := range map[string]string{"/tmp/Design": "/tmp/Design.momento", "/tmp/Design.momento": "/tmp/Design.momento", "/tmp/Design.MOMENTO": "/tmp/Design.MOMENTO"} {
		if got := libraryExportPath(input); got != want {
			t.Errorf("%s: %s", input, got)
		}
	}
}
