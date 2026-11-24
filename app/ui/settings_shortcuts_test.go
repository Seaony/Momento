package ui

import "testing"

// 「空格预览」在主界面按，和主界面快捷键冲突要报出来。
func TestPreviewShortcutConflictsWithMainActions(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	if a.SetShortcut("preview", "CmdOrCtrl+E") {
		t.Fatal("空格预览改成 ⌘E 没有报和导出冲突")
	}
	if a.Settings.ShortcutErrors["preview"] != a.T("shortcut_conflict_in_use", displayShortcut("CmdOrCtrl+E"), a.T("export")) {
		t.Fatalf("冲突提示不对：%q", a.Settings.ShortcutErrors["preview"])
	}
	if !a.SetShortcut("view_list", "CmdOrCtrl+0") {
		t.Fatal("主界面和预览里的快捷键本来可以相同")
	}
}

// 预览开着时，和预览里实际快捷键相同的主界面菜单项摘掉快捷键，按键交给预览。
func TestMenuAcceleratorYieldsToPreview(t *testing.T) {
	a := settingsTestApp("zh-CN", "light")
	a.SetShortcut("view_list", "CmdOrCtrl+0")
	if a.MenuAccelerator("view_list") != "CmdOrCtrl+0" || a.MenuAccelerator("preview_fit") != "" {
		t.Fatal("预览没开时菜单快捷键不对")
	}
	a.Preview.active = &previewWindow{}
	for action, want := range map[string]string{
		"view_list": "", "preview_fit": "CmdOrCtrl+0", "view_masonry": "", "thumbnail_larger": "", "thumbnail_smaller": "", "export": "CmdOrCtrl+E",
	} {
		if got := a.MenuAccelerator(action); got != want {
			t.Fatalf("预览开着时 %s 的菜单快捷键是 %q，想要 %q", action, got, want)
		}
	}
	a.SetShortcut("view_list", "CmdOrCtrl+Shift+=")
	if a.MenuAccelerator("view_list") != "" {
		t.Fatal("预览里 ⇧⌘= 也是放大，菜单不该抢")
	}
	a.Settings.RecordingShortcut = "search"
	if a.MenuAccelerator("export") != "" || a.MenuAccelerator("preview_fit") != "" {
		t.Fatal("录快捷键时菜单仍挂着快捷键")
	}
}
