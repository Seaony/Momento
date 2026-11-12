package library

import (
	"database/sql"
	"errors"
	"testing"
)

func TestUpdateAssetDetailsAndClearSource(t *testing.T) {
	l := newTestLibrary(t)
	seedAsset(t, l, "a")
	if err := l.UpdateAssetDetails(t.Context(), "a", "第一行\n第二行", "https://example.com/source"); err != nil {
		t.Fatal(err)
	}
	got := snapshot(t, l).Assets[0]
	if got.Note != "第一行\n第二行" || got.SourceURL != "https://example.com/source" {
		t.Fatal(got)
	}
	if err := l.UpdateAssetDetails(t.Context(), "a", "", ""); err != nil {
		t.Fatal(err)
	}
	got = snapshot(t, l).Assets[0]
	if got.Note != "" || got.SourceURL != "" {
		t.Fatal("清空未保存", got)
	}
	if err := l.UpdateAssetDetails(t.Context(), "missing", "lost", ""); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
}
