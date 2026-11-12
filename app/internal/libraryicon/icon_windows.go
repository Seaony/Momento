package libraryicon

import (
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var changeNotify = windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")

func Install(path string) error {
	installed, err := installIcon(path)
	if err != nil || !installed {
		return err
	}
	for _, name := range []string{"desktop.ini", iconName} {
		if err := addAttributes(filepath.Join(path, name), windows.FILE_ATTRIBUTE_HIDDEN|windows.FILE_ATTRIBUTE_SYSTEM); err != nil {
			return err
		}
	}
	if err := addAttributes(path, windows.FILE_ATTRIBUTE_READONLY); err != nil {
		return err
	}
	if err := changeNotify.Find(); err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	// 通知资源管理器重读当前目录的图标，不刷新其他文件。
	changeNotify.Call(0x00002000, 0x0005, uintptr(unsafe.Pointer(name)), 0)
	runtime.KeepAlive(name)
	return nil
}

func addAttributes(path string, add uint32) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attributes, err := windows.GetFileAttributes(name)
	if err != nil {
		return err
	}
	if attributes&add == add {
		return nil
	}
	return windows.SetFileAttributes(name, (attributes&^windows.FILE_ATTRIBUTE_NORMAL)|add)
}
