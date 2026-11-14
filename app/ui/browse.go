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
	RandomOrder map[string]int
	TagID       string
}

func (a *App) navigateScope(scope, folderID string, tags bool) {
	if scope == "" {
		scope = "all"
	}
	next := browseLocation{Scope: scope, FolderID: folderID, TagPage: tags}
	if scope == "random" && !tags {
		a.shuffleBrowse()
	}
	a.applyBrowseLocation(next)
}

func (a *App) applyBrowseLocation(location browseLocation) {
	a.applyBrowseSelection(location, nil)
}

func (a *App) applyBrowseSelection(location browseLocation, selectedIDs []string) {
	a.Query.Scope, a.Query.FolderID, a.TagPage = location.Scope, location.FolderID, location.TagPage
	a.Browse.TagID = location.TagID
	a.Query.SmartFolderID = location.SmartFolderID
	if a.MenuChanged != nil {
		a.MenuChanged()
	}
	a.Selected.Clear()
	for _, id := range selectedIDs {
		if _, ok := a.assetByID[id]; ok {
			a.Selected.Add(id)
		}
	}
	a.Toolbar.SortOpen = false
	a.Refine()
}

func (a *App) BrowseTag(id string) {
	a.applyBrowseLocation(browseLocation{Scope: "tag", TagID: id})
}

func (a *App) ShowTagsPage() { a.navigateScope(a.Query.Scope, a.Query.FolderID, true) }

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
	a.queuePreferences()
	a.InvalidateWindows()
}
func (a *App) ChangeThumbnailSize(delta float32) {
	if a.ViewMode != 2 {
		a.SetThumbnailSize(a.thumbnailScale() + delta)
	}
}

var previewSystemItem = mygo.Shell.PreviewItem

func (a *App) assetPreviewShortcut(c *ui.Context, area ui.Element) {
	if a.PreviewInputPending() || a.PreviewOpen() || c.FocusTakesText() || a.Settings.RecordingShortcut != "" {
		return
	}
	mods, key, ok := parseShortcut(a.Shortcut("preview"))
	// 在卡片的祖先上认领按键，避免空格先变成点击或翻页。
	if ok && area.Shortcut(mods, key) {
		a.activatePreviewShortcut(c)
	}
}

func (a *App) activatePreviewShortcut(c *ui.Context) {
	id := a.hovered
	if id == "" && a.selectedIndex >= 0 && a.selectedIndex < len(a.Visible) && a.Selected.Has(a.Visible[a.selectedIndex].ID) {
		id = a.Visible[a.selectedIndex].ID
	}
	if id == "" {
		for _, asset := range a.Visible {
			if a.Selected.Has(asset.ID) {
				id = asset.ID
				break
			}
		}
	}
	if id == "" {
		return
	}
	if a.Shortcut("preview") == "Space" && a.Preview.spaceDown.IsZero() {
		a.Preview.spaceDown = c.Now()
	}
	a.ActivateAsset(id, true)
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
	a.ShowPreview(id, space && a.Shortcut("preview") == "Space")
}
