package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

func TestInspectorPolishScreenshots(t *testing.T) {
	if os.Getenv("MOMENTO_INSPECTOR_POLISH_SCREENSHOTS") != "1" {
		t.Skip("设置 MOMENTO_INSPECTOR_POLISH_SCREENSHOTS=1 生成检查器、标签和智能编辑器截图")
	}
	dir := "/tmp/momento-polish/shots/inspector"
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := newScreenshotFixture(t)
	for _, theme := range []string{"light", "dark"} {
		for _, state := range []string{"single-details", "single-details-bottom", "tag-picker", "folder-picker", "tags", "tags-empty", "smart-tags", "smart-dimensions"} {
			t.Run(theme+"/"+state, func(t *testing.T) {
				base := state
				if state == "tags-empty" {
					base = "small"
				}
				if state == "smart-tags" || state == "smart-dimensions" {
					base = "tags"
				}
				a := fixture.app(t, base, "zh-CN", theme)
				if state == "tags-empty" {
					a.Snapshot.Tags = nil
					a.applySnapshot(a.Snapshot)
					a.TagPage = true
				}
				if state == "smart-tags" || state == "smart-dimensions" {
					a.EditSmartFolder("")
					a.GBrowsing.SmartName = "设计参考"
					a.GBrowsing.SmartRules = []model.SmartRule{{Field: "extension", Operator: "equals", Value: "png"}, {Field: "tag", Operator: "equals", Value: a.Snapshot.Tags[0].ID}, {Field: "color", Operator: "equals", Value: "orange"}}
					if state == "smart-dimensions" {
						a.GBrowsing.SmartRules = []model.SmartRule{{Field: "width", Operator: "gte", Value: "1920"}, {Field: "height", Operator: "gte", Value: "1080"}, {Field: "size", Operator: "lte", Value: "20.5"}, {Field: "added", Operator: "after", Value: "2026-11-01"}, {Field: "note", Operator: "contains", Value: "设计参考"}}
						a.GBrowsing.SmartError = a.T("error_smart_rule")
					}
				}
				tt := ui.NewTester(a.View, 1440, 900)
				tt.SetLocale(a.Texts.Locale())
				tt.SetTitleBar(ui.TitleBar{Height: MainToolbarHeight, Left: 80, Bottom: 37})
				tt.SetWindowActive(true)
				tt.Frame()
				if state == "single-details-bottom" {
					tt.Scroll(1340, 800, 0, 2000)
					tt.Advance(time.Second)
				}
				if state == "smart-tags" {
					if err := tt.Click(a.T("smart_rule_field", 1)); err != nil {
						t.Fatal(err)
					}
					if !tt.HasText(a.T("smart_field_added")) {
						t.Fatal("智能字段下拉未打开")
					}
				}
				writeScreenshot(t, filepath.Join(dir, state+"-zh-CN-"+theme+".png"), tt)
			})
		}
	}
}
