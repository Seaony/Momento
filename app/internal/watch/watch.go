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

// ImportFunc 把一批源文件导入到同一个目标文件夹。单个文件失败记在 result.Failed（和 Errors 一一对应），
// 返回 err 表示整批没有导入。
type ImportFunc func(ctx context.Context, paths []string, folderID string) (model.ImportResult, error)

// 一个已经写完、等着导入的文件。
type readyFile struct {
	key, path string
	info      os.FileInfo
}

type Watcher struct {
	root       string
	importFile ImportFunc
	trash      func(string) error
	event      func(Event)
	mu         sync.Mutex
	rules      []Rule
	scanMu     sync.Mutex
	files      map[string]fileState
	ruleErrors map[string]string
}

// New 建一个监视器。每轮扫描里写完的文件按目标文件夹分组，每组调用一次 importFile。
// 每个文件处理完都会 emit 一次 Event（Path 是这个文件）：同一批里先报成功、再报失败，按规则记错误的调用方最后留下的是错误。
func New(libraryRoot string, importFile ImportFunc, trash func(string) error, event func(Event)) *Watcher {
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

// 每个文件连续两次保持不变才导入；一轮里就绪的文件按目标文件夹合成一次导入，失败的文件保留。
func (w *Watcher) Scan(ctx context.Context) {
	w.scanMu.Lock()
	defer w.scanMu.Unlock()
	w.mu.Lock()
	rules := slices.Clone(w.rules)
	w.mu.Unlock()
	seen := map[string]bool{}
	active := map[string]bool{}
	var roots []string
	// 目标文件夹 → 就绪文件；folders 记分组出现的先后，导入顺序固定。
	ready := map[string][]readyFile{}
	var folders []string
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
			if _, ok := ready[rule.FolderID]; !ok {
				folders = append(folders, rule.FolderID)
			}
			ready[rule.FolderID] = append(ready[rule.FolderID], readyFile{key: key, path: path, info: info})
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
	for _, folderID := range folders {
		if ctx.Err() != nil {
			return
		}
		w.importBatch(ctx, folderID, ready[folderID])
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

// importBatch 一次导入同一目标文件夹的文件。整批出错时所有源文件都不动；
// 否则只把导入成功（含图库里已有的重复图）且导入期间没被改过的源文件移到废纸篓。
func (w *Watcher) importBatch(ctx context.Context, folderID string, files []readyFile) {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.path
	}
	result, err := w.importFile(ctx, paths, folderID)
	if errors.Is(err, assets.ErrImportBusy) {
		// 别的导入在进行，这批下一轮再试。
		return
	}
	failed := map[string]error{}
	if err == nil {
		err = checkResult(result, paths, failed)
	}
	if err != nil {
		slog.Error("自动导入整批失败，源文件全部保留", "folder", folderID, "files", len(files), "error", err)
	} else {
		slog.Info("自动导入一批完成", "folder", folderID, "files", len(files), "imported", result.Imported, "skipped", result.Skipped, "failed", len(failed))
	}
	var failures []Event
	for _, file := range files {
		w.files[file.key] = fileState{info: file.info, attempted: true}
		fileErr := err
		if fileErr == nil {
			fileErr = failed[file.path]
		}
		if fileErr == nil {
			after, statErr := os.Lstat(file.path)
			if statErr != nil {
				fileErr = statErr
			} else if !sameFile(file.info, after) {
				fileErr = ErrSourceChanged
			}
		}
		if fileErr == nil {
			fileErr = w.trash(file.path)
		}
		event := Event{Path: file.path, FolderID: folderID, Result: result, Err: fileErr}
		if fileErr != nil {
			failures = append(failures, event)
			continue
		}
		w.emit(event)
	}
	for _, event := range failures {
		w.emit(event)
	}
}

// checkResult 把 result.Failed 和 Errors 对上号放进 failed；对不上或有文件下落不明时返回 ErrNotImported，整批都不移走。
func checkResult(result model.ImportResult, paths []string, failed map[string]error) error {
	if len(result.Failed) != len(result.Errors) {
		slog.Error("导入结果的失败文件和错误数量对不上", "failed", len(result.Failed), "errors", len(result.Errors))
		return ErrNotImported
	}
	for i, path := range result.Failed {
		if !slices.Contains(paths, path) {
			slog.Error("导入结果里的失败文件不在这批里", "path", path)
			return ErrNotImported
		}
		failed[path] = result.Errors[i]
	}
	// 每个没失败的文件都应算进导入或重复跳过；少了说明有文件被悄悄跳过，分不清是哪个，整批保留。
	if confirmed := result.Imported + result.Skipped + len(failed); confirmed != len(paths) {
		slog.Error("导入结果没有覆盖这批的每个文件", "files", len(paths), "confirmed", confirmed)
		return ErrNotImported
	}
	return nil
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
		slog.Info("自动导入完成，源文件已移到废纸篓", "path", event.Path)
	}
	if w.event != nil {
		w.event(event)
	}
}
