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
	Scroll ui.ScrollState
}

// 标签管理页显示的标签：按顶部搜索框筛选，按名字排序。
func (a *App) visibleTags() []model.Tag {
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
	return tags
}

// 新建标签：标签管理页工具栏右上角的按钮。
func (a *App) NewTag() {
	lib := a.Library
	a.Ask("tags", a.T("new_tag"), "", func(name string) {
		a.Run("create_tag", func(ctx context.Context) error {
			_, err := lib.CreateTag(ctx, name)
			return err
		})
	})
}

// 标签管理页：搜索和新建都在顶部工具栏里，页面里只列标签。
func (a *App) tags(c *ui.Context) {
	p := paletteFor(c.Theme())
	search := strings.TrimSpace(a.Tags.Search)
	tags := a.visibleTags()
	counts := a.Index.Counts().Tags
	scroll := ui.Scroll(c.Key("tag-list")).Scrollbars(false).TrackScroll(&a.Tags.Scroll).Grow(1).FillWidth().Label(a.T("tag_management")).Role(ui.RoleList)
	inset := scroll.AnimateTo("header-inset", a.contentTopInset(), motionSpring)
	scroll.Padding(inset+space16, space24, space24, space24)
	scroll.Children(func() {
		if len(tags) == 0 {
			fadeMotion(ui.Column(c.Key("tag-empty"))).MinHeight(220).FillWidth().Center().Gap(space8).Children(func() {
				art := "no-tags"
				if search != "" {
					art = "search-empty"
				}
				emptyArt(c.Key("tag-empty-art"), art, emptyArtLarge).Margin(0, 0, space8, 0)
				if search == "" {
					ui.Text(c, a.T("no_tags")).FontSize(fontTitle).FontWeight(600).TextColor(p.Text)
					ui.Text(c, a.T("no_tags_hint")).FontSize(fontBody).TextColor(p.TextSecondary)
				} else {
					ui.Text(c, a.T("no_tag_results", search)).FontSize(fontBody)
				}
			})
			return
		}
		_, height := c.Size()
		if bounds := scroll.LayoutBounds(); bounds.H > 0 {
			height = bounds.H
		}
		const rowHeight = space40 + space4
		viewportTop := a.Tags.Scroll.Y - inset - space16
		first := min(len(tags), max(0, int(viewportTop/rowHeight)-2))
		last := min(len(tags), first+int(height/rowHeight)+5)
		fadeMotion(ui.Box(c.Key("tag-rows"))).Height(float32(len(tags) * rowHeight)).Shrink(0).FillWidth().Children(func() {
			for i := first; i < last; i++ {
				tag := tags[i]
				row := ui.Row(c.Key(tag.ID)).Absolute().Top(float32(i*rowHeight)).Left(0).Right(0).Height(rowHeight).Label(tag.Name).Role(ui.RoleListItem).PositionInSet(i+1, len(tags)).Focusable().Cursor(ui.CursorPointer).Gap(space12).AlignItems(ui.Center).Padding(space4, space8).Radius(radiusControl)
				background, iconColor := ui.Transparent, p.TextSecondary
				if row.Hovered() {
					background, iconColor = p.Hover, p.Text
				}
				animateColors(row, background, p.Text)
				if float32((i+1)*rowHeight) > viewportTop+inset && float32(i*rowHeight) < viewportTop+height {
					layoutMotion(row)
				}
				row.Children(func() {
					appIcon(c, "tag").Size(iconSize, iconSize).Shrink(0).TextColor(row.AnimateColor("tag-icon-color", iconColor, motionColor))
					ui.Text(c, tag.Name).ContentTransition(ui.ContentTransition{Animation: &motionSpring}).FontSize(fontBody).Grow(1).MinWidth(0).SingleLine()
					// 数量用和列表日期同一档的弱化字色；改名、删除平时收起，悬停这一行或键盘焦点在这一行（含按钮）时才出现，位置不变免得数量跳动。
					ui.Text(c, a.T("tag_asset_count", formatTagCount(counts[tag.ID]))).ContentTransition(ui.ContentTransition{Numeric: true, Animation: &motionSpring}).
						FontSize(fontSecondary).FontWeight(500).FontFeatures("tnum").TextColor(p.Text.Alpha(themeValue(c, .56, .64))).Shrink(0)
					tools := ui.Row(c.Key("tools")).Gap(space4).Shrink(0).AlignItems(ui.Center)
					shown := row.Hovered() || row.FocusWithin()
					opacity := float32(0)
					if shown {
						opacity = 1
					}
					tools.Opacity(tools.AnimateTo("opacity", opacity, motionSpring)).IgnoreInput(!shown).Children(func() {
						if inspectorIconButton(c.Key("rename"), a.T("rename_tag")+" "+tag.Name, "edit").Tooltip(a.T("rename_tag")).Disabled(a.Busy).Clicked() {
							lib := a.Library
							a.Ask("pencil-simple", a.T("rename_tag"), tag.Name, func(name string) {
								a.Run("rename_tag", func(ctx context.Context) error { return lib.RenameTag(ctx, tag.ID, name) })
							})
						}
						if inspectorIconButton(c.Key("delete"), a.T("delete_tag")+" "+tag.Name, "trash").Tooltip(a.T("delete_tag")).Disabled(a.Busy).Clicked() {
							lib := a.Library
							a.Confirm(a.T("delete_tag"), a.T("delete_tag_detail", tag.Name), a.T("delete"), func() {
								a.Run("delete_tag", func(ctx context.Context) error { return lib.DeleteTag(ctx, tag.ID) })
							})
						}
					})
				})
				row.ContextMenu(func(m *ui.Menu) { a.QuickAccessMenu(m, "tag", tag.ID) })
				if row.DoubleClicked() || row.Submitted() {
					a.BrowseTag(tag.ID)
				}
			}
		})
	})
}

func formatTagCount(count int) string {
	text := strconv.Itoa(count)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}
