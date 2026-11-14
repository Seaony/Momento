# Momento 功能图标

51 个静态物件图标，全应用使用同一套。主色是 `#F47B45`，内容为 `#FFF6EC` 奶油色；填充按清透色板混入 10% 白色，描边由主色降低饱和度并压暗，落到暖棕。深浅界面共用固定本色，选中只改变行背景和文字。

- `icons.js`：物件与几何定义。
- `export.mjs`：复用已安装的 Kubai 引擎，导出静态 SVG 并写入三个 Go 图标文件。引擎、展示页和自检脚本只放临时目录，不打包进应用。
- `LICENSE`：保留 Kubai 许可；这些图标用于 Momento 产品。

使用已安装的技能目录重新导出：

```sh
node design/ui-icons/export.mjs <kubai-icons技能目录>
```

命令会打印临时工作目录。可用其中的 `scripts/check.mjs` 检查 24/28px；`MOMENTO_ICON_WORK` 可以指定工作目录。检查浏览器只开独立的无头进程，不访问用户已有标签页。

原生验收：在 `app/` 中执行 `MOMENTO_ICON_SCREENSHOTS=1 go test ./ui -run '^TestMomentoIcon' -count=1`，生成 `/tmp/momento-polish/shots/icons/native-light.png` 和 `native-dark.png`。每项同时显示真实 20px 和 22px，覆盖全部图标。图标用 `ui.Image` 展示，`ui.Icon` 会抹掉固有颜色。

mygo v0.9.0 支持 SVG 渐变和外描边遮罩，忽略 SVG filters。源文件与静态导出保留投影和顶部内光，原生实际效果以分层、渐变和描边为准；不修改第三方引擎。

| 功能 / 语义名 | 物件 |
| --- | --- |
| 全部素材 `photos` | 叠放的风景照片 |
| 最近使用 `recent` | 圆表盘时钟 |
| 收藏 `favorite` / `heart` | 奶油内心的圆润爱心 |
| 随机浏览 `random` | 三点骰子 |
| 文件夹 `folder` | 装纸的文件夹 |
| 智能文件夹 `smart-folder` | 文件夹加闪光 |
| 未分类 `unfiled` | 空的收纳抽屉 |
| 标签 `tag` | 带孔吊牌 |
| 未打标签 `untagged` | 带虚线的空吊牌 |
| 标签管理 `tags` | 叠放的吊牌 |
| 回收站 `trash` | 带盖与提手的桶 |
| 通用设置 `settings` / `general` | 圆润六齿齿轮 |
| 来源链接 `link` | 两节相扣的链条 |
| 打开来源 `external-link` | 带向外箭头的窗口 |
| 缩略图大小 `thumbnail-size` / `thumbnail` | 大小两幅相框 |
| 筛选 `filter` | 漏斗 |
| 侧栏 `sidebar` | 左侧分栏的窗口 |
| 检查器 `inspector` | 右侧分栏的窗口 |
| 导入 `import` | 向下收件的托盘 |
| 导出 `export` | 向上送出的托盘 |
| 排序 `sort` | 不等长内容的纸页和向下箭头 |
| 瀑布流 `masonry` | 长短错落的照片板 |
| 网格 `grid` | 四格照片板 |
| 列表 `list` / `menu` | 三层资料架 |
| 搜索 `search` | 奶油镜面的放大镜 |
| 后退 `back` / `chevron-left` / `arrow-left` | 向左方向键 |
| 前进 `forward` / `chevron-right` / `arrow-right` | 向右方向键 |
| 下拉 `down` / `chevron-down` | 向下方向键 |
| 新增 `plus` / `add` | 加号按键 |
| 移除规则 `minus` | 减号按键 |
| 关闭 `close` | 叉号按键 |
| 已选 `check` | 勾号按键 |
| 切换选项 `chevrons-up-down` / `chevrons` | 上下双向按键 |
| 幻灯片 `play` / `slideshow` | 播放按键 |
| 暂停 `pause` | 暂停按键 |
| 密码锁 `lock` / `password` | 有锁孔的挂锁 |
| 操控 `controls` | 双键鼠标 |
| 快捷键 `keyboard` / `shortcuts` | 奶油按键的键盘 |
| 通知 `notifications` | 铃铛 |
| 外观 `appearance` | 有奶油色颜料槽的调色盘 |
| 浏览器扩展设置 `browser` | 窗口中的地球 |
| 关于 `info` / `about` | 信息按键 |
| 显示密码 `eye` | 有奶油眼白的眼睛 |
| 隐藏密码 `eye-off` | 带斜线的眼睛 |
| 编辑 `edit` | 奶油笔芯的铅笔 |
| 图库 `library` | 风景封面的相册 |
| 新建图库 `create-library` | 带加号的相册 |
| 安装浏览器扩展 `extension` | 拼图片 |
| 警告 `warning` | 圆角三角告示牌 |
| 向左旋转 `rotate-left` / `restore` | 向左转的旋钮 |
| 向右旋转 `rotate-right` | 向右转的旋钮 |

设置页 `preview` 使用风景照片；侧栏旧中文名称保留映射。内部调用首选 `appIcon(c, name)`；需要裸 SVG 时取 `momentoIcons` 中的标准语义名。
