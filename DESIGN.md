# 界面规范

## 视觉

安静、紧凑的桌面图片工作区。图片是视觉中心，浅色是几乎白的浅暖橙，深色以近黑为主、只留一点暖意。保留 Momento 自己的布局、圆润图标和磨砂，品牌橘只承担明确的交互含义。

## 色彩

品牌色来自暖橘底彩虹刺猬图标，只保留浅色、深色两套色板。「跟随系统」在这两套间切换。全部窗口、浮层、菜单、锁屏和引导共享色板。

| 用途 | 浅色 | 深色 |
| --- | --- | --- |
| Background 内容底 | #FFFCFA | #111010 |
| Sidebar 侧栏底 | #FFF7F1 | #171615 |
| Raised / Selected 浮起面 | #FFFFFF | #201E1D / #232120 |
| SegmentSelected 分段选中块 | #FFFFFF | #2F2C2A |
| Subtle 预览容器、药丸、分段底 | #FDF1E8 | #1A1918 |
| Field / FieldHover 表单浅底 | #F7EEE6 / #F3E8DE | #222020 / #282625 |
| Hover / Pressed | #FBECE0 / #F7E2D2 | #262422 / #2E2B29 |
| Text 正文与标题 | #2B221D | #F4F0ED |
| TextSecondary 次要信息 | #6F5F55 | #B6AEA8 |
| TextTertiary 静态字段标签 | #7D6B60 | #938A84 |
| Accent / Hover / Pressed | #F47B45 / #FA8B59 / #E8733D | #FF9A66 / #FFAB7E / #F48A54 |
| OnAccent 主按钮文字 | #2B221D | #2B221D |
| Focus / Link | #D9622C / #A74320 | #FF9A66 / #FFB48F |
| BorderSubtle / Border（不透明） | #F3E3D6 / #EAD6C6 | #252321 / #322F2D |
| 系统侧栏材质着色 | Sidebar × 62% | Sidebar × 70% |
| 弹出层、菜单、对话框磨砂着色 | Raised × 88% | Raised × 88% |
| Scrim 遮罩 | Text × 25% | 黑 × 55% |

分隔线和中性色边框一律不透明，包括材质上的竖线、横线、卡片边框和虚线入口；框架 Theme.Border 映射 BorderSubtle，让 Split、侧栏底线和工具栏下沿同色。需要更明确的输入边框时用语义 Border，不给边线叠透明度。

表单控件（搜索框、输入框、备注框、密码框、下拉选择）统一走 fieldStyle / menuFieldStyle：Field 浅底 + BorderSubtle 浅边框，悬停 FieldHover，聚焦换 Raised 底和 Focus 边框，不单靠一圈边框区分层级。磨砂着色 88% 是对比度下限：压在纯黑或纯白图片上，次要文字仍不低于 4.5:1。控件悬停、按下统一用 Hover / Pressed。正文、次要文字、主按钮及静态字段标签在各自表面上的对比度不低于 4.5:1；浅色三级标签比参考起点略深，保证小字可读。危险操作保留独立危险色。界面缩放支持 80%、90%、100%、110%、125%、150%，所有窗口同步。

品牌橘只用于主按钮、当前导航图标、卡片选中描边、焦点环、链接、开关打开态和滑块已选段；选中行文字用 Text、底用 Raised。浮起面用 1px BorderSubtle 与暖色轻影（Y 1、模糊 2，浅色 7%），通过 elevated helper 统一；磨砂调用方保留自己的材质，不被实色背景覆盖。

旧偏好读取时把 light-gray 转为 light，把 dark-gray、navy、purple 转为 dark。它们不再作为选项或独立色板存在。

## 字体

和旧版 macOS 一样用系统界面字体（system-ui：macOS 为 SF Pro，Windows 为 Segoe UI，Linux 为 GTK 界面字体），不内置字体。正文 13、次要信息 12；独立成列的数量、尺寸、日期、大小、版本号用 tnum 等宽数字，不换字体。中文按平台回退苹方、微软雅黑或思源 / Noto Sans；日文界面优先日文字体。字体家族由 applyTheme 设置，不缩小文字挤空间。

## 控件

所有尺寸从 ui/themes.go 的 token 取，不在调用处写数字：间距 space4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64；圆角 radiusSmall 4、radiusCard 6、radiusControl 8、radiusPanel 12；字号 fontGroup 11、fontSecondary / fontNumber 12、fontBody 13、fontTitle 16、fontHeading 20、fontDisplay 24；图标 iconSmall 14、iconMedium 16、iconSize 18；控件高 controlSmall 28、controlHeight 32、controlLarge 36、rowHeight 44；复选框 checkSize 16；面板宽 panelNarrow 360、panelDialog 420、panelWide 560；磨砂透明度 materialPanelAlpha / materialSidebarAlpha / materialOverlayAlpha。用户要求的圆形按钮和胶囊仍按完整圆形绘制。素材图片圆角 6、轻暖影、悬停轻微上浮；选中橘色 2px 描边留缝。数量右对齐。

全应用图标使用 Solar Icon Set（CC BY 4.0，署名见 app/ui/SOLAR-LICENSE），线条圆润、拐角柔和，和暖色调一致；普通用 Linear、侧栏当前选中项用 Bold。加号、减号直接用 Solar 的带圈版；Solar 没有不带圈的 × ✓，这两个取自带圈版的符号、去圈后放大；上下尖角由 Alt Arrow Up / Down 缩小拼成；它们和拖动手柄没有填充版，选中时仍用线性版；颜色由 currentColor 继承。普通次要文字色、悬停正文色、选中主题色、主按钮内跟随按钮文字色。侧栏、工具栏、设置导航 18，菜单及检查器 16，小操作 14–16；不自绘、不混入其他图标库。SVG 在 app/ui/icons.go，key 是界面里的图标名，换图标库只改这一个文件。

主题选择在设置与引导共用三张带各自名称的预览卡，选中卡用品牌色描边。所有分段控件共用 SegmentSelected 凸起底块，比 Subtle 底槽明显亮一档，选中底块平滑移动。主按钮禁用时使用 Surface 底和 TextMuted 字色，不对整块按钮降低透明度，输入与无障碍语义仍为禁用。

| 功能 | Solar 图标名（界面 key） |
| --- | --- |
| 全部素材 / 素材预览 | Gallery (`images`) |
| 最近使用 | History (`clock-counter-clockwise`) |
| 已收藏 | Heart (`heart`) |
| 随机浏览 | Shuffle (`shuffle`) |
| 未分类 | Inbox (`tray`) |
| 未打标签 / 标签 | Tag Horizontal (`tag-simple`) |
| 标签管理 | Tag (`tag`) |
| 文件夹 | Folder (`folder`) |
| 智能文件夹 | Folder Favourite (Star) (`folder-simple-star`) |
| 回收站 | Trash Bin Minimalistic (`trash`) |
| 设置 / 通用 | Settings (`gear-six`) |
| 帮助 | Question Circle (`question`) |
| 导入 | Download Minimalistic (`download-simple`) |
| 导出 | Upload Minimalistic (`upload-simple`) |
| 筛选 | Filter (`funnel-simple`) |
| 排序 | Sort From Top To Bottom (`sort-ascending`) |
| 瀑布流 | Widget 4 (`squares-four`) |
| 网格 | Widget (`grid-four`) |
| 列表 / 菜单 | Hamburger Menu (`list`) |
| 搜索 | Magnifer (`magnifying-glass`) |
| 缩略图大小 | Maximize Square Minimalistic (`frame-corners`) |
| 侧栏开关 | Sidebar Minimalistic (`sidebar-simple`) |
| 收起检查器 | Double Alt Arrow Right (`caret-double-right`) |
| 来源链接 | Link Minimalistic 2 (`link-simple`) |
| 拖动排序 | Hamburger Menu (`dots-six-vertical`) |
| 更多操作 | Menu Dots (`dots-three`) |
| 打开、在文件管理器中显示 | Folder Open (`folder-open`) |
| 打开外部来源 | Square Share Line (`arrow-square-out`) |
| 图库 | Gallery Wide (`images-square`) |
| 新建图库 | Add Folder (`folder-plus`) |
| 密码锁 | Lock Keyhole Minimalistic (`lock-simple`) |
| 操控 | Mouse Minimalistic (`mouse-simple`) |
| 快捷键 | Keyboard (`keyboard`) |
| 通知 | Bell (`bell`) |
| 外观 | Palette (`palette`) |
| 浏览器扩展设置 | Global (`globe`) |
| 关于 / 信息 | Info Circle (`info`) |
| 显示 / 隐藏密码 | Eye (`eye`) / Eye Closed (`eye-slash`) |
| 编辑 | Pen 2 (`pencil-simple`) |
| 扩展安装 | Plug Circle (`puzzle-piece`) |
| 警告 | Danger Triangle (`warning`) |
| 左右翻页 | Alt Arrow Left (`caret-left`) / Alt Arrow Right (`caret-right`) |
| 展开 / 收起 | Alt Arrow Down (`caret-down`) / Alt Arrow Up (`caret-up`) |
| 上下双箭头 | Alt Arrow Up + Down 拼合 (`caret-up-down`) |
| 新增 / 移除 | Add Circle (`plus`) / Minus Circle (`minus`) |
| 关闭 / 勾选 | Close Circle 去圈 (`x`) / Check Circle 去圈 (`check`) |

动画共用约 250ms、轻微回弹的弹簧，颜色过渡 120ms；交给 mygo 自动遵守系统减少动态效果，不增加动画开关。浮层从锚点方向淡入并从 0.96 放大，关闭反向；对话框背景淡入，面板轻微上移缩放。

素材只有在视图形状和列数不变的筛选、搜索、排序、增删中，才按稳定身份移动可见卡片，屏幕外直接就位。切换视图时旧视图约 120ms 整体淡出，新视图约 180ms 整体淡入并从 0.98 放大到 1，不把图片卡变形成列表行。侧栏、检查器、窗口宽度及缩略图尺寸变化时，卡片直接跟随实际布局，列数跳变以 60ms 退场、90ms 入场，不逐卡飞行。栏宽保留弹簧前段，在 250ms 终点收准；搜索框不叠加第二层位移动画。100% 帧必须完全落定，旧视图无残影。性能用 12000 张样本与 mygo 帧统计测量；只记录超过阈值的日志时，不据此推算平均帧时间。

## 布局

默认 1100×800，最小 1100×640。侧栏默认 236，可在 196–340 间拖宽；检查器 280。间距采用 4、8、12、16、24。工具栏单行高 60，覆盖滚动内容并模糊背后的图片。窄中栏把搜索和视图切换收进浮层，所有功能仍可操作。搜索默认 ⌘K / Ctrl+K，通过统一快捷键设置改绑、禁用和冲突检查；提示显示实际绑定。侧栏照旧版 macOS（规格见 themes.go 的 sidebar* token）：图库选择器（高 26、图标 20、名称 13 半粗）下是「全部」，一条细分隔线，再平铺最近使用、已收藏、随机浏览、未分类、未打标签、标签管理；再一条分隔线后平铺文件夹，最后一行是虚线框的「新建文件夹」，它也是拖到根目录的落点；智能文件夹打开时同样平铺加新建行。不设分组标题和折叠。行高 30、行距 2、行内留白 8、圆角 9，图标 16 放在 18 宽的位里，图标和文字间距 10，文字 13 中等字重；平时文字和图标是正文色 72%，悬停和选中用正文色并铺 Hover 浅底，选中不换填充图标、不用品牌色。数量 11 半粗、等宽数字、文字色 58%，0 不显示。新建行用 Field 浅底，悬停 FieldHover，文字和图标同普通行。底部一条分隔线下只有回收站、设置、帮助三个 28 的图标按钮，回收站数量是右上角小角标。

## 层次

主窗口与设置窗口使用系统侧栏材质，浅色加接近白的浅暖橙着色、深色加暖深色着色。Linux 无系统材质时用语义 Sidebar 实色。工具栏、浮层保留背景模糊，暖色遮罩保持文字清晰；减少透明度时使用不透明表面。数值定下来后写回 token，并删掉这两个滑块。

分栏用细分隔线和表面亮度区分；图片卡片只加柔和轻影。浮层保留明确的关闭入口。确认按钮写具体动作，遮罩用 Scrim 暖深色半透明。

检查器底与内容区同为 Background，预览容器用 Subtle，输入框用 Border；备注与来源空值显示 Tertiary 占位。非编辑来源链接单行省略，编辑时滚动并保留完整值。标签、文件夹药丸用 12px 关闭图标移除；备注组前使用同样的浅分隔线。

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

预览照旧版 macOS 做成两块独立磨砂卡片：上面图片卡只放图片（内边距 14、图片圆角 12、卡片圆角 30）；下面隔 12 是单独的信息卡（高 52、宽 560，屏幕放不下才缩，圆角 16），居中两行：文件名 13 中等字重、「宽 × 高」11 号次要色；右侧一个 28 的圆形关闭按钮，常驻浅底。没有翻页按钮，←/→ 键翻页。按住空格 250ms 后信息卡第二行换成「继续按住，固定为窗口」，底边出现 2px 进度条。两块卡片是两个原生窗口（信息卡是图片卡的子窗口），都非激活显示，围绕同一中心做 0.92 到 1 的缩放和淡入淡出。初始大小不超过主窗口的 80%，小图自动放大不超过 2 倍。原图解码前使用缓存缩略图。固定后信息卡淡出，只留图片卡，悬停右上角出现关闭按钮。预览里不提供旋转。
