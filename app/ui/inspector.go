package ui

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

type inspectorEdit struct {
	ID, Name, Note, Source  string
	Rename, Details, Failed bool
}

type inspectorState struct {
	LibraryID                                   string
	AssetID, Title, SourceTitle                 string
	Note, SourceNote, SourceURL, SavedSourceURL string
	TagsOpen, FoldersOpen                       bool
	Pending                                     []inspectorEdit
	TagSearch, FolderSearch                     string
	// 正在原地编辑的字段：title、note、source；平时只显示文字，点了才变输入框。
	Editing string
	EditKey ui.Key
	// 文件夹选择器里展开的父文件夹。
	FolderExpanded map[string]bool
	// 最近一次写入失败的原因，放弃修改的确认框里给用户看。
	SaveError string
	// EXIF 不随图库一起读进内存，选中时才从库里取；只留当前这一张。
	EXIFFor, EXIFLoading string
	EXIF                 map[string]string
	// 多选时一起改备注或链接（Editing 是 multi-note、multi-source）：开始编辑时记下是哪些素材、原来的值一不一样，
	// 编辑中途换了选择也只改开始时那批。
	MultiDraft    string
	MultiIDs      []string
	MultiOriginal string
	MultiMixed    bool
	// MultiSaving 为真时一起改的那批还没写完，退出和换图库要等它。
	MultiSaving bool
}

// 照旧版 macOS 检查器的规格。
const (
	inspectorContentWidth   = 280 - 2*space16
	inspectorPreviewMax     = 360
	inspectorLabelWidth     = 74
	inspectorEditRadius     = 7
	inspectorChipHeight     = 26
	inspectorChipRadius     = 9
	inspectorSwatch         = 16
	inspectorSwatchHit      = inspectorSwatch + 4
	inspectorMultiCardW     = 156
	inspectorMultiCardH     = 112
	inspectorPickerWidth    = 240
	inspectorPickerMaxList  = 220
	inspectorFolderRowH     = 26
	inspectorFolderIndent   = 22
	inspectorSectionTitleSz = 10
	// 可编辑行的内边距 6 加边框 1：标题、备注往外伸这么多，文字和小标题对齐，悬停底色伸到两边留白里。
	inspectorEditBleed      = 7
	inspectorEditLineHeight = 1.4 // 可编辑文字显示态和输入框用同一个行高，点进编辑时高度不变，下面的内容不跳
	inspectorTitleWeight    = 600 // 图片标题比正文大一号、加粗，一眼能和下面的信息分开
)

func (a *App) inspector(c *ui.Context) {
	if a.Library != nil && a.Inspector.LibraryID != a.Library.Manifest.LibraryID {
		a.Inspector = inspectorState{LibraryID: a.Library.Manifest.LibraryID}
	}
	selected := a.SelectedAssets()
	// 多选时正在一起改备注或链接，选择变成单张或清空后编辑框不再画出来、收不到失焦：这里替它结束，按开始时那批素材存下。
	if field, ok := strings.CutPrefix(a.Inspector.Editing, "multi-"); ok && len(selected) < 2 {
		a.applyMultiDetail(field)
		a.Inspector.Editing, a.Inspector.EditKey = "", 0
	}
	if len(selected) == 0 {
		a.saveInspectorEdits()
		a.Inspector.AssetID, a.Inspector.Editing = "", ""
		return
	}
	ui.Column(c).Grow(1).FillWidth().MinHeight(0).Children(func() {
		if bar := c.TitleBar(); bar.Right > 0 {
			ui.Box(c.Key("inspector-window-controls")).Height(max(bar.Bottom, bar.Height)).Shrink(0).FillWidth().DragWindow()
		}
		ui.Row(c.Key("inspector-header")).Height(space40).Shrink(0).FillWidth().Padding(space4, space16).Children(func() {
			ui.Spacer(c)
			if inspectorIconButton(c.Key("collapse-inspector"), a.T("close_inspector"), "collapse-inspector").Shrink(0).Clicked() {
				a.CollapseInspector()
			}
		})
		ui.Scroll(c.Key("inspector_content")).Scrollbars(false).Grow(1).Padding(0, space16, space16, space16).Children(func() {
			if len(selected) == 1 {
				asset := selected[0]
				a.syncInspectorAsset(asset)
				fadeMotion(ui.Column(c.Key("inspector-asset-" + asset.ID))).FillWidth().Gap(space12).Children(func() {
					ui.Column(c.Key("inspector-head")).FillWidth().Gap(space8).Children(func() {
						a.inspectorPreview(c, asset)
						a.inspectorColors(c, asset)
						a.inspectorTitle(c, asset)
						a.inspectorRating(c, selected)
					})
					a.inspectorDetails(c, asset)
					a.inspectorTags(c, selected)
					a.inspectorFolders(c, selected)
					a.inspectorNotes(c, asset)
					a.inspectorAnnotations(c, asset)
					a.inspectorEXIF(c, asset)
				})
				return
			}
			a.saveInspectorEdits()
			a.Inspector.AssetID = ""
			if !strings.HasPrefix(a.Inspector.Editing, "multi-") {
				a.Inspector.Editing = ""
			}
			ui.Column(c.Key("inspector-multiple")).FillWidth().Gap(space12).Children(func() {
				a.inspectorStack(c, selected)
				a.inspectorRating(c, selected)
				a.inspectorTags(c, selected)
				a.inspectorFolders(c, selected)
				a.inspectorMultiDetails(c, selected)
			})
		})
	})
}

// 换了素材就把草稿换成新素材的值；还没写进图库的修改从待保存队列里取回。
func (a *App) syncInspectorAsset(asset model.Asset) {
	if a.Inspector.AssetID == asset.ID && a.Inspector.SourceTitle == asset.Name {
		return
	}
	if a.Inspector.AssetID != asset.ID {
		a.saveInspectorEdits()
		a.Inspector.Editing = ""
	}
	a.Inspector.AssetID, a.Inspector.Title, a.Inspector.SourceTitle = asset.ID, asset.Name, asset.Name
	a.Inspector.Note, a.Inspector.SourceNote = asset.Note, asset.Note
	a.Inspector.SourceURL, a.Inspector.SavedSourceURL = asset.SourceURL, asset.SourceURL
	for _, edit := range a.Inspector.Pending {
		if edit.ID == asset.ID {
			if edit.Rename {
				a.Inspector.Title = edit.Name
			}
			if edit.Details {
				a.Inspector.Note = edit.Note
				a.Inspector.SourceURL = edit.Source
			}
		}
	}
}

// 图片按宽度铺满、保持比例，太高的限高；点一下打开预览。
func (a *App) inspectorPreview(c *ui.Context, asset model.Asset) {
	p := paletteFor(c.Theme())
	preview := ui.Column(c.Key("inspector_preview")).FillWidth().Label(a.T("image_preview")).Cursor(ui.CursorPointer).Center()
	preview.Children(func() {
		if bitmap := a.Bitmap(asset); bitmap != nil {
			w, h := bitmap.Size()
			scale := min(float32(inspectorContentWidth)/float32(w), float32(inspectorPreviewMax)/float32(h))
			ui.Image(c, bitmap).Size(float32(w)*scale, float32(h)*scale).Radius(radiusCard).Fit(ui.Contain).Label(a.T("image_preview"))
			return
		}
		ui.Column(c.Key("preview-placeholder")).FillWidth().Height(inspectorContentWidth / 1.25).Radius(radiusCard).Background(p.Subtle).Center().Children(func() {
			if a.Library != nil && a.Thumbnails != nil {
				if err := a.Thumbnails.Error(a.Library.Root, asset); err != nil {
					ui.Text(c, a.T("preview_failed")).FontSize(fontSecondary).TextColor(p.Danger).Tooltip(a.Texts.Error(err))
					return
				}
			}
			ui.Text(c, a.T("loading_preview")).FontSize(fontSecondary).TextColor(p.TextSecondary)
		})
	})
	if preview.Clicked() {
		a.ShowPreview(asset.ID, false)
	}
}

// 色板：浅底圆角条里一排 16 的色块，悬停时上方浮出色值和占比，点击复制色值。
func (a *App) inspectorColors(c *ui.Context, asset model.Asset) {
	if len(asset.Colors) == 0 {
		return
	}
	p := paletteFor(c.Theme())
	colors := asset.Colors[:min(len(asset.Colors), (inspectorContentWidth-10+1)/(inspectorSwatchHit+1))]
	ui.Row(c.Key("inspector-colors")).FillWidth().Height(28).Center().Children(func() {
		strip := ui.Row(c.Key("color-strip")).Padding(3, 5).Gap(1).Radius(10).Background(p.Subtle).Border(1, p.BorderSubtle)
		strip.Children(func() {
			hovered := -1
			for i, color := range colors {
				hex := strings.ToUpper(color.Hex)
				tip := inspectorColorTip(color)
				swatch := ui.ButtonBase(c.Key("swatch_"+hex)).Label(a.T("copy_color", hex)).Description(tip).
					Size(inspectorSwatchHit, inspectorSwatchHit).Padding(2).Radius(inspectorEditRadius).Cursor(ui.CursorPointer)
				background := ui.Transparent
				if swatch.Hovered() {
					background, hovered = p.Hover, i
				}
				swatch.Background(swatch.AnimateColor("hover", background, motionColor))
				swatch.Children(func() {
					ui.Box(c).Size(inspectorSwatch, inspectorSwatch).Radius(5).Background(ui.Hex(hex)).Border(1, p.Text.Alpha(.08))
				})
				if swatch.Clicked() {
					c.WriteClipboardThen(hex, func(err error) {
						if err != nil {
							a.ShowError(err)
						} else {
							a.Notify(a.T("copied_color", hex))
						}
					})
				}
			}
			if hovered < 0 {
				return
			}
			// 提示跟着悬停的色块滑动，离开色板时淡出。
			text := inspectorColorTip(colors[hovered])
			w, h := c.MeasureText(0, ui.Span{Text: text, Font: c.Theme().Font, Size: fontGroup, Weight: 500, Lang: a.Texts.Locale()})
			w, h = w+2*space8, h+2*space4
			center := float32(5 + hovered*(inspectorSwatchHit+1) + inspectorSwatchHit/2)
			bubble := ui.Box(c.Key("color-tip")).Absolute().Left(center-w/2).Top(-h-6).Size(w, h).ZIndex(10).
				Radius(6).Background(c.Theme().Inverse).Center().IgnoreInput(true).AccessHidden(true)
			bubble.Transition(ui.ElementTransition{Animation: &motionSpring, Position: true, Colors: true,
				Enter: &ui.Motion{Scale: .96, Y: 2}, Exit: &ui.Motion{Scale: .96, Y: 2}})
			bubble.Children(func() {
				ui.Text(c, text).FontSize(fontGroup).FontWeight(500).FontFeatures("tnum").TextColor(c.Theme().InverseText).SingleLine()
			})
		})
	})
}

func inspectorColorTip(color model.Color) string {
	hex := strings.ToUpper(color.Hex)
	if color.Proportion <= 0 {
		return hex
	}
	return fmt.Sprintf("%s (%.1f%%)", hex, color.Proportion*100)
}

// 标题平时只是文字，悬停才出浅底和细边，点一下才变成输入框；回车或失焦保存，Esc 放弃。
func (a *App) inspectorTitle(c *ui.Context, asset model.Asset) {
	if a.Inspector.Editing == "title" {
		a.inspectorEditor(c, "asset_title_"+asset.ID, &a.Inspector.Title, a.T("edit_title"), "", fontLead, 6, true, func(cancel bool) {
			if cancel || strings.TrimSpace(a.Inspector.Title) == "" {
				a.Inspector.Title = a.Inspector.SourceTitle
			}
		}).FontWeight(inspectorTitleWeight).Margin(0, -inspectorEditBleed)
		return
	}
	a.inspectorEditable(c, "title-display", a.T("edit_title"), func(p *designPalette) {
		ui.Text(c, a.Inspector.Title).FontSize(fontLead).FontWeight(inspectorTitleWeight).TextColor(p.Text).FillWidth()
	}, func() { a.Inspector.Editing = "title" }).Margin(0, -inspectorEditBleed)
}

// 星级评分：多选时星级都一样才亮出来，不一样显示空星，点一下给全部打同样的星。
func (a *App) inspectorRating(c *ui.Context, selected []model.Asset) {
	value := selected[0].Rating
	for _, asset := range selected[1:] {
		if asset.Rating != value {
			value = 0
			break
		}
	}
	if ratingStars(c.Key("rating"), &value, a.T("rating")).Changed() {
		ids := make([]string, len(selected))
		for i, asset := range selected {
			ids[i] = asset.ID
		}
		a.SetRating(ids, value)
	}
}

// ratingStars 是 mygo 的评分星星，实心星换成品牌色（组件默认取主题的警示色，浅色下是棕色）。
func ratingStars(c *ui.Context, value *int, label string) ui.Element {
	theme := *c.Theme()
	theme.Warning = paletteFor(c.Theme()).Accent
	var stars ui.Element
	ui.Themed(c, &theme).Background(ui.Transparent).Children(func() {
		stars = ui.Rating(c, value, 5).Label(label)
	})
	return stars
}

// 可原地编辑的文字：悬停铺浅底和细边，点一下进入编辑。
func (a *App) inspectorEditable(c *ui.Context, key, label string, content func(p *designPalette), edit func()) ui.Element {
	p := paletteFor(c.Theme())
	// 不设 FillWidth，靠父列拉伸占满：这样调用方加负外边距时会两边一起伸出，FillWidth 只会整体左移。
	button := ui.ButtonBase(c.Key(key)).Label(label).Padding(space4, 6).LineHeight(inspectorEditLineHeight).Radius(inspectorEditRadius).Cursor(ui.CursorPointer).AlignItems(ui.Start)
	background, border := ui.Transparent, ui.Transparent
	if button.Hovered() || button.Focused() {
		background, border = p.Text.Alpha(themeValue(c, .045, .06)), p.BorderSubtle
	}
	button.Background(button.AnimateColor("edit-bg", background, motionColor)).Border(1, button.AnimateColor("edit-border", border, motionColor))
	button.Children(func() { content(p) })
	if button.Clicked() {
		edit()
	}
	return button
}

// 编辑中的输入框：和显示态同样的内边距和字号，品牌色细边；按行数长高（去掉 TextArea 自带的 80 最小高度）。
// single 为真时回车结束编辑（标题、链接），否则回车换行（备注）。
func (a *App) inspectorEditor(c *ui.Context, key string, value *string, label, placeholder string, size float32, lines int, single bool, done func(cancel bool)) ui.Element {
	p := paletteFor(c.Theme())
	input := ui.TextArea(c.Key(key), value).Lines(1, lines).Label(label).Placeholder(placeholder).PlaceholderColor(placeholderColor(c)).
		MinHeight(0).Padding(space4, 6).FontSize(size).LineHeight(inspectorEditLineHeight).Radius(inspectorEditRadius).Background(p.Raised).Border(1, p.Accent.Alpha(.65)).
		FocusRing(false).Scrollbars(false).AutoFocus()
	// 回车、Esc 在输入框处理之前拦下，记下来留到下一帧处理：单行字段回车不插换行，光标停在哪都不改内容。
	input.OnTextKey(func(k ui.TextKey) bool {
		switch {
		case k.Key == ui.KeyEscape && k.Mods == 0:
			a.Inspector.EditKey = ui.KeyEscape
		case k.Key == ui.KeyEnter && k.Mods == 0 && single:
			a.Inspector.EditKey = ui.KeyEnter
		default:
			return false
		}
		return true
	})
	finish := func(cancel bool) {
		done(cancel)
		a.Inspector.Editing, a.Inspector.EditKey = "", 0
		a.saveInspectorEdits()
	}
	// 粘贴进单行字段的换行换成空格。
	if single && strings.ContainsAny(*value, "\r\n") {
		*value = strings.Join(strings.Fields(strings.NewReplacer("\r", " ", "\n", " ").Replace(*value)), " ")
	}
	switch {
	case a.Inspector.EditKey == ui.KeyEscape:
		finish(true)
	case a.Inspector.EditKey == ui.KeyEnter, input.FocusLeft():
		finish(false)
	}
	return input
}

// 浅色、深色各取一个值（多为叠在底色上的正文色比例）。
func themeValue(c *ui.Context, light, dark float32) float32 {
	if c.Theme().Dark {
		return dark
	}
	return light
}

// 区块：一条浅分隔线、大写小标题，再是内容，间距都是 12。
func inspectorSection(c *ui.Context, key, title string, body func()) {
	p := paletteFor(c.Theme())
	ui.Column(c.Key(key)).FillWidth().Gap(space12).Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(p.hairline())
		// 备注不要小标题，占位文字「添加备注」已经说明了用途。
		if title != "" {
			ui.Text(c, strings.ToUpper(title)).FontSize(inspectorSectionTitleSz).FontWeight(600).TextColor(inspectorSectionInk(c, p))
		}
		body()
	})
}

// 一行信息：左边 74 宽的次要色标签，右边左对齐的值，可选中复制。
// inspectorInfo 画一行「标签：值」，返回值那一格，调用方可以再加样式。
func inspectorInfo(c *ui.Context, label, value string) (text ui.Element) {
	p := paletteFor(c.Theme())
	ui.Row(c).FillWidth().AlignItems(ui.Start).Gap(10).Children(func() {
		ui.Text(c, label).FontSize(fontSecondary).TextColor(p.TextTertiary).Width(inspectorLabelWidth).Shrink(0)
		text = ui.Text(c, value).FontSize(fontSecondary).FontFeatures("tnum").TextColor(p.Text).Grow(1).MinWidth(0).Selectable()
	})
	return text
}

func (a *App) inspectorDetails(c *ui.Context, asset model.Asset) {
	inspectorSection(c, "inspector-details", a.T("details"), func() {
		if asset.Extension != "" {
			inspectorInfo(c, a.T("kind"), strings.ToUpper(asset.Extension))
		}
		if width, height := assetDisplaySize(asset); width > 0 && height > 0 {
			inspectorInfo(c, a.T("dimensions"), fmt.Sprintf("%d × %d", width, height))
		}
		if d := assetDuration(asset); d > 0 {
			inspectorInfo(c, a.T("asset_duration"), videoDuration(d))
		}
		if asset.Size > 0 {
			inspectorInfo(c, a.T("file_size"), fileSize(asset.Size))
		}
		if !asset.CreatedAt.IsZero() {
			inspectorInfo(c, a.T("date_added"), a.Texts.DateTime(asset.CreatedAt.Local()))
		}
		// 文件在磁盘上的路径只占一行，放不下的部分用省略号。
		if a.Library != nil && asset.RelativePath != "" {
			inspectorInfo(c, a.T("file_path"), filepath.Join(a.Library.Root, filepath.FromSlash(asset.RelativePath))).SingleLine()
		}
		// 和旧版一样，没有链接就不显示这一行。
		if strings.TrimSpace(a.Inspector.SourceURL) != "" || a.Inspector.Editing == "source" {
			a.inspectorLink(c, asset)
		}
	})
}

// 链接：有地址时点文字打开，悬停变品牌色加下划线，右边浮出编辑按钮；没有地址时点占位文字直接编辑。
func (a *App) inspectorLink(c *ui.Context, asset model.Asset) {
	p := paletteFor(c.Theme())
	// 链接文字自带 6 的内边距，标签后的间距相应减掉，文字和上面各行的值对齐。
	row := ui.Row(c.Key("inspector-link")).FillWidth().AlignItems(ui.Center).Gap(10 - 7)
	row.Children(func() {
		ui.Text(c, a.T("link")).FontSize(fontSecondary).TextColor(p.TextTertiary).Width(inspectorLabelWidth).Shrink(0)
		if a.Inspector.Editing == "source" {
			ui.Box(c.Key("link-editor")).Grow(1).MinWidth(0).Children(func() {
				a.inspectorEditor(c, "asset-source-"+asset.ID, &a.Inspector.SourceURL, a.T("source_url"), a.T("source_url_placeholder"), fontSecondary, 4, true, func(cancel bool) {
					if cancel {
						a.Inspector.SourceURL = a.Inspector.SavedSourceURL
					}
				})
			})
			return
		}
		url := strings.TrimSpace(a.Inspector.SourceURL)
		link := ui.ButtonBase(c.Key("link-open")).Label(a.T("open_source_url")).Description(url).Grow(1).MinWidth(0).
			Padding(3, 6).Radius(inspectorEditRadius).Cursor(ui.CursorPointer).AlignItems(ui.Center)
		background, border, ink := ui.Transparent, ui.Transparent, p.Text
		if link.Hovered() || link.Focused() {
			background, border, ink = p.Text.Alpha(themeValue(c, .05, .08)), p.BorderSubtle, p.Link
		}
		link.Background(link.AnimateColor("link-bg", background, motionColor)).Border(1, link.AnimateColor("link-border", border, motionColor))
		link.Children(func() {
			text := ui.Text(c, url).FontSize(fontSecondary).TextColor(link.AnimateColor("link-ink", ink, motionColor)).SingleLine().Grow(1).MinWidth(0)
			if link.Hovered() {
				text.Underline()
			}
		})
		if link.Clicked() {
			c.OpenURLThen(url, a.ShowError)
		}
		edit := inspectorIconButton(c.Key("link-edit"), a.T("edit_link"), "edit").Size(space20, space20)
		edit.Opacity(edit.AnimateTo("shown", motionTarget(row.Hovered() || edit.Focused()), motionColor))
		if edit.Clicked() {
			a.Inspector.Editing = "source"
		}
	})
}

func (a *App) inspectorNotes(c *ui.Context, asset model.Asset) {
	inspectorSection(c, "inspector-notes", "", func() {
		if a.Inspector.Editing == "note" {
			a.inspectorEditor(c, "asset-note-"+asset.ID, &a.Inspector.Note, a.T("asset_note"), a.T("asset_note_placeholder"), fontSecondary, 10, false, func(cancel bool) {
				if cancel {
					a.Inspector.Note = a.Inspector.SourceNote
				}
			}).Margin(0, -inspectorEditBleed)
			return
		}
		a.inspectorEditable(c, "note-display", a.T("asset_note"), func(p *designPalette) {
			if strings.TrimSpace(a.Inspector.Note) == "" {
				ui.Text(c, a.T("asset_note_placeholder")).FontSize(fontSecondary).TextColor(placeholderColor(c)).FillWidth()
				return
			}
			ui.Text(c, a.Inspector.Note).FontSize(fontSecondary).TextColor(p.Text).FillWidth()
		}, func() { a.Inspector.Editing = "note" }).Margin(0, -inspectorEditBleed)
	})
}

// 多选：备注和链接一起改。值都一样时显示出来照常改；不一样时显示「多个不同的…」，
// 这时留空结束编辑不算清空，免得一不小心把每张不同的备注全抹掉。
func (a *App) inspectorMultiDetails(c *ui.Context, selected []model.Asset) {
	inspectorSection(c, "inspector-multi-details", "", func() {
		for _, field := range []struct {
			key, label, placeholder, mixed string
			value                          func(model.Asset) string
			lines                          int
		}{
			{"note", a.T("asset_note"), a.T("asset_note_placeholder"), a.T("multiple_notes"), func(x model.Asset) string { return x.Note }, 10},
			{"source", a.T("source_url"), a.T("source_url_placeholder"), a.T("multiple_links"), func(x model.Asset) string { return x.SourceURL }, 4},
		} {
			common, mixed := field.value(selected[0]), false
			for _, asset := range selected[1:] {
				if field.value(asset) != common {
					common, mixed = "", true
					break
				}
			}
			editing := "multi-" + field.key
			if a.Inspector.Editing == editing {
				placeholder := field.placeholder
				if a.Inspector.MultiMixed {
					placeholder = field.mixed
				}
				a.inspectorEditor(c, editing, &a.Inspector.MultiDraft, field.label, placeholder, fontSecondary, field.lines, field.key == "source", func(cancel bool) {
					if !cancel {
						a.applyMultiDetail(field.key)
					}
				}).Margin(0, -inspectorEditBleed)
				continue
			}
			a.inspectorEditable(c, editing+"-display", field.label, func(p *designPalette) {
				switch {
				case mixed:
					ui.Text(c, field.mixed).FontSize(fontSecondary).TextColor(placeholderColor(c)).FillWidth()
				case strings.TrimSpace(common) == "":
					ui.Text(c, field.placeholder).FontSize(fontSecondary).TextColor(placeholderColor(c)).FillWidth()
				default:
					ui.Text(c, common).FontSize(fontSecondary).TextColor(p.Text).FillWidth()
				}
			}, func() {
				ids := make([]string, len(selected))
				for i, asset := range selected {
					ids[i] = asset.ID
				}
				a.Inspector.Editing, a.Inspector.MultiIDs = editing, ids
				a.Inspector.MultiDraft, a.Inspector.MultiOriginal, a.Inspector.MultiMixed = common, common, mixed
			}).Margin(0, -inspectorEditBleed)
		}
	})
}

func (a *App) applyMultiDetail(field string) {
	value := a.Inspector.MultiDraft
	if field == "source" {
		value = strings.TrimSpace(value)
	}
	if (a.Inspector.MultiMixed && strings.TrimSpace(value) == "") || (!a.Inspector.MultiMixed && value == a.Inspector.MultiOriginal) {
		return
	}
	ids := make([]string, 0, len(a.Inspector.MultiIDs))
	for _, id := range a.Inspector.MultiIDs {
		if _, ok := a.assetByID[id]; ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 || a.Library == nil {
		return
	}
	lib := a.Library
	write, patch := lib.SetAssetsNote, func(asset *model.Asset) { asset.Note = value }
	if field == "source" {
		write, patch = lib.SetAssetsSourceURL, func(asset *model.Asset) { asset.SourceURL = value }
	}
	a.Inspector.MultiSaving = true
	a.RunPatch("update_asset_details", ids, func(ctx context.Context) error {
		err := write(ctx, ids, value)
		a.Window.Update(func() {
			a.Inspector.MultiSaving = false
			// 没写进去：错误照常弹出，等着退出或换图库的这次先不走，免得以为存好了。
			if err != nil {
				a.quitAfterInspector, a.pendingLibraryAfterEdits = false, ""
			}
		})
		return err
	}, patch)
}

func (a *App) inspectorEXIF(c *ui.Context, asset model.Asset) {
	items := a.inspectorEXIFItems(a.assetEXIF(asset))
	if len(items) == 0 {
		return
	}
	// EXIF 可以收起：点小标题一行切换，收起状态记在偏好里。
	p := paletteFor(c.Theme())
	open := !a.Preferences.EXIFCollapsed
	ui.Column(c.Key("inspector-exif")).FillWidth().Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(p.hairline())
		// 上下各留 12：小标题离分隔线的距离和其他区块一样。
		header := ui.ButtonBase(c.Key("exif-toggle")).Label(a.T("exif")).Expanded(open).FillWidth().Padding(space12, 0).
			AlignItems(ui.Center).Gap(space4).Cursor(ui.CursorPointer)
		ink := inspectorSectionInk(c, p)
		if header.Hovered() {
			ink = p.TextSecondary
		}
		header.TextColor(header.AnimateColor("ink", ink, motionColor))
		header.Children(func() {
			ui.Text(c, strings.ToUpper(a.T("exif"))).FontSize(inspectorSectionTitleSz).FontWeight(600)
			ui.Spacer(c)
			arrow := appIcon(c, "caret-right").Size(12, 12)
			arrow.Rotate(arrow.AnimateTo("rotation", motionTarget(open)*90, motionSpring))
		})
		if header.Clicked() {
			a.Preferences.EXIFCollapsed = open
			a.queuePreferences()
		}
		if !open {
			return
		}
		body := ui.Column(c.Key("exif-body")).FillWidth().Gap(space12)
		body.Transition(ui.ElementTransition{Animation: &motionSpring, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}})
		body.Children(func() {
			for _, item := range items {
				inspectorInfo(c, item[0], item[1])
			}
		})
	})
}

func (a *App) saveInspectorTitle() { a.saveInspectorEdits() }

// 多选：最多三张缩略图错开叠放，底部压一块「已选择 N 个素材」。
func (a *App) inspectorStack(c *ui.Context, selected []model.Asset) {
	p := paletteFor(c.Theme())
	title := a.T("selected_assets", len(selected))
	ui.Box(c.Key("inspector-stack")).FillWidth().Height(156).Label(title).Children(func() {
		rotation := []float32{-6, 1.5, 7}
		offsetX := []float32{-34, 0, 34}
		offsetY := []float32{12, 0, 14}
		for i, asset := range selected[:min(3, len(selected))] {
			card := ui.Box(c.Key("stack-"+asset.ID)).Absolute().Size(inspectorMultiCardW, inspectorMultiCardH).
				Left((inspectorContentWidth-inspectorMultiCardW)/2+offsetX[i]).Top((156-inspectorMultiCardH)/2+offsetY[i]).
				Radius(radiusCard).Background(p.Subtle).Border(1, p.BorderSubtle).Shadow(0, 8, 28, 0, ui.Hex("#000000").Alpha(.24)).
				Rotate(rotation[i]).ZIndex(i).Clip().AccessHidden(true)
			layoutMotion(card)
			card.Children(func() {
				if bitmap := a.Bitmap(asset); bitmap != nil {
					ui.Image(c, bitmap).Fill().Fit(ui.Cover)
				}
			})
		}
		ui.Row(c.Key("stack-count")).Absolute().Left(0).Right(0).Bottom(5).ZIndex(5).Center().Children(func() {
			ui.Row(c).Height(28).Padding(0, 10).Radius(9).Background(p.Raised).Border(1, p.BorderSubtle).Shadow(0, 1, 2, 0, p.Shadow).Center().Children(func() {
				ui.Text(c, title).FontSize(fontSecondary).FontWeight(600).TextColor(p.Text).SingleLine()
			})
		})
	})
}

type inspectorMembership struct {
	ID, Name, ParentID, Icon string
}

func (a *App) inspectorTags(c *ui.Context, selected []model.Asset) {
	items := make([]inspectorMembership, 0, len(a.Snapshot.Tags))
	for _, tag := range a.Snapshot.Tags {
		items = append(items, inspectorMembership{ID: tag.ID, Name: tag.Name})
	}
	slices.SortFunc(items, func(x, y inspectorMembership) int {
		return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name))
	})
	a.inspectorMemberships(c, selected, items, false)
}

func (a *App) inspectorFolders(c *ui.Context, selected []model.Asset) {
	items := make([]inspectorMembership, 0, len(a.Snapshot.Folders))
	for _, folder := range a.Snapshot.Folders {
		items = append(items, inspectorMembership{ID: folder.ID, Name: folder.Name, ParentID: folder.ParentID, Icon: folder.Icon})
	}
	a.inspectorMemberships(c, selected, items, true)
}

// 标签、文件夹区：已加入的显示成药丸，悬停时前面的图标换成移除按钮；最后是加号药丸，点开选择器。
func (a *App) inspectorMemberships(c *ui.Context, selected []model.Asset, items []inspectorMembership, folder bool) {
	kind, plural, icon := "tag", "tags", "hashtag"
	open := &a.Inspector.TagsOpen
	if folder {
		kind, plural, icon = "folder", "folders", "folder"
		open = &a.Inspector.FoldersOpen
	}
	// 点了才取 ID，不在每帧复制全部选中项。
	set := func(id string, add bool) {
		ids := inspectorIDs(selected)
		if folder {
			a.SetFolders(ids, []string{id}, add)
		} else {
			a.SetTags(ids, []string{id}, add)
		}
	}
	p := paletteFor(c.Theme())
	inspectorSection(c, "inspector_"+plural, a.T(plural), func() {
		ui.Row(c.Key("chips")).FillWidth().Wrap().Gap(6).Children(func() {
			for _, item := range items {
				count := a.selectionMembers(item.ID, folder)
				if count == 0 {
					continue
				}
				chip := ui.Row(c.Key("member_"+item.ID)).Height(inspectorChipHeight).MaxWidthPercent(100).Padding(0, 9).Gap(5).
					Radius(inspectorChipRadius).AlignItems(ui.Center).Label(item.Name)
				layoutMotion(chip)
				hovered := chip.Hovered()
				chip.Background(chip.AnimateColor("chip-bg", p.Text.Alpha(themeValue(c, chipAlpha(hovered, .03, .07), chipAlpha(hovered, .04, .12))), motionColor))
				chip.Children(func() {
					remove := ui.ButtonBase(c.Key("remove")).Label(a.T("remove_"+kind, item.Name)).Size(18, 20).Padding(0).Center().Cursor(ui.CursorPointer).Disabled(a.Busy)
					remove.Children(func() {
						switch {
						case hovered || remove.Focused():
							appIcon(c, "close-circle").Size(14, 14).TextColor(p.Text)
						case folder:
							folderIcon(c, item.Icon).Size(12, 12).TextColor(p.Text)
						default:
							appIcon(c, icon).Size(12, 12).TextColor(p.Text)
						}
					})
					if remove.Clicked() {
						set(item.ID, false)
					}
					ui.Text(c, item.Name).FontSize(fontSecondary).FontWeight(500).TextColor(p.Text).SingleLine().MinWidth(0).Tooltip(inspectorMembershipPath(items, item.ID))
					if len(selected) > 1 && count < len(selected) {
						ui.Textf(c, "%d/%d", count, len(selected)).FontFeatures("tnum").FontSize(fontGroup).TextColor(p.TextSecondary)
					}
				})
			}
			add := ui.ButtonBase(c.Key("inspector_add_"+kind)).Label(a.T("add_"+plural)).Size(34, inspectorChipHeight).Padding(0).
				Radius(inspectorChipRadius).Center().Cursor(ui.CursorPointer).Disabled(a.Busy)
			background := p.Text.Alpha(themeValue(c, .045, .08))
			if add.Pressed() || *open {
				background = p.Text.Alpha(themeValue(c, .1, .16))
			} else if add.Hovered() {
				background = p.Text.Alpha(themeValue(c, .07, .12))
			}
			add.Background(add.AnimateColor("add-bg", background, motionColor))
			add.Children(func() { appIcon(c, "add-plain").Size(14, 14).TextColor(p.Text) })
			if add.Clicked() {
				*open = !*open
			}
			picker := floatingPopover(c.Key("inspector_"+kind+"_picker"), add, open, func() {
				ui.Column(c).Width(inspectorPickerWidth).Padding(10).Gap(10).Children(func() {
					if folder {
						a.inspectorFolderPicker(c, selected, items, set)
					} else {
						a.inspectorTagPicker(c, selected, items, set)
					}
				})
			})
			// 选择器在加号正下方居中弹出（放不下时 mygo 会自动挪回窗口里）。
			if picker.Valid() {
				picker.AttachTo(add, ui.AnchorBottom, ui.AnchorTop)
			}
			if !*open {
				a.Inspector.TagSearch, a.Inspector.FolderSearch = inspectorResetSearch(folder, a.Inspector.TagSearch, a.Inspector.FolderSearch)
			}
		})
	})
}

func chipAlpha(hovered bool, rest, hover float32) float32 {
	if hovered {
		return hover
	}
	return rest
}

// 选择器关上时清掉它自己的搜索词，下次打开从头开始。
func inspectorResetSearch(folder bool, tag, folderSearch string) (string, string) {
	if folder {
		return tag, ""
	}
	return "", folderSearch
}

// 标签选择器：搜索框、「全部标签 (N)」和一片标签药丸；搜不到就给一行「创建」。
// 点标签就加上并关掉选择器；回车优先加完全同名的，其次第一个还没加的，都没有就创建。
func (a *App) inspectorTagPicker(c *ui.Context, selected []model.Asset, items []inspectorMembership, set func(string, bool)) {
	p := paletteFor(c.Theme())
	query := strings.TrimSpace(a.Inspector.TagSearch)
	lower := strings.ToLower(query)
	var matches []inspectorMembership
	exact := ""
	for _, item := range items {
		if lower == "" || strings.Contains(strings.ToLower(item.Name), lower) {
			matches = append(matches, item)
		}
		if lower != "" && strings.ToLower(item.Name) == lower {
			exact = item.ID
		}
	}
	all := func(id string) bool { return a.selectionMembers(id, false) == len(selected) }
	choose := func(id string) {
		if a.Busy {
			return
		}
		set(id, true)
		a.Inspector.TagsOpen = false
	}
	_, _, submitted := searchFieldFocus(c.Key("picker-search"), &a.Inspector.TagSearch, a.T("search_tags"), a.T("clear_search"), true)
	if submitted && query != "" {
		switch {
		case exact != "":
			choose(exact)
		case slices.ContainsFunc(matches, func(m inspectorMembership) bool { return !all(m.ID) }):
			choose(matches[slices.IndexFunc(matches, func(m inspectorMembership) bool { return !all(m.ID) })].ID)
		default:
			a.createInspectorMembership(selected, query, false)
		}
	}
	ui.Scroll(c.Key("tag-choices")).Scrollbars(false).MaxHeight(inspectorPickerMaxList).FillWidth().Gap(space8).Children(func() {
		creating := query != "" && exact == ""
		if len(matches) == 0 {
			// 有「创建」行时不再提示「没有匹配的标签」，两句话说的是一回事。
			if !creating {
				inspectorPickerEmpty(c, "no-tags", a.T("no_tags"))
			}
		} else {
			ui.Text(c, fmt.Sprintf("%s (%d)", a.T("all_tags"), len(matches))).FontSize(fontGroup).FontWeight(600).TextColor(p.TextSecondary).Padding(0, 2)
			ui.Row(c.Key("choices")).FillWidth().Wrap().Gap(6).Children(func() {
				for _, item := range matches {
					count := a.selectionMembers(item.ID, false)
					done := count == len(selected)
					label := item.Name
					if count > 0 && !done {
						label += fmt.Sprintf(" (%d/%d)", count, len(selected))
					}
					choice := ui.ButtonBase(c.Key(item.ID)).Label(label).Height(inspectorChipHeight).MaxWidthPercent(100).Padding(0, 9).Gap(6).
						Radius(inspectorChipRadius).AlignItems(ui.Center).Cursor(ui.CursorPointer).Disabled(done || a.Busy).AccessSelected(done)
					ink := p.Text
					if done {
						ink = p.TextSecondary
					}
					choice.Background(choice.AnimateColor("choice-bg", p.Text.Alpha(themeValue(c, chipAlpha(choice.Hovered() || done, .04, .08), chipAlpha(choice.Hovered() || done, .08, .16))), motionColor))
					choice.Children(func() {
						appIcon(c, "hashtag").Size(12, 12).TextColor(ink)
						ui.Text(c, label).FontSize(fontSecondary).FontWeight(500).TextColor(ink).SingleLine().MinWidth(0)
					})
					if choice.Clicked() {
						choose(item.ID)
					}
				}
			})
		}
		if creating {
			a.inspectorCreateRow(c, selected, query, false)
		}
	})
}

// 文件夹选择器：搜索框下是可展开的文件夹树，勾选框切换加入或移出，选择器不关；搜索时连同上级一起显示并展开。
func (a *App) inspectorFolderPicker(c *ui.Context, selected []model.Asset, items []inspectorMembership, set func(string, bool)) {
	query := strings.TrimSpace(a.Inspector.FolderSearch)
	lower := strings.ToLower(query)
	children := map[string][]inspectorMembership{}
	exact := false
	for _, item := range items {
		children[item.ParentID] = append(children[item.ParentID], item)
		if lower != "" && strings.ToLower(item.Name) == lower {
			exact = true
		}
	}
	var matches func(id string) bool
	matches = func(id string) bool {
		for _, child := range children[id] {
			if strings.Contains(strings.ToLower(child.Name), lower) || matches(child.ID) {
				return true
			}
		}
		return false
	}
	if a.Inspector.FolderExpanded == nil {
		a.Inspector.FolderExpanded = map[string]bool{}
	}
	_, _, submitted := searchFieldFocus(c.Key("picker-search"), &a.Inspector.FolderSearch, a.T("search_folders"), a.T("clear_search"), true)
	if submitted && query != "" && !exact {
		a.createInspectorMembership(selected, query, true)
	}
	ui.Scroll(c.Key("folder-choices")).Scrollbars(false).MaxHeight(inspectorPickerMaxList + 40).FillWidth().Gap(1).Children(func() {
		shown := 0
		var walk func(parent string, depth int)
		walk = func(parent string, depth int) {
			for _, item := range children[parent] {
				hasChildren := len(children[item.ID]) > 0
				if lower != "" && !strings.Contains(strings.ToLower(item.Name), lower) && !matches(item.ID) {
					continue
				}
				shown++
				expanded := a.Inspector.FolderExpanded[item.ID] || (lower != "" && hasChildren)
				a.inspectorFolderRow(c, selected, items, item, depth, hasChildren, expanded, set)
				if hasChildren && expanded {
					walk(item.ID, depth+1)
				}
			}
		}
		walk("", 0)
		creating := query != "" && !exact
		if shown == 0 && !creating {
			inspectorPickerEmpty(c, "no-folders", a.T("no_folders"))
		}
		if creating {
			if shown > 0 {
				ui.Box(c).Height(space8 - 1)
			}
			a.inspectorCreateRow(c, selected, query, true)
		}
	})
}

func (a *App) inspectorFolderRow(c *ui.Context, selected []model.Asset, items []inspectorMembership, item inspectorMembership, depth int, hasChildren, expanded bool, set func(string, bool)) {
	p := paletteFor(c.Theme())
	count := a.selectionMembers(item.ID, true)
	checked := count == len(selected)
	row := ui.Row(c.Key("folder-"+item.ID)).FillWidth().Height(inspectorFolderRowH).Padding(0, 9, 0, float32(depth*inspectorFolderIndent)).Gap(2).Radius(11).AlignItems(ui.Center)
	background := ui.Transparent
	if row.Hovered() {
		background = p.Hover
	}
	row.Background(row.AnimateColor("row-bg", background, motionColor))
	row.Children(func() {
		toggle := ui.ButtonBase(c.Key("expand")).Size(20, inspectorFolderRowH).Padding(0).Center().Cursor(ui.CursorPointer)
		if hasChildren {
			toggle.Label(a.T("toggle_folder", item.Name)).Expanded(expanded)
			toggle.Children(func() {
				arrow := appIcon(c, "caret-right").Size(10, 10).TextColor(p.TextSecondary)
				arrow.Rotate(arrow.AnimateTo("rotation", map[bool]float32{false: 0, true: 90}[expanded], motionSpring))
			})
			if toggle.Clicked() {
				a.Inspector.FolderExpanded[item.ID] = !a.Inspector.FolderExpanded[item.ID]
			}
		} else {
			toggle.AccessHidden(true).IgnoreInput(true)
		}
		label := item.Name
		if count > 0 && !checked {
			label += fmt.Sprintf(" (%d/%d)", count, len(selected))
		}
		pick := ui.CheckboxBase(c.Key("pick"), &checked).Label(label).Grow(1).MinWidth(0).FillHeight().Gap(space8).AlignItems(ui.Center).Cursor(ui.CursorPointer).Disabled(a.Busy)
		pick.Children(func() {
			box := ui.Box(c.Key("box")).Size(15, 15).Radius(4.5).Shrink(0).Center()
			fill, border := p.Text.Alpha(themeValue(c, .035, .08)), p.Text.Alpha(themeValue(c, .11, .26))
			if checked || count > 0 {
				fill, border = p.Text.Alpha(themeValue(c, .08, .22)), p.Text.Alpha(themeValue(c, .18, .42))
			}
			box.Background(box.AnimateColor("fill", fill, motionColor)).Border(1, box.AnimateColor("border", border, motionColor))
			box.Children(func() {
				switch {
				case checked:
					appIcon(c, "check").Size(10, 10).TextColor(p.Text)
				case count > 0:
					// 部分素材在这个文件夹里：画一道短横。
					ui.Box(c).Size(7, 1.5).Radius(1).Background(p.Text)
				}
			})
			ui.Text(c, label).FontSize(fontSecondary).FontWeight(600).TextColor(p.Text).SingleLine().Grow(1).MinWidth(0).Tooltip(inspectorMembershipPath(items, item.ID))
		})
		if pick.Changed() {
			set(item.ID, checked)
		}
	})
}

// 选择器空状态：一个淡色小图标加「还没有标签」，居中，颜色比正文浅。
func inspectorPickerEmpty(c *ui.Context, art, text string) {
	p := paletteFor(c.Theme())
	ui.Column(c.Key("picker-empty")).FillWidth().Padding(space12, 0).Gap(space8).AlignItems(ui.Center).Children(func() {
		emptyArt(c.Key("art"), art, emptyArtSmall)
		ui.Text(c, text).FontSize(fontSecondary).TextColor(p.TextTertiary.Alpha(.8)).TextAlign(ui.Center)
	})
}

// 「创建 xxx」行：新建标签或文件夹并直接加到所选素材上。
func (a *App) inspectorCreateRow(c *ui.Context, selected []model.Asset, name string, folder bool) {
	p := paletteFor(c.Theme())
	row := ui.ButtonBase(c.Key("create")).Label(a.T("create_named", name)).FillWidth().Height(34).Padding(0, 10).Gap(space8).
		Radius(11).AlignItems(ui.Center).Cursor(ui.CursorPointer).Disabled(a.Busy)
	row.Background(row.AnimateColor("create-bg", p.Text.Alpha(themeValue(c, chipAlpha(row.Hovered(), .04, .08), chipAlpha(row.Hovered(), .08, .16))), motionColor))
	row.Children(func() {
		appIcon(c, "add-plain").Size(12, 12).TextColor(p.Text)
		ui.Text(c, a.T("create_named", name)).FontSize(fontSecondary).FontWeight(500).TextColor(p.Text).SingleLine().Grow(1).MinWidth(0)
	})
	if row.Clicked() {
		a.createInspectorMembership(selected, name, folder)
	}
}

func (a *App) createInspectorMembership(selected []model.Asset, name string, folder bool) {
	// 别的写入还没结束时 Run 会直接跳过；这时不清搜索词也不关选择器，用户可以稍后再按一次。
	if a.Busy {
		return
	}
	lib, ids := a.Library, inspectorIDs(selected)
	if folder {
		// 文件夹先弹出新建对话框选图标，建好后把所选素材放进去。
		a.Inspector.FoldersOpen, a.Inspector.FolderSearch = false, ""
		a.ShowNewFolder("", name, ids)
		return
	}
	a.Inspector.TagsOpen, a.Inspector.TagSearch = false, ""
	a.Run("create_asset_tag", func(ctx context.Context) error {
		item, err := lib.CreateTag(ctx, name)
		if err != nil {
			return err
		}
		return lib.SetAssetTags(ctx, ids, []string{item.ID}, true)
	})
}

func inspectorMembershipPath(items []inspectorMembership, id string) string {
	var parts []string
	for id != "" {
		i := slices.IndexFunc(items, func(item inspectorMembership) bool { return item.ID == id })
		if i < 0 {
			break
		}
		parts = append(parts, items[i].Name)
		id = items[i].ParentID
	}
	slices.Reverse(parts)
	return strings.Join(parts, " / ")
}

func inspectorIDs(selected []model.Asset) []string {
	ids := make([]string, len(selected))
	for i, asset := range selected {
		ids[i] = asset.ID
	}
	return ids
}

func (a *App) inspectorDraft() (inspectorEdit, bool) {
	state := a.Inspector
	name, source := strings.TrimSpace(state.Title), strings.TrimSpace(state.SourceURL)
	rename := name != "" && name != state.SourceTitle
	details := state.Note != state.SourceNote || source != state.SavedSourceURL
	return inspectorEdit{ID: state.AssetID, Name: name, Note: state.Note, Source: source, Rename: rename, Details: details}, state.AssetID != "" && (rename || details)
}

func (a *App) InspectorEditsPending() bool {
	_, dirty := a.inspectorDraft()
	_, multi := strings.CutPrefix(a.Inspector.Editing, "multi-")
	return dirty || len(a.Inspector.Pending) > 0 || multi || a.Inspector.MultiSaving || a.annotationSavesPending()
}

// 批注写失败也算在这里：退出、换图库时一样给「放弃修改」的出口。
func (a *App) InspectorEditsFailed() bool {
	return slices.ContainsFunc(a.Inspector.Pending, func(edit inspectorEdit) bool { return edit.Failed }) || a.annotationSavesFailed()
}

func (a *App) saveInspectorEdits() {
	edit, dirty := a.inspectorDraft()
	if !dirty || a.Library == nil {
		return
	}
	found := false
	for i, pending := range a.Inspector.Pending {
		if pending.ID == edit.ID {
			a.Inspector.Pending[i] = edit
			found = true
			break
		}
	}
	if !found {
		a.Inspector.Pending = append(a.Inspector.Pending, edit)
	}
	a.ContinueInspectorEdits()
}

// 锁屏和退出前调用；保存失败后再次明确调用才重试。
func (a *App) FlushInspectorEdits() {
	// 多选时正在一起改备注或链接（还没失焦就按了退出、换图库）：按开始时那批存下。
	if field, ok := strings.CutPrefix(a.Inspector.Editing, "multi-"); ok {
		a.applyMultiDetail(field)
		a.Inspector.Editing, a.Inspector.EditKey = "", 0
	}
	a.saveInspectorEdits()
	for i := range a.Inspector.Pending {
		a.Inspector.Pending[i].Failed = false
	}
	a.ContinueInspectorEdits()
	// 预览里正在写的批注也一起存下，之前没写进去的再试一次。
	a.flushPreviewAnnotations()
}

// 每帧推进已失焦的草稿，不碰正在输入的内容，也不循环重试失败写入。
func (a *App) ContinueInspectorEdits() {
	if a.Busy || a.Library == nil {
		return
	}
	for _, edit := range a.Inspector.Pending {
		if !edit.Failed {
			a.runInspectorEdit(edit)
			return
		}
	}
}

func (a *App) runInspectorEdit(edit inspectorEdit) {
	lib := a.Library
	a.RunPatch("update_asset_details", []string{edit.ID}, func(ctx context.Context) error {
		err := func() error {
			if edit.Rename {
				if err := lib.RenameAsset(ctx, edit.ID, edit.Name); err != nil {
					return err
				}
			}
			if edit.Details {
				return lib.UpdateAssetDetails(ctx, edit.ID, edit.Note, edit.Source)
			}
			return nil
		}()
		a.Window.Update(func() { a.inspectorEditFinished(edit, err) })
		return err
	}, func(asset *model.Asset) {
		if edit.Rename {
			asset.Name = edit.Name
		}
		if edit.Details {
			asset.Note, asset.SourceURL = edit.Note, edit.Source
		}
	})
}

func (a *App) inspectorEditFinished(edit inspectorEdit, err error) {
	// 保存期间用户可能又改了同一张图（导入时输入框不锁），队列里已换成新版本：只认这次提交的那一版，新版本留着接着存。
	submitted := func(pending inspectorEdit) bool {
		pending.Failed = edit.Failed
		return pending == edit
	}
	if err != nil {
		a.Inspector.SaveError = a.Texts.Error(err)
		for i, pending := range a.Inspector.Pending {
			if submitted(pending) {
				a.Inspector.Pending[i].Failed = true
			}
		}
		return
	}
	a.Inspector.Pending = slices.DeleteFunc(a.Inspector.Pending, submitted)
	if a.Inspector.AssetID == edit.ID {
		if edit.Rename {
			a.Inspector.SourceTitle = edit.Name
		}
		if edit.Details {
			a.Inspector.SourceNote = edit.Note
			a.Inspector.SavedSourceURL = edit.Source
		}
	}
}

func (a *App) saveInspectorDetails() { a.saveInspectorEdits() }

// 素材的 EXIF：快照里没带（打开图库时不读，省内存和时间）就在后台从库里取，取到之前先不显示这一块。
func (a *App) assetEXIF(asset model.Asset) map[string]string {
	if asset.EXIF != nil {
		return asset.EXIF
	}
	state := &a.Inspector
	if state.EXIFFor == asset.ID {
		return state.EXIF
	}
	if a.Library == nil || a.Window == nil || state.EXIFLoading == asset.ID {
		return nil
	}
	state.EXIFLoading = asset.ID
	lib, id, gen := a.Library, asset.ID, a.generation
	go func() {
		exif, err := lib.AssetEXIF(a.ctx, id)
		a.Window.Update(func() {
			if gen != a.generation || a.Inspector.EXIFLoading != id {
				return
			}
			a.Inspector.EXIFLoading = ""
			if err != nil {
				exif = map[string]string{}
				// 素材刚被删掉时取不到是正常的；别的错误（比如库里的 EXIF 坏了）要让用户看到。
				if !errors.Is(err, sql.ErrNoRows) {
					slog.Error("读取 EXIF 失败", "asset", id, "error", err)
					a.ShowError(err)
				}
			}
			a.Inspector.EXIFFor, a.Inspector.EXIF = id, exif
		})
	}()
	return nil
}

// 放弃还没写进图库的名称、备注和来源链接修改。内容先完整写进日志，用户事后还能从日志里找回。
func (a *App) DiscardInspectorEdits(reason string) bool {
	edits := slices.Clone(a.Inspector.Pending)
	// 输入框里还没提交的内容可能比排队的那份更新，两份都记下。
	if draft, dirty := a.inspectorDraft(); dirty && !slices.Contains(edits, draft) {
		edits = append(edits, draft)
	}
	for _, edit := range edits {
		slog.Warn("放弃未保存的检查器修改", "reason", reason, "asset", edit.ID, "name", edit.Name, "note", edit.Note, "source", edit.Source)
	}
	a.Inspector = inspectorState{}
	annotations := a.discardAnnotationSaves(reason)
	return len(edits) > 0 || annotations
}

// 分组标题比下面的字段名（类型、尺寸……）再浅一档，一眼能分出是标题还是字段。
func inspectorSectionInk(c *ui.Context, p *designPalette) ui.Color {
	return p.TextTertiary.Alpha(themeValue(c, .62, .7))
}

func inspectorIconButton(c *ui.Context, label, icon string) ui.Element {
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(controlSmall, controlSmall).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer)
	p := paletteFor(c.Theme())
	background, text := ui.Transparent, p.TextSecondary
	if button.Pressed() {
		background = p.Pressed
	} else if button.Hovered() {
		background = p.Hover
	}
	if button.Hovered() {
		text = p.Text
	}
	animateColors(button, background, text)
	button.Children(func() { appIcon(c, icon).Size(iconMedium, iconMedium) })
	return button
}

func inspectorRemoveButton(c *ui.Context, label string) ui.Element {
	p := paletteFor(c.Theme())
	button := ui.ButtonBase(c).Label(label).Tooltip(label).Size(controlSmall, controlSmall).Padding(0).Radius(radiusControl).Cursor(ui.CursorPointer)
	background, text := ui.Transparent, p.TextTertiary
	if button.Pressed() {
		background = p.Pressed
	} else if button.Hovered() {
		background = p.Hover
	}
	if button.Hovered() {
		text = p.Text
	}
	animateColors(button, background, text)
	button.Children(func() { appIcon(c, "close").Size(space12, space12) })
	return button
}
