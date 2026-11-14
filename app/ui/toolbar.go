package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type toolbarState struct {
	FilterOpen    bool
	ThumbnailOpen bool
	SortOpen      bool
	ViewOpen      bool
	SearchOpen    bool
	TagSearch     string
}

var toolbarFamilies = []struct{ ID, Hex string }{
	{"red", "#E34848"}, {"coral", "#EE7667"}, {"orange", "#E48A39"}, {"amber", "#D5A234"},
	{"yellow", "#D4BC38"}, {"lime", "#A3B64D"}, {"green", "#539B68"}, {"mint", "#6CB69C"},
	{"teal", "#458C8F"}, {"cyan", "#50A5B9"}, {"sky", "#6BA6D4"}, {"blue", "#527ABD"},
	{"indigo", "#6565B0"}, {"violet", "#8D69B8"}, {"purple", "#A56CB4"}, {"magenta", "#BB6B9D"},
	{"pink", "#D587AA"}, {"rose", "#CB6A86"}, {"brown", "#886747"}, {"olive", "#85834E"},
	{"beige", "#C8BA96"}, {"white", "#FFFFFF"}, {"gray", "#939393"}, {"black", "#252525"},
}

// 所有搜索入口都先展开窄栏里的输入框，再请求聚焦。
func (a *App) FocusSearch() {
	a.Toolbar.SearchOpen = true
	a.Search.Focus()
	a.InvalidateWindows()
}

func (a *App) toolbar(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()
	left, right := float32(12), float32(12)
	if !a.SidebarVisible {
		left += bar.Left
	}
	if !a.InspectorVisible {
		right += bar.Right
	}
	width := a.toolbarWidth(c) - left - right
	compact := width < 720
	hint := displayShortcut(a.Shortcut("search"))
	if !a.Preview.Open && a.ActionShortcut(c, "search") {
		a.FocusSearch()
	}

	row := ui.Row(c.Key("toolbar")).Height(MainToolbarHeight).Shrink(0).FillWidth().Padding(12, right, 12, left).Gap(4).DragWindow().
		Material(toolbarMaterial(t)).BorderWidth(0, 0, 1, 0).BorderColor(warmOverlay(t, .09)).Role(ui.RoleToolbar).FocusGroup(ui.Horizontal).Label(a.T("browse_tools"))
	if bounds := row.LayoutBounds(); bounds.W > 0 {
		width = bounds.W - left - right
		compact = width < 720
	}
	row.Children(func() {
		if !a.SidebarVisible && toolbarIconButton(c.Key("toggle-sidebar"), a.T("toggle_sidebar"), "sidebar").Clicked() {
			a.SidebarVisible = true
		}
		ui.Row(c.Key("history")).Gap(0).Shrink(0).Children(func() {
			if toolbarIconButton(c.Key("history-back"), a.T("navigate_back"), "back").Width(26).Disabled(!a.CanNavigateBack()).Clicked() {
				a.NavigateBack()
			}
			if toolbarIconButton(c.Key("history-forward"), a.T("navigate_forward"), "forward").Width(26).Disabled(!a.CanNavigateForward()).Clicked() {
				a.NavigateForward()
			}
		})
		titleWidth := float32(160)
		if compact {
			titleWidth = 76
		}
		name := a.scopeName()
		count := len(a.Visible)
		if a.TagPage {
			count = len(a.Snapshot.Tags)
		}
		location := ui.Row(c.Key("location")).Width(titleWidth).MinWidth(0).Gap(8).Padding(0, 4)
		if !compact {
			location.Grow(1)
		}
		location.Children(func() {
			ui.Text(c.Key("location-title"), name).FontSize(22).Bold().TextColor(titleColor(t)).SingleLine().Shrink(1).MinWidth(0).Tooltip(name)
			label := a.T("result_count", toolbarNumber(count))
			counter := ui.Text(c.Key("location-count"), toolbarNumber(count)).FontSize(12).TextColor(t.TextMuted).SingleLine().Shrink(0)
			if strings.TrimSpace(a.Query.Search) != "" {
				counter.Label(label).Tooltip(label).Role(ui.RoleStatus)
			}
		})
		if width < 540 {
			search := toolbarIconButton(c.Key("compact-search"), a.T("search_assets"), "search")
			if search.Clicked() {
				a.Toolbar.SearchOpen = !a.Toolbar.SearchOpen
				a.Search.Focus()
			}
			frostedPopover(c.Key("search-popover"), search, &a.Toolbar.SearchOpen, func() { a.toolbarSearch(c, true, hint) }).Width(320)
		} else {
			a.Toolbar.SearchOpen = false
			a.toolbarSearch(c, !compact, hint)
		}
		if width >= 920 {
			a.Toolbar.ThumbnailOpen = false
			a.thumbnailSizeSlider(c.Key("inline-thumbnail-size"))
		} else {
			thumbnail := toolbarIconButton(c.Key("thumbnail-size-button"), a.T("thumbnail_size"), "thumbnail-size").Disabled(a.ViewMode == 2)
			if thumbnail.Clicked() {
				a.Toolbar.ThumbnailOpen = !a.Toolbar.ThumbnailOpen
				a.Toolbar.FilterOpen = false
				a.Toolbar.SortOpen = false
				a.Toolbar.ViewOpen = false
			}
			if a.ViewMode == 2 {
				a.Toolbar.ThumbnailOpen = false
			}
			frostedPopover(c.Key("thumbnail-size-popover"), thumbnail, &a.Toolbar.ThumbnailOpen, func() {
				ui.Column(c).Gap(10).Children(func() {
					ui.Text(c, a.T("thumbnail_size")).FontSize(12).Bold()
					a.thumbnailSizeSlider(c.Key("popover-thumbnail-size"))
				})
			}).Width(160)
		}
		filter := toolbarIconButton(c.Key("filter"), a.T("filter"), "filter")
		if len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) > 0 {
			filter.Background(t.Selection)
		}
		if filter.Clicked() {
			a.Toolbar.FilterOpen = !a.Toolbar.FilterOpen
			a.Toolbar.SortOpen = false
			a.Toolbar.ThumbnailOpen = false
			a.Toolbar.ViewOpen = false
		}
		frostedPopover(c.Key("filters"), filter, &a.Toolbar.FilterOpen, func() { a.filterPanel(c) }).Width(340).MaxHeight(560)
		sort := toolbarIconButton(c.Key("sort"), a.T("sort"), "sort").Disabled(a.Query.Scope == "random")
		if a.Query.Scope == "random" {
			sort.Tooltip(a.T("random_no_sort"))
			a.Toolbar.SortOpen = false
		}
		if sort.Clicked() {
			a.Toolbar.SortOpen = !a.Toolbar.SortOpen
			a.Toolbar.FilterOpen = false
			a.Toolbar.ThumbnailOpen = false
			a.Toolbar.ViewOpen = false
		}
		frostedPopover(c.Key("sorting"), sort, &a.Toolbar.SortOpen, func() { a.sortPanel(c) }).Width(220)
		if compact {
			view := toolbarIconButton(c.Key("view-menu"), a.T("view_mode"), []string{"masonry", "grid", "list"}[a.ViewMode])
			if view.Clicked() {
				a.Toolbar.ViewOpen = !a.Toolbar.ViewOpen
				a.Toolbar.FilterOpen = false
				a.Toolbar.SortOpen = false
				a.Toolbar.ThumbnailOpen = false
			}
			frostedPopover(c.Key("view-popover"), view, &a.Toolbar.ViewOpen, func() { a.viewSwitcher(c) }).Width(120)
		} else {
			a.Toolbar.ViewOpen = false
			a.viewSwitcher(c)
		}
		if toolbarIconButton(c.Key("preview"), a.T("preview"), "slideshow").Disabled(len(a.Visible) == 0 || a.TagPage).Clicked() {
			id := a.Visible[0].ID
			if selected := a.SelectedAssets(); len(selected) > 0 {
				id = selected[0].ID
			}
			a.ShowPreview(id, false)
		}
		button := toolbarButton(c.Key("import"), a.T("import")).Background(t.Accent).TextColor(t.AccentText).Disabled(a.Busy)
		if button.Pressed() {
			button.Background(t.AccentPressed)
		} else if button.Hovered() {
			button.Background(t.AccentHover)
		}
		button.Children(func() {
			appIcon(c, "import").Size(20, 20)
			if width >= 660 {
				ui.Text(c, a.T("import")).SingleLine().Bold()
			}
		})
		if width >= 660 {
			button.Width(96).Padding(0, 10).Gap(6)
		}
		button.Menu(func(m *ui.Menu) {
			if m.Item(a.T("import_files")).Chosen() {
				a.ChooseImport(false)
			}
			if m.Item(a.T("import_folder")).Chosen() {
				a.ChooseImport(true)
			}
		})
		if version := AvailableUpdateVersion(); version != "" {
			update := toolbarIconButton(c.Key("available-update"), a.T("update"), "import").Tooltip(a.T("update_to_version", version)).Background(t.Selection)
			if update.Clicked() {
				ShowAvailableUpdate()
			}
		}
		if toolbarIconButton(c.Key("toggle-inspector"), a.T("toggle_inspector"), "inspector").Clicked() {
			a.InspectorVisible = !a.InspectorVisible
		}
	})
}

func toolbarButton(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(32, 34).Shrink(0).Padding(0).Radius(8).Cursor(ui.CursorPointer)
	if button.Pressed() {
		button.Background(warmOverlay(t, .11))
	} else if button.Hovered() {
		button.Background(warmOverlay(t, .08))
	}
	return button
}

func toolbarIconButton(c *ui.Context, label, name string) ui.Element {
	return toolbarButton(c, label).Children(func() { appIcon(c, name).Size(20, 20) })
}

func (a *App) toolbarSearch(c *ui.Context, full bool, hint string) {
	t := c.Theme()
	field := ui.Row(c.Key("asset-search-field")).Height(34).Grow(1).MinWidth(166).MaxWidth(300).Gap(6).Padding(0, 8).Radius(8).Background(t.Surface.Alpha(.65)).Border(1, t.Border)
	field.Children(func() {
		appIcon(c, "search").Size(20, 20).Shrink(0)
		input := ui.TextInputBase(c.Key("asset-search"), &a.Query.Search).Bind(&a.Search).Search().Placeholder(a.T("search_assets")).Label(a.T("search_assets")).Tooltip(a.T("search_assets")).Grow(1).MinWidth(0).FillHeight().FocusRing(false)
		if input.Changed() {
			a.Refine()
		}
		if a.Query.Search != "" {
			if toolbarIconButton(c.Key("clear-search"), a.T("clear_search"), "close").Size(20, 24).Clicked() {
				a.Query.Search = ""
				a.Refine()
				a.Search.Focus()
			}
		} else if full && hint != "" {
			ui.Text(c, hint).FontSize(11).TextColor(t.TextMuted).SingleLine().Shrink(0).Padding(1, 5).Radius(4).Border(1, t.Border)
		}
	})
	if field.FocusWithin() {
		field.Border(1, t.Focus)
	}
}

func (a *App) viewSwitcher(c *ui.Context) {
	t := c.Theme()
	seg := ui.SegmentedBase(c.Key("view-mode"), &a.ViewMode, 3)
	seg.Track.Gap(2).Padding(2).Radius(8).Background(warmOverlay(t, .04)).Border(1, t.Border).Height(34).Shrink(0).Label(a.T("view_mode")).Children(func() {
		for i, item := range []struct{ ID, Label string }{{"masonry", "view_masonry"}, {"grid", "view_grid"}, {"list", "view_list"}} {
			part := seg.Segment(i).Size(28, 28).Padding(0).Center().Radius(4).Cursor(ui.CursorPointer).Label(a.T(item.Label)).Tooltip(a.T(item.Label))
			if a.ViewMode == i {
				part.Background(t.Background).Border(1, t.Border)
			} else if part.Hovered() {
				part.Background(warmOverlay(t, .08))
			}
			part.Children(func() { appIcon(c, item.ID).Size(20, 20) })
		}
	})
}

func (a *App) filterPanel(c *ui.Context) {
	ui.Column(c).Gap(12).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("filter")).Bold().Grow(1)
			if ui.Button(c, a.T("clear_filters")).Disabled(len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) == 0).Clicked() {
				a.Query.ColorFamilies = nil
				a.Query.TagIDs = nil
				a.Query.Extensions = nil
				a.Refine()
			}
		})
		ui.Scroll(c.Key("filter-scroll")).Height(440).Gap(12).Children(func() {
			ui.Text(c, a.T("filter_colors")).FontSize(12).Bold().TextColor(c.Theme().TextMuted)
			for start := 0; start < len(toolbarFamilies); start += 8 {
				ui.Row(c.Key(start)).Gap(6).Children(func() {
					for _, family := range toolbarFamilies[start:min(start+8, len(toolbarFamilies))] {
						chosen := slices.Contains(a.Query.ColorFamilies, family.ID)
						button := ui.Button(c.Key(family.ID), "").Label(a.T("color_"+family.ID)).Tooltip(a.T("color_"+family.ID)).Size(32, 32).Padding(4).Radius(16)
						if button.Clicked() {
							chosen = !chosen
							a.Query.ColorFamilies = toolbarToggle(a.Query.ColorFamilies, family.ID, chosen)
							a.Refine()
						}
						button.Role(ui.RoleCheckBox).Checked(chosen)
						if chosen {
							button.Border(2, c.Theme().Accent).Background(c.Theme().Selection)
						} else {
							button.Border(1, ui.Transparent)
						}
						button.Children(func() {
							swatch := ui.Hex(family.Hex)
							ui.Box(c).Size(22, 22).Radius(11).Background(swatch).Border(1, c.Theme().Border).Center().Children(func() {
								if chosen {
									ui.Image(c, toolbarColorCheck).Size(14, 14).PassThrough()
								}
							})
						})
					}
				})
			}
			ui.Divider(c)
			ui.Text(c, a.T("filter_tags")).FontSize(12).Bold().TextColor(c.Theme().TextMuted)
			ui.SearchField(c.Key("tag-search"), &a.Toolbar.TagSearch).Placeholder(a.T("search_tags")).Label(a.T("search_tags")).FillWidth()
			tags := slices.Clone(a.Snapshot.Tags)
			slices.SortFunc(tags, func(x, y model.Tag) int { return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)) })
			matched := 0
			for _, tag := range tags {
				if !strings.Contains(strings.ToLower(tag.Name), strings.ToLower(strings.TrimSpace(a.Toolbar.TagSearch))) {
					continue
				}
				matched++
				chosen := slices.Contains(a.Query.TagIDs, tag.ID)
				if ui.Checkbox(c.Key("filter-tag-"+tag.ID), &chosen, tag.Name).FillWidth().Changed() {
					a.Query.TagIDs = toolbarToggle(a.Query.TagIDs, tag.ID, chosen)
					a.Refine()
				}
			}
			if matched == 0 {
				ui.Text(c, a.T("no_tags")).TextColor(c.Theme().TextMuted).FontSize(12)
			}
			ui.Divider(c)
			ui.Text(c, a.T("filter_types")).FontSize(12).Bold().TextColor(c.Theme().TextMuted)
			types := a.toolbarExtensions()
			if len(types) == 0 {
				ui.Text(c, a.T("no_file_types")).TextColor(c.Theme().TextMuted).FontSize(12)
			}
			ui.Row(c).Wrap().Gap(6).Children(func() {
				for _, ext := range types {
					chosen := slices.Contains(a.Query.Extensions, ext)
					if ui.Toggle(c.Key("extension-"+ext), &chosen, strings.ToUpper(ext)).Changed() {
						a.Query.Extensions = toolbarToggle(a.Query.Extensions, ext, chosen)
						a.Refine()
					}
				}
			})
		})
	})
}

func (a *App) sortPanel(c *ui.Context) {
	ui.Column(c).Gap(8).Children(func() {
		ui.Text(c, a.T("sort")).Bold()
		for _, item := range []struct{ ID, Label string }{{"added", "sort_added"}, {"name", "sort_name"}, {"size", "sort_size"}} {
			chosen := a.Query.Sort == item.ID || a.Query.Sort == "" && item.ID == "added"
			row := ui.Button(c.Key(item.ID), a.T(item.Label)).FillWidth().Justify(ui.Start)
			if chosen {
				row.Background(c.Theme().Selection)
			}
			if row.Clicked() {
				a.changeSort(item.ID)
				a.Toolbar.SortOpen = false
			}
		}
		ui.Divider(c)
		descending := a.Query.Descending
		ui.Radio(c, &descending, false, a.T("sort_ascending"))
		ui.Radio(c, &descending, true, a.T("sort_descending"))
		if descending != a.Query.Descending {
			a.Query.Descending = descending
			a.Refine()
		}
	})
}

func (a *App) changeSort(field string) {
	if a.Query.Scope == "random" {
		return
	}
	if a.Query.Sort == field || a.Query.Sort == "" && field == "added" {
		a.Query.Descending = !a.Query.Descending
	} else {
		a.Query.Sort = field
		a.Query.Descending = field == "added"
	}
	a.Refine()
}

func (a *App) toolbarExtensions() []string {
	unique := map[string]bool{}
	for _, asset := range a.Snapshot.Assets {
		ext := strings.TrimPrefix(strings.ToLower(asset.Extension), ".")
		if ext != "" {
			unique[ext] = true
		}
	}
	out := make([]string, 0, len(unique))
	for ext := range unique {
		out = append(out, ext)
	}
	slices.Sort(out)
	return out
}

func toolbarToggle(values []string, value string, on bool) []string {
	at := slices.Index(values, value)
	if on {
		if at < 0 {
			return append(values, value)
		}
		return values
	}
	if at >= 0 {
		return slices.Delete(values, at, at+1)
	}
	return values
}

func toolbarNumber(n int) string {
	text := strconv.Itoa(n)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}

func (a *App) toolbarWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	if width >= 900 {
		if a.SidebarVisible {
			width -= a.SidebarWidth + 1
		}
		if a.InspectorVisible {
			width -= 280
		}
	}
	return width
}

func (a *App) thumbnailSizeSlider(c *ui.Context) {
	ui.Row(c.Key("thumbnail-size-control")).Width(120).Shrink(0).Gap(4).AlignItems(ui.Center).Children(func() {
		disabled := a.ViewMode == 2
		if toolbarIconButton(c.Key("thumbnail-smaller"), a.T("thumbnail_smaller"), "thumbnail-size").Size(24, 28).Disabled(disabled || a.thumbnailScale() <= .5).Clicked() {
			a.ChangeThumbnailSize(-.1)
		}
		size := float64(a.thumbnailScale())
		if ui.Slider(c.Key("thumbnail-size"), &size, .5, 2).Label(a.T("thumbnail_size")).Tooltip(a.T("thumbnail_size")).Grow(1).MinWidth(0).Basis(0).Disabled(disabled).Changed() {
			a.SetThumbnailSize(float32(size))
		}
		if toolbarIconButton(c.Key("thumbnail-larger"), a.T("thumbnail_larger"), "thumbnail-size").Size(28, 28).Disabled(disabled || a.thumbnailScale() >= 2).Clicked() {
			a.ChangeThumbnailSize(.1)
		}
	})
}
