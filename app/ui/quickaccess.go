package ui

import (
	"context"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func (a *App) sidebarQuickAccess(c *ui.Context) {
	if a.sidebarHidden("quick") {
		return
	}
	ui.Text(c, a.T("quick_access")).FontSize(12).Bold().TextColor(c.Theme().TextMuted).Padding(12, 10, 4, 10)
	entries := a.Snapshot.QuickAccess
	if len(entries) == 0 {
		ui.Text(c, a.T("quick_access_empty")).FontSize(12).TextColor(c.Theme().TextMuted).Padding(4, 10, 8, 10)
		return
	}
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
		a.moveQuickAccess(entries[from].ID, max(0, min(to, len(entries)-1)))
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
		row.PopoverMenu(func(m *ui.Menu) {
			if m.Item(a.T("quick_move_up")).Disabled(a.Busy || i == 0).Chosen() {
				a.moveQuickAccess(entry.ID, i-1)
			}
			if m.Item(a.T("quick_move_down")).Disabled(a.Busy || i == len(entries)-1).Chosen() {
				a.moveQuickAccess(entry.ID, i+1)
			}
			m.Separator()
			if m.Item(a.T("quick_unpin")).Disabled(a.Busy).Chosen() {
				a.unpinQuickAccess(entry.ID)
			}
		})
	}).FillWidth().Height(float32(len(entries)) * 34).Shrink(0)
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
func (a *App) moveQuickAccess(id string, index int) {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	a.Run("move_quick_access", func(ctx context.Context) error { return lib.MoveQuickAccess(ctx, id, index) })
}
