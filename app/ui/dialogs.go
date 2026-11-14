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
	a.confirmationDialog(c)
	a.errorDialog(c)
}

func dialogHeading(c *ui.Context, icon, title string) {
	ui.Row(c).FillWidth().Gap(space8).AlignItems(ui.Center).Children(func() {
		appIcon(c, icon).Size(18, 18).TextColor(paletteFor(c.Theme()).TextSecondary)
		ui.Text(c, title).FontSize(16).FontWeight(600).TextColor(paletteFor(c.Theme()).Text).Grow(1)
	})
}

func (a *App) confirmationDialog(c *ui.Context) {
	if !a.ConfirmOpen {
		return
	}
	confirmed := false
	ui.Overlay(c.Key("confirmation"), func() {
		backdrop := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Material(frostedSurface{tint: paletteFor(c.Theme()).Scrim, blur: 12}).Modal()
		if backdrop.OverlayShortcut(0, ui.KeyEscape) {
			a.ConfirmOpen = false
		}
		backdrop.Children(func() {
			panel := ui.Column(c).Width(420).MaxWidthPercent(92).Padding(space24).Gap(space12).Radius(radiusPanel).
				Material(panelMaterial(c.Theme())).
				Role(ui.RoleAlertDialog).Label(a.ConfirmTitle).Description(a.ConfirmDetail)
			elevated(panel, c.Theme())
			dialogMotion(backdrop, panel)
			panel.Children(func() {
				dialogHeading(c, "warning", a.ConfirmTitle)
				if a.ConfirmDetail != "" {
					ui.Text(c, a.ConfirmDetail).FillWidth().TextColor(paletteFor(c.Theme()).TextSecondary)
				}
				ui.Row(c).FillWidth().Wrap().Justify(ui.End).Gap(space8).Margin(space8, 0, 0, 0).Children(func() {
					if settingsButton(c, a.T("cancel")).Clicked() {
						a.ConfirmOpen = false
					}
					confirm := primaryButton(c, a.ConfirmLabel, false).AutoFocus().TextColor(c.Theme().InverseText)
					color := paletteFor(c.Theme()).Danger
					if confirm.Pressed() {
						color = color.Mix(c.Theme().Text, .15)
					} else if confirm.Hovered() {
						color = color.Mix(c.Theme().Text, .08)
					}
					if confirm.Background(confirm.AnimateColor("danger-surface", color, motionColor)).Clicked() {
						a.ConfirmOpen = false
						confirmed = true
					}
				})
			})
		})
	})
	if confirmed {
		action := a.ConfirmAction
		a.ConfirmAction = nil
		if action != nil {
			action()
		}
	}
}

func (a *App) newLibraryDialog(c *ui.Context) {
	settingsPanel(c.Key("new-library"), &a.NewLibraryOpen, 460, func() {
		dialogHeading(c, "create-library", a.T("new_library"))
		ui.Text(c, a.T("new_library_detail")).TextColor(paletteFor(c.Theme()).TextSecondary)
		ui.Text(c, a.T("library_name")).FontWeight(600).Margin(space4, 0, 0, 0)
		name := ui.TextInput(c.Key("library-name"), &a.Dialog.NewName).Label(a.T("library_name")).AutoFocus().FillWidth().Disabled(a.Busy || a.Dialog.Picking)
		ui.Text(c, a.T("location")).FontWeight(600).Margin(space4, 0, 0, 0)
		location := a.Dialog.NewPath
		if location == "" {
			location = a.T("choose_library_location")
		}
		ui.Text(c, location).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary).SingleLine().Tooltip(location)
		if settingsButton(c, a.T("choose_location")).FillWidth().Disabled(a.Busy || a.Dialog.Picking || strings.TrimSpace(a.Dialog.NewName) == "").Clicked() {
			a.pickLibraryLocation()
		}
		ui.Row(c).Gap(space8).Justify(ui.End).Margin(space8, 0, 0, 0).Children(func() {
			if settingsButton(c, a.T("cancel")).Disabled(a.Busy || a.Dialog.Picking).Clicked() {
				a.NewLibraryOpen = false
			}
			ready := !a.Busy && !a.Dialog.Picking && strings.TrimSpace(a.Dialog.NewName) != "" && a.Dialog.NewPath != ""
			create := primaryButton(c, a.T("create_library"), !ready)
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
		dialogHeading(c, "export", a.T("export"))
		ui.Text(c, a.T("export_count", a.Selected.Len())).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
		for _, asset := range a.SelectedAssets() {
			if asset.Rotation != 0 {
				ui.Text(c, a.T("export_rotation_metadata_hint")).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
				break
			}
		}
		ui.Text(c, a.T("export_format")).FontWeight(600).Margin(space4, 0, 0, 0)
		ui.RadioGroup(c, func() {
			animatedRadio(c, &a.Dialog.Format, model.ExportOriginal, a.T("export_original"))
			animatedRadio(c, &a.Dialog.Format, model.ExportJPEG, a.T("export_jpeg"))
			animatedRadio(c, &a.Dialog.Format, model.ExportPNG, a.T("export_png"))
		}).Label(a.T("export_format")).Disabled(a.Dialog.Picking)
		if a.Dialog.Format == model.ExportJPEG {
			fadeMotion(ui.Row(c.Key("jpeg-quality-title")).Margin(space4, 0, 0, 0)).Children(func() {
				ui.Text(c, a.T("jpeg_quality")).FontWeight(600).Grow(1)
				ui.Text(c, fmt.Sprintf("%.0f%%", a.Dialog.Quality*100)).Font(monoFont(c)).FontSize(fontNumber).TextColor(paletteFor(c.Theme()).TextSecondary)
			})
			fadeMotion(ui.Slider(c.Key("jpeg-quality"), &a.Dialog.Quality, 0.01, 1).Label(a.T("jpeg_quality")).FillWidth()).Disabled(a.Dialog.Picking)
		}
		ui.Row(c).Gap(space8).Justify(ui.End).Margin(space8, 0, 0, 0).Children(func() {
			if settingsButton(c, a.T("cancel")).Disabled(a.Dialog.Picking).Clicked() {
				a.ExportOpen = false
			}
			if primaryButton(c, a.T("export"), a.Busy || a.Dialog.Picking || a.Selected.Len() == 0).Clicked() {
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
		ui.Text(c, a.PromptTitle).FontSize(16).FontWeight(600).TextColor(paletteFor(c.Theme()).Text)
		input := ui.TextInput(c.Key("prompt-value"), &a.PromptValue).Label(a.PromptTitle).AutoFocus().FillWidth()
		ui.Row(c).Gap(space8).Justify(ui.End).Margin(space8, 0, 0, 0).Children(func() {
			if settingsButton(c, a.T("cancel")).Clicked() {
				a.PromptOpen = false
				a.PromptSubmit = nil
			}
			ready := strings.TrimSpace(a.PromptValue) != ""
			submit := primaryButton(c, a.T("save"), !ready)
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
		dialogHeading(c, "warning", a.T("operation_failed"))
		ui.Scroll(c).MaxHeight(180).Children(func() {
			ui.Text(c, a.Error).Selectable().TextColor(paletteFor(c.Theme()).Danger)
		})
		ui.Text(c, a.T("log_location")).FontWeight(600).Margin(space4, 0, 0, 0)
		ui.Text(c, logging.Directory()).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary).Selectable()
		ui.Row(c).Gap(space8).Justify(ui.End).Margin(space8, 0, 0, 0).Children(func() {
			if settingsButton(c, a.T("copy_error")).Clicked() {
				details := a.Error + "\n" + a.T("log_location") + ": " + logging.Directory()
				c.WriteClipboardThen(details, func(err error) {
					if err != nil {
						a.ShowError(err)
					} else {
						a.Notify(a.T("copied"))
					}
				})
			}
			if primaryButton(c, a.T("close"), false).Clicked() {
				open = false
			}
		})
	})
	if !open {
		a.Error = ""
	}
}
