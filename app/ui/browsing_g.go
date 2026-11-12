package ui

import (
	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type gBrowsingState struct {
	QuickList   ui.ListState
	SmartOpen   bool
	SmartID     string
	SmartName   string
	SmartMatch  int
	SmartRules  []model.SmartRule
	SmartError  string
	Saving      bool
	RulesScroll ui.ScrollState
	CountsIndex *catalog.Index
	Counts      catalog.Counts
	SmartCounts map[string]int
}

func (a *App) BrowseSmartFolder(id string) {
	current := a.browseLocation()
	next := browseLocation{Scope: "smart", SmartFolderID: id}
	if len(a.Browse.History) == 0 {
		a.Browse.History = []browseLocation{current}
	}
	if next != current {
		a.Browse.History = append(a.Browse.History[:a.Browse.Position+1], next)
		a.Browse.Position = len(a.Browse.History) - 1
	}
	a.applyBrowseLocation(next)
}

func (a *App) quickAccessTarget(entry model.QuickAccess) (name, icon string, count int, selected bool) {
	countKey := entry.TargetID
	switch entry.Kind {
	case "folder":
		for _, folder := range a.Snapshot.Folders {
			if folder.ID == entry.TargetID {
				name = folder.Name
				break
			}
		}
		icon = "文件夹"
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
	if name != "" && a.Index != nil {
		a.gBrowsingCounts()
		switch entry.Kind {
		case "folder":
			count = a.GBrowsing.Counts.Folders[countKey]
		case "tag":
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
	if a.GBrowsing.CountsIndex == a.Index {
		return
	}
	a.GBrowsing.CountsIndex = a.Index
	a.GBrowsing.SmartCounts = map[string]int{}
	a.GBrowsing.Counts = catalog.Counts{}
	if a.Index != nil {
		a.GBrowsing.Counts = a.Index.Counts()
	}
}
func (a *App) smartFolderCount(id string) int {
	if a.Index == nil {
		return 0
	}
	a.gBrowsingCounts()
	if count, exists := a.GBrowsing.SmartCounts[id]; exists {
		return count
	}
	count := len(a.Index.Query(catalog.Query{Scope: "smart", SmartFolderID: id}))
	a.GBrowsing.SmartCounts[id] = count
	return count
}
