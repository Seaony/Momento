package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssetStatusesChecksRowAndPackageFiles(t *testing.T) {
	l := newTestLibrary(t)
	now := "2026-11-22T00:00:00Z"
	if _, err := l.DB.Exec(`INSERT INTO assets(id,relative_path,motion_path,trashed,kind,created_at,updated_at) VALUES
		('active', 'assets/ab/original.png', '', 0, 'image', ?, ?),
		('trashed', 'assets/ab/trashed.png', '', 1, 'image', ?, ?),
		('motion', 'assets/ab/live.png', 'assets/ab/live.mov', 0, 'live', ?, ?)`, now, now, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"assets/ab/original.png", "assets/ab/trashed.png", "assets/ab/live.png", "assets/ab/live.mov"} {
		path := filepath.Join(l.Root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	statuses, err := l.AssetStatuses(t.Context(), []string{"active", "trashed", "motion", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 3 {
		t.Fatalf("状态行数 %d", len(statuses))
	}
	if status := statuses["active"]; !status.OriginalExists || status.Trashed || status.MotionExists {
		t.Fatalf("普通素材状态不对：%+v", status)
	}
	if status := statuses["trashed"]; !status.Trashed || !status.OriginalExists {
		t.Fatalf("回收站素材状态不对：%+v", status)
	}
	if status := statuses["motion"]; !status.OriginalExists || !status.MotionExists || status.Trashed {
		t.Fatalf("实况素材状态不对：%+v", status)
	}
	if _, exists := statuses["missing"]; exists {
		t.Fatal("不存在的素材也返回了状态")
	}
}
