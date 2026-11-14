package ui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type inspectorEdit struct {
	ID, Name, Note, Source  string
	Rename, Details, Failed bool
}

type inspectorState struct {
	LibraryID                                   string
	AssetID, Title, SourceTitle                 string
	Note, SourceNote, SourceURL, SavedSourceURL string
	TagsOpen, FoldersOpen                       bool
	Pending                                     []inspectorEdit
	TagSearch, FolderSearch                     string
}

var inspectorEXIFKeys = []string{
	"fileCreatedAt", "fileModifiedAt", "contentCreatedAt", "resolution",
	"colorModel", "profileName", "cameraMake", "cameraModel", "lensModel", "exposureTime",
	"focalLength", "isoSpeedRatings", "flashFired", "fNumber", "exposureProgram", "meteringMode",
	"whiteBalance", "creator",
}

func (a *App) inspector(c *ui.Context) {
	if a.Library != nil && a.Inspector.LibraryID != a.Library.Manifest.LibraryID {
		a.Inspector = inspectorState{LibraryID: a.Library.Manifest.LibraryID}
	}
	selected := a.SelectedAssets()
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		if len(selected) == 0 {
			a.saveInspectorEdits()
			a.Inspector.AssetID = ""
			a.inspectorScope(c)
			return
		}
		ui.Scroll(c.Key("inspector_content")).Grow(1).Padding(16).Gap(16).Children(func() {
			if len(selected) == 1 {
				a.inspectorAsset(c, selected[0])
			} else {
				a.saveInspectorEdits()
				a.Inspector.AssetID = ""
				ui.Text(c, a.T("selected_assets", len(selected))).FontSize(18).Bold().TextColor(titleColor(c.Theme()))
			}
			a.inspectorTags(c, selected)
			a.inspectorFolders(c, selected)
		})
	})
}

func (a *App) inspectorAsset(c *ui.Context, asset model.Asset) {
	width, height := assetDisplaySize(asset)
	if a.Inspector.AssetID != asset.ID || a.Inspector.SourceTitle != asset.Name {
		if a.Inspector.AssetID != asset.ID {
			a.saveInspectorEdits()
		}
		a.Inspector.AssetID, a.Inspector.Title, a.Inspector.SourceTitle = asset.ID, asset.Name, asset.Name
		a.Inspector.Note, a.Inspector.SourceNote = asset.Note, asset.Note
		a.Inspector.SourceURL, a.Inspector.SavedSourceURL = asset.SourceURL, asset.SourceURL
		for _, edit := range a.Inspector.Pending {
			if edit.ID == asset.ID {
				if edit.Rename {
					a.Inspector.Title = edit.Name
				}
				if edit.Details {
					a.Inspector.Note = edit.Note
					a.Inspector.SourceURL = edit.Source
				}
			}
		}
	}
	preview := ui.Column(c.Key("inspector_preview")).Height(176).Radius(8).
		Background(c.Theme().Background).Cursor(ui.CursorPointer).Center().Clip().Children(func() {
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
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, a.T("title")).FontSize(12).TextColor(c.Theme().TextMuted)
		input := ui.TextInput(c.Key("asset_title_"+asset.ID), &a.Inspector.Title).
			Label(a.T("title")).Disabled(a.Busy)
		if input.Submitted() || input.FocusLeft() {
			a.saveInspectorTitle()
		}
	})
	ui.Column(c).Gap(8).Children(func() {
		a.inspectorDetail(c, "file_type", strings.ToUpper(asset.Extension))
		a.inspectorDetail(c, "dimensions", fmt.Sprintf("%d × %d", width, height))
		a.inspectorDetail(c, "file_size", fileSize(asset.Size))
		a.inspectorDetail(c, "date_added", asset.CreatedAt.Local().Format("2006-01-02 15:04"))
	})
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, a.T("asset_note")).FontSize(12).TextColor(c.Theme().TextMuted)
		note := ui.TextArea(c.Key("asset-note-"+asset.ID), &a.Inspector.Note).Label(a.T("asset_note")).Height(88).FillWidth().Disabled(a.Busy)
		if note.FocusLeft() {
			a.saveInspectorDetails()
		}
	})
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, a.T("source_url")).FontSize(12).TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			source := ui.TextInput(c.Key("asset-source-"+asset.ID), &a.Inspector.SourceURL).Label(a.T("source_url")).Grow(1).MinWidth(0).Disabled(a.Busy)
			if source.Submitted() || source.FocusLeft() {
				a.saveInspectorDetails()
			}
			if strings.TrimSpace(a.Inspector.SourceURL) != "" {
				if inspectorIconButton(c.Key("open-source"), a.T("open_source_url"), "external-link").Clicked() {
					c.OpenURLThen(strings.TrimSpace(a.Inspector.SourceURL), a.ShowError)
				}
			}
		})
	})

	if hasInspectorEXIF(asset.EXIF) {
		ui.Column(c).Gap(8).Children(func() {
			ui.Divider(c)
			ui.Text(c, a.T("exif_metadata")).FontSize(12).Bold().TextColor(titleColor(c.Theme()))
			for _, key := range inspectorEXIFKeys {
				if value := inspectorEXIFValue(asset.EXIF, key); value != "" {
					if key == "flashFired" && (value == "true" || value == "false") {
						value = a.T("exif_flash_" + value)
					}
					a.inspectorDetail(c, "exif_"+key, value)
				}
			}
		})
	}
}

func (a *App) saveInspectorTitle() { a.saveInspectorEdits() }

func (a *App) inspectorDetail(c *ui.Context, key, value string) {
	ui.Row(c).AlignItems(ui.Start).Gap(8).Children(func() {
		ui.Text(c, a.T(key)).FontSize(12).TextColor(c.Theme().TextMuted).Width(112).Shrink(0).NoWrap()
		ui.Text(c, value).FontSize(12).Grow(1).MinWidth(0).Selectable()
	})
}

func hasInspectorEXIF(values map[string]string) bool {
	for _, key := range inspectorEXIFKeys {
		if inspectorEXIFValue(values, key) != "" {
			return true
		}
	}
	return false
}

func inspectorEXIFValue(values map[string]string, key string) string {
	if key != "resolution" {
		return values[key]
	}
	width, height := values["dpiWidth"], values["dpiHeight"]
	if width == "" {
		width = height
	}
	if width == "" {
		return ""
	}
	if height == "" || width == height {
		return width + " dpi"
	}
	return width + " × " + height + " dpi"
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
			ui.Text(c, a.T(plural)).FontSize(12).Bold().TextColor(titleColor(c.Theme()))
			ui.Spacer(c)
			add := inspectorIconButton(c.Key("inspector_add_"+kind), a.T("add_"+plural), "plus").Disabled(a.Busy)
			if add.Clicked() {
				*open = !*open
			}
			frostedPopover(c.Key("inspector_"+kind+"_picker"), add, open, func() {
				ui.Column(c).Width(244).Gap(8).Children(func() {
					ui.SearchField(c, search).Label(a.T("search_" + plural)).Placeholder(a.T("search_" + plural))
					ui.Scroll(c).MaxHeight(220).Gap(4).Children(func() {
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
			ui.Row(c.Key("member_" + item.ID)).Gap(8).Children(func() {
				ui.Text(c, item.Label).FontSize(12).Grow(1).MinWidth(0)
				if len(selected) > 1 {
					ui.Textf(c, "%d/%d", count, len(selected)).FontSize(12).TextColor(c.Theme().TextMuted)
				}
				if inspectorIconButton(c, a.T("remove_"+kind, item.Name), "minus").Disabled(a.Busy).Clicked() {
					set(item.ID, false)
				}
			})
		}
		if !found {
			key := "no_" + plural
			if !folder && len(selected) > 1 {
				key = "selected_no_tags"
			}
			ui.Text(c, a.T(key)).FontSize(12).TextColor(c.Theme().TextMuted)
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

func (a *App) inspectorScope(c *ui.Context) {
	name, count, size := a.scopeSummary()
	ui.Column(c).FillWidth().Padding(16).Gap(16).Children(func() {
		ui.Text(c, name).FontSize(18).Bold().TextColor(titleColor(c.Theme())).FillWidth()
		ui.Divider(c)
		ui.Text(c, a.T("scope_information")).FontSize(12).Bold().TextColor(c.Theme().TextMuted)
		a.inspectorDetail(c, "scope_asset_count", sidebarCount(count))
		a.inspectorDetail(c, "scope_storage", fileSize(size))
		if !a.TagPage && len(a.Visible) != count {
			ui.Text(c, a.T("scope_filtered_count", toolbarNumber(len(a.Visible)))).FontSize(12).TextColor(c.Theme().TextMuted)
		}
	})
}

func (a *App) inspectorDraft() (inspectorEdit, bool) {
	state := a.Inspector
	name, source := strings.TrimSpace(state.Title), strings.TrimSpace(state.SourceURL)
	rename := name != "" && name != state.SourceTitle
	details := state.Note != state.SourceNote || source != state.SavedSourceURL
	return inspectorEdit{ID: state.AssetID, Name: name, Note: state.Note, Source: source, Rename: rename, Details: details}, state.AssetID != "" && (rename || details)
}

func (a *App) InspectorEditsPending() bool {
	_, dirty := a.inspectorDraft()
	return dirty || len(a.Inspector.Pending) > 0
}

func (a *App) InspectorEditsFailed() bool {
	return slices.ContainsFunc(a.Inspector.Pending, func(edit inspectorEdit) bool { return edit.Failed })
}

func (a *App) saveInspectorEdits() {
	edit, dirty := a.inspectorDraft()
	if !dirty || a.Library == nil {
		return
	}
	found := false
	for i, pending := range a.Inspector.Pending {
		if pending.ID == edit.ID {
			a.Inspector.Pending[i] = edit
			found = true
			break
		}
	}
	if !found {
		a.Inspector.Pending = append(a.Inspector.Pending, edit)
	}
	a.ContinueInspectorEdits()
}

// 锁屏和退出前调用；保存失败后再次明确调用才重试。
func (a *App) FlushInspectorEdits() {
	a.saveInspectorEdits()
	for i := range a.Inspector.Pending {
		a.Inspector.Pending[i].Failed = false
	}
	a.ContinueInspectorEdits()
}

// 每帧推进已失焦的草稿，不碰正在输入的内容，也不循环重试失败写入。
func (a *App) ContinueInspectorEdits() {
	if a.Busy || a.Library == nil {
		return
	}
	for _, edit := range a.Inspector.Pending {
		if !edit.Failed {
			a.runInspectorEdit(edit)
			return
		}
	}
}

func (a *App) runInspectorEdit(edit inspectorEdit) {
	lib := a.Library
	a.Run("update_asset_details", func(ctx context.Context) error {
		err := func() error {
			if edit.Rename {
				if err := lib.RenameAsset(ctx, edit.ID, edit.Name); err != nil {
					return err
				}
			}
			if edit.Details {
				return lib.UpdateAssetDetails(ctx, edit.ID, edit.Note, edit.Source)
			}
			return nil
		}()
		a.Window.Update(func() {
			if err != nil {
				for i, pending := range a.Inspector.Pending {
					if pending.ID == edit.ID {
						a.Inspector.Pending[i].Failed = true
					}
				}
				return
			}
			a.Inspector.Pending = slices.DeleteFunc(a.Inspector.Pending, func(pending inspectorEdit) bool { return pending.ID == edit.ID })
			if a.Inspector.AssetID == edit.ID {
				if edit.Rename {
					a.Inspector.SourceTitle = edit.Name
				}
				if edit.Details {
					a.Inspector.SourceNote = edit.Note
					a.Inspector.SavedSourceURL = edit.Source
				}
			}
		})
		return err
	})
}

func (a *App) saveInspectorDetails() { a.saveInspectorEdits() }

func inspectorIconButton(c *ui.Context, label, icon string) ui.Element {
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(28, 28).Padding(0).Radius(8).Cursor(ui.CursorPointer)
	if button.Pressed() {
		button.Background(warmOverlay(c.Theme(), .11))
	} else if button.Hovered() {
		button.Background(warmOverlay(c.Theme(), .06))
	}
	button.TextColor(c.Theme().TextMuted)
	if button.Hovered() {
		button.TextColor(c.Theme().Text)
	}
	button.Children(func() {
		if momentoSymbols[icon] != nil {
			appSymbol(c, icon).Size(14, 14)
		} else {
			appIcon(c, icon).Size(20, 20)
		}
	})
	return button
}
