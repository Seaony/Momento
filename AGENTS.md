# AGENTS.md

## 沟通与范围

- 回复用中文，以「喵」开头和结尾。调查要细，判断要果断。
- 只改本次任务相关代码；错误必须可见且记日志，不用空 catch、假 fallback 或 silent success。
- Momento 是全新的跨平台本地图片素材库，不读取、识别或迁移其他版本的图库和设置。

## 技术与结构

- Go 1.27.3+，mygo **v0.9.0** 原生 GPU 界面；不使用 Web 前端，不用 cgo。
- `app/` 是模块 `github.com/Seaony/Momento/app`。
- `main.go` 负责入口、系统菜单、深度链接、文件关联、单实例和日志。
- `internal/model/` 放共享值类型；`internal/library/` 管图库包、SQLite、文件夹、标签和最近图库；`internal/assets/` 管图片处理、导入导出和回收站；`internal/catalog/` 管内存筛选、排序和计数；`internal/browser/` 管本机扩展服务；`internal/i18n/` 管中英文；`ui/` 放全部界面。
- `internal/` 不引用 mygo/ui 或应用 ui 包，必须能单独测试。
- SQLite 使用 modernc.org/sqlite；数据库只保存包内相对路径；图库是可整体移动、复制和备份的 `.momento` 目录包。
- 数据读写和解码在后台进行，通过 Window.Update 更新界面；缩略图缓存必须有数量和内存上限。
- 用户可见文案全部通过 internal/i18n，中文英文 key 必须齐全。

## 开发

在 `app/` 目录执行：

```sh
go tool mygo dev
go build ./...
go vet ./...
go test ./internal/...
go test ./...
```

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

## 并行与提交

- 共享 model、main.go、ui/app.go、go.mod 只由主会话改；子代理按文件分工，不提交、不建分支或 worktree，构建产物放各自 /tmp 目录。
- 本次重写按任务文件的 P0 顺序推进，每块完成并验证后提交并立即推送 origin master。禁止 force push、secrets、WIP 提交。
- 提交格式 `<type>: <一句简短中文>`，type 为 feat / fix / refactor / test / docs / chore，无正文和署名。
- 不把 develop 的任何内容带进功能分支，不使用改名 refspec。
