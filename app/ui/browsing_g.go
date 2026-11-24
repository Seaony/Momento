package ui

import (
	"slices"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type gBrowsingState struct {
	QuickList     ui.ListState
	SmartOpen     bool
	SmartID       string
	SmartName     string
	SmartMatch    int
	SmartRules    []model.SmartRule
	// smartRuleKeys 和 SmartRules 一一对应，是每条规则的身份：删掉中间一条时收起的是它自己，不是最后一行。
	smartRuleKeys []uint64
	smartRuleNext uint64
	SmartError    string
	Saving        bool
	RulesScroll   ui.ScrollState
	countsVersion uint64 // 计数按 snapshotVersion 缓存；索引会原地更新，不能按指针判断
	countsReady   bool
	Counts        catalog.Counts
	SmartCounts   map[string]int
	smartCounting bool
}

func (a *App) BrowseSmartFolder(id string) {
	a.applyBrowseLocation(browseLocation{Scope: "smart", SmartFolderID: id})
}

func (a *App) quickAccessTarget(entry model.QuickAccess) (name, icon string, count int, selected bool) {
	countKey := entry.TargetID
	switch entry.Kind {
	case "folder":
		for _, folder := range a.Snapshot.Folders {
			if folder.ID == entry.TargetID {
				name, icon = folder.Name, "文件夹"
				if folderIconID(folder.Icon) != "folder" {
					icon = folderIconPrefix + folder.Icon
				}
				break
			}
		}
		selected = !a.TagPage && a.Query.Scope == "folder" && a.Query.FolderID == entry.TargetID
	case "tag":
		for _, tag := range a.Snapshot.Tags {
			if tag.ID == entry.TargetID {
				name = tag.Name
				break
			}
		}
		icon = "标签"
		selected = !a.TagPage && a.Query.Scope == "tag" && a.Browse.TagID == entry.TargetID
	case "smart":
		for _, folder := range a.Snapshot.SmartFolders {
			if folder.ID == entry.TargetID {
				name = folder.Name
				break
			}
		}

		selected = !a.TagPage && a.Query.Scope == "smart" && a.Query.SmartFolderID == entry.TargetID
	}
	if name != "" && a.Preferences.SidebarCounts && a.Index != nil {
		switch entry.Kind {
		case "folder":
			a.gBrowsingCounts()
			count = a.GBrowsing.Counts.Folders[countKey]
		case "tag":
			a.gBrowsingCounts()
			count = a.GBrowsing.Counts.Tags[countKey]
		case "smart":
			count = a.smartFolderCount(countKey)
		}
	}
	return
}

func (a *App) openQuickAccess(entry model.QuickAccess) {
	switch entry.Kind {
	case "folder":
		a.SetScope("folder", entry.TargetID)
	case "tag":
		a.BrowseTag(entry.TargetID)
	case "smart":
		a.BrowseSmartFolder(entry.TargetID)
	}
}

func (a *App) gBrowsingCounts() {
	// 按素材数据版本缓存：只改使用时间不算（智能文件夹的条件里没有使用时间），打开预览、翻页时不用把每个智能文件夹重查一遍。
	if a.GBrowsing.countsReady && a.GBrowsing.countsVersion == a.snapshotVersion {
		return
	}
	a.GBrowsing.countsReady, a.GBrowsing.countsVersion = true, a.snapshotVersion
	a.GBrowsing.Counts = catalog.Counts{}
	if a.Index != nil {
		a.GBrowsing.Counts = a.Index.Counts()
	}
}
func (a *App) smartFolderCount(id string) int {
	if !a.Preferences.SidebarCounts || a.Index == nil {
		return 0
	}
	a.gBrowsingCounts()
	state := &a.GBrowsing
	if state.SmartCounts != nil && (state.smartCounting || a.Window != nil) {
		return state.SmartCounts[id]
	}
	if a.Window == nil {
		state.SmartCounts = countSmartFolders(a.Index, a.Snapshot.SmartFolders)
		return state.SmartCounts[id]
	}
	if !state.smartCounting {
		state.smartCounting = true
		version, generation := a.snapshotVersion, a.generation
		snapshot := a.smartCountSnapshot()
		window := a.Window
		go func() {
			counts := countSmartFolders(catalog.New(snapshot), snapshot.SmartFolders)
			window.Update(func() { a.finishSmartFolderCounts(version, generation, counts) })
		}()
	}
	return 0
}

func (a *App) smartCountSnapshot() model.Snapshot {
	folders := slices.Clone(a.Snapshot.SmartFolders)
	for i := range folders {
		folders[i].Rules = slices.Clone(folders[i].Rules)
	}
	return model.Snapshot{
		Assets:       slices.Clone(a.Snapshot.Assets),
		SmartFolders: folders,
	}
}

func countSmartFolders(index *catalog.Index, folders []model.SmartFolder) map[string]int {
	counts := make(map[string]int, len(folders))
	for _, folder := range folders {
		counts[folder.ID] = index.Count(catalog.Query{Scope: "smart", SmartFolderID: folder.ID})
	}
	return counts
}

func (a *App) finishSmartFolderCounts(version, generation uint64, counts map[string]int) {
	if generation == a.generation {
		a.GBrowsing.smartCounting = false
		if version == a.snapshotVersion {
			a.GBrowsing.SmartCounts = counts
		}
	}
}
