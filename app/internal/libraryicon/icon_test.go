package libraryicon

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/Seaony/Momento/app/resources"
)

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDesktopINIUsesUnicodeAndRelativeIcon(t *testing.T) {
	digest := iconDigest(resources.LibraryICO)
	data := desktopINI(digest)
	if !bytes.HasPrefix(data, []byte{0xff, 0xfe}) {
		t.Fatal("目录配置缺少 Unicode 标记")
	}
	units := make([]uint16, (len(data)-2)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(data[2+i*2:])
	}
	text := string(utf16.Decode(units))
	if !strings.Contains(text, "IconFile="+iconName+"\r\nIconIndex=0") || strings.Contains(text, ":\\") || strings.Contains(text, "IconFile=/") {
		t.Fatalf("目录配置没有使用包内相对图标：%q", text)
	}
	if got, managed := managedDigest(data); !managed || got != digest {
		t.Fatal("不能识别自己写入的配置")
	}
	if _, managed := managedDigest(append(data, 0, 0)); managed {
		t.Fatal("额外的自定义内容被当作自家配置")
	}
}

func TestInstallIconSurvivesMovingLibrary(t *testing.T) {
	root := filepath.Join(t.TempDir(), "图库.momento")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	if installed, err := installIcon(root); err != nil || !installed {
		t.Fatalf("安装目录图标：%v，%v", installed, err)
	}
	ini := readTestFile(t, filepath.Join(root, "desktop.ini"))
	if !bytes.Equal(readTestFile(t, filepath.Join(root, iconName)), resources.LibraryICO) {
		t.Fatal("图库没有带上自己的图标文件")
	}
	moved := filepath.Join(t.TempDir(), "已移动.momento")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if installed, err := installIcon(moved); err != nil || !installed {
		t.Fatalf("移动后的图库不能使用原图标：%v，%v", installed, err)
	}
	if !bytes.Equal(ini, readTestFile(t, filepath.Join(moved, "desktop.ini"))) {
		t.Fatal("移动图库改变了相对配置")
	}
}

func TestInstallIconUpdatesOnlyManagedFiles(t *testing.T) {
	t.Run("old managed icon", func(t *testing.T) {
		root := t.TempDir()
		old := []byte("old library icon")
		if err := os.WriteFile(filepath.Join(root, iconName), old, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "desktop.ini"), desktopINI(iconDigest(old)), 0644); err != nil {
			t.Fatal(err)
		}
		if installed, err := installIcon(root); err != nil || !installed {
			t.Fatalf("旧图标没有更新：%v，%v", installed, err)
		}
		if !bytes.Equal(readTestFile(t, filepath.Join(root, iconName)), resources.LibraryICO) {
			t.Fatal("旧图标仍未替换")
		}
	})
	for _, name := range []string{"desktop.ini", iconName} {
		t.Run("custom "+name, func(t *testing.T) {
			root := t.TempDir()
			custom := []byte("user customization")
			if err := os.WriteFile(filepath.Join(root, name), custom, 0644); err != nil {
				t.Fatal(err)
			}
			if installed, err := installIcon(root); err != nil || installed {
				t.Fatalf("自定义文件没有保留：%v，%v", installed, err)
			}
			if !bytes.Equal(readTestFile(t, filepath.Join(root, name)), custom) {
				t.Fatal("自定义文件被覆盖")
			}
			files, err := os.ReadDir(root)
			if err != nil || len(files) != 1 {
				t.Fatalf("保留自定义文件时仍新增了文件：%v，%v", files, err)
			}
		})
	}
	t.Run("changed managed icon", func(t *testing.T) {
		root := t.TempDir()
		ini := desktopINI(iconDigest(resources.LibraryICO))
		custom := []byte("user replaced the icon")
		if err := os.WriteFile(filepath.Join(root, "desktop.ini"), ini, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, iconName), custom, 0644); err != nil {
			t.Fatal(err)
		}
		if installed, err := installIcon(root); err != nil || installed {
			t.Fatalf("用户替换的图标没有保留：%v，%v", installed, err)
		}
		if !bytes.Equal(readTestFile(t, filepath.Join(root, iconName)), custom) || !bytes.Equal(readTestFile(t, filepath.Join(root, "desktop.ini")), ini) {
			t.Fatal("自定义图标或配置被覆盖")
		}
	})
}

func TestInstallIconDoesNotFollowCustomSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "custom.ini")
	custom := []byte("untouched")
	if err := os.WriteFile(target, custom, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "desktop.ini")); err != nil {
		t.Skipf("当前环境不允许符号链接：%v", err)
	}
	if installed, err := installIcon(root); err != nil || installed {
		t.Fatalf("自定义链接没有保留：%v，%v", installed, err)
	}
	if !bytes.Equal(readTestFile(t, target), custom) {
		t.Fatal("链接指向的文件被改动")
	}
}

func TestInstallIconReportsInvalidPathAndDoesNotClobberNewFile(t *testing.T) {
	if _, err := installIcon(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("缺失目录没有报错")
	}
	path := filepath.Join(t.TempDir(), "existing")
	if err := os.WriteFile(path, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := installIcon(path); err == nil {
		t.Fatal("普通文件被当成了图库目录")
	}
	if err := writeIconFile(path, []byte("replace"), nil); err == nil {
		t.Fatal("新建图标覆盖了已经存在的文件")
	}
	if string(readTestFile(t, path)) != "keep" {
		t.Fatal("已存在的文件被改动")
	}
}
