package ui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/logging"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type dialogsState struct {
	NewName string
	NewIcon string
	// NewDir 是图库要放进的文件夹；包名跟着图库名称走，改名时不用重新选位置。
	NewDir         string
	Picking        bool
	NewWasOpen     bool
	IconPickerOpen bool
	EditName       string
	EditIcon       string
	EditWasOpen    bool
	ExportWasOpen  bool
	Folder         folderDialogState
	Format         model.ExportFormat
	Quality        float64
	ExportMotion   bool // 实况照片连视频一起导出
}

func (a *App) dialogs(c *ui.Context) {
	if a.NewLibraryOpen && !a.Dialog.NewWasOpen {
		a.Dialog.NewName, a.Dialog.NewIcon, a.Dialog.NewDir = a.T("untitled_library"), "", defaultLibraryDir()
		a.Dialog.IconPickerOpen = false
	}
	a.Dialog.NewWasOpen = a.NewLibraryOpen
	if a.EditLibraryOpen && !a.Dialog.EditWasOpen {
		a.Dialog.EditName, a.Dialog.EditIcon = a.libraryName(), a.manifest.Icon
		a.Dialog.IconPickerOpen = false
	}
	a.Dialog.EditWasOpen = a.EditLibraryOpen
	if a.ExportOpen && !a.Dialog.ExportWasOpen {
		a.Dialog.Format = model.ExportOriginal
		a.Dialog.Quality = 0.9
		a.Dialog.ExportMotion = false
	}
	a.Dialog.ExportWasOpen = a.ExportOpen
	a.newLibraryDialog(c)
	a.editLibraryDialog(c)
	a.folderDialog(c)
	a.exportDialog(c)
	a.renameDialog(c)
	a.similarDialog(c)
	a.promptDialog(c)
	a.confirmationDialog(c)
	a.errorDialog(c)
}

func (a *App) confirmationDialog(c *ui.Context) {
	if !a.ConfirmOpen {
		return
	}
	confirmed := false
	ui.Overlay(c.Key("confirmation"), func() {
		backdrop := ui.Column(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(paletteFor(c.Theme()).Scrim).Modal()
		if backdrop.OverlayShortcut(0, ui.KeyEscape) {
			a.ConfirmOpen = false
		}
		backdrop.Children(func() {
			panel := dialogSurface(c, ui.Column(c).Width(panelDialog)).Role(ui.RoleAlertDialog).Label(a.ConfirmTitle).Description(a.ConfirmDetail)
			dialogMotion(backdrop, panel)
			panel.Children(func() {
				var body func()
				if a.ConfirmDetail != "" {
					body = func() { ui.Text(c, a.ConfirmDetail).FillWidth().TextColor(paletteFor(c.Theme()).TextSecondary) }
				}
				dialogLayout(c, "trash", a.ConfirmTitle, body, func() {
					if dialogButton(c, a.T("cancel"), false).Clicked() {
						a.ConfirmOpen = false
					}
					if dialogDangerButton(c, a.ConfirmLabel).AutoFocus().Clicked() {
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
	p := paletteFor(c.Theme())
	disabled := a.Busy || a.Dialog.Picking
	name := strings.TrimSpace(a.Dialog.NewName)
	ready := !disabled && name != "" && a.Dialog.NewDir != ""
	var input ui.Element
	dialogPanel(c.Key("new-library"), &a.NewLibraryOpen, panelDialog, "create-library", a.T("new_library"), func() {
		dialogSection(c, a.T("library_name"), func() {
			input = a.libraryNameField(c, &a.Dialog.NewName, &a.Dialog.NewIcon, &a.Dialog.IconPickerOpen, disabled).AutoFocus()
		})
		dialogSection(c, a.T("location"), func() {
			// 整行点了就换位置；显示最终会建出来的图库包路径，跟着名称变。
			location := a.T("choose_library_location")
			if a.Dialog.NewDir != "" {
				location = libraryDisplayPath(filepath.Join(a.Dialog.NewDir, libraryPackageName(name)))
			}
			row := menuFieldStyle(c, ui.ButtonBase(c.Key("library-location")).FillWidth().Height(controlHeight).Padding(0, space12).Gap(space8).
				AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(a.T("choose_location")).Tooltip(location).Disabled(disabled))
			row.Children(func() {
				appIcon(c, "folder").Size(iconMedium, iconMedium).TextColor(p.TextSecondary).Shrink(0)
				ui.Text(c, location).FontSize(fontSecondary).TextColor(p.TextSecondary).SingleLine().Grow(1).MinWidth(0)
				ui.Text(c, a.T("change_location")).FontSize(fontSecondary).FontWeight(500).TextColor(p.Text).Shrink(0)
			})
			if row.Clicked() {
				a.pickLibraryLocation()
			}
		})
	}, func() {
		if dialogButton(c, a.T("cancel"), disabled).Clicked() {
			a.NewLibraryOpen = false
		}
		create := dialogPrimaryButton(c, a.T("create_library"), !ready)
		if ready && (create.Clicked() || input.Submitted()) {
			a.CreateLibraryAt(filepath.Join(a.Dialog.NewDir, libraryPackageName(name)), name, a.Dialog.NewIcon)
		}
	})
}

func (a *App) editLibraryDialog(c *ui.Context) {
	name := strings.TrimSpace(a.Dialog.EditName)
	ready := !a.Busy && name != ""
	var input ui.Element
	dialogPanel(c.Key("edit-library"), &a.EditLibraryOpen, panelDialog, "pencil-simple", a.T("edit_library"), func() {
		dialogSection(c, a.T("library_name"), func() {
			input = a.libraryNameField(c, &a.Dialog.EditName, &a.Dialog.EditIcon, &a.Dialog.IconPickerOpen, a.Busy).AutoFocus()
		})
	}, func() {
		if dialogButton(c, a.T("cancel"), false).Clicked() {
			a.EditLibraryOpen = false
		}
		save := dialogPrimaryButton(c, a.T("save"), !ready)
		if ready && (save.Clicked() || input.Submitted()) {
			a.EditLibraryOpen = false
			a.EditLibrary(name, a.Dialog.EditIcon)
		}
	})
}

// 对话框里一组：小标题在上，控件在下。
func dialogSection(c *ui.Context, title string, body func()) {
	ui.Column(c).FillWidth().Gap(space8).Children(func() {
		ui.Text(c, title).FontSize(fontSecondary).FontWeight(600).TextColor(paletteFor(c.Theme()).TextSecondary)
		body()
	})
}

// 新建图库默认放进「图片」文件夹；没有就放用户目录。读不到用户目录时留空，让用户自己选。
func defaultLibraryDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		slog.Error("读取用户目录失败", "error", err)
		return ""
	}
	pictures := filepath.Join(home, "Pictures")
	if info, err := os.Stat(pictures); err == nil && info.IsDir() {
		return pictures
	}
	return home
}

func (a *App) pickLibraryLocation() {
	a.Dialog.Picking = true
	title, dir := a.T("choose_library_location"), a.Dialog.NewDir
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: a.Window, Title: title, DefaultPath: dir, Directory: true, CreateDirectories: true,
		})
		a.Window.Update(func() {
			a.Dialog.Picking = false
			if err != nil {
				a.ShowError(err)
			} else if len(paths) > 0 {
				a.Dialog.NewDir = paths[0]
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
	p := paletteFor(c.Theme())
	dialogPanel(c.Key("export"), &a.ExportOpen, panelDialog, "export", a.T("export"), func() {
		ui.Column(c).FillWidth().Gap(space4).Children(func() {
			ui.Text(c, a.T("export_count", a.Selected.Len())).TextColor(p.TextSecondary)
			for _, asset := range a.SelectedAssets() {
				if asset.Rotation != 0 {
					ui.Text(c, a.T("export_rotation_metadata_hint")).FontSize(fontSecondary).TextColor(p.TextSecondary)
					break
				}
			}
		})
		dialogSection(c, a.T("export_format"), func() {
			ui.RadioGroup(c, func() {
				animatedRadio(c, &a.Dialog.Format, model.ExportOriginal, a.T("export_original"))
				animatedRadio(c, &a.Dialog.Format, model.ExportJPEG, a.T("export_jpeg"))
				animatedRadio(c, &a.Dialog.Format, model.ExportPNG, a.T("export_png"))
			}).Label(a.T("export_format")).Disabled(a.Dialog.Picking)
		})
		if a.Dialog.Format == model.ExportJPEG {
			fadeMotion(ui.Column(c.Key("jpeg-quality")).FillWidth().Gap(space8)).Children(func() {
				ui.Row(c).FillWidth().Children(func() {
					ui.Text(c, a.T("jpeg_quality")).FontSize(fontSecondary).FontWeight(600).TextColor(p.TextSecondary).Grow(1)
					ui.Text(c, fmt.Sprintf("%.0f%%", a.Dialog.Quality*100)).FontFeatures("tnum").FontSize(fontNumber).TextColor(p.TextSecondary)
				})
				ui.Slider(c.Key("jpeg-quality-slider"), &a.Dialog.Quality, 0.01, 1).Label(a.T("jpeg_quality")).FillWidth().Disabled(a.Dialog.Picking)
			})
		}
		// 选中的有实况照片才出现：默认只导出图片，勾上后视频和图片同名放在一起。
		if slices.ContainsFunc(a.SelectedAssets(), isLiveAsset) {
			settingsCheckbox(c.Key("export-live-motion"), &a.Dialog.ExportMotion, a.T("export_live_motion")).Disabled(a.Dialog.Picking)
		}
	}, func() {
		if dialogButton(c, a.T("cancel"), a.Dialog.Picking).Clicked() {
			a.ExportOpen = false
		}
		if dialogPrimaryButton(c, a.T("export"), a.Busy || a.Dialog.Picking || a.Selected.Len() == 0).Clicked() {
			a.pickExportDestination()
		}
	})
}

func (a *App) pickExportDestination() {
	items := a.SelectedAssets()
	if len(items) == 0 {
		return
	}
	a.Dialog.Picking = true
	title, format, quality, motion, generation := a.T("export"), a.Dialog.Format, a.Dialog.Quality, a.Dialog.ExportMotion, a.generation
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
				// 用打开对话框时的那批素材：选位置期间自动导入可能改了选择。
				a.ExportTo(path, format, quality, motion, items)
			}
		})
	}()
}

func exportDialogFileName(asset model.Asset, format model.ExportFormat) string {
	name := strings.TrimSuffix(libraryPackageName(asset.Name), ".momento")
	extension := asset.Extension
	// 视频总是原样导出，保存框里的默认文件名也用视频自己的扩展名。
	if isVideoAsset(asset) {
		format = model.ExportOriginal
	}
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
	var input ui.Element
	dialogPanel(c.Key("text-prompt"), &a.PromptOpen, panelDialog, a.PromptIcon, a.PromptTitle, func() {
		input = fieldStyle(c, ui.TextInput(c.Key("prompt-value"), &a.PromptValue).Label(a.PromptTitle).AutoFocus().FillWidth())
	}, func() {
		if dialogButton(c, a.T("cancel"), false).Clicked() {
			a.PromptOpen = false
			a.PromptSubmit = nil
		}
		ready := strings.TrimSpace(a.PromptValue) != ""
		submit := dialogPrimaryButton(c, a.T("save"), !ready)
		if ready && (submit.Clicked() || input.Submitted()) {
			a.PromptOpen = false
			callback := a.PromptSubmit
			a.PromptSubmit = nil
			if callback != nil {
				callback(strings.TrimSpace(a.PromptValue))
			}
		}
	})
}

func (a *App) errorDialog(c *ui.Context) {
	open := a.Error != ""
	p := paletteFor(c.Theme())
	dialogPanel(c.Key("error"), &open, 500, "warning", a.T("operation_failed"), func() {
		ui.Scroll(c).Scrollbars(false).MaxHeight(180).FillWidth().Children(func() {
			ui.Text(c, a.Error).Selectable().TextColor(p.Danger)
		})
		dialogSection(c, a.T("log_location"), func() {
			ui.Text(c, logging.Directory()).FontSize(fontSecondary).TextColor(p.TextSecondary).Selectable()
		})
	}, func() {
		if dialogButton(c, a.T("copy_error"), false).Clicked() {
			details := a.Error + "\n" + a.T("log_location") + ": " + logging.Directory()
			c.WriteClipboardThen(details, func(err error) {
				if err != nil {
					a.ShowError(err)
				} else {
					a.Notify(a.T("copied"))
				}
			})
		}
		if dialogPrimaryButton(c, a.T("close"), false).Clicked() {
			open = false
		}
	})
	if !open {
		a.Error = ""
	}
}

// 新建或编辑文件夹的对话框：名称框左边是图标按钮，点开选图标。
type folderDialogState struct {
	Open, PickerOpen bool
	// ID 为空是新建；Parent 是新建时的上级；Assets 是建好后要放进去的素材（检查器里「创建」时带上）。
	ID, Parent, Name, Icon string
	Assets                 []string
	// 打开时的图库代数；切换、删除或丢失图库后代数会变，草稿不能写进别的图库。
	Generation uint64
}

func (a *App) ShowNewFolder(parentID, name string, assetIDs []string) {
	if a.Library == nil {
		return
	}
	a.Dialog.Folder = folderDialogState{Open: true, Parent: parentID, Name: name, Assets: assetIDs, Generation: a.generation}
}

func (a *App) ShowEditFolder(folder model.Folder) {
	if a.Library == nil {
		return
	}
	a.Dialog.Folder = folderDialogState{Open: true, ID: folder.ID, Name: folder.Name, Icon: folder.Icon, Generation: a.generation}
}

func (a *App) folderDialog(c *ui.Context) {
	state := &a.Dialog.Folder
	if state.Open && (state.Generation != a.generation || a.Library == nil) {
		*state = folderDialogState{}
	}
	name := strings.TrimSpace(state.Name)
	ready := !a.Busy && name != ""
	icon, title, action := "create-library", a.T("new_folder"), a.T("create")
	if state.ID != "" {
		icon, title, action = "pencil-simple", a.T("edit_folder"), a.T("save")
	}
	var input ui.Element
	dialogPanel(c.Key("folder-dialog"), &state.Open, panelDialog, icon, title, func() {
		dialogSection(c, a.T("folder_name"), func() {
			input = a.iconNameField(c.Key("folder-name-field"), &state.Name, &state.PickerOpen, a.Busy, a.T("folder_icon"), a.T("folder_name"),
				func() { ui.Icon(c, folderIconsRegular[folderIconID(state.Icon)]).Size(iconMedium, iconMedium) },
				func() { a.folderIconGrid(c, &state.Icon, &state.PickerOpen) }).AutoFocus()
		})
	}, func() {
		if dialogButton(c, a.T("cancel"), false).Clicked() {
			state.Open = false
		}
		submit := dialogPrimaryButton(c, action, !ready)
		if ready && (submit.Clicked() || input.Submitted()) {
			a.saveFolderDialog(name)
		}
	})
}

func (a *App) saveFolderDialog(name string) {
	state := a.Dialog.Folder
	a.Dialog.Folder.Open = false
	lib := a.Library
	if lib == nil || state.Generation != a.generation {
		return
	}
	if state.ID != "" {
		a.Run("edit_folder", func(ctx context.Context) error { return lib.EditFolder(ctx, state.ID, name, state.Icon) })
		return
	}
	a.Run("new_folder", func(ctx context.Context) error {
		folder, err := lib.CreateFolder(ctx, name, state.Parent, state.Icon)
		if err != nil || len(state.Assets) == 0 {
			return err
		}
		return lib.SetAssetFolders(ctx, state.Assets, []string{folder.ID}, true)
	})
}
