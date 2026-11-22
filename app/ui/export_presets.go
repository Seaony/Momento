package ui

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 最长边最多这么多像素；再大就和原尺寸没区别了，多半是输错。
const maxExportEdge = 100000

// ExportPreset 是存下来的一组导出设置，存在偏好里，导出对话框顶部点一下就套用。
type ExportPreset struct {
	Name string `json:"name"`
	model.ExportOptions
}

// exportOptions 是对话框里现在的设置；problem 不为空时不能导出，也不能存成预设。
func (a *App) exportOptions() (options model.ExportOptions, problem string) {
	d := &a.Dialog
	options = model.ExportOptions{Format: d.Format, Motion: d.ExportMotion}
	if d.Format == model.ExportJPEG {
		options.Quality = d.Quality
	}
	if d.Format != model.ExportOriginal && d.MaxEdgeOn {
		edge, err := strconv.Atoi(strings.TrimSpace(d.MaxEdge))
		if err != nil || edge <= 0 || edge > maxExportEdge {
			return options, a.T("export_max_edge_invalid", maxExportEdge)
		}
		options.MaxEdge = edge
	}
	return options, ""
}

// exportSizeSection 是转格式时的尺寸：勾上「限制最长边」后填像素数，比它小的图不放大。
func (a *App) exportSizeSection(c *ui.Context) {
	p := paletteFor(c.Theme())
	dialogSection(c, a.T("export_size"), func() {
		ui.Row(c.Key("export-size")).FillWidth().Gap(space8).AlignItems(ui.Center).Children(func() {
			settingsCheckbox(c.Key("export-max-edge-on"), &a.Dialog.MaxEdgeOn, a.T("export_max_edge")).Disabled(a.Dialog.Picking)
			if a.Dialog.MaxEdgeOn {
				fieldStyle(c, ui.TextInput(c.Key("export-max-edge"), &a.Dialog.MaxEdge).Label(a.T("export_max_edge")).FontFeatures("tnum").Width(88)).Disabled(a.Dialog.Picking)
				ui.Text(c, a.T("export_pixels")).FontSize(fontSecondary).TextColor(p.TextSecondary)
			}
		})
		if _, problem := a.exportOptions(); problem != "" {
			ui.Text(c, problem).FontSize(fontSecondary).TextColor(p.Danger)
		}
	})
}

// exportPresetChips 列出存下的预设；和当前设置一样的那个高亮。右键可以删除。
func (a *App) exportPresetChips(c *ui.Context) {
	presets := a.Preferences.ExportPresets
	if len(presets) == 0 {
		return
	}
	// 设置有误时不高亮任何预设：半截的设置会和不限尺寸的预设看起来一样。
	current, problem := a.exportOptions()
	dialogSection(c, a.T("export_presets"), func() {
		ui.Row(c.Key("export-presets")).FillWidth().Wrap().Gap(space8).Children(func() {
			remove := -1
			for i, preset := range presets {
				on := problem == "" && sameExportOptions(preset.ExportOptions, current)
				chip := popChip(c.Key(preset.Name), &on, "", preset.Name).Disabled(a.Dialog.Picking)
				if chip.Changed() {
					a.applyExportPreset(preset)
				}
				chip.ContextMenu(func(m *ui.Menu) {
					if menuItem(m, a.T("export_delete_preset"), "trash").Chosen() {
						remove = i
					}
				})
			}
			if remove >= 0 {
				a.Preferences.ExportPresets = slices.Delete(slices.Clone(presets), remove, remove+1)
				a.queuePreferences()
			}
		})
	})
}

func sameExportOptions(a, b model.ExportOptions) bool {
	return a.Format == b.Format && a.MaxEdge == b.MaxEdge && a.Motion == b.Motion &&
		(a.Format != model.ExportJPEG || math.Abs(a.Quality-b.Quality) < .005)
}

func (a *App) applyExportPreset(preset ExportPreset) {
	d := &a.Dialog
	d.Format, d.ExportMotion = preset.Format, preset.Motion
	if preset.Format == model.ExportJPEG && preset.Quality > 0 {
		d.Quality = preset.Quality
	}
	d.MaxEdgeOn = preset.MaxEdge > 0
	if d.MaxEdgeOn {
		d.MaxEdge = strconv.Itoa(preset.MaxEdge)
	}
}

// saveExportPreset 把当前设置存成预设；重名的覆盖，位置不变。
func (a *App) saveExportPreset(name string) {
	name = strings.TrimSpace(name)
	options, problem := a.exportOptions()
	if name == "" || problem != "" {
		return
	}
	preset := ExportPreset{Name: name, ExportOptions: options}
	presets := slices.Clone(a.Preferences.ExportPresets)
	if i := slices.IndexFunc(presets, func(p ExportPreset) bool { return p.Name == name }); i >= 0 {
		presets[i] = preset
	} else {
		presets = append(presets, preset)
	}
	a.Preferences.ExportPresets = presets
	a.queuePreferences()
}
