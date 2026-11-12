# AGENTS.md

## 沟通与范围

- 回复用中文，以「喵」开头和结尾。调查要细，判断要果断。
- 只改本次任务相关代码；错误必须可见且记日志，不用空 catch、假 fallback 或 silent success。
- Momento 是跨平台本地图片素材库。

## 技术与结构

- Go 1.27.3+，mygo **v0.9.0** 原生 GPU 界面；不使用 Web 前端，不用 cgo。
- `app/` 是模块 `github.com/Seaony/Momento/app`。
- 应用版本只在 `app/mygo.json` 中定义，运行时和测试共用同一个版本读取入口。
- `main.go` 负责入口、系统菜单、深度链接、文件关联、单实例和日志；`settings_window.go` 管独立设置窗口，重复打开只聚焦已有窗口。
- `internal/model/` 放共享值类型；`internal/library/` 管图库包、SQLite、文件夹、标签和最近图库；`internal/assets/` 管图片处理、导入导出和回收站；`internal/catalog/` 管内存筛选、排序和计数；`internal/browser/` 管本机扩展服务；`internal/i18n/` 管中英日文；`ui/` 放全部界面。
- `internal/` 不引用 mygo/ui 或应用 ui 包，必须能单独测试。
- SQLite 使用 modernc.org/sqlite；数据库只保存包内相对路径；图库是可整体移动、复制和备份的 `.momento` 目录包。
- 数据读写和解码在后台进行，通过 Window.Update 异步更新界面；需要同步读取主线程状态时用 RunOnMain，不把 Update 当同步调用。缩略图缓存必须有数量和内存上限。
- 用户可见文案全部通过 internal/i18n，中英日 key 与参数占位符必须齐全。主题、应用界面与 mygo 菜单语言即时切换；AppKit 原生文件对话框按钮通常下次启动进程才更新语言，不加重启生效的产品提示。
- 主窗口与设置窗口使用 HiddenInset；按 c.TitleBar() 为各平台按钮留位，左侧无按钮时不预留空行。检查器顶部拖动空白与正文同色，不另画标题带。
- 设置分为通用、外观、侧栏、操控、预览、导入、快捷键、通知、密码锁、浏览器扩展、关于十一页。登录启动读回系统真实状态，失败显示并记日志；扩展开关实际启停本机监听；自动更新检查开启时每次启动检查一次，关闭时仅手动检查。
- 偏好存应用配置目录的 preferences.json，缺字段用默认值；自动记住视图、排序及方向、侧栏与检查器显示状态、侧栏宽度，不增加默认视图设置项。退出前同步保存最新状态，写入失败不能静默丢失。

## 开发

在 `app/` 目录执行：

```sh
go tool mygo dev
go build ./...
go vet ./...
go test ./internal/...
go test ./...
```

- 离线主界面截图：`MOMENTO_SCREENSHOTS=1 go test ./ui -run '^TestStateScreenshots$' -count=1`；设置窗口：`MOMENTO_SETTINGS_SCREENSHOTS=1 go test ./ui -run '^TestSettingsWindowScreenshots$' -count=1`。开发样本：`go run ./cmd/sample-library -path <新路径>.momento -count <数量>`；目标不得覆盖已有数据。
- 写 mygo API 前读 v0.9.0 对应官方 docs 和 examples/gallery，并用 go doc 与编译器核对。不要凭记忆猜 API，不修改第三方或 vendor 代码。
- 界面检查使用 ui.Render / ui.Tester 离线截图，不操作用户的浏览器。
- 保留 com.seaony.Momento、momento://import?url=、.momento 文件关联，以及 127.0.0.1:47641 的扩展 HTTP 契约。
- 不调用测试服或生产服接口。本机开发服务可以测试。
- 日志带时间戳，记录启动、图库打开、导入导出、HTTP 请求与错误，按大小轮转。

## 测试

- 数据生命周期、导入去重、回收站、事务失败、相对路径、文件夹和标签关系是重点。
- 覆盖 JPEG、PNG、GIF、WebP、TIFF、BMP、HEIC、AVIF 的导入和缩略图。
- 筛选和排序保留大样本基准；界面使用少量行为测试和状态截图，不为间距字号写源码匹配测试。
- 修改后运行相关测试、build、vet 和 git diff --check。未执行项目标明 not run，不编造结果。
- 截图只出这次改动影响到的状态，每个状态出中文浅色、深色各一张，逐张看过，对照修改单核对位置、顺序、文案与交互；一次验收一般不超过 20 张，汇报里写明张数。英日文的截断和换行用测试断言检查，不出图；没改到的界面不重出、不复看。只有新页面或整体改版才出中英日、浅深色全套。不能用旧截图代替改动验收。
- 离线窗口按钮留位不能代替 Windows/Linux 标题栏真机验收，登录启动实际效果须三端分别验收。

## 并行与提交

- 主会话负责总控、集成和提交；共享 model、main.go、ui/app.go、go.mod 只由主会话集成修改。子代理按文件独占分工，不碰他人文件，不提交、不建分支或 worktree，构建产物放各自独立目录。
- 按当前任务文件的步骤推进，每块完成并验证后提交并立即推送 origin master。禁止 force push、secrets、WIP 提交。
- 提交格式 `<type>: <一句简短中文>`，type 为 feat / fix / refactor / test / docs / chore，无正文和署名。
- 不把 develop 的任何内容带进功能分支，不使用改名 refspec。
- 发布流程只建立草稿，不自动公开；更新私钥和签名证书不进仓库。交叉编译、打包、正式签名、公证、签名更新和真机运行分别记录验证结果。
