package ui

import (
	"slices"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

func TestSidebarCounts(t *testing.T) {
	for n, want := range map[int]string{0: "0", 14: "14", 12000: "12,000", 1000000: "1,000,000"} {
		if got := sidebarCount(n); got != want {
			t.Errorf("%d: %s", n, got)
		}
	}
}

func TestSidebarFolderMoves(t *testing.T) {
	folders := []model.Folder{{ID: "a", Name: "A", SortIndex: 0}, {ID: "b", Name: "B", SortIndex: 1}, {ID: "c", Name: "C", SortIndex: 2}, {ID: "a1", Name: "A1", ParentID: "a", SortIndex: 0}, {ID: "a2", Name: "A2", ParentID: "a", SortIndex: 1}, {ID: "a11", Name: "A11", ParentID: "a1"}}
	for _, tc := range []struct {
		name, source, target, parent string
		position                     folderDropPosition
		index                        int
		valid                        bool
	}{
		{"same_parent_forward", "a", "c", "", folderAfter, 2, true},
		{"same_parent_backward", "c", "a", "", folderBefore, 0, true},
		{"same_parent_adjacent", "b", "c", "", folderBefore, 1, true},
		{"new_parent", "b", "a", "a", folderInside, 2, true},
		{"new_parent_before", "b", "a2", "a", folderBefore, 1, true},
		{"to_root", "a1", "", "", folderInside, 3, true},
		{"self", "a", "a", "", folderInside, 0, false},
		{"descendant", "a", "a11", "", folderInside, 0, false},
		{"before_descendant", "a", "a11", "", folderBefore, 0, false},
		{"missing_source", "missing", "a", "", folderInside, 0, false},
		{"missing_target", "a", "missing", "", folderInside, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, index, valid := sidebarFolderMove(folders, tc.source, tc.target, tc.position)
			if valid != tc.valid || valid && (parent != tc.parent || index != tc.index) {
				t.Fatalf("got %s %d %v, want %s %d %v", parent, index, valid, tc.parent, tc.index, tc.valid)
			}
		})
	}
	if !sidebarFolderContains(folders, "a", "a11") || sidebarFolderContains(folders, "b", "a11") {
		t.Fatal("子孙判断错误")
	}
}

func TestSidebarDropThirds(t *testing.T) {
	for _, tc := range []struct {
		y    float32
		want folderDropPosition
	}{{0, folderBefore}, {8, folderBefore}, {9, folderInside}, {17, folderInside}, {26, folderAfter}, {34, folderAfter}} {
		if got := sidebarDropPosition(tc.y, 34); got != tc.want {
			t.Errorf("%v: %d", tc.y, got)
		}
	}
}

func TestSidebarRecentReorder(t *testing.T) {
	paths := []string{"a", "b", "c", "d"}
	for _, tc := range []struct {
		rows []int
		to   int
		want []string
	}{{[]int{0}, 3, []string{"b", "c", "a", "d"}}, {[]int{3}, 0, []string{"d", "a", "b", "c"}}, {[]int{0, 2}, 4, []string{"b", "d", "a", "c"}}, {[]int{1}, 2, paths}, {nil, 0, paths}} {
		if got := sidebarReorderPaths(paths, tc.rows, tc.to); !slices.Equal(got, tc.want) {
			t.Fatalf("%v to %d: %v, want %v", tc.rows, tc.to, got, tc.want)
		}
	}
	if !slices.Equal(paths, []string{"a", "b", "c", "d"}) {
		t.Fatal("修改了原始列表")
	}
}
