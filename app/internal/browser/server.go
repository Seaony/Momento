package browser

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
)

const (
	Address     = "127.0.0.1:47641"
	MaxBodySize = 512 << 10
)

type importPayload struct {
	URL          string `json:"url"`
	PageURL      string `json:"pageUrl"`
	PlayFeedback *bool  `json:"playFeedback"`
}

type response struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

func New(importImage func(context.Context, string, string) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		reply := func(code int, body response) {
			level := slog.LevelInfo
			if code >= 400 {
				level = slog.LevelWarn
			}
			slog.Log(r.Context(), level, "浏览器扩展请求", "method", r.Method, "path", r.URL.Path, "status", code, "duration", time.Since(started))
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(code)
			if code == http.StatusNoContent {
				return
			}
			if err := json.NewEncoder(w).Encode(body); err != nil {
				slog.Warn("浏览器扩展响应写入失败", "error", err)
			}
		}

		origins := r.Header.Values("Origin")
		origin := r.Header.Get("Origin")
		if len(origins) > 1 || (origin != "" && !strings.HasPrefix(origin, "chrome-extension://")) {
			reply(http.StatusForbidden, response{Error: "Origin is not allowed."})
			return
		}
		w.Header().Set("Vary", "Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		if r.URL.Path != "/api/v1/status" && r.URL.Path != "/api/v1/import/image" {
			reply(http.StatusNotFound, response{Error: "Unknown endpoint."})
			return
		}
		allowed := http.MethodGet
		if r.URL.Path == "/api/v1/import/image" {
			allowed = http.MethodPost
		}
		w.Header().Set("Allow", allowed+", OPTIONS")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", allowed+", OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			reply(http.StatusNoContent, response{})
			return
		}
		if r.Method != allowed {
			reply(http.StatusMethodNotAllowed, response{Error: "Method is not allowed."})
			return
		}
		if r.URL.Path == "/api/v1/status" {
			reply(http.StatusOK, response{OK: true, Status: "ready"})
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodySize))
		if err != nil {
			var limit *http.MaxBytesError
			if errors.As(err, &limit) {
				reply(http.StatusRequestEntityTooLarge, response{Error: "Request body exceeds 512 KB."})
			} else {
				reply(http.StatusBadRequest, response{Error: "Cannot read request body."})
			}
			return
		}
		var payload importPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			reply(http.StatusBadRequest, response{Error: "Invalid JSON request."})
			return
		}
		payload.URL = strings.TrimSpace(payload.URL)
		payload.PageURL = strings.TrimSpace(payload.PageURL)
		if !validURL(payload.URL) || (payload.PageURL != "" && !validURL(payload.PageURL)) {
			reply(http.StatusBadRequest, response{Error: "Image and page URLs must use HTTP or HTTPS."})
			return
		}
		if importImage == nil {
			reply(http.StatusServiceUnavailable, response{Error: "Momento is not ready."})
			return
		}
		if err := importImage(r.Context(), payload.URL, payload.PageURL); err != nil {
			reply(http.StatusInternalServerError, response{Error: err.Error()})
			return
		}
		reply(http.StatusOK, response{OK: true})
	})
}

func validURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

func Listen(ctx context.Context, handler http.Handler) error {
	if ctx.Err() != nil {
		return nil
	}
	listener, err := net.Listen("tcp4", Address)
	if err != nil {
		return i18n.WrapError(err, "error_browser_start")
	}
	slog.Info("浏览器扩展服务已启动", "address", Address)
	return serve(ctx, listener, handler)
}

func serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       time.Minute,
		MaxHeaderBytes:    32 << 10,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	stopped := make(chan struct{})
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdown); err != nil {
				slog.Warn("浏览器扩展服务停止超时", "error", err)
				if err := server.Close(); err != nil {
					slog.Warn("浏览器扩展连接关闭失败", "error", err)
				}
			}
		case <-stopped:
		}
	}()
	err := server.Serve(listener)
	close(stopped)
	<-shutdownDone
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
