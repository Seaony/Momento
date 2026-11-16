package assets

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func TestImportDownloadFixesExtensionAndAppliesDetails(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	data, err := os.ReadFile(filepath.Join("testdata", "sample.webp"))
	if err != nil {
		t.Fatal(err)
	}
	var referer string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		referer = r.Header.Get("Referer")
		// CDN 常见情况：地址是 .jpg、类型也说 JPEG，实际给的是 WebP。
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(data)
	}))
	defer server.Close()
	request := model.ImportRequest{Name: " 网页\n标题 ", Note: " 备注 ", TagNames: []string{"猫", " 猫 ", "CAT", "cat", ""}}
	result, err := s.ImportDownload(ctx, Download{URL: server.URL + "/photo.jpg?x=1", PageURL: "https://example.com/post", Referer: "https://cdn.example.com/"}, request)
	if err != nil || result.Imported != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	if referer != "https://cdn.example.com/" {
		t.Fatalf("Referer 是 %q", referer)
	}
	got := snapshot(t, ctx, lib)
	a := got.Assets[0]
	if a.Extension != "webp" || a.Name != "网页 标题" || a.Note != "备注" || a.SourceURL != "https://example.com/post" {
		t.Fatalf("%+v", a)
	}
	if len(got.Tags) != 2 || len(a.TagIDs) != 2 {
		t.Fatalf("标签没去重: %+v", got.Tags)
	}
	// 再存一次同一张：已有素材的名称和备注不改，标签照样补上。
	request = model.ImportRequest{Name: "别的名字", TagNames: []string{"狗"}}
	if result, err = s.ImportDownload(ctx, Download{URL: server.URL + "/photo.jpg"}, request); err != nil || result.Skipped != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	a = snapshot(t, ctx, lib).Assets[0]
	if a.Name != "网页 标题" || len(a.TagIDs) != 3 {
		t.Fatalf("重复导入改了已有素材: %+v", a)
	}
	// 没给网页和 Referer 时不自己编 Referer。
	if referer != "" {
		t.Fatalf("Referer 是 %q", referer)
	}
}

func TestImportDownloadRejectsNonMedia(t *testing.T) {
	_, _, s := testLibrary(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>login</html>"))
	}))
	defer server.Close()
	_, err := s.ImportDownload(t.Context(), Download{URL: server.URL + "/page"}, model.ImportRequest{})
	var localized *i18n.LocalizedError
	if !errors.As(err, &localized) || localized.Key != "error_download_format" {
		t.Fatalf("网页没被当成不支持的格式: %v", err)
	}
}

func TestImportUploadUsesOriginalNameAndContent(t *testing.T) {
	ctx, lib, s := testLibrary(t)
	upload := filepath.Join(t.TempDir(), "momento-upload-123")
	fixtureCopy(t, "png", upload)
	// 扩展传来的文件名没有后缀（比如 blob: 地址），按内容补上。
	result, err := s.ImportUpload(ctx, upload, "截图", "https://example.com/page", model.ImportRequest{})
	if err != nil || result.Imported != 1 {
		t.Fatalf("%+v %v", result, err)
	}
	a := snapshot(t, ctx, lib).Assets[0]
	if a.OriginalFileName != "截图.png" || a.Extension != "png" || a.SourceURL != "https://example.com/page" {
		t.Fatalf("%+v", a)
	}
	if _, err := os.Stat(upload); !os.IsNotExist(err) {
		t.Fatalf("上传的临时文件还在: %v", err)
	}
}

func TestSniffExtension(t *testing.T) {
	for _, ext := range []string{"jpg", "png", "gif", "webp", "bmp", "tiff", "heic", "avif"} {
		data, err := os.ReadFile(filepath.Join("testdata", "sample."+ext))
		if err != nil {
			t.Fatal(err)
		}
		if got := sniffExtension(data[:64]); got != ext {
			t.Fatalf("%s 认成了 %q", ext, got)
		}
	}
	for head, want := range map[string]string{
		"\x00\x00\x00\x18ftypqt  ": "mov",
		"\x00\x00\x00\x18ftypisom": "mp4",
		"\x00\x00\x00\x18ftypM4V ": "m4v",
		"\x1a\x45\xdf\xa3":         "webm",
		"8BPS":                     "psd",
		"<html>":                   "",
	} {
		if got := sniffExtension([]byte(head)); got != want {
			t.Fatalf("%q 认成了 %q", head, got)
		}
	}
}

func TestRenameByContentKeepsRawExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DSC_0001.NEF")
	if err := os.WriteFile(path, []byte("MM\x00*"+strings.Repeat("\x00", 60)), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := renameByContent(path, ""); err != nil || got != path {
		t.Fatalf("RAW 被改了后缀: %q %v", got, err)
	}
}

func TestDownloadErrorHidesURLs(t *testing.T) {
	_, _, s := testLibrary(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://cdn.example/%ZZ.png?token=secret")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	_, err := s.ImportDownload(t.Context(), Download{URL: server.URL + "/a.png?sig=private"}, model.ImportRequest{})
	if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "private") {
		t.Fatalf("报错带了网址: %v", err)
	}
}
