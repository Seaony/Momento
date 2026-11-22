package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestImportFolderQueryPlan(t *testing.T) {
	ctx, lib, _ := testLibrary(t)
	for _, query := range []struct {
		name, sql string
		args      []any
		indexed   bool
	}{
		{"旧名称查找", "SELECT id FROM folders WHERE name=? AND COALESCE(parent_id,'')=? ORDER BY sort_index,id LIMIT 1", []any{"folder", ""}, false},
		{"旧排序上界", "SELECT COALESCE(MAX(sort_index)+1,0) FROM folders WHERE COALESCE(parent_id,'')=?", []any{""}, false},
		{"新根目录", importFolderChildrenQuery, []any{nil}, true},
		{"新子目录", importFolderChildrenQuery, []any{"parent"}, true},
	} {
		rows, err := lib.DB.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query.sql, query.args...)
		if err != nil {
			t.Fatal(err)
		}
		var plan []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err = rows.Scan(&id, &parent, &unused, &detail); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			plan = append(plan, detail)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %s", query.name, strings.Join(plan, " | "))
		if indexed := slices.ContainsFunc(plan, func(detail string) bool {
			return strings.Contains(detail, "folders_parent (parent_id=?)")
		}); indexed != query.indexed {
			t.Fatalf("父目录索引计划不符：%v", plan)
		}
	}
}

func TestImportFoldersReuseSortedSiblingAndAppendPerParent(t *testing.T) {
	ctx, lib, service := testLibrary(t)
	_, err := lib.DB.ExecContext(ctx, `
		INSERT INTO folders(id,name,parent_id,sort_index) VALUES
		  ('later','parent',NULL,4),('earlier','parent',NULL,4),('last','last',NULL,18),
		  ('nested','nested','earlier',70),('unrelated','nested','later',90);`)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for _, folder := range []string{"root-a", "root-b", "parent/nested", "parent/child-a", "parent/child-b"} {
		fixtureCopy(t, "png", filepath.Join(source, filepath.FromSlash(folder), "photo.png"))
	}
	if _, err := service.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
		t.Fatal(err)
	}
	snap := snapshot(t, ctx, lib)
	want := map[string]struct {
		parent string
		sort   int
	}{"root-a": {"", 19}, "root-b": {"", 20}, "child-a": {"earlier", 71}, "child-b": {"earlier", 72}}
	wantIDs := []string{"nested"}
	for _, folder := range snap.Folders {
		if expected, ok := want[folder.Name]; ok {
			if folder.ParentID != expected.parent || folder.SortIndex != expected.sort {
				t.Fatalf("目录顺序或父目录不对：%+v", folder)
			}
			wantIDs = append(wantIDs, folder.ID)
		}
	}
	if len(wantIDs) != 5 || len(snap.Assets) != 1 || !slices.Equal(slices.Sorted(slices.Values(snap.Assets[0].FolderIDs)), slices.Sorted(slices.Values(wantIDs))) {
		t.Fatalf("复用或新建的目录未关联到素材：%+v", snap)
	}
}

func BenchmarkImportFolderBatch(b *testing.B) {
	source := b.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "sample.png"))
	if err != nil {
		b.Fatal(err)
	}
	for index := range 200 {
		folder := filepath.Join(source, "parent", fmt.Sprintf("new-%03d", index))
		if err := os.MkdirAll(folder, 0700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "photo.png"), data, 0600); err != nil {
			b.Fatal(err)
		}
	}
	ctx := b.Context()
	for b.Loop() {
		b.StopTimer()
		lib, err := library.Create(ctx, filepath.Join(b.TempDir(), "Bench.momento"), "Bench")
		if err != nil {
			b.Fatal(err)
		}
		_, err = lib.DB.ExecContext(ctx, `
			INSERT INTO folders(id,name) VALUES('parent','parent'),('other','other');
			WITH RECURSIVE numbers(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM numbers WHERE n<100000)
			INSERT INTO folders(id,name,parent_id,sort_index)
			SELECT 'folder-'||n,'folder-'||n,CASE WHEN n<=1000 THEN 'parent' ELSE 'other' END,n FROM numbers;
			INSERT INTO assets(id,relative_path,created_at,updated_at)
			SELECT id,'assets/'||id||'.png','2026-11-22T00:00:00Z','2026-11-22T00:00:00Z'
			FROM folders WHERE parent_id IS NOT NULL;`)
		if err != nil {
			lib.Close()
			b.Fatal(err)
		}
		service := New(lib.Root, lib.DB)
		b.StartTimer()
		result, err := service.Import(ctx, model.ImportRequest{Paths: []string{source}}, nil)
		b.StopTimer()
		closeErr := lib.Close()
		if err != nil || closeErr != nil || result.Imported != 1 || result.Skipped != 199 {
			b.Fatalf("导入结果 %+v，错误 %v / %v", result, err, closeErr)
		}
		b.StartTimer()
	}
}
