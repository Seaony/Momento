package ui

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Seaony/Momento/app/internal/browser"
	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestRemoteBatchFeedbackAfterAllItems(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.Notify = false
	item := func(i int) browser.Request { return browser.Request{BatchID: "b1", Index: i, Total: 3} }
	// 乱序到达：编号最大的先做完也不能提前汇总。
	a.remoteFeedback(item(2), model.ImportResult{Imported: 1, AssetIDs: []string{"a"}}, nil, a.generation)
	a.remoteFeedback(item(0), model.ImportResult{}, i18n.NewError("error_download_status", 403), a.generation)
	if a.Notice != "" || a.Error != "" {
		t.Fatalf("没全部做完就提示了：%q %q", a.Notice, a.Error)
	}
	a.remoteFeedback(item(1), model.ImportResult{Skipped: 1, AssetIDs: []string{"c"}}, nil, a.generation)
	if a.Notice != a.T("import_complete_failed", 1, 1, 1) {
		t.Fatalf("汇总提示是 %q", a.Notice)
	}
	if len(a.remoteBatches) != 0 {
		t.Fatal("这一批结束后记录没清掉")
	}
}

func TestRemoteBatchCanceledStaysQuiet(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.Notify = false
	for i := range 2 {
		a.remoteFeedback(browser.Request{BatchID: "b2", Index: i, Total: 2}, model.ImportResult{}, i18n.NewError("import_canceled"), a.generation)
	}
	if a.Notice != "" || a.Error != "" {
		t.Fatalf("取消整批后还提示了：%q %q", a.Notice, a.Error)
	}
	// 单张：扩展说不要提示就不提示，失败照样报。
	quiet := false
	a.remoteFeedback(browser.Request{PlayFeedback: &quiet}, model.ImportResult{Imported: 1}, nil, a.generation)
	if a.Notice != "" {
		t.Fatalf("不要提示时提示了 %q", a.Notice)
	}
	a.remoteFeedback(browser.Request{}, model.ImportResult{}, errors.New("boom"), a.generation)
	if a.Error == "" {
		t.Fatal("单张失败没报错")
	}
}

// 一批导到一半换了图库：这一批定在第一张写入的图库，剩下的不导进新图库，明确报错；
// 结束时只报数量，不在新图库里按旧图库的 ID 选中（新图库恰好有同内容的素材也不选）。
func TestRemoteBatchStaysWithFirstLibrary(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.Preferences.Notify, a.Preferences.SelectImported = false, true
	a.applySnapshot(model.Snapshot{Assets: []model.Asset{{ID: "old", Name: "Old"}}})
	a.generation = 1
	if a.remoteBatchLibraryChanged("b3") {
		t.Fatal("第一张不应算换了图库")
	}
	item := func(i int) browser.Request { return browser.Request{BatchID: "b3", Index: i, Total: 2} }
	a.remoteFeedback(item(0), model.ImportResult{Imported: 1, AssetIDs: []string{"old"}}, nil, a.generation)
	a.generation++
	if !a.remoteBatchLibraryChanged("b3") {
		t.Fatal("换了图库后，同一批剩下的应被拦下")
	}
	a.remoteFeedback(item(1), model.ImportResult{}, i18n.NewError("error_import_library_changed"), a.generation)
	if a.Notice != a.T("import_complete_failed", 1, 0, 1) {
		t.Fatalf("汇总提示是 %q", a.Notice)
	}
	if a.Selected.Len() != 0 {
		t.Fatal("不应在新图库里选中旧图库的素材")
	}
	if len(a.remoteBatches) != 0 {
		t.Fatal("这一批结束后记录没清掉")
	}
}

// 导入几张后只把这几张并进内存：已有的换成新数据，新的接在后面，标签换成最新的。
func TestMergeAssetsReplacesAndAppends(t *testing.T) {
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}}
	a.applySnapshot(model.Snapshot{Assets: []model.Asset{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}})
	a.mergeAssets(model.Snapshot{
		Assets: []model.Asset{{ID: "b", Name: "B", TagIDs: []string{"t"}}, {ID: "c", Name: "C"}},
		Tags:   []model.Tag{{ID: "t", Name: "T"}},
	})
	if len(a.Snapshot.Assets) != 3 || a.assetByID["b"].TagIDs[0] != "t" || a.assetByID["c"].Name != "C" || len(a.Snapshot.Tags) != 1 {
		t.Fatalf("合并结果不对：%+v", a.Snapshot)
	}
	if counts := a.Index.Counts(); counts.All != 3 || counts.Tags["t"] != 1 {
		t.Fatalf("索引没有跟着更新：%+v", counts)
	}
	if !reflect.DeepEqual(a.Visible, a.Index.Query(a.Query)) {
		t.Fatalf("增量可见列表与完整查询结果不同：%+v", a.Visible)
	}
}

func TestMergeAssetsUpdatesSortedFilteredVisibleList(t *testing.T) {
	a := &App{Query: catalog.Query{Sort: "name", Descending: true, Extensions: []string{"jpg"}}, selectedIndex: -1, anchor: -1}
	a.applySnapshot(model.Snapshot{Assets: []model.Asset{
		{ID: "a", Name: "Alpha", Extension: "jpg"},
		{ID: "b", Name: "Bravo", Extension: "jpg"},
		{ID: "c", Name: "Charlie", Extension: "jpg"},
	}})
	a.Selected.Add("a")
	a.Selected.Add("c")
	a.selectedIndex, a.anchor = 2, 0
	a.previewVisibleIDs()
	a.mergeAssets(model.Snapshot{Assets: []model.Asset{
		{ID: "a", Name: "Delta", Extension: "jpg"},
		{ID: "c", Name: "Charlie", Extension: "png"},
		{ID: "d", Name: "Echo", Extension: "jpg"},
	}})
	if want := a.Index.Query(a.Query); !reflect.DeepEqual(a.Visible, want) {
		t.Fatalf("增量列表 %v，完整查询 %v", a.Visible, want)
	}
	if a.selectedIndex != 1 || a.anchor != -1 || a.Selected.Len() != 1 || !a.Selected.Has("a") {
		t.Fatal("排序或筛选变化后没有保留有效选中状态")
	}
	if visible := a.previewVisibleIDs(); visible["c"] || !visible["d"] {
		t.Fatalf("预览成员缓存没有更新：%v", visible)
	}
}
