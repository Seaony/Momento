package ui

import (
	"bytes"
	"context"
	"path/filepath"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/library"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestPreviewHoldBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name       string
		duration   time.Duration
		close, pin bool
	}{
		{"快按边界前", previewPeekThreshold - time.Nanosecond, false, false},
		{"偷看边界", previewPeekThreshold, true, false},
		{"固定边界前", previewPinThreshold - time.Nanosecond, true, false},
		{"固定边界", previewPinThreshold, false, true},
		{"固定之后", previewPinThreshold + time.Second, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := readyPreviewApp()
			w := a.Preview.active
			w.openedAt = time.Now()
			w.pressedAt = w.openedAt
			w.picture.Held = true
			a.Preview.gesture = w
			w.release(a, w.pressedAt.Add(tc.duration))
			if w.closing != tc.close || w.pinned != tc.pin {
				t.Fatalf("duration=%s close=%v pin=%v", tc.duration, w.closing, w.pinned)
			}
			if tc.pin && (a.Preview.active != nil || len(a.Preview.pinned) != 1 || a.Preview.pinned[0] != w) {
				t.Fatal("固定时重建了会话")
			}
			if tc.close && !w.closedHeld {
				t.Fatal("偷看关闭首帧丢掉了进度层")
			}
		})
	}
}

func TestPreviewHoldProgressAndKeyRepeat(t *testing.T) {
	a := readyPreviewApp()
	w := a.Preview.active
	w.pressedAt = time.Now()
	w.openedAt = w.pressedAt
	w.picture.Held = true
	a.Preview.gesture = w
	if w.holdProgress(w.pressedAt.Add(previewPeekThreshold-time.Nanosecond)) != 0 {
		t.Fatal("250ms前显示进度")
	}
	if w.holdProgress(w.pressedAt.Add(previewPeekThreshold)) != 0 {
		t.Fatal("进度起点不是0")
	}
	if got := w.holdProgress(w.pressedAt.Add((previewPeekThreshold + previewPinThreshold) / 2)); got < .499 || got > .501 {
		t.Fatalf("中点进度=%v", got)
	}
	w.advanceHold(a, w.pressedAt.Add(previewPinThreshold))
	a.ObservePreviewInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeySpace, Repeat: true, Time: w.pressedAt.Add(4 * time.Second)})
	a.ShowPreview("one", true)
	if a.Preview.active != nil || len(a.Preview.pinned) != 1 {
		t.Fatal("自动重复按键创建了额外窗口")
	}
	a.ObservePreviewInput(ui.InputEvent{Kind: ui.InputKeyUp, Key: ui.KeySpace, Time: w.pressedAt.Add(4 * time.Second)})
	if a.Preview.gesture != nil || w.picture.Held || w.closing {
		t.Fatal("固定后的松开关闭了窗口或留下按住状态")
	}
	a.ShowPreview("one", true)
	if a.Preview.active == nil || a.Preview.active == w {
		t.Fatal("下一次按住没有创建独立会话")
	}
}

func TestPreviewSelectionOnlyFollowsTemporaryWindow(t *testing.T) {
	a := readyPreviewApp()
	first := a.Preview.active
	first.picture.Held = true
	first.pressedAt = time.Now().Add(-previewPinThreshold)
	first.advanceHold(a, time.Now())
	first.picture.Held = false
	a.Visible = append(a.Visible, model.Asset{ID: "two", Name: "第二张", Width: 200, Height: 300})
	a.ShowPreview("one", false)
	temporary := a.Preview.active
	a.Selected.Add("two")
	a.selectedIndex = 1
	a.FollowPreviewSelection()
	if first.asset.ID != "one" || temporary.asset.ID != "two" || first.picture.Bitmap == nil {
		t.Fatal("选择变化污染了固定窗口")
	}
	a.Selected.Clear()
	a.FollowPreviewSelection()
	if !temporary.closing || first.closing {
		t.Fatal("清除选择没有只关闭暂态窗口")
	}
	first.focused = true
	if !a.PreviewOpen() || a.PreviewAssetID() != "one" {
		t.Fatal("焦点固定窗口不是菜单目标")
	}
	first.focused = false
	if a.PreviewOpen() {
		t.Fatal("后台固定窗口屏蔽了主窗口快捷键")
	}
}

func TestPreviewCloseAllDisposesActiveAndPinnedImmediately(t *testing.T) {
	a := readyPreviewApp()
	first := a.Preview.active
	first.picture.Held = true
	first.pressedAt = time.Now().Add(-previewPinThreshold)
	first.advanceHold(a, time.Now())
	first.picture.Held = false
	a.ShowPreview("one", false)
	second := a.Preview.active
	a.Preferences.PasswordHash = "configured"
	a.LockNow()
	if a.PreviewOpen() || len(a.Preview.pinned) != 0 || a.Preview.gesture != nil || !first.disposed || !second.disposed || first.picture.Bitmap != nil {
		t.Fatal("全部关闭仍保留可见图片或会话")
	}
}

func TestPreviewWindowBoundsAndCloseAreContinuous(t *testing.T) {
	parent := mygo.Rectangle{X: 100, Y: 100, Width: 1200, Height: 800}
	area := mygo.Rectangle{Width: 1440, Height: 900}
	for _, asset := range []model.Asset{{Width: 4000, Height: 3000}, {Width: 100, Height: 80}, {Width: 200, Height: 4000}} {
		r := previewWindowBounds(parent, area, asset, true)
		if r.Width > 960 || r.Height > 640 || r.X < area.X || r.Y < area.Y || r.X+r.Width > area.Width || r.Y+r.Height > area.Height {
			t.Fatalf("窗口越界：%+v", r)
		}
	}
	a := readyPreviewApp()
	w := a.Preview.active
	start := time.Now()
	w.openedAt = start.Add(-time.Second)
	w.close(a, start)
	lastScale, lastOpacity := float32(2), float32(2)
	for _, fraction := range []float32{0, .35, .70, 1} {
		scale, opacity := w.visual(start.Add(time.Duration(float64(previewWindowDuration) * float64(fraction))))
		if scale > lastScale || opacity > lastOpacity {
			t.Fatal("关闭透明度或缩放非单调")
		}
		lastScale, lastOpacity = scale, opacity
	}
	if lastScale != .92 || lastOpacity != 0 {
		t.Fatalf("关闭终点 %v/%v", lastScale, lastOpacity)
	}
}

func TestPreviewDecodedOriginalKeepsPlaceholderGeometry(t *testing.T) {
	a := previewFixture(800, 600)
	w := a.Preview.active
	p := &w.picture
	solid := func(width, height int) *ui.Bitmap {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.SetRGBA(x, y, color.RGBA{R: 180, G: 92, B: 40, A: 255})
			}
		}
		return ui.NewBitmap(img)
	}
	w.placeholder = solid(80, 60)
	p.Bitmap, p.pixelBitmap = nil, nil
	p.Loading = true
	p.initialized = false
	tt := ui.NewTester(previewTestView(a), 600, 500)
	before, ok := tt.AccessByRole(ui.RoleImage, w.asset.Name)
	if !ok {
		t.Fatal("没有缩略图占位")
	}
	generation, zoom, pan := p.Generation, p.Zoom, p.Pan
	pixels := append([]byte(nil), tt.Image().Pix...)
	original := solid(800, 600)
	p.finishPicture(generation, previewPicture{bitmap: original, pixelBitmap: original.Pixelated()}, nil)
	tt.Frame()
	after, ok := tt.AccessByRole(ui.RoleImage, w.asset.Name)
	if !ok || before.Bounds != after.Bounds || p.Generation != generation || p.Zoom != zoom || p.Pan != pan || !p.initialized {
		t.Fatal("原图替换重置了视口或图片目标矩形")
	}
	crop := image.Rect(int(before.Bounds.X+before.Bounds.W/4), int(before.Bounds.Y+before.Bounds.H/4), int(before.Bounds.X+before.Bounds.W*3/4), int(before.Bounds.Y+before.Bounds.H*3/4))
	old := &image.RGBA{Pix: pixels, Stride: tt.Image().Stride, Rect: tt.Image().Rect}
	for y := crop.Min.Y; y < crop.Max.Y; y++ {
		start, end := old.PixOffset(crop.Min.X, y), old.PixOffset(crop.Max.X, y)
		if !bytes.Equal(old.Pix[start:end], tt.Image().Pix[start:end]) {
			t.Fatal("原图替换发生中间空白或棋盘闪烁")
		}
	}
}

func TestPreviewMainKeyboardKeepsSelectionAndCanReopen(t *testing.T) {
	a := sidebarTestApp()
	a.Preferences.SpaceKey = "preview"
	tt := ui.NewTester(a.View, 1100, 640)
	if err := tt.Click("Landscape"); err != nil {
		t.Fatal(err)
	}
	tt.KeyDown(ui.KeySpace)
	first := a.Preview.active
	if first == nil || !first.picture.Held {
		t.Fatal("焦点卡片的Space按下没有进入按住预览")
	}
	tt.Advance(100 * time.Millisecond)
	tt.KeyUp(ui.KeySpace)
	if !a.PreviewOpen() || first.picture.Held || a.Preview.gesture != nil {
		t.Fatal("焦点卡片的真实KeyUp没有结束快按手势")
	}
	tt.Key(0, ui.KeyEscape)
	if a.PreviewOpen() || !a.Selected.Has("a") {
		t.Fatal("Escape关闭预览时清除了主窗口选中")
	}
	tt.KeyDown(ui.KeySpace)
	tt.KeyUp(ui.KeySpace)
	if !a.PreviewOpen() || a.Preview.active == first {
		t.Fatal("关闭后不能再次打开预览")
	}
	tt.KeyDown(ui.KeySpace)
	tt.KeyUp(ui.KeySpace)
	if a.PreviewOpen() {
		t.Fatal("第二次Space关闭后同一事件重开了预览")
	}
	tt.KeyDown(ui.KeySpace)
	tt.KeyUp(ui.KeySpace)
	if !a.PreviewOpen() {
		t.Fatal("下一次Space不能重新打开")
	}
}

func TestPreviewReconcilesKeyReleaseWithoutElementFocus(t *testing.T) {
	for _, hold := range []time.Duration{100 * time.Millisecond, 800 * time.Millisecond} {
		t.Run(hold.String(), func(t *testing.T) {
			a := readyPreviewApp()
			a.Preview = previewWindows{}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Text(c, "没有元素焦点")
				if c.Shortcut(0, ui.KeySpace) {
					a.ShowPreview("one", true)
				}
				a.ReconcilePreviewKeys(c)
			}, 400, 300)
			tt.KeyDown(ui.KeySpace)
			w := a.Preview.active
			if w == nil || !w.picture.Held {
				t.Fatal("未聚焦元素时没有记录窗口实际按住状态")
			}
			tt.Advance(hold)
			tt.KeyUp(ui.KeySpace)
			if a.Preview.gesture != nil || w.picture.Held {
				t.Fatal("实际KeyUp未释放没有元素焦点的预览手势")
			}
			if a.PreviewOpen() != (hold < previewPeekThreshold) {
				t.Fatal("没有元素焦点时快按/偷看分类错误")
			}
		})
	}
}

func TestPreviewFocusedTemporaryWindowEscapeAndCloseShortcut(t *testing.T) {
	for _, key := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{0, ui.KeyEscape}, {ui.Cmd, ui.KeyW}, {0, ui.KeySpace}} {
		a := previewFixture(288, 192)
		w := a.Preview.active
		w.focused = true
		tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 400, 300)
		tt.Key(key.mods, key.key)
		if !w.disposed || a.PreviewOpen() {
			t.Fatal("点击暂态窗口后Escape或关闭快捷键不能关闭")
		}
	}
}

func TestPreviewSmallFitNeverUpscalesPastTwo(t *testing.T) {
	a := previewFixture(64, 48)
	a.Preferences.PreviewFit = "fit"
	ui.NewTester(previewTestView(a), 300, 220)
	p := &a.Preview.active.picture
	if p.Zoom > 2 || p.fitZoom > 2 {
		t.Fatalf("小图初始缩放超过2倍：%v / %v", p.Zoom, p.fitZoom)
	}
	a.PreviewZoomIn()
	if p.Zoom <= 2 {
		t.Fatal("手动放大仍被初始适应上限拦住")
	}
}

func TestPreviewReducedMotionStillTimesTheHold(t *testing.T) {
	a := previewFixture(288, 192)
	w := a.Preview.active
	tt := ui.NewTesterWith(func(c *ui.Context) { w.view(c, a) }, 400, 300, ui.TesterOptions{Preferences: ui.Preferences{ReduceMotion: true}})
	w.pressedAt = tt.Now()
	w.picture.Held = true
	a.Preview.gesture = w
	tt.Frame()
	tt.Advance(previewPinThreshold)
	if !w.pinned {
		t.Fatal("减少动态效果禁用了长按固定计时")
	}
	w.close(a, tt.Now())
	if !w.disposed || len(a.Preview.pinned) != 0 {
		t.Fatal("减少动态效果仍延迟关闭")
	}
}

func TestPreviewRotationUsesItsOwnLibrary(t *testing.T) {
	ctx := context.Background()
	source := writeDirectionalPreview(t, false)
	roots := []string{filepath.Join(t.TempDir(), "first.momento"), filepath.Join(t.TempDir(), "second.momento")}
	var libraries []*library.Library
	var asset model.Asset
	for _, root := range roots {
		lib, err := library.Create(ctx, root, "图片")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := lib.Close(); err != nil {
				t.Error(err)
			}
		})
		libraries = append(libraries, lib)
		if _, err := assets.New(root, lib.DB).Import(ctx, model.ImportRequest{Paths: []string{source}}, nil); err != nil {
			t.Fatal(err)
		}
		snapshot, err := lib.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if asset.ID != "" && asset.ID != snapshot.Assets[0].ID {
			t.Fatal("相同图片应有相同素材身份")
		}
		asset = snapshot.Assets[0]
	}
	if _, err := rotatePreviewAsset(ctx, roots[0], asset, 90); err != nil {
		t.Fatal(err)
	}
	for i, lib := range libraries {
		snapshot, err := lib.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		expected := 90
		if i == 1 {
			expected = 0
		}
		if snapshot.Assets[0].Rotation != expected {
			t.Fatal("固定窗口旋转写到了另一个图库的同名素材")
		}
	}
}

func TestPreviewHoldHintFitsEnglishAndJapanese(t *testing.T) {
	for _, language := range []string{"en", "ja"} {
		a := previewFixture(64, 48)
		a.Texts.Language = language
		a.Preferences.Language = language
		w := a.Preview.active
		w.picture.Held = true
		w.pressedAt = time.Now().Add(-time.Second)
		var textWidth float32
		tt := ui.NewTester(func(c *ui.Context) {
			w.view(c, a)
			textWidth, _ = c.MeasureText(0, ui.Span{Text: a.T("preview_hold_to_pin"), Size: 12})
		}, 300, 220)
		if textWidth > 250 {
			t.Fatalf("%s固定提示宽%.1f，超过可用250", language, textWidth)
		}
		if !tt.HasText(a.T("preview_hold_to_pin")) {
			t.Fatal("缺少固定提示三语文案")
		}
		for _, element := range tt.AccessAll() {
			if element.Bounds.W > 0 && element.Bounds.H > 0 && (element.Bounds.X < 0 || element.Bounds.Y < 0 || element.Bounds.X+element.Bounds.W > 300.1 || element.Bounds.Y+element.Bounds.H > 220.1) {
				t.Fatalf("元素超出窗口：%+v", element)
			}
		}
	}
}

func TestPreviewPinFitsAndFollowsResizeUntilManualZoom(t *testing.T) {
	a := previewFixture(800, 600)
	w := a.Preview.active
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 400, 300)
	w.picture.Held = true
	w.pressedAt = tt.Now().Add(-previewPinThreshold)
	w.advanceHold(a, tt.Now())
	tt.Frame()
	if !w.picture.fitMode || w.picture.Zoom != w.picture.fitZoom {
		t.Fatal("固定后没有进入适应窗口模式")
	}
	initial := w.picture.Zoom
	tt.SetSize(700, 600)
	if w.picture.Zoom <= initial || w.picture.Zoom != w.picture.fitZoom {
		t.Fatal("固定窗口resize没有同步图片比例")
	}
	w.picture.zoomAt(1.25, w.picture.viewport.W/2, w.picture.viewport.H/2)
	manual := w.picture.Zoom
	tt.SetSize(800, 700)
	if w.picture.Zoom != manual || w.picture.fitMode {
		t.Fatal("手动缩放后resize覆盖了用户比例")
	}
}

func TestPreviewFollowsSelectionOutsideInitialSession(t *testing.T) {
	a := readyPreviewApp()
	w := a.Preview.active
	w.lastSelectionID = w.asset.ID
	a.Visible = []model.Asset{{ID: "new-a", Name: "新范围A"}, {ID: "new-b", Name: "新范围B"}}
	a.Selected.Add("new-a")
	a.selectedIndex = 0
	a.FollowPreviewSelection()
	if w.asset.ID != "new-a" || len(w.items) != 2 || w.items[0].ID != "new-a" {
		t.Fatal("跨范围跟随仍使用旧会话顺序")
	}
	w.move(a, 1)
	if w.asset.ID != "new-b" {
		t.Fatal("跨范围跟随后不能翻页")
	}
	a.FollowPreviewSelection()
	if w.asset.ID != "new-b" {
		t.Fatal("相同主窗选择把预览翻页拉回了原图")
	}
	a.Selected.Clear()
	a.FollowPreviewSelection()
	if !w.disposed {
		t.Fatal("范围清空选中没有关闭暂态")
	}
}

func TestPreviewMultiSelectionRefreshKeepsLastMainSelection(t *testing.T) {
	a := readyPreviewApp()
	w := a.Preview.active
	a.Visible = append(a.Visible, model.Asset{ID: "two", Name: "第二张"})
	a.assetByID = map[string]model.Asset{"one": a.Visible[0], "two": a.Visible[1]}
	w.items = append([]model.Asset(nil), a.Visible...)
	a.Selected.Add("one")
	a.Selected.Add("two")
	a.selectedIndex = -1
	w.mainSelection, w.lastSelectionID = []string{"one", "two"}, "one"
	w.move(a, 1)
	for range 100 {
		a.FollowPreviewSelection()
	}
	if w.asset.ID != "two" || w.lastSelectionID != "one" {
		t.Fatal("多选刷新因map顺序把预览翻页拉回")
	}
}

func TestPreviewHoverKeepsIndependentMainSelectionBaseline(t *testing.T) {
	for _, selected := range []bool{false, true} {
		a := readyPreviewApp()
		a.Preview = previewWindows{}
		a.Visible = append(a.Visible, model.Asset{ID: "two", Name: "悬停图片", Width: 200, Height: 150})
		if selected {
			a.Selected.Add("one")
			a.selectedIndex = 0
		}
		a.ShowPreview("two", false)
		w := a.Preview.active
		a.FollowPreviewSelection()
		if w.disposed || w.asset.ID != "two" {
			t.Fatal("打开悬停图片被既有选择或空选择立刻覆盖")
		}
		if selected {
			a.Selected.Clear()
			a.FollowPreviewSelection()
			if !w.disposed {
				t.Fatal("真实清除选择后暂态没有关闭")
			}
		} else {
			a.Selected.Add("one")
			a.FollowPreviewSelection()
			if w.asset.ID != "one" {
				t.Fatal("空基准之后的真实选择变化没有跟随")
			}
		}
	}
}

func TestPreviewPinsHoveredAssetWithoutChangingSelection(t *testing.T) {
	a := readyPreviewApp()
	a.Preview = previewWindows{}
	a.Visible = append(a.Visible, model.Asset{ID: "two", Name: "悬停图片", Width: 200, Height: 150})
	a.Selected.Add("one")
	a.selectedIndex = 0
	a.ShowPreview("two", true)
	w := a.Preview.active
	a.FollowPreviewSelection()
	w.advanceHold(a, w.pressedAt.Add(previewPinThreshold))
	if !w.pinned || w.asset.ID != "two" || !a.Selected.Has("one") || a.Selected.Has("two") {
		t.Fatal("长按没有固定悬停图片，或修改了主窗选中")
	}
}

func TestPreviewSpaceWithoutTargetDoesNotKeepOldHoldStart(t *testing.T) {
	a := sidebarTestApp()
	tt := ui.NewTester(a.View, 1100, 640)
	tt.KeyDown(ui.KeySpace)
	tt.KeyUp(ui.KeySpace)
	if a.PreviewOpen() || !a.Preview.spaceDown.IsZero() {
		t.Fatal("无目标的Space留下了按住起点")
	}
	tt.Advance(4 * time.Second)
	card, ok := tt.AccessByRole(ui.RoleButton, "Portrait")
	if !ok {
		t.Fatal("缺少悬停素材")
	}
	tt.Move(card.Bounds.X+card.Bounds.W/2, card.Bounds.Y+card.Bounds.H/2)
	tt.KeyDown(ui.KeySpace)
	w := a.Preview.active
	if w == nil || !w.pressedAt.Equal(tt.Now()) || w.pinned {
		t.Fatal("新的悬停预览沿用了旧按下时间")
	}
	tt.Advance(time.Second)
	tt.KeyUp(ui.KeySpace)
	if a.PreviewOpen() || len(a.Preview.pinned) != 0 {
		t.Fatal("不足3秒的偷看被旧时间误固定")
	}
}

func TestPreviewFocusedWindowShortcutsWithoutElementFocus(t *testing.T) {
	a := previewFixture(288, 192)
	w := a.Preview.active
	w.focused = true
	bitmap, pixels := w.picture.Bitmap, w.picture.pixelBitmap
	w.items = append(w.items, model.Asset{ID: "two", Name: "第二张", Width: 288, Height: 192})
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 600, 400)
	tt.Key(0, ui.KeyRight)
	if w.asset.ID != "two" {
		t.Fatal("独立窗口无元素焦点时不能下一张")
	}
	tt.Key(0, ui.KeyLeft)
	if w.asset.ID != "one" {
		t.Fatal("独立窗口无元素焦点时不能上一张")
	}
	w.picture.Bitmap, w.picture.pixelBitmap = bitmap, pixels
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyEqual)
	if w.picture.Zoom != 1.25 {
		t.Fatal("独立窗口无元素焦点时不能放大")
	}
	tt.Key(ui.Cmd, ui.KeyMinus)
	if w.picture.Zoom != 1 {
		t.Fatal("独立窗口无元素焦点时不能缩小")
	}
	tt.Key(ui.Cmd, ui.Key0)
	if w.picture.Zoom != w.picture.fitZoom || !w.picture.fitMode {
		t.Fatal("独立窗口无元素焦点时不能适应窗口")
	}
	tt.Key(ui.Cmd, ui.Key1)
	if w.picture.Zoom != 1 || w.picture.fitMode {
		t.Fatal("独立窗口无元素焦点时不能原始大小")
	}
}
