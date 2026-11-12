# 图库文件图标

`.momento` 图库包的三款候选均保留应用已有的彩虹刺猬。文件夹表示整个可移动的图库包，不改应用图标。

- `01-coral-folder.png`：正面珊瑚色文件夹，刺猬浮在前片上。暂用这款；文件夹轮廓明确，浅深背景都清楚。
- `02-peeking-folder.png`：刺猬从米白色文件夹里探出，脸部较大，轮廓更活泼。
- `03-blue-folder.png`：略带侧角的蓝色文件夹，刺猬头部作为浮雕，后片露出一张图片。
- `comparison.png`：三款在浅深背景下的 180、60、32、16 像素比较。
- `prompts/`：每款候选的完整制作说明，以及单色托盘图标说明。

当前接入资源为 `app/resources/library-icon.png`（1024 像素）、`app/resources/library.ico`（16、32、48、64、128、256 像素）和 `app/resources/darwin/library.icns`。最终款式由用户选择。

macOS：mygo 会把 `resources/darwin/` 的内容放进应用包的 Resources；文档类型声明用 `CFBundleTypeIconFile = library.icns`，导出类型同时声明 `UTTypeIconFile = library.icns`，避免系统退回通用文档图标。现有类型标识、文件关联和目录包声明保持一致。

Windows：mygo v0.9.0 的 `fileAssociations` 没有独立图标字段，安装器默认使用应用可执行文件的第一幅图标。应用启动后为当前用户把 `.momento` 的 `DefaultIcon` 指向随包的 `library.ico`，不改默认打开应用。实际图库目录内另外保存 `.momento-library.ico` 和 Unicode `desktop.ini`，只引用包内相对文件名；移动或复制整个目录后，图标不依赖安装位置。只更新能识别的自家配置和图标，已有自定义内容会保留。真机仍需核对资源管理器的目录包显示。

`app/resources/tray-icon.png` 为 32 像素黑色透明模板，适合 macOS 菜单栏。小尺寸检查包含 16、22 和 32 像素；透明间隙保留眼睛、面部和刺猬轮廓。

实现依据：[Windows 目录自定义](https://learn.microsoft.com/en-us/windows/win32/shell/how-to-customize-folders-with-desktop-ini)、[文件类型图标](https://learn.microsoft.com/en-us/windows/win32/shell/how-to-assign-a-custom-icon-to-a-file-type)、[当前用户文件类型注册](https://learn.microsoft.com/en-us/windows/win32/shell/fa-file-types)。
