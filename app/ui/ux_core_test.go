package ui

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

// 导入进行中点收藏：心形马上变红，导入结束重读快照时也不闪回去；写库仍排在后面。
func TestQueuedFavoriteShowsImmediately(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	before := a.Snapshot
	before.Assets = append([]model.Asset(nil), a.Snapshot.Assets...)
	a.SetFavorite([]string{"2"}, true)
	if !a.assetByID["2"].Favorite {
		t.Fatal("排队时界面上没有先显示收藏")
	}
	if len(a.queuedRuns) != 1 || !a.queuedRuns[0].patch.optimistic {
		t.Fatalf("收藏没有排进写入：%+v", a.queuedRuns)
	}
	a.applySnapshot(before)
	if !a.assetByID["2"].Favorite {
		t.Fatal("重读快照后提前显示的收藏闪回去了")
	}
}

// 移到回收站后给「撤销」，点了排进恢复；恢复本身也有提示。
func TestTrashOffersUndo(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Assets = &assets.Service{}
	a.Busy = true
	a.RequestDelete([]string{"1", "3"})
	if a.noticeAction == nil || a.noticeAction.label != a.T("undo") || a.noticeAction.message != a.T("moved_to_trash", 2) {
		t.Fatalf("删除后没有撤销提示：%+v", a.noticeAction)
	}
	if !a.assetByID["1"].Trashed {
		t.Fatal("删除后界面上没有先移走")
	}
	a.noticeAction.action()
	if a.assetByID["1"].Trashed || a.assetByID["3"].Trashed {
		t.Fatal("撤销后素材没有回来")
	}
	if last := a.queuedRuns[len(a.queuedRuns)-1]; last.label != "restore" {
		t.Fatalf("撤销没有排进恢复：%s", last.label)
	}
	a.Restore([]string{"1"})
	if a.Notice != a.T("restored_assets", 1) {
		t.Fatalf("恢复后的提示是 %q", a.Notice)
	}
}

// F 收藏：鼠标停在选区外就只收藏那一张，否则收藏全部选中的。
func TestFavoriteKeyTargets(t *testing.T) {
	a := assetViewFixture()
	a.Selected.Add("0")
	a.Selected.Add("1")
	a.hovered = "4"
	if got := a.favoriteKeyTargets(); len(got) != 1 || got[0] != "4" {
		t.Fatalf("悬停在选区外：%v", got)
	}
	a.hovered = "1"
	if got := a.favoriteKeyTargets(); len(got) != 2 {
		t.Fatalf("悬停在选区里：%v", got)
	}
	a.hovered = ""
	if got := a.favoriteKeyTargets(); len(got) != 2 {
		t.Fatalf("没有悬停：%v", got)
	}
	if !a.FavoriteValue([]string{"0", "1"}) {
		t.Fatal("没收藏的应该是收藏")
	}
}

// 后台进来的导入（自动导入、浏览器扩展）不动选择和位置，只给「查看」；手动导入在文件夹里就留在文件夹选中新图。
func TestImportFeedbackKeepsPlace(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{Root: "/tmp/lib.momento"}
	a.Preferences = DefaultPreferences()
	a.Preferences.Notify = false
	a.Selected.Add("0")
	a.importFeedback("auto-import", model.ImportResult{Imported: 1, AssetIDs: []string{"5"}})
	if !a.Selected.Has("0") || a.Selected.Len() != 1 {
		t.Fatal("自动导入改了用户的选择")
	}
	if a.noticeAction == nil || a.noticeAction.label != a.T("view") {
		t.Fatalf("自动导入没有给「查看」：%+v", a.noticeAction)
	}
	a.noticeAction.action()
	if !a.Selected.Has("5") {
		t.Fatal("点「查看」没有选中新素材")
	}

	snapshot := a.Snapshot
	snapshot.Folders = []model.Folder{{ID: "f", Name: "Travel"}}
	snapshot.Assets[5].FolderIDs = []string{"f"}
	a.applySnapshot(snapshot)
	a.SetScope("folder", "f")
	a.importFeedback("", model.ImportResult{Imported: 1, AssetIDs: []string{"5"}})
	if a.Query.Scope != "folder" || a.Query.FolderID != "f" || !a.Selected.Has("5") {
		t.Fatalf("在文件夹里导入后被带走了：%s %s", a.Query.Scope, a.Query.FolderID)
	}
	if a.Notice != a.T("import_complete_new", 1) {
		t.Fatalf("没有重复时的提示是 %q", a.Notice)
	}
}

// 菜单项按状态灰掉：没图库时导入没用，没选中时导出没用，列表视图里缩略图大小没用，输入框里复制粘贴照常。
func TestMenuStateDisablesUselessItems(t *testing.T) {
	none := MenuState{}
	if none.MenuItemEnabled("import") || none.MenuItemEnabled("export") || none.MenuItemEnabled("view_list") {
		t.Fatal("没有图库时导入、导出、视图仍可点")
	}
	if !none.MenuItemEnabled("new_library") {
		t.Fatal("没有图库时新建图库应能点")
	}
	open := MenuState{Library: true, Grid: true}
	if open.MenuItemEnabled("export") || open.MenuItemEnabled("copy") {
		t.Fatal("没有选中时导出、复制仍可点")
	}
	if !(MenuState{TextFocus: true}).MenuItemEnabled("copy") {
		t.Fatal("输入框里的复制不能灰掉")
	}
	if (MenuState{Library: true, Browse: true}).MenuItemEnabled("thumbnail_larger") {
		t.Fatal("列表视图里缩略图大小仍可点")
	}
	if (MenuState{Library: true, Browse: true}).MenuItemEnabled("sort") {
		t.Fatal("最近使用、随机浏览里排序没作用，仍可点")
	}
	if (MenuState{Library: true, Sortable: true}).MenuItemEnabled("filter") || (MenuState{Library: true, Sortable: true}).MenuItemEnabled("view_list") {
		t.Fatal("标签管理页、空图库里筛选和视图没作用，仍可点")
	}
	if !(MenuState{Library: true, Browse: true, Sortable: true, Grid: true}).MenuItemEnabled("sort") {
		t.Fatal("正常浏览时排序应能点")
	}
	if (MenuState{Library: true, Busy: true}).MenuItemEnabled("import") {
		t.Fatal("正在导入时导入仍可点")
	}
}

// 导入进行中再拖进来的文件排在后面，不丢；没有图库时说清楚要先打开图库。
func TestDropDuringImportQueues(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	cleaned := false
	a.startPathImport([]string{"/tmp/a.png"}, "", "", func() { cleaned = true })
	if len(a.Import.queued) != 1 || a.Notice != a.T("import_queued") || cleaned {
		t.Fatalf("忙时拖入没有排队：%+v %q", a.Import.queued, a.Notice)
	}
	a.closeLibrary("test")
	if !cleaned || len(a.Import.queued) != 0 {
		t.Fatal("换图库时排队的导入没有清理")
	}
	a.startPathImport([]string{"/tmp/a.png"}, "", "", nil)
	if a.Error == "" {
		t.Fatal("没有图库时拖入没有说明原因")
	}
}

// 较早排队的写入正在写库时整库重读、写完时，都不能把后面已经画在界面上的新状态盖回去（快速收藏再取消），也不清掉选择。
func TestEarlierWriteKeepsNewerQueuedState(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	before := a.Snapshot
	before.Assets = append([]model.Asset(nil), a.Snapshot.Assets...)
	a.Selected.Add("2")
	a.SetFavorite([]string{"2"}, true)
	a.SetFavorite([]string{"2"}, false)
	// 第一笔开始写库：出队，记成正在写的那笔。
	a.runningPatch, a.queuedRuns = a.queuedRuns[0].patch, a.queuedRuns[1:]
	a.applySnapshot(before)
	if a.assetByID["2"].Favorite {
		t.Fatal("写库途中整库重读后，后面的取消收藏被盖回去了")
	}
	a.runningPatch = nil
	if a.assetByID["2"].Favorite || !a.Selected.Has("2") {
		t.Fatalf("第一笔写完后状态不对：favorite=%v selected=%v", a.assetByID["2"].Favorite, a.Selected.Has("2"))
	}
}

// 回收站里的素材还挂着标签：删标签要先确认，不然恢复后标签就没了。
func TestDeleteTagCountsTrashedAssets(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Busy = true
	a.Snapshot.Tags = []model.Tag{{ID: "t", Name: "Old"}}
	a.Snapshot.Assets[0].TagIDs = []string{"t"}
	a.Snapshot.Assets[0].Trashed = true
	a.deleteTag(model.Tag{ID: "t", Name: "Old"})
	if !a.ConfirmOpen || len(a.queuedRuns) != 0 {
		t.Fatalf("只有回收站素材用着的标签被直接删了：confirm=%v queued=%d", a.ConfirmOpen, len(a.queuedRuns))
	}
}

// 清空回收站只删确认时说的那几个；换了图库确认框就关掉。
func TestEmptyTrashConfirmScope(t *testing.T) {
	a := assetViewFixture()
	a.Library = &library.Library{}
	a.Snapshot.Assets[1].Trashed = true
	a.ConfirmEmptyTrash()
	if a.ConfirmDetail != a.T("empty_trash_count_detail", 1) {
		t.Fatalf("确认框数量：%q", a.ConfirmDetail)
	}
	a.closeLibrary("test")
	if a.ConfirmOpen || a.ConfirmAction != nil {
		t.Fatal("换图库后清空回收站的确认框还开着")
	}
}
