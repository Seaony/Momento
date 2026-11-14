package ui

import (
	"context"
	"slices"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 智能文件夹关闭时不显示固定的智能文件夹，固定记录保留。
func (a *App) visibleQuickAccess() []model.QuickAccess {
	if a.Preferences.SmartFolders {
		return a.Snapshot.QuickAccess
	}
	var entries []model.QuickAccess
	for _, entry := range a.Snapshot.QuickAccess {
		if entry.Kind != "smart" {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (a *App) sidebarQuickAccess(c *ui.Context) {
	entries := a.visibleQuickAccess()
	if a.sidebarHidden("quick") || len(entries) == 0 {
		return
	}
	ui.Text(c, a.T("quick_access")).FontSize(fontGroup).FontWeight(600).LetterSpacing(.88).TextColor(paletteFor(c.Theme()).TextSecondary).Padding(0, space12).Height(controlSmall).Margin(space20, 0, 0, 0)
	a.GBrowsing.QuickList.Key = func(i int) any { return entries[i].ID }
	a.GBrowsing.QuickList.Label = func(i int) string { name, _, _, _ := a.quickAccessTarget(entries[i]); return name }
	a.GBrowsing.QuickList.Reorder = func(rows []int, to int) {
		if a.Busy || len(rows) != 1 || rows[0] < 0 || rows[0] >= len(entries) {
			return
		}
		from := rows[0]
		if to > from {
			to--
		}
		a.moveVisibleQuickAccess(entries, from, max(0, min(to, len(entries)-1)))
	}
	ui.List(c.Key("quick-access-list"), &a.GBrowsing.QuickList, len(entries), func(i int) {
		entry := entries[i]
		name, icon, count, selected := a.quickAccessTarget(entry)
		if name == "" {
			return
		}
		var row ui.Element
		if entry.Kind == "smart" {
			row = a.smartSidebarRow(c.Key(entry.ID), name, count, selected)
		} else {
			row = a.sidebarRow(c.Key(entry.ID), name, icon, count, selected)
		}
		if row.Clicked() {
			a.openQuickAccess(entry)
		}
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.T("quick_move_up")).Disabled(a.Busy || i == 0).Chosen() {
				a.moveVisibleQuickAccess(entries, i, i-1)
			}
			if m.Item(a.T("quick_move_down")).Disabled(a.Busy || i == len(entries)-1).Chosen() {
				a.moveVisibleQuickAccess(entries, i, i+1)
			}
			m.Separator()
			if m.Item(a.T("quick_unpin")).Disabled(a.Busy).Chosen() {
				a.unpinQuickAccess(entry.ID)
			}
		})
	}).FillWidth().Height(float32(len(entries)) * 32).Shrink(0)
}

func (a *App) PinQuickAccess(kind, id string) {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	a.Run("pin_quick_access", func(ctx context.Context) error { _, err := lib.PinQuickAccess(ctx, kind, id); return err })
}

func (a *App) QuickAccessMenu(m *ui.Menu, kind, id string) {
	entry, exists := a.pinnedQuickAccess(kind, id)
	if exists {
		if m.Item(a.T("quick_unpin")).Disabled(a.Busy).Chosen() {
			a.unpinQuickAccess(entry.ID)
		}
	} else if m.Item(a.T("quick_pin")).Disabled(a.Busy || a.Library == nil).Chosen() {
		a.PinQuickAccess(kind, id)
	}
}

func (a *App) pinnedQuickAccess(kind, id string) (model.QuickAccess, bool) {
	for _, entry := range a.Snapshot.QuickAccess {
		if entry.Kind == kind && entry.TargetID == id {
			return entry, true
		}
	}
	return model.QuickAccess{}, false
}
func (a *App) unpinQuickAccess(id string) {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	a.Run("unpin_quick_access", func(ctx context.Context) error { return lib.UnpinQuickAccess(ctx, id) })
}

func (a *App) moveVisibleQuickAccess(visible []model.QuickAccess, from, to int) {
	a.moveQuickAccess(visible[from].ID, quickAccessMoveIndex(a.Snapshot.QuickAccess, visible, from, to))
}

// 把可见列表里的目标位置换算成完整列表（去掉被移动项后）的位置，
// 隐藏的智能文件夹固定项保持原来的相对顺序。
func quickAccessMoveIndex(all, visible []model.QuickAccess, from, to int) int {
	id := visible[from].ID
	var others []string
	for _, entry := range all {
		if entry.ID != id {
			others = append(others, entry.ID)
		}
	}
	visibleOthers := slices.Delete(slices.Clone(visible), from, from+1)
	if to < len(visibleOthers) {
		return slices.Index(others, visibleOthers[to].ID)
	}
	if len(visibleOthers) > 0 {
		return slices.Index(others, visibleOthers[len(visibleOthers)-1].ID) + 1
	}
	return len(others)
}

func (a *App) moveQuickAccess(id string, index int) {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	a.Run("move_quick_access", func(ctx context.Context) error { return lib.MoveQuickAccess(ctx, id, index) })
}
