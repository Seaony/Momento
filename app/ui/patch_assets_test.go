package ui

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
)

// 写库后直接改内存的结果，必须和重新从数据库读出来的一样，不然界面和数据会对不上。
func TestAssetPatchesMatchDatabase(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Create(ctx, filepath.Join(t.TempDir(), "Patch.momento"), "Patch")
	if err != nil {
		t.Fatal(err)
	}
	defer lib.Close()
	dir := t.TempDir()
	var paths []string
	for i := range 3 {
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		img.Set(0, 0, color.RGBA{R: uint8(40 * i), A: 255})
		path := filepath.Join(dir, string(rune('a'+i))+".png")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
		f.Close()
		paths = append(paths, path)
	}
	service := assets.New(lib.Root, lib.DB)
	if _, err = service.Import(ctx, model.ImportRequest{Paths: paths}, nil); err != nil {
		t.Fatal(err)
	}
	alpha, err := lib.CreateTag(ctx, "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	beta, err := lib.CreateTag(ctx, "Beta")
	if err != nil {
		t.Fatal(err)
	}
	created, err := lib.CreateFolder(ctx, "Folder", "", "")
	if err != nil {
		t.Fatal(err)
	}
	tagA, tagB, folder := alpha.ID, beta.ID, created.ID
	snapshot, err := lib.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Library: lib, Texts: i18n.Catalog{Language: "en"}, Preferences: DefaultPreferences()}
	a.applySnapshot(snapshot)
	ids := []string{snapshot.Assets[0].ID, snapshot.Assets[1].ID}
	steps := []struct {
		name  string
		db    func() error
		patch func(*model.Asset)
	}{
		{"favorite", func() error { return lib.SetFavorite(ctx, ids, true) }, func(x *model.Asset) { x.Favorite = true }},
		{"rating", func() error { return lib.SetRating(ctx, ids, 4) }, func(x *model.Asset) { x.Rating = 4 }},
		{"rename", func() error { return lib.RenameAssets(ctx, map[string]string{ids[0]: "First", ids[1]: "Second"}) }, func(x *model.Asset) {
			x.Name = map[string]string{ids[0]: "First", ids[1]: "Second"}[x.ID]
		}},
		{"tags-add", func() error { return lib.SetAssetTags(ctx, ids, []string{tagB, tagA}, true) }, func(x *model.Asset) { x.TagIDs = setMembership(x.TagIDs, []string{tagB, tagA}, true) }},
		{"tags-remove", func() error { return lib.SetAssetTags(ctx, ids[:1], []string{tagA}, false) }, nil},
		{"folders", func() error { return lib.SetAssetFolders(ctx, ids, []string{folder}, true) }, func(x *model.Asset) { x.FolderIDs = setMembership(x.FolderIDs, []string{folder}, true) }},
		{"trash", func() error { return service.Trash(ctx, ids) }, func(x *model.Asset) { x.Trashed = true }},
		{"restore", func() error { return service.Restore(ctx, ids[1:]) }, nil},
		{"details", func() error {
			if err := lib.RenameAsset(ctx, ids[0], "Renamed"); err != nil {
				return err
			}
			return lib.UpdateAssetDetails(ctx, ids[0], "note", "https://example.com")
		}, nil},
	}
	for _, step := range steps {
		if err := step.db(); err != nil {
			t.Fatalf("%s：%v", step.name, err)
		}
		target := ids
		switch step.name {
		case "tags-remove":
			target, step.patch = ids[:1], func(x *model.Asset) { x.TagIDs = setMembership(x.TagIDs, []string{tagA}, false) }
		case "restore":
			target, step.patch = ids[1:], func(x *model.Asset) { x.Trashed = false }
		case "details":
			target, step.patch = ids[:1], func(x *model.Asset) { x.Name, x.Note, x.SourceURL = "Renamed", "note", "https://example.com" }
		}
		a.patchAssets(target, step.patch)
		fresh, err := lib.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range fresh.Assets {
			got := a.Snapshot.Assets[i]
			got.UpdatedAt, want.UpdatedAt = want.UpdatedAt, got.UpdatedAt
			if got.ID != want.ID || got.Name != want.Name || got.Note != want.Note || got.SourceURL != want.SourceURL || got.Favorite != want.Favorite || got.Rating != want.Rating || got.Trashed != want.Trashed || !slices.Equal(got.TagIDs, want.TagIDs) || !slices.Equal(got.FolderIDs, want.FolderIDs) {
				t.Fatalf("%s 之后内存和数据库不一致：\n内存 %+v\n库里 %+v", step.name, got, want)
			}
		}
	}
	if counts := a.Index.Counts(); counts.Trash != 1 || counts.Favorites != 1 {
		t.Fatalf("计数没有跟着更新：%+v", counts)
	}
}
