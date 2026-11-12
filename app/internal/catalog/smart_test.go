package catalog

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestRecentUsageWindowAndOrdering(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	index := New(model.Snapshot{Assets: []model.Asset{
		{ID: "old", Name: "A", LastUsedAt: now.AddDate(0, 0, -31)},
		{ID: "boundary", Name: "B", LastUsedAt: now.AddDate(0, 0, -30)},
		{ID: "latest", Name: "Z", LastUsedAt: now},
		{ID: "yesterday", Name: "C", LastUsedAt: now.AddDate(0, 0, -1)},
		{ID: "deleted", LastUsedAt: now, Trashed: true},
		{ID: "unused", Name: "Unused"},
		{ID: "future", LastUsedAt: now.Add(time.Second)},
	}})
	for _, q := range []Query{{Scope: "recent", Now: now}, {Scope: "recent", Now: now, Sort: "name"}, {Scope: "recent", Now: now, Sort: "size", Descending: true}} {
		if got := assetIDs(index.Query(q)); !slices.Equal(got, []string{"latest", "yesterday", "boundary"}) {
			t.Fatal("最近使用未按时间或窗口筛选", got)
		}
	}
	if got := assetIDs(index.Query(Query{Scope: "recent", Now: now, Search: "Z"})); !slices.Equal(got, []string{"latest"}) {
		t.Fatal("最近使用未保留搜索", got)
	}
}

func TestSmartFolderEveryCondition(t *testing.T) {
	created := time.Date(2026, 11, 12, 16, 0, 0, 0, time.Local)
	asset := model.Asset{ID: "a", Name: "Blue SKY", Note: "设计参考\n第二行", Extension: "PNG", Width: 1200, Height: 800, Size: 4096, CreatedAt: created, TagIDs: []string{"tag-id"}, Colors: []model.Color{{Family: "blue", Hex: "#123ABC"}}}
	tests := []struct {
		name string
		rule model.SmartRule
		want bool
	}{
		{"名称包含", model.SmartRule{Field: "name", Operator: "contains", Value: "sky"}, true},
		{"名称相等", model.SmartRule{Field: "name", Operator: "equals", Value: "blue sky"}, true},
		{"备注包含", model.SmartRule{Field: "note", Operator: "contains", Value: "第二行"}, true},
		{"扩展名", model.SmartRule{Field: "extension", Operator: "equals", Value: ".png"}, true},
		{"标签ID", model.SmartRule{Field: "tag", Operator: "equals", Value: "tag-id"}, true},
		{"标签缺失", model.SmartRule{Field: "tag", Operator: "equals", Value: "absent"}, false},
		{"颜色族", model.SmartRule{Field: "color", Operator: "equals", Value: "BLUE"}, true},
		{"颜色值", model.SmartRule{Field: "color", Operator: "equals", Value: "#123abc"}, true},
		{"宽度相等", model.SmartRule{Field: "width", Operator: "eq", Value: "1200"}, true},
		{"宽度至少", model.SmartRule{Field: "width", Operator: "gte", Value: "1200"}, true},
		{"高度至多", model.SmartRule{Field: "height", Operator: "lte", Value: "799"}, false},
		{"文件大小", model.SmartRule{Field: "size", Operator: "gte", Value: "4096"}, true},
		{"添加日期起", model.SmartRule{Field: "added", Operator: "after", Value: "2026-11-12"}, true},
		{"添加日期止包含当日", model.SmartRule{Field: "added", Operator: "before", Value: "2026-11-12"}, true},
		{"添加日期止排除次日", model.SmartRule{Field: "added", Operator: "before", Value: "2026-11-11"}, false},
		{"添加时刻", model.SmartRule{Field: "added", Operator: "after", Value: created.Add(time.Hour).Format(time.RFC3339)}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			folder := model.SmartFolder{Match: "all", Rules: []model.SmartRule{test.rule}}
			if err := ValidateSmartFolder(folder); err != nil {
				t.Fatal(err)
			}
			if got := MatchesSmart(asset, folder); got != test.want {
				t.Fatal("条件结果不符", got, test.want)
			}
		})
	}
	asset.Rotation = 90
	if !MatchesSmart(asset, model.SmartFolder{Match: "all", Rules: []model.SmartRule{{Field: "width", Operator: "eq", Value: "800"}, {Field: "height", Operator: "eq", Value: "1200"}}}) {
		t.Fatal("尺寸未跟随图库旋转")
	}
	asset.Trashed = true
	if MatchesSmart(asset, model.SmartFolder{Match: "all", Rules: []model.SmartRule{{Field: "name", Operator: "contains", Value: "sky"}}}) {
		t.Fatal("智能文件夹包含已删除素材")
	}
}

func TestSmartAddedDateUsesLocalDayBoundaries(t *testing.T) {
	previousLocation := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	t.Cleanup(func() { time.Local = previousLocation })
	dayStart := time.Date(2026, 11, 12, 0, 0, 0, 0, time.Local)
	tests := []struct {
		name, operator, value string
		created               time.Time
		want                  bool
	}{
		{"从当天包含零点", "after", "2026-11-12", dayStart, true},
		{"从当天包含凌晨一点", "after", "2026-11-12", dayStart.Add(time.Hour), true},
		{"从当天排除前一天最后一刻", "after", "2026-11-12", dayStart.Add(-time.Nanosecond), false},
		{"到前一天排除当天零点", "before", "2026-11-11", dayStart, false},
		{"到前一天排除当天凌晨一点", "before", "2026-11-11", dayStart.Add(time.Hour), false},
		{"到前一天包含其最后一刻", "before", "2026-11-11", dayStart.Add(-time.Nanosecond), true},
		{"到当天包含零点", "before", "2026-11-12", dayStart, true},
		{"到当天包含最后一刻", "before", "2026-11-12", dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond), true},
		{"到当天排除次日零点", "before", "2026-11-12", dayStart.AddDate(0, 0, 1), false},
		{"UTC时间戳仍按UTC", "after", "2026-11-12T00:00:00Z", dayStart.Add(time.Hour), false},
		{"带偏移时间戳包含指定时刻", "before", "2026-11-11T17:00:00Z", dayStart.Add(time.Hour), true},
		{"带偏移时间戳排除下一瞬间", "before", "2026-11-12T01:00:00+08:00", dayStart.Add(time.Hour + time.Nanosecond), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			folder := model.SmartFolder{Match: "all", Rules: []model.SmartRule{{Field: "added", Operator: test.operator, Value: test.value}}}
			if got := MatchesSmart(model.Asset{CreatedAt: test.created}, folder); got != test.want {
				t.Fatalf("本地添加时间 %s，条件 %s %s：得到 %v，期望 %v", test.created.Format(time.RFC3339Nano), test.operator, test.value, got, test.want)
			}
		})
	}
	index := New(model.Snapshot{
		Assets:       []model.Asset{{ID: "previous", CreatedAt: dayStart.Add(-time.Nanosecond)}, {ID: "midnight", CreatedAt: dayStart}, {ID: "morning", CreatedAt: dayStart.Add(time.Hour)}},
		SmartFolders: []model.SmartFolder{{ID: "from-today", Match: "all", Rules: []model.SmartRule{{Field: "added", Operator: "after", Value: "2026-11-12"}}}, {ID: "through-yesterday", Match: "all", Rules: []model.SmartRule{{Field: "added", Operator: "before", Value: "2026-11-11"}}}},
	})
	if got := assetIDs(index.Query(Query{Scope: "smart", SmartFolderID: "from-today"})); !slices.Equal(got, []string{"morning", "midnight"}) {
		t.Fatal("从当天查询错过本地凌晨素材", got)
	}
	if got := assetIDs(index.Query(Query{Scope: "smart", SmartFolderID: "through-yesterday"})); !slices.Equal(got, []string{"previous"}) {
		t.Fatal("到前一天查询包含本地当天素材", got)
	}
}

func TestSmartAnyAllAndCombinedQuery(t *testing.T) {
	snapshot := sampleSnapshot()
	folder := model.SmartFolder{ID: "smart", Name: "暖色大图", Match: "all", Rules: []model.SmartRule{{Field: "color", Operator: "equals", Value: "red"}, {Field: "size", Operator: "gte", Value: "1024"}}}
	snapshot.SmartFolders = []model.SmartFolder{folder}
	index := New(snapshot)
	if got := assetIDs(index.Query(Query{Scope: "smart", SmartFolderID: "smart"})); !slices.Equal(got, []string{"a"}) {
		t.Fatal(got)
	}
	if got := index.Query(Query{Scope: "smart", SmartFolderID: "smart", Extensions: []string{"png"}}); len(got) != 0 {
		t.Fatal("智能范围未与工具栏过滤取交集")
	}
	if got := index.Query(Query{Scope: "smart", SmartFolderID: "missing"}); len(got) != 0 {
		t.Fatal("未知智能文件夹返回了全部素材")
	}
	folder.Match = "any"
	folder.Rules[0].Value = "green"
	snapshot.SmartFolders[0] = folder
	if got := assetIDs(New(snapshot).Query(Query{Scope: "smart", SmartFolderID: "smart"})); !slices.Equal(got, []string{"b", "a"}) {
		t.Fatal("任一条件结果错误", got)
	}
}

func TestInvalidSmartRulesFailClosed(t *testing.T) {
	invalid := []model.SmartRule{
		{Field: "unknown", Operator: "equals", Value: "x"},
		{Field: "size", Operator: "gte", Value: "NaN"},
		{Field: "size", Operator: "gte", Value: "-1"},
		{Field: "width", Operator: "gte", Value: "999999999999999999999999999999"},
		{Field: "name", Operator: "drop-table", Value: "x"},
		{Field: "name", Operator: "contains", Value: " "},
		{Field: "extension", Operator: "contains", Value: "png"},
		{Field: "added", Operator: "before", Value: "2026-99-99"},
	}
	for _, rule := range invalid {
		folder := model.SmartFolder{ID: "invalid", Match: "all", Rules: []model.SmartRule{rule}}
		if err := ValidateSmartFolder(folder); err == nil {
			t.Fatal("无效条件被接受", rule)
		}
		if MatchesSmart(model.Asset{Name: "x"}, folder) {
			t.Fatal("无效条件匹配成功", rule)
		}
		index := New(model.Snapshot{Assets: []model.Asset{{ID: "a"}}, SmartFolders: []model.SmartFolder{folder}})
		if len(index.Query(Query{Scope: "smart", SmartFolderID: folder.ID})) != 0 {
			t.Fatal("无效文件夹放行素材")
		}
	}
	for _, folder := range []model.SmartFolder{{Match: "all"}, {Match: "invalid", Rules: []model.SmartRule{{Field: "name", Operator: "contains", Value: "x"}}}} {
		if ValidateSmartFolder(folder) == nil {
			t.Fatal("无效组合被接受")
		}
	}
}

func TestSmartSnapshotRulesAreIndependent(t *testing.T) {
	snapshot := model.Snapshot{SmartFolders: []model.SmartFolder{{ID: "s", Match: "all", Rules: []model.SmartRule{{Field: "name", Operator: "contains", Value: "old"}}}}, QuickAccess: []model.QuickAccess{{ID: "q", Kind: "smart", TargetID: "s"}}}
	index := New(snapshot)
	original := index.Snapshot()
	snapshot.SmartFolders[0].Rules[0].Value = "mutated"
	snapshot.QuickAccess[0].TargetID = "mutated"
	if !reflect.DeepEqual(index.Snapshot(), original) {
		t.Fatal("索引共享了输入规则")
	}
	output := index.Snapshot()
	output.SmartFolders[0].Rules[0].Value = "output"
	output.QuickAccess[0].TargetID = "output"
	if !reflect.DeepEqual(index.Snapshot(), original) {
		t.Fatal("调用方能修改索引规则")
	}
}

func BenchmarkSmartFolder10K(b *testing.B) {
	snapshot := benchmarkSnapshot(10000)
	snapshot.SmartFolders = []model.SmartFolder{{ID: "smart", Match: "all", Rules: []model.SmartRule{{Field: "size", Operator: "gte", Value: "10000"}, {Field: "color", Operator: "equals", Value: "red"}}}}
	index := New(snapshot)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkResult = index.Query(Query{Scope: "smart", SmartFolderID: "smart"})
	}
}
