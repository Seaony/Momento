package logging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const maxSize int64 = 5 << 20

type Writer struct {
	mu    sync.Mutex
	path  string
	file  *os.File
	size  int64
	limit int64
	// threshold 是触发轮转的大小；轮转失败后调高，写满下一个上限再重试。
	threshold int64
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
	w := &Writer{path: path, limit: limit, threshold: limit}
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
	if w.size > 0 && w.size+int64(len(p)) > w.threshold {
		if err := w.rotate(); err != nil {
			// 改名失败（Windows 上别的程序开着日志时常见）：接着往原文件写，不丢日志；
			// 原因写进日志本身，再写满一个上限后才重试，免得每行都重试。
			if w.file == nil {
				if openErr := w.reopen(); openErr != nil {
					return 0, errors.Join(err, openErr)
				}
			}
			note := fmt.Sprintf("time=%s level=WARN msg=日志轮转失败，继续写入当前文件 error=%q\n", time.Now().Format(time.RFC3339Nano), err.Error())
			n, writeErr := w.file.WriteString(note)
			w.size += int64(n)
			if writeErr != nil {
				return 0, errors.Join(err, writeErr)
			}
			w.threshold = w.size + w.limit
		} else {
			w.threshold = w.limit
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// rotate 把当前日志改名成 .1，旧的依次后移，最多留 3 个。先把当前日志挪到暂存名：
// 这一步失败时旧日志一个都没动；后面挪旧日志失败时把当前日志挪回原名。成功后打开新文件。
func (w *Writer) rotate() error {
	err := w.file.Close()
	w.file = nil
	if err != nil {
		return err
	}
	staged := w.path + ".rotating"
	if err := os.Rename(w.path, staged); err != nil {
		return err
	}
	for i := 2; i >= 1; i-- {
		from, to := fmt.Sprintf("%s.%d", w.path, i), fmt.Sprintf("%s.%d", w.path, i+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return errors.Join(err, os.Rename(staged, w.path))
		}
	}
	if err := os.Rename(staged, w.path+".1"); err != nil {
		return errors.Join(err, os.Rename(staged, w.path))
	}
	return w.reopen()
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
