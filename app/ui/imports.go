package ui

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/url"
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
	WatchCancel     context.CancelFunc
	Watcher         *watch.Watcher
	WatchLibrary    string
	WatchGeneration uint64
	WatchRules      []AutoImportRule
	PickerOpen      bool
	PickerFolder    string
	PickerTags      map[string]bool
	PickerReply     chan importChoice
	// 用户手动发起的那批导入可以中途停止；自动导入由监视器自己管，不给停止按钮。
	manualCancel   context.CancelFunc
	manualCanceled bool
	// 导入进行中又拖进来、粘贴进来的文件：排在后面，这批做完接着导，不能丢掉。
	queued []queuedPathImport
}

type queuedPathImport struct {
	generation      uint64
	paths           []string
	folderID, event string
	cleanup         func()
}
type importChoice struct {
	FolderID string
	TagIDs   []string
	Canceled bool
}

// clipboardImageURLs 从剪贴板文字里取出 http(s) 链接，一行一个，最多 50 个；有一行不是链接就当作普通文字，不导入。
func clipboardImageURLs(text string) []string {
	var urls []string
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parsed, err := url.Parse(line)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(line, " \t") {
			return nil
		}
		urls = append(urls, line)
	}
	if len(urls) > 50 {
		return nil
	}
	return urls
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
	// 忙时文件和截图交给 startPathImport 排队，和拖进来的一样不丢；链接的下载队列不分图库，忙时先不收。
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
		// 复制的是图片链接（一行一个）：按链接下载导入，和浏览器扩展、momento:// 链接走同一个队列。
		if urls := clipboardImageURLs(mygo.Clipboard.ReadText()); len(urls) > 0 {
			if a.Busy {
				a.Notify(a.T("busy"))
				return
			}
			for _, url := range urls {
				a.QueueImport(url, "")
			}
			return
		}
		a.Notify(a.T("clipboard_no_image"))
		return
	}
	folder, generation := a.Query.FolderID, a.generation
	go func() {
		dir, err := os.MkdirTemp("", "momento-paste-")
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, "Clipboard.png"), data, 0600)
		}
		a.Window.Update(func() {
			cleanup := func() {
				if dir == "" {
					return
				}
				if err := os.RemoveAll(dir); err != nil {
					slog.Error("清理剪贴板临时图片失败", "error", err)
				}
			}
			if err != nil {
				cleanup()
				a.ShowError(err)
				return
			}
			// 暂存截图的这一下换了图库：截图是粘贴给原来那个图库的，不能导进新图库。
			if generation != a.generation {
				slog.Warn("粘贴的截图暂存期间换了图库，没有导入")
				cleanup()
				a.Notify(a.T("error_import_library_changed"))
				return
			}
			a.startPathImport([]string{filepath.Join(dir, "Clipboard.png")}, folder, "", cleanup)
		})
	}()
}

// imageClipboardPaths 挑出剪贴板文件里能导入的（图片、视频、RAW、PSD 等），和拖进来、选文件导入认同一批格式。
func imageClipboardPaths(paths []string) []string {
	var out []string
	for _, path := range paths {
		if assets.Supported(path) {
			out = append(out, path)
		}
	}
	return out
}

func (a *App) startPathImport(paths []string, folderID, event string, cleanup func()) {
	if a.Library == nil {
		if cleanup != nil {
			cleanup()
		}
		a.ShowError(i18n.NewError("library_required"))
		return
	}
	if a.Busy {
		a.Import.queued = append(a.Import.queued, queuedPathImport{generation: a.generation, paths: slices.Clone(paths), folderID: folderID, event: event, cleanup: cleanup})
		slog.Info("导入排队", "paths", len(paths), "queued", len(a.Import.queued))
		a.Notify(a.T("import_queued"))
		return
	}
	service, lib, gen := a.Assets, a.Library, a.generation
	request := model.ImportRequest{Paths: slices.Clone(paths), FolderID: folderID, ImportKeywords: a.Preferences.ImportKeywords}
	ctx, cancel := context.WithCancel(a.ctx)
	a.Import.manualCancel, a.Import.manualCanceled = cancel, false
	a.Busy = true
	a.Progress = &model.ImportProgress{Stage: "scanning"}
	go func() {
		defer cancel()
		if cleanup != nil {
			defer cleanup()
		}
		_, _ = a.importFiles(ctx, service, lib, gen, request, event)
	}()
}

// 接着导入排在后面的那批；换了图库的作废（路径还在原处，用户可以重新拖）。
func (a *App) startQueuedPathImport() bool {
	for len(a.Import.queued) > 0 {
		next := a.Import.queued[0]
		a.Import.queued = a.Import.queued[1:]
		if next.generation != a.generation || a.Library == nil {
			slog.Warn("图库已切换，丢弃排队的导入", "paths", len(next.paths))
			if next.cleanup != nil {
				next.cleanup()
			}
			continue
		}
		a.startPathImport(next.paths, next.folderID, next.event, next.cleanup)
		return true
	}
	return false
}

// 停止手动导入：整批一起提交，停下后这批都不会进图库，原文件不动。
func (a *App) CancelImport() {
	if a.Import.manualCancel != nil && !a.Import.manualCanceled {
		slog.Info("用户停止导入")
		a.Import.manualCanceled = true
		a.Import.manualCancel()
	}
}

func (a *App) CanCancelImport() bool {
	return a.Import.manualCancel != nil && !a.Import.manualCanceled
}
func (a *App) importFiles(ctx context.Context, service *assets.Service, lib *library.Library, gen uint64, request model.ImportRequest, event string) (model.ImportResult, error) {
	a.workMu.Lock()
	defer a.workMu.Unlock()
	result, importErr := service.Import(ctx, request, func(p model.ImportProgress) {
		a.Window.Update(func() {
			if a.generation == gen {
				a.Progress = &p
			}
		})
	})
	var snapshot model.Snapshot
	refreshErr := importErr
	if importErr == nil {
		snapshot, refreshErr = lib.Snapshot(a.ctx)
	}
	a.Window.Update(func() {
		if a.generation != gen {
			return
		}
		a.Busy = false
		a.Progress = nil
		canceled := a.Import.manualCanceled && errors.Is(importErr, context.Canceled)
		a.Import.manualCancel, a.Import.manualCanceled = nil, false
		if canceled {
			a.Notify(a.T("import_stopped"))
		} else if refreshErr != nil {
			a.ShowError(refreshErr)
		} else {
			a.applySnapshot(snapshot)
			a.importFeedback(event, result)
		}
		a.drainPending()
	})
	return result, importErr
}

func (a *App) ConfigureAutoImport() {
	libraryID := a.libraryID()
	rules := a.bindAutoImportRules(libraryID)
	if a.Import.watching(libraryID, a.generation, rules) {
		return
	}
	if a.Import.WatchCancel != nil {
		a.Import.WatchCancel()
		a.Import.WatchCancel = nil
	}
	if a.Library == nil || a.Window == nil {
		return
	}
	lib, service, gen := a.Library, a.Assets, a.generation
	if a.Import.Watcher == nil || a.Import.WatchLibrary != libraryID || a.Import.WatchGeneration != gen {
		a.Import.Watcher = watch.New(lib.Root, func(ctx context.Context, paths []string, folderID string) (model.ImportResult, error) {
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
			return a.importFiles(ctx, service, lib, gen, model.ImportRequest{Paths: paths, FolderID: folderID, ImportKeywords: keywords}, "auto-import")
		}, mygo.Shell.TrashItem, func(ctx context.Context, events []watch.Event) {
			a.Window.Update(func() {
				if gen != a.generation || ctx.Err() != nil {
					return
				}
				for _, event := range events {
					if event.Err != nil && !errors.Is(event.Err, context.Canceled) {
						if a.Settings.AutoImportErrors == nil {
							a.Settings.AutoImportErrors = map[string]string{}
						}
						a.Settings.AutoImportErrors[event.RulePath] = a.Texts.Error(event.Err)
						if a.Settings.AutoImportMissing == nil {
							a.Settings.AutoImportMissing = map[string]bool{}
						}
						// 只有规则目录本身不在了才算「文件夹不存在」；目录里某个文件被移走只显示原因。
						a.Settings.AutoImportMissing[event.RulePath] = event.Path == event.RulePath && errors.Is(event.Err, fs.ErrNotExist)
					} else {
						delete(a.Settings.AutoImportErrors, event.RulePath)
						delete(a.Settings.AutoImportMissing, event.RulePath)
					}
				}
				a.InvalidateWindows()
			})
		}, func(ctx context.Context, paths []string, motionOwners map[string]string) (map[string]watch.AssetStatus, error) {
			return autoImportAssetStatus(ctx, lib, paths, motionOwners)
		})
	}
	a.Import.WatchLibrary = libraryID
	a.Import.WatchGeneration = gen
	a.Import.WatchRules = slices.Clone(rules)
	var configs []watch.Rule
	for _, rule := range rules {
		configs = append(configs, watch.Rule{Path: rule.Path, FolderID: rule.FolderID, Enabled: rule.Enabled})
	}
	a.Import.Watcher.Update(configs)
	if !slices.ContainsFunc(configs, func(rule watch.Rule) bool { return rule.Enabled }) {
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.Import.WatchCancel = cancel
	go a.Import.Watcher.Run(ctx)
}

func autoImportAssetStatus(ctx context.Context, lib *library.Library, paths []string, motionOwners map[string]string) (map[string]watch.AssetStatus, error) {
	statuses := make(map[string]watch.AssetStatus, len(paths))
	sourceHashes := make(map[string]string, len(paths))
	rowIDs := make([]string, 0, len(paths))
	for _, path := range paths {
		id := motionOwners[path]
		if id == "" {
			hash, err := assets.HashFile(ctx, path)
			if err != nil {
				return nil, err
			}
			sourceHashes[path] = hash
			id = hash
		}
		rowIDs = append(rowIDs, id)
	}
	rows, err := lib.AssetStatuses(ctx, rowIDs)
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		id := motionOwners[path]
		if id == "" {
			id = sourceHashes[path]
		}
		row := rows[id]
		active := row.OriginalExists
		if motionOwners[path] != "" {
			active = active && row.MotionExists
		}
		statuses[path] = watch.AssetStatus{Active: active, Trashed: row.Trashed}
	}
	return statuses, nil
}

// 只返回属于当前这个图库包的规则。规则记的图库位置还在、但不是当前位置，说明当前打开的是一份副本：副本不监视，在设置里说明原因。
// 原位置已经不存在就当作图库被移动了，改记到当前位置；旧规则没记位置的，记上当前位置。
func (a *App) bindAutoImportRules(libraryID string) []AutoImportRule {
	rules := a.Preferences.AutoImport[libraryID]
	if a.Library == nil || len(rules) == 0 {
		return rules
	}
	root := a.Library.Root
	var active []AutoImportRule
	changed := false
	for i, rule := range rules {
		if rule.Library != "" && !samePath(rule.Library, root) {
			// 只有确认原位置已经没有图库才算移动；读不到（权限、外接盘没响应）时按副本处理，宁可不监视也不能让副本接管。
			if _, err := os.Stat(filepath.Join(rule.Library, "manifest.json")); !os.IsNotExist(err) {
				if a.Settings.AutoImportErrors == nil {
					a.Settings.AutoImportErrors = map[string]string{}
				}
				a.Settings.AutoImportErrors[rule.Path] = a.T("auto_import_other_copy", rule.Library)
				slog.Warn("当前图库是另一份图库的副本，不监视它的自动导入文件夹", "rule", rule.Path, "owner", rule.Library, "current", root, "error", err)
				continue
			}
			slog.Info("图库已移动，自动导入规则改到新位置", "rule", rule.Path, "from", rule.Library, "to", root)
			rule.Library = ""
		}
		// 之前在副本里留下的说明，回到原库后不再显示；真正的导入错误不动。
		if copyNote := a.T("auto_import_other_copy", rule.Library); rule.Library != "" && a.Settings.AutoImportErrors[rule.Path] == copyNote {
			delete(a.Settings.AutoImportErrors, rule.Path)
		}
		if rule.Library == "" {
			if !changed {
				rules = slices.Clone(rules)
				changed = true
			}
			rule.Library = root
			rules[i] = rule
		}
		active = append(active, rule)
	}
	if changed {
		a.Preferences.AutoImport[libraryID] = rules
		a.queuePreferences()
	}
	return active
}

func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	left, err := os.Stat(a)
	if err != nil {
		return false
	}
	right, err := os.Stat(b)
	return err == nil && os.SameFile(left, right)
}

func (a *App) remoteImportChoice(ctx context.Context) (importChoice, error) {
	reply := make(chan importChoice, 1)
	var readyErr error
	mygo.RunOnMain(func() {
		if a.IsLocked() {
			reply <- importChoice{}
			return
		}
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
	dialogPanel(c.Key("extension-import-picker"), &open, panelDialog, "import", a.T("extension_picker_title"), func() {
		dialogSection(c, a.T("folders"), func() {
			choices := a.importFolderChoices()
			label := choices[0].Label
			for _, choice := range choices {
				if choice.ID == a.Import.PickerFolder {
					label = choice.Label
					break
				}
			}
			settingsButton(c.Key("picker-folder"), label).FillWidth().Label(a.T("folders")).Menu(func(m *ui.Menu) {
				for _, choice := range choices {
					if m.Item(choice.Label).Radio(choice.ID == a.Import.PickerFolder).Chosen() {
						a.Import.PickerFolder = choice.ID
					}
				}
			})
		})
		dialogSection(c, a.T("tags"), func() {
			ui.Scroll(c).Scrollbars(false).MaxHeight(240).FillWidth().Gap(space8).Children(func() {
				for _, tag := range a.Snapshot.Tags {
					selected := a.Import.PickerTags[tag.ID]
					if settingsCheckbox(c.Key(tag.ID), &selected, tag.Name).Changed() {
						a.Import.PickerTags[tag.ID] = selected
					}
				}
			})
		})
	}, func() {
		if dialogButton(c, a.T("cancel"), false).Clicked() {
			open = false
		}
		if dialogPrimaryButton(c, a.T("import"), false).Clicked() {
			choice := importChoice{FolderID: a.Import.PickerFolder}
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
	if !open {
		a.Import.PickerOpen = false
		if a.Import.PickerReply != nil {
			a.Import.PickerReply <- importChoice{Canceled: true}
			a.Import.PickerReply = nil
		}
	}
}

func (state *importState) watching(libraryID string, generation uint64, rules []AutoImportRule) bool {
	return state.WatchCancel != nil && state.WatchLibrary == libraryID && state.WatchGeneration == generation && reflect.DeepEqual(state.WatchRules, rules)
}

type importFolderChoice struct{ ID, Label string }

func (a *App) importFolderChoices() []importFolderChoice {
	choices := []importFolderChoice{{Label: a.T("auto_import_library_root")}}
	folders := make(map[string]model.Folder, len(a.Snapshot.Folders))
	for _, folder := range a.Snapshot.Folders {
		folders[folder.ID] = folder
	}
	for _, folder := range a.Snapshot.Folders {
		parts := []string{folder.Name}
		for parent := folder.ParentID; parent != ""; {
			ancestor, ok := folders[parent]
			if !ok {
				break
			}
			parts = append(parts, ancestor.Name)
			parent = ancestor.ParentID
		}
		slices.Reverse(parts)
		choices = append(choices, importFolderChoice{ID: folder.ID, Label: strings.Join(parts, " / ")})
	}
	return choices
}

func (a *App) cancelExtensionPicker() {
	a.Import.PickerOpen = false
	if reply := a.Import.PickerReply; reply != nil {
		reply <- importChoice{Canceled: true}
		a.Import.PickerReply = nil
	}
}
