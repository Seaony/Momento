package ui

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/Seaony/Momento/app/internal/watch"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

type importState struct {
	WatchCancel  context.CancelFunc
	WatchLibrary string
	WatchRules   []AutoImportRule
	PickerOpen   bool
	PickerFolder string
	PickerTags   map[string]bool
	PickerReply  chan importChoice
}
type importChoice struct {
	FolderID string
	TagIDs   []string
	Canceled bool
}

func (a *App) PasteImport() {
	if a.IsLocked() {
		a.ShowMainWindow()
		return
	}
	if a.Library == nil {
		a.ShowError(i18n.NewError("library_required"))
		return
	}
	if a.Busy {
		a.Notify(a.T("busy"))
		return
	}
	files, err := mygo.Clipboard.ReadFiles()
	if err != nil && !errors.Is(err, transfer.ErrFormat) {
		a.ShowError(err)
		return
	}
	paths := imageClipboardPaths(files)
	if len(paths) > 0 {
		a.ImportPaths(paths)
		return
	}
	data := mygo.Clipboard.ReadImage()
	if len(data) == 0 {
		a.Notify(a.T("clipboard_no_image"))
		return
	}
	folder := a.Query.FolderID
	a.Busy = true
	go func() {
		dir, err := os.MkdirTemp("", "momento-paste-")
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, "Clipboard.png"), data, 0600)
		}
		a.Window.Update(func() {
			a.Busy = false
			if err != nil {
				if dir != "" {
					if cleanupErr := os.RemoveAll(dir); cleanupErr != nil {
						slog.Error("清理剪贴板临时图片失败", "error", cleanupErr)
					}
				}
				a.ShowError(err)
				return
			}
			a.startPathImport([]string{filepath.Join(dir, "Clipboard.png")}, folder, "", func() {
				if err := os.RemoveAll(dir); err != nil {
					slog.Error("清理剪贴板临时图片失败", "error", err)
				}
			})
		})
	}()
}
func imageClipboardPaths(paths []string) []string {
	var out []string
	for _, path := range paths {
		if slices.Contains([]string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".tif", ".tiff", ".bmp", ".heic", ".heif", ".avif"}, strings.ToLower(filepath.Ext(path))) {
			out = append(out, path)
		}
	}
	return out
}

func (a *App) startPathImport(paths []string, folderID, event string, cleanup func()) {
	if a.Library == nil || a.Busy {
		if cleanup != nil {
			cleanup()
		}
		a.Notify(a.T("busy"))
		return
	}
	service, lib, gen := a.Assets, a.Library, a.generation
	request := model.ImportRequest{Paths: slices.Clone(paths), FolderID: folderID, ImportKeywords: a.Preferences.ImportKeywords}
	a.Busy = true
	a.Progress = &model.ImportProgress{Stage: "scanning"}
	go func() {
		if cleanup != nil {
			defer cleanup()
		}
		_, _ = a.importFiles(a.ctx, service, lib, gen, request, event)
	}()
}
func (a *App) importFiles(ctx context.Context, service *assets.Service, lib *library.Library, gen uint64, request model.ImportRequest, event string) (model.ImportResult, error) {
	a.workMu.Lock()
	defer a.workMu.Unlock()
	result, err := service.Import(ctx, request, func(p model.ImportProgress) {
		a.Window.Update(func() {
			if a.generation == gen {
				a.Progress = &p
			}
		})
	})
	var snapshot model.Snapshot
	if err == nil {
		snapshot, err = lib.Snapshot(ctx)
	}
	a.Window.Update(func() {
		if a.generation != gen {
			return
		}
		a.Busy = false
		a.Progress = nil
		if err != nil {
			a.ShowError(err)
		} else {
			a.applySnapshot(snapshot)
			a.importFeedback(event, result)
		}
		a.drainPending()
	})
	return result, err
}

func (a *App) ConfigureAutoImport() {
	libraryID := a.libraryID()
	rules := a.Preferences.AutoImport[libraryID]
	if a.Import.WatchCancel != nil && a.Import.WatchLibrary == libraryID && reflect.DeepEqual(a.Import.WatchRules, rules) {
		return
	}
	if a.Import.WatchCancel != nil {
		a.Import.WatchCancel()
		a.Import.WatchCancel = nil
	}
	if a.Library == nil || a.Window == nil {
		return
	}
	a.Import.WatchLibrary = libraryID
	a.Import.WatchRules = slices.Clone(rules)
	if len(rules) == 0 {
		return
	}
	lib, service, gen := a.Library, a.Assets, a.generation
	ctx, cancel := context.WithCancel(a.ctx)
	a.Import.WatchCancel = cancel
	w := watch.New(lib.Root, func(ctx context.Context, path, folderID string) (model.ImportResult, error) {
		ready := false
		keywords := false
		mygo.RunOnMain(func() {
			if gen == a.generation && !a.Busy && ctx.Err() == nil {
				a.Busy = true
				ready = true
				keywords = a.Preferences.ImportKeywords
			}
		})
		if !ready {
			return model.ImportResult{}, assets.ErrImportBusy
		}
		return a.importFiles(ctx, service, lib, gen, model.ImportRequest{Paths: []string{path}, FolderID: folderID, ImportKeywords: keywords}, "auto-import")
	}, mygo.Shell.TrashItem, func(event watch.Event) {
		a.Window.Update(func() {
			if gen != a.generation {
				return
			}
			if a.Settings.AutoImportErrors == nil {
				a.Settings.AutoImportErrors = map[string]string{}
			}
			if event.Err != nil && !errors.Is(event.Err, context.Canceled) {
				a.Settings.AutoImportErrors[a.watchRulePath(event.Path)] = a.Texts.Error(event.Err)
				slog.Error("自动导入失败", "path", event.Path, "error", event.Err)
			} else {
				delete(a.Settings.AutoImportErrors, a.watchRulePath(event.Path))
			}
			a.InvalidateWindows()
		})
	})
	var configs []watch.Rule
	for _, rule := range rules {
		configs = append(configs, watch.Rule{Path: rule.Path, FolderID: rule.FolderID, Enabled: rule.Enabled})
	}
	w.Update(configs)
	go w.Run(ctx)
}

func (a *App) remoteImportChoice(ctx context.Context) (importChoice, error) {
	reply := make(chan importChoice, 1)
	var readyErr error
	mygo.RunOnMain(func() {
		if a.Import.PickerOpen {
			readyErr = i18n.NewError("busy")
			return
		}
		a.Import.PickerFolder = ""
		a.Import.PickerTags = map[string]bool{}
		a.Import.PickerReply = reply
		a.Import.PickerOpen = true
		a.ShowMainWindow()
	})
	if readyErr != nil {
		return importChoice{}, readyErr
	}
	select {
	case choice := <-reply:
		if choice.Canceled {
			slog.Info("浏览器扩展导入已取消")
			return choice, i18n.NewError("import_canceled")
		}
		return choice, nil
	case <-ctx.Done():
		a.Window.Update(func() {
			if a.Import.PickerReply == reply {
				a.Import.PickerOpen = false
				a.Import.PickerReply = nil
			}
		})
		return importChoice{}, ctx.Err()
	}
}
func (a *App) extensionImportPicker(c *ui.Context) {
	if !a.Import.PickerOpen {
		return
	}
	open := true
	ui.DialogBase(c.Key("extension-import-picker"), &open, func(backdrop, panel ui.Element) {
		panel.Width(420).Padding(24).Gap(16).Background(c.Theme().Surface).Radius(12).Label(a.T("extension_picker_title"))
		ui.Text(c, a.T("extension_picker_title")).Bold()
		names := []string{a.T("all_assets")}
		if a.Import.PickerFolder == "" {
			a.Import.PickerFolder = names[0]
		}
		for _, folder := range a.Snapshot.Folders {
			names = append(names, folder.Name)
		}
		ui.Select(c.Key("picker-folder"), &a.Import.PickerFolder, names).FillWidth().Label(a.T("folders"))
		ui.Scroll(c).MaxHeight(240).FillWidth().Children(func() {
			for _, tag := range a.Snapshot.Tags {
				selected := a.Import.PickerTags[tag.ID]
				if ui.Checkbox(c.Key(tag.ID), &selected, tag.Name).Changed() {
					a.Import.PickerTags[tag.ID] = selected
				}
			}
		})
		ui.Row(c).Justify(ui.End).Gap(8).Children(func() {
			if ui.Button(c, a.T("cancel")).Clicked() {
				open = false
			}
			if ui.PrimaryButton(c, a.T("import")).Clicked() {
				choice := importChoice{}
				for index, name := range names {
					if index > 0 && name == a.Import.PickerFolder {
						choice.FolderID = a.Snapshot.Folders[index-1].ID
						break
					}
				}
				for id, selected := range a.Import.PickerTags {
					if selected {
						choice.TagIDs = append(choice.TagIDs, id)
					}
				}
				a.Import.PickerReply <- choice
				a.Import.PickerReply = nil
				open = false
			}
		})
	})
	if !open {
		a.Import.PickerOpen = false
		if a.Import.PickerReply != nil {
			a.Import.PickerReply <- importChoice{Canceled: true}
			a.Import.PickerReply = nil
		}
	}
}

func (a *App) watchRulePath(path string) string {
	for _, rule := range a.Preferences.AutoImport[a.libraryID()] {
		root := rule.Path
		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}
		rel, err := filepath.Rel(root, path)
		if rule.Path == path || err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return rule.Path
		}
	}
	return path
}
