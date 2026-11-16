package assets

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/media"
	"github.com/Seaony/Momento/app/internal/model"
)

// Download 是一次从网址导入：URL 是文件地址，PageURL 是所在网页（记成来源），
// Referer 是扩展按 CDN 算好的值（空就用网页地址），Video 表示扩展说这是视频，按视频的大小上限下载。
type Download struct {
	URL, PageURL, Referer string
	Video                 bool
}

func (s *Service) ImportURL(ctx context.Context, imageURL, pageURL string) (model.ImportResult, error) {
	return s.ImportDownload(ctx, Download{URL: imageURL, PageURL: pageURL}, model.ImportRequest{})
}

func (s *Service) ImportURLWithRequest(ctx context.Context, imageURL, pageURL string, request model.ImportRequest) (model.ImportResult, error) {
	return s.ImportDownload(ctx, Download{URL: imageURL, PageURL: pageURL}, request)
}

// ImportDownload 下载一个文件再导入。只导一个文件，它没导进去就算失败。
func (s *Service) ImportDownload(ctx context.Context, d Download, request model.ImportRequest) (result model.ImportResult, err error) {
	parsed, err := url.Parse(d.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return result, i18n.NewError("error_image_url")
	}
	slog.Info("下载文件", "host", parsed.Host, "video", d.Video)
	// 下载用单独的可取消上下文：一段时间没收到新数据就取消。
	downloadCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, d.URL, nil)
	if err != nil {
		return result, err
	}
	userAgent := "Momento"
	if s.AppVersion != "" {
		userAgent += "/" + s.AppVersion
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "image/*,video/*;q=0.9,*/*;q=0.5")
	// 只带扩展给的 Referer；没给就用网页地址，不自己再加别的。
	if referer := d.Referer; referer != "" {
		req.Header.Set("Referer", referer)
	} else if d.PageURL != "" {
		req.Header.Set("Referer", d.PageURL)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: downloadConnectTimeout, KeepAlive: 30 * time.Second}).DialContext
	transport.TLSHandshakeTimeout = downloadConnectTimeout
	transport.ResponseHeaderTimeout = downloadIdleTimeout
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return i18n.NewError("error_redirect_limit")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return i18n.NewError("error_redirect_scheme")
		}
		return nil
	}}
	response, err := client.Do(req)
	if err != nil {
		// 报错里常带完整网址（可能含签名参数，重定向地址也会出现在里面），去掉后界面和日志只显示域名。
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return result, i18n.WrapError(errors.New(redactURLs(err.Error())), "error_download_failed", parsed.Host)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result, i18n.NewError("error_download_status", response.StatusCode)
	}
	contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	contentType = strings.ToLower(contentType)
	name := filepath.Base(parsed.Path)
	if _, params, e := mime.ParseMediaType(response.Header.Get("Content-Disposition")); e == nil && params["filename"] != "" {
		name = params["filename"]
	}
	name = safeName(name)
	limit := maxDownloadBytes
	if d.Video || strings.HasPrefix(contentType, "video/") || isVideo(strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")) {
		limit = maxVideoDownloadBytes
	}
	dir, err := os.MkdirTemp("", "momento-download-")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, err
	}
	tooLarge := i18n.NewError("error_download_too_large", limit>>20)
	if response.ContentLength > limit {
		return result, errors.Join(tooLarge, f.Close())
	}
	idle := time.AfterFunc(downloadIdleTimeout, func() { cancel(errDownloadStalled) })
	written, err := io.Copy(f, idleReader{r: io.LimitReader(contextReader{ctx: downloadCtx, r: response.Body}, limit+1), timer: idle})
	idle.Stop()
	if err != nil && errors.Is(context.Cause(downloadCtx), errDownloadStalled) {
		err = i18n.NewError("error_download_stalled", int(downloadIdleTimeout/time.Second))
	} else if err == nil && written > limit {
		err = tooLarge
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		slog.Warn("下载失败", "host", parsed.Host, "error", err)
		return result, err
	}
	if path, err = renameByContent(path, contentType); err != nil {
		return result, err
	}
	source := d.PageURL
	if source == "" {
		source = d.URL
	}
	request.Paths, request.SourceURL = []string{path}, source
	return s.importOne(ctx, request)
}

// ImportUpload 导入扩展直接传来的文件内容（blob:、截图、浏览器里下载的文件）。path 是已写好的临时文件，name 是原文件名。
func (s *Service) ImportUpload(ctx context.Context, path, name, pageURL string, request model.ImportRequest) (result model.ImportResult, err error) {
	dir, err := os.MkdirTemp("", "momento-upload-")
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	// 换成原文件名放进单独的临时文件夹，导入时名称和格式都按原文件名来。
	named := filepath.Join(dir, safeName(name))
	if err = os.Rename(path, named); err != nil {
		return result, err
	}
	if named, err = renameByContent(named, ""); err != nil {
		return result, err
	}
	request.Paths, request.SourceURL = []string{named}, pageURL
	return s.importOne(ctx, request)
}

func (s *Service) importOne(ctx context.Context, request model.ImportRequest) (model.ImportResult, error) {
	result, err := s.Import(ctx, request, nil)
	// 只导一个文件，它没导进去就是失败，不能让浏览器扩展以为成功了。
	if err == nil && len(result.Errors) > 0 {
		return model.ImportResult{}, errors.Join(result.Errors...)
	}
	return result, err
}

// renameByContent 按文件开头的内容认出真实格式，后缀不对就改过来（CDN 常给 .jpg 地址返回 WebP）。
// 认不出时看文件名，再看 Content-Type；都不是支持的格式就报不支持。
func renameByContent(path, contentType string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	head := make([]byte, 64)
	n, _ := io.ReadFull(f, head)
	if err := f.Close(); err != nil {
		return "", err
	}
	current := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	extension := sniffExtension(head[:n])
	switch {
	case extension == "":
		if Supported(path) {
			return path, nil
		}
		extension = contentTypeExtensions[contentType]
		if extension == "" {
			return "", i18n.NewError("error_download_format", contentType)
		}
	case extension == "tiff" && (media.IsSystemImageExtension(current) || current == "tif"):
		// RAW 多是 TIFF 结构，原后缀更准。
		return path, nil
	case extension == current || extension == "jpg" && current == "jpeg" || extension == "mp4" && current == "m4v":
		return path, nil
	}
	renamed := strings.TrimSuffix(path, filepath.Ext(path)) + "." + extension
	if err := os.Rename(path, renamed); err != nil {
		return "", err
	}
	if current != "" {
		slog.Info("文件后缀和内容不符，按内容改过来", "from", current, "to", extension)
	}
	return renamed, nil
}

var contentTypeExtensions = map[string]string{
	"image/jpeg": "jpg", "image/png": "png", "image/gif": "gif", "image/webp": "webp", "image/tiff": "tiff", "image/bmp": "bmp",
	"image/heic": "heic", "image/heif": "heic", "image/avif": "avif",
	"video/mp4": "mp4", "video/quicktime": "mov", "video/webm": "webm", "video/x-m4v": "m4v",
}

// sniffExtension 按文件头认格式，认不出返回空。
func sniffExtension(head []byte) string {
	switch {
	case bytes.HasPrefix(head, []byte{0xff, 0xd8, 0xff}):
		return "jpg"
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return "gif"
	case len(head) >= 12 && bytes.HasPrefix(head, []byte("RIFF")) && string(head[8:12]) == "WEBP":
		return "webp"
	case bytes.HasPrefix(head, []byte("BM")) && len(head) >= 14:
		return "bmp"
	case bytes.HasPrefix(head, []byte("II*\x00")), bytes.HasPrefix(head, []byte("MM\x00*")):
		return "tiff"
	case bytes.HasPrefix(head, []byte("8BPS")):
		return "psd"
	case bytes.HasPrefix(head, []byte{0x1a, 0x45, 0xdf, 0xa3}):
		return "webm"
	case len(head) >= 12 && string(head[4:8]) == "ftyp":
		switch brand := string(head[8:12]); brand {
		case "heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1":
			return "heic"
		case "avif", "avis":
			return "avif"
		case "qt  ":
			return "mov"
		case "M4V ", "M4VH", "M4VP":
			return "m4v"
		case "isom", "iso2", "iso4", "iso5", "iso6", "mp41", "mp42", "avc1", "dash", "MSNV", "NDAS":
			return "mp4"
		}
	}
	return ""
}

// 下载上限与超时，测试里可以调小。
var (
	maxDownloadBytes       int64 = 200 << 20
	maxVideoDownloadBytes  int64 = 2 << 30
	downloadConnectTimeout       = 15 * time.Second
	downloadIdleTimeout          = 30 * time.Second
	errDownloadStalled           = errors.New("download stalled")
)

// idleReader 每读到新数据就把空闲计时重新开始。
type idleReader struct {
	r     io.Reader
	timer *time.Timer
}

func (r idleReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.timer.Reset(downloadIdleTimeout)
	}
	return n, err
}

var urlPattern = regexp.MustCompile(`(?i)\b[a-z][a-z0-9+.-]*://[^\s"']+`)

// redactURLs 把文字里的网址换成 [url]。
func redactURLs(text string) string {
	return urlPattern.ReplaceAllString(text, "[url]")
}
