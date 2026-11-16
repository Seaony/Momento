package browser

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Seaony/Momento/app/internal/i18n"
)

const (
	Address     = "127.0.0.1:47641"
	MaxBodySize = 512 << 10
	// MaxUploadSize 是 /api/v1/import/file 一次能传的文件大小（blob:、截图、浏览器里下载的文件）。
	MaxUploadSize = 512 << 20
)

// allowedOrigins 是能调用本机接口的扩展。ID 由扩展 manifest.json 里固定的公钥（key）算出，
// 手动加载未打包的扩展也是这个 ID；别的扩展一律拒绝，免得被拿去让 Momento 下载任意网址。
var allowedOrigins = map[string]bool{
	"chrome-extension://bdholohelelbeepbpdapldbibidacaoo": true,
}

// Request 是扩展发来的一次导入。
type Request struct {
	URL     string `json:"url"`
	PageURL string `json:"pageUrl"`
	// Referer 是扩展按 CDN 算好的值，空就用网页地址。
	Referer string `json:"referer"`
	// Kind 是 image 或 video，空按图片；视频按视频的大小上限下载。
	Kind  string   `json:"kind"`
	Title string   `json:"title"`
	Note  string   `json:"note"`
	Tags  []string `json:"tags"`
	// 同一批（比如存一整篇笔记）的编号和总数：同一批只弹一次选位置窗口、只发一次通知。
	BatchID      string `json:"batchId"`
	Index        int    `json:"index"`
	Total        int    `json:"total"`
	PlayFeedback *bool  `json:"playFeedback"`
	// 上传的文件（/api/v1/import/file）：本机临时路径和原文件名，扩展传不了这两个。
	File     string `json:"-"`
	FileName string `json:"-"`
}

// ImportFunc 真正去导入；返回的错误按 classify 归类给扩展。
type ImportFunc func(context.Context, Request) error

// Options 是本机接口的配置。
type Options struct {
	Version string
	Import  ImportFunc
}

type response struct {
	OK           bool          `json:"ok"`
	Error        string        `json:"error,omitempty"`
	Code         string        `json:"code,omitempty"`
	Status       string        `json:"status,omitempty"`
	Version      string        `json:"version,omitempty"`
	Capabilities *capabilities `json:"capabilities,omitempty"`
	JobID        string        `json:"jobId,omitempty"`
	Job          *jobView      `json:"job,omitempty"`
}

// capabilities 告诉扩展这个版本的 Momento 能做什么，扩展按它开功能，新旧版本可以混用。
type capabilities struct {
	Jobs           bool  `json:"jobs"`
	Upload         bool  `json:"upload"`
	Video          bool  `json:"video"`
	Batch          bool  `json:"batch"`
	MaxUploadBytes int64 `json:"maxUploadBytes"`
}

type jobView struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

type serverContextKey struct{}

type handler struct {
	options Options
	jobs    *jobQueue
}

func New(options Options) http.Handler {
	h := &handler{options: options}
	if options.Import != nil {
		h.jobs = newJobQueue(options.Import)
	}
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	reply := func(code int, body response) {
		level := slog.LevelInfo
		if code >= 400 {
			level = slog.LevelWarn
		}
		// 只记路径，不记网址和查询参数，免得把链接里的签名写进日志。
		slog.Log(r.Context(), level, "浏览器扩展请求", "method", r.Method, "path", r.URL.Path, "status", code, "code", body.Code, "duration", time.Since(started))
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
	fail := func(status int, code, message string) {
		reply(status, response{Error: message, Code: code})
	}

	origins := r.Header.Values("Origin")
	origin := strings.TrimSuffix(r.Header.Get("Origin"), "/")
	if len(origins) > 1 || (origin != "" && !allowedOrigins[origin]) {
		fail(http.StatusForbidden, codeForbidden, "Origin is not allowed.")
		return
	}
	w.Header().Set("Vary", "Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	}
	path := r.URL.Path
	allowed := ""
	switch {
	case path == "/api/v1/status":
		allowed = http.MethodGet
	case path == "/api/v1/import", path == "/api/v1/import/image", path == "/api/v1/import/file":
		allowed = http.MethodPost
	case strings.HasPrefix(path, "/api/v1/import/") && !strings.Contains(strings.TrimPrefix(path, "/api/v1/import/"), "/"):
		allowed = http.MethodGet
	default:
		fail(http.StatusNotFound, codeNotFound, "Unknown endpoint.")
		return
	}
	w.Header().Set("Allow", allowed+", OPTIONS")
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", allowed+", OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		reply(http.StatusNoContent, response{})
		return
	}
	if r.Method != allowed {
		fail(http.StatusMethodNotAllowed, codeMethod, "Method is not allowed.")
		return
	}
	if path == "/api/v1/status" {
		reply(http.StatusOK, response{OK: true, Status: "ready", Version: h.options.Version,
			Capabilities: &capabilities{Jobs: true, Upload: true, Video: true, Batch: true, MaxUploadBytes: MaxUploadSize}})
		return
	}
	if h.jobs == nil {
		fail(http.StatusServiceUnavailable, codeNotReady, "Momento is not ready.")
		return
	}
	// 任务跟着这次服务运行走：关掉扩展服务时一起取消；请求断开不影响已经排上的任务。
	runCtx, ok := r.Context().Value(serverContextKey{}).(context.Context)
	if !ok {
		runCtx = context.WithoutCancel(r.Context())
	}
	switch path {
	case "/api/v1/import", "/api/v1/import/image":
		request, status, code, message := readRequest(w, r)
		if code != "" {
			fail(status, code, message)
			return
		}
		j, ok := h.jobs.submit(runCtx, request)
		if !ok {
			fail(http.StatusServiceUnavailable, codeBusy, "Too many imports are waiting.")
			return
		}
		if path == "/api/v1/import" {
			reply(http.StatusAccepted, response{OK: true, JobID: j.id})
			return
		}
		// 旧接口：等这个任务做完再回，旧版扩展照样能用。
		select {
		case <-j.done:
		case <-r.Context().Done():
			fail(http.StatusServiceUnavailable, codeCanceled, "Request was canceled.")
			return
		}
		if j.err != nil {
			code, status := classify(j.err)
			fail(status, code, j.err.Error())
			return
		}
		reply(http.StatusOK, response{OK: true})
	case "/api/v1/import/file":
		request, status, code, message := readUpload(w, r)
		if code != "" {
			fail(status, code, message)
			return
		}
		j, ok := h.jobs.submit(runCtx, request)
		if !ok {
			removeUpload(request.File)
			fail(http.StatusServiceUnavailable, codeBusy, "Too many imports are waiting.")
			return
		}
		reply(http.StatusAccepted, response{OK: true, JobID: j.id})
	default:
		view, ok := h.jobs.view(strings.TrimPrefix(path, "/api/v1/import/"))
		if !ok {
			fail(http.StatusNotFound, codeNotFound, "Unknown import job.")
			return
		}
		reply(http.StatusOK, response{OK: true, Job: &view})
	}
}

// readRequest 读 JSON 导入请求并检查字段。
func readRequest(w http.ResponseWriter, r *http.Request) (Request, int, string, string) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodySize))
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			return Request{}, http.StatusRequestEntityTooLarge, codeTooLarge, "Request body exceeds 512 KB."
		}
		return Request{}, http.StatusBadRequest, codeBadRequest, "Cannot read request body."
	}
	var request Request
	if err := json.Unmarshal(body, &request); err != nil {
		return Request{}, http.StatusBadRequest, codeBadRequest, "Invalid JSON request."
	}
	if message := validate(&request, true); message != "" {
		return Request{}, http.StatusBadRequest, codeBadRequest, message
	}
	return request, 0, "", ""
}

// validate 检查并整理请求字段；needURL 为假时（上传文件）不要求文件地址。
func validate(request *Request, needURL bool) string {
	request.URL = strings.TrimSpace(request.URL)
	request.PageURL = strings.TrimSpace(request.PageURL)
	request.Referer = strings.TrimSpace(request.Referer)
	switch {
	case needURL && !validURL(request.URL), !needURL && request.URL != "" && !validURL(request.URL),
		request.PageURL != "" && !validURL(request.PageURL), request.Referer != "" && !validURL(request.Referer):
		return "File, page and referer URLs must use HTTP or HTTPS."
	case request.Kind != "" && request.Kind != "image" && request.Kind != "video":
		return "Kind must be image or video."
	case request.Total < 0 || request.Index < 0 || request.Total > 0 && request.Index >= request.Total:
		return "Batch index is out of range."
	case len(request.BatchID) > 128 || len(request.Tags) > 50 || utf8.RuneCountInString(request.Title) > 1000 || utf8.RuneCountInString(request.Note) > 10000:
		return "Request fields are too long."
	}
	for _, tag := range request.Tags {
		if utf8.RuneCountInString(tag) > 100 {
			return "Request fields are too long."
		}
	}
	return ""
}

// readUpload 读 multipart 上传：meta 是和 /api/v1/import 一样的 JSON（不用填 url），file 是文件内容，写进临时文件。
func readUpload(w http.ResponseWriter, r *http.Request) (request Request, status int, code, message string) {
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "multipart/form-data" {
		return Request{}, http.StatusBadRequest, codeBadRequest, "Upload must be multipart/form-data."
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize+MaxBodySize)
	reader, err := r.MultipartReader()
	if err != nil {
		return Request{}, http.StatusBadRequest, codeBadRequest, "Cannot read upload."
	}
	var meta []byte
	defer func() {
		if code != "" {
			removeUpload(request.File)
		}
	}()
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			status, code, message = uploadReadError(err)
			return
		}
		switch part.FormName() {
		case "meta":
			if meta, err = io.ReadAll(io.LimitReader(part, MaxBodySize+1)); err != nil {
				status, code, message = uploadReadError(err)
				return
			}
			if len(meta) > MaxBodySize {
				return request, http.StatusRequestEntityTooLarge, codeTooLarge, "Upload metadata exceeds 512 KB."
			}
		case "file":
			if request.File != "" {
				return request, http.StatusBadRequest, codeBadRequest, "Upload has more than one file."
			}
			f, err := os.CreateTemp("", "momento-upload-*")
			if err != nil {
				return request, http.StatusInternalServerError, codeInternal, "Cannot store upload."
			}
			request.File, request.FileName = f.Name(), part.FileName()
			written, copyErr := io.Copy(f, io.LimitReader(part, MaxUploadSize+1))
			if err := errors.Join(copyErr, f.Close()); err != nil {
				status, code, message = uploadReadError(err)
				return
			}
			if written > MaxUploadSize {
				return request, http.StatusRequestEntityTooLarge, codeTooLarge, "File exceeds 512 MB."
			}
		}
		part.Close()
	}
	if request.File == "" || strings.TrimSpace(request.FileName) == "" {
		return request, http.StatusBadRequest, codeBadRequest, "Upload needs a file with a name."
	}
	file, name := request.File, request.FileName
	if len(meta) > 0 {
		if err := json.Unmarshal(meta, &request); err != nil {
			return request, http.StatusBadRequest, codeBadRequest, "Invalid JSON metadata."
		}
	}
	request.File, request.FileName = file, name
	if message := validate(&request, false); message != "" {
		return request, http.StatusBadRequest, codeBadRequest, message
	}
	return request, 0, "", ""
}

func uploadReadError(err error) (int, string, string) {
	var limit *http.MaxBytesError
	if errors.As(err, &limit) {
		return http.StatusRequestEntityTooLarge, codeTooLarge, "Upload exceeds 512 MB."
	}
	return http.StatusBadRequest, codeBadRequest, "Cannot read upload."
}

func removeUpload(path string) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		slog.Warn("删除上传的临时文件失败", "error", err)
	}
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
		// 上传文件最大 512 MB，本机传输很快，留足时间。
		ReadTimeout:    2 * time.Minute,
		IdleTimeout:    time.Minute,
		MaxHeaderBytes: 32 << 10,
		BaseContext: func(net.Listener) context.Context {
			return context.WithValue(ctx, serverContextKey{}, ctx)
		},
		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
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
