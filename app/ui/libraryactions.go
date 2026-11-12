package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
)

func libraryOpenOptions(parent *mygo.Window, title string) mygo.OpenDialogOptions {
	// 图库实际是目录，macOS需要允许把目录包作为目录选择。
	return mygo.OpenDialogOptions{Parent: parent, Title: title, Directory: true, TreatPackagesAsDirectories: runtime.GOOS == "darwin"}
}

func (a *App) ChooseImportLibrary() {
	if a.Busy {
		return
	}
	a.Busy = true
	title := a.T("import_library")
	go func() {
		paths, err := mygo.Dialog.Open(libraryOpenOptions(a.Window, title))
		a.Window.Update(func() {
			a.Busy = false
			if err != nil {
				a.ShowError(err)
				a.drainPending()
				return
			}
			if len(paths) == 0 {
				a.drainPending()
				return
			}
			a.importLibraryFrom(paths[0])
		})
	}()
}

func (a *App) importLibraryFrom(source string) {
	if a.Busy {
		return
	}
	a.Busy = true
	go func() {
		a.workMu.Lock()
		defer a.workMu.Unlock()
		userData, err := mygo.App.Path(mygo.PathUserData)
		var destination string
		if err == nil {
			var imported *library.Library
			imported, err = library.ImportLibrary(a.ctx, source, filepath.Join(userData, "Libraries"))
			if imported != nil {
				destination = imported.Root
				if err == nil {
					var snapshot model.Snapshot
					snapshot, err = imported.Snapshot(a.ctx)
					var missing []model.Asset
					for _, asset := range snapshot.Assets {
						if err != nil {
							break
						}
						_, statErr := os.Stat(filepath.Join(imported.Root, "thumbnails", asset.ID+".png"))
						if os.IsNotExist(statErr) {
							missing = append(missing, asset)
						} else if statErr != nil {
							err = statErr
						}
					}
					if err == nil && len(missing) > 0 {
						err = assets.New(imported.Root, imported.DB).RefreshThumbnails(a.ctx, missing)
					}
				}
				err = errors.Join(err, imported.Close())
			}
		}
		a.Window.Update(func() {
			a.Busy = false
			if err != nil {
				a.ShowError(err)
				a.drainPending()
				return
			}
			a.Notify(a.T("library_imported"))
			a.OpenLibrary(destination)
		})
	}()
}

func (a *App) ChooseExportLibrary() {
	if a.Library == nil || a.Busy {
		return
	}
	lib := a.Library
	a.Busy = true
	title := a.T("export_library")
	defaultPath := filepath.Base(lib.Root)
	go func() {
		destination, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.Window, Title: title, DefaultPath: defaultPath, CreateDirectories: true, Filters: []mygo.FileFilter{{Name: title, Extensions: []string{"momento"}}}})
		a.Window.Update(func() {
			a.Busy = false
			if err != nil {
				a.ShowError(err)
				a.drainPending()
				return
			}
			if destination == "" {
				a.drainPending()
				return
			}
			destination = libraryExportPath(destination)
			a.Run("export_library", func(ctx context.Context) error {
				if err := lib.ExportLibrary(ctx, destination); err != nil {
					return err
				}
				a.Window.Update(func() { a.Notify(a.T("library_exported")) })
				return nil
			})
		})
	}()
}

func libraryExportPath(path string) string {
	if !strings.EqualFold(filepath.Ext(path), ".momento") {
		return path + ".momento"
	}
	return path
}

func (a *App) ClearLibraryCache() {
	if a.Library == nil || a.Busy {
		return
	}
	lib, service := a.Library, a.Assets
	items := append([]model.Asset(nil), a.Snapshot.Assets...)
	gen := a.generation
	a.Run("clear_library_cache", func(ctx context.Context) error {
		err := lib.ClearCache(ctx)
		if err == nil {
			err = service.RefreshThumbnails(ctx, items)
		}
		a.Window.Update(func() {
			if a.generation != gen {
				return
			}
			if a.Thumbnails != nil {
				a.Thumbnails.Clear()
			}
			if err == nil {
				a.Notify(a.T("library_cache_rebuilt"))
			}
		})
		return err
	})
}
