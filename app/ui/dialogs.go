package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/logging"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type dialogsState struct {
	NewName       string
	NewPath       string
	Picking       bool
	NewWasOpen    bool
	ExportWasOpen bool
	Format        model.ExportFormat
	Quality       float64
}

func (a *App) dialogs(c *ui.Context) {
	if a.NewLibraryOpen && !a.Dialog.NewWasOpen {
		a.Dialog.NewName = a.T("untitled_library")
		a.Dialog.NewPath = ""
	}
	a.Dialog.NewWasOpen = a.NewLibraryOpen
	if a.ExportOpen && !a.Dialog.ExportWasOpen {
		a.Dialog.Format = model.ExportOriginal
		a.Dialog.Quality = 0.9
	}
	a.Dialog.ExportWasOpen = a.ExportOpen
	a.newLibraryDialog(c)
	a.exportDialog(c)
	a.promptDialog(c)
	if ui.AlertDialog(c.Key("confirmation"), &a.ConfirmOpen, a.ConfirmTitle, a.ConfirmDetail, a.T("cancel"), a.T("confirm")) == 1 {
		action := a.ConfirmAction
		a.ConfirmAction = nil
		if action != nil {
			action()
		}
	}
	a.errorDialog(c)
}

func (a *App) newLibraryDialog(c *ui.Context) {
	settingsPanel(c.Key("new-library"), &a.NewLibraryOpen, 460, func() {
		ui.Text(c, a.T("new_library")).FontSize(20).Bold()
		ui.Text(c, a.T("new_library_detail")).TextColor(c.Theme().TextMuted)
		ui.Text(c, a.T("library_name")).Bold().Margin(4, 0, 0, 0)
		name := ui.TextInput(c.Key("library-name"), &a.Dialog.NewName).Label(a.T("library_name")).AutoFocus().FillWidth().Disabled(a.Busy || a.Dialog.Picking)
		ui.Text(c, a.T("location")).Bold().Margin(4, 0, 0, 0)
		location := a.Dialog.NewPath
		if location == "" {
			location = a.T("choose_library_location")
		}
		ui.Text(c, location).FontSize(12).TextColor(c.Theme().TextMuted).SingleLine().Tooltip(location)
		if ui.Button(c, a.T("choose_location")).FillWidth().Disabled(a.Busy || a.Dialog.Picking || strings.TrimSpace(a.Dialog.NewName) == "").Clicked() {
			a.pickLibraryLocation()
		}
		ui.Row(c).Gap(8).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
			if ui.Button(c, a.T("cancel")).Disabled(a.Busy || a.Dialog.Picking).Clicked() {
				a.NewLibraryOpen = false
			}
			ready := !a.Busy && !a.Dialog.Picking && strings.TrimSpace(a.Dialog.NewName) != "" && a.Dialog.NewPath != ""
			create := ui.PrimaryButton(c, a.T("create_library")).Disabled(!ready)
			if ready && (create.Clicked() || name.Submitted()) {
				a.CreateLibraryAt(a.Dialog.NewPath, strings.TrimSpace(a.Dialog.NewName))
			}
		})
	})
}

func (a *App) pickLibraryLocation() {
	a.Dialog.Picking = true
	title, name := a.T("choose_library_location"), libraryPackageName(a.Dialog.NewName)
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.Window, Title: title, DefaultPath: name, CreateDirectories: true,
			Filters: []mygo.FileFilter{{Name: "Momento", Extensions: []string{"momento"}}},
		})
		a.Window.Update(func() {
			a.Dialog.Picking = false
			if err != nil {
				a.ShowError(err)
			} else if path != "" {
				if !strings.EqualFold(filepath.Ext(path), ".momento") {
					path += ".momento"
				}
				a.Dialog.NewPath = path
			}
		})
	}()
}

func libraryPackageName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	name = strings.Trim(name, ". ")
	return name + ".momento"
}

func (a *App) exportDialog(c *ui.Context) {
	settingsPanel(c.Key("export"), &a.ExportOpen, 420, func() {
		ui.Text(c, a.T("export")).FontSize(20).Bold()
		ui.Text(c, a.T("export_count", a.Selected.Len())).TextColor(c.Theme().TextMuted)
		for _, asset := range a.SelectedAssets() {
			if asset.Rotation != 0 {
				ui.Text(c, a.T("export_rotation_metadata_hint")).FontSize(12).TextColor(c.Theme().TextMuted)
				break
			}
		}
		ui.Text(c, a.T("export_format")).Bold().Margin(4, 0, 0, 0)
		ui.RadioGroup(c, func() {
			ui.Radio(c, &a.Dialog.Format, model.ExportOriginal, a.T("export_original"))
			ui.Radio(c, &a.Dialog.Format, model.ExportJPEG, a.T("export_jpeg"))
			ui.Radio(c, &a.Dialog.Format, model.ExportPNG, a.T("export_png"))
		}).Label(a.T("export_format")).Disabled(a.Dialog.Picking)
		if a.Dialog.Format == model.ExportJPEG {
			ui.Row(c).Margin(4, 0, 0, 0).Children(func() {
				ui.Text(c, a.T("jpeg_quality")).Bold().Grow(1)
				ui.Text(c, fmt.Sprintf("%.0f%%", a.Dialog.Quality*100)).TextColor(c.Theme().TextMuted)
			})
			ui.Slider(c.Key("jpeg-quality"), &a.Dialog.Quality, 0.01, 1).Label(a.T("jpeg_quality")).FillWidth().Disabled(a.Dialog.Picking)
		}
		ui.Row(c).Gap(8).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
			if ui.Button(c, a.T("cancel")).Disabled(a.Dialog.Picking).Clicked() {
				a.ExportOpen = false
			}
			if ui.PrimaryButton(c, a.T("export")).Disabled(a.Busy || a.Dialog.Picking || a.Selected.Len() == 0).Clicked() {
				a.pickExportDestination()
			}
		})
	})
}

func (a *App) pickExportDestination() {
	items := a.SelectedAssets()
	if len(items) == 0 {
		return
	}
	a.Dialog.Picking = true
	title, format, quality, generation := a.T("export"), a.Dialog.Format, a.Dialog.Quality, a.generation
	name := exportDialogFileName(items[0], format)
	go func() {
		var path string
		var err error
		if len(items) == 1 {
			path, err = mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.Window, Title: title, DefaultPath: name, CreateDirectories: true})
		} else {
			var paths []string
			paths, err = mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.Window, Title: title, Directory: true, CreateDirectories: true})
			if len(paths) > 0 {
				path = paths[0]
			}
		}
		a.Window.Update(func() {
			a.Dialog.Picking = false
			if err != nil {
				a.ShowError(err)
			} else if path != "" && generation == a.generation {
				a.ExportOpen = false
				a.ExportTo(path, format, quality)
			}
		})
	}()
}

func exportDialogFileName(asset model.Asset, format model.ExportFormat) string {
	name := strings.TrimSuffix(libraryPackageName(asset.Name), ".momento")
	extension := asset.Extension
	if format == model.ExportOriginal && asset.OriginalFileName != "" {
		return asset.OriginalFileName
	}
	if format == model.ExportJPEG {
		extension = "jpg"
	} else if format == model.ExportPNG {
		extension = "png"
	}
	return name + "." + extension
}

func (a *App) promptDialog(c *ui.Context) {
	settingsPanel(c.Key("text-prompt"), &a.PromptOpen, 420, func() {
		ui.Text(c, a.PromptTitle).FontSize(20).Bold()
		input := ui.TextInput(c.Key("prompt-value"), &a.PromptValue).Label(a.PromptTitle).AutoFocus().FillWidth()
		ui.Row(c).Gap(8).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
			if ui.Button(c, a.T("cancel")).Clicked() {
				a.PromptOpen = false
				a.PromptSubmit = nil
			}
			ready := strings.TrimSpace(a.PromptValue) != ""
			submit := ui.PrimaryButton(c, a.T("save")).Disabled(!ready)
			if ready && (submit.Clicked() || input.Submitted()) {
				a.PromptOpen = false
				callback := a.PromptSubmit
				a.PromptSubmit = nil
				if callback != nil {
					callback(strings.TrimSpace(a.PromptValue))
				}
			}
		})
	})
}

func (a *App) errorDialog(c *ui.Context) {
	open := a.Error != ""
	settingsPanel(c.Key("error"), &open, 500, func() {
		ui.Text(c, a.T("operation_failed")).FontSize(20).Bold()
		ui.Scroll(c).MaxHeight(180).Children(func() {
			ui.Text(c, a.Error).Selectable().TextColor(c.Theme().Danger)
		})
		ui.Text(c, a.T("log_location")).Bold().Margin(4, 0, 0, 0)
		ui.Text(c, logging.Directory()).FontSize(12).TextColor(c.Theme().TextMuted).Selectable()
		ui.Row(c).Gap(8).Justify(ui.End).Margin(8, 0, 0, 0).Children(func() {
			if ui.Button(c, a.T("copy_error")).Clicked() {
				details := a.Error + "\n" + a.T("log_location") + ": " + logging.Directory()
				c.WriteClipboardThen(details, func(err error) {
					if err != nil {
						a.ShowError(err)
					} else {
						a.Notify(a.T("copied"))
					}
				})
			}
			if ui.PrimaryButton(c, a.T("close")).Clicked() {
				open = false
			}
		})
	})
	if !open {
		a.Error = ""
	}
}
