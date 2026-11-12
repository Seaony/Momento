package ui

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type inspectorState struct {
	LibraryID                   string
	AssetID, Title, SourceTitle string
	TagsOpen, FoldersOpen       bool
	TagSearch, FolderSearch     string
}

var inspectorEXIFKeys = []string{
	"fileCreatedAt", "fileModifiedAt", "contentCreatedAt", "pixelWidth", "pixelHeight", "dpiWidth", "dpiHeight",
	"colorModel", "profileName", "cameraMake", "cameraModel", "lensModel", "exposureTime",
	"focalLength", "isoSpeedRatings", "flashFired", "fNumber", "exposureProgram", "meteringMode",
	"whiteBalance", "creator",
}

func (a *App) inspector(c *ui.Context) {
	if a.Library != nil && a.Inspector.LibraryID != a.Library.Manifest.LibraryID {
		a.Inspector = inspectorState{LibraryID: a.Library.Manifest.LibraryID}
	}
	selected := a.SelectedAssets()
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Height(60).Shrink(0).Padding(16).Children(func() {
			ui.Text(c, a.T("inspector")).Bold()
			ui.Spacer(c)
			if ui.Button(c, "×").Label(a.T("close_inspector")).Size(28, 28).Clicked() {
				a.InspectorVisible = false
			}
		})
		ui.Divider(c)
		if len(selected) == 0 {
			a.saveInspectorTitle()
			a.Inspector.AssetID = ""
			ui.Column(c).Grow(1).Center().Padding(24).Gap(8).Children(func() {
				ui.Text(c, a.T("no_selection")).TextColor(c.Theme().TextMuted)
			})
			return
		}
		ui.Scroll(c.Key("inspector_content")).Grow(1).Padding(16).Gap(18).Children(func() {
			if len(selected) == 1 {
				a.inspectorAsset(c, selected[0])
			} else {
				a.saveInspectorTitle()
				a.Inspector.AssetID = ""
				ui.Text(c, a.T("selected_assets", len(selected))).FontSize(18).Bold()
			}
			a.inspectorTags(c, selected)
			a.inspectorFolders(c, selected)
		})
	})
}

func (a *App) inspectorAsset(c *ui.Context, asset model.Asset) {
	if a.Inspector.AssetID != asset.ID || a.Inspector.SourceTitle != asset.Name {
		if a.Inspector.AssetID != asset.ID {
			a.saveInspectorTitle()
		}
		a.Inspector.AssetID, a.Inspector.Title, a.Inspector.SourceTitle = asset.ID, asset.Name, asset.Name
	}
	preview := ui.Column(c.Key("inspector_preview")).Height(176).Radius(8).
		Background(c.Theme().Background).Center().Clip().Children(func() {
		if bitmap := a.Bitmap(asset); bitmap != nil {
			ui.Image(c, bitmap).Fill().Fit(ui.Contain).Label(a.T("image_preview"))
		} else {
			if a.Library != nil && a.Thumbnails != nil {
				if err := a.Thumbnails.Error(a.Library.Root, asset); err != nil {
					ui.Text(c, a.T("preview_failed")).FontSize(12).TextColor(c.Theme().Danger).Tooltip(a.Texts.Error(err))
					return
				}
			}
			ui.Text(c, a.T("loading_preview")).FontSize(12).TextColor(c.Theme().TextMuted)
		}
	})
	if preview.Clicked() {
		a.ShowPreview(asset.ID, false)
	}
	if len(asset.Colors) > 0 {
		ui.Row(c).Gap(4).Children(func() {
			for _, color := range asset.Colors[:min(8, len(asset.Colors))] {
				hex := color.Hex
				if ui.Button(c.Key("swatch_"+hex), "").Label(a.T("copy_color", hex)).
					Grow(1).MinWidth(0).Height(28).Padding(0).Radius(4).Background(ui.Hex(hex)).
					Tooltip(hex).Clicked() {
					c.WriteClipboardThen(hex, func(err error) {
						if err != nil {
							a.ShowError(err)
						} else {
							a.Notify(a.T("copied_color", hex))
						}
					})
				}
			}
		})
	}
	ui.Column(c).Gap(6).Children(func() {
		ui.Text(c, a.T("title")).FontSize(12).TextColor(c.Theme().TextMuted)
		input := ui.TextInput(c.Key("asset_title_"+asset.ID), &a.Inspector.Title).
			Label(a.T("title")).Disabled(a.Busy)
		if input.Submitted() || input.FocusLeft() {
			a.saveInspectorTitle()
		}
	})
	ui.Column(c).Gap(10).Children(func() {
		a.inspectorDetail(c, "file_type", strings.ToUpper(asset.Extension))
		a.inspectorDetail(c, "dimensions", fmt.Sprintf("%d × %d", asset.Width, asset.Height))
		a.inspectorDetail(c, "file_size", fileSize(asset.Size))
		a.inspectorDetail(c, "date_added", asset.CreatedAt.Local().Format("2006-01-02 15:04"))
		if asset.SourceURL != "" {
			ui.Column(c).Gap(4).Children(func() {
				ui.Text(c, a.T("source_url")).FontSize(12).TextColor(c.Theme().TextMuted)
				label := asset.SourceURL
				if parsed, err := url.Parse(asset.SourceURL); err == nil && parsed.Host != "" {
					label = parsed.Host
				}
				if ui.Link(c, label, "").FontSize(12).Tooltip(asset.SourceURL).Clicked() {
					c.OpenURLThen(asset.SourceURL, func(err error) { a.ShowError(err) })
				}
			})
		}
	})
	if hasInspectorEXIF(asset.EXIF) {
		ui.Column(c).Gap(10).Children(func() {
			ui.Divider(c)
			ui.Text(c, a.T("exif_metadata")).FontSize(12).Bold()
			for _, key := range inspectorEXIFKeys {
				if value := asset.EXIF[key]; value != "" {
					if key == "flashFired" && (value == "true" || value == "false") {
						value = a.T("exif_flash_" + value)
					}
					a.inspectorDetail(c, "exif_"+key, value)
				}
			}
		})
	}
}

func (a *App) saveInspectorTitle() {
	name := strings.TrimSpace(a.Inspector.Title)
	if a.Inspector.AssetID == "" || name == "" || name == a.Inspector.SourceTitle || a.Busy || a.Library == nil {
		return
	}
	lib, id := a.Library, a.Inspector.AssetID
	a.Run("rename_asset", func(ctx context.Context) error { return lib.RenameAsset(ctx, id, name) })
}

func (a *App) inspectorDetail(c *ui.Context, key, value string) {
	ui.Row(c).AlignItems(ui.Start).Gap(8).Children(func() {
		ui.Text(c, a.T(key)).FontSize(12).TextColor(c.Theme().TextMuted).Width(84).Shrink(0)
		ui.Text(c, value).FontSize(12).Grow(1).MinWidth(0).Selectable()
	})
}

func hasInspectorEXIF(values map[string]string) bool {
	for _, key := range inspectorEXIFKeys {
		if values[key] != "" {
			return true
		}
	}
	return false
}

type inspectorMembership struct {
	ID, Name, Label string
}

func (a *App) inspectorTags(c *ui.Context, selected []model.Asset) {
	items := make([]inspectorMembership, 0, len(a.Snapshot.Tags))
	for _, tag := range a.Snapshot.Tags {
		items = append(items, inspectorMembership{tag.ID, tag.Name, tag.Name})
	}
	a.inspectorMemberships(c, selected, items, false)
}

func (a *App) inspectorFolders(c *ui.Context, selected []model.Asset) {
	items := make([]inspectorMembership, 0, len(a.Snapshot.Folders))
	for _, folder := range a.Snapshot.Folders {
		items = append(items, inspectorMembership{folder.ID, folder.Name, inspectorFolderPath(a.Snapshot.Folders, folder.ID)})
	}
	a.inspectorMemberships(c, selected, items, true)
}

func (a *App) inspectorMemberships(c *ui.Context, selected []model.Asset, items []inspectorMembership, folder bool) {
	ids := inspectorIDs(selected)
	kind, plural := "tag", "tags"
	open, search := &a.Inspector.TagsOpen, &a.Inspector.TagSearch
	if folder {
		kind, plural = "folder", "folders"
		open, search = &a.Inspector.FoldersOpen, &a.Inspector.FolderSearch
	}
	set := func(id string, add bool) {
		if folder {
			a.SetFolders(ids, []string{id}, add)
		} else {
			a.SetTags(ids, []string{id}, add)
		}
	}
	ui.Column(c.Key("inspector_" + plural)).Gap(8).Children(func() {
		ui.Divider(c)
		ui.Row(c).Children(func() {
			ui.Text(c, a.T(plural)).FontSize(12).Bold()
			ui.Spacer(c)
			add := ui.Button(c.Key("inspector_add_"+kind), "+").Label(a.T("add_"+plural)).Size(28, 28).Disabled(a.Busy)
			if add.Clicked() {
				*open = !*open
			}
			ui.Popover(c.Key("inspector_"+kind+"_picker"), add, open, func() {
				ui.Column(c).Width(244).Gap(8).Children(func() {
					ui.SearchField(c, search).Label(a.T("search_" + plural)).Placeholder(a.T("search_" + plural))
					ui.Scroll(c).Height(220).Gap(4).Children(func() {
						query := strings.ToLower(strings.TrimSpace(*search))
						for _, item := range items {
							if query != "" && !strings.Contains(strings.ToLower(item.Label), query) {
								continue
							}
							count := inspectorMembershipCount(selected, item.ID, folder)
							all := count == len(selected)
							label := item.Label
							if count > 0 && !all {
								label += fmt.Sprintf(" (%d/%d)", count, len(selected))
							}
							if ui.Checkbox(c.Key(item.ID), &all, label).Disabled(a.Busy).Changed() {
								set(item.ID, all)
							}
						}
						if len(items) == 0 {
							ui.Text(c, a.T("no_"+plural)).FontSize(12).TextColor(c.Theme().TextMuted)
						}
					})
					if ui.Button(c, a.T("create_"+kind)).Disabled(a.Busy).Clicked() {
						*open = false
						lib := a.Library
						a.Ask(a.T("create_"+kind), *search, func(name string) {
							a.Run("create_asset_"+kind, func(ctx context.Context) error {
								if folder {
									item, err := lib.CreateFolder(ctx, name, "")
									if err != nil {
										return err
									}
									return lib.SetAssetFolders(ctx, ids, []string{item.ID}, true)
								}
								item, err := lib.CreateTag(ctx, name)
								if err != nil {
									return err
								}
								return lib.SetAssetTags(ctx, ids, []string{item.ID}, true)
							})
						})
					}
				})
			})
		})
		found := false
		for _, item := range items {
			count := inspectorMembershipCount(selected, item.ID, folder)
			if count == 0 {
				continue
			}
			found = true
			ui.Row(c.Key("member_" + item.ID)).Gap(4).Children(func() {
				ui.Text(c, item.Label).FontSize(12).Grow(1).MinWidth(0)
				if len(selected) > 1 {
					ui.Textf(c, "%d/%d", count, len(selected)).FontSize(12).TextColor(c.Theme().TextMuted)
				}
				if ui.Button(c, "−").Label(a.T("remove_"+kind, item.Name)).Size(28, 28).Disabled(a.Busy).Clicked() {
					set(item.ID, false)
				}
			})
		}
		if !found {
			ui.Text(c, a.T("no_"+plural)).FontSize(12).TextColor(c.Theme().TextMuted)
		}
	})
}

func inspectorIDs(selected []model.Asset) []string {
	ids := make([]string, len(selected))
	for i, asset := range selected {
		ids[i] = asset.ID
	}
	return ids
}

func inspectorMembershipCount(selected []model.Asset, id string, folder bool) int {
	count := 0
	for _, asset := range selected {
		members := asset.TagIDs
		if folder {
			members = asset.FolderIDs
		}
		if slices.Contains(members, id) {
			count++
		}
	}
	return count
}

func inspectorFolderPath(folders []model.Folder, id string) string {
	var parts []string
	for id != "" {
		found := false
		for _, folder := range folders {
			if folder.ID == id {
				parts = append(parts, folder.Name)
				found = true
				id = folder.ParentID
				break
			}
		}
		if !found {
			break
		}
	}
	slices.Reverse(parts)
	return strings.Join(parts, " / ")
}
