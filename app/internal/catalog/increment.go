package catalog

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

func (idx *Index) Merge(partial model.Snapshot) (model.Snapshot, []int, bool) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	snapshot := &idx.snapshot
	foldersChanged := !slices.Equal(snapshot.Folders, partial.Folders)
	metadataChanged := foldersChanged || !reflect.DeepEqual(snapshot.SmartFolders, partial.SmartFolders)
	renamedTags := false
	if !slices.Equal(snapshot.Tags, partial.Tags) {
		oldNames := idx.names
		snapshot.Tags = slices.Clone(partial.Tags)
		idx.names = idx.tagNames()
		for id, name := range oldNames {
			if idx.names[id] != name {
				renamedTags = true
				break
			}
		}
	}
	if foldersChanged {
		snapshot.Folders = slices.Clone(partial.Folders)
	}
	if metadataChanged {
		snapshot.SmartFolders = cloneSmartFolders(partial.SmartFolders)
	}
	if !slices.Equal(snapshot.QuickAccess, partial.QuickAccess) {
		snapshot.QuickAccess = slices.Clone(partial.QuickAccess)
	}

	changed := make([]int, 0, len(partial.Assets))
	var replaced, reordered []int
	for _, asset := range partial.Assets {
		position, exists := idx.positions[asset.ID]
		nextEntry := newEntry(&asset, idx.names)
		if exists {
			old := &snapshot.Assets[position]
			if idx.entries[position].name != nextEntry.name || old.Size != asset.Size || !old.CreatedAt.Equal(asset.CreatedAt) || !old.LastUsedAt.Equal(asset.LastUsedAt) {
				replaced = append(replaced, position)
				reordered = append(reordered, position)
			}
			idx.count(old, -1)
			*old = asset
			idx.entries[position] = nextEntry
		} else {
			position = len(snapshot.Assets)
			idx.positions[asset.ID] = position
			snapshot.Assets = append(snapshot.Assets, asset)
			idx.entries = append(idx.entries, nextEntry)
			reordered = append(reordered, position)
		}
		idx.count(&asset, 1)
		changed = append(changed, position)
	}
	if renamedTags {
		for position := range snapshot.Assets {
			idx.entries[position] = newEntry(&snapshot.Assets[position], idx.names)
		}
	}
	if foldersChanged || renamedTags {
		folders, tags := make(map[string]int, len(snapshot.Folders)), make(map[string]int, len(snapshot.Tags))
		for _, folder := range snapshot.Folders {
			folders[folder.ID] = idx.counts.Folders[folder.ID]
		}
		for _, tag := range snapshot.Tags {
			tags[tag.ID] = idx.counts.Tags[tag.ID]
		}
		idx.counts.Folders, idx.counts.Tags = folders, tags
	} else {
		for _, tag := range snapshot.Tags {
			idx.counts.Tags[tag.ID] += 0
		}
	}
	idx.updateOrders(replaced, reordered, false)
	return *snapshot, changed, metadataChanged || renamedTags
}

func (idx *Index) updateOrders(remove, insert []int, recentOnly bool) {
	if len(insert) == 0 {
		return
	}
	for key, order := range idx.orders {
		if recentOnly && !key.recent {
			continue
		}
		if len(insert) > 64 {
			delete(idx.orders, key)
			continue
		}
		for _, position := range remove {
			offset := slices.Index(order, int32(position))
			order = slices.Delete(order, offset, offset+1)
		}
		for _, position := range insert {
			offset, _ := slices.BinarySearchFunc(order, int32(position), func(left, right int32) int {
				return idx.compare(key, left, right)
			})
			order = slices.Insert(order, offset, int32(position))
		}
		idx.orders[key] = order
	}
}

func queryOrder(q Query) orderKey {
	key := orderKey{field: "added", descending: q.Descending || q.Sort == "", recent: q.Scope == "recent"}
	if q.Sort == "name" || q.Sort == "size" {
		key.field = q.Sort
	}
	return key
}

func AssetOrder(q Query) func(model.Asset, *model.Asset) int {
	key := queryOrder(q)
	return func(left model.Asset, right *model.Asset) int {
		return compareAssets(key, &left, right, strings.ToLower(left.Name), strings.ToLower(right.Name))
	}
}

func (idx *Index) compare(key orderKey, left, right int32) int {
	return compareAssets(key, &idx.snapshot.Assets[left], &idx.snapshot.Assets[right], idx.entries[left].name, idx.entries[right].name)
}

func compareAssets(key orderKey, left, right *model.Asset, leftName, rightName string) int {
	var result int
	if key.recent {
		if result = right.LastUsedAt.Compare(left.LastUsedAt); result != 0 {
			return result
		}
	}
	switch key.field {
	case "name":
		result = strings.Compare(leftName, rightName)
	case "size":
		result = cmp.Compare(left.Size, right.Size)
	default:
		result = left.CreatedAt.Compare(right.CreatedAt)
	}
	if result != 0 {
		if key.descending {
			return -result
		}
		return result
	}
	if result = strings.Compare(leftName, rightName); result != 0 {
		return result
	}
	return strings.Compare(left.ID, right.ID)
}

func (idx *Index) MoveRecent(ids map[string]bool, at time.Time) []int {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	changed := idx.updateLastUsed(ids, at)
	idx.updateOrders(changed, changed, true)
	return changed
}

func (idx *Index) updateLastUsed(ids map[string]bool, at time.Time) []int {
	var changed []int
	for id, used := range ids {
		position, exists := idx.positions[id]
		if !used || !exists {
			continue
		}
		asset := &idx.snapshot.Assets[position]
		if at.After(asset.LastUsedAt) {
			asset.LastUsedAt = at
			changed = append(changed, position)
		}
	}
	return changed
}

func (idx *Index) Count(q Query) int {
	count := 0
	idx.eachMatching(q, func(int) { count++ })
	return count
}

func (idx *Index) Matches(q Query, id string) bool {
	position, exists := idx.positions[id]
	return exists && idx.matcher(q)(position)
}
