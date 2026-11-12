package watch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

var (
	ErrLibraryPath        = i18n.NewError("error_watch_library")
	ErrOverlappingFolders = i18n.NewError("error_watch_overlap")
	ErrNotImported        = i18n.NewError("error_watch_not_imported")
	ErrSourceChanged      = i18n.NewError("error_watch_source_changed")
)

type Rule struct {
	Path     string
	FolderID string
	Enabled  bool
}

type Event struct {
	Path     string
	FolderID string
	Result   model.ImportResult
	Err      error
}

type fileState struct {
	info      os.FileInfo
	attempted bool
}

type Watcher struct {
	root       string
	importFile func(context.Context, string, string) (model.ImportResult, error)
	trash      func(string) error
	event      func(Event)
	mu         sync.Mutex
	rules      []Rule
	scanMu     sync.Mutex
	files      map[string]fileState
	ruleErrors map[string]string
}

func New(libraryRoot string, importFile func(context.Context, string, string) (model.ImportResult, error), trash func(string) error, event func(Event)) *Watcher {
	return &Watcher{root: libraryRoot, importFile: importFile, trash: trash, event: event, files: map[string]fileState{}, ruleErrors: map[string]string{}}
}

func (w *Watcher) Update(rules []Rule) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rules = slices.Clone(rules)
}

func (w *Watcher) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	w.Scan(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Scan(ctx)
		}
	}
}

func ValidatePath(libraryRoot, path string) error {
	root, err := filepath.EvalSymlinks(libraryRoot)
	if err != nil {
		return err
	}
	folder, err := filepath.EvalSymlinks(path)
	if err != nil {
		return err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	folder, err = filepath.Abs(folder)
	if err != nil {
		return err
	}
	info, err := os.Stat(folder)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: %w", path, fs.ErrInvalid)
	}
	for current := folder; ; current = filepath.Dir(current) {
		if strings.EqualFold(filepath.Ext(current), ".momento") {
			return ErrLibraryPath
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	if within(root, folder) || within(folder, root) {
		return ErrLibraryPath
	}
	return nil
}

func within(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && (rel == "." || filepath.IsLocal(rel))
}

// 每个文件连续两次保持不变才导入；失败时保留源文件。
func (w *Watcher) Scan(ctx context.Context) {
	w.scanMu.Lock()
	defer w.scanMu.Unlock()
	w.mu.Lock()
	rules := slices.Clone(w.rules)
	w.mu.Unlock()
	seen := map[string]bool{}
	active := map[string]bool{}
	var roots []string
	for _, rule := range rules {
		if ctx.Err() != nil {
			return
		}
		if !rule.Enabled {
			continue
		}
		active[rule.Path] = true
		err := ValidatePath(w.root, rule.Path)
		var root string
		if err == nil {
			root, err = filepath.EvalSymlinks(rule.Path)
			if err == nil {
				root, err = filepath.Abs(root)
			}
		}
		if err == nil {
			for _, other := range roots {
				if within(root, other) || within(other, root) {
					err = ErrOverlappingFolders
					break
				}
			}
		}
		if err != nil {
			w.ruleError(rule, err)
			continue
		}
		roots = append(roots, root)
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if path != root && strings.EqualFold(filepath.Ext(path), ".momento") {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() || !assets.Supported(path) {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			key := rule.Path + "\x00" + rule.FolderID + "\x00" + path
			seen[key] = true
			previous, ok := w.files[key]
			if !ok || !sameFile(previous.info, info) {
				w.files[key] = fileState{info: info}
				return nil
			}
			if previous.attempted {
				return nil
			}
			result, err := w.importFile(ctx, path, rule.FolderID)
			if errors.Is(err, assets.ErrImportBusy) {
				return nil
			}
			if err == nil && len(result.Errors) > 0 {
				err = errors.Join(result.Errors...)
			}
			if err == nil && result.Imported+result.Skipped == 0 {
				err = ErrNotImported
			}
			if err == nil {
				after, statErr := os.Lstat(path)
				if statErr != nil {
					err = statErr
				} else if !sameFile(info, after) {
					err = ErrSourceChanged
				}
			}
			if err == nil {
				err = w.trash(path)
			}
			w.files[key] = fileState{info: info, attempted: true}
			w.emit(Event{Path: path, FolderID: rule.FolderID, Result: result, Err: err})
			return nil
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			w.ruleError(rule, err)
		} else {
			delete(w.ruleErrors, rule.Path)
		}
	}
	for key := range w.files {
		if !seen[key] {
			delete(w.files, key)
		}
	}
	for path := range w.ruleErrors {
		if !active[path] {
			delete(w.ruleErrors, path)
		}
	}
}

func sameFile(first, second os.FileInfo) bool {
	return first != nil && second != nil && first.Mode().IsRegular() && second.Mode().IsRegular() && os.SameFile(first, second) && first.Size() == second.Size() && first.ModTime().Equal(second.ModTime())
}

func (w *Watcher) ruleError(rule Rule, err error) {
	if previous := w.ruleErrors[rule.Path]; previous == err.Error() {
		return
	}
	w.ruleErrors[rule.Path] = err.Error()
	w.emit(Event{Path: rule.Path, FolderID: rule.FolderID, Err: err})
}

func (w *Watcher) emit(event Event) {
	if event.Err != nil {
		slog.Error("自动导入失败，保留源文件", "path", event.Path, "error", event.Err)
	} else {
		slog.Info("自动导入完成，源文件已移到废纸篓", "path", event.Path, "imported", event.Result.Imported, "skipped", event.Result.Skipped)
	}
	if w.event != nil {
		w.event(event)
	}
}
