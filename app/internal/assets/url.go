package assets

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
)

func (s *Service) ImportURL(ctx context.Context, imageURL, pageURL string) (result model.ImportResult, err error) {
	parsed, err := url.Parse(imageURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return result, i18n.NewError("error_image_url")
	}
	slog.Info("下载图片", "host", parsed.Host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("User-Agent", "Momento/2.0")
	if pageURL != "" {
		req.Header.Set("Referer", pageURL)
	}
	client := &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
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
	if !supported(name) {
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
	_, err = io.Copy(f, contextReader{ctx: ctx, r: response.Body})
	err = errors.Join(err, f.Close())
	if err != nil {
		return result, err
	}
	source := pageURL
	if source == "" {
		source = imageURL
	}
	return s.Import(ctx, model.ImportRequest{Paths: []string{path}, SourceURL: source}, nil)
}
