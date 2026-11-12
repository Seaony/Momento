package logging

import (
	"os"
	"path/filepath"
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
