package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestBundledFontsAreRegistered(t *testing.T) {
	found := map[string]ui.FontFamily{}
	for _, family := range ui.FontFamilies() {
		if family.Registered {
			found[family.Name] = family
		}
	}
	if _, ok := found["Inter"]; !ok {
		t.Fatal("内置 Inter 未注册")
	}
	if family, ok := found["Geist Mono"]; !ok || !family.Monospaced {
		t.Fatal("内置 Geist Mono 未注册为等宽字体")
	}
}
