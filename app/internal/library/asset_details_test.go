package library

import (
	"database/sql"
	"errors"
	"testing"
)

func TestSaveAssetEditDetailsAndClearSource(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	if err := l.SaveAssetEdit(t.Context(), "a", "", false, "第一行\n第二行", "https://example.com/source", true); err != nil {
		t.Fatal(err)
	}
	got := snapshot(t, l).Assets[0]
	if got.Note != "第一行\n第二行" || got.SourceURL != "https://example.com/source" {
		t.Fatal(got)
	}
	if err := l.SaveAssetEdit(t.Context(), "a", "", false, "", "", true); err != nil {
		t.Fatal(err)
	}
	got = snapshot(t, l).Assets[0]
	if got.Note != "" || got.SourceURL != "" {
		t.Fatal("清空未保存", got)
	}
	if err := l.SaveAssetEdit(t.Context(), "missing", "", false, "lost", "", true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
}

// 改名和改详情一起提交：名称不合法时两样都不改，不能只写进一半。
func TestSaveAssetEditIsAllOrNothing(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	before := snapshot(t, l).Assets[0]
	if err := l.SaveAssetEdit(t.Context(), "a", "  ", true, "新备注", "https://example.com/new", true); err == nil {
		t.Fatal("空名称应报错")
	}
	if got := snapshot(t, l).Assets[0]; got.Name != before.Name || got.Note != before.Note || got.SourceURL != before.SourceURL {
		t.Fatalf("失败后数据被改了一半：%+v", got)
	}
	if err := l.SaveAssetEdit(t.Context(), "a", "新名字", true, "新备注", "https://example.com/new", true); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l).Assets[0]; got.Name != "新名字" || got.Note != "新备注" || got.SourceURL != "https://example.com/new" {
		t.Fatalf("没有一起写进去：%+v", got)
	}
}

// 新建标签或文件夹并加到素材上：加不上时连标签、文件夹也不留下。
func TestCreateWithAssetsRollsBack(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	if _, err := l.CreateTagWithAssets(t.Context(), "新标签", []string{"a", "missing"}); err == nil {
		t.Fatal("素材不存在时应报错")
	}
	if _, err := l.CreateFolderWithAssets(t.Context(), "新文件夹", "", "", []string{"missing"}); err == nil {
		t.Fatal("素材不存在时应报错")
	}
	s := snapshot(t, l)
	if len(s.Tags) != 0 || len(s.Folders) != 0 || len(s.Assets[0].TagIDs) != 0 {
		t.Fatalf("失败后留下了半成品：标签 %v，文件夹 %v", s.Tags, s.Folders)
	}
	tag, err := l.CreateTagWithAssets(t.Context(), "新标签", []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot(t, l).Assets[0].TagIDs; len(got) != 1 || got[0] != tag.ID {
		t.Fatalf("新标签没加到素材上：%v", got)
	}
}
