package catalog

import (
	"cmp"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

type Query struct {
	SmartFolderID string
	Now           time.Time
	Scope         string
	FolderID      string
	ColorFamilies []string
	TagIDs        []string
	Extensions    []string
	// Kinds 按素材种类筛（model.KindVideo 等），和 Extensions 是同一组「文件类型」条件：命中任一个就算。
	Kinds      []string
	Search     string
	Sort       string
	Descending bool
}

type Counts struct {
	All, Favorites, Uncategorized, Untagged, Trash int
	Folders, Tags                                  map[string]int
}

type entry struct {
	name, extension string
	tagNames        []string
}

// 排好序的下标按这三样区分：排序字段、方向、是否先按最近使用排。
type orderKey struct {
	field      string
	descending bool
	recent     bool
}

// 结构变化后重建索引；使用时间不影响搜索项和计数，可直接更新。
type Index struct {
	snapshot model.Snapshot
	entries  []entry
	counts   Counts
	// 每种排序方式第一次用到时把全部素材排一次，之后搜索、筛选只在这份顺序上过滤。
	mu     sync.Mutex
	orders map[orderKey][]int32
}

// New 直接接管 snapshot.Assets（不复制，省一份素材内存），调用方之后不能再改这个切片和其中的素材；
// UpdateLastUsed 会原地改它。其他字段会复制。
func New(snapshot model.Snapshot) *Index {
	idx := &Index{
		snapshot: model.Snapshot{
			Assets: snapshot.Assets, Folders: slices.Clone(snapshot.Folders), Tags: slices.Clone(snapshot.Tags),
			SmartFolders: cloneSmartFolders(snapshot.SmartFolders), QuickAccess: slices.Clone(snapshot.QuickAccess),
		},
		entries: make([]entry, len(snapshot.Assets)),
		counts:  Counts{Folders: make(map[string]int), Tags: make(map[string]int)},
		orders:  map[orderKey][]int32{},
	}
	tagNames := make(map[string]string, len(snapshot.Tags))
	for _, tag := range snapshot.Tags {
		tagNames[tag.ID] = strings.ToLower(tag.Name)
		idx.counts.Tags[tag.ID] = 0
	}
	for _, folder := range snapshot.Folders {
		idx.counts.Folders[folder.ID] = 0
	}
	for i, asset := range snapshot.Assets {
		e := entry{name: strings.ToLower(asset.Name), extension: normalizedExtension(asset.Extension)}
		for _, id := range asset.TagIDs {
			e.tagNames = append(e.tagNames, tagNames[id])
		}
		idx.entries[i] = e
		if asset.Trashed {
			idx.counts.Trash++
			continue
		}
		idx.counts.All++
		if asset.Favorite {
			idx.counts.Favorites++
		}
		if len(asset.FolderIDs) == 0 {
			idx.counts.Uncategorized++
		}
		if len(asset.TagIDs) == 0 {
			idx.counts.Untagged++
		}
		for _, id := range asset.FolderIDs {
			idx.counts.Folders[id]++
		}
		for _, id := range asset.TagIDs {
			idx.counts.Tags[id]++
		}
	}
	return idx
}

// Query 按范围、筛选、搜索、排序处理；空排序采用添加时间倒序。
func (idx *Index) Query(q Query) []model.Asset {
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
	// 先按存放顺序过滤（顺着内存读，快），再沿排好的顺序挑出命中的素材。
	matched := make([]bool, len(idx.snapshot.Assets))
	count := 0
	for i := range idx.snapshot.Assets {
		asset := &idx.snapshot.Assets[i]
		if !inScope(asset, q) || (smart != nil && !matchesValidatedSmart(asset, smart)) || !matchesColors(asset.Colors, q.ColorFamilies) || !matchesAny(asset.TagIDs, q.TagIDs) {
			continue
		}
		e := &idx.entries[i]
		if len(extensions)+len(q.Kinds) > 0 && !slices.Contains(extensions, e.extension) && !slices.Contains(q.Kinds, asset.Kind) {
			continue
		}
		if search != "" && !strings.Contains(e.name, search) && !strings.Contains(e.extension, search) {
			if !slices.ContainsFunc(e.tagNames, func(name string) bool { return strings.Contains(name, search) }) {
				continue
			}
		}
		matched[i] = true
		count++
	}
	assets := make([]model.Asset, 0, count)
	if count == 0 {
		return assets
	}
	for _, i := range idx.order(q) {
		if matched[i] {
			assets = append(assets, idx.snapshot.Assets[i])
		}
	}
	return assets
}

// order 返回按查询的排序方式排好的全部素材下标；排序是全序（最后按 ID 定），所以先排全部再过滤和先过滤再排结果一样。
func (idx *Index) order(q Query) []int32 {
	key := orderKey{field: "added", descending: q.Descending || q.Sort == "", recent: q.Scope == "recent"}
	if q.Sort == "name" || q.Sort == "size" {
		key.field = q.Sort
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if order, ok := idx.orders[key]; ok {
		return order
	}
	order := make([]int32, len(idx.snapshot.Assets))
	for i := range order {
		order[i] = int32(i)
	}
	slices.SortFunc(order, func(left, right int32) int {
		a, b := &idx.snapshot.Assets[left], &idx.snapshot.Assets[right]
		var result int
		if key.recent {
			if result = b.LastUsedAt.Compare(a.LastUsedAt); result != 0 {
				return result
			}
		}
		switch key.field {
		case "name":
			result = strings.Compare(idx.entries[left].name, idx.entries[right].name)
		case "size":
			result = cmp.Compare(a.Size, b.Size)
		default:
			result = a.CreatedAt.Compare(b.CreatedAt)
		}
		if result != 0 {
			if key.descending {
				return -result
			}
			return result
		}
		if result = strings.Compare(idx.entries[left].name, idx.entries[right].name); result != 0 {
			return result
		}
		return strings.Compare(a.ID, b.ID)
	})
	idx.orders[key] = order
	return order
}

// UpdateLastUsed 原地改素材的使用时间，New 时传入的切片（调用方手里的快照）会一起变。
// 调用方可能已经先改过同一个切片，所以只要有 ids 就作废「最近使用」的排序，不看这里有没有改动。
func (idx *Index) UpdateLastUsed(ids map[string]bool, at time.Time) {
	if len(ids) == 0 {
		return
	}
	for i := range idx.snapshot.Assets {
		asset := &idx.snapshot.Assets[i]
		if ids[asset.ID] && at.After(asset.LastUsedAt) {
			asset.LastUsedAt = at
		}
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	for key := range idx.orders {
		if key.recent {
			delete(idx.orders, key)
		}
	}
}

func (idx *Index) Counts() Counts {
	counts := idx.counts
	counts.Folders = maps.Clone(counts.Folders)
	counts.Tags = maps.Clone(counts.Tags)
	return counts
}

func (idx *Index) Snapshot() model.Snapshot {
	return model.Snapshot{
		Assets: slices.Clone(idx.snapshot.Assets), Folders: slices.Clone(idx.snapshot.Folders), Tags: slices.Clone(idx.snapshot.Tags),
		SmartFolders: cloneSmartFolders(idx.snapshot.SmartFolders), QuickAccess: slices.Clone(idx.snapshot.QuickAccess),
	}
}

func inScope(asset *model.Asset, q Query) bool {
	if q.Scope == "trash" {
		return asset.Trashed
	}
	if asset.Trashed {
		return false
	}
	switch q.Scope {
	case "recent":
		return !asset.LastUsedAt.IsZero() && !asset.LastUsedAt.Before(q.Now.AddDate(0, 0, -30)) && !asset.LastUsedAt.After(q.Now)
	case "smart", "tag":
		return true
	case "", "all":
		return true
	case "favorites":
		return asset.Favorite
	case "uncategorized":
		return len(asset.FolderIDs) == 0
	case "untagged":
		return len(asset.TagIDs) == 0
	case "folder":
		return slices.Contains(asset.FolderIDs, q.FolderID)
	default:
		return false
	}
}

func matchesAny(values, selected []string) bool {
	return len(selected) == 0 || slices.ContainsFunc(values, func(value string) bool { return slices.Contains(selected, value) })
}

func matchesColors(colors []model.Color, selected []string) bool {
	return len(selected) == 0 || slices.ContainsFunc(colors, func(color model.Color) bool { return slices.Contains(selected, color.Family) })
}

func normalizedExtension(extension string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(extension)), ".")
}

func cloneSmartFolders(folders []model.SmartFolder) []model.SmartFolder {
	copy := slices.Clone(folders)
	for i := range copy {
		copy[i].Rules = slices.Clone(copy[i].Rules)
	}
	return copy
}
