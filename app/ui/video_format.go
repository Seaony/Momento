package ui

import (
	"fmt"
	"time"

	"github.com/Seaony/Momento/app/internal/model"
)

// videoDuration 把时长写成播放器常见的样子：不到 1 小时「0:12」「12:05」，超过写「1:02:03」。
// 不足 1 秒的视频算 0:01，免得卡片上出现 0:00。
func videoDuration(d time.Duration) string {
	seconds := int64((d + time.Second/2) / time.Second)
	if d > 0 && seconds == 0 {
		seconds = 1
	}
	h, m, s := seconds/3600, seconds/60%60, seconds%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// isVideoAsset 判断素材是不是视频（实况照片按图片看，只在悬停或预览时播那段视频）。
func isVideoAsset(asset model.Asset) bool { return asset.Kind == model.KindVideo }

// assetDuration 是素材的视频时长；图片为 0。
func assetDuration(asset model.Asset) time.Duration {
	return time.Duration(asset.DurationMS) * time.Millisecond
}
