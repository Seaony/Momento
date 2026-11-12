package ui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type toolbarState struct {
	FilterOpen bool
	SortOpen   bool
	TagSearch  string
}

var toolbarFamilies = []struct{ ID, Hex string }{
	{"red", "#E34848"}, {"coral", "#EE7667"}, {"orange", "#E48A39"}, {"amber", "#D5A234"},
	{"yellow", "#D4BC38"}, {"lime", "#A3B64D"}, {"green", "#539B68"}, {"mint", "#6CB69C"},
	{"teal", "#458C8F"}, {"cyan", "#50A5B9"}, {"sky", "#6BA6D4"}, {"blue", "#527ABD"},
	{"indigo", "#6565B0"}, {"violet", "#8D69B8"}, {"purple", "#A56CB4"}, {"magenta", "#BB6B9D"},
	{"pink", "#D587AA"}, {"rose", "#CB6A86"}, {"brown", "#886747"}, {"olive", "#85834E"},
	{"beige", "#C8BA96"}, {"white", "#FFFFFF"}, {"gray", "#939393"}, {"black", "#252525"},
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
	ui.Row(c.Key("toolbar")).Height(MainToolbarHeight).Shrink(0).FillWidth().Padding(12, right, 12, left).Gap(6).DragWindow().
		Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Role(ui.RoleToolbar).FocusGroup(ui.Horizontal).Label(a.T("browse_tools")).Children(func() {
		if toolbarIconButton(c.Key("toggle-sidebar"), a.T("toggle_sidebar"), "sidebar").Clicked() {
			a.SidebarVisible = !a.SidebarVisible
		}
		search := ui.SearchField(c.Key("asset-search"), &a.Query.Search).Bind(&a.Search).Placeholder(a.T("search_assets")).Label(a.T("search_assets")).Grow(1).MinWidth(96).Height(34)
		if search.Changed() {
			a.Refine()
		}
		if strings.TrimSpace(a.Query.Search) != "" {
			ui.Text(c, a.T("result_count", toolbarNumber(len(a.Visible)))).SingleLine().Shrink(0).FontSize(12).TextColor(t.TextMuted).Role(ui.RoleStatus)
		}
		filter := toolbarIconButton(c.Key("filter"), a.T("filter"), "filter")
		if len(a.Query.ColorFamilies)+len(a.Query.TagIDs)+len(a.Query.Extensions) > 0 {
			filter.Background(t.Selection)
		}
		if filter.Clicked() {
			a.Toolbar.FilterOpen = !a.Toolbar.FilterOpen
			a.Toolbar.SortOpen = false
		}
		ui.Popover(c.Key("filters"), filter, &a.Toolbar.FilterOpen, func() { a.filterPanel(c) }).Width(340).MaxHeight(560)
		sort := toolbarIconButton(c.Key("sort"), a.T("sort"), "sort")
		if sort.Clicked() {
			a.Toolbar.SortOpen = !a.Toolbar.SortOpen
			a.Toolbar.FilterOpen = false
		}
		ui.Popover(c.Key("sorting"), sort, &a.Toolbar.SortOpen, func() { a.sortPanel(c) }).Width(220)
		a.viewSwitcher(c)
		toolbarIconButton(c.Key("import"), a.T("import"), "import").Disabled(a.Busy).Menu(func(m *ui.Menu) {
			if m.Item(a.T("import_files")).Chosen() {
				a.ChooseImport(false)
			}
			if m.Item(a.T("import_folder")).Chosen() {
				a.ChooseImport(true)
			}
		})
		if version := AvailableUpdateVersion(); version != "" {
			if ui.PrimaryButton(c.Key("available-update"), a.T("update")).Height(34).Shrink(0).Tooltip(a.T("update_to_version", version)).Clicked() {
				ShowAvailableUpdate()
			}
		}
		if toolbarIconButton(c.Key("toggle-inspector"), a.T("toggle_inspector"), "inspector").Clicked() {
			a.InspectorVisible = !a.InspectorVisible
		}
	})
}

func toolbarIconButton(c *ui.Context, label, name string) ui.Element {
	return ui.Button(c, "").Label(label).Tooltip(label).Size(32, 34).Padding(0).Radius(8).Children(func() { appIcon(c, name).Size(20, 20) })
}

func (a *App) viewSwitcher(c *ui.Context) {
	t := c.Theme()
	seg := ui.SegmentedBase(c.Key("view-mode"), &a.ViewMode, 3)
	seg.Track.Gap(0).Padding(2).Radius(8).Background(t.Surface).Height(34).Label(a.T("view_mode")).Children(func() {
		for i, item := range []struct{ ID, Label string }{{"masonry", "view_masonry"}, {"grid", "view_grid"}, {"list", "view_list"}} {
			part := seg.Segment(i).Size(28, 30).Padding(0).Center().Radius(6).Label(a.T(item.Label)).Tooltip(a.T(item.Label))
			if a.ViewMode == i {
				part.Background(t.Background).Border(1, t.Border)
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
									checkColor := ui.RGB(255, 255, 255)
									if 299*int(swatch.R)+587*int(swatch.G)+114*int(swatch.B) > 150000 {
										checkColor = ui.RGB(17, 24, 39)
									}
									ui.Icon(c, toolbarColorCheck).Size(14, 14).TextColor(checkColor).PassThrough()
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
