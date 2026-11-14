package ui

import (
	"slices"
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
	p := paletteFor(t)
	bar := c.TitleBar()
	left := space12 + max(float32(0), bar.Left-a.SidebarShownWidth())
	right := space12 + max(float32(0), bar.Right-a.InspectorShownWidth())
	width := a.toolbarWidth(c) - left - right
	hint := displayShortcut(a.Shortcut("search"))
	if !a.PreviewOpen() && a.ActionShortcut(c, "search") {
		a.FocusSearch()
	}
	row := ui.Row(c.Key("toolbar")).Height(MainToolbarHeight).Shrink(0).FillWidth().Padding(space12, right, space12, left).Gap(space4).DragWindow().
		Material(toolbarMaterial(t)).BorderWidth(0, 0, 1, 0).BorderColor(p.BorderSubtle).Role(ui.RoleToolbar).FocusGroup(ui.Horizontal).Label(a.T("browse_tools"))
	if bounds := row.LayoutBounds(); bounds.W > 0 {
		width = bounds.W - left - right
	}
	row.Children(func() {
		if !a.SidebarVisible {
			button := toolbarIconButton(c.Key("toggle-sidebar"), a.T("toggle_sidebar"), "sidebar").
				Transition(ui.ElementTransition{Animation: &motionColor, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
			if button.Clicked() {
				a.SidebarVisible = true
			}
		}
		if width < 500 {
			search := toolbarIconButton(c.Key("compact-search"), a.T("search_assets"), "search")
			if search.Clicked() {
				a.Toolbar.SearchOpen = !a.Toolbar.SearchOpen
				a.Search.Focus()
			}
			frostedPopover(c.Key("search-popover"), search, &a.Toolbar.SearchOpen, func() { a.toolbarSearch(c, hint) }).Width(360)
		} else {
			a.Toolbar.SearchOpen = false
			a.toolbarSearch(c, hint)
		}
		ui.Spacer(c)
		if width >= 800 {
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
				ui.Column(c).Gap(space12).Children(func() {
					ui.Text(c, a.T("thumbnail_size")).FontSize(fontSecondary).Bold()
					a.thumbnailSizeSlider(c.Key("popover-thumbnail-size"))
				})
			}).Width(160)
		}
		filter := toolbarIconButton(c.Key("filter"), a.T("filter"), "filter", len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) > 0)
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
		if width < 360 {
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
		if version := AvailableUpdateVersion(); version != "" {
			update := toolbarIconButton(c.Key("available-update"), a.T("update"), "import", true).Tooltip(a.T("update_to_version", version))
			if update.Clicked() {
				ShowAvailableUpdate()
			}
		}
		button := ui.ButtonBase(c.Key("import")).Label(a.T("import")).Tooltip(a.T("import")).MinWidth(space32).Height(34).Shrink(0).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer).Disabled(a.Busy).DimDisabled(false)
		background, text := p.Accent, p.OnAccent
		if a.Busy {
			background, text = p.Raised, p.TextSecondary
			button.Border(1, p.BorderSubtle)
		} else if button.Pressed() {
			background = p.AccentPressed
		} else if button.Hovered() {
			background = p.AccentHover
		}
		animateColors(button, background, text)
		button.Children(func() {
			appIcon(c, "import")
			if width >= 660 {
				ui.Text(c, a.T("import")).FontSize(fontBody).SingleLine().Bold()
			}
		})
		if width >= 660 {
			button.Padding(0, space12).Gap(space8)
		}
		button.Menu(func(m *ui.Menu) {
			if m.Item(a.T("import_files")).Chosen() {
				a.ChooseImport(false)
			}
			if m.Item(a.T("import_folder")).Chosen() {
				a.ChooseImport(true)
			}
		})
	})
}

func toolbarIconButton(c *ui.Context, label, name string, selected ...bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(32, 34).Shrink(0).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer)
	background, text := ui.Transparent, p.TextSecondary
	if len(selected) > 0 && selected[0] {
		background, text = p.Subtle, p.Text
	}
	if button.Pressed() {
		background, text = p.Pressed, p.Text
	} else if button.Hovered() {
		background, text = p.Hover, p.Text
	}
	animateColors(button, background, text)
	return button.Children(func() {
		if momentoSymbols[name] != nil {
			appSymbol(c, name)
		} else {
			appIcon(c, name)
		}
	})
}

func (a *App) toolbarSearch(c *ui.Context, hint string) {
	p := paletteFor(c.Theme())
	field := ui.Row(c.Key("asset-search-field")).Height(34).Grow(1).MinWidth(220).MaxWidth(360).Gap(space8).Padding(0, space8).Radius(radiusControl).Border(1, p.Border)
	field.Children(func() {
		appIcon(c, "search").TextColor(p.TextSecondary).Shrink(0)
		input := ui.TextInputBase(c.Key("asset-search"), &a.Query.Search).Bind(&a.Search).Search().Placeholder(a.T("search_assets")).Label(a.T("search_assets")).Tooltip(a.T("search_assets")).Grow(1).MinWidth(0).FillHeight().FontSize(fontBody).FocusRing(false)
		if input.Changed() {
			a.Refine()
		}
		if a.Query.Search != "" {
			if toolbarIconButton(c.Key("clear-search"), a.T("clear_search"), "close").Size(20, 24).Clicked() {
				a.Query.Search = ""
				a.Refine()
				a.Search.Focus()
			}
		} else if hint != "" {
			ui.Text(c, hint).Font(monoFont(c)).FontSize(fontNumber).TextColor(p.TextSecondary).SingleLine().Shrink(0).Padding(0, space4).Radius(radiusSmall).Background(p.Subtle).Border(1, p.BorderSubtle)
		}
	})
	border := p.Border
	if field.FocusWithin() {
		border = p.Focus
	}
	field.Border(1, field.AnimateColor("field-border", border, motionColor))
	animateColors(field, p.Raised, p.Text)
}

func (a *App) viewSwitcher(c *ui.Context) {
	p := paletteFor(c.Theme())
	seg := ui.SegmentedBase(c.Key("view-mode"), &a.ViewMode, 3)
	items := make([]ui.Element, 3)
	seg.Track.Gap(space4).Padding(space4).Radius(radiusControl).Background(p.Subtle).Border(1, p.BorderSubtle).Height(space40).Shrink(0).Label(a.T("view_mode")).Children(func() {
		for i, item := range []struct{ ID, Label string }{{"masonry", "view_masonry"}, {"grid", "view_grid"}, {"list", "view_list"}} {
			part := seg.Segment(i).Size(28, 28).Padding(0).Center().Radius(radiusSmall).Cursor(ui.CursorPointer).Label(a.T(item.Label)).Tooltip(a.T(item.Label))
			styleSegment(c, part, a.ViewMode == i)
			items[i] = part
			part.Children(func() { appIcon(c, item.ID) })
		}
	})
	slidingSegmentTrack(c, seg.Track, items, a.ViewMode, space4, space4)
}

func (a *App) filterPanel(c *ui.Context) {
	p := paletteFor(c.Theme())
	ui.Column(c).Gap(space12).Children(func() {
		ui.Row(c).Children(func() {
			ui.Text(c, a.T("filter")).FontSize(fontBody).Bold().Grow(1)
			clear := ui.ButtonBase(c.Key("clear-filters")).Height(28).Padding(0, space8).Radius(radiusControl).Cursor(ui.CursorPointer).Label(a.T("clear_filters")).Disabled(len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) == 0)
			background, text := ui.Transparent, p.TextSecondary
			if clear.Hovered() {
				background, text = p.Hover, p.Text
			}
			if clear.Pressed() {
				background = p.Pressed
			}
			animateColors(clear, background, text)
			clear.Children(func() { ui.Text(c, a.T("clear_filters")).SingleLine() })
			if clear.Clicked() {
				a.Query.ColorFamilies = nil
				a.Query.TagIDs = nil
				a.Query.Extensions = nil
				a.Refine()
			}
		})
		ui.Scroll(c.Key("filter-scroll")).Height(440).Gap(space12).Children(func() {
			ui.Text(c, a.T("filter_colors")).FontSize(fontGroup).FontWeight(600).LetterSpacing(.6).TextColor(p.TextSecondary)
			for start := 0; start < len(toolbarFamilies); start += 8 {
				ui.Row(c.Key(start)).Gap(space8).Children(func() {
					for _, family := range toolbarFamilies[start:min(start+8, len(toolbarFamilies))] {
						chosen := slices.Contains(a.Query.ColorFamilies, family.ID)
						button := ui.Button(c.Key(family.ID), "").Label(a.T("color_"+family.ID)).Tooltip(a.T("color_"+family.ID)).Size(32, 32).Padding(space4).Radius(16)
						if button.Clicked() {
							chosen = !chosen
							a.Query.ColorFamilies = toolbarToggle(a.Query.ColorFamilies, family.ID, chosen)
							a.Refine()
						}
						button.Role(ui.RoleCheckBox).Checked(chosen)
						if chosen {
							button.Border(2, p.Border)
						} else {
							button.Border(1, ui.Transparent)
						}
						background := ui.Transparent
						if chosen {
							background = p.Subtle
						} else if button.Hovered() {
							background = p.Hover
						}
						animateColors(button, background, p.Text)
						button.Children(func() {
							swatch := ui.Hex(family.Hex)
							ui.Box(c).Size(22, 22).Radius(11).Background(swatch).Border(1, p.BorderSubtle).Center().Children(func() {
								if chosen {
									checkColor := lightPalette.Text
									if float32(swatch.R)*.2126+float32(swatch.G)*.7152+float32(swatch.B)*.0722 < 145 {
										checkColor = darkPalette.Text
									}
									appSymbol(c, "check").Size(14, 14).TextColor(checkColor).PassThrough()
								}
							})
						})
					}
				})
			}
			ui.Divider(c)
			ui.Text(c, a.T("filter_tags")).FontSize(fontGroup).FontWeight(600).LetterSpacing(.6).TextColor(p.TextSecondary)
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
				if settingsCheckbox(c.Key("filter-tag-"+tag.ID), &chosen, tag.Name).FillWidth().Changed() {
					a.Query.TagIDs = toolbarToggle(a.Query.TagIDs, tag.ID, chosen)
					a.Refine()
				}
			}
			if matched == 0 {
				ui.Text(c, a.T("no_tags")).TextColor(p.TextSecondary).FontSize(fontSecondary)
			}
			ui.Divider(c)
			ui.Text(c, a.T("filter_types")).FontSize(fontGroup).FontWeight(600).LetterSpacing(.6).TextColor(p.TextSecondary)
			types := a.toolbarExtensions()
			if len(types) == 0 {
				ui.Text(c, a.T("no_file_types")).TextColor(p.TextSecondary).FontSize(fontSecondary)
			}
			ui.Row(c).Wrap().Gap(space8).Children(func() {
				for _, ext := range types {
					chosen := slices.Contains(a.Query.Extensions, ext)
					if animatedToggle(c.Key("extension-"+ext), &chosen, strings.ToUpper(ext)).Changed() {
						a.Query.Extensions = toolbarToggle(a.Query.Extensions, ext, chosen)
						a.Refine()
					}
				}
			})
		})
	})
}

func (a *App) sortPanel(c *ui.Context) {
	p := paletteFor(c.Theme())
	ui.Column(c).Gap(space8).Children(func() {
		ui.Text(c, a.T("sort")).FontSize(fontBody).Bold()
		for _, item := range []struct{ ID, Label string }{{"added", "sort_added"}, {"name", "sort_name"}, {"size", "sort_size"}} {
			chosen := a.Query.Sort == item.ID || a.Query.Sort == "" && item.ID == "added"
			row := ui.ButtonBase(c.Key(item.ID)).Height(32).FillWidth().Justify(ui.Start).Padding(0, space12).Radius(radiusControl).Cursor(ui.CursorPointer).Label(a.T(item.Label))
			background, text := ui.Transparent, p.TextSecondary
			if chosen {
				background, text = p.Selected, p.Text
				elevated(row, c.Theme())
			}
			if row.Hovered() {
				background, text = p.Hover, p.Text
			}
			if row.Pressed() {
				background = p.Pressed
			}
			animateColors(row, background, text)
			row.Children(func() { ui.Text(c, a.T(item.Label)).FontSize(fontBody).SingleLine() })
			if row.Clicked() {
				a.changeSort(item.ID)
				a.Toolbar.SortOpen = false
			}
		}
		ui.Divider(c)
		descending := a.Query.Descending
		animatedRadio(c, &descending, false, a.T("sort_ascending"))
		animatedRadio(c, &descending, true, a.T("sort_descending"))
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

func (a *App) toolbarWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	return width - a.SidebarShownWidth() - a.InspectorShownWidth()
}

func (a *App) thumbnailSizeSlider(c *ui.Context) {
	ui.Row(c.Key("thumbnail-size-control")).Width(120).Shrink(0).Gap(space4).AlignItems(ui.Center).Children(func() {
		disabled := a.ViewMode == 2
		if toolbarIconButton(c.Key("thumbnail-smaller"), a.T("thumbnail_smaller"), "minus").Size(24, 28).Disabled(disabled || a.thumbnailScale() <= .5).Clicked() {
			a.ChangeThumbnailSize(-.1)
		}
		size := float64(a.thumbnailScale())
		if ui.Slider(c.Key("thumbnail-size"), &size, .5, 2).Label(a.T("thumbnail_size")).Tooltip(a.T("thumbnail_size")).Grow(1).MinWidth(0).Basis(0).Disabled(disabled).Changed() {
			a.SetThumbnailSize(float32(size))
		}
		if toolbarIconButton(c.Key("thumbnail-larger"), a.T("thumbnail_larger"), "plus").Size(28, 28).Disabled(disabled || a.thumbnailScale() >= 2).Clicked() {
			a.ChangeThumbnailSize(.1)
		}
	})
}
