package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func installLibraryFileAssociation() error {
	directory, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return err
	}
	icon, err := filepath.Abs(filepath.Join(directory, "library.ico"))
	if err != nil {
		return err
	}
	if _, err := os.Stat(icon); err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Classes\.momento\DefaultIcon`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	value := `"` + icon + `",0`
	current, _, readErr := key.GetStringValue("")
	if readErr != nil && !errors.Is(readErr, registry.ErrNotExist) {
		return errors.Join(readErr, key.Close())
	}
	if current == value {
		return key.Close()
	}
	if err := errors.Join(key.SetStringValue("", value), key.Close()); err != nil {
		return err
	}
	notify := windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")
	if err := notify.Find(); err != nil {
		return err
	}
	notify.Call(0x08000000, 0, 0, 0)
	return nil
}
