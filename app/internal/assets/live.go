package assets

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Seaony/Momento/app/internal/media"
)

// readImageContentID 读图片里苹果实况照片的内容 ID，测试里可以换掉。
var readImageContentID = media.ImageContentID

// motionSource 是和图片配成实况照片的那段视频。
type motionSource struct {
	path       string
	staged     string
	hash       string
	durationMS int64
}

// isLiveStill 判断这种图片能不能当实况照片的图片那半：苹果导出是 HEIC 或 JPEG。
func isLiveStill(extension string) bool {
	switch extension {
	case "heic", "heif", "jpg", "jpeg":
		return true
	}
	return false
}

// pairLivePhotos 把同一批里同一文件夹、同名的「图片 + MOV」配成实况照片：视频挂到图片上，不再单独导入。
// 只有两边都读得到内容 ID 且一致才配：只是同名的普通照片和视频各导各的；视频读不出来也不配，让它自己报错。
func pairLivePhotos(candidates []candidate) []candidate {
	// 文件夹按原样比（区分大小写的磁盘上 A/ 和 a/ 是两个文件夹），只有文件名不分大小写。
	key := func(c candidate) string {
		base := filepath.Base(c.path)
		return filepath.Join(filepath.Dir(c.path), strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base))))
	}
	stills := map[string]int{}
	for i, c := range candidates {
		if isLiveStill(c.extension) {
			if _, taken := stills[key(c)]; !taken {
				stills[key(c)] = i
			}
		}
	}
	absorbed := map[int]bool{}
	for i, c := range candidates {
		if c.extension != "mov" {
			continue
		}
		still, ok := stills[key(c)]
		if !ok || candidates[still].motion != nil {
			continue
		}
		info, err := probeVideo(c.path)
		if err != nil {
			slog.Warn("同名视频读不出来，不配成实况照片", "file", filepath.Base(c.path), "error", err)
			continue
		}
		imageID, err := readImageContentID(candidates[still].path)
		if err != nil {
			slog.Warn("读不出图片的实况内容 ID，同名图片和视频当两个素材导入", "file", filepath.Base(candidates[still].path), "error", err)
			continue
		}
		if imageID == "" || info.ContentID == "" || imageID != info.ContentID {
			slog.Info("同名图片和视频没有一致的实况内容 ID，当两个素材导入", "image", filepath.Base(candidates[still].path), "video", filepath.Base(c.path), "imageID", imageID, "videoID", info.ContentID)
			continue
		}
		candidates[still].motion = &motionSource{path: c.path, durationMS: info.Duration.Milliseconds()}
		absorbed[i] = true
	}
	if len(absorbed) == 0 {
		return candidates
	}
	kept := candidates[:0:0]
	for i, c := range candidates {
		if !absorbed[i] {
			kept = append(kept, c)
		}
	}
	slog.Info("配出实况照片", "count", len(absorbed))
	return kept
}

// motionRelative 是实况照片视频在包里的位置：和图片同一个文件夹，名字带 .live，不会和单独导入的视频撞名。
func motionRelative(hash, extension string) string {
	return filepath.ToSlash(filepath.Join("assets", hash[:2], hash+".live."+extension))
}
