package ui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
)

type folderDropPosition int

const (
	folderBefore folderDropPosition = iota
	folderInside
	folderAfter
)

func sidebarCount(n int) string {
	s := fmt.Sprint(n)
	for at := len(s) - 3; at > 0; at -= 3 {
		s = s[:at] + "," + s[at:]
	}
	return s
}

func sidebarFolderChildren(folders []model.Folder) map[string][]model.Folder {
	children := make(map[string][]model.Folder)
	for _, folder := range folders {
		children[folder.ParentID] = append(children[folder.ParentID], folder)
	}
	for parent := range children {
		sort.SliceStable(children[parent], func(i, j int) bool {
			a, b := children[parent][i], children[parent][j]
			if a.SortIndex != b.SortIndex {
				return a.SortIndex < b.SortIndex
			}
			if a.Name != b.Name {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
			return a.ID < b.ID
		})
	}
	return children
}

func sidebarFolderContains(folders []model.Folder, ancestorID, id string) bool {
	byID := make(map[string]model.Folder, len(folders))
	for _, f := range folders {
		byID[f.ID] = f
	}
	for id != "" {
		if id == ancestorID {
			return true
		}
		f, ok := byID[id]
		if !ok {
			return false
		}
		id = f.ParentID
	}
	return false
}

func sidebarDropPosition(y, height float32) folderDropPosition {
	if y < height*.25 {
		return folderBefore
	}
	if y > height*.75 {
		return folderAfter
	}
	return folderInside
}

// 插入位置按移走来源后的同级顺序计算，和图库持久化接口一致。
func sidebarFolderMove(folders []model.Folder, sourceID, targetID string, position folderDropPosition) (string, int, bool) {
	var source, target model.Folder
	sourceFound, targetFound := false, targetID == ""
	for _, f := range folders {
		if f.ID == sourceID {
			source = f
			sourceFound = true
		}
		if f.ID == targetID {
			target = f
			targetFound = true
		}
	}
	if !sourceFound || !targetFound || sourceID == targetID {
		return "", 0, false
	}
	parent := target.ParentID
	if position == folderInside {
		parent = targetID
	}
	if targetID == "" {
		parent = ""
	}
	if sidebarFolderContains(folders, source.ID, parent) {
		return "", 0, false
	}
	siblings := sidebarFolderChildren(folders)[parent]
	index := 0
	for _, f := range siblings {
		if f.ID == sourceID {
			continue
		}
		if targetID != "" && position != folderInside && f.ID == targetID {
			if position == folderAfter {
				index++
			}
			return parent, index, true
		}
		index++
	}
	return parent, index, true
}

func sidebarReorderPaths(paths []string, rows []int, to int) []string {
	selected := make(map[int]bool, len(rows))
	for _, row := range rows {
		if row >= 0 && row < len(paths) {
			selected[row] = true
		}
	}
	to = max(0, min(to, len(paths)))
	var moved, remaining []string
	index := 0
	for i, path := range paths {
		if selected[i] {
			moved = append(moved, path)
		} else {
			if i < to {
				index++
			}
			remaining = append(remaining, path)
		}
	}
	if len(moved) == 0 {
		return slices.Clone(paths)
	}
	out := make([]string, 0, len(paths))
	out = append(out, remaining[:index]...)
	out = append(out, moved...)
	out = append(out, remaining[index:]...)
	return out
}
