package ui

import (
	"context"
	"log/slog"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func (a *App) sidebarSmartFolders(c *ui.Context) {
	if a.sidebarHidden("smart") {
		return
	}
	a.sidebarSectionHeader(c, a.T("smart_folders"), a.T("new_smart_folder"), 16, func() { a.EditSmartFolder("") })
	if len(a.Snapshot.SmartFolders) == 0 {
		if sidebarEmptyAction(c.Key("smart-empty"), a.T("smart_empty_action"), "smart-folder").Disabled(a.Busy || a.Library == nil).Clicked() {
			a.EditSmartFolder("")
		}
	}
	for _, folder := range a.Snapshot.SmartFolders {
		count := 0
		if a.Index != nil {
			count = a.smartFolderCount(folder.ID)
		}
		row := a.smartSidebarRow(c.Key(folder.ID), folder.Name, count, !a.TagPage && a.Query.Scope == "smart" && a.Query.SmartFolderID == folder.ID)
		if row.Clicked() {
			a.BrowseSmartFolder(folder.ID)
		}
		row.ContextMenu(func(m *ui.Menu) {
			if m.Item(a.T("edit_smart_folder")).Disabled(a.Busy).Chosen() {
				a.EditSmartFolder(folder.ID)
			}
			if m.Item(a.T("rename_smart_folder")).Disabled(a.Busy).Chosen() {
				lib := a.Library
				a.Ask(a.T("rename_smart_folder"), folder.Name, func(name string) {
					a.Run("rename_smart_folder", func(ctx context.Context) error { return lib.RenameSmartFolder(ctx, folder.ID, name) })
				})
			}
			a.QuickAccessMenu(m, "smart", folder.ID)
			m.Separator()
			if m.Item(a.T("delete_smart_folder")).Disabled(a.Busy).Chosen() {
				a.Confirm(a.T("delete_smart_folder"), a.T("delete_smart_folder_detail", folder.Name), func() {
					if a.Query.Scope == "smart" && a.Query.SmartFolderID == folder.ID {
						a.SetScope("all", "")
					}
					lib := a.Library
					a.Run("delete_smart_folder", func(ctx context.Context) error { return lib.DeleteSmartFolder(ctx, folder.ID) })
				})
			}
		})
	}
}

func (a *App) smartSidebarRow(c *ui.Context, label string, count int, selected bool) ui.Element {
	return a.sidebarRow(c, label, "smart-folder", count, selected)
}

func (a *App) EditSmartFolder(id string) {
	if a.Busy || a.Library == nil {
		return
	}
	state := &a.GBrowsing
	state.SmartID = id
	state.SmartName = ""
	state.SmartMatch = 0
	state.SmartRules = []model.SmartRule{{Field: "name", Operator: "contains"}}
	if id != "" {
		found := false
		for _, folder := range a.Snapshot.SmartFolders {
			if folder.ID == id {
				found = true
				state.SmartName = folder.Name
				state.SmartRules = slices.Clone(folder.Rules)
				for i := range state.SmartRules {
					rule := &state.SmartRules[i]
					if rule.Field != "size" {
						continue
					}
					bytes, err := strconv.ParseInt(rule.Value, 10, 64)
					if err != nil {
						a.ShowError(i18n.NewError("error_smart_rule"))
						return
					}
					mb := big.NewRat(bytes, 1<<20)
					rule.Value = strings.TrimRight(strings.TrimRight(mb.FloatString(20), "0"), ".")
				}
				if folder.Match == "any" {
					state.SmartMatch = 1
				}
				break
			}
		}
		if !found {
			return
		}
	}
	state.SmartOpen = true
	state.SmartError = ""
	state.RulesScroll = ui.ScrollState{}
}

func (a *App) smartFolderDialog(c *ui.Context) {
	state := &a.GBrowsing
	if !state.SmartOpen {
		return
	}
	settingsPanel(c.Key("smart-folder-editor"), &state.SmartOpen, 648, func() {
		title := a.T("smart_folder_create_title")
		if state.SmartID != "" {
			title = a.T("smart_folder_edit_title")
		}
		ui.Text(c, title).FontSize(20).Bold().TextColor(titleColor(c.Theme()))
		ui.TextInput(c.Key("smart-name"), &state.SmartName).FillWidth().Label(a.T("smart_folder_name")).Placeholder(a.T("smart_folder_name")).AutoFocus().Disabled(state.Saving)
		settingsSegments(c.Key("smart-match"), a.T("smart_match"), &state.SmartMatch, a.T("smart_all"), a.T("smart_any")).Disabled(state.Saving)
		_, height := c.Size()
		ui.Scroll(c.Key("smart-rules")).TrackScroll(&state.RulesScroll).MaxHeight(max(120, min(280, height-310))).FillWidth().Gap(8).Children(func() {
			remove := -1
			for i := range state.SmartRules {
				rule := &state.SmartRules[i]
				ui.Row(c.Key(i)).FillWidth().Gap(8).Padding(8).Radius(8).Background(c.Theme().Surface).AlignItems(ui.Center).Children(func() {
					previous := rule.Field
					smartSelect(c.Key("field"), &rule.Field, smartFields, func(key string) string { return a.T("smart_field_" + key) }).Width(132).Label(a.T("smart_rule_field", i+1)).Disabled(state.Saving)
					if previous != rule.Field {
						rule.Operator = smartOperators(rule.Field)[0]
						rule.Value = ""
					}
					smartSelect(c.Key("operator"), &rule.Operator, smartOperators(rule.Field), func(key string) string { return a.T("smart_operator_" + key) }).Width(124).Label(a.T("smart_rule_operator", i+1)).Disabled(state.Saving)
					a.smartRuleValue(c.Key("value"), rule, i)
					removeButton := inspectorIconButton(c.Key("remove"), a.T("remove_smart_rule_number", i+1), "close").Tooltip(a.T("remove_smart_rule")).Disabled(state.Saving || len(state.SmartRules) == 1)
					if removeButton.Clicked() {
						remove = i
					}
				})
			}
			if remove >= 0 {
				state.SmartRules = slices.Delete(state.SmartRules, remove, remove+1)
			}
		})
		if ui.Button(c.Key("add-rule"), a.T("add_smart_rule")).Disabled(state.Saving).Clicked() {
			state.SmartRules = append(state.SmartRules, model.SmartRule{Field: "name", Operator: "contains"})
		}
		if state.SmartError != "" {
			ui.Text(c, state.SmartError).TextColor(c.Theme().Danger).Role(ui.RoleStatus)
		}
		ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
			if ui.Button(c, a.T("cancel")).Disabled(state.Saving).Clicked() {
				state.SmartOpen = false
			}
			label := a.T("save")
			if state.Saving {
				label = a.T("smart_saving")
			}
			if ui.PrimaryButton(c, label).Disabled(state.Saving || a.Busy).Clicked() {
				a.saveSmartFolder()
			}
		})
	})
	if state.Saving {
		state.SmartOpen = true
	}
}

var smartFields = []string{"name", "extension", "tag", "color", "width", "height", "size", "added", "note"}

func smartOperators(field string) []string {
	switch field {
	case "name", "note":
		return []string{"contains", "equals"}
	case "width", "height", "size":
		return []string{"gte", "lte", "eq"}
	case "added":
		return []string{"after", "before"}
	default:
		return []string{"equals"}
	}
}
func smartSelect(c *ui.Context, value *string, options []string, label func(string) string) ui.Element {
	names := make([]string, len(options))
	chosen := ""
	for i, option := range options {
		names[i] = label(option)
		if option == *value {
			chosen = names[i]
		}
	}
	selectParts := ui.SelectBase(c, &chosen)
	control := selectParts.Trigger.Height(32).Padding(4, 10).Gap(8).Radius(8).Border(1, c.Theme().Border).Background(c.Theme().Background).Cursor(ui.CursorPointer)
	if control.Pressed() {
		control.Background(c.Theme().SurfacePressed)
	} else if control.Hovered() {
		control.Background(c.Theme().SurfaceHover)
	}
	control.Children(func() {
		ui.Text(c, chosen).FontSize(13).SingleLine().Grow(1).MinWidth(0)
		appIcon(c, "chevron-down").Size(16, 16).Shrink(0)
	})
	selectParts.Popup(func(panel ui.Element) {
		panel.Margin(4, 0, 0, 0).Padding(4).Radius(12).Material(panelMaterial(c.Theme())).Border(1, c.Theme().Border)
		ui.Scroll(c).MaxHeight(280).FillWidth().Gap(4).Children(func() {
			for i, name := range names {
				item := selectParts.Item(name).Height(32).FillWidth().Padding(4, 10).Radius(8).Cursor(ui.CursorPointer)
				if item.Highlighted() {
					item.Background(c.Theme().Selection)
				} else if options[i] == *value {
					item.Background(c.Theme().Surface)
				}
				item.Children(func() { ui.Text(c, name).FontSize(13).SingleLine() })
			}
		})
	})
	if control.Changed() {
		for i, name := range names {
			if name == chosen {
				*value = options[i]
				break
			}
		}
	}
	return control
}

func (a *App) smartRuleValue(c *ui.Context, rule *model.SmartRule, index int) {
	label := a.T("smart_rule_value", index+1)
	if rule.Field == "tag" {
		if len(a.Snapshot.Tags) == 0 {
			ui.Text(c, a.T("no_tags")).FontSize(12).TextColor(c.Theme().TextMuted).Grow(1).MinWidth(0)
			return
		}
		ids := make([]string, 0, len(a.Snapshot.Tags))
		names := map[string]string{}
		for _, tag := range a.Snapshot.Tags {
			ids = append(ids, tag.ID)
			names[tag.ID] = tag.Name
		}
		smartSelect(c, &rule.Value, ids, func(id string) string { return names[id] }).Grow(1).MinWidth(0).Label(label).Disabled(a.GBrowsing.Saving || len(ids) == 0)
		return
	}
	if rule.Field == "color" {
		ids := make([]string, 0, len(toolbarFamilies))
		for _, family := range toolbarFamilies {
			ids = append(ids, family.ID)
		}
		smartSelect(c, &rule.Value, ids, func(id string) string { return a.T("color_" + id) }).Grow(1).MinWidth(0).Label(label).Disabled(a.GBrowsing.Saving)
		return
	}
	placeholder := ""
	unit := ""
	switch rule.Field {
	case "added":
		placeholder = "YYYY-MM-DD"
	case "width", "height":
		unit = a.T("smart_pixels")
	case "size":
		unit = a.T("smart_megabytes")
	}
	ui.TextInput(c, &rule.Value).Grow(1).MinWidth(0).Label(label).Placeholder(placeholder).Disabled(a.GBrowsing.Saving)
	if unit != "" {
		ui.Text(c, unit).FontSize(12).TextColor(c.Theme().TextMuted).Shrink(0)
	}
}
func (a *App) smartDraft() (model.SmartFolder, error) {
	state := &a.GBrowsing
	folder := model.SmartFolder{ID: state.SmartID, Name: strings.TrimSpace(state.SmartName), Match: "all", Rules: slices.Clone(state.SmartRules)}
	if state.SmartMatch == 1 {
		folder.Match = "any"
	}
	if folder.Name == "" {
		return folder, i18n.NewError("smart_name_required")
	}
	for i := range folder.Rules {
		rule := &folder.Rules[i]
		if rule.Field != "size" {
			continue
		}
		mb, ok := new(big.Rat).SetString(strings.TrimSpace(rule.Value))
		if !ok || mb.Sign() < 0 || strings.Contains(rule.Value, "/") {
			return folder, i18n.NewError("error_smart_rule")
		}
		// 编辑用 MB，保存到最接近的字节；旧条件往返不丢精度。
		mb.Mul(mb, big.NewRat(1<<20, 1))
		bytes, err := strconv.ParseInt(mb.FloatString(0), 10, 64)
		if err != nil {
			return folder, i18n.NewError("error_smart_rule")
		}
		rule.Value = strconv.FormatInt(bytes, 10)
	}
	return folder, catalog.ValidateSmartFolder(folder)
}

func (a *App) saveSmartFolder() {
	if a.Library == nil || a.Busy || a.GBrowsing.Saving {
		return
	}
	folder, err := a.smartDraft()
	if err != nil {
		a.GBrowsing.SmartError = a.Texts.Error(err)
		return
	}
	lib, gen := a.Library, a.generation
	a.Busy = true
	a.GBrowsing.Saving = true
	a.GBrowsing.SmartError = ""
	go func() {
		a.workMu.Lock()
		defer a.workMu.Unlock()
		slog.Info("保存智能文件夹", "id", folder.ID)
		if folder.ID == "" {
			folder, err = lib.CreateSmartFolder(a.ctx, folder.Name, folder.Match, folder.Rules)
		} else {
			err = lib.UpdateSmartFolder(a.ctx, folder.ID, folder.Name, folder.Match, folder.Rules)
		}
		var snapshot model.Snapshot
		savedID := ""
		if err == nil {
			savedID = folder.ID
			snapshot, err = lib.Snapshot(a.ctx)
		}
		a.Window.Update(func() { a.finishSmartFolderSave(gen, savedID, snapshot, err) })
	}()
}
func (a *App) finishSmartFolderSave(generation uint64, id string, snapshot model.Snapshot, err error) {
	if generation != a.generation {
		return
	}
	a.Busy = false
	a.GBrowsing.Saving = false
	if err != nil {
		if id != "" {
			a.GBrowsing.SmartID = id
		}
		slog.Error("保存智能文件夹失败", "error", err)
		a.GBrowsing.SmartError = a.Texts.Error(err)
		return
	}
	a.applySnapshot(snapshot)
	a.GBrowsing.SmartOpen = false
	a.GBrowsing.SmartError = ""
	a.BrowseSmartFolder(id)
	a.drainPending()
}
