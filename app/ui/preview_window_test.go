package ui

import (
	"bytes"

	"image"
	"image/color"
	"strings"
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
	first.hasFocus = true
	if !a.PreviewOpen() || a.PreviewAssetID() != "one" {
		t.Fatal("焦点固定窗口不是菜单目标")
	}
	first.hasFocus = false
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
		r := previewLayout(parent, area, asset, true, "zh-CN", 1)
		// 图片最多占屏幕可用区域 85% 宽、85% 高，留白和底栏另算。
		if r.Width-2*previewImageInset > 1224 || r.Height-previewImageInset-previewFooterPoints(asset, r.Width, "zh-CN", 1) > 765 || r.X < area.X || r.Y < area.Y || r.X+r.Width > area.Width || r.Y+r.Height > area.Height {
			t.Fatalf("窗口越界：%+v", r)
		}
		if r.X+r.Width/2 < parent.X+parent.Width/2-1 || r.X+r.Width/2 > parent.X+parent.Width/2+1 {
			t.Fatalf("预览没有在主窗口中央：%+v", r)
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
	p.Bitmap = nil
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
	p.finishPicture(generation, previewPicture{bitmap: original}, nil)
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
		w.hasFocus = true
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

// 按住空格时不加文字提示，只把关闭按钮换成小圆环；尺寸信息照常显示，元素不超出窗口。
func TestPreviewHoldShowsOnlyRing(t *testing.T) {
	a := previewFixture(64, 48)
	w := a.Preview.active
	w.picture.Held = true
	w.pressedAt = time.Now().Add(-time.Second)
	const width, height = previewMinimumWidth, 240
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, width, height)
	meta, _ := previewDetails(w.asset)
	if !tt.HasText(meta) {
		t.Fatalf("按住时尺寸信息不见了：%v", tt.Texts())
	}
	if _, found := tt.AccessByRole(ui.RoleButton, a.T("close_preview")); found {
		t.Fatal("按住时关闭按钮没有换成圆环")
	}
	for _, element := range tt.AccessAll() {
		if element.Bounds.W > 0 && element.Bounds.H > 0 && (element.Bounds.X < 0 || element.Bounds.Y < 0 || element.Bounds.X+element.Bounds.W > width+.1 || element.Bounds.Y+element.Bounds.H > height+.1) {
			t.Fatalf("元素超出窗口：%+v", element)
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
	if w.asset.ID != "new-a" || len(w.items) != 2 || w.items[0].id != "new-a" {
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
	w.items = previewOrder(a.Visible)
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
	w.hasFocus = true
	bitmap := w.picture.Bitmap
	two := model.Asset{ID: "two", Name: "第二张", Width: 288, Height: 192}
	w.items = append(w.items, previewItem{id: two.ID})
	a.Visible = append(a.Visible, two) // 翻页只走主窗口列表里还在的图
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 600, 400)
	tt.Key(0, ui.KeyRight)
	if w.asset.ID != "two" {
		t.Fatal("独立窗口无元素焦点时不能下一张")
	}
	tt.Key(0, ui.KeyLeft)
	if w.asset.ID != "one" {
		t.Fatal("独立窗口无元素焦点时不能上一张")
	}
	w.picture.Bitmap = bitmap
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

func TestPreviewFooterKeepsSameDetailsOnHover(t *testing.T) {
	asset := model.Asset{Name: "晨雾", Extension: "JPG", Size: 3 << 20, Width: 4000, Height: 3000,
		EXIF: map[string]string{"cameraMake": "SONY", "cameraModel": "ILCE-7M4", "focalLength": "35"}}
	if got := previewFileName(asset); got != "晨雾.jpg" {
		t.Fatalf("文件名没带扩展名：%q", got)
	}
	// 导入只去掉最后一个扩展名，logo.png.png 的名称是 logo.png。
	if got := previewFileName(model.Asset{Name: "logo.png", Extension: "png"}); got != "logo.png.png" {
		t.Fatalf("文件名少了扩展名：%q", got)
	}
	dimensions, size := previewDetails(asset)
	if dimensions != "4000 × 3000" || size != "3.0 MB" {
		t.Fatalf("信息行：%q / %q", dimensions, size)
	}

	a := previewFixture(400, 300)
	w := a.Preview.active
	w.asset.Name, w.asset.Extension, w.asset.EXIF = asset.Name, asset.Extension, asset.EXIF
	tt := ui.NewTester(func(c *ui.Context) { w.view(c, a) }, 428, 366)
	tt.Frame()
	before := strings.Join(tt.Texts(), "|")
	if !tt.HasText("晨雾.jpg") {
		t.Fatalf("没有文件名：%v", tt.Texts())
	}
	tt.Move(100, 366-previewFooterMinimum/2)
	tt.Advance(100 * time.Millisecond)
	if after := strings.Join(tt.Texts(), "|"); after != before || tt.HasText("ILCE-7M4") {
		t.Fatalf("悬停底栏后第二行变了：%q → %q", before, after)
	}
}

func TestPreviewLongNameWrapsAndGrowsWindow(t *testing.T) {
	parent, area := mygo.Rectangle{Width: 1200, Height: 800}, mygo.Rectangle{Width: 1440, Height: 900}
	short := model.Asset{Name: "a", Extension: "png", Width: 100, Height: 80}
	long := short
	long.Name = "一个特别特别长的文件名用来确认预览底栏会把名字完整换行显示而不是截断"
	s, l := previewLayout(parent, area, short, false, "zh-CN", 1), previewLayout(parent, area, long, false, "zh-CN", 1)
	if s.Height != previewMinimumHeight+previewImageInset+previewFooterPoints(short, s.Width, "zh-CN", 1) || l.Height <= s.Height || l.Width != s.Width {
		t.Fatalf("长文件名没有让底栏变高：短 %+v，长 %+v", s, l)
	}
}

// 换图后窗口变高：贴着屏幕底边时整体上移；界面放大时按缩放后的宽度换行、按缩放换算高度。
func TestPreviewResizeStaysOnScreenAndFollowsZoom(t *testing.T) {
	area := mygo.Rectangle{Y: 25, Width: 1440, Height: 875}
	r := previewResized(mygo.Rectangle{X: 100, Y: 700, Width: 300, Height: 200}, 30, area)
	if r.Height != 230 || r.Y+r.Height != area.Y+area.Height {
		t.Fatalf("变高后超出屏幕：%+v", r)
	}
	if r := previewResized(mygo.Rectangle{Y: 100, Height: 200}, 30, area); r.Y != 100 || r.Height != 230 {
		t.Fatalf("不靠底边时不该移动：%+v", r)
	}
	long := model.Asset{Name: "一个比较长的文件名在放大界面后会多换一行", Extension: "png"}
	normal, zoomed := previewFooterPoints(long, 300, "zh-CN", 1), previewFooterPoints(long, 300, "zh-CN", 1.5)
	if zoomed < int(float32(normal)*1.5) {
		t.Fatalf("界面放大后底栏没有按缩放变高：1x %d，1.5x %d", normal, zoomed)
	}
}

// 小屏幕上图片也不超过屏幕可用区域的 85%。
func TestPreviewLayoutRespectsSmallScreen(t *testing.T) {
	area := mygo.Rectangle{Width: 400, Height: 900}
	r := previewLayout(mygo.Rectangle{Width: 400, Height: 900}, area, model.Asset{Name: "a", Extension: "png", Width: 4000, Height: 1000}, false, "zh-CN", 1)
	if image := r.Width - 2*previewImageInset; image > 340 {
		t.Fatalf("图片宽 %d，超过 400 的 85%%", image)
	}
}
