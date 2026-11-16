package ui

import (
	"slices"
	"testing"
)

// 复制文件后粘贴：视频、RAW、PSD 和图片一样按原文件导入，别的文件不收。
func TestImageClipboardPathsAcceptsEveryImportableFormat(t *testing.T) {
	got := imageClipboardPaths([]string{"/a/photo.JPG", "/a/clip.mov", "/a/raw.NEF", "/a/art.psd", "/a/notes.txt", "/a/doc.pdf"})
	want := []string{"/a/photo.JPG", "/a/clip.mov", "/a/raw.NEF", "/a/art.psd"}
	if !slices.Equal(got, want) {
		t.Fatalf("剪贴板文件筛选不对：%v", got)
	}
}
