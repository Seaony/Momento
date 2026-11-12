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
	ui.Column(c).Grow(1).FillWidth().Padding(24).Gap(16).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, a.T("tag_management")).FontSize(22).Bold().Grow(1)
			if ui.PrimaryButton(c.Key("new-tag"), a.T("new_tag")).Disabled(a.Busy).Clicked() {
				lib := a.Library
				a.Ask(a.T("new_tag"), "", func(name string) {
					a.Run("create_tag", func(ctx context.Context) error {
						_, err := lib.CreateTag(ctx, name)
						return err
					})
				})
			}
		})
		if ui.SearchField(c.Key("tag-search"), &a.Tags.Search).Label(a.T("search_tags")).Placeholder(a.T("search_tags")).FillWidth().Changed() {
			a.Tags.List = ui.ListState{}
		}
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
		ui.Text(c, a.T("tag_total", formatTagCount(len(tags)))).FontSize(12).TextColor(c.Theme().TextMuted)
		if len(tags) == 0 {
			ui.Column(c).Grow(1).FillWidth().Center().Gap(8).Children(func() {
				if search == "" {
					ui.Text(c, a.T("no_tags")).FontSize(18).Bold()
					ui.Text(c, a.T("no_tags_hint")).TextColor(c.Theme().TextMuted)
				} else {
					ui.Text(c, a.T("no_tag_results", strings.TrimSpace(a.Tags.Search))).FontSize(16)
				}
			})
			return
		}
		counts := a.Index.Counts().Tags
		a.Tags.List.Key = func(i int) any { return tags[i].ID }
		a.Tags.List.Label = func(i int) string { return tags[i].Name }
		ui.List(c.Key("tag-list"), &a.Tags.List, len(tags), func(i int) {
			tag := tags[i]
			row := ui.Row(c.Key(tag.ID)).Label(tag.Name).Focusable().FillWidth().MinHeight(56).Gap(12).AlignItems(ui.Center).Padding(8, 0).
				BorderWidth(0, 0, 1, 0).BorderColor(c.Theme().Border).Children(func() {
				ui.Text(c, tag.Name).Grow(1).MinWidth(0).SingleLine()
				ui.Text(c, a.T("tag_asset_count", formatTagCount(counts[tag.ID]))).FontSize(12).TextColor(c.Theme().TextMuted)
				if ui.Button(c.Key("rename"), a.T("rename_tag")).Label(a.T("rename_tag") + " " + tag.Name).Disabled(a.Busy).Clicked() {
					lib := a.Library
					a.Ask(a.T("rename_tag"), tag.Name, func(name string) {
						a.Run("rename_tag", func(ctx context.Context) error { return lib.RenameTag(ctx, tag.ID, name) })
					})
				}
				if ui.Button(c.Key("delete"), a.T("delete_tag")).Label(a.T("delete_tag") + " " + tag.Name).Disabled(a.Busy).Clicked() {
					lib := a.Library
					a.Confirm(a.T("delete_tag"), a.T("delete_tag_detail", tag.Name), func() {
						a.Run("delete_tag", func(ctx context.Context) error { return lib.DeleteTag(ctx, tag.ID) })
					})
				}
			})
			if row.DoubleClicked() || row.Submitted() {
				a.BrowseTag(tag.ID)
			}
		}).Grow(1).FillWidth().Label(a.T("tag_management"))
	})
}

func formatTagCount(count int) string {
	text := strconv.Itoa(count)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}
