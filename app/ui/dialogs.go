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
	Folder         folderDialogState
	Format         model.ExportFormat
	Quality        float64
	ExportMotion   bool // 实况照片连视频一起导出
	// 转成 JPEG/PNG 时限制最长边：勾上才生效，像素数是输入框里的文字，导出前再校验。
	MaxEdgeOn bool
	MaxEdge   string
	// 导出设置本次运行里一直记着（第一次打开时才填默认值），ExportDir 是上次导出的文件夹。
	ExportReady bool
	ExportDir   string
	// 文本输入框确认按钮的文字，由 AskWith 设置；为空用「保存」。
	PromptConfirm string
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
	if !a.Dialog.ExportReady {
		a.Dialog.ExportReady = true
		a.Dialog.Format = model.ExportOriginal
		a.Dialog.Quality = 0.9
		a.Dialog.MaxEdge = "2048"
	}
	if !a.PromptOpen {
		a.Dialog.PromptConfirm = ""
	}
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
					// 默认焦点放在取消上：回车或空格不会直接删东西。
					if dialogButton(c, a.T("cancel"), false).AutoFocus().Clicked() {
						a.ConfirmOpen = false
					}
					if dialogDangerButton(c, a.ConfirmLabel).Clicked() {
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

// exportMix 数所选素材里各种东西各有几个，导出对话框按它决定显示哪些选项。
type exportMix struct {
	images, videos, rotated int
	live                    bool
}

// 对话框每帧都要用，按选区缓存，全选十万张时不每帧重数。
func (a *App) exportMix() exportMix {
	cache := a.currentSelection()
	if cache.mix != nil {
		return *cache.mix
	}
	var mix exportMix
	for _, asset := range cache.assets {
		if isVideoAsset(asset) {
			mix.videos++
			continue
		}
		mix.images++
		mix.rotated += btoi(asset.Rotation != 0)
		mix.live = mix.live || isLiveAsset(asset)
	}
	cache.mix = &mix
	return mix
}

func btoi(on bool) int {
	if on {
		return 1
	}
	return 0
}

// 全是视频时导出总是原样，格式、质量、尺寸都用不上。
func (m exportMix) onlyVideos() bool { return m.videos > 0 && m.images == 0 }

func (a *App) exportDialog(c *ui.Context) {
	p := paletteFor(c.Theme())
	mix := a.exportMix()
	dialogPanel(c.Key("export"), &a.ExportOpen, panelDialog, "export", a.T("export"), func() {
		ui.Column(c).FillWidth().Gap(space4).Children(func() {
			ui.Text(c, a.T("export_selected_count", a.Selected.Len())).TextColor(p.TextSecondary)
			if a.Dialog.Format == model.ExportOriginal && mix.rotated > 0 && !mix.onlyVideos() {
				ui.Text(c, a.T("export_rotation_metadata_hint")).FontSize(fontSecondary).TextColor(p.TextSecondary)
			}
		})
		if mix.onlyVideos() {
			return
		}
		a.exportPresetChips(c)
		dialogSection(c, a.T("export_format"), func() {
			formats := []model.ExportFormat{model.ExportOriginal, model.ExportJPEG, model.ExportPNG}
			labels := []string{a.T("export_original"), a.T("export_jpeg"), a.T("export_png")}
			choiceMenu(c.Key("export-format"), a.T("export_format"), max(0, slices.Index(formats, a.Dialog.Format)), labels, func(i int) {
				a.Dialog.Format = formats[i]
			}).AlignSelf(ui.Start).Disabled(a.Dialog.Picking)
			// 混选了视频、又选了转格式：说一句视频不转，免得以为视频也被转了。
			if mix.videos > 0 && a.Dialog.Format != model.ExportOriginal {
				ui.Text(c, a.T("export_video_original")).FontSize(fontSecondary).TextColor(p.TextSecondary)
			}
		})
		if a.Dialog.Format == model.ExportJPEG {
			// 选了 JPEG 才展开质量，对话框跟着长高，不一下跳高。
			revealMotion(ui.Column(c.Key("jpeg-quality")).FillWidth().Gap(space8)).Children(func() {
				ui.Row(c).FillWidth().Children(func() {
					ui.Text(c, a.T("jpeg_quality")).FontSize(fontSecondary).FontWeight(600).TextColor(p.TextSecondary).Grow(1)
					ui.Text(c, fmt.Sprintf("%.0f%%", a.Dialog.Quality*100)).FontFeatures("tnum").FontSize(fontNumber).TextColor(p.TextSecondary)
				})
				ui.Slider(c.Key("jpeg-quality-slider"), &a.Dialog.Quality, 0.01, 1).Label(a.T("jpeg_quality")).FillWidth().Disabled(a.Dialog.Picking)
			})
		}
		if a.Dialog.Format != model.ExportOriginal {
			a.exportSizeSection(c)
		}
		// 选中的有实况照片才出现：默认只导出图片，勾上后视频和图片同名放在一起。
		if mix.live {
			settingsCheckbox(c.Key("export-live-motion"), &a.Dialog.ExportMotion, a.T("export_live_motion")).Disabled(a.Dialog.Picking)
		}
	}, func() {
		_, problem := a.exportOptions()
		if !mix.onlyVideos() {
			if dialogButton(c, a.T("export_save_preset"), a.Dialog.Picking || problem != "").Clicked() {
				a.Ask("export", a.T("export_preset_name"), "", a.saveExportPreset)
			}
		}
		ui.Spacer(c)
		if dialogButton(c, a.T("cancel"), a.Dialog.Picking).Clicked() {
			a.ExportOpen = false
		}
		if dialogPrimaryButton(c, a.T("export"), a.Dialog.Picking || a.Selected.Len() == 0 || problem != "").Clicked() {
			a.pickExportDestination()
		}
	})
}

func (a *App) pickExportDestination() {
	items := a.SelectedAssets()
	if len(items) == 0 {
		return
	}
	options, problem := a.exportOptions()
	if problem != "" {
		return
	}
	a.Dialog.Picking = true
	title, generation, dir := a.T("export"), a.generation, a.Dialog.ExportDir
	name := exportDialogFileName(items[0], options.Format)
	go func() {
		var path string
		var err error
		if len(items) == 1 {
			// 保存框从上次导出的文件夹打开；没导出过就用系统默认位置。
			defaultPath := name
			if dir != "" {
				defaultPath = filepath.Join(dir, name)
			}
			path, err = mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.Window, Title: title, DefaultPath: defaultPath, CreateDirectories: true})
		} else {
			var paths []string
			paths, err = mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.Window, Title: title, DefaultPath: dir, Directory: true, CreateDirectories: true})
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
				a.Dialog.ExportDir = path
				if len(items) == 1 {
					a.Dialog.ExportDir = filepath.Dir(path)
				}
				// 用打开对话框时的那批素材：选位置期间自动导入可能改了选择。
				a.ExportTo(path, options, items)
			}
		})
	}()
}

// 保存框里的默认文件名，和批量导出（assets.exportName）一致：用素材现在的名称，名称为空才用原文件名。
func exportDialogFileName(asset model.Asset, format model.ExportFormat) string {
	name := asset.Name
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(asset.OriginalFileName, filepath.Ext(asset.OriginalFileName))
	}
	name = strings.Trim(strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(name)), ". ")
	if name == "" {
		name = "image"
	}
	extension := asset.Extension
	// 视频总是原样导出，保存框里的默认文件名也用视频自己的扩展名。
	if isVideoAsset(asset) {
		format = model.ExportOriginal
	}
	switch format {
	case model.ExportJPEG:
		extension = "jpg"
	case model.ExportPNG:
		extension = "png"
	}
	return name + "." + strings.ToLower(extension)
}

// AskWith 和 Ask 一样弹出文本输入框，确认按钮的文字由调用方定（比如新建标签用「新建」）。
func (a *App) AskWith(icon, title, initial, confirm string, submit func(string)) {
	a.Ask(icon, title, initial, submit)
	a.Dialog.PromptConfirm = confirm
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
		confirm := a.Dialog.PromptConfirm
		if confirm == "" {
			confirm = a.T("save")
		}
		ready := strings.TrimSpace(a.PromptValue) != ""
		submit := dialogPrimaryButton(c, confirm, !ready)
		if ready && (submit.Clicked() || input.Submitted()) {
			a.PromptOpen = false
			a.Dialog.PromptConfirm = ""
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
		if dialogButton(c, a.T("open_logs"), false).Clicked() {
			a.OpenLogDirectory()
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
	ready := name != ""
	icon, title, action := "create-library", a.T("new_folder"), a.T("create")
	if state.ID != "" {
		icon, title, action = "pencil-simple", a.T("edit_folder"), a.T("save")
	} else if state.Parent != "" {
		title = a.T("new_subfolder")
	}
	var input ui.Element
	dialogPanel(c.Key("folder-dialog"), &state.Open, panelDialog, icon, title, func() {
		dialogSection(c, a.T("folder_name"), func() {
			input = a.iconNameField(c.Key("folder-name-field"), &state.Name, &state.PickerOpen, false, a.T("folder_icon"), a.T("folder_name"),
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
		_, err := lib.CreateFolderWithAssets(ctx, name, state.Parent, state.Icon, state.Assets)
		return err
	})
}
