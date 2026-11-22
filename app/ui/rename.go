package ui

import (
	"context"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 预览最多列这么多行，再多的只说还有几个；规则对全部素材都算。
const renamePreviewRows = 100

type renameState struct {
	Open bool
	// IDs 是打开时选中的素材，按当时列表顺序；序号也按这个顺序编。
	IDs      []string
	Mode     int // 0 按格式，1 替换文字
	Template string
	Start    string
	Find     string
	With     string
	Regex    bool
}

// OpenRename 给这些素材打开重命名对话框；第一次打开时格式是原名，什么都不改，用户在上面加序号、日期等。
func (a *App) OpenRename(ids []string) {
	if a.Library == nil || len(ids) == 0 {
		return
	}
	// 上次用的方式和规则留着，连着给几批素材改名时不用重填；预览会先展示结果。
	previous := a.Rename
	a.Rename = renameState{Open: true, IDs: ids, Mode: previous.Mode, Template: previous.Template, Start: "1", Find: previous.Find, With: previous.With, Regex: previous.Regex}
	if strings.TrimSpace(a.Rename.Template) == "" {
		a.Rename.Template = catalog.TokenName
	}
}

// RenameAsset 给单张素材改名：输入框里预填原名，保存后直接用新名字。
func (a *App) RenameAsset(asset model.Asset) {
	lib := a.Library
	a.Ask("edit", a.T("rename"), asset.Name, func(name string) {
		if name == asset.Name {
			return
		}
		a.RunPatch("rename", []string{asset.ID}, func(ctx context.Context) error { return lib.RenameAsset(ctx, asset.ID, name) }, func(asset *model.Asset) { asset.Name = name })
	})
}

// renameAssets 是对话框打开时那批素材里还在图库中的，按打开时的顺序。
func (a *App) renameAssets() []model.Asset {
	assets := make([]model.Asset, 0, len(a.Rename.IDs))
	for _, id := range a.Rename.IDs {
		if asset, ok := a.assetByID[id]; ok {
			assets = append(assets, asset)
		}
	}
	return assets
}

// renamePlan 算出新名称和要提示的问题；problem 不为空时不能执行。
func (a *App) renamePlan(assets []model.Asset) (names []string, changed int, problem string) {
	state := &a.Rename
	rule := catalog.RenameRule{Replace: state.Mode == 1, Template: state.Template, Find: state.Find, With: state.With, Regex: state.Regex}
	if !rule.Replace && strings.Contains(state.Template, catalog.TokenNumber) {
		start, err := strconv.Atoi(strings.TrimSpace(state.Start))
		if err != nil || start < 0 {
			return nil, 0, a.T("rename_start_invalid")
		}
		rule.Start = start
	}
	tagNames := make(map[string]string, len(a.Snapshot.Tags))
	for _, tag := range a.Snapshot.Tags {
		tagNames[tag.ID] = tag.Name
	}
	names, err := catalog.BatchNames(assets, tagNames, rule)
	if err != nil {
		return nil, 0, a.Texts.Error(err)
	}
	empty := 0
	for i, name := range names {
		if name == "" {
			empty++
		} else if name != assets[i].Name {
			changed++
		}
	}
	if empty > 0 {
		return names, changed, a.T("rename_empty_names", empty)
	}
	return names, changed, ""
}

func (a *App) renameDialog(c *ui.Context) {
	state := &a.Rename
	if !state.Open {
		return
	}
	p := paletteFor(c.Theme())
	assets := a.renameAssets()
	// 换了图库或素材都被删了，就没有可改的了。
	if len(assets) == 0 {
		state.Open = false
		return
	}
	names, changed, problem := a.renamePlan(assets)
	dialogPanel(c.Key("rename"), &state.Open, panelWide, "edit", a.T("rename_title", len(assets)), func() {
		choiceMenu(c.Key("rename-mode"), a.T("rename_mode"), state.Mode, []string{a.T("rename_format"), a.T("rename_replace")}, func(i int) { state.Mode = i }).AlignSelf(ui.Start)
		if state.Mode == 0 {
			dialogSection(c, a.T("rename_template"), func() {
				fieldStyle(c, ui.TextInput(c.Key("rename-template"), &state.Template).Label(a.T("rename_template")).AutoFocus().FillWidth())
				ui.Row(c.Key("rename-tokens")).FillWidth().Wrap().Gap(space8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, a.T("rename_insert")).FontSize(fontSecondary).TextColor(p.TextSecondary)
					for _, token := range [][2]string{{catalog.TokenName, "rename_token_name"}, {catalog.TokenNumber, "rename_token_number"}, {catalog.TokenDate, "rename_token_date"}, {catalog.TokenTags, "rename_token_tags"}} {
						if settingsButton(c.Key(token[0]), a.T(token[1])).Tooltip(token[0]).Clicked() {
							state.Template += token[0]
						}
					}
				})
			})
			if strings.Contains(state.Template, catalog.TokenNumber) {
				dialogSection(c, a.T("rename_start"), func() {
					fieldStyle(c, ui.TextInput(c.Key("rename-start"), &state.Start).Label(a.T("rename_start")).FontFeatures("tnum").Width(120))
				})
			}
		} else {
			dialogSection(c, a.T("rename_find"), func() {
				fieldStyle(c, ui.TextInput(c.Key("rename-find"), &state.Find).Label(a.T("rename_find")).AutoFocus().FillWidth())
			})
			dialogSection(c, a.T("rename_with"), func() {
				fieldStyle(c, ui.TextInput(c.Key("rename-with"), &state.With).Label(a.T("rename_with")).FillWidth())
			})
			settingsCheckbox(c.Key("rename-regex"), &state.Regex, a.T("rename_regex"))
		}
		dialogSection(c, a.T("rename_preview"), func() {
			ui.Scroll(c.Key("rename-preview")).Scrollbars(false).MaxHeight(220).FillWidth().Radius(radiusControl).Background(p.Subtle).Border(1, p.BorderSubtle).Children(func() {
				for i, asset := range assets[:min(len(assets), renamePreviewRows)] {
					newName := asset.Name
					if names != nil {
						newName = names[i]
					}
					a.renamePreviewRow(c.Key(asset.ID), asset, newName)
				}
				if more := len(assets) - renamePreviewRows; more > 0 {
					ui.Text(c, a.T("rename_more", more)).FontSize(fontSecondary).TextColor(p.TextSecondary).Padding(space8, space12)
				}
			})
			if problem != "" {
				ui.Text(c, problem).FontSize(fontSecondary).TextColor(p.Danger)
			}
		})
	}, func() {
		if dialogButton(c, a.T("cancel"), false).Clicked() {
			state.Open = false
		}
		if dialogPrimaryButton(c, a.T("rename_apply"), problem != "" || changed == 0 || a.Busy).Clicked() {
			updates := make(map[string]string, changed)
			ids := make([]string, 0, changed)
			for i, asset := range assets {
				if names[i] != asset.Name {
					updates[asset.ID] = names[i]
					ids = append(ids, asset.ID)
				}
			}
			state.Open = false
			lib := a.Library
			a.RunPatch("rename", ids, func(ctx context.Context) error { return lib.RenameAssets(ctx, updates) }, func(asset *model.Asset) { asset.Name = updates[asset.ID] })
		}
	})
}

// 预览的一行：小缩略图、原名，箭头后面是新名；没变的新名用次要色。
func (a *App) renamePreviewRow(c *ui.Context, asset model.Asset, newName string) {
	p := paletteFor(c.Theme())
	ui.Row(c).FillWidth().Height(40).Padding(0, space12).Gap(space8).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(28, 28).Shrink(0).Radius(radiusSmall).Background(p.Raised).Clip().Children(func() {
			if bitmap := a.Bitmap(asset); bitmap != nil {
				ui.Image(c, bitmap).Fill().Fit(ui.Cover)
			}
		})
		ui.Text(c, asset.Name).FontSize(fontSecondary).TextColor(p.TextSecondary).SingleLine().Grow(1).Basis(0).MinWidth(0)
		appIcon(c, "arrow-right").Size(iconSmall, iconSmall).Shrink(0).TextColor(p.TextTertiary)
		color := p.Text
		if newName == asset.Name {
			color = p.TextSecondary
		}
		ui.Text(c, newName).FontSize(fontSecondary).FontWeight(500).TextColor(color).SingleLine().Grow(1).Basis(0).MinWidth(0)
	})
}
