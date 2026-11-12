package ui

import (
	"cmp"
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type browseLocation struct {
	Scope, FolderID, TagID string
	SmartFolderID          string
	TagPage                bool
}

type browseState struct {
	History     []browseLocation
	Position    int
	RandomOrder map[string]int
	TagID       string
}

func (a *App) browseLocation() browseLocation {
	scope := a.Query.Scope
	if scope == "" {
		scope = "all"
	}
	return browseLocation{Scope: scope, FolderID: a.Query.FolderID, TagID: a.Browse.TagID, SmartFolderID: a.Query.SmartFolderID, TagPage: a.TagPage}
}

func (a *App) navigateScope(scope, folderID string, tags bool) {
	if scope == "" {
		scope = "all"
	}
	next := browseLocation{Scope: scope, FolderID: folderID, TagPage: tags}
	current := a.browseLocation()
	if len(a.Browse.History) == 0 {
		a.Browse.History = []browseLocation{current}
	}
	if next != current {
		a.Browse.History = append(a.Browse.History[:a.Browse.Position+1], next)
		a.Browse.Position = len(a.Browse.History) - 1
	}
	if scope == "random" && !tags {
		a.shuffleBrowse()
	}
	a.applyBrowseLocation(next)
}

func (a *App) applyBrowseLocation(location browseLocation) {
	a.Query.Scope, a.Query.FolderID, a.TagPage = location.Scope, location.FolderID, location.TagPage
	a.Browse.TagID = location.TagID
	a.Query.SmartFolderID = location.SmartFolderID
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
	a.Selected.Clear()
	a.Toolbar.SortOpen = false
	a.Refine()
}

func (a *App) BrowseTag(id string) {
	current := a.browseLocation()
	next := browseLocation{Scope: "tag", TagID: id}
	if len(a.Browse.History) == 0 {
		a.Browse.History = []browseLocation{current}
	}
	if next != current {
		a.Browse.History = append(a.Browse.History[:a.Browse.Position+1], next)
		a.Browse.Position = len(a.Browse.History) - 1
	}
	a.applyBrowseLocation(next)
}

func (a *App) ShowTagsPage()            { a.navigateScope(a.Query.Scope, a.Query.FolderID, true) }
func (a *App) CanNavigateBack() bool    { return a.Browse.Position > 0 }
func (a *App) CanNavigateForward() bool { return a.Browse.Position+1 < len(a.Browse.History) }
func (a *App) NavigateBack() {
	if a.CanNavigateBack() {
		a.Browse.Position--
		a.applyBrowseLocation(a.Browse.History[a.Browse.Position])
	}
}
func (a *App) NavigateForward() {
	if a.CanNavigateForward() {
		a.Browse.Position++
		a.applyBrowseLocation(a.Browse.History[a.Browse.Position])
	}
}

func (a *App) shuffleBrowse() {
	ids := make([]string, 0, len(a.Snapshot.Assets))
	for _, asset := range a.Snapshot.Assets {
		if !asset.Trashed {
			ids = append(ids, asset.ID)
		}
	}
	rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
	a.Browse.RandomOrder = make(map[string]int, len(ids))
	for i, id := range ids {
		a.Browse.RandomOrder[id] = i
	}
}

func (a *App) browseAssets(query catalog.Query) []model.Asset {
	if query.Scope == "tag" {
		query.Scope = "all"
		query.TagIDs = []string{a.Browse.TagID}
	}
	if query.Scope != "random" {
		return a.Index.Query(query)
	}
	query.Scope = "all"
	if a.Browse.RandomOrder == nil {
		a.shuffleBrowse()
	}
	result := a.Index.Query(query)
	for _, asset := range result {
		if _, ok := a.Browse.RandomOrder[asset.ID]; !ok {
			a.Browse.RandomOrder[asset.ID] = len(a.Browse.RandomOrder)
		}
	}
	slices.SortFunc(result, func(left, right model.Asset) int {
		return cmp.Compare(a.Browse.RandomOrder[left.ID], a.Browse.RandomOrder[right.ID])
	})
	return result
}

func (a *App) thumbnailScale() float32 {
	if a.Preferences.ThumbnailSize == 0 {
		return 1
	}
	return max(.5, min(2, a.Preferences.ThumbnailSize))
}
func (a *App) SetThumbnailSize(size float32) {
	size = max(.5, min(2, size))
	if size == a.thumbnailScale() {
		return
	}
	a.Preferences.ThumbnailSize = size
	a.layout = nil
	a.SavePreferences()
}
func (a *App) ChangeThumbnailSize(delta float32) {
	if a.ViewMode != 2 {
		a.SetThumbnailSize(a.thumbnailScale() + delta)
	}
}

var previewSystemItem = mygo.Shell.PreviewItem

func (a *App) assetPreviewShortcut(c *ui.Context, area ui.Element) {
	if a.Preview.Open || c.FocusTakesText() || a.Settings.RecordingShortcut != "" {
		return
	}
	mods, key, ok := parseShortcut(a.Shortcut("preview"))
	// 在卡片的祖先上认领按键，避免空格先变成点击或翻页。
	if ok && area.Shortcut(mods, key) {
		a.activatePreviewShortcut()
	}
}

func (a *App) activatePreviewShortcut() {
	held := a.Shortcut("preview") == "Space"
	if id := a.hovered; id != "" {
		a.ActivateAsset(id, held)
	} else if selected := a.SelectedAssets(); len(selected) > 0 {
		a.ActivateAsset(selected[0].ID, held)
	}
}

func (a *App) ActivateAsset(id string, space bool) {
	var asset model.Asset
	found := false
	for _, candidate := range a.Visible {
		if candidate.ID == id {
			asset, found = candidate, true
			break
		}
	}
	if !found {
		return
	}
	if a.Library != nil {
		path := filepath.Join(a.Library.Root, filepath.FromSlash(asset.RelativePath))
		if space && a.Preferences.SpaceKey == "system" && runtime.GOOS == "darwin" {
			if err := previewSystemItem(path); err != nil {
				a.ShowError(err)
			} else {
				a.MarkAssetsUsed([]string{id})
			}
			return
		}
		if !space && a.Preferences.DoubleClick == "external" {
			if err := mygo.Shell.OpenPath(path); err != nil {
				a.ShowError(err)
			} else {
				a.MarkAssetsUsed([]string{id})
			}
			return
		}
	}
	a.ShowPreview(id, space)
}

func (a *App) scopeSummary() (string, int, int64) {
	name := a.T("all")
	query := catalog.Query{Scope: a.Query.Scope, FolderID: a.Query.FolderID, SmartFolderID: a.Query.SmartFolderID}
	if query.Scope == "random" {
		query.Scope = "all"
		name = a.T("random_browse")
	} else if a.TagPage {
		query.Scope = "all"
		name = a.T("tag_management")
	} else if query.Scope == "tag" {
		query.Scope = "all"
		query.TagIDs = []string{a.Browse.TagID}
		for _, tag := range a.Snapshot.Tags {
			if tag.ID == a.Browse.TagID {
				name = tag.Name
				break
			}
		}
	} else if query.Scope == "smart" {
		for _, folder := range a.Snapshot.SmartFolders {
			if folder.ID == query.SmartFolderID {
				name = folder.Name
				break
			}
		}
	} else if query.Scope == "folder" {
		for _, folder := range a.Snapshot.Folders {
			if folder.ID == query.FolderID {
				name = folder.Name
				break
			}
		}
	} else if query.Scope != "" && query.Scope != "all" {
		name = a.T(query.Scope)
	}
	if a.Index == nil {
		return name, 0, 0
	}
	assets := a.Index.Query(query)
	var size int64
	for _, asset := range assets {
		size += asset.Size
	}
	return name, len(assets), size
}
