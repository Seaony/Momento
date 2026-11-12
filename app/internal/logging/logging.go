package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const maxSize int64 = 5 << 20

type Writer struct {
	mu    sync.Mutex
	path  string
	file  *os.File
	size  int64
	limit int64
}

func Directory() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "Momento")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Momento", "Logs")
	default:
		base := os.Getenv("XDG_STATE_HOME")
		if base == "" {
			base = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(base, "momento")
	}
}

func Open() (*Writer, error) { return open(filepath.Join(Directory(), "momento.log"), maxSize) }

func open(path string, limit int64) (*Writer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	w := &Writer{path: path, limit: limit}
	if err := w.reopen(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) reopen() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	w.file, w.size = f, info.Size()
	return nil
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, fmt.Errorf("日志文件已关闭")
	}
	if w.size > 0 && w.size+int64(len(p)) > w.limit {
		if err := w.file.Close(); err != nil {
			return 0, err
		}
		w.file = nil
		for i := 2; i >= 0; i-- {
			from := w.path
			if i > 0 {
				from = fmt.Sprintf("%s.%d", w.path, i)
			}
			to := fmt.Sprintf("%s.%d", w.path, i+1)
			if i == 2 {
				if err := os.Remove(to); err != nil && !os.IsNotExist(err) {
					return 0, err
				}
			}
			if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
				return 0, err
			}
		}
		if err := w.reopen(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
