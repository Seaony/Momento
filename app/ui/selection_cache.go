package ui

import (
	"slices"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 选中的素材 ID。增删时记一个版本号，派生数据（素材列表、ID、标签和文件夹的成员计数）按版本缓存：
// 全选十万张时，检查器每帧要查上百次成员计数，每次都重新复制十万条素材会让一帧卡上好几秒。
type assetSelection struct {
	ui.Selection[string]
	version uint64
}

func (s *assetSelection) Add(id string)    { s.Selection.Add(id); s.version++ }
func (s *assetSelection) Remove(id string) { s.Selection.Remove(id); s.version++ }
func (s *assetSelection) Clear()           { s.Selection.Clear(); s.version++ }

type selectionCache struct {
	ready        bool
	version      uint64
	data         uint64
	assets       []model.Asset
	ids          []string
	tagCounts    map[string]int
	folderCounts map[string]int
	details      map[string]commonValue // 多选详情里各字段的共同值，按字段名缓存
	any          map[string]bool        // selectionAny 的结果，按问题缓存
	mix          *exportMix             // 导出对话框要的各类数量
	favorite     *bool                  // 点心形是收藏（有没收藏的）还是取消收藏
}

type commonValue struct {
	value string
	mixed bool
}

func (a *App) currentSelection() *selectionCache {
	// 素材数据或可见列表变了，派生数据也要重算。
	data := a.snapshotVersion<<32 ^ a.visibleVersion
	cache := &a.selection
	if cache.ready && cache.version == a.Selected.version && cache.data == data {
		return cache
	}
	assets := a.collectSelectedAssets()
	if assets == nil {
		assets = []model.Asset{}
	}
	*cache = selectionCache{ready: true, version: a.Selected.version, data: data, assets: assets}
	return cache
}

func (a *App) collectSelectedAssets() []model.Asset {
	var out []model.Asset
	if a.assetByID != nil {
		for id := range a.Selected.All() {
			if asset, ok := a.assetByID[id]; ok {
				out = append(out, asset)
			}
		}
		return out
	}
	for _, asset := range a.Visible {
		if a.Selected.Has(asset.ID) {
			out = append(out, asset)
		}
	}
	return out
}

// 返回的切片是共享缓存，调用方只能读、不能改。
func (a *App) SelectedAssets() []model.Asset { return a.currentSelection().assets }

func (a *App) selectedIDs() []string {
	cache := a.currentSelection()
	if cache.ids == nil {
		cache.ids = make([]string, len(cache.assets))
		for i, asset := range cache.assets {
			cache.ids[i] = asset.ID
		}
	}
	// 调用方会把 ID 交给后台任务，给一份自己的副本。
	return append([]string(nil), cache.ids...)
}

// selectionAny 问选中的素材里有没有一张满足 match，按 key 缓存：检查器每帧都要问，全选十万张时不每帧重扫。
func (a *App) selectionAny(key string, match func(model.Asset) bool) bool {
	cache := a.currentSelection()
	if result, ok := cache.any[key]; ok {
		return result
	}
	if cache.any == nil {
		cache.any = map[string]bool{}
	}
	result := slices.ContainsFunc(cache.assets, match)
	cache.any[key] = result
	return result
}

// selectionFavoriteValue 是对全部选中点一下心形的结果：有没收藏的就收藏，全收藏了就取消。悬停在多选卡片上时每帧都要问。
func (a *App) selectionFavoriteValue() bool {
	cache := a.currentSelection()
	if cache.favorite == nil {
		value := false
		for _, asset := range cache.assets {
			if !asset.Favorite {
				value = true
				break
			}
		}
		cache.favorite = &value
	}
	return *cache.favorite
}

// 选中素材里有多少张属于这个标签（folder 为 true 时是文件夹）。
func (a *App) selectionMembers(id string, folder bool) int {
	cache := a.currentSelection()
	counts := &cache.tagCounts
	if folder {
		counts = &cache.folderCounts
	}
	if *counts == nil {
		*counts = map[string]int{}
		for _, asset := range cache.assets {
			members := asset.TagIDs
			if folder {
				members = asset.FolderIDs
			}
			for _, member := range members {
				(*counts)[member]++
			}
		}
	}
	return (*counts)[id]
}

// selectionCommon 是选中素材这一项的共同值；不全一样时 mixed 为 true。全选十万张时每帧都要用，只在选区或数据变了才重算。
func (a *App) selectionCommon(field string, value func(model.Asset) string) (string, bool) {
	cache := a.currentSelection()
	if result, ok := cache.details[field]; ok {
		return result.value, result.mixed
	}
	var result commonValue
	if len(cache.assets) > 0 {
		result.value = value(cache.assets[0])
		for _, asset := range cache.assets[1:] {
			if value(asset) != result.value {
				result = commonValue{mixed: true}
				break
			}
		}
	}
	if cache.details == nil {
		cache.details = map[string]commonValue{}
	}
	cache.details[field] = result
	return result.value, result.mixed
}
