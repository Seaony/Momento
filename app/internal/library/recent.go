package library

import (
	"encoding/json"
	"github.com/Seaony/Momento/app/internal/i18n"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Recent struct {
	Path        string    `json:"path"`
	LibraryID   string    `json:"libraryID"`
	DisplayName string    `json:"displayName"`
	LastOpened  time.Time `json:"lastOpened"`
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
	next := []Recent{entry}
	for _, old := range entries {
		if old.Path != path && len(next) < 10 {
			next = append(next, old)
		}
	}
	return r.write(next)
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
