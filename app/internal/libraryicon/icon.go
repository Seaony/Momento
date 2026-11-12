package libraryicon

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/Seaony/Momento/app/resources"
)

const iconName = ".momento-library.ico"
const desktopTemplate = "; Momento library icon v1\r\n; SHA256=%s\r\n[.ShellClassInfo]\r\nIconFile=" + iconName + "\r\nIconIndex=0\r\nConfirmFileOp=0\r\n"

var errCustomFile = errors.New("已有自定义文件")

// 配置只写包内相对文件名，移动和复制图库时一起保留。
func desktopINI(digest string) []byte {
	units := utf16.Encode([]rune(fmt.Sprintf(desktopTemplate, digest)))
	data := make([]byte, 2+len(units)*2)
	data[0], data[1] = 0xff, 0xfe
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2+i*2:], unit)
	}
	return data
}

func managedDigest(data []byte) (string, bool) {
	if len(data) < 2 || len(data)%2 != 0 || data[0] != 0xff || data[1] != 0xfe {
		return "", false
	}
	units := make([]uint16, (len(data)-2)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(data[2+i*2:])
	}
	text := string(utf16.Decode(units))
	_, tail, ok := strings.Cut(text, "; SHA256=")
	if !ok || len(tail) < sha256.Size*2 {
		return "", false
	}
	digest := tail[:sha256.Size*2]
	if _, err := hex.DecodeString(digest); err != nil || !bytes.Equal(data, desktopINI(digest)) {
		return "", false
	}
	return digest, true
}

func iconDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errCustomFile
	}
	return os.ReadFile(path)
}

func installIcon(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return false, fmt.Errorf("图库不是目录：%s", path)
	}
	iniPath, iconPath := filepath.Join(path, "desktop.ini"), filepath.Join(path, iconName)
	ini, err := readRegular(iniPath)
	if errors.Is(err, errCustomFile) {
		slog.Info("保留图库已有的文件夹配置", "path", iniPath)
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	previousDigest := ""
	if err == nil {
		var managed bool
		previousDigest, managed = managedDigest(ini)
		if !managed {
			slog.Info("保留图库已有的文件夹配置", "path", iniPath)
			return false, nil
		}
	}
	icon, err := readRegular(iconPath)
	if errors.Is(err, errCustomFile) {
		slog.Info("保留图库已有的文件夹图标", "path", iconPath)
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err == nil && !bytes.Equal(icon, resources.LibraryICO) && (previousDigest == "" || iconDigest(icon) != previousDigest) {
		slog.Info("保留图库已有的文件夹图标", "path", iconPath)
		return false, nil
	}
	if !bytes.Equal(icon, resources.LibraryICO) {
		if err := writeIconFile(iconPath, resources.LibraryICO, icon); err != nil {
			return false, err
		}
	}
	wanted := desktopINI(iconDigest(resources.LibraryICO))
	if !bytes.Equal(ini, wanted) {
		if err := writeIconFile(iniPath, wanted, ini); err != nil {
			return false, err
		}
	}
	return true, nil
}

func writeIconFile(path string, data, previous []byte) (err error) {
	if previous == nil {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(data)
		return errors.Join(writeErr, file.Close())
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".momento-icon-*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(file.Name()); removeErr != nil && !os.IsNotExist(removeErr) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(data)
	if err = errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
