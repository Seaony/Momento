package ui

import (
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 筛选和排序弹层在中英日下文字都完整显示；排序点别的字段就换过去，再点当前字段反转方向，弹层不收起。
func TestToolbarPopoversFitAndSortCloses(t *testing.T) {
	for _, language := range []string{"zh", "en", "ja"} {
		t.Run(language, func(t *testing.T) {
			a := sidebarTestApp()
			a.Texts = i18n.Catalog{Language: language}
			a.Toolbar.FilterOpen = true
			// 标签、文件类型只列当前位置里有素材的；有筛选条件时弹层标题栏才有「清除筛选」。
			a.applySnapshot(model.Snapshot{
				Assets: []model.Asset{{ID: "a", Name: "Landscape", Extension: "png", TagIDs: []string{"t1"}, Colors: []model.Color{{Family: "red"}}}},
				Tags:   []model.Tag{{ID: "t1", Name: "Landscape"}},
			})
			a.Query.ColorFamilies = []string{"red"}
			a.Refine()
			type span struct {
				key    string
				size   float32
				weight int
			}
			spans := []span{{"filter", fontBody, 600}, {"clear_filters", fontGroup, 400}, {"filter_colors", fontGroup, 600}, {"filter_tags", fontGroup, 600}, {"filter_types", fontGroup, 600}}
			widths := map[string]float32{}
			tt := ui.NewTester(func(c *ui.Context) {
				a.applyTheme(c)
				for _, s := range spans {
					widths[s.key], _ = c.MeasureText(0, ui.Span{Text: a.T(s.key), Font: c.Theme().Font, Size: s.size, Weight: s.weight, Lang: language})
				}
				for _, key := range []string{"sort_added", "sort_name", "sort_size"} {
					widths[key], _ = c.MeasureText(0, ui.Span{Text: a.T(key), Font: c.Theme().Font, Size: fontBody, Lang: language})
				}
				a.toolbar(c)
			}, 1100, 700)
			tt.SetLocale(a.Texts.Locale())
			tt.Frame()
			// Tester 只能找到整行按钮，量不到文字本身，所以按版面算出文字能用的宽度来比。
			// 标题和「清除筛选」在同一行：弹层宽减去左右留白和清除按钮，剩下的要放得下标题。
			// 工具栏的筛选按钮上也有一个同名的小 ×，这里要的是弹层里那个带文字的。
			var clear ui.Rect
			ok := false
			for _, item := range tt.AccessAll() {
				if item.Name == a.T("clear_filters") && item.Bounds.W > clear.W {
					clear, ok = item.Bounds, true
				}
			}
			if !ok || clear.W-2*popClearPad+1 < widths["clear_filters"] {
				t.Fatalf("清除筛选截断：%+v 需要 %v", clear, widths["clear_filters"])
			}
			if room := float32(popFilterWidth-popInset-(popInset-popClearPad)) - clear.W; room+1 < widths["filter"] {
				t.Fatalf("筛选标题放不下：剩 %v 需要 %v", room, widths["filter"])
			}
			// 小标题不在按钮里，找到的就是文字本身。
			for _, key := range []string{"filter_colors", "filter_tags", "filter_types"} {
				if box, ok := tt.Find(a.T(key)); !ok || box.W+1 < widths[key] {
					t.Fatalf("筛选小标题截断：%s %+v 需要 %v", key, box, widths[key])
				}
			}
			a.Toolbar.FilterOpen, a.Toolbar.SortOpen = false, true
			tt.Frame()
			// 排序行：弹层宽减去内边距、勾选位、方向箭头和间距，按当前字段（带箭头）这种最窄的情况算。
			room := float32(popSortWidth - 2*popSortPadding - popRowPadLeft - popRowPadRight - iconSmall - 2*popRowGap - iconSize)
			for _, key := range []string{"sort_added", "sort_name", "sort_size"} {
				if room+1 < widths[key] {
					t.Fatalf("排序文字放不下：%s 剩 %v 需要 %v", key, room, widths[key])
				}
			}
			if item, ok := tt.AccessByRole(ui.RoleRadio, a.T("sort_added")); !ok || !item.Checked {
				t.Fatal("当前排序字段没有标成选中")
			}
			if err := tt.Click(a.T("sort_name")); err != nil {
				t.Fatal(err)
			}
			if a.Query.Sort != "name" || a.Query.Descending || !a.Toolbar.SortOpen {
				t.Fatalf("换排序字段没有生效，或弹层收起了：%+v open=%v", a.Query, a.Toolbar.SortOpen)
			}
			tt.Frame()
			if err := tt.Click(a.T("sort_name")); err != nil {
				t.Fatal(err)
			}
			if a.Query.Sort != "name" || !a.Query.Descending || !a.Toolbar.SortOpen {
				t.Fatal("再点当前字段没有反转方向")
			}
			if item, ok := tt.AccessByRole(ui.RoleRadio, a.T("sort_name")); !ok || !item.Checked {
				t.Fatal("当前排序字段没有标成选中")
			}
		})
	}
}

// 标签不多时没有搜索框，之前留下的搜索词不能把标签藏起来。
func TestFilterIgnoresHiddenTagSearch(t *testing.T) {
	a := sidebarTestApp()
	a.applySnapshot(model.Snapshot{
		Assets: []model.Asset{{ID: "a", Name: "Landscape", TagIDs: []string{"t1"}}, {ID: "b", Name: "Portrait", TagIDs: []string{"t2"}}},
		Tags:   []model.Tag{{ID: "t1", Name: "Landscape"}, {ID: "t2", Name: "Portrait"}},
	})
	a.Toolbar.TagSearch = "nothing matches"
	a.Toolbar.FilterOpen = true
	tt := ui.NewTester(func(c *ui.Context) { a.applyTheme(c); a.toolbar(c) }, 1100, 700)
	tt.Frame()
	if !tt.HasText("Landscape") || !tt.HasText("Portrait") || tt.HasText(a.T("no_tags")) {
		t.Fatal("搜索框没显示时仍按旧搜索词过滤了标签")
	}
}
