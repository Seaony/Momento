package catalog

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

// referenceQuery 是改用排序缓存之前的 Query 原样实现：先过滤再整体排序。新实现必须和它逐项一致。
func referenceQuery(idx *Index, q Query) []model.Asset {
	search := strings.ToLower(strings.TrimSpace(q.Search))
	if q.Now.IsZero() {
		q.Now = time.Now()
	}
	var smart *model.SmartFolder
	if q.Scope == "smart" {
		for i := range idx.snapshot.SmartFolders {
			if idx.snapshot.SmartFolders[i].ID == q.SmartFolderID {
				smart = &idx.snapshot.SmartFolders[i]
				break
			}
		}
		if smart == nil || ValidateSmartFolder(*smart) != nil {
			return []model.Asset{}
		}
	}
	extensions := make([]string, len(q.Extensions))
	for i, extension := range q.Extensions {
		extensions[i] = normalizedExtension(extension)
	}
	selected := make([]int, 0, len(idx.snapshot.Assets))
	for i, asset := range idx.snapshot.Assets {
		if !inScope(&asset, q) || (smart != nil && !matchesValidatedSmart(&asset, smart)) || !matchesColors(asset.Colors, q.ColorFamilies) || !matchesAny(asset.TagIDs, q.TagIDs) {
			continue
		}
		e := idx.entries[i]
		if len(extensions) > 0 && !slices.Contains(extensions, e.extension) {
			continue
		}
		if search != "" && !strings.Contains(e.name, search) && !strings.Contains(e.extension, search) {
			if !slices.ContainsFunc(e.tagNames, func(name string) bool { return strings.Contains(name, search) }) {
				continue
			}
		}
		selected = append(selected, i)
	}
	descending := q.Descending || q.Sort == ""
	slices.SortFunc(selected, func(left, right int) int {
		a, b := idx.snapshot.Assets[left], idx.snapshot.Assets[right]
		var order int
		if q.Scope == "recent" {
			order = b.LastUsedAt.Compare(a.LastUsedAt)
			if order != 0 {
				return order
			}
		}
		switch q.Sort {
		case "name":
			order = strings.Compare(idx.entries[left].name, idx.entries[right].name)
		case "size":
			order = cmp.Compare(a.Size, b.Size)
		default:
			order = a.CreatedAt.Compare(b.CreatedAt)
		}
		if order != 0 {
			if descending {
				return -order
			}
			return order
		}
		if order = strings.Compare(idx.entries[left].name, idx.entries[right].name); order != 0 {
			return order
		}
		return strings.Compare(a.ID, b.ID)
	})
	assets := make([]model.Asset, len(selected))
	for i, source := range selected {
		assets[i] = idx.snapshot.Assets[source]
	}
	return assets
}

// 随机素材故意让名称、大小、时间大量重复，检验排序的平局规则。
func randomSnapshot(r *rand.Rand, n int) model.Snapshot {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	words := []string{"Amber", "amber", "Moon", "山水", "Reference", "ref", "Sky", ""}
	extensions := []string{"jpg", "JPG", ".png", "webp", "gif", " heic "}
	families := []string{"red", "green", "blue", "orange"}
	snapshot := model.Snapshot{
		Folders: []model.Folder{{ID: "f1"}, {ID: "f2"}, {ID: "f3"}},
		Tags:    []model.Tag{{ID: "t1", Name: "Warm"}, {ID: "t2", Name: "Landscape"}, {ID: "t3", Name: "参考"}},
		SmartFolders: []model.SmartFolder{
			{ID: "s-all", Match: "all", Rules: []model.SmartRule{{Field: "size", Operator: "gte", Value: "20"}, {Field: "color", Operator: "equals", Value: "red"}}},
			{ID: "s-any", Match: "any", Rules: []model.SmartRule{{Field: "name", Operator: "contains", Value: "moon"}, {Field: "tag", Operator: "equals", Value: "t2"}}},
			{ID: "s-bad", Match: "all"},
		},
	}
	pick := func(values []string) []string {
		var out []string
		for _, value := range values {
			if r.IntN(3) == 0 {
				out = append(out, value)
			}
		}
		return out
	}
	// ID 唯一（和数据库主键一致），但顺序打乱。
	perm := r.Perm(n)
	for i := range n {
		asset := model.Asset{
			ID:        fmt.Sprintf("id-%04d", perm[i]),
			Name:      fmt.Sprintf("%s %d", words[r.IntN(len(words))], r.IntN(5)),
			Extension: extensions[r.IntN(len(extensions))],
			Size:      int64(r.IntN(40)),
			CreatedAt: base.Add(time.Duration(r.IntN(6)) * time.Hour),
			Favorite:  r.IntN(3) == 0,
			Trashed:   r.IntN(6) == 0,
			FolderIDs: pick([]string{"f1", "f2", "f3"}),
			TagIDs:    pick([]string{"t1", "t2", "t3"}),
		}
		for _, family := range pick(families) {
			asset.Colors = append(asset.Colors, model.Color{Family: family})
		}
		switch r.IntN(4) {
		case 1:
			asset.LastUsedAt = base.Add(time.Duration(r.IntN(4)) * 24 * time.Hour)
		case 2:
			asset.LastUsedAt = base.AddDate(0, 0, -40)
		case 3:
			asset.LastUsedAt = base.AddDate(0, 1, 0)
		}
		snapshot.Assets = append(snapshot.Assets, asset)
	}
	return snapshot
}

func randomQuery(r *rand.Rand, now time.Time) Query {
	one := func(values ...string) string { return values[r.IntN(len(values))] }
	some := func(values ...string) []string {
		var out []string
		for _, value := range values {
			if r.IntN(4) == 0 {
				out = append(out, value)
			}
		}
		return out
	}
	q := Query{
		Now:           now,
		Scope:         one("", "all", "recent", "favorites", "uncategorized", "untagged", "folder", "smart", "tag", "trash", "unknown"),
		FolderID:      one("f1", "f2", "missing"),
		ScopeTagID:    one("t1", "t2", "missing"),
		SmartFolderID: one("s-all", "s-any", "s-bad", "missing"),
		ColorFamilies: some("red", "green", "blue"),
		TagIDs:        some("t1", "t2", "t3"),
		Extensions:    some("jpg", ".PNG", "webp"),
		Search:        one("", "", "a", "AMBER", " moon ", "山", "warm", "参考", "jpg", "zzz"),
		Sort:          one("", "added", "name", "size", "unknown"),
		Descending:    r.IntN(2) == 0,
	}
	return q
}

func TestQueryMatchesReferenceImplementation(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	for round := range 30 {
		snapshot := randomSnapshot(r, 50+r.IntN(250))
		idx := New(snapshot)
		check := func(stage string) {
			t.Helper()
			for range 200 {
				q := randomQuery(r, now)
				got, want := idx.Query(q), referenceQuery(idx, q)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("第 %d 轮%s，查询 %+v\n结果 %v\n期望 %v", round, stage, q, assetIDs(got), assetIDs(want))
				}
			}
		}
		check("")
		// 改使用时间后，最近使用的排序缓存必须跟着变。
		used := map[string]bool{}
		for _, asset := range snapshot.Assets {
			if r.IntN(4) == 0 {
				used[asset.ID] = true
			}
		}
		idx.MoveRecent(used, now.Add(-time.Duration(r.IntN(48))*time.Hour))
		check("（增量更新使用时间后）")
	}
}

// 调用方先改了共享切片里的使用时间再通知索引（ui 现在就是这么做的），最近使用的排序也要更新。
func TestUpdateLastUsedRefreshesRecentOrderAfterCallerMutation(t *testing.T) {
	now := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	snapshot := model.Snapshot{Assets: []model.Asset{
		{ID: "a", Name: "A", LastUsedAt: now.Add(-2 * time.Hour)},
		{ID: "b", Name: "B", LastUsedAt: now.Add(-time.Hour)},
	}}
	idx := New(snapshot)
	if got := assetIDs(idx.Query(Query{Scope: "recent", Now: now})); !slices.Equal(got, []string{"b", "a"}) {
		t.Fatalf("初始顺序 %v", got)
	}
	snapshot.Assets[0].LastUsedAt = now
	idx.UpdateLastUsed(map[string]bool{"a": true}, now)
	if got := assetIDs(idx.Query(Query{Scope: "recent", Now: now})); !slices.Equal(got, []string{"a", "b"}) {
		t.Fatalf("更新后顺序 %v", got)
	}
}
