package library

import (
	"github.com/Seaony/Momento/app/internal/i18n"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrCloudLibrary = i18n.NewError("error_icloud_library")

func ValidateWorkingPath(path string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return validateWorkingPath(path, home)
}

func validateWorkingPath(path, home string) error {
	resolved, err := resolvePath(path)
	if err != nil {
		return err
	}
	cloud, err := resolvePath(filepath.Join(home, "Library", "Mobile Documents"))
	if err != nil {
		return err
	}
	if withinPath(strings.ToLower(cloud), strings.ToLower(resolved)) {
		return ErrCloudLibrary
	}
	return nil
}

func resolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(abs)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, tail[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		tail = append(tail, filepath.Base(abs))
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", err
		}
		abs = parent
	}
}

func withinPath(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))))
}
