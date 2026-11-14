# 界面规范

## 视觉

安静、紧凑的桌面图片工作区。图片是视觉中心，工具栏、侧栏和检查器使用暖奶油或暖棕表面，选中态用统一的暖橘品牌色。

## 色彩

品牌色来自暖橘底彩虹刺猬图标，只保留浅色、深色两套色板。「跟随系统」在这两套间切换。全部窗口、浮层、菜单、锁屏和引导共享色板。

| 用途 | 浅色 | 深色 |
| --- | --- | --- |
| 主体底色 | #FFFAF5 | #211A16 |
| 控件表面 | #FBF0E5 | #2C231E |
| 悬停 / 按下表面 | #F5EBE0 / #EEE2D6 | #382D26 / #44362D |
| 正文 / 标题 | #3A2318 / #2E1B12 | #F8EEE5 / #FFF7EF |
| 次要文字 | #7E5E4C | #D0B49F |
| 品牌色 / 悬停 / 按下 | #F47B45 / #FA8B59 / #E8733D | #FF9A66 / #FFAC80 / #F48A54 |
| 主按钮文字 | #3A2318 | #2E1B12 |
| 焦点 / 链接 | #D9622C / #9E3514 | #FF9A66 / #FFB48F |
| 边线（不透明） | #F1E6DC | #483A30 |
| 侧栏着色 | rgba(255,246,238,0.86) | rgba(44,35,30,0.78) |

分隔线和中性色边框一律使用不透明边线色，包括材质上的竖线、横线、卡片边框和虚线入口；不把半透明暖色叠在系统材质上画边线。

无边框控件悬停、按下用暖棕半透明层：浅色 rgba(120,72,40,0.06/0.08/0.11)，深色 rgba(232,190,158,0.08/0.11/0.16)。正文、次要文字及主按钮文字在各自表面上的对比度不低于 4.5:1；品牌橙上使用深棕文字。危险操作保留独立危险色。界面缩放支持 80%、90%、100%、110%、125%、150%，所有窗口同步。

旧偏好读取时把 light-gray 转为 light，把 dark-gray、navy、purple 转为 dark。它们不再作为选项或独立色板存在。

## 字体

使用 mygo 系统字体及 CJK 回退。正文 13，次要信息 12，侧栏计数 11.5，侧栏分组标题 11，工具栏位置标题 22，欢迎标题 24。只用常规和粗体，不缩小文字挤空间。

## 控件

普通按钮和输入框圆角 8，浮层与设置分组 12，图库切换器 10；瀑布流图片 8，网格图片 4、选中外圈 10。卡片间距 10；悬停轻微压暗，选中用品牌色描边且与图片留缝。尺寸信息只在悬停时显示小号磨砂胶囊。数量右对齐。

全应用图标使用 Phosphor Icons（MIT），普通 Regular、侧栏当前选中项 Fill；颜色由 currentColor 继承。普通次要文字色、悬停正文色、选中主题色、主按钮内跟随按钮文字色。侧栏、工具栏、设置导航 18，菜单及检查器 16，小操作 14–16；不自绘、不混入其他图标库。SVG 来源和许可保留在 app/ui。

主题选择在设置与引导共用三张带各自名称的预览卡，选中卡用品牌色描边。所有分段控件共用凸起底块，选中底块平滑移动。主按钮禁用时使用 Surface 底和 TextMuted 字色，不对整块按钮降低透明度，输入与无障碍语义仍为禁用。

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

动画共用约 250ms、轻微回弹的弹簧，颜色过渡 120ms；交给 mygo 自动遵守系统减少动态效果，不增加动画开关。浮层从锚点方向淡入并从 0.96 放大，关闭反向；对话框背景淡入，面板轻微上移缩放。稳定素材身份驱动可见卡片布局变化，屏幕外直接就位。性能用 12000 张样本与 mygo 帧统计测量，平均与最差帧时间分别记录。

## 布局

默认 1100×800，最小 1100×640。侧栏默认 236，可在 196–340 间拖宽；检查器 280。间距采用 4、8、12、16、24。工具栏单行高 60，覆盖滚动内容并模糊背后的图片。窄中栏把搜索和视图切换收进浮层，所有功能仍可操作。搜索默认 ⌘K / Ctrl+K，通过统一快捷键设置改绑、禁用和冲突检查；提示显示实际绑定。侧栏导航按全部、最近使用、已收藏、随机浏览、整理、文件夹、智能文件夹排列；空分组显示虚线入口。

## 层次

主窗口与设置窗口使用系统侧栏材质，浅色加暖奶油着色、深色加暖棕着色。Linux 无系统材质时用实色表面。工具栏、浮层用窗口内高斯背景模糊，暖色遮罩保持文字清晰；减少透明度时使用不透明表面。

分栏用细分隔线和表面亮度区分；图片卡片不加装饰阴影。浮层遮住背景并保留明确的关闭入口。

## 规则

一个元素只表达一个意思；无装饰背景、无夸张动效；导入进度与素材计数分开；隐藏没有数据的 EXIF 字段。

## 窗口适配

缩小窗口时内容区自适应列数，侧栏和检查器可收起；不牺牲文字可读性。浅深色、中文英文都检查默认和最小窗口。

## 实现

所有界面使用 mygo/ui；共享布局与主题入口保持统一；数据读写在后台，结果通过 Window.Update 进入主线程。

## 后续

仅记录：用户自选强调色、更多动效，本轮不实现。
