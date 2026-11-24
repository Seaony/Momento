package ui

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/library"
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
	a.AskWith("tags", a.T("new_tag"), "", a.T("create"), a.createTag)
}

// 标签管理页新建标签：已有同名（规范化后相同）的就提示，不新建。
func (a *App) createTag(name string) {
	normalized := library.NormalizeTag(name)
	if slices.ContainsFunc(a.Snapshot.Tags, func(tag model.Tag) bool { return library.NormalizeTag(tag.Name) == normalized }) {
		a.Notify(a.T("tag_exists"))
		return
	}
	lib := a.Library
	a.Run("create_tag", func(ctx context.Context) error {
		_, err := lib.CreateTag(ctx, name)
		return err
	})
}

func (a *App) renameTag(tag model.Tag) {
	lib := a.Library
	a.Ask("pencil-simple", a.T("rename_tag"), tag.Name, func(name string) {
		a.Run("rename_tag", func(ctx context.Context) error { return lib.RenameTag(ctx, tag.ID, name) })
	})
}

// 删除标签：没有素材用它就直接删；否则先确认，写明会从几个素材上移除。
// 回收站里的素材也算：删了标签，它们恢复后就没有这个标签了。
func (a *App) deleteTag(tag model.Tag) {
	lib := a.Library
	remove := func() {
		a.runThenNotify("delete_tag", func(ctx context.Context) error { return lib.DeleteTag(ctx, tag.ID) }, a.T("tag_deleted", tag.Name))
	}
	count := 0
	for i := range a.Snapshot.Assets {
		if slices.Contains(a.Snapshot.Assets[i].TagIDs, tag.ID) {
			count++
		}
	}
	if count == 0 {
		remove()
		return
	}
	a.Confirm(a.T("delete_tag"), a.T("delete_tag_assets_detail", tag.Name, count), a.T("delete"), remove)
}

// 标签管理页：搜索和新建都在顶部工具栏里，页面里只列标签。
func (a *App) tags(c *ui.Context) {
	p := paletteFor(c.Theme())
	search := strings.TrimSpace(a.Tags.Search)
	tags := a.visibleTags()
	counts := a.Index.Counts().Tags
	if len(tags) == 0 {
		// 空状态和素材区一样，在工具栏下方的整块区域里上下左右居中。
		fadeMotion(ui.Column(c.Key("tag-empty"))).Grow(1).FillWidth().Padding(a.contentTopInset(), space24, space24, space24).Center().Gap(space8).Children(func() {
			art := "no-tags"
			if search != "" {
				art = "search-empty"
			}
			emptyArt(c.Key("tag-empty-art"), art, emptyArtLarge).Margin(0, 0, space8, 0)
			if search == "" {
				ui.Text(c, a.T("no_tags")).FontSize(fontTitle).FontWeight(600).TextColor(p.Text)
				ui.Text(c, a.T("no_tags_hint")).FontSize(fontBody).TextColor(p.TextSecondary)
				if dialogPrimaryButton(c.Key("tag-empty-new"), a.T("new_tag"), false).Clicked() {
					a.NewTag()
				}
			} else {
				ui.Text(c, a.T("no_tag_results", search)).FontSize(fontBody)
				if dialogPrimaryButton(c.Key("tag-search-new"), a.T("new_tag_named", search), false).Clicked() {
					a.createTag(search)
				}
			}
		})
		return
	}
	scroll := fadeMotion(ui.Scroll(c.Key("tag-list")).Scrollbars(false)).TrackScroll(&a.Tags.Scroll).Grow(1).FillWidth().Label(a.T("tag_management")).Role(ui.RoleList)
	inset := scroll.AnimateTo("header-inset", a.contentTopInset(), motionSpring)
	scroll.Padding(inset+space16, space24, space24, space24)
	scroll.Children(func() {
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
					showPose(tools, shown).IgnoreInput(!shown).Children(func() {
						if inspectorIconButton(c.Key("rename"), a.T("rename_tag")+" "+tag.Name, "edit").Tooltip(a.T("rename_tag")).Clicked() {
							a.renameTag(tag)
						}
						if inspectorIconButton(c.Key("delete"), a.T("delete_tag")+" "+tag.Name, "trash").Tooltip(a.T("delete_tag")).Clicked() {
							a.deleteTag(tag)
						}
					})
				})
				row.ContextMenu(func(m *ui.Menu) {
					if menuItem(m, a.T("rename_tag"), "pencil-simple").Chosen() {
						a.renameTag(tag)
					}
					m.Separator()
					a.QuickAccessMenu(m, "tag", tag.ID)
					m.Separator()
					if menuItem(m, a.T("delete_tag"), "trash").Chosen() {
						a.deleteTag(tag)
					}
				})
				if row.Clicked() {
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
