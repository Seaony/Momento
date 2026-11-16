package library

import (
	"encoding/json"
	"github.com/Seaony/Momento/app/internal/i18n"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Recent struct {
	Path        string `json:"path"`
	LibraryID   string `json:"libraryID"`
	DisplayName string `json:"displayName"`
	// 打开时从 manifest 抄一份图标编号，图库菜单不用逐个打开图库就能显示。
	Icon       string    `json:"icon,omitempty"`
	LastOpened time.Time `json:"lastOpened"`
}

type RecentStore struct {
	Path string
	mu   sync.Mutex
}

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Momento"), nil
}

func NewRecentStore(configDir string) *RecentStore {
	return &RecentStore{Path: filepath.Join(configDir, "recent-libraries.json")}
}

func (r *RecentStore) read() ([]Recent, error) {
	data, err := os.ReadFile(r.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Recent
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, i18n.WrapError(err, "error_recent_invalid")
	}
	return entries, nil
}

func (r *RecentStore) write(entries []Recent) error {
	if err := os.MkdirAll(filepath.Dir(r.Path), 0700); err != nil {
		return err
	}
	return writeJSON(r.Path, entries)
}

func (r *RecentStore) List() ([]Recent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.read()
}

// Add 记下打开过的图库。已在列表里的只更新名称、图标和打开时间，位置不动（保留用户拖好的顺序）；
// 新图库加到末尾。列表不设上限，条目只在用户移除时才会消失。
func (r *RecentStore) Add(entry Recent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path, err := filepath.Abs(entry.Path)
	if err != nil {
		return err
	}
	entry.Path = path
	if entry.LastOpened.IsZero() {
		entry.LastOpened = time.Now().UTC()
	}
	entries, err := r.read()
	if err != nil {
		return err
	}
	for i, old := range entries {
		if old.Path == path {
			entries[i] = entry
			return r.write(entries)
		}
	}
	return r.write(append(entries, entry))
}

// MostRecent 返回打开时间最近的那条；列表为空时 ok 为 false。启动时用它选要打开的图库，不看列表顺序。
func MostRecent(entries []Recent) (latest Recent, ok bool) {
	for _, entry := range entries {
		if !ok || entry.LastOpened.After(latest.LastOpened) {
			latest, ok = entry, true
		}
	}
	return latest, ok
}

// CheckLibraryPath 检查图库现在能不能打开（外接盘没插、被移走、数据库丢了时会报错）。会读磁盘，
// 网络盘断开时可能卡很久，不要在主线程调用。
func CheckLibraryPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return i18n.WrapError(err, "error_library_unavailable", path)
	}
	if !info.IsDir() {
		return i18n.NewError("error_library_directory")
	}
	manifest, err := os.Stat(filepath.Join(path, "manifest.json"))
	if err != nil {
		return i18n.WrapError(err, "error_library_unavailable", path)
	}
	if !manifest.Mode().IsRegular() {
		return i18n.NewError("error_manifest_invalid")
	}
	// 和 Open 一样要求数据库在：只看目录和清单的话，数据库丢了也会被当成能打开。
	if _, err := os.Stat(filepath.Join(path, "database", "library.sqlite")); err != nil {
		return i18n.WrapError(err, "error_library_unavailable", path)
	}
	return nil
}

// Relocate 把列表里 oldPath 那条换到用户重新选的 newPath，位置和打开时间不变，名称和图标按新位置的清单更新。
// 新位置必须是同一个图库（LibraryID 相同），否则报错、列表不变。新位置原本也在列表里时合并成一条。
func (r *RecentStore) Relocate(oldPath, newPath string) (Recent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	oldPath, err := filepath.Abs(oldPath)
	if err != nil {
		return Recent{}, err
	}
	if err := ValidateWorkingPath(newPath); err != nil {
		return Recent{}, err
	}
	root, err := filepath.Abs(newPath)
	if err != nil {
		return Recent{}, err
	}
	// 和 Open 一样解开链接，之后打开时记下的路径才对得上。
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return Recent{}, err
	}
	if inTrash(root) {
		return Recent{}, i18n.NewError("error_library_in_trash")
	}
	if err := CheckLibraryPath(root); err != nil {
		return Recent{}, err
	}
	manifest, err := readManifest(root)
	if err != nil {
		return Recent{}, err
	}
	entries, err := r.read()
	if err != nil {
		return Recent{}, err
	}
	index := -1
	for i, entry := range entries {
		if entry.Path == oldPath {
			index = i
			break
		}
	}
	if index < 0 {
		return Recent{}, i18n.NewError("error_recent_entry_missing", oldPath)
	}
	if entries[index].LibraryID != manifest.LibraryID {
		return Recent{}, i18n.NewError("error_relocate_other_library", manifest.DisplayName)
	}
	entry := entries[index]
	entry.Path, entry.DisplayName, entry.Icon = root, manifest.DisplayName, manifest.Icon
	next := make([]Recent, 0, len(entries))
	for i, old := range entries {
		switch {
		case i == index:
			next = append(next, entry)
		case old.Path != root:
			next = append(next, old)
		}
	}
	if err := r.write(next); err != nil {
		return Recent{}, err
	}
	slog.Info("重新定位图库", "from", oldPath, "to", root, "libraryID", manifest.LibraryID)
	return entry, nil
}

func (r *RecentStore) Remove(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	entries, err := r.read()
	if err != nil {
		return err
	}
	next := make([]Recent, 0, len(entries))
	for _, entry := range entries {
		if entry.Path != path {
			next = append(next, entry)
		}
	}
	return r.write(next)
}

func (r *RecentStore) Reorder(paths []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entries, err := r.read()
	if err != nil {
		return err
	}
	if len(paths) != len(entries) {
		return i18n.NewError("error_recent_order_all")
	}
	byPath := make(map[string]Recent, len(entries))
	for _, entry := range entries {
		byPath[entry.Path] = entry
	}
	next := make([]Recent, 0, len(entries))
	for _, path := range paths {
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		entry, ok := byPath[path]
		if !ok {
			return i18n.NewError("error_recent_order_invalid")
		}
		next = append(next, entry)
		delete(byPath, path)
	}
	return r.write(next)
}
