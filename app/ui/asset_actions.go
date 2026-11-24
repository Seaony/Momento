package ui

import (
	"bytes"
	"image/png"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

func (a *App) assetPath(asset model.Asset) string {
	return filepath.Join(a.Library.Root, filepath.FromSlash(asset.RelativePath))
}

// OpenAssetExternally 用系统默认应用打开原文件，算一次「使用」。
func (a *App) OpenAssetExternally(asset model.Asset) {
	if a.Library == nil || a.IsLocked() {
		return
	}
	if err := mygo.Shell.OpenPath(a.assetPath(asset)); err != nil {
		slog.Error("用默认应用打开失败", "id", asset.ID, "error", err)
		a.ShowError(err)
		return
	}
	a.MarkAssetsUsed([]string{asset.ID})
}

// CopyAssetPaths 把原文件的完整路径当文字复制，多个一行一个。
func (a *App) CopyAssetPaths(items []model.Asset) {
	if a.Library == nil || a.IsLocked() || len(items) == 0 {
		return
	}
	paths := make([]string, len(items))
	for i, asset := range items {
		paths[i] = a.assetPath(asset)
	}
	// 别的应用占着剪贴板时会写失败，失败要说出来，不能照样提示已复制。
	if err := mygo.Clipboard.Write(transfer.TextData(strings.Join(paths, "\n"))); err != nil {
		slog.Error("复制文件路径失败", "error", err)
		a.ShowError(err)
		return
	}
	// 和复制色值、复制标签一样给一句完成提示，不然看不出有没有复制上。
	if len(paths) == 1 {
		a.Notify(a.T("copied_path"))
	} else {
		a.Notify(a.T("copied_paths", len(paths)))
	}
}

// CopyAssetImage 把图片本身（不是文件引用）放进剪贴板，聊天和设计工具里能直接粘贴。
// 按检查器里设的旋转转好；不是 PNG 的先解码再编码成 PNG，大图在后台做。
func (a *App) CopyAssetImage(asset model.Asset) {
	if a.Library == nil || a.IsLocked() || isVideoAsset(asset) {
		return
	}
	path := a.assetPath(asset)
	go func() {
		data, err := assetPNG(path, asset)
		a.Window.Update(func() {
			if err == nil {
				err = mygo.Clipboard.WriteImage(data)
			}
			if err != nil {
				slog.Error("复制图片失败", "id", asset.ID, "error", err)
				a.ShowError(err)
				return
			}
			a.MarkAssetsUsed([]string{asset.ID})
			a.Notify(a.T("copied_image"))
		})
	}()
}

func assetPNG(path string, asset model.Asset) ([]byte, error) {
	if strings.EqualFold(asset.Extension, "png") && asset.Rotation == 0 {
		return os.ReadFile(path)
	}
	img, err := assets.Decode(path)
	if err != nil {
		return nil, err
	}
	if img, err = assets.RotateImage(img, asset.Rotation); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// CopyTags 记下选中素材的全部标签（并集，按先后顺序），只在 Momento 里用，不占系统剪贴板。
func (a *App) CopyTags() {
	if a.Library == nil || a.IsLocked() || a.TagPage {
		return
	}
	var tags []string
	for _, asset := range a.SelectedAssets() {
		for _, id := range asset.TagIDs {
			if !slices.Contains(tags, id) {
				tags = append(tags, id)
			}
		}
	}
	if len(tags) == 0 {
		a.Notify(a.T("no_tags_to_copy"))
		return
	}
	a.tagClipboard = tags
	a.Notify(a.T("tags_copied", len(tags)))
}

// PasteTags 把复制的标签加到选中的素材上（只加不减），复制后被删掉的标签跳过。
func (a *App) PasteTags() {
	// 回收站里的素材只读（检查器和右键菜单都不给改），快捷键也一样不改。
	if a.Library == nil || a.IsLocked() || a.TagPage || a.Query.Scope == "trash" {
		return
	}
	if len(a.tagClipboard) == 0 {
		a.Notify(a.T("no_tags_copied_yet"))
		return
	}
	ids := a.selectedIDs()
	if len(ids) == 0 {
		return
	}
	tags := slices.DeleteFunc(slices.Clone(a.tagClipboard), func(id string) bool {
		return !slices.ContainsFunc(a.Snapshot.Tags, func(tag model.Tag) bool { return tag.ID == id })
	})
	if len(tags) == 0 {
		a.tagClipboard = nil
		return
	}
	a.SetTags(ids, tags, true)
	a.Notify(a.T("tags_pasted", len(ids)))
}

// OpenFolderPicker 展开检查器里的文件夹选择器，和 Shift+F 打开的是同一个。
func (a *App) OpenFolderPicker() {
	if a.Selected.Len() == 0 || a.TagPage {
		return
	}
	a.RevealInspector()
	a.Inspector.TagsOpen, a.Inspector.FoldersOpen = false, true
}

// SelectAllVisible 选中当前列表里的全部素材，和 ⌘A 一样。
func (a *App) SelectAllVisible() {
	if a.TagPage {
		return
	}
	for _, v := range a.Visible {
		a.Selected.Add(v.ID)
	}
	a.RevealInspector()
	a.FollowPreviewSelection()
}
