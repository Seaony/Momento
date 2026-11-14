package ui

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type tagsState struct {
	Search string
	List   ui.ListState
}

func (a *App) tags(c *ui.Context) {
	search := strings.ToLower(strings.TrimSpace(a.Tags.Search))
	tags := make([]model.Tag, 0, len(a.Snapshot.Tags))
	for _, tag := range a.Snapshot.Tags {
		if search == "" || strings.Contains(strings.ToLower(tag.Name), search) {
			tags = append(tags, tag)
		}
	}
	slices.SortFunc(tags, func(left, right model.Tag) int {
		if order := strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)); order != 0 {
			return order
		}
		return strings.Compare(left.ID, right.ID)
	})
	searchChanged := false
	counts := a.Index.Counts().Tags
	a.Tags.List.Key = func(i int) any {
		if i == 0 {
			return "tag-header"
		}
		if len(tags) == 0 {
			return "tag-empty"
		}
		return tags[i-1].ID
	}
	a.Tags.List.Label = func(i int) string {
		if i == 0 {
			return a.T("tag_management")
		}
		if len(tags) == 0 {
			return a.T("no_tags")
		}
		return tags[i-1].Name
	}
	ui.List(c.Key("tag-list"), &a.Tags.List, max(1, len(tags))+1, func(i int) {
		if i == 0 {
			ui.Column(c.Key("tag-header")).FillWidth().Gap(12).Padding(0, 0, 16, 0).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
					if ui.SearchField(c.Key("tag-search"), &a.Tags.Search).Label(a.T("search_tags")).Placeholder(a.T("search_tags")).Grow(1).MinWidth(0).Changed() {
						searchChanged = true
					}
					if primaryButton(c.Key("new-tag"), a.T("new_tag"), a.Busy).Radius(8).Clicked() {
						lib := a.Library
						a.Ask(a.T("new_tag"), "", func(name string) {
							a.Run("create_tag", func(ctx context.Context) error {
								_, err := lib.CreateTag(ctx, name)
								return err
							})
						})
					}
				})
				ui.Text(c, a.T("tag_total", formatTagCount(len(tags)))).FontSize(12).TextColor(c.Theme().TextMuted)
			})
			return
		}
		if len(tags) == 0 {
			ui.Column(c.Key("tag-empty")).MinHeight(220).FillWidth().Center().Gap(8).Children(func() {
				appIcon(c, "tag").Size(40, 40).Margin(0, 0, 8, 0)
				if search == "" {
					ui.Text(c, a.T("no_tags")).FontSize(18).Bold().TextColor(titleColor(c.Theme()))
					ui.Text(c, a.T("no_tags_hint")).FontSize(13).TextColor(c.Theme().TextMuted)
				} else {
					ui.Text(c, a.T("no_tag_results", strings.TrimSpace(a.Tags.Search))).FontSize(13)
				}
			})
			return
		}
		tag := tags[i-1]
		row := ui.Row(c.Key(tag.ID)).Label(tag.Name).Focusable().Cursor(ui.CursorPointer).FillWidth().MinHeight(44).Gap(12).AlignItems(ui.Center).Padding(4, 8).Radius(8)
		if row.Hovered() {
			row.Background(warmOverlay(c.Theme(), .06))
		}
		row.Children(func() {
			appIcon(c, "tag").Size(20, 20).Shrink(0)
			ui.Text(c, tag.Name).FontSize(13).Grow(1).MinWidth(0).SingleLine()
			ui.Text(c, a.T("tag_asset_count", formatTagCount(counts[tag.ID]))).FontSize(12).TextColor(c.Theme().TextMuted).Shrink(0)
			if inspectorIconButton(c.Key("rename"), a.T("rename_tag")+" "+tag.Name, "edit").Tooltip(a.T("rename_tag")).Disabled(a.Busy).Clicked() {
				lib := a.Library
				a.Ask(a.T("rename_tag"), tag.Name, func(name string) {
					a.Run("rename_tag", func(ctx context.Context) error { return lib.RenameTag(ctx, tag.ID, name) })
				})
			}
			if inspectorIconButton(c.Key("delete"), a.T("delete_tag")+" "+tag.Name, "trash").Tooltip(a.T("delete_tag")).Disabled(a.Busy).Clicked() {
				lib := a.Library
				a.Confirm(a.T("delete_tag"), a.T("delete_tag_detail", tag.Name), func() {
					a.Run("delete_tag", func(ctx context.Context) error { return lib.DeleteTag(ctx, tag.ID) })
				})
			}
		})
		row.ContextMenu(func(m *ui.Menu) { a.QuickAccessMenu(m, "tag", tag.ID) })
		if row.DoubleClicked() || row.Submitted() {
			a.BrowseTag(tag.ID)
		}
	}).Grow(1).FillWidth().Padding(a.contentTopInset()+16, 24, 24, 24).ScrollbarInsets(a.contentTopInset(), 0, 0).Label(a.T("tag_management"))
	if searchChanged {
		a.Tags.List = ui.ListState{}
	}
}

func formatTagCount(count int) string {
	text := strconv.Itoa(count)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}
