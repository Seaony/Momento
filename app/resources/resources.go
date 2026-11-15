package resources

import "embed"

//go:embed welcome-icon.png
var WelcomeIcon []byte

// 图库图标：一套 128 像素的方块小图，文件名就是图标编号；momento.png 是没选图标时用的应用图标。
//
//go:embed library-icons/*.png
var LibraryIcons embed.FS

// 空状态插画：黑色线稿加透明度的蒙版，界面里按主题着色。
//
//go:embed empty/*.png
var EmptyArt embed.FS
