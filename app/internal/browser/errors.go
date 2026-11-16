package browser

import (
	"context"
	"errors"
	"net/http"

	"github.com/Seaony/Momento/app/internal/i18n"
)

// 错误码：扩展按它决定怎么提示（打开图库、重试、换个方式存），不用去猜错误文字。
const (
	codeBadRequest    = "bad_request"
	codeForbidden     = "forbidden"
	codeNotFound      = "not_found"
	codeMethod        = "method_not_allowed"
	codeTooLarge      = "too_large"
	codeUnsupported   = "unsupported"
	codeNoLibrary     = "no_library"
	codeCanceled      = "canceled"
	codeBusy          = "busy"
	codeRemoteRefused = "remote_refused"
	codeDownload      = "download_failed"
	codeNotReady      = "not_ready"
	codeInternal      = "internal"
)

// classify 把导入失败的原因归到错误码和 HTTP 状态码。
func classify(err error) (string, int) {
	if err == nil {
		return "", http.StatusOK
	}
	for _, key := range localizedKeys(err) {
		switch key {
		case "library_required":
			return codeNoLibrary, http.StatusConflict
		case "import_canceled":
			return codeCanceled, http.StatusConflict
		case "busy":
			return codeBusy, http.StatusServiceUnavailable
		case "error_download_status":
			return codeRemoteRefused, http.StatusBadGateway
		case "error_download_failed", "error_download_stalled", "error_redirect_limit", "error_redirect_scheme":
			return codeDownload, http.StatusBadGateway
		case "error_download_too_large", "error_image_too_large":
			return codeTooLarge, http.StatusRequestEntityTooLarge
		case "error_download_format", "error_import_unsupported", "error_import_unsupported_file", "error_decode_image", "error_image_unreadable", "error_video_unreadable", "error_system_decode_unavailable":
			return codeUnsupported, http.StatusUnsupportedMediaType
		case "error_image_url":
			return codeBadRequest, http.StatusBadRequest
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return codeCanceled, http.StatusConflict
	}
	return codeInternal, http.StatusInternalServerError
}

// localizedKeys 从外到里列出错误链上所有本地化错误的 key。
func localizedKeys(err error) []string {
	var keys []string
	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		if localized, ok := e.(*i18n.LocalizedError); ok {
			keys = append(keys, localized.Key)
		}
		switch x := e.(type) {
		case interface{ Unwrap() []error }:
			for _, inner := range x.Unwrap() {
				walk(inner)
			}
		case interface{ Unwrap() error }:
			walk(x.Unwrap())
		}
	}
	walk(err)
	return keys
}
