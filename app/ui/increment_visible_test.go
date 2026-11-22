package ui

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestIncrementalVisibleMatchesFullRefine(t *testing.T) {
	// 固定种子生成重复值和边界数据，增量结果必须和整次刷新逐项一致。
	scopes := []func(snapshot model.Snapshot, now time.Time) catalog.Query{
		func(model.Snapshot, time.Time) catalog.Query { return catalog.Query{Scope: "all"} },
		func(model.Snapshot, time.Time) catalog.Query { return catalog.Query{Scope: "favorites"} },
		func(snapshot model.Snapshot, _ time.Time) catalog.Query {
			return catalog.Query{Scope: "folder", FolderID: snapshot.Folders[0].ID}
		},
		func(model.Snapshot, time.Time) catalog.Query { return catalog.Query{Scope: "tag"} },
		func(_ model.Snapshot, now time.Time) catalog.Query { return catalog.Query{Scope: "recent", Now: now} },
		func(model.Snapshot, time.Time) catalog.Query { return catalog.Query{Scope: "trash"} },
	}
	sortNames := []string{"added", "name", "size", "manual"}

	for scopeIndex, makeScope := range scopes {
		for _, sortName := range sortNames {
			for _, descending := range []bool{false, true} {
				seed := uint64(0x5eed0000 + scopeIndex<<8)
				for _, char := range sortName {
					seed = seed*131 + uint64(char)
				}
				if descending {
					seed++
				}
				name := fmt.Sprintf("%s/%s/%s", scopeLabel(scopeIndex), sortName, directionLabel(descending))
				t.Run(name, func(t *testing.T) {
					random := rand.New(rand.NewPCG(seed, seed^0xa5a5_5a5a))
					app, snapshot, ids, now := newIncrementEquivalenceApp(random, makeScope, sortName, descending)
					for step := range 120 {
						var action string
						action, ids = prepareIncrementalStep(app, random, snapshot, ids, now, step)
						wantVisible, wantSelected := incrementalState(app)
						app.Refine()
						gotVisible, gotSelected := incrementalState(app)
						if !slices.Equal(gotVisible, wantVisible) {
							t.Fatalf("%s 第 %d 步 %s 后 Visible 不一致\n增量 %v\n整查 %v", name, step, action, gotVisible, wantVisible)
						}
						if !slices.Equal(gotSelected, wantSelected) {
							if step == 0 {
								t.Logf("之后计数 %d 013 %v 091 %v", app.Selected.Len(), app.Selected.Has("asset-013"), app.Selected.Has("asset-091"))
							}
							t.Fatalf("%s 第 %d 步 %s 后 Selected 不一致\n增量 %v\n整查 %v", name, step, action, gotSelected, wantSelected)
						}
					}
				})
			}
		}
	}
}

func newIncrementEquivalenceApp(random *rand.Rand, makeScope func(model.Snapshot, time.Time) catalog.Query, sortName string, descending bool) (*App, model.Snapshot, []string, time.Time) {
	now := time.Date(2026, 5, 4, 15, 6, 7, 0, time.UTC)
	snapshot := model.Snapshot{Folders: make([]model.Folder, 5), Tags: make([]model.Tag, 7)}
	for i := range snapshot.Folders {
		snapshot.Folders[i] = model.Folder{ID: fmt.Sprintf("folder-%d", i), Name: fmt.Sprintf("Folder %02d", i%3)}
	}
	for i := range snapshot.Tags {
		snapshot.Tags[i] = model.Tag{ID: fmt.Sprintf("tag-%d", i), Name: fmt.Sprintf("Tag %02d", i%4)}
	}

	names := []string{"Aurora", "Branch", "Cloud", "Delta"}
	for i := range 300 {
		id := fmt.Sprintf("asset-%03d", i)
		createdAt := now.Add(-time.Duration(24*(i%45)+i) * time.Hour)
		var lastUsed time.Time
		if i%5 == 0 {
			lastUsed = now.Add(-time.Duration(i%29) * 24 * time.Hour)
		}
		asset := model.Asset{
			ID: id, Name: names[i%len(names)], Extension: []string{"jpg", "png", "mov"}[i%3],
			Size: int64((i % 9) * 1000), Width: 100 + 40*(i%4), Height: 80 + 30*((i+1)%4),
			CreatedAt: createdAt, LastUsedAt: lastUsed, Favorite: i%3 == 1, Trashed: i%13 == 0,
			Kind: []string{model.KindImage, model.KindVideo}[i%2], Rotation: 90 * (i % 4),
			FolderIDs: randomFolderMembership(snapshot.Folders, random), TagIDs: randomTagMembership(snapshot.Tags, random),
		}
		snapshot.Assets = append(snapshot.Assets, asset)
	}

	query := makeScope(snapshot, now)
	query.Sort, query.Descending = sortName, descending
	app := &App{
		Preferences: DefaultPreferences(), Texts: i18n.Catalog{Language: "en"}, Query: query,
		selectedIndex: -1, anchor: -1,
	}
	if query.Scope == "tag" {
		app.Browse.TagID = snapshot.Tags[0].ID
	}
	app.applySnapshot(snapshot)
	ids := make([]string, len(snapshot.Assets))
	for i, asset := range snapshot.Assets {
		ids[i] = asset.ID
	}
	return app, snapshot, ids, now
}

func randomFolderMembership(folders []model.Folder, random *rand.Rand) []string {
	return randomIDs(folders, func(item model.Folder) string { return item.ID }, random)
}

func randomTagMembership(tags []model.Tag, random *rand.Rand) []string {
	return randomIDs(tags, func(item model.Tag) string { return item.ID }, random)
}

func randomIDs[T any](items []T, id func(T) string, random *rand.Rand) []string {
	count := random.IntN(3)
	out := make([]string, 0, count)
	for range count {
		member := id(items[random.IntN(len(items))])
		if !slices.Contains(out, member) {
			out = append(out, member)
		}
	}
	slices.Sort(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func prepareIncrementalStep(app *App, random *rand.Rand, snapshot model.Snapshot, ids []string, now time.Time, step int) (string, []string) {
	app.Selected.Clear()
	visible := make([]string, len(app.Visible))
	for i, asset := range app.Visible {
		visible[i] = asset.ID
	}
	for range min(13, len(visible)) {
		app.Selected.Add(visible[random.IntN(len(visible))])
	}
	a, b := -1, -1
	if len(app.Visible) > 0 {
		a, b = random.IntN(len(app.Visible)), random.IntN(len(app.Visible))
	}
	app.selectedIndex, app.anchor = a, b

	names := []string{"Aurora", "Branch", "Cloud", "Delta"}
	switch random.IntN(3) {
	case 0:
		targets := uniqueTargets(ids, random, 12)
		app.patchAssets(targets, func(asset *model.Asset) {
			switch random.IntN(8) {
			case 0:
				asset.Favorite = !asset.Favorite
			case 1:
				asset.Trashed = !asset.Trashed
			case 2:
				asset.Name = names[random.IntN(len(names))]
			case 3:
				asset.Size = int64(random.IntN(9) * 1000)
			case 4:
				asset.CreatedAt = now.Add(-time.Duration(random.IntN(45)*24) * time.Hour)
			case 5:
				asset.FolderIDs = randomFolderMembership(snapshot.Folders, random)
			case 6:
				asset.TagIDs = randomTagMembership(snapshot.Tags, random)
			default:
				asset.Width, asset.Height, asset.Rotation = 80+40*random.IntN(5), 60+30*random.IntN(5), 90*random.IntN(4)
			}
		})
		return "patch", ids
	case 1:
		partial := model.Snapshot{Folders: snapshot.Folders, Tags: snapshot.Tags}
		merged := uniqueTargets(ids, random, 16)
		newCount := random.IntN(min(6, len(merged)))
		newIDs := make([]string, 0, newCount)
		for _, id := range merged[newCount:] {
			asset := app.assetByID[id]
			switch random.IntN(7) {
			case 0:
				asset.Favorite = !asset.Favorite
			case 1:
				asset.Trashed = !asset.Trashed
			case 2:
				asset.Name = names[random.IntN(len(names))]
			case 3:
				asset.Size = int64(random.IntN(9) * 1000)
			case 4:
				asset.CreatedAt = now.Add(-time.Duration(random.IntN(45)*24) * time.Hour)
			case 5:
				asset.FolderIDs = randomFolderMembership(snapshot.Folders, random)
			default:
				asset.TagIDs = randomTagMembership(snapshot.Tags, random)
			}
			partial.Assets = append(partial.Assets, asset)
		}
		for range newCount {
			id := nextAssetID(slices.Concat(ids, newIDs))
			partial.Assets = append(partial.Assets, model.Asset{
				ID: id, Name: names[random.IntN(len(names))], Extension: []string{"jpg", "png", "mov"}[random.IntN(3)],
				Size: int64(random.IntN(9) * 1000), Width: 100 + 40*random.IntN(4), Height: 80 + 30*random.IntN(4),
				CreatedAt: now.Add(-time.Duration(random.IntN(45)*24) * time.Hour), Favorite: random.IntN(3) == 0,
				Kind:      []string{model.KindImage, model.KindVideo}[random.IntN(2)],
				FolderIDs: randomFolderMembership(snapshot.Folders, random), TagIDs: randomTagMembership(snapshot.Tags, random),
			})
			newIDs = append(newIDs, id)
		}
		if step%7 == 6 {
			partial.Tags = slices.Clone(snapshot.Tags)
			partial.Tags[step%len(partial.Tags)].Name = fmt.Sprintf("Renamed %02d", step)
		}
		app.mergeAssets(partial)
		for _, asset := range partial.Assets {
			if !slices.Contains(ids, asset.ID) {
				ids = append(ids, asset.ID)
			}
		}
		return "merge", ids
	default:
		at := now.Add(time.Duration(step+1)*time.Hour + time.Duration(random.IntN(60))*time.Minute)
		app.applyAssetsUsed(uniqueTargets(ids, random, 12), at)
		return "use", ids
	}
}

func uniqueTargets(ids []string, random *rand.Rand, limit int) []string {
	out := slices.Clone(ids)
	random.Shuffle(len(out), func(left, right int) { out[left], out[right] = out[right], out[left] })
	return out[:min(1+random.IntN(limit), len(out))]
}

func nextAssetID(ids []string) string {
	next := 0
	for _, id := range ids {
		var number int
		if _, err := fmt.Sscanf(id, "asset-%d", &number); err == nil {
			next = max(next, number+1)
		}
	}
	return fmt.Sprintf("asset-%03d", next)
}

func incrementalState(app *App) (visible, selected []string) {
	visible = make([]string, len(app.Visible))
	for i, asset := range app.Visible {
		visible[i] = asset.ID
	}
	for id := range app.Selected.All() {
		selected = append(selected, id)
	}
	slices.Sort(selected)
	return visible, selected
}

func scopeLabel(index int) string {
	return []string{"all", "favorites", "folder", "tag", "recent", "trash"}[index]
}

func directionLabel(descending bool) string {
	if descending {
		return "descending"
	}
	return "ascending"
}
