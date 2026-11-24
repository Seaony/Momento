package ui

import (
	"fmt"

	"github.com/Seaony/Momento/app/internal/catalog"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

// 文件夹页顶部的子文件夹卡片：封面、名称、数量，点一下进去。
const (
	subfolderCardWidth  float32 = 184
	subfolderCardHeight float32 = 56
	subfolderHeader     float32 = 24
)

type subfolderCard struct {
	Folder model.Folder
	Count  int
	Cover  model.Asset
}

type subfolderCache struct {
	key   string
	cards []subfolderCard
}

// subfolderCards 是当前文件夹的直接子文件夹；数量和点进去后看到的一样（跟着「显示子文件夹内容」走），封面是里面最新添加的一张。
func (a *App) subfolderCards() []subfolderCard {
	if a.Index == nil || a.TagPage || a.Query.Scope != "folder" || a.Query.Search != "" {
		return nil
	}
	key := fmt.Sprintf("%d|%s|%v", a.indexVersion, a.Query.FolderID, a.Preferences.SubfolderContents)
	if a.subfolders.key == key {
		return a.subfolders.cards
	}
	children := sidebarFolderChildren(a.Snapshot.Folders)[a.Query.FolderID]
	cards := make([]subfolderCard, len(children))
	owner := map[string][]int{} // 文件夹 ID → 算进哪几张卡片
	for i, child := range children {
		cards[i].Folder = child
		members := map[string]bool{child.ID: true}
		if a.Preferences.SubfolderContents {
			members = catalog.FolderTree(a.Snapshot.Folders, child.ID)
		}
		for id := range members {
			owner[id] = append(owner[id], i)
		}
	}
	for _, asset := range a.Snapshot.Assets {
		if asset.Trashed {
			continue
		}
		counted := map[int]bool{}
		for _, folderID := range asset.FolderIDs {
			for _, i := range owner[folderID] {
				if counted[i] {
					continue
				}
				counted[i] = true
				cards[i].Count++
				if cards[i].Cover.ID == "" || asset.CreatedAt.After(cards[i].Cover.CreatedAt) {
					cards[i].Cover = asset
				}
			}
		}
	}
	a.subfolders = subfolderCache{key: key, cards: cards}
	return cards
}

// subfolderStripHeight 是卡片区连同下面间距的高度；卡片按宽度自动换行，位置自己算，素材的坐标才能跟着往下挪同样的距离。
func subfolderStripHeight(count int, width float32) float32 {
	if count == 0 {
		return 0
	}
	perRow := max(1, int((width+space8)/(subfolderCardWidth+space8)))
	rows := (count + perRow - 1) / perRow
	return subfolderHeader + space8 + float32(rows)*subfolderCardHeight + float32(rows-1)*space8 + space16
}

// subfolderStrip 画卡片区，返回点开的子文件夹和是否切换了「显示子文件夹内容」。
// 这两个动作会换掉素材列表，调用方要等这一帧的卡片都排完再执行，不然同一帧里的下标会对不上。
func (a *App) subfolderStrip(c *ui.Context, cards []subfolderCard, width float32) (open string, toggled bool) {
	p := paletteFor(c.Theme())
	perRow := max(1, int((width+space8)/(subfolderCardWidth+space8)))
	height := subfolderStripHeight(len(cards), width) - space16
	ui.Box(c.Key("subfolders")).FillWidth().Height(height).Shrink(0).Margin(0, 0, space16, 0).Children(func() {
		ui.Row(c.Key("subfolders-header")).Absolute().Left(0).Right(0).Top(0).Height(subfolderHeader).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.T("subfolders")).FontSize(fontGroup).FontWeight(600).TextColor(p.TextSecondary).SingleLine()
			ui.Spacer(c)
			toggled = settingsCheckbox(c.Key("subfolder-contents"), &a.Preferences.SubfolderContents, a.T("show_subfolder_contents")).Changed()
		})
		for i, card := range cards {
			x := float32(i%perRow) * (subfolderCardWidth + space8)
			y := subfolderHeader + space8 + float32(i/perRow)*(subfolderCardHeight+space8)
			if a.subfolderCardView(c.Key(card.Folder.ID), card).Absolute().Left(x).Top(y).Clicked() {
				open = card.Folder.ID
			}
		}
	})
	return open, toggled
}

func (a *App) subfolderCardView(c *ui.Context, card subfolderCard) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(card.Folder.Name).Size(subfolderCardWidth, subfolderCardHeight).Padding(space8).Gap(space8).AlignItems(ui.Center).
		Radius(radiusCard).Border(1, p.BorderSubtle).Cursor(ui.CursorPointer)
	// 和侧栏的文件夹行同一个右键菜单。
	button.ContextMenu(func(m *ui.Menu) { a.folderMenu(m, card.Folder) })
	background := p.Subtle
	if button.Hovered() {
		background = p.Hover
	}
	animateColors(button, background, p.Text)
	button.Children(func() {
		ui.Box(c).Size(40, 40).Shrink(0).Radius(radiusSmall).Background(p.Raised).Clip().Center().Children(func() {
			if card.Cover.ID != "" {
				if bitmap := a.Bitmap(card.Cover); bitmap != nil {
					ui.Image(c, bitmap).Fill().Fit(ui.Cover)
					return
				}
			}
			folderIcon(c, card.Folder.Icon).TextColor(p.TextTertiary)
		})
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, card.Folder.Name).FontSize(fontBody).FontWeight(500).TextColor(p.Text).SingleLine()
			ui.Text(c, sidebarCount(card.Count)).FontSize(fontSecondary).FontFeatures("tnum").TextColor(p.TextSecondary).SingleLine()
		})
	})
	return button
}
