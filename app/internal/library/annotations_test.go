package library

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

// v5 图库升到 v6 后能加批注；快照按添加顺序读回；改文字、删除；框不对、素材不存在要报错；素材彻底删除时批注一起删。
func TestAnnotations(t *testing.T) {
	old := legacyLibrary(t)
	for _, migration := range []string{migrationV2, migrationV3, migrationV4, migrationV5} {
		if _, err := old.DB.Exec(migration); err != nil {
			t.Fatal(err)
		}
	}
	old.Manifest.SchemaVersion = 5
	if err := writeJSON(filepath.Join(old.Root, "manifest.json"), old.Manifest); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := old.DB.Exec("INSERT INTO assets(id,name,relative_path,created_at,updated_at) VALUES('a','图','assets/aa/a.png',?,?)", now, now); err != nil {
		t.Fatal(err)
	}
	old.Close()
	l, err := Open(t.Context(), old.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if databaseVersion(t, l.DB) != 6 {
		t.Fatal("v5 没有升到 v6")
	}
	first, err := l.AddAnnotation(t.Context(), "a", model.Annotation{X: .1, Y: .2, W: .3, H: .4, Text: "  logo 太小 "})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.Text != "logo 太小" {
		t.Fatalf("返回的批注不对：%+v", first)
	}
	second, err := l.AddAnnotation(t.Context(), "a", model.Annotation{X: .5, Y: .5, W: .5, H: .5})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []model.Annotation{{X: .9, Y: 0, W: .2, H: .1}, {X: .1, Y: .1, W: 0, H: .1}, {X: -.1, Y: 0, W: .1, H: .1}} {
		if _, err := l.AddAnnotation(t.Context(), "a", bad); err == nil {
			t.Fatalf("越界或太小的框没拦下：%+v", bad)
		}
	}
	if _, err := l.AddAnnotation(t.Context(), "missing", model.Annotation{X: 0, Y: 0, W: .1, H: .1}); err == nil {
		t.Fatal("给不存在的素材加批注没有报错")
	}
	if err := l.UpdateAnnotation(t.Context(), second.ID, "配色参考"); err != nil {
		t.Fatal(err)
	}
	if err := l.UpdateAnnotation(t.Context(), "missing", "x"); err == nil {
		t.Fatal("改不存在的批注没有报错")
	}
	notes := snapshot(t, l).Assets[0].Annotations
	if len(notes) != 2 || notes[0].ID != first.ID || notes[1].Text != "配色参考" || notes[0].W != .3 {
		t.Fatalf("快照里的批注不对：%+v", notes)
	}
	if err := l.DeleteAnnotation(t.Context(), first.ID); err != nil {
		t.Fatal(err)
	}
	if err := l.DeleteAnnotation(t.Context(), first.ID); err == nil {
		t.Fatal("重复删除没有报错")
	}
	if _, err := l.DB.Exec("DELETE FROM assets WHERE id='a'"); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := l.DB.QueryRow("SELECT count(*) FROM asset_annotations").Scan(&left); err != nil || left != 0 {
		t.Fatalf("素材删了批注还在：%d %v", left, err)
	}
}
