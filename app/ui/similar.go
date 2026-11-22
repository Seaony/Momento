package ui

import (
	"context"
	"fmt"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 对话框一次最多列这么多组；「全部合并」也只合并列出来的，没看到的不动。
const similarMaxGroups = 100

const similarTile float32 = 96

type similarGroup struct {
	IDs  []string // 按 BestFirst 排好，第一张是建议保留的
	Keep string
}

type similarState struct {
	Open   bool
	Groups []similarGroup
}

// FindSimilar 在后台给图库里不在回收站的静态图算指纹、分组（哪些参与见 Fingerprints），找到了就打开对话框。
func (a *App) FindSimilar() {
	if a.Library == nil || a.Busy {
		return
	}
	service := a.Assets
	items := make([]model.Asset, 0, len(a.Snapshot.Assets))
	for _, asset := range a.Snapshot.Assets {
		if !asset.Trashed {
			items = append(items, asset)
		}
	}
	gen := a.generation
	a.RunPatch("find_similar", nil, func(ctx context.Context) error {
		prints, err := service.Fingerprints(ctx, items, func(done, total int) {
			a.Window.Update(func() {
				if a.generation == gen {
					a.Progress = &model.ImportProgress{Stage: "similar", Processed: done, Total: total}
				}
			})
		})
		var groups [][]model.Asset
		if err == nil {
			groups = assets.SimilarGroups(items, prints)
		}
		a.Window.Update(func() {
			if a.generation != gen {
				return
			}
			a.Progress = nil
			if err != nil {
				return
			}
			if len(groups) == 0 {
				a.Notify(a.T("similar_none"))
				return
			}
			state := similarState{Open: true, Groups: make([]similarGroup, len(groups))}
			for i, group := range groups {
				for _, asset := range group {
					state.Groups[i].IDs = append(state.Groups[i].IDs, asset.ID)
				}
				state.Groups[i].Keep = group[0].ID
			}
			a.Similar = state
		})
		return err
	}, nil)
}

type similarResolved struct {
	index  int // 在 state.Groups 里的位置
	assets []model.Asset
	keep   int
}

// similarGroups 把记下的 ID 换成现在的素材：已经进回收站或删掉的去掉，剩不到两张的组不再列。
func (a *App) similarGroups() []similarResolved {
	var out []similarResolved
	for i, group := range a.Similar.Groups {
		resolved := similarResolved{index: i, keep: -1}
		for _, id := range group.IDs {
			if asset, ok := a.assetByID[id]; ok && !asset.Trashed {
				if id == group.Keep {
					resolved.keep = len(resolved.assets)
				}
				resolved.assets = append(resolved.assets, asset)
			}
		}
		if len(resolved.assets) > 1 {
			// 选中保留的那张已经不在了，退回建议的那张。
			resolved.keep = max(resolved.keep, 0)
			out = append(out, resolved)
		}
	}
	return out
}

func (a *App) similarDialog(c *ui.Context) {
	state := &a.Similar
	if !state.Open {
		return
	}
	groups := a.similarGroups()
	if len(groups) == 0 {
		state.Open = false
		return
	}
	shown := groups[:min(len(groups), similarMaxGroups)]
	p := paletteFor(c.Theme())
	// 点了合并先记下，画完这一帧再执行：合并会删掉组，边画边删会让后面的组错位。
	var merge []similarResolved
	dialogPanel(c.Key("similar"), &state.Open, panelWide, "similar", a.T("similar_title", len(groups)), func() {
		ui.Text(c, a.T("similar_hint")).FontSize(fontSecondary).TextColor(p.TextSecondary)
		ui.Scroll(c.Key("similar-groups")).Scrollbars(false).MaxHeight(440).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(space8).Padding(0, 0, space4, 0).Children(func() {
				for _, group := range shown {
					if a.similarGroupRow(c.Key(a.Similar.Groups[group.index].IDs[0]), group) {
						merge = []similarResolved{group}
					}
				}
				if more := len(groups) - len(shown); more > 0 {
					ui.Text(c, a.T("similar_more", more)).FontSize(fontSecondary).TextColor(p.TextSecondary).Padding(space4, 0)
				}
			})
		})
	}, func() {
		if dialogButton(c, a.T("close"), false).Clicked() {
			state.Open = false
		}
		if dialogPrimaryButton(c, a.T("similar_merge_all", len(shown)), false).Clicked() {
			merge = shown
		}
	})
	a.mergeSimilar(merge)
}

// 一组：缩略图排成一行（多了换行），保留的那张描品牌色边并标「保留」，点别的换成保留它；右边是合并这一组，点了返回 true。
func (a *App) similarGroupRow(c *ui.Context, group similarResolved) (merge bool) {
	p := paletteFor(c.Theme())
	ui.Row(c).FillWidth().Padding(space12).Gap(space12).AlignItems(ui.Center).Radius(radiusControl).Background(p.Subtle).Border(1, p.BorderSubtle).Children(func() {
		ui.Row(c.Key("tiles")).Grow(1).Basis(0).MinWidth(0).Wrap().Gap(space8).Children(func() {
			for i, asset := range group.assets {
				if a.similarTile(c.Key(asset.ID), asset, i == group.keep).Clicked() {
					a.Similar.Groups[group.index].Keep = asset.ID
				}
			}
		})
		// 别的写入没完成时也能点：写入会排队，这一组也马上从对话框拿掉，不会重复提交。
		merge = settingsButton(c.Key("merge"), a.T("similar_merge")).Clicked()
	})
	return merge
}

func (a *App) similarTile(c *ui.Context, asset model.Asset, keep bool) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Width(similarTile).Column().Gap(space4).Cursor(ui.CursorPointer).Label(asset.Name).Tooltip(asset.Name).AccessSelected(keep)
	button.Children(func() {
		border := p.BorderSubtle
		width := float32(1)
		if keep {
			border, width = p.Accent, 2
		} else if button.Hovered() {
			border = p.Border
		}
		ui.Box(c.Key("thumb")).Size(similarTile, similarTile).Radius(radiusCard).Background(p.Raised).Border(width, border).Clip().Children(func() {
			if bitmap := a.Bitmap(asset); bitmap != nil {
				ui.Image(c, bitmap).Fill().Fit(ui.Cover)
			}
			if keep {
				ui.Text(c, a.T("similar_keep")).Absolute().Left(space4).Top(space4).Padding(1, space4).Radius(radiusSmall).
					Background(p.Accent).TextColor(p.OnAccent).FontSize(fontCaption).FontWeight(600)
			}
		})
		ui.Text(c, fmt.Sprintf("%d×%d", asset.Width, asset.Height)).FontFeatures("tnum").FontSize(fontSecondary).TextColor(p.TextSecondary).SingleLine()
		ui.Text(c, fileSize(asset.Size)).FontFeatures("tnum").FontSize(fontSecondary).TextColor(p.TextTertiary).SingleLine()
	})
	return button
}

// mergeSimilar 把这些组各自并进保留的那张，其余进回收站；一个事务写库，成功后直接改内存。
func (a *App) mergeSimilar(groups []similarResolved) {
	if a.Library == nil || len(groups) == 0 {
		return
	}
	merges := make([]library.MergeGroup, 0, len(groups))
	merged := map[string]model.Asset{}
	var ids []string
	for _, group := range groups {
		keep := group.assets[group.keep]
		others := make([]model.Asset, 0, len(group.assets)-1)
		for i, asset := range group.assets {
			if i != group.keep {
				others = append(others, asset)
				ids = append(ids, asset.ID)
			}
		}
		merges = append(merges, library.MergeGroup{Keep: keep, Others: others})
		merged[keep.ID] = library.MergedAsset(keep, others)
		ids = append(ids, keep.ID)
	}
	// 先从对话框里拿掉，免得写库期间再点一次；写失败会弹错误，重新查找即可。
	done := map[int]bool{}
	for _, group := range groups {
		done[group.index] = true
	}
	remaining := a.Similar.Groups[:0:0]
	for i, group := range a.Similar.Groups {
		if !done[i] {
			remaining = append(remaining, group)
		}
	}
	a.Similar.Groups = remaining
	lib := a.Library
	a.RunPatch("merge_similar", ids, func(ctx context.Context) error { return lib.MergeAssets(ctx, merges...) }, func(asset *model.Asset) {
		if m, ok := merged[asset.ID]; ok {
			asset.Note, asset.SourceURL, asset.Favorite = m.Note, m.SourceURL, m.Favorite
			asset.TagIDs, asset.FolderIDs = m.TagIDs, m.FolderIDs
			return
		}
		asset.Trashed = true
	})
}
