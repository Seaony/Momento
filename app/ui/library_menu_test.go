package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/egoist/mygo/ui"
)

// libraryMenuTestApp 准备三个最近图库，当前图库在中间。
func libraryMenuTestApp(t *testing.T) *App {
	t.Helper()
	a := sidebarTestApp()
	dir := t.TempDir()
	a.Library.Root = filepath.Join(dir, "Design.momento")
	a.Recent = []library.Recent{
		{Path: filepath.Join(dir, "Reference.momento"), DisplayName: "Reference"},
		{Path: a.Library.Root, DisplayName: "Design Library"},
		{Path: filepath.Join(dir, "Archive.momento"), DisplayName: "Archive"},
	}
	return a
}

func openLibraryMenu(t *testing.T, a *App) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(a.sidebar, 400, 800)
	if err := tt.Click(a.T("library_menu")); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	return tt
}

func TestLibraryTriggerHidesImageCount(t *testing.T) {
	a := sidebarTestApp()
	tt := ui.NewTester(a.sidebar, 236, 640)
	if !tt.HasText("Design Library") {
		t.Fatal("切换器没有显示图库名称")
	}
	// 切换器下方不再有「N 张图片 / N images / 画像 N 枚」这类数量行。
	for _, item := range tt.AccessAll() {
		if strings.Contains(item.Name, "张图片") || strings.Contains(item.Name, "images") || strings.Contains(item.Name, "枚") {
			t.Fatalf("切换器不应再显示图片数量：%s", item.Name)
		}
	}
	trigger, ok := tt.Find(a.T("library_menu"))
	if !ok || trigger.H != sidebarLibraryHeight {
		t.Fatalf("切换器应为紧凑高度：%+v", trigger)
	}
}

func TestLibraryMenuListsRecentLibrariesWithCurrentCheck(t *testing.T) {
	a := libraryMenuTestApp(t)
	tt := openLibraryMenu(t, a)
	for _, recent := range a.Recent {
		info, ok := tt.AccessByRole(ui.RoleRadio, recent.DisplayName)
		if !ok {
			t.Fatalf("下拉缺少图库行：%s", recent.DisplayName)
		}
		if current := recent.Path == a.Library.Root; (info.Selected || info.Checked) != current {
			t.Fatalf("%s 的当前状态应为 %v", recent.DisplayName, current)
		}
	}
	for _, key := range []string{"rename_library", "clear_library_cache", "delete_library"} {
		if tt.HasText(a.T(key)) {
			t.Fatalf("%s 应移到当前图库的更多菜单里", key)
		}
	}
	// 点当前图库只关菜单，不重新打开。
	if err := tt.Click("Design Library"); err != nil {
		t.Fatal(err)
	}
	if a.Sidebar.LibraryOpen {
		t.Fatal("点当前图库后菜单应关闭")
	}
}

// moreMenuItems 打开某一行的「…」菜单，返回出现的条目。
func moreMenuItems(t *testing.T, a *App, path string, keys []string) []string {
	t.Helper()
	tt := openLibraryMenu(t, a)
	open := true
	a.Sidebar.LibraryMore = map[string]*bool{path: &open}
	tt.Frame()
	tt.Advance(time.Second)
	var shown []string
	for _, key := range keys {
		if tt.HasText(a.T(key)) {
			shown = append(shown, key)
		}
	}
	return shown
}

func TestLibraryMoreMenuDiffersForCurrentLibrary(t *testing.T) {
	keys := []string{"rename_library", "reveal_library", "clear_library_cache", "delete_library", "remove_recent_library"}
	a := libraryMenuTestApp(t)
	current := moreMenuItems(t, a, a.Library.Root, keys)
	if want := []string{"rename_library", "reveal_library", "clear_library_cache", "delete_library"}; !slices.Equal(current, want) {
		t.Fatalf("当前图库菜单项 %v，应为 %v", current, want)
	}
	a = libraryMenuTestApp(t)
	other := moreMenuItems(t, a, a.Recent[0].Path, keys)
	if want := []string{"reveal_library", "remove_recent_library"}; !slices.Equal(other, want) {
		t.Fatalf("其他图库菜单项 %v，应为 %v", other, want)
	}
}

func TestLibraryMoreButtonOpensMenuWithoutSwitching(t *testing.T) {
	a := libraryMenuTestApp(t)
	tt := openLibraryMenu(t, a)
	if err := tt.Click(a.T("library_more")); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if !a.Sidebar.LibraryOpen || !tt.HasText(a.T("remove_recent_library")) {
		t.Fatal("点「…」应打开第一行的更多菜单，且不切换图库、不关闭下拉")
	}
}

func TestLibraryRemoveRecentUpdatesStoreAndList(t *testing.T) {
	a := libraryMenuTestApp(t)
	a.recents = library.NewRecentStore(t.TempDir())
	for _, recent := range a.Recent {
		if err := a.recents.Add(recent); err != nil {
			t.Fatal(err)
		}
	}
	target := a.Recent[0].Path
	tt := openLibraryMenu(t, a)
	open := true
	a.Sidebar.LibraryMore = map[string]*bool{target: &open}
	tt.Frame()
	tt.Advance(time.Second)
	if err := tt.Click(a.T("remove_recent_library")); err != nil {
		t.Fatal(err)
	}
	if a.Sidebar.LibraryOpen {
		t.Fatal("移除后下拉应关闭")
	}
	// 移除在后台写入；测试里没有主循环，所以直接核对存储，再把结果交给界面回调。
	var stored []library.Recent
	deadline := time.Now().Add(2 * time.Second)
	for {
		var err error
		stored, err = a.recents.List()
		if err != nil {
			t.Fatal(err)
		}
		if !containsRecent(stored, target) || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if containsRecent(stored, target) || len(stored) != 2 {
		t.Fatalf("存储里仍有被移除的图库：%+v", stored)
	}
	a.recentRemoved(stored, nil)
	if containsRecent(a.Recent, target) || len(a.Recent) != 2 {
		t.Fatalf("界面列表没有更新：%+v", a.Recent)
	}
}

func TestLibraryRemoveRecentFailureIsShown(t *testing.T) {
	a := libraryMenuTestApp(t)
	before := len(a.Recent)
	a.recentRemoved(nil, os.ErrPermission)
	if a.Error == "" || len(a.Recent) != before {
		t.Fatal("移除失败时应显示错误并保留原列表")
	}
}

func containsRecent(recent []library.Recent, path string) bool {
	for _, r := range recent {
		if r.Path == path {
			return true
		}
	}
	return false
}

func TestLibraryMenuScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_LIBRARY_SHOTS") != "1" {
		t.Skip("设置 MOMENTO_LIBRARY_SHOTS=1 生成图库切换下拉截图")
	}
	dir := "/tmp/momento-polish/shots4/library"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, appearance := range []string{"light", "dark"} {
		for _, state := range []string{"menu", "more"} {
			t.Run(appearance+"/"+state, func(t *testing.T) {
				a := fixture.app(t, "small", "zh-CN", appearance)
				a.InspectorVisible = false
				a.Library.Root = filepath.Join(home, "Pictures", "Momento", "设计参考.momento")
				// 夹具按样例库路径预装了缩略图，换成展示用路径后缓存键跟着换。
				for key, element := range a.Thumbnails.entries {
					entry := element.Value.(thumbnailEntry)
					delete(a.Thumbnails.entries, key)
					key.root = filepath.Clean(a.Library.Root)
					entry.key = key
					element.Value = entry
					a.Thumbnails.entries[key] = element
				}
				a.Library.Manifest.DisplayName = "设计参考"
				a.Recent = []library.Recent{
					{Path: filepath.Join(home, "Pictures", "Momento", "摄影灵感.momento"), DisplayName: "摄影灵感"},
					{Path: a.Library.Root, DisplayName: "设计参考"},
					{Path: filepath.Join(home, "Documents", "Clients", "2026 Spring Campaign", "Brand Moodboard Archive.momento"), DisplayName: "Brand Moodboard Archive"},
				}
				a.Sidebar.LibraryOpen = true
				if state == "more" {
					open := true
					a.Sidebar.LibraryMore = map[string]*bool{a.Library.Root: &open}
				}
				tt := ui.NewTester(a.View, 1100, 800)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				tt.Frame()
				tt.Advance(time.Second)
				writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+appearance+".png"), tt)
			})
		}
	}
}
