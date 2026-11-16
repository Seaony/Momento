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
	Kinds []string
	// Shapes 按形状和比例筛（ShapeLandscape 等，或 "16:9" 这种比例，不分横竖），命中任一个就算。
	Shapes []string
	// Ratings 按星级筛，0 是没评分，命中任一个就算。
	Ratings []int
	// IncludeSubfolders 为 true 时，文件夹范围也包括它下面各层子文件夹里的素材。
	IncludeSubfolders bool
	// folderSet 是 IncludeSubfolders 时算出的文件夹和全部子孙文件夹。
	folderSet  map[string]bool
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
	// note、source 是小写的备注和来源链接，搜索也查这两项；为空时不占内存。
	note, source string
	tagNames     []string
}

func normalizedExtensions(values []string) []string {
	extensions := make([]string, len(values))
	for i, extension := range values {
		extensions[i] = normalizedExtension(extension)
	}
	return extensions
}

// matchesFilters 是文件类型和搜索两项条件（范围、颜色、标签、形状另外判断）。
func (idx *Index) matchesFilters(i int, extensions, kinds []string, search string) bool {
	e := &idx.entries[i]
	if len(extensions)+len(kinds) > 0 && !slices.Contains(extensions, e.extension) && !slices.Contains(kinds, idx.snapshot.Assets[i].Kind) {
		return false
	}
	return search == "" || e.matches(search)
}

// matches 看名称、扩展名、备注、来源链接、标签里有没有搜索词。
func (e *entry) matches(search string) bool {
	return strings.Contains(e.name, search) || strings.Contains(e.extension, search) || strings.Contains(e.note, search) ||
		strings.Contains(e.source, search) || slices.ContainsFunc(e.tagNames, func(name string) bool { return strings.Contains(name, search) })
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
		e := entry{name: strings.ToLower(asset.Name), extension: normalizedExtension(asset.Extension), note: strings.ToLower(asset.Note), source: strings.ToLower(asset.SourceURL)}
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
	// 先按存放顺序过滤（顺着内存读，快），再沿排好的顺序挑出命中的素材。
	matched := make([]bool, len(idx.snapshot.Assets))
	count := 0
	idx.eachMatching(q, func(i int) {
		matched[i] = true
		count++
	})
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

// eachMatching 按存放顺序对命中查询全部条件的素材下标调用 fn；智能文件夹找不到或条件无效时一个都不算。
func (idx *Index) eachMatching(q Query, fn func(int)) {
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
			return
		}
	}
	if q.Scope == "folder" && q.IncludeSubfolders {
		q.folderSet = FolderTree(idx.snapshot.Folders, q.FolderID)
	}
	extensions := normalizedExtensions(q.Extensions)
	for i := range idx.snapshot.Assets {
		asset := &idx.snapshot.Assets[i]
		if !inScope(asset, q) || (smart != nil && !matchesValidatedSmart(asset, smart)) || !matchesColors(asset.Colors, q.ColorFamilies) || !matchesAny(asset.TagIDs, q.TagIDs) {
			continue
		}
		if !idx.matchesFilters(i, extensions, q.Kinds, search) || !matchesShapes(asset, q.Shapes) || (len(q.Ratings) > 0 && !slices.Contains(q.Ratings, asset.Rating)) {
			continue
		}
		fn(i)
	}
}

// order 返回按查询的排序方式排好的全部素材下标；排序是全序（最后按 ID 定），所以先排全部再过滤和先过滤再排结果一样。
func (idx *Index) order(q Query) []int32 {
	key := orderKey{field: "added", descending: q.Descending || q.Sort == "", recent: q.Scope == "recent"}
	if q.Sort == "name" || q.Sort == "size" || q.Sort == "rating" {
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
		case "rating":
			result = cmp.Compare(a.Rating, b.Rating)
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
		if q.folderSet != nil {
			return slices.ContainsFunc(asset.FolderIDs, func(id string) bool { return q.folderSet[id] })
		}
		return slices.Contains(asset.FolderIDs, q.FolderID)
	default:
		return false
	}
}

// FolderTree 是这个文件夹和它下面各层子文件夹的 ID。
func FolderTree(folders []model.Folder, id string) map[string]bool {
	children := map[string][]string{}
	for _, folder := range folders {
		children[folder.ParentID] = append(children[folder.ParentID], folder.ID)
	}
	tree := map[string]bool{}
	pending := []string{id}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		// 数据里万一有环，访问过的不再展开。
		if current == "" || tree[current] {
			continue
		}
		tree[current] = true
		pending = append(pending, children[current]...)
	}
	return tree
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
