package ui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// 全应用的动效只有这几档，同一类变化用同一档：
//   - motionColor：悬停、按下、选中的颜色，和小东西的显隐（勾、行尾按钮），150ms 缓出。跟手的反馈要快，按下去马上有回应。
//   - motionSpring：位置、大小、展开收起、指示块滑动、箭头旋转、数字滚动，约 400ms 的轻弹簧，中途再操作会接着当前速度走。
//     改变布局的动作不能快：太快只看到一下跳变，看不清东西从哪来、到哪去，显得慌。
//   - fadeMotion：东西在原地出现、消失，或整块内容换成另一块（换位置、换页、换设置页、检查器换图），260ms 淡入、180ms 淡出。
//   - revealMotion：一行提示、一块可选内容在原地展开收起，下面的内容跟着滑开。
//   - popupMotion、dialogMotion：弹层和对话框；收起、关闭一律 160～180ms 缓入，比出现快。
//
// 箭头旋转、行尾按钮显隐优先用只重绘的 AnimatePose，不让整个窗口每帧重建。
var motionSpring = ui.Animation{Spring: ui.SpringOptions{Duration: motionLayout, Bounce: .04}}
var motionColor = ui.Animation{Duration: 150 * time.Millisecond}

const (
	// motionLayout：位置、大小变化的时长。弹簧、卡片重排、栏宽展开都用它，侧栏展开时卡片和栏宽同时到位。
	motionLayout = 400 * time.Millisecond
	motionEnter  = 260 * time.Millisecond
	motionExit   = 180 * time.Millisecond
)

// 数字变了只滚动变的那几位：增加从下往上滚，减少传 down 从上往下滚。
func numberMotion(down bool) ui.ContentTransition {
	return ui.ContentTransition{Numeric: true, Down: down, Animation: &motionSpring}
}

// countMotion 给显示数量的文字加滚动：记住上次的数，变大从下往上滚，变小从上往下滚。
func countMotion(e ui.Element, n int) ui.Element {
	last := ui.Local(e, "count-motion", func() struct {
		n    int
		down bool
	} {
		return struct {
			n    int
			down bool
		}{n: n}
	})
	if n != last.n {
		last.down, last.n = n < last.n, n
	}
	return e.ContentTransition(numberMotion(last.down))
}

// 图片换成同一张的清晰版（缩略图换原图）：和淡入一样长的交叉淡化，只重绘。
var pictureMotion = ui.ContentTransition{Animation: &ui.Animation{Duration: motionEnter}}

// 文字或图标换成别的：旧的淡出、新的淡入（图标会轻轻缩放），只重绘不重排。
var replaceMotion = ui.ContentTransition{Animation: &motionSpring}

// 后台读好的图片（缩略图、预览图）到了就淡入。只在图片晚于它所在的卡片出现时才播：
// 卡片建出来时图片已经缓存好（滚进视口、切换页面）就直接显示。
// 用只重绘的 Pose 淡入，不用元素过渡：缩略图陆续到达时不让整个素材区每帧重建。
type imageArrival struct {
	ready bool
	at    time.Time
}

// track 记下图片这一帧有没有，返回是否正在淡入；淡入期间调用方在图片下面继续画占位底，免得露出底色。
func (s *imageArrival) track(c *ui.Context, ready bool) bool {
	if ready && !s.ready {
		s.at = c.Now()
	}
	s.ready = ready
	if !ready || s.at.IsZero() {
		return false
	}
	left := motionEnter + 20*time.Millisecond - c.Now().Sub(s.at)
	if left <= 0 {
		s.at = time.Time{}
		return false
	}
	c.After(left)
	return true
}

// pose 让图片从透明淡入：到达那一帧先摆在全透明，下一帧起交给只重绘的淡入。
func (s *imageArrival) pose(c *ui.Context, image ui.Element) ui.Element {
	pose := ui.Pose{}
	if c.Now().Equal(s.at) {
		pose.Fade = 1
		c.Invalidate()
	}
	return image.AnimatePose(pose, ui.Animation{Duration: motionEnter})
}

// arrival 取元素上记着的图片到达状态。
func arrival(holder ui.Element, key string, ready bool) *imageArrival {
	return ui.Local(holder, key, func() imageArrival { return imageArrival{ready: ready} })
}

// 输错时整块左右轻抖一下，和 macOS 登录框一样；run 每多一次失败就重播。没失败过不调用，免得第一次显示就抖。
// 减少动态效果时 mygo 直接停在最后一帧（0），不抖。
func shakeMotion(e ui.Element, run uint64) {
	x := e.Keyframes("shake", []ui.Keyframe{
		{Value: 0},
		{At: 60 * time.Millisecond, Value: -8, Ease: ui.EaseOut},
		{At: 140 * time.Millisecond, Value: 7, Ease: ui.EaseInOut},
		{At: 210 * time.Millisecond, Value: -4, Ease: ui.EaseInOut},
		{At: 270 * time.Millisecond, Value: 2, Ease: ui.EaseInOut},
		{At: 320 * time.Millisecond, Value: 0, Ease: ui.EaseOut},
	}, ui.KeyframeOptions{Run: run})
	e.Translate(x, 0)
}

// 进度条：mygo 的进度条每次更新直接跳到新值，这里在两次更新之间用 250ms 缓出补上，只重绘不重建界面。
// 倒退（换了一批任务）直接跳，减少动态效果时也直接跳。
type progressMotion struct {
	from, to float32
	changed  time.Time
}

func smoothProgress(c *ui.Context, value float64) ui.Element {
	bar := ui.Progress(c, value)
	if value < 0 {
		return bar
	}
	state := ui.Local(bar, "progress-motion", func() progressMotion { return progressMotion{from: float32(value), to: float32(value)} })
	now, target := c.Now(), float32(min(value, 1))
	if target != state.to {
		shown := state.at(now)
		if target < shown || c.Preferences().ReduceMotion {
			shown = target
		}
		state.from, state.to, state.changed = shown, target, now
	}
	// 轨道是进度条自己的底色，这里只画填充。
	fill := c.Theme().Accent
	bar.Draw(func(pt *ui.Painter, r ui.Rect) {
		shown := state.at(pt.Now())
		if shown != state.to {
			pt.AnimationFrame()
		}
		pt.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W * shown, H: r.H}, fill, r.H/2)
	})
	return bar
}

func (m *progressMotion) at(t time.Time) float32 {
	elapsed := t.Sub(m.changed)
	if m.changed.IsZero() || elapsed >= assetReflowDuration {
		return m.to
	}
	return m.from + (m.to-m.from)*assetReflowEase(float32(elapsed)/float32(assetReflowDuration))
}

func elevated(e ui.Element, t *ui.Theme) ui.Element {
	p := paletteFor(t)
	return e.Border(1, p.BorderSubtle).Shadow(0, 1, 2, 0, p.Shadow)
}

type paneWidthMotion struct {
	target    float32
	changedAt time.Time
}

// instant 为真时（拖动分隔条）直接跳到目标，跟手；展开收起才走弹簧。
func animatePaneWidth(c *ui.Context, e ui.Element, key string, target float32, instant bool) float32 {
	state := ui.Local(e, key+"-target", func() paneWidthMotion { return paneWidthMotion{target: target} })
	if instant {
		state.target, state.changedAt = target, time.Time{}
		return e.Jump(key, target)
	}
	if state.target != target {
		state.target, state.changedAt = target, c.Now()
	}
	// 弹簧的感知时长不等于结束时间，栏宽在约定终点收准，避免继续重排。
	if state.changedAt.IsZero() || c.Now().Sub(state.changedAt) >= motionLayout {
		return e.Jump(key, target)
	}
	value := e.AnimateTo(key, target, motionSpring)
	if !e.Settled(key) {
		c.After(motionLayout - c.Now().Sub(state.changedAt))
	}
	return value
}

func animateColors(e ui.Element, background, text ui.Color) {
	e.Background(e.AnimateColor("surface-color", background, motionColor)).TextColor(e.AnimateColor("text-color", text, motionColor))
}

// 弹层：220ms 减速出现、160ms 收起，不用弹簧，免得尾巴拖长。
func popupMotion(e ui.Element) ui.Element {
	return e.AttachedOrigin().Transition(ui.ElementTransition{Duration: 220 * time.Millisecond, ExitDuration: 160 * time.Millisecond, ExitEase: ui.EaseIn, Position: true,
		Enter: &ui.Motion{Scale: .98, Y: -4, Flip: true}, Exit: &ui.Motion{Scale: .98, Y: -4, Flip: true}})
}

// 对话框弹簧出现，关闭走 motionExit 缓入，比出现快，按了取消就收走。
func dialogMotion(backdrop, panel ui.Element) {
	backdrop.Transition(ui.ElementTransition{Animation: &motionSpring, ExitDuration: motionExit, ExitEase: ui.EaseIn, Colors: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
	panel.Transition(ui.ElementTransition{Animation: &motionSpring, ExitDuration: motionExit, ExitEase: ui.EaseIn, Position: true,
		Enter: &ui.Motion{Scale: .96, Y: 10}, Exit: &ui.Motion{Scale: .97, Y: 6}})
}

// 只淡入淡出，不移动、不缩放。整块换内容时新旧短暂交叠，旧的走得更快，来回快切也不拖残影。
// 只选 Shadows 是为了让过渡只管进出场：不碰位置大小，也不把按钮自己用 AnimateColor 做的悬停变色再平滑一遍。
func fadeMotion(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Duration: motionEnter, ExitDuration: motionExit, Shadows: true, Enter: &ui.Motion{}, Exit: &ui.Motion{}})
}

// 列表里的一行：新加的展开出来、删掉的收起，其余各行滑到新位置让位。
func rowMotion(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Animation: &motionSpring, Position: true, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}})
}

// 在原地展开收起：高度从 0 长出来（内容裁切、同时淡入），收起时反过来，后面的内容跟着滑。
func revealMotion(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Animation: &motionSpring, Height: true, Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}})
}

func layoutMotion(e ui.Element) ui.Element {
	return e.Transition(ui.ElementTransition{Animation: &motionSpring, Position: true, Size: true,
		Enter: &ui.Motion{Scale: .98}, Exit: &ui.Motion{Scale: .98}})
}

// showPose 只重绘地淡入淡出小东西（勾、行尾按钮、控制条）：shown 为假时全透明。
func showPose(e ui.Element, shown bool) ui.Element {
	fade := float32(1)
	if shown {
		fade = 0
	}
	return e.AnimatePose(ui.Pose{Fade: fade}, motionColor)
}

// turnPose 只重绘地转展开箭头。
func turnPose(e ui.Element, degrees float32) ui.Element {
	return e.AnimatePose(ui.Pose{Rotate: degrees}, motionSpring)
}

func motionTarget(on bool) float32 {
	if on {
		return 1
	}
	return 0
}

func (a *App) noticeViewport(c *ui.Context) {
	width, _ := c.Size()
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Padding(space24).AlignItems(ui.Center).Gap(space8)
		for _, notice := range toasts {
			parts := ui.ToastBase(c, notice)
			parts.Root.Row().AlignItems(ui.Center).Gap(space12).Padding(space12, space16).Radius(radiusPanel).
				MaxWidth(width-48).Background(panelColor(c.Theme())).TextColor(c.Theme().Text).
				Border(1, paletteFor(c.Theme()).BorderSubtle).Shadow(0, 1, 2, 0, paletteFor(c.Theme()).Shadow).
				Transition(ui.ElementTransition{Animation: &motionSpring, ExitDuration: motionExit, ExitEase: ui.EaseIn, Position: true, Enter: &ui.Motion{Y: 12}, Exit: &ui.Motion{Y: 8}})
			parts.Root.Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Gap(space4).Children(func() {
					ui.Text(c, notice.Title)
					if notice.Description != "" {
						ui.Text(c, notice.Description).TextColor(c.Theme().TextMuted)
					}
				})
				if notice.Action != "" {
					parts.ActionButton().Padding(space4, space8).Children(func() { ui.Text(c, notice.Action) })
				}
				close := parts.CloseButton().Label(a.T("close")).Tooltip(a.T("close")).Size(space24, space24).Radius(radiusCard).Center()
				background, text := ui.Transparent, c.Theme().TextMuted
				if close.Hovered() {
					background, text = c.Theme().SurfaceHover, c.Theme().Text
				}
				animateColors(close, background, text)
				close.Children(func() { appSymbol(c, "x") })
			})
		}
	})
}
