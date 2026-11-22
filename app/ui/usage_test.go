package ui

import (
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

// 在「最近使用」页打开一张靠后的图：写入使用时间后它马上排到最前，选中的还是它。
func TestRecentScopeReordersAfterUse(t *testing.T) {
	now := time.Now()
	a := &App{Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, Query: catalog.Query{Scope: "recent"}}
	a.applySnapshot(model.Snapshot{Assets: []model.Asset{
		{ID: "old", Name: "Old", LastUsedAt: now.Add(-2 * time.Hour)},
		{ID: "new", Name: "New", LastUsedAt: now.Add(-time.Hour)},
	}})
	if a.Visible[0].ID != "new" || a.Visible[1].ID != "old" {
		t.Fatalf("初始顺序不对：%v", a.Visible)
	}
	a.Selected.Add("old")
	a.selectedIndex = 1
	visible := a.previewVisibleIDs()
	a.applyAssetsUsed([]string{"old"}, now)
	if a.Visible[0].ID != "old" || a.selectedIndex != 0 || !a.Selected.Has("old") {
		t.Fatalf("用过的图没有排到最前或选中丢了：%v，当前 %d", a.Visible, a.selectedIndex)
	}
	if !a.assetByID["old"].LastUsedAt.Equal(now) {
		t.Fatal("素材数据里的使用时间没更新")
	}
	if a.Preview.visibleVersion != a.visibleVersion || !visible["old"] || !visible["new"] {
		t.Fatal("仅改顺序时应保留已经缓存的可见成员")
	}
}
