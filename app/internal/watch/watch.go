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
	RulePath string
	FolderID string
	Result   model.ImportResult
	Err      error
}

// 临时失败（磁盘满、权限、整批出错、移源失败）按次数指数退避重试，间隔封顶后一直按封顶间隔重试：
// 磁盘清出空间、权限改好后文件会自己导进来。格式无法识别这类永久错误 attempted 置真，文件不变就不再尝试。
const (
	retryBackoff = 10 * time.Second
	retryCeiling = 5 * time.Minute
)

type fileState struct {
	info      os.FileInfo
	attempted bool
	failures  int       // 已经失败的次数
	retryAt   time.Time // 这之前不重试
	imported  bool
}

// 导入本身就不可能成功的错误：重试没有意义。
var permanentKeys = map[string]bool{
	"error_import_unsupported_file":   true,
	"error_decode_image":              true,
	"error_image_unreadable":          true,
	"error_video_unreadable":          true,
	"error_system_decode_unavailable": true,
	"error_image_too_large":           true,
	// 库里同一张实况照片配的是另一段视频：等用户处理，重试时视频会因为没有图片可配而被当成单独素材导进去。
	"error_live_motion_conflict": true,
}

func permanent(err error) bool {
	for ; err != nil; err = errors.Unwrap(err) {
		if localized, ok := err.(*i18n.LocalizedError); ok && permanentKeys[localized.Key] {
			return true
		}
	}
	return false
}

func backoff(failures int) time.Duration {
	delay := retryBackoff << (failures - 1)
	if delay <= 0 || delay > retryCeiling {
		return retryCeiling
	}
	return delay
}

// ImportFunc 把一批源文件导入到同一个目标文件夹。单个文件失败记在 result.Failed（和 Errors 一一对应），
// 返回 err 表示整批没有导入。
type ImportFunc func(ctx context.Context, paths []string, folderID string) (model.ImportResult, error)

// 一个已经写完、等着导入的文件。
type readyFile struct {
	path, rulePath string
	info           os.FileInfo
}

type scanEntry struct {
	path string
	dir  bool
}

type scanDirectory struct {
	info    os.FileInfo
	readAt  time.Time
	entries []scanEntry
}

type Watcher struct {
	root       string
	importFile ImportFunc
	trash      func(string) error
	event      func(context.Context, []Event)
	now        func() time.Time
	mu         sync.Mutex
	rules      []Rule
	revision   uint64
	scanMu     sync.Mutex
	scanned    uint64
	files      map[string]fileState
	dirs       map[string]scanDirectory
	ruleErrors map[string]string
}

func New(libraryRoot string, importFile ImportFunc, trash func(string) error, event func(context.Context, []Event)) *Watcher {
	return &Watcher{
		root: libraryRoot, importFile: importFile, trash: trash, event: event, now: time.Now,
		files: map[string]fileState{}, ruleErrors: map[string]string{},
	}
}

func (w *Watcher) Update(rules []Rule) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if slices.Equal(w.rules, rules) {
		return
	}
	w.rules = slices.Clone(rules)
	w.revision++
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
	_, err := resolveRule(libraryRoot, path)
	return err
}

func resolveRule(libraryRoot, path string) (string, error) {
	root, err := filepath.EvalSymlinks(libraryRoot)
	if err != nil {
		return "", err
	}
	folder, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	folder, err = filepath.Abs(folder)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(folder)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s: %w", path, fs.ErrInvalid)
	}
	for current := folder; ; current = filepath.Dir(current) {
		if strings.EqualFold(filepath.Ext(current), ".momento") {
			return "", ErrLibraryPath
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	if within(root, folder) || within(folder, root) {
		return "", ErrLibraryPath
	}
	return folder, nil
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
	revision := w.revision
	w.mu.Unlock()
	if w.scanned != revision {
		clear(w.files)
		w.scanned = revision
	}
	now := w.now()
	seen := make(map[string]bool, len(w.files))
	dirs := make(map[string]scanDirectory, len(w.dirs))
	active := map[string]bool{}
	var roots []string
	var events []Event
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
		root, err := resolveRule(w.root, rule.Path)
		if err == nil {
			for _, other := range roots {
				if within(root, other) || within(other, root) {
					err = ErrOverlappingFolders
					break
				}
			}
		}
		if err != nil {
			events = w.ruleError(events, rule, err)
			continue
		}
		roots = append(roots, root)
		err = w.scanDirectory(ctx, root, now, dirs, func(path string, info os.FileInfo) {
			seen[path] = true
			previous := w.files[path]
			if !sameFile(previous.info, info) {
				w.files[path] = fileState{info: info}
				return
			}
			if previous.attempted || now.Before(previous.retryAt) {
				return
			}
			if _, ok := ready[rule.FolderID]; !ok {
				folders = append(folders, rule.FolderID)
			}
			ready[rule.FolderID] = append(ready[rule.FolderID], readyFile{path: path, rulePath: rule.Path, info: info})
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			events = w.ruleError(events, rule, err)
		} else {
			delete(w.ruleErrors, rule.Path)
		}
	}
	for _, folderID := range folders {
		if ctx.Err() != nil {
			return
		}
		events = w.importBatch(ctx, folderID, ready[folderID], events)
	}
	w.dirs = dirs
	for path := range w.files {
		if !seen[path] {
			delete(w.files, path)
		}
	}
	for path := range w.ruleErrors {
		if !active[path] {
			delete(w.ruleErrors, path)
		}
	}
	if len(events) > 0 && w.event != nil {
		w.event(ctx, events)
	}
}

func (w *Watcher) scanDirectory(ctx context.Context, path string, now time.Time, dirs map[string]scanDirectory, visit func(string, os.FileInfo)) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	cached := w.dirs[path]
	// 定期重读目录，覆盖外接卷时间精度较低或目录修改时间被保留的情况。
	if cached.info == nil || !os.SameFile(info, cached.info) || !info.ModTime().Equal(cached.info.ModTime()) || now.Sub(cached.readAt) >= time.Minute {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		cached = scanDirectory{info: info, readAt: now}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			entryPath := filepath.Join(path, entry.Name())
			if entry.IsDir() && !strings.EqualFold(filepath.Ext(entryPath), ".momento") {
				cached.entries = append(cached.entries, scanEntry{path: entryPath, dir: true})
			} else if entry.Type().IsRegular() && assets.Supported(entryPath) {
				cached.entries = append(cached.entries, scanEntry{path: entryPath})
			}
		}
	}
	for _, entry := range cached.entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.dir {
			if err := w.scanDirectory(ctx, entry.path, now, dirs, visit); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			continue
		}
		info, err := os.Lstat(entry.path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			visit(entry.path, info)
		}
	}
	dirs[path] = cached
	return nil
}

func (w *Watcher) importBatch(ctx context.Context, folderID string, files []readyFile, events []Event) []Event {
	var paths []string
	for _, file := range files {
		if !w.files[file.path].imported {
			paths = append(paths, file.path)
		}
	}
	var result model.ImportResult
	var err error
	failed := map[string]error{}
	if len(paths) > 0 {
		result, err = w.importFile(ctx, paths, folderID)
		if errors.Is(err, assets.ErrImportBusy) {
			return events
		}
		if err != nil && ctx.Err() != nil {
			slog.Warn("自动导入被取消，源文件保留", "folder", folderID, "files", len(paths), "error", err)
			return events
		}
		if err == nil {
			err = checkResult(result, paths, failed)
		}
		if err != nil {
			slog.Error("自动导入整批失败，源文件全部保留", "folder", folderID, "files", len(paths), "error", err)
		} else {
			slog.Info("自动导入一批完成", "folder", folderID, "files", len(paths), "imported", result.Imported, "skipped", result.Skipped, "failed", len(failed))
		}
	}
	var failures []Event
	for _, file := range files {
		state := w.files[file.path]
		state.info = file.info
		var fileErr error
		if !state.imported {
			fileErr = err
			if fileErr == nil {
				fileErr = failed[file.path]
			}
			state.imported = fileErr == nil
		}
		if fileErr == nil {
			after, statErr := os.Lstat(file.path)
			if statErr != nil {
				fileErr = statErr
			} else if !sameFile(file.info, after) {
				fileErr = ErrSourceChanged
			}
		}
		if fileErr == nil && ctx.Err() != nil {
			w.files[file.path] = state
			slog.Warn("自动导入已取消，已入库的源文件保留原地", "path", file.path)
			continue
		}
		if fileErr == nil {
			fileErr = w.trash(file.path)
		}
		switch {
		case fileErr == nil:
			state.attempted = true
		case permanent(fileErr):
			state.attempted = true
		default:
			state.failures++
			state.retryAt = w.now().Add(backoff(state.failures))
			slog.Warn("自动导入失败，稍后重试", "path", file.path, "failures", state.failures, "retryAt", state.retryAt)
		}
		w.files[file.path] = state
		event := Event{Path: file.path, RulePath: file.rulePath, FolderID: folderID, Result: result, Err: fileErr}
		logEvent(event)
		if fileErr != nil {
			failures = append(failures, event)
			continue
		}
		events = append(events, event)
	}
	return append(events, failures...)
}

// checkResult 把 result.Failed 和 Errors 对上号放进 failed；对不上或有文件下落不明时返回 ErrNotImported，整批都不移走。
func checkResult(result model.ImportResult, paths []string, failed map[string]error) error {
	if len(result.Failed) != len(result.Errors) {
		slog.Error("导入结果的失败文件和错误数量对不上", "failed", len(result.Failed), "errors", len(result.Errors))
		return ErrNotImported
	}
	var unmatched map[string]bool
	if len(result.Failed)+len(result.Motion) > 0 {
		unmatched = make(map[string]bool, len(paths))
		for _, path := range paths {
			unmatched[path] = true
		}
	}
	for i, path := range result.Failed {
		if !unmatched[path] || result.Errors[i] == nil {
			slog.Error("导入结果没有唯一有效的失败记录", "path", path)
			return ErrNotImported
		}
		delete(unmatched, path)
		failed[path] = result.Errors[i]
	}
	// 每个没失败的文件都应算进导入或重复跳过；少了说明有文件被悄悄跳过，分不清是哪个，整批保留。
	// 配进实况照片的视频跟着图片走，不单独计数。
	for _, path := range result.Motion {
		if !unmatched[path] {
			slog.Error("导入结果没有唯一有效的实况视频记录", "path", path)
			return ErrNotImported
		}
		delete(unmatched, path)
	}
	if confirmed := result.Imported + result.Skipped + len(failed) + len(result.Motion); confirmed != len(paths) {
		slog.Error("导入结果没有覆盖这批的每个文件", "files", len(paths), "confirmed", confirmed)
		return ErrNotImported
	}
	return nil
}

func sameFile(first, second os.FileInfo) bool {
	return first != nil && second != nil && first.Mode().IsRegular() && second.Mode().IsRegular() && os.SameFile(first, second) && first.Size() == second.Size() && first.ModTime().Equal(second.ModTime())
}

func (w *Watcher) ruleError(events []Event, rule Rule, err error) []Event {
	if previous := w.ruleErrors[rule.Path]; previous == err.Error() {
		return events
	}
	w.ruleErrors[rule.Path] = err.Error()
	event := Event{Path: rule.Path, RulePath: rule.Path, FolderID: rule.FolderID, Err: err}
	logEvent(event)
	return append(events, event)
}

func logEvent(event Event) {
	if event.Err != nil {
		slog.Error("自动导入失败，保留源文件", "path", event.Path, "error", event.Err)
	} else {
		slog.Info("自动导入完成，源文件已移到废纸篓", "path", event.Path)
	}
}
