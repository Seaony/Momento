package catalog

import (
	"cmp"
	"maps"
	"slices"
	"strings"
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
	Search        string
	Sort          string
	Descending    bool
}

type Counts struct {
	All, Favorites, Uncategorized, Untagged, Trash int
	Folders, Tags                                  map[string]int
}

type entry struct {
	name, extension string
	tagNames        []string
}

// Index 对应一次图库快照；数据变化后用新快照重建。
type Index struct {
	snapshot model.Snapshot
	entries  []entry
	counts   Counts
}

func New(snapshot model.Snapshot) *Index {
	idx := &Index{
		snapshot: model.Snapshot{
			Assets: slices.Clone(snapshot.Assets), Folders: slices.Clone(snapshot.Folders), Tags: slices.Clone(snapshot.Tags),
			SmartFolders: cloneSmartFolders(snapshot.SmartFolders), QuickAccess: slices.Clone(snapshot.QuickAccess),
		},
		entries: make([]entry, len(snapshot.Assets)),
		counts:  Counts{Folders: make(map[string]int), Tags: make(map[string]int)},
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
	selected := make([]int, 0, len(idx.snapshot.Assets))
	for i, asset := range idx.snapshot.Assets {
		if !inScope(asset, q) || (smart != nil && !matchesValidatedSmart(asset, *smart)) || !matchesColors(asset.Colors, q.ColorFamilies) || !matchesAny(asset.TagIDs, q.TagIDs) {
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

func inScope(asset model.Asset, q Query) bool {
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
