package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// 弹出层里的搜索框：有内容时 Esc 先清空、弹出层不关；点清除后焦点回到输入框，能接着输入。
func TestSearchFieldInPopoverClearsBeforeClosing(t *testing.T) {
	open, query := false, ""
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(momentoTheme("light", false))
		anchor := ui.Button(c.Key("anchor"), "Open").Absolute().Top(10).Left(10)
		if anchor.Clicked() {
			open = true
		}
		ui.Popover(c.Key("popover"), anchor, &open, func() {
			field, _ := searchField(c.Key("search"), &query, "Find", "Clear")
			field.Width(200)
		})
	}, 400, 300)
	tt.Frame()
	if err := tt.Click("Open"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Find"); err != nil {
		t.Fatal(err)
	}
	tt.Type("cat")
	tt.Frame()
	if query != "cat" {
		t.Fatalf("query = %q", query)
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if query != "" || !open {
		t.Fatalf("after Esc query = %q open = %v, want cleared and still open", query, open)
	}
	tt.Type("dog")
	tt.Frame()
	if err := tt.Click("Clear"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if query != "" || !tt.Focused("Find") {
		t.Fatalf("after clear query = %q focused = %v", query, tt.Focused("Find"))
	}
	tt.Type("x")
	tt.Frame()
	if query != "x" {
		t.Fatalf("typing after clear gave %q", query)
	}
}
