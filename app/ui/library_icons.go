package ui

import (
	"log/slog"
	"slices"

	"github.com/Seaony/Momento/app/resources"
	"github.com/egoist/mygo/ui"
)

// 图库可选的图标，按图标网格里的顺序排；编号存进图库的 manifest。空编号用应用图标。
// 图标后来全换成了动物（mountain 现在是柴犬……），编号已经存在用户的图库里，所以不改名，只换图和显示名。
var libraryIconIDs = []string{
	"mountain", "wave", "tree", "flower", "cloud", "moon",
	"city", "landmark", "travel", "portrait", "culture", "food",
	"pet", "art", "design", "camera", "fashion", "interior",
	"car", "sport", "music", "game", "idea", "heart",
}

const (
	libraryIconColumns = 5  // 网格每行几个：应用图标加 24 个，正好 5 × 5
	libraryIconCell    = 44 // 网格里一格的点击区，也是选中细框的外径
	libraryIconTile    = 32
	libraryIconGap     = 4
	libraryFieldHeight = 36 // 名称输入框和左边图标按钮的高度
)

var libraryIconBitmaps = map[string]*ui.Bitmap{}

// 按编号取图标；不认识的编号（比如新版加的图标在旧版里打开）按应用图标显示。
func libraryIconBitmap(id string) *ui.Bitmap {
	if !slices.Contains(libraryIconIDs, id) {
		id = ""
	}
	if bitmap, ok := libraryIconBitmaps[id]; ok {
		return bitmap
	}
	name := id
	if name == "" {
		name = "momento"
	}
	data, err := resources.LibraryIcons.ReadFile("library-icons/" + name + ".png")
	var bitmap *ui.Bitmap
	if err == nil {
		bitmap, err = ui.DecodeBitmap(data)
	}
	if err != nil {
		slog.Error("读取图库图标失败", "icon", name, "error", err)
		bitmap = appLogo
	}
	libraryIconBitmaps[id] = bitmap
	return bitmap
}

// 图库图标是满画幅方块，按边长切同比例的圆角。
func libraryIcon(c *ui.Context, id string, size float32) ui.Element {
	return ui.Image(c, libraryIconBitmap(id)).Size(size, size).Radius(size * .225).Shrink(0)
}

func (a *App) libraryIconName(id string) string {
	if id == "" {
		return a.T("library_icon_default")
	}
	return a.T("library_icon_" + id)
}

// 图库名称输入框：左边是图标按钮，点开在下面弹出图标网格；返回里面的输入框，调用方看它的 Submitted。
func (a *App) libraryNameField(c *ui.Context, name, icon *string, pickerOpen *bool, disabled bool) ui.Element {
	return a.iconNameField(c.Key("library-name-field"), name, pickerOpen, disabled, a.T("library_icon"), a.T("library_name"),
		func() { libraryIcon(c, *icon, space20) }, func() { a.libraryIconGrid(c, icon, pickerOpen) })
}

// 带图标按钮的名称输入框（图库、文件夹共用）：左边图标按钮点开图标网格，竖线右边是输入框。
func (a *App) iconNameField(c *ui.Context, name *string, pickerOpen *bool, disabled bool, iconLabel, nameLabel string, glyph, grid func()) ui.Element {
	p := paletteFor(c.Theme())
	var input ui.Element
	field := ui.Row(c).FillWidth().Height(libraryFieldHeight).Gap(0).AlignItems(ui.Stretch).Clip()
	field.Children(func() {
		trigger := ui.ButtonBase(c.Key("icon-button")).Width(libraryFieldHeight).Shrink(0).Padding(0).Justify(ui.Center).AlignItems(ui.Center).
			Cursor(ui.CursorPointer).Label(iconLabel).Tooltip(iconLabel).Disabled(disabled).Expanded(*pickerOpen)
		background := ui.Transparent
		if trigger.Pressed() {
			background = p.Pressed
		} else if trigger.Hovered() || *pickerOpen {
			background = p.Hover
		}
		trigger.Background(trigger.AnimateColor("icon-surface", background, motionColor))
		trigger.Children(glyph)
		if trigger.Clicked() {
			*pickerOpen = !*pickerOpen
		}
		if disabled {
			*pickerOpen = false
		}
		floatingPopover(c.Key("icon-picker"), trigger, pickerOpen, grid).Padding(space8)
		ui.Box(c).Width(1).Shrink(0).Background(p.Border)
		input = ui.TextInputBase(c.Key("input"), name).Label(nameLabel).Placeholder(nameLabel).PlaceholderColor(placeholderColor(c)).
			Grow(1).MinWidth(0).Padding(0, space12).FontSize(fontBody).FocusRing(false).Disabled(disabled)
	})
	fieldStyle(c, field)
	return input
}

// 图标网格：第一格是应用图标（默认），后面是内置图标；选中的外面一圈正文色细框，点了就收起。
func (a *App) libraryIconGrid(c *ui.Context, icon *string, open *bool) {
	p := paletteFor(c.Theme())
	ids := append([]string{""}, libraryIconIDs...)
	ui.Column(c.Key("library-icon-grid")).Gap(libraryIconGap).Children(func() {
		for start := 0; start < len(ids); start += libraryIconColumns {
			ui.Row(c).Gap(libraryIconGap).Children(func() {
				for _, id := range ids[start:min(start+libraryIconColumns, len(ids))] {
					name := a.libraryIconName(id)
					chosen := *icon == id
					cell := ui.ButtonBase(c.Key("icon-"+id)).Size(libraryIconCell, libraryIconCell).Shrink(0).Padding(0).
						Radius(libraryIconTile*.225 + (libraryIconCell-libraryIconTile)/2).Cursor(ui.CursorPointer).Justify(ui.Center).AlignItems(ui.Center).
						Label(name).Tooltip(name).Role(ui.RoleRadio).Checked(chosen)
					ring := ui.Transparent
					if chosen {
						ring = p.Text
					} else if cell.Hovered() {
						ring = p.Border
					}
					cell.Border(2, cell.AnimateColor("ring", ring, motionColor))
					cell.Children(func() { libraryIcon(c, id, libraryIconTile).PassThrough() })
					if cell.Clicked() {
						*icon, *open = id, false
					}
				}
			})
		}
	})
}
