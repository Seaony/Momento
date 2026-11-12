package catalog

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func sampleSnapshot() model.Snapshot {
	day := func(n int) time.Time { return time.Date(2026, 1, n, 12, 0, 0, 0, time.UTC) }
	return model.Snapshot{
		Folders: []model.Folder{{ID: "parent", Name: "References"}, {ID: "child", Name: "Nature", ParentID: "parent"}, {ID: "empty", Name: "Empty"}},
		Tags:    []model.Tag{{ID: "warm", Name: "Warm"}, {ID: "landscape", Name: "LANDSCAPE"}, {ID: "unused", Name: "Unused"}},
		Assets: []model.Asset{
			{ID: "a", Name: "Amber", Extension: "JPG", Size: 1024, Favorite: true, CreatedAt: day(1), TagIDs: []string{"warm"}, Colors: []model.Color{{Family: "red"}, {Family: "orange"}}},
			{ID: "b", Name: "Mountain 山水", Extension: "png", Size: 512, CreatedAt: day(2), FolderIDs: []string{"parent"}, TagIDs: []string{"landscape"}, Colors: []model.Color{{Family: "green"}}},
			{ID: "c", Name: "Moon", Extension: "webp", Size: 512, Favorite: true, CreatedAt: day(2), FolderIDs: []string{"child"}, Colors: []model.Color{{Family: "blue"}}},
			{ID: "d", Name: "Unfiled", Extension: "gif", Size: 256, CreatedAt: day(3)},
			{ID: "e", Name: "Removed", Extension: "jpg", Size: 2048, Favorite: true, Trashed: true, CreatedAt: day(4), FolderIDs: []string{"parent"}, TagIDs: []string{"warm"}, Colors: []model.Color{{Family: "red"}}},
		},
	}
}

func assetIDs(assets []model.Asset) []string {
	ids := make([]string, len(assets))
	for i, asset := range assets {
		ids[i] = asset.ID
	}
	return ids
}

func TestScopeFiltersSearch(t *testing.T) {
	index := New(sampleSnapshot())
	tests := []struct {
		name string
		q    Query
		want []string
	}{
		{"默认只显示未删除素材", Query{}, []string{"d", "c", "b", "a"}},
		{"全部", Query{Scope: "all"}, []string{"d", "c", "b", "a"}},
		{"收藏", Query{Scope: "favorites"}, []string{"c", "a"}},
		{"未分类", Query{Scope: "uncategorized"}, []string{"d", "a"}},
		{"未打标签", Query{Scope: "untagged"}, []string{"d", "c"}},
		{"回收站", Query{Scope: "trash"}, []string{"e"}},
		{"父文件夹只含直接成员", Query{Scope: "folder", FolderID: "parent"}, []string{"b"}},
		{"子文件夹", Query{Scope: "folder", FolderID: "child"}, []string{"c"}},
		{"空文件夹", Query{Scope: "folder", FolderID: "empty"}, []string{}},
		{"颜色同组任意匹配", Query{ColorFamilies: []string{"red", "green"}}, []string{"b", "a"}},
		{"非主色也能匹配", Query{ColorFamilies: []string{"orange"}}, []string{"a"}},
		{"标签同组任意匹配", Query{TagIDs: []string{"warm", "landscape"}}, []string{"b", "a"}},
		{"类型忽略大小写和点", Query{Extensions: []string{" .JPG ", "PNG"}}, []string{"b", "a"}},
		{"筛选组之间取交集", Query{ColorFamilies: []string{"red", "green"}, TagIDs: []string{"warm"}, Extensions: []string{"jpg", "png"}}, []string{"a"}},
		{"先限制侧栏范围", Query{Scope: "favorites", TagIDs: []string{"landscape"}}, []string{}},
		{"名称搜索忽略大小写", Query{Search: "  aMbEr  "}, []string{"a"}},
		{"名称搜索中文", Query{Search: "山水"}, []string{"b"}},
		{"扩展名搜索", Query{Search: "JpG"}, []string{"a"}},
		{"标签名称搜索忽略大小写", Query{Search: "landscape"}, []string{"b"}},
		{"不跨字段拼接匹配", Query{Search: "amber jpg"}, []string{}},
		{"过滤再搜索", Query{ColorFamilies: []string{"green"}, Search: "warm"}, []string{}},
		{"回收站保留筛选搜索", Query{Scope: "trash", TagIDs: []string{"warm"}, Search: "removed"}, []string{"e"}},
		{"空白搜索等于无搜索", Query{Search: " \n\t "}, []string{"d", "c", "b", "a"}},
		{"无匹配结果", Query{Search: "missing"}, []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := assetIDs(index.Query(test.q)); !slices.Equal(got, test.want) {
				t.Fatalf("结果 %v，期望 %v", got, test.want)
			}
		})
	}
}

func TestSortingAndDeterministicTies(t *testing.T) {
	now := time.Now()
	index := New(model.Snapshot{Assets: []model.Asset{
		{ID: "z", Name: "Alpha", Size: 10, CreatedAt: now},
		{ID: "a", Name: "alpha", Size: 10, CreatedAt: now},
		{ID: "b", Name: "Beta", Size: 10, CreatedAt: now},
		{ID: "c", Name: "Gamma", Size: 20, CreatedAt: now.Add(time.Second)},
	}})
	tests := []struct {
		name string
		q    Query
		want []string
	}{
		{"默认添加倒序", Query{}, []string{"c", "a", "z", "b"}},
		{"添加升序", Query{Sort: "added"}, []string{"a", "z", "b", "c"}},
		{"添加降序", Query{Sort: "added", Descending: true}, []string{"c", "a", "z", "b"}},
		{"名称升序", Query{Sort: "name"}, []string{"a", "z", "b", "c"}},
		{"名称降序", Query{Sort: "name", Descending: true}, []string{"c", "b", "a", "z"}},
		{"大小升序", Query{Sort: "size"}, []string{"a", "z", "b", "c"}},
		{"大小降序", Query{Sort: "size", Descending: true}, []string{"c", "a", "z", "b"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := assetIDs(index.Query(test.q)); !slices.Equal(got, test.want) {
				t.Fatalf("顺序 %v，期望 %v", got, test.want)
			}
		})
	}
}

func TestCountsIgnoreQueryAndExcludeTrash(t *testing.T) {
	index := New(sampleSnapshot())
	want := Counts{
		All: 4, Favorites: 2, Uncategorized: 2, Untagged: 2, Trash: 1,
		Folders: map[string]int{"parent": 1, "child": 1, "empty": 0},
		Tags:    map[string]int{"warm": 1, "landscape": 1, "unused": 0},
	}
	index.Query(Query{Scope: "folder", FolderID: "parent", Search: "no result", ColorFamilies: []string{"red"}})
	if got := index.Counts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("计数 %+v，期望 %+v", got, want)
	}
	counts := index.Counts()
	counts.Folders["parent"] = 999
	counts.Tags["warm"] = 999
	if !reflect.DeepEqual(index.Counts(), want) {
		t.Fatal("调用方修改返回计数影响了索引")
	}
}

func TestQueriesDoNotReorderSnapshot(t *testing.T) {
	snapshot := sampleSnapshot()
	index := New(snapshot)
	index.Query(Query{Sort: "name", Descending: true})
	index.Query(Query{Sort: "size"})
	if got := index.Snapshot(); !reflect.DeepEqual(got, snapshot) {
		t.Fatalf("查询改变了快照：%+v", got)
	}
	copy := index.Snapshot()
	copy.Assets[0].Name = "Changed"
	copy.Folders[0].Name = "Changed"
	copy.Tags[0].Name = "Changed"
	if got := index.Snapshot(); !reflect.DeepEqual(got, snapshot) {
		t.Fatal("调用方修改返回的顶层数据影响了快照")
	}
}

func TestEmptyLibrary(t *testing.T) {
	index := New(model.Snapshot{})
	if got := index.Query(Query{}); len(got) != 0 {
		t.Fatalf("空图库返回了 %d 项", len(got))
	}
	if got := index.Counts(); got.All != 0 || got.Trash != 0 || len(got.Tags) != 0 || len(got.Folders) != 0 {
		t.Fatalf("空图库计数错误：%+v", got)
	}
}

func benchmarkSnapshot(n int) model.Snapshot {
	snapshot := model.Snapshot{
		Assets: make([]model.Asset, n),
		Tags: []model.Tag{
			{ID: "landscape", Name: "Landscape"}, {ID: "reference", Name: "Reference"},
		},
		Folders: []model.Folder{{ID: "nature", Name: "Nature"}},
	}
	extensions := []string{"jpg", "png", "webp", "tiff"}
	families := []string{"red", "green", "blue", "orange"}
	for i := range snapshot.Assets {
		snapshot.Assets[i] = model.Asset{
			ID: fmt.Sprintf("%064x", i), Name: fmt.Sprintf("Reference %05d", n-i), Extension: extensions[i%4],
			Size: int64((i * 7919) % 1000000), CreatedAt: time.Unix(int64((i*7919)%100000), 0),
			Favorite: i%5 == 0, Trashed: i%23 == 0, FolderIDs: []string{"nature"},
			TagIDs: []string{snapshot.Tags[i%2].ID}, Colors: []model.Color{{Family: families[i%4]}},
		}
	}
	return snapshot
}

var benchmarkResult []model.Asset

func BenchmarkQuery10K(b *testing.B) {
	index := New(benchmarkSnapshot(10000))
	queries := []struct {
		name string
		q    Query
	}{
		{"全部排序", Query{Sort: "added", Descending: true}},
		{"筛选搜索排序", Query{Scope: "folder", FolderID: "nature", ColorFamilies: []string{"red", "blue"}, TagIDs: []string{"landscape"}, Extensions: []string{"jpg", "webp"}, Search: "reference", Sort: "size", Descending: true}},
	}
	for _, query := range queries {
		b.Run(query.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkResult = index.Query(query.q)
			}
		})
	}
}
