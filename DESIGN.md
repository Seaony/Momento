# 界面规范

## 视觉

安静、紧凑的桌面图片工作区。图片是视觉中心，浅色是几乎白的浅暖橙，深色延续暖深色。保留 Momento 自己的布局、Phosphor 和磨砂，品牌橘只承担明确的交互含义。

## 色彩

品牌色来自暖橘底彩虹刺猬图标，只保留浅色、深色两套色板。「跟随系统」在这两套间切换。全部窗口、浮层、菜单、锁屏和引导共享色板。

| 用途 | 浅色 | 深色 |
| --- | --- | --- |
| Background 内容底 | #FFFCFA | #1C1714 |
| Sidebar 侧栏底 | #FFF7F1 | #231C17 |
| Raised / Selected 浮起面 | #FFFFFF | #31271F |
| SegmentSelected 分段选中块 | #FFFFFF | #4D3D31 |
| Subtle 预览容器、药丸、分段底 | #FDF1E8 | #2A211B |
| Hover / Pressed | #FBECE0 / #F7E2D2 | #382B22 / #433126 |
| Text 正文与标题 | #2B221D | #F6EEE7 |
| TextSecondary 次要信息 | #6F5F55 | #CFBCAE |
| TextTertiary 静态字段标签 | #7D6B60 | #B29886 |
| Accent / Hover / Pressed | #F47B45 / #FA8B59 / #E8733D | #FF9A66 / #FFAB7E / #F48A54 |
| OnAccent 主按钮文字 | #2B221D | #2B221D |
| Focus / Link | #D9622C / #A74320 | #FF9A66 / #FFB48F |
| BorderSubtle / Border（不透明） | #F3E3D6 / #EAD6C6 | #3A2D24 / #4B3A2E |
| 系统侧栏材质着色 | Sidebar × 99% | Sidebar × 86% |
| Scrim 暖深遮罩 | Text × 25% | Background × 60% |

分隔线和中性色边框一律不透明，包括材质上的竖线、横线、卡片边框和虚线入口；框架 Theme.Border 映射 BorderSubtle，让 Split、侧栏底线和工具栏下沿同色。需要更明确的输入边框时用语义 Border，不给边线叠透明度。

控件悬停、按下统一用 Hover / Pressed。正文、次要文字、主按钮及静态字段标签在各自表面上的对比度不低于 4.5:1；浅色三级标签比参考起点略深，保证小字可读。危险操作保留独立危险色。界面缩放支持 80%、90%、100%、110%、125%、150%，所有窗口同步。

品牌橘只用于主按钮、当前导航图标、卡片选中描边、焦点环、链接、开关打开态和滑块已选段；选中行文字用 Text、底用 Raised。浮起面用 1px BorderSubtle 与暖色轻影（Y 1、模糊 2，浅色 7%），通过 elevated helper 统一；磨砂调用方保留自己的材质，不被实色背景覆盖。

旧偏好读取时把 light-gray 转为 light，把 dark-gray、navy、purple 转为 dark。它们不再作为选项或独立色板存在。

## 字体

内置 Inter 400 / 500 / 600 / 700 与 Geist Mono 400，保留原始 SIL OFL。正文 13、次要信息 12，分组标题 11、半粗 600、稍加字距；独立成列的数量、尺寸、日期、大小、版本号用 monoFont(c)，11.5～12；句子里的数字保留正文字体。中文按平台回退苹方、微软雅黑或思源 / Noto Sans；日文界面优先日文字体。正文家族由 applyTheme 设置，数值显式使用等宽家族，不缩小文字挤空间。

## 控件

间距使用 space4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64；圆角使用 radiusSmall 4、radiusCard 6、radiusControl 8、radiusPanel 12。用户要求的圆形按钮和胶囊仍按完整圆形绘制。素材图片圆角 6、轻暖影、悬停轻微上浮；选中橘色 2px 描边留缝。数量右对齐。

全应用图标使用 Phosphor Icons（MIT），普通 Regular、侧栏当前选中项 Fill；颜色由 currentColor 继承。普通次要文字色、悬停正文色、选中主题色、主按钮内跟随按钮文字色。侧栏、工具栏、设置导航 18，菜单及检查器 16，小操作 14–16；不自绘、不混入其他图标库。SVG 来源和许可保留在 app/ui。

主题选择在设置与引导共用三张带各自名称的预览卡，选中卡用品牌色描边。所有分段控件共用 SegmentSelected 凸起底块，比 Subtle 底槽明显亮一档，选中底块平滑移动。主按钮禁用时使用 Surface 底和 TextMuted 字色，不对整块按钮降低透明度，输入与无障碍语义仍为禁用。

| 功能 | 官方图标名 |
| --- | --- |
| 全部素材 / 素材预览 | Images (`images`) |
| 最近使用 | ClockCounterClockwise (`clock-counter-clockwise`) |
| 已收藏 | Heart (`heart`) |
| 随机浏览 | Shuffle (`shuffle`) |
| 未分类 | Tray (`tray`) |
| 未打标签 / 标签 | TagSimple (`tag-simple`) |
| 标签管理 | Tag (`tag`) |
| 文件夹 | Folder (`folder`) |
| 智能文件夹 | FolderSimpleStar (`folder-simple-star`) |
| 回收站 | Trash (`trash`) |
| 设置 / 通用 | GearSix (`gear-six`) |
| 帮助 | Question (`question`) |
| 导入 | DownloadSimple (`download-simple`) |
| 导出 | UploadSimple (`upload-simple`) |
| 筛选 | FunnelSimple (`funnel-simple`) |
| 排序 | SortAscending (`sort-ascending`) |
| 瀑布流 | SquaresFour (`squares-four`) |
| 网格 | GridFour (`grid-four`) |
| 列表 / 菜单 | List (`list`) |
| 搜索 | MagnifyingGlass (`magnifying-glass`) |
| 缩略图大小 | FrameCorners (`frame-corners`) |
| 侧栏开关 | SidebarSimple (`sidebar-simple`) |
| 收起检查器 | CaretDoubleRight (`caret-double-right`) |
| 来源链接 | LinkSimple (`link-simple`) |
| 打开外部来源 | ArrowSquareOut (`arrow-square-out`) |
| 图库 | ImagesSquare (`images-square`) |
| 新建图库 | FolderPlus (`folder-plus`) |
| 密码锁 | LockSimple (`lock-simple`) |
| 操控 | MouseSimple (`mouse-simple`) |
| 快捷键 | Keyboard (`keyboard`) |
| 通知 | Bell (`bell`) |
| 外观 | Palette (`palette`) |
| 浏览器扩展设置 | Globe (`globe`) |
| 关于 / 信息 | Info (`info`) |
| 显示 / 隐藏密码 | Eye / EyeSlash (`eye` / `eye-slash`) |
| 编辑 | PencilSimple (`pencil-simple`) |
| 扩展安装 | PuzzlePiece (`puzzle-piece`) |
| 警告 | Warning (`warning`) |
| 左右翻页 | CaretLeft / CaretRight (`caret-left` / `caret-right`) |
| 展开 / 收起 | CaretDown / CaretUp (`caret-down` / `caret-up`) |
| 上下双箭头 | CaretUpDown (`caret-up-down`) |
| 新增 / 移除 | Plus / Minus (`plus` / `minus`) |
| 关闭 / 勾选 | X / Check (`x` / `check`) |
| 左右旋转 | ArrowCounterClockwise / ArrowClockwise (`arrow-counter-clockwise` / `arrow-clockwise`) |

动画共用约 250ms、轻微回弹的弹簧，颜色过渡 120ms；交给 mygo 自动遵守系统减少动态效果，不增加动画开关。浮层从锚点方向淡入并从 0.96 放大，关闭反向；对话框背景淡入，面板轻微上移缩放。

素材只有在视图形状和列数不变的筛选、搜索、排序、增删中，才按稳定身份移动可见卡片，屏幕外直接就位。切换视图时旧视图约 120ms 整体淡出，新视图约 180ms 整体淡入并从 0.98 放大到 1，不把图片卡变形成列表行。侧栏、检查器、窗口宽度及缩略图尺寸变化时，卡片直接跟随实际布局，列数跳变以 60ms 退场、90ms 入场，不逐卡飞行。栏宽保留弹簧前段，在 250ms 终点收准；搜索框不叠加第二层位移动画。100% 帧必须完全落定，旧视图无残影。性能用 12000 张样本与 mygo 帧统计测量；只记录超过阈值的日志时，不据此推算平均帧时间。

## 布局

默认 1100×800，最小 1100×640。侧栏默认 236，可在 196–340 间拖宽；检查器 280。间距采用 4、8、12、16、24。工具栏单行高 60，覆盖滚动内容并模糊背后的图片。窄中栏把搜索和视图切换收进浮层，所有功能仍可操作。搜索默认 ⌘K / Ctrl+K，通过统一快捷键设置改绑、禁用和冲突检查；提示显示实际绑定。侧栏导航按全部、最近使用、已收藏、随机浏览、整理、文件夹、智能文件夹排列；空分组显示虚线入口。

## 层次

主窗口与设置窗口使用系统侧栏材质，浅色加接近白的浅暖橙着色、深色加暖深色着色。Linux 无系统材质时用语义 Sidebar 实色。工具栏、浮层保留背景模糊，暖色遮罩保持文字清晰；减少透明度时使用不透明表面。

分栏用细分隔线和表面亮度区分；图片卡片只加柔和轻影。浮层保留明确的关闭入口。确认按钮写具体动作，遮罩用 Scrim 暖深色半透明。

检查器底与内容区同为 Background，预览容器用 Subtle，输入框用 Border；备注与来源空值显示 Tertiary 占位。非编辑来源链接单行省略，编辑时滚动并保留完整值。标签、文件夹药丸用 12px Phosphor X 移除；备注组前使用同样的浅分隔线。

## 规则

一个元素只表达一个意思；无装饰背景、无夸张动效；导入进度与素材计数分开；隐藏没有数据的 EXIF 字段。

## 窗口适配

缩小窗口时内容区自适应列数，侧栏和检查器可收起；不牺牲文字可读性。浅深色、中文英文都检查默认和最小窗口。

## 实现

所有界面使用 mygo/ui；共享布局与主题入口保持统一；数据读写在后台，结果通过 Window.Update 进入主线程。

## 后续

仅记录：用户自选强调色、更多动效，本轮不实现。

固定预览窗口不跨重启保留，恢复窗口位置和内容留待以后。

## 独立预览窗口

空格按下立即以普通无边框窗口非激活显示，键盘留在主窗口。小于 250ms 松开时保留，再按空格或 Esc 关闭；按住 250ms 到 3 秒之间松开时缩小淡出；满 3 秒把同一窗口置顶固定。进度提示在 250ms 后淡入，固定后淡出并轻弹。固定窗口独立拖动、缩放，可同时存在多张，关闭带动画；主窗口选择变化只跟随暂态预览。

空格优先预览悬停素材，没有悬停才使用选中项；预览不改变选择和检查器。打开时保存主窗口选择的基准，只有选择真的改变才跟随，不能把悬停图切回原选中项。翻页从当前预览图开始，悬停打开也支持长按固定。

窗口使用磨砂背景、12 圆角和柔和投影，打开和关闭做 0.92 到 1 的缩放及淡入淡出。初始大小不超过主窗口的 80%，小图自动放大不超过 2 倍。原图解码前使用缓存缩略图，替换时保持图片位置。旋转和暂态关闭按钮为 32 可点区域、18 图标；翻页按钮为 36，离左右边缘 16、垂直居中；固定窗口悬停关闭按钮为 28。


预览信息胶囊与窗口边缘留 12，胶囊到图片留 12。小图在左右各留 64 容纳图外翻页按钮；大图才允许磨砂按钮叠在图片上。初始与固定窗口最小宽度为 400，进入动画保留临时缩放空间。
