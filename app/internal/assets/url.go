package assets

import (
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
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func (s *Service) ImportURL(ctx context.Context, imageURL, pageURL string) (model.ImportResult, error) {
	return s.ImportURLWithRequest(ctx, imageURL, pageURL, model.ImportRequest{})
}

func (s *Service) ImportURLWithRequest(ctx context.Context, imageURL, pageURL string, request model.ImportRequest) (result model.ImportResult, err error) {
	parsed, err := url.Parse(imageURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return result, i18n.NewError("error_image_url")
	}
	slog.Info("下载图片", "host", parsed.Host)
	// 下载用单独的可取消上下文：30 秒没收到新数据就取消。
	downloadCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, imageURL, nil)
	if err != nil {
		return result, err
	}
	userAgent := "Momento"
	if s.AppVersion != "" {
		userAgent += "/" + s.AppVersion
	}
	req.Header.Set("User-Agent", userAgent)
	if pageURL != "" {
		req.Header.Set("Referer", pageURL)
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
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result, i18n.NewError("error_download_status", response.StatusCode)
	}
	name := filepath.Base(parsed.Path)
	if _, params, e := mime.ParseMediaType(response.Header.Get("Content-Disposition")); e == nil && params["filename"] != "" {
		name = params["filename"]
	}
	name = safeName(name)
	if !Supported(name) {
		contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
		extension := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/gif": "gif", "image/webp": "webp", "image/tiff": "tiff", "image/bmp": "bmp", "image/heic": "heic", "image/heif": "heic", "image/avif": "avif"}[strings.ToLower(contentType)]
		if extension == "" {
			return result, i18n.NewError("error_download_format", contentType)
		}
		name = strings.TrimSuffix(name, filepath.Ext(name)) + "." + extension
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
	tooLarge := i18n.NewError("error_download_too_large", maxDownloadBytes>>20)
	if response.ContentLength > maxDownloadBytes {
		return result, errors.Join(tooLarge, f.Close())
	}
	idle := time.AfterFunc(downloadIdleTimeout, func() { cancel(errDownloadStalled) })
	written, err := io.Copy(f, idleReader{r: io.LimitReader(contextReader{ctx: downloadCtx, r: response.Body}, maxDownloadBytes+1), timer: idle})
	idle.Stop()
	if err != nil && errors.Is(context.Cause(downloadCtx), errDownloadStalled) {
		err = i18n.NewError("error_download_stalled", int(downloadIdleTimeout/time.Second))
	} else if err == nil && written > maxDownloadBytes {
		err = tooLarge
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		slog.Warn("下载图片失败", "host", parsed.Host, "error", err)
		return result, err
	}
	source := pageURL
	if source == "" {
		source = imageURL
	}
	request.Paths, request.SourceURL = []string{path}, source
	result, err = s.Import(ctx, request, nil)
	// 只导一张图，这张没导进去就是失败，不能让浏览器扩展以为成功了。
	if err == nil && len(result.Errors) > 0 {
		return model.ImportResult{}, errors.Join(result.Errors...)
	}
	return result, err
}

// 下载上限与超时，测试里可以调小。
var (
	maxDownloadBytes       int64 = 200 << 20
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
