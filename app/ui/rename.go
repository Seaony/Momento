package ui

import (
	"context"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
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

	generation uint64
	source     *renameSource
	preview    *renamePreview
	pending    *renameRequest
	plan       *renameResult
}

// OpenRename 给这些素材打开重命名对话框；第一次打开时格式是原名，什么都不改，用户在上面加序号、日期等。
func (a *App) OpenRename(ids []string) {
	if a.Library == nil || len(ids) == 0 {
		return
	}
	// 上次用的方式和规则留着，连着给几批素材改名时不用重填；预览会先展示结果。
	previous := a.Rename
	a.Rename = renameState{Open: true, IDs: ids, Mode: previous.Mode, Template: previous.Template, Start: "1", Find: previous.Find, With: previous.With, Regex: previous.Regex, generation: a.generation}
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

type renameInputs struct {
	mode                        int
	template, start, find, with string
	regex                       bool
	texts                       i18n.Catalog
}

type renameSource struct {
	version  uint64
	assets   []model.Asset
	tagNames map[string]string
}

type renamePreview struct {
	inputs  renameInputs
	names   []string
	problem string
}

type renameRequest struct {
	inputs renameInputs
	source *renameSource
}

type renameResult struct {
	ids     []string
	updates map[string]string
	problem string
}

func (a *App) renameInputs() renameInputs {
	state := &a.Rename
	return renameInputs{state.Mode, state.Template, state.Start, state.Find, state.With, state.Regex, a.Texts}
}

func (a *App) renameTagNames() map[string]string {
	tagNames := make(map[string]string, len(a.Snapshot.Tags))
	for _, tag := range a.Snapshot.Tags {
		tagNames[tag.ID] = tag.Name
	}
	return tagNames
}

func (inputs renameInputs) batchNames(assets []model.Asset, tagNames map[string]string, total int) ([]string, string) {
	rule := catalog.RenameRule{Replace: inputs.mode == 1, Template: inputs.template, Find: inputs.find, With: inputs.with, Regex: inputs.regex}
	numbered := !rule.Replace && strings.Contains(rule.Template, catalog.TokenNumber)
	if numbered {
		start, err := strconv.Atoi(strings.TrimSpace(inputs.start))
		if err != nil || start < 0 {
			return nil, inputs.texts.T("rename_start_invalid")
		}
		rule.Start = start
	}
	if !numbered || total == len(assets) {
		names, err := catalog.BatchNames(assets, tagNames, rule)
		return names, inputs.texts.Error(err)
	}

	// 只算预览的几行，但序号仍按整批的最后一个序号补零。
	width := len(strconv.Itoa(rule.Start + total - 1))
	names := make([]string, len(assets))
	for index := range assets {
		number := strconv.Itoa(rule.Start + index)
		number = strings.Repeat("0", max(width-len(number), 0)) + number
		rowRule := rule
		rowRule.Template = strings.ReplaceAll(rule.Template, catalog.TokenNumber, number)
		rowNames, err := catalog.BatchNames(assets[index:index+1], tagNames, rowRule)
		if err != nil {
			return nil, inputs.texts.Error(err)
		}
		names[index] = rowNames[0]
	}
	return names, ""
}

func (request *renameRequest) calculate() *renameResult {
	assets := request.source.assets
	names, problem := request.inputs.batchNames(assets, request.source.tagNames, len(assets))
	if problem != "" {
		return &renameResult{problem: problem}
	}
	empty, changed := 0, 0
	for index, name := range names {
		if name == "" {
			empty++
		} else if name != assets[index].Name {
			changed++
		}
	}
	if empty > 0 {
		return &renameResult{problem: request.inputs.texts.T("rename_empty_names", empty)}
	}
	if changed == 0 {
		return &renameResult{}
	}
	result := &renameResult{ids: make([]string, 0, changed), updates: make(map[string]string, changed)}
	for index, asset := range assets {
		if name := names[index]; name != asset.Name {
			result.updates[asset.ID] = name
			result.ids = append(result.ids, asset.ID)
		}
	}
	return result
}

func (state *renameState) close() {
	state.Open = false
	state.IDs, state.source, state.preview, state.pending, state.plan = nil, nil, nil, nil, nil
}

func (a *App) requestRenamePlan(post func(func())) {
	state := &a.Rename
	request := &renameRequest{inputs: state.preview.inputs, source: state.source}
	state.pending = request
	go func() {
		result := request.calculate()
		post(func() {
			state := &a.Rename
			if state.pending != request {
				return
			}
			state.pending = nil
			if state.Open && state.generation == a.generation && a.snapshotVersion == request.source.version && a.renameInputs() == request.inputs {
				state.plan = result
			}
		})
	}()
}

func (a *App) renameDialog(c *ui.Context) {
	var post func(func())
	if a.Window != nil {
		post = a.Window.Update
	}
	a.renderRenameDialog(c, post)
}

func (a *App) renderRenameDialog(c *ui.Context, post func(func())) {
	state := &a.Rename
	if !state.Open || a.Library == nil || state.generation != a.generation {
		state.close()
		return
	}
	if state.source == nil || state.source.version != a.snapshotVersion {
		// 素材表会原地更新。每版只取一份选区副本，输入时直接复用。
		state.source = &renameSource{version: a.snapshotVersion, assets: a.renameAssets(), tagNames: a.renameTagNames()}
		state.preview, state.plan = nil, nil
	}
	total := len(state.source.assets)
	if total == 0 {
		state.close()
		return
	}
	assets := state.source.assets[:min(total, renamePreviewRows)]
	p := paletteFor(c.Theme())
	dialogPanel(c.Key("rename"), &state.Open, panelWide, "edit", a.T("rename_title", total), func() {
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
		inputs := a.renameInputs()
		if state.preview == nil || state.preview.inputs != inputs {
			names, problem := inputs.batchNames(assets, state.source.tagNames, total)
			state.preview = &renamePreview{inputs: inputs, names: names, problem: problem}
			state.plan = nil
		}
		dialogSection(c, a.T("rename_preview"), func() {
			ui.Scroll(c.Key("rename-preview")).Scrollbars(false).MaxHeight(220).FillWidth().Radius(radiusControl).Background(p.Subtle).Border(1, p.BorderSubtle).Children(func() {
				for i, asset := range assets {
					newName := asset.Name
					if names := state.preview.names; names != nil {
						newName = names[i]
					}
					a.renamePreviewRow(c.Key(asset.ID), asset, newName)
				}
				if more := total - len(assets); more > 0 {
					ui.Text(c, a.T("rename_more", more)).FontSize(fontSecondary).TextColor(p.TextSecondary).Padding(space8, space12)
				}
			})
			problem := state.preview.problem
			if state.plan != nil {
				problem = state.plan.problem
			}
			if problem != "" {
				ui.Text(c, problem).FontSize(fontSecondary).TextColor(p.Danger)
			}
		})
	}, func() {
		if dialogButton(c, a.T("cancel"), false).Clicked() {
			state.Open = false
		}
		plan := state.plan
		ready := state.Open && plan != nil && len(plan.ids) > 0 && !a.Busy
		apply := dialogPrimaryButton(c, a.T("rename_apply"), !ready)
		if ready && apply.Clicked() {
			state.Open = false
			lib := a.Library
			updates := plan.updates
			a.RunPatch("rename", plan.ids, func(ctx context.Context) error { return lib.RenameAssets(ctx, updates) }, func(asset *model.Asset) { asset.Name = updates[asset.ID] })
		}
	})
	if !state.Open {
		state.close()
	} else if post != nil && state.preview.problem == "" && state.pending == nil && state.plan == nil {
		a.requestRenamePlan(post)
	}
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
