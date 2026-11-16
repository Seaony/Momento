package logging

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRotationKeepsThreeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "momento.log")
	w, err := open(path, 8)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"11111111", "22222222", "33333333", "44444444", "55555555"} {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	for suffix, want := range map[string]string{"": "55555555", ".1": "44444444", ".2": "33333333", ".3": "22222222"} {
		got, err := os.ReadFile(path + suffix)
		if err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v", suffix, got, err)
		}
	}
	if _, err := os.Stat(path + ".4"); !os.IsNotExist(err) {
		t.Fatal("保留了多余日志")
	}
}

func TestRotationFailureKeepsWritingAndRecordsReason(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("靠目录只读让改名失败，Windows 和 root 下不成立")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "momento.log")
	w, err := open(path, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	first := strings.Repeat("a", 60)
	if _, err := w.Write([]byte(first)); err != nil {
		t.Fatal(err)
	}
	// 目录只读后改名会失败，模拟 Windows 上日志被别的程序占用。
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	for _, line := range []string{"second..", "third..."} {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("轮转失败后写日志报错：%v", err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	// 失败后写满下一个上限才重试，所以失败原因只记一次。
	if !strings.HasPrefix(text, first) || strings.Count(text, "日志轮转失败") != 1 || !strings.HasSuffix(text, "second..third...") {
		t.Fatalf("日志丢失或没记下失败原因：%q", text)
	}
	if _, err := os.Stat(path + ".1"); !os.IsNotExist(err) {
		t.Fatal("失败时动了旧日志")
	}
	// 恢复后写满下一个上限会重新轮转，之前的内容都在 .1 里。
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(strings.Repeat("x", 100))); err != nil {
		t.Fatal(err)
	}
	rotated, err := os.ReadFile(path + ".1")
	if err != nil || string(rotated) != text {
		t.Fatalf("恢复后没有轮转：%q %v", rotated, err)
	}
	if _, err := os.Stat(path + ".rotating"); !os.IsNotExist(err) {
		t.Fatal("留下了暂存日志")
	}
}
