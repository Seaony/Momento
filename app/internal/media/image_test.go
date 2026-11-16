package media

import "testing"

func TestSystemImageExtensions(t *testing.T) {
	for _, ext := range []string{"NEF", ".dng", "cr3", "psd", "ICO"} {
		if !IsSystemImageExtension(ext) {
			t.Fatalf("%s 应交给系统解码", ext)
		}
	}
	if IsSystemImageExtension("jpg") || IsSystemImageExtension("heic") || !IsRawExtension("ARW") || IsRawExtension("psd") {
		t.Fatal("格式分类不对")
	}
}
