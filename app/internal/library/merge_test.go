package library

import (
	"slices"
	"testing"

	"github.com/Seaony/Momento/app/internal/model"
)

// 合并：标签、文件夹并到保留的那张，备注接起来，其余进回收站；中途失败整体回滚。
func TestMergeAssets(t *testing.T) {
	l := newTestLibrary(t)
	for _, id := range []string{"keep", "dup1", "dup2"} {
		seedAsset(t, l, id)
	}
	tag, err := l.CreateTag(t.Context(), "猫")
	if err != nil {
		t.Fatal(err)
	}
	folder, err := l.CreateFolder(t.Context(), "参考", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetTags(t.Context(), []string{"dup1"}, []string{tag.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SetAssetFolders(t.Context(), []string{"dup2"}, []string{folder.ID}, true); err != nil {
		t.Fatal(err)
	}
	if err := l.SaveAssetEdit(t.Context(), "keep", "", false, "原备注", "", true); err != nil {
		t.Fatal(err)
	}
	if err := l.SaveAssetEdit(t.Context(), "dup1", "", false, "另一段", "https://example.com/1", true); err != nil {
		t.Fatal(err)
	}
	assets := snapshot(t, l).Assets
	var keep, dup1, dup2 model.Asset
	for _, asset := range assets {
		switch asset.ID {
		case "keep":
			keep = asset
		case "dup1":
			dup1 = asset
		case "dup2":
			dup2 = asset
		}
	}
	missing := dup2
	missing.ID = "missing"
	if err := l.MergeAssets(t.Context(), MergeGroup{Keep: keep, Others: []model.Asset{dup1, missing}}); err == nil {
		t.Fatal("合并里有不存在的素材时没有报错")
	}
	for _, asset := range snapshot(t, l).Assets {
		if asset.Trashed || (asset.ID == "keep" && len(asset.TagIDs) != 0) {
			t.Fatal("合并失败没有整体回滚", asset)
		}
	}
	if err := l.MergeAssets(t.Context(), MergeGroup{Keep: keep, Others: []model.Asset{dup1, dup2}}); err != nil {
		t.Fatal(err)
	}
	for _, asset := range snapshot(t, l).Assets {
		switch asset.ID {
		case "keep":
			want := MergedAsset(keep, []model.Asset{dup1, dup2})
			if asset.Trashed || asset.Note != "原备注\n\n另一段" || asset.SourceURL != "https://example.com/1" ||
				!slices.Equal(asset.TagIDs, []string{tag.ID}) || !slices.Equal(asset.FolderIDs, []string{folder.ID}) || asset.Note != want.Note {
				t.Fatalf("保留的那张没合并对：%+v", asset)
			}
		default:
			if !asset.Trashed {
				t.Fatalf("%s 没进回收站", asset.ID)
			}
		}
	}
}

// 「全部合并」是多组一起提交：后面一组失败时，前面已经并好的组也要回滚。
func TestMergeAssetsGroupsRollBackTogether(t *testing.T) {
	l := newTestLibrary(t)
	for _, id := range []string{"a1", "a2", "b1", "b2"} {
		seedAsset(t, l, id)
	}
	byID := map[string]model.Asset{}
	for _, asset := range snapshot(t, l).Assets {
		byID[asset.ID] = asset
	}
	ghost := byID["b2"]
	ghost.ID = "ghost"
	err := l.MergeAssets(t.Context(),
		MergeGroup{Keep: byID["a1"], Others: []model.Asset{byID["a2"]}},
		MergeGroup{Keep: byID["b1"], Others: []model.Asset{ghost}})
	if err == nil {
		t.Fatal("第二组有不存在的素材时没有报错")
	}
	for _, asset := range snapshot(t, l).Assets {
		if asset.Trashed {
			t.Fatalf("失败后 %s 仍进了回收站", asset.ID)
		}
	}
	if err := l.MergeAssets(t.Context(),
		MergeGroup{Keep: byID["a1"], Others: []model.Asset{byID["a2"]}},
		MergeGroup{Keep: byID["b1"], Others: []model.Asset{byID["b2"]}}); err != nil {
		t.Fatal(err)
	}
	for _, asset := range snapshot(t, l).Assets {
		if asset.Trashed != (asset.ID == "a2" || asset.ID == "b2") {
			t.Fatalf("%s 回收站状态不对: %v", asset.ID, asset.Trashed)
		}
	}
}
