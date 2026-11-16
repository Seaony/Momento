package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo/ui"
)

const (
	annotationEditorWidth  float32 = 260
	annotationEditorHeight float32 = 96
	annotationClickSlop    float32 = 4 // 按下到松开移动不到这么多算点击，不算画框
)

// previewNotes 是预览窗口里区域批注的状态。
type previewNotes struct {
	On         bool // 批注模式：在图上拖出框；不在批注模式时框照样显示，只是不能画
	drawing    bool
	start, end ui.Point // 视口坐标里按下和现在的位置
	edit       *annotationEdit
	editorRect ui.Rect
}

// annotationEdit 是正在写文字的批注：ID 为空是刚画的新框。
type annotationEdit struct {
	ID   string
	Rect model.Annotation
	Text string
}

// annotationView 把存下的框换成预览里显示的位置（0～1，相对显示出来的图）：先按手动旋转转，再按看图时的水平翻转镜像。
func annotationView(note model.Annotation, rotation int, flipped bool) (x, y, w, h float64) {
	forward := func(px, py float64) (float64, float64) {
		switch (rotation%360 + 360) % 360 {
		case 90:
			px, py = 1-py, px
		case 180:
			px, py = 1-px, 1-py
		case 270:
			px, py = py, 1-px
		}
		if flipped {
			px = 1 - px
		}
		return px, py
	}
	x0, y0 := forward(note.X, note.Y)
	x1, y1 := forward(note.X+note.W, note.Y+note.H)
	return math.Min(x0, x1), math.Min(y0, y1), math.Abs(x1 - x0), math.Abs(y1 - y0)
}

// annotationStored 是 annotationView 反过来：把显示出来的图上拖出的两个角（0～1）换成要存的框，超出图的部分收进来。
func annotationStored(u0, v0, u1, v1 float64, rotation int, flipped bool) model.Annotation {
	back := func(u, v float64) (float64, float64) {
		u, v = math.Max(0, math.Min(1, u)), math.Max(0, math.Min(1, v))
		if flipped {
			u = 1 - u
		}
		switch (rotation%360 + 360) % 360 {
		case 90:
			return v, 1 - u
		case 180:
			return 1 - u, 1 - v
		case 270:
			return 1 - v, u
		}
		return u, v
	}
	x0, y0 := back(u0, v0)
	x1, y1 := back(u1, v1)
	return model.Annotation{X: math.Min(x0, x1), Y: math.Min(y0, y1), W: math.Abs(x1 - x0), H: math.Abs(y1 - y0)}
}

// liveAnnotations 是素材现在的批注：从图库数据里取，预览窗口里存的素材副本不会跟着改。
// 别的图库留下来的固定预览窗口不显示、也不改批注：素材 ID 可能在当前图库里对上别的图。
func (a *App) liveAnnotations(id string) []model.Annotation {
	return a.assetByID[id].Annotations
}

// 正在后台换图库时也不能改：换好后旧窗口的写入会被当成过期丢掉。
func (w *previewWindow) annotatable(a *App) bool {
	return a.Library != nil && !a.libraryOpening && a.Library.Root == w.root && !isVideoAsset(w.asset)
}

// annotationLayer 在图上画批注框和编号；指针停在框上时在框下面显示文字，正在写的框旁边是输入框。photo 是图在视口里的位置。
func (w *previewWindow) annotationLayer(c *ui.Context, a *App, photo ui.Rect, pointer ui.Point, over bool) {
	p := paletteFor(c.Theme())
	notes := &w.notes
	toView := func(note model.Annotation) ui.Rect {
		x, y, bw, bh := annotationView(note, w.asset.Rotation, w.flipped)
		return ui.Rect{X: photo.X + float32(x)*photo.W, Y: photo.Y + float32(y)*photo.H, W: float32(bw) * photo.W, H: float32(bh) * photo.H}
	}
	if !w.annotatable(a) {
		return
	}
	hovered := -1
	list := a.liveAnnotations(w.asset.ID)
	for i, note := range list {
		box := toView(note)
		active := notes.edit != nil && notes.edit.ID == note.ID
		if over && !notes.drawing && box.Contains(pointer.X, pointer.Y) {
			hovered = i
		}
		fill := p.Accent.Alpha(.08)
		if active || hovered == i {
			fill = p.Accent.Alpha(.18)
		}
		annotationBox(c.Key(note.ID), box, fill, p.Accent).Children(func() {
			ui.Text(c, fmt.Sprint(i+1)).Absolute().Left(2).Top(2).MinWidth(16).Height(16).Radius(8).Padding(0, 4).TextAlign(ui.Center).
				Background(p.Accent).TextColor(p.OnAccent).FontSize(fontCaption).FontWeight(600).FontFeatures("tnum")
		})
	}
	if notes.drawing {
		box := rectFromPoints(notes.start, notes.end)
		annotationBox(c.Key("annotation-draft"), box, p.Accent.Alpha(.12), p.Accent)
	} else if notes.edit != nil && notes.edit.ID == "" {
		annotationBox(c.Key("annotation-new"), toView(notes.edit.Rect), p.Accent.Alpha(.18), p.Accent)
	}
	if notes.edit != nil {
		w.annotationEditor(c, a, toView(notes.edit.Rect))
	} else if hovered >= 0 && strings.TrimSpace(list[hovered].Text) != "" {
		box := toView(list[hovered])
		width := min(annotationEditorWidth, w.picture.viewport.W-2*space8)
		ui.Text(c.Key("annotation-tip"), list[hovered].Text).Absolute().Left(clamp(box.X, space8, w.picture.viewport.W-width-space8)).Top(min(box.Y+box.H+space4, w.picture.viewport.H-28)).
			MaxWidth(width).Padding(space4, space8).Radius(radiusSmall + 2).Background(ui.RGBA(0, 0, 0, .72)).TextColor(ui.RGBA(255, 255, 255, .95)).
			FontSize(fontSecondary).IgnoreInput(true)
	}
}

func annotationBox(c *ui.Context, box ui.Rect, fill, border ui.Color) ui.Element {
	return ui.Box(c).Absolute().Left(box.X).Top(box.Y).Size(max(box.W, 1), max(box.H, 1)).Radius(2).Background(fill).Border(1.5, border).IgnoreInput(true)
}

func rectFromPoints(a, b ui.Point) ui.Rect {
	return ui.Rect{X: min(a.X, b.X), Y: min(a.Y, b.Y), W: float32(math.Abs(float64(b.X - a.X))), H: float32(math.Abs(float64(b.Y - a.Y)))}
}

func clamp(v, low, high float32) float32 { return max(low, min(v, high)) }

// annotationEditor 是写批注文字的小面板：放在框下面，下面放不下就放上面。回车保存，Esc 取消。
func (w *previewWindow) annotationEditor(c *ui.Context, a *App, box ui.Rect) {
	p := paletteFor(c.Theme())
	view := w.picture.viewport
	edit := w.notes.edit
	width := min(annotationEditorWidth, view.W-2*space8)
	top := box.Y + box.H + space8
	if top+annotationEditorHeight > view.H-space8 {
		top = box.Y - space8 - annotationEditorHeight
	}
	top = clamp(top, space8, view.H-annotationEditorHeight-space8)
	left := clamp(box.X, space8, view.W-width-space8)
	w.notes.editorRect = ui.Rect{X: left, Y: top, W: width, H: annotationEditorHeight}
	var input ui.Element
	save, remove, cancel := false, false, false
	elevated(ui.Column(c.Key("annotation-editor")).Absolute().Left(left).Top(top).Width(width).Height(annotationEditorHeight).Padding(space12).Gap(space8).
		Radius(radiusControl+2).Background(panelColor(c.Theme())), c.Theme()).Children(func() {
		input = fieldStyle(c, ui.TextInput(c.Key("annotation-text"), &edit.Text).Label(a.T("annotation_text")).Placeholder(a.T("annotation_placeholder")).AutoFocus().FillWidth())
		ui.Row(c).FillWidth().Gap(space8).AlignItems(ui.Center).Children(func() {
			if edit.ID != "" {
				remove = ui.ButtonBase(c.Key("annotation-delete")).Label(a.T("delete")).Cursor(ui.CursorPointer).Children(func() {
					ui.Text(c, a.T("delete")).FontSize(fontSecondary).TextColor(p.Danger)
				}).Clicked()
			}
			ui.Spacer(c)
			cancel = settingsButton(c.Key("annotation-cancel"), a.T("cancel")).Clicked()
			save = settingsButton(c.Key("annotation-save"), a.T("save")).Clicked()
		})
	})
	switch {
	case save || input.Submitted():
		w.commitAnnotation(a)
	case remove:
		a.deleteAnnotation(w.asset.ID, edit.ID)
		w.notes.edit = nil
	case cancel:
		w.notes.edit = nil
	}
}

// annotationInput 处理批注模式下在图上的按下、拖动和松开：拖出新框，点一下已有的框改它的文字。
func (w *previewWindow) annotationInput(viewport ui.Element, a *App, photo ui.Rect, pointer ui.Point) {
	notes := &w.notes
	if !w.annotatable(a) {
		notes.drawing = false
		return
	}
	pressed := viewport.Pressed()
	switch {
	case pressed && !notes.drawing:
		// 在输入框面板上按下不算画框。
		if notes.edit != nil && notes.editorRect.Contains(pointer.X, pointer.Y) {
			return
		}
		notes.drawing, notes.start, notes.end = true, pointer, pointer
	case pressed:
		notes.end = pointer
	case notes.drawing:
		notes.drawing = false
		notes.end = pointer
		dx, dy := notes.end.X-notes.start.X, notes.end.Y-notes.start.Y
		if dx*dx+dy*dy < annotationClickSlop*annotationClickSlop {
			w.pickAnnotation(a, photo, notes.start)
			return
		}
		if photo.W <= 0 || photo.H <= 0 {
			return
		}
		rect := annotationStored(float64((notes.start.X-photo.X)/photo.W), float64((notes.start.Y-photo.Y)/photo.H),
			float64((notes.end.X-photo.X)/photo.W), float64((notes.end.Y-photo.Y)/photo.H), w.asset.Rotation, w.flipped)
		if rect.W < .01 || rect.H < .01 {
			return
		}
		// 上一个框的文字还没存就画了新框：先把它存了，不悄悄丢掉。
		if notes.edit != nil {
			w.commitAnnotation(a)
		}
		notes.edit = &annotationEdit{Rect: rect}
	}
}

// pickAnnotation 点了图上某处：点在框里就改那个框（重叠时取后加的），点在空白处收起输入框。
func (w *previewWindow) pickAnnotation(a *App, photo ui.Rect, at ui.Point) {
	list := a.liveAnnotations(w.asset.ID)
	for i := len(list) - 1; i >= 0; i-- {
		x, y, bw, bh := annotationView(list[i], w.asset.Rotation, w.flipped)
		box := ui.Rect{X: photo.X + float32(x)*photo.W, Y: photo.Y + float32(y)*photo.H, W: float32(bw) * photo.W, H: float32(bh) * photo.H}
		if box.Contains(at.X, at.Y) {
			// 点的就是正在改的那条：保留输入框里改了一半的字。
			if w.notes.edit != nil && w.notes.edit.ID == list[i].ID {
				return
			}
			if w.notes.edit != nil {
				w.commitAnnotation(a)
			}
			w.notes.edit = &annotationEdit{ID: list[i].ID, Rect: list[i], Text: list[i].Text}
			return
		}
	}
	if w.notes.edit != nil {
		w.commitAnnotation(a)
	}
}

// commitAnnotation 存下正在写的批注并收起输入框：新框加进去；已有的框文字变了才写。
func (w *previewWindow) commitAnnotation(a *App) {
	edit := w.notes.edit
	w.notes.edit = nil
	if edit == nil {
		return
	}
	if !w.annotatable(a) {
		slog.Warn("预览窗口不属于当前图库，批注没有保存", "asset", w.asset.ID, "text", edit.Text)
		return
	}
	if edit.ID == "" {
		note := edit.Rect
		note.Text = edit.Text
		a.addAnnotation(w.asset.ID, note)
		return
	}
	if i := slices.IndexFunc(a.liveAnnotations(w.asset.ID), func(n model.Annotation) bool { return n.ID == edit.ID }); i >= 0 &&
		strings.TrimSpace(a.liveAnnotations(w.asset.ID)[i].Text) != strings.TrimSpace(edit.Text) {
		a.updateAnnotation(w.asset.ID, edit.ID, edit.Text)
	}
}

// ToggleAnnotating 开关预览里的批注模式；关的时候正在写的文字先存下。
func (w *previewWindow) toggleAnnotating(a *App) {
	if !w.annotatable(a) && !w.notes.On {
		return
	}
	if w.notes.On {
		w.commitAnnotation(a)
		w.notes.drawing = false
	}
	w.notes.On = !w.notes.On
}

// annotationSave 是一次还没确认写进图库的批注改动。写成功才从列表里拿掉；写失败留着并标 Failed，
// 退出、换图库时和检查器的修改一样算「还有没存的」，可以重试或明确放弃，不会悄悄丢字。
type annotationSave struct {
	key        int
	generation uint64
	AssetID    string
	Action     string // add、update、delete
	Note       model.Annotation
	Failed     bool
}

func (a *App) addAnnotation(assetID string, note model.Annotation) {
	a.saveAnnotation(annotationSave{AssetID: assetID, Action: "add", Note: note})
}

func (a *App) updateAnnotation(assetID, id, text string) {
	a.saveAnnotation(annotationSave{AssetID: assetID, Action: "update", Note: model.Annotation{ID: id, Text: strings.TrimSpace(text)}})
}

func (a *App) deleteAnnotation(assetID, id string) {
	a.saveAnnotation(annotationSave{AssetID: assetID, Action: "delete", Note: model.Annotation{ID: id}})
}

func (a *App) saveAnnotation(save annotationSave) {
	if a.Library == nil {
		return
	}
	a.annotationKey++
	save.key, save.generation = a.annotationKey, a.generation
	// 同一条批注之前写失败的改动被这次取代：不然下次重试会把新写的字改回旧的。
	if save.Note.ID != "" {
		a.annotationSaves = slices.DeleteFunc(a.annotationSaves, func(s annotationSave) bool {
			if s.Failed && s.Note.ID == save.Note.ID {
				slog.Info("新的批注改动取代之前写失败的", "annotation", s.Note.ID, "old", s.Note.Text)
				return true
			}
			return false
		})
	}
	a.annotationSaves = append(a.annotationSaves, save)
	a.runAnnotationSave(save)
}

func (a *App) runAnnotationSave(save annotationSave) {
	lib := a.Library
	var added model.Annotation
	work := func(ctx context.Context) error {
		var err error
		switch save.Action {
		case "add":
			added, err = lib.AddAnnotation(ctx, save.AssetID, save.Note)
		case "update":
			err = lib.UpdateAnnotation(ctx, save.Note.ID, save.Note.Text)
		case "delete":
			err = lib.DeleteAnnotation(ctx, save.Note.ID)
		}
		return err
	}
	a.run("annotation_"+save.Action, work, &assetPatch{ids: []string{save.AssetID}, failed: func(err error) { a.annotationSaveFailed(save.key, err) }, apply: func(asset *model.Asset) {
		notes := slices.Clone(asset.Annotations)
		switch save.Action {
		case "add":
			notes = append(notes, added)
		case "update":
			for i := range notes {
				if notes[i].ID == save.Note.ID {
					notes[i].Text = save.Note.Text
				}
			}
		case "delete":
			notes = slices.DeleteFunc(notes, func(n model.Annotation) bool { return n.ID == save.Note.ID })
		}
		asset.Annotations = notes
		a.annotationSaves = slices.DeleteFunc(a.annotationSaves, func(s annotationSave) bool { return s.key == save.key })
		a.InvalidatePreviewWindows()
	}})
}

// 写失败：留着这条并标出来。等着退出或换图库时，和检查器写失败一样弹出「放弃修改」的确认，不放弃就留着，下次退出再试。
func (a *App) annotationSaveFailed(key int, err error) {
	for i := range a.annotationSaves {
		if a.annotationSaves[i].key == key {
			a.annotationSaves[i].Failed = true
		}
	}
	a.Inspector.SaveError = a.Texts.Error(err)
}

// retryAnnotationSaves 重新写失败的批注；只在用户再次退出、换图库、锁定时调用，不自己循环重试。
func (a *App) retryAnnotationSaves() {
	for i := range a.annotationSaves {
		if save := a.annotationSaves[i]; save.Failed && save.generation == a.generation {
			a.annotationSaves[i].Failed = false
			a.runAnnotationSave(save)
		}
	}
}

func (a *App) annotationSavesPending() bool { return len(a.annotationSaves) > 0 }

func (a *App) annotationSavesFailed() bool {
	return slices.ContainsFunc(a.annotationSaves, func(s annotationSave) bool { return s.Failed })
}

// discardAnnotationSaves 放弃没写进去的批注，内容先写进日志，事后还能找回。
func (a *App) discardAnnotationSaves(reason string) bool {
	for _, save := range a.annotationSaves {
		slog.Warn("放弃未保存的批注", "reason", reason, "action", save.Action, "asset", save.AssetID, "annotation", save.Note.ID, "text", save.Note.Text)
	}
	discarded := len(a.annotationSaves) > 0
	a.annotationSaves = nil
	return discarded
}

// flushPreviewAnnotations 把各个预览窗口里正在写的批注存下，再重试之前没写进去的；有要等的返回 true。
func (a *App) flushPreviewAnnotations() bool {
	windows := slices.Clone(a.Preview.pinned)
	if a.Preview.active != nil {
		windows = append(windows, a.Preview.active)
	}
	for _, w := range windows {
		w.commitAnnotation(a)
	}
	a.retryAnnotationSaves()
	return a.annotationSavesPending()
}

// AnnotateAsset 打开预览并进入批注模式；noteID 不为空时直接改那一条。
func (a *App) AnnotateAsset(id, noteID string) {
	// 已经开着别的图的预览：直接换成这张（换之前会存下上一张正在写的批注）。
	if w := a.Preview.active; w != nil && !w.closing && w.asset.ID != id {
		if asset, ok := a.previewAsset(id); ok {
			w.setAsset(a, asset)
		}
	} else {
		a.ShowPreview(id, false)
	}
	w := a.Preview.active
	if w == nil || w.asset.ID != id || !w.annotatable(a) {
		return
	}
	w.notes.On = true
	for _, note := range a.liveAnnotations(id) {
		if note.ID != noteID {
			continue
		}
		// 正在改的就是这条：留着输入框里的字；正在改别的：先存下再换。
		if w.notes.edit != nil && w.notes.edit.ID == noteID {
			break
		}
		w.commitAnnotation(a)
		w.notes.edit = &annotationEdit{ID: note.ID, Rect: note, Text: note.Text}
	}
	w.invalidate()
}

// inspectorAnnotations 在检查器里列出这张图的批注，点一条打开预览改它。没有批注时不显示。
func (a *App) inspectorAnnotations(c *ui.Context, asset model.Asset) {
	if len(asset.Annotations) == 0 {
		return
	}
	inspectorSection(c, "inspector-annotations", a.T("annotations"), func() {
		p := paletteFor(c.Theme())
		ui.Column(c).FillWidth().Gap(space4).Children(func() {
			for i, note := range asset.Annotations {
				row := ui.ButtonBase(c.Key(note.ID)).FillWidth().Padding(space4, space4).Gap(space8).Radius(radiusSmall).Cursor(ui.CursorPointer).Label(note.Text)
				if row.Hovered() {
					row.Background(p.Hover)
				}
				row.Children(func() {
					ui.Text(c, fmt.Sprint(i+1)).MinWidth(16).Height(16).Radius(8).Padding(0, 4).TextAlign(ui.Center).Shrink(0).
						Background(p.Accent).TextColor(p.OnAccent).FontSize(fontCaption).FontWeight(600).FontFeatures("tnum")
					if text := strings.TrimSpace(note.Text); text != "" {
						ui.Text(c, text).FontSize(fontSecondary).TextColor(p.Text).Grow(1).MinWidth(0)
					} else {
						ui.Text(c, a.T("annotation_empty")).FontSize(fontSecondary).TextColor(placeholderColor(c)).Grow(1).MinWidth(0)
					}
				})
				if row.Clicked() {
					a.AnnotateAsset(asset.ID, note.ID)
				}
			}
		})
	})
}

// dropPreviewAnnotationDrafts 图库已经没了（外接盘拔掉）时用：没法再写，把各窗口正在写的批注记进日志后清掉，不再发起写入。
func (a *App) dropPreviewAnnotationDrafts(reason string) (dropped bool) {
	windows := slices.Clone(a.Preview.pinned)
	if a.Preview.active != nil {
		windows = append(windows, a.Preview.active)
	}
	for _, w := range windows {
		if edit := w.notes.edit; edit != nil {
			slog.Warn("放弃未保存的批注", "reason", reason, "asset", w.asset.ID, "annotation", edit.ID, "text", edit.Text)
			dropped = true
		}
		w.notes.edit, w.notes.drawing = nil, false
	}
	return dropped
}
